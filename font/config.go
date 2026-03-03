package font

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const DefaultRuntimeFontConfigPath = "config/font.json"

//var DefaultRuntimeFontConfigPath = ""

func LoadResolveOptionsFromFile(path string, base ResolveOptions) (ResolveOptions, error) {
	opts := base
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultRuntimeFontConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return opts, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return opts, fmt.Errorf("parse font config %s failed: %w", path, err)
	}

	if nested, ok := raw["font"]; ok {
		if m, ok := nested.(map[string]any); ok {
			raw = m
		}
	}

	if v, ok := raw["path"]; ok {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				opts.Path = s
			}
		}
	}

	parsed, err := ParseResolveOptions(opts, raw)
	if err != nil {
		return opts, fmt.Errorf("invalid font config %s: %w", path, err)
	}
	return parsed, nil
}
