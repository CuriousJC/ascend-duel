// Package seeds derives every random stream in the game from the one run seed.
//
// **It exists because there was nowhere neutral to put a salt.** Four lived in
// `internal/screens` and one in `internal/decks`, the latter only because `tools/balance`
// cannot import a screen — so no single place saw them all, and nothing could check that two
// consumers had not been given the same one. This package imports nothing but the standard
// library and sits at the bottom of the graph beside `internal/state`, which is what lets the
// screens, the decks and the tools all reach it.
//
// **The salts are unexported on purpose.** A caller names a *stream*, and the only way to get a
// source is to ask for one, so sharing a salt between two concerns is not expressible. That is
// the whole point of the refactor — the rule "a stream is only ever advanced by its own concern"
// used to be a convention and is now a type.
//
// `internal/combat` does not import this and must not: the rules take an injected `*rand.Rand`
// and are deliberately ignorant of where it came from. See the `randomness` skill.
package seeds
