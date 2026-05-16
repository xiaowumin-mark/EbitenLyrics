package lyrics

import (
	"testing"
	"time"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestComputeLinePresentationScalesInactiveLine(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:          line,
		LineIndex:     2,
		ScrollToIndex: 0,
		LatestIndex:   1,
		IsPlaying:     true,
		EnableScale:   true,
	})

	if p.IsActive {
		t.Fatal("line should be inactive")
	}
	if p.TargetScale != 0.97 {
		t.Fatalf("target scale = %v, want 0.97", p.TargetScale)
	}
	if p.TargetAlpha != 1 {
		t.Fatalf("target alpha = %v, want 1", p.TargetAlpha)
	}
}

func TestComputeLinePresentationDimsNonDynamicLine(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:          line,
		LineIndex:     2,
		ScrollToIndex: 0,
		LatestIndex:   1,
		IsPlaying:     true,
		IsNonDynamic:  true,
		EnableScale:   true,
	})

	if p.TargetAlpha != 0.2 {
		t.Fatalf("target alpha = %v, want 0.2", p.TargetAlpha)
	}
}

func TestComputeLinePresentationBuffersActiveLine(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:          line,
		LineIndex:     0,
		ScrollToIndex: 0,
		LatestIndex:   0,
		HasBuffered:   true,
		IsPlaying:     true,
		EnableScale:   true,
	})

	if !p.IsActive {
		t.Fatal("buffered line should be active")
	}
	if p.TargetAlpha != 0.85 {
		t.Fatalf("target alpha = %v, want 0.85", p.TargetAlpha)
	}
	if p.TargetScale != 1 {
		t.Fatalf("target scale = %v, want 1", p.TargetScale)
	}
}

func TestComputeLinePresentationHidesPassedLine(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:            line,
		LineIndex:       0,
		ScrollToIndex:   2,
		LatestIndex:     2,
		HidePassedLines: true,
		IsPlaying:       true,
		EnableScale:     true,
	})

	if p.TargetAlpha != 1e-4 {
		t.Fatalf("target alpha = %v, want 1e-4", p.TargetAlpha)
	}
}

func TestComputeLinePresentationUsesInterludeAnchorForHidePassedLines(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:            line,
		LineIndex:       1,
		ScrollToIndex:   3,
		LatestIndex:     3,
		HidePassedLines: true,
		IsPlaying:       true,
		EnableScale:     true,
		Interlude:       &Interlude{AnchorLineIndex: 0},
	})

	if p.TargetAlpha != 1 {
		t.Fatalf("target alpha = %v, want visible after interlude anchor", p.TargetAlpha)
	}

	p = computeLinePresentation(linePresentationInput{
		Line:            line,
		LineIndex:       0,
		ScrollToIndex:   3,
		LatestIndex:     3,
		HidePassedLines: true,
		IsPlaying:       true,
		EnableScale:     true,
		Interlude:       &Interlude{AnchorLineIndex: 0},
	})
	if p.TargetAlpha != 1e-4 {
		t.Fatalf("target alpha = %v, want hidden before interlude anchor", p.TargetAlpha)
	}
}

func TestSetHidePassedLinesUpdatesLayoutState(t *testing.T) {
	lyrics := &Lyrics{Layout: newLayoutState(), Timeline: newTimelineState()}
	lyrics.SetHidePassedLines(true)
	if !lyrics.Layout.HidePassedLines {
		t.Fatal("SetHidePassedLines(true) should update layout state")
	}
	lyrics.SetHidePassedLines(false)
	if lyrics.Layout.HidePassedLines {
		t.Fatal("SetHidePassedLines(false) should update layout state")
	}
}

func TestNewLayoutStateMatchesRefBlurDefaults(t *testing.T) {
	layout := newLayoutState()
	if !layout.EnableBlur {
		t.Fatal("blur should be enabled by default like ref")
	}
	if layout.BlurStrength != 1 {
		t.Fatalf("blur strength = %v, want ref multiplier 1", layout.BlurStrength)
	}
}

func TestComputeLinePresentationDoesNotHidePassedLineWhenUserScrolling(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	p := computeLinePresentation(linePresentationInput{
		Line:            line,
		LineIndex:       0,
		ScrollToIndex:   2,
		LatestIndex:     2,
		HidePassedLines: false,
		IsPlaying:       true,
		EnableScale:     true,
		IsUserScrolling: true,
	})

	if p.TargetAlpha != 1 {
		t.Fatalf("target alpha = %v, want 1", p.TargetAlpha)
	}
}

func TestComputeLineBlurDisabledForActiveAndUserScroll(t *testing.T) {
	if got := computeLineBlur(computeLineBlurInput{EnableBlur: true, IsActive: true}); got != 0 {
		t.Fatalf("active blur = %v, want 0", got)
	}
	if got := computeLineBlur(computeLineBlurInput{EnableBlur: true, IsUserScrolling: true}); got != 0 {
		t.Fatalf("user scrolling blur = %v, want 0", got)
	}
}

func TestComputeLineBlurIncreasesWithDistance(t *testing.T) {
	near := computeLineBlur(computeLineBlurInput{
		EnableBlur:    true,
		ItemIndex:     3,
		ScrollToIndex: 2,
		LatestIndex:   2,
	})
	far := computeLineBlur(computeLineBlurInput{
		EnableBlur:    true,
		ItemIndex:     6,
		ScrollToIndex: 2,
		LatestIndex:   2,
	})
	if far <= near {
		t.Fatalf("far blur = %v, near blur = %v; want far > near", far, near)
	}
}

func TestComputeLineBlurMatchesRefAroundActiveRegion(t *testing.T) {
	tests := []struct {
		name          string
		itemIndex     int
		scrollToIndex int
		latestIndex   int
		want          float64
	}{
		{name: "previous line", itemIndex: 1, scrollToIndex: 2, latestIndex: 4, want: 3},
		{name: "active start", itemIndex: 2, scrollToIndex: 2, latestIndex: 4, want: 0},
		{name: "active buffered", itemIndex: 3, scrollToIndex: 2, latestIndex: 4, want: 0},
		{name: "first after buffered", itemIndex: 4, scrollToIndex: 2, latestIndex: 4, want: 1},
		{name: "far after buffered", itemIndex: 6, scrollToIndex: 2, latestIndex: 4, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isActive := tt.itemIndex >= tt.scrollToIndex && tt.itemIndex < tt.latestIndex
			got := computeLineBlur(computeLineBlurInput{
				EnableBlur:    true,
				IsActive:      isActive,
				ItemIndex:     tt.itemIndex,
				ScrollToIndex: tt.scrollToIndex,
				LatestIndex:   tt.latestIndex,
			})
			if got != tt.want {
				t.Fatalf("blur = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeLineBlurCompactMatchesRef(t *testing.T) {
	got := computeLineBlur(computeLineBlurInput{
		EnableBlur:    true,
		ItemIndex:     6,
		ScrollToIndex: 2,
		LatestIndex:   4,
		IsCompact:     true,
	})
	if !nearlyEqual(got, 2.4) {
		t.Fatalf("compact blur = %v, want 2.4", got)
	}
}

func TestComputeLineBlurStrengthMultiplier(t *testing.T) {
	got := computeLineBlur(computeLineBlurInput{
		EnableBlur:    true,
		BlurStrength:  2,
		ItemIndex:     6,
		ScrollToIndex: 2,
		LatestIndex:   4,
	})
	if got != 6 {
		t.Fatalf("blur = %v, want strength multiplied 6", got)
	}
}

func TestApplyLinePresentationStoresRenderMode(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.RenderMode = RenderModeSyllable
	applyLinePresentation(line, LinePresentation{TargetAlpha: 1, TargetScale: 1, RenderMode: RenderModeLine})
	if line.RenderMode != RenderModeLine {
		t.Fatalf("render mode = %v, want presentation mode %v", line.RenderMode, RenderModeLine)
	}
}

func TestApplyLinePresentationStoresBlurLevel(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyLinePresentation(line, LinePresentation{TargetAlpha: 1, TargetScale: 1, BlurLevel: 8})
	if line.BlurLevel != 5 {
		t.Fatalf("blur level = %v, want clamped 5", line.BlurLevel)
	}
}

func TestComputeBackgroundPresentationUsesRefScale(t *testing.T) {
	lyrics := &Lyrics{Timeline: newTimelineState(), Layout: newLayoutState()}
	bgLine := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 24)
	p := computeBackgroundLinePresentation(lyrics, bgLine, LinePresentation{IsActive: true})
	if p.TargetAlpha != 0.4 {
		t.Fatalf("active background alpha = %v, want ref opacity 0.4", p.TargetAlpha)
	}
	if p.TargetScale != 1 {
		t.Fatalf("active background scale = %v, want 1", p.TargetScale)
	}
	if !p.ReserveSpace {
		t.Fatal("active background should reserve space")
	}

	p = computeBackgroundLinePresentation(lyrics, bgLine, LinePresentation{IsActive: false})
	if p.TargetScale != 0.75 {
		t.Fatalf("background scale = %v, want 0.75", p.TargetScale)
	}
	if p.TargetAlpha != 0 {
		t.Fatalf("inactive playing background alpha = %v, want 0", p.TargetAlpha)
	}
	if p.ReserveSpace {
		t.Fatal("inactive background should not reserve space while playing")
	}

	lyrics.SetPlaying(false)
	p = computeBackgroundLinePresentation(lyrics, bgLine, LinePresentation{IsActive: false})
	if p.TargetAlpha != 0.4 {
		t.Fatalf("paused background alpha = %v, want ref opacity 0.4", p.TargetAlpha)
	}
	if p.TargetScale != 1 {
		t.Fatalf("paused background scale = %v, want 1", p.TargetScale)
	}
	if !p.ReserveSpace {
		t.Fatal("background should reserve space while paused")
	}
}

func TestPauseResumeUpdatesTimelineAndDots(t *testing.T) {
	lyrics := &Lyrics{Timeline: newTimelineState(), Layout: newLayoutState(), Dots: newInterludeDots(32)}
	lyrics.Dots.SetInterlude(&Interlude{StartTime: 0, EndTime: 6 * time.Second})
	lyrics.Pause()
	if lyrics.Timeline.IsPlaying {
		t.Fatal("Pause should set IsPlaying=false")
	}
	if !lyrics.Dots.Paused {
		t.Fatal("Pause should pause dots")
	}
	lyrics.Resume()
	if !lyrics.Timeline.IsPlaying {
		t.Fatal("Resume should set IsPlaying=true")
	}
	if lyrics.Dots.Paused {
		t.Fatal("Resume should resume dots")
	}
}
