package main

import (
	"errors"
	"log"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// The window opens at the internal resolution; Layout keeps that resolution fixed
	// whatever the window is resized to afterwards.
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Ascending Duel")
	ebiten.SetWindowClosingHandled(true)

	//Create our Game instance
	g := game.NewGame()
	g.GlobalState.ActiveDebug = true

	//Load assets into memory one time at startup
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()
	g.GlobalState.Combatants = data.LoadCombatants()

	// Widgets are no longer wired up here. Each scene builds its own in Init, so main
	// does not need to know which screens have buttons or what pressing them does.

	//Run game is the infinite loop. A deliberate quit comes back as game.ErrClosing,
	//which is a normal exit rather than a failure — anything else is a real error.
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, game.ErrClosing) {
		log.Fatal(err)
	}
}
