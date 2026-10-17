package session

import "testing"

// **The run's account survives a save**, which is the point of it being in the snapshot at all:
// "how did my whole run go" is a question a player asks about a run they came back to.
func TestTheLedgerSurvivesASnapshot(t *testing.T) {
	enemies, bosses := rosters(t)

	s := New(testDeck())
	s.BeginFight(1, "Outer Room", "Giant Bat")
	s.RecordRound([]LedgerRecord{
		noted(VoiceYou, "Duelist attacks with a fire strike"),
		noted(VoiceTerm, "Bash  20  x Keen 2x"),
	}, 40, [2]LedgerStats{})
	s.EndFight(OutcomeWon)

	back, _, err := Resume(enemies, bosses, s.Snapshot(0))
	if err != nil {
		t.Fatalf("a snapshot this build wrote must resume: %v", err)
	}

	fights := back.LedgerFights()
	if len(fights) != 1 {
		t.Fatalf("the resumed run has %d fights in its account, want 1", len(fights))
	}
	f := fights[0]
	if f.Enemy != "Giant Bat" || f.Outcome != OutcomeWon || f.Realm != 1 {
		t.Errorf("the fight came back as %+v", f)
	}
	if f.Dealt() != 40 {
		t.Errorf("the fight came back having dealt %d, want 40", f.Dealt())
	}
	if len(f.Rounds) != 1 || len(f.Rounds[0].Records) != 2 {
		t.Fatalf("the fight came back with %d rounds", len(f.Rounds))
	}
	rec := f.Rounds[0].Records[0]
	if rec.Kind != KindAct || rec.Side != VoiceYou {
		t.Fatalf("a record came back as %+v, which is not how it was written", rec)
	}
	if got, want := rec.Note, "Duelist attacks with a fire strike"; got != want {
		t.Errorf("the record reads %q, want %q", got, want)
	}
}

// **A fight nobody threw a round in leaves nothing behind.** Init runs on entering the combat
// screen and again on a retry, so without this a player walking in and out of a room would fill
// the account with headings for duels that never happened.
func TestAnUnfoughtFightIsNotKept(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(1, "Outer Room", "Giant Bat")
	s.BeginFight(1, "Outer Room", "Giant Bat") // re-entered the room
	s.RecordRound([]LedgerRecord{noted(VoicePlain, "something happened")}, 5, [2]LedgerStats{})
	s.EndFight(OutcomeLost)

	s.BeginFight(1, "Outer Room", "Giant Bat") // the retry, left before a round was thrown
	s.EndFight(OutcomeLost)

	if got := len(s.LedgerFights()); got != 1 {
		t.Fatalf("the account holds %d fights, want the 1 that was actually fought", got)
	}
}

// Two fights in the same room are two records. **The account keys on nothing but the order it was
// told things**, so a room number appearing twice does not fold — folding it would make the account
// claim a fight was won that was lost first.
//
// Retry was what produced that case and Retry is gone (a death ends the run as of 2026-09-03), so
// this is now a property of the ledger rather than a scenario the game can reach. It is kept
// because the property is what stops a future re-entry from silently rewriting history.
func TestTwoFightsInTheSameRoomAreTwoRecords(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(2, "Outer Room", "Cave Troll")
	s.RecordRound([]LedgerRecord{noted(VoicePlain, "lost it")}, 10, [2]LedgerStats{})
	s.EndFight(OutcomeLost)

	s.BeginFight(2, "Outer Room", "Cave Troll")
	s.RecordRound([]LedgerRecord{noted(VoicePlain, "won it")}, 90, [2]LedgerStats{})
	s.EndFight(OutcomeWon)

	fights := s.LedgerFights()
	if len(fights) != 2 {
		t.Fatalf("the account holds %d fights, want 2", len(fights))
	}
	if fights[0].Outcome != OutcomeLost || fights[1].Outcome != OutcomeWon {
		t.Errorf("the two records are %q and %q", fights[0].Outcome, fights[1].Outcome)
	}
	if fights[1].Number != 2 {
		t.Errorf("the second record is fight %d, want 2", fights[1].Number)
	}
}

// The fight in progress is the one the panel draws open, and it stops being that the moment it
// ends.
func TestTheOpenFightIsTheOneStillBeingFought(t *testing.T) {
	s := New(testDeck())
	if _, ok := s.LedgerOpenFight(); ok {
		t.Error("a run with no fights reports one open")
	}

	s.BeginFight(1, "Outer Room", "Giant Bat")
	s.RecordRound([]LedgerRecord{noted(VoicePlain, "a round")}, 1, [2]LedgerStats{})
	if n, ok := s.LedgerOpenFight(); !ok || n != 1 {
		t.Errorf("the live fight is (%d, %v), want (1, true)", n, ok)
	}

	s.EndFight(OutcomeWon)
	if _, ok := s.LedgerOpenFight(); ok {
		t.Error("a settled fight is still reported as being fought")
	}
}

// A consumable spent while planning opens the round it was spent before, in the order it was spent,
// and does not leak into the round after.
func TestAConsumableSpentOpensTheNextRound(t *testing.T) {
	s := New(testDeck())
	s.BeginFight(1, "Outer Room", "Giant Bat")
	s.RecordUse(LedgerRecord{Side: SideYou, Name: "Duelist", Note: "cantrip", Subject: "Cantrip of Might"})
	s.RecordUse(LedgerRecord{Side: SideYou, Name: "Duelist", Note: "rune", Subject: "Embermark"})
	s.RecordRound([]LedgerRecord{noted(VoiceYou, "Duelist attacks with a fire strike")}, 10, [2]LedgerStats{})
	s.RecordRound([]LedgerRecord{noted(VoiceYou, "Duelist attacks with a fire strike")}, 10, [2]LedgerStats{})

	rounds := s.LedgerFights()[0].Rounds
	first := rounds[0].Records
	if len(first) != 3 || first[0].Kind != KindUsed || first[0].Subject != "Cantrip of Might" ||
		first[1].Kind != KindUsed || first[1].Subject != "Embermark" {
		t.Fatalf("round one opened with %+v, want the two uses in order", first)
	}
	if second := rounds[1].Records; len(second) != 1 || second[0].Kind == KindUsed {
		t.Errorf("round two carried a use it was not spent before: %+v", second)
	}
}

// TestARoundOpensOnBothDuelistsFigures. The line the panel writes under a round's heading, and
// that it survives a save.
func TestARoundOpensOnBothDuelistsFigures(t *testing.T) {
	s := New(nil)
	s.BeginFight(1, "Outer Room", "Ice Cheetah")
	s.RecordRound([]LedgerRecord{noted(VoicePlain, "a round")}, 1, [2]LedgerStats{
		{Name: "Duelist", DMG: 10, Life: 50, MaxLife: 60, AP: 6},
		{Name: "Ice Cheetah", DMG: 15, Life: 900, MaxLife: 900, AP: 6},
	})
	want := "Duelist: DMG - 10, Health 50/60, 6 AP vs. Ice Cheetah: DMG - 15, Health 900/900, 6 AP"
	if got := s.LedgerFights()[0].Rounds[0].StartLine(); got != want {
		t.Errorf("the round opens on %q, want %q", got, want)
	}

	back := resumeLedger(s.ledgerSnapshot())
	if got := back.Fights[0].Rounds[0].StartLine(); got != want {
		t.Errorf("after a save the round opens on %q, want %q", got, want)
	}
	if back.Fights[0].Room != "Outer Room" {
		t.Errorf("after a save the fight was in %q", back.Fights[0].Room)
	}
	if got := (LedgerRound{}).StartLine(); got != "" {
		t.Errorf("a round recorded without its figures opens on %q", got)
	}
}
