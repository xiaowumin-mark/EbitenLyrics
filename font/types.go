package font

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Weight int

const (
	WeightThin       Weight = 100
	WeightExtraLight Weight = 200
	WeightLight      Weight = 300
	WeightRegular    Weight = 400
	WeightMedium     Weight = 500
	WeightSemiBold   Weight = 600
	WeightBold       Weight = 700
	WeightExtraBold  Weight = 800
	WeightBlack      Weight = 900
)

type FontRequest struct {
	Families []string
	Weight   Weight
	Italic   bool
}

func (r FontRequest) Normalized() FontRequest {
	r.Weight = normalizeWeight(r.Weight)
	r.Families = normalizeFamilies(r.Families)
	if len(r.Families) == 0 {
		r.Families = DefaultFamilies()
	}
	return r
}

func (r FontRequest) CacheKey() string {
	r = r.Normalized()
	parts := make([]string, 0, len(r.Families))
	for _, family := range r.Families {
		parts = append(parts, normalizeName(family))
	}
	return fmt.Sprintf("%s|%d|%t", strings.Join(parts, ","), r.Weight, r.Italic)
}

type ResolvedFontChain struct {
	Request   FontRequest
	Families  []string
	Primary   *ResolvedFont
	Fallbacks []*ResolvedFont
	Sources   []*text.GoTextFaceSource
}

type ResolvedFont struct {
	Path            string
	CollectionIndex int
	Family          string
	Style           string
	Weight          Weight
	Italic          bool
	Source          *text.GoTextFaceSource
}

func (r *ResolvedFont) entryKey() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s#%d", filepath.Clean(r.Path), r.CollectionIndex)
}
