package systems

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	mouseOver := image.Pt(gs.MouseX, gs.MouseY).In(buttonLocation)

	// A click is a press and a release on the same button.  inpututil reports the
	// tick the state changed, where ebiten.IsMouseButtonPressed reports the button
	// being held, which would fire OnClick on every tick of a held mouse.
	if mouseOver && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		button.PressedInside = true
	}

	// Firing on the release rather than the press lets a misclick be taken back by
	// dragging off the button before letting go
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if mouseOver && button.PressedInside && button.OnClick != nil {
			button.OnClick()
		}
		button.PressedInside = false
	}

	switch {
	case mouseOver && button.PressedInside:
		button.State = models.ButtonStatePressed
	case mouseOver:
		button.State = models.ButtonStateHovered
	default:
		button.State = models.ButtonStateNormal
	}
}

// defaultButtonColor is the full-strength fill for any button that does not name its
// own. It is the old olive ramp's brightest step, so buttons that never set a colour
// land within a shade or two of where they have always looked.
var defaultButtonColor = color.RGBA{R: 95, G: 95, B: 40, A: 255}

// disabledButtonColor is flat grey for every button. A disabled control should read as
// unavailable first and as itself second, so it deliberately ignores BaseColor.
var disabledButtonColor = color.RGBA{R: 35, G: 35, B: 35, A: 255}

// How bright each state draws, as a percentage of the button's full colour. Resting at
// two thirds means hover and press have somewhere to climb to.
const (
	normalStrength  = 65
	hoveredStrength = 82
	pressedStrength = 100
)

// buttonStateColor picks the fill for a button's current state by dimming its colour
// rather than adding to it. Adding a fixed step to every channel walks a saturated
// colour toward white — crimson hovered to a washed-out pink instead of a brighter red,
// and a colour already near 255 had nowhere to go at all. Scaling keeps the hue and
// lets the button light up to exactly the colour it names when pressed.
func buttonStateColor(button *models.Button) color.RGBA {
	full := button.BaseColor
	if full.A == 0 {
		full = defaultButtonColor
	}

	switch button.State {
	case models.ButtonStateHovered:
		return ColorAtStrength(full, hoveredStrength)
	case models.ButtonStatePressed:
		return ColorAtStrength(full, pressedStrength)
	case models.ButtonStateDisabled:
		return disabledButtonColor
	default:
		return ColorAtStrength(full, normalStrength)
	}
}

// ColorAtStrength scales each colour channel to pct of its full value, leaving alpha
// alone so dimming never turns into fading out. Exported because dimming one named
// colour is how anything else gets a matching background without picking a second
// colour by hand.
func ColorAtStrength(c color.RGBA, pct int) color.RGBA {
	scale := func(v uint8) uint8 { return uint8(int(v) * pct / 100) }
	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: c.A}
}

// DrawButton uses
func DrawButton(gs *state.GlobalState, screen *ebiten.Image, button *models.Button) {

	img := ebiten.NewImage(button.Width, button.Height)
	vector.DrawFilledRect(img, 0, 0, float32(button.Width), float32(button.Height), buttonStateColor(button), false)

	// Text is centred by alignment against the button's midpoint rather than by a fixed
	// offset. The old hardcoded Translate(50, 50) only landed correctly on a button of
	// one particular size and put the label off the bottom edge of a shorter one.
	centerButtonTextOp := &text.DrawOptions{}
	centerButtonTextOp.GeoM.Translate(float64(button.Width)/2, float64(button.Height)/2)
	centerButtonTextOp.PrimaryAlign = text.AlignCenter
	centerButtonTextOp.SecondaryAlign = text.AlignCenter
	text.Draw(img, button.Text, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, centerButtonTextOp)

	//Locate our button according to screen coords and adjust to use the center point for translation
	screenLocation := &ebiten.DrawImageOptions{}
	centerX := float64(button.ScreenX) - float64(button.Width)/2
	centerY := float64(button.ScreenY) - float64(button.Height)/2
	screenLocation.GeoM.Translate(centerX, centerY)
	screen.DrawImage(img, screenLocation)

}
