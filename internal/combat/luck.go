package combat

// The gamble: what a golden or a silver card rolls when it is played.
//
// **This moved out of `internal/session` on 2026-09-09** *(owner's call)*. It used to be a
// consumable that rolled once, between turns, and touched no card. It is now an upgrade a card
// permanently carries, so the roll happens where the card is played — which means here, in the
// rules, off an injected source like every other roll in this package.
//
// **It is the second roll `internal/combat` has**, and the argument for it is not lightning's. See
// MECHANICS.md: a mechanic whose entire subject is luck is the one case where certainty deletes the
// thing rather than tightening it, and a gamble that always pays is a purchase. What is new here is
// only *where* it is taken; the design argument was made when the parasite landed and is unchanged.
//
// **It takes its own source and never the one lightning draws from.** Sharing would make every
// shock in a run a function of how many golden cards were played, and every gamble a function of
// how often the player was shocked — two concerns advancing one cursor, which is the rule the
// randomness skill states and the reason `Sources` is a struct rather than a second parameter.

import "math/rand"

// LuckOutcomes is how many faces a gamble's die has to have before one of them can lose.
//
// **Three: the damage, the life, and nothing.** A record naming fewer is refused at load — see
// resolveParasite — because a gamble that always pays is a purchase rather than a gamble, and the
// mistake is one a number in a JSON file could make silently. Silver has only one paying face and
// is held to the same floor, which costs it nothing.
const LuckOutcomes = 3

// LuckDMG, LuckLife and SilverVitae are what a winning roll is worth.
//
// **Go constants rather than record fields** *(owner's call, 2026-09-07)*. A parasite record has
// one Value and gold needs two figures, and the odds are the interesting dial — a second tier of
// luck is written by moving the denominator, not by paying more per hit. The day somebody wants a
// greater gold that grants three DMG is the day these become fields; until then two numbers in the
// file would be two numbers nobody tunes.
const (
	LuckDMG     = 1
	LuckLife    = 5
	SilverVitae = 10
)

// Sources is every stream a resolved round may draw from.
//
// **A struct rather than a second parameter**, because the list has now grown twice and a seventh
// positional argument is how a caller ends up handing the shuffle to the shock roll. Each field is
// its own concern's stream and they are never interchanged — see the randomness skill, which is
// where the rule lives.
//
// **The zero value rolls nothing**, which is what every test and every headless caller passes and
// what `TestRoundIsDeterministic` pins: a nil source means the rules are integer arithmetic.
type Sources struct {
	// Roll is the shock roll — whether an attack under a shock lands at all. See attackMisses.
	Roll *rand.Rand

	// Luck is the gamble a golden or a silver card takes when it is played. See rollGolden.
	Luck *rand.Rand
}

// rollGolden takes one golden card's gamble and reports what it granted, in DMG and in life.
//
// **One roll with three outcomes, not two rolls** *(owner's call, 2026-09-07)*. A d5 where 1 is the
// damage, 2 is the life and 3-5 is nothing — so the two rewards are mutually exclusive on any one
// play and a card cannot pay twice. Two independent rolls would have made a double payout possible
// at 4%, which is a headline outcome rare enough that most runs would never see it and the ones
// that did would price the card off it.
//
// A nil source rolls nothing, which is the whole of the determinism contract for this file.
func rollGolden(odds int, rng *rand.Rand) (dmg, life int) {
	if rng == nil || odds < LuckOutcomes {
		return 0, 0
	}
	switch rng.Intn(odds) {
	case 0:
		return LuckDMG, 0
	case 1:
		return 0, LuckLife
	}
	return 0, 0
}

// rollSilver takes one silver card's gamble and reports the vitae it paid.
//
// **One paying face rather than two**, which is the whole of what makes silver the cheaper metal:
// same die, half the outcomes on it.
func rollSilver(odds int, rng *rand.Rand) int {
	if rng == nil || odds < LuckOutcomes {
		return 0
	}
	if rng.Intn(odds) == 0 {
		return SilverVitae
	}
	return 0
}
