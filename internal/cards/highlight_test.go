package cards

import (
	"image"
	"image/color"
	"testing"
)

// TextInk and TextHighlight: one run of a card's text in a second colour, for a figure something
// else has changed. These assert pixels, like the rest of this package's tests.

func markedSpec(mark string) Spec {
	return Spec{
		Name:    "Slash",
		Form:    FormSlash,
		Cost:    2,
		Element: Fire,
		Text:    "Slashes for 4x DMG",
		Enabled: true,

		TextInk:       color.RGBA{R: 232, G: 106, B: 168, A: 255},
		TextHighlight: mark,
	}
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
	pink := color.RGBA{R: 232, G: 106, B: 168, A: 255}

	marked, err := Render(markedSpec("4x"), Hand, f)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := Render(markedSpec(""), Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	few, many := inkedPixels(marked, pink), inkedPixels(whole, pink)

	if few == 0 {
		t.Fatal("the marked run was not drawn in the second colour at all")
	}
	if few >= many {
		t.Errorf("marking one run coloured %d pixels and marking nothing coloured %d: "+
			"the highlight is not narrowing anything", few, many)
	}
}

func TestAMarkThatIsNotInTheTextChangesNothing(t *testing.T) {
	// A card whose wording moves on without its mark must render as a plain card rather than as a
	// blank or a panic. The mark is looked for and not found; the line is drawn in one colour.
	f := faces(t)

	missing, err := Render(markedSpec("nowhere"), Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	plain := markedSpec("")
	plain.TextInk = color.RGBA{}
	unmarked, err := Render(plain, Hand, f)
	if err != nil {
		t.Fatal(err)
	}

	if !sameImage(missing, unmarked) {
		t.Error("a mark that is not in the text changed how the card renders")
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
