package font

import (
	"bytes"
	"encoding/binary"
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

	"github.com/edsrzf/mmap-go"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	lru "github.com/hashicorp/golang-lru/v2"
	tdfont "github.com/tdewolff/font"
	"golang.org/x/image/font/gofont/goregular"
)

const lastResortFamily = "__font_last_resort__"

const (
	glyphSupportCacheSize     = 16384
	contentSelectionCacheSize = 1024
)

type fontRecord struct {
	Path            string
	CollectionIndex int
	Family          string
	Style           string
	Weight          Weight
	Italic          bool
	Aliases         []string
}

func (r fontRecord) entryKey() string {
	return fmt.Sprintf("%s#%d", filepath.Clean(r.Path), r.CollectionIndex)
}

func (r fontRecord) matchesFamily(family string) bool {
	return r.familyMatchLevel(family) == familyMatchExact
}

func (r fontRecord) resolvedFamily(requested string) string {
	if strings.TrimSpace(requested) != "" && r.matchesFamily(requested) {
		return strings.TrimSpace(requested)
	}
	return r.Family
}

type familyMatchLevel uint8

const (
	familyMatchNone familyMatchLevel = iota
	familyMatchPrefix
	familyMatchExact
)

func (r fontRecord) familyMatchLevel(family string) familyMatchLevel {
	target := normalizeName(family)
	if target == "" {
		return familyMatchNone
	}
	if normalizeName(r.Family) == target {
		return familyMatchExact
	}
	for _, alias := range r.Aliases {
		if normalizeName(alias) == target {
			return familyMatchExact
		}
	}
	if len(target) < 5 {
		return familyMatchNone
	}
	if strings.HasPrefix(normalizeName(r.Family), target) {
		return familyMatchPrefix
	}
	for _, alias := range r.Aliases {
		if strings.HasPrefix(normalizeName(alias), target) {
			return familyMatchPrefix
		}
	}
	return familyMatchNone
}

type loadedFontFile struct {
	path    string
	size    int64
	mapped  mmap.MMap
	sources []*text.GoTextFaceSource
}

type glyphCacheKey struct {
	entry string
	r     rune
}

type contentSelectionKey struct {
	request string
	content string
}

func (f *loadedFontFile) close() {
	if f == nil {
		return
	}
	if f.mapped != nil {
		_ = f.mapped.Unmap()
	}
	f.mapped = nil
	f.sources = nil
}

type FontManager struct {
	mu sync.RWMutex

	customPaths   map[string]string
	fallbackRules map[string][]string
	familyCache   map[string][]fontRecord
	missCache     map[string]struct{}
	pathCache     map[string][]fontRecord
	chainCache    map[string]*ResolvedFontChain

	sourceCache  *lru.Cache[string, *loadedFontFile]
	glyphCache   *lru.Cache[glyphCacheKey, bool]
	contentCache *lru.Cache[contentSelectionKey, []ResolvedFont]

	systemIndexOnce sync.Once
	systemIndex     []fontRecord
	systemIndexErr  error

	systemDirs       []string
	systemFallbacks  []string
	lastResortPath   string
	lastResortSource *text.GoTextFaceSource
}

type FontManagerStats struct {
	LoadedFiles int
	MappedBytes int64
}

func NewFontManager(sourceCacheSize int) *FontManager {
	if sourceCacheSize <= 0 {
		sourceCacheSize = 16
	}
	cache, _ := lru.NewWithEvict[string, *loadedFontFile](sourceCacheSize, func(_ string, value *loadedFontFile) {
		value.close()
	})
	glyphCache, _ := lru.New[glyphCacheKey, bool](glyphSupportCacheSize)
	contentCache, _ := lru.New[contentSelectionKey, []ResolvedFont](contentSelectionCacheSize)
	return &FontManager{
		customPaths:      map[string]string{},
		fallbackRules:    map[string][]string{},
		familyCache:      map[string][]fontRecord{},
		missCache:        map[string]struct{}{},
		pathCache:        map[string][]fontRecord{},
		chainCache:       map[string]*ResolvedFontChain{},
		sourceCache:      cache,
		glyphCache:       glyphCache,
		contentCache:     contentCache,
		systemDirs:       systemFontDirs(),
		systemFallbacks:  systemFallbackFamilies(),
		lastResortSource: mustBuildLastResortSource(),
	}
}

func cloneResolvedFont(font *ResolvedFont) *ResolvedFont {
	if font == nil {
		return nil
	}
	clone := *font
	return &clone
}

func cloneResolvedChain(chain *ResolvedFontChain) *ResolvedFontChain {
	if chain == nil {
		return nil
	}
	clone := &ResolvedFontChain{
		Request:  chain.Request,
		Families: append([]string{}, chain.Families...),
		Primary:  cloneResolvedFont(chain.Primary),
		Sources:  append([]*text.GoTextFaceSource{}, chain.Sources...),
	}
	if len(chain.Fallbacks) > 0 {
		clone.Fallbacks = make([]*ResolvedFont, 0, len(chain.Fallbacks))
		for _, fallback := range chain.Fallbacks {
			clone.Fallbacks = append(clone.Fallbacks, cloneResolvedFont(fallback))
		}
	}
	return clone
}

func cloneResolvedFonts(fonts []*ResolvedFont, keepSource bool) []ResolvedFont {
	if len(fonts) == 0 {
		return nil
	}
	out := make([]ResolvedFont, 0, len(fonts))
	for _, font := range fonts {
		if font == nil {
			continue
		}
		clone := *font
		if !keepSource {
			clone.Source = nil
		}
		out = append(out, clone)
	}
	return out
}

func (m *FontManager) restoreResolvedFonts(entries []ResolvedFont) ([]*ResolvedFont, error) {
	fonts := make([]*ResolvedFont, 0, len(entries))
	for _, entry := range entries {
		font := entry
		font.Source = nil
		loaded, err := m.ensureResolvedFontLoaded(&font)
		if err != nil {
			return nil, err
		}
		fonts = append(fonts, loaded)
	}
	return fonts, nil
}

func (m *FontManager) resetDerivedCachesLocked() {
	m.chainCache = map[string]*ResolvedFontChain{}
	glyphCache, _ := lru.New[glyphCacheKey, bool](glyphSupportCacheSize)
	contentCache, _ := lru.New[contentSelectionKey, []ResolvedFont](contentSelectionCacheSize)
	m.glyphCache = glyphCache
	m.contentCache = contentCache
}

func systemFontDirs() []string {
	dirs := tdfont.DefaultFontDirs()
	out := make([]string, 0, len(dirs))
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		dir = normalizePathInput(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

func systemFallbackFamilies() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"Segoe UI",
			"Microsoft YaHei UI",
			"Microsoft YaHei",
			"Arial",
			"Segoe UI Symbol",
			"Arial Unicode MS",
		}
	case "darwin":
		return []string{
			"SF Pro",
			"PingFang SC",
			"Hiragino Sans GB",
			"Apple Symbols",
			"Helvetica Neue",
		}
	default:
		return []string{
			"Noto Sans",
			"Noto Sans CJK SC",
			"Noto Sans Symbols 2",
			"Source Han Sans SC",
			"DejaVu Sans",
			"WenQuanYi Micro Hei",
		}
	}
}

func DefaultFamilies() []string {
	return append([]string{}, systemFallbackFamilies()...)
}

func DefaultRequest() FontRequest {
	return FontRequest{
		Families: DefaultFamilies(),
		Weight:   WeightMedium,
	}
}

func mustBuildLastResortSource() *text.GoTextFaceSource {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	return source
}

func (m *FontManager) RegisterFallback(userRules map[string][]string) {
	if len(userRules) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for family, rules := range userRules {
		key := normalizeName(family)
		if key == "" {
			continue
		}
		m.fallbackRules[key] = normalizeFamilies(rules)
	}
	m.resetDerivedCachesLocked()
}

func (m *FontManager) RegisterCustomFontPath(name, path string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("font alias is empty")
	}

	path = normalizePathInput(path)
	if !fileExists(path) {
		return fmt.Errorf("font file %q not found", path)
	}

	records, err := inspectFontFile(path)
	if err != nil {
		return err
	}
	aliased := withAlias(records, name)

	m.mu.Lock()
	defer m.mu.Unlock()

	key := normalizeName(name)
	m.customPaths[key] = path
	m.familyCache[key] = append([]fontRecord{}, aliased...)
	delete(m.missCache, key)
	for _, record := range aliased {
		m.pathCache[filepath.Clean(record.Path)] = append([]fontRecord{}, records...)
	}
	m.resetDerivedCachesLocked()
	return nil
}

func withAlias(records []fontRecord, alias string) []fontRecord {
	out := make([]fontRecord, 0, len(records))
	for _, record := range records {
		clone := record
		if alias != "" {
			clone.Aliases = append(clone.Aliases, alias)
		}
		out = append(out, clone)
	}
	return out
}

func (m *FontManager) SetLastResortPath(path string) error {
	path = normalizePathInput(path)
	if path == "" {
		return errors.New("last resort path is empty")
	}
	if !fileExists(path) {
		return fmt.Errorf("font file %q not found", path)
	}

	records, err := inspectFontFile(path)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("font file %q has no usable face", path)
	}

	source, err := m.loadSource(records[0])
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.lastResortPath = path
	m.lastResortSource = source
	m.resetDerivedCachesLocked()
	m.mu.Unlock()
	return nil
}

func (m *FontManager) ResolveChain(req FontRequest) (*ResolvedFontChain, error) {
	req = req.Normalized()
	cacheKey := req.CacheKey()

	m.mu.RLock()
	if cached, ok := m.chainCache[cacheKey]; ok && cached != nil {
		clone := cloneResolvedChain(cached)
		m.mu.RUnlock()
		return clone, nil
	}
	m.mu.RUnlock()

	families := m.buildFamilyChain(req)

	resolved := &ResolvedFontChain{
		Request:  req,
		Families: append([]string{}, families...),
	}

	used := map[string]struct{}{}
	for _, family := range families {
		font, err := m.resolveFontForFamilyDescriptor(family, req)
		if err != nil || font == nil {
			continue
		}
		if _, ok := used[font.entryKey()]; ok {
			continue
		}
		used[font.entryKey()] = struct{}{}
		if resolved.Primary == nil {
			resolved.Primary = font
		} else {
			resolved.Fallbacks = append(resolved.Fallbacks, font)
		}
	}

	if resolved.Primary == nil {
		lastResort := m.lastResortFace()
		resolved.Primary = lastResort
		resolved.Families = append(resolved.Families, lastResortFamily)
		m.mu.Lock()
		m.chainCache[cacheKey] = cloneResolvedChain(resolved)
		m.mu.Unlock()
		return cloneResolvedChain(resolved), nil
	}

	lastResort := m.lastResortFace()
	if lastResort != nil {
		if _, ok := used[lastResort.entryKey()]; !ok {
			resolved.Fallbacks = append(resolved.Fallbacks, lastResort)
			resolved.Families = append(resolved.Families, lastResortFamily)
		}
	}

	m.mu.Lock()
	m.chainCache[cacheKey] = cloneResolvedChain(resolved)
	m.mu.Unlock()
	return cloneResolvedChain(resolved), nil
}

func (m *FontManager) GetFace(req FontRequest, size float64) (text.Face, error) {
	return m.GetFaceForText(req, size, "")
}

func (m *FontManager) GetFaceForText(req FontRequest, size float64, content string) (text.Face, error) {
	if size <= 0 {
		return nil, errors.New("font size must be positive")
	}

	chain, err := m.ResolveChain(req)
	if err != nil {
		return nil, err
	}
	if chain.Primary == nil {
		return nil, errors.New("font chain resolved to zero fonts")
	}

	orderedFonts, err := m.loadFontsForContent(chain, content)
	if err != nil {
		return nil, err
	}
	faces := make([]text.Face, 0, len(orderedFonts))
	for _, font := range orderedFonts {
		if font == nil || font.Source == nil {
			continue
		}
		faces = append(faces, &text.GoTextFace{Source: font.Source, Size: size})
	}
	if len(faces) == 0 {
		return nil, errors.New("font chain contains no usable face")
	}
	if len(faces) == 1 {
		return faces[0], nil
	}
	return text.NewMultiFace(faces...)
}

func (m *FontManager) loadFontsForContent(chain *ResolvedFontChain, content string) ([]*ResolvedFont, error) {
	if chain == nil || chain.Primary == nil {
		return nil, errors.New("resolved chain is empty")
	}
	if content != "" && m != nil && m.contentCache != nil {
		cacheKey := contentSelectionKey{
			request: chain.Request.CacheKey(),
			content: content,
		}
		m.mu.Lock()
		cached, ok := m.contentCache.Get(cacheKey)
		m.mu.Unlock()
		if ok {
			return m.restoreResolvedFonts(cached)
		}
	}

	allFonts := make([]*ResolvedFont, 0, 1+len(chain.Fallbacks))
	allFonts = append(allFonts, chain.Primary)
	allFonts = append(allFonts, chain.Fallbacks...)

	ordered := make([]*ResolvedFont, 0, len(allFonts))
	loaded := map[string]*ResolvedFont{}

	loadFont := func(font *ResolvedFont) (*ResolvedFont, error) {
		if font == nil {
			return nil, nil
		}
		if cached, ok := loaded[font.entryKey()]; ok {
			return cached, nil
		}
		loadedFont, err := m.ensureResolvedFontLoaded(font)
		if err != nil {
			return nil, err
		}
		loaded[loadedFont.entryKey()] = loadedFont
		ordered = append(ordered, loadedFont)
		return loadedFont, nil
	}

	primary, err := loadFont(chain.Primary)
	if err != nil {
		return nil, err
	}
	if primary == nil || primary.Source == nil {
		return nil, errors.New("primary font source is nil")
	}
	if content == "" {
		return ordered, nil
	}

	seenRunes := map[rune]struct{}{}
	for _, r := range content {
		if _, ok := seenRunes[r]; ok {
			continue
		}
		seenRunes[r] = struct{}{}
		if m.loadedFontsHaveRune(ordered, r) {
			continue
		}

		found := false
		for _, fallback := range chain.Fallbacks {
			loadedFallback, err := loadFont(fallback)
			if err != nil || loadedFallback == nil || loadedFallback.Source == nil {
				continue
			}
			if m.fontHasRune(loadedFallback, r) {
				found = true
				break
			}
		}
		if !found {
			lastResort, err := loadFont(m.lastResortFace())
			if err == nil && lastResort != nil && lastResort.Source != nil {
				_ = m.fontHasRune(lastResort, r)
			}
		}
	}

	if content != "" && m != nil && m.contentCache != nil {
		cacheKey := contentSelectionKey{
			request: chain.Request.CacheKey(),
			content: content,
		}
		m.mu.Lock()
		m.contentCache.Add(cacheKey, cloneResolvedFonts(ordered, false))
		m.mu.Unlock()
	}

	return ordered, nil
}

func (m *FontManager) loadedFontsHaveRune(fonts []*ResolvedFont, r rune) bool {
	for _, font := range fonts {
		if font == nil || font.Source == nil {
			continue
		}
		if m.fontHasRune(font, r) {
			return true
		}
	}
	return false
}

func (m *FontManager) fontHasRune(font *ResolvedFont, r rune) bool {
	if m == nil || font == nil || r == 0 {
		return false
	}
	key := glyphCacheKey{
		entry: font.entryKey(),
		r:     r,
	}
	if m.glyphCache != nil {
		m.mu.Lock()
		cached, ok := m.glyphCache.Get(key)
		m.mu.Unlock()
		if ok {
			return cached
		}
	}

	loadedFont := font
	if loadedFont.Source == nil {
		var err error
		loadedFont, err = m.ensureResolvedFontLoaded(font)
		if err != nil || loadedFont == nil || loadedFont.Source == nil {
			return false
		}
	}
	supported := sourceHasRune(loadedFont.Source, r)
	if m.glyphCache != nil {
		m.mu.Lock()
		m.glyphCache.Add(key, supported)
		m.mu.Unlock()
	}
	return supported
}

func (m *FontManager) buildFamilyChain(req FontRequest) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(req.Families)+len(m.systemFallbacks)+4)

	var expand func(string)
	expand = func(family string) {
		family = strings.TrimSpace(family)
		if family == "" {
			return
		}
		key := normalizeName(family)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, family)

		m.mu.RLock()
		rules := append([]string{}, m.fallbackRules[key]...)
		m.mu.RUnlock()
		for _, fallback := range rules {
			expand(fallback)
		}
	}

	for _, family := range req.Families {
		expand(family)
	}
	for _, family := range m.systemFallbacks {
		expand(family)
	}
	return out
}

func (m *FontManager) resolveFontForFamilyDescriptor(family string, req FontRequest) (*ResolvedFont, error) {
	if normalizeName(family) == normalizeName(lastResortFamily) {
		return m.lastResortFace(), nil
	}

	candidates, err := m.lookupFamilyCandidates(family)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("font family %q not found", family)
	}

	record, ok := bestMatchingRecord(candidates, family, req)
	if !ok {
		return nil, fmt.Errorf("font family %q has no usable face", family)
	}

	return &ResolvedFont{
		Path:            record.Path,
		CollectionIndex: record.CollectionIndex,
		Family:          record.resolvedFamily(family),
		Style:           record.Style,
		Weight:          record.Weight,
		Italic:          record.Italic,
	}, nil
}

func (m *FontManager) lastResortFace() *ResolvedFont {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &ResolvedFont{
		Path:   m.lastResortPath,
		Family: lastResortFamily,
		Style:  "Regular",
		Weight: WeightRegular,
		Source: m.lastResortSource,
	}
}

func (m *FontManager) ensureResolvedFontLoaded(font *ResolvedFont) (*ResolvedFont, error) {
	if font == nil {
		return nil, nil
	}
	if font.Source != nil {
		return font, nil
	}
	if normalizeName(font.Family) == normalizeName(lastResortFamily) {
		font.Source = m.lastResortSource
		return font, nil
	}
	source, err := m.loadSource(fontRecord{
		Path:            font.Path,
		CollectionIndex: font.CollectionIndex,
		Family:          font.Family,
		Style:           font.Style,
		Weight:          font.Weight,
		Italic:          font.Italic,
	})
	if err != nil {
		return nil, err
	}
	clone := *font
	clone.Source = source
	return &clone, nil
}

func bestMatchingRecord(records []fontRecord, requestedFamily string, req FontRequest) (fontRecord, bool) {
	bestScore := -1 << 30
	var best fontRecord
	ok := false
	for _, record := range records {
		score := styleScore(record.Weight, record.Italic, req.Weight, req.Italic)
		score += familyScore(record, requestedFamily, req.Families)
		if !ok || score > bestScore {
			bestScore = score
			best = record
			ok = true
		}
	}
	return best, ok
}

func familyScore(record fontRecord, requested string, families []string) int {
	best := 0
	match := func(candidate string, index int) {
		score := 0
		switch {
		case record.matchesFamily(candidate):
			score = 240 - index*8
		case record.familyMatchLevel(candidate) == familyMatchPrefix:
			score = 96 - index*6
		}
		if score > best {
			best = score
		}
	}
	if requested != "" {
		match(requested, 0)
	}
	for idx, family := range families {
		match(family, idx)
	}
	return best
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

func (m *FontManager) lookupFamilyCandidates(family string) ([]fontRecord, error) {
	key := normalizeName(family)
	if key == "" {
		return nil, nil
	}

	m.mu.RLock()
	if cached, ok := m.familyCache[key]; ok {
		out := append([]fontRecord{}, cached...)
		m.mu.RUnlock()
		return out, nil
	}
	if _, ok := m.missCache[key]; ok {
		m.mu.RUnlock()
		return nil, nil
	}
	customPath := m.customPaths[key]
	m.mu.RUnlock()

	if customPath != "" {
		records, err := inspectFontFile(customPath)
		if err != nil {
			return nil, err
		}
		records = withAlias(records, family)
		m.mu.Lock()
		m.familyCache[key] = append([]fontRecord{}, records...)
		m.mu.Unlock()
		return records, nil
	}

	records, err := m.searchFamilyOnDemand(family)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(records) == 0 {
		m.missCache[key] = struct{}{}
		return nil, nil
	}
	m.familyCache[key] = append([]fontRecord{}, records...)
	delete(m.missCache, key)
	return append([]fontRecord{}, records...), nil
}

func (m *FontManager) searchFamilyOnDemand(family string) ([]fontRecord, error) {
	if matches, handled, err := m.searchFamilyPlatform(family); handled {
		return matches, err
	}

	return m.searchFamilyByWalkingDirs(family)
}

func (m *FontManager) searchFamilyByWalkingDirs(family string) ([]fontRecord, error) {
	target := normalizeName(family)
	if target == "" {
		return nil, nil
	}

	var exactMatches []fontRecord
	var prefixMatches []fontRecord
	seen := map[string]struct{}{}

	for _, dir := range m.systemDirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !supportsIndexedFontFile(path) {
				return nil
			}

			path = filepath.Clean(path)
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}

			records, err := m.inspectCachedPath(path)
			if err != nil || len(records) == 0 {
				return nil
			}

			for _, record := range records {
				switch record.familyMatchLevel(family) {
				case familyMatchExact:
					exactMatches = append(exactMatches, record)
				case familyMatchPrefix:
					prefixMatches = append(prefixMatches, record)
				}
			}
			return nil
		})
	}

	records := append(exactMatches, prefixMatches...)
	return filterRecordsByFamily(records, family), nil
}

func filterRecordsByFamily(records []fontRecord, family string) []fontRecord {
	if len(records) == 0 {
		return nil
	}

	exactMatches := make([]fontRecord, 0, len(records))
	prefixMatches := make([]fontRecord, 0, len(records))
	for _, record := range records {
		switch record.familyMatchLevel(family) {
		case familyMatchExact:
			exactMatches = append(exactMatches, record)
		case familyMatchPrefix:
			prefixMatches = append(prefixMatches, record)
		}
	}

	matches := exactMatches
	if len(matches) == 0 {
		matches = prefixMatches
	}
	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if normalizeName(matches[i].Family) == normalizeName(matches[j].Family) {
			if matches[i].Weight == matches[j].Weight {
				if matches[i].Italic == matches[j].Italic {
					if matches[i].Path == matches[j].Path {
						return matches[i].CollectionIndex < matches[j].CollectionIndex
					}
					return matches[i].Path < matches[j].Path
				}
				return !matches[i].Italic && matches[j].Italic
			}
			return matches[i].Weight < matches[j].Weight
		}
		return normalizeName(matches[i].Family) < normalizeName(matches[j].Family)
	})

	out := make([]fontRecord, 0, len(matches))
	for _, record := range matches {
		if record.familyMatchLevel(family) == familyMatchNone {
			continue
		}
		out = append(out, record)
	}
	return out
}

func (m *FontManager) inspectCachedPath(path string) ([]fontRecord, error) {
	path = filepath.Clean(path)

	m.mu.RLock()
	if records, ok := m.pathCache[path]; ok {
		out := append([]fontRecord{}, records...)
		m.mu.RUnlock()
		return out, nil
	}
	m.mu.RUnlock()

	records, err := inspectFontFile(path)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.pathCache[path] = append([]fontRecord{}, records...)
	m.mu.Unlock()
	return records, nil
}

func inspectFontFile(path string) ([]fontRecord, error) {
	mapped, err := mmapFile(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = mapped.Unmap()
	}()

	count := fontFaceCount(mapped)
	if count < 1 {
		count = 1
	}

	records := make([]fontRecord, 0, count)
	for i := 0; i < count; i++ {
		sfnt, err := tdfont.ParseSFNT(mapped, i)
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
		records = append(records, fontRecord{
			Path:            filepath.Clean(path),
			CollectionIndex: i,
			Family:          family,
			Style:           style,
			Weight:          weight,
			Italic:          italic,
		})
	}
	return records, nil
}

func (m *FontManager) loadSource(record fontRecord) (*text.GoTextFaceSource, error) {
	key := filepath.Clean(record.Path)
	asset, ok := m.sourceCache.Get(key)
	if !ok || asset == nil {
		created, err := loadMappedFontFile(key)
		if err != nil {
			return nil, err
		}
		m.sourceCache.Add(key, created)
		asset = created
	}

	if record.CollectionIndex < 0 || record.CollectionIndex >= len(asset.sources) {
		return nil, fmt.Errorf("font index %d out of range for %s", record.CollectionIndex, record.Path)
	}
	return asset.sources[record.CollectionIndex], nil
}

func loadMappedFontFile(path string) (*loadedFontFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	mapped, err := mmapFile(path)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(mapped)
	var sources []*text.GoTextFaceSource
	if isFontCollectionData(mapped) {
		sources, err = text.NewGoTextFaceSourcesFromCollection(reader)
	} else {
		source, singleErr := text.NewGoTextFaceSource(reader)
		if singleErr != nil {
			_ = mapped.Unmap()
			return nil, singleErr
		}
		sources = []*text.GoTextFaceSource{source}
	}
	if err != nil {
		_ = mapped.Unmap()
		return nil, err
	}
	return &loadedFontFile{
		path:    path,
		size:    info.Size(),
		mapped:  mapped,
		sources: sources,
	}, nil
}

func (m *FontManager) Stats() FontManagerStats {
	if m == nil || m.sourceCache == nil {
		return FontManagerStats{}
	}

	stats := FontManagerStats{}
	for _, key := range m.sourceCache.Keys() {
		value, ok := m.sourceCache.Peek(key)
		if !ok || value == nil {
			continue
		}
		stats.LoadedFiles++
		stats.MappedBytes += value.size
	}
	return stats
}

func mmapFile(path string) (mmap.MMap, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mapped, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return mapped, nil
}

func supportsIndexedFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc":
		return true
	default:
		return false
	}
}

func isFontCollectionData(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "ttcf"
}

func fontFaceCount(data []byte) int {
	if len(data) < 12 || !isFontCollectionData(data) {
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

func formatStyle(weight Weight, italic bool) string {
	switch normalizeWeight(weight) {
	case WeightThin:
		weight = WeightThin
	case WeightExtraLight:
		weight = WeightExtraLight
	case WeightLight:
		weight = WeightLight
	case WeightRegular:
		weight = WeightRegular
	case WeightMedium:
		weight = WeightMedium
	case WeightSemiBold:
		weight = WeightSemiBold
	case WeightBold:
		weight = WeightBold
	case WeightExtraBold:
		weight = WeightExtraBold
	case WeightBlack:
		weight = WeightBlack
	default:
		weight = WeightRegular
	}

	style := map[Weight]string{
		WeightThin:       "Thin",
		WeightExtraLight: "ExtraLight",
		WeightLight:      "Light",
		WeightRegular:    "Regular",
		WeightMedium:     "Medium",
		WeightSemiBold:   "SemiBold",
		WeightBold:       "Bold",
		WeightExtraBold:  "ExtraBold",
		WeightBlack:      "Black",
	}[weight]
	if italic {
		style += " Italic"
	}
	return strings.TrimSpace(style)
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

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", ",", "")
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
