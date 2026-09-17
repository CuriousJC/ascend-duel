package ui

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The frame's geometry: the two corner cards, the control column's line, and the strip of square
// controls along the bottom.
//
// **None of it is a fact about a duel**, which is what lets it live here rather than on the combat
// screen — the rects are percentages of a fixed 1920x1080 and the width of a card style, and the
// corner strip is measured from the column's line rather than from the screen's edge. The combat
// screen's own doc comment made that argument about duelistCardRect before this package existed;
// this is the argument taken to its conclusion.
//
// **What stayed on the combat screen is anything measured from the hand.** ControlColumnSlot counts
// up from the action-point bar and sortTabRect hangs off the card band, and both of those are the
// duel's layout rather than the frame's. That is the line: a size or a corner is the frame's, and a
// position derived from the row of cards is the screen's.

// DuelistCardRect is where the player's card sits. **The relic row starts from its right edge**, so
// this is the one place its geometry is written and everything else reads it — see relicPaneRect.
//
// It takes no scene, because the hand row's width is measured against the relic row between the two
// corner cards and nothing about that geometry is a fact about a duel. **A CombatScene method that
// only forwarded to it was deleted on 2026-09-17**: two spellings of one rectangle is two things to
// keep in step for no reading the other did not already give.
func DuelistCardRect(gs *state.GlobalState) image.Rectangle {
	left, top := gs.PctX(DuelistCardLeftPct), gs.PctY(TopRowTopPct)
	return image.Rect(left, top,
		left+cards.DuelistStyle.Width, top+cards.DuelistStyle.Height)
}

// EnemyCardRect is the scene-free form, for the same reason duelistCardRect has one.
func EnemyCardRect(gs *state.GlobalState) image.Rectangle {
	right, top := gs.PctX(enemyCardRightPct), gs.PctY(TopRowTopPct)
	return image.Rect(right-cards.EnemyStyle.Width, top,
		right, top+cards.EnemyStyle.Height)
}

// ControlColumnLeft is the line the panel buttons stand on: the enemy card's left edge.
func ControlColumnLeft(gs *state.GlobalState) int { return EnemyCardRect(gs).Min.X }

// ControlColumnWidth is the column's full width, which is the enemy card's. **Only the sort block
// is this wide**; the panel buttons take ControlButtonWidth.
func ControlColumnWidth() int { return cards.EnemyStyle.Width }

// ChromeCornerSlot is the n'th square along the bottom line, counting **leftward from the corner**.
// Slot 0 is the settings cog's seat; a scene's own square controls take 1, 2 and so on.
//
// **Its right edge is the control column's**, not the screen's, so the column above and the strip
// below read as one corner rather than two things near each other.
func ChromeCornerSlot(gs *state.GlobalState, n int) image.Rectangle {
	right := ControlColumnLeft(gs) + ControlColumnWidth() -
		n*(ChromeButtonSize+ChromeButtonGap)
	top := gs.ScreenHeight - ChromeButtonInset - ChromeButtonSize
	return image.Rect(right-ChromeButtonSize, top, right, top+ChromeButtonSize)
}

// ChromeCornerCenter is that slot's center, which is what models.Button stores.
func ChromeCornerCenter(gs *state.GlobalState, n int) image.Point {
	r := ChromeCornerSlot(gs, n)
	return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
}

// The two corner cards' placement, as percentages of the fixed 1920x1080.
const (
	DuelistCardLeftPct = 1
	enemyCardRightPct  = 99

	// **Both cards share this top**, and the relic row between them aligns to it as well. One
	// percentage rather than three, because what is wanted is that the whole band starts on one
	// line, not that each thing happens to be near the top.
	TopRowTopPct = 2
)

// The square controls along the bottom line, and the column above them.
const (
	// ChromeButtonSize is the square every corner control takes, and ChromeButtonInset is the air
	// under it. ChromeButtonGap is the air between two of them.
	ChromeButtonSize  = 44
	ChromeButtonInset = 10
	ChromeButtonGap   = 10

	// ControlButtonHeight, ControlButtonWidth and ControlButtonText are one control's measurements
	// in the column. **Sizes are the frame's and placement is the screen's** — ControlColumnSlot,
	// which counts up from the action-point bar, stayed on the combat screen for exactly that
	// reason.
	ControlButtonHeight = 44
	ControlButtonWidth  = 120
	ControlButtonText   = 36
)

// The bottom line's occupants, counted leftward from the corner. **Written down here rather than
// each caller knowing its own index**: the cog is placed by internal/game and the other two by the
// shop, and two owners counting one strip independently is how a button ends up drawn over another.
const (
	// ChromeSlotSettings is the frame's cog, in the corner itself.
	ChromeSlotSettings = iota

	// ChromeSlotStones is the shop's pouch button, next to it.
	ChromeSlotStones

	// ChromeSlotAnimations is the animation gallery's door, and it is **the last slot on purpose**:
	// it is drawn only while state.DebugAnimations is on, so with the flag off the strip ends where
	// it always did and nothing moves. A debug square taking a seat between two real controls would
	// shift them the moment the flag was turned on.
	ChromeSlotAnimations
)
