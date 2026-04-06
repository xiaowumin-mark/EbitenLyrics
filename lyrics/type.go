package lyrics

// 文件说明：歌词核心数据结构定义。
// 主要职责：声明行、音节、元素、状态和整体歌词对象的字段布局。

import (
	"EbitenLyrics/anim"
	ft "EbitenLyrics/font"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// LineStatus 表示歌词行在时间轴中的状态。
type LineStatus int

const (
	LineStatusHidden LineStatus = iota
	LineStatusPreviewStatic
	LineStatusPreviewScrolling
	LineStatusActiveEnter
	LineStatusActivePlaying
	LineStatusActiveExit
)

func (s LineStatus) UsesPreviewBitmap() bool {
	switch s {
	case LineStatusPreviewStatic, LineStatusPreviewScrolling:
		return true
	default:
		return false
	}
}

func (s LineStatus) RequiresRealtimeRender() bool {
	switch s {
	case LineStatusActiveEnter, LineStatusActivePlaying, LineStatusActiveExit:
		return true
	default:
		return false
	}
}

func (s LineStatus) CanStartExit() bool {
	switch s {
	case LineStatusActiveEnter, LineStatusActivePlaying:
		return true
	default:
		return false
	}
}

// LyricRenderMode 表示当前歌词渲染模式。
// RenderModeSyllable: 常规逐字/逐词卡拉 OK。
// RenderModeLine: 整行高亮模式（逐行歌词）。
type LyricRenderMode int

const (
	RenderModeSyllable LyricRenderMode = iota
	RenderModeLine
)

type LineSyllable struct {
	StartTime time.Duration
	EndTime   time.Duration
	Syllable  string

	Elements []*SyllableElement

	Alpha float64
}

type SyllableElement struct {
	Text               string
	Position           Position
	SyllableImage      *SyllableImage
	BackgroundBlurText *TextShadow
	NowOffset          float64
	Alpha              float64
	StartTime          time.Duration
	EndTime            time.Duration

	// SyllableIndex / OuterSyllableElementsIndex 用索引避免循环引用。
	SyllableIndex              int
	OuterSyllableElementsIndex int

	Animate          *anim.KeyframeAnimation
	HighlightAnimate *anim.KeyframeAnimation
	UpAnimate        *anim.Tween
}

type Line struct {
	StartTime             time.Duration
	EndTime               time.Duration
	Text                  string
	Syllables             []*LineSyllable
	OuterSyllableElements []*SyllableElement
	TranslatedText        string

	BackgroundLines    []*Line
	Participle         [][]int
	SmartTranslateWrap bool

	// RenderMode 由加载阶段统一判定后写入，布局和动画直接读取该值。
	RenderMode LyricRenderMode

	lineHeight float64
	Padding    float64

	IsBackground bool
	IsDuet       bool

	Image                            *ebiten.Image
	TranslateImage                   *ebiten.Image
	TranslateImageW, TranslateImageH float64
	Position                         Position

	FontManager *ft.FontManager
	FontRequest ft.FontRequest
	fontsize    float64

	isShow bool

	Status LineStatus

	imageDirty          bool
	StatusSettleAnimate *anim.Tween

	ScrollAnimate        *anim.Tween
	AlphaAnimate         *anim.KeyframeAnimation
	GradientColorAnimate *anim.Tween
	ScaleAnimate         *anim.Tween
}

type LyricMeta struct {
	Title        []string
	Artist       []string
	Album        []string
	NcmMusicId   []string
	QQMusicId    []string
	SpotifyId    []string
	AppleMusicId []string
	ISRC         []string
	GitbugId     []string
	GithubUser   string
}

type Lyrics struct {
	Meta  LyricMeta
	Lines []*Line

	// RenderMode 表示整首歌词采用的渲染模式。
	// 通过“多数行判定”得到，避免因首行特例导致误判。
	RenderMode LyricRenderMode

	Position time.Duration

	nowLyrics   []int
	renderIndex []int
	anchorIndex int

	Margin        float64
	HighlightTime time.Duration
	FD            float64

	AnimateManager *anim.Manager
}

func (l *Lyrics) GetNowLyrics() []int {
	return l.nowLyrics
}
