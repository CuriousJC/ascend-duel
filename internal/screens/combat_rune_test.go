package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// TestAnElementRuneReachesTheCardInTheHand.
//
// **The whole point of a rune is that it lands mid-fight**, and until 2026-09-08 no element
// rune did. `resyncHandFromRun` wrote the hand's old color back over the run's new one, to
// preserve a flip relic's recolor — so the run's card really did turn arcane and the card the player
// was holding did not. The round is played out of `s.hand`, so what a Hexmark bought was a change
// that arrived next fight and a consumable that appeared to vanish for nothing.
//
// The form runes were unaffected, which is what made it hard to see: only Element was written
// back over.
//
// This holds both halves of the fix: the rune reaches the hand, and it reaches the run.
func TestAnElementRuneReachesTheCardInTheHand(t *testing.T) {
	run := session.New(combat.PlainCards(combat.Bash, combat.Bash))
	hexmark, ok := session.RuneByKey("hexmark")
	if !ok {
		t.Skip("no hexmark in the catalog")
	}

	gs := &state.GlobalState{Run: run}
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		if c.Element == combat.Arcane {
			t.Fatalf("the fixture deck is already arcane, so this test would pass on a no-op")
		}
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
	}

	ids := s.selectedCardIDs()
	held := s.heldRunCards(gs)
	if !run.ApplyRuneRolling(hexmark, ids, nil) {
		t.Fatalf("hexmark refused %d cards", len(ids))
	}
	s.resyncHandFromRun(gs, held)

	if len(s.hand) != len(ids) {
		t.Fatalf("hand is %d cards, want %d", len(s.hand), len(ids))
	}
	for i, c := range s.hand {
		if c.Card.Element != combat.Arcane {
			t.Errorf("hand card %d is %v after a hexmark, want arcane", i, c.Card.Element)
		}
	}
	for _, c := range run.Deck() {
		if c.Element != combat.Arcane {
			t.Errorf("the run's card is %v after a hexmark, want arcane", c.Element)
		}
	}
}

// **A flip fires as a card is drawn, never after a rune, and a copy is the card the player picked.**
// Under an ice-to-fire orb, an earth card painted ice is held ice, and Mimic spent on it then puts an
// ice copy in the hand — not fire, and not the earth the run's card was before the paint.
func TestARunePaintedCardAndItsCopyAreNotWalkedPastTheOrb(t *testing.T) {
	run := session.New([]combat.Card{{Concept: combat.Bash, Element: combat.Earth}})
	if !run.Wear("flip-ice-to-fire") {
		t.Fatal("the orb would not go on")
	}
	mimic, ok := session.RuneByKey("duplicate-card")
	if !ok {
		t.Skip("no duplicate-card in the catalog")
	}
	paint := session.Rune{Target: session.RuneElement, Element: combat.Ice, Count: 1}

	gs := &state.GlobalState{Run: run}
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		s.hand = append(s.hand, paletteCard{Card: s.drawnAs(c), selected: true})
	}

	spendOn := func(p session.Rune) {
		t.Helper()
		inHand := s.heldRunCards(gs)
		if !run.ApplyRuneRolling(p, s.selectedCardIDs(), nil) {
			t.Fatalf("%s was refused", p.Record)
		}
		s.resyncHandFromRun(gs, inHand)
		s.seatCopies(gs, inHand)
	}

	spendOn(paint)
	if got := s.hand[0].Card.Element; got != combat.Ice {
		t.Fatalf("the painted card is held %v, want ice", got)
	}
	spendOn(mimic)
	if len(s.hand) != 2 {
		t.Fatalf("the hand is %d cards after a Mimic, want 2", len(s.hand))
	}
	if got := s.hand[1].Card; got.Element != combat.Ice || got.ID == s.hand[0].Card.ID {
		t.Errorf("the copy is %v with ID %d, want an ice card of its own", got.Element, got.ID)
	}
}

// TestSpendingARuneLeavesTheHandAtRest.
//
// **A rune's targets were still selected after it landed**, which on this screen means still
// queued: the cards stood proud of the row and kept spending the action points they were queued
// with, on a card the rune had just rewritten underneath the player. The gesture had ended and the
// row still said it was going.
//
// It is pinned here rather than by a picture because what is wrong is the state, not the drawing —
// `settleHand` clears the flags and `sortHand` resyncs the queue off them, and both are arithmetic
// over a slice. What the scenario proved, and no test can, is that the cards are seen to go back.
func TestSpendingARuneLeavesTheHandAtRest(t *testing.T) {
	run := session.New(combat.PlainCards(combat.Bash, combat.Bash, combat.Jab))
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
	}
	s.syncQueue()

	if s.selectedCount() == 0 {
		t.Fatalf("the fixture selected nothing, so this test would pass on an empty hand")
	}

	s.beginSettle()
	runSettle(s)

	if got := s.selectedCount(); got != 0 {
		t.Errorf("%d cards are still selected after the rune landed, want none", got)
	}
	for i, c := range s.hand {
		if c.selected {
			t.Errorf("hand card %d is still selected", i)
		}
	}
	if got := len(s.fighterActions); got != 0 {
		t.Errorf("%d cards are still queued for the round, want none", got)
	}
	if len(s.hand) != len(run.Deck()) {
		t.Errorf("settling changed the hand: %d cards, want %d", len(s.hand), len(run.Deck()))
	}
}

// **The three beats do not overlap, and that is what the sequence is for.** `drawHandRow` checks
// `slidingTo` before it checks the morph, so a card that fell while it was still changing would be
// drawn by its slide and its dissolve would never appear — which is the one thing worth watching.
// The cards therefore stay selected, and stay raised, until the morphs are spent.
func TestTheCardsStayRaisedUntilTheChangeIsFinished(t *testing.T) {
	run := session.New(combat.PlainCards(combat.Bash, combat.Bash, combat.Jab))
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
	}
	s.syncQueue()

	// A morph that will not finish on the first tick, which is every real one.
	s.Theater.morphs = append(s.Theater.morphs, handMorph{
		id: s.hand[0].Card.ID,
		m:  ui.MorphIn(cards.Spec{Name: "anything"}, cards.Hand),
	})

	s.beginSettle()
	s.tickSettle()

	if s.selectedCount() == 0 {
		t.Errorf("the cards came down while the change was still running")
	}
	if len(s.Theater.slides) != 0 {
		t.Errorf("a slide was raised over a running morph: %d", len(s.Theater.slides))
	}
}

// **The fall and the sort are two beats, not one.** A card that came down and slid sideways in one
// movement would make the pair unreadable, so the row does not rearrange until everything has
// landed.
func TestTheRowDoesNotSortUntilTheCardsHaveLanded(t *testing.T) {
	run := session.New(combat.PlainCards(combat.Bash, combat.Bash, combat.Jab))
	s := &CombatScene{run: run}
	for _, c := range run.Deck() {
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
	}
	s.syncQueue()

	s.beginSettle()
	s.tickSettle() // no morphs running, so this is the fall

	if s.selectedCount() != 0 {
		t.Fatalf("the fall did not clear the selection")
	}
	if len(s.Theater.slides) == 0 {
		t.Fatalf("the cards did not fall")
	}
	if s.Theater.settle.stage != settleFalling {
		t.Errorf("the sequence is at stage %v, want the fall", s.Theater.settle.stage)
	}
}

// runSettle runs the sequence to the end, the way Update would over a few dozen frames.
func runSettle(s *CombatScene) {
	for i := 0; i < 600 && s.Theater.settle.Running(); i++ {
		s.Theater.Tick()
		s.tickSettle()
	}
}
