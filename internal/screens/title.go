package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"

	"image/color"
)

// InitTitleScreen places the buttons. Positioning belongs here rather than in Draw:
// Update hit-tests against ScreenX/Y, so a Draw-time assignment leaves the first frame
// testing against zeroes and goes stale any frame Ebiten chooses to skip Draw. The
// internal resolution is fixed, so these coordinates only need computing once.
func InitTitleScreen(gs *state.GlobalState) {
	gs.CombatButton.ScreenX = gs.HalfwayX
	gs.CombatButton.ScreenY = gs.FirstThirdY

	gs.SettingsButton.ScreenX = gs.HalfwayX
	gs.SettingsButton.ScreenY = gs.FirstThirdY + 150

	gs.ExitButton.ScreenX = gs.HalfwayX
	gs.ExitButton.ScreenY = gs.FirstThirdY + 300
}

func UpdateTitleScreen(gs *state.GlobalState) error {

	if gs.NewScreen {
		InitTitleScreen(gs)
		gs.NewScreen = false
	}

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
	// Positions are set in InitTitleScreen; Draw only draws.
	systems.DrawButton(gs, screen, gs.CombatButton)
	systems.DrawButton(gs, screen, gs.SettingsButton)
	systems.DrawButton(gs, screen, gs.ExitButton)
}
