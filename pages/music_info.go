package pages

import (
	"fmt"
	"strings"
)

func bottomLineCreatorTextFromMusic(data map[string]interface{}) string {
	artists := artistNamesFromMusic(data)
	if len(artists) == 0 {
		return ""
	}
	return "创作者：" + strings.Join(artists, "，")
}

func artistNamesFromMusic(data map[string]interface{}) []string {
	if data == nil {
		return nil
	}
	value, ok := firstMapValue(data, "artists", "artist", "singers", "authors", "creators")
	if !ok {
		return nil
	}
	return uniqueNonEmptyStrings(extractArtistNames(value))
}

func firstMapValue(data map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func extractArtistNames(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, extractArtistNames(item)...)
		}
		return out
	case []map[string]interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, extractArtistNames(item)...)
		}
		return out
	case map[string]interface{}:
		for _, key := range []string{"name", "artistName", "nickname", "title"} {
			if name := stringFromAny(v[key]); name != "" {
				return []string{name}
			}
		}
	case []string:
		return v
	case string:
		return splitArtistString(v)
	}
	return nil
}

func stringFromAny(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	}
	return ""
}

func splitArtistString(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', '，', '/', '、', ';', '；':
			return true
		}
		return false
	})
	if len(parts) <= 1 {
		return []string{value}
	}
	return parts
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
