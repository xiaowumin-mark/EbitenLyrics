package pages

import (
	"EbitenLyrics/applebg"
	"EbitenLyrics/router"
	"log"

	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type BC struct {
	router.BaseScene // 嵌入并获得默认实现

	Img *ebiten.Image
	a   *applebg.AppleMusicRenderer
}

func (b *BC) OnCreate() {
	log.Println("BC OnCreate")

	img, err := imaging.Open("E:/projects/visual-lyric/music/Try Everything.png")
	if err != nil {
		log.Fatal(err)
	}
	img = imaging.Resize(img, 1200, 1200, imaging.Lanczos)
	//img = imaging.Blur(img, 60)
	b.Img = ebiten.NewImageFromImage(img)
	w, h := ebiten.WindowSize()
	b.a = applebg.NewAppleMusicRenderer(w, h)
	b.a.SetFPS(30)
	b.a.SetAlbum(b.Img)
}
func (b *BC) OnEnter(params map[string]any) {
	log.Println("BC OnEnter", params)

}
func (b *BC) OnLeave() {
	log.Println("BC OnLeave")
}
func (b *BC) OnDestroy() {
	log.Println("BC OnDestroy")
}

func (b *BC) Update() error {
	b.a.Update()
	return nil
}
func (b *BC) OnResize(w, h int, isFirst bool) {
	if !isFirst {
		b.a.Resize(w, h)
	}
}
func (b *BC) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "BC Scene")
	b.a.Draw(screen)
}
