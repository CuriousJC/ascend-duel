package session

// **What the journey does to the body between fights.**
//
// A wound is carried out of a room and into the next one, and **nothing between fights takes it
// away** — not a won room, not a portal room. Life comes back only through what the player spends
// on it: a potion, a card that heals, a relic. See MECHANICS.md §Life between fights.
//
// **The wound is stored, not the life left.** `MaxLife` is rebuilt from the record every visit and
// then moved by whatever the run is wearing — a flat +25 here, a percentage there — so a stored
// "you have 40 life" would silently become a different fraction the moment a relic was bought or
// sold. A wound is the same wound whatever ceiling it sits under, and its zero value is the honest
// one: a run that has not been hurt yet is healthy, where a stored life of zero would be a corpse.
//
// **That is also what makes the portal bonus a heal of exactly its own size.** The ceiling rises
// and the wound does not, so the life under it rises by the same 25.
//
// **The boss bonus is a count, not a number.** `bossWins` is how many portal rooms the run has
// cleared and the bonus is derived from it, for the reason the realm is derived from the room
// counter: a stored product is a second copy of the same fact, and it is the copy that goes stale.

// bossLifeBonus is what beating a realm's boss adds to the ceiling, flat, once per portal room
// cleared. Flat rather than a percentage, so a run's body grows by the same step on realm seven as
// on realm one.
const bossLifeBonus = 25

// LifeAtFightStart is the life a duelist walks into a room with, given the ceiling they are walking
// in under. It is the wound subtracted from that ceiling.
//
// **It never returns less than one.** A wound deeper than the ceiling is only reachable by taking
// off the relics that were holding the ceiling up, and a run that cannot start a fight is a worse
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

// BossWins is how many portal protectors the run has beaten.
func (s *Session) BossWins() int { return s.bossWins }

// raiseLifeForBosses raises a ceiling by what the run's beaten bosses are worth. It is applied to
// the record's own figure **before** any relic touches it, so a flat +25 relic stays worth 25 and a
// percentage relic scales the whole grown body — which keeps the relic grammar's own ordering note
// in Equip true rather than adding a third rule to remember.
func (s *Session) raiseLifeForBosses(maxLife int) int {
	maxLife += s.bossWins * bossLifeBonus
	if maxLife < 1 {
		maxLife = 1
	}
	return maxLife
}
