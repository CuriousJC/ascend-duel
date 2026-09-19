package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// How far one essence reaches, and what it does when it reaches further than one card.

// A run wearing nothing spends an essence on one card, which is the mechanic.
func TestAnEssenceTakesOneCardBare(t *testing.T) {
	if got := bare(t).EssenceTargets(); got != 1 {
		t.Errorf("a bare run spends an essence on %d cards, want 1", got)
	}
}

// **Two of the relic add again rather than doubling**, which is the whole reason the verb is a
// delta: every delta sums and worn order decides nothing, because addition commutes. Held here
// rather than in the JSON, because the record's Amount is a tuning dial and this is the arithmetic
// it is tuned against.
func TestTheNecklaceAddsACardAndTwoOfThemAddTwo(t *testing.T) {
	for _, tc := range []struct {
		worn []string
		want int
	}{
		{nil, 1},
		{[]string{"essence-targets"}, 2},
	} {
		run := wearing(t, tc.worn...)
		if got := run.EssenceTargets(); got != tc.want {
			t.Errorf("wearing %v spends an essence on %d cards, want %d", tc.worn, got, tc.want)
		}
	}

	// A second copy cannot be bought — a relic already worn is refused — so the summing is checked
	// against the rules directly, which is where a brand or a duplicate would arrive.
	id, ok := combat.RelicByKey("essence-targets")
	if !ok {
		t.Fatal("no relic called essence-targets")
	}
	two := []combat.WornRelic{{Relic: id}, {Relic: id}}
	if got := combat.EssenceTargets(two); got != 3 {
		t.Errorf("two necklaces spend an essence on %d cards, want 3", got)
	}
}

// **Never below one card.** A scaling that rounded away would take the essence mechanic off the run
// rather than making it meaner.
func TestAnEssenceAlwaysReachesAtLeastOneCard(t *testing.T) {
	if got := combat.EssenceTargets(nil); got != 1 {
		t.Errorf("no relics spends an essence on %d cards, want 1", got)
	}
}

// **All or nothing, and the deck is edited from the back forwards.** Two removals in one spend take
// the two cards that were named and nothing else — which is the whole failure the descending walk
// prevents, since eating a card moves every position above it.
func TestOneEssenceEatsEveryCardItWasAimedAt(t *testing.T) {
	run := bare(t)
	eat, ok := essenceWithTarget(TargetRemove)
	if !ok {
		t.Skip("no removing essence in the catalog")
	}

	deck := run.Deck()
	first, last := deck[0], deck[len(deck)-1]
	kept := deck[1]

	if !run.ApplyToAll(eat, []int{first.ID, last.ID}) {
		t.Fatal("the essence would not take both cards")
	}
	if got, want := run.Size(), len(deck)-2; got != want {
		t.Fatalf("the deck is %d cards after eating two of %d, want %d", got, len(deck), want)
	}
	if _, still := run.CardByID(first.ID); still {
		t.Error("the first card named is still in the deck")
	}
	if _, still := run.CardByID(last.ID); still {
		t.Error("the last card named is still in the deck")
	}
	if _, still := run.CardByID(kept.ID); !still {
		t.Error("a card nobody named was eaten, which is the walk going forwards")
	}
}

// **A card named twice is refused, and refusing leaves the deck alone.** Changing one card twice
// would be an essence quietly doing half of what the player was shown.
func TestACardNamedTwiceRefusesTheWholeSpend(t *testing.T) {
	run := bare(t)
	eat, ok := essenceWithTarget(TargetRemove)
	if !ok {
		t.Skip("no removing essence in the catalog")
	}

	id := run.Deck()[0].ID
	size := run.Size()
	if run.ApplyToAll(eat, []int{id, id}) {
		t.Fatal("one card was named twice and the spend went through")
	}
	if run.Size() != size {
		t.Errorf("a refused spend left the deck at %d cards, was %d", run.Size(), size)
	}
	if run.CanApplyToAll(eat, []int{id, id}) {
		t.Error("CanApplyToAll allows what ApplyToAll refuses")
	}
}

// **Every copy a spend mints is handed over, not just the last one.** The combat screen seats them
// in the hand, so a duplicate that only reported one card would leave the other unplayable until
// the next fight.
func TestEveryCopyAnEssenceMintsIsHandedOver(t *testing.T) {
	run := bare(t)
	clone, ok := essenceWithTarget(TargetDuplicate)
	if !ok {
		t.Skip("no duplicating essence in the catalog")
	}

	deck := run.Deck()
	if !run.ApplyToAll(clone, []int{deck[0].ID, deck[1].ID}) {
		t.Fatal("the essence would not copy both cards")
	}
	if got := len(run.Duplicated()); got != 2 {
		t.Errorf("two cards were copied and %d were handed over", got)
	}
	if got, want := run.Size(), len(deck)+2; got != want {
		t.Errorf("the deck is %d cards after copying two, want %d", got, want)
	}
}

// essenceWithTarget is the first essence in the catalog doing one thing, or false when nobody has
// authored one. The tests above are about the spend rather than about any record, so they name a
// target and take whatever fills it.
func essenceWithTarget(target EssenceTarget) (Essence, bool) {
	for _, w := range Essences() {
		if w.Target == target {
			return w, true
		}
	}
	return Essence{}, false
}

// **Every record in the catalog knows how to say it reaches further.** The rewrite is driven off
// two authored shapes, and a line fitting neither is printed as it stands rather than mangled — so
// the failure this catches is an essence that quietly keeps saying CARD while it takes three.
func TestEveryEssenceSaysHowFarItReaches(t *testing.T) {
	for _, w := range Essences() {
		if got := w.TextAt(1); got != w.Text {
			t.Errorf("%s at one card says %q, want the authored %q", w.Record, got, w.Text)
		}
		if got := w.TextAt(2); got == w.Text {
			t.Errorf("%s says %q whether it takes one card or two — its wording fits neither "+
				"shape the rewrite knows", w.Record, got)
		}
	}
}

// The wording itself, on one record of each shape. **The catalog is authored for one card and the
// wider lines are derived**, so this is where the derivation is pinned.
func TestTheWordingAtEveryReach(t *testing.T) {
	for _, tc := range []struct {
		record  string
		targets int
		want    string
	}{
		// A sentence about the cards: the count leads and the verb agrees with it.
		{"change-fire", 2, "2 CARDS BECOME FIRE"},
		{"change-fire", 3, "3 CARDS BECOME FIRE"},
		{"add-dmg", 2, "2 CARDS GAIN DMG +1/2"},
		{"demote", 2, "2 CARDS LOSE STRENGTH -1 AP"},
		// A sentence about the doing: the count lands on what is being done to.
		{"dissolve", 2, "DESTROY 2 CARDS"},
		{"dissolve", 4, "DESTROY 4 CARDS"},
		// **A copy is counted in times rather than in cards** *(owner's call, 2026-09-19)*.
		{"multiply", 2, "COPY CARD TWICE"},
		{"multiply", 3, "COPY CARD 3 TIMES"},
	} {
		w, ok := EssenceByKey(tc.record)
		if !ok {
			t.Errorf("no essence called %s", tc.record)
			continue
		}
		if got := w.TextAt(tc.targets); got != tc.want {
			t.Errorf("%s on %d cards says %q, want %q", tc.record, tc.targets, got, tc.want)
		}
	}
}

// An authored line the rewrite does not recognize is printed as it was written. **Mangling is the
// worse failure**: a sentence nobody can read is harder to spot than one that has not learned to
// count.
func TestAnUnfamiliarLineIsLeftAlone(t *testing.T) {
	w := Essence{Record: "odd", Text: "THE DECK REMEMBERS"}
	if got := w.TextAt(3); got != w.Text {
		t.Errorf("an unfamiliar line became %q, want it left as %q", got, w.Text)
	}
}
