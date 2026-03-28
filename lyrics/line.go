package lyrics

// 文件说明：歌词行对象的创建与基础访问逻辑。
// 主要职责：维护字体、图像、时间轴和行级状态。

import (
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func NewLine(st, et time.Duration, isduet, isbg bool, ts string, font *text.GoTextFaceSource, fallbacks []*text.GoTextFaceSource, fs float64) *Line {
	pos := NewPosition(0, 0, 0, 0)
	baseScale := inactiveLineScale(fs)
	pos.SetScaleX(baseScale)
	pos.SetScaleY(baseScale)
	return &Line{
		StartTime:       st,
		EndTime:         et,
		Syllables:       []*LineSyllable{},
		Text:            "",
		Image:           nil,
		TranslateImage:  nil,
		IsDuet:          isduet,
		IsBackground:    isbg,
		TranslatedText:  ts,
		BackgroundLines: []*Line{},
		Participle:      [][]int{},
		fontsize:        fs,
		isShow:          false,
		Font:            font,
		FallbackFonts:   append([]*text.GoTextFaceSource{}, fallbacks...),
		Position:        pos,
	}
}

func (l *Line) faceSources() []*text.GoTextFaceSource {
	if l == nil {
		return nil
	}
	out := make([]*text.GoTextFaceSource, 0, 1+len(l.FallbackFonts))
	seen := map[*text.GoTextFaceSource]struct{}{}
	if l.Font != nil {
		out = append(out, l.Font)
		seen[l.Font] = struct{}{}
	}
	for _, fallback := range l.FallbackFonts {
		if fallback == nil {
			continue
		}
		if _, ok := seen[fallback]; ok {
			continue
		}
		seen[fallback] = struct{}{}
		out = append(out, fallback)
	}
	return out
}

func (l *Line) composeFace(size float64) text.Face {
	if l == nil || size <= 0 {
		return nil
	}
	sources := l.faceSources()
	if len(sources) == 0 {
		return nil
	}

	faces := make([]text.Face, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		faces = append(faces, &text.GoTextFace{
			Source: source,
			Size:   size,
		})
	}
	if len(faces) == 0 {
		return nil
	}
	if len(faces) == 1 {
		return faces[0]
	}
	multiFace, err := text.NewMultiFace(faces...)
	if err != nil {
		return faces[0]
	}
	return multiFace
}

func (l *Line) activeFace() text.Face {
	if l == nil {
		return nil
	}
	if l.Face != nil {
		return l.Face
	}
	l.Face = l.composeFace(l.fontsize)
	return l.Face
}

func (l *Line) translatedFace() text.Face {
	if l == nil {
		return nil
	}
	return l.composeFace(l.fontsize / 2)
}

func safeImageLength(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 1
	}
	return int(math.Ceil(v))
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
}

func (l *Line) GetTranslatedText() string {
	return l.TranslatedText
}

func (l *Line) SetPadding(padding float64) {
	l.Padding = padding
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
}

func (l *Line) SetTranslateImage(img *ebiten.Image) {
	l.TranslateImage = img
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
}

func (l *Lyrics) GetRenderedindex() []int {
	return l.renderIndex
}
