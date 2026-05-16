package lyrics

import "time"

const (
	interludeTimeOffset = 20 * time.Millisecond
	interludeEndLead    = 250 * time.Millisecond
	interludeMinGap     = 4 * time.Second
)

func computeCurrentInterlude(l *Lyrics) *Interlude {
	if l == nil || len(l.Lines) == 0 {
		return nil
	}
	ensureTimelineState(l)
	currentTime := l.Timeline.CurrentTime + interludeTimeOffset
	currentIndex := l.Timeline.ScrollToIndex
	if currentIndex < 0 {
		currentIndex = 0
	}
	checks := []int{currentIndex - 1, currentIndex, currentIndex + 1}
	for _, anchorIndex := range checks {
		if interlude := computeInterludeGapAt(l.Lines, currentTime, anchorIndex); interlude != nil {
			return interlude
		}
	}
	return nil
}

func computeInterludeGapAt(lines []*Line, currentTime time.Duration, anchorIndex int) *Interlude {
	if anchorIndex < -1 || anchorIndex >= len(lines)-1 {
		return nil
	}
	var gapStart time.Duration
	if anchorIndex >= 0 {
		prevLine := lines[anchorIndex]
		if prevLine == nil {
			return nil
		}
		_, gapStart = lineActiveRange(prevLine)
	}
	nextLine := lines[anchorIndex+1]
	if nextLine == nil {
		return nil
	}
	nextStart, _ := lineActiveRange(nextLine)
	gapEnd := nextStart - interludeEndLead
	if gapEnd < gapStart {
		gapEnd = gapStart
	}
	if gapEnd-gapStart < interludeMinGap {
		return nil
	}
	if gapEnd > currentTime && gapStart < currentTime {
		return &Interlude{
			StartTime:       gapStart,
			EndTime:         gapEnd,
			AnchorLineIndex: anchorIndex,
			IsNextDuet:      nextLine.IsDuet,
		}
	}
	return nil
}
