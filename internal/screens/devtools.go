package screens

// **The dev tools: views a developer wants and a player must never be offered, behind one switch in
// the profile.**
//
// `devTools` in profile.json is the switch, and nothing in the game sets it — it is edited by hand.
// **It is all or nothing**: on, every tool is on; off, none is, and nothing about them is drawn.
// **One binary is both the dev build and the shipped one**, so no build tag decides what a player
// can reach.
//
// **A dev tool is a view and never a rule.** None of them may change an outcome — the same rule the
// game-speed setting is under — which is why a run with them on still earns achievements. A new one
// reads `gs.DevTools`.

import (
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// ApplyDevTools puts the profile's dev tools in force.
func ApplyDevTools(gs *state.GlobalState) {
	gs.DevTools = gs.Profile != nil && gs.Profile.Settings.DevTools
}

// reveals reports whether the Compendium entry or achievement keyed by key reads as found whatever
// the profile holds: a revealing scenario, or the dev tools.
func reveals(gs *state.GlobalState, key string) bool {
	return scenario.Reveals(key) || (gs != nil && gs.DevTools)
}
