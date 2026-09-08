package screens

// **The post-battle screen's half of the tutorial.** See combat_tutorial.go, which is the same
// pair of methods for the screen before this one, and tutorial.go for what they are for.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// tutorialFacts is what this screen can say. **Only the phase**: there is no duel here, so the
// three fight-shaped fields stay at their zero values rather than being filled with something
// plausible. A condition reading one of them on this screen is a script that has asked the wrong
// screen a question, and it should stall visibly rather than be answered with a guess.
func (s *PostBattleScene) tutorialFacts(gs *state.GlobalState) tutorial.Facts {
	if gs.Run == nil {
		return tutorial.Facts{}
	}
	return tutorial.Facts{Phase: gs.Run.Phase().String()}
}

// tutorialRect answers for the one anchor this screen draws: the row of offered worms.
//
// **It is the row rather than one card**, because the lesson is that a worm is the prize and
// either of them is a legitimate answer. Spotlighting one would be telling the player which to
// take, which is the opposite of what the screen is asking them.
func (s *PostBattleScene) tutorialRects(gs *state.GlobalState, a tutorial.Anchor) ([]image.Rectangle, bool) {
	// The duelist card in the build band, which is where the purse is written. **Deferred to
	// `buildCardRect` rather than measured here**, which is what lets the shop answer for the same
	// anchor without the two being able to disagree about where the card is.
	if a == tutorial.AnchorBuildCard {
		return one(buildCardRect(gs)), true
	}
	if a != tutorial.AnchorRewardWorms || len(s.prizes) == 0 {
		return nil, false
	}
	r := s.wormSlot(gs, 0)
	for i := 1; i < len(s.prizes); i++ {
		r = r.Union(s.wormSlot(gs, i))
	}

	// **The offer row is inside the anchor since the gesture reversed** *(2026-09-06)*. The lit
	// square is also the one legal click, so an anchor covering only the worms would have been a
	// lock-up the moment taking one required a card to be selected first: every worm dim, every
	// card unclickable, and a step waiting for a phase that could never arrive. This is the failure
	// CLAUDE.md warns about — the machinery can refuse an ungated step, and cannot tell whether an
	// anchor shows the player how to satisfy its own condition.
	for i := range s.offer {
		r = r.Union(s.offerSlot(gs, i))
	}
	return one(r), true
}

// tutorialCovered is whether anything is over the screen. **Nothing can be**: this screen carries
// no panel of its own and the chrome stands down on it, so the choice is the only thing up. See the
// combat screen's, and tutorial.go for what it is for.
func (s *PostBattleScene) tutorialCovered(*state.GlobalState) bool { return false }
