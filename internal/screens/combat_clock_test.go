package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// The bar has 23 pixels between the tower lines and the table row and spends 20 of them, so it is
// one layout change away from overlapping something. This is the same guard
// TestTheTowerLinesFitBetweenTheCardAndTheTable keeps on the two lines above it.
func TestTheRoundTimerFitsUnderTheTowerLines(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	place, bar := s.towerPlaceRect(gs), s.roundTimerRect(gs)

	if bar.Min.Y < place.Max.Y {
		t.Errorf("the timer starts at y=%d, over the tower lines ending at y=%d",
			bar.Min.Y, place.Max.Y)
	}
	if bar.Min.X != place.Min.X || bar.Max.X != place.Max.X {
		t.Errorf("the timer runs x=%d..%d, want the tower lines' column %d..%d",
			bar.Min.X, bar.Max.X, place.Min.X, place.Max.X)
	}
	if top := tableRowTop(gs); bar.Max.Y > top {
		t.Errorf("the timer reaches y=%d, into the table row at y=%d", bar.Max.Y, top)
	}
}

// Every cell has to be wider than the gap between them, or the bar draws as nothing at the limit
// the game actually ships.
func TestEveryCellOfTheRoundTimerIsDrawable(t *testing.T) {
	gs := testState()
	s := &CombatScene{}
	r := s.roundTimerRect(gs)

	for i := 0; i < combat.DefaultRoundLimit; i++ {
		x0 := r.Min.X + i*r.Dx()/combat.DefaultRoundLimit
		x1 := r.Min.X + (i+1)*r.Dx()/combat.DefaultRoundLimit - roundTimerCellGap
		if x1 <= x0 {
			t.Errorf("cell %d of %d runs x=%d..%d and draws nothing",
				i, combat.DefaultRoundLimit, x0, x1)
		}
	}
}

// **The bar counts rounds resolved, and it never overruns its own rectangle.** A fight is looked at
// for a moment after the round that ended it, so the count can be one past the limit — and a sixth
// filled cell on a five-cell bar would be drawn outside the column.
func TestTheRoundTimerFillsWithTheRoundsAndStopsAtTheLimit(t *testing.T) {
	s := &CombatScene{fighter: &entities.Combatant{
		Duelist: combat.Duelist{RoundLimit: combat.DefaultRoundLimit},
	}}

	for _, tc := range []struct{ round, want int }{
		{0, 0}, // planning the first round: nothing spent yet
		{1, 1},
		{combat.DefaultRoundLimit, combat.DefaultRoundLimit},
		{combat.DefaultRoundLimit + 1, combat.DefaultRoundLimit},
		{-1, 0},
	} {
		s.round = tc.round
		if got := s.roundTimerSpent(combat.DefaultRoundLimit); got != tc.want {
			t.Errorf("round %d fills %d cells, want %d", tc.round, got, tc.want)
		}
	}
}

// A screen with no fighter draws no bar rather than a full one or an empty one — a test, or the
// frame before Init.
func TestAScreenWithNoFighterHasNoClock(t *testing.T) {
	s := &CombatScene{}
	if got := s.roundTimerLimit(); got != 0 {
		t.Errorf("a fighterless screen reports a limit of %d, want none", got)
	}
}
