package session

// **What winning a fight is worth, computed once and paid out a line at a time.**
//
// A win pays three separate things and the post-battle screen narrates each of them as its own
// sentence, with the purse on the duelist card climbing as each one lands. That is the whole
// reason this is a struct rather than three calls to AddVitae inside WonFight: the amounts are
// decided at the moment the fight ends — on the life the fighter walked out with, on the room it
// was fought in, on the purse before any of it — and *arrive* later, on the screen, at reading
// speed.
//
// **Deciding and paying are therefore separate on purpose.** Nothing about the payout may depend
// on when the player gets round to clicking, so the figures are frozen by WonFight and the screen
// only chooses when to hand them over.

import "github.com/curiousjc/ascend-duel/internal/journey"

// lifeVitae is what the life a fighter walked out with pays, by where it sits against the ceiling
// it was measured under: **1 in the bottom third, 2 in the middle third, 3 in the top third, and 5
// for a win on full health**.
//
// **A fighter sitting exactly on a line takes the higher tier**, so 20 of 60 pays 2 and 40 of 60
// pays 3 — which is what the floor of `life*LifeThirds/max` gives, with no third ever rounded.
// Full health is its own tier rather than the top of the third below it, because a win that took
// nothing is the one the payout most wants to name.
//
// **The thirds are of the ceiling at the end of the fight**, relics and all, so a relic that raises
// max life moves the lines with it.
//
// A zero or negative ceiling pays the bottom tier rather than dividing by it.
func lifeVitae(lifeLeft, maxLife int) int {
	switch {
	case maxLife <= 0 || lifeLeft <= 0:
		return lifeTierVitae[0]
	case lifeLeft >= maxLife:
		return lifeFullVitae
	}
	return lifeTierVitae[lifeLeft*LifeThirds/maxLife]
}

// LifeThirds is how many equal tiers the life payout cuts a ceiling into. **It is exported for the
// health bar**, which marks the lines between them: a bar marked in quarters beside a payout cut in
// thirds would be a picture nothing fails on.
const LifeThirds = 3

// lifeTierVitae is the bottom, middle and top third, and lifeFullVitae is a win on full health.
var lifeTierVitae = [LifeThirds]int{1, 2, 3}

const lifeFullVitae = 5

// PropagationPer is the rate the reward screen names in words. It is exported so the sentence
// reads the rule rather than repeating it: a screen printing "for each 10" beside a figure computed
// from a different number would be a lie nothing fails on.
const PropagationPer = propagationPer

// PropagationCeiling is the purse past which holding more earns no more interest — the rate times
// its cap — exported for the same sentence and on the same argument.
const PropagationCeiling = propagationPer * maxPropagation

// roomVitae is what the room itself pays: **3 for a realm's outer room, 4 for its inner room, 5 for
// the portal room that is its boss** *(owner's call, 2026-08-22)*. Flat for the whole journey — a realm-8
// boss pays the same 5 as realm 1's, because the scaling that makes a later fight worth more is the
// life you keep, not the room you keep it in.
var roomVitae = map[journey.Room]int{
	journey.RoomOuter:  3,
	journey.RoomInner:  4,
	journey.RoomPortal: 5,
}

// Spoils is one win's payout, split the way the screen reads it out.
//
// **Zero is the settled state**: each field is cleared as it is claimed, so a claim cannot pay
// twice however many times a screen is re-entered.
type Spoils struct {
	// Propagated is the interest the purse earned, decided **before** either award lands — see
	// propagate, and MECHANICS.md, which states the order: interest on what the run walked out of
	// the fight holding, never on what the fight is about to pay it.
	Propagated int

	// FromLife is what the life the fighter finished on pays — see lifeVitae.
	FromLife int

	// FromRoom is what the room pays, relics included.
	FromRoom int
}

// Total is everything still owed.
func (s Spoils) Total() int { return s.Propagated + s.FromLife + s.FromRoom }

// Spoils is what this win still owes the player.
func (s *Session) Spoils() Spoils { return s.spoils }

// ClaimPropagation, ClaimFromLife and ClaimFromRoom each pay one part into the purse and report
// what they paid. **Claiming twice pays once**, since the field is cleared.
func (s *Session) ClaimPropagation() int { return s.claim(&s.spoils.Propagated) }
func (s *Session) ClaimFromLife() int    { return s.claim(&s.spoils.FromLife) }
func (s *Session) ClaimFromRoom() int    { return s.claim(&s.spoils.FromRoom) }

// ClaimSpoils pays whatever is left, in one go, and reports the total.
//
// **It is the safety net rather than the normal path**: the post-battle screen claims the three
// parts one at a time as it narrates them, and this is what makes leaving the loop any other way —
// a phase with no scene, a test that never draws — still pay a win what it was worth.
func (s *Session) ClaimSpoils() int {
	return s.ClaimPropagation() + s.ClaimFromLife() + s.ClaimFromRoom()
}

func (s *Session) claim(part *int) int {
	n := *part
	*part = 0
	s.AddVitae(n)
	return n
}

// spoilsFor is what a win in this room, ending on this much life of this ceiling, is worth — **before any of it is
// added**, which is what keeps the interest honest.
func (s *Session) spoilsFor(lifeLeft, maxLife int) Spoils {
	if lifeLeft < 0 {
		lifeLeft = 0
	}
	// **Every part is multiplied here, where it is decided**, rather than as it is paid, so the
	// figure the reward screen reads out is the figure that lands in the purse.
	f := s.VitaeFactor()
	return Spoils{
		Propagated: s.propagation() * f,
		FromLife:   lifeVitae(lifeLeft, maxLife) * f,
		FromRoom:   s.PrizeVitae(roomVitae[journey.RoomOf(s.fight)]) * f,
	}
}
