package font

import (
	"github.com/flopp/go-findfont"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func GetAllFonts() []string {
	return findfont.List()
}

func FindFonts(name string) (string, error) {
	return findfont.Find(name)
}

func GetFace(font *text.GoTextFaceSource, size float64) text.Face {
	if font == nil || size <= 0 {
		return nil
	}
	return &text.GoTextFace{
		Source: font,
		Size:   size,
	}
}
