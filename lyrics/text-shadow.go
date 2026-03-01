package lyrics

import (
	"EbitenLyrics/filters"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type TextShadow struct {
	Color           color.RGBA
	LastBlur        float64
	Blur            float64
	Margin          float64
	Text            string
	TextFace        *text.GoTextFaceSource
	Size            float64
	OriginImage     *ebiten.Image
	Image           *ebiten.Image
	TWidth, THeight float64
	Width, Height   float64
	Alpha           float64
}

func NewTextShadow(texts string, textFace *text.GoTextFaceSource, size float64) *TextShadow {
	tw, th := text.Measure(texts, &text.GoTextFace{
		Source: textFace,
		Size:   size,
	}, 1.0)

	return &TextShadow{
		Text:     texts,
		TextFace: textFace,
		Color:    color.RGBA{255, 255, 255, 255},
		Blur:     0.0,
		Margin:   50.0,
		TWidth:   tw,
		THeight:  th,
		Width:    tw + 50*2,
		Height:   th + 50*2,
		Size:     size,
		Alpha:    0,
		LastBlur: -1,
	}
}

func (ts *TextShadow) ensureOriginImage() bool {
	if ts == nil || ts.TextFace == nil || ts.Size <= 0 {
		return false
	}
	if ts.OriginImage != nil {
		return true
	}

	ts.OriginImage = ebiten.NewImage(safeImageLength(ts.Width), safeImageLength(ts.Height))
	op := &text.DrawOptions{}
	op.GeoM.Translate(ts.Margin, ts.Margin)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(ts.OriginImage, ts.Text, &text.GoTextFace{
		Source: ts.TextFace,
		Size:   ts.Size,
	}, op)

	return true
}

func (ts *TextShadow) updateImage() {
	if !ts.ensureOriginImage() {
		return
	}

	if ts.Blur == ts.LastBlur && ts.Image != nil {
		return
	}

	if ts.Blur <= 0 {
		if ts.Image != nil && ts.Image != ts.OriginImage {
			ts.Image.Deallocate()
		}
		ts.Image = ts.OriginImage
		ts.LastBlur = ts.Blur
		return
	}

	blurred := filters.BlurImageShader(ts.OriginImage, ts.Blur)
	if ts.Image != nil && ts.Image != ts.OriginImage {
		ts.Image.Deallocate()
	}
	ts.Image = blurred
	ts.LastBlur = ts.Blur
}

func (ts *TextShadow) Draw(screen *ebiten.Image, p *Position) {
	if ts == nil || screen == nil {
		return
	}
	ts.updateImage()
	if ts.Image == nil {
		return
	}

	opt := NewPosition(0, 0, 0, 0)
	if p != nil {
		opt = *p
	}
	opt.SetX(opt.GetX() - ts.Margin - 2)
	opt.SetY(opt.GetY() - ts.Margin - 2)

	op := &ebiten.DrawImageOptions{}
	op.GeoM = TransformToGeoM(&opt)
	op.ColorScale.ScaleAlpha(float32(ts.Alpha))
	op.Filter = ebiten.FilterLinear
	op.Blend = ebiten.BlendLighter
	screen.DrawImage(ts.Image, op)
}

func (ts *TextShadow) Dispose() {
	if ts == nil {
		return
	}
	if ts.Image != nil && ts.Image != ts.OriginImage {
		ts.Image.Deallocate()
	}
	if ts.OriginImage != nil {
		ts.OriginImage.Deallocate()
	}
	ts.OriginImage = nil
	ts.Image = nil
	ts.LastBlur = -1
}
