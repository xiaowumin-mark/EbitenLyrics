package lyrics

// 文件说明：歌词渲染层实现。
// 主要职责：根据当前状态把歌词行与音节绘制到目标画面。

import (
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	"github.com/xiaowumin-mark/EbitenLyrics/filters"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
)

const (
	hiddenLineRetainDistance = 16
	hiddenLineRetainIdle     = 5 * time.Second
	hiddenLineRetainBudget   = 32
	hiddenLinePruneInterval  = time.Second
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
	if !l.isShow {
		return
	}
	targetW := safeImageLength(l.GetPosition().GetW())
	targetH := safeImageLength(l.GetPosition().GetH())
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}
	if l.Image != nil {
		currentW, currentH := l.Image.Size()
		if currentW == targetW && currentH == targetH {
			l.clearBlurCache()
			l.Image.Clear()
			l.imageDirty = true
			return
		}
		l.clearBlurCache()
		l.Image.Deallocate()
		l.Image = nil
	}
	l.Image = ebiten.NewImage(targetW, targetH)
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
			lp.LP(l.EffectivePaddingLeft()),
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
		l.clearBlurCache()
	}
	drawImage := l.Image
	if l.BlurLevel > 0 {
		drawImage = lineRendererLayer.blurredLineImage(l)
	}

	drawImageResample4x4(
		screen,
		drawImage,
		TransformToGeoM(l.GetPosition()),
		float32(l.GetPosition().GetAlpha()),
		ebiten.BlendLighter,
	)
}

func (RendererLayer) blurredLineImage(l *Line) *ebiten.Image {
	if l == nil || l.Image == nil || l.BlurLevel <= 0 {
		return l.Image
	}
	blurPixels := blurPixelsForLevel(l.BlurLevel)
	cacheKey := blurCacheKey(blurPixels)
	if cacheKey <= 0 {
		return l.Image
	}
	if l.BlurImage != nil && l.BlurCacheSource == l.Image && l.BlurCacheKey == cacheKey {
		return l.BlurImage
	}
	l.clearBlurCache()
	l.BlurCacheSource = l.Image
	l.BlurCacheKey = cacheKey
	l.BlurImage = filters.BlurImageShader(l.Image, lp.LP(blurPixels))
	return l.BlurImage
}

func blurPixelsForLevel(level float64) float64 {
	if level <= 0 || math.IsNaN(level) || math.IsInf(level, 0) {
		return 0
	}
	if level > 5 {
		level = 5
	}
	return level
}

func blurCacheKey(blurPixels float64) int {
	if blurPixels <= 0 || math.IsNaN(blurPixels) || math.IsInf(blurPixels, 0) {
		return 0
	}
	if blurPixels > 5 {
		blurPixels = 5
	}
	return int(math.Round(blurPixels * 1000))
}

func (l *Line) canUseStaticLayer() bool {
	return l != nil &&
		l.isShow &&
		l.Status == LineStatusPreviewStatic &&
		l.GetPosition().GetAlpha() > 0 &&
		l.BlurLevel <= 0
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
	lineRendererLayer.ReleaseLineResources(l)
	l.isShow = false
	l.lastRenderRank = -1
	l.imageDirty = true
	l.setStatus(LineStatusHidden)
}

func (RendererLayer) ReleaseLineResources(l *Line) {
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
			lineRendererLayer.ReleaseLineResources(bgline)
		}
	}
	if l.TranslateImage != nil {
		l.TranslateImage.Deallocate()
		l.TranslateImage = nil
	}
	if l.Image != nil {
		l.clearBlurCache()
		l.Image.Deallocate()
		l.Image = nil
	}
	l.clearBlurCache()
	l.imageDirty = true
	l.invalidateOffsetMetrics()
}

func (RendererLayer) HideLine(l *Line) {
	lineRendererLayer.HideLineWithManager(l, nil)
}

func (RendererLayer) HideLineWithManager(l *Line, manager *anim.Manager) {
	if l == nil {
		return
	}
	for _, bgline := range l.BackgroundLines {
		if bgline != nil {
			lineRendererLayer.HideLineWithManager(bgline, manager)
		}
	}
	wasRealtime := l.Status.RequiresRealtimeRender()
	lineAnimationLayer.disposeLineAnimationsWithManager(l, manager)
	if wasRealtime {
		lineAnimationLayer.resetLinePreviewContent(l)
	}
	l.clearBlurCache()
	l.isShow = false
	l.lastRenderRank = -1
	if wasRealtime {
		l.setStatus(LineStatusPreviewStatic)
	}
}

type hiddenLineResourceCandidate struct {
	index    int
	line     *Line
	distance int
	idle     time.Duration
}

func (RendererLayer) ReclaimHiddenLineResources(l *Lyrics, renderSet map[int]struct{}, anchorIndex int) {
	if l == nil || len(l.Lines) == 0 {
		return
	}
	now := l.Timeline.CurrentTime
	candidates := make([]hiddenLineResourceCandidate, 0)
	for i, line := range l.Lines {
		if line == nil {
			continue
		}
		if _, ok := renderSet[i]; ok {
			continue
		}
		if !lineHasRetainedResources(line) {
			continue
		}
		distance := absInt(i - anchorIndex)
		idle := now - line.lastVisibleAt
		if idle < 0 {
			idle = 0
		}
		if distance > hiddenLineRetainDistance || idle > hiddenLineRetainIdle {
			lineRendererLayer.DisposeLine(line)
			continue
		}
		candidates = append(candidates, hiddenLineResourceCandidate{
			index:    i,
			line:     line,
			distance: distance,
			idle:     idle,
		})
	}
	if len(candidates) <= hiddenLineRetainBudget {
		return
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		if candidates[i].idle != candidates[j].idle {
			return candidates[i].idle < candidates[j].idle
		}
		return candidates[i].index < candidates[j].index
	})
	for _, candidate := range candidates[hiddenLineRetainBudget:] {
		lineRendererLayer.DisposeLine(candidate.line)
	}
}

func (RendererLayer) TickHiddenResourcePrune(l *Lyrics, dt time.Duration) {
	if l == nil || dt <= 0 {
		return
	}
	l.hiddenResourcePruneElapsed += dt
	if l.hiddenResourcePruneElapsed < hiddenLinePruneInterval {
		return
	}
	l.hiddenResourcePruneElapsed = 0
	renderSet := make(map[int]struct{}, len(l.renderIndex))
	for _, index := range l.renderIndex {
		renderSet[index] = struct{}{}
	}
	anchorIndex := l.anchorIndex
	if anchorIndex < 0 {
		anchorIndex = l.Timeline.ScrollToIndex
	}
	if anchorIndex < 0 {
		anchorIndex = 0
	}
	lineRendererLayer.ReclaimHiddenLineResources(l, renderSet, anchorIndex)
}

func lineHasRetainedResources(l *Line) bool {
	if l == nil {
		return false
	}
	if l.Image != nil || l.BlurImage != nil || l.TranslateImage != nil {
		return true
	}
	for _, syllable := range l.Syllables {
		if syllableHasRetainedResources(syllable) {
			return true
		}
	}
	for _, bgline := range l.BackgroundLines {
		if lineHasRetainedResources(bgline) {
			return true
		}
	}
	return false
}

func syllableHasRetainedResources(s *LineSyllable) bool {
	if s == nil {
		return false
	}
	for _, element := range s.Elements {
		if element == nil {
			continue
		}
		if element.BackgroundBlurText != nil {
			return true
		}
		if syllableImageHasRetainedResources(element.SyllableImage) {
			return true
		}
	}
	return false
}

func syllableImageHasRetainedResources(s *SyllableImage) bool {
	return s != nil && (s.TextMask != nil || s.GradientImage != nil || s.HighlightGradientImage != nil)
}

func (RendererLayer) RenderLine(l *Line) {
	if l == nil {
		return
	}
	if l.isShow {
		for _, bgline := range l.BackgroundLines {
			lineRendererLayer.RenderLine(bgline)
		}
		if l.Image == nil {
			lineRendererLayer.RecreateLineImage(l)
		}
		if l.Status.UsesPreviewBitmap() && l.Image != nil && l.GetPosition().GetAlpha() > 0 && l.imageDirty {
			lineRendererLayer.redrawLineImage(l)
		}
		return
	}

	l.isShow = true
	if l.Status == LineStatusHidden {
		l.setStatus(LineStatusPreviewStatic)
	}
	for _, bgline := range l.BackgroundLines {
		lineRendererLayer.RenderLine(bgline)
	}
	if l.TranslateImage == nil && l.TranslatedText != "" {
		lineLayoutLayer.GenerateLineTranslateImage(l)
	}
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
	if l != nil {
		l.Dots.Draw(screen)
		l.Bottom.Draw(screen)
	}
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
