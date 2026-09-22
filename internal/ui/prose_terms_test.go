package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// spanText is the line a set of ledger spans writes, joined so a test can read it as one string.
func spanText(spans []session.LedgerSpan) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

// blowWithEveryFlatTerm is a hand of two cards carrying every figure no card pays: the two flat
// terms, a rung relic's raise on DMG, and the seats that paid each. It is the shape that was
// printing a sum short of its own answer.
//
// **The rung relic's 2 is inside the cards' own figures**, not added to Base — it is base damage
// since 2026-09-14 — so the fixture spends it the way the resolver does.
func blowWithEveryFlatTerm() (combat.Event, []combat.Card) {
	e := handEvent("pair", []int{5, 10}, 100, 0)
	e.HandBonus, e.HandBonusSeats = 2, []bool{true}
	e.HeldBonus, e.HeldBonusCards = 20, 4
	e.HeldBonusSeats = []bool{false, true}
	e.VitaeBonus, e.VitaeBonusSeats = 8, []bool{false, false, true}
	e.Base += e.HeldBonus + e.VitaeBonus
	e.Amount = e.Base

	played := []combat.Card{combat.Of(combat.Jab, combat.Lightning), combat.Of(combat.Thrust, combat.Earth)}
	return e, played
}

// TestTheLedgersSumAddsUpToItsOwnTotal is the tripwire this file exists for.
//
// **The panel printed `5 + 10 x 1 = 37` for a blow of 37** — the card loop walks HandCardCount, so
// the three flat terms a relic pays into Base were simply absent, in the one panel whose whole job
// is to say where a figure came from. A sum that does not come to its own total reads as a bug in
// the rules rather than as a hole in the account.
//
// **It evaluates the line rather than comparing it to a string**, so a fourth flat term arriving on
// the event fails here rather than shipping as another silent gap.
func TestTheLedgersSumAddsUpToItsOwnTotal(t *testing.T) {
	e, played := blowWithEveryFlatTerm()

	line := spanText(sumSpans(sumRecord(e, played)))
	sum, total := evaluateSum(t, line)
	if sum != total {
		t.Errorf("the ledger wrote %q, which comes to %d rather than %d", line, sum, total)
	}
	if total != e.Amount {
		t.Errorf("the ledger's total is %d, want the event's %d", total, e.Amount)
	}
}

// TestTheRungRelicsMultiplierIsOnTheLine holds the other half: a relic that scales the hand is a
// second `x`, never a bigger figure in the first — see combat.Event.HandScale.
func TestTheRungRelicsMultiplierIsOnTheLine(t *testing.T) {
	e, played := blowWithEveryFlatTerm()
	e.HandScale, e.HandScaleSeats = 200, []bool{true}
	e.Amount = e.Base * 2

	line := spanText(sumSpans(sumRecord(e, played)))
	if strings.Count(line, " x ") != 2 {
		t.Errorf("the ledger wrote %q, want the hand's multiplier and the relic's as two terms", line)
	}
	if sum, total := evaluateSum(t, line); sum != total {
		t.Errorf("the ledger wrote %q, which comes to %d rather than %d", line, sum, total)
	}
}

// TestAHeldTermCountsTheCardsThatPaidIt. **The count is the point of the line** *(owner's call,
// 2026-09-14)*: a bare 20 beside a relic's name is a figure the player cannot check, because the
// hand it was counted over is three fights gone by the time the account is read.
func TestAHeldTermCountsTheCardsThatPaidIt(t *testing.T) {
	e, _ := blowWithEveryFlatTerm()
	relics := []combat.WornRelic{{}, {}, {}}

	flats := flatTermRecords(e, relics)
	if len(flats) != 2 {
		t.Fatalf("the working has %d flat terms, want one each for the held cards and the purse", len(flats))
	}

	held := spanText(termLine(flats[0]).Spans)
	if !strings.Contains(held, "4 cards") {
		t.Errorf("the held term reads %q, want the count of cards that paid it", held)
	}
	if !strings.Contains(held, "20") {
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

	if line := spanText(sumSpans(sumRecord(e, played))); strings.Contains(line, "+ 2 ") {
		t.Errorf("the sum reads %q, and the rung relic's raise is already inside the card terms", line)
	}

	raise, ok := handDMGRecord(e, relics)
	if !ok {
		t.Fatal("a rung relic wrote no line of working, want the one saying what it raised")
	}
	if said := spanText(termLine(raise).Spans); !strings.Contains(said, "+2 DMG") {
		t.Errorf("the raise reads %q, want the DMG it added", said)
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
