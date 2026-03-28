package lyrics

// 文件说明：高质量重采样绘制工具。
// 主要职责：在缩放图像时使用自定义着色器获得更稳定的采样效果。

import (
	_ "embed"
	"image"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/resample_bicubic4x4.kage
var resampleBicubic4x4ShaderSource []byte

var (
	resampleShaderOnce        sync.Once
	resampleShader            *ebiten.Shader
	resampleShaderErr         error
	resampleShaderLogFailOnce sync.Once
)

var resampleQuadIndices = [...]uint16{0, 1, 2, 0, 2, 3}

func loadResampleShader() (*ebiten.Shader, error) {
	resampleShaderOnce.Do(func() {
		resampleShader, resampleShaderErr = ebiten.NewShader(resampleBicubic4x4ShaderSource)
	})
	return resampleShader, resampleShaderErr
}

func drawImageLinear(dst, src *ebiten.Image, geom ebiten.GeoM, alpha float32, blend ebiten.Blend) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM = geom
	op.Filter = ebiten.FilterLinear
	op.ColorScale.ScaleAlpha(alpha)
	op.Blend = blend
	dst.DrawImage(src, op)
}

func drawImageResample4x4(dst, src *ebiten.Image, geom ebiten.GeoM, alpha float32, blend ebiten.Blend) {
	if dst == nil || src == nil || alpha <= 0 {
		return
	}

	shader, err := loadResampleShader()
	if err != nil {
		resampleShaderLogFailOnce.Do(func() {
			log.Printf("lyrics: load 4x4 resample shader failed, fallback to FilterLinear: %v", err)
		})
		drawImageLinear(dst, src, geom, alpha, blend)
		return
	}

	b := src.Bounds()
	if b.Empty() {
		return
	}

	vertices := buildImageQuadVertices(b, geom, alpha)
	opts := &ebiten.DrawTrianglesShaderOptions{
		Blend: blend,
	}
	opts.Images[0] = src
	dst.DrawTrianglesShader(vertices[:], resampleQuadIndices[:], shader, opts)
}

func buildImageQuadVertices(b image.Rectangle, geom ebiten.GeoM, alpha float32) [4]ebiten.Vertex {
	sx0, sy0 := float64(b.Min.X), float64(b.Min.Y)
	sx1, sy1 := float64(b.Max.X), float64(b.Max.Y)

	dx0, dy0 := geom.Apply(sx0, sy0)
	dx1, dy1 := geom.Apply(sx1, sy0)
	dx2, dy2 := geom.Apply(sx1, sy1)
	dx3, dy3 := geom.Apply(sx0, sy1)
	scale := alpha

	return [4]ebiten.Vertex{
		{
			DstX:   float32(dx0),
			DstY:   float32(dy0),
			SrcX:   float32(sx0),
			SrcY:   float32(sy0),
			ColorR: scale,
			ColorG: scale,
			ColorB: scale,
			ColorA: alpha,
		},
		{
			DstX:   float32(dx1),
			DstY:   float32(dy1),
			SrcX:   float32(sx1),
			SrcY:   float32(sy0),
			ColorR: scale,
			ColorG: scale,
			ColorB: scale,
			ColorA: alpha,
		},
		{
			DstX:   float32(dx2),
			DstY:   float32(dy2),
			SrcX:   float32(sx1),
			SrcY:   float32(sy1),
			ColorR: scale,
			ColorG: scale,
			ColorB: scale,
			ColorA: alpha,
		},
		{
			DstX:   float32(dx3),
			DstY:   float32(dy3),
			SrcX:   float32(sx0),
			SrcY:   float32(sy1),
			ColorR: scale,
			ColorG: scale,
			ColorB: scale,
			ColorA: alpha,
		},
	}
}
