package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The narrow kind of screen test CLAUDE.md allows: it creates no `ebiten.Image`, needs no window
// and no font, and it guards the one part of the banked figure that is bookkeeping rather than
// geometry — **when the points are credited to the card, and for how long they stay credited**.
//
// Where the figure flies from and to cannot be checked without a screen. That it is credited once,
// on arrival, and survives its own flight being dropped is exactly the part that would fail
// silently: a double credit reads as a Prepare worth twice what it banked, and a credit dropped
// with the flight reads as the AP line flicking back down a second after it went up.

// **The points land when the figure does, not when it is raised.** That is the whole gesture — the
// number reaching the card is what raises the card's figure — so a credit before arrival would be
// the AP line moving while the figure was still crossing the screen, which is the picture this
// replaced.
func TestBankedPointsAreCreditedOnArrival(t *testing.T) {
	s := &CombatScene{}
	s.banks = []bankFlight{{amount: 2, side: combat.SideA, seat: 0,
		t: newTravel(0, bankFlyTicks+bankHoldTicks)}}

	for i := 0; i < bankFlyTicks; i++ {
		if got := s.shownBank(combat.SideA); got != 0 {
			t.Fatalf("the card gained %d AP after %d of %d ticks in the air", got, i, bankFlyTicks)
		}
		s.tickBanks()
	}

	if got := s.shownBank(combat.SideA); got != 2 {
		t.Errorf("the card shows %d banked AP on arrival, want 2", got)
	}
}

// **The credit outlives the figure.** A flight is dropped when its hold expires, and the points
// have to stay on the card until the round's end state is adopted and `BonusAP` takes them over —
// see endOfRound, which is the one place that zeroes them.
func TestBankedPointsOutliveTheirFigure(t *testing.T) {
	s := &CombatScene{}
	s.banks = []bankFlight{{amount: 2, side: combat.SideA, seat: 0,
		t: newTravel(0, bankFlyTicks+bankHoldTicks)}}

	for i := 0; i < bankFlyTicks+bankHoldTicks+2; i++ {
		s.tickBanks()
	}

	if len(s.banks) != 0 {
		t.Errorf("%d figures still in the air after the whole gesture", len(s.banks))
	}
	if got := s.shownBank(combat.SideA); got != 2 {
		t.Errorf("the card shows %d banked AP once the figure has gone, want 2", got)
	}
	if got := s.shownBank(combat.SideB); got != 0 {
		t.Errorf("the opponent's card shows %d banked AP, want 0 - a side was credited for a "+
			"Prepare it did not play", got)
	}
}

// Two Prepares in one turn are two figures and two credits, and `clearBanks` takes both down —
// which `Init` calls, because a fight that ends without a round boundary would otherwise start the
// next one with points on the card.
func TestEveryBankedFigureIsCountedAndClearable(t *testing.T) {
	s := &CombatScene{}
	for i := 0; i < 2; i++ {
		s.banks = append(s.banks, bankFlight{amount: 2, side: combat.SideA, seat: i,
			t: newTravel(0, bankFlyTicks+bankHoldTicks)})
	}

	for i := 0; i < bankFlyTicks; i++ {
		s.tickBanks()
	}
	if got := s.shownBank(combat.SideA); got != 4 {
		t.Errorf("two Prepares showed %d AP, want 4", got)
	}

	s.clearBanks()
	if len(s.banks) != 0 || s.shownBank(combat.SideA) != 0 {
		t.Errorf("after clearBanks: %d figures, %d AP shown, want none of either",
			len(s.banks), s.shownBank(combat.SideA))
	}
}
