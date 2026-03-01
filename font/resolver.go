package font

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	gotextfont "github.com/go-text/typesetting/font"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Weight int

const (
	WeightThin       Weight = 100
	WeightExtraLight Weight = 200
	WeightLight      Weight = 300
	WeightRegular    Weight = 400
	WeightMedium     Weight = 500
	WeightSemiBold   Weight = 600
	WeightBold       Weight = 700
	WeightExtraBold  Weight = 800
	WeightBlack      Weight = 900
)

type ResolveOptions struct {
	Path       string
	Families   []string
	Weight     Weight
	Italic     bool
	RequireCJK bool
}

type ResolvedFace struct {
	Path   string
	Family string
	Style  string
	Weight Weight
	Source *text.GoTextFaceSource
}

var (
	latinProbeRunes = []rune{'A', 'a', '0'}
	cjkProbeRunes   = []rune{'\u4e2d', '\u6587', '\u4f60', '\u597d'}

	allFontsOnce   sync.Once
	allFontsCached []string

	loadSourceCacheMu sync.RWMutex
	loadSourceCache   = map[string][]*text.GoTextFaceSource{}

	resolvedCacheMu sync.RWMutex
	resolvedCache   = map[string]*ResolvedFace{}

	coverageCacheMu sync.RWMutex
	coverageCache   = map[*text.GoTextFaceSource]coverageState{}
)

type coverageState struct {
	latinOK bool
	hasCJK  bool
}

func normalizeWeight(w Weight) Weight {
	if w < WeightThin {
		return WeightThin
	}
	if w > WeightBlack {
		return WeightBlack
	}
	return Weight(int(math.Round(float64(w)/100.0)) * 100)
}

func ParseWeight(raw string) (Weight, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return WeightRegular, errors.New("empty weight")
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return normalizeWeight(Weight(n)), nil
	}

	switch raw {
	case "thin":
		return WeightThin, nil
	case "extralight", "extra-light", "ultralight", "ultra-light":
		return WeightExtraLight, nil
	case "light":
		return WeightLight, nil
	case "regular", "normal", "book":
		return WeightRegular, nil
	case "medium":
		return WeightMedium, nil
	case "semibold", "semi-bold", "demibold", "demi-bold":
		return WeightSemiBold, nil
	case "bold":
		return WeightBold, nil
	case "extrabold", "extra-bold", "ultrabold", "ultra-bold":
		return WeightExtraBold, nil
	case "black", "heavy":
		return WeightBlack, nil
	default:
		return WeightRegular, fmt.Errorf("unknown weight: %s", raw)
	}
}

func DefaultFamilies() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"Microsoft YaHei UI",
			"Microsoft YaHei",
			"PingFang SC",
			"Noto Sans CJK SC",
			"Source Han Sans SC",
			"SimHei",
			"SimSun",
			"Segoe UI",
			"Arial Unicode MS",
		}
	case "darwin":
		return []string{
			"PingFang SC",
			"Hiragino Sans GB",
			"Heiti SC",
			"Noto Sans CJK SC",
			"Source Han Sans SC",
			"SF Pro",
			"Helvetica Neue",
			"Arial Unicode MS",
		}
	default:
		return []string{
			"Noto Sans CJK SC",
			"Source Han Sans SC",
			"WenQuanYi Micro Hei",
			"Droid Sans Fallback",
			"Noto Sans",
			"DejaVu Sans",
		}
	}
}

func DefaultResolveOptions() ResolveOptions {
	return ResolveOptions{
		Families:   DefaultFamilies(),
		Weight:     WeightMedium,
		Italic:     false,
		RequireCJK: true,
	}
}

func ResolveFaceSourceFromEnv() (*ResolvedFace, error) {
	opts := ApplyEnvResolveOptions(DefaultResolveOptions())
	return ResolveFaceSource(opts)
}

type scoredPath struct {
	path  string
	score int
}

func normalizeName(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "")
	return replacer.Replace(s)
}

func normalizeFamilies(families []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(families))
	for _, family := range families {
		family = strings.TrimSpace(family)
		if family == "" {
			continue
		}
		key := normalizeName(family)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, family)
	}
	return out
}

func cloneResolvedFace(face *ResolvedFace) *ResolvedFace {
	if face == nil {
		return nil
	}
	return &ResolvedFace{
		Path:   face.Path,
		Family: face.Family,
		Style:  face.Style,
		Weight: face.Weight,
		Source: face.Source,
	}
}

func cloneSourceSlice(src []*text.GoTextFaceSource) []*text.GoTextFaceSource {
	if len(src) == 0 {
		return nil
	}
	dst := make([]*text.GoTextFaceSource, len(src))
	copy(dst, src)
	return dst
}

func resolveCacheKey(opts ResolveOptions) string {
	families := normalizeFamilies(opts.Families)
	normalizedFamilies := make([]string, 0, len(families))
	for _, family := range families {
		normalizedFamilies = append(normalizedFamilies, normalizeName(family))
	}
	path := strings.TrimSpace(opts.Path)
	if path != "" {
		path = filepath.Clean(path)
	}
	return fmt.Sprintf(
		"%s|%s|%d|%t|%t",
		path,
		strings.Join(normalizedFamilies, ","),
		opts.Weight,
		opts.Italic,
		opts.RequireCJK,
	)
}

func getAllFontsCached() []string {
	allFontsOnce.Do(func() {
		allFontsCached = append(allFontsCached, GetAllFonts()...)
	})
	out := make([]string, len(allFontsCached))
	copy(out, allFontsCached)
	return out
}

func scorePath(path string, families []string) int {
	if len(families) == 0 {
		return 0
	}
	base := normalizeName(filepath.Base(path))
	full := normalizeName(path)
	best := 0
	for i, family := range families {
		nf := normalizeName(family)
		if nf == "" {
			continue
		}
		score := 0
		switch {
		case strings.Contains(base, nf):
			score = 140 - i*4
		case strings.Contains(full, nf):
			score = 120 - i*4
		}
		if score > best {
			best = score
		}
	}

	if strings.Contains(base, "msyh") || strings.Contains(base, "simhei") || strings.Contains(base, "simsun") || strings.Contains(base, "notosanscjk") || strings.Contains(base, "sourcehansans") || strings.Contains(base, "pingfang") {
		best += 60
	}

	return best
}

func scoreFamily(metaFamily string, families []string) int {
	if len(families) == 0 {
		return 0
	}
	normalized := normalizeName(metaFamily)
	best := 0
	for i, family := range families {
		nf := normalizeName(family)
		if nf == "" {
			continue
		}
		score := 0
		switch {
		case normalized == nf:
			score = 220 - i*4
		case strings.Contains(normalized, nf) || strings.Contains(nf, normalized):
			score = 160 - i*4
		}
		if score > best {
			best = score
		}
	}
	return best
}

func styleLabel(style text.Style) string {
	switch style {
	case text.StyleItalic:
		return "italic"
	default:
		return "normal"
	}
}

func scoreStyle(meta text.Metadata, expectedWeight Weight, expectedItalic bool) int {
	diff := math.Abs(float64(normalizeWeight(expectedWeight) - Weight(meta.Weight)))
	score := 220 - int(diff)
	if score < 0 {
		score = 0
	}
	if (meta.Style == text.StyleItalic) == expectedItalic {
		score += 40
	} else {
		score -= 30
	}
	return score
}

func loadSources(path string) ([]*text.GoTextFaceSource, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("empty font path")
	}
	path = filepath.Clean(path)

	loadSourceCacheMu.RLock()
	if cached, ok := loadSourceCache[path]; ok && len(cached) > 0 {
		loadSourceCacheMu.RUnlock()
		return cloneSourceSlice(cached), nil
	}
	loadSourceCacheMu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if sources, err := text.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data)); err == nil && len(sources) > 0 {
		loadSourceCacheMu.Lock()
		loadSourceCache[path] = cloneSourceSlice(sources)
		loadSourceCacheMu.Unlock()
		return sources, nil
	}
	source, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	sources := []*text.GoTextFaceSource{source}
	loadSourceCacheMu.Lock()
	loadSourceCache[path] = cloneSourceSlice(sources)
	loadSourceCacheMu.Unlock()
	return sources, nil
}

func buildCandidates(opts ResolveOptions) []scoredPath {
	all := getAllFontsCached()
	candidates := make([]scoredPath, 0, len(all)+1)
	seen := map[string]struct{}{}

	if opts.Path != "" {
		path := filepath.Clean(opts.Path)
		candidates = append(candidates, scoredPath{path: path, score: 1000})
		seen[path] = struct{}{}
	}

	for _, path := range all {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		candidates = append(candidates, scoredPath{
			path:  path,
			score: scorePath(path, opts.Families),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return len(candidates[i].path) < len(candidates[j].path)
		}
		return candidates[i].score > candidates[j].score
	})

	maxCandidates := 260
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}
	return candidates
}

func supportsRunes(source *text.GoTextFaceSource, probes []rune) bool {
	if source == nil {
		return false
	}
	internal := source.UnsafeInternal()
	face, ok := internal.(*gotextfont.Face)
	if !ok || face == nil {
		return false
	}

	for _, r := range probes {
		if _, ok := face.NominalGlyph(r); !ok {
			return false
		}
	}
	return true
}

func coverageScore(source *text.GoTextFaceSource, requireCJK bool) (score int, hasCJK bool) {
	if source == nil {
		return -2000, false
	}

	coverageCacheMu.RLock()
	state, ok := coverageCache[source]
	coverageCacheMu.RUnlock()
	if !ok {
		state = coverageState{
			latinOK: supportsRunes(source, latinProbeRunes),
			hasCJK:  supportsRunes(source, cjkProbeRunes),
		}
		coverageCacheMu.Lock()
		coverageCache[source] = state
		coverageCacheMu.Unlock()
	}
	latinOK := state.latinOK
	hasCJK = state.hasCJK

	if latinOK {
		score += 40
	} else {
		score -= 200
	}

	if hasCJK {
		score += 220
	} else if requireCJK {
		score -= 1200
	} else {
		score -= 80
	}

	return score, hasCJK
}

func ResolveFaceSource(opts ResolveOptions) (*ResolvedFace, error) {
	opts.Weight = normalizeWeight(opts.Weight)
	if len(opts.Families) == 0 {
		opts.Families = DefaultFamilies()
	}
	opts.Families = normalizeFamilies(opts.Families)

	cacheKey := resolveCacheKey(opts)
	resolvedCacheMu.RLock()
	if cached, ok := resolvedCache[cacheKey]; ok && cached != nil && cached.Source != nil {
		resolvedCacheMu.RUnlock()
		return cloneResolvedFace(cached), nil
	}
	resolvedCacheMu.RUnlock()

	candidates := buildCandidates(opts)
	if len(candidates) == 0 {
		return nil, errors.New("no fonts available")
	}

	type picked struct {
		face   *ResolvedFace
		score  int
		hasAny bool
	}
	best := picked{}

	for _, candidate := range candidates {
		sources, err := loadSources(candidate.path)
		if err != nil {
			continue
		}
		for _, source := range sources {
			if source == nil {
				continue
			}
			meta := source.Metadata()
			currentWeight := normalizeWeight(Weight(meta.Weight))
			score := candidate.score + scoreFamily(meta.Family, opts.Families) + scoreStyle(meta, opts.Weight, opts.Italic)

			coverage, hasCJK := coverageScore(source, opts.RequireCJK)
			score += coverage
			if opts.RequireCJK && !hasCJK {
				continue
			}

			if !best.hasAny || score > best.score {
				best.hasAny = true
				best.score = score
				best.face = &ResolvedFace{
					Path:   candidate.path,
					Family: meta.Family,
					Style:  styleLabel(meta.Style),
					Weight: currentWeight,
					Source: source,
				}
			}
		}
	}

	if !best.hasAny || best.face == nil || best.face.Source == nil {
		if opts.RequireCJK {
			return nil, errors.New("unable to resolve a CJK-capable font source")
		}
		return nil, errors.New("unable to resolve a usable font source")
	}

	resolvedCacheMu.Lock()
	resolvedCache[cacheKey] = cloneResolvedFace(best.face)
	resolvedCacheMu.Unlock()

	return cloneResolvedFace(best.face), nil
}
