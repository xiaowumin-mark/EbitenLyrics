package interm

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Interactive interface {
	Bounds() (x, y, w, h float64)
	OnHoverEnter()
	OnHoverExit()
	OnClick()
	Update()
}

type InteractionManager struct {
	elements []Interactive
	hovered  Interactive
}

func NewInteractionManager() *InteractionManager {
	return &InteractionManager{}
}

func (im *InteractionManager) Add(e Interactive) {
	im.elements = append(im.elements, e)
}

func (im *InteractionManager) Update() {
	mx, my := ebiten.CursorPosition()

	var nowHovered Interactive

	for _, e := range im.elements {
		e.Update()

		x, y, w, h := e.Bounds()
		if float64(mx) >= x && float64(mx) <= x+w &&
			float64(my) >= y && float64(my) <= y+h {
			nowHovered = e
		}
	}

	if nowHovered != im.hovered {
		if im.hovered != nil {
			im.hovered.OnHoverExit()
		}
		if nowHovered != nil {
			nowHovered.OnHoverEnter()
		}
	}

	// 点击检测
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(ebiten.MouseButtonLeft)) {
		if nowHovered != nil {
			nowHovered.OnClick()
		}
	}

	im.hovered = nowHovered
}
