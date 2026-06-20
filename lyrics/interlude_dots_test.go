package lyrics

import (
	"testing"
	"time"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestInterludeDotsMetrics(t *testing.T) {
	dots := newInterludeDots(40)
	if !nearlyEqual(dots.DotSize, 20) {
		t.Fatalf("dot size = %v, want 20", dots.DotSize)
	}
	if !nearlyEqual(dots.Gap, 10) {
		t.Fatalf("gap = %v, want 10", dots.Gap)
	}
	if !nearlyEqual(dots.PaddingX, 8) {
		t.Fatalf("paddingX = %v, want 8", dots.PaddingX)
	}
	if dots.Margin != 16 {
		t.Fatalf("margin = %v, want 16", dots.Margin)
	}
	if !nearlyEqual(dots.PaddingY, 0.5) {
		t.Fatalf("paddingY = %v, want 0.5", dots.PaddingY)
	}
	if dots.TotalHeight() <= dots.Height() {
		t.Fatalf("total height = %v should include margins over height %v", dots.TotalHeight(), dots.Height())
	}
}

func TestInterludeDotsMetricsUseDotRelativeVerticalPadding(t *testing.T) {
	dots := newInterludeDots(40)
	dots.UpdateMetrics(40, 800)
	if !nearlyEqual(dots.DotSize, 20) {
		t.Fatalf("dot size = %v, want clamped font-relative 20", dots.DotSize)
	}
	if !nearlyEqual(dots.PaddingY, dots.DotSize*0.025) {
		t.Fatalf("paddingY = %v, want dot-relative %v", dots.PaddingY, dots.DotSize*0.025)
	}
}

func nearlyEqual(a, b float64) bool {
	if a > b {
		return a-b < 1e-9
	}
	return b-a < 1e-9
}

func TestInterludeDotsSetLayout(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetLayout(true, 12, 34)
	if !dots.Active {
		t.Fatal("dots should be active")
	}
	if dots.Position.GetX() != 12 || dots.Position.GetY() != 34 {
		t.Fatalf("position = %v,%v want 12,34", dots.Position.GetX(), dots.Position.GetY())
	}
	if dots.Position.GetW() != dots.Width() || dots.Position.GetH() != dots.Height() {
		t.Fatal("position size should match dots metrics")
	}
}

func TestInterludeDotsUpdateMetricsKeepsPosition(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetLayout(true, 123, 456)
	dots.UpdateMetrics(40, 800)
	if dots.Position.GetX() != 123 || dots.Position.GetY() != 456 {
		t.Fatalf("position = %v,%v want 123,456", dots.Position.GetX(), dots.Position.GetY())
	}
	if dots.Position.GetW() != dots.Width() || dots.Position.GetH() != dots.Height() {
		t.Fatal("position size should update to new metrics")
	}
}

func TestInterludeDotsSetInterludeInitializesTime(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: time.Second, EndTime: 7 * time.Second})
	if !dots.Active {
		t.Fatal("dots should be active")
	}
	if dots.CurrentTime != time.Second {
		t.Fatalf("current time = %v, want 1s", dots.CurrentTime)
	}
}

func TestInterludeDotsTickAdvancesCurrentTime(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: time.Second, EndTime: 7 * time.Second})
	dots.Tick(100 * time.Millisecond)
	if dots.CurrentTime != 1100*time.Millisecond {
		t.Fatalf("current time = %v, want 1.1s", dots.CurrentTime)
	}
}

func TestInterludeDotsSetInterludeAtSyncsTimelineTime(t *testing.T) {
	dots := newInterludeDots(32)
	interlude := &Interlude{StartTime: time.Second, EndTime: 7 * time.Second}
	dots.SetInterludeAt(interlude, 6750*time.Millisecond)
	if dots.CurrentTime != 6750*time.Millisecond {
		t.Fatalf("current time = %v, want synced 6.75s", dots.CurrentTime)
	}
	if dots.GlobalAlpha >= 1 {
		t.Fatalf("global alpha = %v, want exit phase fade", dots.GlobalAlpha)
	}
}

func TestInterludeDotsSetInterludeAtClampsTimelineTime(t *testing.T) {
	dots := newInterludeDots(32)
	interlude := &Interlude{StartTime: time.Second, EndTime: 7 * time.Second}
	dots.SetInterludeAt(interlude, 500*time.Millisecond)
	if dots.CurrentTime != time.Second {
		t.Fatalf("current time = %v, want clamped start 1s", dots.CurrentTime)
	}
	dots.SetInterludeAt(interlude, 8*time.Second)
	if dots.CurrentTime != 7*time.Second {
		t.Fatalf("current time = %v, want clamped end 7s", dots.CurrentTime)
	}
}

func TestInterludeDotsPauseStopsTime(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: time.Second, EndTime: 7 * time.Second})
	dots.Tick(100 * time.Millisecond)
	current := dots.CurrentTime
	dots.Pause()
	dots.Tick(time.Second)
	if dots.CurrentTime != current {
		t.Fatalf("paused current time = %v, want %v", dots.CurrentTime, current)
	}
	dots.Resume()
	dots.Tick(100 * time.Millisecond)
	if dots.CurrentTime <= current {
		t.Fatalf("resumed current time = %v, want greater than %v", dots.CurrentTime, current)
	}
}

func TestInterludeDotsInactiveDoesNotAdvance(t *testing.T) {
	dots := newInterludeDots(32)
	dots.Tick(time.Second)
	if dots.CurrentTime != 0 {
		t.Fatalf("inactive current time = %v, want 0", dots.CurrentTime)
	}
}

func TestInterludeDotsFadeAndDotSequence(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: 0, EndTime: 6 * time.Second})
	if dots.GlobalAlpha != 0 {
		t.Fatalf("initial global alpha = %v, want 0", dots.GlobalAlpha)
	}
	dots.Tick(750 * time.Millisecond)
	if dots.GlobalAlpha <= 0 || dots.GlobalAlpha >= 1 {
		t.Fatalf("fade-in global alpha = %v, want between 0 and 1", dots.GlobalAlpha)
	}
	if dots.DotAlphas[0] < dots.DotAlphas[1] || dots.DotAlphas[1] < dots.DotAlphas[2] {
		t.Fatalf("dot alpha order = %v, want earlier dots brighter", dots.DotAlphas)
	}
	dots.CurrentTime = 5900 * time.Millisecond
	dots.updateVisualState()
	if dots.GlobalAlpha >= 1 {
		t.Fatalf("ending global alpha = %v, want fading out", dots.GlobalAlpha)
	}
}

func TestInterludeDotsSetInterludeNilDisables(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: 0, EndTime: 6 * time.Second, IsNextDuet: true})
	dots.CurrentTime = 7 * time.Second
	dots.SetInterlude(nil)
	if dots.Active {
		t.Fatal("dots should be inactive after nil interlude")
	}
	if dots.IsDuet {
		t.Fatal("dots duet state should reset after nil interlude")
	}
	if dots.GlobalAlpha != 0 {
		t.Fatalf("global alpha = %v, want 0", dots.GlobalAlpha)
	}
}

func TestInterludeDotsStoresDuetState(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: 0, EndTime: 6 * time.Second, IsNextDuet: true})
	if !dots.IsDuet {
		t.Fatal("dots should store duet state from interlude")
	}
}

func TestInterludeDotsRepeatedSameInterludeSyncsTimeWithoutRestartingAtStart(t *testing.T) {
	dots := newInterludeDots(32)
	interlude := &Interlude{StartTime: 0, EndTime: 6 * time.Second}
	dots.SetInterludeAt(interlude, 2*time.Second)
	dots.SetInterludeAt(interlude, 4*time.Second)
	if dots.CurrentTime != 4*time.Second {
		t.Fatalf("current time = %v, want synced 4s", dots.CurrentTime)
	}
}

func TestInterludeDotsExitHappensBeforeInterludeEnd(t *testing.T) {
	dots := newInterludeDots(32)
	dots.SetInterlude(&Interlude{StartTime: 0, EndTime: 6 * time.Second})
	dots.CurrentTime = 5750 * time.Millisecond
	dots.updateVisualState()
	if dots.GlobalScale <= 0 {
		t.Fatalf("ending scale = %v, want visible exit state", dots.GlobalScale)
	}
	if dots.GlobalAlpha >= 1 {
		t.Fatalf("ending alpha = %v, want fade-out before interlude end", dots.GlobalAlpha)
	}
}

func TestInterludeDotsDuetAlignmentUsesFullWidth(t *testing.T) {
	lyrics := &Lyrics{Width: 1000, Dots: newInterludeDots(32)}
	dotX := interludeDotsXForLine(lyrics, nil, true)
	if !nearlyEqual(dotX, 1000-lyrics.Dots.Width()) {
		t.Fatalf("dot x = %v, want %v", dotX, 1000-lyrics.Dots.Width())
	}
}

func TestInterludeDotsDefaultAlignmentMatchesRefLeftZero(t *testing.T) {
	lyrics := &Lyrics{Width: 1000, Dots: newInterludeDots(32)}
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, true)
	dotX := interludeDotsXForLine(lyrics, line, false)
	if !nearlyEqual(dotX, 0) {
		t.Fatalf("dot x = %v, want ref default left 0", dotX)
	}
}

func TestInterludeDotsDuetAlignmentMatchesRefRightEdge(t *testing.T) {
	lyrics := &Lyrics{Width: 1000, Dots: newInterludeDots(32)}
	line := NewLine(0, time.Second, true, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, true)
	dotX := interludeDotsXForLine(lyrics, line, true)
	if !nearlyEqual(dotX+lyrics.Dots.Width(), lyrics.Width) {
		t.Fatalf("dot right = %v, want player width %v", dotX+lyrics.Dots.Width(), lyrics.Width)
	}
}
