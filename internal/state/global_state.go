// Description: state is the shared state passed to all the other components of the game.
package state

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GlobalState is what is genuinely shared: input, timing, layout, loaded resources,
// and which screen is active. It is threaded by pointer into every scene.
//
// A screen's own working state does NOT belong here — that lives on the scene that
// owns it (see screens.Scene). This struct previously carried the combat screen's
// duel log, playback cursor and combatants, which meant every screen could see them
// and none of them were anyone else's business.
type GlobalState struct {
	//Global Game Stuff
	ActiveDebug    bool
	Debug1, Debug2 string
	ActiveScreen   ActiveScreen
	NewScreen      bool
	Count          int
	CountSecond    int
	MouseX         int
	MouseY         int
	ShouldClose    bool

	//Layout
	ScreenWidth   int
	ScreenHeight  int
	FirstThirdX   int
	SecondThirdX  int
	FirstThirdY   int
	SecondThirdY  int
	FirstQuarterX int
	ThirdQuarterX int
	FirstQuarterY int
	ThirdQuarterY int
	HalfwayX      int
	HalfwayY      int

	//Data
	Combatants map[string]data.CombatantData

	//Assets
	Assets map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts  map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct

}

// NewGlobalState used at the start of the game to start us off
func NewGlobalState() *GlobalState {
	return &GlobalState{
		ActiveScreen: Title,
		NewScreen:    true,
		Assets:       make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:        make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
	}
}

type ActiveScreen int

const (
	Title ActiveScreen = iota
	Ascend
	Combat
	Credits
)

func (active ActiveScreen) String() string {
	switch active {
	case Title:
		return "Title"
	case Ascend:
		return "Ascend"
	case Combat:
		return "Combat"
	case Credits:
		return "Credits"
	default:
		return "Unknown"
	}
}
