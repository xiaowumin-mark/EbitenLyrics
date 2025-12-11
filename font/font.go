package font

import (
	"log"

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
	if font == nil {
		log.Fatalln("Font is nil")
	}
	return &text.GoTextFace{
		Source: font,
		Size:   size,
	}
}
