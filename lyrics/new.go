package lyrics

import (
	"EbitenLyrics/ttml"
	"errors"
	"image/color"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// lineModeMajorityThreshold 定义“多数行”为逐行歌词的比例阈值。
	// 这里使用 60%，目的是降低误判（例如只有个别行恰好是单词/短句的逐字歌词）。
	lineModeMajorityThreshold = 0.6

	// lineModeSingleSyllableMinRunes 定义“单个长音节”的最小长度。
	// 判断逐行时不只看“单个音节”，还要求长度达到一定规模，避免把短词误判成逐行。
	lineModeSingleSyllableMinRunes = 6

	// lineModeMaxTokenRunes 用于限制逐行模式下单个布局单元的最大长度。
	// 目的不是做动画，而是避免极端长英文 token 导致自动换行无法触发。
	lineModeMaxTokenRunes = 24
)

// detectRenderMode 自动识别歌词应使用逐字还是逐行模式。
//
// 判定依据：
// 1) 遍历全部可见歌词行（含背景行），不能只看第一行；
// 2) 如果某行“非空词元只有 1 个且这个词元足够长”，视为逐行特征行；
// 3) 当逐行特征行占比达到阈值（lineModeMajorityThreshold）时，判定整首歌词为逐行模式。
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

// isLineLikeWords 判断一行是否满足“逐行歌词特征”。
// 逐行歌词通常会把整句文本塞进一个词元（一个很长的音节）。
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

	// 满足长度阈值时直接判定为逐行特征。
	if utf8.RuneCountInString(candidate) >= lineModeSingleSyllableMinRunes {
		return true
	}

	// 兜底规则：如果单词元中还包含内部空白（例如 "love me" 被塞在一个词元里），
	// 即使长度略短也按逐行特征处理。
	return strings.IndexFunc(candidate, unicode.IsSpace) >= 0
}

// tokenizeLineWordForLayout 将“整句塞进一个词元”的逐行文本拆成可布局的子单元。
//
// 设计目标：
// 1) 保留空格（用于视觉换行宽度计算）；
// 2) 英文按单词分组，中文按单字分组，尽量避免在可读单词中间断行；
// 3) 对超长英文 token 再次切分，防止一个 token 过长导致换行失效。
func tokenizeLineWordForLayout(word string) []string {
	if word == "" {
		return nil
	}

	var tokens []string
	var buf strings.Builder

	flushASCIIWord := func() {
		if buf.Len() == 0 {
			return
		}
		token := buf.String()
		buf.Reset()
		tokens = append(tokens, splitTokenByRuneLimit(token, lineModeMaxTokenRunes)...)
	}

	for _, r := range word {
		switch {
		// 将任意空白统一成单个空格 token，保持布局宽度语义。
		case unicode.IsSpace(r):
			flushASCIIWord()
			tokens = append(tokens, " ")
		// ASCII 字母/数字连续拼成一个 token（英文单词/数字串）。
		case r <= 0x7f && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			buf.WriteRune(r)
		// 其他字符（CJK、标点等）按单 rune 切分，避免错误合并。
		default:
			flushASCIIWord()
			tokens = append(tokens, string(r))
		}
	}
	flushASCIIWord()

	return tokens
}

// splitTokenByRuneLimit 将超长 token 按 rune 长度切分。
// 该函数主要用于英文超长单词的兜底，避免布局时“永不换行”。
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

// createLineModeSyllables 构建逐行模式用的音节序列。
//
// 输入歌词即使只有一个长音节，也会在此被切成多个“布局单元”：
//   - 这样自动换行仍然可以生效；
//   - 动画层会按 RenderModeLine 把这些单元作为“同一整行”统一高亮，
//     不会出现逐字扫光。
func createLineModeSyllables(ts []ttml.LyricWord, line *Line, face text.Face, fd float64, alpha uint8) ([]*LineSyllable, error) {
	var syllables []*LineSyllable

	for _, word := range ts {
		parts := tokenizeLineWordForLayout(word.Word)
		if len(parts) == 0 {
			continue
		}

		start := time.Duration(word.StartTime) * time.Millisecond
		end := time.Duration(word.EndTime) * time.Millisecond
		if end <= start {
			// 某些解析路径下词元级时间可能缺失，逐行模式回退到整行时间。
			start = line.StartTime
			end = line.EndTime
		}

		for _, part := range parts {
			syllable, err := NewSyllable(
				part,
				start,
				end,
				face,
				fd,
				color.RGBA{255, 255, 255, alpha},
				color.RGBA{255, 255, 255, 60},
				false, // 逐行模式禁用逐字拆分。
			)
			if err != nil {
				return nil, err
			}
			// 逐行模式下默认不高亮，进入唱段时再做整行淡入。
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

	// 防御：如果整行为空，至少放一个空白占位，避免后续布局/渲染空切片边界问题。
	if len(syllables) == 0 {
		placeholder, err := NewSyllable(
			" ",
			line.StartTime,
			line.EndTime,
			face,
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

func New(ttmllines []ttml.LyricLine, screenW float64, f *text.GoTextFaceSource, fallbacks []*text.GoTextFaceSource, fs, fd float64) (*Lyrics, error) {
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
			f,
			fallbacks,
			fs,
		)
		l.RenderMode = lyrics.RenderMode
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
				fallbacks,
				fs/1.5,
			)
			lbg.RenderMode = lyrics.RenderMode
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
	if face == nil {
		return errors.New("line face is nil")
	}

	var syllables []*LineSyllable
	ap := uint8(255)
	if line.IsBackground {
		ap = 130
	}

	// 逐行模式：先把“整句单音节”拆成布局单元，再统一做整行动画。
	if line.RenderMode == RenderModeLine {
		lineModeSyllables, err := createLineModeSyllables(ts, line, face, fd, ap)
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
				face,
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
