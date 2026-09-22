package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
)

// lines is the journal on disk, one raw line per record.
func lines(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatalf("reading the journal: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

// kinds is what each line says it is.
func kinds(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	for _, line := range lines(t, dir) {
		var rec struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("line %q is not a record: %v", line, err)
		}
		out = append(out, rec.Kind)
	}
	return out
}

func TestEveryRecordIsOneLineOfItsOwn(t *testing.T) {
	dir := t.TempDir()
	j := New(profile.At(dir), nil)

	j.Begin(Header{Version: "test", RunCode: "0009D4"})
	j.Write(Record{Kind: KindSelect, Card: 31, Label: "Jab", Seat: 3})
	j.Write(Record{Kind: KindDuel, Targets: []int{31}})

	// **One line each is the whole format**, and it is what lets a reader take the last line
	// before a panic without parsing everything above it.
	if got, want := kinds(t, dir), []string{KindHeader, KindSelect, KindDuel}; len(got) != len(want) {
		t.Fatalf("journal holds %v, want %v", got, want)
	} else {
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("journal holds %v, want %v", got, want)
			}
		}
	}
}

func TestOnlyTheFieldsAKindUsesReachTheFile(t *testing.T) {
	dir := t.TempDir()
	j := New(profile.At(dir), nil)
	j.At(12345)
	j.Begin(Header{Version: "test", RunCode: "0009D4"})
	j.Write(Record{Kind: KindSelect, Card: 31, Label: "Jab", Seat: 3})

	// The narrow record the format promises, whatever the struct looks like in Go: a select says
	// what was picked and nothing about relics, prices or phases.
	got := lines(t, dir)[1]
	want := `{"t":12345,"kind":"select","card":31,"label":"Jab","seat":3}`
	if got != want {
		t.Fatalf("select wrote\n  %s\nwant\n  %s", got, want)
	}
}

func TestStartingARunTakesTheFileAndResumingOneKeepsIt(t *testing.T) {
	dir := t.TempDir()
	j := New(profile.At(dir), nil)

	j.Begin(Header{RunCode: "0009D4"})
	j.Write(Record{Kind: KindDuel})

	// **Resuming appends**, because the journal on disk belongs to the same climb and a Continue
	// that cleared it would throw away everything before this launch.
	j.Resume(Header{RunCode: "0009D4"})
	if got := len(kinds(t, dir)); got != 3 {
		t.Fatalf("a resumed journal holds %d lines, want 3", got)
	}

	// **Starting one truncates**, which is the whole of the retention policy: one file, belonging
	// to the run being played.
	j.Begin(Header{RunCode: "ZZZZZZ"})
	if got := kinds(t, dir); len(got) != 1 || got[0] != KindHeader {
		t.Fatalf("a fresh journal holds %v, want one header", got)
	}
}

func TestAResumedHeaderSaysSoAndAFreshOneDoesNot(t *testing.T) {
	dir := t.TempDir()
	j := New(profile.At(dir), nil)
	j.Begin(Header{RunCode: "0009D4"})
	j.Resume(Header{RunCode: "0009D4"})

	var first, second Header
	all := lines(t, dir)
	if err := json.Unmarshal([]byte(all[0]), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(all[1]), &second); err != nil {
		t.Fatal(err)
	}

	// Two headers in one file is one climb played across two launches, and this is the only thing
	// that says which is which.
	if first.Resumed || !second.Resumed {
		t.Fatalf("headers read resumed=%v then %v, want false then true", first.Resumed, second.Resumed)
	}
	if first.Schema != Schema || first.Platform == "" || first.UTC == "" {
		t.Fatalf("header is missing its identity: %+v", first)
	}
}

func TestANilJournalRecordsNothingAndDoesNotPanic(t *testing.T) {
	// The property forty call sites depend on: a scene writes a line without first asking whether
	// there is a journal. See state.GlobalState.Journal.
	var j *Journal
	j.At(1)
	j.Begin(Header{})
	j.Resume(Header{})
	j.Write(Record{Kind: KindDuel})
}

func TestAChoiceBeforeAHeaderIsDropped(t *testing.T) {
	dir := t.TempDir()
	j := New(profile.At(dir), nil)
	j.Write(Record{Kind: KindDuel})

	// A headerless journal reads as a journal until somebody tries to use it, which is why this is
	// guarded rather than left to the ordering in main.
	if _, err := os.Stat(filepath.Join(dir, FileName)); !os.IsNotExist(err) {
		t.Fatalf("a choice with no header made a file: %v", err)
	}
}

func TestAJournalThatCannotBeWrittenGivesUpAndSaysSoOnce(t *testing.T) {
	// A file where the directory should be: every write fails, and fails the same way twice.
	dir := filepath.Join(t.TempDir(), "wall")
	if err := os.WriteFile(dir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	told := 0
	j := New(profile.At(dir), func(error) { told++ })
	j.Begin(Header{RunCode: "0009D4"})
	j.Write(Record{Kind: KindDuel})
	j.Write(Record{Kind: KindDuel})

	// **Once a session, not once per click.** A box per click is a queue the player has to fight
	// their way out of to keep playing.
	if told != 1 {
		t.Fatalf("the player was told %d times, want 1", told)
	}
}

func TestAnInertStoreIsSilent(t *testing.T) {
	// A machine the game could not work out where to write on. Nothing here may be fatal, and
	// nothing here may be noisy either: there is no failure to report, only nowhere to record.
	told := 0
	j := New(profile.Store{}, func(error) { told++ })
	j.Begin(Header{RunCode: "0009D4"})
	j.Write(Record{Kind: KindDuel})

	if told != 0 {
		t.Fatalf("an inert store told the player %d times, want 0", told)
	}
}
