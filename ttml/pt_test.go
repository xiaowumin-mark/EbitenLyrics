package ttml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPT(t *testing.T) {
	fixture := filepath.Join("..", "Bejeweled.ttml")
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture not found: %s", fixture)
	}

	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture failed: %v", err)
	}

	tt, err := ParseTTML(string(data))
	if err != nil {
		t.Fatalf("parse TTML failed: %v", err)
	}

	jsonData, err := json.Marshal(tt)
	if err != nil {
		t.Fatalf("marshal result failed: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "Bejeweled.json")
	if err := os.WriteFile(outPath, jsonData, 0o644); err != nil {
		t.Fatalf("write output failed: %v", err)
	}
}
