package lyrics

import (
	"math"
	"testing"
	"time"

	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestDefaultSpringParamsMatchRefBaseline(t *testing.T) {
	if defaultLinePosYSpringParams.Mass != 0.9 || defaultLinePosYSpringParams.Damping != 15 || defaultLinePosYSpringParams.Stiffness != 90 {
		t.Fatalf("posY params = %+v, want ref baseline mass 0.9 damping 15 stiffness 90", defaultLinePosYSpringParams)
	}
	if backgroundLineScaleSpringParams.Mass != 1 || backgroundLineScaleSpringParams.Damping != 20 || backgroundLineScaleSpringParams.Stiffness != 50 {
		t.Fatalf("bg scale params = %+v, want ref baseline mass 1 damping 20 stiffness 50", backgroundLineScaleSpringParams)
	}
}

func TestSetLineSpringTargetsForceSnapsPositionAndScale(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	setLineSpringTargets(line, 120, 0.97, true)

	if got := line.GetPosition().GetY(); got != 120 {
		t.Fatalf("y = %v, want 120", got)
	}
	if got := line.GetPosition().GetScaleX(); got != 0.97 {
		t.Fatalf("scale = %v, want 0.97", got)
	}
	if line.PosYSpring == nil || line.PosYSpring.TargetPosition() != 120 {
		t.Fatal("position spring target should be snapped")
	}
}

func TestUpdateLineSpringsMovesTowardTargets(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.GetPosition().SetScaleX(0.97)
	line.GetPosition().SetScaleY(0.97)
	setLineSpringTargets(line, 120, 1, false)
	updateLineSprings(line, 16*time.Millisecond)

	if got := line.GetPosition().GetY(); got <= 0 || got >= 120 {
		t.Fatalf("y = %v, want between 0 and 120", got)
	}
	if got := line.GetPosition().GetScaleX(); got <= 0.97 || got >= 1 {
		t.Fatalf("scale = %v, want between 0.97 and 1", got)
	}
}

func TestComputeLinePosYSpringParamsTightensForFastLines(t *testing.T) {
	slow := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
			NewLine(900*time.Millisecond, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
	}
	fast := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
			NewLine(150*time.Millisecond, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
	}

	slowParams := computeLinePosYSpringParams(slow, 1, false)
	fastParams := computeLinePosYSpringParams(fast, 1, false)
	if fastParams.Stiffness <= slowParams.Stiffness {
		t.Fatalf("fast stiffness = %v, slow stiffness = %v; want fast > slow", fastParams.Stiffness, slowParams.Stiffness)
	}
}

func TestComputeLinePosYSpringParamsUsesPreviousFirstSyllableStart(t *testing.T) {
	prev := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	prev.Syllables = []*LineSyllable{{StartTime: 400 * time.Millisecond, EndTime: 900 * time.Millisecond}}
	current := NewLine(900*time.Millisecond, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	lyrics := &Lyrics{
		Lines:    []*Line{prev, current},
		Timeline: newTimelineState(),
	}

	params := computeLinePosYSpringParams(lyrics, 1, false)
	if params.Stiffness <= 170 || params.Stiffness >= 220 {
		t.Fatalf("stiffness = %v, want dynamic value based on 500ms interval", params.Stiffness)
	}
}

func TestComputeLinePosYSpringParamsMatchesRefDynamicFormula(t *testing.T) {
	prev := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	current := NewLine(450*time.Millisecond, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	lyrics := &Lyrics{
		Lines:    []*Line{prev, current},
		Timeline: newTimelineState(),
	}

	params := computeLinePosYSpringParams(lyrics, 1, false)
	intervalMs := 450.0
	ratio := 1 - (intervalMs-100)/(800-100)
	ratio = math.Pow(ratio, 0.2)
	wantStiffness := 170 + ratio*(220-170)
	wantDamping := math.Sqrt(wantStiffness) * 2.2
	if !nearlyEqual(params.Mass, 0.9) || !nearlyEqual(params.Stiffness, wantStiffness) || !nearlyEqual(params.Damping, wantDamping) {
		t.Fatalf("params = %+v, want mass 0.9 stiffness %v damping %v", params, wantStiffness, wantDamping)
	}
}

func TestComputeLinePosYSpringParamsUsesStableParamsForInterlude(t *testing.T) {
	lyrics := &Lyrics{
		Lines: []*Line{
			NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32),
			NewLine(150*time.Millisecond, 2*time.Second, false, false, "", nil, ft.FontRequest{}, 32),
		},
		Timeline: newTimelineState(),
	}

	params := computeLinePosYSpringParams(lyrics, 1, true)
	if params != defaultLinePosYSpringParams {
		t.Fatalf("interlude params = %+v, want default %+v", params, defaultLinePosYSpringParams)
	}
}
