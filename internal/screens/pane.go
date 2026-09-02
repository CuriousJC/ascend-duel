package screens

// **The pane widget: a titled box with a list of rows in it.**
//
// One screen draws one today — the fight log, which pours a whole fight's sentences into a
// panel over a scrim. It is a widget rather than that dialog's own drawing because a pane is a
// shape the game reuses: a row with a swatch, a coloured verb and an underline, laid out down
// a rectangle, is what any list of things-that-happened looks like.
//
// **It knows nothing about combat.** A row arrives as three strings and two colours; whoever
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
	paneTopPct    = 12
	paneBottomPct = 46

	paneTitleInset = 10 // gap from the pane's top edge to its title
	paneFirstRow   = 45 // gap from the top edge to the first action row
	paneRowHeight  = 30
	paneRowInset   = 10 // gap from the pane's left edge to a row's swatch
	swatchSize     = 16
	swatchGap      = 6 // gap between a swatch and its label

	// The Resolution rows are sentences rather than card names, and there are more of them —
	// a busy round merges to a dozen lines where the flow pane draws at most ten.
	paneTextRowHeight = 22

	// paneTextSize is what a pane writes at unless it says otherwise. See panePlacement.textSize.
	paneTextSize = 16

	// paneBandInset keeps a row's ground clear of the pane's own border, and paneBandRise lifts it
	// so the band sits around the line rather than starting at its cap height.
	paneBandInset = 3
	paneBandRise  = 4
)

// panePlacement is one pane's horizontal slot, label and identifying colour. The
// colours are loud on purpose — these are placeholders for finding the layout, not a
// palette anyone has chosen yet.
type panePlacement struct {
	leftPct, rightPct int
	title             string
	color             color.RGBA

	// **A pane carries its own surface and its own ink**, rather than deriving both from one
	// colour. Resolution went off-white on 2026-08-07 because coloured verb chips on a dim
	// plum ground were hard to read — three saturated colours competing with a fourth behind
	// them. A light ground makes the chips the only saturated thing in the pane.
	//
	// This is the same exception glyphs are documented under in `CLAUDE.md`: the one-colour
	// rule governs how a widget responds to hover, press and disable, and it cannot describe
	// a surface and the thing sitting on it at once. `color` still drives the border and is
	// what the pane is "named", so the scale-don't-add rule keeps working for state.
	fill   color.RGBA // the pane's ground
	ink    color.RGBA // text drawn on that ground
	nowInk color.RGBA // text of the row playback is on: coloured, bold and underlined

	// rowHeight is the pitch this pane draws its rows at. Carried on the placement rather
	// than being one constant because the two panes hold different things: card names, and
	// sentences about what those cards did.
	rowHeight int

	// textSize is the point size the rows are written at. **Zero means paneTextSize**, so a pane
	// that has no opinion gets the size every pane had before this existed.
	//
	// It travels with rowHeight rather than being derived from it: the two have to move together —
	// a bigger face at the old pitch overlaps its neighbours — and which pitch a size wants is a
	// judgement about air between lines, not arithmetic.
	textSize float64

	// bold draws every run bold, not just the marked one. **The ledger takes it and nothing else
	// does** *(2026-09-02, owner asked to see it)*: the panel is read at a distance rather than
	// glanced at during a round, and kubasta at 16 on an off-white ground is light. It is a
	// property of the pane rather than of a row so that a row still says what a *verb* is — the
	// mark keeps its underline, which is what tells the two apart once everything is heavy.
	bold bool

	// firstRow is the gap from the top edge to the first row. A titled pane has to clear its
	// title; the feed has no title and cannot afford to pretend it does — 45 pixels of
	// reserved heading out of an 82-pixel box is most of the box.
	firstRow int
}

// paneEdge is the pink a pane is bordered and named in. Still a placeholder palette.
var paneEdge = color.RGBA{R: 235, G: 105, B: 170, A: 255}

// paneRun is one run of text inside a row, with the colour it is written in.
//
// **A row is runs rather than three fixed slots** *(2026-09-02)*. It was prefix / verb / suffix,
// which was exactly enough for a sentence with one coloured verb in it and not enough for the
// ledger's arithmetic — a figure in its card's colour, a ring's multiplier in the ring pink, the
// hand's own multiplier in the hand's. Storing runs is what lets the panel look like the screen it
// is an account of.
type paneRun struct {
	text string

	// ink is the colour this run is written in. **Zero alpha means "the row's own ink"**, the same
	// convention Button.BaseColor uses.
	ink color.RGBA

	// mark draws it bold and underlined: the verb, and nothing else.
	mark bool
}

// paneRow is one line in a pane: some runs of text, optionally preceded by a colour swatch saying
// whose action it is. A zero-alpha swatch means the row has none, in which case a single unmarked
// run is centred instead of sitting in a column beside the squares.
type paneRow struct {
	runs []paneRun

	swatch color.RGBA

	// highlighted marks the row as the one happening right now, drawn lit against the
	// dim pane behind it.
	highlighted bool

	// band is a colour painted across the whole row before anything is drawn on it, and a
	// zero-alpha band is no band at all.
	//
	// **It is what groups rows into blocks.** The ledger folds a run into one line per fight and
	// opens one at a time; without a ground behind them, an opened fight's forty rows and the next
	// fight's heading are one undifferentiated list. A band says "all of this is the same thing".
	band color.RGBA

	// indent sets a row in from the pane's left edge, in pixels, and **is what stops a row with
	// no swatch being centred**. The arithmetic under a blow is a column of figures: centring it
	// would put every line at a different left edge, which is the one layout a column cannot
	// survive. See prose_terms.go.
	indent int
}

// plainRow is a whole row in the pane's own ink: a heading, a placeholder, a sentence nobody has
// coloured. Most rows outside a duel are one of these.
func plainRow(text string) paneRow {
	return paneRow{runs: []paneRun{{text: text}}}
}

// text is the row as one string, for a caller reading it rather than drawing it — a test, or the
// scripted demo's report.
func (r paneRow) text() string {
	var out string
	for _, run := range r.runs {
		out += run.text
	}
	return out
}

// centred reports whether the row is written down the middle of the pane rather than in the
// column: a lone unmarked run, no swatch and no indent. Headings are the case this exists for.
func (r paneRow) centred() bool {
	return r.swatch.A == 0 && r.indent == 0 && len(r.runs) == 1 && !r.runs[0].mark
}

// panePlacementRect is the column a full-height pane occupies, from its percentages and the
// shared band. **Only Action Flow is placed this way now** — Resolution takes its rectangle
// from the hand instead, so the rect is a parameter rather than something drawPane works out.
func panePlacementRect(gs *state.GlobalState, p panePlacement) image.Rectangle {
	return image.Rect(
		gs.PctX(p.leftPct), gs.PctY(paneTopPct),
		gs.PctX(p.rightPct), gs.PctY(paneBottomPct),
	)
}

// drawPaneFrame draws a pane's fill, border and title in the rectangle given, and reports it
// back as floats. Split out because the card panes fill themselves rather than drawing text
// rows.
func drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle) (x, y, w, h float32) {
	// Not drawBox: a pane names its own ground and its own ink, where drawBox derives a dim
	// fill from one colour. drawBox still serves the caption and the character strip, which
	// have no text on a light ground to worry about.
	x, y = float32(r.Min.X), float32(r.Min.Y)
	w, h = float32(r.Dx()), float32(r.Dy())

	// **Raised, because a pane floats over a scrim.** It is a panel in front of the game rather
	// than a tray cut into it, so the light stays on the top-left edge where a button's is —
	// see systems.BevelRect for why a pane takes two pixels of it and a control takes three.
	systems.BevelRect(screen, r.Min.X, r.Min.Y, r.Dx(), r.Dy(),
		systems.PaneBevelWidth, p.fill, false)
	vector.StrokeRect(screen, x, y, w, h, 2, p.color, false)

	if p.title != "" {
		titleOp := &text.DrawOptions{}
		titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
		titleOp.PrimaryAlign = text.AlignCenter
		titleOp.ColorScale.ScaleWithColor(p.ink)
		text.Draw(screen, p.title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)
	}

	return x, y, w, h
}

// drawPane draws a read-only pane: the frame, then a row per action.
//
// **Free functions rather than methods on the combat scene** *(2026-09-02)*. They never touched a
// field of it, and the ledger — which is chrome, not a scene — draws the same panel: a widget only
// one type could call would have meant a second pane renderer for the same picture.
func drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle, rows []paneRow) {
	x, y, w, _ := drawPaneFrame(gs, screen, p, r)

	size := p.textSize
	if size == 0 {
		size = paneTextSize
	}
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}

	// **The highlight is centred on the text, not offset from the row's top by a constant.**
	// It used to be drawn at rowY-4 with height rowHeight-2, numbers picked by eye against a
	// single 30px pitch. When the Resolution pane arrived at 22 the bar came out 20 tall
	// against a ~19px line sitting 4px lower, so it clipped the text and the swatch along its
	// bottom edge. Measuring the line and centring on it works at any pitch, which is the
	// point — the pane's pitch is now a property of the placement and free to change again.
	_, lineHeight := text.Measure("Ag", face, 0)

	for i, row := range rows {
		rowY := y + float32(p.firstRow) + float32(i*p.rowHeight)

		// The row's own ground, edge to edge inside the pane's border, before anything is written
		// on it. Drawn a whole pitch tall so consecutive banded rows read as one block rather than
		// as stripes.
		if row.band.A != 0 {
			vector.DrawFilledRect(screen, x+paneBandInset, rowY-paneBandRise,
				w-2*paneBandInset, float32(p.rowHeight), row.band, false)
		}

		// **The row playback is on is set in the text itself — coloured, bold and underlined —
		// rather than sat on a lit bar** *(changed 2026-08-07)*. A full-width bar was a fourth
		// saturated block in a pane that already carries a swatch, a verb and a sentence, and on a
		// light ground it had to be pale enough to read through, which left it shouting and saying
		// little. Marking the words is the same signal spent on the thing being read.
		//
		// Bold is faux — the same run drawn again a pixel right. `text/v2` has no synthetic weight
		// and kubasta ships one, so this is the only way to get one without a second font file. At
		// a pixel font's sizes it is exactly what a bold face would do anyway.
		ink := p.ink
		if row.highlighted {
			ink = p.nowInk
		}

		// A lone unmarked run with nothing beside it is a heading, and headings are centred.
		if row.centred() {
			// **A centred run keeps its own ink.** It did not for one build, and the row that
			// needs it most is the one that has a band behind it: a heading on a dark ground
			// written in the panel's near-black ink is a heading nobody can read.
			tint := ink
			if !row.highlighted && row.runs[0].ink.A != 0 {
				tint = row.runs[0].ink
			}

			rowOp := &text.DrawOptions{}
			rowOp.GeoM.Translate(float64(x+w/2), float64(rowY))
			rowOp.PrimaryAlign = text.AlignCenter
			rowOp.ColorScale.ScaleWithColor(tint)
			text.Draw(screen, row.runs[0].text, face, rowOp)
			continue
		}

		textX := x + paneRowInset + float32(row.indent)
		if row.swatch.A != 0 {
			// A swatch turns the row into a column: square on the left, the line beside it, so the
			// squares line up down the pane and the alternation is readable as a pattern rather
			// than as text.
			//
			// **Idle swatches fade toward the pane's own ground**, so the lit one is the strongest
			// thing in the pane whether that ground is dark or light. Scaling toward black — which
			// is what dimming used to mean here — made idle rows *more* contrasty than the lit one
			// the moment the pane went off-white. See systems.ColorToward.
			swatch := row.swatch
			if !row.highlighted {
				swatch = systems.ColorToward(swatch, p.fill, 45)
			}
			// Centred on the line for the same reason everything else is, so the squares sit level
			// with the text they belong to whatever pitch the pane draws at.
			swatchTop := rowY + float32(lineHeight)/2 - swatchSize/2
			vector.DrawFilledRect(screen, x+paneRowInset, swatchTop, swatchSize, swatchSize, swatch, false)
			textX = x + paneRowInset + swatchSize + swatchGap
		}

		// The runs, measured one after the next. **A run with no ink of its own takes the row's**,
		// so a plain sentence is written in one colour and a sum is written in five.
		cursorX := float64(textX)
		for _, run := range row.runs {
			if run.text == "" {
				continue
			}
			tint := run.ink
			if tint.A == 0 || row.highlighted {
				tint = ink
			}
			bold := run.mark || row.highlighted || p.bold

			at := func(dx float64) {
				op := &text.DrawOptions{}
				op.GeoM.Translate(cursorX+dx, float64(rowY))
				op.ColorScale.ScaleWithColor(tint)
				text.Draw(screen, run.text, face, op)
			}
			at(0)
			if bold {
				at(1) // faux bold
			}

			wRun, _ := text.Measure(run.text, face, 0)

			// **The mark is always bold *and* underlined.** That is what makes a verb read as the
			// verb rather than as a word that happens to be coloured — one mark would be ambiguous
			// against a pane that also uses colour for the side and for the live row.
			//
			// **Flush with the bottom of the measured line box**, not a constant above it:
			// text.Measure reports the full line including descent, which is what keeps the rule
			// clear of a descender rather than striking through one.
			if run.mark {
				vector.DrawFilledRect(screen,
					float32(cursorX), rowY+float32(lineHeight)-underlineHeight,
					float32(wRun), underlineHeight, tint, false)
			}

			// Advance by the *unbolded* width, so the second pass thickens the strokes without
			// walking the runs after it out of place.
			cursorX += wRun
		}
	}
}

const (
	// underlineHeight is how thick the verb's underline is. Two pixels rather than one: at
	// kubasta's weight a single pixel reads as an artefact of the font rather than a mark.
	underlineHeight = 2
)
