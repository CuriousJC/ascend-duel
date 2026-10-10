package screens

// **The reward screen types what happened, every line at once, and the purse climbs as it reads.**
//
// The screen used to open with everything already true: three cards on a table and a purse that had
// silently changed while the fight was ending. What a win actually *is* — interest on what you were
// carrying, which third of your life you kept, and what the room itself pays — was arithmetic nobody
// ever saw happen.
//
// So the payout is narrated *(owner's call, 2026-08-22)*. The block types out at the game's own
// speed, and the figure each line names then flies to the duelist card and lands in the purse.
// **Nothing is added before its sentence has been read**, which is what makes the three parts
// distinguishable rather than one number that moved.
//
// **A click fills the block and a click leaves it** *(owner's call, 2026-09-08)*. The fill used to
// be the exit as well, so the one gesture available to a player who wanted the payout *now* was the
// gesture that took it off the screen. **The reward screen advances on nothing but a click**: it
// holds the total for as long as the player wants it. Typing every line at once was tried in the same sitting and put back - the
// lines arrive one at a time, at twice the old speed.
//
// **It may not change what a win is worth.** The amounts are frozen by `session.WonFight` before
// this screen exists — see session/spoils.go — and everything here is a clock. A player who clicks
// through the whole thing gets exactly the same purse as one who watches it, which is the same rule
// playback speed follows in a duel.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

// The narration's two clocks, both **proportions of the game's one speed** — see clock.go, and
// CLAUDE.md, which is why no screen may declare a raw tick count.
// proseCharTicks() is how long one character takes. **Fast on purpose, and doubled again on
// 2026-09-08** *(owner's call)*: this is a sentence appearing, not a teletype, and a player who
// has read it should be waiting on the next line rather than on the rest of this one.
func proseCharTicks() int { return ui.Beat(1, 24) }

// proseLinePause() is the beat held between one finished sentence and the next starting. It is
// what makes the payout read as three separate things, and it is **deliberately not scaled with
// the typing**: the pause is the separation, so shortening it alongside the characters would
// give back the run-together reading that typing every line at once produced.
func proseLinePause() int { return ui.Beat(1, 2) }

// vitaeFlightTicks() is how long a figure takes to reach the duelist card. The purse changes when
// it lands, never when it sets off — the flight *is* the payment arriving.
func vitaeFlightTicks() int { return ui.Beat(3, 4) }

// proseSpan is one colored stretch of a sentence. A line is a few of them, so "the enemy's 4 vitae
// flows to you" can put the figure and the word in crimson and leave the rest of the sentence alone
// — the same distinction `Spec.TextInk` makes on a card, and for the same reason: coloring the
// verb would say the money changed the sentence.
type proseSpan struct {
	text string
	ink  color.RGBA

	// figure draws the span on the figure sheet rather than the prose one — an amount carrying the
	// vitae mark `¤`, which only the figure sheet has. **Decided off the whole span when it is
	// made**, never off what has been typed of it, so "+5 ¤" does not start in one lettering and
	// finish in the other.
	figure bool
}

// proseLine is one line of the block, and pays is what claiming it hands over — nil for a line
// that only says something.
//
// **A line is either centered or a row.** A centered line is its spans across the middle of the
// column. A row is `spans` from the block's left edge and `right` ending on its right edge, so a
// column of rows reads as a rectangle: labels down the left, amounts down the right.
type proseLine struct {
	spans []proseSpan
	right []proseSpan

	centered bool

	// pays is called when the figure this line named has flown to the card. **It is the claim**,
	// so the purse moves at the moment the player watches it arrive.
	pays func(*state.GlobalState) int
}

// plain is the whole line as one string — what is typed, in the order it is typed.
func (l proseLine) plain() string {
	out := ""
	for _, r := range l.spans {
		out += r.text
	}
	for _, r := range l.right {
		out += r.text
	}
	return out
}

// spanWidth is how wide a span draws at the block's type size, on whichever sheet it is set on.
func spanWidth(r proseSpan, face *text.GoTextFace) float64 {
	if r.figure {
		return systems.MeasureFigure(r.text, systems.UIHeightOf(face.Size))
	}
	return systems.MeasureText(r.text, face)
}

func spansWidth(spans []proseSpan, face *text.GoTextFace) float64 {
	w := 0.0
	for _, r := range spans {
		w += spanWidth(r, face)
	}
	return w
}

// typewriter types a block of lines and flies each payment to the duelist card.
type typewriter struct {
	lines []proseLine

	// line is the sentence being typed, shown how many of its characters are up, and wait is the
	// pause held after it finished.
	line, shown, wait, ticks int

	// flight is the figure currently crossing to the duelist card, if any.
	flight vitaeFlight
	flying bool

	// released is whether the block has given the screen back. **Only a caller sets it**
	// *(owner's call, 2026-09-08)* - see release.
	released bool
}

// vitaeFlight is one payment on its way to the purse.
type vitaeFlight struct {
	amount int
	from   image.Point
	trip   ui.Travel
}

// setLines starts the block over.
func (t *typewriter) setLines(lines []proseLine) {
	*t = typewriter{lines: lines}
}

// filled reports whether every sentence is up and every payment has landed.
//
// **It is not the same question as finished**, and the split is the whole of the 2026-09-08 change:
// a block can be complete and still be holding the screen. The reward screen holds it there until a
// second click, because the figures are what the player came to read.
func (t *typewriter) filled() bool {
	return t.line >= len(t.lines) && !t.flying
}

// release hands the screen on. **Nothing in here calls it** - a block does not decide when it is
// done being looked at, and the beat it used to hold before moving on by itself was what put the
// essences up over a total still being read.
func (t *typewriter) release() { t.released = true }

// finished reports whether the narration is done with the screen.
func (t *typewriter) finished() bool { return t.filled() && t.released }

// tick advances the typing, the pause between lines, and whatever is in the air.
//
// **A payment blocks the next sentence.** The line that named it stays alone on screen until the
// figure has landed, so the number on the card and the number in the sentence are never two
// unrelated things moving at once.
func (t *typewriter) tick(gs *state.GlobalState, at func(line int) image.Point) {
	if t.flying {
		t.flight.trip.Tick()
		if t.flight.trip.Done() {
			t.flying = false
		}
		return
	}
	if t.line >= len(t.lines) {
		return
	}

	line := t.lines[t.line]
	full := len([]rune(line.plain()))

	if t.shown < full {
		t.ticks++
		if t.ticks >= proseCharTicks() {
			t.ticks = 0
			t.shown++
		}
		return
	}

	// The sentence is complete: pay what it named, then hold a beat before the next one.
	if line.pays != nil {
		if paid := line.pays(gs); paid > 0 {
			t.flight = vitaeFlight{amount: paid, from: at(t.line), trip: ui.NewTravel(0, vitaeFlightTicks())}
			t.flying = true
		}
		t.lines[t.line].pays = nil
		return
	}

	t.wait++
	if t.wait >= proseLinePause() {
		t.line, t.shown, t.wait = t.line+1, 0, 0
	}
}

// skip finishes the whole block at once: every sentence up, every payment made, nothing in the air.
//
// **It pays through the same claims**, rather than adding the total itself, so the fast path and
// the slow one cannot disagree about what a win was worth.
//
// **It does not release the screen** *(owner's call, 2026-09-08)*. A click that both finished the
// reading and started the essences would have skipped the sentence it was asking to be shown - the
// player asked to see the payout now, not to be past it. The next click leaves.
func (t *typewriter) skip(gs *state.GlobalState) {
	for i := range t.lines {
		if t.lines[i].pays != nil {
			t.lines[i].pays(gs)
			t.lines[i].pays = nil
		}
	}
	t.line, t.shown, t.wait, t.flying = len(t.lines), 0, 0, false
}

// visible is one line's left spans and right spans as far as they have been typed, and whether the
// line is on screen at all. The left part types first, then the right.
func (t *typewriter) visible(i int) ([]proseSpan, []proseSpan, bool) {
	if i > t.line {
		return nil, nil, false
	}
	line := t.lines[i]
	if i < t.line {
		return line.spans, line.right, true
	}

	left := t.shown
	cut := func(spans []proseSpan) []proseSpan {
		out := make([]proseSpan, 0, len(spans))
		for _, r := range spans {
			runes := []rune(r.text)
			if left <= 0 {
				break
			}
			if left < len(runes) {
				r.text = string(runes[:left])
				out = append(out, r)
				left = 0
				break
			}
			out = append(out, r)
			left -= len(runes)
		}
		return out
	}
	spans := cut(line.spans)
	return spans, cut(line.right), true
}

// drawProseLine writes one line, as far as it has been typed, on the baseline row y.
//
// **Placed by measuring the finished line, not the typed part** *(2026-08-22)*, so a sentence does
// not slide sideways as it types: a centered line is centered on its whole width, and a row's right
// part starts where its whole right part has to start to end on the block's edge.
func drawProseLine(screen *ebiten.Image, face *text.GoTextFace, line proseLine, left, right []proseSpan,
	block payoutBlock, y int) {

	x := float64(block.left)
	if line.centered {
		x = float64(block.mid()) - spansWidth(line.spans, face)/2
	}
	drawProseSpans(screen, face, left, x, y)

	if len(line.right) > 0 {
		drawProseSpans(screen, face, right, float64(block.right)-spansWidth(line.right, face), y)
	}
}

// drawProseSpans writes spans left to right from x, each on its own sheet.
func drawProseSpans(screen *ebiten.Image, face *text.GoTextFace, spans []proseSpan, x float64, y int) {
	for _, r := range spans {
		ink := r.ink
		if ink.A == 0 {
			ink = ui.GroundInk
		}
		w := spanWidth(r, face)
		if r.figure {
			drawPayoutFigure(screen, face, r.text, ink, x+w/2, y, 1)
		} else {
			op := &text.DrawOptions{}
			op.GeoM.Translate(x, float64(y))
			op.ColorScale.ScaleWithColor(ink)
			systems.DrawText(screen, r.text, face, op)
		}
		x += w
	}
}

// drawPayoutFigure draws a figure centered on cx with its body where the prose's capitals sit on
// the row y — the same placement systems.DrawUI gives a figure set in a face — and on the vitae
// sheet when the ink is the vitae crimson.
func drawPayoutFigure(screen *ebiten.Image, face *text.GoTextFace, s string, ink color.RGBA,
	cx float64, y int, alpha float32) {

	h := systems.UIHeightOf(face.Size)
	cy := float64(y) + face.Metrics().HAscent - h/2
	sheet, tint := figureSheetFor(ink)
	systems.DrawFigure(screen, s, sheet, tint, cx, cy, h, 1, alpha)
}

// drawVitaeFlight draws the figure on its way to the purse: the row's own amount, on the vitae
// sheet, crossing to the duelist card and easing out so it lands rather than stops.
func (t *typewriter) drawVitaeFlight(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	if !t.flying {
		return
	}

	card := buildCardRect(gs)
	to := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	p := ui.EaseOut(t.flight.trip.Progress())

	x := float64(t.flight.from.X) + (float64(to.X-t.flight.from.X))*p
	y := float64(t.flight.from.Y) + (float64(to.Y-t.flight.from.Y))*p

	// **The same figure the row ends on, set off from where it sat**, so the amount leaving the row
	// and the amount crossing the screen are one thing moving. `from` is the row's figure center on
	// its baseline row, and the figure is drawn on that row's terms as it travels.
	drawPayoutFigure(screen, face, payoutGain(t.flight.amount).text, ui.VitaeInk, x, int(y), 1)
}
