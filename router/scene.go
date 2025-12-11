package router

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Scene interface {
	OnCreate()
	OnEnter(params map[string]any)
	OnLeave()
	OnDestroy()
	OnResize(width, height int, isFirst bool)

	Update() error
	Draw(screen *ebiten.Image)
}

// BaseScene 提供全部生命周期的空实现
type BaseScene struct {
}

func (b *BaseScene) OnCreate()                       {}
func (b *BaseScene) OnEnter(params map[string]any)   {}
func (b *BaseScene) OnLeave()                        {}
func (b *BaseScene) OnDestroy()                      {}
func (b *BaseScene) OnResize(w, h int, isFirst bool) {}
func (b *BaseScene) Update() error                   { return nil }
func (b *BaseScene) Draw(screen *ebiten.Image)       {}
func (b *BaseScene) Layout(w, h int) (int, int)      { return w, h }
