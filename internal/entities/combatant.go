package entities

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Combatant struct {
	Con          int           // Constitution
	Str          int           // Strength
	Spd          int           // Speed
	Life_max     int           // Maximum life
	Life_current int           // Current life
	Sprite       *ebiten.Image // Sprite image
}

// NewButton creates a new button with the given width, height, and text
func NewCombatant(con, str, spd int) *Combatant {
	return &Combatant{
		Con:          con,
		Str:          str,
		Spd:          spd,
		Life_max:     con * 10,
		Life_current: con * 10,
		Sprite:       nil,
	}
}
