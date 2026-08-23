package seeds

// Pins are the opposite operation to the streams in seeds.go: **do not derive, use this exact
// number**. They are kept in their own file, and out of the stream table, because the two are
// easy to confuse and the confusion is expensive — a pin in the table would read as a stream
// somebody forgot to salt.
//
// **Only shared pins live here.** A pin owned by one caller stays with that caller:
// `fixedRunSeed` is `main`'s, and `deckSeedName`/`deckSeed` belong to `internal/screens` because
// they name an entry in that package's own hand catalogue. What lands in this package is a pin
// two packages have to agree on.

// EnemyDeckPin is the pinned opponent shuffle, so a pinned duel deals the same cards both sides.
//
// **The game does not read it by default.** A fight seeds the opponent's pile from the run seed
// — see `shuffleSeeds` in `internal/screens` — and falls back to this only while `deckSeed` pins
// the player's hand, because pinning half a duel reproduces nothing.
//
// **It moved out of `internal/decks` on 2026-08-17**, where it had lived only because a headless
// caller cannot import `internal/screens` and the constant had to sit somewhere both
// could reach. That is the same pressure that produced this package, so it belongs here now —
// and `internal/decks` no longer declares a seed at all, which is right for a package whose job
// is turning card data into rules types.
const EnemyDeckPin int64 = 20260811
