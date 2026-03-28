package main

// 文件说明：程序入口，负责初始化字体、场景路由和 Ebiten 主循环。
// 主要职责：创建全局 Game 状态，并把窗口生命周期转发给当前场景。

import (
	"EbitenLyrics/anim"
	f "EbitenLyrics/font"
	"EbitenLyrics/pages"
	"EbitenLyrics/router"
	"EbitenLyrics/ws"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var game Game

type Game struct {
	animMgr      *anim.Manager
	last         time.Time
	fontManager  *f.FontManager
	fontRequest  f.FontRequest
	lastW, lastH int
}

func (g *Game) currentOutsideSize() (int, int) {
	if ebiten.IsFullscreen() {
		if m := ebiten.Monitor(); m != nil {
			w, h := m.Size()
			if w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return ebiten.WindowSize()
}

func (g *Game) Update() error {
	w, h := g.currentOutsideSize()

	if router.NeedFirstResize() {
		if scene := router.Current(); scene != nil {
			scene.OnResize(w, h, true)
		}
		router.ClearFirstResizeFlag()
		g.lastW = w
		g.lastH = h
	}

	if w != g.lastW || h != g.lastH {
		if scene := router.Current(); scene != nil {
			scene.OnResize(w, h, false)
		}
		g.lastW = w
		g.lastH = h
	}

	scene := router.Current()
	if scene == nil {
		return nil
	}
	return scene.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now
	g.animMgr.Update(dt)
	screen.Clear()

	if scene := router.Current(); scene != nil {
		scene.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		w, h := g.currentOutsideSize()
		if w <= 0 || h <= 0 {
			return 1, 1
		}
		return w, h
	}
	return outsideWidth, outsideHeight
}

func main() {
	initfont()

	ebiten.SetWindowSize(1100, 670)
	game.animMgr = anim.NewManager(false)

	router.Add("home", &pages.Home{
		FontManager:    game.fontManager,
		FontRequest:    game.fontRequest,
		AnimateManager: game.animMgr,
	})
	router.Add("game", &pages.Game{
		FontManager:    game.fontManager,
		FontRequest:    game.fontRequest,
		AnimateManager: game.animMgr,
	})
	router.Add("manage", &pages.Manage{
		FontManager:    game.fontManager,
		FontRequest:    game.fontRequest,
		AnimateManager: game.animMgr,
	})

	router.Go("home", nil)
	game.last = time.Now()

	ebiten.SetWindowTitle("Ebiten Lyrics")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetFullscreen(false)
	ebiten.SetTPS(60)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	go ws.Initws()
	if err := ebiten.RunGameWithOptions(&game, &ebiten.RunGameOptions{
		X11ClassName:    "Ebiten Lyrics",
		X11InstanceName: "Ebiten Lyrics",
	}); err != nil {
		log.Fatal(err)
	}
}

func initfont() {
	game.fontManager = f.DefaultManager()
	req := f.DefaultRequest()
	configPath := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_CONFIG"))
	if configPath == "" {
		configPath = f.DefaultFontConfigPath
	}
	if fromFile, err := game.fontManager.LoadRequestFromFile(configPath, req); err == nil {
		req = fromFile
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("load font config failed: %v", err)
	}
	req, err := game.fontManager.ApplyEnvRequest(req)
	if err != nil {
		log.Fatalf("failed to apply font request: %v", err)
	}
	game.fontRequest = req

	resolved, err := game.fontManager.ResolveChain(req)
	if err != nil || resolved == nil || resolved.Primary == nil {
		log.Fatalf("failed to resolve font: %v", err)
	}
	log.Printf(
		"font selected: family=%q style=%q weight=%d path=%s",
		resolved.Primary.Family,
		resolved.Primary.Style,
		resolved.Primary.Weight,
		resolved.Primary.Path,
	)
}
