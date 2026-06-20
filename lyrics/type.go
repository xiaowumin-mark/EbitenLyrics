package lyrics

// 文件说明：歌词核心数据结构定义。
// 主要职责：声明行、音节、元素、状态和整体歌词对象的字段布局。

import (
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"

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

type TimelineState struct {
	CurrentTime           time.Duration
	LastCurrentTime       time.Duration
	HotLines              map[int]struct{}
	BufferedLines         map[int]struct{}
	ScrollToIndex         int
	IsSeeking             bool
	IsPlaying             bool
	InitialLayoutFinished bool
}

type LinePresentation struct {
	IsActive     bool
	TargetAlpha  float64
	TargetScale  float64
	BlurLevel    float64
	RenderMode   LyricRenderMode
	ReserveSpace bool
}

type Interlude struct {
	StartTime       time.Duration
	EndTime         time.Duration
	AnchorLineIndex int
	IsNextDuet      bool
}

type InterludeDots struct {
	Position    Position
	Active      bool
	Paused      bool
	CurrentTime time.Duration
	StartTime   time.Duration
	EndTime     time.Duration
	DotSize     float64
	Gap         float64
	PaddingX    float64
	PaddingY    float64
	Margin      float64
	GlobalScale float64
	GlobalAlpha float64
	IsDuet      bool
	DotAlphas   [3]float64
}

type BottomLine struct {
	Position        Position
	PosXSpring      *anim.Spring
	PosYSpring      *anim.Spring
	BlurLevel       float64
	Focused         bool
	Active          bool
	Text            string
	LineSize        [2]float64
	FontManager     *ft.FontManager
	FontRequest     ft.FontRequest
	FontSize        float64
	ContentFontSize float64
	PaddingLeft     float64
	PaddingTop      float64
	Image           *ebiten.Image
	BlurImage       *ebiten.Image
	BlurCacheSource *ebiten.Image
	BlurCacheKey    int
	ImageDirty      bool
}

type LayoutAlignAnchor int

const (
	LayoutAlignAnchorTop LayoutAlignAnchor = iota
	LayoutAlignAnchorCenter
	LayoutAlignAnchorBottom
)

type LayoutState struct {
	TargetAlignIndex   int
	LastInterludeState bool
	AlignAnchor        LayoutAlignAnchor
	AlignPosition      float64
	OverscanPx         float64
	HidePassedLines    bool
	EnableBlur         bool
	BlurStrength       float64
	IsUserScrolling    bool
	ScrollOffset       float64
	ScrollMinOffset    float64
	ScrollMaxOffset    float64
	AllowScroll        bool
	IsScrolled         bool
}

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
	HasDuetInSong      bool

	// RenderMode 由加载阶段统一判定后写入，布局和动画直接读取该值。
	RenderMode LyricRenderMode

	lineHeight   float64
	Padding      float64
	PaddingLeft  float64
	PaddingRight float64

	IsBackground bool
	IsDuet       bool

	Image                            *ebiten.Image
	BlurImage                        *ebiten.Image
	BlurCacheSource                  *ebiten.Image
	BlurCacheKey                     int
	TranslateImage                   *ebiten.Image
	TranslateImageW, TranslateImageH float64
	Position                         Position
	Presentation                     LinePresentation
	BlurLevel                        float64

	FontManager *ft.FontManager
	FontRequest ft.FontRequest
	fontsize    float64

	isShow         bool
	lastVisibleAt  time.Duration
	lastRenderRank int

	Status LineStatus

	imageDirty          bool
	StatusSettleAnimate *anim.Tween

	ScrollAnimate        *anim.Tween
	PosYSpring           *anim.Spring
	ScaleSpring          *anim.Spring
	AlphaAnimate         *anim.KeyframeAnimation
	GradientColorAnimate *anim.Tween
	ScaleAnimate         *anim.Tween

	OffsetMetrics lineOffsetMetrics
}

type lineOffsetMetrics struct {
	widths       []float64
	prefix       []float64
	valid        bool
	elementCount int
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
	Meta        LyricMeta
	Lines       []*Line
	FontManager *ft.FontManager
	FontRequest ft.FontRequest

	// RenderMode 表示整首歌词采用的渲染模式。
	// 通过“多数行判定”得到，避免因首行特例导致误判。
	RenderMode LyricRenderMode

	Position time.Duration
	Timeline TimelineState
	Layout   LayoutState
	Dots     InterludeDots
	Bottom   BottomLine

	nowLyrics                  []int
	renderIndex                []int
	anchorIndex                int
	finalLayoutPending         bool
	hiddenResourcePruneElapsed time.Duration

	Margin        float64
	HighlightTime time.Duration
	FD            float64
	Width         float64

	AnimateManager *anim.Manager
}

func (l *Lyrics) GetNowLyrics() []int {
	return l.nowLyrics
}
