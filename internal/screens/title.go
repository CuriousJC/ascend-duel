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

	//TODO:  Move this to a shape or a common or something
	// button creation using vector and new 100,100 image with text being placed upon it
	img := ebiten.NewImage(100, 100)
	vector.DrawFilledRect(img, 0, 0, 100, 100, color.RGBA{R: 255, G: 0, B: 0, A: 255}, false)
	vector.StrokeCircle(img, 50, 50, 25, 5, color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	centerButtonTextOp := &text.DrawOptions{}
	centerButtonTextOp.GeoM.Translate(50, 50)
	text.Draw(img, "BUTTON", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, centerButtonTextOp)
	if gs.CountSecond%2 == 0 {
		screen.DrawImage(img, &ebiten.DrawImageOptions{})
	} else {
		mover := &ebiten.DrawImageOptions{}
		mover.GeoM.Invert()
		mover.GeoM.Translate(200, 200)
		screen.DrawImage(img, mover)
	}

}
