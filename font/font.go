package font

import (
	"EbitenLyrics/lp"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	defaultManagerOnce sync.Once
	defaultManagerInst *FontManager
)

func DefaultManager() *FontManager {
	defaultManagerOnce.Do(func() {
		defaultManagerInst = NewFontManager(16)
	})
	return defaultManagerInst
}

func FaceFromSource(source *text.GoTextFaceSource, size float64) text.Face {
	if source == nil || size <= 0 {
		return nil
	}
	return &text.GoTextFace{
		Source: source,
		Size:   lp.LP(size),
	}
}

func RegisterFallback(userRules map[string][]string) {
	DefaultManager().RegisterFallback(userRules)
}

func RegisterCustomFontPath(name, path string) error {
	return DefaultManager().RegisterCustomFontPath(name, path)
}

func GetFace(req FontRequest, size float64) (text.Face, error) {
	return DefaultManager().GetFace(req, size)
}

func GetFaceForText(req FontRequest, size float64, content string) (text.Face, error) {
	return DefaultManager().GetFaceForText(req, size, content)
}

func FindFaceForRune(r rune, chain []string) (text.Face, error) {
	return DefaultManager().FindFaceForRune(r, chain)
}

func AvailableFamilies() []string {
	return DefaultManager().AvailableFamilies()
}
