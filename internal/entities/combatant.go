package entities

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
)

type Combatant struct {
	Con          int           // Constitution
	Str          int           // Strength
	Spd          int           // Speed
	Life_max     int           // Maximum life
	Life_current int           // Current life
	Sprite       *ebiten.Image // Sprite image
	SpriteRect   image.Rectangle
}

// NewButton creates a new button with the given width, height, and text
func NewCombatant() *Combatant {
	return &Combatant{
		Sprite: nil,
	}
}
