package ws

import (
	"EbitenLyrics/ttml"

	"github.com/go-viper/mapstructure/v2"
)

func ParseLyricsFromMap(lyrics []interface{}) ([]ttml.LyricLine, error) {
	// 预估容量，避免多次内存分配
	lyricLines := make([]ttml.LyricLine, 0, len(lyrics))
	err := mapstructure.Decode(lyrics, &lyricLines)
	if err != nil {
		return nil, err
	}

	// 预分配结果切片容量
	merged := make([]ttml.LyricLine, 0, len(lyricLines)/2)

	for i := 0; i < len(lyricLines); {
		line := &lyricLines[i] // 使用指针避免拷贝

		if line.IsBG {
			// 如果遇到单独的 BG 行而前面没有主行，就跳过
			i++
			continue
		}

		// 收集后面连续的 BG 行
		var bgs []ttml.LyricLine
		j := i + 1

		// 计算连续BG行数量并预分配容量
		bgCount := 0
		for k := j; k < len(lyricLines) && lyricLines[k].IsBG; k++ {
			bgCount++
		}

		if bgCount > 0 {
			bgs = make([]ttml.LyricLine, 0, bgCount)
			for j < len(lyricLines) && lyricLines[j].IsBG {
				bgs = append(bgs, lyricLines[j])
				j++
			}
		}

		// 创建新行并设置BGs
		newLine := *line // 拷贝主行
		newLine.BGs = bgs
		merged = append(merged, newLine)
		i = j
	}

	return merged, nil
}
