package systems

// Bevelling a flat rectangle, from the one colour it names.
//
// **The glyphs got a bevel and the widgets did not**, which is what left the two reading as
// different pictures: a pixel-art glyph with a lit rim sitting on a plain filled rectangle. The
// glyph palette is authored — six named values in glyphs.go, because a silhouette's light has to
// be drawn rather than computed off a fill. A widget's is not authored and must not be, or every
// button in the game would need a palette picked for it and CLAUDE.md's "name one colour" rule
// would be dead.
//
// **So the light and the shade are derived from the fill itself.** Highlight moves the fill toward
// white and shade scales it down, both by a fixed amount, so a button still names one colour and
// gets a lit top edge and a shadowed bottom one for free. That is the correction the rule needed:
// scaling is how a surface expresses *state*, and a bevel is the surface's own light, which is a
// different question and does not spend the colour.

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// How far a bevel's two edges are from the face between them.
//
// **The lit edge moves toward white and the shadowed one scales toward black**, which is the pair
// of operations already in this package rather than a third one: ColorToward for the light,
// because a saturated colour has nowhere to climb by scaling, and ColorAtStrength for the shade,
// because scaling is what holds a hue while it darkens.
//
// 34 and 58 are far enough apart to read as a lit edge and a shadowed one at three pixels, and
// close enough that a dark button's shadow does not go flat black against the ground behind it.
const (
	bevelLightPct = 34
	bevelShadePct = 58
)

// PaneBevelWidth is what a large surface takes, against BevelWidth for a control. See BevelRect
// for why the two differ.
const PaneBevelWidth = 2

// BevelWidth is how thick each edge is drawn on a control.
//
// **Three pixels, and it is the same argument the glyph rim makes.** One pixel reads as an
// outline rather than as light, and the widgets it goes on — a 44px chrome square, a 50px
// button — cannot spare more than three without the face becoming a frame.
const BevelWidth = 3

// BevelFace paints a bevelled rectangle covering the whole of dst: the fill, a lit edge along the
// top and left, and a shadowed one along the bottom and right.
//
// **`sunken` swaps the two**, which is the whole of what "pressed" and "latched" look like. A
// pressed button lit brighter is a button the cursor is merely resting on; a pressed button whose
// light has moved to the bottom edge has gone *in*, and the two states stop competing for the same
// signal. It is why latchedStrength was already the darkest step on the ramp — this is the same
// idea said in geometry rather than in brightness.
//
// **Corners are mitred**, top-left and bottom-right belonging wholly to their own edge, so the two
// diagonals meet rather than one edge running past the other and squaring the corner off.
func BevelFace(dst *ebiten.Image, w, h int, fill color.RGBA, sunken bool) {
	BevelRect(dst, 0, 0, w, h, BevelWidth, fill, sunken)
}

// BevelRect is BevelFace anywhere on a surface and at any depth: the fill, then `width` pixels of
// light along the top and left and shade along the bottom and right.
//
// **The depth is a parameter because a panel is not a button** *(2026-08-24)*. A button is a small
// thing you press and three pixels of light on it read as a control; the panels are the largest
// areas on the screen, and the same three pixels there read as chrome — a frame around the game
// rather than a surface it is played on. Two is what a pane takes.
//
// **Corners are mitred**, top-left and bottom-right belonging wholly to their own edge, so the two
// diagonals meet rather than one edge running past the other and squaring the corner off.
func BevelRect(dst *ebiten.Image, x, y, w, h, width int, fill color.RGBA, sunken bool) {
	light, shade := BevelEdges(fill)
	if sunken {
		light, shade = shade, light
	}

	fx, fy := float32(x), float32(y)
	vector.DrawFilledRect(dst, fx, fy, float32(w), float32(h), fill, false)
	if width <= 0 || w <= 2*width || h <= 2*width {
		return
	}

	for i := 0; i < width; i++ {
		f, n := float32(i), float32(1)
		// Top and left take the light: each row and column is inset by one at both ends, which
		// is what mitres the corner it shares with the edge below or beside it.
		vector.DrawFilledRect(dst, fx+f, fy+f, float32(w-2*i), n, light, false)
		vector.DrawFilledRect(dst, fx+f, fy+f, n, float32(h-2*i), light, false)

		// Bottom and right take the shade, inset the same way from the other corner.
		vector.DrawFilledRect(dst, fx+f, fy+float32(h-1-i), float32(w-2*i), n, shade, false)
		vector.DrawFilledRect(dst, fx+float32(w-1-i), fy+f, n, float32(h-2*i), shade, false)
	}
}

// BevelEdges is the lit and the shadowed version of one fill.
//
// **Exported because `internal/cards` bevels its border with the same two colours** and must do
// its own rasterising: that package renders without a graphics context, so it cannot call
// BevelFace. Sharing the derivation rather than the drawing is what stops a card's light and a
// button's light drifting apart while both claim to be lit from the top left.
func BevelEdges(fill color.RGBA) (light, shade color.RGBA) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: fill.A}
	return ColorToward(fill, white, bevelLightPct), ColorAtStrength(fill, bevelShadePct)
}
