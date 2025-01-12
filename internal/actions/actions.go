package actions

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/state"
)

func CombatButtonAction(gs *state.GlobalState) {
	gs.ActiveScreen = state.Combat
	gs.NewScreen = true
	fmt.Println("Combat button clicked!")
	gs.Debug1 = "Combat button clicked!"
	gs.Debug2 = gs.ActiveScreen.String()
}

func SettingsButtonAction(gs *state.GlobalState) {
	fmt.Println("Settings Button Clicked!")
	gs.NewScreen = true
}

func ExitButtonAction(gs *state.GlobalState) {
	fmt.Println("Exit Button Clicked")
	gs.ShouldClose = true
}
