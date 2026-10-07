// Package carddesc is what a card is called and what it says about itself, in words.
//
// **It exists so a review tool can print the same strings the game shows.** The wording used to
// live entirely in `internal/screens`, which links Ebitengine — so `tools/upgradesheet` kept a
// hand-written snapshot of it and said so, on `tools/cardsheet`'s precedent. That was a fair trade
// while the duplicate was one line; it stopped being one the moment the sheet's whole job became
// "does this card read right", because a page quoting its own wording is the one place a mismatch
// between the tooltip and the card is invisible.
//
// This is the same argument `internal/decks` and `internal/journey` are here for, one axis over: a
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
// **No color, no widths, no line breaks.** This hands back plain strings; the caller decides how
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
// be naming a color that is the absence of one.
//
// **A wildcard is CHROMATIC, not the element it happens to be** *(owner's call, 2026-09-09)*. The
// card still *is* an arcane Skewer — every arcane relic reads it, it is drawn from the arcane row — but the
// title is where the panel says what the player is holding, and what they are holding counts as
// every element. `ARCANE SKEWER` over a line reading `COUNTS AS EVERY ELEMENT` is the panel
// contradicting itself in two lines. See Chromatic.
func Title(c combat.Card) string {
	name := upper(c.Label())
	if word := ElementWord(c); word != "" {
		return word + " " + name
	}
	return name
}

// FormLine is the block's first line: what the card is, and what it costs.
//
// **The form is here because the face stopped saying it** *(2026-09-16)*. A card says its form with
// the mark in its corner, which is the right answer on a card and leaves the player with no way to
// learn what the mark means — the panel is where a picture gets its name. `carddesc` is where the
// naming lives so the review sheets print the word the game does.
//
// **One line rather than two**, because the form and the cost are both one term and a panel of
// one-word lines reads as a list rather than as a card. A formless card — a relic, a fighter —
// says its cost alone rather than the word NONE.
func FormLine(c combat.Card, cost int) string {
	ap := strconv.Itoa(cost) + " AP"
	if word := FormWord(c); word != "" {
		return word + ", " + ap
	}
	return ap
}

// FormWord is the card's form in upper case, or empty for a card that has none.
//
// **Exported for the same reason ElementWord is**: a caller can ask which part of a line is the
// form without re-deriving it. The coloring itself needs nobody to ask — every tooltip line goes
// through `cards.SplitForms`, which matches whole words, so STAB comes out in marble for free.
func FormWord(c combat.Card) string {
	if c.Wild(combat.AxisForm) {
		return Versatile
	}
	if f := c.Form(); f != combat.FormNone {
		return upper(f.String())
	}
	return ""
}

// Chromatic is what a card counting as every element is called.
//
// **A word rather than a color, because the wheel has none left for "all of them"** — see
// CLAUDE.md. Every hue is spoken for, and a CHROMATIC written in one of the five would be claiming
// the one thing the word exists to deny, so it takes the panel's own ink and the *word* is the
// signal.
const Chromatic = "CHROMATIC"

// Versatile is what a card counting as any attack form is called — Chromatic's word on the form
// axis, and for the same reason: the panel says what the card counts as rather than the one form
// it carries, which the corner mark already shows.
const Versatile = "VERSATILE"

// ElementWord is what goes in front of a card's name, or empty for a card with nothing to say about
// color.
//
// **Exported so a caller can ask which part of a title is the element** without re-deriving it.
// The coloring itself needs nobody to ask: `screens.tipLine` runs every tooltip line through
// `cards.ElementSpans`, which matches whole words, so FIRE in `FIRE JAB` comes out in the fire red
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
func Lines(c combat.Card, cost, dmg, scale, rolls int) []string {
	lines := []string{FormLine(c, cost)}
	if effect := EffectLine(c, dmg, scale); effect != "" {
		lines = append(lines, effect)
	}
	return append(lines, RiderLines(c, rolls)...)
}

// EffectLine is the one line saying what the card is worth, in the terms its verb is measured in.
//
// **Three verbs, three units.** An attack is DMG, a shield is a count of blows eaten whole, and a
// defense is a percentage off one blow. A single "amount" line would be the same number meaning
// three different things.
//
// **`scale` reaches the attack line and nothing else**, because no relic moment touches a shield's
// count or a defense's percentage. A scale of zero is read as 100, so a caller that has not thought
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
	}
	return ""
}

// RiderLines is what the card's upgrade adds, one line each — `+10 HEAL WHEN PLAYED`.
//
// **Every rider that happens at a moment says which one in two words: WHEN PLAYED, or WHEN UNPLAYED**
// for one that pays while the card sits in the hand a turn did not play. The relics that read the
// same moment — the jars, the rung relics — use the same two words, so one vocabulary covers both.
//
// **Total over `combat.RiderKinds()`, and TestEveryRiderKindHasTipLines holds it that way.** A rider
// with no line is a rune the player spent whose effect the tooltip does not mention, which is
// the same failure as a rider with no drawing.
//
// **A card carries one**, so this is at most one entry long — except for gold, which is two, because
// its two payouts are mutually exclusive and a single line joining them with "or" reads as a card
// that pays both.
func RiderLines(c combat.Card, rolls int) []string {
	var out []string
	for _, r := range c.RiderList() {
		switch r.Kind {
		case combat.RiderHealOnPlay:
			out = append(out, plus(r.Amount)+" HEAL WHEN PLAYED")
		case combat.RiderShieldOnPlay:
			out = append(out, plus(r.Amount)+" SHIELD WHEN PLAYED")
		case combat.RiderDamageOnPlay:
			out = append(out, plus(r.Amount)+" DMG WHEN PLAYED")
		case combat.RiderDamageInHand:
			out = append(out, plus(r.Amount)+" DMG WHEN UNPLAYED")
		case combat.RiderScaleInHand:
			out = append(out, Multiplier(r.Amount)+" DMG WHEN UNPLAYED")
		case combat.RiderVitaeInHand:
			out = append(out, plus(r.Amount)+" VITAE WHEN UNPLAYED")
		case combat.RiderScaleInCombo:
			out = append(out, Multiplier(r.Amount)+" DMG WHEN PLAYED")
		case combat.RiderWildElement:
			out = append(out, "COUNTS AS EVERY ELEMENT")
		case combat.RiderWildForm:
			out = append(out, "COUNTS AS ANY ATTACK FORM")
		case combat.RiderGolden:
			// **The metal is named first and the odds are the fine print under it** *(owner's call,
			// 2026-09-09)*. The panel opens with what the card *is* — the same line its face carries,
			// in the same color — and only then says what that costs and pays. Two rate lines
			// arriving with nothing over them read as arithmetic about a card whose name the player
			// has to work out from its edge.
			//
			// The odds are the record's; the payouts are combat's constants. Both are read rather
			// than written out, so a retune moves these lines with the rule.
			odds := Odds(r.Amount, rolls, 2)
			out = append(out, Gold+" CARD",
				odds+" WHEN PLAYED: "+plus(combat.LuckDMG)+" DMG",
				odds+" WHEN PLAYED: "+plus(combat.LuckLife)+" MAX LIFE")
		case combat.RiderSilver:
			out = append(out, Silver+" CARD",
				Odds(r.Amount, rolls, 1)+" WHEN PLAYED: "+plus(combat.SilverVitae)+" VITAE")
		}
	}
	return out
}

// Odds writes a gamble's chance as the player reads it — `1 IN 5`, and `2 IN 5` under a relic that
// has doubled the numerator.
//
// **It asks the rules rather than doing the arithmetic**, which is the whole point: `combat.LuckOdds`
// is the same function the roll itself uses, so a printed chance cannot disagree with the die. A
// tooltip that derived its own figure is how a card comes to promise something the resolver does not
// do. `bands` is how many paying outcomes share the die — two for gold, one for silver.
func Odds(amount, rolls, bands int) string {
	num, den := combat.LuckOdds(amount, rolls, bands)
	return strconv.Itoa(num) + " IN " + strconv.Itoa(den)
}

// Fraction writes a percentage as a reduced fraction — 25 as `1/4`, 50 as `1/2`, 30 as `3/10`.
//
// **Every probability in the game is written this way** *(owner's call, 2026-09-18)*. A percentage
// and a fraction would be one fact in two notations, and the player would have to notice they are
// the same number. The fraction wins because the figures that move are *numerators*: a relic that
// doubles a roll turns 1/5 into 2/5, and there is nowhere in `20%` for that to show.
//
// **It is not for a multiplier.** `4x DMG` is a scaling rather than a chance, and `Multiplier` is
// its own function for that reason — a card dealing `4/1 DMG` would be arithmetic pretending to be
// a probability.
func Fraction(pct int) string {
	if pct <= 0 {
		return "0"
	}
	num, den := pct, 100
	for _, d := range []int{2, 5} {
		for num%d == 0 && den%d == 0 {
			num, den = num/d, den/d
		}
	}
	if den == 1 {
		return strconv.Itoa(num)
	}
	return strconv.Itoa(num) + "/" + strconv.Itoa(den)
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

// upper is ASCII upper-casing, which is all the card catalog needs and is what keeps this
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

// Gold and Silver are what the two gambling upgrades are called, on a card and in a panel alike.
//
// **Constants because two packages have to agree on the string.** `internal/screens` colors the
// word by looking for it in the face's text, so a card writing GOLD and a highlight looking for
// GOLDEN would be a word that is never lit — the same trap `carddesc.Chromatic` exists to close.
const (
	Gold   = "GOLD"
	Silver = "SILVER"
)
