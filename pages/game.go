package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/router"
	"EbitenLyrics/ttml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Game struct {
	router.BaseScene // 嵌入并获得默认实现
	AnimateManager   *anim.Manager
	Font             *text.GoTextFaceSource
	sy               *lyrics.LineSyllable
	fontsize         float64

	line *lyrics.Line

	texts   []lyrics.Position
	textimg *ebiten.Image

	ss []*lyrics.LineSyllable

	lyric *lyrics.Lyrics

	tim time.Duration
}

func (g *Game) OnEnter(params map[string]any) {
	log.Println("Game OnEnter", params)
	g.fontsize = 48.0

	// 读取Bejeweled.ttml
	file, err := os.Open("E:\\projects\\visual-lyric\\music\\ME!.ttml")
	if err != nil {
		log.Fatal(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	tt, err := ttml.ParseTTML(string(data))
	if err != nil {
		log.Fatal(err)
	}
	w, _ := ebiten.WindowSize()
	l, err := lyrics.New(tt.LyricLines, float64(w), g.Font, g.fontsize)
	if err != nil {
		log.Fatal(err)
	}
	g.lyric = l
	g.lyric.AnimateManager = g.AnimateManager
	g.lyric.HighlightTime = time.Millisecond * 1000

}

func (g *Game) OnLeave() {

}
func (g *Game) Update() error {

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		router.Go("home", nil)
	}

	g.tim += time.Second / time.Duration(ebiten.TPS())
	g.lyric.Update(g.tim)

	//_, wy := ebiten.Wheel()
	/*for _, e := range g.line.GetSyllables() {
		for _, s := range e.Elements {
			s.SetNowOffset(s.GetNowOffset() + wy*5)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(ebiten.MouseButtonLeft)) {
		g.line.Dispose()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(ebiten.MouseButtonRight)) {
		g.line.Render()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.fontsize += 4.0
		g.line.SetFontSize(g.fontsize)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.fontsize -= 4.0
		g.line.SetFontSize(g.fontsize)
	}*/
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.lyric.Draw(screen)
	//ebitenutil.DebugPrint(screen, fmt.Sprint(g.tim.Seconds()))
	msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) OnResize(w, he int, isFirst bool) {
	log.Println("Game OnResize", w, he, isFirst)
	if !isFirst {
	}
}
