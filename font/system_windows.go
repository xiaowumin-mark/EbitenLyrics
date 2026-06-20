//go:build windows

package font

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsFontsRegistryPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

type windowsRegistryFontEntry struct {
	Name string
	Path string
}

func (m *FontManager) postInit() {
	go m.buildWindowsSystemIndex()
}

func (m *FontManager) searchFamilyPlatform(family string) ([]fontRecord, bool, error) {
	if records := m.peekWindowsSystemIndex(); len(records) > 0 {
		return filterRecordsByFamily(records, family), true, nil
	}

	records, err := m.searchFamilyInWindowsRegistry(family)
	if err != nil {
		return nil, true, err
	}
	if len(records) > 0 {
		return records, true, nil
	}

	records, err = m.searchFamilyInWindowsFontDirectoriesByName(family)
	if err != nil {
		return nil, true, err
	}
	if len(records) > 0 {
		return records, true, nil
	}

	if records := m.peekWindowsSystemIndex(); len(records) > 0 {
		return filterRecordsByFamily(records, family), true, nil
	}
	return nil, true, nil
}

func (m *FontManager) buildWindowsSystemIndex() {
	m.systemIndexOnce.Do(func() {
		entries, err := loadWindowsFontRegistryEntries()
		if err != nil {
			m.mu.Lock()
			m.systemIndexErr = err
			m.systemIndexReady = true
			m.mu.Unlock()
			return
		}

		index := make([]fontRecord, 0, len(entries))
		for _, entry := range entries {
			records, err := m.inspectCachedPath(entry.Path)
			if err != nil || len(records) == 0 {
				continue
			}
			index = append(index, records...)
		}

		m.mu.Lock()
		m.systemIndex = index
		m.systemIndexReady = true
		m.mu.Unlock()
	})
}

func (m *FontManager) peekWindowsSystemIndex() []fontRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.systemIndexReady || len(m.systemIndex) == 0 {
		return nil
	}
	return append([]fontRecord{}, m.systemIndex...)
}

func (m *FontManager) searchFamilyInWindowsRegistry(family string) ([]fontRecord, error) {
	target := normalizeName(family)
	if target == "" {
		return nil, nil
	}

	entries, err := loadWindowsFontRegistryEntries()
	if err != nil {
		return nil, err
	}

	candidates := make([]fontRecord, 0, 8)
	for _, entry := range entries {
		if !windowsRegistryNameLikelyMatches(entry.Name, target) {
			continue
		}

		records, err := m.inspectCachedPath(entry.Path)
		if err != nil || len(records) == 0 {
			continue
		}
		candidates = append(candidates, records...)
	}

	return filterRecordsByFamily(candidates, family), nil
}

func loadWindowsFontRegistryEntries() ([]windowsRegistryFontEntry, error) {
	fontDirs := windowsFontDirectories()
	seen := map[string]struct{}{}
	out := make([]windowsRegistryFontEntry, 0, 512)

	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		key, err := registry.OpenKey(root, windowsFontsRegistryPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		names, err := key.ReadValueNames(-1)
		if err == nil {
			for _, name := range names {
				raw, _, err := key.GetStringValue(name)
				if err != nil {
					continue
				}
				path := resolveWindowsRegistryFontPath(raw, fontDirs)
				if path == "" || !supportsIndexedFontFile(path) {
					continue
				}
				if _, ok := seen[path]; ok {
					continue
				}
				seen[path] = struct{}{}
				out = append(out, windowsRegistryFontEntry{
					Name: strings.TrimSpace(name),
					Path: path,
				})
			}
		}
		_ = key.Close()
	}

	return out, nil
}

func (m *FontManager) searchFamilyInWindowsFontDirectoriesByName(family string) ([]fontRecord, error) {
	target := normalizeName(family)
	if target == "" {
		return nil, nil
	}

	candidates := make([]fontRecord, 0, 8)
	seen := map[string]struct{}{}
	for _, dir := range windowsFontDirectories() {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !supportsIndexedFontFile(path) {
				return nil
			}
			if !windowsFontFileLikelyMatches(path, target) {
				return nil
			}

			path = filepath.Clean(path)
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}

			records, err := m.inspectCachedPath(path)
			if err != nil || len(records) == 0 {
				candidates = append(candidates, aliasRecordsFromFontFileName(path, family)...)
				return nil
			}
			candidates = append(candidates, withAlias(records, family)...)
			return nil
		})
	}

	return filterRecordsByFamily(candidates, family), nil
}

func windowsFontFileLikelyMatches(path, target string) bool {
	if target == "" {
		return false
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	normalized := normalizeName(base)
	if normalized == "" {
		return false
	}
	return normalized == target ||
		strings.HasPrefix(normalized, target) ||
		strings.Contains(normalized, target) ||
		strings.HasPrefix(target, normalized)
}

func aliasRecordsFromFontFileName(path, family string) []fontRecord {
	path = filepath.Clean(path)
	count := 1
	if mapped, err := mmapFile(path); err == nil {
		count = fontFaceCount(mapped)
		_ = mapped.Unmap()
	}
	if count < 1 {
		count = 1
	}

	weight, italic := fontStyleFromPath(path)
	records := make([]fontRecord, 0, count)
	for i := 0; i < count; i++ {
		records = append(records, fontRecord{
			Path:            path,
			CollectionIndex: i,
			Family:          family,
			Style:           formatStyle(weight, italic),
			Weight:          weight,
			Italic:          italic,
			Aliases:         []string{family},
		})
	}
	return records
}

func fontStyleFromPath(path string) (Weight, bool) {
	name := normalizeName(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	italic := strings.Contains(name, "italic")
	for _, candidate := range []struct {
		token  string
		weight Weight
	}{
		{"ultralight", WeightExtraLight},
		{"extralight", WeightExtraLight},
		{"semibold", WeightSemiBold},
		{"extrabold", WeightExtraBold},
		{"black", WeightBlack},
		{"heavy", WeightBlack},
		{"bold", WeightBold},
		{"medium", WeightMedium},
		{"light", WeightLight},
		{"thin", WeightThin},
	} {
		if strings.Contains(name, candidate.token) {
			return candidate.weight, italic
		}
	}
	return WeightRegular, italic
}

func windowsRegistryNameLikelyMatches(name, target string) bool {
	normalized := normalizeWindowsRegistryFontName(name)
	if normalized == "" || target == "" {
		return false
	}
	return normalized == target ||
		strings.HasPrefix(normalized, target) ||
		strings.Contains(normalized, target) ||
		strings.HasPrefix(target, normalized)
}

func normalizeWindowsRegistryFontName(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.Index(name, "("); idx >= 0 {
		name = strings.TrimSpace(name[:idx])
	}
	replacer := strings.NewReplacer(
		" truetype", "",
		" opentype", "",
		" variable", "",
	)
	name = replacer.Replace(strings.ToLower(name))
	return normalizeName(name)
}

func windowsFontDirectories() []string {
	candidates := []string{}
	if windir := normalizePathInput(os.Getenv("WINDIR")); windir != "" {
		candidates = append(candidates, filepath.Join(windir, "Fonts"))
	}
	if systemRoot := normalizePathInput(os.Getenv("SystemRoot")); systemRoot != "" {
		candidates = append(candidates, filepath.Join(systemRoot, "Fonts"))
	}
	if localAppData := normalizePathInput(os.Getenv("LOCALAPPDATA")); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "Microsoft", "Windows", "Fonts"))
	}
	candidates = append(candidates, `C:\Windows\Fonts`)

	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, dir := range candidates {
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

func resolveWindowsRegistryFontPath(raw string, fontDirs []string) string {
	path := normalizePathInput(os.ExpandEnv(strings.TrimSpace(raw)))
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		if fileExists(path) {
			return filepath.Clean(path)
		}
		return ""
	}

	for _, dir := range fontDirs {
		candidate := filepath.Join(dir, path)
		if fileExists(candidate) {
			return filepath.Clean(candidate)
		}
	}
	return ""
}
