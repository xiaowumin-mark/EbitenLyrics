package pages

// 文件说明：示例或调试用的游戏场景。
// 主要职责：加载测试歌词并驱动基础渲染流程。

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/router"
	"EbitenLyrics/ttml"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Game struct {
	router.BaseScene
	AnimateManager *anim.Manager
	Font           *text.GoTextFaceSource
	FontFallbacks  []*text.GoTextFaceSource
	sy             *lyrics.LineSyllable
	fontsize       float64

	line *lyrics.Line

	texts   []lyrics.Position
	textimg *ebiten.Image

	ss []*lyrics.LineSyllable

	lyric *lyrics.Lyrics

	tim time.Duration
}

func (g *Game) loadDemoTTML() (ttml.TTMLLyric, error) {
	candidates := []string{}
	if fromEnv := os.Getenv("EBITENLYRICS_DEMO_TTML"); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	candidates = append(candidates,
		"test-data/Opalite.ttml",
		"test-data/Bejeweled.ttml",
	)

	for _, filePath := range candidates {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		parsed, err := ttml.ParseTTML(string(data))
		if err != nil {
			continue
		}
		return parsed, nil
	}
	return ttml.TTMLLyric{}, fmt.Errorf("no demo ttml found, tried: %v", candidates)
}

func (g *Game) OnEnter(params map[string]any) {
	log.Println("Game OnEnter", params)
	g.fontsize = 48.0
	g.tim = 0

	tt, err := g.loadDemoTTML()
	if err != nil {
		log.Printf("load demo ttml failed: %v", err)
		g.lyric = nil
		return
	}

	w, _ := ebiten.WindowSize()
	l, err := lyrics.New(tt.LyricLines, float64(w), g.Font, g.FontFallbacks, g.fontsize, 1)
	if err != nil {
		log.Printf("init lyric failed: %v", err)
		g.lyric = nil
		return
	}
	g.lyric = l
	g.lyric.AnimateManager = g.AnimateManager
	g.lyric.HighlightTime = time.Millisecond * 1000
}

func (g *Game) OnLeave() {}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		router.Go("home", nil)
	}
	if g.lyric == nil {
		return nil
	}

	g.tim += time.Second / time.Duration(ebiten.TPS())
	g.lyric.Update(g.tim)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.lyric != nil {
		g.lyric.Draw(screen)
	} else {
		ebitenutil.DebugPrint(screen, "No demo TTML found (set EBITENLYRICS_DEMO_TTML)")
	}
	msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) OnResize(w, he int, isFirst bool) {
	log.Println("Game OnResize", w, he, isFirst)
	if !isFirst {
	}
}
