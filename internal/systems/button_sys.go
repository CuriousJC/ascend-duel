package systems

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// UpdateButton may need to do something based on current user input
func UpdateButton(button *models.Button) {
	if button.State == models.ButtonStateDisabled {
		return
	}

	mouseX, mouseY := ebiten.CursorPosition()
	buttonLocation := button.Image.Bounds().Add(image.Pt(button.ScreenX, button.ScreenY))

	if image.Pt(mouseX, mouseY).In(buttonLocation.Bounds()) {
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

func DrawButton(g *state.GlobalState, screen *ebiten.Image, button *models.Button) {
	var buttonColor color.RGBA
	switch button.State {
	case models.ButtonStateNormal:
		buttonColor = color.RGBA{R: 55, G: 55, B: 0, A: 255}
	case models.ButtonStateHovered:
		buttonColor = color.RGBA{R: 75, G: 75, B: 20, A: 255}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HOVVVVVVVVER: %d, %d", g.MouseX, g.MouseY), 450, 700)
	case models.ButtonStatePressed:
		buttonColor = color.RGBA{R: 95, G: 95, B: 40, A: 255}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HELLLLLOOOOOOOOOOO: %d, %d", g.MouseX, g.MouseY), 450, 700)
	case models.ButtonStateDisabled:
		buttonColor = color.RGBA{R: 35, G: 35, B: 35, A: 255}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DISABBBBBLED: %d, %d", g.MouseX, g.MouseY), 450, 700)
	}

	img := ebiten.NewImage(button.Width, button.Height)
	vector.DrawFilledRect(img, 0, 0, float32(button.Width), float32(button.Height), buttonColor, false)
	vector.StrokeCircle(img, 50, 50, 25, 5, color.RGBA{R: 0, G: 255, B: 0, A: 255}, false)
	centerButtonTextOp := &text.DrawOptions{}
	centerButtonTextOp.GeoM.Translate(50, 50)
	text.Draw(img, button.Text, &text.GoTextFace{Source: g.Fonts["kubasta"], Size: 20}, centerButtonTextOp)

	screenLocation := &ebiten.DrawImageOptions{}
	screenLocation.GeoM.Translate(float64(button.ScreenX), float64(button.ScreenY))
	screen.DrawImage(img, screenLocation)

}
