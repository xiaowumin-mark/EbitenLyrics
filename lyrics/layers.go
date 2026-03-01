package lyrics

type LayoutLayer struct{}
type RendererLayer struct{}
type AnimationLayer struct{}

var (
	lineLayoutLayer    = LayoutLayer{}
	lineRendererLayer  = RendererLayer{}
	lineAnimationLayer = AnimationLayer{}
)
