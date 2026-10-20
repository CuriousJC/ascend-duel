package screens

// **The one place a scene hands the run on to the next one.**
//
// Every scene in the loop used to name its successor: the combat screen set PostBattle, the
// post-battle screen set Combat. Four scenes each knowing what comes after them is four files to
// edit to insert a fifth, and nothing that answers "what is the loop" without grepping for
// assignments to ActiveScreen.
//
// **The run owns where it is; this file owns what draws it.** `session.Phase` is the station, and
// `advance` moves the run on and points the game at whichever scene shows the station it lands on.
// The mapping is this way round because `session` must not know a screen exists — see
// session/flow.go.
//
// **A phase with no scene is skipped, not drawn blank.** `screenFor` reports that it has none and
// `advance` keeps walking, so a station can be named in the loop before its screen exists and
// joining it later is one line in the table below, with no existing scene edited.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
)

// phaseScreens is which scene draws which station of the run.
//
// **Absent means not built.** A phase left out of this table is walked past by advance, which is
// how the loop stays complete while a station is still to be written.
var phaseScreens = map[session.Phase]state.ActiveScreen{
	session.PhaseFight:   state.Combat,
	session.PhaseReward:  state.PostBattle,
	session.PhaseShop:    state.Shop,
	session.PhasePortal:  state.Portal,
	session.PhaseVictory: state.Victory,
}

// screenFor is the scene that draws a phase, and whether one exists at all.
func screenFor(p session.Phase) (state.ActiveScreen, bool) {
	s, ok := phaseScreens[p]
	return s, ok
}

// advanceRun moves the run to the next station of its loop and puts that station's scene on screen.
//
// **It is what a scene calls when it is finished**, in place of naming a successor. A scene says
// "done"; where that leads is the run's business and this file's.
//
// **Phases with no scene are walked past**, so an unbuilt station costs nothing. The walk is
// bounded by the number of stations, so a table with nothing in it at all lands the game back on
// the scene it was already showing rather than looping forever.
func advanceRun(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}
	for i := 0; i < session.PhaseCount; i++ {
		gs.Run.Advance()
		takeEndlessPortal(gs)
		if next, ok := screenFor(gs.Run.Phase()); ok {
			gs.ActiveScreen = next
			gs.NewScreen = true

			// **The run is written down here and nowhere else** *(2026-08-25)*. This is the one
			// place a run changes station, and a station boundary is the only moment the run is
			// quiescent enough to snapshot — see save.go. Putting it after the scene is chosen
			// rather than before means a phase walked past for having no scene does not produce a
			// save nobody can resume into.
			saveRun(gs)
			return
		}
	}
}

// takeEndlessPortal walks the run through a portal the endless journey chooses, if one is open — the
// station the player no longer stands at. Taken before the run is saved, so a resume fights the
// realm the draw chose.
//
// **Not journaled**: the journal holds inputs, and this is a draw off the run code that a replay
// makes again.
func takeEndlessPortal(gs *state.GlobalState) {
	if !gs.Run.EndlessPortalDue() {
		return
	}
	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.EndlessPortal, gs.Run.Fight())))
	if key := gs.Run.TakeEndlessPortal(rng); key != "" {
		trace.Logf("flow", "endless journey walked through the portal to %s on realm %d", key, gs.Run.Realm())
	}
}
