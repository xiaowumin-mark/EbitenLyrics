package main

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
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var game Game

type Game struct {
	animMgr         *anim.Manager
	last            time.Time
	mplusFaceSource *text.GoTextFaceSource
	fontFallbacks   []*text.GoTextFaceSource
	lastW, lastH    int
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
		Font:           game.mplusFaceSource,
		FontFallbacks:  game.fontFallbacks,
		AnimateManager: game.animMgr,
	})
	router.Add("game", &pages.Game{
		Font:           game.mplusFaceSource,
		FontFallbacks:  game.fontFallbacks,
		AnimateManager: game.animMgr,
	})
	router.Add("manage", &pages.Manage{
		Font:           game.mplusFaceSource,
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
	opts := f.DefaultResolveOptions()
	configPath := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_CONFIG"))
	if configPath == "" {
		configPath = f.DefaultRuntimeFontConfigPath
	}
	if fromFile, err := f.LoadResolveOptionsFromFile(configPath, opts); err == nil {
		opts = fromFile
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("load font config failed: %v", err)
	}
	opts = f.ApplyEnvResolveOptions(opts)

	resolved, err := f.ResolveFaceSource(opts)
	if err != nil {
		log.Fatalf("failed to resolve font: %v", err)
	}
	game.mplusFaceSource = resolved.Source
	game.fontFallbacks = append([]*text.GoTextFaceSource{}, resolved.Fallbacks...)
	log.Printf(
		"font selected: family=%q style=%q weight=%d path=%s",
		resolved.Family,
		resolved.Style,
		resolved.Weight,
		resolved.Path,
	)
}
