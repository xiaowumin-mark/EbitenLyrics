package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/bgrender"
	LyricsComponent "EbitenLyrics/comps/lyrics"
	"EbitenLyrics/evbus"
	f "EbitenLyrics/font"
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
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	Font          *text.GoTextFaceSource
	FontFallbacks []*text.GoTextFaceSource

	LyricsImageAnim *anim.Tween
	AnimateManager  *anim.Manager

	LyricsControl *LyricsComponent.LyricsComponent
	Cover         *ebiten.Image
	CoverPosition lyrics.Position
	MeshRenderer  *bgrender.MeshGradientRenderer
	meshLastTick  time.Time

	FontSize float64
	FD       float64

	eventsBound bool

	familyChoices []string
	familyIndex   int
	fontWeight    f.Weight
	fontItalic    bool
	requireCJK    bool
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

	lastProgress   time.Duration
	showDebugStats bool
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

func sameFontSources(a, b []*text.GoTextFaceSource) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func (h *Home) applyFontOptions(opts f.ResolveOptions) {
	resolved, err := f.ResolveFaceSource(opts)
	if err != nil {
		log.Printf("resolve runtime font failed: %v", err)
		return
	}

	if h.Font == resolved.Source && h.currentFamily == resolved.Family && sameFontSources(h.FontFallbacks, resolved.Fallbacks) {
		return
	}

	h.Font = resolved.Source
	h.FontFallbacks = append([]*text.GoTextFaceSource{}, resolved.Fallbacks...)
	h.currentFamily = resolved.Family
	h.fontWeight = normalizeRuntimeWeight(resolved.Weight)
	h.fontItalic = strings.Contains(strings.ToLower(resolved.Style), "italic")
	h.setCurrentFamilyChoice(resolved.Family)
	if h.LyricsControl != nil {
		h.LyricsControl.SetFont(resolved.Source, resolved.Fallbacks)
	}

	log.Printf(
		"runtime font applied: family=%q style=%q weight=%d path=%s",
		resolved.Family,
		resolved.Style,
		resolved.Weight,
		resolved.Path,
	)
}

func (h *Home) applyMapFontConfig(cfg map[string]any) {
	baseFamilies := []string{h.currentFamilyChoice()}
	if baseFamilies[0] == "" {
		baseFamilies = f.DefaultFamilies()
	}
	base := f.ResolveOptions{
		Families:   baseFamilies,
		Weight:     h.fontWeight,
		Italic:     h.fontItalic,
		RequireCJK: h.requireCJK,
	}

	opts, err := f.ParseResolveOptions(base, cfg)
	if err != nil {
		log.Printf("invalid font config: %v", err)
		return
	}

	h.fontWeight = opts.Weight
	h.fontItalic = opts.Italic
	h.requireCJK = opts.RequireCJK

	if len(opts.Families) > 0 {
		h.setCurrentFamilyChoice(opts.Families[0])
	}

	h.applyFontOptions(opts)
}

func stringsEqualFoldTrim(a, b string) bool {
	return strings.ToLower(strings.TrimSpace(a)) == strings.ToLower(strings.TrimSpace(b))
}

func (h *Home) cycleFamily(step int) {
	if len(h.familyChoices) == 0 {
		return
	}
	h.familyIndex = (h.familyIndex + step + len(h.familyChoices)) % len(h.familyChoices)

	opts := f.ResolveOptions{
		Families:   []string{h.currentFamilyChoice()},
		Weight:     h.fontWeight,
		Italic:     h.fontItalic,
		RequireCJK: h.requireCJK,
	}
	h.applyFontOptions(opts)
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

	opts := f.ResolveOptions{
		Families:   []string{h.currentFamilyChoice()},
		Weight:     h.fontWeight,
		Italic:     h.fontItalic,
		RequireCJK: h.requireCJK,
	}
	h.applyFontOptions(opts)
}

func (h *Home) toggleItalic() {
	h.fontItalic = !h.fontItalic
	opts := f.ResolveOptions{
		Families:   []string{h.currentFamilyChoice()},
		Weight:     h.fontWeight,
		Italic:     h.fontItalic,
		RequireCJK: h.requireCJK,
	}
	h.applyFontOptions(opts)
}

func (h *Home) loadAndApplyFontConfig(path string) {
	baseFamilies := []string{h.currentFamilyChoice()}
	if baseFamilies[0] == "" {
		baseFamilies = f.DefaultFamilies()
	}

	base := f.ResolveOptions{
		Families:   baseFamilies,
		Weight:     h.fontWeight,
		Italic:     h.fontItalic,
		RequireCJK: h.requireCJK,
	}
	opts, err := f.LoadResolveOptionsFromFile(path, base)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("load runtime font config failed: %v", err)
		}
		return
	}
	h.fontWeight = opts.Weight
	h.fontItalic = opts.Italic
	h.requireCJK = opts.RequireCJK
	if len(opts.Families) > 0 {
		h.setCurrentFamilyChoice(opts.Families[0])
	}
	h.applyFontOptions(opts)
}

func (h *Home) queueLyrics(lines []ttml.LyricLine) {
	h.pendingMu.Lock()
	h.hasPendingLyrics = true
	h.pendingLyrics = lines
	h.pendingMu.Unlock()
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
		h.CoverPosition.W = float64(h.Cover.Bounds().Dx())
		h.CoverPosition.H = float64(h.Cover.Bounds().Dy())
		h.CoverPosition.OriginX = h.CoverPosition.W / 2
		h.CoverPosition.OriginY = h.CoverPosition.H / 2
		w, he := ebiten.WindowSize()
		h.updateCoverTransform(w, he)
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
	maxAbs := math.Max(200, float64(wh)*2.5)
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

func (h *Home) updateCoverTransform(w, he int) {
	if h.CoverPosition.W <= 0 || h.CoverPosition.H <= 0 {
		return
	}
	if w <= 0 || he <= 0 {
		return
	}

	h.CoverPosition.TranslateX = (float64(w) - h.CoverPosition.W) / 2
	h.CoverPosition.TranslateY = (float64(he) - h.CoverPosition.H) / 2

	scaleX := float64(w) / h.CoverPosition.W
	scaleY := float64(he) / h.CoverPosition.H
	finalS := math.Max(scaleX, scaleY) * 1.4
	h.CoverPosition.ScaleX = finalS
	h.CoverPosition.ScaleY = finalS
}

func (h *Home) OnCreate() {
	log.Println("Home OnCreate")
	h.showDebugStats = true
	ww, hh := ebiten.WindowSize()
	h.FontSize = 50
	h.FD = 0.5
	h.requireCJK = true
	h.fontWeight = f.WeightMedium
	h.fontItalic = false
	h.currentFamily = ""
	h.fontConfig = strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_CONFIG"))
	if h.fontConfig == "" {
		h.fontConfig = f.DefaultRuntimeFontConfigPath
	}
	if h.Font != nil {
		meta := h.Font.Metadata()
		h.currentFamily = meta.Family
		h.fontWeight = normalizeRuntimeWeight(f.Weight(meta.Weight))
		h.fontItalic = meta.Style == text.StyleItalic
	}
	h.initFamilyChoices()

	h.LyricsControl = LyricsComponent.NewLyricsComponent(h.AnimateManager, h.Font, h.FontFallbacks, float64(ww), float64(hh), h.FontSize, h.FD)
	h.LyricsControl.Init()
	meshRenderer, err := bgrender.NewMeshGradientRenderer(ww, hh)
	if err != nil {
		log.Printf("create mesh renderer failed: %v", err)
	} else {
		h.MeshRenderer = meshRenderer
	}
	h.CoverPosition = lyrics.NewPosition(0, 0, 0, 0)
	h.memSampleInterval = 500 * time.Millisecond
	h.updateMemoryPanel()
	h.bindEvents()

	h.loadAndApplyFontConfig(h.fontConfig)
}

func (h *Home) OnEnter(params map[string]any) {
	log.Println("Home OnEnter", params)
	if h.Font == nil {
		log.Println("Home page font is nil")
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

	h.applyPendingEvents()
	h.handleWheelScroll()
	if h.MeshRenderer != nil {
		h.MeshRenderer.Update(dt)
	}

	h.CoverPosition.Rotate += 0.5
	if h.CoverPosition.Rotate > 360 {
		h.CoverPosition.Rotate = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		h.FontSize = math.Max(8, h.FontSize+4)
		h.LyricsControl.SetFontSize(h.FontSize)
		log.Println("FontSize:", h.FontSize)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		h.FontSize = math.Max(8, h.FontSize-4)
		h.LyricsControl.SetFontSize(h.FontSize)
		log.Println("FontSize:", h.FontSize)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		h.FD = math.Min(3, h.FD+0.1)
		h.LyricsControl.SetFD(h.FD)
		log.Println("FD:", h.FD)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		h.FD = math.Max(0.0001, h.FD-0.1)
		h.LyricsControl.SetFD(h.FD)
		log.Println("FD:", h.FD)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		h.cycleFamily(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		h.cycleFamily(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		h.cycleWeight(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		h.cycleWeight(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		h.toggleItalic()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		h.loadAndApplyFontConfig(h.fontConfig)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		h.showDebugStats = !h.showDebugStats
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
	if h.showDebugStats {
		return
	}
	msg := fmt.Sprintf(
		"Home Scene\nFS:%.0f FD:%.2f\nFamily:%s\nWeight:%d Italic:%v\nScroll:%v\nF5/F6 family, F7/F8 weight, F9 italic, F10 reload config\nLowFreqVolume: %.2f",
		h.FontSize,
		h.FD,
		h.currentFamily,
		h.fontWeight,
		h.fontItalic,
		h.isUserScrolling,
		h.pendingLowFreqVolume,
	)
	if h.memPanel != "" {
		msg += "\n" + h.memPanel
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (h *Home) OnResize(w, he int, isFirst bool) {
	log.Println("Home OnResize", w, he, isFirst)
	if !isFirst && h.LyricsControl != nil {
		h.LyricsControl.Resize(float64(w), float64(he))
	}
	if h.MeshRenderer != nil {
		h.MeshRenderer.Resize(w, he)
	}
	h.updateCoverTransform(w, he)
}
