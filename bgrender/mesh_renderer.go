package bgrender

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
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
		shaderUniforms: map[string]any{
			"Time":   float32(0),
			"Volume": float32(0),
			"Alpha":  float32(1),
		},
	}
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
	r.volume = volume / 10.0
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

	lerp := math.Min(1, delta.Seconds()*1000/100)
	r.smoothedVolume += (r.volume - r.smoothedVolume) * lerp
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
	r.shaderUniforms["Volume"] = float32(r.smoothedVolume)
	r.shaderUniforms["Alpha"] = float32(1)

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

		// contrast 0.4
		r = (r-128)*0.4 + 128
		g = (g-128)*0.4 + 128
		b = (b-128)*0.4 + 128

		// saturate 3.0
		gray := r*0.3 + g*0.59 + b*0.11
		r = gray*-2.0 + r*3.0
		g = gray*-2.0 + g*3.0
		b = gray*-2.0 + b*3.0

		// contrast 1.7
		r = (r-128)*1.7 + 128
		g = (g-128)*1.7 + 128
		b = (b-128)*1.7 + 128

		// brightness 0.75
		r *= 0.75
		g *= 0.75
		b *= 0.75

		pixels[i] = floatToByte(r)
		pixels[i+1] = floatToByte(g)
		pixels[i+2] = floatToByte(b)
	}

	blurNRGBA(nrgba, 2, 4)
	return ebiten.NewImageFromImage(nrgba)
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

func blurNRGBA(img *image.NRGBA, radius, quality int) {
	if img == nil || radius <= 0 || quality <= 0 {
		return
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	copy(src.Pix, img.Pix)

	for q := 0; q < quality; q++ {
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				sumR, sumG, sumB, sumA := 0, 0, 0, 0
				count := 0
				for ky := -radius; ky <= radius; ky++ {
					yy := y + ky
					if yy < 0 {
						yy = 0
					} else if yy >= h {
						yy = h - 1
					}
					for kx := -radius; kx <= radius; kx++ {
						xx := x + kx
						if xx < 0 {
							xx = 0
						} else if xx >= w {
							xx = w - 1
						}
						i := (yy*w + xx) * 4
						sumR += int(src.Pix[i])
						sumG += int(src.Pix[i+1])
						sumB += int(src.Pix[i+2])
						sumA += int(src.Pix[i+3])
						count++
					}
				}
				if count == 0 {
					continue
				}
				i := (y*w + x) * 4
				dst.Pix[i] = uint8(sumR / count)
				dst.Pix[i+1] = uint8(sumG / count)
				dst.Pix[i+2] = uint8(sumB / count)
				dst.Pix[i+3] = uint8(sumA / count)
			}
		}
		src = dst
	}
	copy(img.Pix, src.Pix)
}
