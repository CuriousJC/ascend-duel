package systems

// A button's authored face: which picture a button wears, and how one picture is drawn at any
// size and in every state.

import (
	"image"
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// The colors a button may name and have a face for. **A color is a picture**, one file each in
// assets/button/, so a button naming anything else falls back to the drawn bevel — and a new color
// is a face to author before it is a value to type. See docs/art/button_art_prompt.MD.
//
// **Red commits or leaves** — DUEL!, a sale, a dialog's way out, the answer that cannot be taken
// back. **Gray is the program rather than the fight** — the chrome, the safe answer, Leave and
// Skip. The rest are one caller or two apiece.
var (
	ButtonRed    = color.RGBA{R: 208, G: 52, B: 58, A: 255}
	ButtonGray   = color.RGBA{R: 92, G: 96, B: 108, A: 255}
	ButtonYellow = color.RGBA{R: 225, G: 200, B: 60, A: 255}
	ButtonBlue   = color.RGBA{R: 70, G: 130, B: 230, A: 255}
	ButtonPink   = color.RGBA{R: 235, G: 105, B: 170, A: 255}
)

// buttonFaces is the face each color wears, by asset key. The default olive is in it, so a button
// naming no color still wears a face.
var buttonFaces = map[color.RGBA]string{
	ButtonRed:          "button-red",
	ButtonGray:         "button-gray",
	ButtonYellow:       "button-yellow",
	ButtonBlue:         "button-blue",
	ButtonPink:         "button-pink",
	defaultButtonColor: "button-olive",
}

const disabledButtonFace = "button-disabled"

// The face's geometry at the size it is authored, in its own pixels. The caps hold all the
// rounding and the middle is identical column to column; the drop is the black the face stands on,
// which a pressed button sinks into.
const (
	buttonFaceCap  = 48
	buttonFaceDrop = 8
)

// How bright a face draws in each state, as a multiple of the picture. **The picture is the button
// at rest**, so rest is 1 and hover climbs above it. Pressed and latched are both "in" and sink the
// face into its own drop; latched is also dimmer, because a mode that is on is pushed in rather
// than lit up — hover and press own the bright end.
const (
	faceHoverScale   = 1.18
	faceLatchedScale = 0.7
)

// buttonFace is the picture a button wears, or nil for a button drawn with the bevel: a color
// with no face, or a face that failed to load.
func buttonFace(gs *state.GlobalState, button *models.Button) *ebiten.Image {
	if button.State == models.ButtonStateDisabled {
		return gs.Assets[disabledButtonFace]
	}
	full := button.BaseColor
	if full.A == 0 {
		full = defaultButtonColor
	}
	key, ok := buttonFaces[full]
	if !ok {
		return nil
	}
	return gs.Assets[key]
}

// faceDrop is how far a face of this height sinks when pressed, in screen pixels.
func faceDrop(face *ebiten.Image, h int) float64 {
	return math.Round(float64(buttonFaceDrop) * float64(h) / float64(face.Bounds().Dy()))
}

// drawButtonFace paints a face into the button's own image: scaled to its height, the middle
// stretched to its width, and moved and dimmed for its state.
func drawButtonFace(button *models.Button, face *ebiten.Image) {
	w, h := button.Width, button.Height
	sw, sh := face.Bounds().Dx(), face.Bounds().Dy()
	scale := float64(h) / float64(sh)
	// **Whole pixels, or the joins show.** A cap ending on a half pixel leaves one column that both
	// neighbouring pieces cover by half, and two half-covered draws do not add up to one opaque one —
	// the column comes out a visible seam.
	capW := math.Round(float64(buttonFaceCap) * scale)
	if 2*capW > float64(w) {
		capW = math.Floor(float64(w) / 2)
	}
	mid := float64(w) - 2*capW

	var dy float64
	var lum float32 = 1
	switch button.State {
	case models.ButtonStateHovered:
		lum = faceHoverScale
	case models.ButtonStatePressed:
		dy = faceDrop(face, h)
	case models.ButtonStateLatched:
		dy, lum = faceDrop(face, h), faceLatchedScale
	}

	piece := func(sx0, sx1 int, dx, dw float64) {
		if dw <= 0 || sx1 <= sx0 {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(dw/float64(sx1-sx0), scale)
		op.GeoM.Translate(dx, dy)
		op.ColorScale.Scale(lum, lum, lum, 1)
		op.Filter = ebiten.FilterLinear
		button.Image.DrawImage(face.SubImage(image.Rect(sx0, 0, sx1, sh)).(*ebiten.Image), op)
	}
	piece(0, buttonFaceCap, 0, capW)
	piece(buttonFaceCap, sw-buttonFaceCap, capW, mid)
	piece(sw-buttonFaceCap, sw, capW+mid, capW)
}

// How an icon button is drawn in each state. It has no drop to sink into, so "in" is the picture
// drawn a little smaller about its center; disabled is the picture dimmed, since there is no
// separate disabled file for each icon.
const (
	iconPressedScale = 0.92
	iconDisabledLum  = 0.45
)

// drawButtonIcon paints an icon button's picture into its own image: scaled whole to the button's
// size, never stretched in parts, and brightened, shrunk or dimmed for its state with the same
// figures a face uses.
func drawButtonIcon(button *models.Button, icon *ebiten.Image) {
	w, h := float64(button.Width), float64(button.Height)
	sw, sh := float64(icon.Bounds().Dx()), float64(icon.Bounds().Dy())

	size := 1.0
	var lum float32 = 1
	switch button.State {
	case models.ButtonStateHovered:
		lum = faceHoverScale
	case models.ButtonStatePressed:
		size = iconPressedScale
	case models.ButtonStateLatched:
		size, lum = iconPressedScale, faceLatchedScale
	case models.ButtonStateDisabled:
		lum = iconDisabledLum
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w*size/sw, h*size/sh)
	op.GeoM.Translate(w*(1-size)/2, h*(1-size)/2)
	op.ColorScale.Scale(lum, lum, lum, 1)
	op.Filter = ebiten.FilterLinear
	button.Image.DrawImage(icon, op)
}

// faceLabelShift is where a label sits on a face, against the button's center: up by half the
// drop, so it is centered on the body rather than on the body and its shadow, and down by the
// whole drop when the face has sunk into it.
func faceLabelShift(button *models.Button, face *ebiten.Image) float64 {
	if button.State == models.ButtonStateDisabled {
		return 0
	}
	drop := faceDrop(face, button.Height)
	if buttonSunken(button) {
		return drop / 2
	}
	return -drop / 2
}
