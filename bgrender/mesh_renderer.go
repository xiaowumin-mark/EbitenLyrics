package bgrender

// 文件说明：网格背景渲染器实现。
// 主要职责：驱动控制点动画、纹理更新和最终绘制输出。

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/hajimehoshi/ebiten/v2"
)

type meshState struct {
	mesh    *BHPMesh
	texture *ebiten.Image
	alpha   float64
}

func (s *meshState) dispose() {
	if s == nil {
		return
	}
	if s.texture != nil {
		s.texture.Deallocate()
		s.texture = nil
	}
	s.mesh = nil
}

type MeshGradientRenderer struct {
	shader *ebiten.Shader

	flowSpeed   float64
	renderScale float64
	maxFPS      int

	paused     bool
	staticMode bool

	manualControl bool
	wireFrame     bool

	hasLyric       bool
	volume         float64
	smoothedVolume float64
	scaleStrength  float64
	targetScale    float64
	smoothedScale  float64
	satStrength    float64
	targetSat      float64
	smoothedSat    float64
	coverStrength  float64
	targetCover    float64
	smoothedCover  float64

	frameTimeMS  float64
	frameElapsed time.Duration
	staticStable bool

	enablePerformanceMonitoring bool
	frameCount                  int
	currentFPS                  int
	fpsAccum                    time.Duration

	logicalWidth  int
	logicalHeight int
	scene         *ebiten.Image
	sceneWidth    int
	sceneHeight   int

	meshStates []*meshState
	isNoCover  bool
	disposed   bool

	shaderUniforms map[string]any
}

func NewMeshGradientRenderer(width, height int) (*MeshGradientRenderer, error) {
	shader, err := loadMeshBGShader()
	if err != nil {
		return nil, err
	}
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	r := &MeshGradientRenderer{
		shader:        shader,
		flowSpeed:     1,
		renderScale:   0.75,
		maxFPS:        60,
		logicalWidth:  width,
		logicalHeight: height,
		isNoCover:     true,
		hasLyric:      true,
		scaleStrength: 0.05,
		satStrength:   1.45,
		coverStrength: 0.10,
		shaderUniforms: map[string]any{
			"Time":            float32(0),
			"Volume":          float32(0),
			"Alpha":           float32(1),
			"SaturationBoost": float32(1),
			"CoverScale":      float32(1),
		},
	}
	r.targetScale = 1
	r.smoothedScale = 1
	r.targetSat = 1
	r.smoothedSat = 1
	r.targetCover = 1
	r.smoothedCover = 1
	r.ensureScene()
	return r, nil
}

func (r *MeshGradientRenderer) ensureScene() {
	if r.disposed {
		return
	}
	targetW := int(math.Ceil(float64(r.logicalWidth) * r.renderScale))
	targetH := int(math.Ceil(float64(r.logicalHeight) * r.renderScale))
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}
	if r.scene != nil && r.sceneWidth == targetW && r.sceneHeight == targetH {
		return
	}
	if r.scene != nil {
		r.scene.Deallocate()
	}
	r.scene = ebiten.NewImage(targetW, targetH)
	r.sceneWidth = targetW
	r.sceneHeight = targetH
}

func (r *MeshGradientRenderer) SetFlowSpeed(speed float64) {
	if speed <= 0 {
		speed = 0.0001
	}
	r.flowSpeed = speed
}

func (r *MeshGradientRenderer) SetRenderScale(scale float64) {
	if scale <= 0 {
		scale = 0.1
	}
	if scale > 2 {
		scale = 2
	}
	r.renderScale = scale
	r.ensureScene()
	r.staticStable = false
}

func (r *MeshGradientRenderer) SetStaticMode(enable bool) {
	r.staticMode = enable
	r.staticStable = false
}

func (r *MeshGradientRenderer) SetFPS(fps int) {
	if fps < 0 {
		fps = 0
	}
	r.maxFPS = fps
}

func (r *MeshGradientRenderer) Pause() {
	r.paused = true
}

func (r *MeshGradientRenderer) Resume() {
	r.paused = false
	r.staticStable = false
}

func (r *MeshGradientRenderer) SetManualControl(enable bool) {
	r.manualControl = enable
	if enable {
		r.staticStable = false
	}
}

func (r *MeshGradientRenderer) SetWireFrame(enable bool) {
	r.wireFrame = enable
	for _, state := range r.meshStates {
		if state != nil && state.mesh != nil {
			state.mesh.SetWireFrame(enable)
		}
	}
}

func (r *MeshGradientRenderer) GetControlPoint(x, y int) *ControlPoint {
	if len(r.meshStates) == 0 {
		return nil
	}
	mesh := r.meshStates[len(r.meshStates)-1].mesh
	if mesh == nil {
		return nil
	}
	return mesh.GetControlPoint(x, y)
}

func (r *MeshGradientRenderer) ResizeControlPoints(width, height int) {
	if len(r.meshStates) == 0 {
		return
	}
	mesh := r.meshStates[len(r.meshStates)-1].mesh
	if mesh == nil {
		return
	}
	if err := mesh.ResizeControlPoints(width, height); err != nil {
		return
	}
	mesh.UpdateMesh()
}

func (r *MeshGradientRenderer) ResetSubdivition(subDivisions int) {
	if len(r.meshStates) == 0 {
		return
	}
	mesh := r.meshStates[len(r.meshStates)-1].mesh
	if mesh == nil {
		return
	}
	mesh.ResetSubdivition(subDivisions)
	mesh.UpdateMesh()
}

func (r *MeshGradientRenderer) EnablePerformanceMonitor(enable bool) {
	r.enablePerformanceMonitoring = enable
	r.frameCount = 0
	r.currentFPS = 0
	r.fpsAccum = 0
}

func (r *MeshGradientRenderer) GetCurrentFPS() int {
	return r.currentFPS
}

func (r *MeshGradientRenderer) SetLowFreqVolume(volume float64) {
	r.volume = clamp(volume, 0, 1)
}

func (r *MeshGradientRenderer) SetScaleStrength(strength float64) {
	r.scaleStrength = clamp(strength, 0, 0.5)
}

func (r *MeshGradientRenderer) ScaleStrength() float64 {
	if r == nil {
		return 0
	}
	return r.scaleStrength
}

func (r *MeshGradientRenderer) SetSaturationStrength(strength float64) {
	r.satStrength = clamp(strength, 0, 3)
}

func (r *MeshGradientRenderer) SaturationStrength() float64 {
	if r == nil {
		return 0
	}
	return r.satStrength
}

func (r *MeshGradientRenderer) SetCoverScaleStrength(strength float64) {
	r.coverStrength = clamp(strength, 0, 1)
}

func (r *MeshGradientRenderer) CoverScaleStrength() float64 {
	if r == nil {
		return 0
	}
	return r.coverStrength
}

func (r *MeshGradientRenderer) SetHasLyric(hasLyric bool) {
	r.hasLyric = hasLyric
}

func (r *MeshGradientRenderer) Update(dt time.Duration) {
	if r.disposed || r.paused || dt <= 0 {
		return
	}
	if r.maxFPS <= 0 {
		return
	}
	if r.staticMode && r.staticStable {
		return
	}

	r.frameElapsed += dt
	interval := time.Second / time.Duration(r.maxFPS)
	if interval <= 0 {
		interval = time.Second / 60
	}
	if r.frameElapsed < interval {
		return
	}

	frameDelta := r.frameElapsed
	r.frameElapsed = 0
	r.frameTimeMS += frameDelta.Seconds() * 1000 * r.flowSpeed

	r.updatePerformance(frameDelta)
	canBeStatic := r.onTick(frameDelta)
	if r.staticMode && canBeStatic {
		r.staticStable = true
	}
}

func (r *MeshGradientRenderer) updatePerformance(dt time.Duration) {
	if !r.enablePerformanceMonitoring {
		return
	}
	r.frameCount++
	r.fpsAccum += dt
	if r.fpsAccum >= time.Second {
		r.currentFPS = int(math.Round(float64(r.frameCount) / r.fpsAccum.Seconds()))
		r.frameCount = 0
		r.fpsAccum = 0
	}
}

func (r *MeshGradientRenderer) onTick(delta time.Duration) bool {
	latest := r.latestState()
	canBeStatic := false
	deltaFactor := delta.Seconds() * 1000 / 500

	if latest != nil {
		if r.manualControl && latest.mesh != nil {
			latest.mesh.UpdateMesh()
		}
		if r.isNoCover {
			active := false
			filtered := r.meshStates[:0]
			for _, state := range r.meshStates {
				if state == nil {
					continue
				}
				state.alpha = math.Max(-0.1, state.alpha-deltaFactor)
				if state.alpha <= -0.1 {
					state.dispose()
					continue
				}
				active = true
				filtered = append(filtered, state)
			}
			r.meshStates = filtered
			canBeStatic = !active
		} else {
			if latest.alpha >= 1.1 {
				if len(r.meshStates) > 1 {
					for i := 0; i < len(r.meshStates)-1; i++ {
						if r.meshStates[i] != nil {
							r.meshStates[i].dispose()
						}
					}
					r.meshStates = r.meshStates[len(r.meshStates)-1:]
				}
			} else {
				latest.alpha = math.Min(1.1, latest.alpha+deltaFactor)
			}
			canBeStatic = len(r.meshStates) == 1 && latest.alpha >= 1.1
		}
	}

	timeConstantMs := 140.0
	if r.volume > r.smoothedVolume {
		timeConstantMs = 55.0
	}
	lerp := 1 - math.Exp(-delta.Seconds()*1000/timeConstantMs)
	r.smoothedVolume += (r.volume - r.smoothedVolume) * lerp
	r.targetScale = backgroundScaleForVolume(r.smoothedVolume, r.scaleStrength)
	scaleTimeConstantMs := 200.0
	if r.targetScale > r.smoothedScale {
		scaleTimeConstantMs = 70.0
	}
	scaleLerp := 1 - math.Exp(-delta.Seconds()*1000/scaleTimeConstantMs)
	r.smoothedScale += (r.targetScale - r.smoothedScale) * scaleLerp
	if r.smoothedScale < 1 {
		r.smoothedScale = 1
	}
	r.targetSat = backgroundSaturationForVolume(r.smoothedVolume, r.satStrength)
	satTimeConstantMs := 260.0
	if r.targetSat > r.smoothedSat {
		satTimeConstantMs = 85.0
	}
	satLerp := 1 - math.Exp(-delta.Seconds()*1000/satTimeConstantMs)
	r.smoothedSat += (r.targetSat - r.smoothedSat) * satLerp
	if r.smoothedSat < 1 {
		r.smoothedSat = 1
	}
	r.targetCover = backgroundCoverScaleForVolume(r.smoothedVolume, r.coverStrength)
	coverTimeConstantMs := 180.0
	if r.targetCover > r.smoothedCover {
		coverTimeConstantMs = 60.0
	}
	coverLerp := 1 - math.Exp(-delta.Seconds()*1000/coverTimeConstantMs)
	r.smoothedCover += (r.targetCover - r.smoothedCover) * coverLerp
	if r.smoothedCover < 1 {
		r.smoothedCover = 1
	}
	return canBeStatic
}

func (r *MeshGradientRenderer) latestState() *meshState {
	if len(r.meshStates) == 0 {
		return nil
	}
	return r.meshStates[len(r.meshStates)-1]
}

func (r *MeshGradientRenderer) HasRenderableState() bool {
	for _, state := range r.meshStates {
		if state != nil && state.mesh != nil && state.texture != nil {
			return true
		}
	}
	return false
}

func (r *MeshGradientRenderer) Draw(screen *ebiten.Image) {
	if r.disposed || screen == nil {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		return
	}
	if r.logicalWidth != sw || r.logicalHeight != sh {
		r.logicalWidth = sw
		r.logicalHeight = sh
		r.ensureScene()
	}
	if r.scene == nil || r.shader == nil {
		return
	}

	r.shaderUniforms["Time"] = float32(r.frameTimeMS / 10000.0)
	r.shaderUniforms["Volume"] = float32(shaderVisualVolume(r.smoothedVolume))
	r.shaderUniforms["Alpha"] = float32(1)
	r.shaderUniforms["SaturationBoost"] = float32(r.smoothedSat)
	r.shaderUniforms["CoverScale"] = float32(r.smoothedCover)

	aspect := float64(r.sceneWidth) / float64(r.sceneHeight)
	scaleX := float64(sw) / float64(r.sceneWidth)
	scaleY := float64(sh) / float64(r.sceneHeight)

	for _, state := range r.meshStates {
		if state == nil || state.mesh == nil || state.texture == nil {
			continue
		}
		tw := state.texture.Bounds().Dx()
		th := state.texture.Bounds().Dy()
		verts := state.mesh.Vertices(r.sceneWidth, r.sceneHeight, tw, th, aspect, r.manualControl)
		inds := state.mesh.Indices()
		if len(verts) == 0 || len(inds) == 0 {
			continue
		}
		scaleVerticesAroundCenter(verts, float32(r.sceneWidth), float32(r.sceneHeight), float32(r.smoothedScale))

		r.scene.Clear()
		drawTrianglesOpts := &ebiten.DrawTrianglesShaderOptions{}
		drawTrianglesOpts.Images[0] = state.texture
		drawTrianglesOpts.Uniforms = r.shaderUniforms
		r.scene.DrawTrianglesShader(verts, inds, r.shader, drawTrianglesOpts)

		alpha := easeInOutSine(clamp(state.alpha, 0, 1))
		if len(r.meshStates) == 1 && state.alpha <= 0 {
			alpha = 1
		}
		drawImageOpts := &ebiten.DrawImageOptions{}
		drawImageOpts.GeoM.Scale(scaleX, scaleY)
		drawImageOpts.ColorScale.ScaleAlpha(float32(alpha))
		screen.DrawImage(r.scene, drawImageOpts)
	}
}

func shaderVisualVolume(volume float64) float64 {
	volume = clamp(volume, 0, 1)
	return math.Pow(volume, 0.7) * 0.22
}

func backgroundScaleForVolume(volume, strength float64) float64 {
	volume = clamp(volume, 0, 1)
	strength = clamp(strength, 0, 0.5)
	return 1 + math.Pow(volume, 0.82)*strength
}

func backgroundSaturationForVolume(volume, strength float64) float64 {
	volume = clamp(volume, 0, 1)
	strength = clamp(strength, 0, 3)
	return 1 + math.Pow(volume, 0.8)*strength
}

func backgroundCoverScaleForVolume(volume, strength float64) float64 {
	volume = clamp(volume, 0, 1)
	strength = clamp(strength, 0, 1)
	return 1 + math.Pow(volume, 0.75)*strength
}

func scaleVerticesAroundCenter(vertices []ebiten.Vertex, width, height, scale float32) {
	if len(vertices) == 0 || width <= 0 || height <= 0 || scale <= 0 {
		return
	}
	if scale == 1 {
		return
	}
	centerX := width * 0.5
	centerY := height * 0.5
	for i := range vertices {
		vertices[i].DstX = centerX + (vertices[i].DstX-centerX)*scale
		vertices[i].DstY = centerY + (vertices[i].DstY-centerY)*scale
	}
}

func (r *MeshGradientRenderer) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	r.logicalWidth = width
	r.logicalHeight = height
	r.ensureScene()
}

func (r *MeshGradientRenderer) Dispose() {
	if r.disposed {
		return
	}
	r.disposed = true
	for _, state := range r.meshStates {
		if state != nil {
			state.dispose()
		}
	}
	r.meshStates = nil
	if r.scene != nil {
		r.scene.Deallocate()
		r.scene = nil
	}
	r.shaderUniforms = nil
}

func easeInOutSine(x float64) float64 {
	return -(math.Cos(math.Pi*x) - 1) / 2
}

func (r *MeshGradientRenderer) SetAlbum(albumSource any, _ ...bool) error {
	if r.disposed {
		return errors.New("renderer is disposed")
	}
	if albumSource == nil {
		r.isNoCover = true
		r.staticStable = false
		return nil
	}
	if s, ok := albumSource.(string); ok && strings.TrimSpace(s) == "" {
		r.isNoCover = true
		r.staticStable = false
		return nil
	}

	srcImage, err := resolveAlbumSource(albumSource)
	if err != nil {
		r.isNoCover = true
		r.staticStable = false
		return err
	}
	processed := preprocessAlbumImage(srcImage)
	if processed == nil {
		r.isNoCover = true
		r.staticStable = false
		return errors.New("processed album image is nil")
	}

	r.isNoCover = false
	r.staticStable = false
	if r.manualControl && len(r.meshStates) > 0 {
		state := r.meshStates[0]
		if state.texture != nil {
			state.texture.Deallocate()
		}
		state.texture = processed
		state.alpha = 1
		return nil
	}

	mesh := NewBHPMesh()
	mesh.SetWireFrame(r.wireFrame)
	mesh.ResetSubdivition(50)

	chosen := pickControlPointPreset()
	if err := mesh.ResizeControlPoints(chosen.Width, chosen.Height); err != nil {
		processed.Deallocate()
		return err
	}
	uPower := 2.0 / float64(chosen.Width-1)
	vPower := 2.0 / float64(chosen.Height-1)
	for _, conf := range chosen.Conf {
		p := mesh.GetControlPoint(conf.CX, conf.CY)
		if p == nil {
			continue
		}
		p.Location[0] = conf.X
		p.Location[1] = conf.Y
		p.SetURot(conf.UR * math.Pi / 180)
		p.SetVRot(conf.VR * math.Pi / 180)
		p.SetUScale(uPower * conf.UP)
		p.SetVScale(vPower * conf.VP)
	}
	mesh.UpdateMesh()

	initialAlpha := 0.0
	if len(r.meshStates) == 0 {
		initialAlpha = 1.0
	}
	r.meshStates = append(r.meshStates, &meshState{
		mesh:    mesh,
		texture: processed,
		alpha:   initialAlpha,
	})
	return nil
}

func pickControlPointPreset() ControlPointPreset {
	if len(ControlPointPresets) == 0 {
		return GenerateControlPoints(6, 6)
	}
	if randFloat64() > 0.8 {
		return GenerateControlPoints(6, 6)
	}
	return ControlPointPresets[randIntn(len(ControlPointPresets))]
}

func resolveAlbumSource(source any) (image.Image, error) {
	switch s := source.(type) {
	case image.Image:
		return s, nil
	case *ebiten.Image:
		return ebitenImageToNRGBA(s), nil
	case string:
		return decodeImageFromPathOrURL(s)
	default:
		return nil, errors.New("unsupported album source type")
	}
}

func ebitenImageToNRGBA(img *ebiten.Image) *image.NRGBA {
	if img == nil {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		w, h = 1, 1
	}
	pix := make([]byte, w*h*4)
	img.ReadPixels(pix)
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(out.Pix, pix)
	return out
}

func decodeImageFromPathOrURL(pathOrURL string) (image.Image, error) {
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return nil, errors.New("empty image path")
	}
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(pathOrURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New("failed to fetch album image")
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		return img, err
	}

	f, err := os.Open(pathOrURL)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func preprocessAlbumImage(src image.Image) *ebiten.Image {
	if src == nil {
		return nil
	}
	small := imaging.Resize(src, 32, 32, imaging.Linear)
	nrgba := imaging.Clone(small)

	pixels := nrgba.Pix
	for i := 0; i+3 < len(pixels); i += 4 {
		r := float64(pixels[i])
		g := float64(pixels[i+1])
		b := float64(pixels[i+2])

		// ref-like fixed tone: low contrast, strong saturation, then contrast + brightness.
		r = (r-128)*0.4 + 128
		g = (g-128)*0.4 + 128
		b = (b-128)*0.4 + 128

		gray := r*0.3 + g*0.59 + b*0.11
		r = gray*-2.0 + r*3.0
		g = gray*-2.0 + g*3.0
		b = gray*-2.0 + b*3.0

		r = (r-128)*1.7 + 128
		g = (g-128)*1.7 + 128
		b = (b-128)*1.7 + 128

		r *= 0.75
		g *= 0.75
		b *= 0.75

		pixels[i] = floatToByte(r)
		pixels[i+1] = floatToByte(g)
		pixels[i+2] = floatToByte(b)
	}

	shuffleCoverTiles(nrgba, 2, rand.New(rand.NewSource(time.Now().UnixNano())))
	blurNRGBA(nrgba, 3.0)
	return ebiten.NewImageFromImage(nrgba)
}

func shuffleCoverTiles(img *image.NRGBA, grid int, rng *rand.Rand) {
	if img == nil || rng == nil || grid < 2 {
		return
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	tileW := w / grid
	tileH := h / grid
	if tileW <= 0 || tileH <= 0 {
		return
	}
	total := grid * grid
	perm := derangedPermutation(total, rng)
	if len(perm) != total {
		return
	}
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(src.Pix, img.Pix)
	for dstIdx, srcIdx := range perm {
		dx := dstIdx % grid
		dy := dstIdx / grid
		sx := srcIdx % grid
		sy := srcIdx / grid

		dstX0 := dx * tileW
		dstY0 := dy * tileH
		srcX0 := sx * tileW
		srcY0 := sy * tileH
		dstX1 := dstX0 + tileW
		dstY1 := dstY0 + tileH
		srcX1 := srcX0 + tileW
		srcY1 := srcY0 + tileH
		if dx == grid-1 {
			dstX1 = w
		}
		if dy == grid-1 {
			dstY1 = h
		}
		if sx == grid-1 {
			srcX1 = w
		}
		if sy == grid-1 {
			srcY1 = h
		}
		for y := 0; y < srcY1-srcY0 && dstY0+y < dstY1; y++ {
			for x := 0; x < srcX1-srcX0 && dstX0+x < dstX1; x++ {
				si := src.PixOffset(srcX0+x, srcY0+y)
				di := img.PixOffset(dstX0+x, dstY0+y)
				copy(img.Pix[di:di+4], src.Pix[si:si+4])
			}
		}
	}
}

func derangedPermutation(n int, rng *rand.Rand) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{0}
	}
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	for attempt := 0; attempt < 64; attempt++ {
		for i := n - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			perm[i], perm[j] = perm[j], perm[i]
		}
		ok := true
		for i := range perm {
			if perm[i] == i {
				ok = false
				break
			}
		}
		if ok {
			return append([]int(nil), perm...)
		}
		for i := range perm {
			perm[i] = i
		}
	}
	for i := 0; i < n-1; i += 2 {
		perm[i], perm[i+1] = i+1, i
	}
	if n%2 == 1 {
		perm[n-1] = n - 2
		perm[n-2] = n - 1
	}
	return perm
}

type albumImageStats struct {
	avgLuma  float64
	minLuma  float64
	maxLuma  float64
	contrast float64
	avgSat   float64
}

func floatToByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func blurNRGBA(img *image.NRGBA, blurPx float64) {
	if img == nil || blurPx <= 0 {
		return
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	sigma := math.Max(blurPx*0.5, 0.01)
	radius := int(math.Ceil(sigma * 3))
	if radius < 1 {
		radius = 1
	}
	weights := gaussianWeights(radius, sigma)
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(src.Pix, img.Pix)
	temp := image.NewNRGBA(image.Rect(0, 0, w, h))
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	horizontalBlurNRGBA(temp, src, weights, radius)
	verticalBlurNRGBA(dst, temp, weights, radius)
	copy(img.Pix, dst.Pix)
}

func gaussianWeights(radius int, sigma float64) []float64 {
	weights := make([]float64, radius+1)
	if radius <= 0 {
		weights[0] = 1
		return weights
	}
	twoSigmaSq := 2 * sigma * sigma
	sum := 0.0
	for i := 0; i <= radius; i++ {
		w := math.Exp(-(float64(i) * float64(i)) / twoSigmaSq)
		weights[i] = w
		if i == 0 {
			sum += w
		} else {
			sum += 2 * w
		}
	}
	for i := range weights {
		weights[i] /= sum
	}
	return weights
}

func horizontalBlurNRGBA(dst, src *image.NRGBA, weights []float64, radius int) {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sumR, sumG, sumB, sumA float64
			for k := -radius; k <= radius; k++ {
				xx := x + k
				if xx < 0 {
					xx = 0
				} else if xx >= w {
					xx = w - 1
				}
				i := src.PixOffset(xx, y)
				wgt := weights[absInt(k)]
				sumR += float64(src.Pix[i]) * wgt
				sumG += float64(src.Pix[i+1]) * wgt
				sumB += float64(src.Pix[i+2]) * wgt
				sumA += float64(src.Pix[i+3]) * wgt
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i] = uint8(sumR + 0.5)
			dst.Pix[i+1] = uint8(sumG + 0.5)
			dst.Pix[i+2] = uint8(sumB + 0.5)
			dst.Pix[i+3] = uint8(sumA + 0.5)
		}
	}
}

func verticalBlurNRGBA(dst, src *image.NRGBA, weights []float64, radius int) {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sumR, sumG, sumB, sumA float64
			for k := -radius; k <= radius; k++ {
				yy := y + k
				if yy < 0 {
					yy = 0
				} else if yy >= h {
					yy = h - 1
				}
				i := src.PixOffset(x, yy)
				wgt := weights[absInt(k)]
				sumR += float64(src.Pix[i]) * wgt
				sumG += float64(src.Pix[i+1]) * wgt
				sumB += float64(src.Pix[i+2]) * wgt
				sumA += float64(src.Pix[i+3]) * wgt
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i] = uint8(sumR + 0.5)
			dst.Pix[i+1] = uint8(sumG + 0.5)
			dst.Pix[i+2] = uint8(sumB + 0.5)
			dst.Pix[i+3] = uint8(sumA + 0.5)
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
