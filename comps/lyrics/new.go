package LyricsComponent

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/ttml"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LyricsComponent struct {
	LyricsControl  *lyrics.Lyrics
	AnimateManager *anim.Manager
	Font           *text.GoTextFaceSource
	Width, Height  float64
	FontSize       float64
	FD             float64
	Image          *ebiten.Image
}

func NewLyricsComponent(anim *anim.Manager, f *text.GoTextFaceSource, w, h, fs, fd float64) *LyricsComponent {
	return &LyricsComponent{
		AnimateManager: anim,
		Font:           f,
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

func (l *LyricsComponent) SetLyrics(ls []ttml.LyricLine) *LyricsComponent {
	l.recreateImage()
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
	}

	control, err := lyrics.New(ls, l.Width, l.Font, l.FontSize, l.FD)
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

func (l *LyricsComponent) SetFont(font *text.GoTextFaceSource) *LyricsComponent {
	if l.LyricsControl == nil {
		return l
	}
	l.Font = font
	for _, line := range l.LyricsControl.Lines {
		line.SetFont(font)
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
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
		l.LyricsControl = nil
	}
	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
	}
}
