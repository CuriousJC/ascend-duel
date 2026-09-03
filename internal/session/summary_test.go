package session

import "testing"

// TestASummaryAddsUpTheAccount is the arithmetic. **Every figure comes off the ledger**, so this is
// the test that catches a total quietly counting the wrong thing — which nothing on screen would.
func TestASummaryAddsUpTheAccount(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(1, "GiantBat")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "one")}, 30)
	s.RecordRound([]LedgerLine{Line(VoicePlain, "two")}, 25)
	s.EndFight(OutcomeWon)

	s.BeginFight(2, "Ogre")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "three")}, 40)
	s.EndFight(OutcomeWon)

	s.BeginFight(3, "Cave Troll")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "four")}, 11)
	s.EndFight(OutcomeLost)

	got := s.Summarise(0, EndedInDefeat)

	if got.Rooms != 3 {
		t.Errorf("three rooms were entered, got %d", got.Rooms)
	}
	if got.Defeated != 2 {
		t.Errorf("two enemies fell, got %d", got.Defeated)
	}
	if got.Dealt != 106 {
		t.Errorf("the run dealt 106, got %d", got.Dealt)
	}
	if got.Rounds != 4 {
		t.Errorf("four rounds were fought, got %d", got.Rounds)
	}
	if got.Ended != EndedInDefeat {
		t.Errorf("the run ended %q, got %q", EndedInDefeat, got.Ended)
	}
}

// TestALostFightIsNotADefeatedEnemy is the distinction the page is judged on, and the easy one to
// get wrong: rooms entered and enemies defeated are different numbers, and on a death they always
// differ by exactly the fight that killed you.
func TestALostFightIsNotADefeatedEnemy(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(1, "GiantBat")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "nope")}, 5)
	s.EndFight(OutcomeLost)

	got := s.Summarise(0, EndedInDefeat)

	if got.Rooms != 1 {
		t.Errorf("one room was entered, got %d", got.Rooms)
	}
	if got.Defeated != 0 {
		t.Errorf("nothing was defeated, got %d", got.Defeated)
	}
}

// TestTheSummaryReportsTheDeepestFloorReached rather than the room the run stopped in. They are the
// same today; the moment the climb lets a player go anywhere but up, the honest answer to "how far
// did you get" is the high-water mark.
//
// **Every fight here throws a round**, because BeginFight drops a preceding record that never did —
// a room entered and left without a round is not a fight. See ledger.go.
func TestTheSummaryReportsTheDeepestFloorReached(t *testing.T) {
	s := New(testDeck())

	for _, f := range []struct {
		floor   int
		enemy   string
		outcome string
	}{
		{1, "GiantBat", OutcomeWon},
		{4, "Ogre", OutcomeWon},
		{2, "Cave Troll", OutcomeLost},
	} {
		s.BeginFight(f.floor, f.enemy)
		s.RecordRound([]LedgerLine{Line(VoicePlain, "a round")}, 1)
		s.EndFight(f.outcome)
	}

	if got := s.Summarise(0, EndedInDefeat); got.Floor != 4 {
		t.Errorf("the run reached floor 4, the summary says %d", got.Floor)
	}
}

// TestAnEmptyRunStillSummarises is the case a player reaches by starting a climb and giving it up
// before fighting anything. **Zeroes rather than an error**, so the screen never has to decide what
// to draw instead.
func TestAnEmptyRunStillSummarises(t *testing.T) {
	s := New(testDeck())

	got := s.Summarise(0, EndedByChoice)

	if got.Rooms != 0 || got.Defeated != 0 || got.Dealt != 0 || got.Rounds != 0 {
		t.Errorf("a run with no fights should be all zeroes, got %+v", got)
	}
	if got.Seed != "000000" {
		t.Errorf("run seed 0 is the code 000000, got %q", got.Seed)
	}
}

// TestTheSummarysSeedIsTheOneItWasHanded is why Summarise takes the seed rather than reading it. A
// Session derives everything from the run seed and never stores it, exactly as Snapshot does.
func TestTheSummarysSeedIsTheOneItWasHanded(t *testing.T) {
	s := New(testDeck())

	// An arbitrary run inside the code space; the point is that it round-trips to a code.
	const seed = 563202

	got := s.Summarise(seed, EndedInDefeat)
	if got.Seed == "" || got.Seed == "000000" {
		t.Fatalf("the summary did not take the seed it was handed: %q", got.Seed)
	}
}
