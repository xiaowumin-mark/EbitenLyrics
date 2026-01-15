package lyrics

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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

	needSptext bool,
) (*LineSyllable, error) {
	var eles []*SyllableElement
	if needSptext {

		lastX := 0.0
		stepTime := (endTime - startTime) / time.Duration(len(text))
		// 逐字拆分text
		for i, t := range text {
			syllableImage, err := CreateSyllableImage(
				string(t),
				font,
				fd,
				startColor,
				endColor,
			)
			if err != nil {
				return nil, err
			}
			po := NewPosition(lastX, 0, syllableImage.Width, syllableImage.Height)
			po.OriginX = po.GetW() / 2
			po.OriginY = po.GetH() * 6 / 5
			eles = append(eles, &SyllableElement{
				Text:          string(t),
				Position:      po,
				SyllableImage: syllableImage,
				NowOffset:     syllableImage.Offset,
				Alpha:         1,
				StartTime:     startTime + stepTime*time.Duration(i),
				EndTime:       stepTime*time.Duration(i+1) + startTime,
			})
			lastX += syllableImage.Width
		}
	} else {
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
		po := NewPosition(0, 0, syllableImage.Width, syllableImage.Height)
		po.OriginX = po.GetW() / 2
		po.OriginY = po.GetH() * 6 / 5
		eles = append(eles, &SyllableElement{
			Text:          text,
			Position:      po,
			SyllableImage: syllableImage,
			NowOffset:     syllableImage.Offset,
			Alpha:         1,
			StartTime:     startTime,
			EndTime:       endTime,
		})
	}
	return &LineSyllable{
		Syllable:  text,
		StartTime: startTime,
		EndTime:   endTime,
		//SyllableImage: syllableImage,
		Elements: eles,
		Alpha:    1.0,
	}, nil
}

func (ls *LineSyllable) Draw(screen *ebiten.Image) {
	for _, ele := range ls.Elements {

		if ele.BackgroundBlurText != nil {

			// 画文本背景模糊
			ele.BackgroundBlurText.Draw(
				screen,
				ele.Position.GetX()+ele.Position.GetTranslateX(),
				ele.Position.GetY()+ele.Position.GetTranslateY(),
			)

		}

		ele.SyllableImage.Draw(screen, ele.NowOffset, ele.Alpha, &ele.Position)
	}
}

func (ls *LineSyllable) SetAlpha(alpha float64) {
	ls.Alpha = alpha
}
func (ls *LineSyllable) GetAlpha() float64 {
	return ls.Alpha
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
	for _, ele := range ls.Elements {
		if ele != nil {
			ele.SyllableImage.Dispose()

		}
	}
}

func (ls *LineSyllable) SetFont(f text.Face) {
	lastX := 0.0
	for _, ele := range ls.Elements {
		ele.SyllableImage.SetFont(f)
		lastX += ele.SyllableImage.GetWidth()
		ele.GetPosition().SetX(lastX)
		ele.GetPosition().SetOriginX(
			ele.GetPosition().GetW() / 2,
		)
		ele.GetPosition().SetOriginY(
			ele.GetPosition().GetH() * 6 / 5,
		)
	}

}

func (ls *LineSyllable) Redraw() {
	for _, ele := range ls.Elements {
		ele.SyllableImage.Redraw()
		ele.GetPosition().SetOriginX(
			ele.GetPosition().GetW() / 2,
		)
		ele.GetPosition().SetOriginY(
			ele.GetPosition().GetH() * 6 / 5,
		)
	}
}
