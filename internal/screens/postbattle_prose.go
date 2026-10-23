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

	"github.com/curiousjc/ascend-duel/internal/cards"
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

	// orb draws the span as `dots` vitae dots in a row — a payment on its row, one dot a vitae.
	// **It appears whole** the moment typing reaches it, rather than a digit at a time.
	orb  bool
	dots int

	// gone is how many of the dots have already left for the purse, which the row no longer draws.
	gone int
}

// A payment's dots on its row: each one's side, and the air between two.
const (
	payoutDotSize = 24
	payoutDotGap  = 4
)

// dotsWidth is how wide n dots stand.
func dotsWidth(n int) float64 {
	if n <= 0 {
		return 0
	}
	return float64(n*payoutDotSize + (n-1)*payoutDotGap)
}

// dotCenterX is the middle of dot i of n, in a row centered on cx.
func dotCenterX(cx float64, i, n int) float64 {
	return cx - dotsWidth(n)/2 + float64(i*(payoutDotSize+payoutDotGap)) + payoutDotSize/2
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
	if r.orb {
		return dotsWidth(r.dots)
	}
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

// inFlight is the part of the payment still in the air: claimed, and not yet in the purse the card
// draws. **Each dot pays its vitae as it lands**, so the purse counts up one at a time.
func (t *typewriter) inFlight() int {
	if !t.flying {
		return 0
	}
	landed := 0
	for i := 0; i < t.flight.amount; i++ {
		if t.flight.dotProgress(i) >= 1 {
			landed++
		}
	}
	return t.flight.amount - landed
}

// withDeparted marks, on line i's amount, the dots that have left it for the purse, so the row stops
// drawing each one as it sets off. Only the line being paid has any in the air.
func (t *typewriter) withDeparted(i int, right []proseSpan) []proseSpan {
	if !t.flying || i != t.line || len(right) == 0 {
		return right
	}
	gone := 0
	for d := 0; d < t.flight.amount; d++ {
		if t.flight.dotProgress(d) > 0 {
			gone++
		}
	}
	out := append([]proseSpan(nil), right...)
	for k := range out {
		if out[k].orb {
			out[k].gone = gone
		}
	}
	return out
}

// dotStagger is the gap between one dot setting off and the next, so a row pours rather than jumps.
func dotStagger() int { return ui.Beat(1, 12) }

// vitaeFlightTotal is the whole of a payment of n dots: the last one's delay plus its flight.
func vitaeFlightTotal(n int) int { return vitaeFlightTicks() + max(0, n-1)*dotStagger() }

// dotProgress is how far dot i is along its flight, 0 to 1.
func (f vitaeFlight) dotProgress(i int) float64 {
	return ui.Clamp01(float64(f.trip.Age-i*dotStagger()) / float64(vitaeFlightTicks()))
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
			t.flight = vitaeFlight{amount: paid, from: at(t.line), trip: ui.NewTravel(0, vitaeFlightTotal(paid))}
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
			if r.orb {
				out = append(out, r)
				left -= len(runes)
				continue
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
		if r.orb {
			drawDots(face, screen, r.dots, x+w/2, float64(y+proseLineHeight/2), r.gone)
		} else if r.figure {
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

// drawDots draws n vitae dots in a row centered on (cx, cy), leaving out the first `gone` of them —
// the ones already on their way to the purse.
func drawDots(face *text.GoTextFace, screen *ebiten.Image, n int, cx, cy float64, gone int) {
	for i := max(0, gone); i < n; i++ {
		ui.DrawOrb(face.Source, screen, cards.OrbVitae, "", dotCenterX(cx, i, n), cy, payoutDotSize)
	}
}

// drawVitaeFlight draws the payment on its way to the purse: the row's dots leaving it one after
// another, each crossing to the duelist card's vitae orb and easing out so it lands rather than
// stops. The card's orb takes each dot's vitae and swells on the frame it arrives — see
// ui.OrbSwells.
func (t *typewriter) drawVitaeFlight(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	if !t.flying {
		return
	}

	card := buildCardRect(gs)
	to, ok := ui.StatOrbCenter(card, cards.DuelistStyle, vitaeStat)
	if !ok {
		to = image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	}

	// **The dots the row ends on, set off from where each sat**, so the payment leaving the row and
	// the payment crossing the screen are one thing moving. `from` is the middle of the row's dots.
	n := t.flight.amount
	fromY := float64(t.flight.from.Y + proseLineHeight/2)
	for i := 0; i < n; i++ {
		p := t.flight.dotProgress(i)
		if p >= 1 {
			continue
		}
		e := ui.EaseOut(p)
		fx := dotCenterX(float64(t.flight.from.X), i, n)
		ui.DrawOrb(face.Source, screen, cards.OrbVitae, "",
			fx+(float64(to.X)-fx)*e, fromY+(float64(to.Y)-fromY)*e, payoutDotSize)
	}
}

// vitaeStat is the purse's row in ui.DuelistSpec's Stats.
const vitaeStat = 2
