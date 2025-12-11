package lyrics

import (
	"EbitenLyrics/ttml"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func NewLine(st, et time.Duration, isduet, isbg bool, ts string) *Line {
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
	}
}

func (l *Line) AddSyllable(syllable *LineSyllable) {
	l.Syllables = append(l.Syllables, syllable)
	l.Text += syllable.Syllable
}
func (l *Line) GetSyllables() []*LineSyllable {
	return l.Syllables
}
func (l *Line) SetSyllables(syllables []*LineSyllable) {
	for _, syllable := range syllables {
		syllable.Dispose()
	}
	l.Syllables = syllables
}

// 设置字体
func (l *Line) SetFont(font text.Face) {
	l.Font = &font
	for _, syllable := range l.Syllables {
		syllable.SyllableImage.SetFont(*l.Font)
	}
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
func (l *Line) Dispose() {
	for _, syllable := range l.Syllables {
		syllable.Dispose()
	}
	l.Syllables = nil
	for _, bgline := range l.BackgroundLines {
		bgline.Dispose()
	}
	l.BackgroundLines = nil
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
func (l *Line) GetPosition() Position {
	return l.Position
}
func (l *Line) SetPosition(pos Position) {
	l.Position = pos
}

// isAsciiString 判断字符串是否仅包含 ASCII 字符
func isAsciiString(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}
func SplitBySpace(line *ttml.LyricLine, includeSpaces bool) [][]int {
	var result [][]int
	var currentWordIndices []int

	for index, element := range line.Words {
		word := element.Word

		// 1. 优先处理纯空格的情况 (即整个 word 都是空白)
		if strings.TrimSpace(word) == "" {
			// 遇到纯空格，意味着前面的分组必须结束
			if currentWordIndices != nil {
				result = append(result, currentWordIndices)
				currentWordIndices = nil
			}

			// 如果需要保留空格作为独立分组
			if includeSpaces {
				result = append(result, []int{index})
			}
			continue
		}

		// 2. 处理非空字符
		if isSingleChineseChar(word) {
			// 情况 A: 单个中文字符
			// 中文通常字字断开，先结束之前的分组
			if currentWordIndices != nil {
				result = append(result, currentWordIndices)
				currentWordIndices = nil
			}
			// 中文自己独立成组
			result = append(result, []int{index})

		} else {
			// 情况 B: 英文或非单字中文的连续部分 (如 "Hello" 或 "Hello ")
			// 先将当前词加入分组
			currentWordIndices = append(currentWordIndices, index)

			// --- 新增逻辑开始 ---
			// 检查单词末尾是否有空格
			// 如果单词是 "Hello "，意味着它是这组词的结尾
			if strings.HasSuffix(word, " ") {
				// 将当前累积的词组（包含当前的这个词）存入结果
				result = append(result, currentWordIndices)
				//以此清空，为下一组做准备
				currentWordIndices = nil
			}
			// --- 新增逻辑结束 ---
		}
	}

	// 3. 循环结束后的收尾工作
	// 如果最后还有残留的词没有被归档（例如最后一个词没有空格结尾）
	if currentWordIndices != nil {
		result = append(result, currentWordIndices)
	}

	return result
}

// 判断是否是中文字符（基本汉字范围）
func isChineseChar(c rune) bool {
	return c >= '\u4e00' && c <= '\u9fff'
}

// 判断字符串是否是单个中文字符
func isSingleChineseChar(s string) bool {
	if len([]rune(s)) != 1 {
		return false
	}
	r := []rune(s)[0]
	return isChineseChar(r)
}
