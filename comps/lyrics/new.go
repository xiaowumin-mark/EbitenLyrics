package LyricsComponent

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/ttml"
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
func (l *LyricsComponent) Init() {
	l.Image = ebiten.NewImage(int(l.Width), int(l.Height))
}
func (l *LyricsComponent) SetLyrics(ls []ttml.LyricLine) *LyricsComponent {
	if l.Image != nil {

		if l.Width > 0 && l.Height > 0 {
			l.Image.Deallocate()
			l.Image = nil
			l.Image = ebiten.NewImage(int(l.Width), int(l.Height))
		}

	}
	if l.LyricsControl != nil {
		l.LyricsControl.Dispose()
	}
	l.LyricsControl, _ = lyrics.New(ls, l.Width, l.Font, l.FontSize, l.FD)
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
	if w < 0 || h < 0 {
		return
	}

	if l.Image != nil {
		l.Image.Deallocate()
		l.Image = nil
		l.Image = ebiten.NewImage(int(w), int(h))
	}
	if l.LyricsControl == nil {
		return
	}
	l.Width, l.Height = w, h
	l.LyricsControl.Resize(w)
}

func (l *LyricsComponent) SetFontSize(fs float64) *LyricsComponent {
	if l.LyricsControl == nil {
		return l
	}
	l.FontSize = fs
	for _, line := range l.LyricsControl.Lines {
		line.SetFontSize(fs)
	}
	l.LyricsControl.Scroll(
		l.LyricsControl.GetNowLyrics(),
		0,
	)
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
	l.LyricsControl.Scroll(
		l.LyricsControl.GetNowLyrics(),
		0,
	)
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
	for i := range l.LyricsControl.GetRenderedindex() {
		l.LyricsControl.Lines[i].SetFD(fd)
		l.LyricsControl.Lines[i].Render()
	}
	return l
}

func (l *LyricsComponent) Draw(screen *ebiten.Image, p *lyrics.Position) {
	if l.Image == nil || l.LyricsControl == nil {
		return
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
	l.LyricsControl.Dispose()
}
