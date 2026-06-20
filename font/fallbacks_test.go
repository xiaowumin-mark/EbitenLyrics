package font

import (
	"runtime"
	"strings"
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

func TestBuildFamilyChainKeepsExplicitFamiliesBeforeRegisteredFallbacks(t *testing.T) {
	manager := NewFontManager(2)
	manager.systemFallbacks = []string{"System Fallback"}
	manager.RegisterFallback(map[string][]string{
		"A": {"C"},
		"B": {"E"},
		"C": {"D"},
	})

	chain := manager.buildFamilyChain(FontRequest{
		Families: []string{"A", "B"},
		Weight:   WeightRegular,
	})
	want := []string{"A", "B", "C", "D", "E", "System Fallback"}
	if strings.Join(chain, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected chain order:\n got %#v\nwant %#v", chain, want)
	}
}

func TestParseRequestRequireCJK(t *testing.T) {
	manager := NewFontManager(2)
	req, err := manager.ParseRequest(FontRequest{
		Families: []string{"A"},
		Weight:   WeightRegular,
	}, map[string]any{
		"requireCJK": true,
	})
	if err != nil {
		t.Fatalf("parse request failed: %v", err)
	}
	if !req.RequireCJK {
		t.Fatalf("expected RequireCJK to be true")
	}
	if !strings.Contains(req.CacheKey(), "|true") {
		t.Fatalf("cache key should include RequireCJK: %q", req.CacheKey())
	}
}

func TestSFProChineseFallsBackToPingFangWhenAvailable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows font registry test")
	}

	manager := NewFontManager(64)
	if records, err := manager.lookupFamilyCandidates("SF Pro"); err != nil || len(records) == 0 {
		t.Skip("SF Pro is not available")
	}
	if records, err := manager.lookupFamilyCandidates("PingFang"); err != nil || len(records) == 0 {
		t.Skip("PingFang is not available")
	}

	req := FontRequest{
		Families:   []string{"SF Pro"},
		Weight:     WeightExtraBold,
		RequireCJK: true,
	}
	chain, err := manager.ResolveChain(req)
	if err != nil {
		t.Fatalf("resolve chain failed: %v", err)
	}
	if chain == nil || chain.Primary == nil {
		t.Fatalf("expected primary font")
	}

	primary, err := manager.ensureResolvedFontLoaded(chain.Primary)
	if err != nil {
		t.Fatalf("load primary failed: %v", err)
	}
	if manager.fontHasRune(primary, '中') {
		return
	}

	fonts, err := manager.loadFontsForContent(chain, cjkProbeText)
	if err != nil {
		t.Fatalf("load content fonts failed: %v", err)
	}
	for _, font := range fonts {
		if font == nil || !manager.fontHasRune(font, '中') {
			continue
		}
		if !strings.Contains(normalizeName(font.Family), "pingfang") {
			t.Fatalf("expected PingFang CJK fallback, got family=%q path=%q", font.Family, font.Path)
		}
		return
	}
	t.Fatalf("expected a CJK fallback for SF Pro")
}
