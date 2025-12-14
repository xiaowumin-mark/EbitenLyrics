package bgrender

import (
	"image/png"
	"os"
	"testing"
)

func Test_ExtractPalette(t *testing.T) {
	// 读取"E:\projects\visual-lyric\music\Opalite.png"
	f, err := os.Open("E:/projects/visual-lyric/music/Opalite.png")
	if err != nil {
		t.Error(err)
	}
	img, err := png.Decode(f)
	if err != nil {
		t.Error(err)
	}
	palette := ExtractPalette(img, 5)
	for _, p := range palette {
		t.Logf("%d,%d,%d,%f", p.R, p.G, p.B, p.Weight)
	}
}
