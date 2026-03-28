package bgrender

// 文件说明：控制点自动生成算法。
// 主要职责：利用噪声与平滑规则生成更自然的背景形变参数。

import "math"

func randomRange(minV, maxV float64) float64 {
	return minV + (maxV-minV)*randFloat64()
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func fract(x float64) float64 {
	return x - math.Floor(x)
}

func noise(x, y float64) float64 {
	return fract(math.Sin(x*12.9898+y*78.233) * 43758.5453)
}

func smoothNoise(x, y float64) float64 {
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	x1 := x0 + 1
	y1 := y0 + 1

	xf := x - x0
	yf := y - y0

	u := xf * xf * (3 - 2*xf)
	v := yf * yf * (3 - 2*yf)

	n00 := noise(x0, y0)
	n10 := noise(x1, y0)
	n01 := noise(x0, y1)
	n11 := noise(x1, y1)

	nx0 := n00*(1-u) + n10*u
	nx1 := n01*(1-u) + n11*u

	return nx0*(1-v) + nx1*v
}

func computeNoiseGradient(
	perlinFn func(x, y float64) float64,
	x, y, epsilon float64,
) (float64, float64) {
	n1 := perlinFn(x+epsilon, y)
	n2 := perlinFn(x-epsilon, y)
	n3 := perlinFn(x, y+epsilon)
	n4 := perlinFn(x, y-epsilon)
	dx := (n1 - n2) / (2 * epsilon)
	dy := (n3 - n4) / (2 * epsilon)
	length := math.Hypot(dx, dy)
	if length <= 1e-9 {
		length = 1
	}
	return dx / length, dy / length
}

func smoothifyControlPoints(
	conf []ControlPointConf,
	w, h int,
	iterations int,
	factor float64,
	factorIterationModifier float64,
) {
	grid := make([][]ControlPointConf, h)
	for j := 0; j < h; j++ {
		grid[j] = make([]ControlPointConf, w)
		for i := 0; i < w; i++ {
			grid[j][i] = conf[j*w+i]
		}
	}

	kernel := [3][3]float64{
		{1, 2, 1},
		{2, 4, 2},
		{1, 2, 1},
	}
	const kernelSum = 16.0

	f := factor
	for iter := 0; iter < iterations; iter++ {
		nextGrid := make([][]ControlPointConf, h)
		for j := 0; j < h; j++ {
			nextGrid[j] = make([]ControlPointConf, w)
			for i := 0; i < w; i++ {
				if i == 0 || i == w-1 || j == 0 || j == h-1 {
					nextGrid[j][i] = grid[j][i]
					continue
				}

				sumX, sumY := 0.0, 0.0
				sumUR, sumVR := 0.0, 0.0
				sumUP, sumVP := 0.0, 0.0
				for dj := -1; dj <= 1; dj++ {
					for di := -1; di <= 1; di++ {
						weight := kernel[dj+1][di+1]
						nb := grid[j+dj][i+di]
						sumX += nb.X * weight
						sumY += nb.Y * weight
						sumUR += nb.UR * weight
						sumVR += nb.VR * weight
						sumUP += nb.UP * weight
						sumVP += nb.VP * weight
					}
				}

				avgX := sumX / kernelSum
				avgY := sumY / kernelSum
				avgUR := sumUR / kernelSum
				avgVR := sumVR / kernelSum
				avgUP := sumUP / kernelSum
				avgVP := sumVP / kernelSum

				cur := grid[j][i]
				nextGrid[j][i] = cp(
					i,
					j,
					cur.X*(1-f)+avgX*f,
					cur.Y*(1-f)+avgY*f,
					cur.UR*(1-f)+avgUR*f,
					cur.VR*(1-f)+avgVR*f,
					cur.UP*(1-f)+avgUP*f,
					cur.VP*(1-f)+avgVP*f,
				)
			}
		}
		grid = nextGrid
		f = clamp(f+factorIterationModifier, 0, 1)
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			conf[j*w+i] = grid[j][i]
		}
	}
}

func GenerateControlPoints(width, height int) ControlPointPreset {
	w := width
	h := height
	if w <= 0 {
		w = int(math.Floor(randomRange(3, 6)))
	}
	if h <= 0 {
		h = int(math.Floor(randomRange(3, 6)))
	}
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}

	variationFraction := randomRange(0.4, 0.6)
	normalOffset := randomRange(0.3, 0.6)
	blendFactor := 0.8
	smoothIters := int(math.Floor(randomRange(3, 5)))
	smoothFactor := randomRange(0.2, 0.3)
	smoothModifier := randomRange(-0.1, -0.05)

	conf := make([]ControlPointConf, 0, w*h)
	dx := 0.0
	dy := 0.0
	if w > 1 {
		dx = 2.0 / float64(w-1)
	}
	if h > 1 {
		dy = 2.0 / float64(h-1)
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			baseX := 0.0
			baseY := 0.0
			if w > 1 {
				baseX = float64(i)/float64(w-1)*2 - 1
			}
			if h > 1 {
				baseY = float64(j)/float64(h-1)*2 - 1
			}

			isBorder := i == 0 || i == w-1 || j == 0 || j == h-1
			pertX := 0.0
			pertY := 0.0
			if !isBorder {
				pertX = randomRange(-variationFraction*dx, variationFraction*dx)
				pertY = randomRange(-variationFraction*dy, variationFraction*dy)
			}

			x := baseX + pertX
			y := baseY + pertY
			ur := 0.0
			vr := 0.0
			up := 1.0
			vp := 1.0
			if !isBorder {
				ur = randomRange(-60, 60)
				vr = randomRange(-60, 60)
				up = randomRange(0.8, 1.2)
				vp = randomRange(0.8, 1.2)

				uNorm := (baseX + 1) / 2
				vNorm := (baseY + 1) / 2
				nx, ny := computeNoiseGradient(smoothNoise, uNorm, vNorm, 0.001)

				offsetX := nx * normalOffset
				offsetY := ny * normalOffset
				distToBorder := math.Min(math.Min(uNorm, 1-uNorm), math.Min(vNorm, 1-vNorm))
				weight := smoothstep(0, 1, distToBorder)
				offsetX *= weight
				offsetY *= weight

				x = x*(1-blendFactor) + (x+offsetX)*blendFactor
				y = y*(1-blendFactor) + (y+offsetY)*blendFactor
			}
			conf = append(conf, cp(i, j, x, y, ur, vr, up, vp))
		}
	}

	smoothifyControlPoints(conf, w, h, smoothIters, smoothFactor, smoothModifier)
	return controlPointPreset(w, h, conf...)
}
