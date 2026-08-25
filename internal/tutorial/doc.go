// Package tutorial is the teaching run's state machine: which step is up, what it points at,
// and what has to happen before it moves on.
//
// **It knows nothing about drawing and imports no Ebitengine**, which is the same rule
// `internal/combat` and `internal/decks` hold and it is here for the same payoff: the script can
// be walked end to end in a test, with no window and no duel, and a step whose condition can
// never fire is caught by `go test` rather than by a player stuck on floor one.
//
// # The three vocabularies, and why all of them are closed
//
// A step names an Anchor (what to point at), a Condition (what advances it) and whether it Gates
// (whether anything outside the anchor still accepts a click). Anchors and conditions are both
// enums parsed from strings at load, and a word the file invents is refused rather than ignored.
//
// That is the rule `data/SKILL.md` states as "do not grow a rules vocabulary in JSON ahead of the
// rules", and the failure it prevents here is specific: a misspelled anchor draws a spotlight
// around the empty rectangle at the origin, and a misspelled condition produces a step nothing
// can satisfy. Both look like a hung tutorial and neither looks like a typo.
//
// # Facts, rather than events
//
// A step advances when something happens in the game. The obvious way to do that is an event call
// at every site where something can happen — `tut.Did("duel-pressed")` in the DUEL! handler, one
// in the worm handler, one per screen — and it is the wrong way, because the failure of a
// forgotten call is a tutorial that silently stops advancing.
//
// So the traffic goes the other way. Each scene publishes a [Facts] once a frame, describing what
// is true right now, and a condition is a predicate over that. A scene that forgets to publish
// reports the zero value, which fails visibly and immediately rather than at one step in twelve.
// It also means the whole script can be driven in a test by writing structs.
//
// # What is deliberately not here
//
// **No rectangles.** An anchor is a name; `internal/screens` owns the geometry, because a
// rectangle is a fact about a layout and this package must stay window-free.
// `TestEveryAnchorHasARectangle` over there is what stops the two drifting apart — the same
// tripwire an `EventKind` has with its choreography entry.
//
// **No persistence.** Whether a given player has seen the tutorial is a profile question, and
// `TODO.md` says the profile does not exist. Today the script is started by hand — see the
// `tutorial` entries in `internal/scenario`.
package tutorial
