package screens

// **The shop's half of the tutorial.** See combat_tutorial.go and tutorial.go.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// tutorialFacts is the phase and nothing else, for the reason the post-battle screen's is: there
// is no duel here to report on, and inventing a figure would let a script's mistake pass.
func (s *ShopScene) tutorialFacts(gs *state.GlobalState) tutorial.Facts {
	if gs.Run == nil {
		return tutorial.Facts{}
	}
	return tutorial.Facts{
		Phase: gs.Run.Phase().String(),

		// **Both read off the run, which is the only thing that knows.** What is worn and what has
		// been drunk outlive this visit, and a count kept by the scene would be a second opinion
		// about a purchase — wrong the moment a relic is sold, or the screen re-entered.
		RelicsWorn: len(gs.Run.Worn()),
		DMGBonus:   gs.Run.DMGBonus(),
	}
}

// tutorialRect answers for the shelf.
//
// **The whole row rather than one card**, exactly as the reward screen's worms are: which relic to
// buy is the player's decision, and a spotlight on one of three would be making it for them.
//
// **The worn row now has one** *(2026-09-06)*. It deliberately did not, because a run reaches its
// first shop wearing nothing and a step pointing at that row would have pointed at an empty band.
// The lesson now buys two relics before it says a word about them, so the row has something in it by
// the time it is pointed at — and it still reports false when empty, which is what keeps the old
// argument's teeth.
func (s *ShopScene) tutorialRects(gs *state.GlobalState, a tutorial.Anchor) ([]image.Rectangle, bool) {
	// The same card the reward screen answers for, through the same function. See that one.
	if a == tutorial.AnchorBuildCard {
		return one(buildCardRect(gs)), true
	}
	// The way out. **Lit even though nothing is locked**, because the step that waits for the run
	// to reach the next fight is waiting on this press and nothing else — see AnchorShopLeave.
	if a == tutorial.AnchorShopLeave {
		return one(buttonRect(s.leaveButton)), true
	}
	// The relics the run actually has on. **False for an empty row**, so a step that reached it too
	// early drops its gate and is visible as a mistake rather than lighting an empty band.
	if a == tutorial.AnchorShopWorn {
		worn := gs.Run.Worn()
		return rowUnion(len(worn), func(i int) image.Rectangle {
			return s.wornSlot(gs, i, len(worn))
		})
	}

	// **The Draught's seat alone, not the potions pane.** See AnchorShopDMGPotion: the lit square
	// is the only legal click, and a pane-wide anchor would let the last vitae go on a Salve while
	// the step waited for a Draught that could no longer be bought.
	if a == tutorial.AnchorShopDMGPotion {
		for i, p := range shopPotions() {
			if p.Effect == session.PotionDMG {
				return one(potionSeat(gs, i)), true
			}
		}
		return nil, false
	}

	if a != tutorial.AnchorShopShelf {
		return nil, false
	}
	// **The relics only, not the two sealed goods beside them** *(2026-08-27)*. The shelf became a
	// five-seat row that day and the anchor deliberately did not grow with it: the lock leaves only
	// what is lit clickable, so a lesson about buying a relic cannot be answered by opening a bag of
	// rocks — and a first shop is not where a player should meet the stones.
	return rowUnion(shelfSize, func(i int) image.Rectangle { return s.shelfSlot(gs, i) })
}

// rowUnion is the rectangle covering n slots of a row, and false for a row with nothing in it.
//
// **One rectangle is right here and would be wrong for the matching cards.** A shelf, a worn row
// and a pane of potions are *contiguous* rows where every seat is part of what the step names — so
// the box round them contains nothing the player should not reach. See combat_tutorial.go's
// matching cards, which is the case where that stops being true.
func rowUnion(n int, slot func(i int) image.Rectangle) ([]image.Rectangle, bool) {
	if n <= 0 {
		return nil, false
	}
	r := slot(0)
	for i := 1; i < n; i++ {
		r = r.Union(slot(i))
	}
	return one(r), true
}

// tutorialCovered is whether the deck panel or the hands ladder is over the screen. See the
// combat screen's, and tutorial.go for what it is for.
func (s *ShopScene) tutorialCovered(*state.GlobalState) bool {
	return s.deck.open || s.hands.open
}
