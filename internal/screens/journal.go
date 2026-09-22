package screens

// **The one thing a screen has to work out before it can write a choice down.**
//
// Everything else a journal line carries is already in hand at the click — a relic key, a seat, a
// price. Deck positions are not: the offer rows on the reward screen and inside a sealed good pick
// cards by where they sit in the run's deck, and a position names a different card the moment
// anything is cut or copied. See journal.Record.Targets, which is identities for exactly that
// reason.

import (
	"github.com/curiousjc/ascend-duel/internal/state"
)

// deckCardIDs turns positions in the run's deck into the identities of the cards standing there.
//
// **A position out of range drops out rather than failing the line.** A journal that refused to
// record a choice because one index had moved would be losing the retrace over the thing it exists
// to keep; a line naming two of three cards still says what was reached for.
func deckCardIDs(gs *state.GlobalState, at []int) []int {
	if gs == nil || gs.Run == nil || len(at) == 0 {
		return nil
	}
	deck := gs.Run.Deck()

	out := make([]int, 0, len(at))
	for _, i := range at {
		if i < 0 || i >= len(deck) {
			continue
		}
		out = append(out, deck[i].ID)
	}
	return out
}
