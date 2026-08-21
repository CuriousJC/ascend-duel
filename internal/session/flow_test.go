package session

import "testing"

// The run's loop. It is arithmetic over a small list, so it is checkable without anything else
// being built — which matters, because two of the four stations have no scene yet and the loop has
// to be right before either arrives.

func TestARunOpensOnTheFight(t *testing.T) {
	// The first thing that happens in a run is a duel. A run that opened on the reward would be
	// paying out for a fight nobody had.
	run := New(testDeck())
	if run.Phase() != PhaseFight {
		t.Fatalf("a run opens on %s", run.Phase())
	}
}

func TestTheLoopComesBackRoundToTheFight(t *testing.T) {
	// **The whole point of a loop is that it closes.** Walking every station once has to land back
	// where it started, or the run stops after one lap in whichever station forgot its successor —
	// which is exactly the failure the hardcoded jumps could produce.
	run := New(testDeck())
	start := run.Phase()

	for i := 0; i < PhaseCount; i++ {
		run.Advance()
	}
	if run.Phase() != start {
		t.Fatalf("a full lap ended on %s, not %s", run.Phase(), start)
	}
}

func TestEveryStationIsVisitedExactlyOncePerLap(t *testing.T) {
	// A station left out of the order is a screen the player never sees; a station in it twice is
	// one they see twice. Neither fails anywhere else.
	run := New(testDeck())
	seen := map[Phase]int{run.Phase(): 1}

	for i := 0; i < PhaseCount-1; i++ {
		run.Advance()
		seen[run.Phase()]++
	}

	if len(seen) != PhaseCount {
		t.Fatalf("a lap visited %d stations, not %d", len(seen), PhaseCount)
	}
	for p, n := range seen {
		if n != 1 {
			t.Errorf("%s was visited %d times in one lap", p, n)
		}
	}
}

func TestEveryPhaseHasAName(t *testing.T) {
	// The names go into traces and are the candidate for what a save file writes down. A phase
	// added without one prints "unknown", which is the sort of thing nobody notices until a bug
	// report is unreadable.
	for _, p := range order {
		if p.String() == "unknown" {
			t.Errorf("phase %d has no name", p)
		}
	}
}

func TestAdvanceIsSeparateFromWinning(t *testing.T) {
	// **Losing advances the phase without advancing the room.** A defeat has to put the same
	// opponent back up, so the room counter is WonFight's business and never Advance's — if these
	// two ever merge, a loss quietly skips a fight.
	run := New(testDeck())
	before := run.Fight()

	run.Advance()
	run.Advance()

	if run.Fight() != before {
		t.Fatalf("advancing the loop moved the room counter from %d to %d", before, run.Fight())
	}
}
