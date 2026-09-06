package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The bubble may overlap an anchor it cannot avoid; it may not cover the thing being chosen.**
//
// The worm step is the case that forced the distinction *(2026-09-06)*. Its anchor covers the two
// prizes *and* the row of cards they are aimed at, because taking a worm needs a card selected
// first — which is most of the screen, so every candidate seat overlaps it and `place` falls back.
// The fallback used to be "the last seat in the list", which is dead centre, and dead centre is on
// top of the two worms the step is telling the player to choose between: a lesson pointing at
// something it is standing in front of.
//
// So the fallback picks the seat covering least of the anchor, and this holds the outcome that
// actually matters — the worms stay visible. It deliberately does **not** assert that the bubble
// misses the whole anchor, because against an anchor this size that is not achievable and a test
// demanding it would be a test demanding no bubble at all.
func TestTheBubbleDoesNotCoverTheWormsItIsPointingAt(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	gs.Run = session.New(nil)
	gs.Run.Teach(tutorial.Load())

	// The screen as the step actually meets it: two prizes on the table and a deck to aim them at.
	// An empty offer row shrinks the anchor enough that the fallback is never reached, which is
	// exactly the shape that would let this pass while the bug was in.
	reward := &PostBattleScene{prizes: make([]prize, 2), offer: make([]int, 8), selected: -1}

	var overlay tutorialOverlay
	bubble := overlay.place(gs, reward, tutorial.Step{
		Anchor: tutorial.AnchorRewardWorms,
		Until:  tutorial.CondPhaseShop,
	})

	for i := range reward.prizes {
		if hidden := bubble.Intersect(reward.wormSlot(gs, i)); !hidden.Empty() {
			t.Errorf("the bubble at %v covers %v of worm %d, which the step is asking the player "+
				"to choose between", bubble, hidden, i)
		}
	}
}
