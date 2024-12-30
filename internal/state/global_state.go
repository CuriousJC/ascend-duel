// Description: state is the shared state passed to all the other components of the game.
package state

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type GlobalState struct {
	Count               int
	CountSecond         int
	MouseX              int
	MouseY              int
	ButtonHue128        int
	ButtonSaturation128 int
	ButtonValue128      int
	ActiveDebug         bool
	Assets              map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts               map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct
	Buttons             []Button
}

func NewGlobalState() *GlobalState {
	return &GlobalState{
		Count:               0,
		CountSecond:         0,
		MouseX:              0,
		MouseY:              0,
		ButtonHue128:        0,
		ButtonSaturation128: 0,
		ButtonValue128:      0,
		ActiveDebug:         true,
		Assets:              make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:               make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
		Buttons:             []Button{},
	}
}

// todo: move to models?
type Button struct {
	Bounds       Rectangle
	Text         string
	Pressed      bool
	IsJustActive bool
}

func CreateButton(bounds Rectangle, text string) *Button {
	return &Button{
		Bounds:       bounds,
		Text:         text,
		Pressed:      false,
		IsJustActive: false,
	}
}

type Size struct {
	Width  int
	Height int
}

type Point struct {
	X int
	Y int
}

type Rectangle struct {
	Point
	Size
}
