package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// The bar hangs off the bottom of the duelist card and has to finish above the table row. Both
// edges move on their own — the card's off topRowTopPct, the table's off handTop — so the fit is
// exactly the kind of thing that goes stale silently.
//
// **It used to sit under the floor-and-room lines**, which are stat rows on the card as of
// 2026-09-15; the bar moved up into the space they left rather than a gap being kept where they
// were.
func TestTheRoundTimerFitsUnderTheDuelistCard(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	card, bar := s.duelistCardRect(gs), s.roundTimerRect(gs)

	if bar.Min.Y != card.Max.Y+towerLineGap {
		t.Errorf("the timer starts at y=%d, want %dpx under the card at y=%d",
			bar.Min.Y, towerLineGap, card.Max.Y)
	}
	if bar.Min.X != card.Min.X || bar.Max.X != card.Max.X {
		t.Errorf("the timer runs x=%d..%d, want the duelist card's column %d..%d",
			bar.Min.X, bar.Max.X, card.Min.X, card.Max.X)
	}
	if pane := s.relicPaneRect(gs); bar.Max.X > pane.Min.X {
		t.Errorf("the timer reaches x=%d, into the relic row at x=%d", bar.Max.X, pane.Min.X)
	}
	// The whole top band has to finish above the table row, the relic count included — that
	// assertion came here when the tower lines' own test went with the lines.
	top := tableRowTop(gs)
	if bar.Max.Y > top {
		t.Errorf("the timer reaches y=%d, into the table row at y=%d", bar.Max.Y, top)
	}
	if count := s.relicCountRect(gs); count.Max.Y > top {
		t.Errorf("the relic count reaches y=%d, into the table row at y=%d", count.Max.Y, top)
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
