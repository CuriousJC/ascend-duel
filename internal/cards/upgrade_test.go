package cards

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// upgradeSpec is a card in one upgrade, drawn in the style the game draws, for comparing two
// renderings of the same card.
func upgradeSpec(u systems.Upgrade) Spec { return styledSpec(u, DefaultUpgradeStyle) }

// styledSpec is upgradeSpec in a named style, for the tests that are about where the ink lands.
func styledSpec(u systems.Upgrade, style UpgradeStyle) Spec {
	return Spec{
		Name:         "Lunge",
		Form:         FormStab,
		Cost:         3,
		Element:      Fire,
		Upgrade:      u,
		UpgradeStyle: style,
		Text:         "2x DMG",
		Enabled:      true,
	}
}

// theFace and theBorder are the two regions the styles divide the card into, as rectangles a
// pixel comparison can be taken over.
//
// **Not the geometry the renderer uses**, deliberately: these are a rectangle well inside the
// border and a strip of the top edge, so a test that passes is one where the difference is
// unmistakable rather than one where it is a rounding away from the boundary.
func theFace() image.Rectangle {
	st := Hand
	in := st.BorderWidth + st.CornerRadius
	return image.Rect(in, in, st.Width-in, st.Height-in)
}

func theBorder() image.Rectangle {
	st := Hand
	return image.Rect(st.Width/3, 0, 2*st.Width/3, st.BorderWidth)
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

// **Each style paints exactly the region it names.** They are a review knob and the owner picks one
// by looking — but a style that reached somewhere it did not claim to would make that comparison a
// comparison of two bugs. This is the table that says what each one promises.
func TestEachStylePaintsTheRegionItNames(t *testing.T) {
	for _, tc := range []struct {
		style        UpgradeStyle
		face, border bool
		what         string
	}{
		{UpgradeBorder, false, true, "the border and nothing else"},
		{UpgradeWash, true, true, "the whole card, border included"},
		{UpgradeWashFace, true, false, "the face, leaving the border alone"},
	} {
		plain := renderOrFail(t, styledSpec(systems.UpgradeNone, tc.style))
		gold := renderOrFail(t, styledSpec(systems.UpgradeGolden, tc.style))

		if changed := !same(plain, gold, theFace()); changed != tc.face {
			t.Errorf("%s: the face changed=%v, and this style paints %s", tc.style, changed, tc.what)
		}
		if changed := !same(plain, gold, theBorder()); changed != tc.border {
			t.Errorf("%s: the border changed=%v, and this style paints %s", tc.style, changed, tc.what)
		}
	}
}

// **The wash reaches every part of the face it covers.** Reaching some of the card and not the rest
// is exactly what the left-column mechanism this replaced did, so the parts are named rather than
// the whole being compared in one go.
func TestTheWashReachesTheWholeFace(t *testing.T) {
	plain := renderOrFail(t, styledSpec(systems.UpgradeNone, UpgradeWash))
	gold := renderOrFail(t, styledSpec(systems.UpgradeGolden, UpgradeWash))

	st := Hand
	for _, tc := range []struct {
		what string
		box  image.Rectangle
	}{
		{"the left column", image.Rect(st.GlyphInset, st.FormTop,
			st.GlyphInset+st.FormSize, st.FormTop+st.FormSize)},
		{"the text band", image.Rect(st.TextColumnLeft, st.TextBandTop, st.Width, st.TextBandBottom)},
		{"the right border", image.Rect(st.Width-st.BorderWidth, 0, st.Width, st.Height)},
		{"the bottom half", image.Rect(0, st.Height/2, st.Width, st.Height)},
	} {
		if same(plain, gold, tc.box) {
			t.Errorf("%s is identical on an upgraded card and a plain one — the wash did not "+
				"reach it", tc.what)
		}
	}
}

// **Every style's name round-trips through the sheet's file names**, which is the only place one is
// written down. Two styles spelling themselves the same would overwrite each other's PNGs and the
// page would show one picture twice.
func TestEveryUpgradeStyleHasItsOwnName(t *testing.T) {
	seen := map[string]UpgradeStyle{}
	for _, u := range UpgradeStyles() {
		if first, ok := seen[u.String()]; ok {
			t.Errorf("styles %d and %d both spell themselves %q", first, u, u.String())
		}
		seen[u.String()] = u
	}
	if len(seen) != len(UpgradeStyles()) {
		t.Errorf("%d styles share %d names", len(UpgradeStyles()), len(seen))
	}
	// The zero value has to be the style the game draws — Spec.UpgradeStyle documents its zero as
	// that, and every caller but the sheet leaves the field alone.
	if DefaultUpgradeStyle != UpgradeStyle(0) {
		t.Errorf("the default style is %s, which is not the zero value", DefaultUpgradeStyle)
	}
}

// **The transparent corners stay transparent.** The wash walks the rounded silhouette rather than
// the image rectangle, which is what stops an upgraded card squaring itself off — the same
// property drawMark's traversal has, and the reason the two share one.
func TestAnUpgradeLeavesTheCornersAlone(t *testing.T) {
	gold := renderOrFail(t, upgradeSpec(systems.UpgradeGolden))
	if c := gold.RGBAAt(0, 0); c.A != 0 {
		t.Errorf("the top-left corner of an upgraded card is %+v, not transparent", c)
	}
}

// **Every upgrade draws a different card**, or two parasites the player spent would look like one
// they spent twice. It is the aggregate version of TestEveryUpgradeHasAnInk below: an ink can exist
// and still be close enough to its neighbour to be indistinguishable on a card.
func TestNoTwoUpgradesDrawTheSameCard(t *testing.T) {
	whole := image.Rect(0, 0, Hand.Width, Hand.Height)
	seen := map[systems.Upgrade]*image.RGBA{}
	for _, u := range systems.Upgrades() {
		seen[u] = renderOrFail(t, upgradeSpec(u))
	}
	for i, a := range systems.Upgrades() {
		for _, b := range systems.Upgrades()[i+1:] {
			if same(seen[a], seen[b], whole) {
				t.Errorf("%s and %s draw the same card", a, b)
			}
		}
	}
}

// **The wildcard is the one upgrade that leaves the form mark hueless**, because it is the one
// whose subject is that the card has no single element. Every other upgrade leaves the element's
// tint where it was — see systems.Upgrade.HuelessForm.
func TestOnlyTheWildcardTakesTheHueOffTheFormMark(t *testing.T) {
	for _, u := range systems.Upgrades() {
		if got, want := u.HuelessForm(), u == systems.UpgradeWild; got != want {
			t.Errorf("%s reports HuelessForm %v, want %v", u, got, want)
		}
	}
}

// **An upgraded card still states.** A disabled one has to read as unavailable whatever has been
// done to it, and a wash applied after the state colouring is exactly where that gets lost.
func TestAnUpgradedCardStillStates(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeGolden)
	dim := rest
	dim.Enabled = false

	whole := image.Rect(0, 0, Hand.Width, Hand.Height)
	if same(renderOrFail(t, rest), renderOrFail(t, dim), whole) {
		t.Error("an upgraded card looks the same afforded and unafforded")
	}
}

// **A cost of zero draws no ticks and does not panic.** Ring and worm cards carry no cost, and an
// upgraded one is not yet possible but costs nothing to be safe about.
func TestAnUpgradedCardWithNoCostRenders(t *testing.T) {
	s := upgradeSpec(systems.UpgradeGolden)
	s.Cost = 0
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
//
// **It no longer requires the ink to be more than one colour.** That check belonged to the left
// column, where a flat ink would have said something an element could already say; a whole-card
// wash in one tint is what nine of the ten upgrades are, and the wildcard's bands are the exception.
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
	}
}

// **UpgradeNone has no ink, which is what a caller checks to leave a card alone.**
func TestNoUpgradeHasNoInk(t *testing.T) {
	if systems.UpgradeInk(systems.UpgradeNone) != nil {
		t.Error("UpgradeNone returned an ink")
	}
}

// **Every upgrade's name round-trips.** Nothing serializes one today — an upgrade is derived from
// the rider a card carries, and riders are stored by name — but the vocabulary is the sheet's
// index and a name that does not parse back is a page that cannot be linked to.
func TestEveryUpgradeNameParsesBack(t *testing.T) {
	for _, u := range systems.Upgrades() {
		got, ok := systems.ParseUpgrade(u.String())
		if !ok || got != u {
			t.Errorf("upgrade %d spells itself %q, which parses back as %d/%v", u, u.String(), got, ok)
		}
	}
	if _, ok := systems.ParseUpgrade("no-such-upgrade"); ok {
		t.Error("an unknown upgrade name resolved to something")
	}
}
