// Package journey is the journey: how many rooms a realm holds, how much harder each room is
// than the one below it, and which opponent stands in each one.
//
// **It is journey generation, not a rule about resolving a round**, which is why it is not in
// internal/combat, and it is not a screen's either — anything headless needs the same arithmetic
// and cannot import a screen, which is the other half of why this is a package.
//
// # A realm is a motif and an element
//
// The roster is data/motifs/<motif>/motif.json, one directory per motif, each holding the
// creatures that can stand in a realm's outer chamber, its inner chamber and its portal room. A realm takes one whole motif
// and one element, and its three rooms are three records of that motif dealt as that element — so
// a fire goblin realm is three goblins in fire, and what the player walked into is something they
// can plan against.
//
// **A motif is never offered twice in one run.** Realm one is offered one theme and every realm
// above it two, one per portal, and every offer is struck off before the next realm is rolled —
// whichever the player walks through. data.MustFillJourney refuses a roster where that could run
// out, at load, and New checks what is left after every draw, so a run never reaches a realm with
// nothing to put in it.
//
// # The curve is indexed by fight, not by realm
//
// Every record writes a base stat line that says what it is worth in the very first room of the
// journey, and ScaleToFight puts it where it actually stands. Stepping per fight rather than per
// realm is what makes a realm's boss harder than its own inner chamber and the next realm's outer
// chamber harder than that boss, with no constraint between two separate numbers to get wrong.
//
// So a creature that only appears high in the journey is **not** authored as a high stat line. It is
// authored as the multiple of its neighbours it is meant to be, and the curve does the rest. That
// is also what keeps the ratio between two motifs fixed however the curve is retuned.
//
// # The offers are the seed's; the pick is the run's
//
// What each realm offers is a function of the run code alone — never of what the player picked on
// an earlier realm, because every offer is spent either way. Which portal the player walked through
// is held by internal/session and saved with the run, so a run code plus its picks is the whole
// path.
//
// **No Ebitengine, ever**, and no randomness of its own: New takes the source it draws from, so
// the caller owns which stream is being advanced. See the randomness skill.
package journey
