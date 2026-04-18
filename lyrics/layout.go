package lyrics

import (
	"math"
	"strings"
	"unicode"

	"github.com/xiaowumin-mark/EbitenLyrics/lp"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LayoutLine struct {
	Text string
	X    float64
	Y    float64
}

type layoutWordItem struct {
	text           string
	width          float64
	syllables      []*LineSyllable
	syllableWidths []float64
}

type layoutTextItem struct {
	text  string
	width float64
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
	ascent := lp.FromLP(float64(metrics.HAscent) * fh)
	descent := lp.FromLP(float64(metrics.HDescent) * fh)
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	measure := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return lp.FromLP(float64(w) * fh)
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

func AutoLayoutSmart(
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
	ascent := lp.FromLP(float64(metrics.HAscent) * fh)
	descent := lp.FromLP(float64(metrics.HDescent) * fh)
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	measure := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return lp.FromLP(float64(w) * fh)
	}

	paragraphs := tokenizeTextParagraphsForLayout(textStr)
	layout := make([]LayoutLine, 0, len(paragraphs))
	lineIndex := 0

	for _, paragraph := range paragraphs {
		if len(paragraph) == 0 {
			layout = append(layout, LayoutLine{
				Text: "",
				X:    0,
				Y:    float64(lineIndex) * lineStep,
			})
			lineIndex++
			continue
		}

		items := buildLayoutTextItems(paragraph, measure)
		if len(items) == 0 {
			layout = append(layout, LayoutLine{
				Text: "",
				X:    0,
				Y:    float64(lineIndex) * lineStep,
			})
			lineIndex++
			continue
		}

		lineRanges := balancedTextLineRanges(items, maxWidth)
		for _, rng := range lineRanges {
			start := rng[0]
			end := rng[1]
			trimStart, trimEnd := trimmedTextLayoutBounds(items, start, end)
			lineText := ""
			lineWidth := 0.0
			if trimStart < trimEnd {
				lineText = joinLayoutTextItems(items[trimStart:trimEnd])
				lineWidth = measure(lineText)
			}

			lineX := 0.0
			switch align {
			case text.AlignCenter:
				lineX = (maxWidth - lineWidth) / 2
			case text.AlignEnd:
				lineX = maxWidth - lineWidth
			}
			if lineX < 0 {
				lineX = 0
			}

			layout = append(layout, LayoutLine{
				Text: lineText,
				X:    lineX,
				Y:    float64(lineIndex) * lineStep,
			})
			lineIndex++
		}
	}

	totalHeight := float64(lineIndex) * lineStep
	return layout, totalHeight
}

func normalizeSyllableText(s string) string {
	if strings.HasSuffix(s, "\n") {
		return strings.TrimSuffix(s, "\n")
	}
	return s
}

func tokenizeTextParagraphsForLayout(textStr string) [][]string {
	var paragraphs [][]string
	var current []string
	var asciiBuf strings.Builder
	var cjkBuf strings.Builder

	flushASCIIWord := func() {
		if asciiBuf.Len() == 0 {
			return
		}
		current = append(current, splitTokenByRuneLimit(asciiBuf.String(), lineModeMaxTokenRunes)...)
		asciiBuf.Reset()
	}
	flushCJKRun := func() {
		if cjkBuf.Len() == 0 {
			return
		}
		current = append(current, splitCJKRunForLayout(cjkBuf.String())...)
		cjkBuf.Reset()
	}
	flushAll := func() {
		flushASCIIWord()
		flushCJKRun()
	}
	flushParagraph := func() {
		flushAll()
		paragraphs = append(paragraphs, current)
		current = nil
	}

	for _, r := range textStr {
		switch {
		case r == '\n':
			flushParagraph()
		case unicode.IsSpace(r):
			flushAll()
			current = append(current, " ")
		case r <= 0x7f && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			flushCJKRun()
			asciiBuf.WriteRune(r)
		case r <= 0x7f && (r == '\'' || r == '-') && asciiBuf.Len() > 0:
			flushCJKRun()
			asciiBuf.WriteRune(r)
		case isCJKLayoutRune(r):
			flushASCIIWord()
			cjkBuf.WriteRune(r)
		default:
			flushAll()
			current = append(current, string(r))
		}
	}
	flushParagraph()

	return paragraphs
}

func syllableCachedWidth(syllable *LineSyllable, fallback func(string) float64) float64 {
	if syllable == nil {
		return 0
	}
	width := 0.0
	hasImageWidth := false
	for _, element := range syllable.Elements {
		if element == nil || element.SyllableImage == nil {
			continue
		}
		width += element.SyllableImage.GetWidth()
		hasImageWidth = true
	}
	if hasImageWidth {
		return width
	}
	return fallback(normalizeSyllableText(syllable.Syllable))
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
	ascent := lp.FromLP(float64(metrics.HAscent) * fh)
	descent := lp.FromLP(float64(metrics.HDescent) * fh)
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	measureText := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return lp.FromLP(float64(w) * fh)
	}

	items := buildLayoutWordItems(layoutData, measureText)
	if len(items) == 0 {
		return nil, 0
	}

	prefix := make([]float64, len(items)+1)
	for i, item := range items {
		prefix[i+1] = prefix[i] + item.width
	}

	totalWidth := trimmedLayoutWidth(items, prefix, 0, len(items))
	if totalWidth <= 0 {
		totalWidth = maxWidth
	}
	minLines := int(math.Ceil(totalWidth / maxWidth))
	if minLines < 1 {
		minLines = 1
	}
	targetWidth := totalWidth / float64(minLines)
	if targetWidth > maxWidth {
		targetWidth = maxWidth
	}

	bestCost := make([]float64, len(items)+1)
	bestNext := make([]int, len(items)+1)
	visited := make([]bool, len(items)+1)

	var solve func(int) float64
	solve = func(start int) float64 {
		if nextContentIndex(items, start) >= len(items) {
			return 0
		}
		if visited[start] {
			return bestCost[start]
		}
		visited[start] = true

		bestCost[start] = math.Inf(1)
		bestNext[start] = -1

		for end := start + 1; end <= len(items); end++ {
			width := trimmedLayoutWidth(items, prefix, start, end)
			if width <= 0 {
				continue
			}
			if width > maxWidth {
				break
			}

			cost := layoutLinePenalty(width, targetWidth, maxWidth, end == len(items))
			if end < len(items) {
				cost += layoutBoundaryPenalty(items, end)
			}
			cost += solve(end)

			if cost < bestCost[start] {
				bestCost[start] = cost
				bestNext[start] = end
			}
		}

		if bestNext[start] == -1 {
			fallbackEnd := start + 1
			if fallbackEnd > len(items) {
				fallbackEnd = len(items)
			}
			bestNext[start] = fallbackEnd
			width := trimmedLayoutWidth(items, prefix, start, fallbackEnd)
			bestCost[start] = layoutLinePenalty(width, targetWidth, maxWidth, fallbackEnd == len(items)) + solve(fallbackEnd)
		}

		return bestCost[start]
	}

	solve(0)

	lineRanges := make([][2]int, 0, minLines)
	for start := 0; nextContentIndex(items, start) < len(items); {
		end := bestNext[start]
		if end <= start {
			end = start + 1
		}
		lineRanges = append(lineRanges, [2]int{start, end})
		start = end
	}

	totalSyllables := 0
	for _, item := range items {
		totalSyllables += len(item.syllables)
	}

	positions := make([]Position, 0, totalSyllables)
	for lineIndex, rng := range lineRanges {
		start := rng[0]
		end := rng[1]
		trimStart, trimEnd := trimmedLayoutBounds(items, start, end)
		lineWidth := 0.0
		if trimStart < trimEnd {
			lineWidth = prefix[trimEnd] - prefix[trimStart]
		}

		lineX := 0.0
		switch align {
		case text.AlignCenter:
			lineX = (maxWidth - lineWidth) / 2
		case text.AlignEnd:
			lineX = maxWidth - lineWidth
		}
		if lineX < 0 {
			lineX = 0
		}

		currentX := lineX
		for idx := start; idx < end; idx++ {
			visible := idx >= trimStart && idx < trimEnd
			wordX := currentX
			for _, syllableWidth := range items[idx].syllableWidths {
				positions = append(positions, NewPosition(
					wordX,
					float64(lineIndex)*lineStep,
					syllableWidth,
					lineHeight,
				))
				if visible {
					wordX += syllableWidth
				}
			}
			if visible {
				currentX = wordX
			}
		}
	}

	totalHeight := float64(len(lineRanges)) * lineStep
	return positions, totalHeight
}

func buildLayoutWordItems(layoutData [][]*LineSyllable, fallback func(string) float64) []layoutWordItem {
	items := make([]layoutWordItem, 0, len(layoutData))
	for _, group := range layoutData {
		if len(group) == 0 {
			continue
		}

		textBuilder := strings.Builder{}
		width := 0.0
		syllableWidths := make([]float64, 0, len(group))
		for _, syllable := range group {
			if syllable == nil {
				continue
			}
			text := normalizeSyllableText(syllable.Syllable)
			syllableWidth := syllableCachedWidth(syllable, fallback)
			textBuilder.WriteString(text)
			width += syllableWidth
			syllableWidths = append(syllableWidths, syllableWidth)
		}
		if len(syllableWidths) == 0 {
			continue
		}
		items = append(items, layoutWordItem{
			text:           textBuilder.String(),
			width:          width,
			syllables:      group,
			syllableWidths: syllableWidths,
		})
	}
	return items
}

func buildLayoutTextItems(tokens []string, measure func(string) float64) []layoutTextItem {
	groupIndexes := splitLyricsIntoGroupsSmart(tokens, true)
	items := make([]layoutTextItem, 0, len(groupIndexes))
	for _, group := range groupIndexes {
		var textBuilder strings.Builder
		for _, idx := range group {
			if idx < 0 || idx >= len(tokens) {
				continue
			}
			textBuilder.WriteString(tokens[idx])
		}
		textValue := textBuilder.String()
		if textValue == "" {
			continue
		}
		items = append(items, layoutTextItem{
			text:  textValue,
			width: measure(textValue),
		})
	}
	return items
}

func balancedTextLineRanges(items []layoutTextItem, maxWidth float64) [][2]int {
	prefix := make([]float64, len(items)+1)
	for i, item := range items {
		prefix[i+1] = prefix[i] + item.width
	}

	totalWidth := trimmedTextLayoutWidth(items, prefix, 0, len(items))
	if totalWidth <= 0 {
		totalWidth = maxWidth
	}
	minLines := int(math.Ceil(totalWidth / maxWidth))
	if minLines < 1 {
		minLines = 1
	}
	targetWidth := totalWidth / float64(minLines)
	if targetWidth > maxWidth {
		targetWidth = maxWidth
	}

	bestCost := make([]float64, len(items)+1)
	bestNext := make([]int, len(items)+1)
	visited := make([]bool, len(items)+1)

	var solve func(int) float64
	solve = func(start int) float64 {
		if nextTextContentIndex(items, start) >= len(items) {
			return 0
		}
		if visited[start] {
			return bestCost[start]
		}
		visited[start] = true
		bestCost[start] = math.Inf(1)
		bestNext[start] = -1

		for end := start + 1; end <= len(items); end++ {
			width := trimmedTextLayoutWidth(items, prefix, start, end)
			if width <= 0 {
				continue
			}
			if width > maxWidth {
				break
			}

			cost := layoutLinePenalty(width, targetWidth, maxWidth, end == len(items))
			if end < len(items) {
				cost += layoutTextBoundaryPenalty(items, end)
			}
			cost += solve(end)

			if cost < bestCost[start] {
				bestCost[start] = cost
				bestNext[start] = end
			}
		}

		if bestNext[start] == -1 {
			fallbackEnd := start + 1
			if fallbackEnd > len(items) {
				fallbackEnd = len(items)
			}
			bestNext[start] = fallbackEnd
			width := trimmedTextLayoutWidth(items, prefix, start, fallbackEnd)
			bestCost[start] = layoutLinePenalty(width, targetWidth, maxWidth, fallbackEnd == len(items)) + solve(fallbackEnd)
		}
		return bestCost[start]
	}

	solve(0)

	lineRanges := make([][2]int, 0, minLines)
	for start := 0; nextTextContentIndex(items, start) < len(items); {
		end := bestNext[start]
		if end <= start {
			end = start + 1
		}
		lineRanges = append(lineRanges, [2]int{start, end})
		start = end
	}
	return lineRanges
}

func isLayoutSpaceText(s string) bool {
	return strings.TrimSpace(s) == ""
}

func isTextLayoutSpace(item layoutTextItem) bool {
	return strings.TrimSpace(item.text) == ""
}

func nextTextContentIndex(items []layoutTextItem, start int) int {
	for i := start; i < len(items); i++ {
		if !isTextLayoutSpace(items[i]) {
			return i
		}
	}
	return len(items)
}

func previousTextContentIndex(items []layoutTextItem, start int) int {
	for i := start; i >= 0; i-- {
		if !isTextLayoutSpace(items[i]) {
			return i
		}
	}
	return -1
}

func trimmedTextLayoutBounds(items []layoutTextItem, start, end int) (int, int) {
	for start < end && isTextLayoutSpace(items[start]) {
		start++
	}
	for end > start && isTextLayoutSpace(items[end-1]) {
		end--
	}
	return start, end
}

func trimmedTextLayoutWidth(items []layoutTextItem, prefix []float64, start, end int) float64 {
	trimStart, trimEnd := trimmedTextLayoutBounds(items, start, end)
	if trimStart >= trimEnd {
		return 0
	}
	return prefix[trimEnd] - prefix[trimStart]
}

func layoutTextBoundaryPenalty(items []layoutTextItem, breakAt int) float64 {
	prevIdx := previousTextContentIndex(items, breakAt-1)
	nextIdx := nextTextContentIndex(items, breakAt)
	if prevIdx < 0 || nextIdx >= len(items) {
		return 0
	}

	prev := strings.TrimSpace(items[prevIdx].text)
	next := strings.TrimSpace(items[nextIdx].text)
	if prev == "" || next == "" {
		return 0
	}

	penalty := 0.0
	if endsWithStrongBreakPunctuation(prev) {
		penalty -= 0.28
	} else if endsWithSoftBreakPunctuation(prev) {
		penalty -= 0.12
	}
	if endsWithOpeningPunctuation(prev) {
		penalty += 0.75
	}
	if startsWithClosingPunctuation(next) {
		penalty += 0.75
	}
	if isWeakEnglishBreakWord(lastASCIIWord(prev)) {
		penalty += 0.24
	}
	if isWeakEnglishBreakWord(firstASCIIWord(next)) {
		penalty += 0.18
	}
	if isWeakChineseBreakEnding(lastNonSpaceRune(prev)) {
		penalty += 0.12
	}
	if isWeakChineseBreakStart(firstNonSpaceRune(next)) {
		penalty += 0.12
	}
	return penalty
}

func joinLayoutTextItems(items []layoutTextItem) string {
	var textBuilder strings.Builder
	for _, item := range items {
		textBuilder.WriteString(item.text)
	}
	return textBuilder.String()
}

func nextContentIndex(items []layoutWordItem, start int) int {
	for i := start; i < len(items); i++ {
		if !isLayoutSpaceText(items[i].text) {
			return i
		}
	}
	return len(items)
}

func previousContentIndex(items []layoutWordItem, start int) int {
	for i := start; i >= 0; i-- {
		if !isLayoutSpaceText(items[i].text) {
			return i
		}
	}
	return -1
}

func trimmedLayoutBounds(items []layoutWordItem, start, end int) (int, int) {
	for start < end && isLayoutSpaceText(items[start].text) {
		start++
	}
	for end > start && isLayoutSpaceText(items[end-1].text) {
		end--
	}
	return start, end
}

func trimmedLayoutWidth(items []layoutWordItem, prefix []float64, start, end int) float64 {
	trimStart, trimEnd := trimmedLayoutBounds(items, start, end)
	if trimStart >= trimEnd {
		return 0
	}
	return prefix[trimEnd] - prefix[trimStart]
}

func layoutLinePenalty(width, targetWidth, maxWidth float64, isLast bool) float64 {
	if width <= 0 {
		return math.Inf(1)
	}

	diff := math.Abs(targetWidth-width) / maxWidth
	cost := diff * diff

	if width > maxWidth {
		overflow := (width - maxWidth) / maxWidth
		cost += 16 + overflow*overflow*16
	}

	if targetWidth > 0 {
		if !isLast && width < targetWidth*0.6 {
			shortage := (targetWidth*0.6 - width) / targetWidth
			cost += 0.45 + shortage*shortage
		}
		if isLast && width < targetWidth*0.45 {
			shortage := (targetWidth*0.45 - width) / targetWidth
			cost += 0.18 + shortage*shortage*0.6
		}
	}

	return cost
}

func layoutBoundaryPenalty(items []layoutWordItem, breakAt int) float64 {
	prevIdx := previousContentIndex(items, breakAt-1)
	nextIdx := nextContentIndex(items, breakAt)
	if prevIdx < 0 || nextIdx >= len(items) {
		return 0
	}

	prev := strings.TrimSpace(items[prevIdx].text)
	next := strings.TrimSpace(items[nextIdx].text)
	if prev == "" || next == "" {
		return 0
	}

	penalty := 0.0
	if endsWithStrongBreakPunctuation(prev) {
		penalty -= 0.28
	} else if endsWithSoftBreakPunctuation(prev) {
		penalty -= 0.12
	}
	if endsWithOpeningPunctuation(prev) {
		penalty += 0.75
	}
	if startsWithClosingPunctuation(next) {
		penalty += 0.75
	}
	if isWeakEnglishBreakWord(lastASCIIWord(prev)) {
		penalty += 0.24
	}
	if isWeakEnglishBreakWord(firstASCIIWord(next)) {
		penalty += 0.18
	}
	if isWeakChineseBreakEnding(lastNonSpaceRune(prev)) {
		penalty += 0.12
	}
	if isWeakChineseBreakStart(firstNonSpaceRune(next)) {
		penalty += 0.12
	}
	return penalty
}

func endsWithStrongBreakPunctuation(s string) bool {
	r := lastNonSpaceRune(s)
	return strings.ContainsRune(".,!?;:，。！？；：、/", r)
}

func endsWithSoftBreakPunctuation(s string) bool {
	r := lastNonSpaceRune(s)
	return strings.ContainsRune(")-]）】》」』”", r)
}

func endsWithOpeningPunctuation(s string) bool {
	r := lastNonSpaceRune(s)
	return strings.ContainsRune("([（【《「『“", r)
}

func startsWithClosingPunctuation(s string) bool {
	r := firstNonSpaceRune(s)
	return strings.ContainsRune(".,!?;:)]}，。！？；：、）】》」』”", r)
}

func firstNonSpaceRune(s string) rune {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return r
		}
	}
	return 0
}

func lastNonSpaceRune(s string) rune {
	var last rune
	for _, r := range s {
		if !unicode.IsSpace(r) {
			last = r
		}
	}
	return last
}

func firstASCIIWord(s string) string {
	fields := splitASCIIWords(strings.ToLower(s))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func lastASCIIWord(s string) string {
	fields := splitASCIIWords(strings.ToLower(s))
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func splitASCIIWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		if r > 0x7f {
			return true
		}
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-')
	})
}

func isWeakEnglishBreakWord(word string) bool {
	switch word {
	case "a", "an", "and", "as", "at", "but", "by", "for", "from", "in", "into", "of", "on", "or", "the", "to", "with":
		return true
	}
	return false
}

func isWeakChineseBreakEnding(r rune) bool {
	return strings.ContainsRune("的了呢吗吧啊呀嘛着过和与及并又也就才还都把被给让向从对在", r)
}

func isWeakChineseBreakStart(r rune) bool {
	return strings.ContainsRune("的了呢吗吧啊呀嘛着过和与及并又也就才还都把被给让向从对在", r)
}
