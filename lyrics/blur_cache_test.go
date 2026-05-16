package lyrics

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestBlurBucketClampsToRefMaximum(t *testing.T) {
	if got := blurCacheKey(0); got != 0 {
		t.Fatalf("bucket = %d, want 0", got)
	}
	if got := blurPixelsForLevel(1.2); got != 1.2 {
		t.Fatalf("blur pixels = %v, want 1.2", got)
	}
	if got := blurPixelsForLevel(99); got != 5 {
		t.Fatalf("blur pixels = %v, want 5", got)
	}
	if got := blurCacheKey(1.234); got != 1234 {
		t.Fatalf("cache key = %d, want 1234", got)
	}
}

func TestBlurRendererUsesLogicalPixelsForCacheKey(t *testing.T) {
	if got := blurCacheKey(2.5); got != 2500 {
		t.Fatalf("cache key = %d, want logical pixel key 2500", got)
	}
}

func TestLineClearBlurCache(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.BlurImage = ebiten.NewImage(4, 4)
	line.BlurCacheSource = ebiten.NewImage(4, 4)
	line.BlurCacheKey = 3000
	line.clearBlurCache()
	if line.BlurImage != nil || line.BlurCacheSource != nil || line.BlurCacheKey != 0 {
		t.Fatalf("blur cache not cleared: image %v source %v key %d", line.BlurImage, line.BlurCacheSource, line.BlurCacheKey)
	}
}

func TestMarkImageDirtyClearsBlurCache(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.BlurImage = ebiten.NewImage(4, 4)
	line.BlurCacheSource = ebiten.NewImage(4, 4)
	line.BlurCacheKey = 3000
	line.markImageDirty()
	if line.BlurImage != nil || line.BlurCacheSource != nil || line.BlurCacheKey != 0 {
		t.Fatal("markImageDirty should clear blur cache")
	}
}
