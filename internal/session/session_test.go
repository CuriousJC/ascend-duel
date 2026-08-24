package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

func testDeck() []combat.Card {
	return []combat.Card{
		{Concept: combat.Strike, Element: combat.Fire},
		{Concept: combat.Strike, Element: combat.Ice},
		{Concept: combat.Smash, Element: combat.Earth},
		{Concept: combat.Prepare, Element: combat.Basic},
	}
}

// TestTheStartingListCannotBeEditedByARun is the counterpart of the same rule on
// `decks.EnemyCards`. `StartingDeck()` is what a run is built from, and a worm that
// reached through into it would be altering what *every* future run opens with.
func TestTheStartingListCannotBeEditedByARun(t *testing.T) {
	start := testDeck()
	run := New(start)

	run.SetElement(0, combat.Lightning)
	run.Remove(1)

	if start[0].Element != combat.Fire {
		t.Errorf("recolouring the run changed the list it was built from: %v", start[0])
	}
	if len(start) != 4 {
		t.Errorf("removing from the run shortened the list it was built from: %d", len(start))
	}
}

// TestDeckHandsBackACopy — anything that sorted or shuffled the result would otherwise be
// reordering what every future fight is dealt, and the damage would outlive whatever did it.
func TestDeckHandsBackACopy(t *testing.T) {
	run := New(testDeck())

	got := run.Deck()
	got[0] = combat.Card{Concept: combat.Cleave, Element: combat.Earth}

	if again := run.Deck(); again[0].Concept != combat.Strike {
		t.Errorf("editing the returned slice changed the run's deck: %v", again[0])
	}
}

// TestRemoveThins pins the operation the whole mechanic turns on, and that it takes the card the
// caller named rather than one beside it.
func TestRemoveThins(t *testing.T) {
	run := New(testDeck())

	if !run.Remove(1) {
		t.Fatal("Remove(1) refused a valid index")
	}
	if run.Size() != 3 {
		t.Errorf("deck is %d after one removal, want 3", run.Size())
	}

	deck := run.Deck()
	for _, c := range deck {
		if c.Concept == combat.Strike && c.Element == combat.Ice {
			t.Error("Remove took some other card: the ice Strike is still here")
		}
	}
}

// TestAnOutOfRangeIndexIsRefused. The offer hands out positions and the deck thins under them, so
// a stale index has to be a no-op rather than a panic mid-run or a silent hit on a neighbour.
func TestAnOutOfRangeIndexIsRefused(t *testing.T) {
	run := New(testDeck())

	for _, i := range []int{-1, 4, 99} {
		if run.Remove(i) {
			t.Errorf("Remove(%d) claimed to work on a deck of 4", i)
		}
		if run.SetElement(i, combat.Fire) {
			t.Errorf("SetElement(%d) claimed to work on a deck of 4", i)
		}
	}
	if run.Size() != 4 {
		t.Errorf("a refused edit still changed the deck: %d cards", run.Size())
	}
}

// TestModifyKeepsTheConcept is the rule that makes a worm safe: it varies a card the game already
// defines. If a recolour could change what card it was, the screen could produce something
// `internal/combat` had never registered.
func TestModifyKeepsTheConcept(t *testing.T) {
	run := New(testDeck())

	before, _ := run.Card(2)
	if !run.SetElement(2, combat.Fire) {
		t.Fatal("SetElement refused a valid index")
	}
	after, _ := run.Card(2)

	if after.Concept != before.Concept {
		t.Errorf("recolouring changed the concept: %v became %v", before.Concept, after.Concept)
	}
	if after.Element != combat.Fire {
		t.Errorf("recolour did not take: element is %v", after.Element)
	}
}

// TestOnlyAWinAdvancesTheRun. A defeat has to put the same opponent back up — that is what makes
// a retry a replay of the fight rather than a skip past it.
func TestOnlyAWinAdvancesTheRun(t *testing.T) {
	run := New(testDeck())

	if run.Fight() != 0 {
		t.Errorf("a new run starts at fight %d, want 0", run.Fight())
	}
	run.WonFight(0)
	run.WonFight(0)
	if run.Fight() != 2 {
		t.Errorf("two wins left the run at fight %d, want 2", run.Fight())
	}
}

// TestTheCatalogueLoads. A bad record panics at init, so reaching this at all is most of the
// check; what is left is that the shipped file is not one worm short of an offer.
func TestTheCatalogueLoads(t *testing.T) {
	all := Worms()
	if len(all) < 2 {
		t.Fatalf("%d worms, and an offer needs two", len(all))
	}

	seen := map[string]bool{}
	for _, w := range all {
		if seen[w.Record] {
			t.Errorf("%s appears twice", w.Record)
		}
		seen[w.Record] = true

		if w.Name == "" || w.Text == "" {
			t.Errorf("%s is missing a name or its text", w.Record)
		}
		if w.Target == TargetElement && w.Element == combat.Basic {
			t.Errorf("%s recolours a card to basic, which takes a colour away", w.Record)
		}
	}
}

// TestABadWormIsRefused walks the ways a record can be wrong. Each one is something a person
// editing worms.json could plausibly type, and every one of them would otherwise produce a reward
// that quietly does nothing.
func TestABadWormIsRefused(t *testing.T) {
	for _, c := range []struct {
		why string
		rec data.WormData
	}{
		{"no key", data.WormData{Name: "X", Target: "remove", Text: "t"}},
		{"no name", data.WormData{WormRecord: "x", Target: "remove", Text: "t"}},
		{"no text", data.WormData{WormRecord: "x", Name: "X", Target: "remove"}},
		{"unknown target", data.WormData{WormRecord: "x", Name: "X", Target: "sharpen", Text: "t"}},
		{"element with no value", data.WormData{WormRecord: "x", Name: "X", Target: "element", Text: "t"}},
		{"element the rules lack", data.WormData{WormRecord: "x", Name: "X", Target: "element", Value: "wind", Text: "t"}},
		{"recolour to basic", data.WormData{WormRecord: "x", Name: "X", Target: "element", Value: "basic", Text: "t"}},
		{"value nothing reads", data.WormData{WormRecord: "x", Name: "X", Target: "remove", Value: "fire", Text: "t"}},
	} {
		if _, err := resolveWorm(c.rec); err == nil {
			t.Errorf("a worm with %s was accepted", c.why)
		}
	}
}

// TestApplyDoesWhatTheTargetSays, and never touches the concept — the property that makes a worm
// safe: it varies a card the game already defines rather than inventing one.
func TestApplyDoesWhatTheTargetSays(t *testing.T) {
	t.Run("element", func(t *testing.T) {
		run := New(testDeck())
		before, _ := run.Card(0)

		if !run.Apply(Worm{Target: TargetElement, Element: combat.Earth}, 0) {
			t.Fatal("Apply refused a valid index")
		}

		after, _ := run.Card(0)
		if after.Concept != before.Concept {
			t.Errorf("recolour changed the concept: %v became %v", before.Concept, after.Concept)
		}
		if after.Element != combat.Earth {
			t.Errorf("recolour did not take: %v", after.Element)
		}
		if run.Size() != 4 {
			t.Errorf("recolour changed the deck size to %d", run.Size())
		}
	})

	t.Run("remove", func(t *testing.T) {
		run := New(testDeck())
		if !run.Apply(Worm{Target: TargetRemove}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		if run.Size() != 3 {
			t.Errorf("deck is %d after a removal, want 3", run.Size())
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		run := New(testDeck())
		want, _ := run.Card(0)

		if !run.Apply(Worm{Target: TargetDuplicate}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		if run.Size() != 5 {
			t.Fatalf("deck is %d after a copy, want 5", run.Size())
		}
		if got, _ := run.Card(4); got != want {
			t.Errorf("copied %v, want %v", got, want)
		}
	})
}

// TestApplyRefusesAnIndexTheDeckDoesNotHold. The offer hands out positions and the deck thins
// under them, so a stale one has to be a no-op rather than a panic mid-run.
func TestApplyRefusesAnIndexTheDeckDoesNotHold(t *testing.T) {
	run := New(testDeck())
	for _, w := range []Worm{{Target: TargetRemove}, {Target: TargetDuplicate},
		{Target: TargetElement, Element: combat.Fire}} {

		if run.Apply(w, 99) {
			t.Errorf("%s claimed to work on index 99 of a deck of 4", w.Target)
		}
	}
	if run.Size() != 4 {
		t.Errorf("a refused worm still changed the deck: %d cards", run.Size())
	}
}

// TestTheNumericTargetsApply covers the two per-card modifiers, which are the only way a card's
// figures move and the reason `combat.Card` grew fields at all.
func TestTheNumericTargetsApply(t *testing.T) {
	t.Run("cost", func(t *testing.T) {
		run := New([]combat.Card{{Concept: combat.Smash}})
		base := combat.ConceptOf(combat.Smash).Cost

		if !run.Apply(Worm{Target: TargetCost, Number: -1}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		got, _ := run.Card(0)
		if got.Cost() != base-1 {
			t.Errorf("cheapened card costs %d, want %d", got.Cost(), base-1)
		}
	})

	t.Run("amount compounds", func(t *testing.T) {
		run := New([]combat.Card{{Concept: combat.Prepare}})
		base := combat.ConceptOf(combat.Prepare).Amount

		run.Apply(Worm{Target: TargetAmount, Number: 150}, 0)
		once, _ := run.Card(0)
		if once.Amount() != base*150/100 {
			t.Errorf("scaled once banks %d, want %d", once.Amount(), base*150/100)
		}

		// A second worm on the same card has to be worth something, so percentages compound
		// rather than replace.
		run.Apply(Worm{Target: TargetAmount, Number: 150}, 0)
		twice, _ := run.Card(0)
		if twice.Amount() <= once.Amount() {
			t.Errorf("scaling twice gave %d against %d once", twice.Amount(), once.Amount())
		}
	})
}

// TestTheLadderWormsMoveOneRung, and refuse rather than doing nothing at the ends.
func TestTheLadderWormsMoveOneRung(t *testing.T) {
	up, _ := combat.Neighbour(combat.Jab, 1)

	run := New([]combat.Card{{Concept: combat.Jab, Element: combat.Fire}})
	if !run.Apply(Worm{Target: TargetPromote}, 0) {
		t.Fatal("promoting a Jab was refused")
	}

	got, _ := run.Card(0)
	if got.Concept != up {
		t.Errorf("promoting a Jab gave %v, want %v", got.Concept, up)
	}
	if got.Element != combat.Fire {
		t.Errorf("promoting changed the element to %v", got.Element)
	}

	// The bottom of a ladder cannot be demoted, and CanApply is what stops the screen offering it.
	// **The bottom is the zero-copy Poke now**, not the Jab — which is the change the new rungs
	// bought: every card a run actually deals can be walked in both directions.
	bottom := New([]combat.Card{{Concept: combat.Poke}})
	if bottom.CanApply(Worm{Target: TargetDemote}, 0) {
		t.Error("CanApply said a Poke could be demoted")
	}
	if bottom.Apply(Worm{Target: TargetDemote}, 0) {
		t.Error("demoting a Poke claimed to work")
	}
	if !New([]combat.Card{{Concept: combat.Jab}}).CanApply(Worm{Target: TargetDemote}, 0) {
		t.Error("a Jab could not be demoted, and the ladder now has a rung under it")
	}
}

// TestCanApplyRefusesAWormThatWouldDoNothing. A reward that lands and changes nothing is a reward
// taken away, so the screen asks before it offers a card.
func TestCanApplyRefusesAWormThatWouldDoNothing(t *testing.T) {
	run := New([]combat.Card{{Concept: combat.Strike, Element: combat.Fire}})

	if run.CanApply(Worm{Target: TargetElement, Element: combat.Fire}, 0) {
		t.Error("recolouring a fire card to fire was offered")
	}
	if !run.CanApply(Worm{Target: TargetElement, Element: combat.Ice}, 0) {
		t.Error("recolouring a fire card to ice was refused")
	}
	if run.CanApply(Worm{Target: TargetRemove}, 99) {
		t.Error("an index the deck does not hold was offered")
	}
}
