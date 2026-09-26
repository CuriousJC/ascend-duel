package screens

// combat_figures.go — which figure sheet a number on the table is drawn from, and how big.
//
// **Every string the hand dialog and the flying figures write goes through `drawMathText`**, and a
// string made only of glyphs the figure set carries is drawn from it rather than set in kubasta.
// The set carries the capitals, so an upper-case word — a hand's name, MISS, BLOCKED — is a figure
// as well; anything with a lower-case letter stays in the font. `mathWidth` is the one measure both
// the layout and the drawing agree on.
//
// **The ink a caller asks for picks the sheet**, so no call site learned about sheets:
//
//   - an element's border color draws that element's sheet, the attack ink the attack sheet, the
//     relic pink the relic sheet and the vitae crimson the vitae sheet — each in its own ramp;
//   - **a dark ink draws the neutral sheet as it is**, white under its black contour. The ground ink
//     every duelist figure and operator is written in cannot be multiplied onto a sheet
//     whose outline is already black: it would come out a black shape with a black edge;
//   - **any other ink multiplies the neutral sheet**, which is pure gray so the ink comes through as
//     itself — drain green, a rider's tint.

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// figureDigitShare is how tall a figure's digits stand against the point size the same string
// would be set at in kubasta. It is the one dial between the two, so the sizes the box is written
// in — `mathTermSize`, `mathTotalSize` — keep meaning what they meant.
const figureDigitShare = 0.36

// figureDarkInk is the luminance under which an ink draws the neutral sheet untinted.
const figureDarkInk = 0.35

// figureHeight is the digit height, in pixels, of a figure standing in for type at `size` points.
func figureHeight(size float64) float64 { return size * figureDigitShare }

// mathWidth is how wide str will be drawn at size: as a figure when the set covers it, in the font
// otherwise. `layOutLine` measures with it and `drawMathText` draws the same decision.
func mathWidth(gs *state.GlobalState, str string, size float64) float64 {
	if systems.FigureCovers(str) {
		return systems.MeasureFigure(str, figureHeight(size))
	}
	w, _ := text.Measure(str, mathFace(gs, size), 0)
	return w
}

// figureSheetFor is the sheet an ink is drawn from, and the ink to multiply it by — zero for none.
func figureSheetFor(tint color.RGBA) (string, color.RGBA) {
	for _, e := range cards.Elements() {
		if e == cards.Basic {
			continue
		}
		if tint == cards.BorderOf(e) && systems.FigureHasSheet(e.String()) {
			return e.String(), color.RGBA{}
		}
	}
	switch tint {
	case ui.VerbInkFor(combat.CategoryAttack):
		return "attack", color.RGBA{}
	case ui.BoostInk, ui.PaneEdge:
		return "relic", color.RGBA{}
	case ui.VitaeInk:
		if systems.FigureHasSheet("vitae") {
			return "vitae", color.RGBA{}
		}
	}
	if tint == cards.BorderOf(cards.Basic) || luminance(tint) < figureDarkInk {
		return systems.FigureNeutral, color.RGBA{}
	}
	return systems.FigureNeutral, tint
}

// luminance is a color's relative brightness, 0..1.
func luminance(c color.RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}
