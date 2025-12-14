package pages

import (
	"EbitenLyrics/router"
	"image"
	_ "image/png"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type AppleMusic3D struct {
	router.BaseScene
	shader       *ebiten.Shader
	albumCover   *ebiten.Image
	blurredCover *ebiten.Image
	time         float32
}

func (a *AppleMusic3D) OnCreate() {
	log.Println("AppleMusic3D OnCreate")

	// 3D 扭曲 + 模糊效果
	shaderCode := `//kage:unit pixels

package main

var Time float
var Intensity float // 扭曲强度

// 简单噪声
func noise(p vec2) float {
	return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453)
}

// 平滑噪声
func smoothNoise(p vec2) float {
	i := floor(p)
	f := fract(p)
	f = f * f * (3.0 - 2.0 * f)
	
	a := noise(i)
	b := noise(i + vec2(1.0, 0.0))
	c := noise(i + vec2(0.0, 1.0))
	d := noise(i + vec2(1.0, 1.0))
	
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// UV 坐标
	uv := texCoord
	
	// 中心点
	center := vec2(0.5, 0.5)
	
	// 计算距离中心的距离
	dist := distance(uv, center)
	
	// 基于时间的扭曲偏移
	t := Time * 0.3
	
	// 使用噪声生成扭曲
	noiseVal1 := smoothNoise(uv * 2.0 + vec2(t, 0.0))
	noiseVal2 := smoothNoise(uv * 2.0 + vec2(0.0, t))
	
	// 扭曲偏移
	offset := vec2(
		(noiseVal1 - 0.5) * Intensity,
		(noiseVal2 - 0.5) * Intensity,
	)
	
	// 添加径向扭曲（从中心向外）
	toCenter := uv - center
	angle := Time * 0.5
	rotation := mat2(
		cos(angle * dist), -sin(angle * dist),
		sin(angle * dist), cos(angle * dist),
	)
	toCenter = rotation * toCenter * (1.0 + dist * 0.2)
	
	// 应用扭曲
	distortedUV := center + toCenter + offset
	
	// 采样图像（带边界处理）
	if distortedUV.x < 0.0 || distortedUV.x > 1.0 || 
	   distortedUV.y < 0.0 || distortedUV.y > 1.0 {
		// 边缘外使用模糊版本
		return imageSrc1At(texCoord)
	}
	
	// 正常采样
	col := imageSrc0At(distortedUV)
	
	// 边缘渐变到模糊
	edgeFade := smoothstep(0.4, 0.6, dist)
	blurred := imageSrc1At(texCoord)
	
	return mix(col, blurred, edgeFade * 0.5)
}`

	sha, err := ebiten.NewShader([]byte(shaderCode))
	if err != nil {
		log.Fatal("Shader error:", err)
	}
	a.shader = sha

	// 加载封面
	f, err := os.Open("E:/projects/visual-lyric/music/Opalite.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// 转换为正方形并缩放
	a.albumCover = ebiten.NewImageFromImage(img)

	// 创建模糊版本（简单缩小再放大）
	smallSize := 32
	smallImg := ebiten.NewImage(smallSize, smallSize)
	op := &ebiten.DrawImageOptions{}

	// 缩小
	w, h := a.albumCover.Bounds().Dx(), a.albumCover.Bounds().Dy()
	scale := float64(smallSize) / float64(max(w, h))
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear
	smallImg.DrawImage(a.albumCover, op)

	// 放大回来（产生模糊效果）
	a.blurredCover = ebiten.NewImage(w, h)
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Scale(1.0/scale, 1.0/scale)
	op2.Filter = ebiten.FilterLinear
	a.blurredCover.DrawImage(smallImg, op2)

	log.Printf("✓ Album loaded: %dx%d", w, h)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *AppleMusic3D) OnEnter(params map[string]any) {
	a.time = 0
}

func (a *AppleMusic3D) OnLeave()   {}
func (a *AppleMusic3D) OnDestroy() {}

func (a *AppleMusic3D) Update() error {
	a.time += 1.0 / 60.0
	return nil
}

func (a *AppleMusic3D) OnResize(w, h int, isFirst bool) {}

func (a *AppleMusic3D) Draw(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	// 将封面缩放到屏幕大小
	coverW := a.albumCover.Bounds().Dx()
	coverH := a.albumCover.Bounds().Dy()

	scaleX := float64(w) / float64(coverW)
	scaleY := float64(h) / float64(coverH)
	scale := math.Max(scaleX, scaleY)

	// 先绘制到临时图像（确保尺寸一致）
	tempCover := ebiten.NewImage(w, h)
	tempBlur := ebiten.NewImage(w, h)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(w-coverW)/2, float64(h-coverH)/2)
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear

	tempCover.DrawImage(a.albumCover, op)
	tempBlur.DrawImage(a.blurredCover, op)

	// 应用 shader
	shaderOp := &ebiten.DrawRectShaderOptions{}
	shaderOp.Uniforms = map[string]interface{}{
		"Time":      a.time,
		"Intensity": float32(0.05), // 扭曲强度，可调整
	}
	shaderOp.Images[0] = tempCover // 原始封面
	shaderOp.Images[1] = tempBlur  // 模糊封面

	screen.DrawRectShader(w, h, a.shader, shaderOp)
}
