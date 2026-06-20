package bgrender

import (
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSetLowFreqVolumeKeepsUnitRange(t *testing.T) {
	r := &MeshGradientRenderer{}
	r.SetLowFreqVolume(0.5)
	if math.Abs(r.volume-0.5) > 1e-9 {
		t.Fatalf("volume = %v, want 0.5", r.volume)
	}
	r.SetLowFreqVolume(2)
	if r.volume != 1 {
		t.Fatalf("volume = %v, want clamped 1", r.volume)
	}
	r.SetLowFreqVolume(-1)
	if r.volume != 0 {
		t.Fatalf("volume = %v, want clamped 0", r.volume)
	}
}

func TestShaderVisualVolumeUsesSmallShaderRange(t *testing.T) {
	if got := shaderVisualVolume(0); got != 0 {
		t.Fatalf("visual volume at zero = %v, want 0", got)
	}
	if got := shaderVisualVolume(1); math.Abs(got-0.22) > 1e-9 {
		t.Fatalf("visual volume at one = %v, want 0.22", got)
	}
	if got := shaderVisualVolume(2); math.Abs(got-0.22) > 1e-9 {
		t.Fatalf("visual volume above one = %v, want clamped 0.22", got)
	}
}

func TestLowFreqVolumeAttackIsFasterThanRelease(t *testing.T) {
	attack := &MeshGradientRenderer{volume: 1, smoothedVolume: 0}
	attack.onTick(25 * time.Millisecond)
	if attack.smoothedVolume <= 0.2 || attack.smoothedVolume >= 0.7 {
		t.Fatalf("attack smoothed volume = %v, want a softened rise", attack.smoothedVolume)
	}

	release := &MeshGradientRenderer{volume: 0, smoothedVolume: 1}
	release.onTick(25 * time.Millisecond)
	if release.smoothedVolume >= 0.9 {
		t.Fatalf("release smoothed volume = %v, want some decay", release.smoothedVolume)
	}
	if release.smoothedVolume <= 0.7 {
		t.Fatalf("release smoothed volume = %v, want slower than attack", release.smoothedVolume)
	}
}

func TestScaleVerticesAroundCenter(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0},
		{DstX: 100, DstY: 100},
	}
	scaleVerticesAroundCenter(verts, 100, 100, 1.2)
	if verts[0].DstX >= 0 || verts[0].DstY >= 0 {
		t.Fatalf("top-left vertex = %+v, want expanded outward", verts[0])
	}
	if verts[1].DstX <= 100 || verts[1].DstY <= 100 {
		t.Fatalf("bottom-right vertex = %+v, want expanded outward", verts[1])
	}
}

func TestBackgroundScaleRespondsToVolume(t *testing.T) {
	r := &MeshGradientRenderer{smoothedVolume: 1, smoothedScale: 1, scaleStrength: 0.05}
	r.onTick(50 * time.Millisecond)
	if r.smoothedScale <= 1.015 {
		t.Fatalf("smoothed scale = %v, want noticeable zoom", r.smoothedScale)
	}
	if r.smoothedScale > 1.05 {
		t.Fatalf("smoothed scale = %v, want moderate zoom", r.smoothedScale)
	}
}

func TestBackgroundSaturationForVolume(t *testing.T) {
	if got := backgroundSaturationForVolume(0, 1.45); got != 1 {
		t.Fatalf("saturation at zero = %v, want 1", got)
	}
	if got := backgroundSaturationForVolume(1, 1.45); math.Abs(got-2.45) > 1e-9 {
		t.Fatalf("saturation at one = %v, want 2.45", got)
	}
	if got := backgroundSaturationForVolume(2, 1.45); math.Abs(got-2.45) > 1e-9 {
		t.Fatalf("saturation above one = %v, want clamped 2.45", got)
	}
	if got := backgroundSaturationForVolume(1, 4); math.Abs(got-4) > 1e-9 {
		t.Fatalf("saturation strength above range = %v, want clamped 4", got)
	}
}

func TestBackgroundSaturationRespondsToVolume(t *testing.T) {
	r := &MeshGradientRenderer{smoothedVolume: 1, smoothedSat: 1, satStrength: 1.45}
	r.onTick(50 * time.Millisecond)
	if r.smoothedSat <= 1.15 {
		t.Fatalf("smoothed saturation = %v, want visible boost", r.smoothedSat)
	}
	if r.smoothedSat > 2.45 {
		t.Fatalf("smoothed saturation = %v, want bounded boost", r.smoothedSat)
	}
}

func TestBackgroundCoverScaleForVolume(t *testing.T) {
	if got := backgroundCoverScaleForVolume(0, 0.1); got != 1 {
		t.Fatalf("cover scale at zero = %v, want 1", got)
	}
	if got := backgroundCoverScaleForVolume(1, 0.1); math.Abs(got-1.1) > 1e-9 {
		t.Fatalf("cover scale at one = %v, want 1.1", got)
	}
	if got := backgroundCoverScaleForVolume(2, 0.1); math.Abs(got-1.1) > 1e-9 {
		t.Fatalf("cover scale above one = %v, want clamped 1.1", got)
	}
	if got := backgroundCoverScaleForVolume(1, 2); math.Abs(got-2) > 1e-9 {
		t.Fatalf("cover scale strength above range = %v, want clamped 2", got)
	}
}

func TestBackgroundCoverScaleRespondsToVolume(t *testing.T) {
	r := &MeshGradientRenderer{smoothedVolume: 1, smoothedCover: 1, coverStrength: 0.1}
	r.onTick(50 * time.Millisecond)
	if r.smoothedCover <= 1.04 {
		t.Fatalf("smoothed cover scale = %v, want visible boost", r.smoothedCover)
	}
	if r.smoothedCover > 1.1 {
		t.Fatalf("smoothed cover scale = %v, want bounded boost", r.smoothedCover)
	}
}

func TestRendererStrengthSettersClamp(t *testing.T) {
	r := &MeshGradientRenderer{}
	r.SetScaleStrength(1)
	if r.ScaleStrength() != 0.5 {
		t.Fatalf("scale strength = %v, want 0.5", r.ScaleStrength())
	}
	r.SetSaturationStrength(5)
	if r.SaturationStrength() != 3 {
		t.Fatalf("saturation strength = %v, want 3", r.SaturationStrength())
	}
	r.SetCoverScaleStrength(1)
	if r.CoverScaleStrength() != 1 {
		t.Fatalf("cover scale strength = %v, want 1", r.CoverScaleStrength())
	}
}
