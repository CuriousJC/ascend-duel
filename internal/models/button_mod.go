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
