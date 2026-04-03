package font

import (
	"path/filepath"
	"testing"
)

const (
	benchFontSize = 48
	benchLatin    = "We stay in motion, chasing fragments of the summer sky."
	benchMixed    = "We stay in motion 追逐夏夜的光，直到星星落下。"
	benchCJK      = "我们在夏夜里追逐星光，直到黎明慢慢降临。"
)

func benchmarkRequest(tb testing.TB, manager *FontManager) FontRequest {
	tb.Helper()
	cfgPath := filepath.Join("..", "config", "font.json")
	req, err := manager.LoadRequestFromFile(cfgPath, DefaultRequest())
	if err == nil {
		return req
	}
	return DefaultRequest()
}

func BenchmarkResolveChainWarm(b *testing.B) {
	manager := NewFontManager(16)
	req := benchmarkRequest(b, manager)

	if _, err := manager.ResolveChain(req); err != nil {
		b.Fatalf("resolve chain failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain, err := manager.ResolveChain(req)
		if err != nil {
			b.Fatalf("resolve chain failed: %v", err)
		}
		if chain == nil || chain.Primary == nil {
			b.Fatal("resolved chain has no primary font")
		}
	}
}

func BenchmarkGetFaceForTextLatinWarm(b *testing.B) {
	manager := NewFontManager(16)
	req := benchmarkRequest(b, manager)

	if _, err := manager.GetFaceForText(req, benchFontSize, benchLatin); err != nil {
		b.Fatalf("get face failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		face, err := manager.GetFaceForText(req, benchFontSize, benchLatin)
		if err != nil {
			b.Fatalf("get face failed: %v", err)
		}
		if face == nil {
			b.Fatal("face is nil")
		}
	}
}

func BenchmarkGetFaceForTextMixedWarm(b *testing.B) {
	manager := NewFontManager(16)
	req := benchmarkRequest(b, manager)

	if _, err := manager.GetFaceForText(req, benchFontSize, benchMixed); err != nil {
		b.Fatalf("get face failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		face, err := manager.GetFaceForText(req, benchFontSize, benchMixed)
		if err != nil {
			b.Fatalf("get face failed: %v", err)
		}
		if face == nil {
			b.Fatal("face is nil")
		}
	}
}

func BenchmarkGetFaceForTextCJKWarm(b *testing.B) {
	manager := NewFontManager(16)
	req := benchmarkRequest(b, manager)

	if _, err := manager.GetFaceForText(req, benchFontSize, benchCJK); err != nil {
		b.Fatalf("get face failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		face, err := manager.GetFaceForText(req, benchFontSize, benchCJK)
		if err != nil {
			b.Fatalf("get face failed: %v", err)
		}
		if face == nil {
			b.Fatal("face is nil")
		}
	}
}

func BenchmarkGlyphLookupPrimaryWarm(b *testing.B) {
	manager := NewFontManager(16)
	req := benchmarkRequest(b, manager)
	chain, err := manager.ResolveChain(req)
	if err != nil {
		b.Fatalf("resolve chain failed: %v", err)
	}
	primary, err := manager.ensureResolvedFontLoaded(chain.Primary)
	if err != nil {
		b.Fatalf("load primary failed: %v", err)
	}

	if !manager.fontHasRune(primary, 'A') {
		b.Skip("primary font does not contain benchmark rune")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !manager.fontHasRune(primary, 'A') {
			b.Fatal("expected rune support")
		}
	}
}
