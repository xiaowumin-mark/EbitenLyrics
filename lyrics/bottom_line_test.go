package lyrics

import (
	"testing"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestNewBottomLineDefaultsToEmpty(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	if bottom.Active {
		t.Fatal("empty bottom line should not be active")
	}
	if bottom.LineSize != [2]float64{1000, 0} {
		t.Fatalf("line size = %v, want [1000 0]", bottom.LineSize)
	}
	if bottom.Position.GetW() != 1000 || bottom.Position.GetH() != 0 {
		t.Fatalf("position size = %v,%v want 1000,0", bottom.Position.GetW(), bottom.Position.GetH())
	}
	if bottom.PosXSpring == nil || bottom.PosYSpring == nil {
		t.Fatal("bottom line springs should be initialized")
	}
}

func TestBottomLineResizeKeepsEmptyHeightZero(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.Resize(800)
	if bottom.LineSize != [2]float64{800, 0} {
		t.Fatalf("line size = %v, want [800 0]", bottom.LineSize)
	}
}

func TestBottomLineTextActivatesHeight(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetText("credits", 32)
	if !bottom.Active {
		t.Fatal("bottom line with text should be active")
	}
	if !bottom.HasContent() {
		t.Fatal("bottom line with text should report content")
	}
	if bottom.LineSize[1] < 57.6 {
		t.Fatalf("height = %v, want at least 57.6", bottom.LineSize[1])
	}
	bottom.SetText("", 32)
	if bottom.Active || bottom.HasContent() || bottom.LineSize[1] != 0 {
		t.Fatalf("cleared bottom line active/content/height = %v/%v/%v, want false/false/0", bottom.Active, bottom.HasContent(), bottom.LineSize[1])
	}
}

func TestBottomLineWhitespaceMatchesEmptyRefSlot(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetText(" \t\n ", 32)
	if bottom.Active || bottom.HasContent() || bottom.LineSize[1] != 0 {
		t.Fatalf("whitespace bottom active/content/height = %v/%v/%v, want false/false/0", bottom.Active, bottom.HasContent(), bottom.LineSize[1])
	}
}

func TestBottomLineSetTextMarksImageDirty(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.ImageDirty = false
	bottom.SetText("credits", 32)
	if !bottom.ImageDirty {
		t.Fatal("set text should mark bottom image dirty")
	}
	if bottom.ContentFontSize <= 0 || bottom.PaddingTop <= 0 {
		t.Fatalf("text metrics font/padding = %v/%v/%v, want positive", bottom.ContentFontSize, bottom.PaddingLeft, bottom.PaddingTop)
	}
	if bottom.PaddingLeft != lineBasePadding(32, 1000) {
		t.Fatalf("left padding = %v, want lyrics padding %v", bottom.PaddingLeft, lineBasePadding(32, 1000))
	}
}

func TestBottomLineResizeRemeasuresText(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetText("credits", 32)
	height := bottom.LineSize[1]
	bottom.ImageDirty = false
	bottom.Resize(600)
	if bottom.LineSize[0] != 600 || bottom.Position.GetW() != 600 {
		t.Fatalf("width = %v/%v, want 600", bottom.LineSize[0], bottom.Position.GetW())
	}
	if bottom.LineSize[1] != height {
		t.Fatalf("height = %v, want unchanged %v for single line fallback", bottom.LineSize[1], height)
	}
	if bottom.PaddingLeft != lineBasePadding(32, 600) {
		t.Fatalf("left padding = %v, want lyrics padding %v", bottom.PaddingLeft, lineBasePadding(32, 600))
	}
	if !bottom.ImageDirty {
		t.Fatal("resize should mark bottom image dirty")
	}
}

func TestBottomLinePaddingAlignsWithLyrics(t *testing.T) {
	bottom := newBottomLine(48, 1200)
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 48)
	applyRefHorizontalLayout(line, 1200, false)
	if bottom.PaddingLeft != line.EffectivePaddingLeft() {
		t.Fatalf("bottom left padding = %v, want line left padding %v", bottom.PaddingLeft, line.EffectivePaddingLeft())
	}
}

func TestBottomLineClearImageClearsBlurCache(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.Image = nil
	bottom.BlurCacheKey = 123
	bottom.BlurCacheSource = nil
	bottom.clearImage()
	if bottom.BlurCacheKey != 0 || bottom.BlurCacheSource != nil {
		t.Fatalf("blur cache key/source = %v/%v, want cleared", bottom.BlurCacheKey, bottom.BlurCacheSource)
	}
	if !bottom.ImageDirty {
		t.Fatal("clear image should mark image dirty")
	}
}

func TestBottomLineSetTransformForceSnaps(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetTransform(12, 34, 2, true, 0)
	if bottom.Position.GetX() != 12 || bottom.Position.GetY() != 34 {
		t.Fatalf("position = %v,%v want 12,34", bottom.Position.GetX(), bottom.Position.GetY())
	}
	if bottom.BlurLevel != 2 {
		t.Fatalf("blur = %v, want 2", bottom.BlurLevel)
	}
}

func TestBottomLineUpdateMovesTowardTarget(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetTransform(0, 120, 0, false, 0)
	bottom.Update(16 * time.Millisecond)
	if bottom.Position.GetY() <= 0 || bottom.Position.GetY() >= 120 {
		t.Fatalf("y = %v, want between 0 and 120", bottom.Position.GetY())
	}
}

func TestBottomLineDelayedTransformWaitsBeforeMoving(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetTransform(0, 120, 0, false, 100*time.Millisecond)
	bottom.Update(16 * time.Millisecond)
	if bottom.Position.GetY() != 0 {
		t.Fatalf("y = %v, want unchanged before delay", bottom.Position.GetY())
	}
	bottom.Update(50 * time.Millisecond)
	bottom.Update(50 * time.Millisecond)
	if bottom.Position.GetY() <= 0 || bottom.Position.GetY() >= 120 {
		t.Fatalf("y = %v, want moving toward delayed target", bottom.Position.GetY())
	}
}

func TestBottomLineForceClearsQueuedTarget(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	bottom.SetTransform(0, 120, 0, false, 100*time.Millisecond)
	bottom.SetTransform(0, 40, 0, true, 0)
	bottom.Update(150 * time.Millisecond)
	if bottom.Position.GetY() != 40 {
		t.Fatalf("y = %v, want force target 40 with queued target cleared", bottom.Position.GetY())
	}
}

func TestLyricsSetBottomLineTextUpdatesBottom(t *testing.T) {
	lyrics := &Lyrics{
		Bottom:         newBottomLine(32, 1000),
		Layout:         newLayoutState(),
		Timeline:       newTimelineState(),
		AnimateManager: anim.NewManager(false),
	}
	lyrics.SetBottomLineText("credits")
	if !lyrics.Bottom.Active || lyrics.Bottom.Text != "credits" {
		t.Fatalf("bottom active/text = %v/%q, want true/credits", lyrics.Bottom.Active, lyrics.Bottom.Text)
	}
	if lyrics.Bottom.LineSize[1] <= 0 {
		t.Fatalf("bottom height = %v, want positive", lyrics.Bottom.LineSize[1])
	}
	lyrics.ClearBottomLine()
	if lyrics.Bottom.Active || lyrics.Bottom.LineSize[1] != 0 {
		t.Fatalf("bottom active/height = %v/%v, want false/0", lyrics.Bottom.Active, lyrics.Bottom.LineSize[1])
	}
}

func TestScrollLyricsToAcceptsBottomAnchor(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, false)
	line.GetPosition().SetH(100)
	lyrics := &Lyrics{
		Lines:          []*Line{line},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	lyrics.Bottom.SetText("credits", 32)

	lineAnimationLayer.scrollLyricsTo(lyrics, nil, len(lyrics.Lines), 0)
	if lyrics.anchorIndex != len(lyrics.Lines) {
		t.Fatalf("anchorIndex = %d, want bottom index %d", lyrics.anchorIndex, len(lyrics.Lines))
	}
	if !lyrics.Bottom.Focused {
		t.Fatal("bottom line should be focused")
	}
	if len(lyrics.renderIndex) != 1 || lyrics.renderIndex[0] != 0 {
		t.Fatalf("renderIndex = %v, want only real line [0]", lyrics.renderIndex)
	}
	if lyrics.Bottom.Position.GetY() <= line.Position.GetY() {
		t.Fatalf("bottom y = %v should be after line y = %v", lyrics.Bottom.Position.GetY(), line.Position.GetY())
	}
}

func TestBottomLineLayoutUpdatesScrollBoundary(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, false)
	line.GetPosition().SetH(100)
	lyrics := &Lyrics{
		Lines:          []*Line{line},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	lyrics.Bottom.SetText("credits", 32)

	lineAnimationLayer.scrollLyricsTo(lyrics, nil, len(lyrics.Lines), 0)
	if lyrics.Bottom.Position.GetY() == 0 {
		t.Fatal("bottom line should receive layout transform")
	}
	if lyrics.Layout.ScrollMaxOffset < lyrics.Layout.ScrollMinOffset {
		t.Fatalf("scroll boundary max %v < min %v", lyrics.Layout.ScrollMaxOffset, lyrics.Layout.ScrollMinOffset)
	}
}

func TestBottomLineHeightContributesToScrollMaxOffset(t *testing.T) {
	lineA := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	lineB := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	for _, line := range []*Line{lineA, lineB} {
		applyRefHorizontalLayout(line, 1000, false)
		line.GetPosition().SetH(100)
	}
	withoutBottom := &Lyrics{
		Lines:          []*Line{lineA},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	withBottom := &Lyrics{
		Lines:          []*Line{lineB},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	withBottom.Bottom.SetText("credits", 32)

	lineAnimationLayer.scrollLyricsTo(withoutBottom, nil, len(withoutBottom.Lines), 0)
	lineAnimationLayer.scrollLyricsTo(withBottom, nil, len(withBottom.Lines), 0)

	delta := withBottom.Layout.ScrollMaxOffset - withoutBottom.Layout.ScrollMaxOffset
	want := withBottom.Bottom.LineSize[1] / 2
	if !nearlyEqual(delta, want) {
		t.Fatalf("scroll max delta = %v, want centered bottom contribution %v", delta, want)
	}
}

func TestScrollBoundaryMaxNeverBelowMin(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState()}
	lineAnimationLayer.updateScrollMinBoundary(lyrics, 300)
	lineAnimationLayer.updateScrollMaxBoundary(lyrics, -100, 1000)
	if lyrics.Layout.ScrollMaxOffset != lyrics.Layout.ScrollMinOffset {
		t.Fatalf("boundary max/min = %v/%v, want max clamped to min", lyrics.Layout.ScrollMaxOffset, lyrics.Layout.ScrollMinOffset)
	}
}

func TestBottomLineBlurIsZeroWhenFocused(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, false)
	line.GetPosition().SetH(100)
	lyrics := &Lyrics{
		Lines:          []*Line{line},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	lyrics.Layout.EnableBlur = true
	lyrics.Bottom.SetText("credits", 32)

	lineAnimationLayer.scrollLyricsTo(lyrics, nil, len(lyrics.Lines), 0)
	if lyrics.Bottom.BlurLevel != 0 {
		t.Fatalf("focused bottom blur = %v, want 0", lyrics.Bottom.BlurLevel)
	}
}

func TestUpdateLinePosYSpringParamsAppliesToBottomLine(t *testing.T) {
	bottom := newBottomLine(32, 1000)
	lyrics := &Lyrics{Bottom: bottom}
	params := anim.SpringParams{Mass: 2, Damping: 33, Stiffness: 222}
	updateLinePosYSpringParams(lyrics, params)
	got := lyrics.Bottom.PosYSpring.Params()
	if got.Mass != params.Mass || got.Damping != params.Damping || got.Stiffness != params.Stiffness {
		t.Fatalf("bottom spring params = %+v, want %+v", got, params)
	}
}

func TestCommitAfterLastLineWithoutBottomContentTargetsLastLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
		Bottom:   newBottomLine(32, 1000),
	}
	commitPlayerTimeState(lyrics, 1500*time.Millisecond, computePlayerTimeState(lyrics, 1500*time.Millisecond))
	if lyrics.Timeline.ScrollToIndex != len(lyrics.Lines)-1 {
		t.Fatalf("scrollToIndex = %d, want last line index %d", lyrics.Timeline.ScrollToIndex, len(lyrics.Lines)-1)
	}
}

func TestCommitAfterLastLineWithWhitespaceBottomTargetsLastLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
		Bottom:   newBottomLine(32, 1000),
	}
	lyrics.Bottom.SetText("   ", 32)
	commitPlayerTimeState(lyrics, 1500*time.Millisecond, computePlayerTimeState(lyrics, 1500*time.Millisecond))
	if lyrics.Timeline.ScrollToIndex != len(lyrics.Lines)-1 {
		t.Fatalf("scrollToIndex = %d, want last line index %d", lyrics.Timeline.ScrollToIndex, len(lyrics.Lines)-1)
	}
}

func TestBottomLineBlurIsZeroWhileUserScrolling(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, false)
	line.GetPosition().SetH(100)
	lyrics := &Lyrics{
		Lines:          []*Line{line},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	lyrics.Layout.EnableBlur = true
	lyrics.Layout.IsUserScrolling = true
	lyrics.Bottom.SetText("credits", 32)

	lineAnimationLayer.scrollLyricsTo(lyrics, []int{0}, 0, 0)
	if lyrics.Bottom.BlurLevel != 0 {
		t.Fatalf("user-scrolling bottom blur = %v, want 0", lyrics.Bottom.BlurLevel)
	}
}

func TestCommitAfterLastLineWithBottomContentTargetsBottomIndex(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
		Bottom:   newBottomLine(32, 1000),
	}
	lyrics.Bottom.SetText("credits", 32)
	commitPlayerTimeState(lyrics, 1500*time.Millisecond, computePlayerTimeState(lyrics, 1500*time.Millisecond))
	if lyrics.Timeline.ScrollToIndex != len(lyrics.Lines) {
		t.Fatalf("scrollToIndex = %d, want bottom index %d", lyrics.Timeline.ScrollToIndex, len(lyrics.Lines))
	}
}
