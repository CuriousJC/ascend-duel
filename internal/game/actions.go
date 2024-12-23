package game

import (
	"fmt"
	//"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func PlayAction() {
	fmt.Println("Play button clicked!")
}

func SettingsAction() {
	fmt.Println("Settings button clicked!")
}

func ExitAction() {
	fmt.Println("Exit button clicked!")
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("frame sx: %d, sy: %d", sx, sy))
}
