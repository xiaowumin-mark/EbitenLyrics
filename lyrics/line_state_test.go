package lyrics

import (
	"testing"
	"time"

	"github.com/xiaowumin-mark/EbitenLyrics/anim"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestLineStatusHelpers(t *testing.T) {
	tests := []struct {
		name             string
		status           LineStatus
		usesPreview      bool
		requiresRealtime bool
		canStartExit     bool
	}{
		{name: "hidden", status: LineStatusHidden},
		{name: "preview static", status: LineStatusPreviewStatic, usesPreview: true},
		{name: "preview scrolling", status: LineStatusPreviewScrolling, usesPreview: true},
		{name: "active enter", status: LineStatusActiveEnter, requiresRealtime: true, canStartExit: true},
		{name: "active playing", status: LineStatusActivePlaying, requiresRealtime: true, canStartExit: true},
		{name: "active exit", status: LineStatusActiveExit, requiresRealtime: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.UsesPreviewBitmap(); got != tt.usesPreview {
				t.Fatalf("UsesPreviewBitmap() = %v, want %v", got, tt.usesPreview)
			}
			if got := tt.status.RequiresRealtimeRender(); got != tt.requiresRealtime {
				t.Fatalf("RequiresRealtimeRender() = %v, want %v", got, tt.requiresRealtime)
			}
			if got := tt.status.CanStartExit(); got != tt.canStartExit {
				t.Fatalf("CanStartExit() = %v, want %v", got, tt.canStartExit)
			}
		})
	}
}

func TestSetStatusMarksPreviewBitmapDirty(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.imageDirty = false

	line.setStatus(LineStatusActivePlaying)
	if line.imageDirty {
		t.Fatal("active status should not mark preview bitmap dirty")
	}

	line.setStatus(LineStatusPreviewStatic)
	if !line.imageDirty {
		t.Fatal("entering preview bitmap mode should mark the image dirty")
	}

	line.imageDirty = false
	line.setStatus(LineStatusPreviewScrolling)
	if line.imageDirty {
		t.Fatal("switching between preview states should reuse the same bitmap")
	}
}

func TestSyncPreviewStateTracksScrollAnimation(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.setStatus(LineStatusPreviewStatic)
	line.imageDirty = false
	line.ScrollAnimate = anim.NewTween("scroll", time.Millisecond, 0, 1, 0, 1, anim.Linear, func(float64) {}, func() {})

	lineAnimationLayer.syncPreviewState(line)
	if line.Status != LineStatusPreviewScrolling {
		t.Fatalf("status = %v, want preview scrolling", line.Status)
	}
	if line.imageDirty {
		t.Fatal("preview scroll transitions should not invalidate the cached bitmap")
	}

	line.ScrollAnimate = nil
	lineAnimationLayer.syncPreviewState(line)
	if line.Status != LineStatusPreviewStatic {
		t.Fatalf("status = %v, want preview static", line.Status)
	}
}

func TestScrollDelayForIndexUsesLeadCompensationModel(t *testing.T) {
	base := scrollLeadTime()
	if got := scrollDelayForIndex(5, 5); got != base {
		t.Fatalf("anchor delay = %v, want %v", got, base)
	}
	if got := scrollDelayForIndex(5, 4); got != base {
		t.Fatalf("upper delay = %v, want %v", got, base)
	}
	if got := scrollDelayForIndex(5, 6); got != base+scrollDelayStep {
		t.Fatalf("lower adjacent delay = %v, want %v", got, base+scrollDelayStep)
	}
	if got := scrollDelayForIndex(5, 7); got != base+2*scrollDelayStep {
		t.Fatalf("lower second delay = %v, want %v", got, base+2*scrollDelayStep)
	}
}

func TestPredictedScrollAnchorIndexUsesLeadTime(t *testing.T) {
	lines := []*Line{
		NewLine(0, 500*time.Millisecond, false, false, "", nil, ft.FontRequest{}, 32),
		NewLine(time.Second, 1500*time.Millisecond, false, false, "", nil, ft.FontRequest{}, 32),
	}

	beforeLeadBoundary := time.Second - scrollLeadTime() - time.Millisecond
	if got := predictedScrollAnchorIndex(lines, beforeLeadBoundary); got != 0 {
		t.Fatalf("anchor before lead boundary = %d, want 0", got)
	}

	atLeadBoundary := time.Second - scrollLeadTime()
	if got := predictedScrollAnchorIndex(lines, atLeadBoundary); got != 1 {
		t.Fatalf("anchor at lead boundary = %d, want 1", got)
	}
}

func TestEnsureScrollAnimationRetargetDoesNotReapplyDelay(t *testing.T) {
	manager := anim.NewManager(false)
	lyrics := &Lyrics{AnimateManager: manager}
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.GetPosition().SetY(0)

	const delay = 40 * time.Millisecond
	lineAnimationLayer.ensureScrollAnimation(line, lyrics, 100, delay, 100*time.Millisecond, anim.Linear)
	manager.Update(50 * time.Millisecond)

	firstY := line.GetPosition().GetY()
	if firstY <= 0 {
		t.Fatalf("initial scroll did not advance, y = %v", firstY)
	}

	lineAnimationLayer.ensureScrollAnimation(line, lyrics, 200, delay, 100*time.Millisecond, anim.Linear)
	manager.Update(20 * time.Millisecond)

	if got := line.GetPosition().GetY(); got <= firstY {
		t.Fatalf("retargeted scroll reused delay and stalled, y = %v, previous = %v", got, firstY)
	}
}

func TestUpdateFinalLayoutStateArmsWhenBackgroundStillVisible(t *testing.T) {
	mainLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	bgLine := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 24)
	bgLine.isShow = true
	bgLine.GetPosition().SetAlpha(1)
	mainLine.BackgroundLines = []*Line{bgLine}

	lyrics := &Lyrics{
		Lines:     []*Line{mainLine},
		nowLyrics: []int{},
	}

	changed := lineAnimationLayer.updateFinalLayoutState(lyrics, true, false)
	if changed {
		t.Fatal("changed should remain false while background line still occupies space")
	}
	if !lyrics.finalLayoutPending {
		t.Fatal("final layout should stay pending until background line exits")
	}
}

func TestUpdateFinalLayoutStateTriggersOnceBackgroundGone(t *testing.T) {
	mainLine := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	bgLine := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 24)
	bgLine.isShow = true
	bgLine.GetPosition().SetAlpha(0)
	mainLine.BackgroundLines = []*Line{bgLine}

	lyrics := &Lyrics{
		Lines:              []*Line{mainLine},
		nowLyrics:          []int{},
		finalLayoutPending: true,
	}

	changed := lineAnimationLayer.updateFinalLayoutState(lyrics, true, false)
	if !changed {
		t.Fatal("changed should become true to force one final relayout")
	}
	if lyrics.finalLayoutPending {
		t.Fatal("final layout pending should be cleared after relayout trigger")
	}
}

func TestNormalizeLineBackgroundSkipsInvisibleExitTweens(t *testing.T) {
	manager := anim.NewManager(false)
	lyrics := &Lyrics{AnimateManager: manager}
	line := NewLine(0, time.Second, false, true, "", nil, ft.FontRequest{}, 32)
	element := &SyllableElement{Position: NewPosition(0, 0, 10, 10)}
	element.Position.SetTranslateY(-6)
	line.OuterSyllableElements = []*SyllableElement{element}

	lineAnimationLayer.NormalizeLine(line, lyrics)

	if element.UpAnimate != nil {
		t.Fatal("background exit should not create per-element lift reset tween")
	}
	if got := element.GetPosition().GetTranslateY(); got != 0 {
		t.Fatalf("background exit translateY = %v, want 0", got)
	}
}

func TestNormalizeLineSkipsZeroDeltaExitTweens(t *testing.T) {
	manager := anim.NewManager(false)
	lyrics := &Lyrics{AnimateManager: manager}
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	element := &SyllableElement{Position: NewPosition(0, 0, 10, 10)}
	line.OuterSyllableElements = []*SyllableElement{element}

	lineAnimationLayer.NormalizeLine(line, lyrics)

	if line.ScaleAnimate != nil {
		t.Fatal("exit should not create a scale tween when scale is already settled")
	}
	if element.UpAnimate != nil {
		t.Fatal("exit should not create a per-element reset tween when translateY is already settled")
	}
}

func TestNormalizeLineSettlesActiveHighlightInsteadOfResettingImmediately(t *testing.T) {
	manager := anim.NewManager(false)
	lyrics := &Lyrics{AnimateManager: manager}
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	element := &SyllableElement{
		Position:           NewPosition(0, 0, 10, 10),
		BackgroundBlurText: &TextShadow{Alpha: 0.75},
	}
	element.Position.SetTranslateX(14)
	element.Position.SetScaleX(1.18)
	element.Position.SetScaleY(1.18)
	element.HighlightAnimate = anim.NewKeyframeAnimation("active-highlight", time.Second, 0, 1, true, nil, nil, nil)
	line.OuterSyllableElements = []*SyllableElement{element}

	lineAnimationLayer.NormalizeLine(line, lyrics)

	if element.HighlightAnimate == nil {
		t.Fatal("active highlight should transition into a settle animation on exit")
	}
	if got := element.GetPosition().GetTranslateX(); got != 14 {
		t.Fatalf("translateX reset too early: got %v, want 14", got)
	}

	manager.Update(50 * time.Millisecond)

	if got := element.GetPosition().GetTranslateX(); got >= 14 || got <= 0 {
		t.Fatalf("translateX = %v, want intermediate value", got)
	}
	if got := element.GetPosition().GetScaleX(); got >= 1.18 || got <= 1 {
		t.Fatalf("scaleX = %v, want intermediate value", got)
	}
	if element.BackgroundBlurText == nil || element.BackgroundBlurText.Alpha >= 0.75 {
		t.Fatal("blur alpha should fade during highlight settle")
	}

	for i := 0; i < 8; i++ {
		manager.Update(50 * time.Millisecond)
	}

	if element.HighlightAnimate != nil {
		t.Fatal("highlight settle animation should finish")
	}
	if got := element.GetPosition().GetTranslateX(); got != 0 {
		t.Fatalf("translateX = %v, want 0", got)
	}
	if got := element.GetPosition().GetScaleX(); got != 1 {
		t.Fatalf("scaleX = %v, want 1", got)
	}
	if element.BackgroundBlurText != nil {
		t.Fatal("blur shadow should be disposed after highlight settle")
	}
}

func TestStaticLayerClassification(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true

	line.setStatus(LineStatusPreviewStatic)
	if !line.canUseStaticLayer() {
		t.Fatal("preview static line should use static layer")
	}
	if line.shouldDrawDynamically() {
		t.Fatal("preview static line should not be drawn dynamically")
	}

	line.setStatus(LineStatusPreviewScrolling)
	if line.canUseStaticLayer() {
		t.Fatal("preview scrolling line should not use static layer")
	}
	if !line.shouldDrawDynamically() {
		t.Fatal("preview scrolling line should be drawn dynamically")
	}

	line.setStatus(LineStatusActivePlaying)
	if line.canUseStaticLayer() {
		t.Fatal("active line should not use static layer")
	}
	if !line.shouldDrawDynamically() {
		t.Fatal("active line should be drawn dynamically")
	}
}
