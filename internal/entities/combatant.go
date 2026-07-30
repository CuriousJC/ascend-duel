package entities

import (
	"image"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/hajimehoshi/ebiten/v2"
)

// LifePerCon converts Constitution into maximum life. Placeholder value — this is
// the first real game rule, kept as one constant so it is cheap to tune.
const LifePerCon = 5

// Combatant is a duelist that can be drawn. The stats live in the embedded
// combat.Duelist so the rules engine can take them without ever seeing a sprite;
// the promoted fields mean gs.Fighter.Str and friends still read the same.
type Combatant struct {
	combat.Duelist

	Sprite     *ebiten.Image // Sprite image
	SpriteRect image.Rectangle
}

// NewCombatantFrom builds a combatant from a data record, slicing its sprite out of
// the given sheet. The caller resolves the sheet from the asset map because entities
// cannot import state — state imports entities.
func NewCombatantFrom(d data.CombatantData, sheet *ebiten.Image) *Combatant {
	rect := image.Rect(d.SpriteRect[0], d.SpriteRect[1], d.SpriteRect[2], d.SpriteRect[3])

	c := &Combatant{
		Duelist: combat.Duelist{
			Con: d.Constitution,
			Str: d.Strength,
			Spd: d.Speed,
		},
		SpriteRect: rect,
		Sprite:     sheet.SubImage(rect).(*ebiten.Image),
	}

	c.MaxLife = c.Con * LifePerCon
	c.CurrentLife = c.MaxLife

	return c
}
