package filters

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// GaussianBlurParams 高斯模糊参数
type GaussianBlurParams struct {
	Weights     [32]float32
	Offsets     [32]float32
	SampleCount int
}

func Downsample(src *ebiten.Image, scale float64) *ebiten.Image {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)

	dst := ebiten.NewImage(nw, nh)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear // 关键！
	dst.DrawImage(src, op)
	return dst
}

func CalculateGaussianParams(radius int) GaussianBlurParams {
	sigma := float64(radius) / 3.0

	var params GaussianBlurParams
	size := radius*2 + 1
	kernel := make([]float64, size)

	sum := 0.0
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-(float64(i * i)) / (2 * sigma * sigma))
		kernel[i+radius] = v
		sum += v
	}

	for i := range kernel {
		kernel[i] /= sum
	}

	idx := 0
	params.Weights[idx] = float32(kernel[radius])
	params.Offsets[idx] = 0
	idx++

	for i := 1; i <= radius && idx < 32; i++ {
		params.Weights[idx] = float32(kernel[radius+i])
		params.Offsets[idx] = float32(i)
		idx++
	}

	params.SampleCount = idx
	return params
}

func GaussianBlur2Pass(
	src *ebiten.Image,
	shader *ebiten.Shader,
	radius int,
) *ebiten.Image {

	params := CalculateGaussianParams(radius)

	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	tmp := ebiten.NewImage(w, h)
	dst := ebiten.NewImage(w, h)

	// X pass
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	op.Uniforms = map[string]interface{}{
		"TextureSize": []float32{float32(w), float32(h)},
		"Direction":   []float32{1, 0},
		"Weights":     params.Weights,
		"Offsets":     params.Offsets,
		"SampleCount": params.SampleCount,
	}
	tmp.DrawRectShader(w, h, shader, op)

	// Y pass
	op.Images[0] = tmp
	op.Uniforms["Direction"] = []float32{0, 1}
	dst.DrawRectShader(w, h, shader, op)

	tmp.Dispose()
	return dst
}

func Upsample(src *ebiten.Image, w, h int) *ebiten.Image {
	dst := ebiten.NewImage(w, h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(w)/float64(src.Bounds().Dx()),
		float64(h)/float64(src.Bounds().Dy()),
	)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(src, op)
	return dst
}

type RealtimeGaussianBlur struct {
	Shader *ebiten.Shader

	// 固定最大参数
	MaxRadius int
	Params    GaussianBlurParams

	// 缓存图像
	tmp  *ebiten.Image
	down *ebiten.Image
	blur *ebiten.Image

	// 原始尺寸
	W, H  int
	Scale float64
}

func NewRealtimeGaussianBlur(
	w, h int,
	shader *ebiten.Shader,
	maxRadius int,
	scale float64, // 0.5 推荐
) *RealtimeGaussianBlur {

	dsW := int(float64(w) * scale)
	dsH := int(float64(h) * scale)

	return &RealtimeGaussianBlur{
		Shader: shader,

		MaxRadius: maxRadius,
		Params:    CalculateGaussianParams(maxRadius),

		tmp:  ebiten.NewImage(dsW, dsH),
		down: ebiten.NewImage(dsW, dsH),
		blur: ebiten.NewImage(dsW, dsH),

		W:     w,
		H:     h,
		Scale: scale,
	}
}

func (g *RealtimeGaussianBlur) Apply(
	src *ebiten.Image,
	radius float64, // 动态的！
) *ebiten.Image {

	// 1️⃣ Clamp
	if radius < 0 {
		radius = 0
	}
	if radius > float64(g.MaxRadius) {
		radius = float64(g.MaxRadius)
	}

	// 2️⃣ 当前 SampleCount
	r := int(math.Ceil(radius))
	if r < 1 {
		// 半径 0 直接返回原图
		return src
	}

	// 3️⃣ Downsample
	{
		g.down.Clear()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(g.Scale, g.Scale)
		op.Filter = ebiten.FilterLinear
		g.down.DrawImage(src, op)
	}

	dsW := g.down.Bounds().Dx()
	dsH := g.down.Bounds().Dy()

	// 4️⃣ X Pass
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = g.down
	op.Uniforms = map[string]interface{}{
		"TextureSize": []float32{float32(dsW), float32(dsH)},
		"Direction":   []float32{1, 0},
		"Weights":     g.Params.Weights,
		"Offsets":     g.Params.Offsets,
		"SampleCount": r,
	}
	g.tmp.DrawRectShader(dsW, dsH, g.Shader, op)

	// 5️⃣ Y Pass
	op.Images[0] = g.tmp
	op.Uniforms["Direction"] = []float32{0, 1}
	g.blur.DrawRectShader(dsW, dsH, g.Shader, op)

	// 6️⃣ Upsample
	dst := ebiten.NewImage(g.W, g.H)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(
			float64(g.W)/float64(dsW),
			float64(g.H)/float64(dsH),
		)
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(g.blur, op)
	}

	return dst
}

func Smooth(current, target, dt float64) float64 {
	k := 1.0 - math.Exp(-dt*12) // 12 ≈ UI 阻尼
	return current + (target-current)*k
}
