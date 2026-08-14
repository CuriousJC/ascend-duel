package decks

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// This package needs no window — it is `data` plus `internal/combat`, neither of which
// imports Ebitengine — which is the same property that lets tools/balance run headlessly and
// the reason the enemy deck lives here rather than on the combat screen.

func testEnemy() combat.Duelist {
	d := combat.Duelist{Con: 20, Str: 10, Spd: 15}
	d.MaxLife, d.CurrentLife = 100, 100
	return d
}

func TestAnEnemyKeepsActingForAWholeDuel(t *testing.T) {
	// **The bug this was written for.** The first version kept the opponent's hand between
	// rounds, the way the player's does. A style only ever takes attacks plus a Guard or a
	// Gather, so everything it could not use stayed put: by round three the hand was seven
	// dead cards, nothing could be drawn on top of them, and the enemy stood still for the
	// rest of the duel. tools/balance reported it as a roster nothing could lose to, which is
	// exactly the kind of silent balance failure that tool exists to catch.
	//
	// Forty rounds is the balance tool's stalemate bound, so this covers a whole duel.
	for _, style := range combat.PlanStyles() {
		p := NewEnemyPile(EnemySeed, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 40; round++ {
			if plan := p.Plan(style, d); len(plan) == 0 {
				t.Fatalf("%v planned nothing in round %d: the hand has locked up", style, round)
			}
		}
	}
}

func TestAPlanOnlyEverSpendsCardsFromTheDeck(t *testing.T) {
	// A style may not conjure a card the deck does not contain — which is the whole point of
	// enemies having one, and what makes enemy_cards.json a file worth editing.
	held := map[combat.Card]bool{}
	for _, c := range EnemyCards() {
		held[c] = true
	}

	for _, style := range combat.PlanStyles() {
		p := NewEnemyPile(EnemySeed, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 20; round++ {
			for _, c := range p.Plan(style, d) {
				if !held[c] {
					t.Fatalf("%v played %v, which is not in the enemy deck", style, c)
				}
			}
		}
	}
}

func TestTheDeckIsConserved(t *testing.T) {
	// Cards may move between the three piles and may not appear or vanish. A pile that leaked
	// would look like a deck that thinned, which the reshuffle would then quietly hide.
	want := len(EnemyCards())

	p := NewEnemyPile(EnemySeed, EnemyHandSize)
	d := testEnemy()

	for round := 1; round <= 30; round++ {
		p.Plan(combat.StyleBrute, d)

		draw, hand, discard := p.Counts()
		if got := draw + hand + discard; got != want {
			t.Fatalf("round %d: %d cards across the piles, want %d (draw %d, hand %d, discard %d)",
				round, got, want, draw, hand, discard)
		}
	}
}

func TestTheSameSeedDealsTheSameDuel(t *testing.T) {
	// The determinism rule at the deck. Two piles on one seed must plan identically, which is
	// what lets a run be replayed and what stops the balance tool reporting a different
	// roster every time it is run.
	a := NewEnemyPile(EnemySeed, EnemyHandSize)
	b := NewEnemyPile(EnemySeed, EnemyHandSize)
	d := testEnemy()

	for round := 1; round <= 20; round++ {
		x, y := a.Plan(combat.StyleSwarm, d), b.Plan(combat.StyleSwarm, d)
		if len(x) != len(y) {
			t.Fatalf("round %d: %v against %v", round, x, y)
		}
		for i := range x {
			if x[i] != y[i] {
				t.Fatalf("round %d: %v against %v", round, x, y)
			}
		}
	}
}

func TestEnemyCardsCannotBeEditedByACaller(t *testing.T) {
	// EnemyCards hands back a copy. Without that, anything that sorted or shuffled the result
	// would be reordering what every future duel is dealt — and the package-level list is
	// built once at init, so the damage would outlive whatever did it.
	// Overwritten with a card the enemy deck does not contain, rather than by swapping two
	// entries. The list opens with four Gathers, so a swap of the first two is a no-op in
	// value terms and the test passed whether or not the copy was real.
	notInDeck := combat.Plain(combat.Retreat)

	first := EnemyCards()
	if len(first) == 0 {
		t.Fatal("the enemy deck is empty")
	}
	original := first[0]
	first[0] = notInDeck

	if second := EnemyCards(); second[0] != original {
		t.Errorf("editing the returned slice changed the deck every enemy draws from: got %v, want %v",
			second[0], original)
	}
}
