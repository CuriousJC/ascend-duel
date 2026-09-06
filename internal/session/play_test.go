package session

import "testing"

// **A play is counted against the rung it formed and against nothing else**, which is the tally's
// whole job: a run that has landed four Full Houses and one Pair should be able to see exactly
// that.
func TestAPlayIsCountedAgainstItsOwnRung(t *testing.T) {
	s := New(nil)

	if !s.RecordHandPlayed("pair") {
		t.Fatal("pair is not a rung the catalogue holds")
	}
	s.RecordHandPlayed("pair")
	s.RecordHandPlayed("concept-full-house")

	if n := s.PlaysOf("pair"); n != 2 {
		t.Errorf("the run has played the pair %d times, want 2", n)
	}
	if n := s.PlaysOf("concept-full-house"); n != 1 {
		t.Errorf("the run has played the full house %d times, want 1", n)
	}
	if n := s.PlaysOf("element-five-of-a-kind"); n != 0 {
		t.Errorf("a rung nobody built reports %d plays, want 0", n)
	}
}

// **A rung the catalogue has not got is refused rather than tallied.** A key nothing recognises is
// a caller reading the wrong field, and a count filed under one would be a column the panel could
// never draw.
func TestATallyOnARungThatDoesNotExistIsRefused(t *testing.T) {
	s := New(nil)
	if s.RecordHandPlayed("concept-pair") {
		t.Error("concept-pair was merged into pair, so it should no longer be tallied")
	}
	if len(s.PlayCounts()) != 0 {
		t.Errorf("the run recorded %v", s.PlayCounts())
	}
}

// **A play count is not a stone.** They are two counters over one rung, and the one that moves the
// multiplier is the stone — so tallying a rung must leave what it pays exactly where it was.
func TestATallyDoesNotMoveWhatARungPays(t *testing.T) {
	s := New(nil)
	before, ok := s.HandMultiplier("pair")
	if !ok {
		t.Fatal("the catalogue has no pair")
	}
	for i := 0; i < 20; i++ {
		s.RecordHandPlayed("pair")
	}
	if after, _ := s.HandMultiplier("pair"); after != before {
		t.Errorf("twenty plays moved the pair from %d to %d", before, after)
	}
}

// **The tally survives a quit**, on the terms a stone does: a count that reset every time the game
// was closed would be a statistic about this sitting rather than about the run.
func TestThePlayTallySurvivesBeingSavedAndResumed(t *testing.T) {
	s := New(nil)
	s.RecordHandPlayed("pair")
	s.RecordHandPlayed("pair")
	s.RecordHandPlayed("form-three-of-a-kind")

	back, _, err := Resume(nil, nil, s.Snapshot(0))
	if err != nil {
		t.Fatalf("the run would not resume: %v", err)
	}
	if n := back.PlaysOf("pair"); n != 2 {
		t.Errorf("the resumed run has played the pair %d times, want 2", n)
	}
	if n := back.PlaysOf("form-three-of-a-kind"); n != 1 {
		t.Errorf("the resumed run has played the form trips %d times, want 1", n)
	}
}

// **A tally naming a rung this build has dropped is skipped rather than fatal**, which is the
// opposite of what a stone gets: a stone is something the player paid for and losing one changes
// what the run pays, where a tally is a statistic and refusing to open a save over one would be the
// machinery mattering more than the game.
func TestASnapshotTallyOnAMissingRungIsDropped(t *testing.T) {
	s := New(nil)
	s.RecordHandPlayed("pair")

	snap := s.Snapshot(0)
	snap.Plays["a-rung-that-was-cut"] = 7

	back, _, err := Resume(nil, nil, snap)
	if err != nil {
		t.Fatalf("a tally on a dropped rung should not refuse the resume: %v", err)
	}
	if n := back.PlaysOf("pair"); n != 1 {
		t.Errorf("the surviving tally reads %d, want 1", n)
	}
}
