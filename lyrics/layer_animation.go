package lyrics

// 文件说明：歌词动画层实现。
// 主要职责：处理滚动、聚焦、高亮和逐字动画的调度。

import (
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
)

var CustomElastic = anim.NewEaseInElastic(1.03, 1.7)

const (
	// maxHighlightWordRunes 限制单次高亮动画的最大字符数，避免超长词元造成夸张抖动。
	maxHighlightWordRunes = 8

	scrollFastGapThreshold = 450 * time.Millisecond
	scrollDurationNormal   = 900 * time.Millisecond
	scrollDurationFast     = 600 * time.Millisecond
	scrollDelayStep        = 50 * time.Millisecond
	scrollDelayIndexOffset = 3

	lineEnterDuration         = 500 * time.Millisecond
	lineExitDuration          = 600 * time.Millisecond
	lineHighlightFadeDuration = 320 * time.Millisecond
	backgroundEnterDuration   = 700 * time.Millisecond
	backgroundEnterDelay      = 200 * time.Millisecond
	backgroundExitDuration    = 300 * time.Millisecond

	scrollReuseTargetEpsilon = 0.5
	animationValueEpsilon    = 0.01
)

var scrollEaseFast = anim.EaseOutQuart

func (l *Lyrics) Scroll(index []int, notInit int) {
	lineAnimationLayer.ScrollLyrics(l, index, notInit)
}

func (l *Lyrics) Update(t time.Duration) {
	lineAnimationLayer.UpdateLyrics(l, t)
}

func (l *Line) ToNormal(lyrics *Lyrics) {
	lineAnimationLayer.NormalizeLine(l, lyrics)
}

func (l *Line) LineAnimate(lyrics *Lyrics, fd float64) {
	lineAnimationLayer.LineAnimate(l, lyrics, fd)
}

func (l *Line) FrameAnimate(lyrics *Lyrics, fd float64) {
	lineAnimationLayer.FrameAnimate(l, lyrics, fd)
}

func (l *Line) DisposeAllAnimations() {
	lineAnimationLayer.DisposeLineAnimations(l)
}

func (l *Lyrics) DisposeAllAnimations() {
	lineAnimationLayer.DisposeLyricsAnimations(l)
}

func (l *Lyrics) Dispose() {
	lineAnimationLayer.DisposeLyrics(l)
}

func cancelManagedAnimation(manager *anim.Manager, a anim.Animation) {
	if a == nil {
		return
	}
	if manager != nil {
		manager.Cancel(a.ID())
		return
	}
	a.Cancel()
}

func (AnimationLayer) cancelLineStatusSettle(l *Line, manager *anim.Manager) {
	if l == nil || l.StatusSettleAnimate == nil {
		return
	}
	cancelManagedAnimation(manager, l.StatusSettleAnimate)
	l.StatusSettleAnimate = nil
}

func (AnimationLayer) syncPreviewState(l *Line) {
	if l == nil || l.Status == LineStatusHidden || l.Status.RequiresRealtimeRender() {
		return
	}
	if l.ScrollAnimate != nil {
		l.setStatus(LineStatusPreviewScrolling)
		return
	}
	l.setStatus(LineStatusPreviewStatic)
}

func (AnimationLayer) settleElementHighlightToRest(e *SyllableElement, lyrics *Lyrics, duration time.Duration) {
	if e == nil {
		return
	}
	if e.HighlightAnimate != nil {
		e.HighlightAnimate.Cancel()
		e.HighlightAnimate = nil
	}

	targetTranslateX := 0.0
	targetScale := 1.0
	startTranslateX := e.GetPosition().GetTranslateX()
	startScaleX := e.GetPosition().GetScaleX()
	startScaleY := e.GetPosition().GetScaleY()
	startBlurAlpha := 0.0
	blur := e.BackgroundBlurText
	if blur != nil {
		startBlurAlpha = blur.Alpha
	}

	if !needsAnimation(startTranslateX, targetTranslateX) &&
		!needsAnimation(startScaleX, targetScale) &&
		!needsAnimation(startScaleY, targetScale) &&
		!needsAnimation(startBlurAlpha, 0) {
		e.GetPosition().SetTranslateX(targetTranslateX)
		e.GetPosition().SetScaleX(targetScale)
		e.GetPosition().SetScaleY(targetScale)
		if blur != nil {
			blur.Dispose()
			if e.BackgroundBlurText == blur {
				e.BackgroundBlurText = nil
			}
		}
		return
	}

	if lyrics == nil || lyrics.AnimateManager == nil || duration <= 0 {
		e.GetPosition().SetTranslateX(targetTranslateX)
		e.GetPosition().SetScaleX(targetScale)
		e.GetPosition().SetScaleY(targetScale)
		if blur != nil {
			blur.Dispose()
			if e.BackgroundBlurText == blur {
				e.BackgroundBlurText = nil
			}
		}
		return
	}

	e.HighlightAnimate = anim.NewKeyframeAnimation(
		uuid.NewString(),
		duration,
		0,
		1,
		true,
		[]anim.Keyframe{
			{Offset: 0, Values: []float64{startTranslateX, startScaleX, startScaleY, startBlurAlpha}, Ease: nil},
			{Offset: 1, Values: []float64{targetTranslateX, targetScale, targetScale, 0}, Ease: anim.EaseOut},
		},
		func(values []float64) {
			e.Position.SetTranslateX(values[0])
			e.Position.SetScaleX(values[1])
			e.Position.SetScaleY(values[2])
			if blur != nil {
				blur.Alpha = values[3]
			}
		},
		func() {
			e.HighlightAnimate = nil
			e.Position.SetTranslateX(targetTranslateX)
			e.Position.SetScaleX(targetScale)
			e.Position.SetScaleY(targetScale)
			if blur != nil {
				blur.Dispose()
				if e.BackgroundBlurText == blur {
					e.BackgroundBlurText = nil
				}
			}
		},
	)
	lyrics.AnimateManager.Add(e.HighlightAnimate)
}

func (AnimationLayer) finishScrollAnimation(l *Line) {
	if l == nil {
		return
	}
	l.ScrollAnimate = nil
	if l.Status == LineStatusPreviewScrolling {
		l.setStatus(LineStatusPreviewStatic)
	}
}

func (AnimationLayer) settleLineToPreview(l *Line, lyrics *Lyrics, delay time.Duration) {
	if l == nil || lyrics == nil {
		return
	}
	lineAnimationLayer.cancelLineStatusSettle(l, lyrics.AnimateManager)
	if delay <= 0 {
		if l.Status == LineStatusActiveExit {
			lineAnimationLayer.syncPreviewState(l)
		}
		return
	}

	l.StatusSettleAnimate = anim.NewTween(
		uuid.NewString(),
		delay,
		0,
		1,
		0,
		1,
		anim.Linear,
		func(float64) {},
		func() {
			l.StatusSettleAnimate = nil
			if l.Status == LineStatusActiveExit {
				lineAnimationLayer.syncPreviewState(l)
			}
		},
	)
	lyrics.AnimateManager.Add(l.StatusSettleAnimate)
}

func scrollLeadTime() time.Duration {
	return time.Duration(scrollDelayIndexOffset) * scrollDelayStep
}

func predictedScrollAnchorIndex(lines []*Line, t time.Duration) int {
	return findScrollAnchorIndexByTime(lines, t+scrollLeadTime())
}

func (AnimationLayer) ensureScrollAnimation(l *Line, lyrics *Lyrics, targetY float64, delay, duration time.Duration, ease anim.EaseFunc) {
	if l == nil || lyrics == nil {
		return
	}
	if math.Abs(l.GetPosition().GetY()-targetY) <= scrollReuseTargetEpsilon {
		if l.ScrollAnimate != nil {
			cancelManagedAnimation(lyrics.AnimateManager, l.ScrollAnimate)
			l.ScrollAnimate = nil
		}
		l.GetPosition().SetY(targetY)
		lineAnimationLayer.finishScrollAnimation(l)
		return
	}
	if l.ScrollAnimate != nil && math.Abs(l.ScrollAnimate.To-targetY) <= scrollReuseTargetEpsilon {
		return
	}
	if l.ScrollAnimate != nil {
		// Retargeting an in-flight scroll tween should continue immediately.
		// Re-applying the entry delay here produces a visible hitch.
		delay = 0
		cancelManagedAnimation(lyrics.AnimateManager, l.ScrollAnimate)
		l.ScrollAnimate = nil
	}
	l.ScrollAnimate = anim.NewTween(
		uuid.NewString(),
		duration,
		delay,
		1,
		l.GetPosition().GetY(),
		targetY,
		ease,
		func(value float64) {
			l.GetPosition().SetY(value)
		},
		func() {
			lineAnimationLayer.finishScrollAnimation(l)
		},
	)
	lyrics.AnimateManager.Add(l.ScrollAnimate)
}

func (AnimationLayer) ScrollLyrics(l *Lyrics, index []int, notInit int) {
	anchorIndex := -1
	if len(index) > 0 {
		anchorIndex = index[0]
	}
	lineAnimationLayer.scrollLyricsTo(l, index, anchorIndex, notInit)
}

func (AnimationLayer) scrollLyricsTo(l *Lyrics, activeIndexes []int, anchorIndex int, notInit int) {
	if l == nil || len(l.Lines) == 0 {
		return
	}

	prevAnchorIndex := l.anchorIndex

	if anchorIndex < 0 {
		if len(activeIndexes) > 0 {
			anchorIndex = activeIndexes[0]
		} else if l.anchorIndex >= 0 && l.anchorIndex < len(l.Lines) {
			anchorIndex = l.anchorIndex
		} else if notInit == 0 {
			anchorIndex = 0
		} else {
			return
		}
	}
	if anchorIndex < 0 {
		anchorIndex = 0
	}
	if anchorIndex >= len(l.Lines) {
		anchorIndex = len(l.Lines) - 1
	}
	l.anchorIndex = anchorIndex

	scrollDuration := scrollDurationNormal
	scrollEase := anim.EaseFunc(CustomElastic)
	if shouldUseFastScroll(l, prevAnchorIndex, anchorIndex) {
		scrollDuration = scrollDurationFast
		scrollEase = scrollEaseFast
	}

	activeSet := make(map[int]struct{}, len(activeIndexes)+1)
	for _, i := range activeIndexes {
		if i < 0 || i >= len(l.Lines) {
			continue
		}
		activeSet[i] = struct{}{}
	}
	if len(activeSet) == 0 {
		activeSet[anchorIndex] = struct{}{}
	}

	_, h := ebiten.WindowSize()
	viewportHeight := lp.FromLP(float64(h))
	offsetY := -viewportHeight / 4
	for i := 0; i < anchorIndex; i++ {
		offsetY += l.Lines[i].Position.GetH()
		if _, ok := activeSet[i]; ok && len(l.Lines[i].BackgroundLines) > 0 {
			for _, bgLine := range l.Lines[i].BackgroundLines {
				if !backgroundLineReservesSpace(bgLine) {
					continue
				}
				offsetY += bgLine.Position.GetH()
			}
		}
	}

	overscan := math.Max(120, viewportHeight*0.45)
	viewportTop := -overscan
	viewportBottom := viewportHeight + overscan
	isInitialPlacement := notInit == 0
	cullTransitionDistance := overscan * 1.35
	snapDistance := math.Max(viewportHeight*0.9, 320)

	lastY := 0.0
	renderSet := make(map[int]struct{}, len(l.Lines)/2+1)
	for i, line := range l.Lines {
		targetLineY := lastY - offsetY
		_, isActive := activeSet[i]
		isAnchor := i == anchorIndex
		currentLineY := line.GetPosition().GetY()
		if isInitialPlacement {
			currentLineY = targetLineY
		}
		lineHeight := line.GetPosition().GetH()
		shouldRender := lineVisibleAt(targetLineY, lineHeight, viewportTop, viewportBottom)
		if !shouldRender && !isInitialPlacement && math.Abs(targetLineY-currentLineY) <= cullTransitionDistance {
			shouldRender = lineVisibleAt(currentLineY, lineHeight, viewportTop, viewportBottom)
		}

		bgTargetY := targetLineY + line.Position.GetH()
		for _, bg := range line.BackgroundLines {
			if bg == nil {
				continue
			}
			currentBgY := bg.GetPosition().GetY()
			if isInitialPlacement {
				currentBgY = bgTargetY
			}
			bgHeight := bg.GetPosition().GetH()
			bgShouldRender := lineVisibleAt(bgTargetY, bgHeight, viewportTop, viewportBottom)
			if !bgShouldRender && !isInitialPlacement && math.Abs(bgTargetY-currentBgY) <= cullTransitionDistance {
				bgShouldRender = lineVisibleAt(currentBgY, bgHeight, viewportTop, viewportBottom)
			}
			if bgShouldRender {
				shouldRender = true
			}
		}
		if isActive || isAnchor {
			shouldRender = true
		}

		if shouldRender {
			renderSet[i] = struct{}{}
		}

		lineTravel := math.Abs(targetLineY - line.GetPosition().GetY())
		if isInitialPlacement || !shouldRender || lineTravel > snapDistance {
			if line.ScrollAnimate != nil {
				cancelManagedAnimation(l.AnimateManager, line.ScrollAnimate)
				line.ScrollAnimate = nil
			}
			line.GetPosition().SetY(targetLineY)
			if !line.Status.RequiresRealtimeRender() {
				lineAnimationLayer.syncPreviewState(line)
			}
		} else {
			delay := scrollDelayForIndex(anchorIndex, i)
			lineAnimationLayer.ensureScrollAnimation(line, l, targetLineY, delay, scrollDuration, scrollEase)
			if !line.Status.RequiresRealtimeRender() {
				line.setStatus(LineStatusPreviewScrolling)
			}
		}

		for _, bg := range line.BackgroundLines {
			bgTravel := math.Abs(bgTargetY - bg.GetPosition().GetY())
			if isInitialPlacement || !shouldRender || bgTravel > snapDistance {
				if bg.ScrollAnimate != nil {
					cancelManagedAnimation(l.AnimateManager, bg.ScrollAnimate)
					bg.ScrollAnimate = nil
				}
				bg.GetPosition().SetY(bgTargetY)
				if !bg.Status.RequiresRealtimeRender() {
					lineAnimationLayer.syncPreviewState(bg)
				}
				continue
			}
			delay := scrollDelayForIndex(anchorIndex, i)
			lineAnimationLayer.ensureScrollAnimation(bg, l, bgTargetY, delay, scrollDuration, scrollEase)
			if !bg.Status.RequiresRealtimeRender() {
				bg.setStatus(LineStatusPreviewScrolling)
			}
		}

		lastY += line.Position.GetH() + l.Margin
		if isActive && len(line.BackgroundLines) > 0 {
			for _, bgLine := range line.BackgroundLines {
				if !backgroundLineReservesSpace(bgLine) {
					continue
				}
				lastY += bgLine.Position.GetH() + l.Margin
			}
		}
	}

	renderIndex := make([]int, 0, len(renderSet))
	for i := range renderSet {
		renderIndex = append(renderIndex, i)
	}
	sort.Ints(renderIndex)
	l.renderIndex = renderIndex

	for i, el := range l.Lines {
		if _, ok := renderSet[i]; ok {
			lineRendererLayer.RenderLine(el)
		} else {
			lineRendererLayer.DisposeLine(el)
		}

		if el.Status.CanStartExit() && !hasInt(l.nowLyrics, i) {
			lineAnimationLayer.NormalizeLine(el, l)
			for _, bg := range el.BackgroundLines {
				lineAnimationLayer.NormalizeLine(bg, l)
			}
			continue
		}

		if el.isShow {
			lineAnimationLayer.syncPreviewState(el)
			for _, bg := range el.BackgroundLines {
				if bg != nil && bg.isShow {
					lineAnimationLayer.syncPreviewState(bg)
				}
			}
		}
	}
}

func (AnimationLayer) UpdateLyrics(l *Lyrics, t time.Duration) {
	if l == nil {
		return
	}
	l.Position = t
	changed := false
	for i, line := range l.Lines {
		if t >= line.StartTime && t < line.EndTime {
			// 逐行模式不需要逐字扫光偏移，避免出现“进度条式”高亮。
			if line.RenderMode != RenderModeLine && useRealtimeOffsetFormula {
				applyRealtimeOffsets(line.OuterSyllableElements, t, l.FD)
				for _, bgLine := range line.BackgroundLines {
					applyRealtimeOffsets(bgLine.OuterSyllableElements, t, l.FD)
				}
			}
			if hasInt(l.nowLyrics, i) {
				continue
			}
			changed = true
			l.nowLyrics = append(l.nowLyrics, i)
			l.nowLyrics = sortIntSlice(l.nowLyrics)
			l.finalLayoutPending = false
			log.Println("lyric enter", i, l.nowLyrics, line.Text)

			lineAnimationLayer.cancelLineStatusSettle(line, l.AnimateManager)
			line.setStatus(LineStatusActiveEnter)
			lineAnimationLayer.LineAnimate(line, l, l.FD)
		} else {
			if hasInt(l.nowLyrics, i) {
				changed = true
				l.nowLyrics = removeInt(l.nowLyrics, i)
				log.Println("lyric leave", i, line.EndTime)
				for _, bg := range line.BackgroundLines {
					log.Println("bg line leave:", bg.EndTime)
				}
			}
		}
	}

	allEnded := lyricsAllEndedAt(l.Lines, t)
	changed = lineAnimationLayer.updateFinalLayoutState(l, allEnded, changed)

	anchor := predictedScrollAnchorIndex(l.Lines, t)
	if anchor >= 0 && (changed || anchor != l.anchorIndex) {
		lineAnimationLayer.scrollLyricsTo(l, l.nowLyrics, anchor, 1)
	}
}

func (AnimationLayer) NormalizeLine(l *Line, lyrics *Lyrics) {
	if l == nil || lyrics == nil {
		return
	}
	lineAnimationLayer.cancelLineStatusSettle(l, lyrics.AnimateManager)
	l.setStatus(LineStatusActiveExit)

	if l.ScaleAnimate != nil {
		l.ScaleAnimate.Cancel()
		l.ScaleAnimate = nil
	}
	if !l.IsBackground {
		targetScale := inactiveLineScale(l.fontsize)
		if needsAnimation(l.GetPosition().GetScaleX(), targetScale) || needsAnimation(l.GetPosition().GetScaleY(), targetScale) {
			l.ScaleAnimate = anim.NewTween(
				uuid.NewString(),
				lineExitDuration,
				0,
				1,
				l.GetPosition().GetScaleX(),
				targetScale,
				anim.EaseInOut,
				func(value float64) {
					l.GetPosition().SetScaleX(value)
					l.GetPosition().SetScaleY(value)
				},
				func() {
					l.ScaleAnimate = nil
					l.GetPosition().SetScaleX(targetScale)
					l.GetPosition().SetScaleY(targetScale)
				},
			)
			lyrics.AnimateManager.Add(l.ScaleAnimate)
		} else {
			l.GetPosition().SetScaleX(targetScale)
			l.GetPosition().SetScaleY(targetScale)
		}
	}
	if l.GradientColorAnimate != nil {
		l.GradientColorAnimate.Cancel()
		l.GradientColorAnimate = nil
	}

	// 逐行模式只做整行亮度渐变，不做逐字抬升还原。
	if l.RenderMode != RenderModeLine {
		highlightSettleDuration := lineHighlightFadeDuration
		if l.IsBackground {
			highlightSettleDuration = backgroundExitDuration
		}
		for _, e := range l.OuterSyllableElements {
			if e == nil {
				continue
			}
			if e.UpAnimate != nil {
				e.UpAnimate.Cancel()
				e.UpAnimate = nil
			}
			lineAnimationLayer.settleElementHighlightToRest(e, lyrics, highlightSettleDuration)
			if l.IsBackground || !needsAnimation(e.GetPosition().GetTranslateY(), 0) {
				e.GetPosition().SetTranslateY(0)
				continue
			}
			e.UpAnimate = anim.NewTween(
				uuid.NewString(),
				lineExitDuration,
				0,
				1,
				e.GetPosition().GetTranslateY(),
				0,
				anim.EaseInOut,
				func(value float64) {
					e.GetPosition().SetTranslateY(value)
				},
				func() {
					e.UpAnimate = nil
					e.GetPosition().SetTranslateY(0)
				},
			)
			lyrics.AnimateManager.Add(e.UpAnimate)
		}
	} else {
		for _, e := range l.OuterSyllableElements {
			if e == nil {
				continue
			}
			// 逐行模式下清理逐字动画残留，确保退出态不会保留局部形变或模糊。
			if e.UpAnimate != nil {
				e.UpAnimate.Cancel()
				e.UpAnimate = nil
			}
			if e.HighlightAnimate != nil {
				e.HighlightAnimate.Cancel()
				e.HighlightAnimate = nil
			}
			if e.BackgroundBlurText != nil {
				e.BackgroundBlurText.Dispose()
				e.BackgroundBlurText = nil
			}
			e.GetPosition().SetTranslateX(0)
			e.GetPosition().SetTranslateY(0)
			e.GetPosition().SetScaleX(1)
			e.GetPosition().SetScaleY(1)
		}
	}

	currentHighlightAlpha := 0.0
	for _, e := range l.OuterSyllableElements {
		if e == nil {
			continue
		}
		if e.Animate != nil {
			e.Animate.Cancel()
			e.Animate = nil
		}
		if e.Alpha > currentHighlightAlpha {
			currentHighlightAlpha = e.Alpha
		}
	}
	if currentHighlightAlpha > 0 && len(l.OuterSyllableElements) > 0 {
		l.GradientColorAnimate = anim.NewTween(
			uuid.NewString(),
			lineHighlightFadeDuration,
			0,
			1,
			currentHighlightAlpha,
			0,
			anim.EaseOut,
			func(value float64) {
				for _, e := range l.OuterSyllableElements {
					if e == nil {
						continue
					}
					e.Alpha = value
				}
			},
			func() {
				for _, e := range l.OuterSyllableElements {
					if e == nil {
						continue
					}
					e.Alpha = 0
				}
				l.GradientColorAnimate = nil
			},
		)
		lyrics.AnimateManager.Add(l.GradientColorAnimate)
	}

	settleDelay := lineExitDuration
	if l.IsBackground {
		if l.AlphaAnimate != nil {
			l.AlphaAnimate.Cancel()
			l.AlphaAnimate = nil
		}
		l.AlphaAnimate = anim.NewKeyframeAnimation(
			uuid.NewString(),
			backgroundExitDuration,
			0,
			1,
			false,
			[]anim.Keyframe{
				{Offset: 0, Values: []float64{l.GetPosition().GetAlpha()}, Ease: anim.EaseInOut},
				{Offset: 1, Values: []float64{0}, Ease: anim.EaseInOut},
			},
			func(value []float64) {
				l.GetPosition().SetAlpha(value[0])
			},
			func() {
				l.AlphaAnimate = nil
				l.GetPosition().SetAlpha(0)
				l.Position.SetTranslateY(0)
				baseScale := inactiveLineScale(l.fontsize)
				l.Position.SetScaleX(baseScale)
				l.Position.SetScaleY(baseScale)
			},
		)
		lyrics.AnimateManager.Add(l.AlphaAnimate)
		settleDelay = backgroundExitDuration
	}

	lineAnimationLayer.settleLineToPreview(l, lyrics, settleDelay)
}

func (AnimationLayer) LineAnimate(l *Line, lyrics *Lyrics, fd float64) {
	if l == nil || lyrics == nil {
		return
	}
	lineAnimationLayer.cancelLineStatusSettle(l, lyrics.AnimateManager)
	l.setStatus(LineStatusActiveEnter)
	if l.ScaleAnimate != nil {
		l.ScaleAnimate.Cancel()
		l.ScaleAnimate = nil
	}
	l.ScaleAnimate = anim.NewTween(
		uuid.NewString(),
		lineEnterDuration,
		0,
		1,
		l.GetPosition().GetScaleX(),
		1,
		anim.EaseInOut,
		func(value float64) {
			l.GetPosition().SetScaleX(value)
			l.GetPosition().SetScaleY(value)
		},
		func() {
			l.ScaleAnimate = nil
			l.GetPosition().SetScaleX(1)
			l.GetPosition().SetScaleY(1)
			if l.Status == LineStatusActiveEnter {
				l.setStatus(LineStatusActivePlaying)
			}
		},
	)
	lyrics.AnimateManager.Add(l.ScaleAnimate)

	lineAnimationLayer.FrameAnimate(l, lyrics, fd)

	for _, it := range l.BackgroundLines {
		lineAnimationLayer.cancelLineStatusSettle(it, lyrics.AnimateManager)
		it.setStatus(LineStatusActiveEnter)
		if it.AlphaAnimate != nil {
			it.AlphaAnimate.Cancel()
			it.AlphaAnimate = nil
		}
		targetScale := inactiveLineScale(it.fontsize)
		it.AlphaAnimate = anim.NewKeyframeAnimation(
			uuid.NewString(),
			backgroundEnterDuration,
			backgroundEnterDelay,
			1,
			false,
			[]anim.Keyframe{
				{Offset: 0, Values: []float64{it.GetPosition().GetAlpha(), it.GetPosition().GetScaleX()}, Ease: anim.EaseOut},
				{Offset: 1, Values: []float64{1, targetScale}, Ease: anim.EaseOut},
			},
			func(value []float64) {
				it.Position.SetAlpha(value[0])
				it.Position.SetScaleX(value[1])
				it.Position.SetScaleY(value[1])
			},
			func() {
				it.AlphaAnimate = nil
				it.Position.SetAlpha(1)
				it.Position.SetScaleX(targetScale)
				it.Position.SetScaleY(targetScale)
				if it.Status == LineStatusActiveEnter {
					it.setStatus(LineStatusActivePlaying)
				}
			},
		)
		lyrics.AnimateManager.Add(it.AlphaAnimate)

		lineAnimationLayer.FrameAnimate(it, lyrics, fd)
	}
}

func currentLineHighlightAlpha(l *Line) float64 {
	if l == nil {
		return 0
	}
	alpha := 0.0
	for _, e := range l.OuterSyllableElements {
		if e == nil {
			continue
		}
		if e.Alpha > alpha {
			alpha = e.Alpha
		}
	}
	return alpha
}

func splitWordElementsByRuneLimit(l *Line, word []int, runeLimit int) ([][]*SyllableElement, bool) {
	if l == nil || len(word) == 0 {
		return nil, false
	}
	var all []*SyllableElement
	totalRunes := 0

	for _, idx := range word {
		if idx < 0 || idx >= len(l.Syllables) {
			continue
		}
		for _, ele := range l.Syllables[idx].Elements {
			if ele == nil {
				continue
			}
			all = append(all, ele)
			totalRunes += utf8.RuneCountInString(strings.TrimSpace(ele.Text))
		}
	}
	if len(all) == 0 {
		return nil, false
	}

	// 关键规则：超过长度阈值的词，整词禁用高亮动画，但仍保留上升动画。
	if runeLimit > 0 && totalRunes > runeLimit {
		return [][]*SyllableElement{all}, false
	}

	return [][]*SyllableElement{all}, true
}

// frameAnimateLineMode 执行逐行模式动画：
// - 不做逐字偏移动画；
// - 不做逐词弹跳/模糊；
// - 仅在行状态切换时做整行高亮渐入。
func (AnimationLayer) frameAnimateLineMode(l *Line, lyrics *Lyrics) {
	if l == nil || lyrics == nil {
		return
	}

	if l.GradientColorAnimate != nil {
		l.GradientColorAnimate.Cancel()
		l.GradientColorAnimate = nil
	}

	startAlpha := currentLineHighlightAlpha(l)
	for _, e := range l.OuterSyllableElements {
		if e == nil {
			continue
		}
		if e.Animate != nil {
			e.Animate.Cancel()
			e.Animate = nil
		}
		if e.HighlightAnimate != nil {
			e.HighlightAnimate.Cancel()
			e.HighlightAnimate = nil
		}
		if e.UpAnimate != nil {
			e.UpAnimate.Cancel()
			e.UpAnimate = nil
		}
		if e.BackgroundBlurText != nil {
			e.BackgroundBlurText.Dispose()
			e.BackgroundBlurText = nil
		}
		// 逐行模式统一重置局部变换，避免逐字动画的残留状态影响整行视觉。
		e.GetPosition().SetTranslateX(0)
		e.GetPosition().SetTranslateY(0)
		e.GetPosition().SetScaleX(1)
		e.GetPosition().SetScaleY(1)
		// 0 偏移可让高亮梯度覆盖整块文本，配合 Alpha 实现“整行高亮”。
		e.NowOffset = 0
	}

	if len(l.OuterSyllableElements) == 0 {
		return
	}

	if startAlpha >= 1 {
		for _, e := range l.OuterSyllableElements {
			if e == nil {
				continue
			}
			e.Alpha = 1
		}
		return
	}

	l.GradientColorAnimate = anim.NewTween(
		uuid.NewString(),
		320*time.Millisecond,
		0,
		1,
		startAlpha,
		1,
		anim.EaseInOut,
		func(value float64) {
			for _, e := range l.OuterSyllableElements {
				if e == nil {
					continue
				}
				e.Alpha = value
			}
		},
		func() {
			for _, e := range l.OuterSyllableElements {
				if e == nil {
					continue
				}
				e.Alpha = 1
			}
			l.GradientColorAnimate = nil
		},
	)
	lyrics.AnimateManager.Add(l.GradientColorAnimate)
}

func (AnimationLayer) FrameAnimate(l *Line, lyrics *Lyrics, fd float64) {
	if l == nil || lyrics == nil {
		return
	}
	if l.RenderMode == RenderModeLine {
		lineAnimationLayer.frameAnimateLineMode(l, lyrics)
		return
	}
	if l.GradientColorAnimate != nil {
		l.GradientColorAnimate.Cancel()
		l.GradientColorAnimate = nil
	}
	if len(l.OuterSyllableElements) == 0 {
		return
	}
	for _, e := range l.OuterSyllableElements {
		if e == nil {
			continue
		}
		e.Alpha = 1
	}

	if useRealtimeOffsetFormula {
		for _, e := range l.OuterSyllableElements {
			if e == nil || e.Animate == nil {
				continue
			}
			e.Animate.Cancel()
			e.Animate = nil
		}
		applyRealtimeOffsets(l.OuterSyllableElements, lyrics.Position, fd)
	} else {
		for elei, e := range l.OuterSyllableElements {
			kf := createFrames(l.OuterSyllableElements, elei, l.OuterSyllableElements[0].StartTime, l.OuterSyllableElements[len(l.OuterSyllableElements)-1].EndTime, fd)
			if e.Animate != nil {
				e.Animate.Cancel()
				e.Animate = nil
			}
			e.Animate = anim.NewKeyframeAnimation(
				uuid.NewString(),
				l.OuterSyllableElements[len(l.OuterSyllableElements)-1].EndTime-l.OuterSyllableElements[0].StartTime,
				l.OuterSyllableElements[0].StartTime-lyrics.Position,
				1,
				true,
				kf,
				func(value []float64) {
					e.NowOffset = value[0]
				},
				func() {},
			)
			lyrics.AnimateManager.Add(e.Animate)
		}
	}

	for _, word := range l.Participle {
		// 超过长度阈值的词禁用高亮动画，但仍保留上升动画。
		highlightChunks, allowHighlight := splitWordElementsByRuneLimit(l, word, maxHighlightWordRunes)
		for _, wordEle := range highlightChunks {
			duration := time.Duration(0)
			for _, ele := range wordEle {
				if ele == nil {
					continue
				}
				duration += ele.EndTime - ele.StartTime
			}
			if len(wordEle) == 0 {
				continue
			}
			if duration <= 0 {
				// 防御：当词元时间戳缺失时，使用 chunk 首尾时间兜底。
				duration = wordEle[len(wordEle)-1].EndTime - wordEle[0].StartTime
				if duration < 0 {
					duration = 0
				}
			}

			for nu, ele := range wordEle {
				if duration >= lyrics.HighlightTime && allowHighlight {
					scl := adaptiveWordScale(float64(duration.Milliseconds()), l.fontsize)
					hl := adaptiveBlurAmount(float64(duration.Milliseconds()), l.fontsize)
					hlap := anim.MapRange(float64(duration.Milliseconds()), 800, 3000, 0.1, 1)

					if ele.BackgroundBlurText == nil {
						ele.BackgroundBlurText = NewTextShadow(ele.Text, l.FontManager, l.FontRequest, l.fontsize)
					}
					ele.BackgroundBlurText.Blur = hl

					ele.HighlightAnimate = anim.NewKeyframeAnimation(
						uuid.NewString(),
						duration+200*time.Millisecond,
						wordEle[0].StartTime-lyrics.Position+duration/time.Duration(len(wordEle))*time.Duration(nu)/2,
						1,
						true,
						[]anim.Keyframe{
							{Offset: 0, Values: []float64{0, 1, 0}, Ease: nil},
							{Offset: 0.5, Values: []float64{getScaleOffset(nu, scl, wordEle), scl, hlap}, Ease: anim.EaseOut},
							{Offset: 1, Values: []float64{0, 1, 0}, Ease: anim.EaseInOut},
						},
						func(values []float64) {
							ele.Position.SetTranslateX(values[0])
							ele.Position.SetScaleX(values[1])
							ele.Position.SetScaleY(values[1])
							if ele.BackgroundBlurText != nil {
								ele.BackgroundBlurText.Alpha = values[2]
							}
						},
						func() {
							if ele.BackgroundBlurText != nil {
								ele.BackgroundBlurText.Dispose()
							}
							ele.BackgroundBlurText = nil
						},
					)
					lyrics.AnimateManager.Add(ele.HighlightAnimate)
				} else {
					// 当高亮被禁用或该词时长不足时，清理残留高亮状态。
					if ele.HighlightAnimate != nil {
						ele.HighlightAnimate.Cancel()
						ele.HighlightAnimate = nil
					}
					if ele.BackgroundBlurText != nil {
						ele.BackgroundBlurText.Dispose()
						ele.BackgroundBlurText = nil
					}
				}

				if ele.UpAnimate != nil {
					ele.UpAnimate.Cancel()
					ele.UpAnimate = nil
				}

				upDuration := ele.EndTime - ele.StartTime + 700*time.Millisecond
				upDelay := ele.StartTime - lyrics.Position
				if duration >= lyrics.HighlightTime {
					upDuration = duration + 700*time.Millisecond
					upDelay = wordEle[0].StartTime - lyrics.Position
				}
				ele.UpAnimate = anim.NewTween(
					uuid.NewString(),
					upDuration,
					upDelay,
					1,
					ele.GetPosition().GetTranslateY(),
					-adaptiveLift(l.fontsize),
					anim.EaseOut,
					func(value float64) {
						ele.GetPosition().SetTranslateY(value)
					},
					func() {},
				)
				lyrics.AnimateManager.Add(ele.UpAnimate)
			}
		}
	}
}

func (AnimationLayer) DisposeLineAnimations(l *Line) {
	lineAnimationLayer.disposeLineAnimationsWithManager(l, nil)
}

func (AnimationLayer) disposeLineAnimationsWithManager(l *Line, manager *anim.Manager) {
	if l == nil {
		return
	}
	if l.ScrollAnimate != nil {
		cancelManagedAnimation(manager, l.ScrollAnimate)
		l.ScrollAnimate = nil
	}
	if l.AlphaAnimate != nil {
		cancelManagedAnimation(manager, l.AlphaAnimate)
		l.AlphaAnimate = nil
	}
	if l.GradientColorAnimate != nil {
		cancelManagedAnimation(manager, l.GradientColorAnimate)
		l.GradientColorAnimate = nil
	}
	if l.ScaleAnimate != nil {
		cancelManagedAnimation(manager, l.ScaleAnimate)
		l.ScaleAnimate = nil
	}
	if l.StatusSettleAnimate != nil {
		cancelManagedAnimation(manager, l.StatusSettleAnimate)
		l.StatusSettleAnimate = nil
	}

	for _, e := range l.OuterSyllableElements {
		if e.Animate != nil {
			cancelManagedAnimation(manager, e.Animate)
			e.Animate = nil
		}
		if e.HighlightAnimate != nil {
			cancelManagedAnimation(manager, e.HighlightAnimate)
			e.HighlightAnimate = nil
		}
		if e.UpAnimate != nil {
			cancelManagedAnimation(manager, e.UpAnimate)
			e.UpAnimate = nil
		}
		if e.BackgroundBlurText != nil {
			e.BackgroundBlurText.Dispose()
			e.BackgroundBlurText = nil
		}
	}
}

func (AnimationLayer) DisposeLyricsAnimations(l *Lyrics) {
	if l == nil {
		return
	}
	for _, line := range l.Lines {
		lineAnimationLayer.disposeLineAnimationsWithManager(line, l.AnimateManager)
		for _, bgLine := range line.BackgroundLines {
			lineAnimationLayer.disposeLineAnimationsWithManager(bgLine, l.AnimateManager)
		}
	}
}

func (AnimationLayer) DisposeLyrics(l *Lyrics) {
	if l == nil {
		return
	}
	lineAnimationLayer.DisposeLyricsAnimations(l)
	for _, line := range l.Lines {
		lineRendererLayer.DisposeLine(line)
	}
	l.nowLyrics = nil
	l.renderIndex = nil
	l.Lines = nil
}

func sortIntSlice(arr []int) []int {
	length := len(arr)
	for i := 0; i < length; i++ {
		for j := 0; j < length-1-i; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
	return arr
}

func hasInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func removeInt(slice []int, val int) []int {
	newSlice := make([]int, 0)
	for _, item := range slice {
		if item != val {
			newSlice = append(newSlice, item)
		}
	}
	return newSlice
}

func getScaleOffset(index int, scale float64, doms []*SyllableElement) float64 {
	if len(doms) == 0 || index < 0 || index >= len(doms) {
		return 0
	}
	centerIndex := (len(doms) - 1) / 2
	cumulativeWidth := 0.0
	for i := 0; i < index; i++ {
		if doms[i] != nil && doms[i].SyllableImage != nil {
			cumulativeWidth += doms[i].SyllableImage.GetWidth()
		}
	}
	centerCumulativeWidth := 0.0
	for i := 0; i < centerIndex; i++ {
		if doms[i] != nil && doms[i].SyllableImage != nil {
			centerCumulativeWidth += doms[i].SyllableImage.GetWidth()
		}
	}

	return (cumulativeWidth - centerCumulativeWidth) * (scale - 1)
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func needsAnimation(from, to float64) bool {
	return math.Abs(from-to) > animationValueEpsilon
}

func adaptiveLineScale(fontSize float64) float64 {
	fontFactor := anim.MapRange(fontSize, 18, 96, 0.75, 1.25)
	scaleBoost := clampFloat(0.03*fontFactor, 0.018, 0.055)
	return 1 + scaleBoost
}

func inactiveLineScale(fontSize float64) float64 {
	fontFactor := anim.MapRange(fontSize, 18, 96, 0.75, 1.25)
	scaleReduce := clampFloat(0.03*fontFactor, 0.018, 0.055)
	return 1 - scaleReduce
}

func adaptiveWordScale(durationMs, fontSize float64) float64 {
	durationBoost := anim.MapRange(durationMs, 800, 3000, 0.02, 0.09)
	fontFactor := anim.MapRange(fontSize, 20, 90, 0.72, 1.28)
	scaleBoost := clampFloat(durationBoost*fontFactor, 0.015, 0.14)
	return 1 + scaleBoost
}

func adaptiveBlurAmount(durationMs, fontSize float64) float64 {
	durationBlur := anim.MapRange(durationMs, 800, 3000, 4.5, 11.5)
	fontFactor := anim.MapRange(fontSize, 20, 90, 0.7, 1.35)
	return clampFloat(durationBlur*fontFactor, 2, 24)
}

func adaptiveLift(fontSize float64) float64 {
	base := fontSize * 0.065
	fontFactor := anim.MapRange(fontSize, 20, 90, 0.75, 1.2)
	return clampFloat(base*fontFactor, 1.8, 12)
}

func lineVisibleAt(y, height, top, bottom float64) bool {
	if height <= 0 {
		height = 1
	}
	maxY := y + height
	return maxY >= top && y <= bottom
}

func backgroundLineReservesSpace(line *Line) bool {
	return line != nil && line.isShow && line.GetPosition().GetAlpha() > 0
}

func (AnimationLayer) updateFinalLayoutState(l *Lyrics, allEnded bool, changed bool) bool {
	if l == nil {
		return changed
	}
	terminalState := len(l.nowLyrics) == 0 && allEnded
	if !terminalState {
		l.finalLayoutPending = false
		return changed
	}

	hasVisibleBackground := lyricsHasVisibleBackgroundLines(l.Lines)
	if changed || hasVisibleBackground {
		l.finalLayoutPending = true
	}
	if l.finalLayoutPending && !hasVisibleBackground {
		changed = true
		l.finalLayoutPending = false
	}
	return changed
}

func lyricsHasVisibleBackgroundLines(lines []*Line) bool {
	for _, line := range lines {
		if line == nil {
			continue
		}
		for _, bg := range line.BackgroundLines {
			if backgroundLineReservesSpace(bg) {
				return true
			}
		}
	}
	return false
}

func lyricsAllEndedAt(lines []*Line, t time.Duration) bool {
	if len(lines) == 0 {
		return false
	}
	maxEnd := time.Duration(0)
	for _, line := range lines {
		if line != nil && line.EndTime > maxEnd {
			maxEnd = line.EndTime
		}
	}
	return t >= maxEnd
}

func findScrollAnchorIndexByTime(lines []*Line, t time.Duration) int {
	if len(lines) == 0 {
		return -1
	}
	if t <= lines[0].StartTime {
		return 0
	}

	lastEnded := 0
	for i, line := range lines {
		if t >= line.StartTime && t < line.EndTime {
			return i
		}
		if t >= line.EndTime {
			lastEnded = i
			continue
		}
		// gap region, keep previous line as anchor.
		return i - 1
	}
	return lastEnded
}

func shouldUseFastScroll(lyrics *Lyrics, currentIndex, targetIndex int) bool {
	if lyrics == nil || currentIndex < 0 || targetIndex < 0 || currentIndex >= len(lyrics.Lines) || targetIndex >= len(lyrics.Lines) {
		return false
	}
	if currentIndex == targetIndex {
		return false
	}

	currentLine := lyrics.Lines[currentIndex]
	targetLine := lyrics.Lines[targetIndex]
	if currentLine == nil || targetLine == nil {
		return false
	}

	gap := targetLine.StartTime - currentLine.EndTime
	return gap < scrollFastGapThreshold
}

func scrollDelayForIndex(anchorIndex, lineIndex int) time.Duration {
	baseDelay := scrollLeadTime()
	// 这里不做处理
	//if lineIndex <= anchorIndex {
	//	return baseDelay
	//}
	return baseDelay + time.Duration(lineIndex-anchorIndex)*scrollDelayStep
}
