package game

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// This package links Ebitengine, so on Linux the test binary wants a display even though nothing
// here opens one — CI runs the whole step under `xvfb-run -a`. See CLAUDE.md.

// TestAPanicBecomesAScreenRatherThanAnExit is the point of the whole feature: the loop keeps
// running, the crashed scene is never asked for again, and what the player gets is a page saying
// what happened rather than a window disappearing.
func TestAPanicBecomesAScreenRatherThanAnExit(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	g.GlobalState.ActiveScreen = state.Combat
	g.GlobalState.Version = "v9.9.9"

	g.crash("something came apart", []byte("a stack"))

	if g.GlobalState.ActiveScreen != state.Crashed {
		t.Fatalf("the game is on %v, want the crash screen", g.GlobalState.ActiveScreen)
	}
	if !g.GlobalState.NewScreen {
		t.Fatal("the crash screen was not asked to init")
	}
	if g.GlobalState.Crash == nil {
		t.Fatal("nothing was recorded for the screen to draw")
	}
	if g.GlobalState.Crash.Panic != "something came apart" {
		t.Fatalf("panic = %q", g.GlobalState.Crash.Panic)
	}
	if g.GlobalState.Crash.Path == "" {
		t.Fatal("no report was written into a store that can be written to")
	}
	if !strings.HasSuffix(g.GlobalState.Crash.Path, ".json") {
		t.Fatalf("the report went to %q", g.GlobalState.Crash.Path)
	}
}

// TestACrashStandsEverythingElseDown holds the rule that the crash screen is the only thing on the
// screen. Each of these draws over whatever is underneath it, and what is underneath is the thing
// that just went wrong.
func TestACrashStandsEverythingElseDown(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	g.GlobalState.EarnedThisSession = []string{"first-steps"}
	g.GlobalState.ModalOpen = true
	g.GlobalState.InputGated = true
	g.ledger.Toggle()

	g.crash("boom", nil)

	if len(g.GlobalState.EarnedThisSession) != 0 {
		t.Error("an achievement toast is still queued over the crash screen")
	}
	if g.ledger.IsOpen() {
		t.Error("the ledger is still open over the crash screen")
	}
	if g.GlobalState.ModalOpen || g.GlobalState.InputGated {
		t.Error("a scene's dialog or the tutorial's shield survived the crash")
	}
	if chromeShowing(g.GlobalState) {
		t.Error("the frame is still drawing controls on the crash screen")
	}
}

// TestASecondPanicQuits is what stops a crash screen that is itself broken writing a report about
// itself once a frame for as long as the window is open.
func TestASecondPanicQuits(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	g.crashed = true

	func() {
		defer g.recoverFrame()
		panic("again")
	}()

	if !g.GlobalState.ShouldClose {
		t.Fatal("a second panic left the game running")
	}
	if g.GlobalState.ActiveScreen == state.Crashed {
		t.Fatal("a second panic wrote a second report")
	}
}

// TestANoticeWaitsOutADuel holds the one thing that separates this box from the achievement toast:
// it is news about a file, and a file is not worth stopping a fight over.
func TestANoticeWaitsOutADuel(t *testing.T) {
	gs := &state.GlobalState{ActiveScreen: state.Combat}
	if noticeAllowed(gs) {
		t.Error("a notice would have gone up over a duel")
	}

	gs.ActiveScreen = state.Crashed
	if noticeAllowed(gs) {
		t.Error("a notice would have gone up over the crash screen")
	}

	gs.ActiveScreen = state.Shop
	if !noticeAllowed(gs) {
		t.Error("a notice was held back between stations, where it belongs")
	}
}

// TestTheCrashReportCarriesWhatWentWrongBefore is the fourth tier: the thing that finally went
// wrong is rarely the first thing that did.
func TestTheCrashReportCarriesWhatWentWrongBefore(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	crashlog.Note("a save went wrong first")

	g.crash("boom", nil)

	found := false
	for _, p := range crashlog.Problems() {
		if strings.Contains(p.What, "a save went wrong first") {
			found = true
		}
	}
	if !found {
		t.Fatal("the earlier problem was not kept for the report")
	}
}
