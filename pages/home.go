package pages

import (
	"EbitenLyrics/anim"
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
	Font *text.GoTextFaceSource

	LyricsImageAnim *anim.Tween
	AnimateManager  *anim.Manager

	LyricsControl *LyricsComponent.LyricsComponent
	Cover         *ebiten.Image
	CoverPosition lyrics.Position

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
	hasPendingFontConfig  bool
	pendingFontConfig     map[string]any
	hasLatestProgress     bool
	latestProgress        time.Duration
	isUserScrolling       bool
	manualScrollOffset    float64
	manualScrollTarget    float64
	manualScrollResumeAt  time.Time
	manualScrollReturnAni *anim.Tween
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

	if h.Font == resolved.Source && h.currentFamily == resolved.Family {
		return
	}

	h.Font = resolved.Source
	h.currentFamily = resolved.Family
	h.fontWeight = normalizeRuntimeWeight(resolved.Weight)
	h.fontItalic = resolved.Style == "italic"
	h.setCurrentFamilyChoice(resolved.Family)
	if h.LyricsControl != nil {
		h.LyricsControl.SetFont(resolved.Source)
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
		hasFontConfig  bool
		fontConfigData map[string]any
	)

	h.pendingMu.Lock()
	if h.hasPendingLyrics {
		hasLyrics = true
		lyricsLines = h.pendingLyrics
		h.hasPendingLyrics = false
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
	}
	if h.hasPendingFontConfig {
		hasFontConfig = true
		fontConfigData = h.pendingFontConfig
		h.hasPendingFontConfig = false
	}
	h.pendingMu.Unlock()

	if hasLyrics && h.LyricsControl != nil {
		h.LyricsControl.SetLyrics(lyricsLines)
		if h.hasLatestProgress && !h.isUserScrolling {
			h.LyricsControl.Update(h.latestProgress)
		}
	}

	if hasCover && coverImage != nil {
		h.Cover = ebiten.NewImageFromImage(coverImage)
		h.CoverPosition.W = float64(h.Cover.Bounds().Dx())
		h.CoverPosition.H = float64(h.Cover.Bounds().Dy())
		h.CoverPosition.OriginX = h.CoverPosition.W / 2
		h.CoverPosition.OriginY = h.CoverPosition.H / 2
		w, he := ebiten.WindowSize()
		h.updateCoverTransform(w, he)
	}

	if hasFontConfig {
		h.applyMapFontConfig(fontConfigData)
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
		h.queueProgress(time.Duration(value) * time.Millisecond)
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

	h.LyricsControl = LyricsComponent.NewLyricsComponent(h.AnimateManager, h.Font, float64(ww), float64(hh), h.FontSize, h.FD)
	h.LyricsControl.Init()
	h.CoverPosition = lyrics.NewPosition(0, 0, 0, 0)
	h.bindEvents()

	h.loadAndApplyFontConfig(h.fontConfig)
}

func (h *Home) OnEnter(params map[string]any) {
	log.Println("Home OnEnter", params)
	if h.Font == nil {
		log.Println("Home page font is nil")
	}
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
}

func (h *Home) Update() error {
	h.applyPendingEvents()
	h.handleWheelScroll()

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

	return nil
}

func (h *Home) Draw(screen *ebiten.Image) {
	msg := fmt.Sprintf(
		"Home Scene\nFS:%.0f FD:%.2f\nFamily:%s\nWeight:%d Italic:%v\nScroll:%v\nF5/F6 family, F7/F8 weight, F9 italic, F10 reload config",
		h.FontSize,
		h.FD,
		h.currentFamily,
		h.fontWeight,
		h.fontItalic,
		h.isUserScrolling,
	)

	if h.Cover != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM = lyrics.TransformToGeoM(&h.CoverPosition)
		screen.DrawImage(h.Cover, op)
	}
	if h.LyricsControl != nil {
		pos := lyrics.NewPosition(0, 0, 0, 0)
		pos.SetTranslateY(h.manualScrollOffset)
		h.LyricsControl.Draw(screen, &pos)
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (h *Home) OnResize(w, he int, isFirst bool) {
	log.Println("Home OnResize", w, he, isFirst)
	if !isFirst && h.LyricsControl != nil {
		h.LyricsControl.Resize(float64(w), float64(he))
	}
	h.updateCoverTransform(w, he)
}
