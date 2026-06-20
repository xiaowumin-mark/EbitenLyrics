package bgrender

import (
	"image"
	"math"
	"testing"
)

func TestGaussianWeightsNormalize(t *testing.T) {
	weights := gaussianWeights(3, 1)
	sum := weights[0]
	for i := 1; i < len(weights); i++ {
		sum += weights[i] * 2
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("weight sum = %v, want 1", sum)
	}
}

func TestBlurNRGBAUsesPixelStrength(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 1))
	center := img.PixOffset(2, 0)
	img.Pix[center] = 255
	img.Pix[center+3] = 255
	blurNRGBA(img, 2.0)

	left := img.Pix[img.PixOffset(1, 0)]
	right := img.Pix[img.PixOffset(3, 0)]
	if left == 0 || right == 0 {
		t.Fatalf("neighbor pixels should receive blur, got left=%d right=%d", left, right)
	}
	if img.Pix[center] >= 255 {
		t.Fatalf("center pixel = %d, want blurred below 255", img.Pix[center])
	}
}
