package screens

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestTheCrashPageSaysWhatABugReportNeeds holds the four facts the page exists to carry. Every one
// of them is something the player has to be able to repeat back, and the build is on the list for
// the reason main.version exists at all.
func TestTheCrashPageSaysWhatABugReportNeeds(t *testing.T) {
	gs := &state.GlobalState{
		ScreenWidth:  state.ScreenWidth,
		ScreenHeight: state.ScreenHeight,
		Version:      "v1.2.3",
		Crash: &state.CrashInfo{
			Panic: "index out of range",
			Code:  "0009D4",
			Path:  "C:/somewhere/crash-20260922T101500Z-0009D4.json",
		},
	}

	want := map[string]string{
		"what happened": "index out of range",
		"run code":      "0009D4",
		"build":         "v1.2.3",
		"report":        "crash-20260922T101500Z-0009D4.json",
	}
	got := crashFacts(gs)
	if len(got) != len(want) {
		t.Fatalf("%d facts on the page, want %d", len(got), len(want))
	}
	for _, f := range got {
		if w, ok := want[f.label]; !ok || w != f.value {
			t.Errorf("%q reads %q, want %q", f.label, f.value, w)
		}
	}
}

// TestAReportThatCouldNotBeWrittenSaysSo is the honest failure. A page that silently omits the file
// has told the player to send something that is not there.
func TestAReportThatCouldNotBeWrittenSaysSo(t *testing.T) {
	gs := &state.GlobalState{
		ScreenWidth:  state.ScreenWidth,
		ScreenHeight: state.ScreenHeight,
		Crash:        &state.CrashInfo{Panic: "boom"},
	}
	for _, f := range crashFacts(gs) {
		if f.label == "report" && f.value != crashNoFile {
			t.Fatalf("the report line reads %q, want %q", f.value, crashNoFile)
		}
	}

	scene := &CrashScene{}
	scene.Init(gs)
	if scene.reveal.State != models.ButtonStateDisabled {
		t.Fatal("SHOW REPORT is live with no report to show")
	}
	if scene.quit.State == models.ButtonStateDisabled {
		t.Fatal("QUIT is dead, and it is the only way out of this page")
	}
}

// TestTheReportLineDoesNotRunOffTheScreen is the fault the first look at this page found: the whole
// path was one value in the block, and the end of it — the thing the page exists to say — was
// simply off the right edge.
func TestTheReportLineDoesNotRunOffTheScreen(t *testing.T) {
	// The fixture is joined rather than written out, because filepath splits on the separator of
	// the OS running the test and a backslash is an ordinary character everywhere but Windows.
	deep := filepath.Join("C:/", "Users", "somebody", "AppData", "Local", "Temp", "claude",
		"c--repos-ascend-duel", "ba39b14f-326f-4fa6-b72a-8216f384c422", "scratchpad",
		"blowup", "prof-crash")
	report := filepath.Join(deep, "crash-20260922T101500Z-0009D4.json")
	gs := &state.GlobalState{
		ScreenWidth:  state.ScreenWidth,
		ScreenHeight: state.ScreenHeight,
		Crash:        &state.CrashInfo{Path: report},
	}

	for _, f := range crashFacts(gs) {
		if f.label != "report" {
			continue
		}
		if strings.ContainsAny(f.value, `/\`) {
			t.Errorf("the report row carries a path, not a file name: %q", f.value)
		}
	}
	if got := crashFolder(gs); got != deep {
		t.Errorf("crashFolder = %q, want %q", got, deep)
	}
}

// TestAnElisionKeepsBothEnds holds why the cut comes out of the middle: the front of a path says
// whose machine it is and the back says which folder, and a tail cut leaves neither.
func TestAnElisionKeepsBothEnds(t *testing.T) {
	const str = "C:/aaaaaaaa/bbbbbbbb/cccccccc/dddddddd/ascend-duel"
	wide := func(s string) float64 { return float64(len([]rune(s))) }

	if got := elideMiddle(str, 100, wide); got != str {
		t.Errorf("a string that fits was cut anyway: %q", got)
	}

	got := elideMiddle(str, 24, wide)
	if wide(got) > 24 {
		t.Fatalf("elided to %q, which is still %v wide", got, wide(got))
	}
	if !strings.HasPrefix(got, "C:/") {
		t.Errorf("elided to %q, which has lost the front of the path", got)
	}
	if !strings.HasSuffix(got, "ascend-duel") {
		t.Errorf("elided to %q, which has lost the folder it names", got)
	}
}

// TestTheCrashPageDrawsWithoutARun is the reason it is a screen rather than a dialog: it has to
// hold up when everything the game was carrying is gone or untrustworthy.
func TestTheCrashPageDrawsWithoutARun(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	scene := &CrashScene{}
	scene.Init(gs)
	if got := crashFacts(gs); len(got) == 0 {
		t.Fatal("a crash with nothing recorded drew an empty page")
	}
}
