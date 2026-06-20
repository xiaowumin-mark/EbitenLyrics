package lyrics

import (
	"testing"
	"time"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestApplyRefHorizontalLayoutUsesFullWidth(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(line, 1000, false)

	if line.GetPosition().GetX() != 0 {
		t.Fatalf("x = %v, want 0", line.GetPosition().GetX())
	}
	if line.GetPosition().GetW() != 1000 {
		t.Fatalf("width = %v, want 1000", line.GetPosition().GetW())
	}
	if line.EffectivePaddingLeft() != 20 || line.EffectivePaddingRight() != 20 {
		t.Fatalf("padding = %v,%v want 20,20", line.EffectivePaddingLeft(), line.EffectivePaddingRight())
	}
}

func TestApplyRefHorizontalLayoutAddsDuetAvoidance(t *testing.T) {
	normal := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	duet := NewLine(0, time.Second, true, false, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(normal, 1000, true)
	applyRefHorizontalLayout(duet, 1000, true)

	if normal.EffectivePaddingRight() != 120 {
		t.Fatalf("normal right padding = %v, want 120", normal.EffectivePaddingRight())
	}
	if duet.EffectivePaddingLeft() != 120 {
		t.Fatalf("duet left padding = %v, want 120", duet.EffectivePaddingLeft())
	}
	if duet.GetPosition().GetX() != 0 || duet.GetPosition().GetW() != 1000 {
		t.Fatalf("duet position = %v,%v want 0,1000", duet.GetPosition().GetX(), duet.GetPosition().GetW())
	}
}

func TestApplyRefHorizontalLayoutAlignsBackgroundPaddingWithMainLine(t *testing.T) {
	mainLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 48)
	bgLine := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(mainLine, 1000, false)
	applyRefHorizontalLayout(bgLine, 1000, false)

	if bgLine.EffectivePaddingLeft() != mainLine.EffectivePaddingLeft() {
		t.Fatalf("background left padding = %v, want main padding %v", bgLine.EffectivePaddingLeft(), mainLine.EffectivePaddingLeft())
	}
	if bgLine.EffectivePaddingRight() != mainLine.EffectivePaddingRight() {
		t.Fatalf("background right padding = %v, want main padding %v", bgLine.EffectivePaddingRight(), mainLine.EffectivePaddingRight())
	}
}

func TestApplyRefHorizontalLayoutAlignsBackgroundDuetAvoidance(t *testing.T) {
	mainLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 48)
	bgLine := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 32)
	duetLine := NewLine(0, time.Second, true, false, "", nil, ft.FontRequest{}, 48)
	bgDuetLine := NewLine(0, time.Second, true, true, "", nil, ft.FontRequest{}, 32)
	applyRefHorizontalLayout(mainLine, 1000, true)
	applyRefHorizontalLayout(bgLine, 1000, true)
	applyRefHorizontalLayout(duetLine, 1000, true)
	applyRefHorizontalLayout(bgDuetLine, 1000, true)

	if bgLine.EffectivePaddingRight() != mainLine.EffectivePaddingRight() {
		t.Fatalf("background right padding = %v, want main padding %v", bgLine.EffectivePaddingRight(), mainLine.EffectivePaddingRight())
	}
	if bgDuetLine.EffectivePaddingLeft() != duetLine.EffectivePaddingLeft() {
		t.Fatalf("background duet left padding = %v, want duet padding %v", bgDuetLine.EffectivePaddingLeft(), duetLine.EffectivePaddingLeft())
	}
}

func TestLineBasePaddingIsClampedForLargeFonts(t *testing.T) {
	if got := lineBasePadding(80, 1600); got != 40 {
		t.Fatalf("padding = %v, want clamped 40", got)
	}
}

func TestLineDuetAvoidanceIsClamped(t *testing.T) {
	if got := lineDuetAvoidance(2000); got != 140 {
		t.Fatalf("avoidance = %v, want clamped 140", got)
	}
}

func TestLineSetPaddingKeepsCompatibility(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.SetPadding(20)
	if line.EffectivePaddingLeft() != 20 || line.EffectivePaddingRight() != 20 {
		t.Fatalf("effective padding = %v,%v want 20,20", line.EffectivePaddingLeft(), line.EffectivePaddingRight())
	}
}
