package lyrics

import (
	"errors"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type SyllableImage struct {
	TextMask      *ebiten.Image
	GradientImage *ebiten.Image
	Offset        float64
	Width         float64
	Height        float64
	StartColor    color.RGBA
	EndColor      color.RGBA
	Fd            float64
	Text          string
	Font          *text.Face
	tempImage     *ebiten.Image
}

func CreateSyllableImage(
	syllable string,
	font text.Face,
	fd float64,
	startColor, endColor color.RGBA,
) (*SyllableImage, error) {
	if font == nil {
		return nil, errors.New("font is nil")
	}

	tw, th := text.Measure(syllable, font, 1.0)
	if tw <= 0 {
		tw = 1
	}
	if th <= 0 {
		th = 1
	}
	_, _, _, offset := generateBackgroundFadeStyle(tw, th, fd)

	return &SyllableImage{
		Offset:     offset,
		Width:      tw,
		Height:     th,
		StartColor: startColor,
		EndColor:   endColor,
		Fd:         fd,
		Font:       &font,
		Text:       syllable,
	}, nil
}

func CreateTextMask(syllable string, font text.Face, w, h float64) *ebiten.Image {
	if syllable == "" {
		syllable = " "
	}
	iw := safeImageLength(w)
	ih := safeImageLength(h)

	textMask := ebiten.NewImage(iw, ih)
	textMask.Fill(color.Transparent)

	opts := &text.DrawOptions{}
	opts.ColorScale.ScaleWithColor(color.White)
	text.Draw(textMask, syllable, font, opts)

	return textMask
}

func CreateGradientImage(width, height int, fd float64, startColor, endColor color.RGBA) (*ebiten.Image, float64) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	from, to, gradient, offset := generateBackgroundFadeStyle(
		float64(width),
		float64(height),
		fd,
	)
	gradientWidth := math.Max(1, float64(width)*gradient)
	gradientImage := ebiten.NewImage(safeImageLength(gradientWidth), 1)

	startPixel := gradientWidth * from
	endPixel := gradientWidth * to
	denominator := endPixel - startPixel
	if math.Abs(denominator) < 1e-6 {
		denominator = 1
	}

	for x := 0; x < safeImageLength(gradientWidth); x++ {
		var c color.RGBA
		xf := float64(x)
		switch {
		case xf < startPixel:
			a := float64(startColor.A) / 255.0
			c = color.RGBA{
				R: uint8(float64(startColor.R) * a),
				G: uint8(float64(startColor.G) * a),
				B: uint8(float64(startColor.B) * a),
				A: startColor.A,
			}
		case xf > endPixel:
			a := float64(endColor.A) / 255.0
			c = color.RGBA{
				R: uint8(float64(endColor.R) * a),
				G: uint8(float64(endColor.G) * a),
				B: uint8(float64(endColor.B) * a),
				A: endColor.A,
			}
		default:
			t := (xf - startPixel) / denominator
			a := uint8(float64(startColor.A) + (float64(endColor.A)-float64(startColor.A))*t)
			r := uint8((float64(startColor.R) + (float64(endColor.R)-float64(startColor.R))*t) * (float64(a) / 255.0))
			g := uint8((float64(startColor.G) + (float64(endColor.G)-float64(startColor.G))*t) * (float64(a) / 255.0))
			b := uint8((float64(startColor.B) + (float64(endColor.B)-float64(startColor.B))*t) * (float64(a) / 255.0))
			c = color.RGBA{R: r, G: g, B: b, A: a}
		}
		gradientImage.Set(x, 0, c)
	}

	return gradientImage, offset
}

func (s *SyllableImage) updateMetrics() {
	if s == nil || s.Font == nil || *s.Font == nil {
		return
	}
	tw, th := text.Measure(s.Text, *s.Font, 1.0)
	if tw <= 0 {
		tw = 1
	}
	if th <= 0 {
		th = 1
	}
	s.Width = tw
	s.Height = th
	_, _, _, offset := generateBackgroundFadeStyle(tw, th, s.Fd)
	s.Offset = offset
}

func (s *SyllableImage) ensureResources() bool {
	if s == nil || s.Font == nil || *s.Font == nil {
		return false
	}
	if s.Width <= 0 || s.Height <= 0 {
		s.updateMetrics()
	}

	targetW := safeImageLength(s.Width)
	targetH := safeImageLength(s.Height)

	if s.TextMask == nil {
		s.TextMask = CreateTextMask(s.Text, *s.Font, s.Width, s.Height)
	}
	if s.GradientImage == nil {
		s.GradientImage, s.Offset = CreateGradientImage(
			targetW,
			targetH,
			s.Fd,
			s.StartColor,
			s.EndColor,
		)
	}
	if s.tempImage != nil {
		w, h := s.tempImage.Size()
		if w != targetW || h != targetH {
			s.tempImage.Deallocate()
			s.tempImage = nil
		}
	}
	if s.tempImage == nil {
		s.tempImage = ebiten.NewImage(targetW, targetH)
	}

	return s.TextMask != nil && s.GradientImage != nil && s.tempImage != nil
}

func (s *SyllableImage) Draw(img *ebiten.Image, offset float64, alpha float64, pos *Position) {
	// Guard against transient resource disposal during hot-reload paths.
	defer func() {
		_ = recover()
	}()

	if s == nil || img == nil || pos == nil {
		return
	}
	if !s.ensureResources() {
		return
	}

	s.tempImage.Clear()
	s.tempImage.DrawImage(s.TextMask, &ebiten.DrawImageOptions{})

	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceIn
	op.GeoM.Translate(offset, 0)
	op.GeoM.Scale(1, math.Max(1, s.Height))
	op.ColorScale.ScaleAlpha(float32(alpha))
	s.tempImage.DrawImage(s.GradientImage, op)

	finalop := &ebiten.DrawImageOptions{}
	finalop.Filter = ebiten.FilterLinear
	finalop.GeoM = TransformToGeoM(pos)
	img.DrawImage(s.tempImage, finalop)
}

func (s *SyllableImage) Dispose() {
	if s == nil {
		return
	}
	if s.TextMask != nil {
		s.TextMask.Deallocate()
		s.TextMask = nil
	}
	if s.GradientImage != nil {
		s.GradientImage.Deallocate()
		s.GradientImage = nil
	}
	if s.tempImage != nil {
		s.tempImage.Deallocate()
		s.tempImage = nil
	}
}

func (s *SyllableImage) GetWidth() float64 {
	return s.Width
}

func (s *SyllableImage) GetHeight() float64 {
	return s.Height
}

func (s *SyllableImage) GetOffset() float64 {
	return s.Offset
}

func generateBackgroundFadeStyle(elementWidth, elementHeight, fadeRatio float64) (float64, float64, float64, float64) {
	if elementWidth <= 0 {
		elementWidth = 1
	}
	if elementHeight <= 0 {
		elementHeight = 1
	}
	if fadeRatio <= 0 {
		fadeRatio = 0.0001
	}

	fadeWidth := elementHeight * fadeRatio
	widthRatio := fadeWidth / elementWidth

	totalAspect := 2 + widthRatio
	if totalAspect <= 0 {
		totalAspect = 2
	}
	widthInTotal := widthRatio / totalAspect
	leftPos := (1 - widthInTotal) / 2

	from := leftPos
	to := leftPos + widthInTotal
	totalPxWidth := elementWidth + fadeWidth
	return from, to, totalAspect, -totalPxWidth
}

func (s *SyllableImage) Redraw() {
	if s == nil || s.Font == nil || *s.Font == nil {
		return
	}

	s.Dispose()
	s.updateMetrics()
	s.ensureResources()
}

func (s *SyllableImage) SetText(t string) {
	s.Text = t
	s.Dispose()
	s.updateMetrics()
}

func (s *SyllableImage) SetFont(f text.Face) {
	if s == nil {
		return
	}
	s.Font = &f
	s.Dispose()
	s.updateMetrics()
}

func (s *SyllableImage) rebuildGradient() {
	if s == nil {
		return
	}
	if s.Width <= 0 || s.Height <= 0 {
		s.updateMetrics()
	}
	_, _, _, offset := generateBackgroundFadeStyle(s.Width, s.Height, s.Fd)
	s.Offset = offset

	if s.GradientImage == nil {
		return
	}
	s.GradientImage.Deallocate()
	s.GradientImage = nil
	s.GradientImage, s.Offset = CreateGradientImage(
		safeImageLength(s.Width),
		safeImageLength(s.Height),
		s.Fd,
		s.StartColor,
		s.EndColor,
	)
}

func (s *SyllableImage) SetStartColor(c color.RGBA) {
	s.StartColor = c
	s.rebuildGradient()
}

func (s *SyllableImage) SetEndColor(c color.RGBA) {
	s.EndColor = c
	s.rebuildGradient()
}

func (s *SyllableImage) SetFd(fd float64) {
	s.Fd = fd
	s.rebuildGradient()
}

func (s *SyllableImage) GetText() string {
	return s.Text
}

func (s *SyllableImage) GetFont() text.Face {
	if s == nil || s.Font == nil {
		return nil
	}
	return *s.Font
}

func (s *SyllableImage) GetStartColor() color.RGBA {
	return s.StartColor
}

func (s *SyllableImage) GetEndColor() color.RGBA {
	return s.EndColor
}

func (s *SyllableImage) GetFd() float64 {
	return s.Fd
}

func (s *SyllableImage) GetTextMask() *ebiten.Image {
	return s.TextMask
}

func (s *SyllableImage) GetGradientImage() *ebiten.Image {
	return s.GradientImage
}

func (s *SyllableImage) GetTempImage() *ebiten.Image {
	return s.tempImage
}
