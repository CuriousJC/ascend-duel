// Package carddesc is what a card is called and what it says about itself, in words.
//
// **It exists so a review tool can print the same strings the game shows.** The wording used to
// live entirely in `internal/screens`, which links Ebitengine — so `tools/upgradesheet` kept a
// hand-written snapshot of it and said so, on `tools/cardsheet`'s precedent. That was a fair trade
// while the duplicate was one line; it stopped being one the moment the sheet's whole job became
// "does this card read right", because a page quoting its own wording is the one place a mismatch
// between the tooltip and the card is invisible.
//
// This is the same argument `internal/decks` and `internal/pyramid` are here for, one axis over: a
// thing a headless caller needs and a screen must not own.
//
// # What belongs here and what does not
//
// **The tooltip's stat block belongs here**, because every term of it is a fact about a
// `combat.Card` plus the two numbers the holder brings — what it costs them and what they hit for.
// Nothing in it needs a window, a layout or a font.
//
// **The arithmetic does not.** A relic's contribution to a card's damage is `internal/screens`'
// business and stays there: it needs the worn relics, which are a fact about a duelist in a fight
// rather than about a card. `screens.cardTip` calls this for the block and appends its own chain.
//
// **No colour, no widths, no line breaks.** This hands back plain strings; the caller decides how
// they are drawn — which is what lets one of them be a tooltip panel and another a table cell in an
// HTML page.
package carddesc

import (
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Title is the card's name with its element in front of it — `FIRE JAB`.
//
// **Upper case, because the tooltip's title is a label rather than a sentence** and the rest of the
// panel is set in the same clipped register the card faces use.
//
// **An elementless card is just its name.** Every creature card is `basic`, and `BASIC NIP` would
// be naming a colour that is the absence of one.
//
// **A wildcard is CHROMATIC, not the element it happens to be** *(owner's call, 2026-09-09)*. The
// card still *is* an arcane Lunge — it burns as one, it is drawn from the arcane row — but the
// title is where the panel says what the player is holding, and what they are holding counts as
// every element. `ARCANE LUNGE` over a line reading `COUNTS AS EVERY ELEMENT` is the panel
// contradicting itself in two lines. See Chromatic.
func Title(c combat.Card) string {
	name := upper(c.Label())
	if word := ElementWord(c); word != "" {
		return word + " " + name
	}
	return name
}

// Chromatic is what a card counting as every element is called.
//
// **A word rather than a colour, because the wheel has none left for "all of them"** — see
// CLAUDE.md. Every hue is spoken for, and a CHROMATIC written in one of the five would be claiming
// the one thing the word exists to deny, so it takes the panel's own ink and the *word* is the
// signal.
const Chromatic = "CHROMATIC"

// ElementWord is what goes in front of a card's name, or empty for a card with nothing to say about
// colour.
//
// **Exported so a caller can ask which part of a title is the element** without re-deriving it.
// The colouring itself needs nobody to ask: `screens.tipLine` runs every tooltip line through
// `cards.ElementRuns`, which matches whole words, so FIRE in `FIRE JAB` comes out in the fire red
// for free — and CHROMATIC does not, which is the intended answer rather than a gap.
func ElementWord(c combat.Card) string {
	if c.Wild(combat.AxisElement) {
		return Chromatic
	}
	if c.Element == combat.Basic {
		return ""
	}
	return upper(c.Element.String())
}

// Lines is the tooltip's stat block: what the card costs, what it does, and what its upgrade adds.
//
// `cost` is what the holder pays — a discount relic makes that a property of the pairing rather than
// of the card — and `dmg` is the holder's own DMG. **A dmg of zero means nobody is holding it**,
// which is the honest state between fights: a run's stats belong to a fight, so the block says
// `2x DMG` rather than a figure worked out against a strength nobody has yet.
//
// `scale` is every relic that reaches this card's damage, compounded, as a percentage — 100 for a
// bare card. **It is passed in rather than worked out**, because the relics belong to a duelist in a
// fight and this package knows about cards; the caller walks `combat.RelicContributionsAt` and hands
// over the product, so the figure printed here is the engine's rather than a second sum.
func Lines(c combat.Card, cost, dmg, scale int) []string {
	lines := []string{strconv.Itoa(cost) + " AP"}
	if effect := EffectLine(c, dmg, scale); effect != "" {
		lines = append(lines, effect)
	}
	return append(lines, RiderLines(c)...)
}

// EffectLine is the one line saying what the card is worth, in the terms its verb is measured in.
//
// **Three verbs, three units.** An attack is DMG, a shield is a count of blows eaten whole, and a
// defence is a percentage off one blow. A single "amount" line would be the same number meaning
// three different things.
//
// **`scale` reaches the attack line and nothing else**, because no relic moment touches a shield's
// count or a defence's percentage. A scale of zero is read as 100, so a caller that has not thought
// about relics gets the bare card rather than a card worth nothing.
func EffectLine(c combat.Card, dmg, scale int) string {
	amount := c.Amount()
	if scale <= 0 {
		scale = 100
	}
	switch c.Spec().Verb {
	case combat.VerbAttack:
		// **The relics are in the figure**, so the headline number is what the card will actually
		// deal. A block stating the card's bare worth over a chain ending in a bigger number would
		// be two DMG figures on one panel with the wrong one at the top.
		amount = amount * scale / 100
		if dmg > 0 {
			return strconv.Itoa(dmg*amount/100) + " DMG"
		}
		return Multiplier(amount) + " DMG"
	case combat.VerbShield:
		if amount == 1 {
			return "1 SHIELD"
		}
		return strconv.Itoa(amount) + " SHIELDS"
	case combat.VerbDefend:
		return strconv.Itoa(amount) + "% OFF ONE BLOW"
	}
	return ""
}

// RiderLines is what the card's upgrade adds, one line each — `+10 HEAL ON PLAY`.
//
// **Total over `combat.RiderKinds()`, and TestEveryRiderKindHasTipLines holds it that way.** A rider
// with no line is a parasite the player spent whose effect the tooltip does not mention, which is
// the same failure as a rider with no drawing.
//
// **A card carries one**, so this is at most one entry long — except for gold, which is two, because
// its two payouts are mutually exclusive and a single line joining them with "or" reads as a card
// that pays both.
func RiderLines(c combat.Card) []string {
	var out []string
	for _, r := range c.RiderList() {
		switch r.Kind {
		case combat.RiderHealOnPlay:
			out = append(out, plus(r.Amount)+" HEAL ON PLAY")
		case combat.RiderShieldOnPlay:
			out = append(out, plus(r.Amount)+" SHIELD ON PLAY")
		case combat.RiderDamageOnPlay:
			out = append(out, plus(r.Amount)+" DMG ON PLAY")
		case combat.RiderDamageInHand:
			out = append(out, plus(r.Amount)+" DMG IN HAND")
		case combat.RiderScaleInHand:
			out = append(out, Multiplier(r.Amount)+" DMG IN HAND")
		case combat.RiderVitaeInHand:
			out = append(out, plus(r.Amount)+" VITAE IN HAND")
		case combat.RiderScaleInCombo:
			// **"IF IT SCORES", not "ON PLAY".** `Blow.Cards` is the scoring set and a turn can play
			// a card that pays nothing into it — a lone Ward beside a pair. The distinction is the
			// whole of what separates this from damage-on-play, and it is one the player can lose.
			out = append(out, Multiplier(r.Amount)+" DMG IF IT SCORES")
		case combat.RiderWildElement:
			out = append(out, "COUNTS AS EVERY ELEMENT")
		case combat.RiderGolden:
			// **The metal is named first and the odds are the fine print under it** *(owner's call,
			// 2026-09-09)*. The panel opens with what the card *is* — the same line its face carries,
			// in the same colour — and only then says what that costs and pays. Two rate lines
			// arriving with nothing over them read as arithmetic about a card whose name the player
			// has to work out from the wash.
			//
			// The odds are the record's; the payouts are combat's constants. Both are read rather
			// than written out, so a retune moves these lines with the rule.
			out = append(out, Gold+" CARD",
				"1 IN "+strconv.Itoa(r.Amount)+" ON PLAY: "+plus(combat.LuckDMG)+" DMG",
				"1 IN "+strconv.Itoa(r.Amount)+" ON PLAY: "+plus(combat.LuckLife)+" MAX LIFE")
		case combat.RiderSilver:
			out = append(out, Silver+" CARD",
				"1 IN "+strconv.Itoa(r.Amount)+" ON PLAY: "+plus(combat.SilverVitae)+" VITAE")
		}
	}
	return out
}

// Multiplier writes a percentage as a multiplier — 200 as `2X`, 250 as `2.5X`.
//
// **The same rule `screens.multiplierText` follows, in upper case.** Every other line in the block
// is upper case and one lower-case `x` in the middle of a column of them reads as a typo. The
// screen's copy stays where it is and stays lower case: it sets prose, inside sentences, where an
// upper-case X would be the shout.
func Multiplier(amount int) string {
	whole, frac := amount/100, amount%100
	switch {
	case frac == 0:
		return strconv.Itoa(whole) + "X"
	case frac%10 == 0:
		return strconv.Itoa(whole) + "." + strconv.Itoa(frac/10) + "X"
	default:
		return strconv.Itoa(whole) + "." + pad(frac) + "X"
	}
}

func plus(n int) string {
	if n < 0 {
		return strconv.Itoa(n)
	}
	return "+" + strconv.Itoa(n)
}

func pad(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// upper is ASCII upper-casing, which is all the card catalogue needs and is what keeps this
// package free of the unicode tables for a job it does on a dozen known strings.
func upper(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 'a' - 'A'
		}
	}
	return string(out)
}

// FaceLines is what the card's upgrade says on the card itself, one authored line each.
//
// **It is the column version of RiderLines, and both live here on purpose.** The tooltip sets
// sentences in a panel; the face sets labels in a 128-pixel column under the card's own verb, so
// the two cannot share one string — `+10 HEAL ON PLAY` is a sentence and `ON PLAY` over `+10 LIFE`
// is a column. What keeping them in one file buys is that the difference is *readable*: the two
// functions sit a screen apart and a wording that has drifted shows as two lines that no longer
// say the same thing, instead of as a disagreement between a screen and a panel nobody diffs.
//
// **The timing is the heading and the payload sits under it** *(2026-09-09)*. The face already
// carries a figure — `DMG 2X` — so a rider's own figure pressed straight underneath is two numbers
// running together with the thing that separates them arriving last. A heading first makes the
// upgrade read as a new clause, and it puts the word the player is actually learning — whether this
// happens when the card is *played* or while it is merely *held* — at the top of the block.
//
// **What a caller controls is the order, not the breaks.** `cards.WrapText` has set the face one
// word to a line since 2026-09-05 — a figure joined to its unit and nothing else — so a heading of
// two words is two lines whatever this returns, and there is no way to buy it back. The budget is
// what that costs: the band holds seven lines, the card's own verb takes two, and the longest
// upgrade here — `IF IT SCORES` over `DMG 1.5X` — takes four. Gold and silver are the reason the
// odds are not on the face; see below.
//
// **Total over combat.RiderKinds(), like RiderLines.** A rider with no face line is a parasite the
// player spent that the card does not mention — the same failure as a rider with no drawing, and
// the one the wash on its own cannot fix: a colour is what carries across a row of eight cards, and
// the words are what answers "what does that mean" without a hover.
func FaceLines(c combat.Card) []string {
	var out []string
	for _, r := range c.RiderList() {
		switch r.Kind {
		case combat.RiderHealOnPlay:
			out = append(out, onPlay, plus(r.Amount)+" LIFE")
		case combat.RiderShieldOnPlay:
			out = append(out, onPlay, plus(r.Amount)+" "+shieldWord(r.Amount))
		case combat.RiderDamageOnPlay:
			out = append(out, onPlay, plus(r.Amount)+" DMG")
		case combat.RiderDamageInHand:
			out = append(out, inHand, plus(r.Amount)+" DMG")
		case combat.RiderScaleInHand:
			out = append(out, inHand, FaceMultiplier(r.Amount)+" DMG")
		case combat.RiderVitaeInHand:
			out = append(out, inHand, plus(r.Amount)+" VITAE")
		case combat.RiderScaleInCombo:
			// **"IF IT SCORES" rather than "ON PLAY"**, for RiderLines' reason: a turn can play a
			// card that pays nothing into the scoring set, and that distinction is the whole of what
			// separates this rider from damage-on-play.
			out = append(out, scoring, FaceMultiplier(r.Amount)+" DMG")
		case combat.RiderWildElement:
			// **No heading, because there is no moment.** Every other rider happens at a time; this
			// one is something the card permanently *is*, and a timing word over it would be
			// answering a question the card does not raise.
			out = append(out, "ANY ELEMENT")
		case combat.RiderGolden, combat.RiderSilver:
			// **The metals say nothing on the face, and they are the only two that may**
			// *(owner's call, 2026-09-09)*. Every other upgrade is a placeholder tint standing on a
			// full wheel, so its colour cannot be relied on to name it and the words are what a
			// player learns it from — see systems.upgradeTint. Gold and silver are the exception in
			// exactly that respect: systems.upgradeSheen is not a placeholder, because gold and
			// silver are what the mechanic is *called*, and a sheen running across a card is
			// something nothing else on the table does. A card whose picture already says GOLD does
			// not need the word.
			//
			// **The whole gamble is the tooltip's**, where there is room for a sentence — see
			// RiderLines, which names the metal and then prints the odds under it. This is the one
			// place the face is deliberately quieter than the panel, and the sheen is what pays
			// for it.
		}
	}
	return out
}

// The three headings a face may carry. **Constants rather than literals** so the set is countable:
// a fourth moment is a line here, which is where the question "does the player already know this
// word" gets asked, rather than a string typed into one case.
//
// **SCORING is one word where the other two are phrases, and the budget is why.** One word to a
// line makes the tooltip's `IF IT SCORES` three lines, which is a heading longer than the card it
// heads; `ON PLAY` and `IN HAND` cost two each and the band can afford those. It is the register
// the rest of the face is already in — HITS, CUTS, SHIELD — so it does not read as a fourth voice.
const (
	onPlay  = "ON PLAY"
	inHand  = "IN HAND"
	scoring = "SCORING"
)

// FaceMultiplier is Multiplier in the case the card faces are set in — 200 as `2x`.
//
// **Lower case, and that is not a style preference.** `cards.WrapText` keeps a figure on its unit's
// line, and `isFigure` reads a trailing `x` and not a trailing `X` — so `2X DMG` breaks into two
// lines where `2x DMG` stays as one. The card's own multiplier has always been written this way;
// this is the upgrade's line joining it rather than a second convention.
func FaceMultiplier(amount int) string {
	return lower(Multiplier(amount))
}

func lower(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'A' && out[i] <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

// Gold and Silver are what the two gambling upgrades are called, on a card and in a panel alike.
//
// **Constants because two packages have to agree on the string.** `internal/screens` colours the
// word by looking for it in the face's text, so a card writing GOLD and a highlight looking for
// GOLDEN would be a word that is never lit — the same trap `carddesc.Chromatic` exists to close.
const (
	Gold   = "GOLD"
	Silver = "SILVER"
)

func shieldWord(n int) string {
	if n == 1 || n == -1 {
		return "SHIELD"
	}
	return "SHIELDS"
}
