package lyrics

func (l *Lyrics) SetUserScrolling(scrolling bool) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	if l.Layout.IsUserScrolling == scrolling {
		return
	}
	l.Layout.IsUserScrolling = scrolling
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) BeginScroll() bool {
	if l == nil {
		return false
	}
	ensureLayoutState(l)
	if !l.Layout.AllowScroll {
		return false
	}
	l.Layout.IsScrolled = true
	return true
}

func (l *Lyrics) ClampScrollOffset() {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	if l.Layout.ScrollMinOffset > l.Layout.ScrollMaxOffset {
		return
	}
	l.Layout.ScrollOffset = clampFloat(l.Layout.ScrollOffset, l.Layout.ScrollMinOffset, l.Layout.ScrollMaxOffset)
}

func (l *Lyrics) SetHidePassedLines(enabled bool) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	if l.Layout.HidePassedLines == enabled {
		return
	}
	l.Layout.HidePassedLines = enabled
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) SetEnableBlur(enabled bool) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	if l.Layout.EnableBlur == enabled {
		return
	}
	l.Layout.EnableBlur = enabled
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) SetBlurStrength(strength float64) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	strength = clampFloat(strength, 0, 4)
	if l.Layout.BlurStrength == strength {
		return
	}
	l.Layout.BlurStrength = strength
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) SetScrollOffset(offset float64) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	if l.Layout.ScrollOffset == offset {
		return
	}
	l.Layout.ScrollOffset = offset
	l.ClampScrollOffset()
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) AddScrollOffset(delta float64) {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	l.SetScrollOffset(l.Layout.ScrollOffset + delta)
}

func (l *Lyrics) ResetScrollOffset() {
	if l == nil {
		return
	}
	ensureLayoutState(l)
	l.Layout.IsScrolled = false
	l.Layout.IsUserScrolling = false
	l.SetScrollOffset(0)
}

func (l *Lyrics) AddWheelScroll(delta float64) {
	if l == nil || !l.BeginScroll() {
		return
	}
	l.SetScrollOffset(l.Layout.ScrollOffset + delta)
}
