package lyrics

import (
	"testing"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func testLine(start, end time.Duration) *Line {
	return NewLine(start, end, false, false, "", nil, ft.FontRequest{}, 32)
}

func TestTimelineEntersAndBuffersLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			testLine(0, time.Second),
			testLine(time.Second, 2*time.Second),
		},
		Timeline: newTimelineState(),
	}

	state := computePlayerTimeState(lyrics, 500*time.Millisecond)
	commit := commitPlayerTimeState(lyrics, 500*time.Millisecond, state)
	syncNowLyricsFromTimeline(lyrics)

	if !setHas(lyrics.Timeline.HotLines, 0) {
		t.Fatal("line 0 should be hot")
	}
	if !setHas(lyrics.Timeline.BufferedLines, 0) {
		t.Fatal("line 0 should be buffered")
	}
	if lyrics.Timeline.ScrollToIndex != 0 {
		t.Fatalf("scrollToIndex = %d, want 0", lyrics.Timeline.ScrollToIndex)
	}
	if len(commit.linesToEnable) != 1 || commit.linesToEnable[0] != 0 {
		t.Fatalf("linesToEnable = %v, want [0]", commit.linesToEnable)
	}
	if len(lyrics.nowLyrics) != 1 || lyrics.nowLyrics[0] != 0 {
		t.Fatalf("nowLyrics = %v, want [0]", lyrics.nowLyrics)
	}
}

func TestTimelineSwitchesBufferedLineOnNewHotLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			testLine(0, time.Second),
			testLine(time.Second, 2*time.Second),
		},
		Timeline: newTimelineState(),
	}

	commitPlayerTimeState(lyrics, 500*time.Millisecond, computePlayerTimeState(lyrics, 500*time.Millisecond))
	state := computePlayerTimeState(lyrics, 1200*time.Millisecond)
	commit := commitPlayerTimeState(lyrics, 1200*time.Millisecond, state)

	if setHas(lyrics.Timeline.HotLines, 0) || !setHas(lyrics.Timeline.HotLines, 1) {
		t.Fatalf("hot lines = %v, want only line 1", sortedSetIDs(lyrics.Timeline.HotLines))
	}
	if setHas(lyrics.Timeline.BufferedLines, 0) || !setHas(lyrics.Timeline.BufferedLines, 1) {
		t.Fatalf("buffered lines = %v, want only line 1", sortedSetIDs(lyrics.Timeline.BufferedLines))
	}
	if len(commit.linesToDisable) != 1 || commit.linesToDisable[0] != 0 {
		t.Fatalf("linesToDisable = %v, want [0]", commit.linesToDisable)
	}
	if len(commit.linesToEnable) != 1 || commit.linesToEnable[0] != 1 {
		t.Fatalf("linesToEnable = %v, want [1]", commit.linesToEnable)
	}
}

func TestTimelineSeekPicksNextLineWhenNoBufferedLine(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			testLine(0, time.Second),
			testLine(2*time.Second, 3*time.Second),
		},
		Timeline:       newTimelineState(),
		AnimateManager: anim.NewManager(false),
	}
	lyrics.Timeline.IsSeeking = true

	state := computePlayerTimeState(lyrics, 1500*time.Millisecond)
	commitPlayerTimeState(lyrics, 1500*time.Millisecond, state)

	if lyrics.Timeline.ScrollToIndex != 1 {
		t.Fatalf("seek scrollToIndex = %d, want 1", lyrics.Timeline.ScrollToIndex)
	}
}

func TestSetCurrentTimeSeekResetsSeekingFlag(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			testLine(0, time.Second),
			testLine(2*time.Second, 3*time.Second),
		},
		Timeline:       newTimelineState(),
		AnimateManager: anim.NewManager(false),
	}

	lyrics.SetCurrentTime(2500*time.Millisecond, true)
	if lyrics.Timeline.IsSeeking {
		t.Fatal("SetCurrentTime should clear transient seek flag after update")
	}
	if lyrics.Timeline.ScrollToIndex != 1 {
		t.Fatalf("scrollToIndex = %d, want 1", lyrics.Timeline.ScrollToIndex)
	}
}

func TestSetPlayingUpdatesTimeline(t *testing.T) {
	lyrics := &Lyrics{Timeline: newTimelineState()}
	lyrics.Pause()
	if lyrics.Timeline.IsPlaying {
		t.Fatal("Pause should set IsPlaying=false")
	}
	lyrics.Resume()
	if !lyrics.Timeline.IsPlaying {
		t.Fatal("Resume should set IsPlaying=true")
	}
}
