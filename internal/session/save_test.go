package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// theSeed is a run code, because that is what a snapshot writes. Any valid one does.
const theSeed = "00H602"

func rosters(t *testing.T) (map[string]data.EnemyData, map[string]data.BossData) {
	t.Helper()
	return data.LoadEnemies(), data.LoadBosses()
}

// TestARunSurvivesBeingSavedAndResumed is the whole feature in one test: everything the player is
// carrying comes back.
func TestARunSurvivesBeingSavedAndResumed(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, err := seeds.Parse(theSeed)
	if err != nil {
		t.Fatal(err)
	}

	s := Start(enemies, bosses, seed)
	s.AddVitae(9)
	s.WonFight(41)
	s.SetPhase(PhaseShop)
	if !s.Wear(Rings()[0]) {
		t.Fatal("the catalogue should have a ring to wear")
	}
	s.SetElement(0, combat.Fire)

	back, gotSeed, err := Resume(enemies, bosses, s.Snapshot(seed))
	if err != nil {
		t.Fatalf("a snapshot this build wrote must resume: %v", err)
	}
	if gotSeed != seed {
		t.Errorf("the saved seed is the run's seed: got %d want %d", gotSeed, seed)
	}
	if back.Vitae() != s.Vitae() || back.Fight() != s.Fight() ||
		back.Phase() != s.Phase() || back.LifeLeft() != s.LifeLeft() {
		t.Errorf("a scalar was lost: %d/%d %d/%d %v/%v %d/%d",
			back.Vitae(), s.Vitae(), back.Fight(), s.Fight(),
			back.Phase(), s.Phase(), back.LifeLeft(), s.LifeLeft())
	}
	if len(back.Worn()) != len(s.Worn()) || back.Worn()[0] != s.Worn()[0] {
		t.Errorf("worn rings were lost: got %v want %v", back.Worn(), s.Worn())
	}
	if back.Size() != s.Size() {
		t.Fatalf("the deck changed size: got %d want %d", back.Size(), s.Size())
	}
	for i := 0; i < s.Size(); i++ {
		want, _ := s.Card(i)
		got, _ := back.Card(i)
		if got != want {
			t.Fatalf("card %d came back different:\n got %+v\nwant %+v", i, got, want)
		}
	}
}

// TestTheClimbIsRebuiltFromTheSeed is the test that has to fail the day the room-choice screen lets
// a player pick what is ahead. The climb is deliberately not in the snapshot — see profile/run.go —
// and that is only safe while it is a function of the run code and nothing else. If this ever goes
// red, the answer is to write the climb into the snapshot, never to weaken the check.
func TestTheClimbIsRebuiltFromTheSeed(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	s := Start(enemies, bosses, seed)
	back, _, err := Resume(enemies, bosses, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}

	for fight := 0; fight < 30; fight++ {
		s.fight, back.fight = fight, fight
		if s.Enemy() != back.Enemy() {
			t.Fatalf("room %d has a different opponent after a resume: %q became %q",
				fight, s.Enemy(), back.Enemy())
		}
	}
}

// TestUnclaimedSpoilsSurvive keeps a run saved at the reward station honest: the payout is frozen by
// WonFight and handed over a sentence at a time, so quitting mid-narration must not cost the rest.
func TestUnclaimedSpoilsSurvive(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	s := Start(enemies, bosses, seed)
	s.WonFight(50)
	s.ClaimFromLife()
	want := s.Spoils()

	back, _, err := Resume(enemies, bosses, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if back.Spoils() != want {
		t.Errorf("what the win still owed was lost: got %+v want %+v", back.Spoils(), want)
	}
}

// TestTheIdentityCounterIsSavedRatherThanRecomputed: a worm removing the newest card takes the
// highest id with it, and a counter derived from what survives would hand that number out twice.
func TestTheIdentityCounterIsSavedRatherThanRecomputed(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	s := Start(enemies, bosses, seed)
	s.Remove(s.Size() - 1)
	want := s.nextCardID

	back, _, err := Resume(enemies, bosses, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if back.nextCardID != want {
		t.Fatalf("the counter must not go backwards: got %d want %d", back.nextCardID, want)
	}

	back.Add(combat.Card{Concept: s.deck[0].Concept})
	newest, _ := back.Card(back.Size() - 1)
	for i := 0; i < back.Size()-1; i++ {
		if c, _ := back.Card(i); c.ID == newest.ID {
			t.Fatalf("id %d was handed out twice", newest.ID)
		}
	}
}

// TestASnapshotNamingSomethingThisBuildHasNotGotIsRefused: every name is resolved rather than
// trusted, so a run that would resume quietly wrong does not resume at all.
func TestASnapshotNamingSomethingThisBuildHasNotGotIsRefused(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, _ := seeds.Parse(theSeed)
	good := Start(enemies, bosses, seed).Snapshot(seed)

	for _, tc := range []struct {
		name string
		bend func(s *profile.RunSnapshot)
	}{
		{"a seed that is not a run code", func(s *profile.RunSnapshot) { s.Seed = "nope!" }},
		{"a station of no loop", func(s *profile.RunSnapshot) { s.Phase = "interlude" }},
		{"a card in no deck", func(s *profile.RunSnapshot) { s.Deck[0].Concept = "nonesuch" }},
		{"a colour that is no element", func(s *profile.RunSnapshot) { s.Deck[0].Element = "beige" }},
		{"a ring in no catalogue", func(s *profile.RunSnapshot) { s.Worn = []string{"nonesuch"} }},
		{"a grown ring in no catalogue", func(s *profile.RunSnapshot) { s.Grown = map[string]int{"nonesuch": 1} }},
		{"a counter below the deck", func(s *profile.RunSnapshot) { s.NextCardID = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bent := *good
			bent.Deck = append([]profile.CardSnapshot(nil), good.Deck...)
			tc.bend(&bent)
			if _, _, err := Resume(enemies, bosses, &bent); err == nil {
				t.Error("should be refused rather than resumed wrong")
			}
		})
	}
}

// TestAResumedRunDoesNotPutTheStartingRingsBackOn: Resume rebuilds a run exactly as it was, where
// New and Start both dress a run that is beginning.
func TestAResumedRunDoesNotPutTheStartingRingsBackOn(t *testing.T) {
	enemies, bosses := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	before := StartingRings
	StartingRings = []string{Rings()[0]}
	defer func() { StartingRings = before }()

	s := Start(enemies, bosses, seed)
	snap := s.Snapshot(seed)
	snap.Worn = nil

	back, _, err := Resume(enemies, bosses, snap)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Worn()) != 0 {
		t.Errorf("a run saved wearing nothing resumes wearing nothing: got %v", back.Worn())
	}
}

// TestEveryPhaseNameParsesBack keeps the two halves of the phase vocabulary in step: a station whose
// name cannot be read back is a run that will not resume from it.
func TestEveryPhaseNameParsesBack(t *testing.T) {
	for i := 0; i < PhaseCount; i++ {
		p := order[i]
		got, ok := ParsePhase(p.String())
		if !ok || got != p {
			t.Errorf("%v writes as %q and does not read back", p, p.String())
		}
	}
	if _, ok := ParsePhase("nonesuch"); ok {
		t.Error("an unknown station must be refused rather than defaulted to the fight")
	}
}
