package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		t.Error("an unrecognized field must be written back, not dropped")
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

// TestOpenHonorsTheEnvironment keeps the one escape hatch working: without it, starting again as a
// new player means finding a file under AppData by hand.
func TestOpenHonorsTheEnvironment(t *testing.T) {
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

// **An export lands beside the profile and is handed back where it went**, because the player's
// next move after pressing the button is to go and find the file.
func TestAnExportIsWrittenBesideTheProfile(t *testing.T) {
	dir := t.TempDir()
	s := At(dir)

	path, err := s.WriteExport("fightlog-0009D4-20260916-141233.json", map[string]string{"seed": "0009D4"})
	if err != nil {
		t.Fatalf("the export must write: %v", err)
	}
	if got := filepath.Dir(path); got != dir {
		t.Errorf("the export went to %s, want %s", got, dir)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the export must be there: %v", err)
	}
	if !strings.Contains(string(raw), `"seed": "0009D4"`) {
		t.Errorf("the export reads %s", raw)
	}
}

// **A name that is not a file name is refused rather than sanitized.** This is the one door out of
// the store that takes a name from further up the game, so a separator or a `..` in it would be a
// way to write anywhere on the machine from a panel button; a quietly renamed export is a file
// nobody can find again.
func TestAnExportNameMayNotLeaveTheStore(t *testing.T) {
	s := At(t.TempDir())
	for _, name := range []string{"", ".", "..", "sub/log.json", `..\log.json`} {
		if _, err := s.WriteExport(name, map[string]string{}); err == nil {
			t.Errorf("%q was accepted as a file name", name)
		}
	}
}

// TestATallySurvivesARoundTrip is the whole premise of screens.settleCounters: a duel's tallies are
// held in memory and land when it ends. A counter that is bumped and then not written is a counter
// that reads zero forever, and an achievement asking for 300 of something can never be earned.
//
// **It fails on a field that is written but not known as well as on one that is neither.** A name
// missing from `known` is carried through as an unrecognized field, so the figure on disk survives
// a save and every bump since the load is dropped on top of it — which looks like working until
// somebody counts.
func TestATallySurvivesARoundTrip(t *testing.T) {
	dir := t.TempDir()

	p, writable, err := LoadProfile(At(dir))
	if err != nil || !writable {
		t.Fatalf("a fresh profile should be writable: %v", err)
	}
	p.Bump("concept:Bash", 2)
	p.Bump("form:slash", 7)
	if err := SaveProfile(At(dir), p); err != nil {
		t.Fatalf("saving: %v", err)
	}

	back, _, err := LoadProfile(At(dir))
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if got := back.Count("concept:Bash"); got != 2 {
		t.Errorf("concept:Bash came back as %d, want 2", got)
	}
	if got := back.Count("form:slash"); got != 7 {
		t.Errorf("form:slash came back as %d, want 7", got)
	}

	// The second save is where a counter carried as an unrecognized field goes wrong: it is put
	// back exactly as it was found, so anything bumped since the load is lost.
	back.Bump("concept:Bash", 3)
	if err := SaveProfile(At(dir), back); err != nil {
		t.Fatalf("saving again: %v", err)
	}
	again, _, err := LoadProfile(At(dir))
	if err != nil {
		t.Fatalf("reading it back again: %v", err)
	}
	if got := again.Count("concept:Bash"); got != 5 {
		t.Errorf("concept:Bash came back as %d after a second bump, want 5", got)
	}
}

// TestAProfileWithNothingTalliedWritesNoCounters holds the other half: the field is `omitempty`, so
// a player who has done nothing has a file that says so rather than one carrying an empty object.
func TestAProfileWithNothingTalliedWritesNoCounters(t *testing.T) {
	dir := t.TempDir()
	p, _, _ := LoadProfile(At(dir))
	if err := SaveProfile(At(dir), p); err != nil {
		t.Fatalf("saving: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, profileFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "counters") {
		t.Errorf("a profile with nothing tallied wrote a counters field:\n%s", raw)
	}
}
