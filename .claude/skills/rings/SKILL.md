---
name: rings
description: The ring grammar - how a ring is written as data, the closed vocabularies it draws on, where each moment fires in the code, and what a ring may never do. Load before designing or discussing a new ring, adding an entry to rings.json or statuses.json, adding a moment or an effect verb, or wiring anything that reads a worn ring.
---

# Rings

**A ring is the only collected thing that is never played.** A card resolves in the turn you
queued it, a worm fires when you pick it, a hand is scored when the attack phase runs — each
already knows *when* it happens. A ring waits, so it has to say so itself.

That is the whole reason the grammar has three parts where the card language has two.

**This skill is for the conversation as much as the plumbing.** "Let's add a ring that does X" is
answered here: which moment it lands on, whether the vocabulary already covers it, and what it
costs if it does not.

## The shape

**A ring is a list of rules.** Each rule is `When` / `If` / `Then`.

```json
{
  "RingRecord": "chilling-ring",
  "Name": "Chilling Ring",
  "Art": "",
  "Text": "Ice attacks CHILL the target.",
  "Rarity": "uncommon",
  "Rules": [
    {
      "When": "attack-lands",
      "If":   { "Element": "ice" },
      "Then": [{ "Do": "apply-status", "Status": "chilled" }]
    }
  ]
}
```

- **`Name`** — the full name, which is what a tooltip titles. **The card face drops a trailing
  "Ring" and breaks the rest a word to a line**, via `RingData.FaceName`; two words is what fits,
  and `TestEveryRingNameFitsItsCard` fails a third.
- **`Art`** — the assets key for the face. **Empty draws `default-ring.png`** via `RingData.ArtKey`,
  which is what most of the file does; a key naming no embedded image fails
  `TestEveryRingDrawsSomething` rather than drawing a blank.
- **`Text`** — the line a player reads. **It is printed now** *(2026-08-21)*: the hover tooltip on
  every ring card, shelf and worn row alike, shows this and nothing generated from the rules. So a
  rule changed without its `Text` is a ring that lies to the player, and `TestEveryRingHasSomethingToSay`
  only catches an empty one, not a stale one.
- **`Rarity`** — `common`, `uncommon` or `rare`, and it decides both the price and how often the
  shelf offers it. There is no `Price` field; see the shop section below.
- **`When`** — which moment wakes the rule. Closed; one Go seat each.
- **`If`** — what has to be true. Optional; **a rule with no `If` always fires**.
- **`Rarity`'s third tier is where a drawback belongs.** Onslaught (2026-08-22) is the first ring
  that takes something away, and `scale-hp` below 100 is how it says so.
- **`Then`** — a list, so one rule can do two things. That is what buys a lightning ring that
  shocks *and* chills with no new vocabulary at all.

**A list of rules rather than one** because a ring can want two different moments — the growing
stat rings below need exactly that, one rule to accumulate and one to apply.

**`Do` is one word carrying both the operation and its subject** — `scale-damage`, not
`{Op: scale, Of: damage}` *(owner's call, 2026-08-17)*. Splitting it into two crossing lists
would buy a grid that is mostly meaningless cells, and `apply-status` sits on neither axis. This
is the same argument that took the mixes out of `hands.json`; do not re-propose it without a
new one.

## The three vocabularies

### `When` — the moments

Every one has a seat that already exists. **Four of the ten fire outside `internal/combat`**,
which is what makes a ring a *run* concept rather than a combat one.

| `When` | Package | Seat | Fires |
|---|---|---|---|
| `card-cost` | `combat` | `Card.Cost()` | per card, whenever a cost is asked for |
| `card-damage` | `combat` | `Card.Damage()` | per card, inside the blow's base sum |
| `attack-lands` | `combat` | `resolveAttackPhase` | once per landed blow |
| `deck-built` | `session` | `session.FightDeck` | once, as the fight's draw pile is built out of the run's deck |
| `card-drawn` | `screens` | `CombatScene.drawHand` | **per card, as it leaves the draw pile for the hand** |
| `fight-start` | `session` | fight setup | once per fight |
| `fight-won` | `session` | after the win | once per win |
| `prizes-dealt` | `screens` | `dealPrizes` | once, as the post-battle cards go down |
| `turn-taken` | `combat` | `playTurn` | once at the end of each of this duelist's own turns, **including an empty one**. Its `If` is matched against the turn as a whole: the rule fires when *any* card of the turn matches |
| `blow-formed` | `combat` | `handEvent` | once per blow, as the base sum is added up — **the only moment that sees the blow rather than a card**, and its `If` matches the *lead* card |

**`card-drawn` is the only moment a screen owns, and the invariant it costs is worth knowing**
*(2026-08-24)*. Every flip reads the card's **original** element, so that two of them cannot chain a
deck to one colour between them — lightning to ice to fire. While the flip fired at `deck-built`
that was true for free, because the fight deck was built out of the run's own cards once and nothing
had recoloured anything yet. Firing per draw, the discard pile is full of cards a flip has already
been through, so **the draw pile has to hold cards in the colours the run owns**: `drawHand`
restores a discarded card before folding it back in, and `session.DrawnAs` must never be handed a
card that has already been drawn. `TestTwoFlipsCannotChainThroughOneCard` is what holds it.

**A drawn card does not remember what it was** *(owner's call, 2026-08-24)*. It carries the colour it
became and nothing else, so a later rule — a `card-damage` ring keyed on ice — matches the card in
the hand rather than the card in the run. The original is still reachable, but only through
`combat.Card.ID` and `session.CardByID`, which are a handle for the layers *above* the rules; **no
rule may read them**. The deck panel is the one caller, and it uses them to draw either face of a
card the player already owns.

**`card-damage` and `card-cost` fire per card, and that is the point.** A form ring doubles
*every* card that matches, not one of them — three slash cards in a turn are three doublings
inside the same blow.

### `If` — the predicates

| Predicate | Matches on | Example |
|---|---|---|
| `Element` | the card's colour | `{ "Element": "ice" }` |
| `Form` | stab / slash / crush / plan | `{ "Form": "slash" }` |
| `Concept` | one named card | `{ "Concept": "Strike" }` |
| `Tier` | **the rung of its form's ladder** a card sits on — its *declared* cost, 1/2/3 | `{ "Tier": 3 }` — Atrophy |
| `Lead` | **the blow's first attack card**, not a fact about the card | `{ "Lead": true }` — Echo. `blow-formed` only; refused elsewhere |
| *(absent)* | always | Banker, Hungry, the stat rings |

**`Tier` reads the declared cost, never the wearer's.** A discount ring makes a Lunge cost 2 to the
duelist wearing it, and a rule matching `Tier: 3` still has to see a Lunge — otherwise two rings
worn together would silently switch each other off, and which one won would depend on the order they
were bought in. Same reading a worm takes.

**`Lead` is the first *positional* predicate, and more are expected** *(owner's call, 2026-08-22)*.
Element, form and concept ask what a card **is**; `Lead` asks where it **sits in the blow**. When the
next one of those arrives — last card, lone card, the card that formed the hand — it belongs here as
a predicate rather than inside a verb. **A verb that names its own scope is the anti-pattern this
replaced**: `echo-attack` meant "the lead card" until the form repeat rings needed the same
arithmetic at a different scope, and one predicate covered both.

**`Concept` names a card by its label**, resolved at load the way a deck list is. A concept *ID*
is registration-ordered and must never be serialized — the label is what is stable.

**A concept ring is a much narrower object than a form ring**, and pricing them the same is a
mistake waiting to happen: Striker covers 4 cards where Keen covers 12.

### `Then` — the effects

Each verb belongs to exactly one moment. **A verb used at the wrong moment is refused at load**,
not ignored.

| `Do` | Moment | Carries | Does |
|---|---|---|---|
| `adjust-cost` | `card-cost` | `Amount` delta | makes a matching card cheaper or dearer |
| `scale-damage` | `card-damage` | `Amount` percent | 200 is double |
| `apply-status` | `attack-lands` | `Status` key | puts a status on the target |
| `set-element` | `card-drawn` | `Element` | the flip: recolours a matching card as it is drawn |
| `demote-card` | `deck-built` | `Amount` rungs | steps a matching attack **down its own form's ladder** — a 3 AP Lunge is dealt as a 2 AP Thrust. Walks `Neighbour`; a card with no rung below it is left alone |
| `add-dmg` | `fight-start` | `Amount` | flat DMG for the fight |
| `add-hp` | `fight-start` | `Amount` | flat HP for the fight |
| `scale-hp` | `fight-start` | `Amount` percent | scales max life; **the one scaling verb meant to go below 100** — 75 takes a quarter off. Applied *after* every `add-hp`, and never below 1 life |
| `grow-on-win` | `fight-won` | `Amount` | adds to **this ring's own accumulator**, once per win |
| `grow-on-turn` | `turn-taken` | `Amount` | the same accumulator, once per matching turn — Momentum |
| `reset-growth` | `turn-taken` | *nothing* | puts the accumulator back to zero. **Growth is applied first and resets second**, so a turn cannot both bank and lose the same step |
| `grow-on-hit` | `attack-lands` | `Amount` | the same accumulator, **once per matching hit** — every landing, echoes and repeats included, so it grows inside a fight and compounds with the rings that multiply landings |
| `scale-propagation` | `fight-won` | `Amount` percent | scales vitae propagation, *after* its cap |
| `adjust-picks` | `prizes-dealt` | `Amount` delta | more post-battle choices |
| `adjust-prize-vitae` | `prizes-dealt` | `Amount` flat | the vitae card pays more |
| `repeat-card` | `blow-formed` | `Amount` landings | every **matching** card lands Amount times, each at **full** damage — the form repeat rings |
| `echo-attack` | `blow-formed` | `Amount` landings | the blow's lead card lands Amount times, at even fractions counting down — 3 is full, 2/3, 1/3. Extra landings from two rings **add** rather than compound; capped at `combat.MaxEchoLandings` |

**Adding a verb is a Go change** — one entry here plus the one place applying it — and that cost
is charged on purpose, exactly as it is for `combat.Verb` and `session.WormTarget`. A file may
never assert a verb into existence.

## Ordering: left to right, and it compounds

**Rings fire in the order they are worn**, left to right along the row *(owner's call,
2026-08-17)*. Two rings that both match the same card both apply, the left one first.

- **That is a determinism rule, not a preference.** Multiplicative effects are order-sensitive,
  so the order has to be one a rule can name — and worn order is the only one the player can
  actually see on screen. See the `randomness` skill.
- **Compounding is intended.** Two slash rings are ×4, not ×2, and that is a build.

## Statuses are their own collection

**A status is data, and it is no longer the same thing as an element** *(owner's call,
2026-08-17)*. `statuses.json`:

```json
{
  "StatusRecord": "chilled",
  "Name":   "CHILLED",
  "Badge":  "ice-effect",
  "Effect": "lose-actions",
  "Amount": 1,
  "Rounds": 2,
  "Text":   "loses one card off the front of each turn"
}
```

**Five effect kinds, closed**: `damage-over-time`, `lose-actions`, `miss-chance`,
`damage-reduction`, `damage-amplification`. A status is a file entry; a *kind* of status is a Go
change.

**`damage-amplification` is the odd one and the shape to know before adding a sixth**
*(2026-08-25)*. Every other kind modifies what its carrier *does*, so it is read off whoever is
acting; this one modifies what its carrier *takes*, so it is read off whoever is being acted upon —
a second site in the damage pipeline, and it reaches the burn tick as well as the blow. It is also
the only percentage with no natural ceiling, so `combat.maxAmplifyPct` caps it where the others are
bounded by *nothing reduces a blow to zero*. WEAKENED is the one record.

**Fully decoupled means fire does not burn on its own** — including for the five rings that ship.
There is no default status per element. This is the 2026-08-16 position held rather than
reversed: the statuses being free is what left rings with nothing to be, and giving a ring a
second fire status later is only possible if the first one was never inherent.

**What decoupling cost, all of it paid on 2026-08-17:**

- `Duelist.Statuses` is indexed by **status**, and its width is `combat.MaxStatuses` — an array
  width rather than a design cap, since a duelist must stay comparable. Registration refuses a
  record past it rather than dropping it.
- `Duelist.Rings` is a fixed array of `WornRing` plus a count, not a bool per element.
- `cards.MaxEffects` is **5 because the file holds five statuses**, checked by
  `TestTheCardHoldsAsManyEffectsAsThereAreStatuses`. The badge row fits six at the current pitch, so
  the fifth cost exactly that one number on 2026-08-25 — **and the sixth is the last one that is
  free.** A seventh is a redesign of the band.
- The badge is read off each record's `Badge`; `card_art.go` no longer keys anything by element.
- `StatusID` is append-only, and it is the *file* that decides the order — inserting a record
  mid-file re-points every status a duelist is carrying.
- **Queries are by effect kind and they sum**: two `lose-actions` statuses take two cards. Nothing
  applies two yet, but choosing between them silently would be a rule nobody wrote down.

## Growing rings hold state, and they are the first thing that does

Every other ring is a pure function of its record. A ring that gains +5 HP after every fight is
not — it carries a number that lives on the run:

```json
{
  "RingRecord": "heart-ring",
  "Name": "Heart Ring",
  "Rules": [
    { "When": "fight-start", "Then": [{ "Do": "add-hp", "Amount": 5 }] },
    { "When": "fight-won",   "Then": [{ "Do": "grow-on-win", "Amount": 5 }] }
  ]
}
```

- **Both growth verbs name their moment** *(owner's call, 2026-08-22)*: `grow-on-win` and
  `grow-on-hit`. `grow-on-win` was `grow` until the second one existed, and a verb whose name does
  not say when it fires reads as the default while the other looks like the special case.
- **`grow-on-win` writes an accumulator on the worn ring**, and the ring's own effect amounts are read
  as `Amount + accumulator`. So this ring is +5 HP in fight one and +100 by fight twenty.
- **`grow-on-hit` writes the same accumulator from inside a fight** *(2026-08-22)* — the Enflamed
  family, +0.1x to their colour on **every matching hit**. A hand with two fire cards is two steps,
  and a fire card an echo ring seats three times is three: it counts *landings*, which is what makes
  it compound with `echo-attack` and `repeat-card` rather than ignoring them. Two consequences
  that `grow-on-win` does not have: the *second* attack of a fight is already stronger than the first, and
  the growth is on the **duelist's** copy until `Session.AbsorbGrowth` reads it back on the win. A
  lost fight forfeits it, which needs no rule: a defeat ends the run.
- **A ring that can reset itself keeps nothing between fights** *(2026-08-22)*. `combat.KeepsGrowth`
  is the question and `Session.AbsorbGrowth` is what asks it: Momentum's streak is a fact about the
  turns of one duel, and banking it would make a good fight a permanent bonus that one plan card had
  once wiped. Heart, the stat rings and the Enflamed family hold no reset and are banked.
- **A turn-wide predicate is matched with *any*, which is how a negation is avoided.** Momentum is
  "grow every turn" plus "reset on a turn holding a plan card" — two positive rules where one rule
  would have needed a `not`. Reach for that shape before proposing negation into the grammar.
- **A step never reads the accumulator it is stepping.** Both verbs take the effect's raw `Amount`,
  so growth is linear; `Amount + Grown` there would compound and no growing ring is meant to.
- **The accumulator lives on `Session`**, keyed by `RingRecord`. It is the first ring state that
  has to survive a fight, and the first that will have to be **serialized** — which is why the
  record key is the identity and not an index.
- **Uncapped, by decision** *(2026-08-17)*. +5 a fight reaches +100 by the top of the tower and
  that is the intent, not an overflow.
- **A growing ring holds exactly one numeric effect** *(2026-08-17, owner's call)*, so the
  accumulator has exactly one thing to feed and does not need to say which. A ring wanting two
  growing numbers is outside the grammar; it needs a decision before it can be authored, not a
  second accumulator field written ahead of it.

## What a ring may never do

Reach for these first when an idea sounds too easy.

- **No ring raises `MaxActions`.** Frozen at five — see *A round is bounded twice* in
  `MECHANICS.md`. A ring may make five cards cheaper; it may never make it six.
- **No ring reduces a blow to zero.** Nothing in the game does.
- **Flips do not compose.** Every `set-element` reads the card's *original* element, so two flips
  cannot chain a deck to one colour and the order they were bought in cannot change the result.
- **Rings are the duelist's only** *(owner's call, 2026-08-17)*. An enemy wears none; affixes are
  the enemy-side counterpart. `attack-lands` is symmetric in the engine, so nothing has to be
  undone if affixes later reuse the machinery.
- **A ring may not change what a *concept* is.** Cost and amount are per-card
  (`Card.CostDelta`, `Card.AmountPct`); form and label are concept-wide, so a ring targeting
  one of those would change every copy in the deck. Same bound a worm has.
- **Five worn at once**, until brands expand it.

## Where the code is — it is built *(2026-08-17)*

| Piece | Where |
|---|---|
| the vocabulary, `RegisterRing`, and every applier | `internal/combat/ring.go` |
| the status catalogue and its lifecycle | `internal/combat/status.go`, `data/statuses.json` |
| parsing `rings.json` into rules, and registering it | `internal/session/ring.go` |
| what a run wears, and its accumulators | `session.Session` — `Wear`, `Worn`, `WornRings`, `Grown` |
| `deck-built` / `fight-start` / `fight-won` | `session.FightDeck`, `session.Equip`, `session.WonFight` |
| `card-drawn` | `session.DrawnAs`, called per card by `screens.CombatScene.drawHand` |
| `prizes-dealt` | `session.Picks` and `session.PrizeVitae`, read by `postbattle.go` |
| the row on screen | `internal/screens/combat_rings.go` — a lookup from worn key to record |
| the whole catalogue as pictures | `go run ./tools/ringsheet` — **grouped by rarity**, card, price, `Text` and rules side by side, and each tier's share of a shelf draw |

**`rings.json` is parsed in `internal/session`**, which already parses worms and for the same
reason: a ring belongs to a *run*. It hands `combat` rules types — `RegisterRing(key, name,
[]RingRule)` — so `data` stays ignorant of the rules and `internal/combat` never reads a file
holding an art key. That is the who-consumes-it test in the `data` skill, answered without a new
package.

**Bad records panic at load**, like every other catalogue: an unknown moment, a verb used at the
wrong moment, a predicate the rules cannot resolve, or a status key that is in no file.

**A duelist wears `[MaxWornRings]WornRing` plus a count**, not a slice — `Duelist` has to stay
comparable, so a worn ring is an ID and its accumulator and the rules themselves live in the
registry. `WearsRing` takes a `RingID`.

**Acquisition landed on 2026-08-21.** `internal/session/shop.go` holds the rules — `RingPrice`,
`SellValue`, `CanBuy`, `Buy`, `Sell` — and `internal/screens/shop.go` is the scene, reached through
`session.PhaseShop`. **A run opens wearing nothing** *(owner's call, 2026-08-21)*, so every element
is inert until the first ring is bought; `session.StartingRings` is the debug seat for putting one
on without playing to a shop and ships empty.

- **A ring carries a `Rarity` in `rings.json`** — `common`, `uncommon` or `rare` — and a record
  whose tier is missing or misspelled panics at load like every other unresolvable word. The tier is
  the whole pricing decision *(owner's call, 2026-08-22)*: common 3 vitae, uncommon 5, rare 7, with
  draw weights of 10 / 4 / 1. A ring is rebalanced by moving it between tiers, never by writing a
  number.
- **The shelf's three seats are weighted draws without replacement**, on those tickets, so a rare
  ring is something a run mostly does not see rather than something it sees and cannot afford.
- **Selling is the only way a ring comes off**, and it pays **the tier's own figure: 1, 2 or 3**.
  Buying at five worn is refused rather than swapped: the trade is two decisions with a price
  between them.
- **A sold ring's accumulator resets to zero** *(owner's call, 2026-08-21)*. `grown` stays keyed by
  record — a ring put back on is the same ring — but not the same number, so parking a growing ring
  in the shop between fights buys nothing.
- **The shelf is three rings the run is not wearing**, drawn off `seeds.ShopStock`, per fight.

## Discussing a new ring

The questions to put to an idea, in order:

1. **Which moment?** If none of the seven fits, that is the finding — say so rather than
   inventing one quietly. A new moment is a Go seat.
2. **Does the `If` exist?** Element, form and concept are the three. Anything else is new
   vocabulary.
3. **Does the `Then` exist?** If not, is it a new verb or a re-use? A new verb is one table row
   plus one place applying it.
4. **Does it hold state?** If it grows, it needs the accumulator and it needs saving.
5. **Does it collide with a ring that exists?** Banker, Soul Taker and Hungry all reached for the
   post-battle screen and two of them nearly did the same job.
6. **What does it cost the player, and does anything price it?** Nothing in the repo measures what
   a ring does to a duel, so every price is judgement. Say that rather than guessing at a number.

**Do not grow the vocabulary ahead of the rings.** A moment or a verb with no ring behind it is
`CostTier` again — see the `data` skill. Ship the rows that have an entry.

## The two vitae rings, and why they are not the same ring

They reached for the same screen and nearly did the same job. Settled *(2026-08-17)*:

- **Banker doubles vitae propagation**, at `fight-won` — `scale-propagation`, 200. It scales a
  rule of the run, defined and capped in *Vitae* in `MECHANICS.md`.
  - **The cap binds the base rate; the ring scales what the cap produced.** At 25 held that is
    +5 bare and +10 wearing Banker. An absolute cap would leave the ring doing nothing past 25,
    which is a ring that stops working when a run can finally afford it.
  - So the order is: count the fives, clamp to +5, then apply every scaling ring left to right.
    Two of them compound, like every other ring effect.
- **Soul Taker** is a **flat +5 to the vitae prize card**, at `prizes-dealt`: 5 becomes 10. Flat,
  not a percentage.
- **Hungry** takes neither — it adds a *pick*, not a value.
