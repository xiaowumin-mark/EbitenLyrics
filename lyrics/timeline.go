package lyrics

import (
	"sort"
	"time"
)

type playerTimeStateResult struct {
	nextHotLines      map[int]struct{}
	addedIDs          map[int]struct{}
	removedHotIDs     map[int]struct{}
	removedBufferedID map[int]struct{}
}

type commitTimeStateResult struct {
	shouldLayout      bool
	shouldResetScroll bool
	linesToEnable     []int
	linesToDisable    []int
}

func newTimelineState() TimelineState {
	return TimelineState{
		HotLines:      make(map[int]struct{}),
		BufferedLines: make(map[int]struct{}),
		ScrollToIndex: 0,
		IsPlaying:     true,
	}
}

func newLayoutState() LayoutState {
	return LayoutState{
		TargetAlignIndex: -1,
		AlignAnchor:      LayoutAlignAnchorCenter,
		AlignPosition:    0.35,
		AllowScroll:      true,
		EnableBlur:       true,
		BlurStrength:     1,
	}
}

func ensureLayoutState(l *Lyrics) {
	if l == nil {
		return
	}
	if l.Layout.TargetAlignIndex == 0 && l.Layout.AlignPosition == 0 && l.Layout.AlignAnchor == LayoutAlignAnchorTop {
		l.Layout = newLayoutState()
	}
}

func ensureTimelineState(l *Lyrics) {
	if l == nil {
		return
	}
	if l.Timeline.HotLines == nil {
		l.Timeline.HotLines = make(map[int]struct{})
	}
	if l.Timeline.BufferedLines == nil {
		l.Timeline.BufferedLines = make(map[int]struct{})
	}
}

func cloneIndexSet(src map[int]struct{}) map[int]struct{} {
	dst := make(map[int]struct{}, len(src))
	for id := range src {
		dst[id] = struct{}{}
	}
	return dst
}

func setHas(set map[int]struct{}, id int) bool {
	_, ok := set[id]
	return ok
}

func sortedSetIDs(set map[int]struct{}) []int {
	ids := make([]int, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func minSetID(set map[int]struct{}) (int, bool) {
	if len(set) == 0 {
		return 0, false
	}
	ids := sortedSetIDs(set)
	return ids[0], true
}

func equalIndexSets(a, b map[int]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for id := range a {
		if !setHas(b, id) {
			return false
		}
	}
	return true
}

func lineActiveRange(line *Line) (time.Duration, time.Duration) {
	if line == nil {
		return 0, 0
	}
	start := line.StartTime
	end := line.EndTime
	for _, bg := range line.BackgroundLines {
		if bg == nil {
			continue
		}
		if bg.StartTime < start {
			start = bg.StartTime
		}
		if bg.EndTime > end {
			end = bg.EndTime
		}
	}
	return start, end
}

func computePlayerTimeState(l *Lyrics, t time.Duration) playerTimeStateResult {
	ensureTimelineState(l)
	result := playerTimeStateResult{
		nextHotLines:      cloneIndexSet(l.Timeline.HotLines),
		addedIDs:          make(map[int]struct{}),
		removedHotIDs:     make(map[int]struct{}),
		removedBufferedID: make(map[int]struct{}),
	}

	for id := range l.Timeline.HotLines {
		if id < 0 || id >= len(l.Lines) || l.Lines[id] == nil {
			delete(result.nextHotLines, id)
			result.removedHotIDs[id] = struct{}{}
			continue
		}
		start, end := lineActiveRange(l.Lines[id])
		if t < start || t >= end {
			delete(result.nextHotLines, id)
			result.removedHotIDs[id] = struct{}{}
		}
	}

	for id, line := range l.Lines {
		if line == nil {
			continue
		}
		start, end := lineActiveRange(line)
		if start <= t && t < end && !setHas(result.nextHotLines, id) {
			result.nextHotLines[id] = struct{}{}
			result.addedIDs[id] = struct{}{}
		}
	}

	for id := range l.Timeline.BufferedLines {
		if !setHas(result.nextHotLines, id) {
			result.removedBufferedID[id] = struct{}{}
		}
	}

	return result
}

func pickScrollToIndexForSeek(t time.Duration, lines []*Line, buffered map[int]struct{}) int {
	if id, ok := minSetID(buffered); ok {
		return id
	}
	for i, line := range lines {
		if line != nil && line.StartTime >= t {
			return i
		}
	}
	return len(lines)
}

func commitPlayerTimeState(l *Lyrics, t time.Duration, stateResult playerTimeStateResult) commitTimeStateResult {
	ensureTimelineState(l)
	commit := commitTimeStateResult{}
	timeline := &l.Timeline
	timeline.CurrentTime = t
	timeline.HotLines = stateResult.nextHotLines

	linesToDisable := make(map[int]struct{})

	if timeline.IsSeeking {
		timeline.BufferedLines = cloneIndexSet(timeline.HotLines)
		timeline.ScrollToIndex = pickScrollToIndexForSeek(t, l.Lines, timeline.BufferedLines)
		for id := range stateResult.removedHotIDs {
			linesToDisable[id] = struct{}{}
		}
		for id := range timeline.HotLines {
			commit.linesToEnable = append(commit.linesToEnable, id)
		}
		for id := range stateResult.removedBufferedID {
			linesToDisable[id] = struct{}{}
		}
		commit.shouldResetScroll = true
		commit.shouldLayout = true
	} else if len(stateResult.addedIDs) > 0 {
		for id := range stateResult.addedIDs {
			timeline.BufferedLines[id] = struct{}{}
			commit.linesToEnable = append(commit.linesToEnable, id)
		}
		for id := range stateResult.removedBufferedID {
			delete(timeline.BufferedLines, id)
			linesToDisable[id] = struct{}{}
		}
		if id, ok := minSetID(timeline.BufferedLines); ok {
			timeline.ScrollToIndex = id
		}
		commit.shouldLayout = true
	} else if len(stateResult.removedBufferedID) > 0 && equalIndexSets(stateResult.removedBufferedID, timeline.BufferedLines) {
		for id := range timeline.BufferedLines {
			if setHas(timeline.HotLines, id) {
				continue
			}
			delete(timeline.BufferedLines, id)
			linesToDisable[id] = struct{}{}
		}
		commit.shouldLayout = true
	}

	if len(timeline.BufferedLines) == 0 && len(l.Lines) > 0 {
		lastLine := l.Lines[len(l.Lines)-1]
		if lastLine != nil && t >= lastLine.EndTime {
			targetIndex := len(l.Lines) - 1
			if l.Bottom.HasContent() {
				targetIndex = len(l.Lines)
			}
			if timeline.ScrollToIndex != targetIndex {
				timeline.ScrollToIndex = targetIndex
				commit.shouldLayout = true
			}
		}
	}

	timeline.LastCurrentTime = t
	commit.linesToEnable = sortedUniqueValidIDs(commit.linesToEnable, len(l.Lines))
	commit.linesToDisable = sortedUniqueValidIDs(sortedSetIDs(linesToDisable), len(l.Lines))
	return commit
}

func sortedUniqueValidIDs(ids []int, max int) []int {
	if len(ids) == 0 {
		return nil
	}
	set := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id >= 0 && id < max {
			set[id] = struct{}{}
		}
	}
	return sortedSetIDs(set)
}

func syncNowLyricsFromTimeline(l *Lyrics) {
	if l == nil {
		return
	}
	l.nowLyrics = sortedSetIDs(l.Timeline.HotLines)
}
