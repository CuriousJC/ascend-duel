package screens

// The shop's half of the shared draw pile: the click that opens the panel. See deckpile.go for the
// pile itself.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// clickedPile is the click that opens and closes the panel. **It runs whether or not the panel
// is up**, which is the combat screen's rule for its own pile: the X is the exit, and this is the
// opener, so a press here while the panel is up must not reach the shelf underneath.
func (s *ShopScene) clickedPile(gs *state.GlobalState, at image.Point) bool {
	if !at.In(deckPileBounds(gs)) {
		return false
	}
	s.deck.Toggle()
	s.armed = ""
	s.tip.Forget()
	return true
}
