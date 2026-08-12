package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
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
		s.deck = append(s.deck, actionCard{Action: combat.Strike, Element: combat.Fire})
	}
	return s
}

func selectedHand(n, selected int) []paletteCard {
	out := make([]paletteCard, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, paletteCard{
			actionCard: actionCard{Action: combat.Jab, Element: combat.Ice},
			selected:   i < selected,
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

	var out, in int
	for _, f := range s.flights {
		if f.outbound {
			out++
			// The row it left had five cards in it, and that row is already gone.
			if f.count != 5 {
				t.Errorf("outbound flight remembers a row of %d, want the 5 it left", f.count)
			}
			continue
		}
		in++
		if f.count != handSize {
			t.Errorf("inbound flight targets a row of %d, want the %d it joins", f.count, handSize)
		}
	}

	if out != 2 {
		t.Errorf("%d cards flew out, want 2", out)
	}
	// Five dealt: three survived the discard, and the hand fills back to eight.
	if in != handSize-3 {
		t.Errorf("%d cards flew in, want %d", in, handSize-3)
	}
}

func TestNothingSelectedRaisesNoFlights(t *testing.T) {
	// A full hand with nothing selected has nothing to move, so pressing DUEL! on an empty
	// queue must not produce a flight with no card behind it.
	s := flightScene(selectedHand(handSize, 0))
	s.spendSelected()

	if len(s.flights) != 0 {
		t.Errorf("%d flights raised for a hand where nothing moved", len(s.flights))
	}
}

func TestInboundSlotsAreSuppressedUntilTheyLand(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	// The three cards that stayed keep being drawn; the five dealt are drawn by their
	// flights instead, so the row leaves their slots empty.
	for i := 0; i < 3; i++ {
		if s.inboundTo(i) {
			t.Errorf("slot %d is suppressed, but that card never left the hand", i)
		}
	}
	for i := 3; i < handSize; i++ {
		if !s.inboundTo(i) {
			t.Errorf("slot %d holds a card still in the air but is being drawn anyway", i)
		}
	}
}

func TestFlightsLandAndStopBeingDrawn(t *testing.T) {
	s := flightScene(selectedHand(5, 2))
	s.spendSelected()

	if len(s.flights) == 0 {
		t.Fatal("no flights to advance")
	}

	// The longest journey is the last card dealt, which waits out its whole stagger first.
	longest := flightTicks
	for _, f := range s.flights {
		if n := f.delay + flightTicks; n > longest {
			longest = n
		}
	}
	for i := 0; i < longest; i++ {
		s.updateFlights()
	}

	if len(s.flights) != 0 {
		t.Errorf("%d flights still in the air after %d ticks", len(s.flights), longest)
	}
	// And every slot is drawn again, which is what makes the hand whole.
	for i := 0; i < len(s.hand); i++ {
		if s.inboundTo(i) {
			t.Errorf("slot %d is still suppressed after every flight landed", i)
		}
	}
}

func TestFlightProgressRunsFromZeroToOne(t *testing.T) {
	f := cardFlight{}
	if got := f.progress(); got != 0 {
		t.Errorf("a flight starts at %v, want 0", got)
	}
	f.age = flightTicks
	if got := f.progress(); got != 1 {
		t.Errorf("a finished flight is at %v, want 1", got)
	}
	if f.live() {
		t.Error("a flight at full age still reports itself live")
	}
}
