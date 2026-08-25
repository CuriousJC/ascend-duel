package screens

// **The shop's half of the tutorial.** See combat_tutorial.go and tutorial.go.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// tutorialFacts is the phase and nothing else, for the reason the post-battle screen's is: there
// is no duel here to report on, and inventing a figure would let a script's mistake pass.
func (s *ShopScene) tutorialFacts(gs *state.GlobalState) tutorial.Facts {
	if gs.Run == nil {
		return tutorial.Facts{}
	}
	return tutorial.Facts{Phase: gs.Run.Phase().String()}
}

// tutorialRect answers for the shelf.
//
// **The whole row rather than one card**, exactly as the reward screen's worms are: which ring to
// buy is the player's decision, and a spotlight on one of three would be making it for them.
//
// **There is deliberately no anchor for the worn row.** A run reaches its first shop wearing
// nothing — see session.StartingRings, which ships empty — so the row a step pointed at would be
// empty on the one visit the tutorial is present for, and an anchor that can only ever resolve to
// nothing is vocabulary ahead of a use.
func (s *ShopScene) tutorialRect(gs *state.GlobalState, a tutorial.Anchor) (image.Rectangle, bool) {
	// The same card the reward screen answers for, through the same function. See that one.
	if a == tutorial.AnchorBuildCard {
		return buildCardRect(gs), true
	}
	// The way out. **Lit even though nothing is locked**, because the step that waits for the run
	// to reach the next fight is waiting on this press and nothing else — see AnchorShopLeave.
	if a == tutorial.AnchorShopLeave {
		return buttonRect(s.leaveButton), true
	}
	if a != tutorial.AnchorShopShelf {
		return image.Rectangle{}, false
	}
	return rowUnion(shelfSize, func(i int) image.Rectangle { return s.shelfSlot(gs, i) })
}

// rowUnion is the rectangle covering n slots of a row, and false for a row with nothing in it.
func rowUnion(n int, slot func(i int) image.Rectangle) (image.Rectangle, bool) {
	if n <= 0 {
		return image.Rectangle{}, false
	}
	r := slot(0)
	for i := 1; i < n; i++ {
		r = r.Union(slot(i))
	}
	return r, true
}

// tutorialCovered is whether the deck panel or the hands ladder is over the screen. See the
// combat screen's, and tutorial.go for what it is for.
func (s *ShopScene) tutorialCovered(*state.GlobalState) bool {
	return s.deck.open || s.hands.open
}
