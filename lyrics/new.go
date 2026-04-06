package lyrics

import (
	ft "EbitenLyrics/font"
	"EbitenLyrics/ttml"
	"errors"
	"image/color"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	lineModeMajorityThreshold      = 0.6
	lineModeSingleSyllableMinRunes = 6
	lineModeMaxTokenRunes          = 24
)

func detectRenderMode(ttmllines []ttml.LyricLine) LyricRenderMode {
	total := 0
	lineLike := 0

	observe := func(words []ttml.LyricWord) {
		if len(words) == 0 {
			return
		}
		total++
		if isLineLikeWords(words) {
			lineLike++
		}
	}

	for _, line := range ttmllines {
		observe(line.Words)
		for _, bgLine := range line.BGs {
			observe(bgLine.Words)
		}
	}

	if total == 0 {
		return RenderModeSyllable
	}
	if float64(lineLike)/float64(total) >= lineModeMajorityThreshold {
		return RenderModeLine
	}
	return RenderModeSyllable
}

func isLineLikeWords(words []ttml.LyricWord) bool {
	nonEmptyCount := 0
	candidate := ""

	for _, word := range words {
		text := strings.TrimSpace(word.Word)
		if text == "" {
			continue
		}
		nonEmptyCount++
		if nonEmptyCount > 1 {
			return false
		}
		candidate = text
	}

	if nonEmptyCount != 1 {
		return false
	}

	if utf8.RuneCountInString(candidate) >= lineModeSingleSyllableMinRunes {
		return true
	}

	return strings.IndexFunc(candidate, unicode.IsSpace) >= 0
}

func tokenizeLineWordForLayout(word string) []string {
	if word == "" {
		return nil
	}

	var tokens []string
	var asciiBuf strings.Builder
	var cjkBuf strings.Builder

	flushASCIIWord := func() {
		if asciiBuf.Len() == 0 {
			return
		}
		token := asciiBuf.String()
		asciiBuf.Reset()
		tokens = append(tokens, splitTokenByRuneLimit(token, lineModeMaxTokenRunes)...)
	}
	flushCJKRun := func() {
		if cjkBuf.Len() == 0 {
			return
		}
		token := cjkBuf.String()
		cjkBuf.Reset()
		tokens = append(tokens, splitCJKRunForLayout(token)...)
	}
	flushAll := func() {
		flushASCIIWord()
		flushCJKRun()
	}

	for _, r := range word {
		switch {
		case unicode.IsSpace(r):
			flushAll()
			tokens = append(tokens, " ")
		case r <= 0x7f && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			flushCJKRun()
			asciiBuf.WriteRune(r)
		case r <= 0x7f && (r == '\'' || r == '-') && asciiBuf.Len() > 0:
			flushCJKRun()
			asciiBuf.WriteRune(r)
		case isCJKLayoutRune(r):
			flushASCIIWord()
			cjkBuf.WriteRune(r)
		default:
			flushAll()
			tokens = append(tokens, string(r))
		}
	}
	flushAll()

	return tokens
}

func isCJKLayoutRune(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

func splitCJKRunForLayout(run string) []string {
	if run == "" {
		return nil
	}

	runes := []rune(run)
	out := make([]string, 0, (len(runes)+1)/2)
	for i := 0; i < len(runes); {
		remaining := len(runes) - i
		size := chooseCJKChunkSize(remaining)
		if size > remaining {
			size = remaining
		}
		out = append(out, string(runes[i:i+size]))
		i += size
	}
	return out
}

func chooseCJKChunkSize(remaining int) int {
	switch {
	case remaining <= 4:
		return remaining
	case remaining%3 == 0:
		return 3
	default:
		return 2
	}
}

func splitTokenByRuneLimit(token string, limit int) []string {
	if token == "" || limit <= 0 {
		return nil
	}
	if utf8.RuneCountInString(token) <= limit {
		return []string{token}
	}

	runes := []rune(token)
	out := make([]string, 0, (len(runes)+limit-1)/limit)
	for i := 0; i < len(runes); i += limit {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

func createLineModeSyllables(ts []ttml.LyricWord, line *Line, fd float64, alpha uint8) ([]*LineSyllable, error) {
	var syllables []*LineSyllable

	for _, word := range ts {
		parts := tokenizeLineWordForLayout(word.Word)
		if len(parts) == 0 {
			continue
		}

		start := time.Duration(word.StartTime) * time.Millisecond
		end := time.Duration(word.EndTime) * time.Millisecond
		if end <= start {
			start = line.StartTime
			end = line.EndTime
		}

		for _, part := range parts {
			syllable, err := NewSyllable(
				part,
				start,
				end,
				line.FontManager,
				line.FontRequest,
				line.fontsize,
				fd,
				color.RGBA{255, 255, 255, alpha},
				color.RGBA{255, 255, 255, 60},
				false,
			)
			if err != nil {
				return nil, err
			}
			for _, element := range syllable.Elements {
				if element == nil {
					continue
				}
				element.Alpha = 0
				element.NowOffset = 0
			}
			syllables = append(syllables, syllable)
		}
	}

	if len(syllables) == 0 {
		placeholder, err := NewSyllable(
			" ",
			line.StartTime,
			line.EndTime,
			line.FontManager,
			line.FontRequest,
			line.fontsize,
			fd,
			color.RGBA{255, 255, 255, alpha},
			color.RGBA{255, 255, 255, 60},
			false,
		)
		if err != nil {
			return nil, err
		}
		for _, element := range placeholder.Elements {
			if element == nil {
				continue
			}
			element.Alpha = 0
			element.NowOffset = 0
		}
		syllables = append(syllables, placeholder)
	}

	return syllables, nil
}

func New(ttmllines []ttml.LyricLine, screenW float64, fontManager *ft.FontManager, req ft.FontRequest, fs, fd float64) (*Lyrics, error) {
	var lyrics Lyrics
	lyrics.FD = fd
	lyrics.anchorIndex = -1
	lyrics.RenderMode = detectRenderMode(ttmllines)
	for _, line := range ttmllines {
		l := NewLine(
			time.Duration(line.StartTime)*time.Millisecond,
			time.Duration(line.EndTime)*time.Millisecond,
			line.IsDuet,
			line.IsBG,
			line.TranslatedLyric,
			fontManager,
			req,
			fs,
		)
		l.RenderMode = lyrics.RenderMode
		l.Position.SetW(screenW * 0.9)
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
				fontManager,
				req,
				fs/1.5,
			)
			lbg.RenderMode = lyrics.RenderMode
			lbg.Position.SetW(screenW * 0.9)
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
	if face == nil {
		return errors.New("line face is nil")
	}

	var syllables []*LineSyllable
	ap := uint8(255)
	if line.IsBackground {
		ap = 130
	}

	if line.RenderMode == RenderModeLine {
		lineModeSyllables, err := createLineModeSyllables(ts, line, fd, ap)
		if err != nil {
			return err
		}
		line.SetSyllables(lineModeSyllables)
		return nil
	}

	wordGroups := SplitBySpaceTTML(ts, true)
	for _, group := range wordGroups {
		duration := time.Duration(0)
		for _, idx := range group {
			duration += time.Duration(ts[idx].EndTime-ts[idx].StartTime) * time.Millisecond
		}
		needSplitCharsByDuration := duration >= 800*time.Millisecond

		for _, idx := range group {
			w := ts[idx]
			syllable, err := NewSyllable(
				w.Word,
				time.Duration(w.StartTime)*time.Millisecond,
				time.Duration(w.EndTime)*time.Millisecond,
				line.FontManager,
				line.FontRequest,
				line.fontsize,
				fd,
				color.RGBA{255, 255, 255, ap},
				color.RGBA{255, 255, 255, 60},
				needSplitCharsByDuration,
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
