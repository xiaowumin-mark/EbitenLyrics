package uis

import "fmt"

type Button struct {
	X, Y, W, H float64
	Hovered    bool
	Text       string
}

func (b *Button) Bounds() (float64, float64, float64, float64) {
	return b.X, b.Y, b.W, b.H
}

func (b *Button) OnHoverEnter() { b.Hovered = true }
func (b *Button) OnHoverExit()  { b.Hovered = false }
func (b *Button) OnClick()      { fmt.Println("按钮被点击:", b.Text) }
func (b *Button) Update()       {}
