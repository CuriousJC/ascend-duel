package session

// The two parasites that do not act on a card: the gamble and the echo.
//
// They are here rather than in parasite.go because neither is an alteration to the deck, which is
// what that file is about. A luck parasite moves the two run-level figures a potion moves; a
// chimera moves nothing at all and is resolved into something else before anything reads it.

import "math/rand"

// LuckOutcomes is how many faces a luck parasite's die has to have before one of them can lose.
//
// **Three: the damage, the life, and nothing.** A record naming fewer is refused at init — see
// resolveParasite — because a gamble that always pays is a purchase rather than a gamble, and the
// mistake is one a number in a JSON file could make silently.
const LuckOutcomes = 3

// LuckDMG and LuckLife are what a winning roll is worth.
//
// **Go constants rather than record fields** *(owner's call, 2026-09-07)*. A parasite record has
// one Value and this needs two figures, and the odds are the interesting dial — a second tier of
// luck is written by moving the denominator, not by paying more per hit. The day somebody wants a
// greater luck that grants three DMG is the day these become fields; until then two numbers in the
// file would be two numbers nobody tunes.
const (
	LuckDMG  = 1
	LuckLife = 5
)

// LuckResult is what one roll did, for the screen to show.
//
// **A pair of figures rather than an enum**, so a dud is the zero value and the caller asks "did
// anything happen" by asking whether either is set — the same shape `Session.Granted` takes.
type LuckResult struct {
	DMG  int
	Life int
}

// Any reports whether the roll paid anything at all.
func (r LuckResult) Any() bool { return r.DMG != 0 || r.Life != 0 }

// Lucked is what the last luck parasite rolled, for the screen to draw.
//
// **Not snapshotted**, for the reason `duplicated` and `granted` are not: it is a handover between
// an apply and the frame that draws it, and a resumed run has nothing to announce.
func (s *Session) Lucked() LuckResult { return s.lucked }

// LuckRolls is how many times this run has gambled.
//
// **It is saved**, and it is what separates one roll from the next inside a fight — see
// `seeds.LuckRoll`. A dedicated counter rather than the bonuses themselves, because three rolls in
// five pay nothing and two consecutive duds would otherwise be seeded identically for ever.
func (s *Session) LuckRolls() int { return s.luckRolls }

// rollLuck spends one luck parasite and reports what it granted.
//
// **The counter steps whether or not the roll paid.** It is the stream's cursor rather than a tally
// of winnings, so a dud that did not advance it would hand the next spending the same face.
func (s *Session) rollLuck(odds int, rng *rand.Rand) LuckResult {
	s.luckRolls++

	out := LuckResult{}
	switch rng.Intn(odds) {
	case 0:
		out.DMG = LuckDMG
		s.dmgBonus += LuckDMG
	case 1:
		out.Life = LuckLife
		s.lifeBonus += LuckLife
	}
	s.lucked = out
	return out
}

// Echoes resolves a parasite into the one that will actually fire.
//
// **Every reader goes through it**, so a chimera is never asked what it costs or how many cards it
// names — it has no answer to either. For everything else it is the identity, which is what lets
// the call sites stay unconditional.
//
// It reports false when a chimera has nothing to copy: a run where no parasite has ever been spent.
// That is a refusal rather than a fallback, because a chimera that landed and did nothing is a
// consumable the player paid for and did not get.
func (s *Session) Echoes(p Parasite) (Parasite, bool) {
	if p.Target != ParasiteChimera {
		return p, true
	}
	if s.lastParasite == "" {
		return Parasite{}, false
	}
	echoed, ok := ParasiteByKey(s.lastParasite)
	if !ok || echoed.Target == ParasiteChimera {
		// A key the catalogue no longer holds, or a chimera that somehow recorded itself. Neither
		// can happen today — `rememberParasite` writes a resolved record and `Resume` refuses an
		// unknown one — and both would be a chimera firing nothing if they did.
		return Parasite{}, false
	}
	return echoed, true
}

// EchoedName is what a chimera would fire, for a tooltip or a card face to say. Empty when there is
// nothing to copy.
func (s *Session) EchoedName(p Parasite) string {
	echoed, ok := s.Echoes(p)
	if !ok || echoed.Record == p.Record {
		return ""
	}
	return echoed.Name
}

// rememberParasite records what was just spent, so a chimera has something to copy.
//
// **The resolved record, never the chimera.** That is what makes two chimeras in a row both fire
// the thing behind them rather than the second one copying the first into nothing.
func (s *Session) rememberParasite(resolved Parasite) {
	if resolved.Target == ParasiteChimera {
		return
	}
	s.lastParasite = resolved.Record
}

// LastParasite is the record a chimera would copy, by key. Empty on a run that has spent none.
func (s *Session) LastParasite() string { return s.lastParasite }
