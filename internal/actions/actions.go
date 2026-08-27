package actions

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// GoToCombat switches to the combat screen. NewScreen makes the incoming scene run
// its Init before its first Update.
func GoToCombat(gs *state.GlobalState) {
	gs.ActiveScreen = state.Combat
	gs.NewScreen = true
	fmt.Println("Combat button clicked!")
	gs.Debug1 = "Combat button clicked!"
	gs.Debug2 = gs.ActiveScreen.String()
}

// OpenSettings goes to the settings screen, remembering where the player was so Back can put
// them there. The chrome's cog does the same thing from every other screen — see game/chrome.go,
// which cannot call this because actions sits below it and the frame is not a scene.
func OpenSettings(gs *state.GlobalState) {
	gs.ReturnScreen = gs.ActiveScreen
	gs.ActiveScreen = state.Settings
	gs.NewScreen = true
}

// QuitGame asks the loop to stop. game.Update turns this into game.ErrClosing.
func QuitGame(gs *state.GlobalState) {
	fmt.Println("Exit Button Clicked")
	gs.ShouldClose = true
}
