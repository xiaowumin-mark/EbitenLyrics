package lyrics

import (
	"EbitenLyrics/anim"
	"math"
	"time"
)

func createFrames(blocks []*SyllableElement, index int, lineTime time.Duration, fadeRatio float64) []anim.Keyframe {
	var frames []anim.Keyframe

	/*backgroundImage, backgroundSize, backgroungPX, _ := generateBackgroundFadeStyle(blocks[index].Ele.Get("offsetWidth").Float(), blocks[index].Ele.Get("offsetHeight").Float(), fadeRatio)
	blocks[index].Ele.Get("style").Set("backgroundImage", backgroundImage)
	blocks[index].Ele.Get("style").Set("backgroundSize", backgroundSize)
	blocks[index].Ele.Get("style").Set("backgroundPositionX", fmt.Sprintf("%vpx", backgroungPX))*/

	ElWidth := blocks[index].SyllableImage.GetWidth()
	ElHeight := blocks[index].SyllableImage.GetHeight()
	hr := ElHeight * fadeRatio
	fbw := ElWidth + hr

	// 计算总持续时间(以最后一个单词的结束时间为准)
	totalDuration := lineTime
	if len(blocks) > 0 {
		lastBlockEnd := blocks[len(blocks)-1].EndTime
		totalDuration = lastBlockEnd - blocks[0].StartTime
	}

	// 计算当前单词之前的累计宽度
	widthBeforeSelf := 0.0
	for i := 0; i < index; i++ {
		widthBeforeSelf += blocks[i].SyllableImage.GetWidth()
	}
	if index > 0 {
		widthBeforeSelf += hr // 第一个单词有额外的渐变宽度
	}

	minOffset := -fbw
	clampOffset := func(x float64) float64 {
		if x < minOffset {
			return minOffset
		}
		if x > 0 {
			return 0
		}
		return x
	}

	currentPos := -widthBeforeSelf - ElWidth - hr
	timeOffset := 0.0
	lastPos := currentPos
	lastTime := 0.0

	pushFrame := func() {
		// 确保时间在0-1范围内
		time := math.Max(0, math.Min(1, timeOffset))
		duration := time - lastTime
		moveOffset := currentPos - lastPos
		d := 0.0
		if moveOffset != 0 {
			d = math.Abs(duration / moveOffset)
		}

		// 处理边界情况
		if currentPos > minOffset && lastPos < minOffset {
			staticTime := math.Abs(lastPos-minOffset) * d
			//frames = append(frames, map[string]interface{}{
			//	"backgroundPositionX": fmt.Sprintf("%fpx", clampOffset(lastPos)),
			//	"offset":              lastTime + staticTime,
			//})

			frames = append(frames, anim.Keyframe{
				Offset: lastTime + staticTime,
				Values: []float64{clampOffset(lastPos)},
				Ease:   anim.Linear,
			})
		}

		if currentPos > 0 && lastPos < 0 {
			staticTime := math.Abs(lastPos) * d
			//frames = append(frames, map[string]interface{}{
			//	"backgroundPositionX": fmt.Sprintf("%fpx", clampOffset(currentPos)),
			//	"offset":              lastTime + staticTime,
			//})
			frames = append(frames, anim.Keyframe{
				Offset: lastTime + staticTime,
				Values: []float64{clampOffset(currentPos)},
				Ease:   anim.Linear,
			})
		}

		//frames = append(frames, map[string]interface{}{
		//	"backgroundPositionX": fmt.Sprintf("%fpx", clampOffset(currentPos)),
		//	"offset":              time,
		//})

		frames = append(frames, anim.Keyframe{
			Offset: time,
			Values: []float64{clampOffset(currentPos)},
			Ease:   anim.Linear,
		})

		lastPos = currentPos
		lastTime = time
	}

	// 初始帧
	pushFrame()

	lastTimeStamp := 0.0
	for i, block := range blocks {
		// 停顿阶段
		curTimeStamp := float64(block.StartTime - blocks[0].StartTime)
		staticDuration := curTimeStamp - lastTimeStamp
		timeOffset += staticDuration / float64(totalDuration)
		if staticDuration > 0 {
			pushFrame()
		}
		lastTimeStamp = curTimeStamp

		// 移动阶段
		fadeDuration := float64((block.EndTime - block.StartTime))
		timeOffset += fadeDuration / float64(totalDuration)
		currentPos += block.SyllableImage.GetWidth()

		// 第一个和最后一个单词有额外的渐变处理
		if i == 0 {
			currentPos += hr * 1.5
		}
		if i == len(blocks)-1 {
			currentPos += hr * 0.5
		}

		if fadeDuration > 0 {
			pushFrame()
		}
		lastTimeStamp += fadeDuration
	}
	if len(frames) > 0 {
		//if lastFrame, ok := frames[len(frames)-1].(map[string]interface{}); ok {
		//	lastFrame["offset"] = 1.0
		//}
		frames[len(frames)-1].Offset = 1.0
	}
	return frames
}
