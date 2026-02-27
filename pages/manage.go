package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/router"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Manage struct {
	router.BaseScene
	Font           *text.GoTextFaceSource
	AnimateManager *anim.Manager
}

func (m *Manage) OnCreate() {
	log.Println("Manage OnCreate")

}
func (m *Manage) OnEnter(params map[string]any) {
	log.Println("Manage OnEnter", params)
}
func (m *Manage) OnLeave() {
	log.Println("Manage OnLeave")
}
func (m *Manage) OnDestroy() {
	log.Println("Manage OnDestroy")
}
func (m *Manage) Update() error {
	return nil
}
func (m *Manage) Draw(screen *ebiten.Image) {
}
func (m *Manage) OnResize(w, he int, isFirst bool) {
	log.Println("Manage OnResize", w, he, isFirst)
}
