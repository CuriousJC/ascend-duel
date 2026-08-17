package decks

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// This package needs no window — it is `data` plus `internal/combat`, neither of which
// imports Ebitengine — which is the same property that lets tools/balance run headlessly and
// the reason the enemy decks live here rather than on the combat screen.

func testEnemy() combat.Duelist {
	d := combat.Duelist{DMG: 10, Actions: 5}
	d.MaxLife, d.CurrentLife = 100, 100
	return d
}

// sampleRecords is a handful of records spanning the roster rather than all ninety-six: these
// tests are about the *pile*, and running every enemy through forty rounds each says the same
// thing ninety-six times.
func sampleRecords(t *testing.T) []string {
	t.Helper()

	all := EnemyRecords()
	if len(all) < 8 {
		t.Fatalf("only %d enemies have decks", len(all))
	}

	var out []string
	for i := 0; i < len(all); i += len(all) / 8 {
		out = append(out, all[i])
	}
	return out
}

func TestEveryEnemyHasADeck(t *testing.T) {
	// **The failure this replaced was silent.** Every enemy used to draw from one shared list, so
	// a record could not be short of cards; now a record whose Cards array was never written would
	// deal nothing and stand still for a whole duel. buildEnemyDecks panics on an empty one — this
	// is the positive side of that, and it also holds the hand size against the smallest deck.
	for _, name := range EnemyRecords() {
		deck := EnemyCards(name)
		if len(deck) < EnemyHandSize {
			t.Errorf("%s has %d cards against a hand of %d, so it draws its whole deck every round",
				name, len(deck), EnemyHandSize)
		}

		attacks := 0
		for _, c := range deck {
			if c.Spec().Verb == combat.VerbAttack && c.Spec().Target == combat.TargetOpponent {
				attacks++
			}
		}
		if attacks == 0 {
			t.Errorf("%s holds no attack aimed at anyone, so it can never win", name)
		}
	}
}

func TestAnEnemyKeepsActingForAWholeDuel(t *testing.T) {
	// **The bug this was written for.** The first version kept the opponent's hand between
	// rounds, the way the player's does. The planner only ever takes what it can spend, so
	// everything it could not use stayed put: by round three the hand was seven dead cards,
	// nothing could be drawn on top of them, and the enemy stood still for the rest of the duel.
	// tools/balance reported it as a roster nothing could lose to, which is exactly the kind of
	// silent balance failure that tool exists to catch.
	//
	// Forty rounds is the balance tool's stalemate bound, so this covers a whole duel.
	for _, name := range sampleRecords(t) {
		p := NewEnemyPile(name, seeds.EnemyDeckPin, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 40; round++ {
			if plan := p.Plan(d); len(plan) == 0 {
				t.Fatalf("%s planned nothing in round %d: the hand has locked up", name, round)
			}
		}
	}
}

func TestAPlanOnlyEverSpendsCardsFromTheDeck(t *testing.T) {
	// The planner may not conjure a card the deck does not contain — which is the whole point of
	// enemies having one, and what makes the Cards array in enemies.json worth editing.
	for _, name := range sampleRecords(t) {
		held := map[combat.Card]bool{}
		for _, c := range EnemyCards(name) {
			held[c] = true
		}

		p := NewEnemyPile(name, seeds.EnemyDeckPin, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 20; round++ {
			for _, c := range p.Plan(d) {
				if !held[c] {
					t.Fatalf("%s played %v, which is not in its deck", name, c)
				}
			}
		}
	}
}

func TestOneEnemysCardsAreNotAnothers(t *testing.T) {
	// **Labels are scoped to their record and concepts are not shared.** Half the roster has a
	// card called Bite; if two of them resolved to one concept, a turn holding both would count a
	// pair that the deck lists never put together — and retuning one creature's Bite would retune
	// every other creature's.
	seen := map[combat.ConceptID]string{}
	for _, name := range EnemyRecords() {
		for _, c := range EnemyCards(name) {
			if owner, taken := seen[c.Concept]; taken && owner != name {
				t.Fatalf("%s and %s share the concept %q", owner, name, c.Label())
			}
			seen[c.Concept] = name
		}
	}
}

func TestTheDeckIsConserved(t *testing.T) {
	// Cards may move between the three piles and may not appear or vanish. A pile that leaked
	// would look like a deck that thinned, which the reshuffle would then quietly hide.
	for _, name := range sampleRecords(t) {
		want := len(EnemyCards(name))

		p := NewEnemyPile(name, seeds.EnemyDeckPin, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 30; round++ {
			p.Plan(d)

			draw, hand, discard := p.Counts()
			if got := draw + hand + discard; got != want {
				t.Fatalf("%s round %d: %d cards across the piles, want %d (draw %d, hand %d, discard %d)",
					name, round, got, want, draw, hand, discard)
			}
		}
	}
}

func TestTheSameSeedDealsTheSameDuel(t *testing.T) {
	// The determinism rule at the deck. Two piles on one seed must plan identically, which is
	// what lets a run be replayed and what stops the balance tool reporting a different
	// roster every time it is run.
	for _, name := range sampleRecords(t) {
		a := NewEnemyPile(name, seeds.EnemyDeckPin, EnemyHandSize)
		b := NewEnemyPile(name, seeds.EnemyDeckPin, EnemyHandSize)
		d := testEnemy()

		for round := 1; round <= 20; round++ {
			x, y := a.Plan(d), b.Plan(d)
			if len(x) != len(y) {
				t.Fatalf("%s round %d: %v against %v", name, round, x, y)
			}
			for i := range x {
				if x[i] != y[i] {
					t.Fatalf("%s round %d: %v against %v", name, round, x, y)
				}
			}
		}
	}
}

func TestEnemyCardsCannotBeEditedByACaller(t *testing.T) {
	// EnemyCards hands back a copy. Without that, anything that sorted or shuffled the result
	// would be reordering what every future duel is dealt — and the decks are built once at init,
	// so the damage would outlive whatever did it.
	//
	// Overwritten with a card that enemy's deck does not contain, rather than by swapping two
	// entries. A deck opens with a run of identical cards, so a swap of the first two is a no-op
	// in value terms and the test passed whether or not the copy was real.
	name := EnemyRecords()[0]
	notInDeck := combat.Plain(combat.Cleave)

	first := EnemyCards(name)
	if len(first) == 0 {
		t.Fatalf("%s has an empty deck", name)
	}
	original := first[0]
	first[0] = notInDeck

	if second := EnemyCards(name); second[0] != original {
		t.Errorf("editing the returned slice changed the deck %s draws from: got %v, want %v",
			name, second[0], original)
	}
}
