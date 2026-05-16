package lyrics

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	"github.com/xiaowumin-mark/EbitenLyrics/filters"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
)

func newBottomLine(fontSize, width float64) BottomLine {
	if width < 0 {
		width = 0
	}
	bottom := BottomLine{
		Position:        NewPosition(0, 0, width, 0),
		LineSize:        [2]float64{width, 0},
		FontSize:        fontSize,
		ContentFontSize: bottomLineContentFontSize(fontSize),
		PaddingLeft:     bottomLinePaddingLeft(fontSize, width),
		PaddingTop:      bottomLinePaddingTop(fontSize),
		ImageDirty:      true,
	}
	bottom.ensureSprings()
	return bottom
}

func (b *BottomLine) Resize(width float64) {
	if b == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	b.Position.SetW(width)
	b.LineSize[0] = width
	b.PaddingLeft = bottomLinePaddingLeft(b.FontSize, width)
	b.measure()
	b.markImageDirty()
}

func (b *BottomLine) Dispose() {
	if b == nil {
		return
	}
	b.clearImage()
}

func (b *BottomLine) SetFocused(focused bool) {
	if b == nil {
		return
	}
	b.Focused = focused
}

func (b *BottomLine) SetText(text string, fontSize float64) {
	if b == nil {
		return
	}
	if fontSize <= 0 {
		fontSize = b.FontSize
	}
	if fontSize <= 0 {
		fontSize = 24
	}
	b.FontSize = fontSize
	b.ContentFontSize = bottomLineContentFontSize(fontSize)
	b.PaddingLeft = bottomLinePaddingLeft(fontSize, b.LineSize[0])
	b.PaddingTop = bottomLinePaddingTop(fontSize)
	b.Text = text
	b.Active = strings.TrimSpace(text) != ""
	b.measure()
	b.markImageDirty()
}

func (b *BottomLine) HasContent() bool {
	return b != nil && b.Active && strings.TrimSpace(b.Text) != "" && b.LineSize[1] > 0
}

func (b *BottomLine) SetFont(fontManager *ft.FontManager, req ft.FontRequest, fontSize float64) {
	if b == nil {
		return
	}
	if fontSize <= 0 {
		fontSize = b.FontSize
	}
	if fontSize <= 0 {
		fontSize = 24
	}
	b.FontManager = fontManager
	b.FontRequest = req.Normalized()
	b.FontSize = fontSize
	b.ContentFontSize = bottomLineContentFontSize(fontSize)
	b.PaddingLeft = bottomLinePaddingLeft(fontSize, b.LineSize[0])
	b.PaddingTop = bottomLinePaddingTop(fontSize)
	b.measure()
	b.markImageDirty()
}

func (l *Lyrics) SetBottomLineText(text string) {
	if l == nil {
		return
	}
	l.Bottom.SetFont(l.FontManager, l.FontRequest, l.DotsFontSize())
	l.Bottom.SetText(text, l.DotsFontSize())
	lineAnimationLayer.requestScrollRelayout(l)
}

func (l *Lyrics) ClearBottomLine() {
	l.SetBottomLineText("")
}

func (b *BottomLine) ensureSprings() {
	if b == nil {
		return
	}
	if b.PosXSpring == nil {
		b.PosXSpring = anim.NewSpring(b.Position.GetX(), defaultLinePosYSpringParams)
	}
	if b.PosYSpring == nil {
		b.PosYSpring = anim.NewSpring(b.Position.GetY(), defaultLinePosYSpringParams)
	}
}

func (b *BottomLine) SetTransform(x, y, blur float64, force bool, delay time.Duration) {
	if b == nil {
		return
	}
	b.ensureSprings()
	b.BlurLevel = blur
	if force {
		b.PosXSpring.SetPosition(x)
		b.PosYSpring.SetPosition(y)
		b.Position.SetX(x)
		b.Position.SetY(y)
		return
	}
	b.PosXSpring.SetTargetPosition(x, delay)
	b.PosYSpring.SetTargetPosition(y, delay)
}

func (b *BottomLine) Update(dt time.Duration) {
	if b == nil || dt <= 0 {
		return
	}
	b.ensureSprings()
	b.PosXSpring.Update(dt)
	b.PosYSpring.Update(dt)
	b.Position.SetX(b.PosXSpring.CurrentPosition())
	b.Position.SetY(b.PosYSpring.CurrentPosition())
}

func bottomLineContentFontSize(fontSize float64) float64 {
	if fontSize <= 0 {
		fontSize = 24
	}
	return math.Max(12, fontSize*0.52)
}

func bottomLinePaddingTop(fontSize float64) float64 {
	if fontSize <= 0 {
		fontSize = 24
	}
	return math.Max(8, fontSize*0.38)
}

func bottomLinePaddingLeft(fontSize, width float64) float64 {
	return lineBasePadding(fontSize, width)
}

func (b *BottomLine) measure() {
	if b == nil {
		return
	}
	width := b.LineSize[0]
	if width < 0 {
		width = 0
	}
	height := 0.0
	if b.Active {
		textHeight := b.ContentFontSize
		if face := b.face(); face != nil {
			_, measuredHeight := text.Measure(b.Text, face, 1.0)
			measuredHeight = lp.FromLP(measuredHeight)
			if measuredHeight > textHeight {
				textHeight = measuredHeight
			}
		}
		height = b.PaddingTop*2 + textHeight
		minHeight := b.FontSize * 1.8
		if height < minHeight {
			height = minHeight
		}
	}
	b.Position.SetW(width)
	b.Position.SetH(height)
	b.Position.SetOriginX(width / 2)
	b.Position.SetOriginY(height / 2)
	b.LineSize = [2]float64{width, height}
}

func (b *BottomLine) face() text.Face {
	if b == nil || b.FontManager == nil || b.ContentFontSize <= 0 {
		return nil
	}
	face, err := b.FontManager.GetFaceForText(b.FontRequest, b.ContentFontSize, b.Text)
	if err != nil {
		return nil
	}
	return face
}

func (b *BottomLine) markImageDirty() {
	if b == nil {
		return
	}
	b.ImageDirty = true
	b.clearBlurCache()
}

func (b *BottomLine) clearImage() {
	if b == nil {
		return
	}
	b.clearBlurCache()
	if b.Image != nil {
		b.Image.Deallocate()
		b.Image = nil
	}
	b.ImageDirty = true
}

func (b *BottomLine) clearBlurCache() {
	if b == nil {
		return
	}
	if b.BlurImage != nil {
		b.BlurImage.Deallocate()
		b.BlurImage = nil
	}
	b.BlurCacheSource = nil
	b.BlurCacheKey = 0
}

func (b *BottomLine) ensureImage() bool {
	if b == nil || !b.HasContent() || b.LineSize[0] <= 0 || b.LineSize[1] <= 0 {
		b.clearImage()
		return false
	}
	width := safeImageLength(b.LineSize[0])
	height := safeImageLength(b.LineSize[1])
	if width <= 0 || height <= 0 {
		return false
	}
	if b.Image != nil {
		currentW, currentH := b.Image.Size()
		if currentW != width || currentH != height {
			b.clearImage()
		}
	}
	if b.Image == nil {
		b.Image = ebiten.NewImage(width, height)
		b.ImageDirty = true
	}
	if !b.ImageDirty {
		return true
	}
	b.Image.Clear()
	face := b.face()
	if face == nil {
		return true
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(lp.LP(b.PaddingLeft), lp.LP(b.PaddingTop))
	op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 210})
	text.Draw(b.Image, b.Text, face, op)
	b.ImageDirty = false
	b.clearBlurCache()
	return true
}

func (b *BottomLine) drawImage() *ebiten.Image {
	if b == nil || !b.ensureImage() || b.Image == nil {
		return nil
	}
	if b.BlurLevel <= 0 {
		return b.Image
	}
	blurPixels := blurPixelsForLevel(b.BlurLevel)
	cacheKey := blurCacheKey(blurPixels)
	if cacheKey <= 0 {
		return b.Image
	}
	if b.BlurImage != nil && b.BlurCacheSource == b.Image && b.BlurCacheKey == cacheKey {
		return b.BlurImage
	}
	b.clearBlurCache()
	b.BlurCacheSource = b.Image
	b.BlurCacheKey = cacheKey
	b.BlurImage = filters.BlurImageShader(b.Image, lp.LP(blurPixels))
	return b.BlurImage
}

func (b *BottomLine) Draw(screen *ebiten.Image) {
	if b == nil || screen == nil || !b.Active || b.Position.GetAlpha() <= 0 {
		return
	}
	drawImage := b.drawImage()
	if drawImage == nil {
		return
	}
	alpha := 0.58
	if b.Focused {
		alpha = 0.86
	}
	drawImageResample4x4(screen, drawImage, TransformToGeoM(&b.Position), float32(alpha*b.Position.GetAlpha()), ebiten.BlendLighter)
}
