package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/comps/liquidglass"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

type demoGame struct {
	ui        *ebitenui.UI
	glass     *liquidglass.Component
	lastTick  time.Time
	status    *widget.Label
	blur      *widget.Slider
	thickness *widget.Slider
	glassBtns []*liquidglass.UIButton
}

func newFace(size float64) (text.Face, error) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}
	return &text.GoTextFace{
		Source: source,
		Size:   size,
	}, nil
}

func newButton(label string, face text.Face, onPressed func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    eimage.NewNineSliceColor(color.NRGBA{0x2a, 0x36, 0x4d, 0xdd}),
			Hover:   eimage.NewNineSliceColor(color.NRGBA{0x38, 0x4a, 0x68, 0xee}),
			Pressed: eimage.NewNineSliceColor(color.NRGBA{0x1d, 0x28, 0x39, 0xff}),
		}),
		widget.ButtonOpts.Text(label, &face, &widget.ButtonTextColor{
			Idle: color.White,
		}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(8)),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(88, 34),
		),
		widget.ButtonOpts.PressedHandler(func(*widget.ButtonPressedEventArgs) {
			if onPressed != nil {
				onPressed()
			}
		}),
	)
}

func newSlider(min, max, current int, onChanged func(value int)) *widget.Slider {
	return widget.NewSlider(
		widget.SliderOpts.MinMax(min, max),
		widget.SliderOpts.InitialCurrent(current),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  eimage.NewNineSliceColor(color.NRGBA{0xff, 0xff, 0xff, 0x45}),
				Hover: eimage.NewNineSliceColor(color.NRGBA{0xff, 0xff, 0xff, 0x70}),
			},
			&widget.ButtonImage{
				Idle:    eimage.NewNineSliceColor(color.NRGBA{0xcf, 0xe4, 0xff, 0xff}),
				Hover:   eimage.NewNineSliceColor(color.NRGBA{0xff, 0xff, 0xff, 0xff}),
				Pressed: eimage.NewNineSliceColor(color.NRGBA{0x9f, 0xc6, 0xf2, 0xff}),
			},
		),
		widget.SliderOpts.FixedHandleSize(12),
		widget.SliderOpts.TrackOffset(0),
		widget.SliderOpts.PageSizeFunc(func() int {
			return 1
		}),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.Current)
			}
		}),
		widget.SliderOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(220, 16),
		),
	)
}

func newGame() (*demoGame, error) {
	glass, err := liquidglass.New()
	if err != nil {
		return nil, err
	}
	glass.SetBackgroundMode(liquidglass.BackgroundGrid)
	glass.Resize(1200, 720)
	glass.SetMouse(600, 360)
	glass.ResetMouseSpringToMouse()

	face, err := newFace(16)
	if err != nil {
		glass.Dispose()
		return nil, err
	}
	titleFace, err := newFace(20)
	if err != nil {
		glass.Dispose()
		return nil, err
	}

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(color.NRGBA{0x0f, 0x14, 0x20, 0xd8})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(14)),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding:            widget.NewInsetsSimple(16),
		})),
	)
	root.AddChild(panel)

	game := &demoGame{
		glass:    glass,
		lastTick: time.Now(),
	}

	panel.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("Liquid Glass + ebitenui", &titleFace, &widget.LabelColor{
			Idle: color.NRGBA{0xe9, 0xf2, 0xff, 0xff},
		}),
	))
	panel.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("鼠标移动会驱动玻璃体，Esc 退出", &face, &widget.LabelColor{
			Idle: color.NRGBA{0xc4, 0xd2, 0xe8, 0xff},
		}),
	))

	panel.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("BlurRadius", &face, &widget.LabelColor{Idle: color.White}),
	))
	game.blur = newSlider(1, 20, int(math.Round(game.glass.Params().BlurRadius)), func(value int) {
		game.glass.Params().BlurRadius = float64(value)
	})
	panel.AddChild(game.blur)

	panel.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("RefThickness", &face, &widget.LabelColor{Idle: color.White}),
	))
	game.thickness = newSlider(1, 80, int(math.Round(game.glass.Params().RefThickness)), func(value int) {
		game.glass.Params().RefThickness = float64(value)
	})
	panel.AddChild(game.thickness)

	buttonRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel.AddChild(buttonRow)

	syncSliders := func() {
		p := game.glass.Params()
		game.blur.Current = int(math.Round(p.BlurRadius))
		game.thickness.Current = int(math.Round(p.RefThickness))
	}

	defaultBtn, err := liquidglass.NewUIButton("Default", face, func() {
		game.glass.SetParams(liquidglass.DefaultParams())
		syncSliders()
	})
	if err != nil {
		glass.Dispose()
		return nil, err
	}
	chromeBtn, err := liquidglass.NewUIButton("Chrome", face, func() {
		game.glass.SetParams(liquidglass.ChromeParams())
		syncSliders()
	})
	if err != nil {
		glass.Dispose()
		defaultBtn.Dispose()
		return nil, err
	}
	softBtn, err := liquidglass.NewUIButton("Soft", face, func() {
		game.glass.SetParams(liquidglass.SoftParams())
		syncSliders()
	})
	if err != nil {
		glass.Dispose()
		defaultBtn.Dispose()
		chromeBtn.Dispose()
		return nil, err
	}
	game.glassBtns = []*liquidglass.UIButton{defaultBtn, chromeBtn, softBtn}
	buttonRow.AddChild(defaultBtn)
	buttonRow.AddChild(chromeBtn)
	buttonRow.AddChild(softBtn)

	panel.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("普通按钮（对比）", &face, &widget.LabelColor{Idle: color.NRGBA{0xc4, 0xd2, 0xe8, 0xff}}),
	))
	panel.AddChild(newButton("Normal Button", face, func() {}))

	game.status = widget.NewLabel(
		widget.LabelOpts.Text("", &face, &widget.LabelColor{
			Idle: color.NRGBA{0xdb, 0xe5, 0xf5, 0xff},
		}),
	)
	panel.AddChild(game.status)

	game.ui = &ebitenui.UI{Container: root}
	return game, nil
}

func (g *demoGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	now := time.Now()
	dt := now.Sub(g.lastTick)
	g.lastTick = now

	mx, my := ebiten.CursorPosition()
	g.glass.SetMouse(float64(mx), float64(my))
	g.glass.Update(dt)

	g.status.Label = fmt.Sprintf(
		"Blur=%d  Thickness=%d  FPS=%.0f",
		int(math.Round(g.glass.Params().BlurRadius)),
		int(math.Round(g.glass.Params().RefThickness)),
		ebiten.ActualFPS(),
	)

	g.ui.Update()
	return nil
}

func (g *demoGame) Draw(screen *ebiten.Image) {
	g.glass.Draw(screen)
	g.ui.Draw(screen)
}

func (g *demoGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return 1, 1
	}
	g.glass.Resize(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowSize(1200, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Liquid Glass + ebitenui Example")

	game, err := newGame()
	if err != nil {
		log.Fatal(err)
	}
	defer game.glass.Dispose()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
