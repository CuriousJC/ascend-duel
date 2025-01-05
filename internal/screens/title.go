package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"image/color"
)

func UpdateTitleScreen(gs *state.GlobalState) error {

	// Update our button while updating our screen
	systems.UpdateButton(gs, gs.CombatButton)
	systems.UpdateButton(gs, gs.SettingsButton)
	systems.UpdateButton(gs, gs.ExitButton)

	return nil
}

func DrawTitleScreen(gs *state.GlobalState, screen *ebiten.Image) {

	screen.Fill(color.RGBA{
		R: 109,
		G: 141,
		B: 138,
		A: 255,
	})

	//TITLE
	//
	var title *ebiten.Image
	if gs.CountSecond > 9 && gs.CountSecond%10 == 0 {
		title = gs.Assets["titleEaster_png"]
	} else {
		title = gs.Assets["title_png"]
	}
	bounds := title.Bounds()
	imageCenterX := float64(bounds.Dx()) / 2
	imageCenterY := float64(bounds.Dy()) / 2
	op := &colorm.DrawImageOptions{}
	scaleFactor := 0.75
	op.GeoM.Scale(scaleFactor, scaleFactor)
	op.GeoM.Translate(float64(gs.HalfwayX)-imageCenterX*scaleFactor, 150-imageCenterY*scaleFactor)
	hue := float64(1)
	saturation := float64(1)
	value := float64(1)
	var c colorm.ColorM
	c.ChangeHSV(hue, saturation, value)
	colorm.DrawImage(screen, title, c, op)

	//BUTTONS
	//
	// Define where our button will be on the screen and then draw it
	gs.CombatButton.ScreenX = gs.HalfwayX
	gs.CombatButton.ScreenY = gs.FirstThirdY
	systems.DrawButton(gs, screen, gs.CombatButton)

	gs.SettingsButton.ScreenX = gs.HalfwayX
	gs.SettingsButton.ScreenY = gs.FirstThirdY + 150
	systems.DrawButton(gs, screen, gs.SettingsButton)

	gs.ExitButton.ScreenX = gs.HalfwayX
	gs.ExitButton.ScreenY = gs.FirstThirdY + 300
	systems.DrawButton(gs, screen, gs.ExitButton)
}
