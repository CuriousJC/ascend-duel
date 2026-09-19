package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

func testDeck() []combat.Card {
	return []combat.Card{
		{Concept: combat.Bash, Element: combat.Fire},
		{Concept: combat.Bash, Element: combat.Ice},
		{Concept: combat.Smash, Element: combat.Earth},
		{Concept: combat.Brace, Element: combat.Basic},
	}
}

// TestTheStartingListCannotBeEditedByARun is the counterpart of the same rule on
// `decks.EnemyCards`. `StartingDeck()` is what a run is built from, and an essence that
// reached through into it would be altering what *every* future run opens with.
func TestTheStartingListCannotBeEditedByARun(t *testing.T) {
	start := testDeck()
	run := New(start)

	run.SetElement(0, combat.Lightning)
	run.Remove(1)

	if start[0].Element != combat.Fire {
		t.Errorf("recoloring the run changed the list it was built from: %v", start[0])
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

	if again := run.Deck(); again[0].Concept != combat.Bash {
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
		if c.Concept == combat.Bash && c.Element == combat.Ice {
			t.Error("Remove took some other card: the ice Bash is still here")
		}
	}
}

// TestAnOutOfRangeIndexIsRefused. The offer hands out positions and the deck thins under them, so
// a stale index has to be a no-op rather than a panic mid-run or a silent hit on a neighbor.
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

// TestModifyKeepsTheConcept is the rule that makes an essence safe: it varies a card the game already
// defines. If a recolor could change what card it was, the screen could produce something
// `internal/combat` had never registered.
func TestModifyKeepsTheConcept(t *testing.T) {
	run := New(testDeck())

	before, _ := run.Card(2)
	if !run.SetElement(2, combat.Fire) {
		t.Fatal("SetElement refused a valid index")
	}
	after, _ := run.Card(2)

	if after.Concept != before.Concept {
		t.Errorf("recoloring changed the concept: %v became %v", before.Concept, after.Concept)
	}
	if after.Element != combat.Fire {
		t.Errorf("recolor did not take: element is %v", after.Element)
	}
}

// TestOnlyAWinAdvancesTheRun. A defeat has to put the same opponent back up — that is what makes
// a retry a replay of the fight rather than a skip past it.
func TestOnlyAWinAdvancesTheRun(t *testing.T) {
	run := New(testDeck())

	if run.Fight() != 0 {
		t.Errorf("a new run starts at fight %d, want 0", run.Fight())
	}
	run.WonFight(0, 0)
	run.WonFight(0, 0)
	if run.Fight() != 2 {
		t.Errorf("two wins left the run at fight %d, want 2", run.Fight())
	}
}

// TestTheCatalogLoads. A bad record panics at init, so reaching this at all is most of the
// check; what is left is that the shipped file is not one essence short of an offer.
func TestTheCatalogLoads(t *testing.T) {
	all := Essences()
	if len(all) < 2 {
		t.Fatalf("%d essences, and an offer needs two", len(all))
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
			t.Errorf("%s recolors a card to basic, which takes a color away", w.Record)
		}
	}
}

// TestABadEssenceIsRefused walks the ways a record can be wrong. Each one is something a person
// editing essences.json could plausibly type, and every one of them would otherwise produce a reward
// that quietly does nothing.
func TestABadEssenceIsRefused(t *testing.T) {
	for _, c := range []struct {
		why string
		rec data.EssenceData
	}{
		{"no key", data.EssenceData{Name: "X", Target: "remove", Text: "t"}},
		{"no name", data.EssenceData{EssenceRecord: "x", Target: "remove", Text: "t"}},
		{"no text", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "remove"}},
		{"unknown target", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "sharpen", Text: "t"}},
		{"element with no value", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "element", Text: "t"}},
		{"element the rules lack", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "element", Value: "wind", Text: "t"}},
		{"recolor to basic", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "element", Value: "basic", Text: "t"}},
		{"value nothing reads", data.EssenceData{EssenceRecord: "x", Name: "X", Target: "remove", Value: "fire", Text: "t"}},
	} {
		if _, err := resolveEssence(c.rec); err == nil {
			t.Errorf("an essence with %s was accepted", c.why)
		}
	}
}

// TestApplyDoesWhatTheTargetSays, and never touches the concept — the property that makes an essence
// safe: it varies a card the game already defines rather than inventing one.
func TestApplyDoesWhatTheTargetSays(t *testing.T) {
	t.Run("element", func(t *testing.T) {
		run := New(testDeck())
		before, _ := run.Card(0)

		if !run.Apply(Essence{Target: TargetElement, Element: combat.Earth}, 0) {
			t.Fatal("Apply refused a valid index")
		}

		after, _ := run.Card(0)
		if after.Concept != before.Concept {
			t.Errorf("recolor changed the concept: %v became %v", before.Concept, after.Concept)
		}
		if after.Element != combat.Earth {
			t.Errorf("recolor did not take: %v", after.Element)
		}
		if run.Size() != 4 {
			t.Errorf("recolor changed the deck size to %d", run.Size())
		}
	})

	t.Run("remove", func(t *testing.T) {
		run := New(testDeck())
		if !run.Apply(Essence{Target: TargetRemove}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		if run.Size() != 3 {
			t.Errorf("deck is %d after a removal, want 3", run.Size())
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		run := New(testDeck())
		want, _ := run.Card(0)

		if !run.Apply(Essence{Target: TargetDuplicate}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		if run.Size() != 5 {
			t.Fatalf("deck is %d after a copy, want 5", run.Size())
		}
		// **A copy is a different card, and its identity says so** *(2026-08-24)*. Everything
		// describing the card has to match; the ID must not, or two cards in one deck would answer
		// to one number and anything looking an original up would find whichever came first.
		got, _ := run.Card(4)
		if got.ID == want.ID {
			t.Errorf("the copy carries the original's id %d", got.ID)
		}
		got.ID, want.ID = 0, 0
		if got != want {
			t.Errorf("copied %v, want %v", got, want)
		}
	})
}

// TestApplyRefusesAnIndexTheDeckDoesNotHold. The offer hands out positions and the deck thins
// under them, so a stale one has to be a no-op rather than a panic mid-run.
func TestApplyRefusesAnIndexTheDeckDoesNotHold(t *testing.T) {
	run := New(testDeck())
	for _, w := range []Essence{{Target: TargetRemove}, {Target: TargetDuplicate},
		{Target: TargetElement, Element: combat.Fire}} {

		if run.Apply(w, 99) {
			t.Errorf("%s claimed to work on index 99 of a deck of 4", w.Target)
		}
	}
	if run.Size() != 4 {
		t.Errorf("a refused essence still changed the deck: %d cards", run.Size())
	}
}

// TestTheNumericTargetsApply covers the two per-card modifiers, which are the only way a card's
// figures move and the reason `combat.Card` grew fields at all.
func TestTheNumericTargetsApply(t *testing.T) {
	t.Run("cost", func(t *testing.T) {
		run := New([]combat.Card{{Concept: combat.Smash}})
		base := combat.ConceptOf(combat.Smash).Cost

		if !run.Apply(Essence{Target: TargetCost, Number: -1}, 0) {
			t.Fatal("Apply refused a valid index")
		}
		got, _ := run.Card(0)
		if got.Cost() != base-1 {
			t.Errorf("cheapened card costs %d, want %d", got.Cost(), base-1)
		}
	})

	t.Run("amount compounds", func(t *testing.T) {
		run := New([]combat.Card{{Concept: combat.Brace}})
		base := combat.ConceptOf(combat.Brace).Amount

		run.Apply(Essence{Target: TargetAmount, Number: 150}, 0)
		once, _ := run.Card(0)
		if once.Amount() != base*150/100 {
			t.Errorf("scaled once banks %d, want %d", once.Amount(), base*150/100)
		}

		// A second essence on the same card has to be worth something, so percentages compound
		// rather than replace.
		run.Apply(Essence{Target: TargetAmount, Number: 150}, 0)
		twice, _ := run.Card(0)
		if twice.Amount() <= once.Amount() {
			t.Errorf("scaling twice gave %d against %d once", twice.Amount(), once.Amount())
		}
	})
}

// TestTheLadderEssencesMoveOneRung, and wrap round rather than stopping at the ends.
func TestTheLadderEssencesMoveOneRung(t *testing.T) {
	up, _ := combat.Neighbor(combat.Jab, 1)

	run := New([]combat.Card{{Concept: combat.Jab, Element: combat.Fire}})
	if !run.Apply(Essence{Target: TargetPromote}, 0) {
		t.Fatal("promoting a Jab was refused")
	}

	got, _ := run.Card(0)
	if got.Concept != up {
		t.Errorf("promoting a Jab gave %v, want %v", got.Concept, up)
	}
	if got.Element != combat.Fire {
		t.Errorf("promoting changed the element to %v", got.Element)
	}

	// **The ends are joined, so no card is ever an illegal pick.** The bottom of the stab ladder
	// demoted lands on its top rung, and the top promoted lands back on the bottom.
	rungs := combat.Ladder(combat.Poke)
	top := rungs[len(rungs)-1]

	bottom := New([]combat.Card{{Concept: combat.Poke}})
	if !bottom.CanApply(Essence{Target: TargetDemote}, 0) {
		t.Error("CanApply said the bottom rung could not be demoted")
	}
	if !bottom.Apply(Essence{Target: TargetDemote}, 0) {
		t.Fatal("demoting the bottom rung was refused")
	}
	if got, _ := bottom.Card(0); got.Concept != top {
		t.Errorf("demoting the bottom rung gave %v, want the top rung %v", got.Concept, top)
	}

	over := New([]combat.Card{{Concept: top}})
	if !over.Apply(Essence{Target: TargetPromote}, 0) {
		t.Fatal("promoting the top rung was refused")
	}
	if got, _ := over.Card(0); got.Concept != combat.Poke {
		t.Errorf("promoting the top rung gave %v, want the bottom rung %v", got.Concept, combat.Poke)
	}
}

// TestCanApplyRefusesOnlyACardThatIsNotThere. A pick that would change nothing is the player's to
// make and their essence to waste; what cannot be applied is an index the deck does not hold.
func TestCanApplyRefusesOnlyACardThatIsNotThere(t *testing.T) {
	run := New([]combat.Card{{Concept: combat.Bash, Element: combat.Fire}})

	if !run.CanApply(Essence{Target: TargetElement, Element: combat.Fire}, 0) {
		t.Error("recoloring a fire card to fire was refused")
	}
	if !run.Apply(Essence{Target: TargetElement, Element: combat.Fire}, 0) {
		t.Error("recoloring a fire card to fire did not take")
	}
	if !run.CanApply(Essence{Target: TargetElement, Element: combat.Ice}, 0) {
		t.Error("recoloring a fire card to ice was refused")
	}
	if run.CanApply(Essence{Target: TargetRemove}, 99) {
		t.Error("an index the deck does not hold was offered")
	}
}
