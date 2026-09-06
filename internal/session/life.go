package session

// **What the climb does to the body between fights** *(owner's call, 2026-09-06).*
//
// A duel used to start at full life whatever the last one cost, so damage was a fact about one
// round and never about the run. It is a fact about the *floor* now: a wound is carried out of a
// room and into the next one, and the only thing that takes it away is beating the floor's
// stairway protector. See MECHANICS.md §Life between fights.
//
// **The wound is stored, not the life left.** `MaxLife` is rebuilt from the record every visit and
// then moved by whatever the run is wearing — a flat +25 here, a percentage there — so a stored
// "you have 40 life" would silently become a different fraction the moment a ring was bought or
// sold. A wound is the same wound whatever ceiling it sits under, and its zero value is the honest
// one: a run that has not been hurt yet is healthy, where a stored life of zero would be a corpse.
//
// **The boss bonus is a count, not a number.** `bossWins` is how many stairways the run has
// climbed and the multiplier is derived from it, for the reason the floor is derived from the room
// counter: a stored product is a second copy of the same fact, and it is the copy that goes stale.

// bossLifePct is what beating a floor's boss does to the ceiling, as a percentage of what the
// ceiling already was — so it compounds, floor on floor *(owner's call, 2026-09-06)*. A run that
// climbs seven stairways is carrying about seven and a half times the body it started with, which
// is the curve the ascent's own scaling is meant to be climbed against.
const bossLifePct = 133

// LifeAtFightStart is the life a duelist walks into a room with, given the ceiling they are walking
// in under. It is the wound subtracted from that ceiling.
//
// **It never returns less than one.** A wound deeper than the ceiling is only reachable by taking
// off the rings that were holding the ceiling up, and a run that cannot start a fight is a worse
// failure than a run that starts one on a sliver.
func (s *Session) LifeAtFightStart(maxLife int) int {
	life := maxLife - s.hurt
	if life < 1 {
		life = 1
	}
	return life
}

// Hurt is how much life the run is down, and it is what a screen asks when it wants to draw the
// player as they actually are rather than as the record describes them.
func (s *Session) Hurt() int { return s.hurt }

// BossWins is how many stairway protectors the run has beaten.
func (s *Session) BossWins() int { return s.bossWins }

// scaleLifeForBosses raises a ceiling by what the run's beaten bosses are worth. It is applied to
// the record's own figure **before** any ring touches it, so a flat +25 stays worth 25 and a
// percentage ring scales the whole grown body — which keeps the ring grammar's own ordering note
// in Equip true rather than adding a third rule to remember.
func (s *Session) scaleLifeForBosses(maxLife int) int {
	for i := 0; i < s.bossWins; i++ {
		maxLife = maxLife * bossLifePct / 100
	}
	if maxLife < 1 {
		maxLife = 1
	}
	return maxLife
}
