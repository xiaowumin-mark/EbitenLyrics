package lyrics

import (
	"EbitenLyrics/filters"
	_ "EbitenLyrics/filters"
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
	OriginImage     *ebiten.Image // 源图片
	Image           *ebiten.Image // 模糊图片
	TWidth, THeight float64
	Width, Height   float64
	Alpha           float64
}

func NewTextShadow(texts string, textFace *text.GoTextFaceSource, size float64) *TextShadow {
	// 计算文字的宽高
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
		LastBlur: 0,
	}
}

func (ts *TextShadow) Draw(screen *ebiten.Image, X, Y float64) {
	if ts.OriginImage == nil {
		ts.OriginImage = ebiten.NewImage(int(ts.Width), int(ts.Height))
		op := &text.DrawOptions{}
		op.GeoM.Translate(ts.Margin, ts.Margin)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(ts.OriginImage, ts.Text, &text.GoTextFace{
			Source: ts.TextFace,
			Size:   ts.Size,
		}, op)
	}
	if ts.Blur != ts.LastBlur {
		ts.Image = filters.BlurImageShader(ts.OriginImage, ts.Blur)
		ts.LastBlur = ts.Blur
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(X-ts.Margin, Y-ts.Margin)
	op.ColorScale.ScaleAlpha(float32(ts.Alpha))
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(
		ts.Image,
		op,
	)

}

func (ts *TextShadow) Dispose() {
	//ts.OriginImage.Deallocate()
	if ts.OriginImage != nil {
		ts.OriginImage.Deallocate()
	}

	//ts.Image.Deallocate()
	if ts.Image != nil {
		ts.Image.Deallocate()
	}
	ts.OriginImage = nil
	ts.Image = nil
}
