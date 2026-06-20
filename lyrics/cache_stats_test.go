package lyrics

import (
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestSharedImageCacheStatsTracksRefs(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	start := color.RGBA{255, 255, 255, 255}
	end := color.RGBA{255, 255, 255, 60}

	first, err := CreateSyllableImage("cache", manager, req, 32, 0.6, start, end)
	if err != nil {
		t.Fatalf("create first syllable image: %v", err)
	}
	second, err := CreateSyllableImage("cache", manager, req, 32, 0.6, start, end)
	if err != nil {
		t.Fatalf("create second syllable image: %v", err)
	}

	if !first.ensureResources() || !second.ensureResources() {
		t.Fatal("expected resources to be created")
	}

	stats := SharedImageCacheStats()
	if stats.TextMaskEntries != 1 || stats.TextMaskRefs != 2 {
		t.Fatalf("text mask stats = %+v, want one shared mask with two refs", stats)
	}
	if stats.GradientEntries != 2 || stats.GradientRefs != 4 {
		t.Fatalf("gradient stats = %+v, want base/highlight gradients shared by two images", stats)
	}

	first.Dispose()
	stats = SharedImageCacheStats()
	if stats.TextMaskEntries != 1 || stats.TextMaskRefs != 1 {
		t.Fatalf("after one dispose text mask stats = %+v, want one remaining ref", stats)
	}
	if stats.GradientEntries != 2 || stats.GradientRefs != 2 {
		t.Fatalf("after one dispose gradient stats = %+v, want one base/highlight ref", stats)
	}

	second.Dispose()
	stats = SharedImageCacheStats()
	if stats.TextMaskEntries != 0 || stats.TextMaskRefs != 0 || stats.GradientEntries != 0 || stats.GradientRefs != 0 {
		t.Fatalf("after all dispose stats = %+v, want empty cache", stats)
	}
}

func TestSyllableImageDrawRehydratesResourcesBeforeDrawing(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	syllable, err := CreateSyllableImage(
		"draw",
		manager,
		req,
		32,
		0.6,
		color.RGBA{255, 255, 255, 255},
		color.RGBA{255, 255, 255, 60},
	)
	if err != nil {
		t.Fatalf("create syllable image: %v", err)
	}
	if !syllable.ensureResources() {
		t.Fatal("expected initial resources")
	}
	syllable.Dispose()
	if syllable.TextMask != nil || syllable.GradientImage != nil || syllable.HighlightGradientImage != nil {
		t.Fatal("dispose should clear local resource handles")
	}

	width := safeImageLength(syllable.Width)
	height := safeImageLength(syllable.Height)
	dst := ebiten.NewImage(width, height)
	pos := NewPosition(0, 0, syllable.Width, syllable.Height)
	syllable.Draw(dst, syllable.GetOffset(), 1, &pos)

	if syllable.TextMask == nil || syllable.GradientImage == nil {
		t.Fatal("draw should restore base resource handles")
	}
	if syllable.HighlightGradientImage != nil {
		t.Fatal("base draw should not create highlight gradient")
	}
	syllable.DrawHighlight(dst, syllable.GetOffset(), 1, &pos)
	if syllable.HighlightGradientImage == nil {
		t.Fatal("highlight draw should restore highlight resource handle")
	}
}

func TestSyllableScratchPoolHasRetainBudget(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	for i := 0; i < scratchMaxRetained*4; i++ {
		img := syllableScratchImages.acquire(17+i*37, 23+i%5)
		syllableScratchImages.release(img)
	}

	stats := SharedImageCacheStats()
	if stats.ScratchImages > scratchMaxRetained {
		t.Fatalf("scratch images = %d, want <= %d", stats.ScratchImages, scratchMaxRetained)
	}
	if stats.ScratchPixels > scratchMaxRetainedPixel {
		t.Fatalf("scratch pixels = %d, want <= %d", stats.ScratchPixels, scratchMaxRetainedPixel)
	}
}

func TestRenderCacheStatsIncludesLineAndBottomCaches(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true
	line.Image = ebiten.NewImage(4, 4)
	line.BlurImage = ebiten.NewImage(4, 4)
	line.imageDirty = true

	background := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 24)
	background.isShow = true
	background.Image = ebiten.NewImage(4, 4)
	background.imageDirty = false
	line.BackgroundLines = []*Line{background}

	lyrics := &Lyrics{Lines: []*Line{line}, Bottom: newBottomLine(32, 500)}
	lyrics.Bottom.SetText("Creator", 32)
	lyrics.Bottom.Image = ebiten.NewImage(4, 4)
	lyrics.Bottom.BlurImage = ebiten.NewImage(4, 4)
	lyrics.Bottom.ImageDirty = true

	stats := lyrics.RenderCacheStats()
	if stats.VisibleLines != 2 || stats.LineImages != 2 || stats.DirtyLineImages != 1 || stats.LineBlurImages != 1 {
		t.Fatalf("line cache stats = %+v, want visible/images/dirty/blur 2/2/1/1", stats)
	}
	if !stats.BottomImage || !stats.BottomImageDirty || !stats.BottomBlurImage {
		t.Fatalf("bottom cache stats = %+v, want image/dirty/blur true", stats)
	}

	line.Dispose()
	lyrics.Bottom.Dispose()
}

func TestHideLineKeepsRenderableResources(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	line := NewLine(0, time.Second, false, false, "", manager, req, 32)
	syllable, err := NewSyllable(
		"cache",
		0,
		time.Second,
		manager,
		req,
		32,
		0.6,
		color.RGBA{255, 255, 255, 255},
		color.RGBA{255, 255, 255, 60},
		false,
	)
	if err != nil {
		t.Fatalf("create syllable: %v", err)
	}
	line.SetSyllables([]*LineSyllable{syllable})
	line.GetPosition().SetW(200)
	line.GetPosition().SetH(80)
	line.Render()
	line.BlurImage = ebiten.NewImage(4, 4)
	if line.Image == nil {
		t.Fatal("render should create a line image")
	}
	if stats := SharedImageCacheStats(); stats.TextMaskEntries == 0 || stats.GradientEntries == 0 {
		t.Fatalf("resources were not created: %+v", stats)
	}

	lineRendererLayer.HideLine(line)
	if line.isShow {
		t.Fatal("hidden line should not stay drawable")
	}
	if line.Image == nil {
		t.Fatal("soft hide should keep the line image")
	}
	if line.BlurImage != nil {
		t.Fatal("soft hide should release derived blur image")
	}
	if stats := SharedImageCacheStats(); stats.TextMaskEntries == 0 || stats.GradientEntries == 0 {
		t.Fatalf("soft hide should keep shared resources: %+v", stats)
	}
	line.Render()
	if stats := SharedImageCacheStats(); stats.TextMaskEntries != 1 || stats.TextMaskRefs != 1 {
		t.Fatalf("rerender after soft hide should reuse shared text mask: %+v", stats)
	}

	line.Dispose()
	if line.Image != nil {
		t.Fatal("hard dispose should release the line image")
	}
	if stats := SharedImageCacheStats(); stats.TextMaskEntries != 0 || stats.GradientEntries != 0 {
		t.Fatalf("hard dispose should release shared resources: %+v", stats)
	}
}

func TestRecreateLineImageReusesSameSizeBitmap(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true
	line.GetPosition().SetW(120)
	line.GetPosition().SetH(48)

	lineRendererLayer.RecreateLineImage(line)
	first := line.Image
	if first == nil {
		t.Fatal("first recreate should allocate an image")
	}

	lineRendererLayer.RecreateLineImage(line)
	if line.Image != first {
		t.Fatal("same-size recreate should reuse the existing image")
	}

	line.GetPosition().SetH(64)
	lineRendererLayer.RecreateLineImage(line)
	if line.Image == nil {
		t.Fatal("resized recreate should leave an image")
	}
	if line.Image == first {
		t.Fatal("different-size recreate should allocate a new image")
	}
	line.Dispose()
}

func TestResizeVisibleLineCanRenderMainLyricsAfterResourceRelease(t *testing.T) {
	PurgeSharedImageCache()
	t.Cleanup(PurgeSharedImageCache)

	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	line := NewLine(0, time.Second, false, false, "translation", manager, req, 32)
	line.HasDuetInSong = false
	line.isShow = true
	line.setStatus(LineStatusPreviewStatic)
	line.GetPosition().SetAlpha(1)
	applyRefHorizontalLayout(line, 500, false)
	syllable, err := NewSyllable(
		"main",
		0,
		time.Second,
		manager,
		req,
		32,
		0.6,
		color.RGBA{255, 255, 255, 255},
		color.RGBA{255, 255, 255, 60},
		false,
	)
	if err != nil {
		t.Fatalf("create syllable: %v", err)
	}
	line.SetSyllables([]*LineSyllable{syllable})
	line.Layout()
	line.Render()
	if line.Image == nil {
		t.Fatal("initial render should create line image")
	}
	if syllable.Elements[0].SyllableImage.GradientImage == nil {
		t.Fatal("initial render should create main lyric resources")
	}

	line.Resize(700)
	if line.Image == nil {
		t.Fatal("resize should recreate visible line image")
	}
	line.Render()
	if line.Image == nil {
		t.Fatal("render after resize should keep line image")
	}
	if syllable.Elements[0].SyllableImage.TextMask == nil ||
		syllable.Elements[0].SyllableImage.GradientImage == nil ||
		syllable.Elements[0].SyllableImage.HighlightGradientImage == nil {
		t.Fatal("render after resize should restore main lyric resources")
	}
	line.Dispose()
}

func TestRenderLineKeepsCleanPreviewBitmap(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true
	line.setStatus(LineStatusPreviewStatic)
	line.GetPosition().SetAlpha(1)
	line.GetPosition().SetW(4)
	line.GetPosition().SetH(4)
	line.Image = ebiten.NewImage(4, 4)
	line.Image.Fill(color.RGBA{12, 34, 56, 255})
	line.imageDirty = false
	image := line.Image

	lineRendererLayer.RenderLine(line)
	if line.Image != image {
		t.Fatal("clean preview bitmap should be reused")
	}
	if line.imageDirty {
		t.Fatal("clean preview bitmap should stay clean")
	}
	line.Dispose()
}

func TestHideLineWithManagerCancelsPendingAnimations(t *testing.T) {
	manager := anim.NewManager(false)
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.ScaleAnimate = anim.NewTween("scale", time.Second, 0, 1, 0, 1, anim.Linear, func(float64) {}, func() {})
	manager.Add(line.ScaleAnimate)

	lineRendererLayer.HideLineWithManager(line, manager)
	manager.Update(time.Second / 60)
	if line.ScaleAnimate != nil {
		t.Fatal("soft hide should clear the line animation reference")
	}
}

func TestTickHiddenResourcePruneReleasesIdleCache(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.Image = ebiten.NewImage(4, 4)
	line.lastVisibleAt = 0
	line.isShow = false
	lyrics := &Lyrics{
		Lines:       []*Line{line},
		anchorIndex: 0,
		Timeline: TimelineState{
			CurrentTime: hiddenLineRetainIdle + time.Second,
		},
	}

	lineRendererLayer.TickHiddenResourcePrune(lyrics, hiddenLinePruneInterval)
	if line.Image != nil {
		t.Fatal("idle hidden resource prune should hard-release the line image")
	}
}
