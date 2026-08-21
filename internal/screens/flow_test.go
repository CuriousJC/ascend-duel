package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The phase-to-scene table. It walks a map and a small list and creates nothing, so it runs
// headless like the rest of this package's narrow exception — see doc.go.

func TestEveryRegisteredPhaseNamesAScreenTheGameHas(t *testing.T) {
	// **A phase pointing at a screen nobody registered is a black screen**, because game.scene
	// falls back to the title and the run would sit there with its own idea of where it is. The
	// registry itself lives in internal/game and cannot be imported from here without a cycle, so
	// what is checkable is that every value names a real ActiveScreen.
	for phase, screen := range phaseScreens {
		if screen.String() == "Unknown" {
			t.Errorf("phase %s maps to an ActiveScreen the game does not have", phase)
		}
	}
}

func TestTheFightAndTheRewardBothHaveAScene(t *testing.T) {
	// The two stations that are built. This fails the day one is dropped from the table by
	// accident, which would otherwise show up as the loop silently skipping a whole screen.
	for _, p := range []session.Phase{session.PhaseFight, session.PhaseReward} {
		if _, ok := screenFor(p); !ok {
			t.Errorf("%s has no scene", p)
		}
	}
}

func TestAdvanceWalksPastAStationWithNoScene(t *testing.T) {
	// **This is what lets the loop name the shop and the room choice before either exists.** From
	// the reward, the next two stations have no scene, so advance has to keep walking and land on
	// the fight rather than pointing the game at nothing.
	gs := &state.GlobalState{Run: session.New(session.StartingDeck())}
	gs.Run.SetPhase(session.PhaseReward)

	advanceRun(gs)

	if gs.Run.Phase() != session.PhaseFight {
		t.Fatalf("advancing from the reward landed on %s", gs.Run.Phase())
	}
	if gs.ActiveScreen != state.Combat {
		t.Fatalf("the screen went to %s", gs.ActiveScreen)
	}
	if !gs.NewScreen {
		t.Error("the incoming scene was not asked to run its Init")
	}
}

func TestAdvanceFromTheFightReachesTheReward(t *testing.T) {
	gs := &state.GlobalState{Run: session.New(session.StartingDeck())}

	advanceRun(gs)

	if gs.Run.Phase() != session.PhaseReward {
		t.Fatalf("advancing from the fight landed on %s", gs.Run.Phase())
	}
	if gs.ActiveScreen != state.PostBattle {
		t.Fatalf("the screen went to %s", gs.ActiveScreen)
	}
}

func TestAdvanceWithNoRunChangesNothing(t *testing.T) {
	// A scene finishing before main has built a run is a wiring mistake, not a state the game can
	// reach — but it must not be a nil dereference in the middle of a frame.
	gs := &state.GlobalState{ActiveScreen: state.Title}

	advanceRun(gs)

	if gs.ActiveScreen != state.Title || gs.NewScreen {
		t.Fatal("advance moved a game with no run")
	}
}
