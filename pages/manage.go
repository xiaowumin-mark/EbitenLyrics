package pages

// 文件说明：管理页场景骨架。
// 主要职责：承载后续管理界面或工具页面的生命周期实现。

import (
	"EbitenLyrics/anim"
	f "EbitenLyrics/font"
	"EbitenLyrics/router"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type Manage struct {
	router.BaseScene
	FontManager    *f.FontManager
	FontRequest    f.FontRequest
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
