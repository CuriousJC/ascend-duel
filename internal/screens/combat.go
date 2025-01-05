package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

func UpdateCombatScreen(gs *state.GlobalState) error {

	//Handle stuff happening on the combat screen

	return nil
}

func DrawCombatScreen(gs *state.GlobalState, screen *ebiten.Image) {

	screen.Fill(color.RGBA{
		R: 50,
		G: 50,
		B: 50,
		A: 255,
	})

	combatHeadingOp := &text.DrawOptions{}
	combatHeadingOp.GeoM.Translate(50, 50)
	text.Draw(screen, "Duel to the Top of the Pyramid", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, combatHeadingOp)

}
