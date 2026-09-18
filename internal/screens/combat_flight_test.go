package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The flights, which are checkable without a window because emitting one is bookkeeping —
// a card, an index and the size of the row it belonged to. Only the drawing needs a screen,
// and the drawing is a lerp between two rectangles this code decides.
//
// What these guard is the invariant the whole design rests on: **the cards have already
// moved by the time a flight exists.** If that ever stops being true, the hand is in two
// states at once and every predicate on this screen has to learn about it.

// flightScene is a scene with enough deck to deal from and no combatants, which is all
// spendSelected touches. Nothing here creates an ebiten.Image.
func flightScene(hand []paletteCard) *CombatScene {
	s := &CombatScene{hand: hand}
	for i := 0; i < 30; i++ {
		s.deck = append(s.deck, combat.Card{Concept: combat.Bash, Element: combat.Fire})
	}
	return s
}

func selectedHand(n, selected int) []paletteCard {
	out := make([]paletteCard, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, paletteCard{
			Card:     combat.Card{Concept: combat.Jab, Element: combat.Ice},
			selected: i < selected,
		})
	}
	return out
}

func TestSpendingRaisesAFlightForEveryCardThatMoves(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	// The logical move is complete: two cards gone, the hand dealt back to size. This is
	// what the animation is a ghost of, and it is true before a frame is drawn.
	if len(s.discard) != 2 {
		t.Fatalf("discard holds %d cards, want the 2 that were selected", len(s.discard))
	}
	if len(s.hand) != handSize {
		t.Fatalf("hand holds %d cards, want it dealt back to %d", len(s.hand), handSize)
	}

	var out int
	for _, f := range s.Theater.flights {
		if !f.outbound {
			t.Error("a card flew in on a cardFlight; dealing belongs to the deal now")
			continue
		}
		out++
		// The row it left had five cards in it, and that row is already gone.
		if f.count != 5 {
			t.Errorf("outbound flight remembers a row of %d, want the 5 it left", f.count)
		}
	}

	if out != 2 {
		t.Errorf("%d cards flew out, want 2", out)
	}

	// **The cards coming the other way are the deal's** *(2026-09-15)*, which is what makes the
	// opening hand and a refill one gesture. Five dealt: three survived the discard, and the hand
	// fills back to eight.
	in := s.Theater.deal.cards
	if len(in) != handSize-3 {
		t.Fatalf("%d cards were dealt, want %d", len(in), handSize-3)
	}
	for _, c := range in {
		if c.Count != handSize {
			t.Errorf("a dealt card targets a row of %d, want the %d it joins", c.Count, handSize)
		}
		// No run behind this scene, so no ring touches anything: one face, the pile's.
		if len(c.faces) != 1 || c.faces[0].ring != -1 {
			t.Errorf("a dealt card carries %d faces, want the pile's alone", len(c.faces))
		}
	}
	if s.Theater.deal.rings != nil {
		t.Errorf("%d rings in a cascade with no run behind it", len(s.Theater.deal.rings))
	}
}

func TestNothingSelectedRaisesNoFlights(t *testing.T) {
	// A full hand with nothing selected has nothing to move, so pressing DUEL! on an empty
	// queue must not produce a flight with no card behind it.
	s := flightScene(selectedHand(handSize, 0))
	s.spendSelected()

	if len(s.Theater.flights) != 0 {
		t.Errorf("%d flights raised for a hand where nothing moved", len(s.Theater.flights))
	}
}

func TestInboundSlotsAreSuppressedUntilTheyLand(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	// The three cards that stayed keep being drawn; the five dealt are drawn by the deal
	// instead, so the row leaves their slots empty.
	for i := 0; i < 3; i++ {
		if s.dealtTo(i) {
			t.Errorf("slot %d is suppressed, but that card never left the hand", i)
		}
	}
	for i := 3; i < handSize; i++ {
		if !s.dealtTo(i) {
			t.Errorf("slot %d holds a card still in the air but is being drawn anyway", i)
		}
	}
}

// The deal is the one mover the theater's own tick does not drive, so a scene that never called
// tickDeal would sit on a suppressed row forever. This is that handover.
func TestTheDealLandsAndTheRowComesBack(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	if !s.Theater.deal.Running() {
		t.Fatal("no deal to advance")
	}

	// The longest journey is the last card dealt, which waits out its whole stagger first. With no
	// run behind the scene there is no cascade, so the sequence goes straight to the sort.
	longest := flightTicks()
	for _, c := range s.Theater.deal.cards {
		if n := c.flight.Delay + flightTicks(); n > longest {
			longest = n
		}
	}
	for i := 0; i <= longest; i++ {
		s.tickDeal()
	}

	if s.Theater.deal.Running() {
		t.Errorf("the deal is still running after %d ticks", longest)
	}
	for i := 0; i < len(s.hand); i++ {
		if s.dealtTo(i) {
			t.Errorf("slot %d is still suppressed after the deal finished", i)
		}
	}
}

func TestFlightsLandAndStopBeingDrawn(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	if len(s.Theater.flights) == 0 {
		t.Fatal("no flights to advance")
	}

	// The longest journey is the last card dealt, which waits out its whole stagger first.
	longest := flightTicks()
	for _, f := range s.Theater.flights {
		if n := f.Delay + flightTicks(); n > longest {
			longest = n
		}
	}
	for i := 0; i < longest; i++ {
		s.Theater.Tick()
	}

	if len(s.Theater.flights) != 0 {
		t.Errorf("%d flights still in the air after %d ticks", len(s.Theater.flights), longest)
	}
	// And every slot is drawn again, which is what makes the hand whole.
	for i := 0; i < len(s.hand); i++ {
		if s.inboundTo(i) {
			t.Errorf("slot %d is still suppressed after every flight landed", i)
		}
	}
}

func TestTravelRunsFromZeroToOneAndHoldsForItsDelay(t *testing.T) {
	// The clock every moving card shares, since 2026-08-12. **age counts from zero including
	// the delay**, which is what makes it one counter rather than two that have to be kept in
	// step — the shape the old cardFlight and resolvedCard each had a different version of.
	tr := ui.NewTravel(3, 10)

	if !tr.Waiting() {
		t.Error("a travel with a delay does not start on the launch pad")
	}
	if got := tr.Progress(); got != 0 {
		t.Errorf("a waiting travel is at %v, want 0", got)
	}

	for i := 0; i < 3; i++ {
		tr.Tick()
	}
	if tr.Waiting() {
		t.Error("the travel is still waiting after its delay is spent")
	}
	if got := tr.Progress(); got != 0 {
		t.Errorf("a travel just off the pad is at %v, want 0", got)
	}

	for i := 0; i < 10; i++ {
		tr.Tick()
	}
	if got := tr.Progress(); got != 1 {
		t.Errorf("a finished travel is at %v, want 1", got)
	}
	if !tr.Done() {
		t.Error("a travel at full age does not report itself done")
	}

	// It stops rather than counting forever, so a card sitting in its seat costs one
	// comparison a frame.
	age := tr.Age
	tr.Tick()
	if tr.Age != age {
		t.Errorf("a landed travel kept counting, %d to %d", age, tr.Age)
	}
}

// TestTheQueueLeavesWithTheCardsItNamed.
//
// **The hand's name came back to its planning seat at the end of a round.** `spendSelected` takes
// the played cards out of the hand and the queue is a list of exactly those cards, but the rebuild
// had been left to `finishDeal` — so for the length of a deal the queue still named a hand whose
// cards were gone. `previewBlow` re-derived it and `drawPlannedHand` painted the preview back onto
// the table, after the creature had already answered it.
//
// It belongs with the flights because it is the same invariant they guard: the hand, the piles and
// the queue are all correct the moment `spendSelected` returns, and only the drawing lags.
func TestTheQueueLeavesWithTheCardsItNamed(t *testing.T) {
	s := flightScene(selectedHand(5, 3))

	// Alive on both sides, or planning() is false and the preview would be silent whatever the
	// queue held — which would make this pass without proving anything.
	alive := combat.Duelist{MaxLife: 50, CurrentLife: 50, DMG: 5}
	s.fighter = &entities.Combatant{Duelist: alive}
	s.enemy = &entities.Combatant{Duelist: alive}

	s.syncQueue()
	if len(s.fighterActions) != 3 {
		t.Fatalf("the fixture queued %d cards, want the 3 that are selected", len(s.fighterActions))
	}
	if _, ok := s.previewAttack(); !ok {
		t.Fatalf("the fixture names no hand, so this test could not tell the preview had gone")
	}

	s.spendSelected()

	if got := len(s.fighterActions); got != 0 {
		t.Errorf("%d cards are still queued the moment the hand was spent, want none", got)
	}
	if _, ok := s.previewAttack(); ok {
		t.Errorf("the preview still names a hand whose cards have left the row")
	}
}
