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
	tipPad       = 24
	tipTitleSize = 39
	tipLineSize  = 32
	tipLineGap   = 12

	// tipGap is how far the panel sits from the thing it explains, and tipEdge how close it may come
	// to the edge of the screen.
	tipGap  = 14
	tipEdge = 8
)

// **The dark panel, and it is the game's only one.** Every card, pane and log is an off-white
// surface, so an overlay that is another off-white would read as one more card rather than as
// something on top of everything. Dark is the one contrast left.
//
// **Exported because the tutorial bubble is the same object** *(owner's call, 2026-09-18)*. Bob's
// panel was a second hand-picked near-black with its own near-white ink, one degree cooler than
// this one and different for no reason anybody chose — which is exactly the thing the color rule
// asks to be derived rather than written down twice. Two dark panels of prose over a light table
// are one surface, so there is one set of names for it.
//
// **The alpha belongs to the caller, not to the surface.** A tooltip is faintly transparent
// because it covers the thing it is explaining; a bubble is opaque because it does not. That is
// the one value a second panel is expected to set for itself.
var (
	PanelSurface = color.RGBA{R: 38, G: 35, B: 30, A: 244}

	// PanelSpeech is what the panel is saying — a tooltip's title, Bob's prose. PanelInk is the
	// quieter register underneath it: a stat line, a term of an arithmetic.
	PanelSpeech = color.RGBA{R: 245, G: 242, B: 236, A: 255}
	PanelInk    = color.RGBA{R: 208, G: 202, B: 190, A: 255}

	PanelEdgeInk = color.RGBA{R: 92, G: 84, B: 72, A: 255}
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
		return // no font: a tooltip is the one thing that must not be a colored box with no words
	}

	// **Wrapped once and then both measured and drawn from the same lines.** The panel used to
	// measure what the caller had broken; now that it breaks them itself, a second wrap for the
	// drawing would be a second answer to the same question and the two would differ on the day a
	// rounding changed.
	title, body, w, h := tipLayout(t, face, titleFace)
	at := tipPlace(gs, t.Anchor, w, h)

	vector.FillRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), PanelSurface, false)
	vector.StrokeRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), 1, PanelEdgeInk, false)

	y := at.Y + tipPad
	for _, line := range title {
		// **The title's runs are drawn by the same function the body's are**, so a colored word in
		// a title cannot end up placed differently from the same word one line down.
		DrawRuns(screen, line, titleFace, at.X+tipPad, y, PanelSpeech)
		y += int(tipTitleSize) + tipLineGap
	}
	for _, line := range body {
		DrawRuns(screen, line, face, at.X+tipPad, y, PanelInk)
		y += int(tipLineSize) + tipLineGap
	}
}

// DrawRuns draws one line as its runs, each in its own ink, left to right.
//
// **Exported alongside WrapRuns**, because the tutorial bubble sets colored prose through the same
// pair. A second drawing of a colored line is a second place a run's advance can be measured
// differently from where it is placed.
//
// **Measured and placed rather than drawn twice**, exactly as `cards.drawMarkedLine` is on the card
// face — overdrawing a colored run on top of the whole line composites two sets of antialiased
// edges and reads as a smudge. The two rasterizers are unrelated and the rule is the same.
//
// **A run with no ink takes `plain`**, so a caller that never thinks about color is drawn exactly
// as it was before runs existed. It is a parameter rather than a constant because the title and the
// body have different default inks and share this drawing.
//
// **A run naming a material is set in it instead**, the grain showing through the letters — see
// tooltip_texture.go. It is placed and measured exactly as a colored run is, so a form word sits
// where the same word in one color would.
func DrawRuns(screen *ebiten.Image, line models.TipLine, face *text.GoTextFace, x, y int, plain color.RGBA) {
	for _, run := range line {
		if run.Text == "" {
			continue
		}
		ink := run.Ink
		if ink.A == 0 {
			ink = plain
		}
		// **The material is tried first and the ink is what a missing one falls back to.** A run
		// naming a texture nobody has filed draws as a colored word rather than as a gap.
		if run.Texture == "" || !drawTexturedRun(screen, run.Text, run.Texture, face, x, y) {
			drawTipLine(screen, run.Text, face, x, y, ink)
		}
		w, _ := text.Measure(run.Text, face, 0)
		x += int(w)
	}
}

// PanelWeight is how far a second pass is offset to thicken the panel's type, in pixels at the
// internal resolution. Zero is the font as drawn.
//
// **Synthesized rather than a second font file, because Kubasta has one weight.** It is a static
// face with no variation axes — RobotoFlex is the only variable font embedded, and swapping the
// panel's typeface to get a bolder one would make the tooltip the one place in the game set in a
// different face. A second pass a fraction of a pixel to the side is the standard way to thicken
// a single-weight face and it keeps the letterforms.
//
// **Horizontal only.** Offsetting vertically as well fills the counters of a and e at this size
// and the type stops being legible before it stops looking bold.
//
// **One dial for the whole dark-panel family**, so Bob's bubble is set at the same weight as a
// tooltip — the two are one surface and type that disagreed across them would say they are not.
var PanelWeight = 0.9

// drawTipLine draws one run, twice, a hair apart.
//
// **This is the deliberate exception to the rule two lines up.** Overdrawing a *colored* run on
// top of a whole line composites two different sets of glyph edges and reads as a smudge; this is
// the same string in the same ink, so what the second pass composites is the same shape a fraction
// over — which is weight rather than blur.
func drawTipLine(screen *ebiten.Image, s string, face *text.GoTextFace, x, y int, ink color.RGBA) {
	offsets := []float64{0}
	if PanelWeight > 0 {
		offsets = append(offsets, PanelWeight)
	}
	for _, dx := range offsets {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x)+dx, float64(y))
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, s, face, op)
	}
}

// tipLayout wraps the panel's text and measures what came back.
//
// **One function, because the two cannot be allowed to disagree.** A panel measured against one
// set of lines and drawn from another is a box with text hanging out of it, and it only shows up
// on the strings long enough to wrap.
//
// **Measured rather than estimated**: the alternative is a character count times a guessed width,
// which is wrong the first time a line carries a wide glyph.
func tipLayout(t *models.Tooltip, face, titleFace *text.GoTextFace) (title, body []models.TipLine, w, h int) {
	widest := 0.0
	measure := func(line models.TipLine, f *text.GoTextFace) {
		// **Measured run by run and summed**, because that is how the line is drawn: measuring the
		// joined string would let kerning across a join make the panel a pixel narrower than what
		// goes in it.
		lw := 0.0
		for _, run := range line {
			lw += measureText(run.Text, f)
		}
		if lw > widest {
			widest = lw
		}
	}

	if len(t.Title) > 0 {
		title = WrapRuns(t.Title, titleFace, tipMaxW)
		for _, line := range title {
			measure(line, titleFace)
			h += int(tipTitleSize) + tipLineGap
		}
	}
	for _, line := range t.Lines {
		for _, wrapped := range WrapRuns(line, face, tipMaxW) {
			body = append(body, wrapped)
			measure(wrapped, face)
			h += int(tipLineSize) + tipLineGap
		}
	}
	if h > 0 {
		h -= tipLineGap // the gap after the last line is not part of the panel
	}
	return title, body, int(widest) + tipPad*2, h + tipPad*2
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
