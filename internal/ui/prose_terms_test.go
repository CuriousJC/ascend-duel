package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// workingSum is a hit's working read back as one sum — each step's operator and figure, its label
// left out — so a test can check that the column adds up: `20 x 1.5 x 2 = 60`.
func workingSum(rec session.LedgerRecord) string {
	var parts []string
	for _, l := range hitLines(rec, "") {
		if l.Voice != session.VoiceWork || len(l.Spans) < 2 {
			continue
		}
		parts = append(parts, strings.TrimSpace(l.Spans[0].Text+l.Spans[1].Text))
	}
	return strings.Join(parts, " ")
}

// spanText is the line a set of ledger spans writes, joined so a test can read it as one string.
func spanText(spans []session.LedgerSpan) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

// blowWithEveryFlatTerm is a hand of two hits carrying every figure no card pays: a rung relic's
// raise on DMG, the held cards' raise, the purse's raise, and the seats that paid each.
//
// **Every raise is inside the cards' own figures**, not a term — they are base damage — so the
// fixture spends them the way the resolver does.
func blowWithEveryFlatTerm() (combat.Event, []combat.Card) {
	e := handEvent("pair", []int{5, 10}, 100, 0)
	e.HandBonus, e.HandBonusSeats = 2, []bool{true}
	e.HeldDMG, e.HeldDMGCards = 20, 4
	e.HeldDMGSeats = []bool{false, true}
	e.VitaeDMG, e.VitaeDMGSeats = 8, []bool{false, false, true}
	restateHits(&e)

	played := []combat.Card{combat.Of(combat.Jab, combat.Lightning), combat.Of(combat.Thrust, combat.Earth)}
	return e, played
}

// restateHits works a fixture's hits out again after its flat terms or its scale changed, the way
// the resolver does: each hit is its card's term plus every flat term, times the multiplier, times
// the hand scale.
func restateHits(e *combat.Event) {
	e.Amount = 0
	for i := 0; i < e.HandCardCount; i++ {
		hit := e.HandAmounts[i] * e.Multiplier / 100
		if e.HandScale != 0 && e.HandScale != 100 {
			hit = hit * e.HandScale / 100
		}
		e.HitAmounts[i] = hit
		e.Amount += hit
	}
}

// TestEveryHitsLineAddsUpToItsOwnTotal is the tripwire this file exists for: a line of working that
// does not come to its own figure reads as a bug in the rules rather than as a hole in the account.
//
// **It evaluates the line rather than comparing it to a string**, so a third flat term arriving on
// the event fails here rather than shipping as a silent gap.
func TestEveryHitsLineAddsUpToItsOwnTotal(t *testing.T) {
	e, played := blowWithEveryFlatTerm()

	for i := 0; i < e.HandCardCount; i++ {
		line := workingSum(hitRecord(e, i, played[i], nil))
		sum, total := evaluateSum(t, line)
		if sum != total {
			t.Errorf("hit %d wrote %q, which comes to %d rather than %d", i, line, sum, total)
		}
		if total != e.HitAmounts[i] {
			t.Errorf("hit %d's total is %d, want the event's %d", i, total, e.HitAmounts[i])
		}
	}
}

// TestTheWorkingEndsOnEveryHitTogether. The last line is the whole turn, so the column of hit
// figures has an answer under it.
func TestTheWorkingEndsOnEveryHitTogether(t *testing.T) {
	e, played := blowWithEveryFlatTerm()

	recs := HandTermRecords(e, nil, played)
	last := recs[len(recs)-1]
	if last.Role != session.RoleTotal || last.Total != e.Amount {
		t.Errorf("the working ends on a %q of %d, want the total of every hit, %d", last.Role, last.Total, e.Amount)
	}
	hits := 0
	for _, r := range recs {
		if r.Role == session.RoleHit {
			hits++
			if r.Hit != hits {
				t.Errorf("hit line %d says it is hit %d; hits are counted from one", hits, r.Hit)
			}
		}
	}
	if hits != e.HandCardCount {
		t.Errorf("the working has %d hit lines, want one per hit", hits)
	}
}

// TestTheRungRelicsMultiplierIsOnTheLine holds the other half: a relic that scales the hand is a
// second `x`, never a bigger figure in the first — see combat.Event.HandScale — and **two of them
// are two steps**, each beside what it alone multiplied by, before the hand's, which is last.
func TestTheRungRelicsMultiplierIsOnTheLine(t *testing.T) {
	e, played := blowWithEveryFlatTerm()
	e.HandScale, e.HandScaleSeats, e.HandScaleBy = 600, []bool{true, true}, []int{200, 300}
	restateHits(&e)
	relics := []combat.WornRelic{{}, {}}

	rec := hitRecord(e, 0, played[0], relics)
	line := workingSum(rec)
	if sum, total := evaluateSum(t, line); sum != total {
		t.Errorf("the ledger wrote %q, which comes to %d rather than %d", line, sum, total)
	}

	steps := hitLines(rec, "")
	hand := -1
	for i, l := range steps {
		if len(l.Spans) == 3 && l.Spans[1].Ink == session.InkHand {
			hand = i
		}
	}
	// **The hand is the last step before the sum**, and the relics that multiply it come before it.
	if hand != len(steps)-2 || hand < 3 {
		t.Fatalf("the hand's multiplier is not the last step, with two relics before it: %v", steps)
	}
	for k, want := range []int{200, 300} {
		step := steps[hand-2+k]
		if len(step.Spans) != 3 || step.Spans[1].Text != HandMultiplierText(want) || step.Spans[2].Ink != session.InkRelic {
			t.Errorf("hand relic step %d reads %q, want its own x%s labelled with its relic",
				k, step.Text(), HandMultiplierText(want))
		}
	}
}

// TestEveryPlusIntoTheDMGIsWritten is the jump from the duelist's own DMG to the figure the cards
// multiply: **written once, at the top of the turn** — the duelist's own figure, each relic and held
// card that added to it as its own labelled step, and what they came to — and every hit then starts
// from that figure.
func TestEveryPlusIntoTheDMGIsWritten(t *testing.T) {
	e := combat.Event{Kind: combat.KindHand, HandCardCount: 1, Multiplier: 100}
	e.HandDMGBase, e.HeldRiderPct = 20, 100
	e.VitaeDMG, e.VitaeDMGSeats = 200, []bool{true}
	e.HeldDMG, e.HeldDMGCards, e.HeldDMGSeats = 8, 4, []bool{false, true}
	e.DMGRaiseBy = []int{200, 8}
	e.HandDMG = 228
	e.HandCardPct[0], e.HandCardBase[0], e.HandAmounts[0], e.HitAmounts[0] = 50, 114, 114, 114
	e.Amount = 114

	recs := HandTermRecords(e, []combat.WornRelic{{}, {}}, []combat.Card{combat.Of(combat.Jab, combat.Ice)})
	for _, r := range recs {
		if r.Role == session.RoleDMG {
			t.Errorf("a raise is written as a line of its own, want it inside the hit: %+v", r)
		}
	}
	hit := recs[0]
	if text := lineText(hitLines(hit, "")); !strings.Contains(text, "228") || strings.Contains(text, "+ 200") {
		t.Errorf("the hit reads %q, want it to start from the turn's 228 and not build it again", text)
	}
	if sum, total := evaluateSum(t, workingSum(hit)); sum != total {
		t.Errorf("the working %q comes to %d rather than %d", workingSum(hit), sum, total)
	}

	rec, ok := DMGRecord(e, []combat.WornRelic{{}, {}}, session.SideYou, "Duelist")
	if !ok {
		t.Fatal("a turn whose DMG was raised writes no DMG record")
	}
	text := lineText(LedgerLines([]session.LedgerRecord{rec}))
	for _, want := range []string{"Duelist DMG grows for the attack", "20", "+ 200", "+ 8", "= 228", "Total DMG"} {
		if !strings.Contains(text, want) {
			t.Errorf("the turn's DMG reads %q, want %q in it", text, want)
		}
	}

	bare := e
	bare.HeldDMG, bare.VitaeDMG, bare.HandDMG, bare.DMGRaiseBy = 0, 0, 20, nil
	if _, ok := DMGRecord(bare, nil, session.SideYou, "Duelist"); ok {
		t.Error("a turn swung at the duelist's own DMG writes a DMG record, with nothing to explain")
	}
}

// TestEveryStepOfAHitIsLabelled is the point of writing a hit down a column: every figure in it
// says what produced it, so nothing in the working is a number the player has to take on trust.
func TestEveryStepOfAHitIsLabelled(t *testing.T) {
	e, played := blowWithEveryFlatTerm()
	e.HandScale, e.HandScaleSeats, e.HandScaleBy = 200, []bool{true}, []int{200}
	restateHits(&e)

	for _, l := range hitLines(hitRecord(e, 0, played[0], []combat.WornRelic{{}, {}, {}}), "") {
		if l.Voice != session.VoiceWork || strings.HasPrefix(strings.TrimSpace(l.Text()), "=") {
			continue
		}
		if len(l.Spans) != 3 || strings.TrimSpace(l.Spans[2].Text) == "" || l.Spans[2].Col == 0 {
			t.Errorf("the step %q has no label on the label column", l.Text())
		}
	}
}

// TestAHeldTermCountsTheCardsThatPaidIt. **The count is the point of the line** *(owner's call,
// 2026-09-14)*: a bare 20 beside a relic's name is a figure the player cannot check, because the
// hand it was counted over is three fights gone by the time the account is read.
func TestAHeldTermCountsTheCardsThatPaidIt(t *testing.T) {
	e, _ := blowWithEveryFlatTerm()
	relics := []combat.WornRelic{{}, {}, {}}

	raises := handDMGRecords(e, relics)
	if len(raises) != 3 {
		t.Fatalf("the working has %d raises, want the rung's, the held cards' and the purse's", len(raises))
	}

	held := spanText(termLine(raises[1]).Spans)
	if !strings.Contains(held, "4 cards") {
		t.Errorf("the held term reads %q, want the count of cards that paid it", held)
	}
	if !strings.Contains(held, "+20 DMG") {
		t.Errorf("the held term reads %q, want the figure it paid", held)
	}
}

// TestTheRungRelicsRaiseIsSaidButNeverSummed. **A relic that raises DMG is inside every term**, so
// writing it as a term too prints a sum over its own total — which is the mistake the flat terms
// made in the other direction. It still has to be *said*: a relic folded into a figure the game
// already shows is a relic the player cannot tell from a better hand.
func TestTheRungRelicsRaiseIsSaidButNeverSummed(t *testing.T) {
	e, played := blowWithEveryFlatTerm()
	relics := []combat.WornRelic{{}, {}, {}}

	if line := workingSum(hitRecord(e, 0, played[0], relics)); strings.Contains(line, "+ 2 ") {
		t.Errorf("the hit reads %q, and the rung relic's raise is already inside the card term", line)
	}

	raises := handDMGRecords(e, relics)
	if len(raises) != 3 {
		t.Fatalf("the raises wrote %d lines of working, want the rung's, the held cards' and the purse's", len(raises))
	}
	if said := spanText(termLine(raises[0]).Spans); !strings.Contains(said, "+2 DMG") {
		t.Errorf("the rung's raise reads %q, want the DMG it added", said)
	}
	if said := spanText(termLine(raises[2]).Spans); !strings.Contains(said, "the purse") || !strings.Contains(said, "+8 DMG") {
		t.Errorf("the purse's raise reads %q, want it named and the DMG it added", said)
	}
	if line := workingSum(hitRecord(e, 0, played[0], relics)); strings.Contains(line, "+ 8 ") {
		t.Errorf("the hit reads %q, and the purse's raise is already inside the card term", line)
	}
}

// TestOneCardKeptBackIsNotFourCards. The noun is counted in one place; a line reading "1 cards" is
// the kind of thing nobody sees in review and everybody sees in play.
func TestOneCardKeptBackIsNotFourCards(t *testing.T) {
	if got := cardCount(1); got != "1 card" {
		t.Errorf("one card reads %q, want %q", got, "1 card")
	}
	if got := cardCount(3); got != "3 cards" {
		t.Errorf("three cards read %q, want %q", got, "3 cards")
	}
}

// evaluateSum reads a written sum back — `5 + 10 + 2 x 1.5 = 25` — and reports what its terms come
// to and what it claims. Multipliers apply to everything added before them, which is the order the
// resolver works in.
func evaluateSum(t *testing.T, line string) (sum, total int) {
	t.Helper()

	halves := strings.Split(line, " = ")
	if len(halves) != 2 {
		t.Fatalf("the sum %q has no total", line)
	}
	total, err := strconv.Atoi(strings.TrimSpace(halves[1]))
	if err != nil {
		t.Fatalf("the sum %q ends %q, which is not a figure", line, halves[1])
	}

	value, mult := 0.0, false
	// The working writes a relic's own multiplier in brackets on the term it priced; nothing in
	// this file's cases does, and a bracket would need a parser rather than a walk.
	for _, field := range strings.Fields(strings.TrimSpace(halves[0])) {
		switch field {
		case "+":
			mult = false
			continue
		case "x":
			mult = true
			continue
		}
		n, err := strconv.ParseFloat(field, 64)
		if err != nil {
			t.Fatalf("the sum %q holds %q, which is neither a figure nor an operator", line, field)
		}
		if mult {
			value *= n
			continue
		}
		value += n
	}
	return int(value), total
}
