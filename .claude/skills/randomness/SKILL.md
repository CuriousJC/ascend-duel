---
name: randomness
description: How randomness is implemented in this repo - the run seed, the separate salted streams and what each one owns, the pins used by tools and demos, and the rules a new roll has to satisfy. Load before adding any roll, adding or seeding a stream, touching a salt or a seed, writing a shuffle, or deciding whether a mechanic should be random at all.
---

# Randomness and determinism

Runs will eventually be **replayable from a seed**: the same tower, the same enemies, the same
rolls, so a player can retry a run and make different choices. Nothing replays yet — `Session`
does not exist — but every roll written now either preserves that property or quietly destroys
it, and the second kind is invisible until the day someone tries to replay something.

**Combat is stochastic as of 2026-08-14.** Lightning rolls. That is exactly the case these
rules were written to survive, so follow them rather than reading the first roll as permission
for the second.

## The three rules that are never bent

- **Never call the `math/rand` package-level functions** — `rand.Intn`, `rand.Float64`,
  `rand.Shuffle`, and every other one. They draw from a global source shared with every other
  caller in the process, so the same seed replays differently depending on what else ran.
  Randomness always comes from an explicit `*rand.Rand` carried on state.
- **Every consumer gets its own stream, and a stream is only ever advanced by its own
  concern.** Sharing one between two concerns means a change to either silently rerolls the
  other.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before
  playback begins, so animation speed, the planned game-speed setting and any skip button are
  free to alter pacing and must not alter results. The same constraint binds `internal/trace`,
  `internal/idle`, the scripted demo, and both debug flags.

## The run seed

**`GlobalState.RunSeed` is the root, and `main` sets it once** — from `fixedRunSeed` if that
constant is non-zero, otherwise from the clock — and logs it, which is what lets a bug report
name a run.

**Reading the clock there is not a breach of "no `time.Now()` in game rules".** Choosing a
seed is the one place a run is allowed to be unpredictable; it happens once, outside the rules,
in `main`. Everything downstream derives from the number it picked.

`fixedRunSeed` is the debugging toggle — zero rolls a new run, anything else pins one.

## The streams

**`internal/seeds` derives every one of them, and it is the only place a salt exists.** A caller
names a stream and asks for a seed:

```go
seeds.For(gs.RunSeed, seeds.EnemySelect)                  // per run
seeds.ForFight(gs.RunSeed, seeds.PlayerDeck, fightIndex)  // per fight, zero-based
```

**The salts are unexported**, so sharing one between two concerns is not expressible — "a stream
is only ever advanced by its own concern" is a type rather than a convention. Asking for the
wrong scope **panics**: seeding a per-fight stream as if it were per-run would hand every fight
the same shuffle, which looks like it worked.

`Stream` is **append-only**, like `combat.Element` and `systems.GlyphKind` — the ordinal indexes
the salt table, so inserting one mid-list re-points every stream after it.

| Stream | Scope | Used by | Sharing it would reroll |
|---|---|---|---|
| `seeds.EnemySelect` | run | `roster` (`internal/screens/combat.go`) | the whole tower, on any change to loot or offers |
| `seeds.CombatRoll` | run | `CombatScene.combatRNG`, injected into `ResolveRound` | every shock in the run, on any change to draw |
| `seeds.PlayerDeck` | fight | `CombatScene.rng` | every catalogued hand in `internal/screens/seeds.go` |
| `seeds.EnemyDeck` | fight | `decks.EnemyPile` | the player's opening hand, per the entry below |
| Loot offers | — | **not built** | — |
| Floor offers | — | **not built** | — |
| Between-fights reward hand | — | **not built** | which cards you are offered to alter |
| Which alterations are offered | — | **not built** | — |

**The last two are the next streams to land**, and they are the first real test of whether one
stream can answer two questions. The reward hand is a fresh deal off the whole run deck,
ignoring the fight's own piles, so it must not share the player's shuffle: doing so would make
the offer a function of how many cards were drawn in the fight just won. Whether the *menu* of
alterations wants a stream of its own or can ride the same one is undecided — apply the test
above rather than assuming either way.

**Tower layout draws no randomness.** It is fixed at 8 floors × 3 fights, endless later.

**Enemy selection is shuffled within each floor band**, not across the roster, so a run opens
on a different opponent without a floor-eight enemy ever being fight one.

**Per-fight streams are why a defeat and a retry deal that fight again** rather than dealing a
new one — the same property the enemy roster has, and for the same reason: nothing re-rolls a
run until `Session` exists.

### The two card shuffles are separate, and that is the worked example

Sharing them would make the player's opening hand a function of how many cards the opponent
happened to draw, so **every entry in `internal/screens/seeds.go` would break the first time an
enemy deck was retuned**. A named hand has to stay a fact about the player's deck alone.

That is the shape of the argument to apply to any new stream: **ask what it would silently
reroll.**

### Why it is a package *(2026-08-17)*

The salts used to live in three packages — four in `internal/screens`, one in `internal/decks`,
one in `tools/balance` — and `decks` held one **only** because the tool cannot import a screen.
So no single place saw them all and **nothing could check that two consumers had not been given
the same salt**. `TestEverySaltIsDistinct` is that check, and it could not be written before the
package existed.

`internal/seeds` imports nothing but the standard library and sits at the bottom of the graph
beside `internal/state`, which is what lets the screens, the decks and the tools all reach it.
**`internal/combat` does not import it and must not** — the rules take an injected `*rand.Rand`
and are deliberately ignorant of where it came from.

`fightStride` is internal to the package: a large odd number, mixed in as
`(fightIndex+1) * fightStride`, so consecutive fights are not consecutive seeds and fight zero
does not cancel the stride. The `+1` is in `ForFight` rather than at a call site, which is where
it used to be forgotten.

**The salt values may be changed freely today and will not always be.** Nothing persists a run,
so changing one changes nothing observable. That ends the day a save file or a shareable seed
exists — which is the reason this package was built before those were.

## The pins, and what each is for

Pins exist so a picture, a test or a balance number is reproducible. **Every one of them is
off by default**, because a pinned game is not the game.

| Pin | Default | Fixes |
|---|---|---|
| `fixedRunSeed` (`main.go`) | 0 — rolled from the clock | the whole run: enemies, shocks, both shuffles |
| `deckSeedName` / `deckSeed` (`combat_deck.go`) | `""` — unpinned | the player's hand *and* the opponent's, together |
| `seeds.EnemyDeckPin` | always used by `tools/balance` | the opponent's shuffle |
| `balanceSeed` (`tools/balance`) | always | the lightning rolls in a balance run |
| `oddsSeed` (`tools/handodds`) | always | which hands the rarity sample deals |

**`deckSeed` pins both sides or neither.** Half a reproducible duel is worse than none: the
hand looks right and the fight still differs. When it is non-zero the opponent's pile falls
back to `seeds.EnemyDeckPin`, which lives in `internal/seeds/pins.go` — pins are kept out of the
stream table on purpose, because a pin sitting in it would read as a stream nobody salted.

**A seed is an opening hand**, because the shuffle is deterministic — see
`internal/screens/seeds.go` for the named catalogue (a different file from this package, and
older than it) and `go run ./tools/seeds` for re-checking it. Re-run that tool after touching
`data/duelist_cards.json`, `startingDeck` or `handSize`: a named hand is a fact about one
particular deck, and changing the deck silently deals something else.

## Adding a roll — the argument comes before the code

**Rewrite a random-sounding rule rather than let it in.** Lightning is the deliberate
exception, not the precedent. It was taken because unreliability is what lightning *is*, and
because the alternatives — breaking the hand, cutting the multiplier — were weighed and written
down in `MECHANICS.md`.

**Certainty is often the better game as well as the cheaper code.** It matches the rule hands
otherwise follow: what you committed to cannot be silently undone. **A second roll needs the
same argument made from scratch**, in `MECHANICS.md`, not an appeal to lightning.

What a roll costs, using the one that exists as the measure — both of these were predicted
before it landed and both are now paid:

- `tools/balance` became a distribution rather than an exact answer. A posture winning half its
  duels and one winning all of them still print the same line.
- The stream advances per attack phase, so a change early in a duel reshuffles every roll
  after it.

## Adding a stream — the checklist

1. **Say what it would silently reroll if it shared an existing stream.** If the answer is
   "nothing", it probably does not need its own — but check the reverse too: what does the
   existing consumer reroll for *it*.
2. Add a `Stream` constant at the **end** of the list in `internal/seeds/seeds.go`, and a row to
   the table beside it carrying a fresh salt and the `perFight` flag. `TestEverySaltIsDistinct`
   and `TestNoTwoSeedsCollide` cover it from there without being edited.
3. Add a row to the stream table above, filling in what sharing it would reroll.
4. If it belongs to a mechanic, the *design* argument goes in `MECHANICS.md`. This file holds
   the plumbing.

## The rest of the discipline

- **`internal/combat` has no clock and exactly one roll.** It is otherwise integer arithmetic,
  and `TestRoundIsDeterministic` pins that a nil source resolves identically every time. The
  source is a `*rand.Rand` parameter on `ResolveRound`; a nil one means no rolls, which is what
  every test and every headless caller passes.
- **The deck lives on the scene, not in `internal/combat`.** Keeping the shuffle out of the
  rules package is what preserves its purity, its tests and the headless balance sim. Moving
  draw into `combat` is a real option later, but it has to arrive as an injected source
  parameter on `ResolveRound` and it changes `TestRoundIsDeterministic` — see the deckbuilder
  entry in `TODO.md` first.
- **Do not pre-roll randomness into fixed-size slices.** A seeded `*rand.Rand` already is an
  infinite deterministic list, and the planned endless tower gives no worst case to size an
  array against. A reroll simply advances the cursor.
- **Never let map iteration order affect an outcome.** Go deliberately randomises it.
  `gs.Combatants` is a map, and so is the enemy roster — iterate a sorted key slice
  (`data.EnemyOrder`, `data.RingOrder`) whenever a choice depends on order.
- **No `time.Now()` in game rules.** Wall-clock decisions cannot be replayed. Tick counters are
  fine; they are part of the simulation. The one exception is choosing the run seed, above.
- **`internal/music` has no `math/rand` either.** Its drum noise is a 15-bit shift register
  seeded from each note's start frame, so two renders cannot differ — which is what
  `music_test.go` pins.
