package lp

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	mu          sync.RWMutex
	systemScale = 1.0
	userScale   = 1.0
)

func clampScale(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 1.0
	}
	return v
}

func SetSystemScale(v float64) {
	mu.Lock()
	systemScale = clampScale(v)
	mu.Unlock()
}

func SetUserScale(v float64) {
	mu.Lock()
	userScale = clampScale(v)
	mu.Unlock()
}

func SystemScale() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return systemScale
}

func UserScale() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return userScale
}

func Scale() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return systemScale * userScale
}

func RefreshSystemScale() float64 {
	v := ebiten.DeviceScaleFactor()
	SetSystemScale(v)
	return SystemScale()
}

func LP(value float64) float64 {
	return value * Scale()
}

func FromLP(value float64) float64 {
	s := Scale()
	if s <= 0 {
		return value
	}
	return value / s
}

func LPInt(v int) int {
	return int(math.Round(LP(float64(v))))
}

func FromLPInt(v int) int {
	return int(math.Round(FromLP(float64(v))))
}

func LPSize(value float64) int {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 1
	}
	return int(math.Ceil(LP(value)))
}
