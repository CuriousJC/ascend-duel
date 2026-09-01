---
name: data
description: The game's static data - the ten JSON files in data/, the loader pattern, the card language every card in the game is written in, who is allowed to read which file, and where validation happens. Load before adding a file to data/, adding or changing a field on one, authoring cards or enemies or bosses or rings or tutorial steps, or writing a loader.
---

# The data files

`data/` is **JSON next to a small Go loader**, and that is the pattern for all static game data.
It is the bottom of the dependency graph: **it imports nothing but the standard library**, which
is what lets every layer above read it, and it **must never import upward**.

| File | Loader | Holds |
|---|---|---|
| `duelists.json` | `LoadDuelists` | who the player can be: three stats and their card back |
| `enemies.json` | `LoadEnemies` | 96 opponents: three stats, their own deck, portrait, valid floors |
| `bosses.json` | `LoadBosses` | 30 stairway protectors: the enemy shape, with one floor instead of a band |
| `duelist_cards.json` | `LoadDuelistCards` | the player's deck, in the card language |
| `rings.json` | `LoadRings` | the rings that exist: name, art key, a line of text, a price, and a list of `When`/`If`/`Then` rules |
| `statuses.json` | `LoadStatuses` | what a landed attack can leave standing: a name, a badge, one of four effect kinds, an amount and a duration |
| `hands.json` | `LoadHands` | the poker hands on each of three matching axes, and what each multiplies a blow by |
| `worms.json` | `LoadWorms` | the deck alterations offered between fights |
| `stones.json` | `LoadStones` | one rung-raiser per hand: which rung it raises, and what its card says |
| `tutorial.json` | `LoadTutorial` | the tutorial script: what Bob says, what he points at, what moves him on |

## Who may read what, and why it is not "whether it is data"

**Three files are read by `internal/combat` itself**: `hands.json`, `duelist_cards.json` and
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
registers every enemy concept. **No Ebitengine in it, ever**, so an enemy deck can be built
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

`Label` · `Verb` (attack / defend / shield) · `Amount`, read against the verb · `Cost` ·
`Target` (opponent / self) · `Form` · `Elements` · `Copies`

- **`Elements` and `Copies` are two axes and neither substitutes for the other.** The player's
  attacks ship one per colour; its defences ship one per colour too; an enemy — all `basic`
  — has only `Copies`, so that field carries its whole deck size.
- **Deck size is a consequence of a file you can read**: 9 attacks × 5 colours plus 3 defences × 5
  colours = **60**.
- **There is no `Category` column.** Which phase a card is in falls out of the verb. Carrying both would
  let a file say a card is an attack that raises shields.
- **`Copies` is the difficulty dial and it is sharper than it looks** — four copies of a 1 AP
  card in one turn is a Four of a Kind at 5x. Four is also the ceiling of the hand ladder.
- **No player card is drab** *(2026-08-25)*. Every card in the deck ships in one of the five
  elements, the three defences included — a colour is worth a hand axis and a ring discount even
  where nothing the card does is elemental.
- **Enemy cards are all `basic` and `FormNone`**, and that is deliberate rather than sloppy.
  The colour is read and carried, but `MECHANICS.md` has affixes *transforming* a basic deck
  into an element, so a colour typed into `enemies.json` would pre-empt a mechanic that does not
  exist. A form would be worse: it would claim an enemy card forms hands, and hands are the
  player's axis.

### Validation lives at registration, not in a cross-check

**`combat.RegisterConcept` is the validation.** Cost, damage, category and form used to be
switch statements over a closed `ActionKind` enum with a `CostTier` in the JSON that
`data.CheckCostTiers` asserted against them. That held fourteen concepts and could not hold the
~400 a per-enemy deck list produces, so the card became a record and both went.

What is checked now: a verb the vocabulary has, a cost that can be paid, an amount that does
something, a defence under 100% (**nothing may stop a blow outright**), and a shield count no higher
than the attacks one turn can throw. **A card does not say
who it lands on** — the verb decides, an attack on the opponent and everything else on its own
duelist, and there is no field to disagree with.

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

### Bosses

`bosses.json` is **the enemy record with `ValidFloors` replaced by a single `Floor`, plus a
`Title`**, because a
boss guards the stairway of exactly one floor. `BossData.Enemy()` converts, so `internal/decks`,
`internal/entities` and `internal/cards` read one shape and never learn which pool an opponent came
from — the only two places that tell them apart are `pyramid.EnemyAt`, which answers a stairway
room from the boss pool, and the screen's `enemyFromRecord`.

**A separate file rather than an `IsBoss` column** *(2026-08-23)*: the two are placed by different
rules, and a flag would let a record be both while making every selection read it before it could
trust the floors.

- **Its portrait key ends `-boss`**, and the files live in `assets/boss/`. Both portrait families
  are globbed into one flat map keyed by filename stem, so the suffix is the whole of what stops a
  boss called `Sentry` from colliding with a creature of the same name.
- **Stats are pitched above the enemies of its own floor** — roughly 1.6x HP, and DMG above the
  hardest hitter in every band that reaches the floor — and the deck is dearer than a roster deck:
  60/120/250/300 against 50/100/200, and a 60% guard against 50%. `TestABossIsToughAgainstTheFloorItGuards`
  in `internal/pyramid` fails on a boss the floor below it could out-hit.
- **`Name` is the bare first name and `Title` is the rest** *(owner's call, 2026-08-24)* — `Jerry`
  and `the Toll-Taker`. They were one string, and the card could not hold it: `EnemyStyle` centres
  a name on one unwrapped line, so half the thirty rendered with a letter clipped off each end.
  The card takes `Name`; `Title` is for a hover nothing has built yet, and **nothing in the game
  reads it today**. `BossData.FullName()` joins them, so the hover and a review sheet cannot join
  them differently. It is a stored field rather than a split at render time because no rule finds
  the seam — `Bayaz, First of the Magi` breaks at a comma and `The Maw` has no title at all.
  `TestEveryOpponentNameFitsItsCard` in `internal/screens` holds the pool against the card's width.
  **The roster is not in that test yet**: five creatures are over the line and each belongs to a
  family whose other members fit, so trimming a subset would read as a mistake — see the test's
  comment.
- **A record whose deck is empty panics** exactly as an enemy's does, and one whose record name
  collides with an enemy's panics too — the deck registry is keyed by record and would otherwise be
  ambiguous.

### Rings

**A ring is what makes its element do anything** *(2026-08-16)*: an attack applies a status only
if its owner wears that element's ring, so an unringed fire Strike is a plain Strike with a red
border. `Element` is the field that carries it — parsed in `internal/screens` with
`combat.ParseElement`, because `internal/combat` may not read this file. A name the rules do not
have is logged rather than dropped.

**What is worn is on the run, not in the file.** A run opens wearing nothing *(2026-08-21)* and
buys its rings in the shop, so every element is inert until the first one is bought.
`session.StartingRings` is the debug seat for putting one on without playing to a shop — the ring
counterpart of `deckSeedName`, and it ships empty.

**`Art` is an assets key, not a path**, and specifically a `LoadImageData` key rather than a
`LoadAssets` one, because a ring's picture is drawn *into* a card by `internal/cards`, which has
no graphics context.

**`Rarity` is the price and the odds at once** *(2026-08-22, replacing a per-ring `Price`)*. One of
`common`, `uncommon` or `rare`; `data.Rarity` turns it into what the shop charges — 3, 5, 7 — and how
many tickets the ring holds in the shelf draw — 10, 4, 1. Three tiers rather than seventeen numbers,
because a per-ring price could only be judged one ring at a time. **What a ring sells back for is
deliberately not a field** — it is the tier's own figure, 1 / 2 / 3, computed in
`internal/session/shop.go`. A record whose rarity is absent or misspelled **panics at load**, like
every other word this file gets wrong.

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
their bounds live in `Card.Cost()` and `Card.Amount()`. **Form and label are still
concept-wide**, so a worm targeting one of those would change every copy of that card in the deck —
make the argument in MECHANICS.md again before adding one.

**`amount` reaches every card with one worm**, because what the figure means depends on the verb.
That is the card language paying off, and it is the shape to reach for before adding a target.

### Stones

`stones.json` is **the worms' shape pointed at the hand ladder instead of at a card**: a record
names a rung by its `hands.json` key, and using one raises that rung's multiplier by a tenth of the
catalogue figure for the rest of the run.

**Parsed and validated in `internal/session`, like the worms and for the same reason** — a stone is
held by a *run*. `internal/combat` owns the arithmetic and the seat a count sits in
(`combat/stone.go`), because what a rung pays is a rule.

**One stone per rung and one rung per stone, and every rung must have one.** A second stone on a
rung, or a rung with none, panics at load: a rung with no stone can never be raised and nothing
else would notice.

**No amount field, on purpose.** A record could carry `"Percent": 10` and it would be the `CostTier`
mistake again — a rules vocabulary declared in JSON ahead of the rules. The tenth is one decision
about the whole mechanic and it lives in Go. It becomes a field the day two stones want to be worth
different amounts, and not before.

**The figure is not in the `Text` either.** What a stone is worth depends on its rung's multiplier,
so `+11` written into the file goes stale the first time `hands.json` is tuned — silently, since
nothing reads a card's text. The record carries the sentence and `screens.stoneSpec` carries the
arithmetic.

### The tutorial script

`tutorial.json` is **an ordered list of steps wrapped in the run the lesson needs** — the steps are
a sequence like `duelist_cards.json`, since file order is play order and a map would put the run's
first lesson wherever Go's hashing felt like it.

- **`Seed`, `Enemy` and `Match` sit above the steps** *(2026-08-25)*. Bob promises four matching
  cards and a fight ended in one blow, and both are facts about one deal against one creature rather
  than about the game. They were pinned by `internal/scenario` while a fixture was the only way to
  start the lesson; the day the profile became a real trigger, the tutorial ran on whatever the clock
  rolled and described a hand it had not dealt. **A promise and the thing that makes it true belong
  in the same file.**
- **`Match` is a fourth closed vocabulary** — `concept`, `form` or `element`, the three axes a hand
  is scored on — and a script that points at a matching set without naming one is refused at load.
  The lit square and the condition that lets the player past it are the same cards, and which cards
  those are depends entirely on the axis, so a default would be a lesson pointing confidently at the
  wrong ones.

- **Two closed vocabularies, both enforced in `internal/tutorial` rather than here**: an `Anchor`
  (what to point at) and an `Until` (what advances the step). Neither is defaulted — a misspelled
  anchor would draw a spotlight round the empty rectangle at the origin and a misspelled condition
  would produce a step nothing can satisfy, and both look like a hung tutorial rather than a typo.
- **How much of the screen a step locks is *not* a field.** It is derived from `Until`: a step
  that wants reading locks everything, one that wants a click locks all but its anchor, and one
  waiting on an outcome locks nothing. It was a field for a few hours on 2026-08-25 and a step
  about which room you are standing in used it to leave the screen live while the player queued
  two cards nobody had mentioned. A field could only ever disagree with the condition.
- **An anchor names the control, never the region around it.** `first-card` exists because `hand`
  let a step that asked for one card accept five.

### Hands

`hands.json` is **one list of nineteen**: six poker rungs, Pair through Five of a Kind, on each of
three axes, plus the one High Card they fall back to. Each carries a key, an ID, a name, a `match`,
`groups` and a percent `multiplier`. Exactly one applies, winning on its multiplier, ties going to
the narrowest axis.

**`match` is the axis, and it is required** *(2026-08-19)* — `concept` (copies of the same card),
`form` (stab/slash/crush) or `element`. A missing or unknown one is refused at init rather than
defaulted: an entry landing on the wrong axis by omission would be a balance change nobody made.
`groups` counts distinct values **on that axis**, so `[3,2]` on `element` is three cards of one
colour and two of another.

**Keys carry the axis and the names are long**: `concept-two-pair` / `form-two-pair` /
`element-two-pair`, drawn as *Card Two Pair*, *Form Two Pair*, *Elemental Two Pair*. IDs are banded
— 1 high card, 10s concept, 20s form, 30s element — so a new axis or rung lands without moving one.
The five-of-a-kind rungs did exactly that on 2026-08-19, landing as 15, 25 and 35.

**The three ladders are priced apart and are meant to be.** They come from measured reachability
against the real 48-card deck rather than from poker's ordering: a form pair is a 99% hand at 110
and a concept Four of a Kind a 0.1% hand at 500. The model is in MECHANICS.md; do not "fix" the
ladders into agreement.

**Two of the three five-of-a-kind rungs could not be measured, and MECHANICS.md says so entry by
entry.** An elemental five costs 7 AP against a 6 AP turn and a card five needs a fifth copy of a
concept the deck does not ship, so their numbers are an extrapolation and a judgement rather than a
`ln(1/P)`. **Run `go run ./tools/handodds` before changing any of them**, and `-ap 8` for the rung
the plain budget cannot reach.

**A hand is a damage multiplier and nothing else** *(2026-08-17, owner's call)*. There is no
reward vocabulary to extend, no mix axis counting distinct colours, and no `scope` field — statuses
come from elements and rings, and the matcher counts every card in the turn because that is what it
does, not because an entry asked it to — what a card is worth to a hand is decided by the axis it is
counted on. **Adding a rung is one entry in the JSON**;
adding anything a hand can *buy* is a design decision, not a field.

**The multiplier multiplies the hand's own cards, and `100` is the identity** *(2026-08-18)*. A
blow is `(sum of the hand's cards) x multiplier / 100`, so `high-card` carries `100` rather than the
`0` it held while the percent applied to a separate swing added on top of the cards. **`0` is now an
attack phase that deals nothing** and is refused for every hand; a multi-card hand at or below `100`
is refused too, being one a player would be punished for building. Below `100` is legal for the
High Card alone and would be a penalty — deliberately allowed, because taking a lever out of the
file is the opposite of what the narrowing was for.

A malformed catalogue panics at init — including a missing `high-card` entry, since a hand the
engine cannot name is the one failure this model produces. Two shape checks sit beside it: a hand
wanting more cards than a turn holds, and one wanting more groups than its axis has values, since
only three forms and five elements ever reach a blow.

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

- **More rings.** The grammar is built and seventeen are authored; growing the *vocabulary* — a new
  moment or a new effect verb — is a Go change, and is meant to be. Buying and selling landed on
  2026-08-21, so a new record needs a `Rarity` as well as its rules.
- **More worms.** `worms.json` exists and holds ten across seven targets. Growing it is one record
  each; growing the *target vocabulary* is not, and MECHANICS.md says why.
- **Brands** — permanent for the run, altering the container where rings alter the contents. The
  mechanic is decided in `MECHANICS.md`; there is no `brands.json` and no acquisition.

Each is a new file or a new field asking the same question at step 1 above.
