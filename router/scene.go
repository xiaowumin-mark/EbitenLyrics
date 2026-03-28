package router

// 文件说明：定义场景接口以及默认空实现，统一页面生命周期约定。
// 主要职责：让不同页面以一致的方式接入创建、进入、离开、绘制和尺寸变化流程。

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
