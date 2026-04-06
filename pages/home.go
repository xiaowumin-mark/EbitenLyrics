package pages

// 文件说明：主页场景，负责歌词展示、背景联动、调试信息和运行时交互。
// 主要职责：接收事件、更新渲染组件、处理滚轮与字体配置热切换。

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/bgrender"
	LyricsComponent "EbitenLyrics/comps/lyrics"
	"EbitenLyrics/debugpanel"
	"EbitenLyrics/evbus"
	f "EbitenLyrics/font"
	"EbitenLyrics/lp"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/router"
	"EbitenLyrics/ttml"
	"EbitenLyrics/ws"
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var runtimeWeightSteps = []f.Weight{
	f.WeightLight,
	f.WeightRegular,
	f.WeightMedium,
	f.WeightSemiBold,
	f.WeightBold,
	f.WeightExtraBold,
}

type Home struct {
	router.BaseScene
	FontManager *f.FontManager
	FontRequest f.FontRequest

	LyricsImageAnim *anim.Tween
	AnimateManager  *anim.Manager

	LyricsControl *LyricsComponent.LyricsComponent
	Cover         *ebiten.Image
	CoverPosition lyrics.Position
	MeshRenderer  *bgrender.MeshGradientRenderer
	meshLastTick  time.Time

	FontSize           float64
	FD                 float64
	UserScale          float64
	SmartTranslateWrap bool

	eventsBound bool

	familyChoices []string
	familyIndex   int
	fontWeight    f.Weight
	fontItalic    bool
	currentFamily string
	fontConfig    string

	pendingMu             sync.Mutex
	hasPendingLyrics      bool
	pendingLyrics         []ttml.LyricLine
	hasPendingProgress    bool
	pendingProgress       time.Duration
	hasPendingCover       bool
	pendingCover          image.Image
	hasPendingLowFreq     bool
	pendingLowFreqVolume  float64
	hasPendingFontConfig  bool
	pendingFontConfig     map[string]any
	hasLatestProgress     bool
	latestProgress        time.Duration
	isUserScrolling       bool
	manualScrollOffset    float64
	manualScrollTarget    float64
	manualScrollResumeAt  time.Time
	manualScrollReturnAni *anim.Tween

	lastMemSampleAt   time.Time
	memSampleInterval time.Duration
	memPanel          string

	lastProgress       time.Duration
	DebugPanel         *debugpanel.Panel
	debugInputCaptured bool
}

type lyricsDebugStats struct {
	mainLines     int
	bgLines       int
	renderedLines int
	activeLines   int
	syllables     int
	elements      int

	totalImages      int
	approxImageBytes uint64
	coverImages      int
	lineImages       int
	translateImages  int
	textMaskImages   int
	gradientImages   int
	tempImages       int
	shadowImages     int
}

func dedupFamilies(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, family := range in {
		family = strings.TrimSpace(family)
		if family == "" {
			continue
		}
		key := strings.ToLower(family)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, family)
	}
	return out
}

func normalizeRuntimeWeight(weight f.Weight) f.Weight {
	if weight < f.WeightThin {
		return f.WeightThin
	}
	if weight > f.WeightBlack {
		return f.WeightBlack
	}
	rounded := int(math.Round(float64(weight)/100.0)) * 100
	return f.Weight(rounded)
}

func formatBytesIEC(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%dB", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(v)/float64(div), "KMGTPE"[exp])
}

func (h *Home) collectLyricsDebugStats() lyricsDebugStats {
	stats := lyricsDebugStats{}
	seenImages := map[*ebiten.Image]struct{}{}

	addImage := func(img *ebiten.Image) {
		if img == nil {
			return
		}
		if _, ok := seenImages[img]; ok {
			return
		}
		seenImages[img] = struct{}{}

		w, he := img.Bounds().Dx(), img.Bounds().Dy()
		if w <= 0 || he <= 0 {
			return
		}
		stats.totalImages++
		stats.approxImageBytes += uint64(w) * uint64(he) * 4
	}

	if h.Cover != nil {
		stats.coverImages = 1
		addImage(h.Cover)
	}

	if h.LyricsControl == nil || h.LyricsControl.LyricsControl == nil {
		return stats
	}
	control := h.LyricsControl.LyricsControl
	stats.renderedLines = len(control.GetRenderedindex())
	stats.activeLines = len(control.GetNowLyrics())

	var walkLine func(line *lyrics.Line, isBackground bool)
	walkLine = func(line *lyrics.Line, isBackground bool) {
		if line == nil {
			return
		}
		if isBackground {
			stats.bgLines++
		} else {
			stats.mainLines++
		}
		stats.syllables += len(line.Syllables)
		stats.elements += len(line.OuterSyllableElements)

		if line.Image != nil {
			stats.lineImages++
			addImage(line.Image)
		}
		if line.TranslateImage != nil {
			stats.translateImages++
			addImage(line.TranslateImage)
		}

		for _, element := range line.OuterSyllableElements {
			if element == nil {
				continue
			}
			if element.SyllableImage != nil {
				if element.SyllableImage.TextMask != nil {
					stats.textMaskImages++
					addImage(element.SyllableImage.TextMask)
				}
				if element.SyllableImage.GradientImage != nil {
					stats.gradientImages++
					addImage(element.SyllableImage.GradientImage)
				}
				if element.SyllableImage.GetTempImage() != nil {
					stats.tempImages++
					addImage(element.SyllableImage.GetTempImage())
				}
			}
			if element.BackgroundBlurText != nil {
				if element.BackgroundBlurText.OriginImage != nil {
					stats.shadowImages++
					addImage(element.BackgroundBlurText.OriginImage)
				}
				if element.BackgroundBlurText.Image != nil && element.BackgroundBlurText.Image != element.BackgroundBlurText.OriginImage {
					stats.shadowImages++
					addImage(element.BackgroundBlurText.Image)
				}
			}
		}

		for _, bgLine := range line.BackgroundLines {
			walkLine(bgLine, true)
		}
	}

	for _, line := range control.Lines {
		walkLine(line, false)
	}

	return stats
}

func (h *Home) updateMemoryPanel() {
	if h.memSampleInterval <= 0 {
		h.memSampleInterval = 500 * time.Millisecond
	}
	now := time.Now()
	if !h.lastMemSampleAt.IsZero() && now.Sub(h.lastMemSampleAt) < h.memSampleInterval {
		return
	}
	h.lastMemSampleAt = now

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	stats := h.collectLyricsDebugStats()

	h.memPanel = fmt.Sprintf(
		"Mem HeapAlloc:%s HeapInuse:%s HeapSys:%s NextGC:%s NumGC:%d Goroutines:%d\n"+
			"Image Total:%d Approx:%s Cover:%d Line:%d TS:%d Mask:%d Grad:%d Temp:%d Shadow:%d\n"+
			"Lyrics Main:%d BG:%d Rendered:%d Active:%d Syllables:%d Elements:%d",
		formatBytesIEC(mem.HeapAlloc),
		formatBytesIEC(mem.HeapInuse),
		formatBytesIEC(mem.HeapSys),
		formatBytesIEC(mem.NextGC),
		mem.NumGC,
		runtime.NumGoroutine(),
		stats.totalImages,
		formatBytesIEC(stats.approxImageBytes),
		stats.coverImages,
		stats.lineImages,
		stats.translateImages,
		stats.textMaskImages,
		stats.gradientImages,
		stats.tempImages,
		stats.shadowImages,
		stats.mainLines,
		stats.bgLines,
		stats.renderedLines,
		stats.activeLines,
		stats.syllables,
		stats.elements,
	)
}

func (h *Home) setFontSize(size float64) {
	h.FontSize = math.Max(8, size)
	if h.LyricsControl != nil {
		h.LyricsControl.SetFontSize(h.FontSize)
	}
}

func (h *Home) setFD(fd float64) {
	h.FD = math.Max(0.0001, math.Min(3, fd))
	if h.LyricsControl != nil {
		h.LyricsControl.SetFD(h.FD)
	}
}

func (h *Home) setUserScale(scale float64) {
	h.UserScale = math.Max(0.25, math.Min(4.0, scale))
	lp.SetUserScale(h.UserScale)
	if h.LyricsControl != nil && h.LyricsControl.LyricsControl != nil {
		for _, line := range h.LyricsControl.LyricsControl.Lines {
			if line == nil {
				continue
			}
			line.Dispose()
			line.Render()
		}
		w, he := ebiten.WindowSize()
		h.LyricsControl.Resize(lp.FromLP(float64(w)), lp.FromLP(float64(he)))
		if h.hasLatestProgress {
			h.LyricsControl.Update(h.latestProgress)
		}
	}
}

func (h *Home) initFamilyChoices() {
	choices := append([]string{}, f.DefaultFamilies()...)
	if h.currentFamily != "" {
		choices = append([]string{h.currentFamily}, choices...)
	}
	h.familyChoices = dedupFamilies(choices)
	if len(h.familyChoices) == 0 {
		h.familyChoices = []string{"Microsoft YaHei UI", "Noto Sans CJK SC", "Segoe UI"}
	}
	h.familyIndex = 0
}

func (h *Home) availableFamilyChoices() []string {
	choices := append([]string{}, h.familyChoices...)
	if h.FontManager != nil {
		choices = append(choices, h.FontManager.AvailableFamilies()...)
	}
	if h.currentFamily != "" {
		choices = append([]string{h.currentFamily}, choices...)
	}
	h.familyChoices = dedupFamilies(choices)
	if len(h.familyChoices) == 0 {
		h.familyChoices = []string{"Microsoft YaHei UI", "Noto Sans CJK SC", "Segoe UI"}
	}
	if h.familyIndex < 0 || h.familyIndex >= len(h.familyChoices) {
		h.familyIndex = 0
	}
	return h.familyChoices
}

func (h *Home) setCurrentFamilyChoice(family string) {
	family = strings.TrimSpace(family)
	if family == "" {
		return
	}
	for i, it := range h.familyChoices {
		if stringsEqualFoldTrim(it, family) {
			h.familyIndex = i
			return
		}
	}
	h.familyChoices = append([]string{family}, h.familyChoices...)
	h.familyChoices = dedupFamilies(h.familyChoices)
	h.familyIndex = 0
}

func (h *Home) currentFamilyChoice() string {
	if len(h.familyChoices) == 0 {
		return ""
	}
	if h.familyIndex < 0 || h.familyIndex >= len(h.familyChoices) {
		h.familyIndex = 0
	}
	return h.familyChoices[h.familyIndex]
}

func (h *Home) weightChoiceLabels() []string {
	out := make([]string, 0, len(runtimeWeightSteps))
	for _, weight := range runtimeWeightSteps {
		out = append(out, fmt.Sprintf("%d", weight))
	}
	return out
}

func (h *Home) currentWeightChoiceIndex() int {
	for i, weight := range runtimeWeightSteps {
		if weight == h.fontWeight {
			return i
		}
	}
	return 0
}

func (h *Home) setWeightChoiceIndex(index int) {
	if index < 0 || index >= len(runtimeWeightSteps) {
		return
	}
	h.fontWeight = runtimeWeightSteps[index]
	h.applyFontRequest(f.FontRequest{
		Families: []string{h.currentFamilyChoice()},
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	})
}

func (h *Home) setFamilyChoiceIndex(index int) {
	if index < 0 || index >= len(h.familyChoices) {
		return
	}
	h.familyIndex = index
	h.applyFontRequest(f.FontRequest{
		Families: []string{h.currentFamilyChoice()},
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	})
}

func (h *Home) applyFontRequest(req f.FontRequest) {
	req = req.Normalized()
	if h.FontRequest.CacheKey() == req.CacheKey() {
		return
	}

	resolved, err := h.FontManager.ResolveChain(req)
	if err != nil || resolved == nil || resolved.Primary == nil {
		log.Printf("resolve runtime font failed: %v", err)
		return
	}

	h.FontRequest = req
	h.currentFamily = resolved.Primary.Family
	h.fontWeight = normalizeRuntimeWeight(resolved.Primary.Weight)
	h.fontItalic = resolved.Primary.Italic
	h.setCurrentFamilyChoice(resolved.Primary.Family)
	if h.LyricsControl != nil {
		h.LyricsControl.SetFont(h.FontManager, req)
	}

	log.Printf(
		"runtime font applied: family=%q style=%q weight=%d path=%s",
		resolved.Primary.Family,
		resolved.Primary.Style,
		resolved.Primary.Weight,
		resolved.Primary.Path,
	)
}

func (h *Home) applyMapFontConfig(cfg map[string]any) {
	baseFamilies := []string{h.currentFamilyChoice()}
	if baseFamilies[0] == "" {
		baseFamilies = f.DefaultFamilies()
	}
	base := f.FontRequest{
		Families: baseFamilies,
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	}

	req, err := h.FontManager.ParseRequest(base, cfg)
	if err != nil {
		log.Printf("invalid font config: %v", err)
		return
	}

	h.fontWeight = req.Weight
	h.fontItalic = req.Italic
	if len(req.Families) > 0 {
		h.setCurrentFamilyChoice(req.Families[0])
	}

	h.applyFontRequest(req)
}

func stringsEqualFoldTrim(a, b string) bool {
	return strings.ToLower(strings.TrimSpace(a)) == strings.ToLower(strings.TrimSpace(b))
}

func (h *Home) cycleFamily(step int) {
	if len(h.familyChoices) == 0 {
		return
	}
	h.familyIndex = (h.familyIndex + step + len(h.familyChoices)) % len(h.familyChoices)

	h.applyFontRequest(f.FontRequest{
		Families: []string{h.currentFamilyChoice()},
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	})
}

func (h *Home) cycleWeight(step int) {
	if len(runtimeWeightSteps) == 0 {
		return
	}
	idx := 0
	for i, w := range runtimeWeightSteps {
		if w == h.fontWeight {
			idx = i
			break
		}
	}
	idx = (idx + step + len(runtimeWeightSteps)) % len(runtimeWeightSteps)
	h.fontWeight = runtimeWeightSteps[idx]

	h.applyFontRequest(f.FontRequest{
		Families: []string{h.currentFamilyChoice()},
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	})
}

func (h *Home) toggleItalic() {
	h.fontItalic = !h.fontItalic
	h.applyFontRequest(f.FontRequest{
		Families: []string{h.currentFamilyChoice()},
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	})
}

func (h *Home) loadAndApplyFontConfig(path string) {
	baseFamilies := []string{h.currentFamilyChoice()}
	if baseFamilies[0] == "" {
		baseFamilies = f.DefaultFamilies()
	}

	base := f.FontRequest{
		Families: baseFamilies,
		Weight:   h.fontWeight,
		Italic:   h.fontItalic,
	}
	req, err := h.FontManager.LoadRequestFromFile(path, base)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("load runtime font config failed: %v", err)
		}
		return
	}
	h.fontWeight = req.Weight
	h.fontItalic = req.Italic
	if len(req.Families) > 0 {
		h.setCurrentFamilyChoice(req.Families[0])
	}
	h.applyFontRequest(req)
}

func (h *Home) debugSummaryText() string {
	lines := []string{
		fmt.Sprintf("字体: %s", h.currentFamily),
		fmt.Sprintf("字重: %d", h.fontWeight),
		fmt.Sprintf("斜体: %v", h.fontItalic),
		fmt.Sprintf("用户滚动: %v", h.isUserScrolling),
		fmt.Sprintf("低频音量: %.2f", h.pendingLowFreqVolume),
		"快捷键: Esc 显示/隐藏面板, F2 液态玻璃测试, F5/F6 切字体, F7/F8 切字重, F9 切斜体, F10 重载字体配置, F11 全屏",
	}
	return strings.Join(lines, "\n")
}

func (h *Home) runtimeStatusText() string {
	listening, connections := ws.StatusSnapshot()
	wsStatus := "未启动"
	switch {
	case listening && connections > 0:
		wsStatus = fmt.Sprintf("已连接 (%d)", connections)
	case listening:
		wsStatus = "监听中 端口:11445"
	}

	return fmt.Sprintf(
		"WebSocket: %s\nTPS: %.2f\nFPS: %.2f",
		wsStatus,
		ebiten.ActualTPS(),
		ebiten.ActualFPS(),
	)
}

func (h *Home) setupDebugPanel() {
	panel := debugpanel.New(
		"首页调试",
		image.Rect(lp.LPInt(18), lp.LPInt(18), lp.LPInt(360), lp.LPInt(560)),
	)

	panel.Group("运行状态", true).
		Description("实时显示连接状态与渲染帧率。").
		Text("", func() string {
			return h.runtimeStatusText()
		})

	panel.Group("歌词", true).
		Description("运行时调整歌词布局与动画参数。").
		Float("字体大小", &h.FontSize, 8, 120, 1, 0, func(value float64) {
			h.setFontSize(value)
		}).
		Float("渐变宽度比", &h.FD, 0.0001, 3, 0.01, 2, func(value float64) {
			h.setFD(value)
		}).
		Bool("智能翻译换行", &h.SmartTranslateWrap, func(value bool) {
			h.setSmartTranslateWrap(value)
		})

	panel.Group("字体", true).
		Select("字体族", func() []string {
			return h.availableFamilyChoices()
		}, func() int {
			return h.familyIndex
		}, func(index int) {
			h.setFamilyChoiceIndex(index)
		}).
		Select("字重", func() []string {
			return h.weightChoiceLabels()
		}, func() int {
			return h.currentWeightChoiceIndex()
		}, func(index int) {
			h.setWeightChoiceIndex(index)
		}).
		Bool("斜体", &h.fontItalic, func(value bool) {
			h.applyFontRequest(f.FontRequest{
				Families: []string{h.currentFamilyChoice()},
				Weight:   h.fontWeight,
				Italic:   value,
			})
		}).
		Action("重载字体配置", func() {
			h.loadAndApplyFontConfig(h.fontConfig)
		})

	panel.Group("窗口", false).
		Float("用户缩放", &h.UserScale, 0.25, 4.0, 0.05, 2, func(value float64) {
			h.setUserScale(value)
		}).
		Action("切换全屏", func() {
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
		})

	panel.Group("统计", true).
		Text("", func() string {
			return h.debugSummaryText()
		}).
		Text("内存", func() string {
			return h.memPanel
		})

	h.DebugPanel = panel
}

func (h *Home) queueLyrics(lines []ttml.LyricLine) {
	h.pendingMu.Lock()
	h.hasPendingLyrics = true
	h.pendingLyrics = lines
	h.pendingMu.Unlock()
}

func (h *Home) setSmartTranslateWrap(enabled bool) {
	h.SmartTranslateWrap = enabled
	if h.LyricsControl != nil {
		h.LyricsControl.SetSmartTranslateWrap(enabled)
	}
}

func (h *Home) queueProgress(progress time.Duration) {
	h.pendingMu.Lock()
	h.hasPendingProgress = true
	h.pendingProgress = progress
	h.latestProgress = progress
	h.hasLatestProgress = true
	h.pendingMu.Unlock()
}

func (h *Home) queueCover(img image.Image) {
	h.pendingMu.Lock()
	h.hasPendingCover = true
	h.pendingCover = img
	h.pendingMu.Unlock()
}

func (h *Home) queueLowFreqVolume(volume float64) {
	h.pendingMu.Lock()
	h.hasPendingLowFreq = true
	h.pendingLowFreqVolume = volume
	h.pendingMu.Unlock()
}

func cloneMapStringAny(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func (h *Home) queueFontConfig(cfg map[string]any) {
	h.pendingMu.Lock()
	h.hasPendingFontConfig = true
	h.pendingFontConfig = cloneMapStringAny(cfg)
	h.pendingMu.Unlock()
}

func (h *Home) applyPendingEvents() {
	var (
		hasLyrics      bool
		lyricsLines    []ttml.LyricLine
		hasProgress    bool
		progress       time.Duration
		hasCover       bool
		coverImage     image.Image
		hasLowFreq     bool
		lowFreqVolume  float64
		hasFontConfig  bool
		fontConfigData map[string]any
	)

	h.pendingMu.Lock()
	if h.hasPendingLyrics {
		hasLyrics = true
		lyricsLines = h.pendingLyrics
		h.hasPendingLyrics = false
		h.pendingLyrics = nil
	}
	if h.hasPendingProgress {
		hasProgress = true
		progress = h.pendingProgress
		h.hasPendingProgress = false
	}
	if h.hasPendingCover {
		hasCover = true
		coverImage = h.pendingCover
		h.hasPendingCover = false
		h.pendingCover = nil
	}
	if h.hasPendingLowFreq {
		hasLowFreq = true
		lowFreqVolume = h.pendingLowFreqVolume
		h.hasPendingLowFreq = false
	}
	if h.hasPendingFontConfig {
		hasFontConfig = true
		fontConfigData = h.pendingFontConfig
		h.hasPendingFontConfig = false
		h.pendingFontConfig = nil
	}
	h.pendingMu.Unlock()

	if hasLyrics && h.LyricsControl != nil {
		h.LyricsControl.SetLyrics(lyricsLines)
		if h.hasLatestProgress && !h.isUserScrolling {
			h.LyricsControl.Update(h.latestProgress)
		}
	}

	if hasCover && coverImage != nil {
		if h.Cover != nil {
			h.Cover.Dispose()
			h.Cover = nil
		}
		h.Cover = ebiten.NewImageFromImage(coverImage)
		h.CoverPosition.W = lp.FromLP(float64(h.Cover.Bounds().Dx()))
		h.CoverPosition.H = lp.FromLP(float64(h.Cover.Bounds().Dy()))
		h.CoverPosition.OriginX = h.CoverPosition.W / 2
		h.CoverPosition.OriginY = h.CoverPosition.H / 2
		w, he := ebiten.WindowSize()
		h.updateCoverTransform(lp.FromLP(float64(w)), lp.FromLP(float64(he)))
		if h.MeshRenderer != nil {
			if err := h.MeshRenderer.SetAlbum(coverImage); err != nil {
				log.Printf("mesh renderer set album failed: %v", err)
			}
			log.Printf("mesh renderer set album success")
		}
	}

	if hasFontConfig {
		h.applyMapFontConfig(fontConfigData)
	}

	if hasLowFreq && h.MeshRenderer != nil {
		h.MeshRenderer.SetLowFreqVolume(lowFreqVolume)
	}

	if hasProgress && h.LyricsControl != nil && !h.isUserScrolling {
		h.LyricsControl.Update(progress)
	}
}

func (h *Home) beginUserScroll() {
	h.isUserScrolling = true
	h.manualScrollResumeAt = time.Now().Add(time.Second)
	if h.manualScrollReturnAni != nil {
		h.manualScrollReturnAni.Cancel()
		h.manualScrollReturnAni = nil
	}
}

func (h *Home) clampScrollTarget() {
	_, wh := ebiten.WindowSize()
	maxAbs := math.Max(200, lp.FromLP(float64(wh))*2.5)
	if h.manualScrollTarget > maxAbs {
		h.manualScrollTarget = maxAbs
	}
	if h.manualScrollTarget < -maxAbs {
		h.manualScrollTarget = -maxAbs
	}
}

func (h *Home) handleWheelScroll() {
	_, wy := ebiten.Wheel()
	if wy != 0 {
		h.beginUserScroll()
		h.manualScrollTarget += -wy * math.Max(24, h.FontSize*0.85)
		h.clampScrollTarget()
	}

	if h.isUserScrolling {
		h.manualScrollOffset += (h.manualScrollTarget - h.manualScrollOffset) * 0.23
		if math.Abs(h.manualScrollTarget-h.manualScrollOffset) < 0.2 {
			h.manualScrollOffset = h.manualScrollTarget
		}

		if time.Now().After(h.manualScrollResumeAt) {
			h.isUserScrolling = false
			from := h.manualScrollOffset
			if h.manualScrollReturnAni != nil {
				h.manualScrollReturnAni.Cancel()
				h.manualScrollReturnAni = nil
			}
			h.manualScrollReturnAni = anim.NewTween(
				"home-manual-scroll-return",
				380*time.Millisecond,
				0,
				1,
				from,
				0,
				anim.EaseOut,
				func(value float64) {
					h.manualScrollOffset = value
					h.manualScrollTarget = value
				},
				func() {
					h.manualScrollOffset = 0
					h.manualScrollTarget = 0
					h.manualScrollReturnAni = nil
				},
			)
			if h.AnimateManager != nil {
				h.AnimateManager.Add(h.manualScrollReturnAni)
			}
			if h.hasLatestProgress && h.LyricsControl != nil {
				h.LyricsControl.Update(h.latestProgress)
			}
		}
	}
}

func (h *Home) bindEvents() {
	if h.eventsBound {
		return
	}
	h.eventsBound = true

	evbus.Bus.Subscribe("ws:setLyric", func(value []interface{}) {
		d, err := ws.ParseLyricsFromMap(value)
		if err != nil {
			log.Printf("parse lyric failed: %v", err)
			return
		}
		h.queueLyrics(d)
	})

	evbus.Bus.Subscribe("ws:progress", func(value float64) {
		newProgress := time.Duration(value) * time.Millisecond

		// 1. 计算当前收到的进度与上次进度的差值
		diff := newProgress - h.lastProgress

		// --- 处理向后跳动的情况 ---
		if diff < 0 {
			// 如果向后跳动的距离很小（比如在 500ms 以内），我们认为是抖动，直接忽略
			// 注意：-500ms < diff < 0
			if diff > -500*time.Millisecond {
				return
			}
			// 如果向后跳动的距离很大（比如超过 500ms），我们认为是用户手动“回拖”了进度条
			// 这时候我们需要强制更新，否则进度条会卡住
		}

		// --- 处理向前跳动过大的情况 (可选) ---
		// 如果你发现向前也会偶尔闪跳，也可以加一个上限判断，但通常向前跳动是正常的

		// 更新状态并执行回调
		h.lastProgress = newProgress
		h.queueProgress(newProgress)
	})

	evbus.Bus.Subscribe("ws:fontConfig", func(value map[string]any) {
		h.queueFontConfig(value)
	})

	evbus.Bus.Subscribe("ws:cover", func(img image.Image) {
		if img == nil {
			return
		}
		h.queueCover(img)
	})

	evbus.Bus.Subscribe("ws:lowFreqVolume", func(value float64) {
		h.queueLowFreqVolume(value)
	})
}

func (h *Home) updateCoverTransform(w, he float64) {
	if h.CoverPosition.W <= 0 || h.CoverPosition.H <= 0 {
		return
	}
	if w <= 0 || he <= 0 {
		return
	}

	h.CoverPosition.TranslateX = (w - h.CoverPosition.W) / 2
	h.CoverPosition.TranslateY = (he - h.CoverPosition.H) / 2

	scaleX := w / h.CoverPosition.W
	scaleY := he / h.CoverPosition.H
	finalS := math.Max(scaleX, scaleY) * 1.4
	h.CoverPosition.ScaleX = finalS
	h.CoverPosition.ScaleY = finalS
}

func (h *Home) OnCreate() {
	log.Println("Home OnCreate")
	if h.FontManager == nil {
		h.FontManager = f.DefaultManager()
	}
	if len(h.FontRequest.Families) == 0 {
		h.FontRequest = f.DefaultRequest()
	}
	ww, hh := ebiten.WindowSize()
	h.FontSize = 50
	h.FD = 0.5
	h.UserScale = lp.UserScale()
	h.SmartTranslateWrap = true
	h.fontWeight = h.FontRequest.Weight
	h.fontItalic = h.FontRequest.Italic
	h.currentFamily = ""
	h.fontConfig = strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_CONFIG"))
	if h.fontConfig == "" {
		h.fontConfig = f.DefaultFontConfigPath
	}
	if resolved, err := h.FontManager.ResolveChain(h.FontRequest); err == nil && resolved != nil && resolved.Primary != nil {
		h.currentFamily = resolved.Primary.Family
		h.fontWeight = normalizeRuntimeWeight(resolved.Primary.Weight)
		h.fontItalic = resolved.Primary.Italic
	}
	h.initFamilyChoices()

	h.LyricsControl = LyricsComponent.NewLyricsComponent(
		h.AnimateManager,
		h.FontManager,
		h.FontRequest,
		lp.FromLP(float64(ww)),
		lp.FromLP(float64(hh)),
		h.FontSize,
		h.FD,
	)
	h.LyricsControl.Init()
	h.LyricsControl.SetSmartTranslateWrap(h.SmartTranslateWrap)
	meshRenderer, err := bgrender.NewMeshGradientRenderer(ww, hh)
	if err != nil {
		log.Printf("create mesh renderer failed: %v", err)
	} else {
		h.MeshRenderer = meshRenderer
	}
	h.CoverPosition = lyrics.NewPosition(0, 0, 0, 0)
	h.memSampleInterval = 500 * time.Millisecond
	h.updateMemoryPanel()
	h.setupDebugPanel()
	h.bindEvents()

	h.loadAndApplyFontConfig(h.fontConfig)
}

func (h *Home) OnEnter(params map[string]any) {
	log.Println("Home OnEnter", params)
	if h.FontManager == nil {
		log.Println("Home page font manager is nil")
	}
	h.meshLastTick = time.Now()
}

func (h *Home) OnLeave() {
	log.Println("Home OnLeave")
}

func (h *Home) OnDestroy() {
	log.Println("Home OnDestroy")
	if h.LyricsControl != nil {
		h.LyricsControl.Dispose()
		h.LyricsControl = nil
	}
	if h.Cover != nil {
		h.Cover.Dispose()
		h.Cover = nil
	}
	if h.MeshRenderer != nil {
		h.MeshRenderer.Dispose()
		h.MeshRenderer = nil
	}
}

func (h *Home) Update() error {
	now := time.Now()
	if h.meshLastTick.IsZero() {
		h.meshLastTick = now
	}
	dt := now.Sub(h.meshLastTick)
	h.meshLastTick = now

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && h.DebugPanel != nil {
		h.DebugPanel.Toggle()
	}

	h.applyPendingEvents()
	h.debugInputCaptured = false
	if h.DebugPanel != nil {
		captured, err := h.DebugPanel.Update()
		if err != nil {
			return err
		}
		h.debugInputCaptured = captured
	}
	if !h.debugInputCaptured {
		h.handleWheelScroll()
	}
	if h.MeshRenderer != nil {
		h.MeshRenderer.Update(dt)
	}

	h.CoverPosition.Rotate += 0.5
	if h.CoverPosition.Rotate > 360 {
		h.CoverPosition.Rotate = 0
	}

	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		h.setFontSize(h.FontSize + 4)
		log.Println("FontSize:", h.FontSize)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		h.setFontSize(h.FontSize - 4)
		log.Println("FontSize:", h.FontSize)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		h.setFD(h.FD + 0.1)
		log.Println("FD:", h.FD)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		h.setFD(h.FD - 0.1)
		log.Println("FD:", h.FD)
	}

	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		h.cycleFamily(1)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		h.cycleFamily(-1)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		h.cycleWeight(1)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		h.cycleWeight(-1)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		h.toggleItalic()
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		h.loadAndApplyFontConfig(h.fontConfig)
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if !h.debugInputCaptured && inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		router.Go("liquid_glass_test", nil)
		return nil
	}

	h.updateMemoryPanel()

	return nil
}

func (h *Home) Draw(screen *ebiten.Image) {

	if h.MeshRenderer != nil && h.MeshRenderer.HasRenderableState() {
		h.MeshRenderer.Draw(screen)
	} else if h.Cover != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM = lyrics.TransformToGeoM(&h.CoverPosition)
		screen.DrawImage(h.Cover, op)
	}
	if h.LyricsControl != nil {
		pos := lyrics.NewPosition(0, 0, 0, 0)
		pos.SetTranslateY(h.manualScrollOffset)
		h.LyricsControl.Draw(screen, &pos)
	}
	if h.DebugPanel != nil {
		h.DebugPanel.Draw(screen)
	}
}

func (h *Home) OnResize(w, he int, isFirst bool) {
	log.Println("Home OnResize", w, he, isFirst)
	if !isFirst && h.LyricsControl != nil {
		h.LyricsControl.Resize(lp.FromLP(float64(w)), lp.FromLP(float64(he)))
	}
	if h.MeshRenderer != nil {
		h.MeshRenderer.Resize(w, he)
	}
	h.updateCoverTransform(lp.FromLP(float64(w)), lp.FromLP(float64(he)))
}
