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

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
