package main

import (
	"EbitenLyrics/anim"
	f "EbitenLyrics/font"
	"EbitenLyrics/pages"
	"EbitenLyrics/router"
	"EbitenLyrics/ws"
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var game Game

type Game struct {
	animMgr         *anim.Manager
	last            time.Time
	mplusFaceSource *text.GoTextFaceSource
	lastW, lastH    int
}

func (g *Game) Update() error {
	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now
	g.animMgr.Update(dt)
	w, h := ebiten.WindowSize()
	// 如果是页面刚刚切换过来，立即触发一次 isFirst=true
	if router.NeedFirstResize() {
		if router.Current() != nil {
			router.Current().OnResize(w, h, true)
		}
		router.ClearFirstResizeFlag()
		g.lastW = w
		g.lastH = h
	}

	// 检测尺寸是否变化
	if w != g.lastW || h != g.lastH {
		if router.Current() != nil {
			router.Current().OnResize(w, h, false)
		}
		g.lastW = w
		g.lastH = h
	}

	return router.Current().Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Clear()
	router.Current().Draw(screen)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return ebiten.WindowSize()
}

func main() {

	//log.Println(f.GetAllFonts())
	initfont()
	ebiten.SetWindowSize(1100, 670)
	game.animMgr = anim.NewManager(false)
	router.Add("home", &pages.Home{
		Font:           game.mplusFaceSource,
		AnimateManager: game.animMgr,
	})
	router.Add("game", &pages.Game{
		Font:           game.mplusFaceSource,
		AnimateManager: game.animMgr,
	})
	router.Add("bc", &pages.BC{
		/*Font:           game.mplusFaceSource,
		AnimateManager: game.animMgr,*/
	})
	router.Add("uv", &pages.BC{})

	router.Go("uv", nil)

	game.last = time.Now()

	ebiten.SetWindowTitle("Ebiten Lyrics")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetFullscreen(false)

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	go ws.Initws()
	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}

func initfont() {
	//fn, err := f.FindFonts("segoeui.ttf")
	//fn, err := f.FindFonts("HarmonyOS_Sans_SC_Medium.ttf")
	fn, err := f.FindFonts("msyh.ttc")
	if err != nil {
		panic(err)
	}
	sources, err := loadTTCSources(fn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("子字体数量：", len(sources))

	// 打印每个子字体的 metadata（例如名字等）
	for i, s := range sources {
		md := s.Metadata()
		fmt.Printf("index=%d family=%q style=%q\n", i, md.Family, md.Style)
	}

	//fmt.Println("字体成功读取:", face)

	/*game.mplusFaceSource, err = text.NewGoTextFaceSource(file)
	if err != nil {
		panic(err)
	}*/
	game.mplusFaceSource = sources[0]
}
func loadTTCSources(path string) ([]*text.GoTextFaceSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Parse collection: 返回每个子字体的 GoTextFaceSource 列表（适用于 .ttc）
	sources, err := text.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return sources, nil
}
