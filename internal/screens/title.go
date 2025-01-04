package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

func UpdateTitleScreen(gs *state.GlobalState) error {

	// Update our button while updating our screen
	systems.UpdateButton(gs.HelloButton)

	// TODO: systems.UpdateButton(gs.ExitButton)

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

	ThirdTextOp := &text.DrawOptions{}
	ThirdTextOp.GeoM.Translate(0, 200)
	text.Draw(screen, "BOUNDS TESTING", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, ThirdTextOp)

	// Define where our button will be on the screen and then draw it
	gs.HelloButton.ScreenX = 300
	gs.HelloButton.ScreenY = 300
	systems.DrawButton(gs, screen, gs.HelloButton)
}
