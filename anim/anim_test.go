package anim

import (
	"math"
	"testing"
	"time"
)

func TestEaseFunctionsStayFiniteAtCommonSamples(t *testing.T) {
	samples := []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1}
	eases := map[string]EaseFunc{
		"Linear":           Linear,
		"EaseIn":           EaseIn,
		"EaseOut":          EaseOut,
		"EaseInOut":        EaseInOut,
		"EaseOutElastic":   EaseOutElastic,
		"EaseInSine":       EaseInSine,
		"EaseOutSine":      EaseOutSine,
		"EaseInOutSine":    EaseInOutSine,
		"EaseInCubic":      EaseInCubic,
		"EaseOutCubic":     EaseOutCubic,
		"EaseInOutCubic":   EaseInOutCubic,
		"EaseInQuart":      EaseInQuart,
		"EaseOutQuart":     EaseOutQuart,
		"EaseInOutQuart":   EaseInOutQuart,
		"EaseInQuint":      EaseInQuint,
		"EaseOutQuint":     EaseOutQuint,
		"EaseInOutQuint":   EaseInOutQuint,
		"EaseInExpo":       EaseInExpo,
		"EaseOutExpo":      EaseOutExpo,
		"EaseInOutExpo":    EaseInOutExpo,
		"EaseInCirc":       EaseInCirc,
		"EaseOutCirc":      EaseOutCirc,
		"EaseInOutCirc":    EaseInOutCirc,
		"EaseInBack":       EaseInBack,
		"EaseOutBack":      EaseOutBack,
		"EaseInOutBack":    EaseInOutBack,
		"EaseInBounce":     EaseInBounce,
		"EaseOutBounce":    EaseOutBounce,
		"EaseInOutBounce":  EaseInOutBounce,
		"SmoothStep":       SmoothStep,
		"SmootherStep":     SmootherStep,
		"NewEaseSpringOut": NewEaseSpringOut(6, 8),
	}

	for name, ease := range eases {
		for _, sample := range samples {
			got := ease(sample)
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s returned invalid value at %.2f: %v", name, sample, got)
			}
		}
	}
}

func TestEaseFunctionsHonorBasicEndpoints(t *testing.T) {
	eases := map[string]EaseFunc{
		"Linear":           Linear,
		"EaseIn":           EaseIn,
		"EaseOut":          EaseOut,
		"EaseInOut":        EaseInOut,
		"EaseOutElastic":   EaseOutElastic,
		"EaseInSine":       EaseInSine,
		"EaseOutSine":      EaseOutSine,
		"EaseInOutSine":    EaseInOutSine,
		"EaseInCubic":      EaseInCubic,
		"EaseOutCubic":     EaseOutCubic,
		"EaseInOutCubic":   EaseInOutCubic,
		"EaseInQuart":      EaseInQuart,
		"EaseOutQuart":     EaseOutQuart,
		"EaseInOutQuart":   EaseInOutQuart,
		"EaseInQuint":      EaseInQuint,
		"EaseOutQuint":     EaseOutQuint,
		"EaseInOutQuint":   EaseInOutQuint,
		"EaseInExpo":       EaseInExpo,
		"EaseOutExpo":      EaseOutExpo,
		"EaseInOutExpo":    EaseInOutExpo,
		"EaseInCirc":       EaseInCirc,
		"EaseOutCirc":      EaseOutCirc,
		"EaseInOutCirc":    EaseInOutCirc,
		"EaseInBack":       EaseInBack,
		"EaseOutBack":      EaseOutBack,
		"EaseInOutBack":    EaseInOutBack,
		"EaseInBounce":     EaseInBounce,
		"EaseOutBounce":    EaseOutBounce,
		"EaseInOutBounce":  EaseInOutBounce,
		"SmoothStep":       SmoothStep,
		"SmootherStep":     SmootherStep,
		"NewEaseSpringOut": NewEaseSpringOut(6, 8),
	}

	for name, ease := range eases {
		if got := ease(0); math.Abs(got) > 1e-9 {
			t.Fatalf("%s expected ease(0)=0, got %v", name, got)
		}
		if got := ease(1); math.Abs(got-1) > 1e-9 {
			t.Fatalf("%s expected ease(1)=1, got %v", name, got)
		}
	}
}

func TestEaseCombinators(t *testing.T) {
	reversed := ReverseEase(EaseIn)
	if got := reversed(0.75); got <= 0 {
		t.Fatalf("ReverseEase produced unexpected value: %v", got)
	}

	mirrored := MirrorEase(EaseIn)
	left := mirrored(0.25)
	right := mirrored(0.75)
	if math.Abs((1-right)-left) > 1e-9 {
		t.Fatalf("MirrorEase symmetry mismatch: left=%v right=%v", left, right)
	}

	chained := ChainEase(EaseIn, EaseOut, 0.3)
	if got := chained(0); math.Abs(got) > 1e-9 {
		t.Fatalf("ChainEase expected 0 at start, got %v", got)
	}
	if got := chained(1); math.Abs(got-1) > 1e-9 {
		t.Fatalf("ChainEase expected 1 at end, got %v", got)
	}
}

func TestManagerDropsCanceledPendingTween(t *testing.T) {
	manager := NewManager(false)
	updated := false
	finished := false
	tween := NewTween(
		"pending-tween",
		time.Second,
		0,
		1,
		0,
		1,
		Linear,
		func(float64) { updated = true },
		func() { finished = true },
	)

	manager.Add(tween)
	tween.Cancel()
	manager.Update(16 * time.Millisecond)

	if updated || finished {
		t.Fatalf("canceled pending tween ran callbacks: updated=%v finished=%v", updated, finished)
	}
	if len(manager.pending) != 0 || len(manager.active) != 0 {
		t.Fatalf("manager retained canceled tween: pending=%d active=%d", len(manager.pending), len(manager.active))
	}
}

func TestManagerDropsCanceledPendingKeyframeAnimation(t *testing.T) {
	manager := NewManager(false)
	updated := false
	finished := false
	animation := NewKeyframeAnimation(
		"pending-keyframe",
		time.Second,
		0,
		1,
		false,
		[]Keyframe{
			{Offset: 0, Values: []float64{0}},
			{Offset: 1, Values: []float64{1}},
		},
		func([]float64) { updated = true },
		func() { finished = true },
	)

	manager.Add(animation)
	animation.Cancel()
	manager.Update(16 * time.Millisecond)

	if updated || finished {
		t.Fatalf("canceled pending keyframe ran callbacks: updated=%v finished=%v", updated, finished)
	}
	if len(manager.pending) != 0 || len(manager.active) != 0 {
		t.Fatalf("manager retained canceled keyframe: pending=%d active=%d", len(manager.pending), len(manager.active))
	}
}
