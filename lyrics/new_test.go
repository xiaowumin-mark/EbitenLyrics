package lyrics

import (
	"testing"

	ttml "github.com/xiaowumin-mark/EbitenLyrics/ttml"
)

func TestMaxLineEndWithBackgroundKeepsMainWhenBackgroundEndsEarlier(t *testing.T) {
	line := ttml.LyricLine{
		EndTime: 1500,
		BGs: []ttml.LyricLine{
			{EndTime: 1100},
			{EndTime: 1400},
		},
	}

	if got := maxLineEndWithBackground(line); got != 1500 {
		t.Fatalf("maxLineEndWithBackground() = %d, want %d", got, 1500)
	}
}

func TestMaxLineEndWithBackgroundUsesBackgroundMaxEnd(t *testing.T) {
	line := ttml.LyricLine{
		EndTime: 1500,
		BGs: []ttml.LyricLine{
			{EndTime: 1600},
			{EndTime: 2200},
			{EndTime: 1800},
		},
	}

	if got := maxLineEndWithBackground(line); got != 2200 {
		t.Fatalf("maxLineEndWithBackground() = %d, want %d", got, 2200)
	}
}
