package lyrics

import (
	"testing"
	"time"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func newInterludeTestLine(start, end time.Duration, duet bool) *Line {
	return NewLine(start, end, duet, false, "", nil, ft.FontRequest{}, 32)
}

func TestComputeCurrentInterludeReturnsNilForShortGap(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(0, time.Second, false),
			newInterludeTestLine(3*time.Second, 4*time.Second, false),
		},
		Timeline: newTimelineState(),
	}
	lyrics.Timeline.CurrentTime = 2 * time.Second
	lyrics.Timeline.ScrollToIndex = 1

	if got := computeCurrentInterlude(lyrics); got != nil {
		t.Fatalf("interlude = %+v, want nil for short gap", got)
	}
}

func TestComputeCurrentInterludeFindsLongGap(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(0, time.Second, false),
			newInterludeTestLine(7*time.Second, 8*time.Second, false),
		},
		Timeline: newTimelineState(),
	}
	lyrics.Timeline.CurrentTime = 2 * time.Second
	lyrics.Timeline.ScrollToIndex = 1

	got := computeCurrentInterlude(lyrics)
	if got == nil {
		t.Fatal("expected interlude for long gap")
	}
	if got.AnchorLineIndex != 0 {
		t.Fatalf("anchor = %d, want 0", got.AnchorLineIndex)
	}
	if got.StartTime != time.Second {
		t.Fatalf("start = %v, want gap start 1s", got.StartTime)
	}
	if got.EndTime != 6750*time.Millisecond {
		t.Fatalf("end = %v, want 6.75s", got.EndTime)
	}
}

func TestComputeCurrentInterludeBeforeFirstLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(6*time.Second, 7*time.Second, false),
		},
		Timeline: newTimelineState(),
	}
	lyrics.Timeline.CurrentTime = time.Second
	lyrics.Timeline.ScrollToIndex = 0

	got := computeCurrentInterlude(lyrics)
	if got == nil {
		t.Fatal("expected interlude before first line")
	}
	if got.AnchorLineIndex != -1 {
		t.Fatalf("anchor = %d, want -1", got.AnchorLineIndex)
	}
	if got.StartTime != 0 {
		t.Fatalf("start = %v, want gap start 0", got.StartTime)
	}
}

func TestComputeCurrentInterludeStopsBeforeNextLineLead(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(0, time.Second, false),
			newInterludeTestLine(7*time.Second, 8*time.Second, false),
		},
		Timeline: newTimelineState(),
	}
	lyrics.Timeline.CurrentTime = 6800 * time.Millisecond
	lyrics.Timeline.ScrollToIndex = 1

	if got := computeCurrentInterlude(lyrics); got != nil {
		t.Fatalf("interlude = %+v, want nil after lead cutoff", got)
	}
}

func TestComputeCurrentInterludeMarksNextDuet(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(0, time.Second, false),
			newInterludeTestLine(7*time.Second, 8*time.Second, true),
		},
		Timeline: newTimelineState(),
	}
	lyrics.Timeline.CurrentTime = 2 * time.Second
	lyrics.Timeline.ScrollToIndex = 1

	got := computeCurrentInterlude(lyrics)
	if got == nil {
		t.Fatal("expected interlude")
	}
	if !got.IsNextDuet {
		t.Fatal("expected next duet flag")
	}
}

func TestComputeLinePosYSpringParamsUsesStableParamsDuringInterlude(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			newInterludeTestLine(0, time.Second, false),
			newInterludeTestLine(7*time.Second, 8*time.Second, false),
		},
		Timeline: newTimelineState(),
	}

	params := computeLinePosYSpringParams(lyrics, 1, true)
	if params.Stiffness != defaultLinePosYSpringParams.Stiffness || params.Damping != defaultLinePosYSpringParams.Damping {
		t.Fatalf("params = %+v, want stable %+v", params, defaultLinePosYSpringParams)
	}
}
