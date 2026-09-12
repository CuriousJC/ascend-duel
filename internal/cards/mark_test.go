package cards

import (
	"image"
	"testing"
)

// **Marks are a set and the set has to stay one.** A card can be several things at once — broken
// *and* pointed at — and the bug a bitmask exists to prevent is the caller having to rank them.
func TestMarksCompose(t *testing.T) {
	both := MarkShattered | MarkHighlit

	if !both.Has(MarkShattered) || !both.Has(MarkHighlit) {
		t.Error("a card carrying two marks does not report both")
	}
	if MarkNone.Has(MarkShattered) || MarkNone.Has(MarkHighlit) {
		t.Error("an unmarked card reports a mark")
	}
	if MarkShattered.Has(MarkHighlit) || MarkHighlit.Has(MarkShattered) {
		t.Error("two marks share a bit, so one card cannot carry them both")
	}
}

// **A mark has to change the picture, and a set of them has to change it differently from either
// one alone.** The rasterisers are pixel work with no other check on them: a wash that landed
// outside the rounded silhouette, or a compose order that let one mark overwrite the other
// entirely, would both look like nothing being wrong.
func TestEveryMarkChangesTheFace(t *testing.T) {
	f := faces(t)
	base := Spec{Name: "Bash", Form: FormSlash, Cost: 2, Element: Fire, Text: "Slashes for 2x DMG"}

	plain := renderMark(t, base, f)
	seen := map[string]bool{"plain": true}

	for _, c := range []struct {
		name string
		mark Mark
	}{
		{"shattered", MarkShattered},
		{"highlit", MarkHighlit},
		{"both", MarkShattered | MarkHighlit},
	} {
		spec := base
		spec.Mark = c.mark
		got := renderMark(t, spec, f)

		if samePixels(plain, got) {
			t.Errorf("%s draws the same face as an unmarked card", c.name)
		}
		key := string(got.Pix)
		if seen[key] {
			t.Errorf("%s draws the same face as another mark", c.name)
		}
		seen[key] = true

		// The corners are transparent and must stay so, or a marked card is a rectangle.
		if got.Pix[3] != 0 {
			t.Errorf("%s filled in the card's top-left corner", c.name)
		}
	}
}

// renderMark draws one spec at hand size, failing the test rather than returning an error.
func renderMark(t *testing.T, s Spec, f *Faces) *image.RGBA {
	t.Helper()
	img, err := Render(s, Hand, f)
	if err != nil {
		t.Fatalf("rendering %q: %v", s.Name, err)
	}
	return img
}

// samePixels is byte equality over two faces of the same size.
func samePixels(a, b *image.RGBA) bool {
	if len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
