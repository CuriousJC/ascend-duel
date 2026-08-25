package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAMissingProfileIsANewPlayer(t *testing.T) {
	p, writable, err := LoadProfile(At(t.TempDir()))
	if err != nil {
		t.Fatalf("a directory with nothing in it should not be an error: %v", err)
	}
	if !writable {
		t.Error("a new player's profile must be writable, or nothing is ever recorded")
	}
	if p.TutorialSeen {
		t.Error("a player nobody has taught has not seen the tutorial")
	}
}

func TestAnInertStoreNeverFailsALaunch(t *testing.T) {
	p, writable, err := LoadProfile(Store{})
	if err != nil || p == nil {
		t.Fatalf("a machine with nowhere to save still has to hand back a profile: %v", err)
	}
	if writable {
		t.Error("nowhere to write is not writable")
	}
	if err := SaveProfile(Store{}, p); err == nil {
		t.Error("saving to nowhere should report that it could not")
	}
}

func TestAnAwardSurvivesARoundTrip(t *testing.T) {
	s := At(t.TempDir())

	p, _, _ := LoadProfile(s)
	if !p.Award(AchievementFirstSteps) {
		t.Fatal("the first award of an achievement is new")
	}
	if p.Award(AchievementFirstSteps) {
		t.Error("awarding twice must report the second as nothing new, or a toast fires every win")
	}
	p.TutorialSeen = true
	if err := SaveProfile(s, p); err != nil {
		t.Fatal(err)
	}

	back, writable, err := LoadProfile(s)
	if err != nil || !writable {
		t.Fatalf("a profile this build wrote must read back writable: %v", err)
	}
	if !back.TutorialSeen || !back.Has(AchievementFirstSteps) {
		t.Errorf("round trip lost something: %+v", back)
	}
}

func TestSetsAreSortedOnDisk(t *testing.T) {
	s := At(t.TempDir())
	p, _, _ := LoadProfile(s)
	for _, k := range []string{"zebra", "apple", "middle"} {
		p.Award(k)
	}
	if err := SaveProfile(s, p); err != nil {
		t.Fatal(err)
	}

	var raw struct {
		Achievements []string `json:"achievements"`
	}
	readJSON(t, filepath.Join(s.Dir(), profileFile), &raw)

	want := []string{"apple", "middle", "zebra"}
	for i := range want {
		if raw.Achievements[i] != want[i] {
			t.Fatalf("achievements are written sorted, so a save is diffable: got %v", raw.Achievements)
		}
	}
}

func TestAnEmptySetIsWrittenAsAListAndNotNull(t *testing.T) {
	s := At(t.TempDir())
	p, _, _ := LoadProfile(s)
	if err := SaveProfile(s, p); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(s.Dir(), profileFile))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(splitJSONKeys(t, raw), "achievements") {
		t.Fatal("every field this build writes should be present")
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back["achievements"] == nil {
		t.Error("an empty set is [] rather than null, so the file reads as a list nobody added to")
	}
}

// TestAFutureProfileIsNotWrittenOver is the migration policy, and it is the one that cannot be
// fixed after the fact: a build that overwrites a newer file destroys what that build recorded.
func TestAFutureProfileIsNotWrittenOver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, profileFile)
	writeJSON(t, path, map[string]any{
		"version":      Version + 1,
		"tutorialSeen": true,
		"achievements": []string{"from-the-future"},
	})

	p, writable, err := LoadProfile(At(dir))
	if err != nil {
		t.Fatalf("a newer file is readable, not an error: %v", err)
	}
	if writable {
		t.Fatal("a profile from a newer build must not be written over")
	}
	if !p.TutorialSeen || !p.Has("from-the-future") {
		t.Error("what this build does understand is still read")
	}
}

// TestACorruptProfileIsAFreshOneAndIsNotOverwritten keeps both halves: the game launches, and the
// file someone might yet recover by hand is left alone.
func TestACorruptProfileIsAFreshOneAndIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, profileFile)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, writable, err := LoadProfile(At(dir))
	if err == nil {
		t.Error("a corrupt file should be reported, so it can be logged")
	}
	if p == nil || writable {
		t.Fatal("a corrupt file gives a usable profile that is not written back")
	}
}

// TestAFieldThisBuildDoesNotKnowSurvivesASave is what stops an older build silently deleting a
// newer one's progress when the version happens not to have moved.
func TestAFieldThisBuildDoesNotKnowSurvivesASave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, profileFile)
	writeJSON(t, path, map[string]any{
		"version":       Version,
		"tutorialSeen":  false,
		"somethingElse": []string{"kept"},
	})

	p, writable, err := LoadProfile(At(dir))
	if err != nil || !writable {
		t.Fatalf("same version, so writable: %v", err)
	}
	p.TutorialSeen = true
	if err := SaveProfile(At(dir), p); err != nil {
		t.Fatal(err)
	}

	var back map[string]any
	readJSON(t, path, &back)
	if back["somethingElse"] == nil {
		t.Error("an unrecognised field must be written back, not dropped")
	}
	if back["tutorialSeen"] != true {
		t.Error("and this build's own change still lands")
	}
}

// TestASaveLeavesNoTemporaryFile guards the atomic write: a leftover .tmp beside a profile is how a
// half-written save gets read one day.
func TestASaveLeavesNoTemporaryFile(t *testing.T) {
	s := At(t.TempDir())
	p, _, _ := LoadProfile(s)
	p.TutorialSeen = true
	if err := SaveProfile(s, p); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("%s was left behind by an atomic write", e.Name())
		}
	}
}

// TestOpenHonoursTheEnvironment keeps the one escape hatch working: without it, starting again as a
// new player means finding a file under AppData by hand.
func TestOpenHonoursTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DirEnv, dir)
	if got := Open().Dir(); got != dir {
		t.Errorf("%s should move the whole directory: got %q", DirEnv, got)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

func splitJSONKeys(t *testing.T, raw []byte) []string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
