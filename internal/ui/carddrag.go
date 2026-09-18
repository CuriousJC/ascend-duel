package ui

// **Picking a card up and putting it down somewhere else.** Every row of cards that can be
// reordered uses this, so the gesture is one gesture rather than one per screen.
//
// It was the action box's alone until 2026-08-26 — a press lifecycle written into the hand, over
// the hand's own list, with the hand's own indices. The worn relic row then needed the same thing,
// and worn order is a *rule* there, so a second implementation would have been a second set of
// off-by-ones on the one row where a mistake changes what a duel does.
//
// **What is shared is the lifecycle, not the list.** A press is a candidate click until the cursor
// has traveled `dragThreshold`; past that the card leaves the row and rides the cursor; a release
// inside the row lands it at whatever index the cursor is over and a release outside puts it back.
// Everything about *which* cards, *where* they sit and *what a click means* is the row's, through
// dragRow.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// dragRow is what a row of cards has to be able to answer for cardDrag to run on it.
//
// **Every method takes the state**, because a row's geometry is a function of the screen size and
// none of these can be precomputed: the hand's pitch closes up as cards are drawn and the relic
// row's spreads as relics are bought.
//
// **`RowLift` and `RowReturn` are a pair and the row decides what they mean.** The hand actually
// removes the card from its list, because the list is the hand. The relic row removes nothing: the
// run is the authority on what is worn, so the row only remembers which seat is empty and commits
// the whole move at the drop. Both end up in the same place, which is what `RowReturn(from, to)`
// says — the card that was at `from` is now at `to`.
// RaisedSeat is which card of an overlapping row the cursor is resting on, or -1.
//
// **One reading for every row in the game** *(owner's call, 2026-09-17)*. The hand, the worn relics
// and the sack are the same widget at three widths holding three kinds of card, and all three
// overlap once they are full — so "the card I am pointing at is a sliver of the one in front of it"
// is one problem with one answer. The caller draws the seat this names *last*, so it comes out
// whole; it is raised rather than moved, so the row does not rearrange itself under a cursor that is
// about to click.
//
// **Counted from the top of the stack down**, because the last seat drawn is the one actually on
// top. That makes the card that lifts the card a click would land on, by construction rather than by
// two pieces of code agreeing.
//
// **`raise` is the caller's dwell, and every caller passes the tooltip's own** — see
// models.Tooltip.Showing, so the card and the panel about it arrive on the same tick *(owner's call,
// 2026-09-17)*. Lifting the instant the cursor arrives makes a row flinch at a cursor crossing it on
// the way somewhere else, which is the strobe DwellTicks already exists to stop; lifting halfway was
// tried and reads as a stutter, one gesture answered twice a beat apart. One dwell, one moment.
func RaisedSeat(gs *state.GlobalState, row DragRow, raise bool) int {
	if !raise || !gs.CursorAllowed() {
		return -1
	}
	at := image.Pt(gs.MouseX, gs.MouseY)
	return HoveredSeat(at, row.RowLen(), func(i int) image.Rectangle { return row.RowSlot(gs, i) })
}

// HoveredSeat is which seat of an overlapping row a point lands on: **the last one drawn that
// covers it**, or -1.
//
// **Every row in this game hit-tests through this, and that is the whole point of it** *(owner's
// call, 2026-09-18)*. A row of cards is drawn front to back in index order, so once it is packed
// tight enough to overlap, the card the player can see under the cursor is the *last* one covering
// that point — and a walk that stops at the first one answers with the card behind it. That had
// gone wrong in four places at once: the worn relics, the sack and the build band each grew their
// own loop and each walked forwards, so a tooltip named one card while the row raised another and
// a click took a third. The hand walked backwards and was right, which is what made the failure so
// hard to see — one row of the four behaved.
//
// **It takes a count and a seat function rather than a DragRow**, because the rows that need it are
// not all draggable: the reward screen's offer, the vial's, the pouch's shelf. An interface only
// two of the callers could satisfy would have left the others writing the loop again, which is the
// thing that went wrong in the first place.
//
// **A seat is a rectangle and a card is a rounded picture**, so the transparent corners of the card
// on top still answer for the few pixels of the card behind them. That is a handful of pixels at
// each end of a 280-tall card and is deliberately not solved here: clipping to the silhouette means
// every caller knowing which style it drew, which is a bigger seam than the one it closes.
func HoveredSeat(at image.Point, n int, slot func(i int) image.Rectangle) int {
	for i := n - 1; i >= 0; i-- {
		if at.In(slot(i)) {
			return i
		}
	}
	return -1
}

type DragRow interface {
	// RowLen is how many cards the row holds with nothing lifted.
	RowLen() int

	// RowSlot is where the card at i is drawn, and therefore where it is clicked.
	RowSlot(gs *state.GlobalState, i int) image.Rectangle

	// RowZone is the region a release has to land in for the drop to count as a reorder.
	RowZone(gs *state.GlobalState) image.Rectangle

	// RowDropIndex is which seat the cursor is over, already clamped to the row.
	RowDropIndex(gs *state.GlobalState) int

	// RowLift takes the card at i out of the row, visually or actually.
	RowLift(i int)

	// RowReturn puts it down: the card that was at `from` now sits at `to`. A canceled drag
	// passes the same index twice, which every row has to treat as putting it back untouched.
	RowReturn(from, to int)

	// RowClick is a press that never traveled far enough to become a drag.
	RowClick(i int)
}

// CardDrag is the press currently in progress over one row. The zero value is no press, which is
// what lets it be a plain field rather than a pointer that has to be nil-checked at every use.
type CardDrag struct {
	// held is a button down over a card; active is that press having traveled far enough to be a
	// drag. A press is a candidate click until it is active, and only an active one has lifted
	// anything out of the row.
	held   bool
	active bool

	// originIndex is the index the card was picked up from, in the row as it stood before the
	// lift. Read through origin().
	originIndex int

	pressX, pressY int

	// grabDX/grabDY keep the cursor where it landed on the card, so picking one up does not snap
	// it to the cursor.
	grabDX, grabDY int
}

// Dragging reports whether a card is currently riding the cursor. Drawing reads it: the row skips
// the seat the card left and the card is drawn last, over everything.
func (d *CardDrag) Dragging() bool { return d.active }

// Origin is the seat the dragged card was picked up from. A row's drawing reads it to leave that
// seat empty; it means nothing unless dragging reports true.
func (d *CardDrag) Origin() int { return d.originIndex }

// At is where the dragged card's top-left corner sits this frame.
func (d *CardDrag) At(gs *state.GlobalState) image.Point {
	return image.Pt(gs.MouseX-d.grabDX, gs.MouseY-d.grabDY)
}

// Update runs one tick of the lifecycle. The caller decides *whether* the row is live — a round
// resolving, a gated tutorial step, a panel covering the screen — and calls cancel instead when it
// is not.
func (d *CardDrag) Update(gs *state.GlobalState, row DragRow) {
	if !d.held {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			d.begin(gs, row)
		}
		return
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		d.end(gs, row)
		return
	}

	d.promote(gs, row)
}

// begin records a press over a card without yet committing to what it means.
//
// **Through HoveredSeat**, so a press means the card the row raises and the tooltip describes. Every
// row overlaps once it is full and the one on top is the last drawn; see HoveredSeat.
func (d *CardDrag) begin(gs *state.GlobalState, row DragRow) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	i := HoveredSeat(at, row.RowLen(), func(i int) image.Rectangle { return row.RowSlot(gs, i) })
	if i < 0 {
		return
	}
	slot := row.RowSlot(gs, i)
	*d = CardDrag{
		held:        true,
		originIndex: i,
		pressX:      gs.MouseX,
		pressY:      gs.MouseY,
		grabDX:      gs.MouseX - slot.Min.X,
		grabDY:      gs.MouseY - slot.Min.Y,
	}
}

// promote turns a held press into a drag once the cursor has moved far enough, lifting the card out
// of the row as it does.
//
// **The lift happens here rather than on release**, so the seat empties under the cursor and the
// drop index is measured against the row the card is actually going to land in.
func (d *CardDrag) promote(gs *state.GlobalState, row DragRow) {
	if d.active {
		return
	}
	if Abs(gs.MouseX-d.pressX) < dragThreshold && Abs(gs.MouseY-d.pressY) < dragThreshold {
		return
	}

	d.active = true
	row.RowLift(d.originIndex)
}

// end resolves the press: a drag lands the card, a press that never traveled is a click.
//
// **Released outside the row, the card goes back where it came from.** There is no drag-to-discard
// gesture anywhere in the game, so a drop into empty space has to mean "never mind" rather than
// something destructive the player cannot see coming.
func (d *CardDrag) end(gs *state.GlobalState, row DragRow) {
	origin, active := d.originIndex, d.active
	*d = CardDrag{}

	if !active {
		row.RowClick(origin)
		return
	}

	to := origin
	if image.Pt(gs.MouseX, gs.MouseY).In(row.RowZone(gs)) {
		to = row.RowDropIndex(gs)
	}
	row.RowReturn(origin, to)
}

// Cancel puts any in-flight card back and forgets the press.
//
// **It puts the card back rather than simply forgetting**, which is what a screen going
// uninteractable mid-press needs: a gate coming up while a card is lifted would otherwise leave it
// stuck to the cursor with no release that can put it down.
func (d *CardDrag) Cancel(row DragRow) {
	if d.active {
		row.RowReturn(d.originIndex, d.originIndex)
	}
	*d = CardDrag{}
}
