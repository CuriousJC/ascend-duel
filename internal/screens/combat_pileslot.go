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

	// **One character on a square**, which is what a control this size can carry — the same
	// reason the sort column is single letters. It is the size the toggles draw their labels at.
	pileSlotTextSize = 30
)

// pileSlotRect is the slot: the left end of the pile's caption line, with the deck count on the
// right end of the same line.
//
// **Both edges come off the pile**, never off a percentage — a control placed any other way would
// drift the first time the pile moved, which is the staleness ringPaneRect was rewritten to avoid
// and which this file has now survived twice.
//
// **It moved above the pile with it on 2026-09-04** *(owner's call)*. It stood to the pile's left
// while the pile was in the bottom-right corner; in the duelist card's column there is nothing to
// the left, and what is to the right at that height is the action-point bar.
func pileSlotRect(gs *state.GlobalState) image.Rectangle {
	caption := deckCaptionRect(gs)
	return image.Rect(caption.Min.X, caption.Min.Y,
		caption.Min.X+pileSlotSize, caption.Max.Y)
}

// modalUp reports whether any of this screen's dialogs is covering it.
//
// **One predicate rather than the conditions spelled out at every call site**, because every
// control on this screen has to go dead for all of them and the failure is silent: a button left
// live under a dialog is a round edited through a panel the player is only reading.
func (s *CombatScene) modalUp() bool {
	return s.showDeck || s.hands.open
}
