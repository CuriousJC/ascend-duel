package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// A new run opens on the shipped limit, and a fighter it equips is actually on it. The second half
// is the one that matters: a run holding the number while the duelist fought without it would be a
// mechanic that existed everywhere except in the duel.
func TestARunEquipsItsFighterWithTheClock(t *testing.T) {
	s := New(nil)

	if got := s.RoundLimit(); got != combat.DefaultRoundLimit {
		t.Errorf("a new run is on a %d-round clock, want %d", got, combat.DefaultRoundLimit)
	}
	if got := s.Equip(combat.Duelist{}).RoundLimit; got != combat.DefaultRoundLimit {
		t.Errorf("the equipped fighter carries a limit of %d, want %d",
			got, combat.DefaultRoundLimit)
	}
}

// The clock can be moved, which is the whole reason it is a field — and it cannot be turned off by
// a drawback that reaches zero, because zero is unlimited inside the rules.
func TestTheClockMovesAndCannotBeSwitchedOff(t *testing.T) {
	s := New(nil)

	s.SetRoundLimit(7)
	if got := s.Equip(combat.Duelist{}).RoundLimit; got != 7 {
		t.Errorf("after buying two rounds the fighter is on %d, want 7", got)
	}

	s.SetRoundLimit(0)
	if got := s.RoundLimit(); got != 1 {
		t.Errorf("a limit of zero resolved to %d, want it clamped up to 1 - zero is no clock at "+
			"all in the rules, so obeying it would take the mechanic off the run", got)
	}
}

func TestTheClockSurvivesASnapshot(t *testing.T) {
	s := New(nil)
	s.SetRoundLimit(6)

	back, _, err := Resume(nil, nil, s.Snapshot(0))
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got := back.RoundLimit(); got != 6 {
		t.Errorf("a resumed run is on a %d-round clock, want the 6 it was saved on", got)
	}
}

// **A save written before the clock existed carries no number**, and JSON hands that back as zero —
// which means "no clock" everywhere else. Resuming onto it would take the mechanic off a run that
// was playing under it, silently. See resumeRoundLimit, and CLAUDE.md on zero values.
func TestAnOlderSaveResumesOntoTheDefaultClock(t *testing.T) {
	snap := New(nil).Snapshot(0)
	snap.RoundLimit = 0

	back, _, err := Resume(nil, nil, snap)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if got := back.RoundLimit(); got != combat.DefaultRoundLimit {
		t.Errorf("a save with no limit resumed onto %d, want the default %d",
			got, combat.DefaultRoundLimit)
	}
}
