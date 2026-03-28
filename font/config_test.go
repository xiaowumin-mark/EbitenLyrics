package font

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestLoadFontRequestFromFile_ResolveRelativePath(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	fontDir := filepath.Join(cfgDir, "fonts")
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		t.Fatalf("create font dir failed: %v", err)
	}

	fontPath := filepath.Join(fontDir, "Primary.ttf")
	if err := os.WriteFile(fontPath, goregular.TTF, 0o644); err != nil {
		t.Fatalf("write font failed: %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "font.json")
	cfg := `{"font":{"path":"fonts/Primary.ttf","weight":700}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	manager := NewFontManager(4)
	req, err := manager.LoadRequestFromFile(cfgPath, DefaultRequest())
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if req.Weight != WeightBold {
		t.Fatalf("unexpected weight: got %d want %d", req.Weight, WeightBold)
	}
	if len(req.Families) == 0 || req.Families[0] == "" {
		t.Fatalf("expected registered custom family alias")
	}

	chain, err := manager.ResolveChain(req)
	if err != nil {
		t.Fatalf("resolve chain failed: %v", err)
	}
	if chain.Primary == nil {
		t.Fatalf("expected primary font")
	}
	if filepath.Clean(chain.Primary.Path) != filepath.Clean(fontPath) {
		t.Fatalf("unexpected path: got %q want %q", chain.Primary.Path, fontPath)
	}
}

func TestLoadFontRequestFromFile_KeepBaseFamiliesWhenNoFamilyKey(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("create cfg dir failed: %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "font.json")
	cfg := `{"font":{"italic":true}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	base := FontRequest{
		Families: []string{"Inter"},
		Weight:   WeightMedium,
	}
	req, err := NewFontManager(4).LoadRequestFromFile(cfgPath, base)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if len(req.Families) != 1 || req.Families[0] != "Inter" {
		t.Fatalf("base families should stay unchanged: got %#v", req.Families)
	}
	if !req.Italic {
		t.Fatalf("unexpected italic: got %v want true", req.Italic)
	}
}
