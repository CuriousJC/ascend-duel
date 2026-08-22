package screens

// **The reward screen types what happened, one sentence at a time, and the purse climbs as it
// reads.**
//
// The screen used to open with everything already true: three cards on a table and a purse that had
// silently changed while the fight was ending. What a win actually *is* — interest on what you were
// carrying, a tenth of the life you kept, and what the room itself pays — was arithmetic nobody
// ever saw happen.
//
// So the payout is narrated *(owner's call, 2026-08-22)*. Each sentence types out at the game's own
// speed, and the figure it names then flies to the duelist card and lands in the purse. **Nothing
// is added before its sentence has been read**, which is what makes the three parts distinguishable
// rather than one number that moved.
//
// **It may not change what a win is worth.** The amounts are frozen by `session.WonFight` before
// this screen exists — see session/spoils.go — and everything here is a clock. A player who clicks
// through the whole thing gets exactly the same purse as one who watches it, which is the same rule
// playback speed follows in a duel.

import (
	"image"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

// The narration's two clocks, both **proportions of the game's one speed** — see clock.go, and
// CLAUDE.md, which is why no screen may declare a raw tick count.
var (
	// proseCharTicks is how long one character takes. **Fast on purpose**: this is a sentence
	// appearing, not a teletype, and a player who has read it should be waiting on the next line
	// rather than on the rest of this one.
	proseCharTicks = beat(1, 12)

	// proseLinePause is the beat held between one finished sentence and the next starting. It is
	// what makes the payout read as three separate things.
	proseLinePause = beat(1, 2)

	// vitaeFlightTicks is how long a figure takes to reach the duelist card. The purse changes when
	// it lands, never when it sets off — the flight *is* the payment arriving.
	vitaeFlightTicks = beat(3, 4)
)

// proseRun is one coloured stretch of a sentence. A line is a few of them, so "the enemy's 4 vitae
// flows to you" can put the figure and the word in crimson and leave the rest of the sentence alone
// — the same distinction `Spec.TextInk` makes on a card, and for the same reason: colouring the
// verb would say the money changed the sentence.
type proseRun struct {
	text string
	ink  color.RGBA
}

// proseLine is one sentence, and pays is what claiming it hands over — nil for a line that only
// says something.
type proseLine struct {
	runs []proseRun

	// pays is called when the figure this line named has flown to the card. **It is the claim**,
	// so the purse moves at the moment the player watches it arrive.
	pays func(*state.GlobalState) int
}

func (l proseLine) plain() string {
	out := ""
	for _, r := range l.runs {
		out += r.text
	}
	return out
}

// typewriter types a block of lines and flies each payment to the duelist card.
type typewriter struct {
	lines []proseLine

	// line is the sentence being typed, shown how many of its characters are up, and wait is the
	// pause held after it finished.
	line, shown, wait, ticks int

	// flight is the figure currently crossing to the duelist card, if any.
	flight  vitaeFlight
	flying  bool
	fromMid image.Point
}

// vitaeFlight is one payment on its way to the purse.
type vitaeFlight struct {
	amount int
	from   image.Point
	trip   travel
}

// setLines starts the block over.
func (t *typewriter) setLines(lines []proseLine) {
	*t = typewriter{lines: lines}
}

// finished reports whether every sentence is up and every payment has landed.
func (t *typewriter) finished() bool {
	return t.line >= len(t.lines) && !t.flying
}

// tick advances the typing, the pause between lines, and whatever is in the air.
//
// **A payment blocks the next sentence.** The line that named it stays alone on screen until the
// figure has landed, so the number on the card and the number in the sentence are never two
// unrelated things moving at once.
func (t *typewriter) tick(gs *state.GlobalState, at func(line int) image.Point) {
	if t.flying {
		t.flight.trip.tick()
		if t.flight.trip.done() {
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
		if t.ticks >= proseCharTicks {
			t.ticks = 0
			t.shown++
		}
		return
	}

	// The sentence is complete: pay what it named, then hold a beat before the next one.
	if line.pays != nil {
		if paid := line.pays(gs); paid > 0 {
			t.flight = vitaeFlight{amount: paid, from: at(t.line), trip: newTravel(0, vitaeFlightTicks)}
			t.flying = true
		}
		t.lines[t.line].pays = nil
		return
	}

	t.wait++
	if t.wait >= proseLinePause {
		t.line, t.shown, t.wait = t.line+1, 0, 0
	}
}

// skip finishes the whole block at once: every sentence up, every payment made, nothing in the air.
//
// **It pays through the same claims**, rather than adding the total itself, so the fast path and
// the slow one cannot disagree about what a win was worth.
func (t *typewriter) skip(gs *state.GlobalState) {
	for i := range t.lines {
		if t.lines[i].pays != nil {
			t.lines[i].pays(gs)
			t.lines[i].pays = nil
		}
	}
	t.line, t.shown, t.wait, t.flying = len(t.lines), 0, 0, false
}

// visible is the runs of one line as far as they have been typed, and whether the line is on screen
// at all.
func (t *typewriter) visible(i int) ([]proseRun, bool) {
	if i > t.line {
		return nil, false
	}
	line := t.lines[i]
	if i < t.line {
		return line.runs, true
	}

	left := t.shown
	out := make([]proseRun, 0, len(line.runs))
	for _, r := range line.runs {
		runes := []rune(r.text)
		if left <= 0 {
			break
		}
		if left < len(runes) {
			out = append(out, proseRun{text: string(runes[:left]), ink: r.ink})
			break
		}
		out = append(out, r)
		left -= len(runes)
	}
	return out, true
}

// drawProseLine writes one line's runs centred on x, and reports the rectangle the whole line
// occupies — which is what a payment flies out of.
//
// **Centred by measuring the finished line, not the typed part** *(2026-08-22)*, so a sentence
// does not slide sideways as it types. A line that grew from its own centre would be a line the eye
// has to keep re-finding.
func drawProseLine(screen *ebiten.Image, face *text.GoTextFace, full string, runs []proseRun,
	centerX, y int) {

	width, _ := text.Measure(full, face, 0)
	x := float64(centerX) - width/2

	for _, r := range runs {
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, float64(y))
		ink := r.ink
		if ink.A == 0 {
			ink = groundInk
		}
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, r.text, face, op)

		w, _ := text.Measure(r.text, face, 0)
		x += w
	}
}

// drawVitaeFlight draws the figure on its way to the purse: a crimson number crossing to the
// duelist card, easing out so it lands rather than stops.
func (t *typewriter) drawVitaeFlight(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	if !t.flying {
		return
	}

	card := buildCardRect(gs)
	to := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	p := easeOut(t.flight.trip.progress())

	x := float64(t.flight.from.X) + (float64(to.X-t.flight.from.X))*p
	y := float64(t.flight.from.Y) + (float64(to.Y-t.flight.from.Y))*p

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(vitaeInk)
	text.Draw(screen, "+"+strconv.Itoa(t.flight.amount), face, op)
}
