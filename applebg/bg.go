package applebg

import (
	"EbitenLyrics/filters"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Renderer 定义渲染器接口
type Renderer interface {
	Update() error
	Draw(screen *ebiten.Image)
	SetFlowSpeed(speed float64)
	SetRenderScale(scale float64)
	SetStaticMode(enable bool)
	SetFPS(fps int)
	Pause()
	Resume()
	SetAlbum(img *ebiten.Image) error
	SetLowFreqVolume(volume float64)
	SetHasLyric(hasLyric bool)
	Resize(width, height int)
	Dispose()
}

// SpriteLayer 表示一个旋转的图层
type SpriteLayer struct {
	image        *ebiten.Image
	rotation     float64
	rotateSpeed  float64
	scale        float64
	offsetX      float64
	offsetY      float64
	orbitSpeed   float64
	orbitRadius  float64
	initialAngle float64
}

// Container 容器，包含多个图层和透明度
type Container struct {
	layers []*SpriteLayer
	alpha  float64
	time   float64
}

// AppleMusicRenderer Apple Music 风格的渲染器
type AppleMusicRenderer struct {
	width       int
	height      int
	renderScale float64
	flowSpeed   float64
	staticMode  bool
	paused      bool
	fps         int

	currentContainer *Container
	lastContainers   []*Container

	// 缓存的渲染图像
	offscreen   *ebiten.Image
	blurBuffer1 *ebiten.Image
	blurBuffer2 *ebiten.Image

	mu sync.RWMutex

	blurShader *ebiten.Shader
}

// Resize 改变渲染器大小
func (r *AppleMusicRenderer) Resize(width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.width == width && r.height == height {
		return
	}

	r.width = width
	r.height = height
	r.initBuffers()

	// 重新计算所有图层的缩放和位置
	maxSize := math.Max(float64(width), float64(height))

	// 更新当前容器
	if r.currentContainer != nil {
		for i, layer := range r.currentContainer.layers {
			if layer.image == nil {
				continue
			}
			imgWidth := float64(layer.image.Bounds().Dx())

			switch i {
			case 0:
				layer.scale = maxSize * math.Sqrt2 / imgWidth
			case 1:
				layer.scale = maxSize * 0.8 / imgWidth
			case 2:
				layer.scale = maxSize * 0.5 / imgWidth
				layer.orbitRadius = maxSize / 4
			case 3:
				layer.scale = maxSize * 0.25 / imgWidth
				layer.orbitRadius = maxSize / 4 * 0.1
			}
		}
	}

	// 更新过期容器
	for _, container := range r.lastContainers {
		for i, layer := range container.layers {
			if layer.image == nil {
				continue
			}
			imgWidth := float64(layer.image.Bounds().Dx())

			switch i {
			case 0:
				layer.scale = maxSize * math.Sqrt2 / imgWidth
			case 1:
				layer.scale = maxSize * 0.8 / imgWidth
			case 2:
				layer.scale = maxSize * 0.5 / imgWidth
				layer.orbitRadius = maxSize / 4
			case 3:
				layer.scale = maxSize * 0.25 / imgWidth
				layer.orbitRadius = maxSize / 4 * 0.1
			}
		}
	}
}

// NewAppleMusicRenderer 创建新的渲染器
func NewAppleMusicRenderer(width, height int) *AppleMusicRenderer {
	r := &AppleMusicRenderer{
		width:       width,
		height:      height,
		renderScale: 0.75,
		flowSpeed:   4.0,
		staticMode:  false,
		paused:      false,
		fps:         30,
	}
	// 加载模糊着色器
	f, err := os.ReadFile(filepath.Join("kages", "blur.kage"))
	if err != nil {
		panic(err)
	}
	r.blurShader, err = ebiten.NewShader(f)
	if err != nil {
		panic(err)
	}

	r.initBuffers()
	return r
}

// initBuffers 初始化渲染缓冲区
func (r *AppleMusicRenderer) initBuffers() {
	w := int(float64(r.width) * r.renderScale)
	h := int(float64(r.height) * r.renderScale)

	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	r.offscreen = ebiten.NewImage(w, h)
	r.blurBuffer1 = ebiten.NewImage(w, h)
	r.blurBuffer2 = ebiten.NewImage(w, h)
}

// SetFlowSpeed 设置流动速度
func (r *AppleMusicRenderer) SetFlowSpeed(speed float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flowSpeed = speed
}

// SetRenderScale 设置渲染比例
func (r *AppleMusicRenderer) SetRenderScale(scale float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renderScale = scale
	r.initBuffers()
}

// SetStaticMode 设置静态模式
func (r *AppleMusicRenderer) SetStaticMode(enable bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.staticMode = enable
}

// SetFPS 设置帧率
func (r *AppleMusicRenderer) SetFPS(fps int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fps = fps
	if fps > 0 {
		ebiten.SetTPS(fps)
	}
}

// Pause 暂停动画
func (r *AppleMusicRenderer) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = true
}

// Resume 恢复动画
func (r *AppleMusicRenderer) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = false
}

// SetLowFreqVolume 设置低频音量（预留接口）
func (r *AppleMusicRenderer) SetLowFreqVolume(volume float64) {
	// 可以根据音量调整动画效果
}

// SetHasLyric 设置是否有歌词（预留接口）
func (r *AppleMusicRenderer) SetHasLyric(hasLyric bool) {
	// 可以根据歌词状态调整效果
}

// SetAlbum 设置专辑图片
func (r *AppleMusicRenderer) SetAlbum(img *ebiten.Image) error {
	if img == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建新容器
	container := &Container{
		layers: make([]*SpriteLayer, 4),
		alpha:  0.0,
		time:   0.0,
	}

	// 初始化4个图层，不同的大小和旋转速度
	maxSize := math.Max(float64(r.width), float64(r.height))

	// 图层1: 最大，慢速正向旋转
	container.layers[0] = &SpriteLayer{
		image:        img,
		rotation:     rand.Float64() * math.Pi * 2,
		rotateSpeed:  1.0 / 1000.0,
		scale:        maxSize * math.Sqrt2 / float64(img.Bounds().Dx()),
		offsetX:      0,
		offsetY:      0,
		orbitSpeed:   0,
		orbitRadius:  0,
		initialAngle: 0,
	}

	// 图层2: 中大，快速反向旋转
	container.layers[1] = &SpriteLayer{
		image:        img,
		rotation:     rand.Float64() * math.Pi * 2,
		rotateSpeed:  -1.0 / 500.0,
		scale:        maxSize * 0.8 / float64(img.Bounds().Dx()),
		offsetX:      0,
		offsetY:      0,
		orbitSpeed:   0,
		orbitRadius:  0,
		initialAngle: 0,
	}

	// 图层3: 中小，正向旋转 + 圆周运动
	container.layers[2] = &SpriteLayer{
		image:        img,
		rotation:     rand.Float64() * math.Pi * 2,
		rotateSpeed:  1.0 / 1000.0,
		scale:        maxSize * 0.5 / float64(img.Bounds().Dx()),
		offsetX:      0,
		offsetY:      0,
		orbitSpeed:   0.75 / 1000.0,
		orbitRadius:  maxSize / 4,
		initialAngle: 0,
	}

	// 图层4: 最小，反向旋转 + 小幅圆周运动
	container.layers[3] = &SpriteLayer{
		image:        img,
		rotation:     rand.Float64() * math.Pi * 2,
		rotateSpeed:  -1.0 / 750.0,
		scale:        maxSize * 0.25 / float64(img.Bounds().Dx()),
		offsetX:      0,
		offsetY:      0,
		orbitSpeed:   0.75 * 0.006,
		orbitRadius:  maxSize / 4 * 0.1,
		initialAngle: 0,
	}

	// 将当前容器移到过期列表
	if r.currentContainer != nil {
		r.lastContainers = append(r.lastContainers, r.currentContainer)
	}

	r.currentContainer = container

	return nil
}

// Update 更新逻辑
func (r *AppleMusicRenderer) Update() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused {
		return nil
	}

	delta := 1.0

	// 更新过期容器的透明度
	for i := len(r.lastContainers) - 1; i >= 0; i-- {
		container := r.lastContainers[i]
		container.alpha -= delta / 60.0
		if container.alpha <= 0 {
			// 移除完全透明的容器
			r.lastContainers = append(r.lastContainers[:i], r.lastContainers[i+1:]...)
		}
	}

	// 更新当前容器
	if r.currentContainer != nil {
		// 淡入
		if r.currentContainer.alpha < 1.0 {
			r.currentContainer.alpha += delta / 60.0
			if r.currentContainer.alpha > 1.0 {
				r.currentContainer.alpha = 1.0
			}
		}

		// 更新时间和旋转
		r.currentContainer.time += delta * r.flowSpeed

		for i, layer := range r.currentContainer.layers {
			layer.rotation += delta * layer.rotateSpeed * r.flowSpeed

			// 图层3和4有圆周运动
			if i == 2 {
				layer.offsetX = layer.orbitRadius * math.Cos(r.currentContainer.time*layer.orbitSpeed)
				layer.offsetY = layer.orbitRadius * math.Cos(r.currentContainer.time*layer.orbitSpeed)
			} else if i == 3 {
				layer.offsetX = layer.orbitRadius + math.Cos(r.currentContainer.time*layer.orbitSpeed)
				layer.offsetY = layer.orbitRadius + math.Cos(r.currentContainer.time*layer.orbitSpeed)
			}
		}

		// 静态模式检查
		if r.staticMode && r.currentContainer.alpha >= 1.0 && len(r.lastContainers) == 0 {
			r.paused = true
		}
	}

	return nil
}

// Draw 绘制
func (r *AppleMusicRenderer) Draw(screen *ebiten.Image) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.offscreen.Clear()

	// 绘制所有容器
	containers := append(r.lastContainers, r.currentContainer)
	for _, container := range containers {
		if container == nil {
			continue
		}
		r.drawContainer(container)
	}

	// 应用模糊和颜色调整效果
	r.applyEffects()

	// 缩放到屏幕大小
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(r.width)/float64(r.offscreen.Bounds().Dx()),
		float64(r.height)/float64(r.offscreen.Bounds().Dy()),
	)
	screen.DrawImage(r.offscreen, op)
}

// drawContainer 绘制单个容器
func (r *AppleMusicRenderer) drawContainer(container *Container) {
	centerX := float64(r.offscreen.Bounds().Dx()) / 2
	centerY := float64(r.offscreen.Bounds().Dy()) / 2

	for i, layer := range container.layers {
		if layer.image == nil {
			continue
		}

		op := &ebiten.DrawImageOptions{}

		// 设置锚点到中心
		bounds := layer.image.Bounds()
		op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

		// 缩放
		op.GeoM.Scale(layer.scale, layer.scale)

		// 旋转
		op.GeoM.Rotate(layer.rotation)

		// 位置（根据图层调整）
		posX := centerX
		posY := centerY

		if i == 1 {
			posX = centerX * 0.8
			posY = centerY * 0.8
		}

		posX += layer.offsetX * r.renderScale
		posY += layer.offsetY * r.renderScale

		op.GeoM.Translate(posX, posY)

		// 透明度
		op.ColorScale.ScaleAlpha(float32(container.alpha))

		r.offscreen.DrawImage(layer.image, op)
	}
}

// applyEffects 应用视觉效果
func (r *AppleMusicRenderer) applyEffects() {
	// 简化的模糊效果：通过多次采样实现
	// 真实实现需要使用shader或更复杂的算法

	// 调整亮度和饱和度（简化版）
	// Ebiten 需要使用 ColorScale 或 shader 来实现完整的颜色矩阵
	// 这里使用 ColorScale 做基本调整
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(1.2, 1.2, 1.2, 1.0) // 提高亮度

	r.blurBuffer1.Clear()
	r.blurBuffer1.DrawImage(r.offscreen, op)

	// 简单的盒式模糊
	r.boxBlur(r.blurBuffer1, r.offscreen, 5)
}

// boxBlur 简单的盒式模糊
func (r *AppleMusicRenderer) boxBlur(src, dst *ebiten.Image, radius int) {
	// 这是一个简化实现，真实的模糊需要更复杂的算法
	// 或使用 Ebiten 的 shader 功能
	dst.Clear()

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	d := filters.GaussianBlur2Pass(src, r.blurShader, 10)
	dst.DrawImage(d, op)
	d.Deallocate()
	d = nil
}

// Dispose 释放资源
func (r *AppleMusicRenderer) Dispose() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.offscreen != nil {
		r.offscreen.Dispose()
	}
	if r.blurBuffer1 != nil {
		r.blurBuffer1.Dispose()
	}
	if r.blurBuffer2 != nil {
		r.blurBuffer2.Dispose()
	}
}
