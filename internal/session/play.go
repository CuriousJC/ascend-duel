package session

import "github.com/curiousjc/ascend-duel/internal/combat"

// How often the run has actually built each rung of the hand ladder.
//
// **A tally, and nothing in the rules reads it** *(owner's call, 2026-09-05)*. A run has two
// numbers per rung now: the *stones* on it, which are what a shop sold the player and are what the
// multiplier is read through — see stone.go — and the *plays*, which are what the player has done
// with it. They are deliberately two counters rather than one derived from the other: buying a
// rock and landing a Full House are different achievements, and a single figure could not say
// which of them happened.
//
// **It lives on the run rather than on the combat screen**, for the reason the ledger does: a
// screen's state is thrown away by the next `Init`, and the interesting count is the whole climb's.
//
// **The rules never see it.** `combat.Duelist` carries stone counts because the resolver has to
// read them; it carries no play counts, because a hand pays what it pays however often it has been
// formed. Anything that wanted to change that would be a mechanic, and would go through stones.

// RecordHandPlayed adds one to a rung's tally, by hand key, and reports whether the catalogue holds
// that rung.
//
// **The bool is the validation**, exactly as `combat.HandSlot`'s is: a key nothing recognises is a
// caller reading the wrong field, and a tally that quietly accepted one would grow a column the
// panel could never draw.
func (s *Session) RecordHandPlayed(hand string) bool {
	if !knownHand(hand) {
		return false
	}
	if s.plays == nil {
		s.plays = map[string]int{}
	}
	s.plays[hand]++
	return true
}

// PlaysOf is how many times this run has formed one rung, by hand key.
func (s *Session) PlaysOf(hand string) int { return s.plays[hand] }

// PlayCounts is every rung this run has formed, by hand key, as a copy.
//
// **A copy for the reason `StoneCounts` hands one back**: the map is the run's, and a caller that
// wrote to it would be keeping the run's own account from outside the one method allowed to.
func (s *Session) PlayCounts() map[string]int {
	out := make(map[string]int, len(s.plays))
	for k, n := range s.plays {
		if n != 0 {
			out[k] = n
		}
	}
	return out
}

// knownHand reports whether the catalogue this build loaded holds a rung by that key. It asks
// `combat.HandSlot`, which is the same question a stone is validated with, so a tally and an
// upgrade cannot disagree about which rungs exist.
func knownHand(hand string) bool {
	_, ok := combat.HandSlot(hand)
	return ok
}
