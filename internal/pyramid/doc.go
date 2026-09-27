// Package pyramid is the climb: how many rooms a floor holds, how much harder each room is
// than the one below it, and which opponent stands in each one.
//
// **It is tower generation, not a rule about resolving a round**, which is why it is not in
// internal/combat, and it is not a screen's either — anything headless needs the same arithmetic
// and cannot import a screen, which is the other half of why this is a package.
//
// # A floor is a motif and an element
//
// The roster is data/motifs/<motif>/motif.json, one directory per motif, each holding the
// creatures that can stand in a floor's outer chamber, its inner chamber and its portal room. A floor takes one whole motif
// and one element, and its three rooms are three records of that motif dealt as that element — so
// a fire goblin floor is three goblins in fire, and what the player walked into is something they
// can plan against.
//
// **A motif is never offered twice in one run.** Floor one is offered one theme and every floor
// above it two, one per portal, and every offer is struck off before the next floor is rolled —
// whichever the player walks through. data.MustBeClimbable refuses a roster where that could run
// out, at load, and New checks what is left after every draw, so a run never reaches a floor with
// nothing to put in it.
//
// # The curve is indexed by fight, not by floor
//
// Every record writes a base stat line that says what it is worth in the very first room of the
// tower, and ScaleToFight puts it where it actually stands. Stepping per fight rather than per
// floor is what makes a floor's boss harder than its own inner chamber and the next floor's outer
// chamber harder than that boss, with no constraint between two separate numbers to get wrong.
//
// So a creature that only appears high in the tower is **not** authored as a high stat line. It is
// authored as the multiple of its neighbours it is meant to be, and the curve does the rest. That
// is also what keeps the ratio between two motifs fixed however the curve is retuned.
//
// # The offers are the seed's; the pick is the run's
//
// What each floor offers is a function of the run code alone — never of what the player picked on
// an earlier floor, because every offer is spent either way. Which portal the player walked through
// is held by internal/session and saved with the run, so a run code plus its picks is the whole
// path.
//
// **No Ebitengine, ever**, and no randomness of its own: New takes the source it draws from, so
// the caller owns which stream is being advanced. See the randomness skill.
package pyramid
