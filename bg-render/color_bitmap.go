package bgrender

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// GenerateColorStrip 生成1D颜色条（256x1）用于 shader 采样
func GenerateColorStrip(palette []PaletteColor, width int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, 1))

	if len(palette) == 0 {
		return img
	}

	// 构建累积权重数组
	cumWeights := make([]float64, len(palette)+1)
	for i, p := range palette {
		cumWeights[i+1] = cumWeights[i] + p.Weight
	}
	total := cumWeights[len(palette)]

	// 归一化
	for i := range cumWeights {
		cumWeights[i] /= total
	}

	// 为每个像素插值颜色
	for x := 0; x < width; x++ {
		t := float64(x) / float64(width-1)

		// 找到 t 对应的颜色段
		idx := 0
		for i := 0; i < len(palette); i++ {
			if t >= cumWeights[i] && t < cumWeights[i+1] {
				idx = i
				break
			}
		}

		// 在相邻颜色之间插值
		var r, g, b float64
		if idx < len(palette)-1 {
			// 计算局部插值系数
			localT := (t - cumWeights[idx]) / (cumWeights[idx+1] - cumWeights[idx])

			c1 := palette[idx]
			c2 := palette[idx+1]

			r = lerp(float64(c1.R), float64(c2.R), localT)
			g = lerp(float64(c1.G), float64(c2.G), localT)
			b = lerp(float64(c1.B), float64(c2.B), localT)
		} else {
			// 最后一个颜色
			c := palette[idx]
			r, g, b = float64(c.R), float64(c.G), float64(c.B)
		}

		img.SetRGBA(x, 0, color.RGBA{
			R: uint8(math.Min(255, math.Max(0, r))),
			G: uint8(math.Min(255, math.Max(0, g))),
			B: uint8(math.Min(255, math.Max(0, b))),
			A: 255,
		})
	}

	return img
}

// CreateColorStrip 从图片创建颜色条纹
func CreateColorStrip(img image.Image) *ebiten.Image {
	const width = 256 // 1D 颜色条宽度

	// 提取调色板
	palette := ExtractPalette(img, 5)

	// 生成1D颜色条
	strip := GenerateColorStrip(palette, width)

	return ebiten.NewImageFromImage(strip)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
