package bgrender

// 文件说明：背景渲染器统一包装层。
// 主要职责：对外暴露专辑图、音量、尺寸和暂停等控制接口。

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Renderer matches the feature set of the web BackgroundRenderer in Go/Ebiten.
type Renderer interface {
	SetFlowSpeed(speed float64)
	SetRenderScale(scale float64)
	SetStaticMode(enable bool)
	SetFPS(fps int)
	Pause()
	Resume()
	SetAlbum(albumSource any, isVideo ...bool) error
	SetLowFreqVolume(volume float64)
	SetHasLyric(hasLyric bool)
	Update(dt time.Duration)
	Draw(screen *ebiten.Image)
	Resize(width, height int)
	Dispose()
}

// BackgroundRender is a small wrapper equivalent to web's BackgroundRender.
type BackgroundRender struct {
	renderer Renderer
}

func NewBackgroundRender(renderer Renderer) *BackgroundRender {
	return &BackgroundRender{renderer: renderer}
}

func (b *BackgroundRender) SetFlowSpeed(speed float64) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetFlowSpeed(speed)
}

func (b *BackgroundRender) SetRenderScale(scale float64) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetRenderScale(scale)
}

func (b *BackgroundRender) SetStaticMode(enable bool) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetStaticMode(enable)
}

func (b *BackgroundRender) SetFPS(fps int) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetFPS(fps)
}

func (b *BackgroundRender) Pause() {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Pause()
}

func (b *BackgroundRender) Resume() {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Resume()
}

func (b *BackgroundRender) SetAlbum(albumSource any, isVideo ...bool) error {
	if b == nil || b.renderer == nil {
		return nil
	}
	return b.renderer.SetAlbum(albumSource, isVideo...)
}

func (b *BackgroundRender) SetAlbumImage(img image.Image) error {
	if b == nil || b.renderer == nil {
		return nil
	}
	return b.renderer.SetAlbum(img, false)
}

func (b *BackgroundRender) SetLowFreqVolume(volume float64) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetLowFreqVolume(volume)
}

func (b *BackgroundRender) SetHasLyric(hasLyric bool) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.SetHasLyric(hasLyric)
}

func (b *BackgroundRender) Update(dt time.Duration) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Update(dt)
}

func (b *BackgroundRender) Draw(screen *ebiten.Image) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Draw(screen)
}

func (b *BackgroundRender) Resize(width, height int) {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Resize(width, height)
}

func (b *BackgroundRender) Dispose() {
	if b == nil || b.renderer == nil {
		return
	}
	b.renderer.Dispose()
	b.renderer = nil
}
