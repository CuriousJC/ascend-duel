package cards

import (
	"image"
	"image/color"
	"testing"
)

// Spec.Highlights: runs of a card's text set in their own colour. These assert pixels, like the
// rest of this package's tests.

var (
	pinkInk   = color.RGBA{R: 232, G: 106, B: 168, A: 255}
	fireInk   = BorderOf(Fire)
	arcaneInk = BorderOf(Arcane)
)

func markedSpec(runs ...TextRun) Spec {
	s := Spec{
		Name:    "Slice",
		Form:    FormSlash,
		Cost:    2,
		Element: Fire,
		Text:    "Slashes for 4x DMG",
		Enabled: true,
	}
	copy(s.Highlights[:], runs)
	return s
}

// inkedPixels counts how many pixels of the rendered card are near enough to a colour to have been
// drawn in it. **Near enough**, because text is antialiased: an exact match would count only the
// solid core of a glyph, which is a handful of pixels at 18pt and would make the test flap.
func inkedPixels(img *image.RGBA, want color.RGBA) int {
	near := func(a, b uint8) bool {
		if a > b {
			return int(a)-int(b) < 40
		}
		return int(b)-int(a) < 40
	}

	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A > 200 && near(c.R, want.R) && near(c.G, want.G) && near(c.B, want.B) {
				n++
			}
		}
	}
	return n
}

func TestOnlyTheMarkedRunTakesTheSecondColour(t *testing.T) {
	// **The figure, not the sentence** *(owner's call, 2026-08-21)*. Colouring the whole line says a
	// ring changed the card; colouring "4x" says it changed the number, which is what happened.
	f := faces(t)

	marked, err := Render(markedSpec(TextRun{Run: "4x", Ink: pinkInk}), Hand, f)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := Render(markedSpec(
		TextRun{Run: "Slashes", Ink: pinkInk},
		TextRun{Run: "for", Ink: pinkInk},
		TextRun{Run: "4x", Ink: pinkInk},
		TextRun{Run: "DMG", Ink: pinkInk},
	), Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	few, many := inkedPixels(marked, pinkInk), inkedPixels(whole, pinkInk)

	if few == 0 {
		t.Fatal("the marked run was not drawn in the second colour at all")
	}
	if few >= many {
		t.Errorf("marking one run coloured %d pixels and marking every word coloured %d: "+
			"the highlight is not narrowing anything", few, many)
	}
}

func TestAMarkThatIsNotInTheTextChangesNothing(t *testing.T) {
	// A card whose wording moves on without its mark must render as a plain card rather than as a
	// blank or a panic. The mark is looked for and not found; the line is drawn in one colour.
	f := faces(t)

	missing, err := Render(markedSpec(TextRun{Run: "nowhere", Ink: pinkInk}), Hand, f)
	if err != nil {
		t.Fatal(err)
	}
	unmarked, err := Render(markedSpec(), Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	if !sameImage(missing, unmarked) {
		t.Error("a mark that is not in the text changed how the card renders")
	}
}

func TestTwoRunsTakeTwoColoursOnOneCard(t *testing.T) {
	// The case the single-run version could not draw: a ring naming an element and a status that
	// belongs to a different one. Both have to appear, in their own colours, on one face.
	f := faces(t)

	spec := markedSpec(
		TextRun{Run: "Slashes", Ink: fireInk},
		TextRun{Run: "DMG", Ink: arcaneInk},
	)
	img, err := Render(spec, Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	if n := inkedPixels(img, fireInk); n == 0 {
		t.Error("the first run was not drawn in its own colour")
	}
	if n := inkedPixels(img, arcaneInk); n == 0 {
		t.Error("the second run was not drawn in its own colour")
	}
}

// splitRuns is where the whole-word rule and the first-claim rule live, and both are cheaper to
// assert on strings than on pixels.

func TestARunOnlyMatchesAWholeWord(t *testing.T) {
	// ICE is inside SLICE and PRICE. A substring match would paint three letters of a card's name
	// in the ice blue, which reads as a rendering fault rather than as a colour meaning something.
	segs := SplitRuns("CARD BECOMES SLICE", []TextRun{{Run: "ICE", Ink: fireInk}})

	if len(segs) != 1 || segs[0].Ink.A != 0 {
		t.Errorf("ICE matched inside SLICE: %v", segs)
	}
}

func TestEveryOccurrenceOfARunIsColoured(t *testing.T) {
	// One entry covers a word a sentence says twice — "apply BURNING status … BURNING enemies" —
	// so a repeat does not cost a second seat in a fixed array.
	segs := SplitRuns("BURNING and BURNING", []TextRun{{Run: "BURNING", Ink: fireInk}})

	n := 0
	for _, s := range segs {
		if s.Ink == fireInk {
			n++
		}
	}
	if n != 2 {
		t.Errorf("coloured %d occurrences of BURNING, want 2: %v", n, segs)
	}
}

func TestTheFirstRunToClaimAPositionKeepsIt(t *testing.T) {
	// The caller sorts by length, so BURNING is offered before BURN. If the shorter one could take
	// the front of the longer, "BURNING" would draw as a coloured BURN and a default-ink ING.
	segs := SplitRuns("BURNING", []TextRun{
		{Run: "BURNING", Ink: fireInk},
		{Run: "BURN", Ink: arcaneInk},
	})

	if len(segs) != 1 || segs[0].Text != "BURNING" || segs[0].Ink != fireInk {
		t.Errorf("the longer run did not keep the word: %v", segs)
	}
}

func TestSplittingCoversTheWholeLine(t *testing.T) {
	// Segments are drawn one after another from a single starting x, so anything dropped between
	// them would shift the rest of the line left rather than leave a gap.
	line := "Fire attacks BURN the target."
	segs := SplitRuns(line, []TextRun{
		{Run: "Fire", Ink: fireInk},
		{Run: "BURN", Ink: arcaneInk},
	})

	joined := ""
	for _, s := range segs {
		joined += s.Text
	}
	if joined != line {
		t.Errorf("the segments join to %q, want %q", joined, line)
	}
}

func sameImage(a, b *image.RGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
