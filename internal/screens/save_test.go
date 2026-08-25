package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// saveState is a global state with a scratch profile behind it, which is all the three functions in
// save.go read.
func saveState(t *testing.T) *state.GlobalState {
	t.Helper()
	store := profile.At(t.TempDir())
	prof, writable, err := profile.LoadProfile(store)
	if err != nil {
		t.Fatal(err)
	}
	seed, _ := seeds.Parse("00H602")
	enemies, bosses := data.LoadEnemies(), data.LoadBosses()
	return &state.GlobalState{
		Store: store, Profile: prof, ProfileWritable: writable,
		RunSeed: seed, Enemies: enemies, Bosses: bosses,
		Run: session.Start(enemies, bosses, seed),
	}
}

// TestAdvancingTheRunWritesIt is the wiring: the run reaching a new station is the one moment
// anything is saved, and a resume is only ever as good as that having happened.
func TestAdvancingTheRunWritesIt(t *testing.T) {
	gs := saveState(t)
	if _, ok, _ := profile.LoadRun(gs.Store); ok {
		t.Fatal("nothing should be on disk before the run moves")
	}

	gs.Run.WonFight(40)
	advanceRun(gs)

	snap, ok, err := profile.LoadRun(gs.Store)
	if err != nil || !ok {
		t.Fatalf("advancing a phase must write the run: %v", err)
	}
	if snap.Phase != gs.Run.Phase().String() || snap.Fight != gs.Run.Fight() {
		t.Errorf("the snapshot does not match the run: %+v", snap)
	}

	// And what was written comes back as the run it was.
	back, _, err := session.Resume(gs.Enemies, gs.Bosses, snap)
	if err != nil {
		t.Fatalf("what advanceRun wrote must resume: %v", err)
	}
	if back.Phase() != gs.Run.Phase() || back.Size() != gs.Run.Size() {
		t.Errorf("resumed run differs: %v/%d vs %v/%d",
			back.Phase(), back.Size(), gs.Run.Phase(), gs.Run.Size())
	}
}

// TestTheFirstWinIsRecordedOnce keeps the achievement honest: it fires on every win and the file
// holds one, which is what makes it "the first enemy you ever beat".
func TestTheFirstWinIsRecordedOnce(t *testing.T) {
	gs := saveState(t)

	awardFirstSteps(gs)
	awardFirstSteps(gs)

	back, _, err := profile.LoadProfile(gs.Store)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Has(profile.AchievementFirstSteps) {
		t.Fatal("winning a duel should record first-steps")
	}
	if len(back.Achievements) != 1 {
		t.Errorf("awarding twice records once: got %v", back.Achievements)
	}
}

// TestTheTutorialIsMarkedSeen is what stops Bob turning up on every launch forever.
func TestTheTutorialIsMarkedSeen(t *testing.T) {
	gs := saveState(t)
	markTutorialSeen(gs)

	back, _, err := profile.LoadProfile(gs.Store)
	if err != nil {
		t.Fatal(err)
	}
	if !back.TutorialSeen {
		t.Error("finishing or skipping the lesson must be recorded, or it runs again next launch")
	}
}

// TestAnUnwritableProfileIsNotWrittenTo: a file from a newer build must survive an older build
// being run against it, and an award in memory must not be the thing that destroys it.
func TestAnUnwritableProfileIsNotWrittenTo(t *testing.T) {
	gs := saveState(t)
	gs.ProfileWritable = false

	awardFirstSteps(gs)
	markTutorialSeen(gs)

	if _, ok, _ := profile.LoadRun(gs.Store); ok {
		t.Error("nothing about a profile award writes a run")
	}
	back, _, err := profile.LoadProfile(gs.Store)
	if err != nil {
		t.Fatal(err)
	}
	if back.TutorialSeen || back.Has(profile.AchievementFirstSteps) {
		t.Error("a profile marked unwritable must not be written over")
	}
}

// TestSavingWithNoRunIsHarmless: every one of these is called from a scene, and a scene can be
// reached in a test or a fixture with no run behind it.
func TestSavingWithNoRunIsHarmless(t *testing.T) {
	saveRun(nil)
	awardFirstSteps(nil)
	markTutorialSeen(nil)
	saveRun(&state.GlobalState{})
}
