package lyrics

import (
	"strings"
	"testing"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	layoutBenchMaxWidth    = 860
	layoutBenchLineSpacing = 12
	layoutBenchFontSize    = 48
)

func benchmarkFace(tb testing.TB) text.Face {
	tb.Helper()
	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	face, err := manager.GetFaceForText(req, layoutBenchFontSize, "We stay in motion 追逐夏夜的光，直到星星落下。")
	if err != nil {
		tb.Fatalf("get face failed: %v", err)
	}
	return face
}

func benchmarkSyllables(tb testing.TB, face text.Face) [][]*LineSyllable {
	tb.Helper()
	words := []string{
		"We", " ", "stay", " ", "in", " ", "motion", " ",
		"追", "逐", "夏", "夜", "的", "光", "，",
		"until", " ", "the", " ", "stars", " ", "fall", ".",
	}
	group := make([]*LineSyllable, 0, len(words))
	for _, word := range words {
		width, height := text.Measure(word, face, 1.0)
		group = append(group, &LineSyllable{
			Syllable: word,
			Elements: []*SyllableElement{
				{
					Text: word,
					SyllableImage: &SyllableImage{
						Text:   word,
						Width:  width,
						Height: height,
					},
				},
			},
		})
	}
	return [][]*LineSyllable{group}
}

func BenchmarkAutoLayoutText(b *testing.B) {
	face := benchmarkFace(b)
	content := strings.Repeat("We stay in motion 追逐夏夜的光，直到星星落下。 ", 4)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines, h := AutoLayout(content, face, layoutBenchMaxWidth, layoutBenchLineSpacing, 1, text.AlignStart)
		if len(lines) == 0 || h <= 0 {
			b.Fatal("layout returned empty result")
		}
	}
}

func BenchmarkAutoLayoutSyllableCachedWidths(b *testing.B) {
	face := benchmarkFace(b)
	layoutData := benchmarkSyllables(b, face)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		positions, h := AutoLayoutSyllable(layoutData, face, layoutBenchMaxWidth, layoutBenchLineSpacing, 1, text.AlignStart)
		if len(positions) == 0 || h <= 0 {
			b.Fatal("layout returned empty result")
		}
	}
}
