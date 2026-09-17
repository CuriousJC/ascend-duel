package ui

// **The pane widget: a titled box with a list of rows in it.**
//
// One screen draws one today — the fight log, which pours a whole fight's sentences into a
// panel over a scrim. It is a widget rather than that dialog's own drawing because a pane is a
// shape the game reuses: a row with a swatch, a colored verb and an underline, laid out down
// a rectangle, is what any list of things-that-happened looks like.
//
// **It knows nothing about combat.** A row arrives as three strings and two colors; whoever
// built it decided what the words are. That is what makes it structurally impossible for a
// panel to disagree with the round it reports — see prose.go, which does the deciding.
//
// Split out of combat_panes.go on 2026-08-21, which held the widget and the prose together.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The band a full-height pane occupies. **Resolution left it on 2026-08-11** and moved down
// to the strip above the hand — see the feed constants below — so today this describes only
// Action Flow, which is not drawn. The space between 12% and 46% is deliberately empty and
// spoken for.
const (
	paneTitleInset = 10 // gap from the pane's top edge to its title
	PaneFirstRow   = 45 // gap from the top edge to the first action row
	paneRowHeight  = 30
	paneRowInset   = 10 // gap from the pane's left edge to a row's swatch
	swatchSize     = 16
	swatchGap      = 6 // gap between a swatch and its label

	// paneTextSize is what a pane writes at unless it says otherwise. See panePlacement.textSize.
	paneTextSize = 16

	// paneBandInset keeps a row's ground clear of the pane's own border, and paneBandRise lifts it
	// so the band sits around the line rather than starting at its cap height.
	paneBandInset = 3
	paneBandRise  = 4
)

// PanePlacement is one pane's label, identifying color and the way it draws its rows.
//
// **It carries no horizontal slot any more.** It held a left and a right percentage while
// several panes shared the band and each took a column of it; the rectangle is a parameter to
// drawPane now, because the one pane left — the ledger's — is sized against the panel it sits
// in rather than against a slice of the screen.
type PanePlacement struct {
	Title string
	Color color.RGBA

	// **A pane carries its own surface and its own ink**, rather than deriving both from one
	// color. Resolution went off-white on 2026-08-07 because colored verb chips on a dim
	// plum ground were hard to read — three saturated colors competing with a fourth behind
	// them. A light ground makes the chips the only saturated thing in the pane.
	//
	// This is the same exception glyphs are documented under in `CLAUDE.md`: the one-color
	// rule governs how a widget responds to hover, press and disable, and it cannot describe
	// a surface and the thing sitting on it at once. `color` still drives the border and is
	// what the pane is "named", so the scale-don't-add rule keeps working for state.
	Fill   color.RGBA // the pane's ground
	Ink    color.RGBA // text drawn on that ground
	NowInk color.RGBA // text of the row playback is on: colored, bold and underlined

	// RowHeight is the pitch this pane draws its rows at. Carried on the placement rather
	// than being one constant because the two panes hold different things: card names, and
	// sentences about what those cards did.
	RowHeight int

	// TextSize is the point size the rows are written at. **Zero means paneTextSize**, so a pane
	// that has no opinion gets the size every pane had before this existed.
	//
	// It travels with rowHeight rather than being derived from it: the two have to move together —
	// a bigger face at the old pitch overlaps its neighbors — and which pitch a size wants is a
	// judgment about air between lines, not arithmetic.
	TextSize float64

	// Bold draws every run Bold, not just the marked one. **The ledger takes it and nothing else
	// does** *(2026-09-02, owner asked to see it)*: the panel is read at a distance rather than
	// glanced at during a round, and kubasta at 16 on an off-white ground is light. It is a
	// property of the pane rather than of a row so that a row still says what a *verb* is — the
	// mark keeps its underline, which is what tells the two apart once everything is heavy.
	Bold bool

	// FirstRow is the gap from the top edge to the first row. A titled pane has to clear its
	// title; the feed has no title and cannot afford to pretend it does — 45 pixels of
	// reserved heading out of an 82-pixel box is most of the box.
	FirstRow int

	// RightInset is a column down the pane's right edge that the rows may not use.
	//
	// **It narrows the content and never the frame** *(owner's call, 2026-09-12)*. The ledger runs
	// a scrollbar down that edge, and a heading's band is drawn edge to edge inside the border —
	// so the band ran under the bar and a centered heading was centered on a width half of which was
	// behind it. The border still reaches the panel's own edge, because the pane is the thing the
	// scrollbar sits *on*.
	RightInset int
}

// PaneEdge is the pink a pane is bordered and named in. Still a placeholder palette.
var PaneEdge = color.RGBA{R: 235, G: 105, B: 170, A: 255}

// paneSpan is one run of text inside a row, with the color it is written in.
//
// **A row is spans rather than three fixed slots** *(2026-09-02)*. It was prefix / verb / suffix,
// which was exactly enough for a sentence with one colored verb in it and not enough for the
// ledger's arithmetic — a figure in its card's color, a relic's multiplier in the relic pink, the
// hand's own multiplier in the hand's. Storing spans is what lets the panel look like the screen it
// is an account of.
type paneSpan struct {
	Text string

	// Ink is the color this run is written in. **Zero alpha means "the row's own Ink"**, the same
	// convention Button.BaseColor uses.
	Ink color.RGBA

	// Mark draws it bold and underlined: the verb, and nothing else.
	Mark bool
}

// PaneRow is one line in a pane: some spans of text, optionally preceded by a color swatch saying
// whose action it is. A zero-alpha swatch means the row has none, in which case a single unmarked
// run is centered instead of sitting in a column beside the squares.
type PaneRow struct {
	Spans []paneSpan

	Swatch color.RGBA

	// Highlighted marks the row as the one happening right now, drawn lit against the
	// dim pane behind it.
	Highlighted bool

	// Band is a color painted across the whole row before anything is drawn on it, and a
	// zero-alpha Band is no Band at all.
	//
	// **It is what groups rows into blocks.** The ledger folds a run into one line per fight and
	// opens one at a time; without a ground behind them, an opened fight's forty rows and the next
	// fight's heading are one undifferentiated list. A Band says "all of this is the same thing".
	Band color.RGBA

	// Indent sets a row in from the pane's left edge, in pixels, and **is what stops a row with
	// no swatch being centered**. The arithmetic under a blow is a column of figures: centering it
	// would put every line at a different left edge, which is the one layout a column cannot
	// survive. See prose_terms.go.
	Indent int
}

// PlainRow is a whole row in the pane's own ink: a heading, a placeholder, a sentence nobody has
// colored. Most rows outside a duel are one of these.
func PlainRow(text string) PaneRow {
	return PaneRow{Spans: []paneSpan{{Text: text}}}
}

// Text is the row as one string, for a caller reading it rather than drawing it — a test, or the
// scripted demo's report.
func (r PaneRow) Text() string {
	var out string
	for _, span := range r.Spans {
		out += span.Text
	}
	return out
}

// centered reports whether the row is written down the middle of the pane rather than in the
// column: a lone unmarked span, no swatch and no indent. Headings are the case this exists for.
func (r PaneRow) centered() bool {
	return r.Swatch.A == 0 && r.Indent == 0 && len(r.Spans) == 1 && !r.Spans[0].Mark
}

// drawPaneFrame draws a pane's fill, border and title in the rectangle given, and reports it
// back as floats. Split out because the card panes fill themselves rather than drawing text
// rows.
func drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p PanePlacement, r image.Rectangle) (x, y, w, h float32) {
	// Not drawBox: a pane names its own ground and its own ink, where drawBox derives a dim
	// fill from one color. drawBox still serves the caption and the character strip, which
	// have no text on a light ground to worry about.
	x, y = float32(r.Min.X), float32(r.Min.Y)
	w, h = float32(r.Dx()), float32(r.Dy())

	// **Raised, because a pane floats over a scrim.** It is a panel in front of the game rather
	// than a tray cut into it, so the light stays on the top-left edge where a button's is —
	// see systems.BevelRect for why a pane takes two pixels of it and a control takes three.
	systems.BevelRect(screen, r.Min.X, r.Min.Y, r.Dx(), r.Dy(),
		systems.PaneBevelWidth, p.Fill, false)
	vector.StrokeRect(screen, x, y, w, h, 2, p.Color, false)

	if p.Title != "" {
		titleOp := &text.DrawOptions{}
		titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
		titleOp.PrimaryAlign = text.AlignCenter
		titleOp.ColorScale.ScaleWithColor(p.Ink)
		text.Draw(screen, p.Title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)
	}

	return x, y, w, h
}

// DrawPane draws a read-only pane: the frame, then a row per action.
//
// **Free functions rather than methods on the combat scene** *(2026-09-02)*. They never touched a
// field of it, and the ledger — which is chrome, not a scene — draws the same panel: a widget only
// one type could call would have meant a second pane renderer for the same picture.
func DrawPane(gs *state.GlobalState, screen *ebiten.Image, p PanePlacement, r image.Rectangle, rows []PaneRow) {
	x, y, w, _ := drawPaneFrame(gs, screen, p, r)

	size := p.TextSize
	if size == 0 {
		size = paneTextSize
	}
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}

	// **The highlight is centered on the text, not offset from the row's top by a constant.**
	// It used to be drawn at rowY-4 with height rowHeight-2, numbers picked by eye against a
	// single 30px pitch. When the Resolution pane arrived at 22 the bar came out 20 tall
	// against a ~19px line sitting 4px lower, so it clipped the text and the swatch along its
	// bottom edge. Measuring the line and centering on it works at any pitch, which is the
	// point — the pane's pitch is now a property of the placement and free to change again.
	_, lineHeight := text.Measure("Ag", face, 0)

	// The width the rows actually have, which is the pane less whatever it keeps down its right
	// edge. See panePlacement.rightInset.
	contentW := w - float32(p.RightInset)

	for i, row := range rows {
		rowY := y + float32(p.FirstRow) + float32(i*p.RowHeight)

		// The row's own ground, edge to edge inside the pane's border, before anything is written
		// on it. Drawn a whole pitch tall so consecutive banded rows read as one block rather than
		// as stripes.
		if row.Band.A != 0 {
			vector.FillRect(screen, x+paneBandInset, rowY-paneBandRise,
				contentW-2*paneBandInset, float32(p.RowHeight), row.Band, false)
		}

		// **The row playback is on is set in the text itself — colored, bold and underlined —
		// rather than sat on a lit bar** *(changed 2026-08-07)*. A full-width bar was a fourth
		// saturated block in a pane that already carries a swatch, a verb and a sentence, and on a
		// light ground it had to be pale enough to read through, which left it shouting and saying
		// little. Marking the words is the same signal spent on the thing being read.
		//
		// Bold is faux — the same run drawn again a pixel right. `text/v2` has no synthetic weight
		// and kubasta ships one, so this is the only way to get one without a second font file. At
		// a pixel font's sizes it is exactly what a bold face would do anyway.
		ink := p.Ink
		if row.Highlighted {
			ink = p.NowInk
		}

		// A lone unmarked run with nothing beside it is a heading, and headings are centered.
		if row.centered() {
			// **A centered run keeps its own ink.** It did not for one build, and the row that
			// needs it most is the one that has a band behind it: a heading on a dark ground
			// written in the panel's near-black ink is a heading nobody can read.
			tint := ink
			if !row.Highlighted && row.Spans[0].Ink.A != 0 {
				tint = row.Spans[0].Ink
			}

			rowOp := &text.DrawOptions{}
			rowOp.GeoM.Translate(float64(x+contentW/2), float64(rowY))
			rowOp.PrimaryAlign = text.AlignCenter
			rowOp.ColorScale.ScaleWithColor(tint)
			text.Draw(screen, row.Spans[0].Text, face, rowOp)
			continue
		}

		textX := x + paneRowInset + float32(row.Indent)
		if row.Swatch.A != 0 {
			// A swatch turns the row into a column: square on the left, the line beside it, so the
			// squares line up down the pane and the alternation is readable as a pattern rather
			// than as text.
			//
			// **Idle swatches fade toward the pane's own ground**, so the lit one is the strongest
			// thing in the pane whether that ground is dark or light. Scaling toward black — which
			// is what dimming used to mean here — made idle rows *more* contrasty than the lit one
			// the moment the pane went off-white. See systems.ColorToward.
			swatch := row.Swatch
			if !row.Highlighted {
				swatch = systems.ColorToward(swatch, p.Fill, 45)
			}
			// Centered on the line for the same reason everything else is, so the squares sit level
			// with the text they belong to whatever pitch the pane draws at.
			swatchTop := rowY + float32(lineHeight)/2 - swatchSize/2
			vector.FillRect(screen, x+paneRowInset, swatchTop, swatchSize, swatchSize, swatch, false)
			textX = x + paneRowInset + swatchSize + swatchGap
		}

		// The spans, measured one after the next. **A span with no ink of its own takes the row's**,
		// so a plain sentence is written in one color and a sum is written in five.
		cursorX := float64(textX)
		for _, span := range row.Spans {
			if span.Text == "" {
				continue
			}
			tint := span.Ink
			if tint.A == 0 || row.Highlighted {
				tint = ink
			}
			bold := span.Mark || row.Highlighted || p.Bold

			at := func(dx float64) {
				op := &text.DrawOptions{}
				op.GeoM.Translate(cursorX+dx, float64(rowY))
				op.ColorScale.ScaleWithColor(tint)
				text.Draw(screen, span.Text, face, op)
			}
			at(0)
			if bold {
				at(1) // faux bold
			}

			wSpan, _ := text.Measure(span.Text, face, 0)

			// **The mark is always bold *and* underlined.** That is what makes a verb read as the
			// verb rather than as a word that happens to be colored — one mark would be ambiguous
			// against a pane that also uses color for the side and for the live row.
			//
			// **Flush with the bottom of the measured line box**, not a constant above it:
			// text.Measure reports the full line including descent, which is what keeps the rule
			// clear of a descender rather than striking through one.
			if span.Mark {
				vector.FillRect(screen,
					float32(cursorX), rowY+float32(lineHeight)-underlineHeight,
					float32(wSpan), underlineHeight, tint, false)
			}

			// Advance by the *unbolded* width, so the second pass thickens the strokes without
			// walking the spans after it out of place.
			cursorX += wSpan
		}
	}
}

const (
	// underlineHeight is how thick the verb's underline is. Two pixels rather than one: at
	// kubasta's weight a single pixel reads as an artefact of the font rather than a mark.
	underlineHeight = 2
)
