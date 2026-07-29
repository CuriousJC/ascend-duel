package main

import (
	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2"
	"log"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Ascending Duel")
	ebiten.SetWindowClosingHandled(true)

	//Create our Game instance
	g := game.NewGame()
	g.GlobalState.ActiveDebug = true

	//Load assets into memory one time at startup
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()
	g.GlobalState.Combatants = data.LoadCombatants()

	// Buttons are assigned actions as part of the initial creation
	g.GlobalState.CombatButton = models.NewButton(275, 100, "Combat", func() { actions.CombatButtonAction(g.GlobalState) })
	g.GlobalState.SettingsButton = models.NewButton(275, 100, "Settings", func() { actions.SettingsButtonAction(g.GlobalState) })
	g.GlobalState.ExitButton = models.NewButton(275, 100, "Exit", func() { actions.ExitButtonAction(g.GlobalState) })

	//Run game is the infinite loop
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
