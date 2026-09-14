package screens

// **The working under a blow: what each card was worth, and which relic priced it.**
//
// The fight log printed the total and none of the arithmetic — `(144 x 1.9 = 273)` — so a
// multiplier read as a number the game had decided rather than one the player had built. The hand
// dialog spells the sum out at the size of the screen while the blow lands, and then it is gone.
// These are the same figures written as lines that keep: what the run's account is *for* is being
// read back after the fight, which is the one thing the dialog cannot do. *(2026-09-02)*
//
// **Every figure comes off the event and nothing here multiplies, adds or rounds.** `HandAmounts`,
// `HandRelicScale`, `HandLanding` and `HandGrown` are all filled by the resolver — see
// combat.Event, where each says why it is on the event rather than being re-derived. This is a
// second *drawing* of one event, exactly as combat_mathbox.go is, and it is under the same rule: a
// figure it wanted that the event does not carry goes on the event.

import (
	"fmt"
	"strconv"
	"strings"

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

	relics := s.wornBy(e.Side)
	out := make([]session.LedgerLine, 0, e.HandCardCount+1)

	// **What raised the DMG comes before the cards it raised**, because that is the order the
	// arithmetic happens in: the rung is read, the duelist swings bigger, and only then is there a
	// term to write. See handDMGLines.
	out = append(out, handDMGLines(e, relics)...)

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		idx := e.HandCards[i]
		if idx < 0 || idx >= len(played) {
			continue
		}
		card := played[idx]
		ink := elementInk(card.Element)

		spans := []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", termCardName(card)), Ink: ink},
			{Text: fmt.Sprintf("%4d", termBase(e, i)), Ink: ink},
		}
		spans = append(spans, termNotes(e, i, relics)...)

		out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Spans: spans})
	}

	// **Then the flat terms, in the order the sum adds them.** They are the same three the hand
	// dialog draws after the cards, and leaving them out is what made this panel print a sum that
	// did not come to its own total. See flatTermLines.
	out = append(out, flatTermLines(e, relics)...)

	// **The sum, under the terms it adds up, in the figures the hand dialog flew into place.** It
	// is the last line rather than the first because that is the order the arithmetic happens in
	// and the order the dialog acts it out in: the cards, then what they came to.
	out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Spans: handMathSpans(e, played)})
	return out
}

// flatTermLines is the working under the three terms no card paid: the rung's own, the cards the
// turn kept back, and the purse. Each is one line, named for the relic that put it there.
//
// **They were missing until 2026-09-14 and the panel was wrong because of it.** The card loop
// above walks `HandCardCount`, so a blow whose Base includes a relic's flat term printed
// `5 + 10 x 1 = 37` — a sum three short of its own answer, in the one panel whose job is to say
// where a figure came from. The hand dialog has drawn all three since they existed; this is the
// same list written as lines that keep. See combat_mathbox.go's mathScript, which is the other
// drawing of it.
//
// **A count goes beside the figure wherever there is one to give** *(owner's call, 2026-09-14)*.
// `Jar of Ice (4 cards)  20` can be checked against the hand that was being held; a bare 20 is a
// number the player has to take on trust three fights later, when the hand is long gone.
func flatTermLines(e combat.Event, relics []combat.WornRelic) []session.LedgerLine {
	var out []session.LedgerLine

	if e.HeldBonus != 0 {
		out = append(out, flatTerm(relicNames(relics, e.HeldBonusSeats),
			cardCount(e.HeldBonusCards)+" kept back", e.HeldBonus))
	}
	if e.VitaeBonus != 0 {
		out = append(out, flatTerm(relicNames(relics, e.VitaeBonusSeats), "the purse", e.VitaeBonus))
	}
	return out
}

// handDMGLines is what a relic did to the DMG this blow was swung at, written above the terms it
// moved.
//
// **It is not a term and must never be written as one** *(owner's call, 2026-09-14)*. A rung relic
// used to add a flat figure to the sum; what it does now is raise the duelist's DMG for the length
// of one blow, so a Twinned Ring on a duelist of 14 makes a Pair swing at 16 and every card in it
// grows by its own multiplier. That figure is already inside each term below, which is why this
// line carries no figure in the sum's column: it says where the bigger terms came from.
//
// **Without it the relic would be invisible.** The one thing a relic may never be is folded into a
// number the game already shows — a player whose cards quietly got bigger has no way to tell a
// relic from a better hand.
func handDMGLines(e combat.Event, relics []combat.WornRelic) []session.LedgerLine {
	if e.HandBonus == 0 {
		return nil
	}
	return []session.LedgerLine{{Voice: session.VoiceTerm, Spans: []session.LedgerSpan{
		{
			Text: fmt.Sprintf("%-14s", relicNames(relics, e.HandBonusSeats)+" ("+handTitle(e)+")"),
			Ink:  session.InkRelic,
		},
		{Text: fmt.Sprintf("+%d DMG", e.HandBonus), Ink: session.InkRelic},
	}}}
}

// flatTerm is one of those lines: the relic, what it counted, and what it paid.
//
// **The name column is the card terms' column**, so a flat term lands its figure in the same place
// a card's does and the working reads as one column of figures rather than as two lists.
// **The relic's name takes the relic ink and the figure does not**, which is the split the dialog
// makes for the same reason: a relic put the term in the sum, but the term is the hand paying
// rather than a number a relic moved on a card. See mathScript, which draws it in the ground's ink.
func flatTerm(name, note string, amount int) session.LedgerLine {
	return session.LedgerLine{Voice: session.VoiceTerm, Spans: []session.LedgerSpan{
		{Text: fmt.Sprintf("%-14s", name+" ("+note+")"), Ink: session.InkRelic},
		{Text: fmt.Sprintf("%4d", amount)},
	}}
}

// cardCount is "1 card" or "4 cards" — the same pluralising shieldCount does for shields, and the
// one place this noun is counted.
func cardCount(n int) string {
	if n == 1 {
		return "1 card"
	}
	return strconv.Itoa(n) + " cards"
}

// relicNames is every relic that paid a term, in worn order — which is firing order.
//
// **All of them, not the leftmost.** The dialog flies one figure out of one card and has to pick,
// which is what `handBonusSeat` is; a line has room to say that two jars paid, and a line naming
// one of two would be wrong about the half it left out.
func relicNames(relics []combat.WornRelic, seats [combat.MaxWornRelics]bool) string {
	var named []string
	for seat, paid := range seats {
		if paid {
			named = append(named, relicName(relics, seat))
		}
	}
	if len(named) == 0 {
		return "a relic"
	}
	return strings.Join(named, " + ")
}

// termBase is the figure a term's own card was worth **before its relics** — the number the relic's
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

// termCardName is what a landing is called on its own line: the card, and its color when it has
// one. **Not cardPhrase**, which writes a clause for a sentence — "attacks with a fire cut" — where
// this is a label in a column of figures.
func termCardName(c combat.Card) string {
	name := combat.ConceptOf(c.Concept).Label
	if c.Element == combat.Basic {
		return name
	}
	return name + " (" + lower(c.Element.String()) + ")"
}

// termNotes is what the relics did to one term: the landings they bought, then the figures they
// priced it at, in worn order — which is firing order.
//
// **A landing and a multiplier are said differently because they are different things.** An echo
// relic buys a *term* and contributes no figure, so a line reading `x Echo 1x` would credit it with
// arithmetic it did not do; see combat.Event.HandLanding, which is a separate array for exactly
// that reason.
//
// **A relic firing at the identity still fired.** A fresh Enflamed is 1x and is written, on
// relicNote's rule: leaving it out is how a growing relic's climb off 1x becomes invisible.
func termNotes(e combat.Event, term int, relics []combat.WornRelic) []session.LedgerSpan {
	var notes []session.LedgerSpan

	for seat := range e.HandLanding[term] {
		if e.HandLanding[term][seat] {
			notes = append(notes, session.LedgerSpan{
				Text: "  + " + relicName(relics, seat) + " lands it again",
				Ink:  session.InkRelic,
			})
		}
	}

	for seat, pct := range e.HandRelicScale[term] {
		if pct <= 0 {
			continue
		}
		note := "  x " + relicName(relics, seat) + " " + handMultiplierText(pct) + "x"

		// **What the relic stood at after this term**, and only when it moved. A growing relic is
		// the one case where the same relic prices two terms of one blow differently, and the
		// player watching it climb during the blow has nothing to read it off afterwards.
		if grown := e.HandGrown[term][seat]; term > 0 && grown != e.HandGrown[term-1][seat] {
			note += fmt.Sprintf(" (grown %d)", grown)
		}
		notes = append(notes, session.LedgerSpan{Text: note, Ink: session.InkRelic})
	}

	return notes
}

// handMathSpans is the blow written out as the sum it is, in the colors the hand dialog uses:
// `10 + 10 + (10 x 2) x 2.5 = 100`.
//
// **A relic's figure stays with the term it priced**, in brackets, rather than being folded into the
// term or hung on the end of the whole sum. Folding it in was what the line did until 2026-09-02
// and it hid the relic; hanging it on the end would read as multiplying every term, which is not
// what happened and does not come to the total.
//
// **Every figure comes off the event.** Base, Multiplier, the per-term amounts and the total are
// all the resolver's, so the line cannot claim a sum the round did not use.
func handMathSpans(e combat.Event, played []combat.Card) []session.LedgerSpan {
	var spans []session.LedgerSpan

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		if len(spans) > 0 {
			spans = append(spans, session.LedgerSpan{Text: " + "})
		}

		ink := ""
		if idx := e.HandCards[i]; idx >= 0 && idx < len(played) {
			ink = elementInk(played[idx].Element)
		}

		base, scales := termBase(e, i), relicFactors(e, i)
		if len(scales) == 0 {
			spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(base), Ink: ink})
			continue
		}

		spans = append(spans, session.LedgerSpan{Text: "(" + strconv.Itoa(base), Ink: ink})
		for _, pct := range scales {
			spans = append(spans, session.LedgerSpan{
				Text: " x " + handMultiplierText(pct), Ink: session.InkRelic,
			})
		}
		spans = append(spans, session.LedgerSpan{Text: ")", Ink: ink})
	}

	// **The flat terms, after the cards and before the multiplier**, which is where the resolver
	// adds them into Base — see combat.Event.HeldBonus. Written in no ink at all, the dialog's
	// ground: a relic put them in the sum and the term is still the hand paying.
	//
	// **HandBonus is not among them.** It is base damage rather than a term, so it is already
	// inside every figure above — see handDMGLines.
	for _, flat := range []int{e.HeldBonus, e.VitaeBonus} {
		if flat == 0 {
			continue
		}
		if len(spans) > 0 {
			spans = append(spans, session.LedgerSpan{Text: " + "})
		}
		spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(flat)})
	}

	// A blow whose event carries no terms still has its two figures. Nothing produces one today;
	// saying the sum it did is better than a line reading `x 1.5 = 30` with nothing in front.
	if len(spans) == 0 {
		spans = append(spans, session.LedgerSpan{Text: strconv.Itoa(e.Base)})
	}

	spans = append(spans,
		// **No mark.** An underline under a multiplier in the middle of a sum reads as a
		// typesetting accident; the whole panel is bold already, and the figure's place in the
		// line is what says what it is.
		session.LedgerSpan{Text: " x " + handMultiplierText(e.Multiplier), Ink: session.InkHand},
	)

	// **A rung relic is a second multiplier, after the hand's own** *(combat.Event.HandScale)*, so
	// it is a second `x` on the line rather than a bigger figure in the first — the same rule the
	// dialog draws it under, and the reason `Multiplier` still reads as the rung the player built.
	if e.HandScale != 0 && e.HandScale != 100 {
		spans = append(spans, session.LedgerSpan{
			Text: " x " + handMultiplierText(e.HandScale), Ink: session.InkRelic,
		})
	}

	spans = append(spans,
		session.LedgerSpan{Text: " = "},
		session.LedgerSpan{Text: strconv.Itoa(e.Amount), Ink: session.InkTotal},
	)
	return spans
}

// relicFactors is every relic multiplier that priced one term, in worn order — which is firing order.
//
// **A relic firing at the identity still fired.** A fresh Enflamed is 1x and is written, on
// relicNote's rule: leaving it out is how a growing relic's climb off 1x becomes invisible.
func relicFactors(e combat.Event, term int) []int {
	var out []int
	for _, pct := range e.HandRelicScale[term] {
		if pct > 0 {
			out = append(out, pct)
		}
	}
	return out
}

// relicName is the relic on a worn seat. **A seat the wearer does not have is named rather than
// blank** — a saved account has to read as something, and "a relic" is honest where an empty gap is
// a line the player would read as a bug.
func relicName(relics []combat.WornRelic, seat int) string {
	if seat < 0 || seat >= len(relics) {
		return "a relic"
	}
	return combat.RelicOf(relics[seat].Relic).Name
}

// wornBy is what a side is wearing, in worn order. The opponent wears nothing today — creatures
// have no fingers — so this is the player's row in every case that matters, and it is asked by
// side rather than assumed so that the day one does, the account says which relic.
func (s *CombatScene) wornBy(side combat.Side) []combat.WornRelic {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c == nil {
		return nil
	}
	return c.Duelist.WornRelics()
}
