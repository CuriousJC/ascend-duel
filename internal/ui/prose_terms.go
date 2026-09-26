package ui

// **The working under a blow, read off the event and written down as records.**
//
// The hand dialog works every hit out at the size of the screen while it lands, and then it is
// gone. These are the same figures kept: each hit's card, which relic priced it, what raised the
// DMG, the multipliers and what the hit came to. What the run's account is *for* is being read back
// after the fight, which is the one thing the dialog cannot do.
//
// **Every figure comes off the event and nothing here multiplies, adds or rounds.** `HandAmounts`,
// `HitAmounts`, `HandRelicScale`, `HandLanding` and `HandGrown` are all filled by the resolver — see
// combat.Event, where each says why it is on the event rather than being re-derived. This is a
// second *reading* of one event, exactly as combat_mathbox.go is, and it is under the same rule: a
// figure it wanted that the event does not carry goes on the event.
//
// **It produces records rather than sentences**, so the words can be decided when the panel draws
// them — see ledger_prose.go, and session/record.go for the argument.

import (
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// termIndent is how far a term is set in from a row's left edge, so the working reads as something
// underneath the blow rather than as more lines of the round.
const termIndent = 24

// HandTermRecords is the whole working under one blow, in the order the arithmetic happens in:
// what raised the DMG, a line per hit, and the total.
//
// `played` is the side's resolved actions in order, which is what `HandCards` indexes. A hand
// naming an action the walk did not see writes no line rather than guessing at one; that cannot
// happen from a resolved round and is checked because the alternative is a panic in a panel.
func HandTermRecords(e combat.Event, relics []combat.WornRelic, played []combat.Card) []session.LedgerRecord {
	if e.HandCardCount <= 0 {
		return nil
	}

	out := make([]session.LedgerRecord, 0, e.HandCardCount+4)

	// **What raised the DMG comes before the hits it raised**, because that is the order the
	// arithmetic happens in: the rung is read, the duelist swings bigger, and only then is there a
	// hit to write.
	out = append(out, handDMGRecords(e, relics)...)

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		idx := e.HandCards[i]
		if idx < 0 || idx >= len(played) {
			continue
		}
		out = append(out, hitRecord(e, i, played[idx], relics))
	}

	return append(out, session.LedgerRecord{
		Kind: session.KindTerm, Role: session.RoleTotal, Total: e.Amount,
	})
}

// hitRecord is one hit written out: its card, its arithmetic from the card's term to the figure,
// and what each relic did to it. **Its Hit is counted from one**, which is how the hit's outcomes
// find this line — see session.LedgerRecord.Hit.
func hitRecord(e combat.Event, i int, card combat.Card, relics []combat.WornRelic) session.LedgerRecord {
	term := session.LedgerSum{Scales: relicFactors(e, i), Element: ElementName(card.Element)}
	term.PlayAdd = e.HandPlayAdd[i]
	if pct := e.HandPlayPct[i]; pct != 0 && pct != 100 {
		term.PlayPct = pct
	}
	if dmg, pct, ok := e.TermSplit(i); ok {
		term.Split, term.DMG, term.Weight = true, dmg, pct
	} else {
		term.Base = TermBase(e, i)
	}

	rec := session.LedgerRecord{
		Kind:       session.KindTerm,
		Role:       session.RoleHit,
		Hit:        i + 1,
		Card:       combat.ConceptOf(card.Concept).Label,
		Element:    ElementName(card.Element),
		Factors:    append(termFactors(e, i, relics), playRiderFactors(e, i)...),
		Terms:      []session.LedgerSum{term},
		Multiplier: e.Multiplier,
		HandScale:  e.HandScale,
		Total:      e.HitAmounts[i],
	}
	return rec
}

// handDMGRecords is what the relics did to the DMG this blow was swung at: the rung's raise, the
// cards kept back, then the purse.
//
// **A count goes beside the held cards' figure**: `Jar of Ice (4 cards kept back) +20 DMG` can be
// checked against the hand that was being held; a bare 20 is a number the player has to take on
// trust three fights later, when the hand is long gone.
//
// **It is not a term and must never be written as one.** A rung relic raises the duelist's DMG for
// the length of one blow, so a Twinned Ring on a duelist of 14 makes a Pair swing at 16 and every
// card in it grows by its own multiplier. That figure is already inside each card term, which is
// why this record carries no figure for the hand dialog's column: it says where the bigger terms came from.
//
// **Without it the relic would be invisible**, which is the one thing a relic may never be — a
// player whose cards quietly got bigger has no way to tell a relic from a better hand.
func handDMGRecords(e combat.Event, relics []combat.WornRelic) []session.LedgerRecord {
	var out []session.LedgerRecord
	if e.HandBonus != 0 {
		out = append(out, session.LedgerRecord{
			Kind:   session.KindTerm,
			Role:   session.RoleDMG,
			Relic:  relicNames(relics, e.HandBonusSeats),
			Hand:   HandTitle(e),
			Amount: e.HandBonus,
		})
	}
	if e.HeldDMG != 0 {
		out = append(out, session.LedgerRecord{
			Kind:   session.KindTerm,
			Role:   session.RoleDMG,
			Relic:  relicNames(relics, e.HeldDMGSeats),
			Note:   cardCount(e.HeldDMGCards) + " kept back",
			Amount: e.HeldDMG,
		})
	}
	if e.VitaeDMG != 0 {
		out = append(out, session.LedgerRecord{
			Kind:   session.KindTerm,
			Role:   session.RoleDMG,
			Relic:  relicNames(relics, e.VitaeDMGSeats),
			Note:   "the purse",
			Amount: e.VitaeDMG,
		})
	}
	return out
}

// termFactors is what the relics did to one hit: the landings they bought, then the figures they
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

		// **What the relic stood at after this hit**, and only when it moved. A growing relic is
		// the one case where the same relic prices two hits of one blow differently, and the
		// player watching it climb has nothing to read it off afterwards.
		if grown := e.GrownAt(term, seat); term > 0 && grown != e.GrownAt(term-1, seat) {
			f.Grown = grown
		}
		out = append(out, f)
	}

	return out
}

// playRiderFactors is what the hit's own card's riders did to it, named by the rune that puts each
// rider on a card: the flat DMG first, then the percentage, the order the engine applies them in.
func playRiderFactors(e combat.Event, term int) []session.LedgerFactor {
	var out []session.LedgerFactor
	if add := e.HandPlayAdd[term]; add != 0 {
		out = append(out, session.LedgerFactor{Relic: riderName(combat.RiderDamageOnPlay), Add: add})
	}
	if pct := e.HandPlayPct[term]; pct != 0 && pct != 100 {
		out = append(out, session.LedgerFactor{Relic: riderName(combat.RiderScaleInCombo), Scale: pct})
	}
	return out
}

// riderName is what the account calls a rider: the name of the rune that puts it on a card, since
// that is the name the player bought it under. A rider no rune carries falls back to its rule name.
func riderName(kind combat.RiderKind) string {
	for _, r := range session.Runes() {
		if r.Rider == kind {
			return r.Name
		}
	}
	return kind.String()
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
// **All of them, not the leftmost.** A line has room to say that two jars paid, and a line naming
// one of two would be wrong about the half it left out.
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

// relicFactors is every relic multiplier that priced one hit's card term, in worn order — which is
// firing order. See termFactors on why an identity multiplier is kept.

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
