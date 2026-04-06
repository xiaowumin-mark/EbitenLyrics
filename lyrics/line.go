package lyrics

// 文件说明：歌词行对象的创建与基础访问逻辑。
// 主要职责：维护字体、图像、时间轴和行级状态。

import (
	ft "EbitenLyrics/font"
	"EbitenLyrics/lp"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func NewLine(st, et time.Duration, isduet, isbg bool, ts string, fontManager *ft.FontManager, req ft.FontRequest, fs float64) *Line {
	pos := NewPosition(0, 0, 0, 0)
	baseScale := inactiveLineScale(fs)
	pos.SetScaleX(baseScale)
	pos.SetScaleY(baseScale)
	return &Line{
		StartTime:          st,
		EndTime:            et,
		Syllables:          []*LineSyllable{},
		Text:               "",
		Image:              nil,
		TranslateImage:     nil,
		IsDuet:             isduet,
		IsBackground:       isbg,
		TranslatedText:     ts,
		BackgroundLines:    []*Line{},
		Participle:         [][]int{},
		SmartTranslateWrap: true,
		fontsize:           fs,
		isShow:             false,
		Status:             LineStatusHidden,
		imageDirty:         true,
		FontManager:        fontManager,
		FontRequest:        req.Normalized(),
		Position:           pos,
	}
}

func (l *Line) composeFace(size float64) text.Face {
	return l.composeFaceForText("", size)
}

func (l *Line) composeFaceForText(content string, size float64) text.Face {
	if l == nil || size <= 0 {
		return nil
	}
	if l.FontManager == nil {
		return nil
	}
	face, err := l.FontManager.GetFaceForText(l.FontRequest, size, content)
	if err != nil {
		return nil
	}
	return face
}

func (l *Line) activeFace() text.Face {
	return l.composeFaceForText(l.Text, l.fontsize)
}

func (l *Line) translatedFace() text.Face {
	if l == nil {
		return nil
	}
	return l.composeFaceForText(l.TranslatedText, l.fontsize/2)
}

func safeImageLength(v float64) int {
	return lp.LPSize(v)
}

func (l *Line) GetSyllables() []*LineSyllable {
	return l.Syllables
}

func (l *Line) SetSyllables(syllables []*LineSyllable) {
	for _, oldSyllable := range l.Syllables {
		if oldSyllable != nil {
			oldSyllable.Dispose()
		}
	}
	l.Syllables = syllables

	var textBuilder strings.Builder
	for _, syllable := range l.Syllables {
		textBuilder.WriteString(syllable.Syllable)
	}
	l.Text = textBuilder.String()

	var outerSyllableElements []*SyllableElement
	index := 0
	for i, syllable := range l.Syllables {
		outerSyllableElements = append(outerSyllableElements, syllable.Elements...)
		for _, element := range syllable.Elements {
			element.SyllableIndex = i
			element.OuterSyllableElementsIndex = index
			index++
		}
	}
	l.OuterSyllableElements = outerSyllableElements
	l.markImageDirty()
}

func (l *Line) GetFace(isc bool) text.Face {
	_ = isc
	return l.activeFace()
}

func (l *Line) AddBackgroundLine(line *Line) {
	l.BackgroundLines = append(l.BackgroundLines, line)
}

func (l *Line) GetBackgroundLines() []*Line {
	return l.BackgroundLines
}

func (l *Line) GetText() string {
	return l.Text
}

func (l *Line) SetTranslatedText(ts string) {
	l.TranslatedText = ts
	l.markImageDirty()
}

func (l *Line) SetSmartTranslateWrap(enabled bool) {
	l.SmartTranslateWrap = enabled
	for _, bg := range l.BackgroundLines {
		if bg == nil {
			continue
		}
		bg.SetSmartTranslateWrap(enabled)
	}
}

func (l *Line) GetTranslatedText() string {
	return l.TranslatedText
}

func (l *Line) SetPadding(padding float64) {
	l.Padding = padding
	l.markImageDirty()
}

func (l *Line) GetPadding() float64 {
	return l.Padding
}

func (l *Line) GetStartTime() time.Duration {
	return l.StartTime
}

func (l *Line) GetEndTime() time.Duration {
	return l.EndTime
}

func (l *Line) Duration() time.Duration {
	return l.EndTime - l.StartTime
}

func (l *Line) IsInTime(t time.Duration) bool {
	return t >= l.StartTime && t <= l.EndTime
}

func (l *Line) GetIsDuet() bool {
	return l.IsDuet
}

func (l *Line) GetIsBackground() bool {
	return l.IsBackground
}

func (l *Line) GetImage() *ebiten.Image {
	return l.Image
}

func (l *Line) GetTranslateImage() *ebiten.Image {
	return l.TranslateImage
}

func (l *Line) SetImage(img *ebiten.Image) {
	l.Image = img
	l.markImageDirty()
}

func (l *Line) SetTranslateImage(img *ebiten.Image) {
	l.TranslateImage = img
	l.markImageDirty()
}

func (l *Line) GetPosition() *Position {
	return &l.Position
}

func (l *Line) SetPosition(pos Position) {
	l.Position = pos
}

func (l *Line) SetFD(fd float64) {
	for _, e := range l.OuterSyllableElements {
		if e.SyllableImage == nil {
			continue
		}
		e.SyllableImage.SetFd(fd)
		e.NowOffset = e.SyllableImage.Offset
	}
	for _, line := range l.BackgroundLines {
		line.SetFD(fd)
	}
	l.markImageDirty()
}

func (l *Lyrics) GetRenderedindex() []int {
	return l.renderIndex
}

func (l *Line) markImageDirty() {
	if l == nil {
		return
	}
	l.imageDirty = true
}

func (l *Line) setStatus(status LineStatus) {
	if l == nil || l.Status == status {
		return
	}
	prev := l.Status
	l.Status = status
	if status.UsesPreviewBitmap() && !prev.UsesPreviewBitmap() {
		l.markImageDirty()
	}
}
