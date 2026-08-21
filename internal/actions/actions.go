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

// OpenSettings is a stub. There is no settings screen yet, and deliberately no
// NewScreen here — setting it without changing screen would cost a skipped Draw and
// a pointless re-Init of the screen already showing.
func OpenSettings(gs *state.GlobalState) {
	fmt.Println("Settings Button Clicked!")
}

// QuitGame asks the loop to stop. game.Update turns this into game.ErrClosing.
func QuitGame(gs *state.GlobalState) {
	fmt.Println("Exit Button Clicked")
	gs.ShouldClose = true
}
