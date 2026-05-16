package lyrics

import "time"

func (l *Lyrics) SetCurrentTime(t time.Duration, isSeek bool) {
	if l == nil {
		return
	}
	ensureTimelineState(l)
	l.Timeline.IsSeeking = isSeek
	l.Update(t)
	if isSeek {
		l.Timeline.IsSeeking = false
	}
}

func (l *Lyrics) SetPlaying(playing bool) {
	if l == nil {
		return
	}
	ensureTimelineState(l)
	if l.Timeline.IsPlaying == playing {
		return
	}
	l.Timeline.IsPlaying = playing
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) Pause() {
	if l != nil {
		l.Dots.Pause()
	}
	l.SetPlaying(false)
}

func (l *Lyrics) Resume() {
	if l != nil {
		l.Dots.Resume()
	}
	l.SetPlaying(true)
}
