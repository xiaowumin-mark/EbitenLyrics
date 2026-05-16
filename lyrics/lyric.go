package lyrics

// 文件说明：歌词整体对象的补充实现。
// 主要职责：承接整体级别的状态更新与辅助逻辑。

// This file intentionally keeps the package-level documentation anchor.
// Core implementations are split into:
// - layer_layout.go
// - layer_renderer.go
// - layer_animation.go

import "math"

const (
	staticLayerSignatureSeed  uint64 = 1469598103934665603
	staticLayerSignaturePrime uint64 = 1099511628211
)

func mixStaticLayerSignature(signature *uint64, value uint64) {
	*signature ^= value
	*signature *= staticLayerSignaturePrime
}

func staticLayerFloatBits(value float64) uint64 {
	return math.Float64bits(math.Round(value * 1000))
}

func (l *Lyrics) StaticLayerSignature() (uint64, bool, bool) {
	if l == nil {
		return 0, false, false
	}

	signature := staticLayerSignatureSeed
	hasStatic := false
	needsRebuild := false

	accumulate := func(index int, line *Line) {
		if !line.canUseStaticLayer() {
			return
		}
		hasStatic = true
		if line.imageDirty || line.Image == nil {
			needsRebuild = true
		}
		mixStaticLayerSignature(&signature, uint64(index+1))
		mixStaticLayerSignature(&signature, staticLayerFloatBits(line.GetPosition().GetX()))
		mixStaticLayerSignature(&signature, staticLayerFloatBits(line.GetPosition().GetY()))
		mixStaticLayerSignature(&signature, staticLayerFloatBits(line.GetPosition().GetScaleX()))
		mixStaticLayerSignature(&signature, staticLayerFloatBits(line.GetPosition().GetScaleY()))
		mixStaticLayerSignature(&signature, staticLayerFloatBits(line.GetPosition().GetAlpha()))
	}

	for _, i := range l.renderIndex {
		if i < 0 || i >= len(l.Lines) {
			continue
		}
		line := l.Lines[i]
		accumulate(i*2, line)
		for bgIndex, bgLine := range line.BackgroundLines {
			accumulate(i*2+bgIndex+1, bgLine)
		}
	}

	if !hasStatic {
		return 0, false, false
	}
	return signature, true, needsRebuild
}

type RenderCacheStats struct {
	VisibleLines          int
	LineImages            int
	DirtyLineImages       int
	LineBlurImages        int
	BottomImage           bool
	BottomImageDirty      bool
	BottomBlurImage       bool
	SharedImageCacheStats ImageCacheStats
}

func (l *Lyrics) RenderCacheStats() RenderCacheStats {
	stats := RenderCacheStats{SharedImageCacheStats: SharedImageCacheStats()}
	if l == nil {
		return stats
	}

	accumulate := func(line *Line) {
		if line == nil || !line.isShow {
			return
		}
		stats.VisibleLines++
		if line.Image != nil {
			stats.LineImages++
		}
		if line.imageDirty {
			stats.DirtyLineImages++
		}
		if line.BlurImage != nil {
			stats.LineBlurImages++
		}
	}

	for _, line := range l.Lines {
		accumulate(line)
		if line == nil {
			continue
		}
		for _, bgLine := range line.BackgroundLines {
			accumulate(bgLine)
		}
	}

	stats.BottomImage = l.Bottom.Image != nil
	stats.BottomImageDirty = l.Bottom.ImageDirty
	stats.BottomBlurImage = l.Bottom.BlurImage != nil
	return stats
}
