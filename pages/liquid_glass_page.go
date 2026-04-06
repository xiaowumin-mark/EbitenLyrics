package pages

import (
	"EbitenLyrics/comps/liquidglass"
	"EbitenLyrics/debugpanel"
	"EbitenLyrics/router"
	"fmt"
	"image"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type LiquidGlassTest struct {
	router.BaseScene

	glass  *liquidglass.Component
	params *liquidglass.Params

	debugPanel         *debugpanel.Panel
	debugInputCaptured bool

	bgPath   string
	bgMode   int
	lastTick time.Time
}

func (p *LiquidGlassTest) OnCreate() {
	glass, err := liquidglass.New()
	if err != nil {
		log.Printf("LiquidGlassTest init failed: %v", err)
		return
	}
	p.glass = glass
	p.params = glass.Params()
	p.lastTick = time.Now()

	p.loadBackground()
	p.setupDebugPanel()
}

func (p *LiquidGlassTest) OnEnter(params map[string]any) {
	p.lastTick = time.Now()
	if p.glass == nil {
		return
	}
	w, h := ebiten.WindowSize()
	p.glass.SetMouse(float64(w)*0.5, float64(h)*0.5)
	p.glass.ResetMouseSpringToMouse()
}

func (p *LiquidGlassTest) OnLeave() {}

func (p *LiquidGlassTest) OnDestroy() {
	if p.glass != nil {
		p.glass.Dispose()
		p.glass = nil
		p.params = nil
	}
}

func (p *LiquidGlassTest) OnResize(w, h int, isFirst bool) {
	if p.glass != nil {
		p.glass.Resize(w, h)
	}
}

func (p *LiquidGlassTest) Update() error {
	now := time.Now()
	dt := now.Sub(p.lastTick)
	p.lastTick = now

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		router.Go("home", nil)
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) && p.debugPanel != nil {
		p.debugPanel.Toggle()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && p.params != nil {
		p.params.FreezeFollow = !p.params.FreezeFollow
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		p.applyPresetDefault()
	}

	p.debugInputCaptured = false
	if p.debugPanel != nil {
		captured, err := p.debugPanel.Update()
		if err != nil {
			return err
		}
		p.debugInputCaptured = captured
	}

	if p.glass != nil {
		mx, my := ebiten.CursorPosition()
		p.glass.SetMouse(float64(mx), float64(my))
		p.glass.Update(dt)
	}
	return nil
}

func (p *LiquidGlassTest) Draw(screen *ebiten.Image) {
	if p.glass != nil {
		p.glass.Draw(screen)
	}
	ebitenutil.DebugPrintAt(screen, p.statusText(), 14, 14)
	if p.debugPanel != nil {
		p.debugPanel.Draw(screen)
	}
}

func (p *LiquidGlassTest) setupDebugPanel() {
	panel := debugpanel.New("液态玻璃组件", image.Rect(14, 14, 420, 660))

	panel.Group("基础", true).
		Select("背景", p.bgOptions, p.getBgOptionIndex, p.setBgOptionIndex).
		Float("模糊半径", &p.params.BlurRadius, 1, 20, 1, 0, func(v float64) {
			p.params.BlurRadius = v
		}).
		Bool("边缘强制模糊", &p.params.BlurEdge, func(v bool) {
			p.params.BlurEdge = v
		}).
		Float("色调 R", &p.params.TintR, 0, 255, 1, 0, func(v float64) { p.params.TintR = v }).
		Float("色调 G", &p.params.TintG, 0, 255, 1, 0, func(v float64) { p.params.TintG = v }).
		Float("色调 B", &p.params.TintB, 0, 255, 1, 0, func(v float64) { p.params.TintB = v }).
		Float("色调透明度", &p.params.TintA, 0, 1, 0.01, 2, func(v float64) { p.params.TintA = v }).
		Action("预设：默认", p.applyPresetDefault).
		Action("预设：镜面", p.applyPresetChrome).
		Action("预设：柔和", p.applyPresetSoft)

	panel.Group("折射", true).
		Float("厚度", &p.params.RefThickness, 1, 80, 0.1, 1, func(v float64) { p.params.RefThickness = v }).
		Float("折射因子", &p.params.RefFactor, 1, 4, 0.01, 2, func(v float64) { p.params.RefFactor = v }).
		Float("色散", &p.params.RefDispersion, 0, 50, 0.1, 1, func(v float64) { p.params.RefDispersion = v }).
		Float("菲涅尔范围", &p.params.RefFresnelRange, 1, 100, 0.1, 1, func(v float64) { p.params.RefFresnelRange = v }).
		Float("菲涅尔硬度", &p.params.RefFresnelHardness, 0, 1.2, 0.01, 2, func(v float64) { p.params.RefFresnelHardness = v }).
		Float("菲涅尔强度", &p.params.RefFresnelFactor, 0, 1.2, 0.01, 2, func(v float64) { p.params.RefFresnelFactor = v })

	panel.Group("眩光", true).
		Float("范围", &p.params.GlareRange, 1, 100, 0.1, 1, func(v float64) { p.params.GlareRange = v }).
		Float("硬度", &p.params.GlareHardness, 0, 1.2, 0.01, 2, func(v float64) { p.params.GlareHardness = v }).
		Float("强度", &p.params.GlareFactor, 0, 1.2, 0.01, 2, func(v float64) { p.params.GlareFactor = v }).
		Float("收敛", &p.params.GlareConvergence, 0, 1, 0.01, 2, func(v float64) { p.params.GlareConvergence = v }).
		Float("反侧系数", &p.params.GlareOppositeFactor, 0, 1, 0.01, 2, func(v float64) { p.params.GlareOppositeFactor = v }).
		Float("角度", &p.params.GlareAngleDeg, -180, 180, 1, 0, func(v float64) { p.params.GlareAngleDeg = v })

	panel.Group("形状", true).
		Bool("显示圆形", &p.params.ShowShape1, func(v bool) { p.params.ShowShape1 = v }).
		Float("宽度", &p.params.ShapeWidth, 20, 800, 1, 0, func(v float64) { p.params.ShapeWidth = v }).
		Float("高度", &p.params.ShapeHeight, 20, 800, 1, 0, func(v float64) { p.params.ShapeHeight = v }).
		Float("圆角百分比", &p.params.ShapeRadiusPct, 1, 100, 0.1, 1, func(v float64) { p.params.ShapeRadiusPct = v }).
		Float("超椭圆度", &p.params.ShapeRoundness, 2, 7, 0.01, 2, func(v float64) { p.params.ShapeRoundness = v }).
		Float("融合率", &p.params.MergeRate, 0, 0.3, 0.01, 2, func(v float64) { p.params.MergeRate = v })

	panel.Group("动画", false).
		Bool("冻结跟随", &p.params.FreezeFollow, func(v bool) { p.params.FreezeFollow = v }).
		Float("弹簧刚度 K", &p.params.SpringStiffness, 1, 120, 1, 0, func(v float64) { p.params.SpringStiffness = v }).
		Float("弹簧阻尼 D", &p.params.SpringDamping, 0, 30, 0.1, 1, func(v float64) { p.params.SpringDamping = v }).
		Float("速度拉伸系数", &p.params.SpringSizeFactor, 0, 50, 0.1, 1, func(v float64) { p.params.SpringSizeFactor = v })

	panel.Group("调试", false).
		Float("步骤视图", &p.params.Step, 0, 9, 1, 0, func(v float64) { p.params.Step = v }).
		Text("", func() string { return p.statusText() })

	p.debugPanel = panel
}

func (p *LiquidGlassTest) statusText() string {
	if p.params == nil {
		return "液态玻璃组件初始化失败"
	}
	mx, my := 0.0, 0.0
	sx, sy := 0.0, 0.0
	if p.glass != nil {
		mx, my = p.glass.Mouse()
		sx, sy = p.glass.MouseSpring()
	}
	return fmt.Sprintf(
		"液态玻璃组件演示（已抽取）\nEsc: 返回 | F1: 面板 | Space: 冻结跟随 | R: 默认预设\n背景: %s | 模糊:%0.0f | 步骤:%0.0f\n鼠标:(%.0f, %.0f) 弹簧:(%.0f, %.0f)",
		p.bgOptions()[p.getBgOptionIndex()],
		p.params.BlurRadius,
		p.params.Step,
		mx, my,
		sx, sy,
	)
}

func (p *LiquidGlassTest) applyPresetDefault() {
	if p.glass == nil {
		return
	}
	p.glass.SetParams(liquidglass.DefaultParams())
	p.params = p.glass.Params()
}

func (p *LiquidGlassTest) applyPresetChrome() {
	if p.glass == nil {
		return
	}
	p.glass.SetParams(liquidglass.ChromeParams())
	p.params = p.glass.Params()
}

func (p *LiquidGlassTest) applyPresetSoft() {
	if p.glass == nil {
		return
	}
	p.glass.SetParams(liquidglass.SoftParams())
	p.params = p.glass.Params()
}

func (p *LiquidGlassTest) bgOptions() []string {
	return []string{
		"照片",
		"网格",
		"条纹",
		"上下分色",
	}
}

func (p *LiquidGlassTest) getBgOptionIndex() int {
	return p.bgMode
}

func (p *LiquidGlassTest) setBgOptionIndex(index int) {
	if index < 0 || index >= len(p.bgOptions()) {
		return
	}
	p.bgMode = index
	if p.glass != nil {
		p.glass.SetBackgroundMode(liquidglass.BackgroundMode(index))
	}
}

func (p *LiquidGlassTest) loadBackground() {
	if p.glass == nil {
		return
	}
	candidates := []string{}
	if fromEnv := strings.TrimSpace(os.Getenv("EBITENLYRICS_LIQUID_BG")); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	candidates = append(candidates,
		"test-data/liquid-bg.jpg",
		"test-data/liquid-bg.jpeg",
		"test-data/liquid-bg.png",
	)
	path, err := p.glass.LoadBackgroundFromCandidates(candidates)
	if err != nil {
		log.Printf("LiquidGlassTest background fallback: %v", err)
		return
	}
	p.bgPath = path
	log.Printf("LiquidGlassTest background loaded: %s", path)
}
