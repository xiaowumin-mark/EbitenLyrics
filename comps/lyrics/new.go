package LyricsComponent

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/ttml"
	"log"
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LyricsComponent struct {
	LyricsControl  *lyrics.Lyrics
	AnimateManager *anim.Manager
	Font           *text.GoTextFaceSource
	FallbackFonts  []*text.GoTextFaceSource
	Width, Height  float64
	FontSize       float64
	FD             float64
	Image          *ebiten.Image
}

func NewLyricsComponent(anim *anim.Manager, f *text.GoTextFaceSource, fallbacks []*text.GoTextFaceSource, w, h, fs, fd float64) *LyricsComponent {
	return &LyricsComponent{
		AnimateManager: anim,
		Font:           f,
		FallbackFonts:  append([]*text.GoTextFaceSource{}, fallbacks...),
		Width:          w,
		Height:         h,
		FontSize:       fs,
		FD:             fd,
	}
}

func safeImageSize(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 1
	}
	return int(math.Ceil(v))
}

func (l *LyricsComponent) recreateImage() {
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
	}
	l.Image = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
}

func (l *LyricsComponent) Init() {
	l.recreateImage()
}

func releaseLyricsMemory() {
	lyrics.PurgeSharedImageCache()
	runtime.GC()
	debug.FreeOSMemory()
}

/*func (l *LyricsComponent) SetLyrics(ls []ttml.LyricLine) *LyricsComponent {
	l.recreateImage()
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
		releaseLyricsMemory()
	}

	control, err := lyrics.New(ls, l.Width, l.Font, l.FallbackFonts, l.FontSize, l.FD)
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

	if l.Image == nil {
		l.Image = ebiten.NewImage(safeImageSize(l.Width), safeImageSize(l.Height))
	}

	control, err := lyrics.New(ls, l.Width, l.Font, l.FallbackFonts, l.FontSize, l.FD)
	if err != nil {
		log.Printf("lyrics init failed: %v", err)
		return l
	}
	l.LyricsControl = control
	l.LyricsControl.AnimateManager = l.AnimateManager
	l.LyricsControl.HighlightTime = time.Millisecond * 800
	l.LyricsControl.Scroll([]int{0}, 0)
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
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
	return l
}

func (l *LyricsComponent) SetFont(font *text.GoTextFaceSource, fallbacks []*text.GoTextFaceSource) *LyricsComponent {
	if l.LyricsControl == nil {
		return l
	}
	l.Font = font
	l.FallbackFonts = append([]*text.GoTextFaceSource{}, fallbacks...)
	for _, line := range l.LyricsControl.Lines {
		line.SetFont(font, fallbacks)
	}
	l.LyricsControl.Scroll(l.LyricsControl.GetNowLyrics(), 0)
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

func (l *LyricsComponent) Draw(screen *ebiten.Image, p *lyrics.Position) {
	if screen == nil || l.LyricsControl == nil {
		return
	}
	if l.Image == nil {
		l.recreateImage()
	}
	l.Image.Clear()
	l.LyricsControl.Draw(l.Image)

	dr := &ebiten.DrawImageOptions{}
	if p != nil {
		dr.GeoM = lyrics.TransformToGeoM(p)
	}
	screen.DrawImage(l.Image, dr)
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
	if released {
		releaseLyricsMemory()
	}
}
