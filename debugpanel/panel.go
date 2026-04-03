package debugpanel

import (
	"image"
	"strings"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
)

type Panel struct {
	title   string
	bounds  image.Rectangle
	visible bool
	ui      debugui.DebugUI
	groups  []*Group
}

type Group struct {
	label       string
	expanded    bool
	controls    []control
	description string
}

type control interface {
	draw(ctx *debugui.Context)
}

type boolControl struct {
	label    string
	value    *bool
	onChange func(bool)
}

type floatControl struct {
	label    string
	value    *float64
	low      float64
	high     float64
	step     float64
	digits   int
	onChange func(float64)
}

type selectControl struct {
	label   string
	options func() []string
	get     func() int
	set     func(int)
}

type actionControl struct {
	label  string
	action func()
}

type textControl struct {
	label string
	get   func() string
}

func New(title string, bounds image.Rectangle) *Panel {
	return &Panel{
		title:   title,
		bounds:  bounds,
		visible: true,
	}
}

func (p *Panel) SetVisible(visible bool) {
	p.visible = visible
}

func (p *Panel) Visible() bool {
	return p.visible
}

func (p *Panel) Toggle() {
	p.visible = !p.visible
}

func (p *Panel) Group(label string, expanded bool) *Group {
	group := &Group{
		label:    label,
		expanded: expanded,
	}
	p.groups = append(p.groups, group)
	return group
}

func (p *Panel) Update() (bool, error) {
	if !p.visible {
		return false, nil
	}
	state, err := p.ui.Update(func(ctx *debugui.Context) error {
		ctx.Window(p.title, p.bounds, func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1}, []int{-1})
			ctx.Panel(func(layout debugui.ContainerLayout) {
				ctx.SetGridLayout([]int{-1}, nil)
				ctx.Loop(len(p.groups), func(i int) {
					p.groups[i].draw(ctx)
				})
			})
		})
		return nil
	})
	return state != 0, err
}

func (p *Panel) Draw(screen *ebiten.Image) {
	if !p.visible {
		return
	}
	p.ui.Draw(screen)
}

func (g *Group) Description(text string) *Group {
	g.description = strings.TrimSpace(text)
	return g
}

func (g *Group) Bool(label string, value *bool, onChange func(bool)) *Group {
	g.controls = append(g.controls, &boolControl{
		label:    label,
		value:    value,
		onChange: onChange,
	})
	return g
}

func (g *Group) Float(label string, value *float64, low, high, step float64, digits int, onChange func(float64)) *Group {
	g.controls = append(g.controls, &floatControl{
		label:    label,
		value:    value,
		low:      low,
		high:     high,
		step:     step,
		digits:   digits,
		onChange: onChange,
	})
	return g
}

func (g *Group) Select(label string, options func() []string, get func() int, set func(int)) *Group {
	g.controls = append(g.controls, &selectControl{
		label:   label,
		options: options,
		get:     get,
		set:     set,
	})
	return g
}

func (g *Group) Action(label string, action func()) *Group {
	g.controls = append(g.controls, &actionControl{
		label:  label,
		action: action,
	})
	return g
}

func (g *Group) Text(label string, get func() string) *Group {
	g.controls = append(g.controls, &textControl{
		label: label,
		get:   get,
	})
	return g
}

func (g *Group) draw(ctx *debugui.Context) {
	ctx.Header(g.label, g.expanded, func() {
		ctx.SetGridLayout([]int{-1}, nil)
		if g.description != "" {
			drawTextBlock(ctx, g.description)
		}
		ctx.Loop(len(g.controls), func(i int) {
			g.controls[i].draw(ctx)
		})
	})
}

func (c *boolControl) draw(ctx *debugui.Context) {
	if c.value == nil {
		return
	}
	handler := ctx.Checkbox(c.value, c.label)
	if c.onChange != nil {
		handler.On(func() {
			c.onChange(*c.value)
		})
	}
}

func (c *floatControl) draw(ctx *debugui.Context) {
	if c.value == nil {
		return
	}
	if c.label != "" {
		ctx.Text(c.label)
	}
	handler := ctx.SliderF(c.value, c.low, c.high, c.step, c.digits)
	if c.onChange != nil {
		handler.On(func() {
			c.onChange(*c.value)
		})
	}
}

func (c *selectControl) draw(ctx *debugui.Context) {
	if c.options == nil || c.get == nil || c.set == nil {
		return
	}
	options := c.options()
	if c.label != "" {
		ctx.Text(c.label)
	}
	if len(options) == 0 {
		ctx.Text("(no options)")
		return
	}
	selected := c.get()
	if selected < 0 {
		selected = 0
	}
	if selected >= len(options) {
		selected = len(options) - 1
	}
	ctx.Dropdown(&selected, options).On(func() {
		c.set(selected)
	})
}

func (c *actionControl) draw(ctx *debugui.Context) {
	if c.action == nil {
		return
	}
	ctx.Button(c.label).On(c.action)
}

func (c *textControl) draw(ctx *debugui.Context) {
	if c.get == nil {
		return
	}
	if c.label != "" {
		ctx.Text(c.label)
	}
	drawTextBlock(ctx, c.get())
}

func drawTextBlock(ctx *debugui.Context, text string) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		ctx.Text(line)
	}
}
