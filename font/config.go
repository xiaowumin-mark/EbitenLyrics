package font

// 文件说明：加载运行时字体配置文件。
// 主要职责：把 JSON 配置合并到字体解析选项中。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	var rawPath string
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
		rawPath = s
		break
	}

	parsed, err := ParseResolveOptions(opts, raw)
	if err != nil {
		return opts, fmt.Errorf("invalid font config %s: %w", path, err)
	}
	if rawPath != "" {
		parsed.Path = resolvePathFromBase(rawPath, filepath.Dir(path))
	}
	return parsed, nil
}
