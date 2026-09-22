package crashlog

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
)

// quiet empties the ring and stops the tests printing a running commentary: Note logs as well as
// records, which is wanted in a game and is noise in a test.
func quiet(t *testing.T) {
	t.Helper()
	forget()
	log.SetOutput(io.Discard)
	t.Cleanup(func() {
		forget()
		log.SetOutput(os.Stderr)
	})
}

func TestAReportLandsInTheStore(t *testing.T) {
	quiet(t)
	s := profile.At(t.TempDir())

	path, err := Write(s, Build(State{Version: "v1.2.3", RunSeed: 7}, "boom", []byte("stack")))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the report back: %v", err)
	}
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the report is not readable: %v", err)
	}
	if got.Schema != Schema || got.Version != "v1.2.3" || got.Panic != "boom" {
		t.Fatalf("the report does not say what happened: %+v", got)
	}
	if got.RunCode == "" {
		t.Fatal("a report with no run code names no run")
	}
	if got.Stack != "stack" {
		t.Fatalf("stack = %q, want the one that was handed over", got.Stack)
	}
}

// TestTheNameLeadsWithTheTime is the whole reason pruning can sort by name: the UTC stamp comes
// first and the run code follows, so alphabetical order and chronological order are the same one.
func TestTheNameLeadsWithTheTime(t *testing.T) {
	quiet(t)
	s := profile.At(t.TempDir())

	path, err := Write(s, Build(State{RunSeed: 1}, "boom", nil))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	name := filepath.Base(path)
	if !strings.HasPrefix(name, filePrefix) || !strings.HasSuffix(name, fileSuffix) {
		t.Fatalf("name = %q, want crash-<utc>-<code>.json", name)
	}
	stamp := strings.TrimPrefix(name, filePrefix)
	if len(stamp) < len(stampFormat) || stamp[8] != 'T' || stamp[15] != 'Z' {
		t.Fatalf("name = %q, want the UTC stamp leading", name)
	}
}

// TestOldReportsArePruned holds the rule that a config directory may not grow without bound. It is
// the bug that only ever shows up on the machine of the player who plays most.
func TestOldReportsArePruned(t *testing.T) {
	quiet(t)
	dir := t.TempDir()
	s := profile.At(dir)

	// Written by hand rather than through Write, because several reports written in one second
	// would share a name and overwrite each other — which is a fact about the test's speed rather
	// than about the game.
	for i := 0; i < keep+4; i++ {
		name := fmt.Sprintf("%s202609%02dT000000Z-AAAAAA%s", filePrefix, i+1, fileSuffix)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("planting a report: %v", err)
		}
	}
	if _, err := Write(s, Build(State{}, "boom", nil)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if n := len(s.Names(filePrefix)); n > keep {
		t.Fatalf("%d reports left, want at most %d", n, keep)
	}
}

// TestAnInertStoreIsNotFatal is the rule the whole package is under: a machine that cannot write is
// a machine whose crash is not recorded, never a machine that crashes twice.
func TestAnInertStoreIsNotFatal(t *testing.T) {
	quiet(t)
	if _, err := Write(profile.Store{}, Build(State{}, "boom", nil)); err == nil {
		t.Fatal("an inert store reported success with nowhere to write")
	}
}

// TestNoticesDoNotRepeat holds the one thing that keeps a failing save from putting a box in front
// of the player at every phase transition for the rest of a run.
func TestNoticesDoNotRepeat(t *testing.T) {
	quiet(t)
	Tell("the run could not be saved")
	Tell("the run could not be saved")

	if got := Notice(); got != "the run could not be saved" {
		t.Fatalf("Notice = %q", got)
	}
	Dismiss()
	if got := Notice(); got != "" {
		t.Fatalf("a repeat queued a second notice: %q", got)
	}
}

// TestNoticesAreCapped stops a failure that repeats with a different wording each time from
// building a queue the player has to click their way out of.
func TestNoticesAreCapped(t *testing.T) {
	quiet(t)
	for i := 0; i < noticeCap*3; i++ {
		Tell("problem %d", i)
	}
	n := 0
	for Notice() != "" {
		Dismiss()
		n++
	}
	if n > noticeCap {
		t.Fatalf("%d notices queued, want at most %d", n, noticeCap)
	}
}

// TestTheProblemRingIsBounded holds the other half: a process that runs for hours may not keep
// every line it ever logged.
func TestTheProblemRingIsBounded(t *testing.T) {
	quiet(t)
	Tick(42)
	for i := 0; i < ringSize*2; i++ {
		Note("problem %d", i)
	}
	got := Problems()
	if len(got) != ringSize {
		t.Fatalf("%d problems kept, want %d", len(got), ringSize)
	}
	if got[0].What == "problem 0" {
		t.Fatal("the ring kept the oldest problems, want the newest")
	}
	if got[0].Tick != 42 {
		t.Fatalf("tick = %d, want the simulation tick", got[0].Tick)
	}
}

// TestAReportNamesNoPath is the rule that has to hold before there is anywhere to send a report,
// because a field added on the assumption that it stays local is a field that leaves the machine
// the day the send button lands.
func TestAReportNamesNoPath(t *testing.T) {
	quiet(t)
	dir := t.TempDir()
	Note("could not write %s", filepath.Join(dir, "run.json"))

	raw, err := json.Marshal(Build(State{Version: "v1", InstallID: "abc"}, "boom", nil))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	// The *problems* tier can carry whatever a call site put in it, so the check is on the report's
	// own fields: nothing this package fills in may name a directory.
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	got.Problems = nil
	clean, _ := json.Marshal(got)
	if strings.Contains(string(clean), dir) {
		t.Fatalf("the report names a path on this machine: %s", clean)
	}
}

// TestACrashKeepsTheJournalItWasWriting holds the half of the retention policy that makes one
// journal safe: there is one of it and the next run truncates it, so a crash has to take its own
// copy or the run worth retracing is the run the next launch overwrites.
func TestACrashKeepsTheJournalItWasWriting(t *testing.T) {
	dir := t.TempDir()
	s := profile.At(dir)

	if err := s.AppendLine("journal.jsonl", map[string]any{"kind": "duel"}); err != nil {
		t.Fatal(err)
	}

	path, err := Write(s, Build(State{RunSeed: 1}, "boom", nil), "journal.jsonl")
	if err != nil {
		t.Fatalf("writing the report: %v", err)
	}

	// **Under the report's own base name**, so the two sort together in a directory sorted by when
	// and are pruned together below.
	beside := strings.TrimSuffix(path, ".json") + ".jsonl"
	raw, err := os.ReadFile(beside)
	if err != nil {
		t.Fatalf("the journal was not kept beside the report: %v", err)
	}
	if !strings.Contains(string(raw), `"duel"`) {
		t.Fatalf("the copy holds %q, want the journal's own line", raw)
	}
}

// TestPruningCountsReportsAndTakesTheirCompanionsWithThem holds both halves of the sweep. A report
// is counted by its own file, so the allowance cannot shrink the day a companion joined it; and a
// pruned report leaves nothing behind, so the directory cannot fill with orphaned journals.
//
// **The directory is fabricated rather than crashed into being.** A report's name leads with a
// whole second, so writing a dozen through Write would mean sleeping a dozen seconds to keep them
// apart — and what is under test is the sweep, not the naming.
func TestPruningCountsReportsAndTakesTheirCompanionsWithThem(t *testing.T) {
	quiet(t)
	dir := t.TempDir()
	s := profile.At(dir)

	const made = keep + 3
	for i := 0; i < made; i++ {
		base := fmt.Sprintf("%s2026010%dT000000Z-0009D4", filePrefix, i)
		if _, err := s.WriteExport(base+fileSuffix, map[string]any{"n": i}); err != nil {
			t.Fatal(err)
		}
		if err := s.AppendLine(base+".jsonl", map[string]any{"n": i}); err != nil {
			t.Fatal(err)
		}
	}

	Prune(s)

	var reports, companions int
	for _, n := range s.Names(filePrefix) {
		switch {
		case strings.HasSuffix(n, fileSuffix):
			reports++
		case strings.HasSuffix(n, ".jsonl"):
			companions++
		}
	}

	// Prune makes room for the report about to be written, so what survives is one under the
	// allowance.
	if want := keep - 1; reports != want {
		t.Fatalf("%d reports survived, want %d", reports, want)
	}
	if companions != reports {
		t.Fatalf("%d journals beside %d reports, want one each", companions, reports)
	}
}
