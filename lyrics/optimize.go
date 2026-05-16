package lyrics

import (
	"strings"

	ttml "github.com/xiaowumin-mark/EbitenLyrics/ttml"
)

func optimizeLyricLines(lines []ttml.LyricLine) []ttml.LyricLine {
	optimized := cloneTTMLLines(lines)
	normalizeLyricSpaces(optimized)
	resetLyricLineTimestamps(optimized)
	syncMainAndBackgroundLineTimes(optimized)
	cleanUnintentionalLineOverlaps(optimized)
	tryAdvanceLyricStartTimes(optimized)
	return optimized
}

func cloneTTMLLines(lines []ttml.LyricLine) []ttml.LyricLine {
	out := make([]ttml.LyricLine, len(lines))
	for i := range lines {
		out[i] = cloneTTMLLine(lines[i])
	}
	return out
}

func cloneTTMLLine(line ttml.LyricLine) ttml.LyricLine {
	line.Words = append([]ttml.LyricWord(nil), line.Words...)
	line.BGs = cloneTTMLLines(line.BGs)
	return line
}

func normalizeLyricSpaces(lines []ttml.LyricLine) {
	for i := range lines {
		for j := range lines[i].Words {
			word := lines[i].Words[j].Word
			if strings.TrimSpace(word) == "" {
				continue
			}
			leadingSpace := word[:len(word)-len(strings.TrimLeft(word, " \t\n\r"))]
			trailingSpace := word[len(strings.TrimRight(word, " \t\n\r")):]
			core := strings.Join(strings.Fields(word), " ")
			lines[i].Words[j].Word = leadingSpace + core + trailingSpace
		}
		normalizeLyricSpaces(lines[i].BGs)
	}
}

func resetLyricLineTimestamps(lines []ttml.LyricLine) {
	for i := range lines {
		resetSingleLineTimestamps(&lines[i])
		resetLyricLineTimestamps(lines[i].BGs)
	}
}

func resetSingleLineTimestamps(line *ttml.LyricLine) {
	if line == nil || len(line.Words) == 0 {
		return
	}
	if len(line.Words) == 1 && line.Words[0].StartTime == 0 && line.Words[0].EndTime == 0 && (line.StartTime != 0 || line.EndTime != 0) {
		line.Words[0].StartTime = line.StartTime
		line.Words[0].EndTime = line.EndTime
		return
	}
	line.StartTime = line.Words[0].StartTime
	line.EndTime = line.Words[len(line.Words)-1].EndTime
}

func syncMainAndBackgroundLineTimes(lines []ttml.LyricLine) {
	for i := range lines {
		line := &lines[i]
		for bgIndex := range line.BGs {
			bg := &line.BGs[bgIndex]
			start, end, ok := combinedWordTimeRange(line.Words, bg.Words)
			if !ok {
				start = minInt(line.StartTime, bg.StartTime)
				end = maxInt(line.EndTime, bg.EndTime)
			} else {
				start = minInt(start, minInt(line.StartTime, bg.StartTime))
				end = maxInt(end, maxInt(line.EndTime, bg.EndTime))
			}
			line.StartTime = start
			line.EndTime = end
			bg.StartTime = start
			bg.EndTime = end
		}
	}
}

func combinedWordTimeRange(wordGroups ...[]ttml.LyricWord) (int, int, bool) {
	found := false
	start := 0
	end := 0
	for _, words := range wordGroups {
		for _, word := range words {
			if strings.TrimSpace(word.Word) == "" {
				continue
			}
			if !found {
				start = word.StartTime
				end = word.EndTime
				found = true
				continue
			}
			start = minInt(start, word.StartTime)
			end = maxInt(end, word.EndTime)
		}
	}
	return start, end, found
}

func cleanUnintentionalLineOverlaps(lines []ttml.LyricLine) {
	for i := 0; i < len(lines)-1; i++ {
		line := &lines[i]
		nextLine := lines[i+1]
		overlap := line.EndTime - nextLine.StartTime
		if overlap <= 0 {
			continue
		}
		nextDuration := nextLine.EndTime - nextLine.StartTime
		threshold := nextDuration / 10
		intentional := overlap > 100 && overlap > threshold
		if intentional {
			continue
		}
		line.EndTime = nextLine.StartTime
		for bgIndex := range line.BGs {
			if line.BGs[bgIndex].EndTime > nextLine.StartTime {
				line.BGs[bgIndex].EndTime = nextLine.StartTime
			}
		}
	}
}

func tryAdvanceLyricStartTimes(lines []ttml.LyricLine) {
	const defaultAdvanceAmount = 600
	const fallbackAdvanceAmount = 400
	const fallbackAdvanceRatio = 0.3

	prevLineStart := 0
	prevLineEnd := 0
	prevGroupStart := 0
	prevGroupEnd := 0
	hasPrev := false

	for i := range lines {
		line := &lines[i]
		originalStart := line.StartTime
		originalEnd := line.EndTime
		targetAdvance := defaultAdvanceAmount
		safeBoundary := 0

		if hasPrev {
			if originalStart >= prevLineEnd {
				targetAdvance = defaultAdvanceAmount
				safeBoundary = prevGroupEnd
			} else {
				targetAdvance = fallbackAdvanceAmount
				prevDuration := prevLineEnd - prevLineStart
				safeBoundary = prevLineStart + int(float64(prevDuration)*fallbackAdvanceRatio)
			}
		}

		newStart := maxInt(safeBoundary, line.StartTime-targetAdvance)
		if newStart < line.StartTime {
			line.StartTime = newStart
			for bgIndex := range line.BGs {
				line.BGs[bgIndex].StartTime = newStart
			}
		}

		if hasPrev {
			overlapsPrevGroup := originalStart < prevGroupEnd && originalEnd > prevGroupStart
			if overlapsPrevGroup {
				prevGroupStart = minInt(prevGroupStart, originalStart)
				prevGroupEnd = maxInt(prevGroupEnd, originalEnd)
			} else {
				prevGroupStart = originalStart
				prevGroupEnd = originalEnd
			}
		} else {
			prevGroupStart = originalStart
			prevGroupEnd = originalEnd
		}

		prevLineStart = originalStart
		prevLineEnd = originalEnd
		hasPrev = true
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
