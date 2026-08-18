---
name: data
description: The game's static data - the seven JSON files in data/, the loader pattern, the card language every card in the game is written in, who is allowed to read which file, and where validation happens. Load before adding a file to data/, adding or changing a field on one, authoring cards or enemies or rings, or writing a loader.
---

# The data files

`data/` is **JSON next to a small Go loader**, and that is the pattern for all static game data.
It is the bottom of the dependency graph: **it imports nothing but the standard library**, which
is what lets every layer above read it, and it **must never import upward**.

| File | Loader | Holds |
|---|---|---|
| `duelists.json` | `LoadDuelists` | who the player can be: three stats and their card back |
| `enemies.json` | `LoadEnemies` | 96 opponents: three stats, their own deck, portrait, valid floors |
| `duelist_cards.json` | `LoadDuelistCards` | the player's deck, in the card language |
| `rings.json` | `LoadRings` | the rings that exist: name, art key, a line of text, and a list of `When`/`If`/`Then` rules |
| `statuses.json` | `LoadStatuses` | what a landed attack can leave standing: a name, a badge, one of four effect kinds, an amount and a duration |
| `combos.json` | `LoadCombos` | the six poker hands and what each multiplies a blow by |
| `worms.json` | `LoadWorms` | the deck alterations offered between fights |

## Who may read what, and why it is not "whether it is data"

**Three files are read by `internal/combat` itself**: `combos.json`, `duelist_cards.json` and
`statuses.json`. The rest are consumed by `screens`, `decks`, `session` or `entities`.

**`statuses.json` joined them on 2026-08-17** and passes the same test: how much a status is worth,
how long it lasts and which of four things it does are rules by definition — the engine cannot
resolve a round without them, and its own tests could not run if a screen had to hand them over.
Its `Badge` is the exception the engine ignores, exactly as it ignores a ring's `Art`.

**`rings.json` is the counter-example, and it is read by `internal/session`.** A ring's rules *are*
rules, but the record carries an art key and a ring belongs to a *run* — so `session` parses the
strings into `combat` types and calls `RegisterRing`. Same shape as `decks` for enemy cards.

**The test is who consumes a file, not whether it is data.** A card's cost and damage are rules
by definition; a portrait key, an art key and a floor band are a screen's or a roster's
business. A rule reaching for one of those would mean the rules had grown an opinion about
pictures.

**`internal/decks` exists for the one case that does not fit.** Enemy cards live in
`enemies.json` beside portraits and floor bands, so `internal/combat` reading that file directly
would cross the line above — and `data` may not import the rules to hand them over. `decks` sits
between the two and is the only package allowed to turn a JSON card list into rules types. It
registers every enemy concept. **No Ebitengine in it, ever**, because `tools/balance` imports it
headlessly.

## The loader pattern

Every file follows the same four lines: an `//go:embed` var, a struct with JSON tags, a `Load…`
that unmarshals it, and — for anything returning a map — a sorted `…Order` walk.

- **`//go:embed`, never a file read.** The data ships inside the binary.
- **A bad file panics**, with the filename in the message. It fails at launch rather than
  producing a roster quietly missing a record.
- **`EnemyOrder` and `RingOrder` are not optional.** `LoadEnemies` and `LoadRings` return maps
  and Go randomises map iteration, so anything whose *outcome* depends on order must walk a
  sorted key slice. See the `randomness` skill.
- **`LoadDuelistCards` returns a slice**, deliberately: the deck is built by walking it in
  order, and file order is grid order, so the JSON reads as the table in `MECHANICS.md`.
- **Loaded once.** `main` puts the roster, the duelists and the rings on `GlobalState`;
  `internal/combat` and `internal/decks` read their own files at package init.

## The card language

**Every card in the game is written in one language** *(2026-08-16)* — the player's and all 96
enemies'. Eight fields:

`Label` · `Verb` (attack / defend / bank / draw) · `Amount`, read against the verb · `Cost` ·
`Target` (opponent / self) · `Family` · `Elements` · `Copies`

- **`Elements` and `Copies` are two axes and neither substitutes for the other.** The player's
  attacks ship one per colour; its plans ship four copies of one colour; an enemy — all `basic`
  — has only `Copies`, so that field carries its whole deck size.
- **Deck size is a consequence of a file you can read**: 9 attacks × 4 colours plus 3 plans × 4
  copies = **48**.
- **There is no `Category` column.** Attack-or-plan falls out of the verb. Carrying both would
  let a file say a card is an attack that banks points.
- **`Copies` is the difficulty dial and it is sharper than it looks** — four copies of a 1 AP
  card in one turn is a Four of a Kind at 5x. Four is also the ceiling of the hand ladder.
- **No player card is drab except the plans** *(2026-08-15)*. Attacks are always coloured; the
  plans are basic because nothing they do is elemental.
- **Enemy cards are all `basic` and `FamilyNone`**, and that is deliberate rather than sloppy.
  The colour is read and carried, but `MECHANICS.md` has affixes *transforming* a basic deck
  into an element, so a colour typed into `enemies.json` would pre-empt a mechanic that does not
  exist. A family would be worse: it would claim an enemy card combos, and combos are the
  player's axis.

### Validation lives at registration, not in a cross-check

**`combat.RegisterConcept` is the validation.** Cost, damage, category and family used to be
switch statements over a closed `ActionKind` enum with a `CostTier` in the JSON that
`data.CheckCostTiers` asserted against them. That held fourteen concepts and could not hold the
~400 a per-enemy deck list produces, so the card became a record and both went.

What is checked now: a verb the vocabulary has, a cost that can be paid, an amount that does
something, a defence under 100% (**nothing may stop a blow outright**), and no bank or draw
aimed at the opponent — drain and mill are designed and unbuilt, so they are refused rather than
accepted and silently applied to the wrong duelist.

**A bad record panics at package init**, so it fails on launch rather than mid-duel. A deck list
or a catalogue naming something the rules cannot resolve is the same failure and takes the same
exit.

**Concept IDs are registration-ordered and must never be serialized.** The player's twelve
register first; enemy concepts follow as their decks are built. That is deterministic for one
build of one data set, which is a weaker promise than stability. Save formats name things.

## The records

### Three stats, and every one is the number it sounds like *(2026-08-16)*

`DMG`, `Actions`, `HP`, on both the duelist and the enemy records. `Constitution` and `Speed`
were conversions — `Con × 5` was life, `4 + Spd/10` was the action budget — so the roster was
tuned in units nobody could act on, and twenty-four distinct Speeds produced three distinct
budgets. **Every enemy's HP was doubled when the fields changed**, because the roster had been
written against a game where a turn landed several small blows.

### Duelists and enemies are separate files

Their fields do not overlap — an enemy has a portrait, a deck and a floor band; a duelist has a
card back. One struct would make every field optional and none of them mean anything.

**`ValidFloors` is `[lowest, highest]`** against the planned 8-floor tower, so a Dragon is not
on floor one. Nothing generates floors yet, so today it only sorts the fight order.

**A portrait's key is its filename stem**, unlike every other asset: 96 of them come in through
one `//go:embed enemy/*-portrait.png` glob, so renaming a file means editing the JSON. That is
the price of not hand-maintaining 192 lines nobody could review.

### Rings

**A ring is what makes its element do anything** *(2026-08-16)*: an attack applies a status only
if its owner wears that element's ring, so an unringed fire Strike is a plain Strike with a red
border. `Element` is the field that carries it — parsed in `internal/screens` with
`combat.ParseElement`, because `internal/combat` may not read this file. A name the rules do not
have is logged rather than dropped.

**What is worn is `startingRings` in `internal/screens`, not the file.** Four rings exist and the
player starts in three; earth is left off so a launch tests the gate as well as the statuses.
Temporary, and the counterpart of `deckSeedName` — buying and equipping needs `Session`.

**`Art` is an assets key, not a path**, and specifically a `LoadImageData` key rather than a
`LoadAssets` one, because a ring's picture is drawn *into* a card by `internal/cards`, which has
no graphics context.

### Worms

`worms.json` is **the card language pointed at a card that already exists**: a `Target` naming
which aspect changes, and a `Value` where the target needs one. `element` takes a colour;
`remove` and `duplicate` take none and **refuse one if it is supplied** — a record carrying a
value nothing reads is somebody expecting a mechanic the game does not have.

**Parsed and validated in `internal/session`, not in `internal/combat`.** A worm acts on the
*run's* deck, and the rules have no deck — the who-consumes-it test again.

**Seven targets**: `element`, `remove`, `duplicate`, `cost`, `amount`, `promote`, `demote`. The
vocabulary is closed the way the card verbs are — a new one is a Go change plus one place applying
it, never something a file can assert into existence.

**`cost` and `amount` are per-card**, carried on `combat.Card` as `CostDelta` and `AmountPct`, and
their bounds live in `Card.Cost()` and `Card.Amount()`. **Family and label are still
concept-wide**, so a worm targeting one of those would change every copy of that card in the deck —
make the argument in MECHANICS.md again before adding one.

**`amount` reaches every card with one worm**, because what the figure means depends on the verb.
That is the card language paying off, and it is the shape to reach for before adding a target.

### Combos

`combos.json` is **one list**: the six poker hands, High Card through Four of a Kind, each with a
key, an ID, a name, `groups` and a percent `multiplier`. Exactly one applies, winning on its
multiplier.

**A combo is a damage multiplier and nothing else** *(2026-08-17, owner's call)*. There is no
reward vocabulary to extend, no second axis counting colours, and no `scope` field — statuses come
from elements and rings, and the matcher counts attacks aimed at the opponent because that is what
`formsBlow` says, not because an entry asked it to. **Adding a rung is one entry in the JSON**;
adding anything a hand can *buy* is a design decision, not a field.

A malformed catalogue panics at init — including a missing `high-card` entry, since a hand the
engine cannot name is the one failure this model produces.

## Adding a file, or a field

1. **Say who will consume it.** That decides whether `internal/combat` may read it, and whether
   it needs a `decks`-shaped package in between.
2. Four lines: `//go:embed`, the tagged struct, the `Load…`, and a sorted `…Order` if it returns
   a map.
3. **Do not grow a rules vocabulary in JSON ahead of the rules.** The ring grammar is the worked
   example of doing it the other way round *(2026-08-17)*: every moment, predicate and effect verb
   in `rings.json` has a Go seat that refuses it at load if it is used wrongly, and a word the file
   invents does not exist. `CostTier` is what happens when a file declares something the rules also
   know.
4. If the file describes a mechanic, the *design* goes in `MECHANICS.md`. This skill is the
   plumbing.

## What is coming

The data is about to grow three ways at once, which is why this was carved out of `CLAUDE.md`:

- **More rings.** The grammar is built and sixteen are authored; growing the *vocabulary* — a new
  moment or a new effect verb — is a Go change, and is meant to be. What does not exist is buying,
  selling or unequipping.
- **More worms.** `worms.json` exists and holds ten across seven targets. Growing it is one record
  each; growing the *target vocabulary* is not, and MECHANICS.md says why.
- **Brands** — permanent for the run, altering the container where rings alter the contents. The
  mechanic is decided in `MECHANICS.md`; there is no `brands.json` and no acquisition.

Each is a new file or a new field asking the same question at step 1 above.
