package font

import (
	"errors"

	gotextfont "github.com/go-text/typesetting/font"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (m *FontManager) FindFaceForRune(r rune, chain []string) (text.Face, error) {
	return m.findFaceForRuneWithSize(r, chain, 1, FontRequest{
		Families: chain,
		Weight:   WeightRegular,
	})
}

func (m *FontManager) findFaceForRuneWithSize(r rune, chain []string, size float64, req FontRequest) (text.Face, error) {
	if size <= 0 {
		size = 1
	}
	req.Families = append([]string{}, chain...)
	resolved, err := m.ResolveChain(req)
	if err != nil {
		return nil, err
	}
	fonts, err := m.loadFontsForContent(resolved, string(r))
	if err != nil {
		return nil, err
	}
	for _, font := range fonts {
		if font != nil && sourceHasRune(font.Source, r) {
			return &text.GoTextFace{Source: font.Source, Size: size}, nil
		}
	}
	return nil, errors.New("no face in fallback chain covers rune")
}

func sourceHasRune(source *text.GoTextFaceSource, r rune) bool {
	if source == nil {
		return false
	}
	internal := source.UnsafeInternal()
	face, ok := internal.(*gotextfont.Face)
	if !ok || face == nil {
		return false
	}
	_, ok = face.NominalGlyph(r)
	return ok
}
