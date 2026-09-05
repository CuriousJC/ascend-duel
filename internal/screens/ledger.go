package screens

// **The ledger panel: the whole run's account, scrollable, with the working under every blow.**
//
// It replaced the fight log on 2026-09-02 *(owner's call)*. The log held one fight, could not be
// scrolled, and dropped its earliest rows with a `... N earlier` line that admitted the problem in
// its own comment. Three things changed and each was asked for:
//
//   - **The run, not the fight.** `session.Session` keeps the account, so it survives the fight
//     that wrote it and answers "how did my whole climb go" as well as "what just happened".
//   - **A dragged scrollbar** *(owner's call)*, because the input vocabulary is clicks, drags and
//     hover and a wheel would be a fourth verb. See models.Scrollbar.
//   - **The arithmetic.** Every term of a blow, with the ring that priced it named beside it. See
//     prose_terms.go.
//
// **The current fight is expanded and the past is one line a fight, opened by clicking it**
// *(owner's call)*. A thirty-fight run is thousands of lines, and a panel that opened onto all of
// them would be a scrollbar with nothing to aim at.
//
// **It is chrome, not a scene.** It is reachable from every screen, and a screen cannot be one:
// leaving the combat screen and coming back re-runs `Init`, which deals a fresh duel — so a
// ledger that navigated would destroy the fight it was opened to read about. `internal/game`
// owns the button and holds this panel; while it is up the active scene is not updated at all,
// which freezes pacing and, as everywhere else, cannot change an outcome.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ledgerPane is the panel the account is written on: **the fight log's own colours**, because
// these are the log's rows and an off-white ground is what makes three saturated verb inks
// readable. See pane.go.
var ledgerPane = panePlacement{
	title:     "THE LEDGER",
	color:     paneEdge,
	fill:      color.RGBA{R: 234, G: 230, B: 224, A: 255},
	ink:       color.RGBA{R: 34, G: 32, B: 38, A: 255},
	nowInk:    color.RGBA{R: 178, G: 22, B: 106, A: 255},
	rowHeight: ledgerRowHeight,
	firstRow:  paneFirstRow,
	textSize:  ledgerTextSize,
	bold:      true,
}

// The two grounds a fight is drawn on.
//
// **A heading is a dark blue band and its fight's rows sit on a tint of the same blue** *(owner's
// call, 2026-09-02)*. A run is one line per fight with one of them opened, so without a ground the
// forty rows of the opened fight and the next fight's heading are one undifferentiated list — the
// band says where a fight starts and the tint says how far it goes.
//
// **Blue because nothing else on this panel is.** Hue belongs to the elements — see CLAUDE.md — and
// this is a *surface* rather than a mark on one: it is behind the text, at a strength no element
// would be drawn at, so it groups rows without competing with the figures written on it.
//
// **There are two of each and consecutive fights alternate between them** *(owner's call,
// 2026-09-02)*. Both are dark blue, because the pair has to read as one device rather than as two
// meanings — the second is a colder, lighter navy, far enough apart to tell where one fight's block
// ends and the next begins without the run looking striped in two different colours.
var (
	ledgerBandInk = color.RGBA{R: 236, G: 240, B: 248, A: 255}
	ledgerBands   = [2]color.RGBA{{R: 28, G: 52, B: 92, A: 255}, {R: 44, G: 78, B: 118, A: 255}}
	ledgerGrounds = [2]color.RGBA{{R: 222, G: 228, B: 238, A: 255}, {R: 214, G: 226, B: 234, A: 255}}
)

const (
	// **The ledger is written bigger than the panes it inherited from** *(owner's call,
	// 2026-09-02)*: 20 points against the 16 every other pane uses, at a 27px pitch against 22.
	// It is read at a distance and at length rather than glanced at during a round, and a column
	// of figures at 16 on an off-white ground was hard work.
	//
	// **The two move together.** A bigger face at the old pitch overlaps its neighbours, and a
	// bigger pitch alone just spreads small text out. The panel's capacity is derived from the
	// pitch, so it falls out of this rather than being re-tuned by hand.
	ledgerTextSize  = 20
	ledgerRowHeight = 27

	// ledgerBottomInset is the air kept under the last row, so a full panel does not write its
	// bottom line onto its own edge.
	ledgerBottomInset = 16

	// The scrollbar's column, on the panel's right edge.
	ledgerScrollWidth = 18
	ledgerScrollInset = 10
)

// ledgerRow is one drawn line and what clicking it does.
//
// **A fight's heading is the only thing on this panel that can be clicked**, which is why the row
// carries a fight number rather than a callback: the panel has one action and a row either offers
// it or does not.
type ledgerRow struct {
	row paneRow

	// fight is the record a click on this row folds or unfolds, or 0 for a row that is not a
	// heading.
	fight int
}

// LedgerPanel is the panel and everything it remembers between frames.
//
// **Exported because internal/game holds it**, which is the one place outside this package that
// knows a screen exists. It is the deliberate consequence of the ledger being chrome.
type LedgerPanel struct {
	open bool

	// expanded is which past fights the player has opened, by fight number. **A set of the
	// exceptions rather than a flag per fight**, so a run's account does not carry a growing
	// structure for the ordinary case, which is that everything old is folded.
	expanded map[int]bool

	scroll *models.Scrollbar
	closer modalCloser

	// rows is the built list, rebuilt when the account or the folding changed rather than every
	// frame — a run's ledger is thousands of lines and this is a panel, not an animation.
	rows []ledgerRow

	// built is the signature the rows were built from. See ledgerSignature.
	built ledgerKey

	// pinned says the panel has not been scrolled by hand since it opened, in which case it stays
	// at the bottom — **the newest thing that happened, which is what a player opening it
	// mid-fight is looking for**. A drag unpins it, so scrolling back does not fight the panel.
	pinned bool
}

// ledgerKey is what the built rows depend on. **A signature rather than a dirty flag**, because
// the account grows underneath the panel — a round can finish while it is open — and a flag would
// have to be raised by whoever wrote the round, which is a call site in another package.
type ledgerKey struct {
	fights, rounds, lines, folds int
}

// IsOpen reports whether the panel is up. The frame reads it to know the scene is inert.
func (p *LedgerPanel) IsOpen() bool { return p != nil && p.open }

// Toggle opens or closes the panel.
func (p *LedgerPanel) Toggle() {
	p.open = !p.open
	if p.open {
		// **Opened at the bottom**, on the newest lines. A run's account read from the top would
		// open on the first room of the tower every time.
		p.pinned = true
		p.built = ledgerKey{}
	}
}

// Close shuts it, for a caller that has a reason of its own.
func (p *LedgerPanel) Close() { p.open = false }

// Update runs the panel: the closing X, the scrollbar, and a click on a heading.
//
// **Nothing else in the game is updated while it is up**, which is the frame's business rather
// than this file's — see internal/game/chrome.go. What matters here is that this panel can change
// nothing about a run: it folds rows and it scrolls.
func (p *LedgerPanel) Update(gs *state.GlobalState) {
	if !p.open {
		return
	}

	p.build(gs)

	if p.closer.update(gs) {
		p.open = false
		return
	}

	r := ledgerPanelRect(gs)
	capacity := ledgerCapacity(r)

	if p.scroll == nil {
		p.scroll = models.NewScrollbar(ledgerScrollWidth, r.Dy()-ledgerPane.firstRow-ledgerBottomInset)
		p.scroll.Ground = ledgerPane.fill
	}
	track := ledgerScrollRect(r)
	p.scroll.Width, p.scroll.Height = track.Dx(), track.Dy()
	p.scroll.ScreenX = track.Min.X + track.Dx()/2
	p.scroll.ScreenY = track.Min.Y + track.Dy()/2
	p.scroll.Total, p.scroll.Visible = len(p.rows), capacity

	if p.pinned {
		p.scroll.Offset = len(p.rows) - capacity
	}

	before := p.scroll.Offset
	systems.UpdateScrollbar(gs, p.scroll)
	if p.scroll.Offset != before {
		// A hand on the bar is the player choosing where to be, and the panel stops following the
		// end of the account until it is opened again.
		p.pinned = false
	}

	p.clickRow(gs, r, capacity)
}

// clickRow folds or unfolds the fight whose heading was clicked.
//
// **The row's rectangle is derived the way drawPane derives it**, from the same first-row offset
// and pitch, so the line that lights up under the cursor is the line that is drawn there. Two
// arithmetics over one layout is how a panel comes to have a clickable region that is not where
// the text is.
func (p *LedgerPanel) clickRow(gs *state.GlobalState, r image.Rectangle, capacity int) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)
	if !at.In(r) {
		return
	}

	for i := 0; i < capacity; i++ {
		idx := p.scroll.Offset + i
		if idx < 0 || idx >= len(p.rows) || p.rows[idx].fight == 0 {
			continue
		}
		top := r.Min.Y + ledgerPane.firstRow + i*ledgerPane.rowHeight
		if at.Y < top || at.Y >= top+ledgerPane.rowHeight {
			continue
		}
		if p.expanded == nil {
			p.expanded = map[int]bool{}
		}
		n := p.rows[idx].fight
		p.expanded[n] = !p.expanded[n]

		// A fold changes what is on the panel under the cursor, so the view stops following the
		// end of the account — otherwise opening an old fight would scroll straight past it.
		p.pinned = false
		p.built = ledgerKey{}
		return
	}
}

// Draw covers the screen and writes the account onto the panel.
func (p *LedgerPanel) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !p.open {
		return
	}

	modalScrim(screen)

	r := ledgerPanelRect(gs)
	capacity := ledgerCapacity(r)

	offset := 0
	if p.scroll != nil {
		offset = p.scroll.Offset
	}
	if offset > len(p.rows)-capacity {
		offset = len(p.rows) - capacity
	}
	if offset < 0 {
		offset = 0
	}

	rows := make([]paneRow, 0, capacity)
	for i := offset; i < len(p.rows) && len(rows) < capacity; i++ {
		rows = append(rows, p.rows[i].row)
	}

	drawPane(gs, screen, ledgerPane, r, rows)

	if p.scroll != nil {
		systems.DrawScrollbar(gs, screen, p.scroll)
	}
	p.closer.draw(gs, screen)
}

// ledgerPanelRect is the panel's footprint: the modal one, so the player learns one shape.
func ledgerPanelRect(gs *state.GlobalState) image.Rectangle { return modalPanelRect(gs) }

// ledgerScrollRect is the bar's column, down the panel's right edge and clear of the title.
func ledgerScrollRect(r image.Rectangle) image.Rectangle {
	right := r.Max.X - ledgerScrollInset
	return image.Rect(right-ledgerScrollWidth, r.Min.Y+ledgerPane.firstRow,
		right, r.Max.Y-ledgerBottomInset)
}

// ledgerCapacity is how many rows the panel holds. **Derived from the panel and the pitch rather
// than written down**, for the reason the log derived its own: a constant claiming a capacity the
// panel does not have is a panel that silently drops lines.
func ledgerCapacity(r image.Rectangle) int {
	n := (r.Dy() - ledgerPane.firstRow - ledgerBottomInset) / ledgerPane.rowHeight
	if n < 1 {
		return 1
	}
	return n
}

// build lays the account out as rows, if anything it depends on has changed.
func (p *LedgerPanel) build(gs *state.GlobalState) {
	key := ledgerSignature(gs, p.expanded)
	if key == p.built && p.rows != nil {
		return
	}
	p.built = key
	p.rows = ledgerRows(gs, p.expanded)
}

// ledgerSignature is what the built rows depend on: how much account there is, and how much of it
// is folded open.
func ledgerSignature(gs *state.GlobalState, expanded map[int]bool) ledgerKey {
	var key ledgerKey
	if gs.Run == nil {
		return key
	}
	for _, f := range gs.Run.LedgerFights() {
		key.fights++
		key.rounds += len(f.Rounds)
		for _, r := range f.Rounds {
			key.lines += len(r.Lines)
		}
	}
	for _, open := range expanded {
		if open {
			key.folds++
		}
	}
	return key
}

// ledgerRows is the whole panel's content: each fight as a heading, expanded or not.
//
// **The fight being fought is always expanded and never clickable.** It is the one record the
// player is in the middle of, and a heading that folded it away would hide the thing the panel was
// most likely opened for.
func ledgerRows(gs *state.GlobalState, expanded map[int]bool) []ledgerRow {
	if gs.Run == nil {
		return []ledgerRow{{row: plainRow("No run to account for")}}
	}

	fights := gs.Run.LedgerFights()
	if len(fights) == 0 {
		return []ledgerRow{{row: plainRow("Nothing has happened yet")}}
	}

	open, hasOpen := gs.Run.LedgerOpenFight()

	var rows []ledgerRow
	for i, f := range fights {
		live := hasOpen && f.Number == open

		// **The alternation is by position in the list, not by fight number**, so a run whose
		// records do not start at 1 still reads as stripes rather than starting on whichever
		// colour the arithmetic happened to land on.
		band, ground := ledgerBands[i%2], ledgerGrounds[i%2]

		// The heading's own band, written in a light ink because it is text on a dark ground —
		// the one place on this panel where that is true.
		head := plainRow(ledgerHeading(f, live || expanded[f.Number]))
		head.band = band
		head.runs[0].ink = ledgerBandInk
		rows = append(rows, ledgerRow{row: head, fight: fightToggle(f, live)})

		if !live && !expanded[f.Number] {
			continue
		}
		// **Everything under an opened heading sits on a tint of the heading's own blue**, so a
		// fight reads as a block with a lid rather than as a heading followed by whatever came
		// next. It is the whole reason the band exists: the panel scrolls, and a heading that has
		// scrolled off the top has to leave something behind saying which fight is being read.
		for _, round := range f.Rounds {
			head := plainRow(fmt.Sprintf("- Round %d -", round.Number))
			head.band = ground
			rows = append(rows, ledgerRow{row: head})

			for _, l := range paneRowsFor(round.Lines) {
				l.band = ground
				rows = append(rows, ledgerRow{row: l})
			}
		}
	}
	return rows
}

// fightToggle is the fight number a heading folds, or 0 for one that cannot be folded.
func fightToggle(f session.LedgerFight, live bool) int {
	if live {
		return 0
	}
	return f.Number
}

// ledgerHeading is a fight's one line: where it was, who was in it, and how it went.
//
// **A summary a player can act on rather than a label.** Rounds and damage dealt are the two
// figures that say whether a fight went the way it should have, which is what "where did my run go
// wrong" is actually asking. The marker says whether clicking will open it.
func ledgerHeading(f session.LedgerFight, open bool) string {
	marker := "+"
	if open {
		marker = "-"
	}

	outcome := "fighting now"
	switch f.Outcome {
	case session.OutcomeWon:
		outcome = fmt.Sprintf("won in %d, %d dealt", f.RoundCount(), f.Dealt())
	case session.OutcomeLost:
		outcome = fmt.Sprintf("lost in %d, %d dealt", f.RoundCount(), f.Dealt())
	}

	return fmt.Sprintf("%s  Fight %d - floor %d - %s - %s", marker, f.Number, f.Floor, f.Enemy, outcome)
}
