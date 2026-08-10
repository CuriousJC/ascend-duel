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

	// Style is how this combatant fights, for the ones the game plans for. It sits here
	// rather than on Duelist because it is not a rule the resolver reads: ResolveRound is
	// handed a queued set and never asks who chose it, which is exactly what keeps the
	// engine symmetric and lets a balance sim drive both sides.
	Style combat.PlanStyle

	// Sprite is **nil for the fighter**, and that is the normal case rather than a
	// missing asset. The fighter's sprite and health bar were replaced by the character
	// block on the combat screen, so nothing draws one — and once the last third-party
	// sheet went, giving it a placeholder would have meant shipping art to satisfy a
	// field nobody reads.
	//
	// Anything that draws a combatant has to check. See drawCombatant.
	Sprite     *ebiten.Image
	SpriteRect image.Rectangle
}

// NewCombatantFrom builds a combatant from a data record, slicing its sprite out of
// the given sheet. The caller resolves the sheet from the asset map because entities
// cannot import state — state imports entities.
//
// **A nil sheet is legal and produces a combatant with no sprite.** A record with an
// empty SpriteSheet is one nothing draws, which is true of the fighter; the alternative
// was carrying a picture purely so this line had something to slice.
func NewCombatantFrom(d data.CombatantData, sheet *ebiten.Image) *Combatant {
	rect := image.Rect(d.SpriteRect[0], d.SpriteRect[1], d.SpriteRect[2], d.SpriteRect[3])

	// An unknown or missing style falls back to brute rather than failing the load. A record
	// that predates the field still has to produce a fightable enemy.
	style, _ := combat.ParsePlanStyle(d.PlanStyle)

	c := &Combatant{
		Duelist: combat.Duelist{
			Con: d.Constitution,
			Str: d.Strength,
			Spd: d.Speed,
		},
		Style:      style,
		SpriteRect: rect,
	}
	if sheet != nil && !rect.Empty() {
		c.Sprite = sheet.SubImage(rect).(*ebiten.Image)
	}

	c.MaxLife = c.Con * LifePerCon
	c.CurrentLife = c.MaxLife

	return c
}
