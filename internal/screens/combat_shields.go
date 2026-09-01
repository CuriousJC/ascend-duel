package screens

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The shield count on the duelist card, kept in step with playback rather than with the model.
//
// **It is `shownLife`'s pattern, and the file it replaced held the same pattern for banked action
// points** *(2026-08-31)*. `combat.Duelist.Shields` is not written until the round's end state is
// adopted, so a pip row reading the model would fill up a whole opposing turn after the card that
// filled it and empty a whole turn after the attack that ate it. The engine's arithmetic is
// untouched; what this adds is the same figure arriving on the beat it is drawn.
//
// **It cannot change an outcome.** The round was decided before a frame of it was drawn, and this
// is a view over what the log already says.
//
// **There is no flight.** A banked point travelled from the card that banked it to the AP line it
// raised, because a number changing somewhere else needs to be watched crossing the gap. A shield
// is a pip appearing in a row on the same card the player is already looking at, so the arrival is
// the drawing.

// noteShields records what an announced shield event leaves standing.
//
// **The three kinds carry the count in different fields**, which is not an inconsistency: a raise
// says how many it added in `Amount` and what is standing in `Life`, the same split `KindDamage`
// makes between the blow and the life left; a block and an expiry have no "added" figure, so
// `Amount` is the count itself.
func (s *CombatScene) noteShields(e combat.Event) {
	side, count := e.Side, 0
	switch e.Kind {
	case combat.KindRaised:
		count = e.Life
	case combat.KindBlocked, combat.KindExpired:
		side, count = e.Target, e.Amount
	default:
		return
	}
	if side < 0 || int(side) >= len(s.theatre.shieldsShown) {
		return
	}
	s.theatre.shieldsShown[side] = count
	s.theatre.shieldsSeen[side] = true
}

// shownShields is how many shields to draw on a side's card: what playback has reached if
// anything this round has said so, and the adopted model otherwise.
//
// **The fallback is what makes the planning phase right.** A shield raised at the end of the last
// round is standing while the player builds this one, and no event has fired yet — so the model is
// the only thing that knows, and it is correct until the first announcement overtakes it.
func (s *CombatScene) shownShields(side combat.Side, model int) int {
	if side < 0 || int(side) >= len(s.theatre.shieldsShown) {
		return model
	}
	if !s.theatre.shieldsSeen[side] {
		return model
	}
	return s.theatre.shieldsShown[side]
}
