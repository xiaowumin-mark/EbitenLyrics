package lyrics

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
)

const interludeDotsTargetBreatheDuration = 1500 * time.Millisecond

func newInterludeDots(fontSize float64) InterludeDots {
	dots := InterludeDots{}
	dots.UpdateMetrics(fontSize, 0)
	dots.resetVisualState()
	return dots
}

func (l *Lyrics) DotsFontSize() float64 {
	if l == nil {
		return 24
	}
	for _, line := range l.Lines {
		if line != nil && line.fontsize > 0 {
			return line.fontsize
		}
	}
	return 24
}

func (d *InterludeDots) UpdateMetrics(fontSize, viewportHeight float64) {
	if fontSize <= 0 {
		fontSize = 24
	}
	preferred := fontSize * 0.5
	if viewportHeight > 0 {
		preferred = viewportHeight * 0.01
	}
	d.DotSize = clampDotsFloat(preferred, fontSize*0.5, fontSize*3)
	d.Gap = fontSize * 0.25
	d.PaddingX = fontSize * 0.75
	// CSS ref uses padding-block: 2.5% on an element whose height is dot size,
	// so vertical padding is dotSize * 2.5%, not viewport height * 2.5%.
	d.PaddingY = d.DotSize * 0.025
	d.Margin = fontSize * 0.4
	d.Position.SetW(d.Width())
	d.Position.SetH(d.Height())
}

func (d *InterludeDots) Width() float64 {
	return d.PaddingX*2 + d.DotSize*3 + d.Gap*2
}

func (d *InterludeDots) Height() float64 {
	return d.PaddingY*2 + d.DotSize
}

func (d *InterludeDots) TotalHeight() float64 {
	return d.Height() + d.Margin*2
}

func (d *InterludeDots) SetLayout(active bool, x, y float64) {
	d.Active = active
	d.Position.SetX(x)
	d.Position.SetY(y)
	d.Position.SetW(d.Width())
	d.Position.SetH(d.Height())
}

func interludeDotsXForLine(l *Lyrics, line *Line, isNextDuet bool) float64 {
	if l == nil {
		return 0
	}
	if isNextDuet {
		return math.Max(0, l.Width-l.Dots.Width())
	}
	return 0
}

func (d *InterludeDots) SetInterlude(interlude *Interlude) {
	if interlude != nil {
		d.SetInterludeAt(interlude, interlude.StartTime)
		return
	}
	d.SetInterludeAt(nil, 0)
}

func (d *InterludeDots) SetInterludeAt(interlude *Interlude, currentTime time.Duration) {
	if interlude == nil {
		d.Active = false
		d.StartTime = 0
		d.EndTime = 0
		d.CurrentTime = 0
		d.resetVisualState()
		return
	}
	d.Active = true
	d.IsDuet = interlude.IsNextDuet
	d.StartTime = interlude.StartTime
	d.EndTime = interlude.EndTime
	d.CurrentTime = clampDuration(currentTime, interlude.StartTime, interlude.EndTime)
	d.updateVisualState()
}

func clampDuration(value, min, max time.Duration) time.Duration {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (d *InterludeDots) Tick(dt time.Duration) {
	if d == nil || !d.Active || d.Paused || dt <= 0 {
		return
	}
	d.CurrentTime += dt
	d.updateVisualState()
}

func (d *InterludeDots) Pause() {
	if d == nil {
		return
	}
	d.Paused = true
}

func (d *InterludeDots) Resume() {
	if d == nil {
		return
	}
	d.Paused = false
}

func (d *InterludeDots) updateVisualState() {
	if d.EndTime <= d.StartTime {
		d.resetVisualState()
		return
	}
	interludeDuration := d.EndTime - d.StartTime
	currentDuration := d.CurrentTime - d.StartTime
	if currentDuration < 0 {
		currentDuration = 0
	}
	if currentDuration > interludeDuration {
		d.GlobalScale = 0
		d.GlobalAlpha = 0
		d.DotAlphas = [3]float64{}
		return
	}

	cycles := math.Ceil(float64(interludeDuration) / float64(interludeDotsTargetBreatheDuration))
	if cycles < 1 {
		cycles = 1
	}
	breatheDuration := time.Duration(float64(interludeDuration) / cycles)
	if breatheDuration <= 0 {
		breatheDuration = interludeDotsTargetBreatheDuration
	}

	currentMs := float64(currentDuration.Milliseconds())
	interludeMs := float64(interludeDuration.Milliseconds())
	breatheMs := float64(breatheDuration.Milliseconds())
	scale := math.Sin(1.5*math.Pi-(currentMs/breatheMs)*2)/20 + 1
	globalOpacity := 1.0

	if currentDuration < 2*time.Second {
		scale *= easeOutExpoDuration(currentDuration, 2*time.Second)
	}
	if currentDuration < 500*time.Millisecond {
		globalOpacity = 0
	} else if currentDuration < time.Second {
		globalOpacity *= float64((currentDuration - 500*time.Millisecond).Milliseconds()) / 500
	}

	remaining := interludeDuration - currentDuration
	if remaining < 750*time.Millisecond {
		x := float64((750*time.Millisecond - remaining).Milliseconds()) / 750 / 2
		scale *= 1 - easeInOutBack(x)
	}
	if remaining < 375*time.Millisecond {
		globalOpacity *= clamp01(float64(remaining.Milliseconds()) / 375)
	}

	dotsDurationMs := math.Max(interludeMs-750, 0)
	d.GlobalScale = clampPositive(scale) * 0.7
	d.GlobalAlpha = clamp01(globalOpacity)
	d.DotAlphas[0] = clampDotsFloat(((currentMs*3)/dotsDurationMs)*0.75, 0.25, 1)
	d.DotAlphas[1] = clampDotsFloat((((currentMs-dotsDurationMs/3)*3)/dotsDurationMs)*0.75, 0.25, 1)
	d.DotAlphas[2] = clampDotsFloat((((currentMs-(dotsDurationMs/3)*2)*3)/dotsDurationMs)*0.75, 0.25, 1)
}

func (d *InterludeDots) resetVisualState() {
	d.GlobalScale = 0
	d.GlobalAlpha = 0
	d.IsDuet = false
	d.DotAlphas = [3]float64{}
}

func (d *InterludeDots) Draw(screen *ebiten.Image) {
	if d == nil || screen == nil || !d.Active || d.DotSize <= 0 || d.GlobalAlpha <= 0 || d.GlobalScale <= 0 {
		return
	}
	centerX := d.Position.GetX() + d.Position.GetW()/2
	if d.IsDuet {
		centerX = d.Position.GetX() + d.Position.GetW()
	}
	centerY := d.Position.GetY() + d.Position.GetH()/2
	for i := 0; i < 3; i++ {
		localX := d.PaddingX + d.DotSize/2 + float64(i)*(d.DotSize+d.Gap)
		localY := d.PaddingY + d.DotSize/2
		dotCenterX := centerX + (localX-d.Position.GetW()/2)*d.GlobalScale
		dotCenterY := centerY + (localY-d.Position.GetH()/2)*d.GlobalScale
		radius := d.DotSize * d.GlobalScale / 2
		alpha := uint8(clamp01(d.GlobalAlpha*d.DotAlphas[i]) * 255)
		if alpha == 0 {
			continue
		}
		vector.FillCircle(
			screen,
			float32(lp.LP(dotCenterX)),
			float32(lp.LP(dotCenterY)),
			float32(lp.LP(radius)),
			color.NRGBA{R: 255, G: 255, B: 255, A: alpha},
			true,
		)
	}
}

func easeOutExpoDuration(current, total time.Duration) float64 {
	if total <= 0 {
		return 1
	}
	x := clamp01(float64(current) / float64(total))
	if x == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*x)
}

func easeInOutBack(x float64) float64 {
	x = clamp01(x)
	c1 := 1.70158
	c2 := c1 * 1.525
	if x < 0.5 {
		return math.Pow(2*x, 2) * ((c2+1)*2*x - c2) / 2
	}
	return (math.Pow(2*x-2, 2)*((c2+1)*(x*2-2)+c2) + 2) / 2
}

func clampPositive(v float64) float64 {
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func clamp01(v float64) float64 {
	return clampDotsFloat(v, 0, 1)
}

func clampDotsFloat(v, min, max float64) float64 {
	if math.IsNaN(v) {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
