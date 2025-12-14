package pages

import (
	"EbitenLyrics/anim"
	"EbitenLyrics/lyrics"
	"EbitenLyrics/router"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Game struct {
	router.BaseScene // 嵌入并获得默认实现
	AnimateManager   *anim.Manager
	Font             *text.GoTextFaceSource
	sy               *lyrics.LineSyllable
	fontsize         float64

	line *lyrics.Line

	texts   []lyrics.Position
	textimg *ebiten.Image

	ss []*lyrics.LineSyllable
}

func (g *Game) OnEnter(params map[string]any) {
	log.Println("Game OnEnter", params)
	g.fontsize = 48.0
	/*g.sy, err = lyrics.NewSyllable(
		"Hello, Ebiten Lyrics!",
		0,
		5000*time.Millisecond,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		1.0,
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 255, A: 100},
	)
	if err != nil {
		log.Println(err)
	}
	g.sy.Position.SetX(100)
	g.sy.Position.SetY(100)*/
	w, _ := ebiten.WindowSize()
	//var texts = "Hello, ebiten Lyrics!"
	var testa = []string{"Hel", "lo, ", "ebi", "ten ", "Ly", "rics!"}
	g.line = lyrics.NewLine(0, 5000*time.Millisecond, false, false, "你好,ebiten歌词")
	g.line.Position.SetW(float64(w))
	g.line.SetPadding(20)
	g.line.SetFont(g.Font)
	g.line.SetFontSize(g.fontsize)
	for _, syllable := range testa {
		syllable, err := lyrics.NewSyllable(
			string(syllable),
			0,
			5000*time.Millisecond,
			&text.GoTextFace{
				Source: g.Font,
				Size:   g.fontsize,
			},
			1.0,
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			color.RGBA{R: 0, G: 0, B: 255, A: 100},
			false,
		)
		if err != nil {
			log.Println(err)
		}
		g.line.AddSyllable(syllable)
	}
	g.line.Layout()

	/*a, err := lyrics.NewSyllable(
		"Hel",
		0,
		0,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		1.0,
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 100},
		false,
	)
	if err != nil {
		log.Println(err)
	}
	b, err := lyrics.NewSyllable(
		"llo ",
		0,
		0,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		1.0,
		color.RGBA{R: 0, G: 255, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 100},
		false,
	)
	if err != nil {
		log.Println(err)
	}
	c, err := lyrics.NewSyllable(
		"wo",
		0,
		0,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		1.0,
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		color.RGBA{R: 0, G: 0, B: 100},
		false,
	)
	if err != nil {
		log.Println(err)
	}
	d, err := lyrics.NewSyllable(
		"rld!",
		0,
		0,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		1.0,
		color.RGBA{R: 0, G: 255, B: 255, A: 255},
		color.RGBA{R: 0, G: 0, B: 100},
		false,
	)
	if err != nil {
		log.Println(err)
	}
	g.ss = append(g.ss, a)
	g.ss = append(g.ss, b)
	g.ss = append(g.ss, c)
	g.ss = append(g.ss, d)
	var ls [][]*lyrics.LineSyllable
	var l []*lyrics.LineSyllable
	l = append(l, a)
	l = append(l, b)
	ls = append(ls, l)
	l = nil
	l = append(l, c)
	l = append(l, d)
	ls = append(ls, l)
	log.Println("分词结果", ls)
	w, _ := ebiten.WindowSize()
	var h float64
	g.texts, h = lyrics.AutoLayoutSyllable(
		ls,
		&text.GoTextFace{
			Source: g.Font,
			Size:   g.fontsize,
		},
		float64(w),
		0,
		1.0,
		text.AlignStart,
	)
	g.textimg = ebiten.NewImage(w, int(math.Ceil(h))) // 使用向上取整
	g.textimg.Fill(color.Opaque)
	for i, t := range g.texts {
		op := &text.DrawOptions{}
		log.Println(t.GetX())
		op.GeoM.Translate(t.X, t.Y)
		op.ColorScale.ScaleWithColor(color.Black)
		text.Draw(
			g.textimg,
			g.ss[i].Syllable,
			&text.GoTextFace{
				Source: g.Font,
				Size:   g.fontsize,
			},
			op,
		)
	}*/

}

func (g *Game) OnLeave() {

}
func (g *Game) Update() error {

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		router.Go("home", nil)
	}
	/*
		// 鼠标滚轮
		_, wy := ebiten.Wheel()
		if wy != 0 {
			g.sy.SetNowOffset(g.sy.GetNowOffset() + wy*5)
			//g.sy.Position.ScaleX += wy * 0.1
			g.sy.Position.SetScaleX(g.sy.Position.GetScaleX() + wy*0.1)
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.fontsize += 4.0
			g.sy.SyllableImage.SetFont(&text.GoTextFace{
				Source: g.Font,
				Size:   g.fontsize,
			})
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.fontsize -= 4.0
			g.sy.SyllableImage.SetFont(&text.GoTextFace{
				Source: g.Font,
				Size:   g.fontsize,
			})
		}
	*/

	_, wy := ebiten.Wheel()
	for _, e := range g.line.GetSyllables() {
		for _, s := range e.Elements {
			s.SetNowOffset(s.GetNowOffset() + wy*5)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(ebiten.MouseButtonLeft)) {
		g.line.Dispose()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(ebiten.MouseButtonRight)) {
		g.line.Render()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	/*op := ebiten.DrawImageOptions{}
	op.GeoM = lyrics.TransformToGeoM(g.sy.GetPosition())
	op.Filter = ebiten.FilterLinear
	g.sy.SyllableImage.Draw(screen, g.sy.GetNowOffset(), g.sy.Position.GetAlpha())*/
	/*op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 0)
	screen.DrawImage(g.textimg, op)*/

	g.line.Draw(screen)
	/*if g.line.TranslateImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(1500, 0)
		screen.DrawImage(g.line.TranslateImage, op)
	}*/
}

func (g *Game) OnResize(w, he int, isFirst bool) {
	log.Println("Game OnResize", w, he, isFirst)
	if !isFirst {
		//g.line.Resize(float64(w))
		g.line.Position.SetW(float64(w))
		g.line.Layout()
	}
}
