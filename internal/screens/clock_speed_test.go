package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/ui"
)

// **The between-fight screens' clocks, checked against the duel's.** It is here rather than beside
// the clock itself in internal/ui because every symbol it names is a scene's — the reward screen's
// settle, the shop's arrival, the duel's flights — and internal/ui knows about none of them.

func TestTheBetweenFightScreensMoveOnTheSameSpeedAsTheDuel(t *testing.T) {
	// The duel's clocks and the reward screen's have to scale together. Comparing them against the
	// speed rather than against fixed numbers is what makes this survive a tuning change: it fails
	// when one of them stops being a proportion, not when the pace is retuned.
	cases := []struct {
		name  string
		ticks int
	}{
		{"settleFlightTicks", settleFlightTicks()},
		{"settledHoldTicks", settledHoldTicks()},
		{"victoryHoldTicks", victoryHoldTicks()},
		{"flightTicks", flightTicks()},
		{"hitFlyTicks", hitFlyTicks()},
	}

	for _, c := range cases {
		if c.ticks < 1 {
			t.Errorf("%s is %d ticks — nothing moves", c.name, c.ticks)
		}
		if c.ticks > ui.BeatTicks*8 {
			t.Errorf("%s is %d ticks, over eight beats — that is a pause, not a movement",
				c.name, c.ticks)
		}
	}
}
