package screens

// **The dev tools: views a developer wants and a player must never be offered, behind one switch in
// the profile.**
//
// `devTools` in profile.json is the switch, and nothing in the game sets it — it is edited by hand.
// With it on, the settings screen grows a dev column with one toggle per tool; with it off, every
// tool is off whatever the toggles say, and the column is not drawn. **One binary is both the dev
// build and the shipped one**, so no build tag decides what a player can reach.
//
// **A dev tool is a view and never a rule.** None of them may change an outcome — the same rule the
// game-speed setting is under — which is why a run with them on still earns achievements. A new one
// is a field on profile.DevSettings, a line in ApplyDevTools, and a row in devRows.

import (
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// ApplyDevTools puts the profile's dev tools in force: each one on only while devTools is.
func ApplyDevTools(gs *state.GlobalState) {
	var dev profile.DevSettings
	if gs.Profile != nil && gs.Profile.Settings.DevTools {
		dev = gs.Profile.Settings.Dev
	}
	gs.DebugGameplay = dev.PerfectInformation
	gs.DebugAnimations = dev.AnimationGallery
	gs.RevealCompendium = dev.RevealCompendium
}

// devToolsOn reports whether the profile has the dev column turned on.
func devToolsOn(gs *state.GlobalState) bool {
	return gs.Profile != nil && gs.Profile.Settings.DevTools
}

// reveals reports whether the Compendium entry or achievement keyed by key reads as found whatever
// the profile holds: a revealing scenario, or the dev tool.
func reveals(gs *state.GlobalState, key string) bool {
	return scenario.Reveals(key) || (gs != nil && gs.RevealCompendium)
}

// devRow is one toggle in the settings screen's dev column: its label and the switch it flips.
type devRow struct {
	label string
	on    func(d *profile.DevSettings) *bool
}

// devRows is the dev column, top to bottom. **The label names the tool, and the latch says whether it
// is on** — the fullscreen toggle's rule.
var devRows = []devRow{
	{"REVEAL COMPENDIUM", func(d *profile.DevSettings) *bool { return &d.RevealCompendium }},
	{"ANIMATION GALLERY", func(d *profile.DevSettings) *bool { return &d.AnimationGallery }},
	{"PERFECT INFORMATION", func(d *profile.DevSettings) *bool { return &d.PerfectInformation }},
}
