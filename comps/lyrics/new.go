package LyricsComponent

// 文件说明：歌词组件入口，负责把歌词核心模块接入页面。
// 主要职责：管理歌词对象的创建、更新、缩放、绘制与资源释放。

import (
	"log"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
	"github.com/xiaowumin-mark/EbitenLyrics/lyrics"

	ttml "github.com/xiaowumin-mark/EbitenLyrics/ttml"

	"github.com/hajimehoshi/ebiten/v2"
)

const lyricsSwitchFadeDuration = 280 * time.Millisecond

type LyricsComponent struct {
	LyricsControl      *lyrics.Lyrics
	AnimateManager     *anim.Manager
	FontManager        *ft.FontManager
	FontRequest        ft.FontRequest
	Width, Height      float64
	FontSize           float64
	FD                 float64
	SmartTranslateWrap bool
	Image              *ebiten.Image
	StaticImage        *ebiten.Image
	TransitionImage    *ebiten.Image

	staticLayerSignature uint64
	staticLayerReady     bool
	switchFadeStart      time.Time
	switchFadeDuration   time.Duration
	switchFadeActive     bool
}

func NewLyricsComponent(anim *anim.Manager, fontManager *ft.FontManager, req ft.FontRequest, w, h, fs, fd float64) *LyricsComponent {
	return &LyricsComponent{
		AnimateManager:     anim,
		FontManager:        fontManager,
		FontRequest:        req.Normalized(),
		Width:              w,
		Height:             h,
		FontSize:           fs,
		FD:                 fd,
		SmartTranslateWrap: true,
		switchFadeDuration: lyricsSwitchFadeDuration,
	}
}

func safeImageSize(v float64) int {
	return lp.LPSize(v)
}

func (l *LyricsComponent) recreateImage() {
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
	}
	if l.StaticImage != nil {
		l.StaticImage.Deallocate()
		l.StaticImage = nil
	}
	if l.TransitionImage != nil {
		l.TransitionImage.Deallocate()
		l.TransitionImage = nil
	}
	l.Image = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
	l.StaticImage = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
	l.staticLayerSignature = 0
	l.staticLayerReady = false
	l.switchFadeStart = time.Time{}
	l.switchFadeActive = false
}

func (l *LyricsComponent) Init() {
	l.recreateImage()
}

func releaseLyricsMemory() {
	lyrics.PurgeSharedImageCache()
	runtime.GC()
	debug.FreeOSMemory()
}

func crossfadeProgress(start time.Time, duration time.Duration, now time.Time) float64 {
	if start.IsZero() || duration <= 0 {
		return 1
	}
	if now.Before(start) {
		return 0
	}
	progress := float64(now.Sub(start)) / float64(duration)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func (l *LyricsComponent) finishSwitchFade() {
	l.switchFadeActive = false
	l.switchFadeStart = time.Time{}
}

func (l *LyricsComponent) ensureTransitionImage() {
	if l.TransitionImage != nil {
		currentW, currentH := l.TransitionImage.Size()
		if currentW != safeImageSize(l.Width) || currentH != safeImageSize(l.Height) {
			l.TransitionImage.Deallocate()
			l.TransitionImage = nil
		}
	}
	if l.TransitionImage == nil {
		l.TransitionImage = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
	}
}

func drawImageWithAlpha(dst, src *ebiten.Image, alpha float64) {
	if dst == nil || src == nil || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(src, op)
}

func (l *LyricsComponent) drawToScreen(screen *ebiten.Image, image *ebiten.Image, p *lyrics.Position, alpha float64) {
	if screen == nil || image == nil || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	dr := &ebiten.DrawImageOptions{}
	dr.ColorScale.ScaleAlpha(float32(alpha))
	if p != nil {
		dr.GeoM = lyrics.TransformToGeoM(p)
	}
	screen.DrawImage(image, dr)
}

func (l *LyricsComponent) renderLyricsImage() bool {
	if l.LyricsControl == nil {
		return false
	}
	if l.Image == nil || l.StaticImage == nil {
		l.recreateImage()
	}
	l.Image.Clear()
	signature, hasStaticLayer, needsStaticRebuild := l.LyricsControl.StaticLayerSignature()
	if hasStaticLayer {
		if !l.staticLayerReady || l.staticLayerSignature != signature || needsStaticRebuild {
			l.StaticImage.Clear()
			l.LyricsControl.DrawStatic(l.StaticImage)
			l.staticLayerSignature = signature
			l.staticLayerReady = true
		}
		l.Image.DrawImage(l.StaticImage, &ebiten.DrawImageOptions{})
	} else {
		l.staticLayerSignature = 0
		l.staticLayerReady = false
	}
	l.LyricsControl.DrawDynamic(l.Image)
	return true
}

func (l *LyricsComponent) snapshotCurrentFrame() bool {
	now := time.Now()
	progress := 1.0
	hasPrevious := false
	if l.switchFadeActive && l.TransitionImage != nil {
		progress = crossfadeProgress(l.switchFadeStart, l.switchFadeDuration, now)
		hasPrevious = progress < 1
	}

	hasCurrent := l.renderLyricsImage()
	if !hasPrevious && !hasCurrent {
		return false
	}

	var snapshot *ebiten.Image
	if hasPrevious {
		snapshot = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
		drawImageWithAlpha(snapshot, l.TransitionImage, 1-progress)
		if hasCurrent {
			drawImageWithAlpha(snapshot, l.Image, progress)
		}
		if l.TransitionImage != nil {
			l.TransitionImage.Deallocate()
		}
		l.TransitionImage = snapshot
		return true
	}

	l.ensureTransitionImage()
	l.TransitionImage.Clear()
	drawImageWithAlpha(l.TransitionImage, l.Image, 1)
	return true
}

/*func (l *LyricsComponent) SetLyrics(ls []ttml.LyricLine) *LyricsComponent {
	l.recreateImage()
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
		releaseLyricsMemory()
	}

	control, err := lyrics.New(ls, l.Width, l.FontManager, l.FontRequest, l.FontSize, l.FD)
	if err != nil {
		log.Printf("lyrics init failed: %v", err)
		return l
	}
	l.LyricsControl = control
	l.LyricsControl.AnimateManager = l.AnimateManager
	l.LyricsControl.HighlightTime = time.Millisecond * 800
	l.LyricsControl.Scroll([]int{0}, 0)
	return l
}*/
// 在 comps/lyrics/new.go 中优化 SetLyrics 方法
func (l *LyricsComponent) SetLyrics(ls []ttml.LyricLine) *LyricsComponent {
	hadOutgoingFrame := l.snapshotCurrentFrame()

	// 重用现有图像，避免频繁分配
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
		releaseLyricsMemory()
	}

	// 只有在尺寸变化时才重新创建图像
	if l.Image != nil {
		currentW, currentH := l.Image.Size()
		if currentW != safeImageSize(l.Width) || currentH != safeImageSize(l.Height) {
			l.Image.Deallocate()
			l.Image = nil
		}
	}
	if l.StaticImage != nil {
		currentW, currentH := l.StaticImage.Size()
		if currentW != safeImageSize(l.Width) || currentH != safeImageSize(l.Height) {
			l.StaticImage.Deallocate()
			l.StaticImage = nil
		}
	}

	if l.Image == nil || l.StaticImage == nil {
		l.recreateImage()
	}

	// 重新处理歌词的的结束时间
	for i := range ls {
		for _, bg := range ls[i].BGs {
			if ls[i].EndTime < bg.EndTime {
				ls[i].EndTime = bg.EndTime
			}
		}
	}

	control, err := lyrics.New(ls, l.Width, l.FontManager, l.FontRequest, l.FontSize, l.FD)
	if err != nil {
		log.Printf("lyrics init failed: %v", err)
		if hadOutgoingFrame {
			l.switchFadeStart = time.Now()
			l.switchFadeActive = true
		} else {
			l.finishSwitchFade()
		}
		return l
	}
	l.LyricsControl = control
	l.LyricsControl.AnimateManager = l.AnimateManager
	l.LyricsControl.HighlightTime = time.Millisecond * 800
	for _, line := range l.LyricsControl.Lines {
		line.SetSmartTranslateWrap(l.SmartTranslateWrap)
	}
	l.LyricsControl.Scroll([]int{0}, 0)
	l.staticLayerSignature = 0
	l.staticLayerReady = false
	if hadOutgoingFrame {
		l.switchFadeStart = time.Now()
		l.switchFadeActive = true
	} else {
		l.finishSwitchFade()
	}
	return l
}

func (l *LyricsComponent) Update(t time.Duration) {
	if l.LyricsControl == nil {
		return
	}
	l.LyricsControl.Position = t
	l.LyricsControl.Update(l.LyricsControl.Position)
}

func (l *LyricsComponent) Resize(w, h float64) {
	if w <= 0 || h <= 0 {
		return
	}
	l.Width, l.Height = w, h
	l.recreateImage()
	if l.LyricsControl == nil {
		return
	}
	l.LyricsControl.Resize(w)
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
}

func (l *LyricsComponent) SetFontSize(fs float64) *LyricsComponent {
	if l.LyricsControl == nil || fs <= 0 {
		return l
	}
	l.FontSize = fs
	for _, line := range l.LyricsControl.Lines {
		line.SetFontSize(fs)
	}
	currentPosition := l.LyricsControl.Position
	l.LyricsControl.Update(currentPosition)
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
	l.staticLayerSignature = 0
	l.staticLayerReady = false
	return l
}

func (l *LyricsComponent) SetFont(fontManager *ft.FontManager, req ft.FontRequest) *LyricsComponent {
	if l.LyricsControl == nil {
		return l
	}
	l.FontManager = fontManager
	l.FontRequest = req.Normalized()
	for _, line := range l.LyricsControl.Lines {
		line.SetFont(fontManager, l.FontRequest)
	}
	currentPosition := l.LyricsControl.Position
	l.LyricsControl.Update(currentPosition)
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
	l.staticLayerSignature = 0
	l.staticLayerReady = false
	return l
}

func (l *LyricsComponent) SetFD(fd float64) *LyricsComponent {
	if l.LyricsControl == nil {
		return l
	}
	l.FD = fd
	l.LyricsControl.FD = fd

	for _, line := range l.LyricsControl.Lines {
		line.SetFD(fd)
	}
	for _, idx := range l.LyricsControl.GetRenderedindex() {
		if idx < 0 || idx >= len(l.LyricsControl.Lines) {
			continue
		}
		l.LyricsControl.Lines[idx].SetFD(fd)
		l.LyricsControl.Lines[idx].Render()
	}
	return l
}

func (l *LyricsComponent) SetSmartTranslateWrap(enabled bool) *LyricsComponent {
	l.SmartTranslateWrap = enabled
	if l.LyricsControl == nil {
		return l
	}
	for _, line := range l.LyricsControl.Lines {
		line.SetSmartTranslateWrap(enabled)
		line.GenerateTSImage()
	}
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
	return l
}

func (l *LyricsComponent) Draw(screen *ebiten.Image, p *lyrics.Position) {
	if screen == nil {
		return
	}

	hasCurrentFrame := l.renderLyricsImage()
	if l.switchFadeActive && l.TransitionImage != nil {
		progress := crossfadeProgress(l.switchFadeStart, l.switchFadeDuration, time.Now())
		if progress >= 1 {
			l.finishSwitchFade()
		} else {
			l.drawToScreen(screen, l.TransitionImage, p, 1-progress)
			if hasCurrentFrame {
				l.drawToScreen(screen, l.Image, p, progress)
			}
			return
		}
	}
	if hasCurrentFrame {
		l.drawToScreen(screen, l.Image, p, 1)
	}
}

func (l *LyricsComponent) Dispose() {
	released := false
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
		released = true
	}
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
		released = true
	}
	if l.StaticImage != nil {
		l.StaticImage.Deallocate()
		l.StaticImage = nil
		released = true
	}
	if l.TransitionImage != nil {
		l.TransitionImage.Deallocate()
		l.TransitionImage = nil
		released = true
	}
	l.staticLayerSignature = 0
	l.staticLayerReady = false
	l.finishSwitchFade()
	if released {
		releaseLyricsMemory()
	}
}
