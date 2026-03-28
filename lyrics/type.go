package lyrics

import (
	"EbitenLyrics/anim"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// LineStatus 表示歌词行在时间轴中的状态。
type LineStatus int

const (
	Normal LineStatus = iota
	Hot
	Buffered
)

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

	BackgroundLines []*Line
	Participle      [][]int

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

	Font          *text.GoTextFaceSource
	FallbackFonts []*text.GoTextFaceSource
	fontsize      float64
	Face          text.Face

	isShow bool

	Status LineStatus

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
