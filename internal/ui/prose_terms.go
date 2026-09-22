package ui

// **The working under a blow, read off the event and written down as records.**
//
// The hand dialog spells a blow's sum out at the size of the screen while it lands, and then it is
// gone. These are the same figures kept: what each card was worth, which relic priced it, and what
// the whole thing came to. What the run's account is *for* is being read back after the fight,
// which is the one thing the dialog cannot do.
//
// **Every figure comes off the event and nothing here multiplies, adds or rounds.** `HandAmounts`,
// `HandRelicScale`, `HandLanding` and `HandGrown` are all filled by the resolver — see
// combat.Event, where each says why it is on the event rather than being re-derived. This is a
// second *reading* of one event, exactly as combat_mathbox.go is, and it is under the same rule: a
// figure it wanted that the event does not carry goes on the event.
//
// **It produces records rather than sentences**, so the words can be decided when the panel draws
// them — see ledger_prose.go, and session/record.go for the argument. What it keeps is the
// knowledge of the event's fat arrays, which belongs beside the helpers that already read them
// rather than in a scene.

import (
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// termIndent is how far a term is set in from a row's left edge, so the working reads as something
// underneath the blow rather than as more lines of the round.
const termIndent = 24

// HandTermRecords is the whole working under one blow, in the order the arithmetic happens in.
//
// `played` is the side's resolved actions in order, which is what `HandCards` indexes. A hand
// naming an action the walk did not see writes no term rather than guessing at one; that cannot
// happen from a resolved round and is checked because the alternative is a panic in a panel.
func HandTermRecords(e combat.Event, relics []combat.WornRelic, played []combat.Card) []session.LedgerRecord {
	if e.HandCardCount <= 0 {
		return nil
	}

	out := make([]session.LedgerRecord, 0, e.HandCardCount+3)

	// **What raised the DMG comes before the cards it raised**, because that is the order the
	// arithmetic happens in: the rung is read, the duelist swings bigger, and only then is there a
	// term to write.
	if r, ok := handDMGRecord(e, relics); ok {
		out = append(out, r)
	}

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		idx := e.HandCards[i]
		if idx < 0 || idx >= len(played) {
			continue
		}
		card := played[idx]
		out = append(out, session.LedgerRecord{
			Kind:    session.KindTerm,
			Role:    session.RoleCard,
			Card:    combat.ConceptOf(card.Concept).Label,
			Element: ElementName(card.Element),
			Base:    TermBase(e, i),
			Factors: termFactors(e, i, relics),
		})
	}

	// **Then the flat terms, in the order the sum adds them.** Leaving them out is what made this
	// panel print a sum that did not come to its own total.
	out = append(out, flatTermRecords(e, relics)...)

	// **The sum, under the terms it adds up.** It is the last line rather than the first because
	// that is the order the arithmetic happens in and the order the dialog acts it out in: the
	// cards, then what they came to.
	out = append(out, sumRecord(e, played))
	return out
}

// handDMGRecord is what a relic did to the DMG this blow was swung at.
//
// **It is not a term and must never be written as one.** A rung relic raises the duelist's DMG for
// the length of one blow, so a Twinned Ring on a duelist of 14 makes a Pair swing at 16 and every
// card in it grows by its own multiplier. That figure is already inside each card term, which is
// why this record carries no figure for the sum's column: it says where the bigger terms came from.
//
// **Without it the relic would be invisible**, which is the one thing a relic may never be — a
// player whose cards quietly got bigger has no way to tell a relic from a better hand.
func handDMGRecord(e combat.Event, relics []combat.WornRelic) (session.LedgerRecord, bool) {
	if e.HandBonus == 0 {
		return session.LedgerRecord{}, false
	}
	return session.LedgerRecord{
		Kind:   session.KindTerm,
		Role:   session.RoleDMG,
		Relic:  relicNames(relics, e.HandBonusSeats),
		Hand:   HandTitle(e),
		Amount: e.HandBonus,
	}, true
}

// flatTermRecords is the working under the two terms no card paid: the cards the turn kept back,
// and the purse. Each names the relic that put it in the sum.
//
// **A count goes beside the figure wherever there is one to give.** `Jar of Ice (4 cards kept
// back) 20` can be checked against the hand that was being held; a bare 20 is a number the player
// has to take on trust three fights later, when the hand is long gone.
func flatTermRecords(e combat.Event, relics []combat.WornRelic) []session.LedgerRecord {
	var out []session.LedgerRecord
	if e.HeldBonus != 0 {
		out = append(out, session.LedgerRecord{
			Kind: session.KindTerm, Role: session.RoleFlat,
			Relic:  relicNames(relics, e.HeldBonusSeats),
			Note:   cardCount(e.HeldBonusCards) + " kept back",
			Amount: e.HeldBonus,
		})
	}
	if e.VitaeBonus != 0 {
		out = append(out, session.LedgerRecord{
			Kind: session.KindTerm, Role: session.RoleFlat,
			Relic:  relicNames(relics, e.VitaeBonusSeats),
			Note:   "the purse",
			Amount: e.VitaeBonus,
		})
	}
	return out
}

// sumRecord is the blow as the sum it is: every landing's figures, the flat terms, the rung's
// multiplier, a rung relic's second one, and the total.
//
// **HandBonus is not among the flats.** It is base damage rather than a term, so it is already
// inside every figure above — see handDMGRecord.
func sumRecord(e combat.Event, played []combat.Card) session.LedgerRecord {
	rec := session.LedgerRecord{
		Kind:       session.KindTerm,
		Role:       session.RoleSum,
		Multiplier: e.Multiplier,
		HandScale:  e.HandScale,
		Base:       e.Base,
		Total:      e.Amount,
	}

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		term := session.LedgerSum{Scales: relicFactors(e, i)}
		if idx := e.HandCards[i]; idx >= 0 && idx < len(played) {
			term.Element = ElementName(played[idx].Element)
		}
		if dmg, pct, ok := e.TermSplit(i); ok {
			term.Split, term.DMG, term.Weight = true, dmg, pct
		} else {
			term.Base = TermBase(e, i)
		}
		rec.Terms = append(rec.Terms, term)
	}

	for _, flat := range []int{e.HeldBonus, e.VitaeBonus} {
		if flat != 0 {
			rec.Flats = append(rec.Flats, flat)
		}
	}
	return rec
}

// termFactors is what the relics did to one term: the landings they bought, then the figures they
// priced it at, in worn order — which is firing order.
//
// **A relic firing at the identity still fired.** A fresh Enflamed is 1x and is written: leaving it
// out is how a growing relic's climb off 1x becomes invisible.
func termFactors(e combat.Event, term int, relics []combat.WornRelic) []session.LedgerFactor {
	var out []session.LedgerFactor

	for seat := range e.HandLanding[term] {
		if e.HandLanding[term][seat] {
			out = append(out, session.LedgerFactor{Relic: relicName(relics, seat), Landed: true})
		}
	}

	for seat, pct := range e.HandRelicScale[term] {
		if pct <= 0 {
			continue
		}
		f := session.LedgerFactor{Relic: relicName(relics, seat), Scale: pct}

		// **What the relic stood at after this term**, and only when it moved. A growing relic is
		// the one case where the same relic prices two terms of one blow differently, and the
		// player watching it climb during the blow has nothing to read it off afterwards.
		if grown := e.GrownAt(term, seat); term > 0 && grown != e.GrownAt(term-1, seat) {
			f.Grown = grown
		}
		out = append(out, f)
	}

	return out
}

// cardCount is "1 card" or "4 cards" — the same pluralising ShieldCount does for shields, and the
// one place this noun is counted.
func cardCount(n int) string {
	if n == 1 {
		return "1 card"
	}
	return strconv.Itoa(n) + " cards"
}

// relicNames is every relic that paid a term, in worn order — which is firing order.
//
// **All of them, not the leftmost.** The dialog flies one figure out of one card and has to pick;
// a line has room to say that two jars paid, and a line naming one of two would be wrong about the
// half it left out.
func relicNames(relics []combat.WornRelic, seats []bool) string {
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

// TermBase is the figure a term's own card was worth **before its relics** — the number the relic's
// multiplier is written beside, so the two together read as the arithmetic that was done.
//
// **It falls back to the landed figure** when the event carries no base for the term, which is what
// an event built by an older build or by hand looks like. A term with no figure at all would be a
// line of working with a hole in it.
func TermBase(e combat.Event, term int) int {
	if b := e.HandCardBase[term]; b > 0 {
		return b
	}
	return e.HandAmounts[term]
}

// relicFactors is every relic multiplier that priced one term, in worn order — which is firing
// order. See termFactors on why an identity multiplier is kept.
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

// ElementName is an element's name, or "" for a card that carries none — which is what keeps a
// basic card out of the element vocabulary rather than naming a color nothing draws.
func ElementName(e combat.Element) string {
	if e == combat.Basic {
		return ""
	}
	return e.String()
}
