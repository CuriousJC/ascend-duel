package screens

// The control column: what stands down the right-hand side of the combat screen, between the
// enemy card and the bottom of the action-point bar.
//
// **It is two groups rather than one stack** *(2026-09-04, owner's call)*, and they are anchored
// at opposite ends because they belong to different things:
//
//   - **The sort tabs** are a block of three, no air between them, hung off the *cards'* right
//     edge and starting on the hand's top line. They arrange the hand and belong to it, so they
//     are tied to the row rather than to the column — see sortTabRect.
//   - **The two panel buttons**, HANDS and the frame's LEDGER, stack *upward* from the bottom of
//     the action-point bar on the column's own line. They open a page over the game and belong to
//     the screen, not to the row.
//
// **The line the second group stands on is the enemy card's left edge**, so the corner and the
// buttons under it read as one strip. The cards stop sortColumnGap short of it — see
// cardBandWidth, which is what that costs.
//
// **It is exported because the frame is drawn by internal/game**, which imports this package. The
// ledger belongs to no scene, which is what makes it chrome — but a control standing in a column
// has to be placed by whoever owns the column, or the two drift apart the first time either moves.
// The arrow already points this way; nothing new is imported to make it work.
//
// **The buttons carry words, not characters** *(2026-09-04, owner's call)*. They were 44px squares
// holding `$`, `T`, `E` and `L`, which is what a column 44 pixels wide can carry, and a symbol
// that has to be learned is worse than a word that does not.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

const (
	// ControlButtonHeight is one rung of either group. The height is the square buttons' 44 — the
	// footprint the game's other secondary controls take — kept although the buttons are no longer
	// square, so the rhythm did not change when the labels did.
	ControlButtonHeight = 44

	// ControlButtonGap is the air between the two panel buttons. **The sort tabs do not use it**:
	// they are one block, and air between them would make them three buttons again.
	ControlButtonGap = 8

	// ControlButtonText is the label size. **18 rather than the default 20**, because `Element` on
	// a tab and `LEDGER` on a narrow button both have to fit without being abbreviated back, which
	// is what spelling them out was for.
	ControlButtonText = 36

	// ControlButtonWidth is how wide HANDS and LEDGER are: **the wider word and a little more,
	// not the column** *(2026-09-04, owner's call)*. `LEDGER` measures about 62 pixels of kubasta
	// at 18, so this is the label with half again around it. A control taking a card's width to
	// carry one word reads as a pane rather than a button, which is what these two open rather
	// than what they are.
	ControlButtonWidth = 200

	// sortTabWidth is the block's width, and it is the enemy card's so the block, the cards it
	// arranges and the corner above it are one measure.
	sortTabGap = 0
)

// ControlColumnLeft is the line the panel buttons stand on: the enemy card's left edge.
func ControlColumnLeft(gs *state.GlobalState) int { return enemyCardRect(gs).Min.X }

// ControlColumnWidth is the column's full width, which is the enemy card's. **Only the sort block
// is this wide**; the panel buttons take ControlButtonWidth.
func ControlColumnWidth() int { return cards.EnemyStyle.Width }

// The two panel buttons, counted **up from the bottom of the action-point bar**. They are written
// down here rather than each caller knowing its own index: one is placed by this package and one
// by internal/game, and two owners counting the same column independently is how a button ends up
// drawn over another.
const (
	SlotLedger = iota
	SlotHands

	// ControlColumnSlots is how many there are, and what TestTheSortColumnStartsOnTheHandsTopEdge
	// measures against the screen.
	ControlColumnSlots
)

// ControlColumnSlot is a panel button's rectangle, counting up from the action-point bar.
//
// **The bottom of LEDGER is the bottom of that bar** *(2026-09-04, owner's call)*, which is what
// ties the pair to the hand's own furniture rather than leaving them floating in the column. They
// stack upward from there, so adding a third would grow the group toward the cards rather than
// off the bottom of the screen.
func ControlColumnSlot(gs *state.GlobalState, i int) image.Rectangle {
	left := ControlColumnLeft(gs)
	bottom := apBarBottom(gs) - i*(ControlButtonHeight+ControlButtonGap)
	return image.Rect(left, bottom-ControlButtonHeight, left+ControlButtonWidth, bottom)
}

// ControlColumnSlotCentre is that slot's centre, which is what models.Button stores.
func ControlColumnSlotCentre(gs *state.GlobalState, i int) image.Point {
	r := ControlColumnSlot(gs, i)
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
}

// sortTabRect is the i'th tab of the sort block: full column width, no gap above or below it, and
// the block's top edge on the hand's top edge.
//
// **It hangs off the cards rather than off the column** *(2026-09-04, owner's call)*. The block
// arranges the hand, so it is tied to the row it arranges: its left edge is where the widest hand
// stops, which is sortColumnGap left of the column's own line. That leaves the block ending short
// of the enemy card's right edge by the same amount — accepted, because what the block should look
// attached to is the cards under it and not the card above it.
//
// **The left edge is the card band's, which does not narrow.** handBand does, as the hand is
// spent; a block tied to that would slide sideways mid-round.
func sortTabRect(gs *state.GlobalState, i int) image.Rectangle {
	left := handBandLeft(gs) + cardBandWidth(gs)
	top := handTop(gs) + i*(ControlButtonHeight+sortTabGap)
	return image.Rect(left, top, left+ControlColumnWidth(), top+ControlButtonHeight)
}
