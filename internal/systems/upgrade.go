package systems

// Upgrades: **a card saying on its own face that the run has altered it.**
//
// A parasite has been able to change a card since 2026-08-27 — recolour it, reform it, hang a
// rider on it — and until now none of that reached the drawing. Sixteen parasites attach seven
// kinds of rider and the only place a rider was visible was the tooltip prose, which is a hand of
// altered cards the player cannot read at a glance. `combat.MaxCardRiders` has been 3 since it was
// written *because the face has room for three badges*; the room was reserved and never used.
//
// This is the first thing to use it, and it is deliberately not a badge. What a wildcard changes
// is the one fact the left column already states — the card's element — so it says it by taking
// the column over rather than by adding a fourth mark to a card that has three.
//
// # Why the vocabulary is here and not in internal/cards
//
// `internal/cards` already reaches into this package for every glyph it draws, including the four
// authored form marks, and an upgrade's ink is loaded and cached exactly the way those are. A
// second enum in `cards` mirroring one here would be two lists to keep in step for no gain.
//
// **What this package must not learn is what a rider is.** An Upgrade is a *presentation* value:
// something visible has happened to this card, and here is what to paint it with.
// `internal/screens` is where a rider becomes an Upgrade, on the same terms `Spec.TextInk` is
// where a ring becomes a colour.

import (
	"bytes"
	"image"
	"image/draw"
	"log"

	"github.com/curiousjc/ascend-duel/assets"
)

// Upgrade is a visible alteration to a card.
//
// **Append-only, and it may never be serialized as a number.** It is an ordinal indexing a
// registry and a cache, exactly like GlyphKind, Element and ConceptID — and a run snapshot
// outlives the build that wrote it. Nothing writes one down today: an upgrade is *derived* from
// the riders a card carries, which are themselves stored by name.
type Upgrade int

const (
	// UpgradeNone is a card the run has not visibly altered, which is almost every card. It is
	// the zero value so a Spec built without thinking about upgrades draws as it always did.
	UpgradeNone Upgrade = iota

	// UpgradeWild is a card that counts as every element at once. It paints the left column from
	// a rainbow rather than from one element's colour, which is as close to "all five" as a
	// column that has to state one thing can get.
	UpgradeWild
)

// Upgrades is every visible upgrade in a fixed order, for anything that walks them —
// `tools/upgradesheet`, and the tests that hold this list against the art registry.
//
// UpgradeNone is deliberately absent: it is the absence of an upgrade rather than one of them,
// the same way RiderNone is left out of combat.RiderKinds.
func Upgrades() []Upgrade {
	return []Upgrade{UpgradeWild}
}

func (u Upgrade) String() string {
	switch u {
	case UpgradeWild:
		return "wild"
	default:
		return "none"
	}
}

// ParseUpgrade resolves an upgrade from its name and reports failure rather than falling back to
// one, the posture every other closed vocabulary in this codebase takes.
func ParseUpgrade(name string) (Upgrade, bool) {
	for _, u := range Upgrades() {
		if u.String() == name {
			return u, true
		}
	}
	return UpgradeNone, false
}

// UpgradeInkSize is the square every upgrade ink is authored at.
//
// **It is the form marks' size on purpose.** The mark is the biggest thing an ink has to cover,
// so an ink at least that big can be sampled down for the cost ticks without ever being sampled
// *up*. See formArtSize, which is the same number for the same reason.
const UpgradeInkSize = formArtSize

// upgradeArt is where each upgrade's ink comes from: an assets.LoadImageData key, never a path,
// so a file can be refiled without touching this.
//
// **Every upgrade in Upgrades() must have an entry**, which TestEveryUpgradeHasAnInk holds. An
// upgrade with no ink would draw as an ordinary card and be invisible, which is the exact failure
// the whole idea exists to fix.
var upgradeArt = map[Upgrade]string{
	UpgradeWild: "wildcardupgrade_png",
}

// upgradeInkCache holds the decoded inks. Decoding a PNG is not a per-card operation, let alone a
// per-frame one — the same argument artCache is under.
var upgradeInkCache = map[Upgrade]*image.RGBA{}

// UpgradeInk is the colour source an upgrade paints with, as a premultiplied RGBA square of
// UpgradeInkSize. It returns nil for UpgradeNone, which is what a caller checks to fall back to
// the card's own element.
//
// **It is handed out whole rather than resized.** A caller samples it — `internal/cards` maps a
// glyph pixel's brightness onto it for one mode and a position within the mark for the other, and
// the cost ticks project it across their own stack — so resizing here would be throwing away
// resolution three different callers each want differently.
//
// A decode failure is fatal for the reason renderArt's is: the bytes are compiled into the
// binary, so there is no runtime condition under which this fails on one machine and not another.
// It means the file is not a PNG, which is a build problem rather than a case to fall back from.
func UpgradeInk(u Upgrade) *image.RGBA {
	if u == UpgradeNone {
		return nil
	}
	if img, ok := upgradeInkCache[u]; ok {
		return img
	}

	key, ok := upgradeArt[u]
	if !ok {
		log.Fatalf("upgrade %q has no ink in upgradeArt", u)
	}
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		log.Fatalf("upgrade ink %q is not in assets.LoadImageData", key)
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		log.Fatalf("failed to decode upgrade ink %q: %v", key, err)
	}

	b := src.Bounds()
	if b.Dx() != UpgradeInkSize || b.Dy() != UpgradeInkSize {
		log.Fatalf("upgrade ink %q is %dx%d, want %dx%d",
			key, b.Dx(), b.Dy(), UpgradeInkSize, UpgradeInkSize)
	}

	// Into an RGBA of its own: a PNG with alpha decodes to NRGBA, and every caller samples this
	// alongside premultiplied values.
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), src, b.Min, draw.Src)

	upgradeInkCache[u] = img
	return img
}
