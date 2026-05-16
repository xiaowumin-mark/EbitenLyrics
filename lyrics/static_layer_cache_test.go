package lyrics

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ft "github.com/xiaowumin-mark/EbitenLyrics/font"
)

func TestStaticLayerSignatureReportsRebuildOnlyWhenBitmapDirty(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true
	line.Image = ebiten.NewImage(10, 10)
	line.GetPosition().SetAlpha(1)
	line.imageDirty = false
	line.setStatus(LineStatusPreviewStatic)
	line.imageDirty = false

	lyrics := &Lyrics{Lines: []*Line{line}, renderIndex: []int{0}}
	signature, hasStatic, needsRebuild := lyrics.StaticLayerSignature()
	if !hasStatic || needsRebuild || signature == 0 {
		t.Fatalf("static signature = %d has=%v rebuild=%v, want stable ready layer", signature, hasStatic, needsRebuild)
	}

	line.imageDirty = true
	_, hasStatic, needsRebuild = lyrics.StaticLayerSignature()
	if !hasStatic || !needsRebuild {
		t.Fatalf("dirty static layer has=%v rebuild=%v, want rebuild", hasStatic, needsRebuild)
	}
}

func TestStaticLayerSignatureChangesWhenTransformChanges(t *testing.T) {
	line := NewLine(0, time.Second, false, false, "", nil, ft.FontRequest{}, 32)
	line.isShow = true
	line.Image = ebiten.NewImage(10, 10)
	line.GetPosition().SetAlpha(1)
	line.imageDirty = false
	line.setStatus(LineStatusPreviewStatic)
	line.imageDirty = false

	lyrics := &Lyrics{Lines: []*Line{line}, renderIndex: []int{0}}
	first, _, _ := lyrics.StaticLayerSignature()
	line.GetPosition().SetY(12)
	second, _, _ := lyrics.StaticLayerSignature()
	if first == second {
		t.Fatalf("static signature did not change after transform update: %d", first)
	}
}
