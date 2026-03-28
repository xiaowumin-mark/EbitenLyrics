package lyrics

// 文件说明：歌词自动布局算法。
// 主要职责：测量文本尺寸并给出换行与排版结果。

import (
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LayoutLine struct {
	Text string
	X    float64
	Y    float64
}

func AutoLayout(
	textStr string,
	face text.Face,
	maxWidth float64,
	lineSpacing float64,
	fh float64,
	align text.Align,
) ([]LayoutLine, float64) {
	if face == nil {
		return nil, 0
	}
	if maxWidth < 1 {
		maxWidth = 1
	}

	metrics := face.Metrics()
	ascent := float64(metrics.HAscent) * fh
	descent := float64(metrics.HDescent) * fh
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	measure := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return float64(w) * fh
	}

	var tokens []string
	var buf strings.Builder
	flushBuf := func() {
		if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}

	for _, r := range textStr {
		switch {
		case r == '\n':
			flushBuf()
			tokens = append(tokens, "\n")
		case r == ' ':
			flushBuf()
			tokens = append(tokens, " ")
		case r <= 0x7f && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			buf.WriteRune(r)
		default:
			flushBuf()
			tokens = append(tokens, string(r))
		}
	}
	flushBuf()

	var lines []string
	current := ""
	for _, tok := range tokens {
		if tok == "\n" {
			lines = append(lines, current)
			current = ""
			continue
		}

		if current == "" {
			current = tok
			continue
		}

		if measure(current+tok) > maxWidth {
			lines = append(lines, current)
			current = tok
		} else {
			current += tok
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	var layout []LayoutLine
	for i, line := range lines {
		w := measure(line)
		x := 0.0
		switch align {
		case text.AlignCenter:
			x = (maxWidth - w) / 2
		case text.AlignEnd:
			x = maxWidth - w
		}
		if x < 0 {
			x = 0
		}
		layout = append(layout, LayoutLine{
			Text: line,
			X:    x,
			Y:    float64(i) * lineStep,
		})
	}

	totalHeight := float64(len(lines)) * lineStep
	return layout, totalHeight
}

func normalizeSyllableText(s string) string {
	if strings.HasSuffix(s, "\n") {
		return strings.TrimSuffix(s, "\n")
	}
	return s
}

func AutoLayoutSyllable(
	layoutData [][]*LineSyllable,
	face text.Face,
	maxWidth float64,
	lineSpacing float64,
	fh float64,
	align text.Align,
) ([]Position, float64) {
	if face == nil {
		return nil, 0
	}
	if maxWidth < 1 {
		maxWidth = 1
	}

	metrics := face.Metrics()
	ascent := float64(metrics.HAscent) * fh
	descent := float64(metrics.HDescent) * fh
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	measureText := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return float64(w) * fh
	}
	measureWord := func(syllables []*LineSyllable) float64 {
		wordWidth := 0.0
		for _, syllable := range syllables {
			if syllable == nil {
				continue
			}
			wordWidth += measureText(normalizeSyllableText(syllable.Syllable))
		}
		return wordWidth
	}

	var lines [][]*LineSyllable
	current := make([]*LineSyllable, 0)
	currentWidth := 0.0

	for _, word := range layoutData {
		if len(word) == 0 {
			continue
		}

		wordWidth := measureWord(word)
		if len(current) > 0 && currentWidth+wordWidth > maxWidth {
			lines = append(lines, current)
			current = append(make([]*LineSyllable, 0, len(word)), word...)
			currentWidth = wordWidth
		} else {
			current = append(current, word...)
			currentWidth += wordWidth
		}
	}
	if len(current) > 0 {
		lines = append(lines, current)
	}

	var syllablePositions []Position
	for lineIndex, line := range lines {
		lineWidth := measureWord(line)
		lineX := 0.0
		switch align {
		case text.AlignCenter:
			lineX = (maxWidth - lineWidth) / 2
		case text.AlignEnd:
			lineX = maxWidth - lineWidth
		}

		currentX := lineX
		for _, syllable := range line {
			if syllable == nil {
				continue
			}
			syllableWidth := measureText(normalizeSyllableText(syllable.Syllable))
			syllablePositions = append(syllablePositions, NewPosition(
				currentX,
				float64(lineIndex)*lineStep,
				syllableWidth,
				lineHeight,
			))
			currentX += syllableWidth
		}
	}

	totalHeight := float64(len(lines)) * lineStep
	return syllablePositions, totalHeight
}
