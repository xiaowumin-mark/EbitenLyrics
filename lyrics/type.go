package lyrics

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type LineSyllable struct {
	StartTime time.Duration // 开始时间
	EndTime   time.Duration // 结束时间
	Syllable  string        // 音节内容

	Elements []*SyllableElement // 音节元素列表

	Alpha float64 // 当前透明度（用于渐变计算）
	//Position Position // 音节位置和变换信息
}

type SyllableElement struct {
	Text          string
	Position      Position
	SyllableImage *SyllableImage // 音节图像数据结构
	NowOffset     float64        // 当前偏移位置（用于渐变计算）
	//Offset        float64        // 偏移位置
	Alpha float64 // 当前透明度（用于渐变计算）
}

type Line struct {
	StartTime      time.Duration   // 歌词行开始时间
	EndTime        time.Duration   // 歌词行结束时间
	Text           string          // 歌词行内容
	Syllables      []*LineSyllable // 音节列表
	TranslatedText string          // 翻译内容

	BackgroundLines []*Line // 背景歌词行

	Participle [][]int // 歌词行 participle

	lineHeight float64

	Padding float64

	// 行标记
	IsBackground bool // 是否为背景歌词行
	IsDuet       bool // 是否为对唱行

	Image          *ebiten.Image // 渲染该行歌词的图像
	TranslateImage *ebiten.Image // 渲染该行翻译歌词的图像
	Position       Position      // 歌词行位置和变换信息

	Font     *text.GoTextFaceSource
	fontsize float64
	Face     *text.Face

	isShow bool
}

type LyricMeta struct {
	Title        []string // 歌曲标题
	Artist       []string // 歌手信息
	Album        []string // 专辑信息
	NcmMusicId   []string // 网易云音乐ID
	QQMusicId    []string // QQ音乐ID
	SpotifyId    []string // Spotify ID
	AppleMusicId []string // Apple Music ID
	ISRC         []string // 国际标准录音编码
	GitbugId     []string // GitHub ID
	GithubUser   string   // GitHub 用户名
}

type Lyrics struct {
	Meta  LyricMeta // 歌词元数据
	Lines []Line    // 歌词行列表
}
