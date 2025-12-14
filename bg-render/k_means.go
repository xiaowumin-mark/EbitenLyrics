package bgrender

import (
	"image"
	"math"
	"math/rand"
	"sort"
)

type ColorLab struct {
	L, A, B float64
}

type PaletteColor struct {
	R, G, B uint8
	Weight  float64 // 像素占比
}

type Sample struct {
	Lab ColorLab
	RGB [3]uint8
}

func labToRGB(lab ColorLab) (uint8, uint8, uint8) {
	y := (lab.L + 16) / 116
	x := lab.A/500 + y
	z := y - lab.B/200

	f := func(t float64) float64 {
		if t*t*t > 0.008856 {
			return t * t * t
		}
		return (t - 16.0/116.0) / 7.787
	}

	X := f(x) * 0.95047
	Y := f(y) * 1.00000
	Z := f(z) * 1.08883

	R := X*3.2406 + Y*-1.5372 + Z*-0.4986
	G := X*-0.9689 + Y*1.8758 + Z*0.0415
	B := X*0.0557 + Y*-0.2040 + Z*1.0570

	linearToSRGB := func(c float64) float64 {
		if c <= 0.0031308 {
			return 12.92 * c
		}
		return 1.055*math.Pow(c, 1.0/2.4) - 0.055
	}

	r := math.Max(0, math.Min(1, linearToSRGB(R)))
	g := math.Max(0, math.Min(1, linearToSRGB(G)))
	b := math.Max(0, math.Min(1, linearToSRGB(B)))

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

func labDist(a, b ColorLab) float64 {
	dL := a.L - b.L
	dA := a.A - b.A
	dB := a.B - b.B
	return dL*dL + dA*dA + dB*dB
}

type Cluster struct {
	Center  ColorLab
	Samples []Sample
}

func kMeans(samples []Sample, k, iterations int) []Cluster {
	clusters := make([]Cluster, k)

	// 初始化
	for i := 0; i < k; i++ {
		clusters[i].Center = samples[rand.Intn(len(samples))].Lab
	}

	for iter := 0; iter < iterations; iter++ {
		for i := range clusters {
			clusters[i].Samples = clusters[i].Samples[:0]
		}

		// 分配
		for _, s := range samples {
			best := 0
			bestDist := math.MaxFloat64
			for i := range clusters {
				d := labDist(s.Lab, clusters[i].Center)
				if d < bestDist {
					bestDist = d
					best = i
				}
			}
			clusters[best].Samples = append(clusters[best].Samples, s)
		}

		// 更新中心
		for i := range clusters {
			if len(clusters[i].Samples) == 0 {
				continue
			}
			var l, a, b float64
			for _, s := range clusters[i].Samples {
				l += s.Lab.L
				a += s.Lab.A
				b += s.Lab.B
			}
			n := float64(len(clusters[i].Samples))
			clusters[i].Center = ColorLab{l / n, a / n, b / n}
		}
	}

	return clusters
}

func saturation(r, g, b uint8) float64 {
	fr := float64(r) / 255
	fg := float64(g) / 255
	fb := float64(b) / 255
	max := math.Max(fr, math.Max(fg, fb))
	min := math.Min(fr, math.Min(fg, fb))
	if max == 0 {
		return 0
	}
	return (max - min) / max
}

func pickRepresentative(c Cluster) (rgb [3]uint8, ok bool) {
	if len(c.Samples) == 0 {
		return rgb, false
	}

	bestDist := math.MaxFloat64
	for _, s := range c.Samples {
		d := labDist(s.Lab, c.Center)
		if d < bestDist {
			bestDist = d
			rgb = s.RGB
		}
	}

	// 过滤条件（经验值）
	sat := saturation(rgb[0], rgb[1], rgb[2])
	lum := c.Center.L

	if sat < 0.15 || lum < 20 || lum > 90 {
		return rgb, false
	}

	return rgb, true
}

func ExtractPalette(img image.Image, k int) []PaletteColor {
	b := img.Bounds()
	step := 4

	var samples []Sample

	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			samples = append(samples, Sample{
				Lab: rgbToLab(r8, g8, b8),
				RGB: [3]uint8{r8, g8, b8},
			})
		}
	}

	clusters := kMeans(samples, k, 12)

	total := float64(len(samples))
	var palette []PaletteColor

	for _, c := range clusters {
		rgb, ok := pickRepresentative(c)
		if !ok {
			continue
		}
		palette = append(palette, PaletteColor{
			R:      rgb[0],
			G:      rgb[1],
			B:      rgb[2],
			Weight: float64(len(c.Samples)) / total,
		})
	}

	sort.Slice(palette, func(i, j int) bool {
		return palette[i].Weight > palette[j].Weight
	})

	return palette
}

func rgbToLab(r, g, b uint8) ColorLab {
	// 1️⃣ sRGB → Linear RGB
	R := srgbToLinear(float64(r) / 255.0)
	G := srgbToLinear(float64(g) / 255.0)
	B := srgbToLinear(float64(b) / 255.0)

	// 2️⃣ Linear RGB → XYZ (D65)
	X := R*0.4124 + G*0.3576 + B*0.1805
	Y := R*0.2126 + G*0.7152 + B*0.0722
	Z := R*0.0193 + G*0.1192 + B*0.9505

	// Normalize
	X /= 0.95047
	Y /= 1.00000
	Z /= 1.08883

	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Pow(t, 1.0/3.0)
		}
		return 7.787*t + 16.0/116.0
	}

	fx, fy, fz := f(X), f(Y), f(Z)

	return ColorLab{
		L: 116*fy - 16,
		A: 500 * (fx - fy),
		B: 200 * (fy - fz),
	}
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
