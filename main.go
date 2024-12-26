package main

import (
	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Ascending Duel")

	g := game.NewGame()
	g.Assets = assets.LoadAssets()
	g.Fonts = assets.LoadFonts()

	buttonPlay := game.NewButton(g, 100, 50, 120, 40, "Play", game.PlayAction)
	g.Buttons = append(g.Buttons, buttonPlay)

	buttonSettings := game.NewButton(g, 100, 100, 120, 40, "Settings", game.SettingsAction)
	g.Buttons = append(g.Buttons, buttonSettings)

	buttonExit := game.NewButton(g, 100, 150, 120, 40, "Exit", game.ExitAction)
	g.Buttons = append(g.Buttons, buttonExit)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
