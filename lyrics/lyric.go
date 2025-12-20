package lyrics

import (
	"EbitenLyrics/anim"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
)

var CustomElastic = anim.NewEaseInElastic(1.1, 1.5)

func (l *Lyrics) Scroll(index []int, notInit int) {
	if len(l.Lines) == 0 {
		return
	}

	renderIndex := make([]int, 0)
	// 渲染now[0] 的上3句，下10句
	for i := index[0] - 4; i <= index[0]+10; i++ {
		// 是否在歌词的长度范围内
		if i < 0 || i >= len(l.Lines) {
			continue
		}
		renderIndex = append(renderIndex, i)
	}
	l.renderIndex = renderIndex
	log.Println("渲染的歌词", renderIndex)

	for index, el := range l.Lines {
		if has(renderIndex, index) {
			el.Render()
		} else {
			el.Dispose()

		}

		if (el.Status == Buffered || el.Status == Hot) && !has(l.nowLyrics, index) {
			//toNormal(el, game)
			el.ToNormal(l)

			for _, bg := range el.BackgroundLines {
				bg.ToNormal(l)

			}

		}
	}
	_, h := ebiten.WindowSize()
	// 计算歌词的y坐标
	var offsetY float64 = -float64(h) / 4
	for i := 0; i < index[0]; i++ {
		offsetY += l.Lines[i].Position.GetH()
		if has(index, i) && len(l.Lines[i].BackgroundLines) > 0 {
			for _, bgLine := range l.Lines[i].BackgroundLines {
				offsetY += bgLine.Position.GetH()
			}
		}
	}

	var lastY float64 = 0
	for i, line := range l.Lines {
		//line.GetPosition().SetY(lastY - offsetY)

		var bganimatedur time.Duration = 1000 * time.Millisecond
		var ddur time.Duration = 1
		if !has(l.renderIndex, i) {
			bganimatedur = 0
			ddur = 0
		}
		if line.ScrollAnimate != nil {
			line.ScrollAnimate.Cancel()
			line.ScrollAnimate = nil
		}
		line.ScrollAnimate = anim.NewTween(
			uuid.NewString(),
			bganimatedur*time.Duration(notInit),
			time.Duration((math.Abs(float64(index[0]-i-3)))*50)*time.Millisecond*ddur*time.Duration(notInit),
			1,
			line.GetPosition().GetY(),
			lastY-offsetY,
			CustomElastic,
			func(value float64) {
				line.GetPosition().SetY(value)
			},
			func() {
			},
		)
		l.AnimateManager.Add(line.ScrollAnimate)

		// bgs
		for _, bg := range line.BackgroundLines {
			if bg.ScrollAnimate != nil {
				bg.ScrollAnimate.Cancel()
				bg.ScrollAnimate = nil
			}
			bg.ScrollAnimate = anim.NewTween(
				uuid.NewString(),
				bganimatedur*time.Duration(notInit),
				time.Duration((math.Abs(float64(index[0]-i-3)))*50)*time.Millisecond*ddur*time.Duration(notInit),
				1,
				bg.GetPosition().GetY(),
				//game.Elements[i].Height+linem,
				lastY-offsetY+line.Position.GetH(),
				CustomElastic,
				func(value float64) {
					bg.GetPosition().SetY(value)
				},
				func() {
				},
			)
			l.AnimateManager.Add(bg.ScrollAnimate)
		}

		lastY += line.Position.GetH() + l.Margin
		if has(index, i) && len(line.BackgroundLines) > 0 {
			for _, bgLine := range line.BackgroundLines {
				lastY += bgLine.Position.GetH() + l.Margin
			}
		}
	}
}

func (l *Lyrics) Draw(screen *ebiten.Image) {
	for _, i := range l.renderIndex {
		l.Lines[i].Draw(screen)
		for _, bgLine := range l.Lines[i].BackgroundLines {
			bgLine.Draw(screen)
		}
	}
}

func bubbleSort(arr []int) []int {
	length := len(arr) //数据总长度（个数）
	for i := 0; i < length; i++ {
		for j := 0; j < length-1-i; j++ {
			if arr[j] > arr[j+1] { //和相邻的比
				arr[j], arr[j+1] = arr[j+1], arr[j] //对换位置
			}
		}
	}
	return arr
}

func has(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func remove(slice []int, val int) []int {
	newSlice := make([]int, 0)
	for _, item := range slice {
		if item != val {
			newSlice = append(newSlice, item)
		}
	}
	return newSlice
}

func (l *Lyrics) Update(t time.Duration) {
	l.Position = t
	for i, line := range l.Lines {
		if t >= line.StartTime && t < line.EndTime {
			// 如果在nowindex中，则跳过
			if has(l.nowLyrics, i) {
				continue
			}
			// 如果不在nowindex中，则加入
			l.nowLyrics = append(l.nowLyrics, i)
			log.Println("歌词进入：", i, l.nowLyrics, line.Text)

			line.Status = Hot
			l.Scroll(bubbleSort(l.nowLyrics), 1)

			line.LineAnimate(l)

		} else {
			// 如果在nowindex中，则移出
			if has(l.nowLyrics, i) {
				l.nowLyrics = remove(l.nowLyrics, i)
				log.Println("歌词退出：", i)
				line.Status = Buffered
			}
		}
	}
}

func (l *Line) ToNormal(lyrics *Lyrics) {
	for _, e := range l.OuterSyllableElements {

		if e.UpAnimate != nil {
			e.UpAnimate.Cancel()
			e.UpAnimate = nil
		}
		// 上浮动画回位
		e.UpAnimate = anim.NewTween(
			uuid.NewString(),
			time.Duration(600)*time.Millisecond,
			0,
			1,
			e.GetPosition().GetTranslateY(),
			0,
			anim.EaseInOut,
			func(value float64) {
				//e.TransformY = value
				e.GetPosition().SetTranslateY(value)
			},
			func() {
				//e.TransformY = 0
				e.GetPosition().SetTranslateY(0)
			},
		)
		lyrics.AnimateManager.Add(e.UpAnimate)
	}
	/*if l.GradientColorAnimate != nil {
		l.GradientColorAnimate.Cancel()
		l.GradientColorAnimate = nil
	}
	var ap float64 = 255
	if l.IsBackground {
		ap = 125
	}
	l.GradientColorAnimate = anim.NewTween(
		uuid.NewString(),
		time.Duration(600)*time.Millisecond,
		0,
		1,
		ap,
		100,
		anim.EaseInOut,
		func(value float64) {
			for _, e := range l.OuterSyllableElements {
				e.SyllableImage.SetStartColor(
					color.RGBA{255, 255, 255, uint8(value)},
				)
			}
		},
		func() {
			for _, e := range l.OuterSyllableElements {
				e.NowOffset = e.SyllableImage.GetOffset()
				e.SyllableImage.SetStartColor(
					color.RGBA{255, 255, 255, uint8(ap)},
				)
			}

		},
	)*/
	for _, e := range l.OuterSyllableElements {
		e.NowOffset = e.SyllableImage.GetOffset()
	}
	//lyrics.AnimateManager.Add(l.GradientColorAnimate)

	if l.IsBackground {
		if l.AlphaAnimate != nil {
			l.AlphaAnimate.Cancel()
			l.AlphaAnimate = nil
		}
		l.AlphaAnimate = anim.NewTween(
			uuid.NewString(),
			time.Duration(300)*time.Millisecond,
			0,
			1,
			l.GetPosition().GetAlpha(),
			0,
			anim.EaseInOut,
			func(value float64) {
				l.GetPosition().SetAlpha(value)
			},
			func() {
				l.GetPosition().SetAlpha(0)
			},
		)
		lyrics.AnimateManager.Add(l.AlphaAnimate)
	}
}
