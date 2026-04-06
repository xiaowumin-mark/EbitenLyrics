package lyrics

// 文件说明：文本阴影效果封装。
// 主要职责：缓存阴影位图并按位置参数绘制到屏幕。

import (
	"EbitenLyrics/filters"
	ft "EbitenLyrics/font"
	"EbitenLyrics/lp"
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
	FontManager     *ft.FontManager
	FontRequest     ft.FontRequest
	Size            float64
	OriginImage     *ebiten.Image
	Image           *ebiten.Image
	TWidth, THeight float64
	Width, Height   float64
	Alpha           float64
}

func NewTextShadow(texts string, fontManager *ft.FontManager, req ft.FontRequest, size float64) *TextShadow {
	var face text.Face
	if fontManager != nil {
		face, _ = fontManager.GetFaceForText(req, size, texts)
	}
	tw, th := 1.0, 1.0
	if face != nil {
		tw, th = text.Measure(texts, face, 1.0)
		tw = lp.FromLP(tw)
		th = lp.FromLP(th)
	}

	return &TextShadow{
		Text:        texts,
		FontManager: fontManager,
		FontRequest: req.Normalized(),
		Color:       color.RGBA{255, 255, 255, 255},
		Blur:        0.0,
		Margin:      50.0,
		TWidth:      tw,
		THeight:     th,
		Width:       tw + 50*2,
		Height:      th + 50*2,
		Size:        size,
		Alpha:       0,
		LastBlur:    -1,
	}
}

func (ts *TextShadow) ensureOriginImage() bool {
	if ts == nil || ts.FontManager == nil || ts.Size <= 0 {
		return false
	}
	if ts.OriginImage != nil {
		return true
	}
	face, err := ts.FontManager.GetFaceForText(ts.FontRequest, ts.Size, ts.Text)
	if err != nil || face == nil {
		return false
	}

	ts.OriginImage = ebiten.NewImage(safeImageLength(ts.Width), safeImageLength(ts.Height))
	op := &text.DrawOptions{}
	op.GeoM.Translate(lp.LP(ts.Margin), lp.LP(ts.Margin))
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(ts.OriginImage, ts.Text, face, op)

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

	drawImageResample4x4(
		screen,
		ts.Image,
		TransformToGeoM(&opt),
		float32(ts.Alpha),
		ebiten.BlendLighter,
	)
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
