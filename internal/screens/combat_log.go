package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// The fight log: every round of this fight in sentences, behind a button beside the draw pile.
//
// **It replaced the Resolution feed rather than joining it** *(built and the feed removed,
// 2026-08-18)*. That feed held one round, cleared at the start of the next, and had no scroll
// gesture to reach an earlier one — so the account of a fight was something the player had to
// have been watching. That was survivable while it was the only thing narrating a round. It
// stopped being that: the table shows both hands, cards fly to their seats, the sum is acted out
// and the damage figure travels into the bar it empties. Once the pictures say what happened, a
// running list of sentences under them is a second telling that costs a band of the screen — and
// what the sentences are *good* for is the thing the feed could not do, which is being read back.
//
// **It says nothing the events do not.** `logRows` is the walk, and it was the feed's walk: the
// prose, the merging of an action with its outcome, the one line per slot. Nothing was rewritten
// for the log, which is why a round reads here exactly as it read while it was happening.
//
// **It is the second dialog in the game and obeys the first one's rules.** A scrim over
// everything, state.ModalOpen set so the frame's own chrome stands down, every other control
// dead, and the button that opened it drawn again on top — there is no Escape key and no right
// click, so a modal has to make its exit the brightest thing on screen or it is a trap.

// The Log button: a square beside the draw pile, sharing its bottom edge.
//
// **It stands beside the pile rather than in the button strip** because the two are the same kind
// of control — something opened to look at — as against Discard and DUEL!, which commit a round.
// The bottom line of the screen now reads: what you can afford, the two things you can do about
// it, then the two things you can read.
const (
	logButtonSize = 44

	// The gap between the button and the pile's own left edge. Wider than the sort column's 8,
	// because these two controls are not a set: the pile and the log are separate things that
	// happen to stand together, and a gap tight enough to read as a stack would say they were
	// one widget.
	logButtonToDeckGap = 18

	// **One character on a square**, exactly as the sort column is and for the same reason — the
	// button is too small for a word. `L` does not collide with the `$`/`T`/`E` of the sort
	// column, which is the only other place a single character on this screen means something.
	// **It carries no tooltip yet**, though the widget exists; see the tooltip entry in TODO.md.
	logButtonLabel    = "L"
	logButtonTextSize = 30
)

// logButtonRect is where the button stands: immediately left of the pile, bottom edges level.
//
// **Both edges come off the pile**, never off a percentage. The pile is itself hung off the
// screen's bottom-right corner, so a button placed any other way would drift the first time that
// inset changed — the same staleness ringPaneRect was rewritten to avoid.
//
// It reads the pile's *bounds* rather than its front card, exactly as buttonStripSlots does: the
// backs are drawn up and to the left, so the front card's edge is not the pile's edge.
func logButtonRect(gs *state.GlobalState) image.Rectangle {
	stack := deckStackBounds(gs)
	right := stack.Min.X - logButtonToDeckGap
	bottom := stack.Max.Y
	return image.Rect(right-logButtonSize, bottom-logButtonSize, right, bottom)
}

// buildLogButton wires the button to the scene, like every other widget on this screen.
//
// **It takes the sort column's slate**, which is the colour this screen already uses for a control
// that cannot change a round. Crimson and the Discard yellow both commit one; opening a log
// commits nothing, and a committing colour would be the button saying something untrue about what
// pressing it costs.
func (s *CombatScene) buildLogButton() {
	b := models.NewButton(logButtonSize, logButtonSize, logButtonLabel, s.toggleLog)
	b.BaseColor = sortButtonColor
	b.TextSize = logButtonTextSize
	s.logButton = b
}

// toggleLog shows or hides the fight log.
func (s *CombatScene) toggleLog() {
	s.showLog = !s.showLog
}

// modalUp reports whether either dialog is covering the screen.
//
// **One predicate rather than two conditions spelled out at every call site**, because every
// control on this screen has to go dead for both and the failure is silent: a button left live
// under a dialog is a round edited through a panel the player is only reading.
func (s *CombatScene) modalUp() bool {
	return s.showDeck || s.showLog || s.combos.open
}

// updateLogButton runs the button.
//
// **It is dead while any dialog is up** *(owner's call, 2026-08-24)*, the log's own included. It
// used to run under its own overlay because it was the only thing that closed one; the panels
// carry a red X now, so an opener only ever opens.
func (s *CombatScene) updateLogButton(gs *state.GlobalState) {
	s.logButton.Latched = s.showLog
	setEnabled(s.logButton, !s.modalUp())
	systems.UpdateButton(gs, s.logButton)
}

// drawLogButton draws it.
func (s *CombatScene) drawLogButton(gs *state.GlobalState, screen *ebiten.Image) {
	systems.DrawButton(gs, screen, s.logButton)
}

// logPane is the panel the log is written on.
//
// **It is the Resolution feed's colours, with a title and a heading gap.** The rows are the
// feed's rows and this is where they are read now, so the panel keeps the surface they were
// legible on: an off-white ground is what makes three saturated verb inks readable — the reason
// Resolution went light in the first place, and a dark dialog would have undone it.
var logPane = panePlacement{
	title:     "Fight log",
	color:     paneEdge,
	fill:      color.RGBA{R: 234, G: 230, B: 224, A: 255},
	ink:       color.RGBA{R: 34, G: 32, B: 38, A: 255},
	nowInk:    color.RGBA{R: 178, G: 22, B: 106, A: 255},
	rowHeight: paneTextRowHeight,
	firstRow:  paneFirstRow,
}

// logPanelRect is the dialog's footprint, which is the deck overlay's.
//
// **Two dialogs at two sizes would read as two kinds of thing.** They are the same kind of thing —
// a panel covering the screen, closed by the control that opened it — so they take the same
// rectangle and the player learns one shape.
func logPanelRect(gs *state.GlobalState) image.Rectangle { return modalPanelRect(gs) }

// logHintReserve is the room kept at the bottom for the closing hint, so a full log cannot write
// its last line over the one sentence saying how to get out.
const logHintReserve = 40

// logCapacity is how many rows the panel holds. Derived from the panel and the pitch rather than
// written down, for the reason the feed derived its own: a constant claiming a capacity the panel
// does not have is a panel that silently drops lines.
func logCapacity(r image.Rectangle) int {
	n := (r.Dy() - paneFirstRow - logHintReserve) / paneTextRowHeight
	if n < 1 {
		return 1
	}
	return n
}

// fightLogRows is every round of this fight, oldest first, each under its own heading.
//
// **The round in progress is included only as far as playback has reached**, which is the same
// slice the feed drew and the same protection: the dialog can be opened mid-playback, and a log
// built from the whole resolved round would hand the player the rest of it.
//
// **A heading is a row with no swatch and no verb**, which drawPane centres — so the rounds read
// as blocks with a line between them rather than as one list the reader has to count through.
func (s *CombatScene) fightLogRows() []paneRow {
	var rows []paneRow

	add := func(n int, events []combat.Event) {
		if len(events) == 0 {
			return
		}
		rows = append(rows, paneRow{prefix: fmt.Sprintf("- Round %d -", n)})
		rows = append(rows, s.logRows(events)...)
	}

	for i, events := range s.rounds {
		add(i+1, events)
	}

	end := s.cursor + 1
	if end > len(s.log) {
		end = len(s.log)
	}
	if end > 0 {
		add(len(s.rounds)+1, s.log[:end])
	}

	if len(rows) == 0 {
		return []paneRow{{prefix: "Nothing has happened yet"}}
	}
	return rows
}

// drawLogOverlay covers the screen with the fight so far.
//
// **Overflow keeps the newest rows and says how many it dropped**, which is the answer the deck
// overlay's `+N more not shown` already gives. It is honest and it
// is not what this panel wants to be: a log is for reading *back*, and with no scroll gesture in
// the input vocabulary the earliest rounds of a long fight are reachable only by the panel getting
// bigger. See TODO.md.
func (s *CombatScene) drawLogOverlay(gs *state.GlobalState, screen *ebiten.Image) {
	if !s.showLog {
		return
	}

	// The scrim every modal puts up; see modal.go.
	modalScrim(screen)

	r := logPanelRect(gs)
	rows := s.fightLogRows()

	if capacity := logCapacity(r); len(rows) > capacity {
		hidden := len(rows) - capacity + 1
		rows = append([]paneRow{{prefix: fmt.Sprintf("... %d earlier", hidden)}}, rows[hidden:]...)
	}

	s.drawPane(gs, screen, logPane, r, rows)
}
