package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/evbus"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/router"
	"EbitenLyrics/ws"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Home struct {
	router.BaseScene // 嵌入并获得默认实现
	Font             *text.GoTextFaceSource

	LyricsImageAnim *anim.Tween
	AnimateManager  *anim.Manager

	LyricsControl *lyrics.Lyrics

	Cover         *ebiten.Image
	CoverPosition lyrics.Position
}

func (h *Home) OnCreate() {
	log.Println("Home OnCreate")
}
func (h *Home) OnEnter(params map[string]any) {
	log.Println("Home OnEnter", params)

	if h.Font == nil {
		log.Fatalln("Home page Font is nil")
	}

	h.CoverPosition = lyrics.NewPosition(0, 0, 0, 0)
	evbus.Bus.Subscribe("ws:setLyric", func(value []interface{}) {
		log.Println(value)
		d, err := ws.ParseLyricsFromMap(value)
		if err != nil {
			log.Fatalln(err)
		}

		if h.LyricsControl != nil {
			h.LyricsControl.Dispose()

		}
		ww, _ := ebiten.WindowSize()
		h.LyricsControl, err = lyrics.New(d, float64(ww), h.Font, 50)
		if err != nil {
			log.Fatalln(err)
		}
		h.LyricsControl.AnimateManager = h.AnimateManager
		h.LyricsControl.HighlightTime = time.Millisecond * 800
		if len(h.LyricsControl.Lines) > 0 {
			h.LyricsControl.Scroll([]int{0}, 0)
		}

	})
	evbus.Bus.Subscribe("ws:progress", func(value float64) {
		if h.LyricsControl != nil {
			h.LyricsControl.Position = time.Duration(value) * time.Millisecond
			h.LyricsControl.Update(h.LyricsControl.Position)
		}
	})
	evbus.Bus.Subscribe("ws:cover", func(img *ebiten.Image) {
		h.Cover = img
		h.CoverPosition.W = float64(img.Bounds().Dx())
		h.CoverPosition.H = float64(img.Bounds().Dy())
		h.CoverPosition.OriginX = h.CoverPosition.W / 2
		h.CoverPosition.OriginY = h.CoverPosition.H / 2
		screenWidth, screenHeight := ebiten.WindowSize()
		scaleX := float64(screenWidth) / h.CoverPosition.W
		scaleY := float64(screenHeight) / h.CoverPosition.H
		finalS := math.Max(scaleX, scaleY) * 1.4

		h.CoverPosition.TranslateX = (float64(screenWidth) - h.CoverPosition.W) / 2
		h.CoverPosition.TranslateY = (float64(screenHeight) - h.CoverPosition.H) / 2

		h.CoverPosition.ScaleX = finalS
		h.CoverPosition.ScaleY = finalS

	})

}

func (h *Home) OnLeave() {
	log.Println("Home OnLeave")
}

func (h *Home) OnDestroy() {
	log.Println("Home OnDestroy")
}

func (h *Home) Update() error {

	/*h.CoverPosition.Rotate += 0.5
	if h.CoverPosition.Rotate > 360 {
		h.CoverPosition.Rotate = 0
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		router.Go("game", nil)
	}*/

	return nil
}

func (h *Home) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Home Scene")

	if h.Cover != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM = lyrics.TransformToGeoM(&h.CoverPosition)
		screen.DrawImage(h.Cover, op)
	}
	if h.LyricsControl != nil {

		h.LyricsControl.Draw(screen)
	}

}

func (h *Home) OnResize(w, he int, isFirst bool) {
	log.Println("Home OnResize", w, he, isFirst)
	if !isFirst {
		if h.LyricsControl != nil {
			h.LyricsControl.Resize(float64(w))
		}

		h.CoverPosition.TranslateX = (float64(w) - h.CoverPosition.W) / 2
		h.CoverPosition.TranslateY = (float64(he) - h.CoverPosition.H) / 2

		scaleX := float64(w) / h.CoverPosition.W
		scaleY := float64(he) / h.CoverPosition.H
		finalS := math.Max(scaleX, scaleY) * 1.4
		h.CoverPosition.ScaleX = finalS
		h.CoverPosition.ScaleY = finalS
	}
}
