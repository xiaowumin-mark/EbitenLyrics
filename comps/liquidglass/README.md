# LiquidGlass Component

`comps/liquidglass` is a reusable Ebiten liquid-glass renderer component.

## Quick Usage

```go
glass, err := liquidglass.New()
if err != nil {
    return err
}
defer glass.Dispose()

glass.Resize(screenW, screenH)
glass.SetBackgroundMode(liquidglass.BackgroundPhoto)
glass.SetBackgroundImageFromPath("test-data/liquid-bg.jpg")

params := glass.Params()
params.RefThickness = 40
params.RefFactor = 1.8
params.RefDispersion = 12

// in Update():
mx, my := ebiten.CursorPosition()
glass.SetMouse(float64(mx), float64(my))
glass.Update(dt)

// in Draw():
glass.Draw(screen)
```

## API

- `New() (*Component, error)`
- `(*Component).Dispose()`
- `(*Component).Resize(w, h int)`
- `(*Component).SetMouse(x, y float64)`
- `(*Component).Update(dt time.Duration)`
- `(*Component).Draw(screen *ebiten.Image)`
- `(*Component).Params() *Params`
- `(*Component).SetParams(p Params)`
- `(*Component).SetBackgroundMode(mode BackgroundMode)`
- `(*Component).SetBackgroundImage(img image.Image)`
- `(*Component).SetBackgroundImageFromPath(path string) error`
- `(*Component).LoadBackgroundFromCandidates(paths []string) (string, error)`

## Presets

- `liquidglass.DefaultParams()`
- `liquidglass.ChromeParams()`
- `liquidglass.SoftParams()`

## With ebitenui

Render order is:

1. `glass.Draw(screen)` (background)
2. `ui.Draw(screen)` (UI panel on top)

Runnable sample:

- `examples/liquid_glass_ebitenui/main.go`
- Run: `go run ./examples/liquid_glass_ebitenui`

### Liquid Glass Button Widget

You can also use a liquid-glass style button as an `ebitenui` widget:

```go
btn, err := liquidglass.NewUIButton("Play", face, func() {
    // click handler
})
if err != nil {
    return err
}
btn.BlurRadius = 8
btn.PressScale = 1.035

container.AddChild(btn)
```
