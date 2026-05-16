package lyrics

import (
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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
