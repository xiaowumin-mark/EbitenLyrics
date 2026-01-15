package lyrics

import (
	"EbitenLyrics/anim"
	"math"
	"time"
)

// createFrames 生成卡拉OK逐字动画的关键帧
// targetIndex: 当前正在生成动画的单词索引
// blocks: 整行歌词的所有单词信息
// lineStartTime, lineEndTime: 整行歌词的起止时间（用于计算相对时间比例）
// fadeRatio: 渐变宽度比例
func createFrames(blocks []*SyllableElement, targetIndex int, lineStartTime, lineEndTime time.Duration, fadeRatio float64) []anim.Keyframe {
	if len(blocks) == 0 || targetIndex >= len(blocks) {
		return nil
	}

	targetBlock := blocks[targetIndex]

	// 获取尺寸信息 (假设 GetWidth 返回 float64)
	elWidth := targetBlock.SyllableImage.GetWidth()
	// 注意：TS代码中包含了 padding，如果你这里的 GetWidth 不含 padding，需要额外加上
	// elPadding := targetBlock.Padding

	elHeight := targetBlock.SyllableImage.GetHeight()
	fadeWidth := elHeight * fadeRatio

	// 1. 计算总持续时间 (对应 TS 的 totalFadeDuration)
	// 取 (最后一个单词结束时间) 和 (整行结束时间) 的较大值，确保动画覆盖整行
	maxEndTime := lineEndTime
	lastBlockEnd := blocks[len(blocks)-1].EndTime
	if lastBlockEnd > maxEndTime {
		maxEndTime = lastBlockEnd
	}
	totalDuration := float64(maxEndTime - lineStartTime)
	if totalDuration <= 0 {
		totalDuration = 1 // 防止除以0
	}

	// 2. 计算当前单词之前的累计宽度 (用于定位)
	widthBeforeSelf := 0.0
	for i := 0; i < targetIndex; i++ {
		widthBeforeSelf += blocks[i].SyllableImage.GetWidth()
	}
	// TS 逻辑：只要列表不为空，初始偏移就要加上 fadeWidth
	if len(blocks) > 0 {
		widthBeforeSelf += fadeWidth
	}

	// 3. 设定边界值
	// TS: minOffset = -(word.width + word.padding * 2 + fadeWidth)
	// 这里的 elWidth 应该对应 TS 的 width + padding*2
	minOffset := -(elWidth + fadeWidth)

	clampOffset := func(x float64) float64 {
		if x < minOffset {
			return minOffset
		}
		if x > 0 {
			return 0
		}
		return x
	}

	// 4. 初始化状态
	// TS: curPos = -widthBeforeSelf - word.width - word.padding - fadeWidth;
	currentPos := -widthBeforeSelf - elWidth - fadeWidth
	timeOffset := 0.0

	var frames []anim.Keyframe
	lastPos := currentPos
	lastTime := 0.0

	// 内部函数：生成关键帧
	pushFrame := func() {
		// 归一化时间 [0, 1]
		t := math.Max(0, math.Min(1, timeOffset))
		duration := t - lastTime
		moveOffset := currentPos - lastPos

		// d = duration / distance (即 1/速度)
		d := 0.0
		if math.Abs(moveOffset) > 1e-6 { // 避免浮点数除以0
			d = math.Abs(duration / moveOffset)
		}

		// 边界处理 1: 刚刚离开起始静止状态 (从 <minOffset 变为 >minOffset)
		if currentPos > minOffset && lastPos < minOffset {
			staticTime := math.Abs(lastPos-minOffset) * d
			frames = append(frames, anim.Keyframe{
				Offset: lastTime + staticTime,
				Values: []float64{clampOffset(lastPos)}, // 此时实际上就是 minOffset
				Ease:   anim.Linear,
			})
		}

		// 边界处理 2: 刚刚到达结束静止状态 (从 <0 变为 >0)
		if currentPos > 0 && lastPos < 0 {
			staticTime := math.Abs(lastPos) * d
			frames = append(frames, anim.Keyframe{
				Offset: lastTime + staticTime,
				Values: []float64{clampOffset(currentPos)}, // 此时实际上就是 0
				Ease:   anim.Linear,
			})
		}

		// 添加当前帧
		frames = append(frames, anim.Keyframe{
			Offset: t,
			Values: []float64{clampOffset(currentPos)},
			Ease:   anim.Linear,
		})

		lastPos = currentPos
		lastTime = t
	}

	// 添加初始帧
	pushFrame()

	// 5. 模拟时间轴流逝 (对应 TS 的 splittedWords.forEach)
	lastTimeStamp := 0.0 // 这里的单位与 totalDuration 保持一致 (float64)

	for j, block := range blocks {
		// --- 停顿阶段 (Gap) ---
		// 计算相对于行开始的时间点
		curBlockStartRelative := float64(block.StartTime - lineStartTime)
		staticDuration := curBlockStartRelative - lastTimeStamp

		timeOffset += staticDuration / totalDuration
		if staticDuration > 0 {
			pushFrame()
		}
		lastTimeStamp = curBlockStartRelative

		// --- 移动阶段 (Singing) ---
		fadeDuration := float64(block.EndTime - block.StartTime)
		timeOffset += fadeDuration / totalDuration

		// 核心位移累加
		currentPos += block.SyllableImage.GetWidth()

		// 特殊处理：第一个和最后一个单词的额外位移
		// 这是一个 trick，为了让渐变在开始和结束时看起来更自然
		if j == 0 {
			currentPos += fadeWidth * 1.5
		}
		if j == len(blocks)-1 {
			currentPos += fadeWidth * 0.5
		}

		if fadeDuration > 0 {
			pushFrame()
		}
		lastTimeStamp += fadeDuration
	}

	// 确保最后一帧是 1.0 (修补浮点数误差)
	if len(frames) > 0 {
		frames[len(frames)-1].Offset = 1.0
		// 确保最后一帧的位置也是归位的
		// frames[len(frames)-1].Values[0] = 0 // 视情况开启，通常 clampOffset 已经保证了
	}

	return frames
}
