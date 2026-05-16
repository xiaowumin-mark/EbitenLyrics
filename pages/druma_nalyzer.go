package pages

import (
	"fmt"
	"image"
	"log"
	"math"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	"github.com/xiaowumin-mark/EbitenLyrics/bgrender"
	"github.com/xiaowumin-mark/EbitenLyrics/evbus"
	f "github.com/xiaowumin-mark/EbitenLyrics/font"
	"github.com/xiaowumin-mark/EbitenLyrics/lp"
	"github.com/xiaowumin-mark/EbitenLyrics/lyrics"
	"github.com/xiaowumin-mark/EbitenLyrics/router"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type DrumAnalyzer struct {
	router.BaseScene
	FontManager    *f.FontManager
	FontRequest    f.FontRequest
	AnimateManager *anim.Manager

	DrumEnergy       float64
	DrumEnergyString string

	Cover         *ebiten.Image
	CoverPosition lyrics.Position
	MeshRenderer  *bgrender.MeshGradientRenderer
	meshLastTick  time.Time
}

func (m *DrumAnalyzer) updateCoverTransform(w, he float64) {
	if m.CoverPosition.W <= 0 || m.CoverPosition.H <= 0 {
		return
	}
	if w <= 0 || he <= 0 {
		return
	}

	m.CoverPosition.TranslateX = (w - m.CoverPosition.W) / 2
	m.CoverPosition.TranslateY = (he - m.CoverPosition.H) / 2

	scaleX := w / m.CoverPosition.W
	scaleY := he / m.CoverPosition.H
	finalS := math.Max(scaleX, scaleY) * 1.4
	m.CoverPosition.ScaleX = finalS
	m.CoverPosition.ScaleY = finalS
}
func (m *DrumAnalyzer) OnCreate() {
	log.Println("DrumAnalyzer OnCreate")
	//ws:lowFreqVolume
	evbus.Bus.Subscribe("ws:lowFreqVolume", func(data any) {
		if v, ok := data.(float64); ok {
			m.DrumEnergy = v
		}

	})
	ww, hh := ebiten.WindowSize()
	meshRenderer, err := bgrender.NewMeshGradientRenderer(ww, hh)
	if err != nil {
		log.Printf("create mesh renderer failed: %v", err)
	} else {
		m.MeshRenderer = meshRenderer
	}
	m.CoverPosition = lyrics.NewPosition(0, 0, 0, 0)

	evbus.Bus.Subscribe("ws:cover", func(img image.Image) {
		if img == nil {
			return
		}

		m.Cover = ebiten.NewImageFromImage(img)
		m.CoverPosition.W = lp.FromLP(float64(m.Cover.Bounds().Dx()))
		m.CoverPosition.H = lp.FromLP(float64(m.Cover.Bounds().Dy()))
		m.CoverPosition.OriginX = m.CoverPosition.W / 2
		m.CoverPosition.OriginY = m.CoverPosition.H / 2
		w, he := ebiten.WindowSize()
		m.updateCoverTransform(lp.FromLP(float64(w)), lp.FromLP(float64(he)))
		if m.MeshRenderer != nil {
			if err := m.MeshRenderer.SetAlbum(m.Cover); err != nil {
				log.Printf("mesh renderer set album failed: %v", err)
			}
			log.Printf("mesh renderer set album success")
		}
	})
}
func (m *DrumAnalyzer) OnEnter(params map[string]any) {
	log.Println("DrumAnalyzer OnEnter", params)
	m.meshLastTick = time.Now()
}
func (m *DrumAnalyzer) OnLeave() {
	log.Println("DrumAnalyzer OnLeave")
}
func (m *DrumAnalyzer) OnDestroy() {
	log.Println("DrumAnalyzer OnDestroy")
}
func (m *DrumAnalyzer) Update() error {
	now := time.Now()
	if m.meshLastTick.IsZero() {
		m.meshLastTick = now
	}
	dt := now.Sub(m.meshLastTick)
	m.meshLastTick = now

	if m.Cover != nil && m.MeshRenderer != nil {
		m.MeshRenderer.SetLowFreqVolume(m.DrumEnergy)
	}
	m.DrumEnergyString = fmt.Sprintf("%.2f", m.DrumEnergy) + " |"
	for i := 0; i < 100; i++ {
		if m.DrumEnergy > float64(i)/100 {
			m.DrumEnergyString += "#"
		} else {
			m.DrumEnergyString += " "
		}
	}
	m.DrumEnergyString += "|"

	if m.MeshRenderer != nil {
		m.MeshRenderer.Update(dt)
	}
	return nil
}
func (m *DrumAnalyzer) Draw(screen *ebiten.Image) {

	if m.MeshRenderer != nil && m.MeshRenderer.HasRenderableState() {
		m.MeshRenderer.Draw(screen)
	}
	// debug draw
	ebitenutil.DebugPrintAt(
		screen,
		m.DrumEnergyString,
		0,
		0,
	)
}
func (m *DrumAnalyzer) OnResize(w, he int, isFirst bool) {
	log.Println("DrumAnalyzer OnResize", w, he, isFirst)
	if m.MeshRenderer != nil {
		m.MeshRenderer.Resize(w, he)
	}
}
