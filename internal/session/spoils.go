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

import "github.com/curiousjc/ascend-duel/internal/pyramid"

// lifeShareDivisor turns life left into vitae: **a tenth of it, rounded down** *(owner's call,
// 2026-08-22)*. Sixty life left is six vitae, and a win on four life is worth nothing from this
// half of the payout.
//
// **It is a share of the life remaining, not of the maximum**, which is what makes it a reward for
// fighting well rather than a rebate. A ring that raises max life pays out more here indirectly,
// and that is intended.
const lifeShareDivisor = 10

// roomVitae is what the room itself pays: **3 for a floor's outer room, 4 for its inner room, 5 for
// the stairway that is its boss** *(owner's call, 2026-08-22)*. Flat for the whole climb — a floor-8
// boss pays the same 5 as floor 1's, because the scaling that makes a later fight worth more is the
// life you keep, not the room you keep it in.
var roomVitae = map[pyramid.Room]int{
	pyramid.RoomOuter:    3,
	pyramid.RoomInner:    4,
	pyramid.RoomStairway: 5,
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

	// FromLife is a tenth of the life the fighter finished on.
	FromLife int

	// FromRoom is what the room pays, rings included.
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

// spoilsFor is what a win in this room, ending on this much life, is worth — **before any of it is
// added**, which is what keeps the interest honest.
func (s *Session) spoilsFor(lifeLeft int) Spoils {
	if lifeLeft < 0 {
		lifeLeft = 0
	}
	return Spoils{
		Propagated: s.propagation(),
		FromLife:   lifeLeft / lifeShareDivisor,
		FromRoom:   s.PrizeVitae(roomVitae[pyramid.RoomOf(s.fight)]),
	}
}
