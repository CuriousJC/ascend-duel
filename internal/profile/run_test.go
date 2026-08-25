package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNoSavedRunIsNotAnError(t *testing.T) {
	r, ok, err := LoadRun(At(t.TempDir()))
	if err != nil || ok || r != nil {
		t.Fatalf("a player who has never played has no run: %v %v %v", r, ok, err)
	}
}

func TestARunSurvivesARoundTrip(t *testing.T) {
	s := At(t.TempDir())
	want := &RunSnapshot{
		Seed: "00H602", Fight: 4, Phase: "shop", Vitae: 12, LifeLeft: 33,
		Worn:  []string{"keen", "ember"},
		Grown: map[string]int{"ember": 3},
		// Kept because a run saved at the reward station has a payout part-narrated, and dropping
		// it would pay the player less for having quit.
		Spoils:     SpoilsSnapshot{Propagated: 2, FromLife: 3, FromRoom: 4},
		NextCardID: 61,
		Deck: []CardSnapshot{
			{ID: 1, Concept: "strike", Element: "fire"},
			{ID: 7, Concept: "jab", Element: "ice", CostDelta: -1, AmountPct: 150},
		},
	}
	if err := SaveRun(s, want); err != nil {
		t.Fatal(err)
	}

	got, ok, err := LoadRun(s)
	if err != nil || !ok {
		t.Fatalf("what was just written must read back: %v", err)
	}
	if got.Seed != want.Seed || got.Fight != want.Fight || got.Phase != want.Phase ||
		got.Vitae != want.Vitae || got.LifeLeft != want.LifeLeft || got.NextCardID != want.NextCardID {
		t.Errorf("a scalar was lost:\n got %+v\nwant %+v", got, want)
	}
	if got.Spoils != want.Spoils {
		t.Errorf("unclaimed spoils were lost: got %+v", got.Spoils)
	}
	if len(got.Worn) != 2 || got.Worn[0] != "keen" {
		t.Errorf("worn order is a rule, not a presentation detail: got %v", got.Worn)
	}
	if got.Grown["ember"] != 3 {
		t.Errorf("a growing ring's accumulator was lost: got %v", got.Grown)
	}
	if len(got.Deck) != 2 || got.Deck[1].AmountPct != 150 || got.Deck[1].ID != 7 {
		t.Errorf("the deck was lost: got %+v", got.Deck)
	}
}

// TestACorruptRunIsNoRun is the harder line the run takes than the profile: half a tower is worse
// than none, and losing it costs one run rather than a career.
func TestACorruptRunIsNoRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, runFile), []byte("{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, ok, err := LoadRun(At(dir))
	if err == nil {
		t.Error("a corrupt run should be reported so it can be logged")
	}
	if ok || r != nil {
		t.Error("and it must not be resumed")
	}
}

func TestARunFromANewerBuildIsNotResumed(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, runFile), map[string]any{"version": Version + 1, "seed": "00H602"})
	if _, ok, err := LoadRun(At(dir)); ok || err != nil {
		t.Errorf("a snapshot this build cannot vouch for is not resumed: %v %v", ok, err)
	}
}

func TestDeletingARunIsContentWithThereNotBeingOne(t *testing.T) {
	s := At(t.TempDir())
	if err := DeleteRun(s); err != nil {
		t.Fatalf("deleting nothing is not a failure: %v", err)
	}
	if err := SaveRun(s, &RunSnapshot{Seed: "00H602"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRun(s); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := LoadRun(s); ok {
		t.Error("a deleted run must not still be there")
	}
}

// TestTheTwoFilesAreIndependent is the reason there are two: a broken save must not cost the
// achievements sitting beside it.
func TestTheTwoFilesAreIndependent(t *testing.T) {
	dir := t.TempDir()
	s := At(dir)

	p, _, _ := LoadProfile(s)
	p.Award(AchievementFirstSteps)
	if err := SaveProfile(s, p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runFile), []byte("ruined"), 0o644); err != nil {
		t.Fatal(err)
	}

	back, writable, err := LoadProfile(s)
	if err != nil || !writable || !back.Has(AchievementFirstSteps) {
		t.Errorf("a ruined run must leave the profile untouched: %v %v %+v", err, writable, back)
	}
}
