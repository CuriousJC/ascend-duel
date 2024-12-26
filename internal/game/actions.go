package game

import (
	"fmt"
	//"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func PlayAction(g *Game) {
	fmt.Println("Play button clicked!")
	//TODO:  change the color of the button here?  maybe?  I think set the color and use it
	//  in the draw method
}

func SettingsAction(g *Game) {
	fmt.Println("Settings button clicked!")
}

func ExitAction(g *Game) {
	fmt.Println("Exit button clicked!")
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("frame sx: %d, sy: %d", sx, sy))
}
