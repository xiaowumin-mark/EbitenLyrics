package lyrics

// 文件说明：音节图像资源生成与绘制实现。
// 主要职责：创建遮罩、渐变、高亮资源并按偏移量进行混合绘制。

import (
	ft "EbitenLyrics/font"
	"EbitenLyrics/lp"
	"errors"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type SyllableImage struct {
	TextMask               *ebiten.Image
	GradientImage          *ebiten.Image
	HighlightGradientImage *ebiten.Image
	textMaskKey            textMaskKey
	gradientKey            gradientKey
	highlightGradientKey   gradientKey
	hasTextKey             bool
	hasGradKey             bool
	hasHighlightGradKey    bool
	Offset                 float64
	Width                  float64
	Height                 float64
	StartColor             color.RGBA
	EndColor               color.RGBA
	Fd                     float64
	Text                   string
	FontManager            *ft.FontManager
	FontRequest            ft.FontRequest
	FontSize               float64
	tempImage              *ebiten.Image
}

func CreateSyllableImage(
	syllable string,
	fontManager *ft.FontManager,
	req ft.FontRequest,
	fontSize float64,
	fd float64,
	startColor, endColor color.RGBA,
) (*SyllableImage, error) {
	if fontManager == nil {
		return nil, errors.New("font manager is nil")
	}
	font, err := fontManager.GetFaceForText(req, fontSize, syllable)
	if err != nil || font == nil {
		return nil, errors.New("font face is nil")
	}

	tw, th := text.Measure(syllable, font, 1.0)
	tw = lp.FromLP(tw)
	th = lp.FromLP(th)
	if tw <= 0 {
		tw = 1
	}
	if th <= 0 {
		th = 1
	}
	_, _, _, offset := generateBackgroundFadeStyle(tw, th, fd)

	return &SyllableImage{
		Offset:      offset,
		Width:       tw,
		Height:      th,
		StartColor:  startColor,
		EndColor:    endColor,
		Fd:          fd,
		FontManager: fontManager,
		FontRequest: req.Normalized(),
		FontSize:    fontSize,
		Text:        syllable,
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
	if s == nil {
		return
	}
	font := s.resolveFace()
	if font == nil {
		return
	}
	tw, th := text.Measure(s.Text, font, 1.0)
	tw = lp.FromLP(tw)
	th = lp.FromLP(th)
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
	if s == nil {
		return false
	}
	if s.Width <= 0 || s.Height <= 0 {
		s.updateMetrics()
	}

	targetW := safeImageLength(s.Width)
	targetH := safeImageLength(s.Height)

	if s.TextMask == nil {
		img, key := acquireTextMask(s.Text, s.FontManager, s.FontRequest, s.FontSize, s.Width, s.Height)
		s.TextMask = img
		s.textMaskKey = key
		s.hasTextKey = img != nil
	}
	if s.GradientImage == nil {
		img, key := acquireGradient(targetW, targetH, s.Fd, s.StartColor, s.EndColor)
		s.GradientImage = img
		s.gradientKey = key
		s.hasGradKey = img != nil
	}
	if s.HighlightGradientImage == nil {
		startColor, endColor := s.highlightGradientColors()
		img, key := acquireGradient(targetW, targetH, s.Fd, startColor, endColor)
		s.HighlightGradientImage = img
		s.highlightGradientKey = key
		s.hasHighlightGradKey = img != nil
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

	return s.TextMask != nil && s.GradientImage != nil && s.HighlightGradientImage != nil && s.tempImage != nil
}

func (s *SyllableImage) releaseTextMask() {
	if s == nil {
		return
	}
	if s.hasTextKey {
		releaseTextMask(s.textMaskKey)
	}
	s.TextMask = nil
	s.textMaskKey = textMaskKey{}
	s.hasTextKey = false
}

func (s *SyllableImage) releaseGradient() {
	if s == nil {
		return
	}
	if s.hasGradKey {
		releaseGradient(s.gradientKey)
	}
	s.GradientImage = nil
	s.gradientKey = gradientKey{}
	s.hasGradKey = false
}

func (s *SyllableImage) releaseHighlightGradient() {
	if s == nil {
		return
	}
	if s.hasHighlightGradKey {
		releaseGradient(s.highlightGradientKey)
	}
	s.HighlightGradientImage = nil
	s.highlightGradientKey = gradientKey{}
	s.hasHighlightGradKey = false
}

func (s *SyllableImage) resetResources() {
	if s == nil {
		return
	}
	s.releaseTextMask()
	s.releaseGradient()
	s.releaseHighlightGradient()
	if s.tempImage != nil {
		s.tempImage.Deallocate()
		s.tempImage = nil
	}
}

func (s *SyllableImage) Draw(img *ebiten.Image, offset float64, alpha float64, pos *Position) {
	s.drawMasked(img, s.GradientImage, offset, alpha, pos, ebiten.BlendSourceOver)
}

func (s *SyllableImage) DrawHighlight(img *ebiten.Image, offset float64, alpha float64, pos *Position) {
	s.drawMasked(img, s.HighlightGradientImage, offset, alpha, pos, ebiten.BlendLighter)
}

func (s *SyllableImage) drawMasked(img, gradient *ebiten.Image, offset float64, alpha float64, pos *Position, blend ebiten.Blend) {
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
	if gradient == nil {
		return
	}

	s.tempImage.Clear()
	s.tempImage.DrawImage(s.TextMask, &ebiten.DrawImageOptions{})

	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceIn
	op.GeoM.Translate(lp.LP(offset), 0)
	op.GeoM.Scale(1, math.Max(1, lp.LP(s.Height)))
	op.ColorScale.ScaleAlpha(float32(alpha))
	s.tempImage.DrawImage(gradient, op)

	finalop := &ebiten.DrawImageOptions{}
	finalop.GeoM = TransformToGeoM(pos)
	drawImageResample4x4(img, s.tempImage, finalop.GeoM, 1, blend)
}

func (s *SyllableImage) Dispose() {
	if s == nil {
		return
	}
	s.resetResources()
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
	if s == nil {
		return
	}

	s.resetResources()
	s.updateMetrics()
	s.ensureResources()
}

func (s *SyllableImage) SetText(t string) {
	if s.Text == t {
		return
	}
	s.Text = t
	s.resetResources()
	s.updateMetrics()
}

func (s *SyllableImage) SetFont(fontManager *ft.FontManager, req ft.FontRequest, fontSize float64) {
	if s == nil {
		return
	}
	if s.FontManager == fontManager && s.FontRequest.CacheKey() == req.CacheKey() && s.FontSize == fontSize {
		return
	}
	s.FontManager = fontManager
	s.FontRequest = req.Normalized()
	s.FontSize = fontSize
	s.resetResources()
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

	s.releaseGradient()
	s.releaseHighlightGradient()
}

func (s *SyllableImage) SetStartColor(c color.RGBA) {
	if s.StartColor == c {
		return
	}
	s.StartColor = c
	s.rebuildGradient()
}

func (s *SyllableImage) SetEndColor(c color.RGBA) {
	if s.EndColor == c {
		return
	}
	s.EndColor = c
	s.rebuildGradient()
}

func (s *SyllableImage) SetFd(fd float64) {
	if s.Fd == fd {
		return
	}
	s.Fd = fd
	s.updateMetrics()
	s.releaseGradient()
	s.releaseHighlightGradient()
}

func (s *SyllableImage) GetText() string {
	return s.Text
}

func (s *SyllableImage) GetFont() text.Face {
	return s.resolveFace()
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

func (s *SyllableImage) highlightGradientColors() (color.RGBA, color.RGBA) {
	deltaAlpha := subtractChannel(s.StartColor.A, s.EndColor.A)
	return color.RGBA{
			R: s.StartColor.R,
			G: s.StartColor.G,
			B: s.StartColor.B,
			A: deltaAlpha,
		},
		color.RGBA{
			R: s.EndColor.R,
			G: s.EndColor.G,
			B: s.EndColor.B,
			A: 0,
		}
}

func subtractChannel(a, b uint8) uint8 {
	if a <= b {
		return 0
	}
	return a - b
}

func (s *SyllableImage) resolveFace() text.Face {
	if s == nil || s.FontManager == nil || s.FontSize <= 0 {
		return nil
	}
	face, err := s.FontManager.GetFaceForText(s.FontRequest, s.FontSize, s.Text)
	if err != nil {
		return nil
	}
	return face
}
