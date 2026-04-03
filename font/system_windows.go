//go:build windows

package font

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsFontsRegistryPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

func (m *FontManager) searchFamilyPlatform(family string) ([]fontRecord, bool, error) {
	records, err := m.windowsSystemFontIndex()
	if err != nil {
		return nil, false, err
	}
	if len(records) == 0 {
		return nil, false, nil
	}
	return filterRecordsByFamily(records, family), true, nil
}

func (m *FontManager) windowsSystemFontIndex() ([]fontRecord, error) {
	m.systemIndexOnce.Do(func() {
		paths, err := loadWindowsFontRegistryPaths()
		if err != nil {
			m.systemIndexErr = err
			return
		}

		index := make([]fontRecord, 0, len(paths))
		for _, path := range paths {
			records, err := m.inspectCachedPath(path)
			if err != nil || len(records) == 0 {
				continue
			}
			index = append(index, records...)
		}
		m.systemIndex = index
	})

	if m.systemIndexErr != nil {
		return nil, m.systemIndexErr
	}
	return append([]fontRecord{}, m.systemIndex...), nil
}

func loadWindowsFontRegistryPaths() ([]string, error) {
	fontDirs := windowsFontDirectories()
	seen := map[string]struct{}{}
	out := make([]string, 0, 512)

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
				out = append(out, path)
			}
		}
		_ = key.Close()
	}

	return out, nil
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
