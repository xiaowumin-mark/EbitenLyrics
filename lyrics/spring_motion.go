package lyrics

import (
	"math"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
)

var (
	defaultLinePosYSpringParams     = anim.SpringParams{Mass: 0.9, Damping: 15, Stiffness: 90}
	defaultLineScaleSpringParams    = anim.SpringParams{Mass: 2, Damping: 25, Stiffness: 100}
	backgroundLineScaleSpringParams = anim.SpringParams{Mass: 1, Damping: 20, Stiffness: 50}
)

func ensureLineSprings(line *Line) {
	if line == nil {
		return
	}
	if line.PosYSpring == nil {
		line.PosYSpring = anim.NewSpring(line.GetPosition().GetY(), defaultLinePosYSpringParams)
	}
	if line.ScaleSpring == nil {
		params := defaultLineScaleSpringParams
		if line.IsBackground {
			params = backgroundLineScaleSpringParams
		}
		line.ScaleSpring = anim.NewSpring(line.GetPosition().GetScaleX(), params)
	}
}

func setLineSpringTargets(line *Line, targetY, targetScale float64, force bool, delay ...time.Duration) {
	if line == nil {
		return
	}
	ensureLineSprings(line)
	if force {
		line.PosYSpring.SetPosition(targetY)
		line.ScaleSpring.SetPosition(targetScale)
		line.GetPosition().SetY(targetY)
		line.GetPosition().SetScaleX(targetScale)
		line.GetPosition().SetScaleY(targetScale)
		return
	}
	var d time.Duration
	if len(delay) > 0 {
		d = delay[0]
	}
	line.PosYSpring.SetTargetPosition(targetY, d)
	line.ScaleSpring.SetTargetPosition(targetScale, d)
}

func updateLineSprings(line *Line, dt time.Duration) bool {
	if line == nil {
		return false
	}
	ensureLineSprings(line)
	oldY := line.GetPosition().GetY()
	oldScaleX := line.GetPosition().GetScaleX()
	line.PosYSpring.Update(dt)
	line.ScaleSpring.Update(dt)
	newY := line.PosYSpring.CurrentPosition()
	newScale := line.ScaleSpring.CurrentPosition()
	line.GetPosition().SetY(newY)
	line.GetPosition().SetScaleX(newScale)
	line.GetPosition().SetScaleY(newScale)
	return math.Abs(oldY-newY) > animationValueEpsilon || math.Abs(oldScaleX-newScale) > animationValueEpsilon
}

func updateLyricsSprings(l *Lyrics, dt time.Duration) {
	if l == nil || dt <= 0 {
		return
	}
	l.Dots.Tick(dt)
	l.Bottom.Update(dt)
	for _, line := range l.Lines {
		updateLineSprings(line, dt)
		for _, bg := range line.BackgroundLines {
			updateLineSprings(bg, dt)
		}
	}
}

func computeLinePosYSpringParams(l *Lyrics, scrollToIndex int, isInterludeActive bool) anim.SpringParams {
	if l == nil || scrollToIndex <= 0 || scrollToIndex >= len(l.Lines) || l.Timeline.IsSeeking || isInterludeActive {
		return defaultLinePosYSpringParams
	}
	current := l.Lines[scrollToIndex]
	prev := l.Lines[scrollToIndex-1]
	if current == nil || prev == nil {
		return defaultLinePosYSpringParams
	}
	prevStart := prev.StartTime
	if len(prev.Syllables) > 0 {
		prevStart = prev.Syllables[0].StartTime
	}
	interval := current.StartTime - prevStart
	intervalMs := float64(interval.Milliseconds())
	if intervalMs < 100 {
		intervalMs = 100
	}
	if intervalMs > 800 {
		intervalMs = 800
	}
	ratio := 1 - (intervalMs-100)/(800-100)
	ratio = math.Pow(ratio, 0.2)
	stiffness := 170 + ratio*(220-170)
	damping := math.Sqrt(stiffness) * 2.2
	return anim.SpringParams{Mass: 0.9, Damping: damping, Stiffness: stiffness}
}

func updateLinePosYSpringParams(l *Lyrics, params anim.SpringParams) {
	if l == nil {
		return
	}
	l.Bottom.ensureSprings()
	l.Bottom.PosYSpring.UpdateParams(params)
	for _, line := range l.Lines {
		ensureLineSprings(line)
		line.PosYSpring.UpdateParams(params)
		for _, bg := range line.BackgroundLines {
			ensureLineSprings(bg)
			bg.PosYSpring.UpdateParams(params)
		}
	}
}
