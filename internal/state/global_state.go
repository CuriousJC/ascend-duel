// Description: state is the shared state passed to all the other components of the game.
package state

import (
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GlobalState is the shared state that all components of the game use to know what to do
// and what they will act upon during changes
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

	//Models
	CombatButton   *models.Button
	SettingsButton *models.Button
	ExitButton     *models.Button

	//Entities
	Fighter *entities.Combatant
	Enemy   *entities.Combatant

	//Assets
	Assets map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts  map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct

}

// NewGlobalState used at the start of the game to start us off
func NewGlobalState() *GlobalState {
	return &GlobalState{
		ActiveScreen: Combat,
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
