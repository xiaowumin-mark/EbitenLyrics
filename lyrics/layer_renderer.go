package lyrics

import "github.com/hajimehoshi/ebiten/v2"

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
}

func (RendererLayer) DrawLine(l *Line, screen *ebiten.Image) {
	if l == nil || screen == nil || !l.isShow {
		return
	}
	if l.Image == nil {
		lineRendererLayer.RecreateLineImage(l)
	}
	if l.Image == nil {
		return
	}

	l.Image.Clear()
	for _, syllable := range l.Syllables {
		syllable.Draw(l.Image)
	}

	if l.TranslateImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(l.Padding, l.GetPosition().GetH()-l.TranslateImageH-l.Padding)
		l.Image.DrawImage(l.TranslateImage, op)
	}

	drawImageResample4x4(
		screen,
		l.Image,
		TransformToGeoM(l.GetPosition()),
		float32(l.GetPosition().GetAlpha()),
		ebiten.BlendLighter,
	)
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
}
func (RendererLayer) RenderLine(l *Line) {
	if l == nil || l.isShow {
		return
	}
	l.isShow = true
	for _, syllable := range l.Syllables {
		syllable.Redraw()
	}
	for _, bgline := range l.BackgroundLines {
		lineRendererLayer.RenderLine(bgline)
	}
	lineLayoutLayer.GenerateLineTranslateImage(l)
	lineRendererLayer.RecreateLineImage(l)
}

func (RendererLayer) DrawLyrics(l *Lyrics, screen *ebiten.Image) {
	if l == nil || screen == nil {
		return
	}
	for _, i := range l.renderIndex {
		if i < 0 || i >= len(l.Lines) {
			continue
		}
		lineRendererLayer.DrawLine(l.Lines[i], screen)
		for _, bgLine := range l.Lines[i].BackgroundLines {
			lineRendererLayer.DrawLine(bgLine, screen)
		}
	}
}
