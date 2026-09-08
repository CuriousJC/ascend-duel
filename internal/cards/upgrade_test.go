package cards

import (
	"image"
	"image/color"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// upgradeSpec is a card in one upgrade, for comparing two renderings of the same card.
func upgradeSpec(u systems.Upgrade, mode TintMode) Spec {
	return Spec{
		Name:        "Lunge",
		Form:        FormStab,
		Cost:        3,
		Element:     Fire,
		Upgrade:     u,
		UpgradeTint: mode,
		Text:        "2x DMG",
		Enabled:     true,
	}
}

// render is one card, or a fatal test failure.
func renderOrFail(t *testing.T, s Spec) *image.RGBA {
	t.Helper()
	img, err := Render(s, Hand, faces(t))
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	return img
}

// **An upgrade takes the whole left column or none of it.** The form mark and the cost ticks are
// one statement about the card, and a rainbow mark over fire-red ticks would say two different
// things in the one place the card says one. Both halves are checked, because wiring one and
// forgetting the other is the failure this is written against.
func TestAnUpgradeTakesTheWholeLeftColumn(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone, DefaultTintMode))
	wild := renderOrFail(t, upgradeSpec(systems.UpgradeWild, DefaultTintMode))

	st := Hand
	mark := image.Rect(st.GlyphInset, st.FormTop,
		st.GlyphInset+st.FormSize, st.FormTop+st.FormSize)
	stack, ok := dashStack(upgradeSpec(systems.UpgradeWild, DefaultTintMode), st)
	if !ok {
		t.Fatal("a three-cost card has no tick stack")
	}

	for _, tc := range []struct {
		what string
		box  image.Rectangle
	}{
		{"the form mark", mark},
		{"the cost ticks", stack},
	} {
		if same(plain, wild, tc.box) {
			t.Errorf("%s is identical on an upgraded card and a plain one — the upgrade did not "+
				"reach it", tc.what)
		}
	}
}

// **Everything outside the left column is untouched.** An upgrade says one thing about the card
// and the rest of the face is not its business — the name, the effect text and the border all
// belong to the card, not to what has been done to it.
func TestAnUpgradeLeavesTheRestOfTheCardAlone(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone, DefaultTintMode))
	wild := renderOrFail(t, upgradeSpec(systems.UpgradeWild, DefaultTintMode))

	st := Hand
	// **The text band and the right border, not simply "right of TextColumnLeft".** The form
	// mark's 32px box starts at GlyphInset and runs past TextColumnLeft: the mark sits *above*
	// the text rather than beside it, so a rectangle taken from the column's left edge to the
	// card's right edge contains the mark itself and would fail on the change it is checking is
	// contained.
	for _, tc := range []struct {
		what string
		box  image.Rectangle
	}{
		{"the text band", image.Rect(st.TextColumnLeft, st.TextBandTop, st.Width, st.TextBandBottom)},
		{"the right border", image.Rect(st.Width-st.BorderWidth, 0, st.Width, st.Height)},
		{"everything below the left column", image.Rect(0, st.Height/2, st.Width, st.Height)},
	} {
		if !same(plain, wild, tc.box) {
			t.Errorf("an upgrade changed %s", tc.what)
		}
	}
}

// **The three modes are three different pictures.** They exist so the owner can choose between
// them off tools/upgradesheet, and two that rendered the same would be a choice that is not one.
func TestTheThreeTintModesDiffer(t *testing.T) {
	st := Hand
	mark := image.Rect(st.GlyphInset, st.FormTop,
		st.GlyphInset+st.FormSize, st.FormTop+st.FormSize)

	seen := map[TintMode]*image.RGBA{}
	for _, m := range TintModes() {
		seen[m] = renderOrFail(t, upgradeSpec(systems.UpgradeWild, m))
	}
	for i, a := range TintModes() {
		for _, b := range TintModes()[i+1:] {
			if same(seen[a], seen[b], mark) {
				t.Errorf("%s and %s draw the same mark", a, b)
			}
		}
	}
}

// **The ticks and the mark still share one state.** That is the rule
// TestTheTicksAndTheBorderShareOneState holds for an ordinary card, and an upgrade drawing its own
// ticks is exactly the place a second state switch gets written by accident.
func TestAnUpgradedColumnStillStates(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeWild, DefaultTintMode)
	dim := rest
	dim.Enabled = false

	a, b := renderOrFail(t, rest), renderOrFail(t, dim)
	stack, ok := dashStack(rest, Hand)
	if !ok {
		t.Fatal("a three-cost card has no tick stack")
	}
	if same(a, b, stack) {
		t.Error("an upgraded card's ticks look the same afforded and unafforded")
	}
}

// **A cost of zero draws no ticks and does not panic.** Ring and worm cards carry no cost, and an
// upgraded one is not yet possible but costs nothing to be safe about — the flat path has always
// had this guard and the upgraded one has to have it too.
func TestAnUpgradedCardWithNoCostDrawsNoTicks(t *testing.T) {
	s := upgradeSpec(systems.UpgradeWild, DefaultTintMode)
	s.Cost = 0
	if _, ok := dashStack(s, Hand); ok {
		t.Fatal("a zero-cost card reports a tick stack")
	}
	renderOrFail(t, s) // must not panic
}

// same reports whether two renderings agree over a rectangle.
func same(a, b *image.RGBA, box image.Rectangle) bool {
	box = box.Intersect(a.Bounds()).Intersect(b.Bounds())
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// **Every upgrade in the vocabulary has an ink.** One without would draw as an ordinary card and
// be invisible, which is the exact failure the whole idea exists to fix — so it fails here rather
// than shipping a parasite that appears to do nothing.
func TestEveryUpgradeHasAnInk(t *testing.T) {
	for _, u := range systems.Upgrades() {
		ink := systems.UpgradeInk(u)
		if ink == nil {
			t.Fatalf("upgrade %q has no ink", u)
		}
		b := ink.Bounds()
		if b.Dx() != systems.UpgradeInkSize || b.Dy() != systems.UpgradeInkSize {
			t.Errorf("upgrade %q's ink is %dx%d, want %d square",
				u, b.Dx(), b.Dy(), systems.UpgradeInkSize)
		}
		if flat(ink) {
			t.Errorf("upgrade %q's ink is one colour, so the column it paints says nothing an "+
				"element could not have said", u)
		}
	}
}

// **UpgradeNone has no ink, which is what a caller checks to fall back to the element.**
func TestNoUpgradeHasNoInk(t *testing.T) {
	if systems.UpgradeInk(systems.UpgradeNone) != nil {
		t.Error("UpgradeNone returned an ink")
	}
}

// flat reports whether every opaque pixel of an ink is the same colour.
func flat(ink *image.RGBA) bool {
	b := ink.Bounds()
	var first color.RGBA
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := ink.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			if !found {
				first, found = c, true
				continue
			}
			if c != first {
				return false
			}
		}
	}
	return true
}
