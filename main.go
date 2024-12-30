package main

import (
	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/game"
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

	//LOAD
	g := game.NewGame()
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()

	//RUN
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
