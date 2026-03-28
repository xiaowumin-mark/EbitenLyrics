package font

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *FontManager) ParseRequest(base FontRequest, cfg map[string]any) (FontRequest, error) {
	req := base.Normalized()
	if cfg == nil {
		return req, nil
	}

	if v, ok := cfg["family"]; ok {
		families := parseFamiliesFromAny(v)
		if len(families) > 0 {
			req.Families = families
		}
	}
	if v, ok := cfg["families"]; ok {
		families := parseFamiliesFromAny(v)
		if len(families) > 0 {
			req.Families = families
		}
	}
	if v, ok := cfg["weight"]; ok {
		w, err := normalizeWeightValue(v)
		if err != nil {
			return req, err
		}
		req.Weight = w
	}
	if v, ok := cfg["italic"]; ok {
		req.Italic = normalizeBool(v, req.Italic)
	}

	var customPath string
	for _, key := range []string{"path", "fontPath", "font_file", "file"} {
		v, ok := cfg[key]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = normalizePathInput(s)
		if s != "" {
			customPath = s
			break
		}
	}
	if customPath != "" {
		alias := fmt.Sprintf("__custom_%s__", filepath.Base(customPath))
		if err := m.RegisterCustomFontPath(alias, customPath); err != nil {
			return req, err
		}
		req.Families = append([]string{alias}, req.Families...)
	}

	return req.Normalized(), nil
}

func ParseFontRequest(base FontRequest, cfg map[string]any) (FontRequest, error) {
	return DefaultManager().ParseRequest(base, cfg)
}

func (m *FontManager) ApplyEnvRequest(base FontRequest) (FontRequest, error) {
	cfg := map[string]any{}

	if family := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_FAMILY")); family != "" {
		cfg["families"] = family
	}
	if weightRaw := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_WEIGHT")); weightRaw != "" {
		cfg["weight"] = weightRaw
	}
	if italicRaw := strings.TrimSpace(strings.ToLower(os.Getenv("EBITENLYRICS_FONT_ITALIC"))); italicRaw != "" {
		cfg["italic"] = italicRaw
	}
	if path := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_PATH")); path != "" {
		cfg["path"] = path
	}

	return m.ParseRequest(base, cfg)
}

func ApplyEnvFontRequest(base FontRequest) (FontRequest, error) {
	return DefaultManager().ApplyEnvRequest(base)
}

func parseFamiliesFromAny(raw any) []string {
	switch v := raw.(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, p := range v {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func normalizeWeightValue(raw any) (Weight, error) {
	switch v := raw.(type) {
	case string:
		return ParseWeight(v)
	case int:
		return normalizeWeight(Weight(v)), nil
	case int32:
		return normalizeWeight(Weight(v)), nil
	case int64:
		return normalizeWeight(Weight(v)), nil
	case float32:
		return normalizeWeight(Weight(int(v))), nil
	case float64:
		return normalizeWeight(Weight(int(v))), nil
	default:
		return WeightRegular, fmt.Errorf("unsupported weight type: %T", raw)
	}
}

func normalizeBool(raw any, defaultValue bool) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		if s == "" {
			return defaultValue
		}
		if s == "1" || s == "true" || s == "yes" || s == "on" {
			return true
		}
		if s == "0" || s == "false" || s == "no" || s == "off" {
			return false
		}
		return defaultValue
	case int:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	default:
		return defaultValue
	}
}
