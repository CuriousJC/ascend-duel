package screens

import (
	"github.com/curiousjc/ascend-duel/internal/session"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The order cards run in along a deck row. Windowless, like the other tests in this
// package — sortPileEntries was pulled out of drawPileGrid precisely so this could exist.

func TestDeckRowRunsFormByForm(t *testing.T) {
	// **One card of every concept the deck holds, rather than one row of the panel** *(2026-08-15)*.
	// It used to take the basic row, which held all twelve concepts back when attacks shipped in a
	// drab variant. They no longer do — the attacks are coloured and only the plans are basic — so
	// no *row* crosses a form boundary at all, and a per-row sample would assert nothing.
	//
	// The sort is a function of the cards it is handed, so a synthetic row exercises it exactly as
	// a real one does, and this one is what the panel's ordering claim is really about: forms in
	// order, cheapest first inside each.
	//
	// Shuffled into a wrong order first so the test exercises the sort and not the input.
	var row []pileEntry
	seenConcept := map[combat.ConceptID]bool{}
	for _, c := range session.StartingDeck() {
		if seenConcept[c.Concept] {
			continue
		}
		seenConcept[c.Concept] = true
		row = append(row, pileEntry{card: c, available: true, lit: true})
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
		rank := formRank(e.card.Form())
		cost := e.card.Cost()

		if rank < lastRank {
			t.Fatalf("%s (%s) came after a later form — the forms are out of order",
				e.card.Label(), e.card.Form())
		}
		if rank == lastRank && cost < lastCost {
			t.Errorf("%s costs %d and follows a %d-cost card in the same form",
				e.card.Label(), cost, lastCost)
		}
		if rank != lastRank {
			lastCost = -1
		}
		lastRank, lastCost = rank, cost
	}

	// And every form is actually present, or the assertions above pass vacuously on a row
	// that happens to hold one.
	seen := map[int]bool{}
	for _, e := range row {
		seen[formRank(e.card.Form())] = true
	}
	if len(seen) != len(combat.Forms()) {
		t.Errorf("the row holds %d forms, want all %d", len(seen), len(combat.Forms()))
	}
}

func TestDeckOrderDoesNotDependOnWhichPileACardIsIn(t *testing.T) {
	// **The panel's governing idea**: a card does not move when it is played, it only
	// dims. So the same cards must sort into the same sequence whatever their
	// availability — otherwise drawing a card would shuffle the row around it.
	var all, flipped []pileEntry
	for _, c := range session.StartingDeck() {
		all = append(all, pileEntry{card: c, available: true, lit: true})
		flipped = append(flipped, pileEntry{card: c, available: false})
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

func TestFormRankIsIndependentOfTheEnumsOwnOrder(t *testing.T) {
	// formRank is written out rather than read off the enum, because the enum's order is what
	// an expanded hand ID is derived from and that is a rule. This asserts the display order is
	// what was asked for, and will fail if someone "simplifies" it back to the enum.
	want := []combat.Form{combat.FormStab, combat.FormSlash, combat.FormCrush, combat.FormPlan}
	for i, f := range want {
		if got := formRank(f); got != i {
			t.Errorf("%s ranks %d, want %d", f, got, i)
		}
	}

	// The opponent's formless cards sort last rather than colliding with a real form — they
	// are never in the player's deck, so what matters is only that they do not land inside it.
	if got := formRank(combat.FormNone); got <= formRank(combat.FormPlan) {
		t.Errorf("FormNone ranks %d, at or before the plans at %d",
			got, formRank(combat.FormPlan))
	}
}
