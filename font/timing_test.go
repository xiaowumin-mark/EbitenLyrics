package font

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFontTimingProfile(t *testing.T) {
	manager := NewFontManager(16)
	cfgPath := filepath.Join("..", "config", "font.json")
	req, err := manager.LoadRequestFromFile(cfgPath, DefaultRequest())
	if err != nil {
		req = DefaultRequest()
	}

	measure := func(label string, fn func() error) time.Duration {
		t.Helper()
		start := time.Now()
		if err := fn(); err != nil {
			t.Fatalf("%s failed: %v", label, err)
		}
		d := time.Since(start)
		t.Logf("%s took %s", label, d)
		return d
	}

	measure("cold-resolve-chain", func() error {
		chain, err := manager.ResolveChain(req)
		if err != nil {
			return err
		}
		if chain == nil || chain.Primary == nil {
			t.Fatal("resolve chain returned no primary font")
		}
		return nil
	})

	measure("cold-get-face-latin", func() error {
		face, err := manager.GetFaceForText(req, 48, benchLatin)
		if err != nil {
			return err
		}
		if face == nil {
			t.Fatal("latin face is nil")
		}
		return nil
	})

	measure("cold-get-face-mixed", func() error {
		face, err := manager.GetFaceForText(req, 48, benchMixed)
		if err != nil {
			return err
		}
		if face == nil {
			t.Fatal("mixed face is nil")
		}
		return nil
	})

	measure("warm-resolve-chain", func() error {
		chain, err := manager.ResolveChain(req)
		if err != nil {
			return err
		}
		if chain == nil || chain.Primary == nil {
			t.Fatal("resolve chain returned no primary font")
		}
		return nil
	})

	measure("warm-get-face-latin", func() error {
		face, err := manager.GetFaceForText(req, 48, benchLatin)
		if err != nil {
			return err
		}
		if face == nil {
			t.Fatal("latin face is nil")
		}
		return nil
	})

	measure("warm-get-face-mixed", func() error {
		face, err := manager.GetFaceForText(req, 48, benchMixed)
		if err != nil {
			return err
		}
		if face == nil {
			t.Fatal("mixed face is nil")
		}
		return nil
	})

	stats := manager.Stats()
	t.Logf("loaded_files=%d mapped_font=%s", stats.LoadedFiles, bytesIEC64(stats.MappedBytes))
}
