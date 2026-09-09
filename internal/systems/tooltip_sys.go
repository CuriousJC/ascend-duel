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

// The tooltip: how the panel behaves, and how it is drawn.
//
// **What it says is never decided here.** A scene hands it a title and a list of lines; this file
// knows about a dwell, a rectangle and a surface. That is the same division `DrawButton` keeps —
// the button holds its own label and this package holds what a press looks like — and it is what
// lets the wording live in `internal/screens` beside the rest of the game's prose.

// The panel's own measurements. Pixels rather than percentages: a tooltip is sized by its text, and
// text is not a percentage of anything.
const (
	tipPad       = 12
	tipTitleSize = 20
	tipLineSize  = 17
	tipLineGap   = 6

	// tipGap is how far the panel sits from the thing it explains, and tipEdge how close it may come
	// to the edge of the screen.
	tipGap  = 14
	tipEdge = 8

	tipCorner = 6
)

// The panel is dark on a light table, which is the one contrast the game has left: every card, pane
// and log is an off-white surface, so an overlay that is another off-white would read as one more
// card rather than as something on top of everything. Same face a button paints.
var (
	tipSurface = color.RGBA{R: 38, G: 35, B: 30, A: 244}
	tipTitle   = color.RGBA{R: 245, G: 242, B: 236, A: 255}
	tipInk     = color.RGBA{R: 208, G: 202, B: 190, A: 255}
	tipEdgeInk = color.RGBA{R: 92, G: 84, B: 72, A: 255}
)

// UpdateTooltip advances the dwell and forgets whatever the scene stopped pointing at.
//
// **Call it once, after everything that might Point.** A scene aims the tooltip while it works out
// what is under the cursor; this is what turns "nothing was aimed at this tick" into a hidden panel,
// so no scene has to remember to hide one. Same handshake `state.ModalOpen` uses.
func UpdateTooltip(gs *state.GlobalState, t *models.Tooltip) {
	if !t.Pointed() {
		t.Forget()
		return
	}
	t.Release()

	if t.Dwell < t.DwellTicks {
		t.Dwell++
	}
}

// DrawTooltip puts the panel on screen, beside whatever it is about. It draws nothing until the
// dwell is served.
//
// **Beside the anchor rather than under the cursor**, so the panel never covers the card it is
// explaining and does not slide about as the hand moves inside one card. It goes to the right, and
// flips to the left when there is no room; the same on the vertical, clamped so a panel is never
// half off the screen.
func DrawTooltip(gs *state.GlobalState, screen *ebiten.Image, t *models.Tooltip) {
	if !t.Showing() {
		return
	}

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tipLineSize}
	titleFace := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tipTitleSize}
	if face.Source == nil {
		return // no font: a tooltip is the one thing that must not be a coloured box with no words
	}

	w, h := tipSize(t, face, titleFace)
	at := tipPlace(gs, t.Anchor, w, h)

	vector.DrawFilledRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), tipSurface, false)
	vector.StrokeRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), 1, tipEdgeInk, false)

	y := at.Y + tipPad
	if len(t.Title) > 0 {
		// **The title's runs are drawn by the same function the body's are**, so a coloured word in
		// a title cannot end up placed differently from the same word one line down.
		drawTipRuns(screen, t.Title, titleFace, at.X+tipPad, y, tipTitle)
		y += int(tipTitleSize) + tipLineGap
	}
	for _, line := range t.Lines {
		drawTipRuns(screen, line, face, at.X+tipPad, y, tipInk)
		y += int(tipLineSize) + tipLineGap
	}
}

// drawTipRuns draws one line as its runs, each in its own ink, left to right.
//
// **Measured and placed rather than drawn twice**, exactly as `cards.drawMarkedLine` is on the card
// face — overdrawing a coloured run on top of the whole line composites two sets of antialiased
// edges and reads as a smudge. The two rasterisers are unrelated and the rule is the same.
//
// **A run with no ink takes `plain`**, so a caller that never thinks about colour is drawn exactly
// as it was before runs existed. It is a parameter rather than a constant because the title and the
// body have different default inks and share this drawing.
func drawTipRuns(screen *ebiten.Image, line models.TipLine, face *text.GoTextFace, x, y int, plain color.RGBA) {
	for _, run := range line {
		if run.Text == "" {
			continue
		}
		ink := run.Ink
		if ink.A == 0 {
			ink = plain
		}
		drawTipLine(screen, run.Text, face, x, y, ink)
		w, _ := text.Measure(run.Text, face, 0)
		x += int(w)
	}
}

func drawTipLine(screen *ebiten.Image, s string, face *text.GoTextFace, x, y int, ink color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, s, face, op)
}

// tipSize measures the panel against its longest line. **Measured rather than estimated**: the
// alternative is a character count times a guessed width, which is wrong the first time a line
// carries a wide glyph and shows up as text running out of a box.
func tipSize(t *models.Tooltip, face, titleFace *text.GoTextFace) (w, h int) {
	widest := 0.0
	if len(t.Title) > 0 {
		// Run by run and summed, for the reason the body's lines are: the title is drawn that way
		// too now, and measuring the joined string would let kerning across a join make the panel a
		// pixel narrower than what goes in it.
		for _, run := range t.Title {
			w, _ := text.Measure(run.Text, titleFace, 0)
			widest += w
		}
		h += int(tipTitleSize) + tipLineGap
	}
	for _, line := range t.Lines {
		// **Measured run by run and summed**, because that is how the line is drawn: measuring the
		// joined string would let kerning across a join make the panel a pixel narrower than what
		// goes in it.
		lw := 0.0
		for _, run := range line {
			w, _ := text.Measure(run.Text, face, 0)
			lw += w
		}
		if lw > widest {
			widest = lw
		}
		h += int(tipLineSize) + tipLineGap
	}
	if h > 0 {
		h -= tipLineGap // the gap after the last line is not part of the panel
	}
	return int(widest) + tipPad*2, h + tipPad*2
}

// tipPlace is where the panel goes: to the right of the anchor, flipping and clamping rather than
// running off the screen.
func tipPlace(gs *state.GlobalState, anchor image.Rectangle, w, h int) image.Point {
	x := anchor.Max.X + tipGap
	if x+w > gs.ScreenWidth-tipEdge {
		x = anchor.Min.X - tipGap - w
	}
	if x < tipEdge {
		x = tipEdge
	}

	// Top-aligned with the thing it explains, which reads as belonging to it; pushed up only when
	// the panel would otherwise fall off the bottom.
	y := anchor.Min.Y
	if y+h > gs.ScreenHeight-tipEdge {
		y = gs.ScreenHeight - tipEdge - h
	}
	if y < tipEdge {
		y = tipEdge
	}
	return image.Pt(x, y)
}
