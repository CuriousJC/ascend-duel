package screens

// **What the reward screen says, and where the essences come in from.**
//
// The sentences are here rather than in prose.go because prose.go is the vocabulary a *log* line
// draws on — a verb for an attack, a phrase for a card — and this is a fixed script that happens
// once a fight. What the two have in common is the rule: **the words are presentation over figures
// something else already decided**, and nothing in this file computes a payout. It reads
// `session.Spoils`, which `WonFight` froze.

import (
	"image"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// payoutLines is the script. **The words are `ui.SayPayout*`, out of data/wording.json**; this
// fills their holes and lays them out.
//
// **The account is a rectangle** *(owner's call)*: a heading ruled out to the block's two edges,
// then a row per claim — what pays on the left, a leader, the amount on the right — then the total
// as one more row. The prompt for the essences is not part of the account and is not here: it is in
// a box above the way out — see drawEssencePrompt.
//
// **Each row that names a figure claims it**, and the claim is what the flight to the duelist card
// is paying in — see typewriter.tick. The three parts are read out in the order the run decided
// them: interest on what you were carrying, the third of your life you kept, then what the room pays.
//
// **The total is computed here and not read off the purse**, because the purse is still climbing
// while the line is typed. It is the figure the three claims are about to add up to.
func payoutLines(gs *state.GlobalState) []proseLine {
	if gs.Run == nil {
		return nil
	}
	spoils := gs.Run.Spoils()
	total := gs.Run.Vitae() + spoils.Total()
	face := payoutFace(gs)

	var lines []proseLine
	lines = append(lines, payoutHeading(face))

	// **The interest row is always there, +0 included**, like the other two: a purse too small to
	// earn any is the row telling the player what holding more would have paid. A zero claim flies
	// nothing — see typewriter.tick.
	lines = append(lines, payoutRow(face, saidProse(ui.SayPayoutInterest, ui.Slots{
		"per": ui.Plain(strconv.Itoa(session.PropagationPer)),
		"max": ui.Plain(strconv.Itoa(session.PropagationCeiling)),
	}), payoutGain(spoils.Propagated), func(gs *state.GlobalState) int { return gs.Run.ClaimPropagation() }))

	lines = append(lines, payoutRow(face, saidProse(ui.SayPayoutLife, nil), payoutGain(spoils.FromLife),
		func(gs *state.GlobalState) int { return gs.Run.ClaimFromLife() }))

	lines = append(lines, payoutRow(face, saidProse(ui.SayPayoutRoom, nil), payoutGain(spoils.FromRoom),
		func(gs *state.GlobalState) int { return gs.Run.ClaimFromRoom() }))

	lines = append(lines, proseLine{centered: true})

	// The total is a row like the claims, and claims nothing: the purse the three are adding up to.
	lines = append(lines, payoutRow(face, saidProse(ui.SayPayoutTotal, nil),
		vitaeFigure(ui.SayText(ui.SayPayoutPurse, ui.Slots{"n": ui.Plain(strconv.Itoa(total))})), nil))

	return lines
}

// payoutAccountLines is how many lines the account is — the heading, the three claims, the gap and
// the total. The prompt for the essences is not one of them; it has a box of its own beside them.
const payoutAccountLines = 6

// payoutBlockWidth is how wide the account is: every row starts on its left edge and ends on its
// right, which is what makes the block a rectangle. It sits centered in the payout column.
const payoutBlockWidth = 560

// payoutBlock is the account's two edges, in screen pixels.
type payoutBlock struct{ left, right int }

func (b payoutBlock) mid() int { return (b.left + b.right) / 2 }

func payoutBlockOf(gs *state.GlobalState) payoutBlock {
	mid := proseColumnMid(gs)
	return payoutBlock{left: mid - payoutBlockWidth/2, right: mid + payoutBlockWidth/2}
}

// payoutFace is the type the block is set in.
func payoutFace(gs *state.GlobalState) *text.GoTextFace {
	return &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: proseTextSize}
}

// payoutHeading is the title ruled out to both edges with the leader mark.
func payoutHeading(face *text.GoTextFace) proseLine {
	title := saidProse(ui.SayPayoutHeading, nil)
	n := leaderCount(face, (float64(payoutBlockWidth)-spansWidth(title, face))/2-systems.MeasureText(" ", face))
	dots := strings.Repeat(ui.SayPayoutLeader, n)

	spans := []proseSpan{leaderSpan(dots + " ")}
	spans = append(spans, title...)
	spans = append(spans, leaderSpan(" "+dots))
	return proseLine{centered: true, spans: spans}
}

// payoutRow is one claim: its words, a leader filling the block to the amount, and the amount.
func payoutRow(face *text.GoTextFace, label []proseSpan, amount proseSpan,
	pays func(*state.GlobalState) int) proseLine {

	space := systems.MeasureText(" ", face)
	n := leaderCount(face, float64(payoutBlockWidth)-spansWidth(label, face)-spanWidth(amount, face)-2*space)
	return proseLine{
		spans: append(label, leaderSpan(" "+strings.Repeat(ui.SayPayoutLeader, n)+" ")),
		right: []proseSpan{amount},
		pays:  pays,
	}
}

// leaderCount is how many leader marks fit in a width, never fewer than none.
func leaderCount(face *text.GoTextFace, width float64) int {
	mark := systems.MeasureText(ui.SayPayoutLeader, face)
	if mark <= 0 || width <= 0 {
		return 0
	}
	return int(width / mark)
}

// leaderSpan is a run of leader marks, faded toward the ground so the eye travels along it to the
// amount rather than stopping on it.
func leaderSpan(s string) proseSpan {
	return proseSpan{text: s, ink: systems.ColorToward(ui.GroundInk, ui.ScreenGround, 55)}
}

// payoutGain is a figure being paid in, on the vitae sheet.
func payoutGain(n int) proseSpan {
	return vitaeFigure(ui.SayText(ui.SayPayoutGain, ui.Slots{"n": ui.Plain(strconv.Itoa(n))}))
}

// vitaeFigure is an amount in the vitae crimson, set on the figure sheet whenever it can be.
func vitaeFigure(s string) proseSpan {
	return proseSpan{text: s, ink: ui.VitaeInk, figure: systems.FigureCovers(s)}
}

// saidProse fills a template and colors its spans for the typewriter. **A span the prose sheet
// cannot set and the figure sheet can is a figure** — the vitae mark, written as its own colored
// word in a template.
func saidProse(tmpl string, slots ui.Slots) []proseSpan {
	said := ui.Say(tmpl, slots)
	out := make([]proseSpan, 0, len(said))
	for _, s := range said {
		out = append(out, proseSpan{
			text:   s.Text,
			ink:    ui.InkNamed(s.Ink),
			figure: !systems.ProseCovers(s.Text) && systems.FigureCovers(s.Text),
		})
	}
	return out
}

// proseLineAt is where a line's payment sets off from: the middle of its amount, on its row.
//
// **It is handed the whole script**, because the block is laid out from its bottom edge up — where
// the third line sits is a fact about how many there are, see proseTop — and because the amount's
// middle is a fact about how wide it is.
func proseLineAt(gs *state.GlobalState, lines []proseLine, i int) image.Point {
	y := proseTop(gs, len(lines)) + i*proseLineGap
	block := payoutBlockOf(gs)
	if i >= len(lines) || len(lines[i].right) == 0 {
		return image.Pt(block.mid(), y)
	}
	w := spansWidth(lines[i].right, payoutFace(gs))
	return image.Pt(block.right-int(w/2), y)
}

// drawProse puts the narration up: every line typed so far, and whatever figure is in the air.
func (s *PostBattleScene) drawProse(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	block := payoutBlockOf(gs)
	for i, line := range s.prose.lines {
		left, right, on := s.prose.visible(i)
		if !on {
			break
		}
		drawProseLine(screen, face, line, left, right, block,
			proseTop(gs, len(s.prose.lines))+i*proseLineGap)
	}

	s.prose.drawVitaeFlight(gs, screen, face)
}

// flyEssencesIn starts the offer's arrival: the essences come in from the sides of the screen.
//
// **They fly rather than appear**, which is the rule everywhere in this game and is doing real work
// here — the payout beside them says two essences are bleeding from the enemy, and a card that was
// already on screen would contradict it. **It runs at Init now that the visit has one stage**
// *(2026-09-18)*, so the flight is the screen opening rather than the narration handing over.
func (s *PostBattleScene) flyEssencesIn() {
	for i := range s.entry {
		s.entry[i] = ui.NewTravel(i*essenceEntryStagger(), essenceEntryTicks())
	}
}

// The essences' arrival: how long one takes to cross in, and how far apart the two set off.
func essenceEntryTicks() int   { return ui.Beat(1, 1) }
func essenceEntryStagger() int { return ui.Beat(1, 4) }

// essenceArrivingAt is where one offered essence is *drawn* while it flies in — off the near side of the
// screen at the start of its journey, and in its seat by the end.
//
// **The seat itself never moves**, which is why this is separate from `essenceSlot`: the hit test is
// against the seat, so a card can be clicked the moment it is on screen and the flight stays a
// thing to look at. Presentation may never change what a click means.
func (s *PostBattleScene) essenceArrivingAt(gs *state.GlobalState, i int) image.Point {
	seat := s.essenceSlot(gs, i)
	if i >= len(s.entry) || s.entry[i].Done() {
		return seat.Min
	}

	// Left-hand cards come in from the left edge, right-hand ones from the right, so the two arrive
	// from opposite sides rather than sweeping across each other.
	from := image.Rect(-cardWidth-40, seat.Min.Y, -40, seat.Max.Y)
	if seat.Min.X >= gs.PctX(50) {
		from = image.Rect(gs.ScreenWidth+40, seat.Min.Y, gs.ScreenWidth+cardWidth+40, seat.Max.Y)
	}
	return ui.FlyingTo(from, seat, s.entry[i])
}

// Where the settled stage's two lines of type sit. **Measured from the build band, never from the
// top of the screen** — see the drops in postbattle.go for what reading absolute pixels cost.
func offerTitleTop(gs *state.GlobalState) int { return buildBandBottom(gs) + offerTitleDrop }
func offerHintTop(gs *state.GlobalState) int  { return buildBandBottom(gs) + offerHintDrop }

// proseTop is where the payout's first line sits, for a script of n lines.
//
// **The block is centered between the two panels** *(owner's call)*: the build band above it and
// the dealt row below, so the account sits in the middle of the space it is read in rather than
// against either edge.
//
// **The last line's *height* is what the block's bottom is, not the pitch.** proseLineGap is the
// distance between two lines and says nothing about where the last one ends; see proseLineHeight.
func proseTop(gs *state.GlobalState, n int) int {
	if n < 1 {
		n = 1
	}
	h := (n-1)*proseLineGap + proseLineHeight
	top, bottom := buildBandBottom(gs), offerRowTop(gs)
	return top + (bottom-top-h)/2
}

// essencePromptRect is where the offer's prompt is written: the width of the way out, standing
// directly above it, so the question and the answer that is not a card read as one column.
func essencePromptRect(gs *state.GlobalState, n int) image.Rectangle {
	_, button := essenceRowSeats(gs, n)
	bottom := button.Min.Y - essenceRowGap
	return image.Rect(button.Min.X, bottom-essencePromptHeight, button.Max.X, bottom)
}

// essencePromptHeight is one line of the payout's type with air above and below it.
const essencePromptHeight = proseLineHeight + 2*16

// drawEssencePrompt writes the offer's prompt centered in its place above the way out, straight on
// the table with no panel behind it.
func drawEssencePrompt(gs *state.GlobalState, screen *ebiten.Image, face *text.GoTextFace, n int) {
	r := essencePromptRect(gs, n)
	spans := saidProse(ui.SayPayoutEssence, nil)
	x := float64(r.Min.X+r.Dx()/2) - spansWidth(spans, face)/2
	drawProseSpans(screen, face, spans, x, r.Min.Y+(r.Dy()-proseLineHeight)/2)
}
