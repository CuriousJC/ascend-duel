package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// TestATakenActionCardJoinsTheDeckWithANewIdentity. The card goes in whatever it arrived carrying, its
// traits included, under an identity no other card has.
func TestATakenActionCardJoinsTheDeckWithANewIdentity(t *testing.T) {
	run := New(StartingDeck())
	before := run.Size()
	mirrored, ok := combat.ParseCardTrait("mirrored")
	if !ok {
		t.Fatal("card_traits.json has no mirrored")
	}
	card := ActionPool()[0].WithTrait(mirrored, true)

	run.TakeActionCard(card)

	if run.Size() != before+1 {
		t.Fatalf("the deck is %d cards after a take, want %d", run.Size(), before+1)
	}
	got := run.Deck()[run.Size()-1]
	if !got.HasTrait(mirrored) || got.ID == 0 {
		t.Errorf("the taken card came in as %+v", got)
	}
	for _, c := range run.Deck()[:before] {
		if c.ID == got.ID {
			t.Errorf("the taken card shares identity %d with a card already in the deck", got.ID)
		}
	}
	if run.Stats().Total(StatCardTrait+"mirrored") != 1 {
		t.Error("taking a mirrored card did not count the trait")
	}
}

// TestTheActionPoolIsOneOfEachCard. A concept the deck holds several copies of is no likelier in a
// bundle than one it holds once.
func TestTheActionPoolIsOneOfEachCard(t *testing.T) {
	seen := map[combat.Card]bool{}
	for _, c := range ActionPool() {
		if seen[c] {
			t.Errorf("the pool holds %s %s twice", c.Element, c.Label())
		}
		seen[c] = true
	}
}
