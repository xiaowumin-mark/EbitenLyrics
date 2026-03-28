package font

// 文件说明：处理运行时传入的字体选项和环境变量覆盖。
// 主要职责：把动态配置规范化为统一的 `ResolveOptions`。

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseResolveOptions(base ResolveOptions, cfg map[string]any) (ResolveOptions, error) {
	opts := base
	if cfg == nil {
		return opts, nil
	}

	for _, key := range []string{"path", "fontPath", "font_file", "file"} {
		if v, ok := cfg[key]; ok {
			if s, ok := v.(string); ok {
				s = normalizePathInput(s)
				if s != "" {
					opts.Path = s
					break
				}
			}
		}
	}

	if v, ok := cfg["family"]; ok {
		families := parseFamiliesFromAny(v)
		if len(families) > 0 {
			opts.Families = families
		}
	}
	if v, ok := cfg["families"]; ok {
		families := parseFamiliesFromAny(v)
		if len(families) > 0 {
			opts.Families = families
		}
	}
	if v, ok := cfg["weight"]; ok {
		w, err := normalizeWeightValue(v)
		if err != nil {
			return opts, err
		}
		opts.Weight = w
	}
	if v, ok := cfg["italic"]; ok {
		opts.Italic = normalizeBool(v, opts.Italic)
	}
	if v, ok := cfg["requireCJK"]; ok {
		opts.RequireCJK = normalizeBool(v, opts.RequireCJK)
	}

	return opts, nil
}

func ApplyEnvResolveOptions(base ResolveOptions) ResolveOptions {
	opts := base
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
	if requireCJKRaw := strings.TrimSpace(strings.ToLower(os.Getenv("EBITENLYRICS_FONT_REQUIRE_CJK"))); requireCJKRaw != "" {
		cfg["requireCJK"] = requireCJKRaw
	}
	if parsed, err := ParseResolveOptions(opts, cfg); err == nil {
		opts = parsed
	}

	if path := strings.TrimSpace(os.Getenv("EBITENLYRICS_FONT_PATH")); path != "" {
		opts.Path = path
	}
	return opts
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
		_ = strconv.ErrSyntax
		return defaultValue
	}
}
