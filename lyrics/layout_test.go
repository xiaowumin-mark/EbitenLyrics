package lyrics

import (
	ft "EbitenLyrics/font"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func testLayoutFace(tb testing.TB) text.Face {
	tb.Helper()
	manager := ft.NewFontManager(16)
	req := ft.DefaultRequest()
	face, err := manager.GetFaceForText(req, 48, "Alpha Bravo 想念你的夜晚")
	if err != nil {
		tb.Fatalf("get face failed: %v", err)
	}
	return face
}

func makeTestSyllable(text string, width float64) *LineSyllable {
	return &LineSyllable{
		Syllable: text,
		Elements: []*SyllableElement{
			{
				Text: text,
				SyllableImage: &SyllableImage{
					Text:   text,
					Width:  width,
					Height: 20,
				},
			},
		},
	}
}

func TestSplitCJKRunForLayoutAvoidsSingleRuneTail(t *testing.T) {
	parts := splitCJKRunForLayout("想念你的夜晚里")
	if strings.Join(parts, "") != "想念你的夜晚里" {
		t.Fatalf("unexpected recomposed text: %q", strings.Join(parts, ""))
	}
	for _, part := range parts {
		runes := []rune(part)
		if len(runes) < 2 || len(runes) > 4 {
			t.Fatalf("unexpected chunk %q with rune len %d", part, len(runes))
		}
	}
}

func TestAutoLayoutSyllableBalancesLinesAndTrimsBreakSpaces(t *testing.T) {
	face := testLayoutFace(t)
	layoutData := [][]*LineSyllable{
		{makeTestSyllable("Alpha", 40)},
		{makeTestSyllable(" ", 10)},
		{makeTestSyllable("Bravo", 40)},
		{makeTestSyllable(" ", 10)},
		{makeTestSyllable("go", 20)},
	}

	positions, height := AutoLayoutSyllable(layoutData, face, 110, 12, 1, text.AlignStart)
	if len(positions) != 5 {
		t.Fatalf("unexpected position count: %d", len(positions))
	}
	if height <= 0 {
		t.Fatalf("unexpected height: %f", height)
	}
	if positions[2].GetY() <= positions[0].GetY() {
		t.Fatalf("expected Bravo to wrap to the next line: first=%f bravo=%f", positions[0].GetY(), positions[2].GetY())
	}
	if positions[4].GetY() != positions[2].GetY() {
		t.Fatalf("expected trailing word to stay on Bravo line: bravo=%f go=%f", positions[2].GetY(), positions[4].GetY())
	}
	if positions[2].GetX() != 0 {
		t.Fatalf("expected wrapped line to start without leading space width, got x=%f", positions[2].GetX())
	}
}

func TestSplitLyricsIntoGroupsSmartKeepsEnglishSyllablesInWord(t *testing.T) {
	words := []string{"You ", "can't ", "spell ", "awesome ", "wi", "thout ", "me"}
	groups := splitLyricsIntoGroupsSmart(words, false)

	if len(groups) != 6 {
		t.Fatalf("unexpected group count: %d", len(groups))
	}
	if len(groups[4]) != 2 || groups[4][0] != 4 || groups[4][1] != 5 {
		t.Fatalf("expected wi/thout to stay in one word group, got %#v", groups[4])
	}
}

func TestAutoLayoutSyllableDoesNotBreakInsideGroupedWord(t *testing.T) {
	face := testLayoutFace(t)
	layoutData := [][]*LineSyllable{
		{makeTestSyllable("awesome", 50)},
		{makeTestSyllable(" ", 10)},
		{makeTestSyllable("wi", 15), makeTestSyllable("thout", 25)},
		{makeTestSyllable(" ", 10)},
		{makeTestSyllable("me", 20)},
	}

	positions, _ := AutoLayoutSyllable(layoutData, face, 70, 12, 1, text.AlignStart)
	if len(positions) != 6 {
		t.Fatalf("unexpected position count: %d", len(positions))
	}
	if positions[2].GetY() != positions[3].GetY() {
		t.Fatalf("expected grouped syllables to remain on the same line: wi=%f thout=%f", positions[2].GetY(), positions[3].GetY())
	}
}

func TestAutoLayoutSmartBalancesTranslationWrap(t *testing.T) {
	face := testLayoutFace(t)
	lines, height := AutoLayoutSmart("Alpha Bravo go", face, 200, 12, 1, text.AlignStart)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if height <= 0 {
		t.Fatalf("unexpected height: %f", height)
	}
	if strings.TrimSpace(lines[0].Text) != "Alpha" {
		t.Fatalf("unexpected first line: %q", lines[0].Text)
	}
	if strings.TrimSpace(lines[1].Text) != "Bravo go" {
		t.Fatalf("unexpected second line: %q", lines[1].Text)
	}
}
