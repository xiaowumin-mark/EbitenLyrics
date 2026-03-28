package font

// 文件说明：系统字体解析核心实现。
// 主要职责：扫描字体索引、评估字重与覆盖率，并挑选主字体及回退字体。

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
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

	tdfont "github.com/tdewolff/font"

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
	Path      string
	Family    string
	Style     string
	Weight    Weight
	Source    *text.GoTextFaceSource
	Fallbacks []*text.GoTextFaceSource
	Sources   []*text.GoTextFaceSource
}

type scriptCoverage uint32

const (
	coverageLatin scriptCoverage = 1 << iota
	coverageCJK
	coverageHiragana
	coverageHangul
	coverageCyrillic
	coverageArabic
	coverageHebrew
	coverageDevanagari
)

const (
	fontIndexCacheVersion = 1
	defaultFontIndexPath  = "cache/font-index.gob"
	invalidFontScore      = -1 << 30
)

type coverageProbe struct {
	bit   scriptCoverage
	runes []rune
}

type fontIndexEntry struct {
	Path            string
	CollectionIndex int
	Family          string
	Style           string
	Weight          Weight
	Italic          bool
	Coverage        scriptCoverage
}

type fontIndexCache struct {
	Version  int
	Entries  []fontIndexEntry
	Generics map[string][]string
}

type fontIndex struct {
	entries     []fontIndexEntry
	generics    map[string][]string
	byFamily    map[string][]int
	byPath      map[string][]int
	byBaseName  map[string][]int
	uniquePaths []string
	defaultSans []string
	defaultCJK  []string
}

type resolvedEntry struct {
	entry  fontIndexEntry
	source *text.GoTextFaceSource
}

var (
	coverageProbes = []coverageProbe{
		{bit: coverageLatin, runes: []rune{'A', 'a', '0'}},
		{bit: coverageCJK, runes: []rune{'你', '好', '中'}},
		{bit: coverageHiragana, runes: []rune{'あ', 'い'}},
		{bit: coverageHangul, runes: []rune{'한', '국'}},
		{bit: coverageCyrillic, runes: []rune{'Ж', 'Я'}},
		{bit: coverageArabic, runes: []rune{'م', 'ر'}},
		{bit: coverageHebrew, runes: []rune{'ש', 'ל'}},
		{bit: coverageDevanagari, runes: []rune{'क', 'ह'}},
	}
	fallbackProbeOrder = []scriptCoverage{
		coverageCJK,
		coverageHiragana,
		coverageHangul,
		coverageCyrillic,
		coverageArabic,
		coverageHebrew,
		coverageDevanagari,
	}

	resolvedCacheMu sync.RWMutex
	resolvedCache   = map[string]*ResolvedFace{}

	sourceCacheMu sync.RWMutex
	sourceCache   = map[string]*text.GoTextFaceSource{}

	systemFontIndexOnce sync.Once
	systemFontIndex     *fontIndex
	systemFontIndexErr  error
)

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
			"Noto Sans CJK SC",
			"Source Han Sans SC",
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

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", ",", "")
	return replacer.Replace(s)
}

func normalizePathInput(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"'")
	if path == "" {
		return ""
	}
	path = os.ExpandEnv(path)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, path[2:])
			}
		}
	}
	return filepath.Clean(path)
}

func resolvePathFromBase(path, baseDir string) string {
	path = normalizePathInput(path)
	if path == "" {
		return ""
	}
	baseDir = normalizePathInput(baseDir)
	if filepath.IsAbs(path) {
		return path
	}
	if baseDir != "" {
		return filepath.Clean(filepath.Join(baseDir, path))
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return path
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
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
	clone := &ResolvedFace{
		Path:   face.Path,
		Family: face.Family,
		Style:  face.Style,
		Weight: face.Weight,
		Source: face.Source,
	}
	if len(face.Fallbacks) > 0 {
		clone.Fallbacks = append([]*text.GoTextFaceSource{}, face.Fallbacks...)
	}
	if len(face.Sources) > 0 {
		clone.Sources = append([]*text.GoTextFaceSource{}, face.Sources...)
	}
	return clone
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

func fontIndexCachePath() string {
	if fromEnv := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_CACHE")); fromEnv != "" {
		return resolvePathFromBase(fromEnv, "")
	}
	return resolvePathFromBase(defaultFontIndexPath, "")
}

func shouldRefreshFontIndex() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("EBITENLYRICS_FONT_REFRESH")))
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func loadSystemFontIndex() (*fontIndex, error) {
	systemFontIndexOnce.Do(func() {
		systemFontIndex, systemFontIndexErr = openOrBuildFontIndex()
	})
	return systemFontIndex, systemFontIndexErr
}

func openOrBuildFontIndex() (*fontIndex, error) {
	cachePath := fontIndexCachePath()
	if !shouldRefreshFontIndex() {
		if cache, err := readFontIndexCache(cachePath); err == nil {
			return newFontIndex(cache), nil
		}
	}

	cache, err := buildFontIndexCache()
	if err != nil {
		return nil, err
	}
	if err := writeFontIndexCache(cachePath, cache); err != nil {
		// Cache write failure should not block rendering.
	}
	return newFontIndex(cache), nil
}

func readFontIndexCache(path string) (*fontIndexCache, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cache fontIndexCache
	if err := gob.NewDecoder(f).Decode(&cache); err != nil {
		return nil, err
	}
	if cache.Version != fontIndexCacheVersion {
		return nil, fmt.Errorf("font index cache version mismatch: %d", cache.Version)
	}
	if len(cache.Entries) == 0 {
		return nil, errors.New("font index cache is empty")
	}
	return &cache, nil
}

func writeFontIndexCache(path string, cache *fontIndexCache) error {
	if cache == nil {
		return errors.New("font index cache is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(cache)
}

func newFontIndex(cache *fontIndexCache) *fontIndex {
	idx := &fontIndex{
		entries:     append([]fontIndexEntry{}, cache.Entries...),
		generics:    cloneGenericFamilies(cache.Generics),
		byFamily:    map[string][]int{},
		byPath:      map[string][]int{},
		byBaseName:  map[string][]int{},
		defaultSans: append([]string{}, cache.Generics["sans-serif"]...),
		defaultCJK:  append([]string{}, DefaultFamilies()...),
	}
	for i, entry := range idx.entries {
		path := filepath.Clean(entry.Path)
		idx.entries[i].Path = path
		idx.byPath[path] = append(idx.byPath[path], i)
		base := normalizeName(filepath.Base(path))
		if base != "" {
			idx.byBaseName[base] = append(idx.byBaseName[base], i)
		}
		family := normalizeName(entry.Family)
		if family != "" {
			idx.byFamily[family] = append(idx.byFamily[family], i)
		}
	}
	idx.uniquePaths = make([]string, 0, len(idx.byPath))
	for path := range idx.byPath {
		idx.uniquePaths = append(idx.uniquePaths, path)
	}
	sort.Strings(idx.uniquePaths)
	return idx
}

func cloneGenericFamilies(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = append([]string{}, values...)
	}
	return out
}

func buildFontIndexCache() (*fontIndexCache, error) {
	dirs := tdfont.DefaultFontDirs()
	seenPaths := map[string]struct{}{}
	entries := []fontIndexEntry{}

	for _, dir := range dirs {
		dir = normalizePathInput(dir)
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !supportsIndexedFontFile(path) {
				return nil
			}
			path = filepath.Clean(path)
			if _, ok := seenPaths[path]; ok {
				return nil
			}
			seenPaths[path] = struct{}{}

			found, err := inspectFontFile(path)
			if err != nil || len(found) == 0 {
				return nil
			}
			entries = append(entries, found...)
			return nil
		})
	}

	if len(entries) == 0 {
		return nil, errors.New("no system fonts indexed")
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Family == entries[j].Family {
			if entries[i].Weight == entries[j].Weight {
				if entries[i].Italic == entries[j].Italic {
					if entries[i].Path == entries[j].Path {
						return entries[i].CollectionIndex < entries[j].CollectionIndex
					}
					return entries[i].Path < entries[j].Path
				}
				return !entries[i].Italic && entries[j].Italic
			}
			return entries[i].Weight < entries[j].Weight
		}
		return entries[i].Family < entries[j].Family
	})

	return &fontIndexCache{
		Version:  fontIndexCacheVersion,
		Entries:  entries,
		Generics: cloneGenericFamilies(tdfont.DefaultGenericFonts()),
	}, nil
}

func supportsIndexedFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc":
		return true
	default:
		return false
	}
}

func inspectFontFile(path string) ([]fontIndexEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	count := fontFaceCount(data)
	if count < 1 {
		count = 1
	}

	entries := make([]fontIndexEntry, 0, count)
	for i := 0; i < count; i++ {
		sfnt, err := tdfont.ParseSFNT(data, i)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			break
		}

		family := pickFontName(sfnt, tdfont.NamePreferredFamily, tdfont.NameFontFamily)
		if family == "" {
			continue
		}
		subfamily := pickFontName(sfnt, tdfont.NamePreferredSubfamily, tdfont.NameFontSubfamily)
		weight, italic := fontStyleFromSFNT(sfnt)
		style := strings.TrimSpace(subfamily)
		if style == "" {
			style = formatStyle(weight, italic)
		}

		entries = append(entries, fontIndexEntry{
			Path:            filepath.Clean(path),
			CollectionIndex: i,
			Family:          family,
			Style:           style,
			Weight:          weight,
			Italic:          italic,
			Coverage:        detectCoverage(sfnt),
		})
	}

	return entries, nil
}

func fontFaceCount(data []byte) int {
	if len(data) < 12 {
		return 1
	}
	if string(data[:4]) != "ttcf" {
		return 1
	}
	return int(binary.BigEndian.Uint32(data[8:12]))
}

func pickFontName(sfnt *tdfont.SFNT, names ...tdfont.NameID) string {
	if sfnt == nil || sfnt.Name == nil {
		return ""
	}
	for _, name := range names {
		if value := firstDecodedName(sfnt.Name.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func firstDecodedName[T interface{ String() string }](records []T) string {
	for _, record := range records {
		value := strings.TrimSpace(record.String())
		if value != "" {
			return value
		}
	}
	return ""
}

func fontStyleFromSFNT(sfnt *tdfont.SFNT) (Weight, bool) {
	weight := WeightRegular
	italic := false

	if sfnt != nil && sfnt.OS2 != nil {
		if sfnt.OS2.UsWeightClass != 0 {
			weight = normalizeWeight(Weight(sfnt.OS2.UsWeightClass))
		}
		italic = (sfnt.OS2.FsSelection & 0x0001) != 0
	}
	if sfnt != nil && sfnt.Post != nil && math.Abs(sfnt.Post.ItalicAngle) > 0.01 {
		italic = true
	}
	return weight, italic
}

func detectCoverage(sfnt *tdfont.SFNT) scriptCoverage {
	if sfnt == nil {
		return 0
	}
	var coverage scriptCoverage
	for _, probe := range coverageProbes {
		matched := true
		for _, r := range probe.runes {
			if sfnt.GlyphIndex(r) == 0 {
				matched = false
				break
			}
		}
		if matched {
			coverage |= probe.bit
		}
	}
	return coverage
}

func resolveExplicitFontPath(path string, idx *fontIndex) (string, error) {
	path = normalizePathInput(path)
	if path == "" {
		return "", errors.New("empty font path")
	}

	tryPaths := []string{path}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			tryPaths = append([]string{filepath.Clean(abs)}, tryPaths...)
		}
	}
	for _, candidate := range tryPaths {
		if fileExists(candidate) {
			return filepath.Clean(candidate), nil
		}
	}

	if idx != nil {
		base := normalizeName(filepath.Base(path))
		for _, candidate := range idx.uniquePaths {
			if normalizeName(filepath.Base(candidate)) == base {
				return candidate, nil
			}
		}
		for _, candidate := range idx.uniquePaths {
			normBase := normalizeName(filepath.Base(candidate))
			if strings.Contains(normBase, base) || strings.Contains(base, normBase) {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("font file %q not found", path)
}

func styleScore(actual Weight, actualItalic bool, expectedWeight Weight, expectedItalic bool) int {
	diff := math.Abs(float64(normalizeWeight(expectedWeight) - normalizeWeight(actual)))
	score := 220 - int(diff)
	if score < 0 {
		score = 0
	}
	if actualItalic == expectedItalic {
		score += 40
	} else {
		score -= 30
	}
	return score
}

func familyScore(family string, families []string) int {
	if len(families) == 0 {
		return 0
	}
	normalized := normalizeName(family)
	best := 0
	for i, familyName := range families {
		target := normalizeName(familyName)
		if target == "" {
			continue
		}
		score := 0
		switch {
		case normalized == target:
			score = 240 - i*6
		case strings.Contains(normalized, target) || strings.Contains(target, normalized):
			score = 160 - i*6
		}
		if score > best {
			best = score
		}
	}
	return best
}

func primaryCoverageScore(coverage scriptCoverage, requireCJK bool) int {
	score := 0
	if coverage&coverageLatin != 0 {
		score += 40
	} else {
		score -= 180
	}
	if coverage&coverageCJK != 0 {
		score += 220
	} else if requireCJK {
		score -= 140
	}
	return score
}

func fallbackCoverageScore(coverage scriptCoverage, target scriptCoverage) int {
	if coverage&target == 0 {
		return invalidFontScore
	}
	score := 200
	if coverage&coverageLatin != 0 {
		score += 25
	}
	if target == coverageCJK && coverage&coverageCJK != 0 {
		score += 40
	}
	return score
}

func formatStyle(weight Weight, italic bool) string {
	weight = normalizeWeight(weight)
	var style string
	switch weight {
	case WeightThin:
		style = "Thin"
	case WeightExtraLight:
		style = "ExtraLight"
	case WeightLight:
		style = "Light"
	case WeightRegular:
		style = "Regular"
	case WeightMedium:
		style = "Medium"
	case WeightSemiBold:
		style = "SemiBold"
	case WeightBold:
		style = "Bold"
	case WeightExtraBold:
		style = "ExtraBold"
	case WeightBlack:
		style = "Black"
	default:
		style = "Regular"
	}
	if italic {
		style += " Italic"
	}
	return style
}

func entryKey(entry fontIndexEntry) string {
	return fmt.Sprintf("%s#%d", entry.Path, entry.CollectionIndex)
}

func loadFaceSource(entry fontIndexEntry) (*text.GoTextFaceSource, error) {
	key := entryKey(entry)

	sourceCacheMu.RLock()
	if cached, ok := sourceCache[key]; ok && cached != nil {
		sourceCacheMu.RUnlock()
		return cached, nil
	}
	sourceCacheMu.RUnlock()

	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return nil, err
	}

	var source *text.GoTextFaceSource
	if entry.CollectionIndex > 0 || isFontCollectionData(data) {
		sfnt, err := tdfont.ParseSFNT(data, entry.CollectionIndex)
		if err != nil {
			return nil, err
		}
		source, err = text.NewGoTextFaceSource(bytes.NewReader(sfnt.Write()))
		if err != nil {
			return nil, err
		}
	} else {
		source, err = text.NewGoTextFaceSource(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
	}

	sourceCacheMu.Lock()
	sourceCache[key] = source
	sourceCacheMu.Unlock()
	return source, nil
}

func isFontCollectionData(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "ttcf"
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

func detectSourceCoverage(source *text.GoTextFaceSource) scriptCoverage {
	if source == nil {
		return 0
	}
	var coverage scriptCoverage
	for _, probe := range coverageProbes {
		if supportsRunes(source, probe.runes) {
			coverage |= probe.bit
		}
	}
	return coverage
}

func resolvePrimaryEntry(idx *fontIndex, opts ResolveOptions) (fontIndexEntry, error) {
	if strings.TrimSpace(opts.Path) != "" {
		resolvedPath, err := resolveExplicitFontPath(opts.Path, idx)
		if err != nil {
			return fontIndexEntry{}, err
		}

		entries, err := entriesForPath(idx, resolvedPath)
		if err != nil {
			return fontIndexEntry{}, err
		}
		best, ok := bestPrimaryEntry(entries, opts)
		if !ok {
			return fontIndexEntry{}, fmt.Errorf("no usable font face in %s", resolvedPath)
		}
		return best, nil
	}

	candidates := idx.entries
	if len(opts.Families) > 0 {
		indexes := []int{}
		seen := map[int]struct{}{}
		for _, family := range opts.Families {
			for _, i := range idx.byFamily[normalizeName(family)] {
				if _, ok := seen[i]; ok {
					continue
				}
				seen[i] = struct{}{}
				indexes = append(indexes, i)
			}
		}
		if len(indexes) > 0 {
			candidates = make([]fontIndexEntry, 0, len(indexes))
			for _, i := range indexes {
				candidates = append(candidates, idx.entries[i])
			}
		}
	}

	best, ok := bestPrimaryEntry(candidates, opts)
	if !ok {
		return fontIndexEntry{}, errors.New("unable to resolve a usable font source")
	}
	return best, nil
}

func bestPrimaryEntry(entries []fontIndexEntry, opts ResolveOptions) (fontIndexEntry, bool) {
	bestScore := invalidFontScore
	var best fontIndexEntry
	ok := false
	for _, entry := range entries {
		score := familyScore(entry.Family, opts.Families)
		score += styleScore(entry.Weight, entry.Italic, opts.Weight, opts.Italic)
		score += primaryCoverageScore(entry.Coverage, opts.RequireCJK)
		if len(opts.Families) > 0 && familyScore(entry.Family, opts.Families) == 0 {
			score -= 80
		}
		if !ok || score > bestScore {
			bestScore = score
			best = entry
			ok = true
		}
	}
	return best, ok
}

func entriesForPath(idx *fontIndex, path string) ([]fontIndexEntry, error) {
	path = filepath.Clean(path)
	if idx != nil {
		if indexes, ok := idx.byPath[path]; ok && len(indexes) > 0 {
			entries := make([]fontIndexEntry, 0, len(indexes))
			for _, i := range indexes {
				entries = append(entries, idx.entries[i])
			}
			return entries, nil
		}
	}
	return inspectFontFile(path)
}

func fallbackFamilyHints(idx *fontIndex) []string {
	hints := append([]string{}, idx.defaultCJK...)
	hints = append(hints, idx.defaultSans...)
	return normalizeFamilies(hints)
}

func resolveFallbackEntries(idx *fontIndex, opts ResolveOptions, primary resolvedEntry) ([]resolvedEntry, error) {
	coverage := detectSourceCoverage(primary.source)
	if coverage == 0 {
		coverage = primary.entry.Coverage
	}

	used := map[string]struct{}{
		entryKey(primary.entry): {},
	}
	fallbacks := make([]resolvedEntry, 0, 4)
	hints := fallbackFamilyHints(idx)

	for _, target := range fallbackProbeOrder {
		if coverage&target != 0 {
			continue
		}
		entry, ok := bestFallbackEntry(idx.entries, target, opts, hints, used)
		if !ok {
			continue
		}
		source, err := loadFaceSource(entry)
		if err != nil {
			continue
		}
		if detectSourceCoverage(source)&target == 0 {
			continue
		}
		fallbacks = append(fallbacks, resolvedEntry{entry: entry, source: source})
		coverage |= detectSourceCoverage(source)
		used[entryKey(entry)] = struct{}{}
		if len(fallbacks) >= 4 {
			break
		}
	}

	if len(fallbacks) == 0 || coverage&coverageLatin == 0 {
		if entry, ok := bestFallbackEntry(idx.entries, coverageLatin, opts, hints, used); ok {
			source, err := loadFaceSource(entry)
			if err == nil {
				fallbacks = append(fallbacks, resolvedEntry{entry: entry, source: source})
			}
		}
	}

	return fallbacks, nil
}

func bestFallbackEntry(entries []fontIndexEntry, target scriptCoverage, opts ResolveOptions, hints []string, used map[string]struct{}) (fontIndexEntry, bool) {
	bestScore := invalidFontScore
	var best fontIndexEntry
	ok := false
	for _, entry := range entries {
		if _, exists := used[entryKey(entry)]; exists {
			continue
		}
		score := fallbackCoverageScore(entry.Coverage, target)
		if score <= invalidFontScore {
			continue
		}
		score += styleScore(entry.Weight, entry.Italic, opts.Weight, opts.Italic)
		score += familyScore(entry.Family, hints)
		score += familyScore(entry.Family, opts.Families) / 2
		if !ok || score > bestScore {
			bestScore = score
			best = entry
			ok = true
		}
	}
	return best, ok
}

func ResolveFaceSource(opts ResolveOptions) (*ResolvedFace, error) {
	opts.Weight = normalizeWeight(opts.Weight)
	opts.Path = normalizePathInput(opts.Path)
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

	idx, err := loadSystemFontIndex()
	if err != nil {
		return nil, err
	}

	primaryEntry, err := resolvePrimaryEntry(idx, opts)
	if err != nil {
		return nil, err
	}
	primarySource, err := loadFaceSource(primaryEntry)
	if err != nil {
		return nil, err
	}

	primary := resolvedEntry{
		entry:  primaryEntry,
		source: primarySource,
	}
	fallbackEntries, err := resolveFallbackEntries(idx, opts, primary)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedFace{
		Path:   primaryEntry.Path,
		Family: primaryEntry.Family,
		Style:  primaryEntry.Style,
		Weight: primaryEntry.Weight,
		Source: primarySource,
	}
	resolved.Sources = []*text.GoTextFaceSource{primarySource}
	for _, fallback := range fallbackEntries {
		if fallback.source == nil || fallback.source == primarySource {
			continue
		}
		resolved.Fallbacks = append(resolved.Fallbacks, fallback.source)
		resolved.Sources = append(resolved.Sources, fallback.source)
	}

	resolvedCacheMu.Lock()
	resolvedCache[cacheKey] = cloneResolvedFace(resolved)
	resolvedCacheMu.Unlock()
	return cloneResolvedFace(resolved), nil
}
