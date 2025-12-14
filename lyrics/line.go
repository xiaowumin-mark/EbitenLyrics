package lyrics

import (
	"image/color"
	"log"
	"strings"
	"time"

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
		fontsize:        40,
		isShow:          true,
	}
}

func (l *Line) Layout() {
	w := l.Position.GetW()
	// 分词
	l.Participle = SplitBySpace(l, true)
	log.Println("分词完成", l.Participle)

	// 计算位置
	var ls [][]*LineSyllable
	// 遍历分词结果
	for _, p := range l.Participle {
		var syllables []*LineSyllable
		for _, i := range p {
			syllables = append(syllables, l.Syllables[i])
		}
		ls = append(ls, syllables)
	}
	log.Println("分词结果", ls)
	lineXTL := l.Padding
	al := text.AlignStart
	if l.IsDuet {
		al = text.AlignEnd
		lineXTL = -l.Padding
	}

	var poss []Position
	var height float64
	poss, height = AutoLayoutSyllable(
		ls,
		*l.GetFace(true),
		w-l.Padding*2,
		l.lineHeight,
		1,
		al,
	)
	height += l.Padding * 2
	log.Println(height, w-l.Padding*2)
	for _, pos := range poss {
		log.Println(pos.GetX(), pos.GetH())
	}

	/*for i, pos := range poss {
		log.Println(l.Syllables[i].Syllable)
		//log.Println(pos.GetW(), pos.GetH())
		log.Println(pos.GetX(), pos.GetW())
		pos.SetX(pos.GetX() + lineXTL)
		pos.SetY(pos.GetY() + l.Padding)
		lastX := pos.GetX()
		for _, element := range l.Syllables[i].Elements {
			element.GetPosition().SetX(lastX)
			element.GetPosition().SetY(pos.GetY())

			lastX += element.SyllableImage.GetWidth()
		}

	}*/

	idx := 0
	for _, word := range ls { // 按 ls 的组
		for range word { // 遍历音节
			pos := poss[idx]

			syll := l.Syllables[idx]

			pos.SetX(pos.GetX() + lineXTL)
			pos.SetY(pos.GetY() + l.Padding)
			lastX := pos.GetX()
			for _, element := range syll.Elements {
				element.GetPosition().SetX(lastX)
				element.GetPosition().SetY(pos.GetY())

				lastX += element.SyllableImage.GetWidth()
			}
			log.Println(syll.Syllable)
			log.Println(pos.GetX(), pos.GetW(), pos.GetH())
			idx++
		}
	}

	tsh := l.GetTranslateImageHeight()
	l.Position.SetH(height + tsh)

}

// 生成翻译图片
func (l *Line) GenerateTSImage() {
	if l.TranslatedText == "" {
		return
	}
	al := text.AlignStart
	if l.IsDuet {
		al = text.AlignCenter
	}
	poss, h := AutoLayout(
		l.TranslatedText,
		&text.GoTextFace{
			Source: l.Font,
			Size:   l.fontsize / 2,
		},
		l.Position.GetW()-l.Padding*2,
		l.lineHeight,
		1,
		al,
	)
	l.TranslateImage = ebiten.NewImage(int(l.Position.GetW()-l.Padding*2), int(h))
	for i, pos := range poss {
		log.Println(i, pos.Text, pos.X)
		op := &text.DrawOptions{}
		op.GeoM.Translate(
			pos.X,
			pos.Y,
		)
		op.ColorScale.ScaleWithColor(color.White)
		op.ColorScale.ScaleAlpha(0.4)
		text.Draw(l.TranslateImage, pos.Text, &text.GoTextFace{
			Source: l.Font,
			Size:   l.fontsize / 2,
		}, op)
	}
}

func (l *Line) GetTranslateImageHeight() float64 {
	if l.TranslateImage == nil {
		/*_, h := AutoLayout(
			l.TranslatedText,
			&text.GoTextFace{
				Source: l.Font,
				Size:   l.fontsize / 2,
			},
			l.Position.GetW()-l.Padding*2,
			l.lineHeight,
			1,
			text.AlignStart,
		)
		return h*/
		l.GenerateTSImage()

	}
	return float64(l.TranslateImage.Bounds().Dy())
}
func (l *Line) Draw(screen *ebiten.Image) {
	if !l.isShow {
		return
	}
	if l.Image == nil {
		l.Image = ebiten.NewImage(int(l.Position.GetW()), int(l.Position.GetH()))
	}
	l.Image.Clear()
	for _, syllable := range l.Syllables {
		syllable.Draw(l.Image)
	}

	// 画翻译
	if l.TranslateImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(l.Padding, l.Position.GetH()-l.GetTranslateImageHeight()-l.Padding)
		screen.DrawImage(l.TranslateImage, op)
	}
	screen.DrawImage(l.Image, nil)
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
func (l *Line) SetFont(font *text.GoTextFaceSource) {
	l.Font = font
	for _, syllable := range l.Syllables {
		syllable.SetFont(*l.GetFace(true))
	}
	l.GenerateTSImage()
	l.Layout()
}

func (l *Line) SetFontSize(fontsize float64) {
	l.fontsize = fontsize
	for _, syllable := range l.Syllables {
		syllable.SetFont(*l.GetFace(true))
	}
	l.GenerateTSImage()
	l.Layout()
}

func (l *Line) GetFace(isc bool) *text.Face {
	var face text.Face
	if isc {
		face = &text.GoTextFace{
			Source: l.Font,
			Size:   l.fontsize,
		}
		return &face
	} else {
		if l.Font != nil {
			face = &text.GoTextFace{
				Source: l.Font,
				Size:   l.fontsize,
			}
			return &face
		} else {
			return l.Face
		}
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
func (l *Line) Dispose() {
	l.isShow = false
	for _, syllable := range l.Syllables {
		syllable.Dispose()
	}
	for _, bgline := range l.BackgroundLines {
		bgline.Dispose()
	}
	if l.TranslateImage != nil {
		l.TranslateImage.Deallocate()
	}

}

func (l *Line) Render() {
	for _, syllable := range l.Syllables {
		syllable.Redraw()
	}
	for _, bgline := range l.BackgroundLines {
		bgline.Render()
	}
	l.GenerateTSImage()
	l.isShow = true
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

func SplitBySpace(line *Line, includeSpaces bool) [][]int {
	var result [][]int
	var currentWordIndices []int

	for index, element := range line.Syllables {
		word := element.Syllable

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

// 宽度变化并且重新渲染
func (l *Line) Resize(width float64) {
	l.Dispose()
	l.Position.SetW(width)
	l.Layout()
	l.Render()
}
