package lyrics

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

	// 1️⃣ 行高计算
	metrics := face.Metrics()
	ascent := float64(metrics.HAscent) * fh
	descent := float64(metrics.HDescent) * fh
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	// 2️⃣ 测量函数（像素）
	measure := func(s string) float64 {
		w, _ := text.Measure(s, face, 0)
		return float64(w) * fh
	}

	// 3️⃣ 分词（中文按字，英文按词，空格保留）
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

	// 4️⃣ 自动换行（token 级）
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

	// 5️⃣ 生成布局
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

func AutoLayoutSyllable(
	layoutData [][]*LineSyllable,
	face text.Face,
	maxWidth float64,
	lineSpacing float64,
	fh float64,
	align text.Align,
) ([]Position, float64) {

	// 1️⃣ 计算行高（像素单位）
	metrics := face.Metrics()
	ascent := float64(metrics.HAscent) * fh
	descent := float64(metrics.HDescent) * fh
	lineHeight := ascent + descent
	lineStep := lineHeight + lineSpacing

	// 2️⃣ 文本测量函数（直接返回像素宽度）
	measure := func(s []*LineSyllable) float64 {
		var w float64
		for _, syllable := range s {
			// 关键：只计算可见字符宽度（忽略 \n）
			syllableStr := syllable.Syllable
			if len(syllableStr) > 0 && syllableStr[len(syllableStr)-1] == '\n' {
				syllableStr = syllableStr[:len(syllableStr)-1]
			}
			width, _ := text.Measure(syllableStr, face, 0)
			w += float64(width)
		}
		return w * fh
	}

	// 3️⃣ 自动换行（计算行级布局）
	var lines [][]*LineSyllable
	current := []*LineSyllable{}

	for _, word := range layoutData {
		// 临时处理：移除单词末尾的 \n（不影响原数据）
		wordWithoutBreak := make([]*LineSyllable, len(word))
		copy(wordWithoutBreak, word)
		if len(wordWithoutBreak) > 0 && len(wordWithoutBreak[len(wordWithoutBreak)-1].Syllable) > 0 {
			if wordWithoutBreak[len(wordWithoutBreak)-1].Syllable[len(wordWithoutBreak[len(wordWithoutBreak)-1].Syllable)-1] == '\n' {
				wordWithoutBreak[len(wordWithoutBreak)-1].Syllable = wordWithoutBreak[len(wordWithoutBreak)-1].Syllable[:len(wordWithoutBreak[len(wordWithoutBreak)-1].Syllable)-1]
			}
		}

		// 计算当前单词（处理后的）的宽度
		wordWidth := measure(wordWithoutBreak)
		if len(current) > 0 && measure(current)+wordWidth > maxWidth {
			lines = append(lines, current)
			current = wordWithoutBreak
		} else {
			current = append(current, wordWithoutBreak...)
		}
	}
	if len(current) > 0 {
		lines = append(lines, current)
	}

	// 4️⃣ 生成每个音节的精确位置（核心修改！）
	var syllablePositions []Position
	lineY := 0.0
	for i, line := range lines {
		// 计算当前行的起始X（根据对齐方式）
		lineWidth := measure(line)
		lineX := 0.0
		switch align {
		case text.AlignCenter:
			lineX = (maxWidth - lineWidth) / 2
		case text.AlignEnd:
			lineX = maxWidth - lineWidth
		}

		// 计算当前行每个音节的精确X坐标
		currentX := lineX
		for _, syllable := range line {
			// 计算当前音节的宽度（排除 \n）
			syllableStr := syllable.Syllable
			if len(syllableStr) > 0 && syllableStr[len(syllableStr)-1] == '\n' {
				syllableStr = syllableStr[:len(syllableStr)-1]
			}
			width, _ := text.Measure(syllableStr, face, 0)
			syllableWidth := float64(width) * fh

			// 添加到音节位置列表
			syllablePositions = append(syllablePositions, NewPosition(
				currentX,            // X坐标
				float64(i)*lineStep, // Y坐标（行高 * 行号）
				syllableWidth,       // 宽度
				lineHeight,          // 行高（用于渲染，实际绘制时可能不需要）
			))

			// 更新当前X（为下一个音节准备）
			currentX += syllableWidth
		}
		lineY = float64(i+1) * lineStep
	}

	// 5️⃣ 总高度
	totalHeight := lineY

	return syllablePositions, totalHeight
}
