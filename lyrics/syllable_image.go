package lyrics

import (
	"image/color"

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
	tempImage     *ebiten.Image // 【新增】用于合成的缓存图像
}

// 实现创建音节图像
func CreateSyllableImage(syllable string, font text.Face, fd float64,
	startColor, endColor color.RGBA,
) (*SyllableImage, error) {
	// 获取音节图像的宽高
	tw, th := text.Measure(
		syllable,
		font,
		1.0,
	)

	// 生成文字蒙版
	textMask := CreateTextMask(syllable, font, tw, th)
	// 生成渐变图像
	gradientImage, offset := CreateGradientImage(
		int(tw),
		int(th),
		fd,
		startColor,
		endColor,
	)
	return &SyllableImage{
		TextMask:      textMask,
		GradientImage: gradientImage,
		Offset:        offset,
		Width:         tw,
		Height:        th,
		StartColor:    startColor,
		EndColor:      endColor,
		Fd:            fd,
		Font:          &font,
		Text:          syllable,
		tempImage:     ebiten.NewImage(int(tw), int(th)),
	}, nil
}

func CreateTextMask(syllable string, font text.Face, w, h float64) *ebiten.Image {

	// 生成文字蒙版
	textMask := ebiten.NewImage(int(w), int(h))
	textMask.Fill(color.Transparent)
	opts := &text.DrawOptions{}
	opts.ColorScale.ScaleWithColor(color.White)
	text.Draw(
		textMask,
		syllable,
		font,
		opts,
	)
	return textMask
}

func CreateGradientImage(width, height int, fd float64, startColor, endColor color.RGBA) (*ebiten.Image, float64) {

	// 生成渐变图像
	from, to, gradient, offset := generateBackgroundFadeStyle(
		float64(width),
		float64(height),
		fd,
	)
	gradientWidth := float64(width) * gradient
	gradientImage := ebiten.NewImage(int(gradientWidth), 1)

	startPixel := gradientWidth * from
	endPixel := gradientWidth * to

	for x := 0; x < int(gradientWidth); x++ {
		var c color.RGBA
		if float64(x) < startPixel {
			//c = startRGBA
			a := float64(startColor.A) / 255.0
			c = color.RGBA{
				R: uint8(float64(startColor.R) * a),
				G: uint8(float64(startColor.G) * a),
				B: uint8(float64(startColor.B) * a),
				A: startColor.A,
			}
		} else if float64(x) > endPixel {
			a := float64(endColor.A) / 255.0
			c = color.RGBA{
				R: uint8(float64(endColor.R) * a),
				G: uint8(float64(endColor.G) * a),
				B: uint8(float64(endColor.B) * a),
				A: endColor.A,
			}
		} else {
			t := (float64(x) - startPixel) / (endPixel - startPixel)
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

func (s *SyllableImage) Draw(img *ebiten.Image, offset float64, alpha float64) {
	if s.TextMask == nil || s.GradientImage == nil {
		// 重绘
		s.Redraw()
	}
	s.tempImage.Clear()
	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear
	s.tempImage.DrawImage(s.TextMask, opts)

	// 绘制渐变图像
	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceIn
	op.GeoM.Translate(offset, 0)
	op.GeoM.Scale(1, s.Height*1.5) // 高度缩放到和文字蒙版一样高
	op.ColorScale.ScaleAlpha(float32(alpha))
	s.tempImage.DrawImage(s.GradientImage, op)

	finalop := &ebiten.DrawImageOptions{}
	finalop.Filter = ebiten.FilterLinear
	img.DrawImage(s.tempImage, finalop)

}

func (s *SyllableImage) Dispose() {
	s.TextMask.Deallocate()
	s.GradientImage.Deallocate()
	s.tempImage.Deallocate()
	s.TextMask = nil
	s.GradientImage = nil
	s.tempImage = nil
	s.Font = nil
	s.Text = ""
	s.StartColor = color.RGBA{}
	s.EndColor = color.RGBA{}
	s.Fd = 0
	s = nil
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

	fadeWidth := elementHeight * fadeRatio
	widthRatio := fadeWidth / elementWidth

	totalAspect := 2 + widthRatio
	widthInTotal := widthRatio / totalAspect
	leftPos := (1 - widthInTotal) / 2

	from := leftPos
	to := (leftPos + widthInTotal)

	totalPxWidth := elementWidth + fadeWidth
	return from, to, totalAspect, -totalPxWidth
}

// 重绘函数
func (s *SyllableImage) Redraw() {
	s.tempImage.Deallocate()
	s.GradientImage.Deallocate()
	s.TextMask.Deallocate()
	// 重新创建文字蒙版和渐变图像
	// 计算文字宽高
	tw, th := text.Measure(
		s.Text,
		*s.Font,
		1.0,
	)
	// 生成文字蒙版
	s.TextMask = CreateTextMask(s.Text, *s.Font, tw, th)
	// 生成渐变图像
	s.GradientImage, s.Offset = CreateGradientImage(
		int(tw),
		int(th),
		s.Fd,
		s.StartColor,
		s.EndColor,
	)
	s.Width = tw
	s.Height = th
	s.tempImage = ebiten.NewImage(int(tw), int(th))
}

// 将结构体中的属性写成函数 并重绘
func (s *SyllableImage) SetText(t string) {
	s.Text = t
	s.Redraw()
}
func (s *SyllableImage) SetFont(f text.Face) {
	s.Font = &f
	s.Redraw()

}
func (s *SyllableImage) SetStartColor(c color.RGBA) {
	s.StartColor = c
	// 只需要重绘渐变图像
	s.GradientImage.Clear()
	s.GradientImage, s.Offset = CreateGradientImage(
		int(s.Width),
		int(s.Height),
		s.Fd,
		s.StartColor,
		s.EndColor,
	)
}
func (s *SyllableImage) SetEndColor(c color.RGBA) {
	s.EndColor = c
	// 只需要重绘渐变图像
	s.GradientImage.Clear()
	s.GradientImage, s.Offset = CreateGradientImage(
		int(s.Width),
		int(s.Height),
		s.Fd,
		s.StartColor,
		s.EndColor,
	)
}
func (s *SyllableImage) SetFd(fd float64) {
	s.Fd = fd
	// 只需要重绘渐变图像
	s.GradientImage.Clear()
	s.GradientImage, s.Offset = CreateGradientImage(
		int(s.Width),
		int(s.Height),
		s.Fd,
		s.StartColor,
		s.EndColor,
	)
}
func (s *SyllableImage) GetText() string {
	return s.Text
}
func (s *SyllableImage) GetFont() text.Face {
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
