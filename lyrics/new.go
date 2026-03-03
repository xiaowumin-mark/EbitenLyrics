package lyrics

import (
	"EbitenLyrics/ttml"
	"errors"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func New(ttmllines []ttml.LyricLine, screenW float64, f *text.GoTextFaceSource, fs, fd float64) (*Lyrics, error) {
	var lyrics Lyrics
	lyrics.FD = fd
	lyrics.anchorIndex = -1
	for _, line := range ttmllines {
		l := NewLine(
			time.Duration(line.StartTime)*time.Millisecond,
			time.Duration(line.EndTime)*time.Millisecond,
			line.IsDuet,
			line.IsBG,
			line.TranslatedLyric,
			f,
			fs,
		)
		l.Position.SetW(screenW * 0.8)
		l.SetPadding(20)
		if line.IsDuet {
			l.Position.SetX(screenW - l.Position.GetW())
		}
		if err := CreateSyllable(line.Words, l, fd); err != nil {
			return nil, err
		}
		l.Layout()

		for _, bgline := range line.BGs {
			lbg := NewLine(
				time.Duration(bgline.StartTime)*time.Millisecond,
				time.Duration(bgline.EndTime)*time.Millisecond,
				bgline.IsDuet,
				bgline.IsBG,
				bgline.TranslatedLyric,
				f,
				fs/1.5,
			)
			lbg.Position.SetW(screenW * 0.8)
			lbg.SetPadding(20)
			if bgline.IsDuet {
				lbg.Position.SetX(screenW - lbg.Position.GetW())
			}
			if err := CreateSyllable(bgline.Words, lbg, fd); err != nil {
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

func CreateSyllable(ts []ttml.LyricWord, line *Line, fd float64) error {
	if line == nil {
		return errors.New("line is nil")
	}
	face := line.GetFace(false)
	if face == nil || *face == nil {
		return errors.New("line face is nil")
	}

	var syllables []*LineSyllable
	ap := uint8(255)
	if line.IsBackground {
		ap = 130
	}
	wordGroups := SplitBySpaceTTML(ts, true)
	for _, group := range wordGroups {
		duration := time.Duration(0)
		for _, idx := range group {
			duration += time.Duration(ts[idx].EndTime-ts[idx].StartTime) * time.Millisecond
		}
		needSplitChars := duration >= 800*time.Millisecond

		for _, idx := range group {
			w := ts[idx]
			syllable, err := NewSyllable(
				w.Word,
				time.Duration(w.StartTime)*time.Millisecond,
				time.Duration(w.EndTime)*time.Millisecond,
				*face,
				fd,
				color.RGBA{255, 255, 255, ap},
				color.RGBA{255, 255, 255, 60},
				needSplitChars,
			)
			if err != nil {
				return err
			}
			syllables = append(syllables, syllable)
		}
	}

	line.SetSyllables(syllables)
	return nil
}
