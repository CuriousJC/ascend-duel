package systems

// Upgrades: **what the run has permanently made a card, and the color it is named in.**
//
// A card has a form, an element and an action. Those three compose freely and a rune may move
// any of them — and none of them is an upgrade, because the card's own face already states all
// three. An **upgrade** is the fourth thing, and a card carries exactly one: see
// `combat.MaxCardRiders`.
//
// **The card draws an upgrade as its face**, authored art named per upgrade in
// `data/upgrade_art.json` — see `internal/cards/upgrade_art.go`. What this file holds is the rest of
// an upgrade's presentation: its name, and the ink a word or a signal naming it is colored from.
//
// # Why every upgrade is an ink and not a color
//
// Ten of the eleven are one flat tint and would have been happier as a `color.RGBA`. The other is
// the wildcard, whose subject is that the card has no single element, so its ink is the five
// element colors in bands — CHROMATIC in a tooltip title is set letter by letter across them. An
// ink is the shape that holds both: the authored PNG for the one that needs a picture, a generated
// square for the rest, and one sampling path in `internal/cards` that never asks which it got.
//
// **What this package must not learn is what a rider is.** An Upgrade is a *presentation* value:
// something visible has happened to this card, and here is what to paint it with.
// `internal/screens` is where a rider becomes an Upgrade, on the same terms `Spec.TextInk` is
// where a relic becomes a color.
//
// # The colors are placeholders and they are standing on a full wheel
//
// Hue is spent: five elements, a relic's pink, the two verbs, the two duelists and the ground.
// Eleven upgrades wanting eleven distinguishable tints is more hue than the game has left, so
// what is here is picked to be *told apart* rather than to mean anything — gold and silver are
// metals rather than hues and are the exception.

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"log"
)

// Upgrade is a visible alteration to a card.
//
// **Append-only, and it may never be serialized as a number.** It is an ordinal indexing a
// registry and a cache, exactly like Element and ConceptID — and a run snapshot
// outlives the build that wrote it. Nothing writes one down today: an upgrade is *derived* from
// the one rider a card carries, which is itself stored by name.
type Upgrade int

const (
	// UpgradeNone is a card the run has not altered, which is almost every card. It is the zero
	// value so a Spec built without thinking about upgrades draws as it always did.
	UpgradeNone Upgrade = iota

	// UpgradeWild is a card that counts as every element at once. **The only upgrade whose ink is
	// a picture**: the five element colors in bands, which is as close to "all five" as one word
	// can get. It is also the only one that leaves the form mark hueless — see HuelessForm.
	UpgradeWild

	// UpgradeGolden is a card that gambles for a permanent bonus every time it is played.
	UpgradeGolden

	// UpgradeSilver is UpgradeGolden's cheaper metal, gambling for vitae.
	UpgradeSilver

	// UpgradeHeal is a card that restores life as it is played.
	UpgradeHeal

	// UpgradeShield is a card that raises shields as it is played.
	UpgradeShield

	// UpgradeDamage is a card that adds to its duelist's DMG for the blow it is played into.
	UpgradeDamage

	// UpgradeCombo is a card that multiplies the blow, when it is one of the cards the hand was
	// formed from.
	UpgradeCombo

	// UpgradeHeldDamage is a card worth DMG for as long as it is *not* played.
	UpgradeHeldDamage

	// UpgradeHeldScale is a card that multiplies the blow for as long as it is not played.
	UpgradeHeldScale

	// UpgradeHeldVitae is a card that pays vitae for as long as it is not played.
	UpgradeHeldVitae

	// UpgradeVersatile is a card that counts as any attack form at once — the wildcard on the form
	// axis. A flat placeholder tint like the other flat ones; it leaves the form mark alone, since
	// the mark still says the form the card is.
	UpgradeVersatile
)

// Upgrades is every visible upgrade in a fixed order, for anything that walks them —
// `tools/upgradesheet`, and the tests that hold this list against the ink registry.
//
// UpgradeNone is deliberately absent: it is the absence of an upgrade rather than one of them,
// the same way RiderNone is left out of combat.RiderKinds.
func Upgrades() []Upgrade {
	return []Upgrade{
		UpgradeWild, UpgradeGolden, UpgradeSilver,
		UpgradeHeal, UpgradeShield, UpgradeDamage, UpgradeCombo,
		UpgradeHeldDamage, UpgradeHeldScale, UpgradeHeldVitae,
		UpgradeVersatile,
	}
}

func (u Upgrade) String() string {
	switch u {
	case UpgradeWild:
		return "wild"
	case UpgradeGolden:
		return "golden"
	case UpgradeSilver:
		return "silver"
	case UpgradeHeal:
		return "heal"
	case UpgradeShield:
		return "shield"
	case UpgradeDamage:
		return "damage"
	case UpgradeCombo:
		return "combo"
	case UpgradeHeldDamage:
		return "held-damage"
	case UpgradeHeldScale:
		return "held-scale"
	case UpgradeHeldVitae:
		return "held-vitae"
	case UpgradeVersatile:
		return "versatile"
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

// HuelessForm reports whether this upgrade leaves the card's form mark untinted.
//
// **True for the wildcard alone, and it is not a special case dressed up as a rule** *(owner's
// call, 2026-09-09)*. The left column exists to state the card's element. A wildcard's element is
// still what the card *is* — fire relics still read it, it is still drawn from the fire row — but what it
// *counts as* is every element at once, so a column stating one of them is stating the less useful
// half of the truth. It goes hueless, and the card's edges say the rest.
//
// Every other upgrade leaves the element alone, because none of them is about the element.
func (u Upgrade) HuelessForm() bool { return u == UpgradeWild }

// UpgradeInkSize is the square every upgrade ink is authored or generated at. A band pattern needs
// enough squares to read as bands and not enough to alias.
const UpgradeInkSize = 32

// upgradeArt is where an upgrade's ink comes from when it is a *picture*: an assets.LoadImageData
// key, never a path, so a file can be refiled without touching this.
//
// **The wildcard is the only entry and that is the design** — see the file comment. Everything else
// is a flat tint and is generated from upgradeTint below, which is what a placeholder color should
// cost: a line, not a PNG somebody has to draw before the mechanic can be looked at.
var upgradeArt = map[Upgrade]string{
	UpgradeWild: "wildcardupgrade_png",
}

// upgradeTint is the flat color an upgrade is named in, for every one that is not a picture.
//
// **Every upgrade in Upgrades() must be in exactly one of these two maps**, which
// TestEveryUpgradeHasAnInk holds. An upgrade in neither would have nothing to color its word or
// its signal with.
//
// **These are placeholders standing on a full wheel** — see the file comment, which is where that
// is argued rather than repeated per line. The two metals are the exception and are not
// placeholders: gold and silver are what the mechanic is called.
var upgradeTint = map[Upgrade]color.RGBA{
	// The four that fire when the card is played, kept warm and apart from the held three below,
	// so which half of the bargain a card is on reads before which upgrade it is.
	UpgradeHeal:   {R: 226, G: 132, B: 150, A: 255},
	UpgradeShield: {R: 150, G: 116, B: 92, A: 255},
	UpgradeDamage: {R: 206, G: 118, B: 74, A: 255},
	UpgradeCombo:  {R: 190, G: 96, B: 120, A: 255},

	// The three that pay while the card is *held*, kept cool for the same reason.
	UpgradeHeldDamage: {R: 92, G: 140, B: 148, A: 255},
	UpgradeHeldScale:  {R: 108, G: 154, B: 128, A: 255},
	UpgradeHeldVitae:  {R: 128, G: 148, B: 176, A: 255},

	// The form wildcard. A placeholder green, like the flat tints above.
	UpgradeVersatile: {R: 46, G: 139, B: 107, A: 255},

	// The two metals. **Not placeholders**: gold and silver are what the mechanic is called.
	UpgradeGolden: {R: 226, G: 176, B: 46, A: 255},
	UpgradeSilver: {R: 214, G: 222, B: 236, A: 255},
}

// upgradeInkCache holds the decoded and generated inks. Decoding a PNG is not a per-card operation,
// let alone a per-frame one — the same argument artCache is under.
var upgradeInkCache = map[Upgrade]*image.RGBA{}

// UpgradeInk is the color source an upgrade is named in, as a premultiplied RGBA square of
// UpgradeInkSize. It returns nil for UpgradeNone.
//
// **It is handed out whole rather than resized.** The caller samples it across its own span, so a
// flat ink gives one color and a banded one gives bands — which is what lets one path color GOLD
// and CHROMATIC.
//
// A decode failure is fatal for the reason renderArt's is: the bytes are compiled into the binary,
// so there is no runtime condition under which this fails on one machine and not another. It means
// the file is not a PNG, which is a build problem rather than a case to fall back from.
func UpgradeInk(u Upgrade) *image.RGBA {
	if u == UpgradeNone {
		return nil
	}
	if img, ok := upgradeInkCache[u]; ok {
		return img
	}

	var out *image.RGBA
	switch key, drawn := upgradeArt[u]; {
	case drawn:
		out = decodeInk(key)
	default:
		tint, ok := upgradeTint[u]
		if !ok {
			log.Fatalf("upgrade %q has neither art nor a tint", u)
		}
		out = flatInk(tint)
	}
	upgradeInkCache[u] = out
	return out
}

// flatInk is one color as an ink square, which is what makes a plain tint indistinguishable from a
// picture at the sampling site.
func flatInk(c color.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, UpgradeInkSize, UpgradeInkSize))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return out
}

// decodeInk reads an authored ink out of the embedded assets and holds it to the one size every
// caller assumes.
func decodeInk(key string) *image.RGBA {
	raw := embeddedImages()[key]
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

	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	return out
}

// UpgradeTint is the flat color an upgrade is identified by, for a caller that wants the color
// without the ink square UpgradeInk hands out.
//
// **It exists so a signal on the combat screen is drawn in the color of the rider that threw it** —
// see screens.cardSignal. A firework in a color of its own would be a second vocabulary for the
// same upgrade, and the two would drift the first time a placeholder tint was retuned.
//
// UpgradeNone and anything unlisted answer with a zero color, which a caller checks the same way
// it checks UpgradeInk for nil.
func UpgradeTint(u Upgrade) color.RGBA {
	return upgradeTint[u]
}
