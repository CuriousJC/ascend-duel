// Description: state is the shared state passed to all the other components of the game.
package state

import (
	"fmt"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GlobalState is the shared state that all components of the game use to know what to do
// and what they will act upon during changes
type GlobalState struct {
	ActiveDebug         bool
	ActiveScreen        string
	Count               int
	CountSecond         int
	MouseX              int
	MouseY              int
	ButtonHue128        int
	ButtonSaturation128 int
	ButtonValue128      int
	HelloButton         *models.Button //-- this is what I want to draw first on my title screen

	Assets map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts  map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct

	Debug1 string
	Debug2 string
	Debug3 string
	Debug4 string
}

// NewGlobalState used at the start of the game to start us off
func NewGlobalState() *GlobalState {
	return &GlobalState{
		ActiveDebug:         true,
		ActiveScreen:        "title",
		Count:               0,
		CountSecond:         0,
		MouseX:              0,
		MouseY:              0,
		ButtonHue128:        0,
		ButtonSaturation128: 0,
		ButtonValue128:      0,
		Assets:              make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:               make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
		HelloButton:         models.NewButton(300, 300, "Hello", func() { fmt.Println("Hello button clicked!") }),
	}
}
