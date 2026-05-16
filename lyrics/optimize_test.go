package lyrics

import (
	"testing"

	ttml "github.com/xiaowumin-mark/EbitenLyrics/ttml"
)

func TestOptimizeClonesAndResetsLineTimestamps(t *testing.T) {
	input := []ttml.LyricLine{{
		StartTime: 999,
		EndTime:   1000,
		Words: []ttml.LyricWord{
			{StartTime: 100, EndTime: 200, Word: "a"},
			{StartTime: 300, EndTime: 500, Word: "b"},
		},
	}}

	got := optimizeLyricLines(input)
	if got[0].StartTime != 0 || got[0].EndTime != 500 {
		t.Fatalf("optimized range = %d..%d, want advanced 0..500", got[0].StartTime, got[0].EndTime)
	}
	if input[0].StartTime != 999 || input[0].EndTime != 1000 {
		t.Fatal("optimize should not mutate input")
	}
}

func TestOptimizePreservesBoundarySpaces(t *testing.T) {
	input := []ttml.LyricLine{{
		Words: []ttml.LyricWord{
			{StartTime: 0, EndTime: 100, Word: "hello "},
			{StartTime: 100, EndTime: 200, Word: " world"},
		},
	}}

	got := optimizeLyricLines(input)
	if got[0].Words[0].Word != "hello " {
		t.Fatalf("first word = %q, want trailing space preserved", got[0].Words[0].Word)
	}
	if got[0].Words[1].Word != " world" {
		t.Fatalf("second word = %q, want leading space preserved", got[0].Words[1].Word)
	}
}

func TestOptimizeSyncsBackgroundTiming(t *testing.T) {
	input := []ttml.LyricLine{{
		StartTime: 1000,
		EndTime:   2000,
		Words:     []ttml.LyricWord{{StartTime: 1000, EndTime: 2000, Word: "main"}},
		BGs: []ttml.LyricLine{{
			StartTime: 800,
			EndTime:   2300,
			Words:     []ttml.LyricWord{{StartTime: 800, EndTime: 2300, Word: "bg"}},
		}},
	}}

	got := optimizeLyricLines(input)
	if got[0].StartTime != 200 || got[0].EndTime != 2300 {
		t.Fatalf("main range = %d..%d, want 200..2300 after sync and advance", got[0].StartTime, got[0].EndTime)
	}
	if got[0].BGs[0].StartTime != got[0].StartTime || got[0].BGs[0].EndTime != got[0].EndTime {
		t.Fatalf("background range = %d..%d, want synced to main %d..%d", got[0].BGs[0].StartTime, got[0].BGs[0].EndTime, got[0].StartTime, got[0].EndTime)
	}
}

func TestOptimizeCleansUnintentionalOverlap(t *testing.T) {
	input := []ttml.LyricLine{
		{StartTime: 0, EndTime: 1100, Words: []ttml.LyricWord{{StartTime: 0, EndTime: 1100, Word: "a"}}},
		{StartTime: 1000, EndTime: 2000, Words: []ttml.LyricWord{{StartTime: 1000, EndTime: 2000, Word: "b"}}},
	}

	got := optimizeLyricLines(input)
	if got[0].EndTime != got[1].StartTime {
		t.Fatalf("first end = %d, want clipped to next start %d", got[0].EndTime, got[1].StartTime)
	}
}
