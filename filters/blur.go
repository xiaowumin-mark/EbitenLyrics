package filters

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// 最大模糊半径限制 (受限于 GPU Uniform 数组大小，32 已经很大了，足以覆盖非常柔和的阴影)
const MaxKernelRadius = 32

// ----------------------------------------------------
// Kage Shader 代码 (运行在 GPU 上)
// ----------------------------------------------------
var blurShaderSrc = []byte(`
//kage:unit pixels

package main

// Uniform 变量，由 Go 代码传入
var Direction vec2       // 模糊方向: (1, 0) 或 (0, 1)
var Weights [32]float    // 预计算的高斯权重 (只存一半，因为是对称的)
var Radius int           // 实际使用的半径

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// 获取中心像素颜色，并乘以中心权重 (Weights[0])
	// imageSrc0UnsafeAt 在像素坐标模式下直接通过 (x, y) 取样
	sum := imageSrc0UnsafeAt(srcPos) * Weights[0]

	// 循环累加两侧的像素
	// 注意：GPU 循环次数通常建议是常量，但 Ebiten/Kage 支持变量循环
	// 我们使用硬编码的最大值作为上限，内部通过 if 判断跳出
	for i := 1; i < 32; i++ {
		if i > Radius {
			break
		}
		
		offset := Direction * float(i)
		w := Weights[i]

		// 累加正方向和负方向的像素 (利用高斯核的对称性)
		sum += imageSrc0UnsafeAt(srcPos + offset) * w
		sum += imageSrc0UnsafeAt(srcPos - offset) * w
	}

	return sum
}
`)

// ----------------------------------------------------
// BlurRenderer: 模糊渲染器结构体
// ----------------------------------------------------
// 建议在 Game 结构体中持有一个 BlurRenderer 实例，而不是每次调用都创建
type BlurRenderer struct {
	shader  *ebiten.Shader
	tempImg *ebiten.Image // 中间缓冲层，避免反复分配内存

	// 缓存，减少内存分配
	//weights [MaxKernelRadius]float
	weights [MaxKernelRadius]float32
}

// NewBlurRenderer 初始化渲染器
func NewBlurRenderer() (*BlurRenderer, error) {
	shader, err := ebiten.NewShader(blurShaderSrc)
	if err != nil {
		return nil, err
	}
	return &BlurRenderer{
		shader: shader,
	}, nil
}

// Blur 对 src 进行模糊处理并返回结果。
// 注意：返回的图像在下一次调用 Blur 时可能会被复用/覆盖，如果需要持久保存，请自行 clone。
func (r *BlurRenderer) Blur(src *ebiten.Image, radius int, sigma float64) *ebiten.Image {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w == 0 || h == 0 {
		return src
	}

	r.ensureTempImage(w, h)

	// 这里假设你每次都 New 一个结果图（如果 Result 也是复用的，Result 也要用 Copy 模式）
	resultImg := ebiten.NewImage(w, h)

	if radius >= MaxKernelRadius {
		radius = MaxKernelRadius - 1
	}
	if radius < 0 {
		radius = 0
	}
	r.calcGaussianWeights(radius, sigma)

	// ------------------------------------------------
	// Pass 1: 水平模糊 (Source -> Temp)
	// ------------------------------------------------
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	op.Uniforms = map[string]interface{}{
		"Direction": []float32{1.0, 0.0},
		"Weights":   r.weights,
		"Radius":    radius,
	}

	// 【关键修复】：使用 Copy 模式
	// 这告诉 Ebiten：不管 tempImg 原来有什么，直接忽略，强制写入新颜色。
	// 这比 r.tempImg.Clear() 更快！
	op.CompositeMode = ebiten.CompositeModeCopy

	r.tempImg.DrawRectShader(w, h, r.shader, op)

	// ------------------------------------------------
	// Pass 2: 垂直模糊 (Temp -> Result)
	// ------------------------------------------------
	op.Images[0] = r.tempImg
	op.Uniforms = map[string]interface{}{
		"Direction": []float32{0.0, 1.0},
		"Weights":   r.weights,
		"Radius":    radius,
	}

	// ResultImg 是新的，其实默认模式也行，但保持一致用 Copy 更保险
	op.CompositeMode = ebiten.CompositeModeCopy

	resultImg.DrawRectShader(w, h, r.shader, op)

	return resultImg
}

// ensureTempImage 确保临时图像存在且尺寸正确
func (r *BlurRenderer) ensureTempImage(w, h int) {
	if r.tempImg != nil {
		tw, th := r.tempImg.Bounds().Dx(), r.tempImg.Bounds().Dy()
		if tw == w && th == h {
			return
		}
		r.tempImg.Dispose() // 尺寸不匹配，释放旧的
	}
	r.tempImg = ebiten.NewImage(w, h)
}

// calcGaussianWeights 计算高斯权重放入 r.weights 数组
func (r *BlurRenderer) calcGaussianWeights(radius int, sigma float64) {
	// 初始化
	for i := range r.weights {
		r.weights[i] = 0
	}

	if radius == 0 {
		r.weights[0] = 1.0
		return
	}

	// 临时切片用于计算归一化
	// 注意：我们只需要计算 [0...radius]，shader 利用对称性处理负方向
	tempWeights := make([]float64, radius+1)
	sum := 0.0
	twoSigmaSq := 2 * sigma * sigma

	for i := 0; i <= radius; i++ {
		x := float64(i)
		w := math.Exp(-(x * x) / twoSigmaSq)
		tempWeights[i] = w

		// 归一化总和: 中心点加一次，其他点加两次(因为对称)
		if i == 0 {
			sum += w
		} else {
			sum += 2.0 * w
		}
	}

	// 归一化并填入 float32 数组 (Shader 需要 float32 兼容)
	for i := 0; i <= radius; i++ {
		r.weights[i] = float32(tempWeights[i] / sum)
	}
}

// ----------------------------------------------------
// 简易调用封装 (CSS 风格)
// ----------------------------------------------------

// 全局单例 (为了方便直接调用，实际项目中建议自行管理生命周期)
var globalBlurRenderer *BlurRenderer

func BlurImageShader(sourceImg *ebiten.Image, blurPixels float64) *ebiten.Image {
	if blurPixels <= 0 {
		return ebiten.NewImageFromImage(sourceImg)
	}

	// 懒加载初始化
	if globalBlurRenderer == nil {
		var err error
		globalBlurRenderer, err = NewBlurRenderer()
		if err != nil {
			panic(err) // 实际中应妥善处理错误
		}
	}

	// 转换参数
	sigma := blurPixels * 0.5
	radius := int(math.Ceil(sigma * 3.0))

	return globalBlurRenderer.Blur(sourceImg, radius, sigma)
}
