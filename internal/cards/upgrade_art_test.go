package cards

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
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

// face is the hand card's face inside the border ring, kept clear of the rounded corners so a pass
// is the art and not a rounding at the curve.
func face() image.Rectangle {
	return image.Rect(Hand.BorderWidth, Hand.CornerRadius+2, Hand.Width-Hand.BorderWidth, Hand.Height-Hand.CornerRadius-2)
}

// **An upgrade is the whole face.** Its art covers the card inside the border ring, so every part
// of the face — the left edge a stacked row shows, the middle and the right — differs from the
// plain card's.
func TestAnUpgradeCoversTheFace(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone))
	top, bottom := Hand.CornerRadius+2, Hand.Height-Hand.CornerRadius-2
	third := Hand.Width / 3
	for _, u := range systems.Upgrades() {
		card := renderOrFail(t, upgradeSpec(u))
		for i, name := range []string{"left", "middle", "right"} {
			band := image.Rect(i*third+Hand.BorderWidth, top, (i+1)*third-Hand.BorderWidth, bottom)
			if same(plain, card, band) {
				t.Errorf("%s: the %s of the face is the plain card's", u, name)
			}
		}
	}
}

// **The ring is left alone**, because the ring is the card's state.
func TestAnUpgradeLeavesTheBorderAlone(t *testing.T) {
	plain := renderOrFail(t, upgradeSpec(systems.UpgradeNone))
	ring := image.Rect(0, Hand.Height/2, Hand.BorderWidth, Hand.Height/2+20)
	for _, u := range systems.Upgrades() {
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

// **Every upgrade record draws art that exists.** A key naming no file draws nothing, which is a
// card that looks unaltered — the one failure the art exists to prevent. An empty Art is an
// undrawn record and draws the default, so the default has to be there too.
func TestEveryUpgradeRecordHasItsArt(t *testing.T) {
	def := data.UpgradeArtData{}.ArtKey()
	if systems.ArtMark(def, 1, 1) == nil {
		t.Fatalf("the default upgrade art %q is not in the assets", def)
	}
	for k, e := range data.LoadUpgradeArt() {
		if systems.ArtMark(e.ArtKey(), 1, 1) == nil {
			t.Errorf("upgrade %q names art %q, which is not in the assets", k, e.ArtKey())
		}
	}
}

// **An upgrade with its own picture draws a different card from the default and from each
// other.** An undrawn record shares the default picture, deliberately — that is what the fallback
// means — so only the ones drawn are held apart.
func TestOwnUpgradeArtIsItsOwn(t *testing.T) {
	whole := image.Rect(0, 0, Hand.Width, Hand.Height)
	seen := map[string]*image.RGBA{}
	for _, u := range systems.Upgrades() {
		e, _ := UpgradeArtOf(u)
		if _, ok := seen[e.ArtKey()]; !ok {
			seen[e.ArtKey()] = renderOrFail(t, upgradeSpec(u))
		}
	}
	for a, ia := range seen {
		for b, ib := range seen {
			if a < b && same(ia, ib, whole) {
				t.Errorf("upgrades %s and %s draw the same card", a, b)
			}
		}
	}
}

// **The art scales with the card**, so the deck panel's half-size card is the hand card smaller
// rather than a second design — and it still draws.
func TestTheMiniCardDrawsTheUpgrade(t *testing.T) {
	plain, err := Render(upgradeSpec(systems.UpgradeNone), Mini, faces(t))
	if err != nil {
		t.Fatal(err)
	}
	gold, err := Render(upgradeSpec(systems.UpgradeGolden), Mini, faces(t))
	if err != nil {
		t.Fatal(err)
	}
	left := image.Rect(Mini.BorderWidth, Mini.Height/3, Mini.Width/3, 2*Mini.Height/3)
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
// done to it — and the art has to fade with it, or the gold reads louder than the card.
func TestAnUpgradedCardStillStates(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeGolden)
	dim := rest
	dim.Enabled = false

	left := face()
	if same(renderOrFail(t, rest), renderOrFail(t, dim), left) {
		t.Error("the gold art looks the same afforded and unafforded")
	}
}

// **Selecting a card does not change its art.** The card lifts out of the row, and that is what
// says it was picked; the art brightening on the click would be the card changing under the
// cursor.
func TestTheUpgradeArtIsTheSameAtRestAndSelected(t *testing.T) {
	rest := upgradeSpec(systems.UpgradeGolden)
	picked := rest
	picked.Selected = true

	left := face()
	// Under the cost ticks, above the badge and left of the text, which all state the selection
	// themselves: the bare art and nothing else.
	inner := image.Rect(left.Min.X, Hand.Height*2/5, Hand.TextColumnLeft, Hand.BadgeTop)
	if !same(renderOrFail(t, rest), renderOrFail(t, picked), inner) {
		t.Error("the gold art changes when the card is selected")
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

// **UpgradeNone has no ink and draws no art.**
func TestNoUpgradeHasNoInkAndNoEdge(t *testing.T) {
	if systems.UpgradeInk(systems.UpgradeNone) != nil {
		t.Error("UpgradeNone returned an ink")
	}
	if _, ok := UpgradeArtOf(systems.UpgradeNone); ok {
		t.Error("UpgradeNone returned upgrade art")
	}
}

// **Every upgrade's name round-trips**, since upgrade_art.json keys its records by these names.
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

// **Upgrade art is committed at the card's own size**, like every other card picture: the
// generator's output is reduced once by tools/relicart and nothing resamples it at draw time but
// the half-size card's single averaging step.
func TestEveryUpgradeArtIsTheCardsOwnSize(t *testing.T) {
	images := assets.LoadImageData()
	for k, a := range data.LoadUpgradeArt() {
		raw := images[a.ArtKey()]
		if len(raw) == 0 {
			continue // TestEveryUpgradeRecordHasItsArt reports a missing file
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: %v", k, err)
			continue
		}
		if cfg.Width != Hand.Width || cfg.Height != Hand.Height {
			t.Errorf("%s is %dx%d, want the card's %dx%d", k, cfg.Width, cfg.Height, Hand.Width, Hand.Height)
		}
	}
}

// **A figure authored on a transparent ground is drawn as it is over an upgrade**, not matted:
// the matte measures a ground off the picture's outer ring, and a transparent ring would read
// every pixel as ground. A figure that vanished would leave the card plain upgrade art.
func TestATransparentFigureSurvivesAnUpgrade(t *testing.T) {
	fig := image.NewNRGBA(image.Rect(0, 0, Hand.Width, Hand.Height))
	for y := 100; y < 180; y++ {
		for x := 80; x < 140; x++ {
			fig.SetNRGBA(x, y, color.NRGBA{R: 30, G: 40, B: 200, A: 255})
		}
	}
	s := upgradeSpec(systems.UpgradeGolden)
	bare := renderOrFail(t, s)
	s.Art = fig
	with := renderOrFail(t, s)
	if same(bare, with, image.Rect(90, 110, 130, 170)) {
		t.Error("a transparent figure drew nothing over the gold face")
	}
}
