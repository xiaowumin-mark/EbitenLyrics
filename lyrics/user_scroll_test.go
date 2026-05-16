package lyrics

import (
	"testing"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestSetUserScrollingUpdatesLayoutState(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState(), AnimateManager: anim.NewManager(false)}
	lyrics.SetUserScrolling(true)
	if !lyrics.Layout.IsUserScrolling {
		t.Fatal("SetUserScrolling(true) should update layout state")
	}
	lyrics.SetUserScrolling(false)
	if lyrics.Layout.IsUserScrolling {
		t.Fatal("SetUserScrolling(false) should update layout state")
	}
}

func TestRequestScrollRelayoutUsesTimelineScrollToIndex(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
			NewLine(time.Second, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	for _, line := range lyrics.Lines {
		applyRefHorizontalLayout(line, 1000, false)
		line.Layout()
	}
	lyrics.Timeline.ScrollToIndex = 1
	lyrics.anchorIndex = 0

	lineAnimationLayer.requestScrollRelayout(lyrics)
	if lyrics.anchorIndex != 1 {
		t.Fatalf("anchorIndex = %d, want timeline scrollToIndex 1", lyrics.anchorIndex)
	}
}

func TestSetScrollOffsetUpdatesLayoutState(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState(), AnimateManager: anim.NewManager(false)}
	lyrics.Layout.ScrollMinOffset = -200
	lyrics.Layout.ScrollMaxOffset = 200
	lyrics.SetScrollOffset(120)
	if lyrics.Layout.ScrollOffset != 120 {
		t.Fatalf("scroll offset = %v, want 120", lyrics.Layout.ScrollOffset)
	}
	lyrics.AddScrollOffset(-20)
	if lyrics.Layout.ScrollOffset != 100 {
		t.Fatalf("scroll offset = %v, want 100", lyrics.Layout.ScrollOffset)
	}
	lyrics.ResetScrollOffset()
	if lyrics.Layout.ScrollOffset != 0 {
		t.Fatalf("scroll offset = %v, want 0", lyrics.Layout.ScrollOffset)
	}
}

func TestSetBlurStrengthUpdatesLayoutState(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState(), AnimateManager: anim.NewManager(false)}
	lyrics.SetBlurStrength(2.5)
	if lyrics.Layout.BlurStrength != 2.5 {
		t.Fatalf("blur strength = %v, want 2.5", lyrics.Layout.BlurStrength)
	}
	lyrics.SetBlurStrength(10)
	if lyrics.Layout.BlurStrength != 4 {
		t.Fatalf("blur strength = %v, want clamped 4", lyrics.Layout.BlurStrength)
	}
}

func TestAddWheelScrollMarksScrolledAndClamps(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState(), AnimateManager: anim.NewManager(false)}
	lyrics.Layout.ScrollMinOffset = -50
	lyrics.Layout.ScrollMaxOffset = 50
	lyrics.AddWheelScroll(100)
	if !lyrics.Layout.IsScrolled {
		t.Fatal("wheel scroll should mark IsScrolled")
	}
	if lyrics.Layout.ScrollOffset != 50 {
		t.Fatalf("scroll offset = %v, want clamped 50", lyrics.Layout.ScrollOffset)
	}
	lyrics.ResetScrollOffset()
	if lyrics.Layout.IsScrolled || lyrics.Layout.ScrollOffset != 0 {
		t.Fatalf("reset scroll state = scrolled %v offset %v, want false 0", lyrics.Layout.IsScrolled, lyrics.Layout.ScrollOffset)
	}
}

func TestScrollOffsetAffectsLineLayoutTargets(t *testing.T) {
	baseLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	scrolledLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	for _, line := range []*Line{baseLine, scrolledLine} {
		applyRefHorizontalLayout(line, 1000, false)
		line.GetPosition().SetH(100)
	}
	base := &Lyrics{
		Lines:          []*Line{baseLine},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	scrolled := &Lyrics{
		Lines:          []*Line{scrolledLine},
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	scrolled.Layout.ScrollOffset = 80

	lineAnimationLayer.scrollLyricsTo(base, []int{0}, 0, 0)
	lineAnimationLayer.scrollLyricsTo(scrolled, []int{0}, 0, 0)

	if got := baseLine.GetPosition().GetY() - scrolledLine.GetPosition().GetY(); got != 80 {
		t.Fatalf("scroll offset y delta = %v, want 80", got)
	}
}

func TestScrollLayoutUpdatesBoundaries(t *testing.T) {
	lines := []*Line{
		NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		NewLine(time.Second, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
	}
	lyrics := &Lyrics{
		Lines:          lines,
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	for _, line := range lyrics.Lines {
		applyRefHorizontalLayout(line, 1000, false)
		line.GetPosition().SetH(100)
	}
	lineAnimationLayer.scrollLyricsTo(lyrics, []int{1}, 1, 0)
	if lyrics.Layout.ScrollMinOffset != -100 {
		t.Fatalf("min offset = %v, want -100", lyrics.Layout.ScrollMinOffset)
	}
	if lyrics.Layout.ScrollMaxOffset < lyrics.Layout.ScrollMinOffset {
		t.Fatalf("max offset = %v less than min %v", lyrics.Layout.ScrollMaxOffset, lyrics.Layout.ScrollMinOffset)
	}
}

func TestInitialLayoutUsesZeroDelayLikeRefSyncLayout(t *testing.T) {
	lines := []*Line{
		NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		NewLine(time.Second, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
	}
	lyrics := &Lyrics{
		Lines:          lines,
		Timeline:       newTimelineState(),
		Layout:         newLayoutState(),
		Dots:           newInterludeDots(32),
		Bottom:         newBottomLine(32, 1000),
		Width:          1000,
		AnimateManager: anim.NewManager(false),
	}
	for _, line := range lyrics.Lines {
		applyRefHorizontalLayout(line, 1000, false)
		line.GetPosition().SetH(100)
	}

	lineAnimationLayer.scrollLyricsTo(lyrics, []int{1}, 1, 0)
	if lyrics.Bottom.PosYSpring == nil {
		t.Fatal("bottom spring should be initialized")
	}
	if lyrics.Bottom.PosYSpring.TargetPosition() != lyrics.Bottom.Position.GetY() {
		t.Fatalf("bottom initial target/current = %v/%v, want snapped without delayed target", lyrics.Bottom.PosYSpring.TargetPosition(), lyrics.Bottom.Position.GetY())
	}
}
