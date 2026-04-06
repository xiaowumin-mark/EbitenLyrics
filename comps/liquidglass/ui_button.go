package liquidglass

import (
	"image"
	"image/color"
	"math"

	"EbitenLyrics/filters"

	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type UIButton struct {
	Label     string
	Face      textv2.Face
	TextColor color.Color

	BlurRadius   float64
	PressScale   float64
	HoverScale   float64
	CornerRadius float64

	OnClicked func()

	widget   *widget.Widget
	hovering bool
	pressing bool
	scale    float64
	target   float64

	shader *ebiten.Shader
	bg     *ebiten.Image
	blur   *ebiten.Image
	out    *ebiten.Image
}

func NewUIButton(label string, face textv2.Face, onClicked func()) (*UIButton, error) {
	shader, err := ebiten.NewShader(liquidGlassShaderSource)
	if err != nil {
		return nil, err
	}

	b := &UIButton{
		Label:        label,
		Face:         face,
		TextColor:    color.NRGBA{R: 0xEE, G: 0xF4, B: 0xFF, A: 0xFF},
		BlurRadius:   11,
		PressScale:   1.022,
		HoverScale:   1.008,
		CornerRadius: 0, // auto capsule
		OnClicked:    onClicked,
		scale:        1,
		target:       1,
		shader:       shader,
	}
	b.createWidget()
	return b, nil
}

func (b *UIButton) Dispose() {
	if b.bg != nil {
		b.bg.Dispose()
		b.bg = nil
	}
	if b.blur != nil {
		b.blur.Dispose()
		b.blur = nil
	}
	if b.out != nil {
		b.out.Dispose()
		b.out = nil
	}
}

func (b *UIButton) GetWidget() *widget.Widget {
	return b.widget
}

func (b *UIButton) Validate() {}

func (b *UIButton) PreferredSize() (int, int) {
	width := 140
	height := 44
	if b.Face != nil && b.Label != "" {
		tw, th := textv2.Measure(b.Label, b.Face, 0)
		width = maxInt(width, int(math.Ceil(tw))+30)
		height = maxInt(height, int(math.Ceil(th))+20)
	}
	if b.widget != nil {
		width = maxInt(width, b.widget.MinWidth)
		height = maxInt(height, b.widget.MinHeight)
	}
	return width, height
}

func (b *UIButton) SetLocation(rect image.Rectangle) {
	b.widget.Rect = rect
}

func (b *UIButton) SetupInputLayer(def input.DeferredSetupInputLayerFunc) {}

func (b *UIButton) Update(updObj *widget.UpdateObject) {
	b.widget.Update(updObj)

	if b.widget.Disabled {
		b.target = 1
	} else if b.pressing {
		b.target = maxFloat(1, b.PressScale)
	} else if b.hovering {
		b.target = maxFloat(1, b.HoverScale)
	} else {
		b.target = 1
	}
	b.scale += (b.target - b.scale) * 0.28
	if math.Abs(b.target-b.scale) < 0.0005 {
		b.scale = b.target
	}
}

func (b *UIButton) Render(screen *ebiten.Image) {
	if !b.widget.IsVisible() {
		return
	}
	b.widget.Render(screen)

	rect := b.widget.Rect
	w, h := rect.Dx(), rect.Dy()
	if w <= 2 || h <= 2 {
		return
	}

	b.ensureBuffers(w, h)

	b.bg.Clear()
	capture := &ebiten.DrawImageOptions{}
	capture.GeoM.Translate(float64(-rect.Min.X), float64(-rect.Min.Y))
	b.bg.DrawImage(screen, capture)

	if b.blur != nil {
		b.blur.Dispose()
	}
	blurRadius := int(math.Round(clampFloat(b.BlurRadius, 1, 20)))
	b.blur = filters.BlurImageShader(b.bg, float64(blurRadius))

	b.out.Clear()
	mx, my := ebiten.CursorPosition()
	localMX := clampFloat(float64(mx-rect.Min.X), 0, float64(w))
	localMY := clampFloat(float64(my-rect.Min.Y), 0, float64(h))
	if !b.hovering && !b.pressing {
		localMX = float64(w) * 0.5
		localMY = float64(h) * 0.5
	}

	cornerR := b.resolvedCornerRadius(w, h)
	shapeW := float64(w) - 2
	shapeH := float64(h) - 2
	shapeR := clampFloat(cornerR-1, 3, math.Min(shapeW, shapeH)*0.48)
	glareFactor := 0.62
	if b.hovering {
		glareFactor = 0.72
	}
	if b.pressing {
		glareFactor = 0.82
	}

	shaderOp := &ebiten.DrawRectShaderOptions{}
	shaderOp.Images[0] = b.bg
	shaderOp.Images[1] = b.blur
	shaderOp.Uniforms = map[string]any{
		"Resolution": []float32{float32(w), float32(h)},
		"Mouse":      []float32{float32(localMX), float32(localMY)},
		"MouseSpring": []float32{
			float32(localMX),
			float32(localMY),
		},
		"ShowShape1":     float32(1),
		"ShapeWidth":     float32(shapeW),
		"ShapeHeight":    float32(shapeH),
		"ShapeRadius":    float32(shapeR),
		"ShapeRoundness": float32(5),
		"MergeRate":      float32(0.03),

		"RefThickness":       float32(8),
		"RefFactor":          float32(1.12),
		"RefDispersion":      float32(1.2),
		"RefFresnelRange":    float32(24),
		"RefFresnelHardness": float32(0.18),
		"RefFresnelFactor":   float32(0.12),

		"GlareRange":          float32(16),
		"GlareConvergence":    float32(0.66),
		"GlareOppositeFactor": float32(0.78),
		"GlareFactor":         float32(glareFactor),
		"GlareHardness":       float32(0.2),
		"GlareAngle":          float32(-35.0 / 180.0 * math.Pi),

		"BlurEdge": float32(1),
		"Tint": []float32{
			0.965, 0.98, 1, 0.032,
		},
		"Step": float32(9),
	}
	b.out.DrawRectShader(w, h, b.shader, shaderOp)
	maskPath := roundedRectPath(0, 0, float32(w), float32(h), float32(cornerR))
	vector.FillPath(b.out, &maskPath, nil, &vector.DrawPathOptions{
		AntiAlias: true,
		Blend:     ebiten.BlendDestinationIn,
	})
	vector.FillCircle(
		b.out,
		float32(float64(w)*0.35+localMX*0.2),
		float32(float64(h)*0.2+localMY*0.15),
		float32(math.Min(float64(w), float64(h))*0.22),
		color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x12},
		true,
	)

	centerX := float64(rect.Min.X) + float64(w)*0.5
	centerY := float64(rect.Min.Y) + float64(h)*0.5
	drawScaledRoundedRectFill(
		screen,
		centerX,
		centerY+1.4,
		float64(w),
		float64(h),
		b.scale,
		cornerR,
		color.NRGBA{R: 0, G: 0, B: 0, A: 0x24},
	)

	drawOp := &ebiten.DrawImageOptions{}
	drawOp.GeoM.Translate(-float64(w)*0.5, -float64(h)*0.5)
	drawOp.GeoM.Scale(b.scale, b.scale)
	drawOp.GeoM.Translate(centerX, centerY)
	if b.widget.Disabled {
		drawOp.ColorScale.Scale(1, 1, 1, 0.55)
	}
	screen.DrawImage(b.out, drawOp)

	borderColor := color.NRGBA{R: 0xE8, G: 0xF0, B: 0xFF, A: 0x70}
	if b.hovering {
		borderColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xA8}
	}
	if b.pressing {
		borderColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xD0}
	}
	drawScaledRoundedRectStroke(
		screen,
		centerX,
		centerY,
		float64(w),
		float64(h),
		b.scale,
		cornerR,
		1.5,
		borderColor,
	)
	drawScaledRoundedRectFill(
		screen,
		centerX,
		centerY-float64(h)*0.16*b.scale,
		float64(w)*0.88,
		float64(h)*0.34,
		b.scale,
		math.Max(2, cornerR*0.7),
		color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x18},
	)

	if b.Face != nil && b.Label != "" {
		textW, textH := textv2.Measure(b.Label, b.Face, 0)
		metrics := b.Face.Metrics()
		textOp := &textv2.DrawOptions{}
		textOp.GeoM.Translate(
			centerX-textW*0.5,
			centerY-textH*0.5+metrics.HAscent,
		)
		textColor := b.TextColor
		if textColor == nil {
			textColor = color.White
		}
		r, g, bb, a := textColor.RGBA()
		textOp.ColorScale.Scale(
			float32(r)/0xffff,
			float32(g)/0xffff,
			float32(bb)/0xffff,
			float32(a)/0xffff,
		)
		if b.widget.Disabled {
			textOp.ColorScale.Scale(1, 1, 1, 0.7)
		}
		textv2.Draw(screen, b.Label, b.Face, textOp)
	}
}

func (b *UIButton) ensureBuffers(w, h int) {
	if b.bg == nil || b.bg.Bounds().Dx() != w || b.bg.Bounds().Dy() != h {
		if b.bg != nil {
			b.bg.Dispose()
		}
		b.bg = ebiten.NewImage(w, h)
	}
	if b.out == nil || b.out.Bounds().Dx() != w || b.out.Bounds().Dy() != h {
		if b.out != nil {
			b.out.Dispose()
		}
		b.out = ebiten.NewImage(w, h)
	}
}

func (b *UIButton) createWidget() {
	b.widget = widget.NewWidget(
		widget.WidgetOpts.TrackHover(true),
		widget.WidgetOpts.CursorEnterHandler(func(args *widget.WidgetCursorEnterEventArgs) {
			if !b.widget.Disabled {
				b.hovering = true
			}
		}),
		widget.WidgetOpts.CursorExitHandler(func(args *widget.WidgetCursorExitEventArgs) {
			b.hovering = false
		}),
		widget.WidgetOpts.MouseButtonPressedHandler(func(args *widget.WidgetMouseButtonPressedEventArgs) {
			if b.widget.Disabled {
				return
			}
			if args.Button == ebiten.MouseButtonLeft {
				b.pressing = true
			}
		}),
		widget.WidgetOpts.MouseButtonReleasedHandler(func(args *widget.WidgetMouseButtonReleasedEventArgs) {
			if args.Button == ebiten.MouseButtonLeft {
				b.pressing = false
			}
		}),
		widget.WidgetOpts.MouseButtonClickedHandler(func(args *widget.WidgetMouseButtonClickedEventArgs) {
			if b.widget.Disabled {
				return
			}
			if args.Button == ebiten.MouseButtonLeft && b.OnClicked != nil {
				b.OnClicked()
			}
		}),
	)
}

func drawScaledRoundedRectStroke(dst *ebiten.Image, centerX, centerY, w, h, scale, radius, strokeWidth float64, c color.Color) {
	sw := w * scale
	sh := h * scale
	x := float32(centerX - sw*0.5)
	y := float32(centerY - sh*0.5)
	r := float32(clampFloat(radius*scale, 3, math.Min(sw, sh)*0.48))

	path := roundedRectPath(x, y, float32(sw), float32(sh), r)
	vector.StrokePath(dst, &path, &vector.StrokeOptions{
		Width:    float32(strokeWidth),
		LineJoin: vector.LineJoinRound,
	}, &vector.DrawPathOptions{
		AntiAlias: true,
		ColorScale: func() ebiten.ColorScale {
			var cs ebiten.ColorScale
			cs.ScaleWithColor(c)
			return cs
		}(),
	})
}

func drawScaledRoundedRectFill(dst *ebiten.Image, centerX, centerY, w, h, scale, radius float64, c color.Color) {
	sw := w * scale
	sh := h * scale
	x := float32(centerX - sw*0.5)
	y := float32(centerY - sh*0.5)
	r := float32(clampFloat(radius*scale, 2, math.Min(sw, sh)*0.48))
	path := roundedRectPath(x, y, float32(sw), float32(sh), r)
	vector.FillPath(dst, &path, nil, &vector.DrawPathOptions{
		AntiAlias: true,
		ColorScale: func() ebiten.ColorScale {
			var cs ebiten.ColorScale
			cs.ScaleWithColor(c)
			return cs
		}(),
	})
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (b *UIButton) resolvedCornerRadius(w, h int) float64 {
	if b.CornerRadius > 0 {
		return clampFloat(b.CornerRadius, 3, math.Min(float64(w), float64(h))*0.5-1)
	}
	return clampFloat(float64(h)*0.5-1, 3, math.Min(float64(w), float64(h))*0.5-1)
}

func roundedRectPath(x, y, w, h, r float32) vector.Path {
	var p vector.Path
	if w <= 0 || h <= 0 {
		return p
	}
	if r < 0 {
		r = 0
	}
	maxR := minFloat32(w, h) * 0.5
	if r > maxR {
		r = maxR
	}
	if r <= 0 {
		p.MoveTo(x, y)
		p.LineTo(x+w, y)
		p.LineTo(x+w, y+h)
		p.LineTo(x, y+h)
		p.Close()
		return p
	}

	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.Arc(x+w-r, y+r, r, -math.Pi*0.5, 0, vector.Clockwise)
	p.LineTo(x+w, y+h-r)
	p.Arc(x+w-r, y+h-r, r, 0, math.Pi*0.5, vector.Clockwise)
	p.LineTo(x+r, y+h)
	p.Arc(x+r, y+h-r, r, math.Pi*0.5, math.Pi, vector.Clockwise)
	p.LineTo(x, y+r)
	p.Arc(x+r, y+r, r, math.Pi, math.Pi*1.5, vector.Clockwise)
	p.Close()
	return p
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
