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
)

// hover points the tooltip at whatever is under the cursor, or at nothing.
//
// **A modal wins outright.** The deck overlay covers the screen, so the hand and the rings beneath
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
	if s.hands.open {
		return
	}
	if s.showDeck {
		hoverDeckPanel(gs, at, s.deckView, s.fightContents(), &s.tip)
		return
	}
	if s.showLog {
		return
	}

	if s.hoverHand(gs, at) || s.hoverRings(gs, at) {
		return
	}
	s.hoverFighters(gs, at)
}

// hoverHand walks the hand from the right, because the row overlaps and the card drawn last is the
// one on top. Same order `beginPress` takes, and for the same reason.
func (s *CombatScene) hoverHand(gs *state.GlobalState, at image.Point) bool {
	if s.drag.dragging() {
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

	for i := len(s.hand) - 1; i >= 0; i-- {
		slot := s.cardSlot(gs, i)
		if !at.In(slot) {
			continue
		}
		card := s.hand[i].actionCard
		title, lines := cardTip(card, heldBy(s.fighter.Duelist, card))
		s.tip.Point(slot, title, lines)
		return true
	}
	return false
}

// hoverRings explains a worn ring, and says where it sits in the firing order. **The order is the
// information**: rings fire left to right and compound, so which of two doublings applies first is
// a fact about the row rather than about either ring.
func (s *CombatScene) hoverRings(gs *state.GlobalState, at image.Point) bool {
	worn := wornRings(gs)
	if len(worn) == 0 {
		return false
	}

	r := s.ringPaneRect(gs)
	for i, record := range worn {
		corner := ringSlotAt(r, i, len(worn))
		slot := image.Rect(corner.X, corner.Y, corner.X+cardWidth, corner.Y+cardHeight)
		if !at.In(slot) {
			continue
		}
		title, lines := ringTip(record, i, len(worn))
		s.tip.Point(slot, title, lines)
		return true
	}
	return false
}

// hoverFighters explains either duelist card: their figures, and every status standing on them.
//
// **This is the only place a badge is readable.** The row of pictures under the enemy's health says
// that something is running and nothing anywhere says what — the sentence has been in
// `statuses.json` since the day statuses became data, with nowhere to print it.
func (s *CombatScene) hoverFighters(gs *state.GlobalState, at image.Point) {
	if seat := s.enemyCardRect(gs); at.In(seat) {
		title, lines := duelistTip(s.enemy.Name, s.enemy.Duelist)
		s.tip.Point(seat, title, lines)
		return
	}
	if seat := s.duelistCardRect(gs); at.In(seat) {
		title, lines := duelistTip(s.fighter.Name, s.fighter.Duelist)
		s.tip.Point(seat, title, lines)
	}
}
