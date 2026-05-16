package anim

import (
	"testing"
	"time"
)

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
	if s.Velocity() != velocity {
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
