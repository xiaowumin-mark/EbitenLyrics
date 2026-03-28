package font

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultFontConfigPath = "config/font.json"

func (m *FontManager) LoadRequestFromFile(path string, base FontRequest) (FontRequest, error) {
	req := base.Normalized()
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultFontConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return req, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return req, fmt.Errorf("parse font config %s failed: %w", path, err)
	}
	if nested, ok := raw["font"]; ok {
		if m, ok := nested.(map[string]any); ok {
			raw = m
		}
	}

	for _, key := range []string{"path", "fontPath", "font_file", "file"} {
		v, ok := raw[key]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		raw[key] = resolvePathFromBase(s, filepath.Dir(path))
		break
	}

	req, err = m.ParseRequest(req, raw)
	if err != nil {
		return req, fmt.Errorf("invalid font config %s: %w", path, err)
	}
	return req, nil
}

func LoadFontRequestFromFile(path string, base FontRequest) (FontRequest, error) {
	return DefaultManager().LoadRequestFromFile(path, base)
}
