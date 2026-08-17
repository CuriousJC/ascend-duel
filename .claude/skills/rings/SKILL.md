---
name: rings
description: The ring grammar - how a ring is written as data, the closed vocabularies it draws on, where each moment fires in the code, and what a ring may never do. Load before designing or discussing a new ring, adding an entry to rings.json or statuses.json, adding a moment or an effect verb, or wiring anything that reads a worn ring.
---

# Rings

**A ring is the only collected thing that is never played.** A card resolves in the turn you
queued it, a worm fires when you pick it, a combo is scored when the attack phase runs — each
already knows *when* it happens. A ring waits, so it has to say so itself.

That is the whole reason the grammar has three parts where the card language has two.

**This skill is for the conversation as much as the plumbing.** "Let's add a ring that does X" is
answered here: which moment it lands on, whether the vocabulary already covers it, and what it
costs if it does not.

## The shape

**A ring is a list of rules.** Each rule is `When` / `If` / `Then`.

```json
{
  "RingRecord": "frozen-ring",
  "Name": "Frozen Ring",
  "Art": "frozenring_png",
  "Text": "Ice attacks CHILL the target.",
  "Rules": [
    {
      "When": "attack-lands",
      "If":   { "Element": "ice" },
      "Then": [{ "Do": "apply-status", "Status": "chilled" }]
    }
  ]
}
```

- **`When`** — which moment wakes the rule. Closed; one Go seat each.
- **`If`** — what has to be true. Optional; **a rule with no `If` always fires**.
- **`Then`** — a list, so one rule can do two things. That is what buys a lightning ring that
  shocks *and* chills with no new vocabulary at all.

**A list of rules rather than one** because a ring can want two different moments — the growing
stat rings below need exactly that, one rule to accumulate and one to apply.

**`Do` is one word carrying both the operation and its subject** — `scale-damage`, not
`{Op: scale, Of: damage}` *(owner's call, 2026-08-17)*. Splitting it into two crossing lists
would buy a grid that is mostly meaningless cells, and `apply-status` sits on neither axis. This
is the same argument that took the mixes out of `combos.json`; do not re-propose it without a
new one.

## The three vocabularies

### `When` — the moments

Every one has a seat that already exists. **Three of the seven fire outside `internal/combat`**,
which is what makes a ring a *run* concept rather than a combat one.

| `When` | Package | Seat | Fires |
|---|---|---|---|
| `card-cost` | `combat` | `Card.Cost()` | per card, whenever a cost is asked for |
| `card-damage` | `combat` | `Card.Damage()` | per card, inside the blow's base sum |
| `attack-lands` | `combat` | `resolveAttackPhase` | once per landed blow |
| `deck-built` | `session` | run deck assembly | once, when the deck is made |
| `fight-start` | `session` | fight setup | once per fight |
| `fight-won` | `session` | after the win | once per win |
| `prizes-dealt` | `screens` | `dealPrizes` | once, as the post-battle cards go down |

**`card-damage` and `card-cost` fire per card, and that is the point.** A family ring doubles
*every* card that matches, not one of them — three slash cards in a turn are three doublings
inside the same blow.

### `If` — the predicates

| Predicate | Matches on | Example |
|---|---|---|
| `Element` | the card's colour | `{ "Element": "ice" }` |
| `Family` | stab / slash / crush / plan | `{ "Family": "slash" }` |
| `Concept` | one named card | `{ "Concept": "Strike" }` |
| *(absent)* | always | Banker, Hungry, the stat rings |

**`Concept` names a card by its label**, resolved at load the way a deck list is. A concept *ID*
is registration-ordered and must never be serialized — the label is what is stable.

**A concept ring is a much narrower object than a family ring**, and pricing them the same is a
mistake waiting to happen: Striker covers 4 cards where Keen covers 12.

### `Then` — the effects

Each verb belongs to exactly one moment. **A verb used at the wrong moment is refused at load**,
not ignored.

| `Do` | Moment | Carries | Does |
|---|---|---|---|
| `adjust-cost` | `card-cost` | `Amount` delta | makes a matching card cheaper or dearer |
| `scale-damage` | `card-damage` | `Amount` percent | 200 is double |
| `apply-status` | `attack-lands` | `Status` key | puts a status on the target |
| `set-element` | `deck-built` | `Element` | the flip: recolours every matching card |
| `add-dmg` | `fight-start` | `Amount` | flat DMG for the fight |
| `add-hp` | `fight-start` | `Amount` | flat HP for the fight |
| `grow` | `fight-won` | `Amount` | adds to **this ring's own accumulator** |
| `adjust-propagation` | `fight-won` | `Amount` per five held | adds to the vitae interest rate |
| `adjust-picks` | `prizes-dealt` | `Amount` delta | more post-battle choices |
| `adjust-prize-vitae` | `prizes-dealt` | `Amount` flat | the vitae card pays more |

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

**Four effect kinds, closed**: `damage-over-time`, `lose-actions`, `miss-chance`,
`damage-reduction`. A status is a file entry; a *kind* of status is a Go change.

**Fully decoupled means fire does not burn on its own** — including for the four rings that ship.
There is no default status per element. This is the 2026-08-16 position held rather than
reversed: the statuses being free is what left rings with nothing to be, and giving a ring a
second fire status later is only possible if the first one was never inherent.

**What decoupling costs, and all of it is real:**

- `Duelist.Statuses` is `[ElementCount]Status`, indexed by element. It has to be indexed by
  **status**. One element applying two statuses is the case that breaks it.
- `Duelist.Rings` is `[ElementCount]bool`. A slash multiplier has no element to be a bit under.
  It becomes a slice of rules-level rings.
- `cards.MaxEffects` is **4** because there are four elements. Decoupled statuses can exceed
  four badges on one card; `effectKeys` in `card_art.go` maps element→badge and becomes
  status→badge.
- `Status` becomes an append-only ID, the same hazard `Element` and `GlyphKind` carry.

## Growing rings hold state, and they are the first thing that does

Every other ring is a pure function of its record. A ring that gains +5 HP after every fight is
not — it carries a number that lives on the run:

```json
{
  "RingRecord": "heart-ring",
  "Name": "Heart Ring",
  "Rules": [
    { "When": "fight-start", "Then": [{ "Do": "add-hp", "Amount": 5 }] },
    { "When": "fight-won",   "Then": [{ "Do": "grow",   "Amount": 5 }] }
  ]
}
```

- **`grow` writes an accumulator on the worn ring**, and the ring's own effect amounts are read
  as `Amount + accumulator`. So this ring is +5 HP in fight one and +100 by fight twenty.
- **The accumulator lives on `Session`**, keyed by `RingRecord`. It is the first ring state that
  has to survive a fight, and the first that will have to be **serialized** — which is why the
  record key is the identity and not an index.
- **Uncapped, by decision** *(2026-08-17)*. +5 a fight reaches +100 by the top of the tower and
  that is the intent, not an overflow.
- `[?]` **A growing ring should hold one numeric effect until this is settled.** With two, it is
  not stated which the accumulator feeds. Ask before authoring one.

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
  (`Card.CostDelta`, `Card.AmountPct`); family and label are concept-wide, so a ring targeting
  one of those would change every copy in the deck. Same bound a worm has.
- **Five worn at once**, until brands expand it.

## Where the code goes

**`rings.json` is parsed in `internal/session`**, which already parses worms and for the same
reason: a ring belongs to a *run*. `session` hands `combat` a rules-level `[]Ring` on the
duelist, so `data` stays ignorant of the rules and `internal/combat` never reads a file holding
an art key. That is the who-consumes-it test in the `data` skill, answered without a new package.

**Bad records panic at load**, like every other catalogue: an unknown moment, a verb used at the
wrong moment, a predicate the rules cannot resolve, or a status key that is in no file.

## Discussing a new ring

The questions to put to an idea, in order:

1. **Which moment?** If none of the seven fits, that is the finding — say so rather than
   inventing one quietly. A new moment is a Go seat.
2. **Does the `If` exist?** Element, family and concept are the three. Anything else is new
   vocabulary.
3. **Does the `Then` exist?** If not, is it a new verb or a re-use? A new verb is one table row
   plus one place applying it.
4. **Does it hold state?** If it grows, it needs the accumulator and it needs saving.
5. **Does it collide with a ring that exists?** Banker, Soul Taker and Hungry all reached for the
   post-battle screen and two of them nearly did the same job.
6. **What does it cost the player, and does anything price it?** `tools/balance` measures postures
   against the roster and knows nothing about rings — so a damage ring is currently unmeasurable.
   Say that rather than guessing at a number.

**Do not grow the vocabulary ahead of the rings.** A moment or a verb with no ring behind it is
`CostTier` again — see the `data` skill. Ship the rows that have an entry.

## The two vitae rings, and why they are not the same ring

They reached for the same screen and nearly did the same job. Settled *(2026-08-17)*:

- **Banker** adds **a second +1 per 5 held** into vitae propagation, at `fight-won`. It scales a
  rule of the run — see *Vitae* in `MECHANICS.md`, which is where propagation itself is defined
  and capped at +5.
- **Soul Taker** is a **flat +5 to the vitae prize card**, at `prizes-dealt`: 5 becomes 10. Flat,
  not a percentage.
- **Hungry** takes neither — it adds a *pick*, not a value.

`[?]` **Whether propagation's +5 cap binds Banker as well.** At 25 held the pair would pay +10
against a rule that says +5. Carried in `MECHANICS.md` under *Vitae*; **settle it before building
either ring**.
