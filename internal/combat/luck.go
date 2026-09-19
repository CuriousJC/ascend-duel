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
// only *where* it is taken; the design argument was made when the rune landed and is unchanged.
//
// **It takes its own source and never the one lightning draws from.** Sharing would make every
// shock in a run a function of how many golden cards were played, and every gamble a function of
// how often the player was shocked — two concerns advancing one cursor, which is the rule the
// randomness skill states and the reason `Sources` is a struct rather than a second parameter.

import "math/rand"

// LuckOutcomes is how many faces a gamble's die has to have before one of them can lose.
//
// **Three: the damage, the life, and nothing.** A record naming fewer is refused at load — see
// resolveRune — because a gamble that always pays is a purchase rather than a gamble, and the
// mistake is one a number in a JSON file could make silently. Silver has only one paying face and
// is held to the same floor, which costs it nothing.
const LuckOutcomes = 3

// LuckDMG, LuckLife and SilverVitae are what a winning roll is worth.
//
// **Go constants rather than record fields** *(owner's call, 2026-09-07)*. A rune record has
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
// **The die is widened rather than shrunk, and that is what lets a relic scale it** *(2026-09-18)*.
// The face count is `odds * pctDie` and each outcome takes a band `scale` wide, so at the identity
// scale of 100 this is exactly the d5 it has always been — 1 in 5 damage, 1 in 5 life, 3 in 5
// nothing — and at 200 each paying band is twice as wide. **Still one sample**, which is the
// determinism half: a relic may change what a roll means and may never change how often the stream
// is drawn from. See DoScaleRolls.
func rollGolden(odds, scale int, rng *rand.Rand) (dmg, life int) {
	if rng == nil || odds < LuckOutcomes {
		return 0, 0
	}

	faces := odds * pctDie
	band := luckBand(scale, faces, 2)

	switch roll := rng.Intn(faces); {
	case roll < band:
		return LuckDMG, 0
	case roll < 2*band:
		return 0, LuckLife
	}
	return 0, 0
}

// LuckOdds is one gamble's chance as a numerator over a denominator, reduced to the terms the
// player reads: `1, 5` bare, and `2, 5` under a relic that has doubled the numerator.
//
// **It is the roll's own arithmetic, exported rather than restated** *(2026-09-18)*. `carddesc`
// prints what this returns, so what a card promises and what the die does are one calculation — a
// tooltip deriving its own figure is how a card comes to lie about its odds, and the clamp in
// luckBand is exactly the kind of detail a second copy would miss.
//
// `bands` is how many paying outcomes share the die: two for gold, one for silver.
func LuckOdds(odds, scale, bands int) (num, den int) {
	if odds < LuckOutcomes {
		return 0, odds
	}
	if scale <= 0 {
		scale = 100
	}

	faces := odds * pctDie
	band := luckBand(scale, faces, bands)

	// Back into the record's own denominator, which is the number the card was authored around —
	// a chance printed over 500 would be true and unreadable.
	num, den = band, pctDie
	for _, d := range []int{2, 5} {
		for num%d == 0 && den%d == 0 {
			num, den = num/d, den/d
		}
	}
	return num, odds * den
}

// pctDie is how many faces one unit of the old die is cut into, which is what makes a percentage
// scale expressible without a second sample. A hundred, so a scale is read directly as its own
// band width.
const pctDie = 100

// luckBand is how wide one paying outcome is, held so that **at least one losing face survives**.
//
// **That is LuckOutcomes' rule generalized.** A record naming fewer than three faces is refused at
// load because a gamble that always pays is a purchase; a relic wide enough to cover the die would
// do the same thing from the other direction, and it would do it at runtime where no loader can
// see it. `bands` is how many paying outcomes share the die — two for gold, one for silver.
func luckBand(scale, faces, bands int) int {
	if scale < 1 {
		scale = 1
	}
	if most := (faces - 1) / bands; scale > most {
		return most
	}
	return scale
}

// rollSilver takes one silver card's gamble and reports the vitae it paid.
//
// **One paying face rather than two**, which is the whole of what makes silver the cheaper metal:
// same die, half the outcomes on it.
func rollSilver(odds, scale int, rng *rand.Rand) int {
	if rng == nil || odds < LuckOutcomes {
		return 0
	}

	faces := odds * pctDie
	if rng.Intn(faces) < luckBand(scale, faces, 1) {
		return SilverVitae
	}
	return 0
}
