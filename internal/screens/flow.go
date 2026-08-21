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
// **A phase with no scene is skipped, not drawn blank.** The room choice is in the loop already and
// has no scene yet, so `screenFor` reports that it has none and `advance` keeps walking. **The shop
// is what that bought** *(2026-08-21)*: it was walked past for four days and joining the loop was
// one line in the table below, with no existing scene edited.

import (
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// phaseScreens is which scene draws which station of the run.
//
// **Absent means not built.** A phase left out of this table is walked past by advance, which is
// how the loop stays complete while two of its four stations are still to be written.
var phaseScreens = map[session.Phase]state.ActiveScreen{
	session.PhaseFight:  state.Combat,
	session.PhaseReward: state.PostBattle,
	session.PhaseShop:   state.Shop,
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
		if next, ok := screenFor(gs.Run.Phase()); ok {
			gs.ActiveScreen = next
			gs.NewScreen = true
			return
		}
	}
}
