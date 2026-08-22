// Package scenario plugs a fixed set of rings, a fixed opening hand and a chosen enemy into a
// launched game, so an interaction between them can be *looked at* rather than played towards.
//
// # Why it exists
//
// A ring is bought from a shelf of three, a hand is dealt from a shuffled deck, and an enemy is
// whoever the climb put in the room. Every one of those is deliberate, and together they make
// "does Echo actually multiply Enflamed's growth" a question that takes twenty minutes of play to
// ask once. The rules are unit-tested and the arithmetic is pinned; what is not pinned is what the
// combination *looks like* on the screen, which is the one thing no test can answer.
//
// This is the ring-and-hand counterpart of `deckSeedName` and `session.StartingRings`, which do
// the same job one axis at a time.
//
// # It is compiled out, and that is the point
//
//	go run -tags scenario .                                # the first scenario in the file
//	ASCEND_DUEL_SCENARIO=echo-flurry go run -tags scenario .
//	go run .                                               # nothing: every function is a zero value
//
// A build tag rather than a runtime flag, for the reason `internal/trace` and `internal/idle` are:
// this hands the player a chosen hand and a chosen set of rings, which is not instrumentation a
// shipped binary may carry. **It must stay deletable in one commit**, so the call sites are three
// guarded lines and nothing else in the game knows this package exists.
//
// Unlike the two debug flags in `state`, this **deliberately changes outcomes**. It is a fixture,
// not a view — which is exactly why it may never ship.
//
// # The file is here rather than in data/
//
// `scenarios.json` sits beside this package rather than in `data/`, because `data/` is the game's
// own catalogue — every file there is loaded by every build and describes what the game *is*. A
// scenario describes a thing being tested. Filing it with the cards and the enemies would embed a
// debug fixture into a release binary and imply the game reads it.
package scenario
