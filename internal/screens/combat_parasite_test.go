package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestAnElementParasiteReachesTheCardInTheHand.
//
// **The whole point of a parasite is that it lands mid-fight**, and until 2026-09-08 no element
// parasite did. `resyncHandFromRun` wrote the hand's old colour back over the run's new one, to
// preserve a flip relic's recolour — so the run's card really did turn arcane and the card the player
// was holding did not. The round is played out of `s.hand`, so what a Hexbore bought was a change
// that arrived next fight and a consumable that appeared to vanish for nothing.
//
// The form parasites were unaffected, which is what made it hard to see: only Element was written
// back over.
//
// This holds both halves of the fix: the parasite reaches the hand, and it reaches the run.
func TestAnElementParasiteReachesTheCardInTheHand(t *testing.T) {
	run := session.New(combat.PlainCards(combat.Bash, combat.Bash))
	hexbore, ok := session.ParasiteByKey("hexbore")
	if !ok {
		t.Skip("no hexbore in the catalogue")
	}

	gs := &state.GlobalState{Run: run}
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		if c.Element == combat.Arcane {
			t.Fatalf("the fixture deck is already arcane, so this test would pass on a no-op")
		}
		s.hand = append(s.hand, paletteCard{actionCard: c, selected: true})
	}

	ids := s.selectedCardIDs()
	if !run.ApplyParasiteRolling(hexbore, ids, nil) {
		t.Fatalf("hexbore refused %d cards", len(ids))
	}
	s.resyncHandFromRun(gs)

	if len(s.hand) != len(ids) {
		t.Fatalf("hand is %d cards, want %d", len(s.hand), len(ids))
	}
	for i, c := range s.hand {
		if c.actionCard.Element != combat.Arcane {
			t.Errorf("hand card %d is %v after a hexbore, want arcane", i, c.actionCard.Element)
		}
	}
	for _, c := range run.Deck() {
		if c.Element != combat.Arcane {
			t.Errorf("the run's card is %v after a hexbore, want arcane", c.Element)
		}
	}
}
