package systems

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// UpdateButton handles setting the state of a button and calling it's OnClick even if needed
func UpdateButton(gs *state.GlobalState, button *models.Button) {
	if button.State == models.ButtonStateDisabled {
		return
	}

	// Adjust button bounds to use the center point as the reference
	centerX := button.ScreenX - button.Width/2
	centerY := button.ScreenY - button.Height/2
	buttonLocation := button.Image.Bounds().Add(image.Pt(centerX, centerY))

	if image.Pt(gs.MouseX, gs.MouseY).In(buttonLocation.Bounds()) {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			button.State = models.ButtonStatePressed
			if button.OnClick != nil {
				button.OnClick()
			}
		} else {
			button.State = models.ButtonStateHovered
		}
	} else {
		button.State = models.ButtonStateNormal
	}
}

// DrawButton uses
func DrawButton(gs *state.GlobalState, screen *ebiten.Image, button *models.Button) {

	//Create our button image
	var buttonColor color.RGBA
	switch button.State {
	case models.ButtonStateNormal:
		buttonColor = color.RGBA{R: 55, G: 55, B: 0, A: 255}
	case models.ButtonStateHovered:
		buttonColor = color.RGBA{R: 75, G: 75, B: 20, A: 255}
	case models.ButtonStatePressed:
		buttonColor = color.RGBA{R: 95, G: 95, B: 40, A: 255}
	case models.ButtonStateDisabled:
		buttonColor = color.RGBA{R: 35, G: 35, B: 35, A: 255}
	}
	img := ebiten.NewImage(button.Width, button.Height)
	vector.DrawFilledRect(img, 0, 0, float32(button.Width), float32(button.Height), buttonColor, false)

	//Place text on our button
	centerButtonTextOp := &text.DrawOptions{}
	centerButtonTextOp.GeoM.Translate(50, 50)
	text.Draw(img, button.Text, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, centerButtonTextOp)

	//Locate our button according to screen coords and adjust to use the center point for translation
	screenLocation := &ebiten.DrawImageOptions{}
	centerX := float64(button.ScreenX) - float64(button.Width)/2
	centerY := float64(button.ScreenY) - float64(button.Height)/2
	screenLocation.GeoM.Translate(centerX, centerY)
	screen.DrawImage(img, screenLocation)

}
