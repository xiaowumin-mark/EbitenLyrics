package lyrics

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/ttml"
	"image/color"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func New(ttmllines []ttml.LyricLine, screenW float64, f *text.GoTextFaceSource, fs float64) (*Lyrics, error) {
	var lyrics Lyrics
	for _, line := range ttmllines {
		l := NewLine(time.Duration(line.StartTime)*time.Millisecond, time.Duration(line.EndTime)*time.Millisecond, line.IsDuet, line.IsBG, line.TranslatedLyric, f, fs)
		l.Position.SetW(screenW * 0.8)
		l.SetPadding(20)
		if line.IsDuet {
			l.Position.SetX(screenW - l.Position.GetW())
		}
		err := CreateSyllable(line.Words, l)
		if err != nil {
			return nil, err
		}
		l.Layout()
		for _, bgline := range line.BGs {
			lbg := NewLine(time.Duration(bgline.StartTime)*time.Millisecond, time.Duration(bgline.EndTime)*time.Millisecond, bgline.IsDuet, bgline.IsBG, bgline.TranslatedLyric, f, fs/1.5)
			lbg.Position.SetW(screenW * 0.8)
			lbg.SetPadding(20)
			if line.IsDuet {
				lbg.Position.SetX(screenW - lbg.Position.GetW())
			}
			err := CreateSyllable(bgline.Words, lbg)
			if err != nil {
				return nil, err
			}
			lbg.Layout()
			lbg.Position.SetAlpha(0)
			l.AddBackgroundLine(lbg)

		}
		lyrics.Lines = append(lyrics.Lines, l)
	}
	return &lyrics, nil
}

func CreateSyllable(ts []ttml.LyricWord, line *Line) error {
	var syllables []*LineSyllable
	var ap uint8 = 255
	if line.IsBackground {
		ap = 130
	}
	p := SplitBySpaceTTML(ts, true)
	for _, pi := range p {
		var du time.Duration
		for _, e := range pi {
			du += time.Duration(ts[e].EndTime-ts[e].StartTime) * time.Millisecond
		}
		issp := false
		if du >= 800*time.Millisecond {
			issp = true
		}
		for _, ti := range pi {
			t := ts[ti]
			s, err := NewSyllable(
				t.Word,
				time.Duration(t.StartTime)*time.Millisecond,
				time.Duration(t.EndTime)*time.Millisecond,
				*line.GetFace(false),
				1,
				color.RGBA{255, 255, 255, ap},
				color.RGBA{255, 255, 255, 60},
				issp,
			)
			if err != nil {
				return err
			}
			syllables = append(syllables, s)
		}

	}
	line.SetSyllables(syllables)
	return nil
}

func (l *Line) LineAnimate(lyrics *Lyrics) {
	l.FrameAnimate(lyrics)

	for _, it := range l.BackgroundLines {
		it.AlphaAnimate = anim.NewKeyframeAnimation(
			uuid.NewString(),
			time.Duration(700)*time.Millisecond,
			time.Duration(200)*time.Millisecond,
			1,
			false,
			[]anim.Keyframe{
				{
					Offset: 0,
					Values: []float64{it.GetPosition().GetAlpha(), 0.92},
					Ease:   anim.EaseOut,
				},
				{
					Offset: 1,
					Values: []float64{1, 1},
					Ease:   anim.EaseOut,
				},
			},
			func(value []float64) {
				//e.Alpha = value
				/*for _, e := range it.Syllables {
					e.Alpha = value
				}*/
				it.Position.SetAlpha(value[0])
				it.Position.SetScaleX(value[1])
				it.Position.SetScaleY(value[1])
			},
			func() {
				/*for _, e := range it.Elements {
					e.Alpha = 1
				}*/
				it.Position.SetAlpha(1)
				it.Position.SetScaleX(1)
				it.Position.SetScaleY(1)
			},
		)
		lyrics.AnimateManager.Add(it.AlphaAnimate)

		it.FrameAnimate(lyrics)
	}
}

func (l *Line) FrameAnimate(lyrics *Lyrics) {
	l.Status = Hot

	for elei, e := range l.OuterSyllableElements {
		kf := createFrames(l.OuterSyllableElements, elei, l.OuterSyllableElements[0].StartTime, l.OuterSyllableElements[len(l.OuterSyllableElements)-1].EndTime, 1)
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
			func() {
				//e.NowOffset = 0、
				l.Status = Buffered
			},
		)
		lyrics.AnimateManager.Add(e.Animate)
	}

	for _, word := range l.Participle {
		var duration time.Duration
		for _, i := range word {
			duration += l.Syllables[i].EndTime - l.Syllables[i].StartTime
		}

		var wordEle []*SyllableElement
		for _, syllable := range word {
			wordEle = append(wordEle, l.Syllables[syllable].Elements...)
		}

		for nu, ele := range wordEle {
			if duration >= lyrics.HighlightTime {
				// 创建shadow

				scl := anim.MapRange(
					float64(duration.Milliseconds()),
					800,
					3000,
					1.02,
					1.09,
				)

				hl := anim.MapRange(
					float64(duration.Milliseconds()),
					800,
					3000,
					2,
					10,
				)

				hlap := anim.MapRange(
					float64(duration.Milliseconds()),
					800,
					3000,
					0.1,
					0.9,
				)

				if ele.BackgroundBlurText == nil {
					ele.BackgroundBlurText = NewTextShadow(ele.Text, l.Font, l.fontsize)
					ele.BackgroundBlurText.Blur = hl
					// 复制一份position到ele.BackgroundBlurText
				}

				ele.HighlightAnimate = anim.NewKeyframeAnimation(
					uuid.NewString(),
					duration+time.Duration(200)*time.Millisecond,
					wordEle[0].StartTime-lyrics.Position+duration/time.Duration(len(wordEle))*time.Duration(nu)/2,
					1,
					true,
					[]anim.Keyframe{
						{
							Offset: 0,
							Values: []float64{0, 1, 0},
							Ease:   nil,
						},
						{
							Offset: 0.5,
							Values: []float64{getScaleOffset(nu, scl, wordEle), scl, hlap},
							Ease:   anim.EaseOut,
						},
						{
							Offset: 1,
							Values: []float64{0, 1, 0},
							Ease:   anim.EaseInOut,
						},
					},
					func(values []float64) {
						ele.Position.SetTranslateX(values[0])
						//ele.Scale = values[1]
						ele.Position.SetScaleX(values[1])
						ele.Position.SetScaleY(values[1])
						if ele.BackgroundBlurText != nil {
							ele.BackgroundBlurText.Alpha = values[2]
						}

					}, func() {
						//ele.BackgroundBlurText.Dispose()
						if ele.BackgroundBlurText != nil {
							ele.BackgroundBlurText.Dispose()
						}
						ele.BackgroundBlurText = nil
					},
				)
				lyrics.AnimateManager.Add(ele.HighlightAnimate)
				if ele.UpAnimate != nil {
					ele.UpAnimate.Cancel()
					ele.UpAnimate = nil
				}
				ele.UpAnimate = anim.NewTween(
					uuid.NewString(),
					duration+time.Duration(700)*time.Millisecond,
					//time.Duration(e.Word.StartTime-line.Line.StartTime)*time.Millisecond,
					wordEle[0].StartTime-lyrics.Position,
					1,
					ele.GetPosition().GetTranslateY(),
					-l.fontsize*0.1,
					anim.EaseOut,
					func(value float64) {
						//ele.TransformY = value
						ele.GetPosition().SetTranslateY(value)
					},
					func() {
					},
				)
				lyrics.AnimateManager.Add(ele.UpAnimate)
			} else {
				if ele.UpAnimate != nil {
					ele.UpAnimate.Cancel()
					ele.UpAnimate = nil
				}
				ele.UpAnimate = anim.NewTween(
					uuid.NewString(),
					ele.EndTime-ele.StartTime+700*time.Millisecond,
					//time.Duration(e.Word.StartTime-line.Line.StartTime)*time.Millisecond,
					ele.StartTime-lyrics.Position,
					1,
					ele.GetPosition().GetTranslateY(),
					-l.fontsize*0.1,
					anim.EaseOut,
					func(value float64) {
						//ele.TransformY = value
						ele.GetPosition().SetTranslateY(value)
					},
					func() {
					},
				)
				lyrics.AnimateManager.Add(ele.UpAnimate)
			}
		}
	}

}

func getScaleOffset(index int, scale float64, doms []*SyllableElement) float64 {
	centerIndex := (len(doms) - 1) / 2
	cumulativeWidth := 0.0
	for i := 0; i < index; i++ {
		cumulativeWidth += doms[i].SyllableImage.GetWidth()
	}
	centerCumulativeWidth := 0.0
	for i := 0; i < centerIndex; i++ {
		centerCumulativeWidth += doms[i].SyllableImage.GetWidth()
	}

	return (cumulativeWidth - centerCumulativeWidth) * (scale - 1)
}
