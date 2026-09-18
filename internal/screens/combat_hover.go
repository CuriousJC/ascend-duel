package screens

// **What the cursor is resting on, on the combat screen.**
//
// One pass a tick, run after everything that could have moved a card. It asks each row in the order
// the screen draws them — topmost first — and stops at the first hit, because the panel explains one
// thing and the thing it explains is the one the player can see.
//
// **Hover explains; long press is the same reveal on a touchscreen** *(owner's call, 2026-08-21)*.
// MECHANICS.md recorded the opposite split when hover was rejected — hover un-occludes, long press
// explains — and this is the reversal, not an oversight. Nothing here knows which input asked: the
// day a press can ask, it calls `Point` with the same lines.
//
// **Every rectangle here is the one the card is drawn in**, taken from the same slot function the
// drawing uses. A tooltip hit-tested against geometry of its own would eventually describe the card
// next to the one under the pointer, which is worse than no tooltip at all.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// hover points the tooltip at whatever is under the cursor, or at nothing.
//
// **A modal wins outright.** The deck overlay covers the screen, so the hand and the relics beneath
// it are not being looked at even though their rectangles are still where they were; the fight log
// covers the same ground and explains itself in words already.
func (s *CombatScene) hover(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// **A gated step takes the tooltips with the clicks.** A tooltip is an invitation to look at
	// something, and the whole of a gating step is that there is one thing to look at.
	if !gs.CursorAllowed() {
		return
	}

	// **The hands panel explains itself in words**, so there is nothing under it to point at —
	// and a hand card's tooltip drawn through a dialog is the failure this branch exists to stop.
	if s.hands.IsOpen() {
		return
	}
	if s.showDeck {
		ui.HoverDeckPanel(gs, at, s.DeckView, s.fightContents(), &s.tip)
		return
	}
	if s.hoverHand(gs, at) || s.hoverRelics(gs, at) {
		return
	}
	if s.hoverRoundTimer(gs, at) {
		return
	}
	s.hoverFighters(gs, at)
}

// hoverRoundTimer explains the clock. **It is asked before the fighter cards** for the reason the
// order of this whole file is what it is — the bar is drawn over the ground under the duelist card
// and the two rectangles do not overlap today, so the order costs nothing and stops mattering the
// day the column is re-laid.
func (s *CombatScene) hoverRoundTimer(gs *state.GlobalState, at image.Point) bool {
	limit := s.roundTimerLimit()
	if limit < 1 {
		return false
	}
	r := s.roundTimerRect(gs)
	if !at.In(r) {
		return false
	}
	title, lines := ui.RoundTimerTip(s.roundTimerSpent(limit), limit)
	s.tip.Point(r, ui.TipLine(title), ui.TipLines(lines))
	return true
}

// hoverHand explains the card the cursor is resting on. **Through ui.HoveredSeat**, which is the one
// walk every row in the game hit-tests with — the row overlaps and the card drawn last is the one on
// top, so the press, the raise and the tooltip name one card by construction.
func (s *CombatScene) hoverHand(gs *state.GlobalState, at image.Point) bool {
	if s.drag.Dragging() {
		return false // a card in the air is being moved, not read
	}

	// **Only while the queue can still be edited** *(2026-08-21)*. A played card stays in `s.hand`
	// until the round finishes — `spendSelected` runs at the end — while being *drawn* on the
	// table, so hovering its old seat explained a card that had visibly flown away. That is a
	// failure hover has and a click does not: a click on a vacated seat does nothing, where a
	// tooltip cheerfully answers for the card that used to be there.
	if !s.planning() {
		return false
	}

	i := ui.HoveredSeat(at, len(s.hand), func(i int) image.Rectangle { return s.cardSlot(gs, i) })
	if i < 0 {
		return false
	}
	slot := s.cardSlot(gs, i)
	card := s.hand[i].Card
	title, lines := ui.CardTip(card, ui.HeldBy(s.fighter.Duelist, card))
	s.tip.Point(slot, ui.TipLine(title), ui.TipLines(lines))
	return true
}

// hoverRelics explains a worn relic, and says where it sits in the firing order. **The order is the
// information**: relics fire left to right and compound, so which of two doublings applies first is
// a fact about the row rather than about either relic.
func (s *CombatScene) hoverRelics(gs *state.GlobalState, at image.Point) bool {
	// **The consumables pane shares this door**, exactly as it does on the build band: it is the
	// other half of the same row, and a caller that had to remember two calls is a caller that will
	// eventually make one.
	if hoverConsumables(gs, s.consumablePaneRect(gs), at, &s.tip) {
		return true
	}

	worn := wornRelics(gs)
	if len(worn) == 0 {
		return false
	}

	// **The row's own seats, through the row's own adapter** — relicRow is what the drag and the
	// raise are already addressed through, so a seat measured a second time here is a seat that can
	// disagree with the card the player sees lifted.
	row := s.relicRow(gs)
	i := ui.HoveredSeat(at, len(worn), func(i int) image.Rectangle { return row.RowSlot(gs, i) })
	if i < 0 {
		return false
	}
	slot := row.RowSlot(gs, i)
	title, lines := ui.RelicTip(worn[i], i, len(worn))
	s.tip.Point(slot, ui.TipLine(title), ui.TipLines(lines))
	return true
}

// hoverFighters explains either duelist card: their figures, and every status standing on them.
//
// **This is the only place a badge is readable.** The row of pictures under the enemy's health says
// that something is running and nothing anywhere says what — the sentence has been in
// `statuses.json` since the day statuses became data, with nowhere to print it.
func (s *CombatScene) hoverFighters(gs *state.GlobalState, at image.Point) {
	if seat := ui.EnemyCardRect(gs); at.In(seat) {
		title, lines := ui.DuelistTip(s.enemy.Name, s.enemy.Duelist)
		s.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
		return
	}
	if seat := ui.DuelistCardRect(gs); at.In(seat) {
		title, lines := ui.DuelistTip(s.fighter.Name, s.fighter.Duelist)
		s.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
	}
}
