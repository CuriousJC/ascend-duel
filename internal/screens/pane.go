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

	// firstRow is the gap from the top edge to the first row. A titled pane has to clear its
	// title; the feed has no title and cannot afford to pretend it does — 45 pixels of
	// reserved heading out of an 82-pixel box is most of the box.
	firstRow int
}

// paneEdge is the pink a pane is bordered and named in. Still a placeholder palette.
var paneEdge = color.RGBA{R: 235, G: 105, B: 170, A: 255}

// paneRow is one line in a pane: a label, optionally preceded by a colour swatch
// saying whose action it is. A zero-alpha swatch means the row has none, in which case
// the label is centred instead of sitting in a column beside the squares.
type paneRow struct {
	// A row is drawn as three runs, so the verb in the middle can be coloured, bolded and
	// underlined while the words either side of it are not. Rows that are not a sentence —
	// a card name in Action Flow, a placeholder — put everything in prefix and leave the
	// other two empty, which is why prefix rather than verb is the one that always has to
	// be set.
	prefix, verb, suffix string

	// verbInk is the colour the verb itself is written in. **Zero alpha means "the row's own
	// ink"**, the same convention Button.BaseColor uses, and it is what the neutral category
	// takes — see verbInkFor. Storing a colour rather than a category keeps drawPane from
	// having to know anything about combat.
	verbInk color.RGBA

	swatch color.RGBA

	// highlighted marks the row as the one happening right now, drawn lit against the
	// dim pane behind it.
	highlighted bool
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
func (s *CombatScene) drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle) (x, y, w, h float32) {
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
func (s *CombatScene) drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, r image.Rectangle, rows []paneRow) {
	x, y, w, _ := s.drawPaneFrame(gs, screen, p, r)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	// **The highlight is centred on the text, not offset from the row's top by a constant.**
	// It used to be drawn at rowY-4 with height rowHeight-2, numbers picked by eye against a
	// single 30px pitch. When the Resolution pane arrived at 22 the bar came out 20 tall
	// against a ~19px line sitting 4px lower, so it clipped the text and the swatch along its
	// bottom edge. Measuring the line and centring on it works at any pitch, which is the
	// point — the pane's pitch is now a property of the placement and free to change again.
	_, lineHeight := text.Measure("Ag", face, 0)

	for i, row := range rows {
		rowY := y + float32(p.firstRow) + float32(i*p.rowHeight)
		rowOp := &text.DrawOptions{}

		// **The row playback is on is set in the text itself — coloured, bold and underlined —
		// rather than sat on a lit bar** *(changed 2026-08-07)*. A full-width bar was a fourth
		// saturated block in a pane that already carries a swatch, a verb chip and a sentence,
		// and on a light ground it had to be pale enough to read through, which left it
		// shouting and saying little. Marking the words is the same signal spent on the thing
		// the reader is actually looking at.
		//
		// Bold is faux — the same run drawn again a pixel right. `text/v2` has no synthetic
		// weight and kubasta ships one, so this is the only way to get one without a second
		// font file. At a pixel font's sizes it is exactly what a bold face would do anyway.
		ink := p.ink
		if row.highlighted {
			ink = p.nowInk
		}

		// A row with no verb is a single centred or left-aligned run and keeps the old path.
		// One with a verb has to be laid out left to right so the chip can be measured into
		// place, which rules out centring it — a sentence in a list wants a common left edge
		// anyway.
		if row.swatch.A == 0 && row.verb == "" {
			rowOp.GeoM.Translate(float64(x+w/2), float64(rowY))
			rowOp.PrimaryAlign = text.AlignCenter
			rowOp.ColorScale.ScaleWithColor(ink)
			text.Draw(screen, row.prefix, face, rowOp)
			continue
		}

		textX := x + paneRowInset
		if row.swatch.A != 0 {
			// A swatch turns the row into a column: square on the left, the line beside it,
			// so the squares line up down the pane and the alternation is readable as a
			// pattern rather than as text.
			//
			// **Idle swatches fade toward the pane's own ground**, so the lit one is the
			// strongest thing in the pane whether that ground is dark or light. Scaling
			// toward black — which is what dimming used to mean here — made idle rows *more*
			// contrasty than the lit one the moment Resolution went off-white. See
			// systems.ColorToward.
			swatch := row.swatch
			if !row.highlighted {
				swatch = systems.ColorToward(swatch, p.fill, 45)
			}
			// Centred on the line for the same reason the bar is, so the squares sit level
			// with the text they belong to whatever pitch the pane draws at.
			swatchTop := rowY + float32(lineHeight)/2 - swatchSize/2
			vector.DrawFilledRect(screen, x+paneRowInset, swatchTop, swatchSize, swatchSize, swatch, false)
			textX = x + paneRowInset + swatchSize + swatchGap
		}

		// Three runs, measured one after the next. The verb is written in its category's own
		// colour — red for attack, blue for defend — so a round can
		// be scanned for what *kind* of thing happened before any of it is read.
		cursorX := float64(textX)
		draw := func(str string, tint color.RGBA, bold bool) {
			if str == "" {
				return
			}
			at := func(dx float64) {
				op := &text.DrawOptions{}
				op.GeoM.Translate(cursorX+dx, float64(rowY))
				op.ColorScale.ScaleWithColor(tint)
				text.Draw(screen, str, face, op)
			}
			at(0)
			if bold {
				at(1) // faux bold
			}

			// Advance by the *unbolded* width, so the second pass thickens the strokes without
			// walking the runs after it out of place.
			wRun, _ := text.Measure(str, face, 0)
			cursorX += wRun
		}

		draw(row.prefix, ink, row.highlighted)
		if row.verb != "" {
			// **The verb is always bold and always underlined, on every row.** That is what makes
			// it read as the verb rather than as a word that happens to be coloured — one mark
			// would be ambiguous against a pane that also uses colour for the side and for the
			// live row, and three together are unmistakable at a glance.
			verbInk := row.verbInk
			if verbInk.A == 0 {
				verbInk = ink
			}

			verbLeft := float32(cursorX)
			wVerb, _ := text.Measure(row.verb, face, 0)
			draw(row.verb, verbInk, true)

			// **Flush with the bottom of the measured line box**, not a constant above it. The
			// underline used to sit under a chip whose height was fixed at 18 against a 22px
			// pitch; with no chip the only thing it can be positioned against is the text, and
			// text.Measure already reports the full line including descent. That is what keeps
			// it clear of a descender — a rule three pixels up from the baseline strikes
			// straight through one — and what lets either pane's pitch change again.
			vector.DrawFilledRect(screen,
				verbLeft, rowY+float32(lineHeight)-underlineHeight,
				float32(wVerb), underlineHeight,
				verbInk, false)
		}
		draw(row.suffix, ink, row.highlighted)
	}
}

const (
	// underlineHeight is how thick the verb's underline is. Two pixels rather than one: at
	// kubasta's weight a single pixel reads as an artefact of the font rather than a mark.
	underlineHeight = 2
)
