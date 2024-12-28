package game

import (
	"fmt"
	//"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func PlayAction(g *Game) {
	fmt.Println("Play button clicked!")
	g.buttonHue128 += 10
	g.buttonSaturation128 += 10
	g.buttonValue128 += 10
}

func SettingsAction(g *Game) {
	fmt.Println("Settings button clicked!")
	g.buttonHue128 -= 10
	g.buttonSaturation128 -= 10
	g.buttonValue128 -= 10
}

func ExitAction(g *Game) {
	fmt.Println("Exit button clicked!")
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("frame sx: %d, sy: %d", sx, sy))
}
