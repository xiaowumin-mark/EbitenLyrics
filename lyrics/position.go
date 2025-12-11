package lyrics

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Position struct {
	X, Y                   float64 // 初始左上角位置
	W, H                   float64 // 原始宽高
	TranslateX, TranslateY float64
	ScaleX, ScaleY         float64
	Rotate                 float64 // 角度（度）
	OriginX, OriginY       float64 // 变换中心（相对于左上角）
	Alpha                  float64
}

// ------------------------------------------------------
// 1. 生成标准 GeoM（最重要）
// transform-origin: (OriginX, OriginY)
// transform: translate + rotate + scale (与 CSS 一致)
// ------------------------------------------------------

func TransformToGeoM(p *Position) (m ebiten.GeoM) {
	m = ebiten.GeoM{}

	// 1) 把图像移到原点，使 Origin 成为 (0,0)
	m.Translate(-p.OriginX, -p.OriginY)

	// 2) Scale（绕 origin）
	m.Scale(p.ScaleX, p.ScaleY)

	// 3) Rotate（绕 origin）
	r := p.Rotate * math.Pi / 180
	m.Rotate(r)

	// 4) 移回 origin
	m.Translate(p.OriginX, p.OriginY)

	// 5) 最终平移位置：X,Y + TranslateX,Y
	m.Translate(p.X+p.TranslateX, p.Y+p.TranslateY)

	return
}

// ------------------------------------------------------
// 2. 获取变换后左上角坐标（即世界坐标）
// ------------------------------------------------------

func GetTransformedXY(p *Position) (float64, float64) {
	m := TransformToGeoM(p)
	return m.Apply(0, 0)
}

// ------------------------------------------------------
// 3. 获取四个角点坐标（用于点击检测等）
// ------------------------------------------------------

type Point struct {
	X, Y float64
}

func GetCorners(p *Position) [4]Point {
	m := TransformToGeoM(p)

	x0, y0 := m.Apply(0, 0)     // 左上
	x1, y1 := m.Apply(p.W, 0)   // 右上
	x2, y2 := m.Apply(p.W, p.H) // 右下
	x3, y3 := m.Apply(0, p.H)   // 左下

	return [4]Point{
		{X: x0, Y: y0},
		{X: x1, Y: y1},
		{X: x2, Y: y2},
		{X: x3, Y: y3},
	}
}

// ------------------------------------------------------
// 4. 获取 AABB（轴对齐包围盒，用于碰撞检测）
// ------------------------------------------------------

func GetAABB(p *Position) (minX, minY, maxX, maxY float64) {
	corners := GetCorners(p)

	minX = corners[0].X
	maxX = corners[0].X
	minY = corners[0].Y
	maxY = corners[0].Y

	for _, c := range corners {
		if c.X < minX {
			minX = c.X
		}
		if c.X > maxX {
			maxX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y > maxY {
			maxY = c.Y
		}
	}

	return
}

func NewPosition(x, y, w, h float64) Position {
	return Position{
		X:          x,
		Y:          y,
		W:          w,
		H:          h,
		OriginX:    w / 2,
		OriginY:    h / 2,
		Alpha:      1.0,
		ScaleX:     1.0,
		ScaleY:     1.0,
		Rotate:     0.0,
		TranslateX: 0.0,
		TranslateY: 0.0,
	}
}

// 将每个属性拆分为函数调用
func (p *Position) SetX(x float64)                   { p.X = x }
func (p *Position) SetY(y float64)                   { p.Y = y }
func (p *Position) SetW(w float64)                   { p.W = w }
func (p *Position) SetH(h float64)                   { p.H = h }
func (p *Position) SetAlpha(alpha float64)           { p.Alpha = alpha }
func (p *Position) SetScaleX(scaleX float64)         { p.ScaleX = scaleX }
func (p *Position) SetScaleY(scaleY float64)         { p.ScaleY = scaleY }
func (p *Position) SetRotate(rotate float64)         { p.Rotate = rotate }
func (p *Position) SetTranslateX(translateX float64) { p.TranslateX = translateX }
func (p *Position) SetTranslateY(translateY float64) { p.TranslateY = translateY }
func (p *Position) SetOriginX(originX float64)       { p.OriginX = originX }
func (p *Position) SetOriginY(originY float64)       { p.OriginY = originY }

// Get
func (p *Position) GetX() float64          { return p.X }
func (p *Position) GetY() float64          { return p.Y }
func (p *Position) GetW() float64          { return p.W }
func (p *Position) GetH() float64          { return p.H }
func (p *Position) GetAlpha() float64      { return p.Alpha }
func (p *Position) GetScaleX() float64     { return p.ScaleX }
func (p *Position) GetScaleY() float64     { return p.ScaleY }
func (p *Position) GetRotate() float64     { return p.Rotate }
func (p *Position) GetTranslateX() float64 { return p.TranslateX }
func (p *Position) GetTranslateY() float64 { return p.TranslateY }
func (p *Position) GetOriginX() float64    { return p.OriginX }
func (p *Position) GetOriginY() float64    { return p.OriginY }
