package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/router"
	"EbitenLyrics/uis"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/colornames"
)

type Home struct {
	router.BaseScene // 嵌入并获得默认实现
	ui               *ebitenui.UI
	Font             *text.GoTextFaceSource
	LyricsImage      *ebiten.Image
	LyricsImageY     float64
	LyricsIsOpen     bool
	LyricsImageAnim  *anim.Tween
	AnimateManager   *anim.Manager
}

func (h *Home) OnCreate() {
	log.Println("Home OnCreate")
}
func (h *Home) OnEnter(params map[string]any) {
	log.Println("Home OnEnter", params)
	h.LyricsIsOpen = false
	if h.Font == nil {
		log.Fatalln("Home page Font is nil")
	}

	h.ui = uis.CreateUI(h.Font)
	w, hi := ebiten.WindowSize()
	h.LyricsImage = ebiten.NewImage(w, hi)
	h.LyricsImage.Fill(colornames.Wheat)
	h.LyricsImageY = float64(hi)

}

func (h *Home) OnLeave() {
	log.Println("Home OnLeave")
}

func (h *Home) OnDestroy() {
	log.Println("Home OnDestroy")
}

func (h *Home) Update() error {
	h.ui.Update()
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		router.Go("game", nil)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if h.LyricsIsOpen {
			if h.LyricsImageAnim != nil {
				h.LyricsImageAnim.Cancel()
			}
			// 打开歌词面板
			h.LyricsImageAnim = anim.NewTween(
				"lyrics-open",
				time.Millisecond*400,
				0,
				1,
				h.LyricsImageY,
				0,
				anim.EaseOut,
				func(v float64) {
					h.LyricsImageY = v
				},
				nil,
			)
			h.AnimateManager.Add(h.LyricsImageAnim)
		} else {
			if h.LyricsImageAnim != nil {
				h.LyricsImageAnim.Cancel()
			}
			// 关闭歌词面板
			_, hi := ebiten.WindowSize()
			h.LyricsImageAnim = anim.NewTween(
				"lyrics-close",
				time.Millisecond*400,
				0,
				1,
				h.LyricsImageY,
				float64(hi),
				anim.EaseOut,
				func(v float64) {
					h.LyricsImageY = v
				},
				nil,
			)
			h.AnimateManager.Add(h.LyricsImageAnim)

		}
		h.LyricsIsOpen = !h.LyricsIsOpen
	}
	return nil
}

func (h *Home) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Home Scene")
	if h.ui != nil {
		h.ui.Draw(screen)
	}
	if h.LyricsImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, h.LyricsImageY)
		screen.DrawImage(h.LyricsImage, op)
	}
}

func (h *Home) OnResize(w, he int, isFirst bool) {
	log.Println("Home OnResize", w, he, isFirst)
	// 调整歌词面板大小
	h.LyricsImage = ebiten.NewImage(w, he)
	h.LyricsImage.Fill(colornames.Wheat)
	h.LyricsImageY = float64(he)

}

// createButtonImage 创建一个纯色的 NineSliceColor 图像作为按钮的背景
// 注意：实际项目中你可能需要加载 PNG 图像来制作更美观的按钮。
func createButtonImage() *widget.ButtonImage {
	// 默认状态的颜色
	idle := image.NewNineSliceColor(colornames.Darkgray)
	// 悬停状态的颜色
	hover := image.NewNineSliceColor(colornames.Gray)
	// 按下状态的颜色
	pressed := image.NewNineSliceColor(colornames.Dimgray)

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}
}

func Mix(a, b color.Color, percent float64) color.Color {
	rgba := func(c color.Color) (r, g, b, a uint8) {
		r16, g16, b16, a16 := c.RGBA()
		return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8), uint8(a16 >> 8)
	}
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + percent*(float64(y)-float64(x))))
	}
	r1, g1, b1, a1 := rgba(a)
	r2, g2, b2, a2 := rgba(b)

	return color.RGBA{
		R: lerp(r1, r2),
		G: lerp(g1, g2),
		B: lerp(b1, b2),
		A: lerp(a1, a2),
	}
}

// 2. **【新增】** 假定你加载了按钮的图像资源（必须项）
/*buttonImage := createButtonImage() // 假设这个函数能返回 *widget.ButtonImage

// 3. 根容器使用 AnchorLayout 实现居中
root := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.White),
	),
	widget.ContainerOpts.Layout(widget.NewAnchorLayout(
		widget.AnchorLayoutOpts.Padding(&widget.Insets{}),
	)),
)

f := ft.GetFace(h.Font, 20)
left := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.Indianred),
	),
	widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			StretchVertical:    true,
		}),
		widget.WidgetOpts.MinSize(50, 50),
	),
)
right := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.Mediumseagreen),
	),
	widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			StretchVertical:    true,
		}),
		widget.WidgetOpts.MinSize(50, 50),
	),
)
up := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.Goldenrod),
	),
	widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			VerticalPosition:  widget.AnchorLayoutPositionStart,
			StretchHorizontal: true,
		}),
		widget.WidgetOpts.MinSize(50, 50),
	),
)
down := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.Steelblue),
	),
	widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			VerticalPosition:  widget.AnchorLayoutPositionEnd,
			StretchHorizontal: true,
		}),
		widget.WidgetOpts.MinSize(50, 50),
	),
)
center := widget.NewContainer(
	widget.ContainerOpts.BackgroundImage(
		image.NewNineSliceColor(colornames.Darkslategray),
	),
	widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			StretchHorizontal:  true,
			StretchVertical:    true,
		}),
		widget.WidgetOpts.MinSize(50, 50),
	),
)

mainv := widget.NewContainer(
	widget.ContainerOpts.Layout(widget.NewRowLayout(
		widget.RowLayoutOpts.Direction(
			widget.DirectionVertical,
		),
		widget.RowLayoutOpts.Spacing(0),
	)),
)
button := widget.NewButton(
	widget.ButtonOpts.TextLabel("Button"),
	widget.ButtonOpts.TextFace(&f),
	widget.ButtonOpts.Image(buttonImage), // 使用加载的图像
	widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
		println("Button clicked!")
	}),
	widget.ButtonOpts.TextColor(&widget.ButtonTextColor{
		Idle:    colornames.Gainsboro,
		Hover:   colornames.Gainsboro,
		Pressed: Mix(colornames.Gainsboro, colornames.Black, 0.4),
	}),
	widget.ButtonOpts.WidgetOpts(
		widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		}),
		widget.WidgetOpts.MinSize(96, 64),
	),
)

root.AddChild(center)
mainv.AddChild(button)
center.AddChild(mainv)

root.AddChild(left)
root.AddChild(right)
root.AddChild(up)
root.AddChild(down)*/

// 4. 初始化 UI 并设置主题
