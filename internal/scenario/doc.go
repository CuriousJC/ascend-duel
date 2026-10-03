// Package scenario plugs a fixed set of relics, a fixed opening hand and a chosen enemy into a
// launched game, so an interaction between them can be *looked at* rather than played toward.
//
// # Why it exists
//
// A relic is bought from a shelf of three, a hand is dealt from a shuffled deck, and an enemy is
// whoever the journey put in the room. Every one of those is deliberate, and together they make
// "does Echo actually multiply Weight of the Flame's growth" a question that takes twenty minutes of play to
// ask once. The rules are unit-tested and the arithmetic is pinned; what is not pinned is what the
// combination *looks like* on the screen, which is the one thing no test can answer.
//
// This is the relic-and-hand counterpart of `deckSeedName` and `session.StartingRelics`, which do
// the same job one axis at a time.
//
// # It is compiled out, and that is the point
//
//	go run -tags scenario .                                # the first scenario in the file
//	DUELLO_SCENARIO=echo-flurry go run -tags scenario .
//	go run .                                               # nothing: every function is a zero value
//
// A build tag rather than a runtime flag, for the reason `internal/trace` and `internal/idle` are:
// this hands the player a chosen hand and a chosen set of relics, which is not instrumentation a
// shipped binary may carry. **It must stay deletable in one commit**, so the call sites are three
// guarded lines and nothing else in the game knows this package exists.
//
// Unlike the two debug flags in `state`, this **deliberately changes outcomes**. It is a fixture,
// not a view — which is exactly why it may never ship.
//
// # The file is here rather than in data/
//
// `scenarios.json` sits beside this package rather than in `data/`, because `data/` is the game's
// own catalog — every file there is loaded by every build and describes what the game *is*. A
// scenario describes a thing being tested. Filing it with the cards and the enemies would embed a
// debug fixture into a release binary and imply the game reads it.
//
// # What a fixture can plug in
//
//   - **Three call sites, each one guarded line**: `main` sets `session.StartingRelics`, `Init` picks
//     the enemy, `resetDeck` plugs the hand. Nothing else in the game knows the package exists. The
//     `//go:embed` is in the `_on` file, so an untagged build carries neither the fixture nor the reader.
//   - **The hand is dealt over the shuffle rather than through it.** The draw pile is untouched, so the
//     second hand of the fight is a normal one and the fixture is only the opening.
//   - **A misspelled relic, card or enemy fails the launch**, at package init, before a window opens. A
//     fixture that quietly tests something else is worse than a game that will not start.
//   - **It can open the game on a named screen**: `"Screen": "reward"` or `"shop"`, with `Fight`,
//     `Vitae` and `Life` saying what state to arrive in, because a between-fights screen is otherwise a
//     duel away every time. It sets the run's *phase* and lets `screens/flow.go` decide the scene, so
//     the run never disagrees with what is on screen.
//   - **`"Essences"` plants the satchel**, beside `Runes` and `Stones`. An essence is normally spent the
//     instant it is taken, so a run *carrying* one into a duel is a state no amount of playing reaches —
//     see MECHANICS.md §An essence can be carried into a fight.
//   - **It can pin the seed and replace the whole deck.** `"Seed"` is a six-character Crockford base32
//     run code and outranks `fixedRunSeed`, and `"Deck"` sets the run's deck outright rather than
//     dealing over the shuffle the way `"Hand"` does — through `session.StartingDeckList`, the deck
//     counterpart of `StartingRelics`. A fixture that promises what the player is holding needs both,
//     since a refill otherwise deals a card nobody mentioned.
//   - **A deck line and a hand card may carry `"Riders"`**, by the names `combat.RiderKind` writes, with
//     a figure after a colon where the kind takes one — `"damage-on-play:10"`, or the bare
//     `"wild-element"`. `Runes` is the right fixture for looking at the *dialog* and the wrong one for
//     looking at what an altered card does to a hand, which needs a turn played to spend the consumable
//     first.
//   - **`"Teach": true` forces the tutorial on the run** whatever the profile says, and is the only way
//     to see it a second time.
//   - **Every entry carries a `Note` saying what question it answers**, printed at startup. A fixture
//     whose purpose nobody remembers is a fixture that gets deleted.
//   - **`"Dummy": true` is a fight that cannot end.** Both duelists get `DummyLife` and the clock goes to
//     `DummyRounds`, so a scenario can be *played with* rather than survived. **It is not a record under
//     `data/motifs/`, deliberately**: `data/` is the game's own catalog, loaded by every build, drawn on
//     the roster sheet and reachable by the journey's own roll. So it changes the *stats* of whichever
//     opponent was already there, and the fight keeps a real portrait, a real deck and a real set of
//     blows. **The clock is 999 rather than off**, because `session.SetRoundLimit` refuses to stop the
//     clock and the fixture goes the long way round rather than being given a back door into the rules.
//     `"RoundLimit": N` is the same dial on its own, and `"Actions": N` widens the turn's budget. **It
//     does not lift `combat.MaxActions`**: a turn is still five cards however cheap they are.
//
// # The two tools
//
// **`tools/scenariodeck` writes the `Deck` block, and it is deliberately a generator rather than a
// filter vocabulary.** `-form slash -size 40`, `-elements fire,ice`, `-cost 1-2`, `-riders golden:5`;
// it prints JSON to stdout and **never touches a file**. A `"DeckOf"` filter read at launch would be a
// *second card-selection language* living in a debug fixture, which has to stay in step with
// `data/duelist_cards.json` and `internal/decks` — and being a debug fixture is exactly why nobody
// would notice when it drifted. What lands in the file is the literal list, so `scenarios.json` stays a
// thing that can be read and checked. **The filters are meant to be extended**: the next axis is one
// `flag.String` and one clause in `pick`.
//
// **`tools/scenariosheet` is how the fixtures get found.** The page carries each fixture's Note
// against what it actually plugs in, with the launch command ready to copy. **It reads the JSON off
// disk rather than importing this package**, because importing it would mean building `tools/sheets`
// under `-tags scenario`; the cost is a second view of the record struct, and the tripwire is
// `DisallowUnknownFields`, which fails the sheet loudly when a field is added to one and not the other.
package scenario
