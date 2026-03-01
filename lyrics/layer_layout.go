package lyrics

import (
	"EbitenLyrics/ttml"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (l *Line) Layout() {
	lineLayoutLayer.LayoutLine(l)
}

func (l *Line) GenerateTSImage() {
	lineLayoutLayer.GenerateLineTranslateImage(l)
}

func (l *Line) SetFont(font *text.GoTextFaceSource) {
	lineLayoutLayer.SetLineFont(l, font)
}

func (l *Line) SetFontSize(fontsize float64) {
	lineLayoutLayer.SetLineFontSize(l, fontsize)
}

func (l *Line) Resize(width float64) {
	lineLayoutLayer.ResizeLine(l, width)
}

func (l *Lyrics) Resize(w float64) {
	lineLayoutLayer.ResizeLyrics(l, w)
}

func (LayoutLayer) LayoutLine(l *Line) {
	if l == nil {
		return
	}

	w := l.GetPosition().GetW()
	face := l.activeFace()
	if face == nil {
		return
	}

	l.Participle = SplitBySpace(l, true)

	var grouped [][]*LineSyllable
	var orderedIndexes []int
	for _, p := range l.Participle {
		var syllables []*LineSyllable
		for _, idx := range p {
			if idx < 0 || idx >= len(l.Syllables) {
				continue
			}
			syllables = append(syllables, l.Syllables[idx])
			orderedIndexes = append(orderedIndexes, idx)
		}
		if len(syllables) > 0 {
			grouped = append(grouped, syllables)
		}
	}

	align := text.AlignStart
	if l.IsDuet {
		align = text.AlignEnd
	}

	maxWidth := w - l.Padding*2
	if maxWidth < 1 {
		maxWidth = 1
	}

	positions, height := AutoLayoutSyllable(grouped, face, maxWidth, l.lineHeight, 1, align)
	height += l.Padding * 2

	for posIdx, pos := range positions {
		if posIdx >= len(orderedIndexes) {
			break
		}
		syllableIndex := orderedIndexes[posIdx]
		if syllableIndex < 0 || syllableIndex >= len(l.Syllables) {
			continue
		}
		syll := l.Syllables[syllableIndex]

		pos.SetX(pos.GetX() + l.Padding)
		pos.SetY(pos.GetY() + l.Padding)
		lastX := pos.GetX()
		for _, element := range syll.Elements {
			element.GetPosition().SetX(lastX)
			element.GetPosition().SetY(pos.GetY())
			if element.SyllableImage != nil {
				lastX += element.SyllableImage.GetWidth()
			}
		}
	}

	if l.TranslatedText != "" && l.TranslateImageH == 0 {
		lineLayoutLayer.GenerateLineTranslateImage(l)
	}

	l.GetPosition().SetH(height + l.TranslateImageH)
	l.GetPosition().SetOriginY(l.GetPosition().GetH() / 2)
	if l.IsDuet {
		l.GetPosition().SetOriginX(l.GetPosition().GetW())
	} else {
		l.GetPosition().SetOriginX(0)
	}
	if l.IsBackground {
		l.GetPosition().SetOriginY(l.GetPosition().GetH())
	}
}

func (LayoutLayer) GenerateLineTranslateImage(l *Line) {
	if l == nil {
		return
	}
	if strings.TrimSpace(l.TranslatedText) == "" || l.Font == nil {
		l.TranslateImageW = 0
		l.TranslateImageH = 0
		if l.TranslateImage != nil {
			l.TranslateImage.Deallocate()
			l.TranslateImage = nil
		}
		return
	}

	align := text.AlignStart
	if l.IsDuet {
		align = text.AlignEnd
	}

	maxWidth := l.GetPosition().GetW() - l.Padding*2
	if maxWidth < 1 {
		maxWidth = 1
	}

	positions, h := AutoLayout(
		l.TranslatedText,
		&text.GoTextFace{Source: l.Font, Size: l.fontsize / 2},
		maxWidth,
		l.lineHeight,
		1,
		align,
	)
	l.TranslateImageW = maxWidth
	l.TranslateImageH = h

	if !l.isShow {
		if l.TranslateImage != nil {
			l.TranslateImage.Deallocate()
			l.TranslateImage = nil
		}
		return
	}

	if l.TranslateImage != nil {
		l.TranslateImage.Deallocate()
	}
	l.TranslateImage = ebiten.NewImage(safeImageLength(maxWidth), safeImageLength(h))
	for _, pos := range positions {
		op := &text.DrawOptions{}
		op.GeoM.Translate(pos.X, pos.Y)
		op.ColorScale.ScaleWithColor(color.White)
		op.ColorScale.ScaleAlpha(0.4)
		text.Draw(l.TranslateImage, pos.Text, &text.GoTextFace{Source: l.Font, Size: l.fontsize / 2}, op)
	}
}

func (LayoutLayer) SetLineFont(l *Line, font *text.GoTextFaceSource) {
	if l == nil {
		return
	}
	l.Font = font
	face := l.activeFace()
	if face == nil {
		return
	}
	for _, syllable := range l.Syllables {
		syllable.SetFont(face)
	}
	lineLayoutLayer.GenerateLineTranslateImage(l)
	lineLayoutLayer.LayoutLine(l)
	lineRendererLayer.RecreateLineImage(l)
}

func (LayoutLayer) SetLineFontSize(l *Line, fontsize float64) {
	if l == nil || fontsize <= 0 {
		return
	}
	l.fontsize = fontsize
	face := l.activeFace()
	if face == nil {
		return
	}
	for _, syllable := range l.Syllables {
		syllable.SetFont(face)
	}
	lineLayoutLayer.GenerateLineTranslateImage(l)
	lineLayoutLayer.LayoutLine(l)
	lineRendererLayer.RecreateLineImage(l)
}

func (LayoutLayer) ResizeLine(l *Line, width float64) {
	if l == nil || width <= 0 {
		return
	}
	l.GetPosition().SetW(width * 0.8)
	if l.IsDuet {
		l.GetPosition().SetX(width - l.GetPosition().GetW())
	}
	lineLayoutLayer.GenerateLineTranslateImage(l)
	lineLayoutLayer.LayoutLine(l)
	if l.isShow {
		lineRendererLayer.RecreateLineImage(l)
	}
	for _, bgline := range l.BackgroundLines {
		lineLayoutLayer.ResizeLine(bgline, width)
	}
}

func (LayoutLayer) ResizeLyrics(l *Lyrics, w float64) {
	if l == nil {
		return
	}
	for _, line := range l.Lines {
		lineLayoutLayer.ResizeLine(line, w)
	}
}

func splitBySpaceWords(words []string, includeSpaces bool) [][]int {
	var result [][]int
	var currentWordIndices []int

	flush := func() {
		if len(currentWordIndices) > 0 {
			result = append(result, currentWordIndices)
			currentWordIndices = nil
		}
	}

	for index, word := range words {
		if strings.TrimSpace(word) == "" {
			flush()
			if includeSpaces {
				result = append(result, []int{index})
			}
			continue
		}

		if isSingleChineseChar(word) {
			flush()
			result = append(result, []int{index})
			continue
		}

		currentWordIndices = append(currentWordIndices, index)
		if strings.HasSuffix(word, " ") {
			flush()
		}
	}

	flush()
	return result
}

func SplitBySpace(line *Line, includeSpaces bool) [][]int {
	words := make([]string, 0, len(line.Syllables))
	for _, element := range line.Syllables {
		words = append(words, element.Syllable)
	}
	return splitBySpaceWords(words, includeSpaces)
}

func SplitBySpaceTTML(line []ttml.LyricWord, includeSpaces bool) [][]int {
	words := make([]string, 0, len(line))
	for _, element := range line {
		words = append(words, element.Word)
	}
	return splitBySpaceWords(words, includeSpaces)
}

func isChineseChar(c rune) bool {
	return c >= '\u4e00' && c <= '\u9fff'
}

func isSingleChineseChar(s string) bool {
	runes := []rune(s)
	if len(runes) != 1 {
		return false
	}
	return isChineseChar(runes[0])
}
