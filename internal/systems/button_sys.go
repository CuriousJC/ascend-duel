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

	// A latched button holds its own appearance whatever the cursor is doing, so a set of
	// modes says which one is active. Applied after the switch rather than inside it because
	// hover and press are still real states underneath — the latch simply outranks them,
	// which is what stops the active mode flickering back to a resting face as the cursor
	// crosses it.
	if button.Latched {
		button.State = models.ButtonStateLatched
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
//
// **Latched is the one that goes the other way.** A mode that is on is drawn *darker* than
// the ones that are off, not brighter: hover and press already own the bright end of the
// ramp, and a latched button lit to full strength reads as a button the cursor is sitting
// on. Pushed in, rather than lit up.
const (
	normalStrength  = 65
	hoveredStrength = 82
	pressedStrength = 100
	latchedStrength = 38
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
	case models.ButtonStateLatched:
		return ColorAtStrength(full, latchedStrength)
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

// ColorToward moves a colour pct of the way to `ground`, which is **what dimming means when
// the background is light**.
//
// `ColorAtStrength` scales toward black, and that reads as "quieter" only because the screen
// it was written for is dark. On the Resolution pane's off-white ground it does the opposite:
// a swatch scaled to 55% comes out *darker* than the ground and so louder than the full-
// strength one beside it, which put the idle rows in front of the lit one. That was visible in
// a capture before it was understood, and it is the whole reason this exists.
//
// Neither scaling nor adding a fixed step can express this — the destination is not black and
// not white, it is whatever the surface happens to be. So it is a third operation rather than
// a tweak to the first, and `CLAUDE.md`'s "scale, never add" rule still governs widget *state*
// against a known ground.
func ColorToward(c, ground color.RGBA, pct int) color.RGBA {
	mix := func(from, to uint8) uint8 {
		return uint8(int(from) + (int(to)-int(from))*pct/100)
	}
	return color.RGBA{
		R: mix(c.R, ground.R),
		G: mix(c.G, ground.G),
		B: mix(c.B, ground.B),
		A: c.A,
	}
}

// DrawButton uses
// DrawButton blits the button's cached face onto the screen, repainting that face first if
// anything about it changed.
//
// **It used to allocate an *ebiten.Image every call.** Three buttons at 60fps is 180 new GPU
// textures a second to draw three rectangles that mostly do not change, and `Button.Image`
// was already sitting there at exactly the right size being used only for hit-test bounds.
// Now it is what gets painted, and a press repaints one button once instead of the screen
// repainting all of them forever.
func DrawButton(gs *state.GlobalState, screen *ebiten.Image, button *models.Button) {
	if needsPaint(button) {
		paintButton(gs, button)
	}

	//Locate our button according to screen coords and adjust to use the center point for translation
	screenLocation := &ebiten.DrawImageOptions{}
	centerX := float64(button.ScreenX) - float64(button.Width)/2
	centerY := float64(button.ScreenY) - float64(button.Height)/2
	screenLocation.GeoM.Translate(centerX, centerY)
	screen.DrawImage(button.Image, screenLocation)
}

// needsPaint reports whether the cached face is stale.
//
// The size check is not paranoia: Image is allocated once in NewButton, so a caller that
// changes Width or Height afterwards would otherwise get the old face stretched or clipped
// with no clue why. Comparing bounds catches it and paintButton reallocates.
func needsPaint(button *models.Button) bool {
	return !button.Painted ||
		button.PaintedState != button.State ||
		button.PaintedText != button.Text ||
		button.PaintedTextSize != button.TextSize ||
		button.PaintedColor != button.BaseColor ||
		button.Image == nil ||
		button.Image.Bounds().Dx() != button.Width ||
		button.Image.Bounds().Dy() != button.Height
}

// paintButton renders the face into Button.Image and records what it drew.
func paintButton(gs *state.GlobalState, button *models.Button) {
	if button.Image == nil ||
		button.Image.Bounds().Dx() != button.Width ||
		button.Image.Bounds().Dy() != button.Height {
		button.Image = ebiten.NewImage(button.Width, button.Height)
	}

	// Clear before repainting. The fill below is opaque and covers the whole face, but a
	// shorter label would otherwise leave the tail of a longer one behind it.
	button.Image.Clear()
	vector.DrawFilledRect(button.Image, 0, 0,
		float32(button.Width), float32(button.Height), buttonStateColor(button), false)

	// Text is centred by alignment against the button's midpoint rather than by a fixed
	// offset. The old hardcoded Translate(50, 50) only landed correctly on a button of
	// one particular size and put the label off the bottom edge of a shorter one.
	centerButtonTextOp := &text.DrawOptions{}
	centerButtonTextOp.GeoM.Translate(float64(button.Width)/2, float64(button.Height)/2)
	centerButtonTextOp.PrimaryAlign = text.AlignCenter
	centerButtonTextOp.SecondaryAlign = text.AlignCenter
	text.Draw(button.Image, button.Text,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: textSizeOf(button)}, centerButtonTextOp)

	button.Painted = true
	button.PaintedState = button.State
	button.PaintedText = button.Text
	button.PaintedTextSize = button.TextSize
	button.PaintedColor = button.BaseColor
}

// defaultButtonTextSize is what a button that names no size draws its label at, which is
// what every button in the game drew at when the size was written into paintButton.
const defaultButtonTextSize = 20

// textSizeOf resolves a button's label size, zero meaning the default — the same "use the
// default" convention BaseColor's zero alpha uses.
func textSizeOf(button *models.Button) float64 {
	if button.TextSize <= 0 {
		return defaultButtonTextSize
	}
	return button.TextSize
}
