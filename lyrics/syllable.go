package lyrics

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func NewSyllable(
	text string,
	startTime,
	endTime time.Duration,
	// SyllableImage
	font text.Face,
	fd float64,
	startColor,
	endColor color.RGBA,
) (*LineSyllable, error) {
	syllableImage, err := CreateSyllableImage(
		text,
		font,
		fd,
		startColor,
		endColor,
	)
	if err != nil {
		return nil, err
	}
	return &LineSyllable{
		Syllable:      text,
		StartTime:     startTime,
		EndTime:       endTime,
		SyllableImage: syllableImage,
		NowOffset:     syllableImage.Offset,
		Alpha:         1.0,
		Position: NewPosition(
			0, 0, 0, 0,
		),
	}, nil
}

func (ls *LineSyllable) SetNowOffset(offset float64) {
	ls.NowOffset = offset
}
func (ls *LineSyllable) GetNowOffset() float64 {
	return ls.NowOffset
}
func (ls *LineSyllable) SetAlpha(alpha float64) {
	ls.Alpha = alpha
}
func (ls *LineSyllable) GetAlpha() float64 {
	return ls.Alpha
}
func (ls *LineSyllable) SetPosition(pos Position) {
	ls.Position = pos
}
func (ls *LineSyllable) GetPosition() *Position {
	return &ls.Position
}
func (ls *LineSyllable) SetSyllableImage(s *SyllableImage) {
	ls.SyllableImage = s
}

func (ls *LineSyllable) GetSyllableImage() *SyllableImage {
	return ls.SyllableImage
}

func (ls *LineSyllable) SetStartTime(t time.Duration) {
	ls.StartTime = t
}
func (ls *LineSyllable) SetEndTime(t time.Duration) {
	ls.EndTime = t
}
func (ls *LineSyllable) GetStartTime() time.Duration {
	return ls.StartTime
}
func (ls *LineSyllable) GetEndTime() time.Duration {
	return ls.EndTime
}
func (ls *LineSyllable) GetSyllable() string {
	return ls.Syllable
}
func (ls *LineSyllable) SetSyllable(s string) {
	ls.Syllable = s
}
func (ls *LineSyllable) Duration() time.Duration {
	return ls.EndTime - ls.StartTime
}
func (ls *LineSyllable) IsInTime(t time.Duration) bool {
	return t >= ls.StartTime && t <= ls.EndTime
}
func (ls *LineSyllable) Dispose() {
	if ls.SyllableImage != nil {
		ls.SyllableImage.Dispose()
	}
}
