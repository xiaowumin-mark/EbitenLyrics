package font

import (
	"runtime"
	"testing"
)

func containsFamily(families []string, target string) bool {
	for _, family := range families {
		if family == target {
			return true
		}
	}
	return false
}

func TestDefaultFamiliesIncludeLanguageFallbacks(t *testing.T) {
	families := DefaultFamilies()
	if len(families) == 0 {
		t.Fatal("default fallback families should not be empty")
	}

	switch runtime.GOOS {
	case "windows":
		for _, family := range []string{"Malgun Gothic", "Yu Gothic UI", "Segoe UI Emoji"} {
			if !containsFamily(families, family) {
				t.Fatalf("windows fallback chain should include %q", family)
			}
		}
	case "darwin":
		for _, family := range []string{"Apple SD Gothic Neo", "Apple Color Emoji"} {
			if !containsFamily(families, family) {
				t.Fatalf("darwin fallback chain should include %q", family)
			}
		}
	default:
		for _, family := range []string{"Noto Sans CJK KR", "Noto Color Emoji"} {
			if !containsFamily(families, family) {
				t.Fatalf("linux fallback chain should include %q", family)
			}
		}
	}
}
