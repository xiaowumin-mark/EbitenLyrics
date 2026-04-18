package lyrics

// 文件说明：歌词渲染层实现。
// 主要职责：根据当前状态把歌词行与音节绘制到目标画面。

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
)

func (l *Line) recreateLineImage() {
	lineRendererLayer.RecreateLineImage(l)
}

func (l *Line) Draw(screen *ebiten.Image) {
	lineRendererLayer.DrawLine(l, screen)
}

func (l *Line) Dispose() {
	lineRendererLayer.DisposeLine(l)
}

func (l *Line) Render() {
	lineRendererLayer.RenderLine(l)
}

func (l *Lyrics) Draw(screen *ebiten.Image) {
	lineRendererLayer.DrawLyrics(l, screen)
}

func (l *Lyrics) DrawStatic(screen *ebiten.Image) {
	lineRendererLayer.DrawLyricsStatic(l, screen)
}

func (l *Lyrics) DrawDynamic(screen *ebiten.Image) {
	lineRendererLayer.DrawLyricsDynamic(l, screen)
}

func (RendererLayer) RecreateLineImage(l *Line) {
	if l == nil {
		return
	}
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
	}
	if !l.isShow {
		return
	}
	l.Image = ebiten.NewImage(
		safeImageLength(l.GetPosition().GetW()),
		safeImageLength(l.GetPosition().GetH()),
	)
	l.imageDirty = true
}

func (RendererLayer) redrawLineImage(l *Line) {
	if l == nil || l.Image == nil {
		return
	}

	l.Image.Clear()
	for _, syllable := range l.Syllables {
		syllable.Draw(l.Image)
	}

	if l.TranslateImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(
			lp.LP(l.Padding),
			lp.LP(l.GetPosition().GetH()-l.TranslateImageH-l.Padding),
		)
		l.Image.DrawImage(l.TranslateImage, op)
	}

	l.imageDirty = false
}

func (RendererLayer) DrawLine(l *Line, screen *ebiten.Image) {
	if l == nil || screen == nil || !l.isShow {
		return
	}
	if l.GetPosition().GetAlpha() <= 0 {
		return
	}
	if l.Image == nil {
		lineRendererLayer.RecreateLineImage(l)
	}
	if l.Image == nil {
		return
	}

	if l.Status.RequiresRealtimeRender() || l.imageDirty {
		lineRendererLayer.redrawLineImage(l)
	}

	drawImageResample4x4(
		screen,
		l.Image,
		TransformToGeoM(l.GetPosition()),
		float32(l.GetPosition().GetAlpha()),
		ebiten.BlendLighter,
	)
}

func (l *Line) canUseStaticLayer() bool {
	return l != nil &&
		l.isShow &&
		l.Status == LineStatusPreviewStatic &&
		l.GetPosition().GetAlpha() > 0
}

func (l *Line) shouldDrawDynamically() bool {
	return l != nil &&
		l.isShow &&
		l.GetPosition().GetAlpha() > 0 &&
		!l.canUseStaticLayer()
}

func (RendererLayer) DisposeLine(l *Line) {
	if l == nil {
		return
	}
	for _, syllable := range l.Syllables {
		if syllable != nil {
			syllable.Dispose()
		}
	}
	for _, bgline := range l.BackgroundLines {
		if bgline != nil {
			lineRendererLayer.DisposeLine(bgline)
		}
	}
	if l.TranslateImage != nil {
		l.TranslateImage.Deallocate()
		l.TranslateImage = nil
	}
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
	}
	l.isShow = false
	l.imageDirty = true
	l.setStatus(LineStatusHidden)
}
func (RendererLayer) RenderLine(l *Line) {
	if l == nil {
		return
	}
	if l.isShow {
		for _, bgline := range l.BackgroundLines {
			lineRendererLayer.RenderLine(bgline)
		}
		l.markImageDirty()
		if l.Image == nil {
			lineRendererLayer.RecreateLineImage(l)
		}
		if l.Status.UsesPreviewBitmap() && l.Image != nil && l.GetPosition().GetAlpha() > 0 {
			lineRendererLayer.redrawLineImage(l)
		}
		return
	}

	l.isShow = true
	if l.Status == LineStatusHidden {
		l.setStatus(LineStatusPreviewStatic)
	}
	for _, syllable := range l.Syllables {
		syllable.Redraw()
	}
	for _, bgline := range l.BackgroundLines {
		lineRendererLayer.RenderLine(bgline)
	}
	lineLayoutLayer.GenerateLineTranslateImage(l)
	lineRendererLayer.RecreateLineImage(l)
	if l.Status.UsesPreviewBitmap() && l.Image != nil && l.GetPosition().GetAlpha() > 0 {
		lineRendererLayer.redrawLineImage(l)
	}
}

func (RendererLayer) DrawLyrics(l *Lyrics, screen *ebiten.Image) {
	lineRendererLayer.drawLyricsFiltered(l, screen, func(line *Line) bool {
		return line != nil && line.isShow && line.GetPosition().GetAlpha() > 0
	})
}

func (RendererLayer) DrawLyricsStatic(l *Lyrics, screen *ebiten.Image) {
	lineRendererLayer.drawLyricsFiltered(l, screen, func(line *Line) bool {
		return line.canUseStaticLayer()
	})
}

func (RendererLayer) DrawLyricsDynamic(l *Lyrics, screen *ebiten.Image) {
	lineRendererLayer.drawLyricsFiltered(l, screen, func(line *Line) bool {
		return line.shouldDrawDynamically()
	})
}

func (RendererLayer) drawLyricsFiltered(l *Lyrics, screen *ebiten.Image, include func(*Line) bool) {
	if l == nil || screen == nil {
		return
	}
	for _, i := range l.renderIndex {
		if i < 0 || i >= len(l.Lines) {
			continue
		}
		line := l.Lines[i]
		if include == nil || include(line) {
			lineRendererLayer.DrawLine(line, screen)
		}
		for _, bgLine := range line.BackgroundLines {
			if include != nil && !include(bgLine) {
				continue
			}
			lineRendererLayer.DrawLine(bgLine, screen)
		}
	}
}
