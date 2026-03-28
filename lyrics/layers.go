package lyrics

// 文件说明：歌词分层职责标记。
// 主要职责：把布局、渲染、动画能力按层拆分到独立实现中。

type LayoutLayer struct{}
type RendererLayer struct{}
type AnimationLayer struct{}

var (
	lineLayoutLayer    = LayoutLayer{}
	lineRendererLayer  = RendererLayer{}
	lineAnimationLayer = AnimationLayer{}
)
