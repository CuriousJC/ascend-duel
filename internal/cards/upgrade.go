package cards

// Upgrades on a card face: **washing the whole finished card in the upgrade's own ink.**
//
// An upgrade is what the run has permanently made a card, and a card carries exactly one — see
// `systems.Upgrade`, which is where the vocabulary and the argument for it live. This file is the
// drawing: the ink sampled across the card's rectangle and every pixel of the card pulled toward
// what it finds there.
//
// # It took the left column until 2026-09-09
//
// The first upgrade was the wildcard, and it painted the form mark and the cost ticks from a
// rainbow rather than from one element's colour. That mechanism — three tint modes, a sampled
// glyph, a gradient projected across the cost stack — was deleted rather than kept beside this one,
// on the rule that a removal is a deletion: nine of the ten upgrades have nothing to say about the
// element, so a left column in gold would be the element slot saying something that is not about
// the element. `tools/upgradesheet` is where the comparison lived and it now shows whole cards.
//
// # Where it goes is a question to settle by looking, so all three answers are here
//
// `UpgradeStyle` is the border, the whole card, or the face-without-the-border. It is a review knob
// in exactly the shape `TintMode` was — the game draws one and `tools/upgradesheet` draws all of
// them — because "does a gold border say enough" is not a question anybody wins by arguing.
//
// # It is washInside, and that is on purpose
//
// The mark file already had to pull a whole card toward a colour — the tutorial's red, the
// shatter's dim — so the only thing new here is that the colour varies per pixel. Sharing the
// traversal is what keeps an upgraded card and a marked one agreeing about which pixels are inside
// the rounded silhouette and which are the transparent corners.
//
// **The order is upgrade first, mark second, and mark.go says why.** An upgrade is what the card
// *is* and belongs in the face; a mark is the card's situation and belongs on top of it. A gold
// card the tutorial is pointing at reads as gold and pointed-at, in that order.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// UpgradeStyle is *where* an upgrade is painted on the card.
//
// **Two answers, kept side by side because the question is one to settle by looking.** That is the
// shape `TintMode` had before it: a review knob, drawn in every value by `tools/upgradesheet` and
// in exactly one by the game. Nothing outside this package and that tool names one, and no file
// writes it down — **not append-only and never serialized.**
//
// **What settled it is that the border is already saying something** *(owner's call, 2026-09-09)*.
// It carries the card's state — resting, selected, unaffordable, dragged — so the default washes
// everything *but* the ring: the card goes gold and the ring goes on saying what it was saying. The
// other two are kept because how loud an upgrade should be is still open.
type UpgradeStyle int

const (
	// UpgradeWashFace washes everything *but* the border: the card goes gold and the ring around it
	// stays exactly what it was. **It is what the game draws** *(owner's call, 2026-09-09)*, and it
	// is **first in the enum so it is the zero value** — the rule `Spec.UpgradeStyle` documents:
	// every caller but the sheet never thinks about this, and they must all get the picture the game
	// draws.
	//
	// **The border is the card's *state* and the upgrade is not.** Resting, selected, unaffordable
	// and being dragged are all said by the ring, in a wash away from the neutral grey; an upgrade
	// painted over that is a second thing in the one place the card says the first. Keeping them
	// apart is what lets a queued gold card read as queued *and* gold rather than as one or the
	// other winning.
	//
	// **It also keeps the card's outline.** A card washed to its very edge sits on the table with
	// nothing separating it from the table, and a hand of eight is eight shapes that have to be told
	// apart before any of them is read.
	UpgradeWashFace UpgradeStyle = iota

	// UpgradeBorder paints the card's border band and nothing else — the opposite of the default,
	// and the one that costs the state signal it would be sharing the ring with. Kept because it is
	// the quietest answer and the question of how loud an upgrade should be is not closed.
	UpgradeBorder

	// UpgradeWash pulls the whole finished face toward the ink, the border included. It is the
	// loudest of the three and the most unambiguous — nobody misses a gold card — and it is the one
	// that costs the most legibility.
	UpgradeWash
)

// DefaultUpgradeStyle is what the game draws. See UpgradeWashFace for why it is the zero value.
const DefaultUpgradeStyle = UpgradeWashFace

func (u UpgradeStyle) String() string {
	switch u {
	case UpgradeWash:
		return "wash"
	case UpgradeBorder:
		return "border"
	default:
		return "wash-face"
	}
}

// UpgradeStyles is every style in a fixed order, for the sheet that compares them. **The default
// leads**, so the page reads as "here is what the game draws, and here is what it declined".
func UpgradeStyles() []UpgradeStyle {
	return []UpgradeStyle{UpgradeWashFace, UpgradeBorder, UpgradeWash}
}

// drawUpgrade paints a finished card's upgrade, in whichever style the spec asked for.
//
// **The ink is projected across the whole card in every style, not across the region being
// painted.** A flat ink gives a flat tint either way; the wildcard's five bands give five bands
// running down the *card*, so the border variant picks up the band each edge pixel sits in and the
// two styles agree about which colour belongs where. Projecting across the border ring instead
// would put a whole rainbow on each of four edges.
//
// A card with no upgrade is left exactly as it was, which is almost every card.
func drawUpgrade(dst *image.RGBA, s Spec, st Style) {
	ink := systems.UpgradeInk(s.Upgrade)
	if ink == nil {
		return
	}
	w, h, radius := st.Width, st.Height, st.CornerRadius
	box := image.Rect(0, 0, w, h)
	at := func(x, y int) color.RGBA { return sampleBox(ink, image.Pt(x, y), box) }

	switch s.UpgradeStyle {
	case UpgradeWash:
		washInsideFrom(dst, w, h, radius, systems.UpgradeWashPct, at)
	case UpgradeBorder:
		washRegionFrom(dst, w, h, radius, systems.UpgradeBorderPct, at, borderOf(st))
	default:
		washRegionFrom(dst, w, h, radius, systems.UpgradeWashPct, at, faceOf(st))
	}
}

// borderOf reports whether a pixel is in the card's border ring, and faceOf whether it is inside
// that ring.
//
// **One geometry, read two ways**, so the border style and the wash-the-face style cannot disagree
// about where the boundary is and leave a card with a one-pixel seam between them. The inner shape
// is the one `roundedBorder` fills — inset by the border width, with the radius shrunk to match, so
// the two curves stay parallel.
func borderOf(st Style) func(x, y int) bool {
	inside := faceOf(st)
	return func(x, y int) bool { return !inside(x, y) }
}

func faceOf(st Style) func(x, y int) bool {
	bw := st.BorderWidth
	iw, ih := st.Width-2*bw, st.Height-2*bw
	if bw <= 0 || iw <= 0 || ih <= 0 {
		// A style with no border — the deck panel's, at zero width — is all face. Reporting the
		// whole card as border would paint a borderless card entirely in the upgrade's ink.
		return func(int, int) bool { return true }
	}
	return func(x, y int) bool { return insideRounded(iw, ih, st.CornerRadius-bw, x-bw, y-bw) }
}

// sampleBox maps a point inside box onto the ink and returns the colour there, unpremultiplied —
// which is what a caller mixing toward it wants, since the ink's own alpha is not the card's.
func sampleBox(ink *image.RGBA, at image.Point, box image.Rectangle) color.RGBA {
	b := ink.Bounds()
	x := scaleInto(at.X-box.Min.X, box.Dx(), b.Dx())
	y := scaleInto(at.Y-box.Min.Y, box.Dy(), b.Dy())
	return unpremultiply(ink.RGBAAt(b.Min.X+x, b.Min.Y+y))
}

// unpremultiply takes a stored pixel back to straight colour, which is what a wash wants: a hue to
// mix toward rather than a colour already scaled by somebody else's alpha.
func unpremultiply(c color.RGBA) color.RGBA {
	if c.A == 0 || c.A == 255 {
		return c
	}
	up := func(v uint8) uint8 {
		n := int(v) * 255 / int(c.A)
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return color.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: 255}
}

// scaleInto maps v in 0..span onto 0..into-1, clamped. Nearest rather than interpolated: the ink
// is a small authored picture and averaging its bands would mud the colours it exists to state.
func scaleInto(v, span, into int) int {
	if into <= 0 || span <= 0 {
		return 0
	}
	out := v * into / span
	if out < 0 {
		return 0
	}
	if out >= into {
		return into - 1
	}
	return out
}
