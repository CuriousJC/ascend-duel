package session

import "testing"

// **The run's account survives a save**, which is the point of it being in the snapshot at all:
// "how did my whole run go" is a question a player asks about a run they came back to.
func TestTheLedgerSurvivesASnapshot(t *testing.T) {
	enemies, bosses := rosters(t)

	s := New(testDeck())
	s.BeginFight(1, "Giant Bat")
	s.RecordRound([]LedgerLine{
		{Voice: VoiceYou, Runs: []LedgerRun{
			{Text: "Duelist "},
			{Text: "attacks", Ink: InkAttack, Mark: true},
			{Text: " with a fire strike"},
		}},
		Line(VoiceTerm, "Strike  20  x Keen 2x"),
	}, 40)
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
	if f.Enemy != "Giant Bat" || f.Outcome != OutcomeWon || f.Floor != 1 {
		t.Errorf("the fight came back as %+v", f)
	}
	if f.Dealt() != 40 {
		t.Errorf("the fight came back having dealt %d, want 40", f.Dealt())
	}
	if len(f.Rounds) != 1 || len(f.Rounds[0].Lines) != 2 {
		t.Fatalf("the fight came back with %d rounds", len(f.Rounds))
	}
	line := f.Rounds[0].Lines[0]
	if line.Voice != VoiceYou || len(line.Runs) != 3 {
		t.Fatalf("a line came back as %+v, which is not how it was written", line)
	}
	if verb := line.Runs[1]; verb.Text != "attacks" || verb.Ink != InkAttack || !verb.Mark {
		t.Errorf("the marked verb came back as %+v", verb)
	}
	if got, want := line.Text(), "Duelist attacks with a fire strike"; got != want {
		t.Errorf("the line reads %q, want %q", got, want)
	}
}

// **A fight nobody threw a round in leaves nothing behind.** Init runs on entering the combat
// screen and again on a retry, so without this a player walking in and out of a room would fill
// the account with headings for duels that never happened.
func TestAnUnfoughtFightIsNotKept(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(1, "Giant Bat")
	s.BeginFight(1, "Giant Bat") // re-entered the room
	s.RecordRound([]LedgerLine{Line(VoicePlain, "something happened")}, 5)
	s.EndFight(OutcomeLost)

	s.BeginFight(1, "Giant Bat") // the retry, left before a round was thrown
	s.EndFight(OutcomeLost)

	if got := len(s.LedgerFights()); got != 1 {
		t.Fatalf("the account holds %d fights, want the 1 that was actually fought", got)
	}
}

// A retry is its own record: a defeat fought again is a different duel with a different shuffle,
// and folding the two would make the account claim a fight was won that was lost first.
func TestARetryIsItsOwnRecord(t *testing.T) {
	s := New(testDeck())

	s.BeginFight(2, "Cave Troll")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "lost it")}, 10)
	s.EndFight(OutcomeLost)

	s.BeginFight(2, "Cave Troll")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "won it")}, 90)
	s.EndFight(OutcomeWon)

	fights := s.LedgerFights()
	if len(fights) != 2 {
		t.Fatalf("the account holds %d fights, want 2", len(fights))
	}
	if fights[0].Outcome != OutcomeLost || fights[1].Outcome != OutcomeWon {
		t.Errorf("the two records are %q and %q", fights[0].Outcome, fights[1].Outcome)
	}
	if fights[1].Number != 2 {
		t.Errorf("the retry is fight %d, want 2", fights[1].Number)
	}
}

// The fight in progress is the one the panel draws open, and it stops being that the moment it
// ends.
func TestTheOpenFightIsTheOneStillBeingFought(t *testing.T) {
	s := New(testDeck())
	if _, ok := s.LedgerOpenFight(); ok {
		t.Error("a run with no fights reports one open")
	}

	s.BeginFight(1, "Giant Bat")
	s.RecordRound([]LedgerLine{Line(VoicePlain, "a round")}, 1)
	if n, ok := s.LedgerOpenFight(); !ok || n != 1 {
		t.Errorf("the live fight is (%d, %v), want (1, true)", n, ok)
	}

	s.EndFight(OutcomeWon)
	if _, ok := s.LedgerOpenFight(); ok {
		t.Error("a settled fight is still reported as being fought")
	}
}
