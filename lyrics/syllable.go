package lyrics

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func NewSyllable(
	t string,
	startTime,
	endTime time.Duration,
	font text.Face,
	fd float64,
	startColor,
	endColor color.RGBA,
	needSptext bool,
) (*LineSyllable, error) {
	chars := []rune(t)
	if needSptext && len(chars) == 0 {
		needSptext = false
	}

	var eles []*SyllableElement
	if needSptext {
		lastX := 0.0
		stepTime := time.Duration(0)
		if len(chars) > 0 {
			stepTime = (endTime - startTime) / time.Duration(len(chars))
		}

		for i, ch := range chars {
			syllableImage, err := CreateSyllableImage(
				string(ch),
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
				Text:          string(ch),
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
			t,
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
			Text:          t,
			Position:      po,
			SyllableImage: syllableImage,
			NowOffset:     syllableImage.Offset,
			Alpha:         1,
			StartTime:     startTime,
			EndTime:       endTime,
		})
	}

	return &LineSyllable{
		Syllable:  t,
		StartTime: startTime,
		EndTime:   endTime,
		Elements:  eles,
		Alpha:     1.0,
	}, nil
}

func (ls *LineSyllable) Draw(screen *ebiten.Image) {
	for _, ele := range ls.Elements {
		if ele == nil || ele.SyllableImage == nil {
			continue
		}

		if ele.BackgroundBlurText != nil {
			ele.BackgroundBlurText.Draw(screen, &ele.Position)
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
		if ele == nil {
			continue
		}
		if ele.BackgroundBlurText != nil {
			ele.BackgroundBlurText.Dispose()
			ele.BackgroundBlurText = nil
		}
		if ele.SyllableImage != nil {
			ele.SyllableImage.Dispose()
		}
	}
}

func (ls *LineSyllable) SetFont(f text.Face) {
	lastX := 0.0
	for _, ele := range ls.Elements {
		if ele == nil || ele.SyllableImage == nil {
			continue
		}
		ele.SyllableImage.SetFont(f)
		ele.GetPosition().SetX(lastX)
		lastX += ele.SyllableImage.GetWidth()
		ele.GetPosition().SetOriginX(ele.GetPosition().GetW() / 2)
		ele.GetPosition().SetOriginY(ele.GetPosition().GetH() * 6 / 5)
	}
}

func (ls *LineSyllable) Redraw() {
	for _, ele := range ls.Elements {
		if ele == nil || ele.SyllableImage == nil {
			continue
		}
		ele.SyllableImage.Redraw()
		ele.GetPosition().SetOriginX(ele.GetPosition().GetW() / 2)
		ele.GetPosition().SetOriginY(ele.GetPosition().GetH() * 6 / 5)
	}
}
