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

// tutorialRect answers for the one anchor this screen draws: the row of offered essences.
//
// **It is the row rather than one card**, because the lesson is that an essence is the prize and
// either of them is a legitimate answer. Spotlighting one would be telling the player which to
// take, which is the opposite of what the screen is asking them.
func (s *PostBattleScene) tutorialRects(gs *state.GlobalState, a tutorial.Anchor) ([]image.Rectangle, bool) {
	// The duelist card in the build band, which is where the purse is written. **Deferred to
	// `buildCardRect` rather than measured here**, which is what lets the shop answer for the same
	// anchor without the two being able to disagree about where the card is.
	if a == tutorial.AnchorBuildCard {
		return one(buildCardRect(gs)), true
	}

	// The payout's account: its first four lines — interest, health kept, the enemy's vitae and the
	// total — across the payout column, measured off the same layout drawProse types them on.
	if a == tutorial.AnchorPayout {
		n := len(s.prose.lines)
		if n < payoutAccountLines {
			return nil, false
		}
		top := proseTop(gs, n)
		return one(image.Rect(tutorialMargin, top-6,
			gs.PctX(payoutColumnPct)-tutorialMargin,
			top+(payoutAccountLines-1)*proseLineGap+proseLineHeight+6)), true
	}

	if a != tutorial.AnchorRewardEssences || len(s.prizes) == 0 {
		return nil, false
	}
	// **The two essences and nothing else** *(owner's call)*. The step waits for the run to reach
	// the shop, which locks nothing, so the anchor is a picture rather than a gate and need not
	// cover the row of cards an essence is aimed at. A frame each, since they are two things.
	lit := make([]image.Rectangle, 0, len(s.prizes))
	for i := range s.prizes {
		lit = append(lit, s.essenceSlot(gs, i))
	}
	return lit, true
}

// tutorialKeepClear is what the bubble stays off beyond the lit essences: **the row of cards an
// essence is aimed at** *(owner's call)*, since taking one means choosing from that row, so the
// bubble goes above the essences rather than over the hand.
func (s *PostBattleScene) tutorialKeepClear(gs *state.GlobalState, a tutorial.Anchor) []image.Rectangle {
	if a != tutorial.AnchorRewardEssences || len(s.offer) == 0 {
		return nil
	}
	r := s.offerSlot(gs, 0)
	for i := range s.offer {
		r = r.Union(s.offerSlot(gs, i))
	}
	return []image.Rectangle{r}
}

// payoutAccountLines is how many of payoutLines are the account of the vitae: the three claims and
// the total, before the line about the essences.
const payoutAccountLines = 4

// tutorialCovered is whether anything is over the screen. **Nothing can be**: this screen carries
// no panel of its own and the chrome stands down on it, so the choice is the only thing up. See the
// combat screen's, and tutorial.go for what it is for.
func (s *PostBattleScene) tutorialCovered(*state.GlobalState) bool { return false }
