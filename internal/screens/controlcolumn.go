package screens

// The controls of the bottom third that are not cards: the sort tabs and the two panel buttons.
// **The bottom third is one layout on every screen that draws it** — the fight, the reward screen,
// the shop, the sealed good and the portal — so these are placed here once rather than by each
// screen.
//
//   - **The sort tabs** are a block of three, no air between them, standing at the right end of the
//     card band and centered on the hand's height. They arrange the hand and belong to it — see
//     sortTabRect.
//   - **The two panel buttons**, HANDS and the frame's LEDGER, stand in a row under the draw pile.
//     They open a page over the game and belong to the screen, not to the row — see underPileSlot.
//
// **It is exported because the frame is drawn by internal/game**, which imports this package. The
// ledger belongs to no scene, which is what makes it chrome — but a control standing beside the
// pile has to be placed by whoever owns the pile, or the two drift apart the first time either
// moves. The arrow already points this way; nothing new is imported to make it work.
//
// **The two panel buttons are pictures** — the fanned hand and the open scroll, each a
// ui.PanelButtonSize square, and so are the sort tabs — an arrow, an axe, a flame.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

const (
	// ControlButtonHeight is one rung of either group. The height is the square buttons' 44 — the
	// footprint the game's other secondary controls take — kept although the buttons are no longer
	// square, so the rhythm did not change when the labels did.

	// ControlButtonGap is the air between the two panel buttons. **The sort tabs do not use it**:
	// they are one block, and air between them would make them three buttons again.
	ControlButtonGap = 8

	// ControlButtonText is the label size. **18 rather than the default 20**, because `Element` on
	// a tab and `LEDGER` on a narrow button both have to fit without being abbreviated back, which
	// is what spelling them out was for.

	// The panel buttons are ui.PanelButtonSize squares, **narrower than the column**: a control
	// taking a card's width reads as a pane rather than a button, which is what these two open
	// rather than what they are. TestThePanelButtonsStackUpFromTheAPBar holds it.

	// sortTabGap is the air between two sort tabs: none, so the three read as one control.
	sortTabGap = 0
)

// The two panel buttons, counted **from the right of the row under the pile**. They are written
// down here rather than each caller knowing its own index: one is placed by this package and one
// by internal/game, and two owners counting the same row independently is how a button ends up
// drawn over another.
const (
	SlotLedger = iota
	SlotHands

	// ControlColumnSlots is how many there are, and what TestTheSortColumnStartsOnTheHandsTopEdge
	// measures against the screen.
	ControlColumnSlots
)

// ControlColumnSlot is a panel button's rectangle: under the draw pile, side by side — see
// underPileSlot — on every screen that has them, because the pile stands in the same place on all
// of them.
func ControlColumnSlot(gs *state.GlobalState, i int) image.Rectangle {
	return underPileSlot(gs, i)
}

// underPileSlot is a panel button's square: in a row under the draw pile,
// HANDS on the left and LEDGER on the right, the pile's width shared out so the air before, between
// and after the two is equal, and the row centered between the pile's bottom and its count.
func underPileSlot(gs *state.GlobalState, i int) image.Rectangle {
	pile := deckStackRect(gs)
	size := ui.PanelButtonSize
	gap := (pile.Dx() - ControlColumnSlots*size) / (ControlColumnSlots + 1)
	row := ControlColumnSlots*size + (ControlColumnSlots-1)*gap
	left := pile.Min.X + (pile.Dx()-row)/2 + (ControlColumnSlots-1-i)*(size+gap)
	top := pile.Max.Y + (deckCountRect(gs).Min.Y-pile.Max.Y-size)/2
	return image.Rect(left, top, left+size, top+size)
}

// ControlColumnSlotCenter is that slot's center, which is what models.Button stores.
func ControlColumnSlotCenter(gs *state.GlobalState, i int) image.Point {
	r := ControlColumnSlot(gs, i)
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
}

// The bottom line: the strip of square buttons that ends at the screen's bottom-right corner.
//
// **It is the column's other half** *(owner's call, 2026-09-06)*, and it lives here for the reason
// the column does: two owners measuring the same corner independently is how a button ends up drawn
// on top of another. The frame's settings cog is slot 0 and everything else walks leftward from it,
// so a screen adding a square control never has to know where the cog is — only that it is first.
//
// **The shop was the failure this fixes.** Its deck, stones and hands buttons were each placed by a
// rule of their own, measured from the screen's right edge rather than from the frame's; the hands
// button and the cog ended up in the same pixels, and the ledger stood in a column above a row that
// knew nothing about it.
const ()

// The bottom line's occupants, counted leftward from the corner. **Written down here rather than
// each caller knowing its own index**, which is the rule the column above is already under: the cog
// is placed by internal/game and the other two by the shop, and two owners counting one strip
// independently is how a button ends up drawn over another.
const ()

// sortTabRect is the i'th tab of the sort block: a SortTabSize square, no gap above or below it,
// and the block centered on the dealt cards' height.
//
// **It hangs off the cards rather than off the column** *(2026-09-04, owner's call)*. The block
// arranges the hand, so it is tied to the row it arranges: its left edge is where the widest hand
// stops — ui.SortColumnLeft, which ends the block SortColumnGap short of the enemy card's right
// edge, because what the block should look attached to is the cards beside it.
//
// **The left edge is the card band's, which does not narrow.** handBand does, as the hand is
// spent; a block tied to that would slide sideways mid-round.
// sortBlockDrop is how far below a row's top edge the block starts, so it sits centered on the
// cards beside it. Shared by every row the tabs stand beside.
func sortBlockDrop() int {
	n := len(ui.SortButtonSpecs)
	return (cardHeight - n*ui.SortTabSize - (n-1)*sortTabGap) / 2
}

func sortTabRect(gs *state.GlobalState, i int) image.Rectangle {
	left := handBandLeft(gs) + cardBandWidth(gs)
	top := handTop(gs) + sortBlockDrop() + i*(ui.SortTabSize+sortTabGap)
	return image.Rect(left, top, left+ui.SortTabSize, top+ui.SortTabSize)
}
