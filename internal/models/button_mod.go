package models

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type ButtonState int

const (
	ButtonStateNormal ButtonState = iota
	ButtonStateHovered
	ButtonStatePressed
	ButtonStateDisabled

	// ButtonStateLatched is the active member of a set of modes. **Darker than resting, where
	// pressed is brighter**: a momentary press is the button lighting up under the cursor, and
	// a latched one is a control that has been pushed in and stayed there. Reusing the pressed
	// appearance made the active mode read as a button the cursor was on top of.
	ButtonStateLatched
)

// Captures the data of the button but doesn't handle drawing or updating
type Button struct {
	Width   int           //Width of the button
	Height  int           //Height of the button
	ScreenX int           //X position of the button on the screen it will be drawn upon
	ScreenY int           //Y position of the button on the screen it will be drawn upon
	Image   *ebiten.Image //A button is an image, which has a rectangle and allows for "in" logic for a mouse click pointer
	Text    string
	State   ButtonState
	OnClick func()

	// BaseColor is the button at full strength — the colour it reaches when pressed.
	// It rests dimmer than this and brightens toward it on hover, so a button only has
	// to name the one colour it actually wants to be. The zero value means "use the
	// default", which is why buttons that never set it keep the original olive.
	// Disabled ignores it — a disabled button should not still read as its own colour.
	BaseColor color.RGBA

	// PressedInside records that the mouse went down while over this button, so
	// the release can tell a real click from a drag that happened to end here.
	PressedInside bool

	// TextSize is the label's point size. Zero means the default, which is what every
	// button that never sets it has always drawn at — a square button carrying a single
	// character wants a much bigger one than a button carrying a word.
	TextSize float64

	// Latched holds the button at its latched appearance while it is set, which is what
	// turns a momentary button into one of a set of modes: the active member stays marked
	// after the cursor has gone somewhere else.
	//
	// It is a general widget idea rather than a rule about one screen, which is why it
	// lives here and not on the scene that first wanted it. A latched button still fires
	// OnClick normally — pressing the active member is a real press, and what happens next
	// is the scene's business. Disabled still wins: a dead control reads as unavailable
	// first and as active second.
	Latched bool

	// What Image currently holds, so DrawButton can repaint it only when something
	// visible has changed rather than every frame.
	//
	// A button's face is a function of exactly these three things plus its size, so
	// comparing them is a complete test for "is the cached picture still right". Painted
	// starts false, which is what forces the first paint — a zero-valued Button has an
	// Image full of nothing and no state worth trusting.
	//
	// These are data about the cached render, not behaviour, so they belong on the struct
	// like everything else here. Only DrawButton writes them.
	Painted         bool
	PaintedState    ButtonState
	PaintedText     string
	PaintedTextSize float64
	PaintedColor    color.RGBA
}

// NewButton creates a new button with the given width, height, and text
func NewButton(width, height int, text string, onClick func()) *Button {
	return &Button{
		Width:   width,
		Height:  height,
		ScreenX: 0,
		ScreenY: 0,
		Image:   ebiten.NewImage(width, height),
		Text:    text,
		State:   ButtonStateNormal,
		OnClick: onClick,
	}
}
