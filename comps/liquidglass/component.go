package liquidglass

import (
	"EbitenLyrics/filters"
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed shaders/liquid_glass.kage
var liquidGlassShaderSource []byte

type BackgroundMode int

const (
	BackgroundPhoto BackgroundMode = iota
	BackgroundGrid
	BackgroundBars
	BackgroundHalf
)

type Params struct {
	ShowShape1     bool
	ShapeWidth     float64
	ShapeHeight    float64
	ShapeRadiusPct float64
	ShapeRoundness float64
	MergeRate      float64

	RefThickness       float64
	RefFactor          float64
	RefDispersion      float64
	RefFresnelRange    float64
	RefFresnelHardness float64
	RefFresnelFactor   float64

	GlareRange          float64
	GlareConvergence    float64
	GlareOppositeFactor float64
	GlareFactor         float64
	GlareHardness       float64
	GlareAngleDeg       float64

	BlurRadius float64
	BlurEdge   bool
	TintR      float64
	TintG      float64
	TintB      float64
	TintA      float64
	Step       float64

	SpringStiffness  float64
	SpringDamping    float64
	SpringSizeFactor float64
	FreezeFollow     bool
}

func DefaultParams() Params {
	return Params{
		ShowShape1:     true,
		ShapeWidth:     290,
		ShapeHeight:    180,
		ShapeRadiusPct: 78,
		ShapeRoundness: 5,
		MergeRate:      0.05,

		RefThickness:       20,
		RefFactor:          1.4,
		RefDispersion:      7,
		RefFresnelRange:    30,
		RefFresnelHardness: 0.2,
		RefFresnelFactor:   0.2,

		GlareRange:          30,
		GlareConvergence:    0.5,
		GlareOppositeFactor: 0.8,
		GlareFactor:         0.9,
		GlareHardness:       0.2,
		GlareAngleDeg:       -45,

		BlurRadius: 12,
		BlurEdge:   true,
		TintR:      255,
		TintG:      255,
		TintB:      255,
		TintA:      0.03,
		Step:       9,

		SpringStiffness:  38,
		SpringDamping:    10,
		SpringSizeFactor: 10,
	}
}

func ChromeParams() Params {
	p := DefaultParams()
	p.RefDispersion = 18
	p.RefFresnelFactor = 0.48
	p.GlareFactor = 1.1
	p.GlareConvergence = 0.7
	p.GlareAngleDeg = -28
	p.TintA = 0.01
	return p
}

func SoftParams() Params {
	p := DefaultParams()
	p.RefThickness = 26
	p.RefFactor = 1.2
	p.RefDispersion = 3
	p.RefFresnelFactor = 0.1
	p.GlareFactor = 0.62
	p.GlareHardness = 0.12
	p.BlurRadius = 16
	p.TintA = 0.08
	return p
}

type Component struct {
	shader *ebiten.Shader
	params Params

	backgroundMode BackgroundMode
	bgOriginal     *ebiten.Image
	bgScaled       *ebiten.Image
	bgBlurred      *ebiten.Image

	screenW int
	screenH int

	mouseX        float64
	mouseY        float64
	mouseSpringX  float64
	mouseSpringY  float64
	mouseSpringVx float64
	mouseSpringVy float64

	lastBlurRadius int
	blurDirty      bool
}

func New() (*Component, error) {
	shader, err := ebiten.NewShader(liquidGlassShaderSource)
	if err != nil {
		return nil, err
	}
	return &Component{
		shader:         shader,
		params:         DefaultParams(),
		backgroundMode: BackgroundPhoto,
		lastBlurRadius: -1,
		blurDirty:      true,
	}, nil
}

func (c *Component) Dispose() {
	if c.bgOriginal != nil {
		c.bgOriginal.Dispose()
		c.bgOriginal = nil
	}
	if c.bgScaled != nil {
		c.bgScaled.Dispose()
		c.bgScaled = nil
	}
	if c.bgBlurred != nil {
		c.bgBlurred.Dispose()
		c.bgBlurred = nil
	}
}

func (c *Component) Params() *Params {
	return &c.params
}

func (c *Component) SetParams(p Params) {
	c.params = p
	c.blurDirty = true
}

func (c *Component) SetBackgroundMode(mode BackgroundMode) {
	if c.backgroundMode == mode {
		return
	}
	c.backgroundMode = mode
	c.rebuildBackground()
	c.blurDirty = true
}

func (c *Component) BackgroundMode() BackgroundMode {
	return c.backgroundMode
}

func (c *Component) SetBackgroundImage(img image.Image) {
	if c.bgOriginal != nil {
		c.bgOriginal.Dispose()
		c.bgOriginal = nil
	}
	if img != nil {
		c.bgOriginal = ebiten.NewImageFromImage(img)
	}
	c.rebuildBackground()
	c.blurDirty = true
}

func (c *Component) SetBackgroundImageFromPath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.SetBackgroundImage(decoded)
	return nil
}

func (c *Component) LoadBackgroundFromCandidates(paths []string) (string, error) {
	var lastErr error
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := c.SetBackgroundImageFromPath(path); err == nil {
			return path, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no valid background candidate")
	}
	return "", lastErr
}

func (c *Component) Resize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	c.screenW = w
	c.screenH = h
	c.rebuildBackground()
	c.blurDirty = true
	c.rebuildBlurBackgroundIfNeeded(true)
}

func (c *Component) SetMouse(x, y float64) {
	c.mouseX = clampFloat(x, 0, float64(maxInt(1, c.screenW)))
	c.mouseY = clampFloat(y, 0, float64(maxInt(1, c.screenH)))
}

func (c *Component) SetMouseSpringPosition(x, y float64) {
	c.mouseSpringX = x
	c.mouseSpringY = y
}

func (c *Component) ResetMouseSpringToMouse() {
	c.mouseSpringX = c.mouseX
	c.mouseSpringY = c.mouseY
	c.mouseSpringVx = 0
	c.mouseSpringVy = 0
}

func (c *Component) Mouse() (float64, float64) {
	return c.mouseX, c.mouseY
}

func (c *Component) MouseSpring() (float64, float64) {
	return c.mouseSpringX, c.mouseSpringY
}

func (c *Component) Update(dt time.Duration) {
	if c.screenW <= 0 || c.screenH <= 0 {
		return
	}
	delta := dt.Seconds()
	if delta <= 0 {
		delta = 1.0 / 60.0
	}
	if delta > 0.05 {
		delta = 0.05
	}

	targetX := c.mouseX
	targetY := c.mouseY
	if c.params.FreezeFollow {
		targetX = c.mouseSpringX
		targetY = c.mouseSpringY
	}

	k := math.Max(0.1, c.params.SpringStiffness)
	d := math.Max(0.0, c.params.SpringDamping)
	ax := (targetX-c.mouseSpringX)*k - c.mouseSpringVx*d
	ay := (targetY-c.mouseSpringY)*k - c.mouseSpringVy*d
	c.mouseSpringVx += ax * delta
	c.mouseSpringVy += ay * delta
	c.mouseSpringX += c.mouseSpringVx * delta
	c.mouseSpringY += c.mouseSpringVy * delta

	c.rebuildBlurBackgroundIfNeeded(false)
}

func (c *Component) Draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	if c.bgScaled != nil {
		screen.DrawImage(c.bgScaled, nil)
	} else {
		screen.Fill(color.RGBA{R: 18, G: 28, B: 45, A: 255})
	}

	if c.shader == nil || c.bgScaled == nil || c.bgBlurred == nil || c.screenW <= 0 || c.screenH <= 0 {
		return
	}

	stretchX := c.params.ShapeWidth + math.Abs(c.mouseSpringVx)*c.params.ShapeWidth*c.params.SpringSizeFactor/4500.0
	stretchY := c.params.ShapeHeight + math.Abs(c.mouseSpringVy)*c.params.ShapeHeight*c.params.SpringSizeFactor/4500.0
	stretchX = clampFloat(stretchX, 20, 900)
	stretchY = clampFloat(stretchY, 20, 900)
	shapeRadius := math.Min(stretchX, stretchY) * c.params.ShapeRadiusPct / 100.0 * 0.5
	shapeRadius = clampFloat(shapeRadius, 1, math.Min(stretchX, stretchY)*0.5-1)

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = c.bgScaled
	op.Images[1] = c.bgBlurred
	op.Uniforms = map[string]any{
		"Resolution": []float32{float32(c.screenW), float32(c.screenH)},
		"Mouse":      []float32{float32(c.mouseX), float32(c.mouseY)},
		"MouseSpring": []float32{
			float32(c.mouseSpringX),
			float32(c.mouseSpringY),
		},

		"ShowShape1":     boolToFloat32(c.params.ShowShape1),
		"ShapeWidth":     float32(stretchX),
		"ShapeHeight":    float32(stretchY),
		"ShapeRadius":    float32(shapeRadius),
		"ShapeRoundness": float32(c.params.ShapeRoundness),
		"MergeRate":      float32(c.params.MergeRate),

		"RefThickness":       float32(c.params.RefThickness),
		"RefFactor":          float32(c.params.RefFactor),
		"RefDispersion":      float32(c.params.RefDispersion),
		"RefFresnelRange":    float32(c.params.RefFresnelRange),
		"RefFresnelHardness": float32(c.params.RefFresnelHardness),
		"RefFresnelFactor":   float32(c.params.RefFresnelFactor),

		"GlareRange":          float32(c.params.GlareRange),
		"GlareConvergence":    float32(c.params.GlareConvergence),
		"GlareOppositeFactor": float32(c.params.GlareOppositeFactor),
		"GlareFactor":         float32(c.params.GlareFactor),
		"GlareHardness":       float32(c.params.GlareHardness),
		"GlareAngle":          float32(c.params.GlareAngleDeg / 180.0 * math.Pi),

		"BlurEdge": boolToFloat32(c.params.BlurEdge),
		"Tint": []float32{
			float32(c.params.TintR / 255.0),
			float32(c.params.TintG / 255.0),
			float32(c.params.TintB / 255.0),
			float32(c.params.TintA),
		},
		"Step": float32(c.params.Step),
	}
	screen.DrawRectShader(c.screenW, c.screenH, c.shader, op)
}

func (c *Component) rebuildBackground() {
	if c.screenW <= 0 || c.screenH <= 0 {
		return
	}
	if c.bgScaled != nil {
		c.bgScaled.Dispose()
		c.bgScaled = nil
	}

	bg := ebiten.NewImage(c.screenW, c.screenH)
	switch c.backgroundMode {
	case BackgroundGrid:
		drawGridBackground(bg)
	case BackgroundBars:
		drawBarsBackground(bg)
	case BackgroundHalf:
		drawHalfBackground(bg)
	default:
		if c.bgOriginal != nil {
			drawPhotoBackground(bg, c.bgOriginal, c.screenW, c.screenH)
		} else {
			drawGridBackground(bg)
		}
	}
	c.bgScaled = bg
}

func drawPhotoBackground(dst, src *ebiten.Image, w, h int) {
	bw := src.Bounds().Dx()
	bh := src.Bounds().Dy()
	if bw <= 0 || bh <= 0 {
		drawGridBackground(dst)
		return
	}
	scale := math.Max(float64(w)/float64(bw), float64(h)/float64(bh))
	dw := float64(bw) * scale
	dh := float64(bh) * scale
	tx := (float64(w) - dw) * 0.5
	ty := (float64(h) - dh) * 0.5

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(tx, ty)
	dst.DrawImage(src, op)
}

func drawGridBackground(dst *ebiten.Image) {
	w := float64(dst.Bounds().Dx())
	h := float64(dst.Bounds().Dy())
	dst.Fill(color.RGBA{R: 245, G: 247, B: 250, A: 255})
	cell := 40.0
	for y := 0.0; y < h; y += cell {
		for x := 0.0; x < w; x += cell {
			if int(x/cell+y/cell)%2 == 0 {
				ebitenutil.DrawRect(dst, x, y, cell, cell, color.RGBA{R: 232, G: 236, B: 242, A: 255})
			}
		}
	}
}

func drawBarsBackground(dst *ebiten.Image) {
	w := float64(dst.Bounds().Dx())
	h := float64(dst.Bounds().Dy())
	dst.Fill(color.RGBA{R: 250, G: 250, B: 252, A: 255})
	for i := 0.0; i < w; i += 24 {
		alpha := uint8(45)
		if int(i/24)%3 == 0 {
			alpha = 80
		}
		ebitenutil.DrawRect(dst, i, 0, 12, h, color.RGBA{R: 170, G: 184, B: 210, A: alpha})
	}
}

func drawHalfBackground(dst *ebiten.Image) {
	w := float64(dst.Bounds().Dx())
	h := float64(dst.Bounds().Dy())
	ebitenutil.DrawRect(dst, 0, 0, w, h*0.5, color.RGBA{R: 234, G: 241, B: 252, A: 255})
	ebitenutil.DrawRect(dst, 0, h*0.5, w, h*0.5, color.RGBA{R: 48, G: 62, B: 88, A: 255})
}

func (c *Component) rebuildBlurBackgroundIfNeeded(force bool) {
	if c.bgScaled == nil {
		return
	}
	radius := int(math.Round(clampFloat(c.params.BlurRadius, 1, 20)))
	if !force && !c.blurDirty && radius == c.lastBlurRadius {
		return
	}
	c.lastBlurRadius = radius
	c.blurDirty = false

	if c.bgBlurred != nil {
		c.bgBlurred.Dispose()
		c.bgBlurred = nil
	}
	c.bgBlurred = filters.BlurImageShader(c.bgScaled, float64(radius))
}

func boolToFloat32(v bool) float32 {
	if v {
		return 1
	}
	return 0
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
