package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The order cards run in along a deck row. Windowless, like the other tests in this
// package — sortPileEntries was pulled out of drawPileGrid precisely so this could exist.

func TestDeckRowRunsFamilyByFamily(t *testing.T) {
	// **One card of every concept the deck holds, rather than one row of the panel** *(2026-08-15)*.
	// It used to take the basic row, which held all twelve concepts back when attacks shipped in a
	// drab variant. They no longer do — the attacks are coloured and only the plans are basic — so
	// no *row* crosses a family boundary at all, and a per-row sample would assert nothing.
	//
	// The sort is a function of the cards it is handed, so a synthetic row exercises it exactly as
	// a real one does, and this one is what the panel's ordering claim is really about: families in
	// order, cheapest first inside each.
	//
	// Shuffled into a wrong order first so the test exercises the sort and not the input.
	var row []pileEntry
	seenConcept := map[combat.ConceptID]bool{}
	for _, e := range startingDeck {
		if seenConcept[e.card.Concept] {
			continue
		}
		seenConcept[e.card.Concept] = true
		row = append(row, pileEntry{e.card, true})
	}
	if len(row) == 0 {
		t.Fatal("the starting deck is empty")
	}
	for i, j := 0, len(row)-1; i < j; i, j = i+1, j-1 {
		row[i], row[j] = row[j], row[i]
	}

	sortPileEntries(row)

	lastRank, lastCost := -1, -1
	for _, e := range row {
		rank := familyRank(e.card.Family())
		cost := e.card.Cost()

		if rank < lastRank {
			t.Fatalf("%s (%s) came after a later family — the families are out of order",
				e.card.Label(), e.card.Family())
		}
		if rank == lastRank && cost < lastCost {
			t.Errorf("%s costs %d and follows a %d-cost card in the same family",
				e.card.Label(), cost, lastCost)
		}
		if rank != lastRank {
			lastCost = -1
		}
		lastRank, lastCost = rank, cost
	}

	// And every family is actually present, or the assertions above pass vacuously on a row
	// that happens to hold one.
	seen := map[int]bool{}
	for _, e := range row {
		seen[familyRank(e.card.Family())] = true
	}
	if len(seen) != len(combat.Families()) {
		t.Errorf("the row holds %d families, want all %d", len(seen), len(combat.Families()))
	}
}

func TestDeckOrderDoesNotDependOnWhichPileACardIsIn(t *testing.T) {
	// **The panel's governing idea**: a card does not move when it is played, it only
	// dims. So the same cards must sort into the same sequence whatever their
	// availability — otherwise drawing a card would shuffle the row around it.
	var all, flipped []pileEntry
	for _, e := range startingDeck {
		all = append(all, pileEntry{e.card, true})
		flipped = append(flipped, pileEntry{e.card, false})
	}

	sortPileEntries(all)
	sortPileEntries(flipped)

	for i := range all {
		if all[i].card != flipped[i].card {
			t.Fatalf("position %d holds %v when drawable and %v when spent — the row moves as cards are played",
				i, all[i].card, flipped[i].card)
		}
	}
}

func TestFamilyRankIsIndependentOfTheEnumsOwnOrder(t *testing.T) {
	// familyRank is written out rather than read off the enum, because the enum's order is what
	// an expanded combo ID is derived from and that is a rule. This asserts the display order is
	// what was asked for, and will fail if someone "simplifies" it back to the enum.
	want := []combat.Family{combat.FamilyStab, combat.FamilySlash, combat.FamilyCrush, combat.FamilyPlan}
	for i, f := range want {
		if got := familyRank(f); got != i {
			t.Errorf("%s ranks %d, want %d", f, got, i)
		}
	}

	// The opponent's familyless cards sort last rather than colliding with a real family — they
	// are never in the player's deck, so what matters is only that they do not land inside it.
	if got := familyRank(combat.FamilyNone); got <= familyRank(combat.FamilyPlan) {
		t.Errorf("FamilyNone ranks %d, at or before the plans at %d",
			got, familyRank(combat.FamilyPlan))
	}
}
