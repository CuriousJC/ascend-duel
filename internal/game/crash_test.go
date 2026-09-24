package game

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
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

	g.crash("something came apart", []byte("a stack"), nil)

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

	g.crash("boom", nil, nil)

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

	g.crash("boom", nil, nil)

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

// quietScene is a screen with nothing to say: it implements ui.Scene and not ui.Reporter, which is
// every screen in the game but one.
type quietScene struct{}

func (quietScene) Init(*state.GlobalState)                {}
func (quietScene) Update(*state.GlobalState) error        { return nil }
func (quietScene) Draw(*state.GlobalState, *ebiten.Image) {}

// talkativeScene implements ui.Reporter.
type talkativeScene struct{ quietScene }

func (talkativeScene) Report() map[string]any { return map[string]any{"round": 3} }

// brokenScene is the case the guard exists for: the scene being asked to describe itself is the one
// that has just panicked.
type brokenScene struct{ quietScene }

func (brokenScene) Report() map[string]any { panic("not now") }

// TestASceneWithNothingToSayIsNotAsked holds the optional half of ui.Reporter: a new screen is not
// broken by not having one, it simply contributes no tier.
func TestASceneWithNothingToSayIsNotAsked(t *testing.T) {
	if got := sceneState(quietScene{}); got != nil {
		t.Fatalf("a scene that implements nothing said %v", got)
	}
	got := sceneState(talkativeScene{})
	if got == nil || got["round"] != 3 {
		t.Fatalf("scene = %v, want what the screen said", got)
	}
}

// TestASceneThatPanicsDescribingItselfCostsOnlyItsOwnTier is the rule every reader in crash.go is
// under, at its sharpest: this one calls a method on the exact object that just came apart.
func TestASceneThatPanicsDescribingItselfCostsOnlyItsOwnTier(t *testing.T) {
	if got := sceneState(brokenScene{}); got != nil {
		t.Fatalf("scene = %v, want nothing at all", got)
	}
}

// TestTheSceneTierReachesTheReport is the wire: the screen that was up when the game came apart is
// the screen the report describes.
func TestTheSceneTierReachesTheReport(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	g.GlobalState.ActiveScreen = state.Combat
	g.scenes[state.Combat] = talkativeScene{}

	g.crash("boom", nil, nil)

	raw, err := os.ReadFile(g.GlobalState.Crash.Path)
	if err != nil {
		t.Fatalf("reading the report: %v", err)
	}
	var report crashlog.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("the report is not readable: %v", err)
	}
	if report.Scene["round"] != float64(3) {
		t.Fatalf("scene = %v, want the screen's own account", report.Scene)
	}
}

// TestAnUpdatePanicHasNoPictureToTake is the absence rule, and it is why shotOf takes a screen that
// may be nil: a panic between two frames has no half-drawn screen to read.
func TestAnUpdatePanicHasNoPictureToTake(t *testing.T) {
	if got := shotOf(nil); got != nil {
		t.Fatalf("shotOf invented %d bytes with no frame to read", len(got))
	}
}
