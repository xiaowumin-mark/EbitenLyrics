package lyrics

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LayoutLine struct {
	Text string
	X    float64
	Y    float64
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
