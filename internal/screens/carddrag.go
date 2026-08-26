package screens

// **Picking a card up and putting it down somewhere else.** Every row of cards that can be
// reordered uses this, so the gesture is one gesture rather than one per screen.
//
// It was the action box's alone until 2026-08-26 — a press lifecycle written into the hand, over
// the hand's own list, with the hand's own indices. The worn ring row then needed the same thing,
// and worn order is a *rule* there, so a second implementation would have been a second set of
// off-by-ones on the one row where a mistake changes what a duel does.
//
// **What is shared is the lifecycle, not the list.** A press is a candidate click until the cursor
// has travelled `dragThreshold`; past that the card leaves the row and rides the cursor; a release
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
// none of these can be precomputed: the hand's pitch closes up as cards are drawn and the ring
// row's spreads as rings are bought.
//
// **`rowLift` and `rowReturn` are a pair and the row decides what they mean.** The hand actually
// removes the card from its list, because the list is the hand. The ring row removes nothing: the
// run is the authority on what is worn, so the row only remembers which seat is empty and commits
// the whole move at the drop. Both end up in the same place, which is what `rowReturn(from, to)`
// says — the card that was at `from` is now at `to`.
type dragRow interface {
	// rowLen is how many cards the row holds with nothing lifted.
	rowLen() int

	// rowSlot is where the card at i is drawn, and therefore where it is clicked.
	rowSlot(gs *state.GlobalState, i int) image.Rectangle

	// rowZone is the region a release has to land in for the drop to count as a reorder.
	rowZone(gs *state.GlobalState) image.Rectangle

	// rowDropIndex is which seat the cursor is over, already clamped to the row.
	rowDropIndex(gs *state.GlobalState) int

	// rowLift takes the card at i out of the row, visually or actually.
	rowLift(i int)

	// rowReturn puts it down: the card that was at `from` now sits at `to`. A cancelled drag
	// passes the same index twice, which every row has to treat as putting it back untouched.
	rowReturn(from, to int)

	// rowClick is a press that never travelled far enough to become a drag.
	rowClick(i int)
}

// cardDrag is the press currently in progress over one row. The zero value is no press, which is
// what lets it be a plain field rather than a pointer that has to be nil-checked at every use.
type cardDrag struct {
	// held is a button down over a card; active is that press having travelled far enough to be a
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

// dragging reports whether a card is currently riding the cursor. Drawing reads it: the row skips
// the seat the card left and the card is drawn last, over everything.
func (d *cardDrag) dragging() bool { return d.active }

// origin is the seat the dragged card was picked up from. A row's drawing reads it to leave that
// seat empty; it means nothing unless dragging reports true.
func (d *cardDrag) origin() int { return d.originIndex }

// at is where the dragged card's top-left corner sits this frame.
func (d *cardDrag) at(gs *state.GlobalState) image.Point {
	return image.Pt(gs.MouseX-d.grabDX, gs.MouseY-d.grabDY)
}

// update runs one tick of the lifecycle. The caller decides *whether* the row is live — a round
// resolving, a gated tutorial step, a panel covering the screen — and calls cancel instead when it
// is not.
func (d *cardDrag) update(gs *state.GlobalState, row dragRow) {
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
// **Backwards through the row**, because overlapping cards are drawn front to back in index order:
// the last card covering a point is the one visibly on top of it, and that is the one the press has
// to mean. Both rows overlap once they are full.
func (d *cardDrag) begin(gs *state.GlobalState, row dragRow) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := row.rowLen() - 1; i >= 0; i-- {
		slot := row.rowSlot(gs, i)
		if !at.In(slot) {
			continue
		}
		*d = cardDrag{
			held:        true,
			originIndex: i,
			pressX:      gs.MouseX,
			pressY:      gs.MouseY,
			grabDX:      gs.MouseX - slot.Min.X,
			grabDY:      gs.MouseY - slot.Min.Y,
		}
		return
	}
}

// promote turns a held press into a drag once the cursor has moved far enough, lifting the card out
// of the row as it does.
//
// **The lift happens here rather than on release**, so the seat empties under the cursor and the
// drop index is measured against the row the card is actually going to land in.
func (d *cardDrag) promote(gs *state.GlobalState, row dragRow) {
	if d.active {
		return
	}
	if abs(gs.MouseX-d.pressX) < dragThreshold && abs(gs.MouseY-d.pressY) < dragThreshold {
		return
	}

	d.active = true
	row.rowLift(d.originIndex)
}

// end resolves the press: a drag lands the card, a press that never travelled is a click.
//
// **Released outside the row, the card goes back where it came from.** There is no drag-to-discard
// gesture anywhere in the game, so a drop into empty space has to mean "never mind" rather than
// something destructive the player cannot see coming.
func (d *cardDrag) end(gs *state.GlobalState, row dragRow) {
	origin, active := d.originIndex, d.active
	*d = cardDrag{}

	if !active {
		row.rowClick(origin)
		return
	}

	to := origin
	if image.Pt(gs.MouseX, gs.MouseY).In(row.rowZone(gs)) {
		to = row.rowDropIndex(gs)
	}
	row.rowReturn(origin, to)
}

// cancel puts any in-flight card back and forgets the press.
//
// **It puts the card back rather than simply forgetting**, which is what a screen going
// uninteractable mid-press needs: a gate coming up while a card is lifted would otherwise leave it
// stuck to the cursor with no release that can put it down.
func (d *cardDrag) cancel(row dragRow) {
	if d.active {
		row.rowReturn(d.originIndex, d.originIndex)
	}
	*d = cardDrag{}
}
