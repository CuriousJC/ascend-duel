package screens

//TODO:  Add some buttons on to the screen that do stuff.  Maybe start with "Exit"

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"image/color"
)

func UpdateTitleScreen(gs *state.GlobalState) error {

	return nil
}

func DrawTitleScreen(screen *ebiten.Image, gs *state.GlobalState) {

	screen.Fill(color.RGBA{
		R: 109,
		G: 141,
		B: 138,
		A: 255,
	})

	SecondTextOp := &text.DrawOptions{}
	SecondTextOp.GeoM.Translate(0, 400)
	SecondTextOp.LineSpacing = 30
	if gs.CountSecond%2 == 0 {
		text.Draw(screen, "TITLE SCREEN EVEN", &text.GoTextFace{Source: gs.Fonts["firaSansRegular"], Size: 20}, SecondTextOp)
	} else {
		text.Draw(screen, "TITLE SCREEN ODD", &text.GoTextFace{Source: gs.Fonts["robotoFlexRegular"], Size: 20}, SecondTextOp)
	}

}
