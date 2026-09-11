package systems

// Upgrades: **the whole card saying what the run has permanently made it.**
//
// A card has a form, an element and an action. Those three compose freely and a parasite may move
// any of them — and none of them is an upgrade, because the card's own face already states all
// three. An **upgrade** is the fourth thing, and a card carries exactly one: see
// `combat.MaxCardRiders`, which came down to one on the same day this became a whole-card overlay.
//
// # It took the left column until 2026-09-09, and now it takes the card
//
// The first upgrade was the wildcard, and it said itself by painting the form mark and the cost
// ticks from a rainbow instead of from one element's colour — which was the right answer while
// there was one upgrade and its whole subject was the element. It stopped being the right answer
// the moment there were ten: nine of them have nothing to do with the element, so a left column in
// gold would be the element slot saying something that is not about the element.
//
// **So an upgrade washes the finished face, the border included** *(owner's call, 2026-09-09)*. It
// is the treatment `cards.MarkHighlit` already uses for the tutorial's red and `MarkShattered` for
// a break — the difference being what the two are *about*, which is the distinction the vocabulary
// keeps: an upgrade is what the card permanently is and is painted into the face, and a mark is the
// card's situation and is painted over the top of it. A broken gold card reads as both.
//
// # Why every upgrade is an ink and not a colour
//
// Nine of the ten are one flat tint and would have been happier as a `color.RGBA`. The tenth is the
// wildcard, whose subject is that the card has no single element, so its wash is the five element
// colours in bands — and a vocabulary where one entry is a picture and nine are colours is two
// mechanisms with a `switch` between them. An ink is the shape that holds both: the authored PNG
// for the one that needs a picture, a generated square for the nine that do not, and one sampling
// path in `internal/cards` that never asks which it got.
//
// **What this package must not learn is what a rider is.** An Upgrade is a *presentation* value:
// something visible has happened to this card, and here is what to paint it with.
// `internal/screens` is where a rider becomes an Upgrade, on the same terms `Spec.TextInk` is
// where a relic becomes a colour.
//
// # The colours are placeholders and they are standing on a full wheel
//
// **Said out loud because it is a real cost** *(2026-09-09)*. CLAUDE.md records that hue is spent:
// five elements, a relic's pink, the two verbs, the two duelists and now the ground. Ten upgrades
// wanting ten distinguishable tints is more hue than the game has left, so what is here is
// deliberately temporary — gold and silver are metals rather than hues and are safe, and the other
// eight are picked to be *told apart* rather than to mean anything. The permanent answer is
// probably an authored ink each, the way the wildcard has one.

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"

	"github.com/curiousjc/ascend-duel/assets"
)

// Upgrade is a visible alteration to a card.
//
// **Append-only, and it may never be serialized as a number.** It is an ordinal indexing a
// registry and a cache, exactly like GlyphKind, Element and ConceptID — and a run snapshot
// outlives the build that wrote it. Nothing writes one down today: an upgrade is *derived* from
// the one rider a card carries, which is itself stored by name.
type Upgrade int

const (
	// UpgradeNone is a card the run has not altered, which is almost every card. It is the zero
	// value so a Spec built without thinking about upgrades draws as it always did.
	UpgradeNone Upgrade = iota

	// UpgradeWild is a card that counts as every element at once. **The only upgrade whose wash is
	// a picture**: the five element colours in bands, which is as close to "all five" as one card
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
// still what the card *is* — it still burns, it is still drawn from the fire row — but what it
// *counts as* is every element at once, so a column stating one of them is stating the less useful
// half of the truth. It goes hueless, and the card's wash says the rest.
//
// Every other upgrade leaves the element alone, because none of them is about the element.
func (u Upgrade) HuelessForm() bool { return u == UpgradeWild }

// UpgradeWashPct is how far an upgraded card is pulled toward its ink, in percent.
//
// **Unmistakable across a row of eight and not enough to stop the card being read.** It is well past
// `cards.highlightWash`'s 30, and the reason is the card's own surface rather than a taste for loud
// cards: the tutorial's red is far from off-white and moves it at any strength, where a metal is
// *close* to off-white and anything gentler than this comes out as cream. Gold was tried at 22, 34
// and 40 and read as warm paper at all three. See sheenInk, which is the other half of that fix.
//
// **What it costs is that the eight flat placeholders are loud**, which is the right direction for a
// placeholder to be wrong in — a subtle one is one nobody notices needs replacing.
const UpgradeWashPct = 55

// UpgradeBorderPct is how far a card's *border* is pulled toward the ink, for the style that paints
// the border and leaves the face alone.
//
// **Far higher than the wash, because the border is 3px and has one job.** A wash has the whole
// card to be noticed on and has to leave the text readable underneath it; a border has neither
// problem — nothing is written on it — so the only question is whether it reads as gold from across
// the table, and the answer to that is "take it".
//
// **Not 100.** The border still says the card's *state* — resting, selected, unaffordable — through
// `Spec.atState`, and replacing it outright would delete that signal on every upgraded card. Leaving
// a fifth of the underlying colour is what lets a selected gold card still read as selected.
const UpgradeBorderPct = 80

// UpgradeInkSize is the square every upgrade ink is authored or generated at.
//
// **It is the form marks' size on purpose.** The mark was the biggest thing an ink had to cover
// when an upgrade took the left column, and the number is kept now that an ink is sampled across a
// whole card: a band pattern needs enough squares to read as bands and not enough to alias.
const UpgradeInkSize = formArtSize

// upgradeArt is where an upgrade's ink comes from when it is a *picture*: an assets.LoadImageData
// key, never a path, so a file can be refiled without touching this.
//
// **The wildcard is the only entry and that is the design** — see the file comment. Everything else
// is a flat tint and is generated from upgradeTint below, which is what a placeholder colour should
// cost: a line, not a PNG somebody has to draw before the mechanic can be looked at.
var upgradeArt = map[Upgrade]string{
	UpgradeWild: "wildcardupgrade_png",
}

// upgradeTint is the flat colour an upgrade washes a card in, for the nine that are not a picture.
//
// **Every upgrade in Upgrades() must be in exactly one of these two maps**, which
// TestEveryUpgradeHasAnInk holds. An upgrade in neither would wash a card in nothing and be
// invisible, which is the exact failure the whole idea exists to fix.
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
}

// upgradeSheen is the two metals, which are a *gradient* rather than a flat colour.
//
// **A flat gold on an off-white card is cream, at any wash strength** *(2026-09-09)*. That is not a
// tuning problem: the card's surface is already a pale warm neutral, so pulling it toward a pale
// warm yellow moves the hue a little and the lightness not at all, and a card that reads as slightly
// warmer paper is not a gold card. What makes metal read as metal is a **sheen** — a light running
// across it — and the ink mechanism already carries a picture for the wildcard, so a generated
// gradient costs a function rather than a second path.
//
// **The dark end is what does the work.** A metal band that only brightens has nowhere to go on a
// light card; one that darkens on both shoulders puts a real edge on the card, and the eye reads the
// pair as a highlight travelling over a surface.
//
// The eight flat tints above are deliberately *not* sheened: they are placeholders, and a
// placeholder that looks finished is one nobody replaces.
var upgradeSheen = map[Upgrade]color.RGBA{
	UpgradeGolden: {R: 226, G: 176, B: 46, A: 255},
	UpgradeSilver: {R: 214, G: 222, B: 236, A: 255},
}

// upgradeInkCache holds the decoded and generated inks. Decoding a PNG is not a per-card operation,
// let alone a per-frame one — the same argument artCache is under.
var upgradeInkCache = map[Upgrade]*image.RGBA{}

// UpgradeInk is the colour source an upgrade washes with, as a premultiplied RGBA square of
// UpgradeInkSize. It returns nil for UpgradeNone, which is what a caller checks to leave a card
// alone.
//
// **It is handed out whole rather than resized.** The caller samples it across the card's own
// rectangle, so a flat ink gives a flat wash and a banded one gives bands — which is what lets one
// path draw a gold card and a rainbow one.
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
		if metal, ok := upgradeSheen[u]; ok {
			out = sheenInk(metal)
			break
		}
		tint, ok := upgradeTint[u]
		if !ok {
			log.Fatalf("upgrade %q has neither art, a sheen nor a tint", u)
		}
		out = flatInk(tint)
	}
	upgradeInkCache[u] = out
	return out
}

// flatInk is one colour as an ink square, which is what makes a plain tint indistinguishable from a
// picture at the sampling site.
func flatInk(c color.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, UpgradeInkSize, UpgradeInkSize))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return out
}

// sheenInk is one colour as a diagonal band of light, which is what makes a metal read as metal
// rather than as tinted paper. See upgradeSheen.
//
// **The diagonal, not a row or a column.** A horizontal band would run along the card's text lines
// and read as a highlighter; a vertical one would line up with the left column. A diagonal crosses
// both and is the direction everything else in the game is lit from — see systems.BevelEdges.
func sheenInk(c color.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, UpgradeInkSize, UpgradeInkSize))
	last := 2 * (UpgradeInkSize - 1)
	for y := 0; y < UpgradeInkSize; y++ {
		for x := 0; x < UpgradeInkSize; x++ {
			// 0 at the top-left corner, 1 at the bottom-right, so the band runs corner to corner.
			at := float64(x+y) / float64(last)
			out.SetRGBA(x, y, alongSheen(c, at))
		}
	}
	return out
}

// alongSheen is the metal's colour a fraction of the way across the band: dark at both shoulders,
// the named colour at the crest, and a little past it into white at the very centre.
func alongSheen(c color.RGBA, at float64) color.RGBA {
	// A raised cosine, so the crest is broad and the shoulders fall away smoothly. A linear ramp
	// gives a visible crease down the middle of every card.
	lit := 0.5 - 0.5*math.Cos(2*math.Pi*at)

	dark := ColorAtStrength(c, sheenDarkPct)
	light := ColorToward(c, color.RGBA{R: 255, G: 255, B: 255, A: 255}, sheenLightToward)

	mix := func(d, l uint8) uint8 {
		return uint8(float64(d) + (float64(l)-float64(d))*lit)
	}
	return color.RGBA{R: mix(dark.R, light.R), G: mix(dark.G, light.G), B: mix(dark.B, light.B), A: 255}
}

// The two ends of a metal's band. **The dark end is the far one on purpose** — see upgradeSheen:
// a band that only brightens has nowhere to go on a light card.
const (
	sheenDarkPct     = 52
	sheenLightToward = 34
)

// decodeInk reads an authored ink out of the embedded assets and holds it to the one size every
// caller assumes.
func decodeInk(key string) *image.RGBA {
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

	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	return out
}

// UpgradeTint is the flat colour an upgrade is identified by, for a caller that wants the colour
// without the ink square UpgradeInk hands out.
//
// **It exists so a signal on the combat screen is drawn in the colour of the rider that threw it**
// *(2026-09-10)* — see screens.cardSignal. A firework in a colour of its own would be a second
// vocabulary for something the card face already says, and the two would drift the first time a
// placeholder tint was retuned.
//
// **The metals answer with their sheen's own colour**, since those two are pictures rather than
// flat tints and the sheen is what the card is washed in. UpgradeNone and anything unlisted answer
// with a zero colour, which a caller checks the same way it checks UpgradeInk for nil.
func UpgradeTint(u Upgrade) color.RGBA {
	if c, ok := upgradeSheen[u]; ok {
		return c
	}
	return upgradeTint[u]
}
