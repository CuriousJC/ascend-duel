package systems

// figure_line.go — a line of interface lettering, set in the figure glyphs wherever the set covers it.
//
// **The game's type is two sheets and every line is one or the other.** The prose sheet is for
// reading — see prose_line.go. The figure sheet is the interface's own lettering: a button's label,
// the hand's shout, the sum, a readout, a code to copy down. DrawUI is DrawText's twin for the
// second, taking exactly what text.Draw takes, so a line is moved between the two by renaming one
// call. A line the set does not cover is drawn in the font instead, whole.

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// uiFigureShare is how tall a line's digits stand against the point size its face names — the
// button labels' own share, so a readout and a label at one size are one size.
const uiFigureShare = buttonFigureShare

// figureUntinted is the luminance under which an ink draws the neutral sheet as it is: a dark ink
// multiplied into the sheet would bury its outline.
const figureUntinted = 0.25

// UIHeightOf is the height a line at font `size` takes in the figure glyphs.
func UIHeightOf(size float64) float64 { return size * uiFigureShare }

// MeasureUI is how wide s is as DrawUI would set it.
func MeasureUI(s string, face *text.GoTextFace) float64 {
	if FigureCovers(s) {
		return MeasureFigure(s, face.Size*uiFigureShare)
	}
	w, _ := text.Measure(s, face, 0)
	return w
}

// DrawUI draws s as text.Draw would — the options' translation is the anchor, its alignments place
// the line against it, its color scale is the ink — in the figure glyphs when the set covers s.
func DrawUI(dst *ebiten.Image, s string, face *text.GoTextFace, op *text.DrawOptions) {
	if !FigureCovers(s) {
		text.Draw(dst, s, face, op)
		return
	}
	h := face.Size * uiFigureShare
	w := MeasureFigure(s, h)
	ax, ay := op.GeoM.Apply(0, 0)

	cx := ax + w/2
	switch op.PrimaryAlign {
	case text.AlignCenter:
		cx = ax
	case text.AlignEnd:
		cx = ax - w/2
	}

	// **The figure's body sits where the font's capitals would have**, so a line moved onto the
	// sheet stays on the line it was laid out on.
	m := face.Metrics()
	baseline := ay + m.HAscent
	switch op.SecondaryAlign {
	case text.AlignCenter:
		baseline = ay - (m.HAscent+m.HDescent)/2 + m.HAscent
	case text.AlignEnd:
		baseline = ay - m.HDescent
	}
	cy := baseline - h/2

	cs := op.ColorScale
	ink := color.RGBA{R: unit(cs.R()), G: unit(cs.G()), B: unit(cs.B()), A: 255}
	if drawsSheetAsIs(ink, figureUntinted*255) {
		ink = color.RGBA{}
	}
	DrawFigure(dst, s, FigureNeutral, ink, cx, cy, h, 1, cs.A())
}
