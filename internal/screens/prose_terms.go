package screens

// **The working under a blow: what each card was worth, and which ring priced it.**
//
// The fight log printed the total and none of the arithmetic — `(144 x 1.9 = 273)` — so a
// multiplier read as a number the game had decided rather than one the player had built. The hand
// dialog spells the sum out at the size of the screen while the blow lands, and then it is gone.
// These are the same figures written as lines that keep: what the run's account is *for* is being
// read back after the fight, which is the one thing the dialog cannot do. *(2026-09-02)*
//
// **Every figure comes off the event and nothing here multiplies, adds or rounds.** `HandAmounts`,
// `HandRingScale`, `HandLanding` and `HandGrown` are all filled by the resolver — see
// combat.Event, where each says why it is on the event rather than being re-derived. This is a
// second *drawing* of one event, exactly as combat_mathbox.go is, and it is under the same rule: a
// figure it wanted that the event does not carry goes on the event.

import (
	"fmt"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// termIndent is how far a term is set in from a row's left edge, so the working reads as something
// underneath the blow rather than as more lines of the round.
const termIndent = 24

// handTermLines is one line per landing of a hand, in the order the sum counts them.
//
// **A term is a landing, not a card** — an echoed card seats the same index two or three times
// with a figure each — which is why the card is named on every line rather than only the first: a
// player reading back a Three of a Kind wants to see Cut three times, not one Cut and two orphan
// numbers.
//
// `played` is the side's resolved actions in order, which is what `HandCards` indexes. A hand
// naming an action the walk did not see writes no line rather than guessing at one; that cannot
// happen from a resolved round and is checked because the alternative is a panic in a panel.
func (s *CombatScene) handTermLines(e combat.Event, played []combat.Card) []session.LedgerLine {
	if e.HandCardCount <= 0 {
		return nil
	}

	rings := s.wornBy(e.Side)
	out := make([]session.LedgerLine, 0, e.HandCardCount+1)

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		idx := e.HandCards[i]
		if idx < 0 || idx >= len(played) {
			continue
		}
		card := played[idx]
		ink := elementInk(card.Element)

		runs := []session.LedgerRun{
			{Text: fmt.Sprintf("%-14s", termCardName(card)), Ink: ink},
			{Text: fmt.Sprintf("%4d", termBase(e, i)), Ink: ink},
		}
		runs = append(runs, termNotes(e, i, rings)...)

		out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Runs: runs})
	}

	// **The sum, under the terms it adds up, in the figures the hand dialog flew into place.** It
	// is the last line rather than the first because that is the order the arithmetic happens in
	// and the order the dialog acts it out in: the cards, then what they came to.
	out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Runs: handMathRuns(e, played)})
	return out
}

// termBase is the figure a term's own card was worth **before its rings** — the number the ring's
// multiplier is written beside, so the two together read as the arithmetic that was done.
//
// **It falls back to the landed figure** when the event carries no base for the term, which is what
// an event built by an older build or by hand looks like. A term with no figure at all would be a
// line of working with a hole in it.
func termBase(e combat.Event, term int) int {
	if b := e.HandCardBase[term]; b > 0 {
		return b
	}
	return e.HandAmounts[term]
}

// termCardName is what a landing is called on its own line: the card, and its colour when it has
// one. **Not cardPhrase**, which writes a clause for a sentence — "attacks with a fire cut" — where
// this is a label in a column of figures.
func termCardName(c combat.Card) string {
	name := combat.ConceptOf(c.Concept).Label
	if c.Element == combat.Basic {
		return name
	}
	return name + " (" + lower(c.Element.String()) + ")"
}

// termNotes is what the rings did to one term: the landings they bought, then the figures they
// priced it at, in worn order — which is firing order.
//
// **A landing and a multiplier are said differently because they are different things.** An echo
// ring buys a *term* and contributes no figure, so a line reading `x Echo 1x` would credit it with
// arithmetic it did not do; see combat.Event.HandLanding, which is a separate array for exactly
// that reason.
//
// **A ring firing at the identity still fired.** A fresh Enflamed is 1x and is written, on
// ringNote's rule: leaving it out is how a growing ring's climb off 1x becomes invisible.
func termNotes(e combat.Event, term int, rings []combat.WornRing) []session.LedgerRun {
	var notes []session.LedgerRun

	for seat := range e.HandLanding[term] {
		if e.HandLanding[term][seat] {
			notes = append(notes, session.LedgerRun{
				Text: "  + " + ringName(rings, seat) + " lands it again",
				Ink:  session.InkRing,
			})
		}
	}

	for seat, pct := range e.HandRingScale[term] {
		if pct <= 0 {
			continue
		}
		note := "  x " + ringName(rings, seat) + " " + handMultiplierText(pct) + "x"

		// **What the ring stood at after this term**, and only when it moved. A growing ring is
		// the one case where the same ring prices two terms of one blow differently, and the
		// player watching it climb during the blow has nothing to read it off afterwards.
		if grown := e.HandGrown[term][seat]; term > 0 && grown != e.HandGrown[term-1][seat] {
			note += fmt.Sprintf(" (grown %d)", grown)
		}
		notes = append(notes, session.LedgerRun{Text: note, Ink: session.InkRing})
	}

	return notes
}

// handMathRuns is the blow written out as the sum it is, in the colours the hand dialog uses:
// `10 + 10 + (10 x 2) x 2.5 = 100`.
//
// **A ring's figure stays with the term it priced**, in brackets, rather than being folded into the
// term or hung on the end of the whole sum. Folding it in was what the line did until 2026-09-02
// and it hid the ring; hanging it on the end would read as multiplying every term, which is not
// what happened and does not come to the total.
//
// **Every figure comes off the event.** Base, Multiplier, the per-term amounts and the total are
// all the resolver's, so the line cannot claim a sum the round did not use.
func handMathRuns(e combat.Event, played []combat.Card) []session.LedgerRun {
	var runs []session.LedgerRun

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		if len(runs) > 0 {
			runs = append(runs, session.LedgerRun{Text: " + "})
		}

		ink := ""
		if idx := e.HandCards[i]; idx >= 0 && idx < len(played) {
			ink = elementInk(played[idx].Element)
		}

		base, scales := termBase(e, i), ringFactors(e, i)
		if len(scales) == 0 {
			runs = append(runs, session.LedgerRun{Text: strconv.Itoa(base), Ink: ink})
			continue
		}

		runs = append(runs, session.LedgerRun{Text: "(" + strconv.Itoa(base), Ink: ink})
		for _, pct := range scales {
			runs = append(runs, session.LedgerRun{
				Text: " x " + handMultiplierText(pct), Ink: session.InkRing,
			})
		}
		runs = append(runs, session.LedgerRun{Text: ")", Ink: ink})
	}

	// A blow whose event carries no terms still has its two figures. Nothing produces one today;
	// saying the sum it did is better than a line reading `x 1.5 = 30` with nothing in front.
	if len(runs) == 0 {
		runs = append(runs, session.LedgerRun{Text: strconv.Itoa(e.Base)})
	}

	runs = append(runs,
		// **No mark.** An underline under a multiplier in the middle of a sum reads as a
		// typesetting accident; the whole panel is bold already, and the figure's place in the
		// line is what says what it is.
		session.LedgerRun{Text: " x " + handMultiplierText(e.Multiplier), Ink: session.InkHand},
		session.LedgerRun{Text: " = "},
		session.LedgerRun{Text: strconv.Itoa(e.Amount), Ink: session.InkTotal},
	)
	return runs
}

// ringFactors is every ring multiplier that priced one term, in worn order — which is firing order.
//
// **A ring firing at the identity still fired.** A fresh Enflamed is 1x and is written, on
// ringNote's rule: leaving it out is how a growing ring's climb off 1x becomes invisible.
func ringFactors(e combat.Event, term int) []int {
	var out []int
	for _, pct := range e.HandRingScale[term] {
		if pct > 0 {
			out = append(out, pct)
		}
	}
	return out
}

// ringName is the ring on a worn seat. **A seat the wearer does not have is named rather than
// blank** — a saved account has to read as something, and "a ring" is honest where an empty gap is
// a line the player would read as a bug.
func ringName(rings []combat.WornRing, seat int) string {
	if seat < 0 || seat >= len(rings) {
		return "a ring"
	}
	return combat.RingOf(rings[seat].Ring).Name
}

// wornBy is what a side is wearing, in worn order. The opponent wears nothing today — creatures
// have no fingers — so this is the player's row in every case that matters, and it is asked by
// side rather than assumed so that the day one does, the account says which ring.
func (s *CombatScene) wornBy(side combat.Side) []combat.WornRing {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c == nil {
		return nil
	}
	return c.Duelist.WornRings()
}
