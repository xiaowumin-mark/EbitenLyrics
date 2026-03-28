package font

import (
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"testing"
)

type memorySnapshot struct {
	HeapAlloc  uint64
	HeapInuse  uint64
	HeapSys    uint64
	MappedFont int64
	LoadedFont int
}

func takeMemorySnapshot(m *FontManager) memorySnapshot {
	runtime.GC()
	debug.FreeOSMemory()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	stats := m.Stats()
	return memorySnapshot{
		HeapAlloc:  ms.HeapAlloc,
		HeapInuse:  ms.HeapInuse,
		HeapSys:    ms.HeapSys,
		MappedFont: stats.MappedBytes,
		LoadedFont: stats.LoadedFiles,
	}
}

func bytesIEC(v uint64) string {
	const unit = 1024
	if v < unit {
		return stringInt(v) + "B"
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return floatString(float64(v)/float64(div), 1) + string("KMGTPE"[exp]) + "iB"
}

func bytesIEC64(v int64) string {
	if v <= 0 {
		return "0B"
	}
	return bytesIEC(uint64(v))
}

func signedBytesIEC(delta int64) string {
	if delta < 0 {
		return "-" + bytesIEC64(-delta)
	}
	return "+" + bytesIEC64(delta)
}

func stringInt(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func floatString(v float64, prec int) string {
	return strconv.FormatFloat(v, 'f', prec, 64)
}

func logStage(t *testing.T, stage string, before, after memorySnapshot) {
	t.Helper()
	t.Logf(
		"%s heap_alloc=%s (%+s) heap_inuse=%s (%+s) heap_sys=%s (%+s) mapped_font=%s (%+s) loaded_files=%d (%+d)",
		stage,
		bytesIEC(after.HeapAlloc),
		signedBytesIEC(int64(after.HeapAlloc)-int64(before.HeapAlloc)),
		bytesIEC(after.HeapInuse),
		signedBytesIEC(int64(after.HeapInuse)-int64(before.HeapInuse)),
		bytesIEC(after.HeapSys),
		signedBytesIEC(int64(after.HeapSys)-int64(before.HeapSys)),
		bytesIEC64(after.MappedFont),
		signedBytesIEC(after.MappedFont-before.MappedFont),
		after.LoadedFont,
		after.LoadedFont-before.LoadedFont,
	)
}

func TestFontInitializationMemoryProfile(t *testing.T) {
	manager := NewFontManager(16)
	base := takeMemorySnapshot(manager)

	configPath := filepath.Join("..", "config", "font.json")
	req, err := manager.LoadRequestFromFile(configPath, DefaultRequest())
	if err != nil {
		t.Fatalf("load request failed: %v", err)
	}
	afterLoadConfig := takeMemorySnapshot(manager)
	logStage(t, "load-config", base, afterLoadConfig)

	chain, err := manager.ResolveChain(req)
	if err != nil {
		t.Fatalf("resolve chain failed: %v", err)
	}
	if chain.Primary == nil {
		t.Fatalf("primary font is nil")
	}
	afterResolve := takeMemorySnapshot(manager)
	logStage(t, "resolve-chain", afterLoadConfig, afterResolve)
	t.Logf("resolved primary family=%q style=%q weight=%d path=%s", chain.Primary.Family, chain.Primary.Style, chain.Primary.Weight, chain.Primary.Path)

	face, err := manager.GetFace(req, 48)
	if err != nil {
		t.Fatalf("get face failed: %v", err)
	}
	if face == nil {
		t.Fatalf("face is nil")
	}
	afterGetFace := takeMemorySnapshot(manager)
	logStage(t, "get-face", afterResolve, afterGetFace)
}
