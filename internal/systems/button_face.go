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
// back. **Jade is everything else**: the chrome, the safe answer, Leave and Skip, and any button
// that names no color. A latched button wears amber whatever it names — see latchedButtonFace.
//
// ButtonBlue has no face: it is the stroke round a dialog, a line rather than a button.
var (
	ButtonRed  = color.RGBA{R: 208, G: 52, B: 58, A: 255}
	ButtonJade = color.RGBA{R: 61, G: 122, B: 98, A: 255}
	ButtonBlue = color.RGBA{R: 70, G: 130, B: 230, A: 255}
)

// buttonFaces is the face each color wears, by asset key. The default color is in it, so a button
// naming no color still wears a face.
var buttonFaces = map[color.RGBA]string{
	ButtonRed:          "button-carnelian",
	ButtonJade:         "button-jade",
	defaultButtonColor: "button-jade",
}

const (
	disabledButtonFace = "button-disabled"

	// latchedButtonFace is what a latched button wears: the mode that is on, in amber.
	latchedButtonFace = "button-amber"
)

// The face's geometry at the size it is authored, in its own pixels: the two caps hold all the
// rounding, and the middle between them tiles seamlessly left to right.
const buttonFaceCap = 48

// How bright a face draws in each state, as a multiple of the picture. **The picture is the button
// at rest**, so rest is 1 and hover climbs above it. **Every face is polished stone and stands on
// no drop**, so a press darkens one rather than sinking it; a latched button is wearing amber,
// which is its whole latched look. faceLatchedScale is for an icon with no latched drawing of its
// own — see drawButtonIcon.
const (
	faceHoverScale   = 1.18
	facePressedScale = 0.82
	faceLatchedScale = 0.7
)

// buttonFace is the picture a button wears, or nil for a button drawn with the bevel: a color with
// no face, or a face that failed to load.
func buttonFace(gs *state.GlobalState, button *models.Button) *ebiten.Image {
	switch button.State {
	case models.ButtonStateDisabled:
		return gs.Assets[disabledButtonFace]
	case models.ButtonStateLatched:
		if face := gs.Assets[latchedButtonFace]; face != nil {
			return face
		}
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

// drawButtonFace paints a face into the button's own image: the caps scaled to its height, the
// middle tiled across its width, and the whole dimmed or brightened for its state.
//
// **The middle repeats a whole number of times**, each copy stretched a little to fill the span
// exactly. A copy cut short against the right cap put the start of the stone beside the end of it,
// which is a seam; whole copies meet the cap with the join the art was drawn to make.
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

	var lum float32 = 1
	switch button.State {
	case models.ButtonStateHovered:
		lum = faceHoverScale
	case models.ButtonStatePressed:
		lum = facePressedScale
	}

	piece := func(sx0, sx1 int, dx, dw float64) {
		if dw <= 0 || sx1 <= sx0 {
			return
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(dw/float64(sx1-sx0), scale)
		op.GeoM.Translate(dx, 0)
		op.ColorScale.Scale(lum, lum, lum, 1)
		op.Filter = ebiten.FilterLinear
		button.Image.DrawImage(face.SubImage(image.Rect(sx0, 0, sx1, sh)).(*ebiten.Image), op)
	}
	piece(0, buttonFaceCap, 0, capW)
	tile := float64(sw-2*buttonFaceCap) * scale
	copies := math.Max(1, math.Round(mid/tile))
	for i := 0.0; i < copies; i++ {
		x0, x1 := math.Round(mid*i/copies), math.Round(mid*(i+1)/copies)
		piece(buttonFaceCap, sw-buttonFaceCap, capW+x0, x1-x0)
	}
	piece(sw-buttonFaceCap, sw, capW+mid, capW)
}

// How an icon button is drawn in each state. It has no drop to sink into, so "in" is the picture
// drawn a little smaller about its center; disabled is the picture dimmed, since there is no
// separate disabled file for each icon.
const (
	iconPressedScale = 0.92
	iconDisabledLum  = 0.45

	// iconPressedSuffix names an icon's latched drawing: `icon-sort-cost-pressed` beside
	// `icon-sort-cost`. **The art is its own latched look**, so it is drawn as it is.
	iconPressedSuffix = "-pressed"
)

// drawButtonIcon paints an icon button's picture into its own image: scaled whole to the button's
// size, never stretched in parts, and brightened, shrunk or dimmed for its state with the same
// figures a face uses.
//
// pressed is the icon's own latched drawing, nil for an icon that has none. **A latched button with
// one wears it whole**; without one it is the resting picture shrunk and darkened.
func drawButtonIcon(button *models.Button, icon, pressed *ebiten.Image) {
	w, h := float64(button.Width), float64(button.Height)

	size := 1.0
	var lum float32 = 1
	switch button.State {
	case models.ButtonStateHovered:
		lum = faceHoverScale
	case models.ButtonStatePressed:
		size = iconPressedScale
	case models.ButtonStateLatched:
		if pressed != nil {
			icon = pressed
		} else {
			size, lum = iconPressedScale, faceLatchedScale
		}
	case models.ButtonStateDisabled:
		lum = iconDisabledLum
	}
	sw, sh := float64(icon.Bounds().Dx()), float64(icon.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w*size/sw, h*size/sh)
	op.GeoM.Translate(w*(1-size)/2, h*(1-size)/2)
	op.ColorScale.Scale(lum, lum, lum, 1)
	op.Filter = ebiten.FilterLinear
	button.Image.DrawImage(icon, op)
}

// faceLabelShift is where a label sits on a face, against the button's center: one pixel down while
// pressed, as the bevel moves its own, and centered otherwise.
func faceLabelShift(button *models.Button) float64 {
	if button.State == models.ButtonStatePressed {
		return 1
	}
	return 0
}
