package anim

import (
	"math"
	"testing"
	"time"
)

func nearlyEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func refUnderdampedPosition(from, velocity, to float64, params SpringParams, t float64) float64 {
	delta := to - from
	dampingFrequency := math.Sqrt(4.0*params.Mass*params.Stiffness - params.Damping*params.Damping)
	leftover := (params.Damping*delta - 2.0*params.Mass*velocity) / dampingFrequency
	dfm := (0.5 * dampingFrequency) / params.Mass
	dm := -(0.5 * params.Damping) / params.Mass
	return to - (math.Cos(t*dfm)*delta+math.Sin(t*dfm)*leftover)*math.Exp(t*dm)
}

func refCriticalPosition(from, velocity, to float64, params SpringParams, t float64) float64 {
	delta := to - from
	w := -math.Sqrt(params.Stiffness / params.Mass)
	leftover := -w*delta - velocity
	return to - (delta+t*leftover)*math.Exp(t*w)
}

func TestSpringMovesTowardTarget(t *testing.T) {
	s := NewSpring(0, SpringParams{Mass: 1, Damping: 15, Stiffness: 90})
	s.SetTargetPosition(100)
	s.Update(16 * time.Millisecond)
	if s.CurrentPosition() <= 0 {
		t.Fatalf("spring did not move toward target, position = %v", s.CurrentPosition())
	}
}

func TestSpringRetargetPreservesVelocity(t *testing.T) {
	s := NewSpring(0, SpringParams{Mass: 1, Damping: 10, Stiffness: 100})
	s.SetTargetPosition(100)
	s.Update(16 * time.Millisecond)
	velocity := s.Velocity()
	if velocity == 0 {
		t.Fatal("spring should have velocity after update")
	}
	s.SetTargetPosition(200)
	if !nearlyEqual(s.Velocity(), velocity, 1e-9) {
		t.Fatalf("retarget changed velocity = %v, want %v", s.Velocity(), velocity)
	}
}

func TestSpringSetPositionSnapsAndStops(t *testing.T) {
	s := NewSpring(0, SpringParams{})
	s.SetTargetPosition(100)
	s.Update(16 * time.Millisecond)
	s.SetPosition(50)
	if s.CurrentPosition() != 50 || s.TargetPosition() != 50 || s.Velocity() != 0 {
		t.Fatalf("SetPosition state = pos %v target %v velocity %v, want 50 50 0", s.CurrentPosition(), s.TargetPosition(), s.Velocity())
	}
}

func TestSpringDelayedTargetWaitsBeforeMoving(t *testing.T) {
	s := NewSpring(0, SpringParams{Mass: 1, Damping: 15, Stiffness: 90})
	s.SetTargetPosition(100, 50*time.Millisecond)
	s.Update(16 * time.Millisecond)
	if s.TargetPosition() != 0 || s.CurrentPosition() != 0 {
		t.Fatalf("spring moved before delay: pos %v target %v", s.CurrentPosition(), s.TargetPosition())
	}
	s.Update(50 * time.Millisecond)
	if s.TargetPosition() != 100 {
		t.Fatalf("target = %v, want delayed target 100", s.TargetPosition())
	}
	if s.CurrentPosition() <= 0 {
		t.Fatalf("spring should move after delay, position = %v", s.CurrentPosition())
	}
}

func TestSpringMatchesRefUnderdampedSolver(t *testing.T) {
	params := SpringParams{Mass: 0.9, Damping: 15, Stiffness: 90}
	s := NewSpring(0, params)
	s.SetTargetPosition(100)
	s.Update(100 * time.Millisecond)
	want := refUnderdampedPosition(0, 0, 100, params, 0.1)
	if !nearlyEqual(s.CurrentPosition(), want, 1e-9) {
		t.Fatalf("position = %.12f, want ref %.12f", s.CurrentPosition(), want)
	}
}

func TestSpringMatchesRefCriticalSolver(t *testing.T) {
	params := SpringParams{Mass: 1, Damping: 20, Stiffness: 100}
	s := NewSpring(0, params)
	s.SetTargetPosition(100)
	s.Update(100 * time.Millisecond)
	want := refCriticalPosition(0, 0, 100, params, 0.1)
	if !nearlyEqual(s.CurrentPosition(), want, 1e-9) {
		t.Fatalf("position = %.12f, want ref %.12f", s.CurrentPosition(), want)
	}
}

func TestSpringUpdateUsesFullDeltaLikeRef(t *testing.T) {
	params := SpringParams{Mass: 0.9, Damping: 15, Stiffness: 90}
	s := NewSpring(0, params)
	s.SetTargetPosition(100)
	s.Update(120 * time.Millisecond)
	want := refUnderdampedPosition(0, 0, 100, params, 0.12)
	if !nearlyEqual(s.CurrentPosition(), want, 1e-9) {
		t.Fatalf("position = %.12f, want unclamped ref %.12f", s.CurrentPosition(), want)
	}
}
