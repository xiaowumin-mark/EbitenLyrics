package bgrender

import (
	"errors"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type ControlPoint struct {
	Color    [3]float64
	Location [2]float64
	UTangent [2]float64
	VTangent [2]float64

	uRot   float64
	vRot   float64
	uScale float64
	vScale float64
}

func newControlPoint() *ControlPoint {
	p := &ControlPoint{
		Color:  [3]float64{1, 1, 1},
		uScale: 1,
		vScale: 1,
	}
	p.updateUTangent()
	p.updateVTangent()
	return p
}

func (p *ControlPoint) URot() float64 {
	return p.uRot
}

func (p *ControlPoint) VRot() float64 {
	return p.vRot
}

func (p *ControlPoint) UScale() float64 {
	return p.uScale
}

func (p *ControlPoint) VScale() float64 {
	return p.vScale
}

func (p *ControlPoint) SetURot(value float64) {
	p.uRot = value
	p.updateUTangent()
}

func (p *ControlPoint) SetVRot(value float64) {
	p.vRot = value
	p.updateVTangent()
}

func (p *ControlPoint) SetUScale(value float64) {
	p.uScale = value
	p.updateUTangent()
}

func (p *ControlPoint) SetVScale(value float64) {
	p.vScale = value
	p.updateVTangent()
}

func (p *ControlPoint) updateUTangent() {
	p.UTangent[0] = math.Cos(p.uRot) * p.uScale
	p.UTangent[1] = math.Sin(p.uRot) * p.uScale
}

func (p *ControlPoint) updateVTangent() {
	p.VTangent[0] = -math.Sin(p.vRot) * p.vScale
	p.VTangent[1] = math.Cos(p.vRot) * p.vScale
}

type BHPMesh struct {
	subDivisions int
	cpWidth      int
	cpHeight     int

	controlPoints []*ControlPoint

	vertexWidth  int
	vertexHeight int

	positionsX []float32
	positionsY []float32
	uvX        []float32
	uvY        []float32
	colorR     []float32
	colorG     []float32
	colorB     []float32

	vertices []ebiten.Vertex
	indices  []uint16

	wireFrame bool
}

func NewBHPMesh() *BHPMesh {
	m := &BHPMesh{
		subDivisions: 10,
		cpWidth:      3,
		cpHeight:     3,
	}
	_ = m.ResizeControlPoints(3, 3)
	return m
}

func (m *BHPMesh) SetWireFrame(enable bool) {
	m.wireFrame = enable
	// Ebiten DrawTrianglesShader has no line primitive. Keep triangle topology.
}

func (m *BHPMesh) GetControlPoint(x, y int) *ControlPoint {
	if x < 0 || x >= m.cpWidth || y < 0 || y >= m.cpHeight {
		return nil
	}
	return m.controlPoints[x+y*m.cpWidth]
}

func (m *BHPMesh) ResetSubdivition(subDivisions int) {
	if subDivisions < 2 {
		subDivisions = 2
	}
	m.subDivisions = subDivisions
	m.rebuildBuffers()
}

func (m *BHPMesh) ResizeControlPoints(width, height int) error {
	if width < 2 || height < 2 {
		return errors.New("control points must be at least 2x2")
	}
	m.cpWidth = width
	m.cpHeight = height
	m.controlPoints = make([]*ControlPoint, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := newControlPoint()
			p.Location[0] = float64(x)/float64(width-1)*2 - 1
			p.Location[1] = float64(y)/float64(height-1)*2 - 1
			p.SetUScale(2 / float64(width-1))
			p.SetVScale(2 / float64(height-1))
			m.controlPoints[x+y*width] = p
		}
	}
	m.rebuildBuffers()
	return nil
}

func (m *BHPMesh) rebuildBuffers() {
	if m.cpWidth < 2 || m.cpHeight < 2 {
		return
	}

	maxSub := m.maxSafeSubdivision()
	if m.subDivisions > maxSub {
		m.subDivisions = maxSub
	}
	if m.subDivisions < 2 {
		m.subDivisions = 2
	}

	m.vertexWidth = (m.cpWidth - 1) * m.subDivisions
	m.vertexHeight = (m.cpHeight - 1) * m.subDivisions
	vertexCount := m.vertexWidth * m.vertexHeight
	if vertexCount <= 0 {
		return
	}

	m.positionsX = make([]float32, vertexCount)
	m.positionsY = make([]float32, vertexCount)
	m.uvX = make([]float32, vertexCount)
	m.uvY = make([]float32, vertexCount)
	m.colorR = make([]float32, vertexCount)
	m.colorG = make([]float32, vertexCount)
	m.colorB = make([]float32, vertexCount)
	m.vertices = make([]ebiten.Vertex, vertexCount)

	cells := (m.vertexWidth - 1) * (m.vertexHeight - 1)
	if cells <= 0 {
		m.indices = nil
		return
	}
	m.indices = make([]uint16, cells*6)
	idx := 0
	for y := 0; y < m.vertexHeight-1; y++ {
		for x := 0; x < m.vertexWidth-1; x++ {
			i00 := uint16(y*m.vertexWidth + x)
			i10 := uint16(y*m.vertexWidth + x + 1)
			i01 := uint16((y+1)*m.vertexWidth + x)
			i11 := uint16((y+1)*m.vertexWidth + x + 1)

			m.indices[idx] = i00
			m.indices[idx+1] = i10
			m.indices[idx+2] = i01
			m.indices[idx+3] = i10
			m.indices[idx+4] = i11
			m.indices[idx+5] = i01
			idx += 6
		}
	}
}

func (m *BHPMesh) maxSafeSubdivision() int {
	patchCount := (m.cpWidth - 1) * (m.cpHeight - 1)
	if patchCount <= 0 {
		return 2
	}
	maxSub := int(math.Floor(math.Sqrt(float64(math.MaxUint16) / float64(patchCount))))
	if maxSub < 2 {
		return 2
	}
	return maxSub
}

func hermite(t, p0, p1, m0, m1 float64) float64 {
	t2 := t * t
	t3 := t2 * t
	h00 := 2*t3 - 3*t2 + 1
	h10 := t3 - 2*t2 + t
	h01 := -2*t3 + 3*t2
	h11 := t3 - t2
	return h00*p0 + h01*p1 + h10*m0 + h11*m1
}

func evalPatch(
	u, v float64,
	p00, p01, p10, p11 *ControlPoint,
	component int,
) float64 {
	corner00 := p00.Location[component]
	corner01 := p01.Location[component]
	corner10 := p10.Location[component]
	corner11 := p11.Location[component]
	dv00 := p00.VTangent[component]
	dv01 := p01.VTangent[component]
	dv10 := p10.VTangent[component]
	dv11 := p11.VTangent[component]
	du00 := p00.UTangent[component]
	du01 := p01.UTangent[component]
	du10 := p10.UTangent[component]
	du11 := p11.UTangent[component]

	a0 := hermite(v, corner00, corner01, dv00, dv01)
	a1 := hermite(v, corner10, corner11, dv10, dv11)
	b0 := hermite(v, du00, du01, 0, 0)
	b1 := hermite(v, du10, du11, 0, 0)
	return hermite(u, a0, a1, b0, b1)
}

func evalPatchColor(
	u, v float64,
	p00, p01, p10, p11 *ControlPoint,
	component int,
) float64 {
	a0 := hermite(v, p00.Color[component], p01.Color[component], 0, 0)
	a1 := hermite(v, p10.Color[component], p11.Color[component], 0, 0)
	return hermite(u, a0, a1, 0, 0)
}

func (m *BHPMesh) UpdateMesh() {
	if m.cpWidth < 2 || m.cpHeight < 2 || m.subDivisions < 2 {
		return
	}
	if len(m.positionsX) == 0 {
		m.rebuildBuffers()
	}
	subM1 := m.subDivisions - 1
	if subM1 <= 0 {
		return
	}
	totalU := subM1 * (m.cpWidth - 1)
	totalV := subM1 * (m.cpHeight - 1)
	if totalU <= 0 || totalV <= 0 {
		return
	}
	invTotalU := 1.0 / float64(totalU)
	invTotalV := 1.0 / float64(totalV)

	for patchY := 0; patchY < m.cpHeight-1; patchY++ {
		for patchX := 0; patchX < m.cpWidth-1; patchX++ {
			p00 := m.GetControlPoint(patchX, patchY)
			p01 := m.GetControlPoint(patchX, patchY+1)
			p10 := m.GetControlPoint(patchX+1, patchY)
			p11 := m.GetControlPoint(patchX+1, patchY+1)
			if p00 == nil || p01 == nil || p10 == nil || p11 == nil {
				continue
			}

			baseX := patchX * m.subDivisions
			baseY := patchY * m.subDivisions
			for u := 0; u < m.subDivisions; u++ {
				nu := float64(u) / float64(subM1)
				for v := 0; v < m.subDivisions; v++ {
					nv := float64(v) / float64(subM1)
					gridX := baseX + u
					gridY := baseY + v
					vertexIdx := gridX + gridY*m.vertexWidth
					if vertexIdx < 0 || vertexIdx >= len(m.positionsX) {
						continue
					}

					px := evalPatch(nu, nv, p00, p01, p10, p11, 0)
					py := evalPatch(nu, nv, p00, p01, p10, p11, 1)
					pr := evalPatchColor(nu, nv, p00, p01, p10, p11, 0)
					pg := evalPatchColor(nu, nv, p00, p01, p10, p11, 1)
					pb := evalPatchColor(nu, nv, p00, p01, p10, p11, 2)

					globalU := patchX*subM1 + u
					globalV := patchY*subM1 + v

					m.positionsX[vertexIdx] = float32(px)
					m.positionsY[vertexIdx] = float32(py)
					m.uvX[vertexIdx] = float32(float64(globalU) * invTotalU)
					m.uvY[vertexIdx] = float32(1.0 - float64(globalV)*invTotalV)
					m.colorR[vertexIdx] = float32(clamp(pr, 0, 1))
					m.colorG[vertexIdx] = float32(clamp(pg, 0, 1))
					m.colorB[vertexIdx] = float32(clamp(pb, 0, 1))
				}
			}
		}
	}
}

func (m *BHPMesh) Vertices(renderWidth, renderHeight, texWidth, texHeight int, aspect float64, manual bool) []ebiten.Vertex {
	if renderWidth <= 0 || renderHeight <= 0 {
		return nil
	}
	if texWidth <= 0 || texHeight <= 0 {
		return nil
	}
	if len(m.vertices) == 0 {
		return nil
	}
	if aspect <= 0 {
		aspect = 1
	}

	w := float64(renderWidth)
	h := float64(renderHeight)
	tw := float32(texWidth)
	th := float32(texHeight)
	for i := range m.vertices {
		x := float64(m.positionsX[i])
		y := float64(m.positionsY[i])
		if !manual {
			if aspect > 1 {
				y *= aspect
			} else {
				x /= aspect
			}
		}

		dstX := (x*0.5 + 0.5) * w
		dstY := (y*0.5 + 0.5) * h
		m.vertices[i] = ebiten.Vertex{
			DstX: float32(dstX),
			DstY: float32(dstY),
			// DrawTrianglesShader expects source coordinates in source-image pixels.
			// Ebiten will convert these to texture coordinates before entering Kage.
			SrcX:   m.uvX[i] * tw,
			SrcY:   m.uvY[i] * th,
			ColorR: m.colorR[i],
			ColorG: m.colorG[i],
			ColorB: m.colorB[i],
			ColorA: 1,
		}
	}
	return m.vertices
}

func (m *BHPMesh) Indices() []uint16 {
	return m.indices
}
