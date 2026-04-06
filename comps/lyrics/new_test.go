package LyricsComponent

import (
	"testing"
	"time"
)

func TestCrossfadeProgressClamps(t *testing.T) {
	start := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	duration := 280 * time.Millisecond

	if got := crossfadeProgress(start, duration, start.Add(-10*time.Millisecond)); got != 0 {
		t.Fatalf("progress before start = %v, want 0", got)
	}
	if got := crossfadeProgress(start, duration, start.Add(140*time.Millisecond)); got != 0.5 {
		t.Fatalf("progress mid fade = %v, want 0.5", got)
	}
	if got := crossfadeProgress(start, duration, start.Add(500*time.Millisecond)); got != 1 {
		t.Fatalf("progress after end = %v, want 1", got)
	}
}

func TestCrossfadeProgressHandlesMissingStartOrDuration(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)

	if got := crossfadeProgress(time.Time{}, 280*time.Millisecond, now); got != 1 {
		t.Fatalf("progress with zero start = %v, want 1", got)
	}
	if got := crossfadeProgress(now, 0, now); got != 1 {
		t.Fatalf("progress with zero duration = %v, want 1", got)
	}
}
