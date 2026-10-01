package cards

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// upgradeSpec is a card in one upgrade, for comparing two renderings of the same card.
func upgradeSpec(u systems.Upgrade) Spec {
	return Spec{
		Name:    "Skewer",
		Form:    FormStab,
		Cost:    3,
		Element: Fire,
		Upgrade: u,
		Text:    "2x DMG",
		Enabled: true,
	}
}

// renderOrFail is one card, or a fatal test failure.
func renderOrFail(t *testing.T, s Spec) *image.RGBA {
	t.Helper()
	img, err := Render(s, Hand, faces(t))
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	return img
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

// strips is where an edge record draws on the hand card, left then right, kept clear of the
// rounded corners so a pass is the strip and not a rounding at the curve.
func strips(e data.EdgeData) (left, right image.Rectangle) {
	st := Hand
	top, bottom := st.CornerRadius+2, st.Height-st.CornerRadius-2
	left = image.Rect(e.Inset, top, e.Inset+e.Width, bottom)
	right = image.Rect(st.Width-e.Inset-e.Width, top, st.Width-e.Inset, bottom)
	return left, right
}

// **An upgrade draws on the edges its record names and nowhere else.** The strip is the whole of
// how an upgrade is drawn, so the rest of the face is the plain card's, pixel for pixel — a change
// there is a wash creeping back.
func TestAnUpgradeDrawsOnItsEdgesAndNowhereElse(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone))
	for _, u := range systems.Upgrades() {
		e, ok := EdgeOf(u)
		if !ok {
			t.Fatalf("%s has no edge", u)
		}
		card := renderOrFail(t, upgradeSpec(u))
		left, right := strips(e)
		if same(plain, card, left) {
			t.Errorf("%s: the left edge is the plain card's", u)
		}
		both := e.Sides == data.EdgeBoth
		if same(plain, card, right) == both {
			t.Errorf("%s: the right edge changed=%v, and its record says Sides %q", u, !both, e.Sides)
		}
		// Below the form mark, which the wildcard draws hueless on purpose.
		end := Hand.Width
		if both {
			end = right.Min.X
		}
		between := image.Rect(left.Max.X, Hand.FormTop+Hand.FormSize, end, Hand.Height)
		if !same(plain, card, between) {
			t.Errorf("%s: the face beside the strips changed", u)
		}
	}
}

// **At an inset of the border width or more, the ring is left alone**, because the ring is the
// card's state. A record that sets a smaller inset has asked for the ring and is not checked.
func TestAnInsetStripLeavesTheBorderAlone(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone))
	ring := image.Rect(0, Hand.Height/2, Hand.BorderWidth, Hand.Height/2+20)
	for _, u := range systems.Upgrades() {
		if e, _ := EdgeOf(u); e.Inset < Hand.BorderWidth {
			continue
		}
		if !same(plain, renderOrFail(t, upgradeSpec(u)), ring) {
			t.Errorf("%s paints the border ring", u)
		}
	}
}

// **The transparent corners stay transparent**, or an upgraded card squares itself off.
func TestAnUpgradeLeavesTheCornersAlone(t *testing.T) {
	for _, u := range systems.Upgrades() {
		if c := renderOrFail(t, upgradeSpec(u)).RGBAAt(0, 0); c.A != 0 {
			t.Errorf("the top-left corner of a %s card is %+v, not transparent", u, c)
		}
	}
}

// **Every edge record draws art that exists.** A key naming no file draws nothing, which is a
// card that looks unaltered — the one failure the strip exists to prevent. An empty Art is an
// undrawn record and draws the default, so the default has to be there too.
func TestEveryEdgeRecordHasItsArt(t *testing.T) {
	if systems.ArtMark(data.DefaultEdgeArt, 1, 1) == nil {
		t.Fatalf("the default edge art %q is not in the assets", data.DefaultEdgeArt)
	}
	for k, e := range data.LoadEdges() {
		if systems.ArtMark(e.ArtKey(), 1, 1) == nil {
			t.Errorf("edge %q names art %q, which is not in the assets", k, e.ArtKey())
		}
	}
}

// **An upgrade with its own picture draws a different card from the default and from each
// other.** An undrawn record shares the default picture, deliberately — that is what the fallback
// means — so only the ones drawn are held apart.
func TestOwnEdgesAreTheirOwn(t *testing.T) {
	whole := image.Rect(0, 0, Hand.Width, Hand.Height)
	seen := map[string]*image.RGBA{}
	for _, u := range systems.Upgrades() {
		e, _ := EdgeOf(u)
		if _, ok := seen[e.ArtKey()]; !ok {
			seen[e.ArtKey()] = renderOrFail(t, upgradeSpec(u))
		}
	}
	for a, ia := range seen {
		for b, ib := range seen {
			if a < b && same(ia, ib, whole) {
				t.Errorf("edges %s and %s draw the same card", a, b)
			}
		}
	}
}

// **The strip scales with the card**, so the deck panel's half-size card is the hand card smaller
// rather than a second design — and it still draws.
func TestTheMiniCardDrawsTheStrip(t *testing.T) {
	plain, err := Render(upgradeSpec(systems.UpgradeNone), Mini, faces(t))
	if err != nil {
		t.Fatal(err)
	}
	gold, err := Render(upgradeSpec(systems.UpgradeGolden), Mini, faces(t))
	if err != nil {
		t.Fatal(err)
	}
	e, _ := EdgeOf(systems.UpgradeGolden)
	left := image.Rect(e.Inset/2, Mini.Height/3, e.Inset/2+e.Width/2, 2*Mini.Height/3)
	if same(plain, gold, left) {
		t.Error("the half-size gold card's left edge is the plain card's")
	}
}

// **The wildcard is the one upgrade that leaves the form mark hueless**, because it is the one
// whose subject is that the card has no single element. See systems.Upgrade.HuelessForm.
func TestOnlyTheWildcardTakesTheHueOffTheFormMark(t *testing.T) {
	for _, u := range systems.Upgrades() {
		if got, want := u.HuelessForm(), u == systems.UpgradeWild; got != want {
			t.Errorf("%s reports HuelessForm %v, want %v", u, got, want)
		}
	}
}

// **An upgraded card still states.** A disabled one has to read as unavailable whatever has been
// done to it — and the strip has to fade with it, or the gold reads louder than the card.
func TestAnUpgradedCardStillStates(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeGolden)
	dim := rest
	dim.Enabled = false

	e, _ := EdgeOf(systems.UpgradeGolden)
	left, _ := strips(e)
	if same(renderOrFail(t, rest), renderOrFail(t, dim), left) {
		t.Error("the gold strip looks the same afforded and unafforded")
	}
}

// **Selecting a card does not change its strip.** The card lifts out of the row, and that is what
// says it was picked; the strip brightening on the click would be the card changing under the
// cursor.
func TestTheStripIsTheSameAtRestAndSelected(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeGolden)
	picked := rest
	picked.Selected = true

	e, _ := EdgeOf(systems.UpgradeGolden)
	left, _ := strips(e)
	// Under the cost ticks, above the badge and left of the text, which all state the selection
	// themselves: the bare strip and nothing else.
	inner := image.Rect(left.Min.X, Hand.Height*2/5, Hand.TextColumnLeft, Hand.BadgeTop)
	if !same(renderOrFail(t, rest), renderOrFail(t, picked), inner) {
		t.Error("the gold strip changes when the card is selected")
	}
}

// **A cost of zero draws no ticks and does not panic.**
func TestAnUpgradedCardWithNoCostRenders(t *testing.T) {
	s := upgradeSpec(systems.UpgradeGolden)
	s.Cost = 0
	renderOrFail(t, s) // must not panic
}

// **Every upgrade in the vocabulary has an ink**, which is what its word and its signal are
// colored from.
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

// **UpgradeNone has no ink and draws no edge.**
func TestNoUpgradeHasNoInkAndNoEdge(t *testing.T) {
	if systems.UpgradeInk(systems.UpgradeNone) != nil {
		t.Error("UpgradeNone returned an ink")
	}
	if _, ok := EdgeOf(systems.UpgradeNone); ok {
		t.Error("UpgradeNone returned an edge")
	}
}

// **Every upgrade's name round-trips**, since edges.json keys its records by these names.
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
