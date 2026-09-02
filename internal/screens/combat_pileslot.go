package screens

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// **The slot beside the draw pile, and what the screen's dialogs have in common.**
//
// The fight log used to be here: a square `L` button next to the pile, opening a panel holding
// this fight's rounds. **It became the ledger on 2026-09-02** *(owner's call)* — the run's whole
// account rather than one fight's, scrollable, with the arithmetic under each blow, and reachable
// from every screen rather than from this one. It is therefore chrome now and lives in
// `internal/game`; see `internal/screens/ledger.go` for the panel and `internal/session/ledger.go`
// for what it is drawn from.
//
// What stays here is the *geometry*, because four other controls are placed against it: the
// bucket stands in this slot, and the deck, hands and pouch toggles are all sized from it. The
// square beside the pile is a shape this screen uses, and it outlived the one button that
// happened to be the first thing in it.

// The slot beside the draw pile: a square sharing the pile's bottom edge.
const (
	pileSlotSize = 44

	// The gap between the slot and the pile's own left edge. Wider than the sort column's 8,
	// because these two controls are not a set: the pile and whatever stands beside it are
	// separate things that happen to stand together, and a gap tight enough to read as a stack
	// would say they were one widget.
	pileSlotToDeckGap = 18

	// **One character on a square**, which is what a control this size can carry — the same
	// reason the sort column is single letters. It is the size the toggles draw their labels at.
	pileSlotTextSize = 30
)

// pileSlotRect is the slot: immediately left of the pile, bottom edges level.
//
// **Both edges come off the pile**, never off a percentage. The pile is itself hung off the
// screen's bottom-right corner, so a control placed any other way would drift the first time that
// inset changed — the same staleness ringPaneRect was rewritten to avoid.
//
// It reads the pile's *bounds* rather than its front card, exactly as buttonStripSlots does: the
// backs are drawn up and to the left, so the front card's edge is not the pile's edge.
func pileSlotRect(gs *state.GlobalState) image.Rectangle {
	stack := deckStackBounds(gs)
	right := stack.Min.X - pileSlotToDeckGap
	bottom := stack.Max.Y
	return image.Rect(right-pileSlotSize, bottom-pileSlotSize, right, bottom)
}

// modalUp reports whether any of this screen's dialogs is covering it.
//
// **One predicate rather than the conditions spelled out at every call site**, because every
// control on this screen has to go dead for all of them and the failure is silent: a button left
// live under a dialog is a round edited through a panel the player is only reading.
func (s *CombatScene) modalUp() bool {
	return s.showDeck || s.hands.open
}
