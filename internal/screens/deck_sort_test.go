package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The order cards run in along a deck row. Windowless, like the other tests in this
// package — sortPileEntries was pulled out of drawPileGrid precisely so this could exist.

func TestDeckRowRunsAttacksThenDefendsThenPrepares(t *testing.T) {
	// One element's worth of the real deck, deliberately shuffled into a wrong order
	// first so the test is exercising the sort and not the input.
	var row []pileEntry
	for _, e := range startingDeck {
		if e.card.element == elementFire {
			row = append(row, pileEntry{e.card, true})
		}
	}
	if len(row) == 0 {
		t.Fatal("no fire cards in the starting deck")
	}
	for i, j := 0, len(row)-1; i < j; i, j = i+1, j-1 {
		row[i], row[j] = row[j], row[i]
	}

	sortPileEntries(row)

	lastRank, lastCost := -1, -1
	for _, e := range row {
		rank := categoryRank(e.card.action.Category())
		cost := e.card.action.Cost()

		if rank < lastRank {
			t.Fatalf("%s (%s) came after a %s card — categories are out of order",
				e.card.action, e.card.action.Category(), combat.Category(lastRank))
		}
		if rank == lastRank && cost < lastCost {
			t.Errorf("%s costs %d and follows a %d-cost card in the same category",
				e.card.action, cost, lastCost)
		}
		if rank != lastRank {
			lastCost = -1
		}
		lastRank, lastCost = rank, cost
	}

	// And the three runs are actually all present, or the assertions above pass
	// vacuously on a row that happens to hold one category.
	seen := map[int]bool{}
	for _, e := range row {
		seen[categoryRank(e.card.action.Category())] = true
	}
	if len(seen) != 3 {
		t.Errorf("the row holds %d categories, want all 3", len(seen))
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

func TestCategoryRankIsIndependentOfResolutionOrder(t *testing.T) {
	// categoryRank is written out rather than read off the enum, because the enum's order
	// is the *resolution* order and that is a rule. This asserts the display order is what
	// was asked for, and will fail if someone "simplifies" it back to the enum.
	want := []combat.Category{combat.CategoryAttack, combat.CategoryDefend, combat.CategoryPrepare}
	for i, c := range want {
		if got := categoryRank(c); got != i {
			t.Errorf("%s ranks %d, want %d", c, got, i)
		}
	}
}
