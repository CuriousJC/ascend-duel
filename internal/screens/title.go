package screens

//TODO:  Add some buttons on to the screen that do stuff.  Maybe start with "Exit"

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

	ThirdTextOp := &text.DrawOptions{}
	ThirdTextOp.GeoM.Translate(0, 200)
	text.Draw(screen, "BOUNDS TESTING", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, ThirdTextOp)

	vector.DrawFilledRect(screen, 100, 100, 200, 200, color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)

	vector.StrokeCircle(screen, 300, 300, 50, 5, color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)

	//TODO:  Keep playing with vector stuff:
	//https://ebitengine.org/en/examples/vector.html

}
