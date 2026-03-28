package font

// 文件说明：字体配置加载测试。
// 主要职责：验证相对路径、别名文件和默认路径行为。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolveOptionsFromFile_ResolveRelativePath(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("create cfg dir failed: %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "font.json")
	cfg := `{"font":{"path":"fonts/Primary.ttf","weight":700}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	opts, err := LoadResolveOptionsFromFile(cfgPath, DefaultResolveOptions())
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	wantPath := filepath.Clean(filepath.Join(cfgDir, "fonts/Primary.ttf"))
	if opts.Path != wantPath {
		t.Fatalf("unexpected path: got %q want %q", opts.Path, wantPath)
	}
	if opts.Weight != WeightBold {
		t.Fatalf("unexpected weight: got %d want %d", opts.Weight, WeightBold)
	}
}

func TestLoadResolveOptionsFromFile_ResolveRelativePathAliasFile(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("create cfg dir failed: %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "font.json")
	cfg := `{"font":{"file":"./fonts/Fallback.ttc","italic":true}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	opts, err := LoadResolveOptionsFromFile(cfgPath, DefaultResolveOptions())
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	wantPath := filepath.Clean(filepath.Join(cfgDir, "fonts/Fallback.ttc"))
	if opts.Path != wantPath {
		t.Fatalf("unexpected path: got %q want %q", opts.Path, wantPath)
	}
	if !opts.Italic {
		t.Fatalf("unexpected italic: got %v want true", opts.Italic)
	}
}

func TestLoadResolveOptionsFromFile_KeepBasePathWhenNoPathKey(t *testing.T) {
	cfgDir := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("create cfg dir failed: %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "font.json")
	cfg := `{"font":{"family":"Noto Sans CJK SC"}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	base := DefaultResolveOptions()
	base.Path = filepath.Clean(filepath.Join(t.TempDir(), "preset.ttf"))

	opts, err := LoadResolveOptionsFromFile(cfgPath, base)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if opts.Path != base.Path {
		t.Fatalf("base path should stay unchanged: got %q want %q", opts.Path, base.Path)
	}
}
