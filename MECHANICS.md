# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

**Running code:** cards and categories, phase resolution, hands, the 12-concept / 60-card
deck, and the elements and their statuses inside `internal/combat`. Those sections say so.
Everything else here is design that nothing implements yet.

**Hands are data** as of 2026-08-14 — `data/hands.json`, read by `internal/combat`, which is
the one edge from the rules into that package.

**A turn resolves one attack, and hands are how it is scored** *(2026-08-14)*. This is the
largest rules change the game has taken and it reaches into almost every section below: attack
cards are read as a *set* rather than played one at a time, the hand and the element makeup they
form are two multipliers that **add**, defends became percentage reductions instead of
negations, and lightning went back to being a roll. Where an older rule survives it is because
it was re-decided, not because it was left alone.

**And it is the player's rule, not the game's** *(2026-08-17, owner's call)*. An enemy's attack
cards resolve **one at a time, in the order its planner chose them**, each landing its own blow —
`Duelist.SoloAttacks` is the flag and `resolveSoloAttacks` is the phase. See "Enemies do not
hand" below.

---

## The thrust

**The primary thrust of the game is building a deck and an engine that bend the rules in a
way that lets the player win.**

Rings and brands are rule-modifiers first and stat-boosters second — more actions, cheaper
cards, free cards, and stats too, since a stat is just another rule to bend. Every constant
below is a candidate for something to bend.

The consequence for the code: rules cannot stay `const`. **Most of them already stopped being
constants** *(2026-08-16)* — a card's cost, damage, defence percentage, bank and draw are fields on
its record, so retuning one is a file edit. What is left as a compile-time constant is the
actions-per-round cap and the status magnitudes, read by functions with no access to the run. They need a **carrier** — a modifier set
passed alongside the duelists that `internal/combat` reads instead of the constants. Cost
becomes a function of the card *and* that carrier, the way `Damage` is already a function of
the action and the wielder.

Attributes do **not** need this. `DMG`, `Actions` and `HP` are already fields on `Duelist`, and
`ResolveRound` takes duelists by value, so a ring granting `+5 DMG` just hands it a different
duelist. Base values live in `data/duelists.json` and `data/enemies.json`, and are expected to
move with playtesting. **There is no conversion left to freeze** *(2026-08-16)* — see below.

---

## Attributes and scaling

**Three stats, and every one of them is the number it sounds like** *(2026-08-16)*: `DMG`,
`Actions` and `HP` on `Duelist`, all three straight out of `data/duelists.json` and
`data/enemies.json`. Life is HP. The action-point budget is `Actions + BonusAP`. Damage is
`DMG × the card's own multiplier ÷ 100`.

**All three conversions were removed over two days, and each for the same reason: a stat that
leads to a second stat and stops there is a step the player must learn and can never act on.**

- **`Str` became `DMG`** *(2026-08-16)*. The conversion was an identity — `Strike.Damage(Str)`
  returned `Str` — so two names described one number.
- **`Con` became `HP`** *(2026-08-16)*. Life was `Con × 5`, so the roster was tuned in units of
  a fifth of a life total.
- **`Spd` became `Actions`** *(2026-08-16)*, and it was the clearest of the three. The budget was
  `4 + Spd/10`, so the **twenty-four distinct Speed values across the ninety-six enemies produced
  three distinct budgets** — most of the hand-tuning in that column was never felt by anyone.

**Every enemy's HP doubled in the same change**, at the owner's call: the roster was written
against a game where a turn landed several small blows, and one blow per turn with hand
multipliers made every fight far shorter than the numbers read. See *Enemies* below for what
`tools/balance` says about the result, which is not yet a balanced game.

**The AP floor went with it.** `ActionPoints` clamped to a minimum of 1 because a chill
subtracted from it; nothing subtracts now, every term is non-negative, and the clamp was
arithmetic that could not fire. A future subtraction brings its own floor.

**Damage reduction is percentages all the way down, and no attribute is one of them.** Three
things cut a blow and they are all the same shape: the **earth status** blunts what the attacker
deals, the four **defend cards** each take a percentage off what arrives, and `guardDivisor`
halves whatever is left. A durable combatant is one with high `HP` or earth on it. Anything
that should reduce damage extends one of those three rather than arriving as a fourth system —
two mechanics quietly stacking is the failure to avoid.

**They compose multiplicatively and in a fixed order**: concept damage, the hand multiplier,
the attacker's weight, then every raised defend, then the guard. Weight sits on the attacker's
side of that line because it says how hard they can still swing; everything after it happens to
a blow that has already been blunted.

`[?]` **How enemies scale up the tower.** Enemies are fully-specified records with no level
term. A level multiplier — damage `level × 10`, speed `level × 5` — is the shape that has been
suggested. Nothing scales with floor today.

---

## Cards

### Forms and types

**Two axes, and they are not the same one** *(2026-08-15)*. `combat.Category` says *when* a card
resolves and has two values; `combat.Form` says *what kind of card it is* and has four. Category
is the coarser and is derivable from the form — everything outside Plan is an attack — so the
**form is what a card puts on its face** and what a hand is counted on.

**The attack set is a 3x3 ladder: three forms by three cost tiers, filled**, and the tiers are
identical across the forms. A form is *which* pair you are building toward, never a stronger
or weaker way to build one.

| Form | 1 AP · 0.5× | 2 AP · 1× | 3 AP · 2× |
|---|---|---|---|
| **stab** | Jab | Thrust | Lunge |
| **slash** | Cut | Slash | Cleave |
| **crush** | Bash | Strike | Smash |

| Form | Concept | AP | Effect |
|---|---|---|---|
| **stab** | Jab / Thrust / Lunge | 1 / 2 / 3 | Stabs for `DMG/2` (min 1) / `DMG` / `DMG × 2` |
| **slash** | Cut / Slash / Cleave | 1 / 2 / 3 | Slashes for the same three figures |
| **crush** | Bash / Strike / Smash | 1 / 2 / 3 | Crushes for the same three figures |
| **plan** | Prepare | 1 | Banks +2 AP for the next round |
| | Plan | 2 | Draws **2 extra cards** into the next round's hand |
| | Defend | 3 | Takes **50%** off the blow aimed at you next |

**Nine attack concepts × four colours = 36 cards; three plans × four copies = 12.** A **48-card
deck**. **No card in the player's deck is drab** *(2026-08-15)*: every attack ships in one of the
four primary elements and the only basic cards are the plans, because nothing a plan does is
elemental and a coloured Defend would be a colour that meant nothing.

**The plans sit on the same 1/2/3 ladder as the attacks**, so nothing in the game costs four. What
each buys is a different currency at a rising price: Prepare pays in points, Plan pays in cards,
Defend pays in survival.

**`Strike` is the 1× reference the ladder is written against**, and that is why the crush form
holds the name: `DMG` on the fighter card is `Strike.Damage(DMG)`, so the figure the player reads
is what one middle-rung card deals. Nothing stops that reference moving to another form's middle
rung; it is one constant. **What it is no longer is a term in the damage formula** *(2026-08-18)* —
a hand's multiplier applies to the cards that formed it, not to a reference swing added on top, so
`DMG` reaches a blow only through the cards themselves.

**The opponent has two cards of its own and they belong to no form** — `Attack` (2 AP, `DMG`) and
`Heavy` (3 AP, `DMG × 2`), priced against the player's tiers. `FormNone` is a real answer rather
than a fallthrough: forms are the *player's* deck axis, and an enemy card claiming to be a crush
would be claiming membership of a deck the player can build hands against. They draw with a blank corner.

**What the three forms do not yet have is a reason to be three.** They cost the same, hit the
same, and differ only in which cards pair with which. That is enough to make a hand a *choice* —
holding two Cleaves is not the same as holding a Cleave and a Smash — and it is deliberately the
whole of it for now. Riders that differ in *kind* per form are the shape to reach for next, and
the grid's old rule applies: a form that is only a different word is three cards and one
decision.

**Every card carries its effect in words on its face**, verb first, filling the card beside the
cost column. The attack text names the form's verb — "Stabs for 2x DMG" — rather than opening
"Deal" on all nine, so the corner letter is not carrying the distinction alone. The wording is
`cardEffects` in `internal/screens`, beside the prose the fight log uses: the rules package
names actions and never describes them. **Short words are a hard constraint** — the column is about
a dozen characters wide — and two tests hold the wording to it.

**The corner mark is a letter, and that is scaffolding** *(2026-08-15)*. S for stab, **D for
slash** — Stab took the S first, and two forms sharing an initial would leave the corner saying
nothing — C for crush, P for plan. The glyph machinery is untouched underneath: `cards.Form.glyph()`
returns nothing for every form today, and putting silhouettes back is one return value.

### Every raised plan answers the blow, and they multiply

**The plan cards a duelist has up are a set, not a queue.** The opponent's turn produces one
attack, so *every* raised card meets it and each takes its percentage off what is left. **Order is
not read**, and every card is spent on the one attack it answered. They all expire together at the
start of their owner's next turn.

**Defend halves, and it is the only card that reduces a blow at all** *(2026-08-15)*. Three points
of a four-to-six point budget is most of a round, which is what a halving is meant to cost.
Multiplying rather than adding is what stops several cards reaching past zero by accident: two
Defends take three quarters and a third takes seven eighths, a curve that never arrives.

**The plan form is three cards where the attack ladder is nine**, and it is deliberately not a
3x3 grid of its own. Prepare is the cheapest card in the game and Defend the dearest; what sits
between them is one card rather than a rung, because a grid filled with cards that differ only by a
number is the trap this deck was rebuilt to avoid.

### Concepts and deck composition

**An attack concept ships as four cards: one per primary element.** That is the rule for adding an
attack, not just a description of the starting deck. **A plan ships as four basics** — one concept,
four copies — because a plan has no elemental behaviour to carry.

36 + 12 = **48 cards**, implemented. A hand of eight against that is 17% of the deck, against 27%
when the deck was 30. What answers draw variance is the Discard button and now Plan.

**Four copies of a concept is the ceiling of the *starting deck*, and it shapes the hand table.**
No attack concept ships more than four times, so **a Card Four of a Kind necessarily shows all four
colours** — four copies of a concept are four different elements, so it is also the hand that lands
every status the player is ringed for. **The ladder still goes to five** *(2026-08-19)*: five of one
*form* is a 1-in-2700 turn off the starting deck and five of one *colour* is reachable once a
Prepare has banked, while a Card Five of a Kind needs a fifth copy of a concept and so exists only
after a `duplicate` worm. See the reachability table below.

**The deck list is data.** `data/duelist_cards.json` holds the twelve concepts, the form and
category each declares, the elements each ships in, and how many copies. `startingDeck` is built
from it.

### The card language

**A card carries its own rules** *(2026-08-16)*. Cost, damage, category and form used to be
switch statements over a closed `ActionKind` enum of fourteen constants, with the JSON declaring a
`CostTier` that was checked against them. That holds twelve player cards. It cannot hold three or
four bespoke cards for each of ninety-six enemies — roughly four hundred concepts, each wanting its
own multiplier and its own name — so the card stopped being an enum value and became a record.

Eight fields, and the player's twelve are written in the same language as every enemy's:

| Field | Means |
|---|---|
| `Label` | what the card face says, and — scoped by its owner — the rules identity |
| `Verb` | **attack · defend · bank · draw**. A closed vocabulary; a fifth is a Go change |
| `Amount` | read against the verb: % of DMG, % off the blow, points banked, cards drawn |
| `Cost` | action points |
| `Target` | **opponent · self** |
| `Form` | stab / slash / crush / plan, or none — the player's deck axis |
| `Elements` | which colours the concept ships in |
| `Copies` | how many of each |

- **There is no `Category` column.** Attack-or-plan falls out of the verb, and carrying both would
  let a file say a card is an attack that banks points.
- **`Elements` and `Copies` are two axes and neither substitutes for the other.** The player's
  attacks ship one per colour and its plans four of one colour — the same four cards reached along
  different axes. An enemy's cards are all `basic`, so `Copies` carries its whole deck size.
- **A key is scoped to its owner** (`ClearSlime1.Engulf`). Forty creatures want a card called
  `Bite` and they do not all want it at the same multiplier; the label collides freely, the key
  must not.
- **`Verb × Target` is a grid, and only recoil is new.** An attack aimed at `self` costs its owner
  life and forms no hand. Banking or drawing at the opponent — drain and mill — is designed and
  **refused at registration** rather than accepted and silently redirected.
- **Validation replaced the cross-check.** `CheckCostTiers` compared a declared cost against the
  rules and had nothing left to compare once the file became the rules. `combat.RegisterConcept`
  refuses an unknown verb, a defence of 100% or more, a zero amount, and the unbuilt half of the
  grid. A bad record panics at init.

**A card never names a status, and that is load-bearing.** See *Elements* — what a colour does is
decided by the source of that colour on the card's owner, and a ring may later decide *which* fire
a fire card applies. A card that named its own status would be deciding something that is not its
to decide.

**52 was considered and rejected**, and the arithmetic changed but the answer did not. The
playing-card instinct argues for 13 ranks × 4 suits, and the fifth "suit" here is `basic` — which
this document calls the absence of an element, not a colour of its own. With `basic` a variant the
attacks live on multiples of five; without it nothing is plain. The ladder decides it instead: nine
attack concepts is what three forms by three tiers produces.

### Hover and long press

**Hover explains, and long press is the same reveal on a touchscreen** *(owner's call, 2026-08-21;
built the same day)*. This reverses the split recorded when hover was first considered — *hover
un-occludes, long press explains* — and the reversal is the point rather than an oversight: the
thing a player needs from a card is not a bigger picture of it but the arithmetic behind its figure.

**Resting the cursor on something explains it.** A card gives the whole damage chain term by term —
your DMG, the card's own multiplier, every ring that matches, the result, and a line saying the hand
multiplier comes after; a ring gives its authored line from `rings.json` and where it fires in the
worn order; a fighter card gives its figures and every status standing on it, which is the only
place a badge can be read.

- **It arrives after a dwell of a beat and a half**, about six tenths of a second, so a cursor
  crossing the hand on its way to DUEL! does not strobe eight panels. Half a beat was tried first
  and read as a flicker following the mouse: a panel that appears before you have decided to want
  it is not answering a question. It is a proportion of the game's one speed like everything else
  that moves, so a speed setting will carry it.
- **The hand explains itself only while the queue can be edited.** A played card is still in the
  hand model during playback while being drawn on the table, so its old seat would otherwise answer
  for a card that had visibly flown away.
- **The panel is placed beside the thing, not under the cursor**, so it never covers what it is
  about and does not slide around inside one card.
- **Nothing in it recomputes a rule.** Every figure comes off the same walk the engine compounds,
  `combat.RingContributionsAt`. A tooltip doing its own arithmetic would be a second implementation
  of the engine printed in a box.

**Long press is what a touchscreen or a controller would use for the same reveal**, and it is not
built. Un-occluding an overlapped card is still unbuilt too, and is now a separate want from
explaining one.

A press remains a three-way decision the day long press lands — move past `dragThreshold` is a
drag, held past a tick count without moving is a long press, released before either is a click that
toggles selection. The distance and time thresholds must not fight each other.

---

## Elements

### The set

| Tier | Elements |
|---|---|
| **primary** | ice, fire, lightning, earth |

**There are four elements and no more** *(decided 2026-08-14)*. Every one of them has cards, a
colour and a status, so an element is a complete thing rather than a name waiting for rules.
Anything wanting a fifth has to arrive with all three.

`basic` is the absence of an element, not a fifth colour. It replaced `none`/`plain` in the
code's naming.

### Colour

| Element | Colour |
|---|---|
| basic | mid grey |
| fire | orange |
| ice | medium blue |
| lightning | yellow |
| earth | green |

`cards.BorderOf` is the live table; this one says what the colours are *for*. Basic is a mid
grey rather than the near-white it used to be, because the surface went off-white and a
near-white border on it is invisible.

One collision is live: **the player's green swatch sits near earth's green**, which earth's
move off brown on 2026-08-14 made sharper rather than created. "Green is you, grey is them" is
a screen-wide rule and an element breaks it. What holds it together for now is that the two are
never seen side by side — a swatch is a square in a pane row, a border is the edge of a card —
so the fix is deferred rather than done. Either the sides stop being colour-coded or earth
takes a green far enough from `playerSwatch` to read as a different idea.

### Statuses

*Implemented in `internal/combat/status.go`.* Each element has a status it applies **to whoever
took the blow** — **and only if the attacker is wearing that element's ring**:

| Element | Status | What it does |
|---|---|---|
| **fire** | burn | 10% of the attacker's DMG at the end of each round it survives |
| **ice** | chill | one card off the front of every turn it outlives |
| **lightning** | shock | 25% chance the victim's attack misses, rolled every attack |
| **earth** | weight | the victim deals 25% less damage |

**Statuses are off by default, and the ring is what switches one on** *(2026-08-16)*. An
unringed fire attack is a plain attack with a red border: it forms hands exactly as any other
card does and it leaves nothing behind. `combat.Duelist.Rings` is the flag array, indexed by
element exactly as `Statuses` is, and `resolveAttackPhase` reads it off the **attacker** before
applying anything.

**Why the reversal.** Statuses given away free left the first three rings with nothing to *be* —
every ring had to invent a second mechanic to sell, because the thing its element does was
already happening. Charging a ring for it makes the element set a hand axis on its own terms and
makes a ring the thing that turns a colour into a rule. It also gives the loot a shape: what a
ring buys is legible in one line of card text, and the second and third rings are worth buying
because one ring is one element.

**Enemies never wear rings.** The zero value is what an enemy is hydrated with and nothing sets
it, so an enemy's colours are inert by construction rather than by a rule written down somewhere
else.

### One rule, two sources — the intersection *(2026-08-16, owner's call)*

**An element does something only where a card's colour meets a source of that colour on its
owner.** The player's source is a **ring**. An enemy's is an **elemental affix** — its own, or the
floor's. Neither side gets statuses free; both get them at an intersection.

What this buys is that `Duelist.Rings` turns out to be the general mechanism rather than the
player's half of one: an affix sets flags in the same array. Nothing new is needed for it, and the
name is what should eventually change rather than the machinery.

**A card still never names a status**, and that is the reason the rule is worth stating this way. A
ring may later confer *which* fire a fire card applies — different rings, different burns — so the
decision belongs to the source and not to the card. See *The card language*.

**Enemy statuses are blocked on affixes, which do not exist.** Every enemy card is authored
`basic`, so today the whole element system still runs in one direction only. Colouring an enemy
card before an affix can gate it would hand it a free status, which is exactly what this rule
forbids.

**A self-side status is designed and unbuilt.** Each of the four statuses is what a colour does to
an *opponent*; aimed at yourself each has a mirror — fire enflames, ice focuses, lightning charges,
earth wards. **Lightning-on-self must not be a roll**, because a shock is the only randomness in
the engine and a second one needs its argument made from scratch. None of it is built: recoil
lands as plain self-damage, and a self-status with no source is not a status. Eight badges instead
of four is the art bill when it is.

**A run opens wearing no rings at all** *(owner's call, 2026-08-21)*, so **every element is inert
until the first one is bought**: an ice Strike is a plain Strike with a blue border. That is what
makes the shop the first thing a run saves for. `session.StartingRings` is the seat for putting one
on without playing to a shop — the ring counterpart of `deckSeedName` — and it ships empty.

**A status shows as a badge along the bottom of the enemy card** *(2026-08-16)*, from
`assets/effect/`. It is the only place a standing status is stated, and it has to be: two of the
four bite something the player has not done yet — a chill takes a card off a turn not yet queued,
a weight blunts a blow not yet swung — so without a badge they are learned by being surprised.
The row is centred and closes up as it fills. Earth's art is a placeholder. **The player's card
carries no badges**, because nothing can put a status on the player: the enemy wears no rings.

**Element crossed into `internal/combat` on 2026-08-12**, which is what this section had been
waiting on and what unblocked ring discounts and the flip ring with it.
`combat.Element` is a rules type, `combat.Card` is a concept plus an element, and `[]Card`
replaced `[]ActionKind` (now `ConceptID`) through `ResolveRound`, `ResolutionOrder`, `Slot`, `PlanFor`, `CostOf`
and every planner. The screen's own `element` type and its `actionCard` struct are gone —
`actionCard` is an alias for `combat.Card`, so the hand, the queue and the round are one type
and a card is never converted between them.

**Cost is a property of the pairing** *(2026-08-17)*. `Card.Cost()` is the card's own printed
figure and `Duelist.CardCost` is what it costs the duelist holding it, discounts included — which
is what everything that spends or checks a budget reads.

#### The trigger: the cards in the hand that formed

**Decided, rewritten by one blow per turn, and rewritten again by the ring grammar** *(2026-08-17)*.
The rings match against **the cards that formed the attack**, and each `apply-status` they fire lands
once however many cards matched it — so the four elemental rings still read as "one status per
distinct non-basic colour", and a form or concept ring reaches the same moment by the same route.
An all-basic hand lands nothing, because no elemental rule matches a colourless card. A plan card
carries its element for the ring discount and applies nothing itself;
the alternative — every card applying its status — would make a 1-AP Prepare as good a delivery as
a 1-AP Jab and turn the plan phase into the status engine. (The plans are all basic today, so this
is a rule waiting for a card rather than one currently biting.)

**This is the whole of what colour does to a blow** *(2026-08-17)*. Distinct colours used to pay a
second damage multiplier on top of the hand's — the *mix* axis — and that is gone: an element earns
its keep by what it leaves on the victim, not by hitting harder.

Three consequences, all of them changes from the per-card version:

- **A colour is counted once however many cards carry it.** Two ice Strikes and an ice Jab land
  one chill, where three separate ice hits used to land three. Status volume moved from "how many
  coloured cards" to "how many *different* coloured cards".
- **Cards outside the hand carry no colour at all.** `Strike, Jab, Strike` in fire, ice, fire is a
  fire Pair — one burn — and the ice Jab contributes neither damage nor a chill.
- **The status lands because the hand formed, not because the blow hurt.** A hand halved by a
  Defend still connected, and making the status conditional on the final figure would let a
  defensive card silently un-apply an element the attacker had already paid for.

The cost, stated: **element is mechanically inert on the three plan cards**, and they are all
basic, so today it is inert on nothing that exists.

**Magnitude is per hit, not per card.** A fire Jab and a fire Smash apply the same burn, so the
cheapest attack in the deck is the cheapest status delivery. The concept ladder prices damage;
the element ladder does not exist. Making status scale with the card is a second axis and a
design change. **Fire is the one that scales, and it scales off the *duelist*** — 10% of DMG —
which is a different axis from the card and does not reopen this one.

#### One lifecycle, learned once

**Nothing stacks; a second hit resets the clock, and everything clears at the end of the round
after the one that applied it.** `statusDuration` is 2 round-ends and it is one number for all
four deliberately. It cannot be 1: side B acts second, so a status B applied would expire before
it ever bit anything.

**Stacking went on 2026-08-16.** Amounts added until then, which made a status something to pile
on rather than something to keep up — and with one blow a turn, four stacks was four cards spent
saying one word louder. A ring that *does* stack is a ring someone can design; the base rule
being "no" is what leaves it somewhere to go. The two caps went with it: `shockMissCapPct` and
`weightCapPct` existed to stop four stacks reaching a certainty, and there is no longer a fourth
stack to cap.

Per-element tuning is one constant each away. Run `tools/balance` before moving one.

#### Lightning is a roll, and it is the only one in the rules

**A shock is a 25% chance the turn's attack misses, rolled on every attack the shock outlives.**
Nothing is consumed by a roll: with no stacks to wear down, a shock that spent itself on contact
would be a two-round status that reliably lasted one attack — a duration doing no work.

**This reverses the deterministic version taken two days earlier**, and one blow per turn is
what forced it. A certain miss used to delete one attack out of several; now it deletes the
whole turn, so a 1 AP lightning Jab could erase an 8 AP Four of a Kind outright. The alternatives
considered were breaking the hand or cutting the multiplier; a roll was chosen because lightning
should feel unreliable, which is a design reason rather than a balance one.

**It can never be a certainty**, which is what one blow per turn demands: a defence that always
works deletes a whole opposing turn for the price of one card. That used to need a cap over four
stacks; with stacking gone the ceiling is the number itself.

**What it costs, accepted rather than argued away:**

- `internal/combat` is no longer pure integer arithmetic. It takes an injected `*rand.Rand` on
  `ResolveRound` — never a package global, per the determinism rules — and a nil source means
  "no rolls", which is how tests and previews stay exact.
- `tools/balance` becomes a **distribution rather than an exact answer**. It currently plays one
  fixed-seed sample per matchup, which is reproducible but is one draw of many; multi-sample
  reporting is open work.
- The stream advances **per attack phase**, so a change early in a duel reshuffles every roll
  after it. That is the cost `MECHANICS.md` predicted when it argued against the roll; it is
  real and it is now paid.
- It breaks the rule hands otherwise follow — *what you committed to cannot be silently undone*.
  Lightning is the deliberate exception, and it is the only one.

#### The rest, and what each cost

- **Ice takes a card, not a point** *(2026-08-16)*. It cut the budget by 1 AP until then, which is
  the quietest status the game could have had: a duelist a point short queued a cheaper card and
  lost nothing they could name. A chilled duelist loses a card off the front of its turn instead,
  and the front of a turn is its attacks — so ice costs a swing. What that cost: an AP cut was
  felt while the player was still choosing, and a card taken off a committed turn is felt after
  they have.
  - **It is the only thing in the game that takes an action** *(2026-08-17)*. Hands could until
    then, and the machinery was shared; hands buy damage alone now, so the chill is read straight
    off the status and `Duelist` carries no separate counter. The action points are **not**
    refunded: a chill is tempo *and* economy.
  - **It bites on every turn it outlives**, rather than being spent when it bites — the status
    counting down is what ends it, and a second hit resets the clock rather than deepening it.
  - **The asymmetry phases impose is carried by the status.** Side A acts first, so ice A lands
    takes a card from B in the same round; ice B lands finds A has already acted and bites in the
    round after.
  - **A chill deletes cards before the hand is matched**, so a chilled duelist cannot swing with a
    turn it never took. That ordering is why the hand is worked out *inside* a turn rather than at
    the top of the round.
  - **In the log** it is announced as `KindChilled`, one event per card lost, which is what keeps
    playback's one-beat-per-slot invariant true.
- **Fire scales with the attacker.** A burn ticks for **10% of the DMG of whoever lit it**, read
  once and frozen onto the victim — a duelist whose DMG changes later does not retroactively burn
  harder. It floors at 1, the same rule Jab's damage follows, so a duelist under 10 DMG lights a
  burn that does something.
  - Fire needed state that outlives an action and got it. `KindBurned` fires from `endRound`,
    side A then side B, and the screen's `applyEvent` reads it alongside `KindDamage` because a
    burn changes a life total with nobody acting. **A burn can kill**, and produces a
    `KindDefeated` when it does.
- **Earth applies attacker-side, before any defence.** Weight says how hard you can still swing,
  so the order is: the hand's own cards, the hand multiplier, the attacker's weight, then every
  raised plan card. Everything the defender does therefore happens to a blow that has already been
  blunted. **Rounding is toward zero**, matching the defend reductions and `scaleDamage`, so it is
  predictable from the reductions already in the game. **25% rather than the 10% it was**, because
  10% that cannot stack is a status nobody notices landing.
- **Statuses got a home**, and it is `Duelist.Statuses [ElementCount]Status` — an array indexed
  by element, not four named fields. That is what makes *"consume the status this element
  applies"* expressible and is the difference between a system and four ad-hoc fields. The
  price: **`Element` is append-only**, like `GlyphKind`. The raised-defence set
  stays where it is — those are card effects, and filing them in a table indexed by colour would
  say they were not.

#### The balance numbers are gone and have not been retaken

`tools/balance` carries four element postures — all-out in a colour, same concepts and same
6 AP, so a coloured row read against `all-out` is what the element is worth. **Every figure it
has ever produced was measured against the multi-blow model and none of them survive the
rewrite.** Damage now runs through a hand multiplier, defends reduce instead of negating, and
lightning rolls, so the old table would be a picture that lies — the same reason a stale glyph
sheet is worse than none. It is deleted rather than annotated.

What has to be re-measured before anything is tuned, and roughly what to expect:

- **Damage is much larger and enemy HP has not moved.** Two Strikes at DMG 10 is `20 × 1.5` = 30
  where it used to be 20, and four Lunges are `80 × 5` = **400**. Retuning enemy life totals is the
  owner's call and is expected, not a bug report. *(The figures here were `20 + 10×1.5 = 35` and
  `10×5 = 50` on top of the cards until 2026-08-18, when the swing term went; see the damage
  formula above.)*
- **And the postures changed again on 2026-08-15.** The deck rework replaced every row: `trips`
  and `cheap-trips` are what the tool now reads a build against, and the old defend-column rows
  have no cards behind them. `defending` wins 16 of 96 and `planning` 1, both at the shallow end.
- **Shock's `[?]` is reopened, not answered.** It used to beat the entire roster by cancelling
  attacks for free; a 25%-per-stack roll capped at 75% is a different card and its price is
  unknown. `shockMissPct` and `shockMissCapPct` are the levers now, not `shockPerHit`.
- **The tool is a distribution now.** One fixed-seed sample per matchup is reproducible but is
  one draw; a posture that wins 51% of the time and one that wins 100% currently look identical.
- **`[?]` Every element beats plain, and the gap narrowed on 2026-08-17 without closing.** A fire
  Strike costs what a Strike costs and does strictly more. Distinct colours used to pay a damage
  multiplier on top of that, which widened the gap; with the mix axis cut, a coloured card is ahead
  only by the status it leaves — and only if the attacker wears the ring. That is a consequence of
  cost being per concept, which is deliberate and is what the ring discount is designed around.
  Worth deciding whether basic is a *cheaper* card or simply the thing an affix transforms.

---

## Resolution — phases

**Implemented.** Chosen as an experiment and built the next day; it
is what ships. Still open to reconsideration, but it is the model now, not a proposal.

A round is **a whole turn each**. Everything one side queued resolves before the other side
does anything, and within a turn the two categories go in order:

1. that side's **attack** — *one* blow, whatever it was assembled from
2. that side's **plans**, one card at a time: banks taken, defences raised
3. then the other side, the same way

**Plans come last** *within a turn* because the opponent moves next, so a defence raised at the
end of your turn is up when their blow arrives. A Prepare banks for the round after and does not
care where it sits; resolving plans first would mean every defence expired before anything could
be aimed at it.

**The combat screen lays a turn out in exactly this order**, and leaves a gap at the boundary — the
row on the table reads attacks, break, plans. That is not decoration: it is the round's two phases
made visible in the one place the round is a picture rather than a list.

**The attack phase is a single event, and that is the 2026-08-14 change.** Every attack card
queued is announced, the hand they form is announced, and then one figure of damage lands. The
plans are now the only cards that still resolve one at a time, because each does something to
its own duelist rather than contributing to a shared blow.

**And the phase says one thing, not one thing per card.** The announcements still happen — each is
a beat, and the screen raises the card that made it — but the *sentence* is the hand's: "HAND!
Duelist lands a Pair (20 x 1.5 = 30), 30 damage". Five lines saying a Strike was
swung describe five blows, which is exactly the reading this rule was written to end. **A blow that
forms no hand still gets its own ordinary sentence**, because a High Card is not a hand and
announcing one over every attack would empty the word.

**The sentence is a record, and it is no longer the only thing that says what a blow was made of**
*(2026-08-18)*. The hand dialog acts the sum out at the size of the screen on the beat the hand
fires — the hand's name shouted beside the cards it names, then each card's own figure flying down
into a line, then the multiplier, then the answer. It exists because the Resolution feed's line was sixteen
points of arithmetic on the third row of a three-row box: it *recorded* the sum correctly and never
showed which card paid which part of it, so the multiplier read as a number the game had decided
rather than one the player had built. **It says nothing the event does not carry and computes
nothing**, and the one thing it changes is pacing — playback holds while it runs. See the
`combat-screen` skill.

**The prepare phase is gone and its card moved** *(2026-08-15)*. Prepares used to run *before* the
attack, on the grounds that nothing they did reached it. With three categories collapsed to two,
Prepare joined the defences at the end of the turn — which changes nothing, because banking pays
next round either way. A plan that *did* feed the hand would need the phase order reopened, and
that is the argument to make rather than to quietly reorder.

**Why:** the interleaving may not be possible for players to grok. That is the whole reason.
It also simplifies — actions are gathered into their categories inside `ResolutionOrder`, and
hidden information survives untouched, because that is a single pure function which both
`ResolveRound` and the table's two rows read.

### Defense expiry — the rule this turns on

**A defense expires at the start of its owner's next turn, not at the round boundary.**

Side B acts last, so a defense cleared at the boundary would protect B from nothing it ever
faces — its own guard would go up after every attack it could possibly answer. Expiring at the
owner's next turn instead means every defense covers exactly one opposing turn whichever side
raised it: for A that is later the same round, for B it is early the next one.

The engine has no notion of "player" and must stay symmetric, so this is not a detail. It also
means expiry is a fact about **turns**, not about the action sequence — it lives in
`ResolveRound`, not in `ResolutionOrder`, because a side that queues nothing still has a turn
and still loses its guard in it.

### Initiative is gone

There is no initiative. With one contiguous turn per side there is no exchange for a faster
action to lead, so initiative was a number on every card reporting a distinction the resolver
had stopped making. `Spd` still buys action points and still never buys priority.

Ordering *within* a category is queue order, and **as of 2026-08-14 nothing reads it.** See
`TODO.md` for what would have to be true to bring initiative back — and for the same problem
arriving from the other direction, now that dragging has no mechanical effect at all.

### What this cost, recorded honestly

- **It reverses an earlier decision**, which replaced volley-per-side with alternation
  on the grounds that *"two monolithic volleys gave the player nothing to manipulate."*
  Phase-based is not the same as volley-per-side — it groups by category within a turn — but
  it is closer to the rejected thing than to what alternation was.
- **Cross-phase reordering means nothing.** A defense cannot be dragged ahead of an attack. This
  entry used to say within-phase ordering still mattered and mattered more than before; **that
  is no longer true.** Counted hands read the turn as a set and defends compose without an order,
  so as of 2026-08-14 **dragging a card changes nothing the engine can see**. That is a genuine
  loss of a designed interaction and it is tracked in `TODO.md`, not written off here.
- **Guard persistence dissolved**, as predicted. The old *"a raised Guard lasts until its
  owner's next action"* and the deliberate quirk that an idle duelist kept its guard are both
  gone. Guard itself went on 2026-08-15; Defend is what does its job now, as a plan card that is
  spent on the blow it answered rather than a flag that survived it.
- **It changes what makes a chill rare** — see the ice status under *Elements*.

`ResolutionOrder` being one pure function is what made this cheap, exactly as hoped: one
function body plus its tests, and both consumers followed without being touched. Three other
candidate models are recorded in `TODO.md`; this is a fourth.

---

## Hands

**This is where the game is meant to be.** Throwing whatever you drew at the opponent works;
*choosing a shape* and building a deck toward it is meant to work better. Hands are the
mechanism that pays for that choice.

**A hand is a damage multiplier and nothing else** *(2026-08-17, owner's call)*. It buys no
status, no action points and no action off the opponent's turn. Statuses come from **elements
and the rings that arm them**, and that split is the whole reason this section is now short:
there is one axis, one number per rung, and one place to look for what a hand is worth.

Hands are **discovered**, not given, and discovery persists on the **profile** — part of the
roguelike unlock structure, not the run. No profile exists yet, so every hand is currently
live; when one does, discovery gates the *table* and nothing else changes.

### The catalogue is data

*`data/hands.json`, with `data/hands_data.go` holding its shape and
`internal/combat/hand_table.go` turning it into rules. The vocabulary and the matcher are in
`hand.go`.*

**`data` holds the shape, `internal/combat` holds the meaning**, which is the division
`RegisterConcept` already draws for the deck lists. The file says how many copies of a card a rung
wants and what it pays; only the rules can say what a turn is and how wide one can be, so that is
where a malformed catalogue is refused.

**This is the one thing in `data/` that the rules themselves read**, and it is why
`internal/combat` imports that package at all. Everything else there is consumed by `screens`,
`decks` or `entities` — layers *above* the rules — so the rules never needed it. That is the
line to hold if a seventh list is proposed: **ask who reads it, not whether it is data.** `data`
imports nothing but the standard library, so the edge costs `internal/combat` neither its
testability nor its freedom from Ebitengine.

**A malformed catalogue panics at package init**, exactly as a deck whose declared cost tiers
disagree with the rules does. A hand silently dropped is a balance change nobody made.

### The pattern: three axes, and they wear poker's names

A hand counts **cards that agree** in the set that formed one attack — which is exactly what a
poker hand counts, so it wears poker's names honestly: High Card, Pair, Two Pair, Three of a Kind,
Full House, Four of a Kind.

**What they have to agree *on* is the hand's own axis** *(2026-08-19, owner's call)*, and there are
three:

| Axis | Cards agree on | Two that form a pair | Two that do not |
|---|---|---|---|
| `concept` | the same card | ice Bash + fire Bash | Bash + Strike |
| `form` | stab, slash or crush | Bash + Smash (both crush) | Bash + Thrust |
| `element` | fire, ice, lightning or earth | ice Bash + ice Thrust | ice Bash + fire Bash |

**Every rung exists once per axis**, as its own catalogue entry rather than as one entry with three
readings — so a Card Three of a Kind and an Elemental Three of a Kind can be priced apart, which
they have to be: one wants three copies of a nine-copy concept and the other three of nine cards
sharing a colour.

**The axes are not parallel, and the nesting is the thing to hold onto.** A concept fixes a form,
so **every card hand is also a form hand**; element is independent of both, which is why an ice
Bash beside a fire Bash is a card hand and no kind of elemental one. That asymmetry is what the
tie-break and the multiplier ordering below both exist to answer.

**A card with no value on an axis matches nothing on it.** `FormNone` and `Basic` are absences
rather than values, so an enemy's formless colourless deck cannot build a form or an elemental hand
at all — its whole ladder is the concept axis, which is what its `Copies` field was always buying.
The player's plans are basic too, though they have already been excluded for being plans.

**Exactly one hand still applies, and a tie goes to the narrowest axis.** Two Bashes satisfy the
Card Pair and the Form Pair at once; the narrower one is what the player aimed at, so `concept`
beats `form` beats `element` whenever the multipliers are level. `combat.Axis` is written in that
order for exactly this reason and is never serialized, so the order is free to mean something.

**Exactly one hand applies.** It wins on its multiplier — four Strikes are a Four of a Kind rather
than also the pair and the trips inside it — so a turn produces one hand with no ranking
machinery beyond that comparison.

**A lone attack forms no hand.** That is the fallback: when nothing counts, the single
hardest-hitting attack card is the blow, ties going to the card queued first.

**The fallback almost never fires any more, and that is the biggest single consequence of the three
axes** *(2026-08-19)*. Two attacks used to have to be the same card to both count; now they need
only share a form or a colour, so a turn of two mismatched attacks that landed the bigger one alone
lands the sum of both times a multiplier. Smash + Strike at DMG 10 goes from **20** — the Smash, by
itself — to **33**, and none of that comes from the multiplier being generous: 1.1x of two cards
beats 1.0x of one. Two attacks that agree on nothing at all are now the rare case rather than the
common one, and the High Card is what names it.

**Attack cards outside the hand contribute nothing.** `Strike, Jab, Strike` is a Pair; the Jab is
announced, is not in the hand, adds no damage and carries no colour. That is a stated rule rather
than a consequence — it is what makes *choosing a shape* pay more than throwing everything you
drew.

**Only attacks aimed at the opponent are counted, and that is the matcher's rule rather than the
catalogue's** *(2026-08-17)*. An entry used to name the categories it counted; the matcher's own
test is strictly narrower — it also excludes recoil — so the field could never change what was
counted and only invited an entry to claim otherwise. Plan cards are not counted, and a Prepare
cannot join a hand.

### Damage: one blow, one multiplier

A turn deals damage **once**, in the attack phase, and the figure is:

```
(damage of each card in the hand)  ×  (hand multiplier)
```

So a pair of Lunges at DMG 10 is `(20 + 20) × 1.5` = **60**.

| Hand | Cards | Multiplier |
|---|---|---|
| **High Card** | `[1]` | ×1 |
| **Pair** | `[2]` | ×1.5 |
| **Two Pair** | `[2,2]` | ×1.75 |
| **Three of a Kind** | `[3]` | ×2 |
| **Full House** | `[3,2]` | ×3 |
| **Four of a Kind** | `[4]` | ×5 |

**The multiplier multiplies the cards, and it did not always** *(2026-08-18, owner's call)*. Until
then the formula carried a third term — `Σ cards + DMG × multiplier` — where `DMG` was a reference
swing of one 1× attack at the attacker's strength, *added on top of* what the cards dealt. The
percent therefore bought a **fixed figure rather than a proportion**: at DMG 10 a Four of a Kind
was worth +50 whether it was built from four Jabs dealing 5 each or four Lunges dealing 20 each —
2.5× the base in the first case and 0.6× in the second. **The ladder paid least to the decks that
had climbed furthest**, which is backwards, and the arithmetic could not be read off `hands.json`
because the number the percent applied to was not in the file.

Two things follow and both were the reason for the change:

- **The ladder is tunable from `data/hands.json` alone.** The percent now applies to a figure the
  file's reader can see, so an entry means what it says.
- **A hand is worth more on bigger cards, in proportion.** A Pair of Lunges beats a Pair of Jabs by
  exactly the 4× the cards themselves are apart.

**The High Card pays the identity** *(2026-08-18)*. When a turn builds no pair or better, the
single hardest-hitting attack card is the blow and what lands is exactly its face damage — which
is now `×1` rather than the `×0` it carried while the multiplier was a bonus term, since ×0 would
be an attack phase that dealt nothing. It is in `hands.json` with a name and an ID so the log can
say what happened on the turn that happens most often; **a blow the engine could not name is the
one failure this model can have**, which is why the loader panics without it.

**It is fallen back to rather than matched.** Counting is the wrong way to pick it — `matchCountOf`
fills groups largest-count-first and would hand back whichever concept appeared most, not the card
that hits hardest — so `matchHand` skips every one-card hand and `biggestAttack` answers the
question on damage. `Blow.Formed()` draws the same line for the screen's hand preview: the High
Card is a hand, and it is not something anybody built.

**Colour buys statuses and no damage** *(2026-08-17)*. The distinct non-basic elements in the
formed hand each land their status, gated on the attacker wearing that element's ring; basic is
not a colour and never counts, so two basic Strikes and an ice Strike show one colour. That list
is all that survives of the second axis.

### What the axis cost, recorded honestly

- **Counted matching only.** A hand reads the turn as a set, so a Jab between two Strikes does
  not break the pair. The **run** match kind — N consecutive cards, which a sequence hand needs
  — is gone. The consequence is below, under *Sequences*.
- **A hand cut short still pays out.** Nothing can interrupt it — a turn's attacks resolve as one
  event — so this is true by construction rather than by rule.
- **The bottom rung fires constantly**, and as of 2026-08-19 it is priced as such: the form and
  elemental pairs are 98–99% hands paying 110, which is a floor rather than a reward. The open
  question of whether the ladder should start at Two Pair is **answered by pricing instead** — a
  near-certain rung pays near the identity, so it costs nothing to leave it in and it keeps the
  bottom of the ladder legible.
- **Poker's ranking does not transfer to this deck, and the ladders are now priced off measured
  rarity rather than off poker** *(2026-08-19)*. Poker's ordering comes from 52 cards, 4 suits and
  13 ranks; here a concept has 4 copies, a colour 9 and a form 12, and the turn is bounded by AP
  rather than by the draw. See *The multipliers come from how often a hand can actually be built*
  above for the model and the table.
- **Narrowing to damage alone cost ten enemies, measured** *(2026-08-17)*. `tools/balance` goes
  from 74 walls out of 96 to **84**, and every one of the ten is a hand posture — `trips` lost
  eleven wins and `cheap-trips` five, while the four coloured postures lost none. **What was doing
  the work was the lost action, not the lost colour multiplier**: a Three of a Kind used to take an
  action off the opponent's next turn, every round, which is what those postures were winning on.
  Cutting the colour multiplier flipped no outcome in the sample at all. The ladder's own numbers
  were kept unchanged through the cut on purpose, so this figure is the size of the hole before any
  retuning.
- **Dropping the swing term flipped no outcome either, and that is a fact about the tool as much as
  about the change** *(2026-08-18)*. `tools/balance` prints the same 84 walls and a byte-identical
  roster table before and after. The damage genuinely moved — `trips` deals 50 → **60** a round and
  `cheap-trips` 35 → **30**, so `cheap-trips` now takes four rounds to kill a Giant Rat where it
  took three — but the tool reports **who won, not how fast**, so a posture that still wins and a
  posture that now wins with a quarter of its life left print the same line. Read the unchanged
  table as "no posture crossed the win/lose line", never as "nothing changed", and treat
  multi-sample or rounds-to-kill reporting as the thing the tool needs before this ladder is tuned
  against it.
- **The three axes are a broad buff, measured** *(2026-08-19)*. `tools/balance` goes from **84
  walls out of 96 to 76**, and the roster table moves everywhere rather than at the margin: Clear
  Pod falls to `all-out` in 4 rounds where it took 6, and to `cheap-trips` in 2 where it took 4.
  Almost none of that is the multipliers. Two things did it — a turn's mismatched attacks now
  *sum* instead of the biggest one landing alone, and `cheap-trips` (Bash Bash Bash Strike, four
  crush cards) stopped being a concept Three of a Kind at 200 and became a **form Four of a Kind at
  320**. **Nothing on the enemy side was retuned to absorb it**; the ascent curve and the roster are
  exactly what they were, so this is the size of the shift before any answer to it.
- **The change is a buff at the top and a nerf at the bottom.** A hand of the dearest cards gains
  and a hand of the cheapest loses, because the fixed term it replaced was worth proportionally
  more to small cards. At DMG 10: four Lunges go 130 → **400**, four Jabs 70 → **100**, three Bashes
  35 → **30**. **Nothing in the ladder was retuned to absorb that** — the percents are exactly what
  they were — so the numbers above are the size of the shift before any tuning, and the tuning is
  one file.

### The catalogue's shape

`data/hands.json` holds one list of **nineteen** entries: six rungs on each of three axes, plus the
one High Card they all fall back to. **A hand carries a key, an ID, a name, a `match`, `groups` and
a `multiplier` in percent** — nothing else. `groups` naming *distinct values on the hand's own axis*
is why `[3,2]` is a full house and can never be satisfied by five cards sharing one value.

**`match` is required and never defaulted.** An entry that landed on the wrong axis by omission
would be a balance change nobody made, so a missing or unknown one is refused at init like any other
malformed record. Two further refusals live beside it: a hand wanting more cards than a turn holds,
and one wanting more groups than its axis has values — only three forms and four elements ever reach
a blow, so a four-group form hand is a rung nobody could climb and would otherwise fail silently.

**A hand names one axis, not one per group.** A mixed hand — three ice cards *and* a pair of
Bashes — is deliberately not expressible; reopening it is a schema change and should be argued for
here first.

**Keys carry their axis and the names are long, for now** *(2026-08-19, owner's call)*.
`concept-two-pair`, `form-two-pair`, `element-two-pair`; on screen, **Card Two Pair**, **Form Two
Pair**, **Elemental Two Pair**. The file's word is `concept` and the player's is *Card*, which is
the one deliberate mismatch — a player has never heard of a concept. The longest name is
`ELEMENTAL THREE OF A KIND!`; it measured 1220 pixels of a 1280-wide screen while the name was
shouted at 124 points, and about 790 since the name settled at one size of 80.
`TestTheWidestHandNameFitsTheScreen` is what fails if a name or a type size grows past the screen.

**Adding a rung is one entry.** There is no reward vocabulary to extend, which is the point of the
narrowing: the only things a new entry can say are which axis it counts on, what shape it wants, and
what that shape is worth.

### The multipliers come from how often a hand can actually be built *(2026-08-19)*

The three ladders are **not** the same numbers, because the axes are nowhere near equally hard. The
starting deck is 36 attack cards — **4 per concept, 12 per form, 9 per element** — dealt into a hand
of eight against a 6 AP, 5-card turn, and that arithmetic is what the ladder is priced against
rather than poker's.

Reachability, from a two-million-hand simulation of round one — *can this turn afford some set
forming this rung* — with the multiplier each was given:

| Rung | concept | | form | | element | |
|---|---|---|---|---|---|---|
| | reach | pays | reach | pays | reach | pays |
| Pair | 78.9% | 115 | 99.5% | 110 | 98.4% | 110 |
| Two Pair | 10.8% | 230 | 31.3% | 170 | 23.6% | 185 |
| Three of a Kind | 7.1% | 255 | 60.7% | 180 | 42.3% | 195 |
| Full House | 0.39% | 425 | 2.7% | 310 | 1.1% | 365 |
| Four of a Kind | 0.11% | 500 | 5.2% | 320 | 1.8% | 375 |
| Five of a Kind | — | 745 | 0.036% | 565 | — | 665 |

The rule that produced them: **`100 + 58.6 × ln(1/P)`, floored at 110, rounded to five, then forced
to climb within each ladder.** The constant is set by anchoring the rarest hand in the game — a
concept Four of a Kind, one turn in a thousand — at the 500 it already carried, so the concept
ladder is recognisably the tuned one it was and the other two are priced off the same curve.

**The five-of-a-kind row is the one that is not all measurement** *(2026-08-19)*, and it is written
out because a number that came from somewhere else must not read as one that came from the tool:

- **Form Five of a Kind, 565, is straight off the curve.** 0.036% is one turn in 2,774 — five cards
  of one form is four 1 AP copies plus a 2 AP one, exactly the 6 AP budget — which makes it the
  rarest thing a round-one hand can actually build, rarer than a Card Four of a Kind, and it is
  paid accordingly.
- **Elemental Five of a Kind, 665, is an estimate.** Five cards of one colour cost 7 AP at the
  cheapest, since a colour holds one card per form per tier, so its round-one reachability is
  **zero** and `ln(1/P)` has nothing to work with. It was measured at 8 AP instead — the turn after
  a Prepare — where form five is 0.859% and element five 0.160%, a gap of 98.5 on the curve, and
  that step was added to the form rung's price. `go run ./tools/handodds -ap 8` is the run.
- **Card Five of a Kind, 745, is a judgement.** The starting deck ships four copies of a concept,
  so nothing can deal a fifth: the rung exists for the `duplicate` worm and cannot be measured
  against any deck the tool models. It carries the concept-over-form premium the ladder already
  shows — +75 at trips, +115 at the full house, +180 at four of a kind — applied once more at
  +180 over the form rung. **It is the one number in the table with no probability behind it**, and
  the thing to re-derive first if the worm ever makes five copies common.

Three things fall out of it and are worth keeping:

- **A near-certain hand pays near the identity.** The form and elemental pairs are 98–99% hands, so
  they are a floor rather than a reward — what they buy is the *sum of both cards*, which is
  already most of the change.
- **The narrower axis pays more at every rung**, which is what stops the nesting from making the
  concept ladder dead content: a card hand is always also a form hand, so if the form rung paid the
  same, nobody would ever have a reason to build the narrower one.
  `TestANarrowerAxisPaysMore` holds it.
- **The ladders cross, and that is intended.** A form Three of a Kind (180) pays less than a card
  Two Pair (230) though it uses fewer cards, because it is eight times as easy to build.

`[?]` **Two Pair is rarer than Three of a Kind on the form and element axes** — 31% against 61%,
23% against 42% — because the binding constraint here is cards and AP, not draws: two pair needs
four cards across two values where trips needs three of one. Rarity alone would price two pair
*above* trips and invert the poker ordering the names promise. The ladders are forced to climb
instead, so those two rungs are the least rarity-honest numbers in the table. Fixing it properly
means either accepting the inversion or renaming the rungs.

`[?]` **The best hand is chosen on multiplier, not on what it would deal**, and now that the
multipliers no longer climb with card count across axes, those two can disagree. A turn of
`Jab Jab Jab Cut Cut` has a card Three of a Kind (255, three cards) and a card Two Pair (230, four
cards); the matcher takes the trips at 382 damage where the two pair was worth 460. **This predates
the three axes** — the old ladder had the same hole at 200 against 175 — but it is easier to hit
now. The fix is to pick on the resulting blow and tie-break on multiplier, which is knowable before
resolution and stays deterministic; it is not done, because it is a rules change beyond the axis
work.

**Hand IDs are written in the file, and the hazard is gone** *(2026-08-16)*. An entry's ID used to
be the base in `hands.json` plus the card's enum value, so inserting a card mid-enum shifted every
ID above it — an open question against profile discovery. One entry now covers every concept in the
game, so there is **one ID per catalogue key** and it is written down rather than derived.
Reordering the cards cannot renumber a hand a player has already found. **They are banded by axis**
— 1 for the High Card, 10–15 concept, 20–25 form, 30–35 element — renumbered on 2026-08-19 while no
profile exists to record them. The banding paid for itself the same day: the five-of-a-kind rungs
landed as 15, 25 and 35 without moving a single existing ID. Once a profile does exist, they
freeze.

**Straights are dropped rather than invented** — the concepts have no natural order to be
consecutive in.

What keeps the top of the ladder rare is the deck and the budget: three Strikes is exactly 6 AP,
a starting fighter's entire budget, and **five Strikes is 10 AP**, reachable only by spending a
whole round on Prepares — and five *Strikes* is not even dealable, the deck holding four. That
trade is the hand working as intended, and it is why the five-of-a-kind rungs are the cheapest
cards of a form or a colour rather than of a concept.

### Sequences — the capability the rewrite dropped

**There are no ordered hands, and there is no longer a way to write one.** The schema's `run`
match kind went with the rewrite, so `ice Strike then fire Strike` cannot be expressed at all.

Two entries were recorded here as buildable and are **not**: *Burnt Icecube* (ice then fire,
doubling the DoT) and *Extinguishing Strike* (fire then ice, firing the DoT as one critical hit).
They are kept as a note rather than a table because they are the only argument on record for
order meaning anything, and **order now means nothing anywhere in the game** — see *What this
cost* under Resolution, and the defend section under Cards.

`[?]` **This is the open question the rewrite created.** Either drag-to-reorder is decoration and
should stop being presented as a decision, or something has to read order again. Sequences are
the obvious candidate and would need the run matcher rebuilt against the one-blow model — a
sequence is a shape *within* the hand, not a second blow. Extinguishing Strike additionally needs
a reward that *consumes* a status — and since a hand now pays damage and nothing else, that is a
larger change than it was: it would mean reopening the reward vocabulary this narrowing closed.

### Requirements

- **Hands are rules and live in `internal/combat`**, matching on the resolved cards. The
  screen must never derive one; that is what makes the written account structurally incapable
  of lying about the round.
- **A `KindHand` event** carrying what fired. *Done.* It carries a `HandID`, the multiplier and
  the list of cards that formed the hand, and the screen looks the name up with `HandByID` — so a
  hand renamed is renamed once.
- **The hand event carries its own card list, not a span.** A counted hand is not contiguous —
  Two Pair can be two cards, a card that earned nothing, and two more — so the screen brackets
  what the engine names and never derives it from a pattern length.
- **`KindChilled` counts as a slot in playback** even though nothing happened, or the
  log runs a row short for the rest of the round.
- **A place to browse hands** — a reference the player can return to. Probably belongs with the
  profile rather than inside a duel. `Hands()` exists for it to read.
- **The attack phase writes one line, and it is the hand's.** *Done.* Attack cards no longer draw
  a row each — a turn of five Strikes read as five actions and one figure, which is the model the
  pane was contradicting. The line carries the arithmetic (`20 x 1.5 = 30`) off the event, so
  the sum shown is the sum used, and the damage attaches to it. **The hand dialog now carries the
  same arithmetic at the size of the screen**, spelled out card by card; the line stays because it
  is the record and the dialog is the moment. **What is still not drawn** is a row that a chill
  deleted; that one stands.
- **A preview of the hand while planning is wanted and does not exist.** `BlowFor` is exported
  and is the same function the engine uses, so a previewed hand would be the hand that fires by
  construction. Nothing calls it from the screen yet.

---

## A round is bounded twice

**By cost and by count, independently and on purpose.**

- **AP budget** gates what can be afforded.
- **A hard cap on actions per round** gates how much can happen at all, and holds even when
  everything is free.

**Done.** The cap is five, and it is `Duelist.MaxActions()` beside `ActionPoints()`
rather than the screen's old `maxSelected` constant. It moved for a concrete reason as well as
a tidy one: **the opponent's planner has to obey it exactly as the player's selection does**,
and a cap enforced only by the screen was a cap the enemy ignored.

**The cap is five permanently, and nothing may ever raise it**. This
reverses the reason it was made a method — "so a brand or ring raising it has somewhere to bite"
— and the reversal is the point:

- **A fixed five is what makes hand concepts possible.** Poker hands exist *because* you always
  hold exactly five; that is what lets "a full house" be a permanent, learnable, nameable thing
  rather than a coincidence of how big your hand happened to get. The named five-card shapes are
  built — a Full House wants five cards in a 3-and-2 shape — and every one of them needs the five
  to be a constant the player can plan against for the life of a run.
  The catalogue loader enforces it directly: **a hand asking for more cards than a turn can hold
  is refused at package init.**
- **A growable cap would dilute every shape as it grew.** A Four of a Kind is an all-in commitment
  at a cap of five and routine at a cap of seven. The hands would quietly get cheaper every time
  capacity went up, which is the opposite of a reward for building toward them.
- **It is still a method, and still should be.** Rings and brands need somewhere to bite for
  everything *else* they do, and a method that reads the duelist costs nothing. What changed is
  that this particular lever is off the table: **no ring, brand or hand raises `MaxActions`.**

The consequence for the banking card: a plan cannot buy action slots, so Prepare buys
points instead (+2). An earlier draft of a 4-AP bank granted +2 AP and +2 slots specifically to reach
six- and seven-card hand hands; that is exactly the dilution above and it was cut.

Discounts **can take a card to free**, which is what makes the count bound load-bearing rather
than incidental — and with the cap frozen, a discount ring's ceiling is five free cards rather
than an ever-widening round.

---

## Rings

- **Bought after every fight, with vitae.** *(Built 2026-08-21 — see The shop, below.)*
- **Five at once**, until brands expand capacity. *(ideas.md's "extra fingers bought from a
  shop" is superseded.)*
- **The cap is never displayed.** It surfaces naturally when you try to buy a sixth.
- **No ring changes how many cards can be played.** `MaxActions` is frozen at five — see *A
  round is bounded twice*. A ring may make five cards cheaper, never make it six.
- **Rings are the duelist's only** *(2026-08-17)*. An enemy wears none; affixes are the
  enemy-side counterpart.

### A ring is written in a grammar *(2026-08-17, owner's call)*

**Every ring is data, in a `When` / `If` / `Then` grammar**, and it is **built** *(2026-08-17)*:
`data/rings.json` is written in it, `internal/session` parses it, and `internal/combat/ring.go`
holds the vocabulary and refuses a rule that misuses it. The full vocabulary, the code seat each
moment lands on, and the questions to put to a new ring idea live in
[.claude/skills/rings/SKILL.md](.claude/skills/rings/SKILL.md); this is the argument for the
shape.

**A ring is the only collected thing that is never played.** A card resolves in the turn you
queued it, a worm fires when you pick it, a hand is scored when the attack phase runs — each
already knows *when*. A ring waits, so it says so itself, and that is the third part the card
language does not need.

- **A ring holds a *list* of rules.** Forced by the growing stat rings, which accumulate at one
  moment and apply at another; it generalises to any ring wanting two.
- **`Then` is a list too**, which is what buys a lightning ring that shocks *and* chills with no
  new vocabulary.
- **Seven moments, and only four are in `internal/combat`.** The other three fire in `session`
  and on the post-battle screen, which is what makes a ring a **run** concept the rules consult
  rather than a combat one. `rings.json` is therefore parsed in `internal/session`, beside the
  worms and for the same reason.
- **Rings fire left to right, in worn order.** A determinism rule, not a preference: multiplicative
  effects are order-sensitive, and worn order is the only order the player can see. **Compounding
  is intended** — two slash rings are ×4.
- **A ring may only bend a rule the game already has.** Banker scales vitae propagation, so
  propagation had to be designed first. This is the test to apply to any new ring.

**What the grammar cost, and every item was real work:**

- `Duelist.Rings` was `[ElementCount]bool`, which a form multiplier had no element to be a bit
  under. It is a fixed array of `WornRing` — a `RingID` and its accumulator — plus a count, which is
  the shape the defend set already used and the reason a duelist is still comparable.
- `Duelist.Statuses` was indexed by element and is indexed by **status** — see below.
- **Growing rings hold state**, the first ring thing that does, and the first that must be
  **serialized**: an accumulator on `Session`, keyed by `RingRecord`, which is why the record key
  is the identity rather than an index. **Uncapped, by decision** — a +5 HP ring is +100 by the
  top of the tower and that is the intent. **One numeric effect per growing ring**, so the
  accumulator never has to say which of two it feeds.
- **`tools/balance` still cannot see any of this.** It puts the four elemental rings on its fighter
  and nothing else, so a damage ring, a discount ring or a stat ring changes every number it prints
  and none of them are worn. **A ring's balance is unmeasurable until a worn set is a posture axis** —
  say so rather than guessing at a multiplier.

### Statuses are their own collection, and no longer an element *(2026-08-17, owner's call)*

**A status is data**, in `statuses.json`: a key, a name, a badge, one of four closed effect kinds
(`damage-over-time`, `lose-actions`, `miss-chance`, `damage-reduction`), an amount and a duration.

**Fully decoupled — fire does not burn on its own**, including for the four rings that ship. This
holds the 2026-08-16 position rather than reversing it: the statuses being free is what left
rings with nothing to be, and a *second* fire status arriving on a different ring later is only
possible if the first was never inherent to the colour.

**What it cost, all of it paid on 2026-08-17:** `Duelist.Statuses` re-indexed from element to
status, and its width is now `MaxStatuses` — an array width rather than a design cap, which
registration refuses to grow past because a duelist has to stay comparable. `cards.MaxEffects` is
still 4, but it is 4 *because the file holds four statuses* rather than because there are four
elements, and `TestTheCardHoldsAsManyEffectsAsThereAreStatuses` is what turns authoring a fifth into
a visible layout decision — the badge row fits six at the current pitch, so that is a number rather
than a redesign. `effectKeys` in `card_art.go` was a table keyed by element and is now read straight
off each status record's `Badge`. And `StatusID` is append-only, carrying the hazard `Element` and
`GlyphKind` already carry — with the file, not the enum, deciding the order.

### The rings that are designed

**Every row below is in `data/rings.json` and works** *(2026-08-17)*. **Only the discount and the
flip predate the grammar**; the rest came out of it. **All of them are reachable in a run since
2026-08-21**, bought and sold in the shop — see The shop, below.

**Bulwark** (+25 HP) is the one name still invented rather than taken from this table — Heart is the
skill's own name for the growing one. The discount ring was **Thrifty** until 2026-08-22, when it
became **Warm** and grew three siblings.

| Ring | Moment | Does |
|---|---|---|
| **Burning / Chilling / Shocking / Weighted** | `attack-lands` | the four colours' status rings, split off on 2026-08-22 and priced uncommon |
| **Fire / Ice / Lightning / Earth** | `card-damage` | doubles every card of that colour — *element* multipliers, where Keen/Heavy/Needle are form ones |
| **Storm** | `attack-lands` | lightning shocks *and* chills |
| **Keen / Heavy / Needle** | `card-damage` | doubles **every** slash / crush / stab card in the turn |
| **Striker** | `card-damage` | doubles every Strike — a concept ring, 4 cards where a form covers 12, and priced accordingly |
| **Banker** | `fight-won` | a second +1 vitae per 5 held, on top of propagation |
| **Soul Taker** | `prizes-dealt` | the vitae prize card pays +10 rather than +5. A **flat** +5, not a scaling |
| **Hungry** | `prizes-dealt` | two post-battle choices instead of one |
| **stat rings** | `fight-start` | +10 DMG, +25 HP — and growing variants that gain per fight |
| **Momentum** | `card-damage` + `turn-taken` | every card gains +0.2x DMG per turn with no plan card in it; a plan card wipes the streak |
| **Enflamed / Frostbitten / Lithium / Granite** | `card-damage` + `attack-lands` | their colour gains +0.1x DMG per landed hit of that colour, and keeps it while worn |
| **Echo** | `blow-formed` | the blow's first attack card lands three times: full, 2/3, 1/3 |
| **Flurry / Rend / Aftershock** | `blow-formed` | every stab / slash / crush card lands **twice**, both at full DMG |
| **Atrophy** | `deck-built` | every 3 AP attack is dealt as its 2 AP version |
| **Onslaught** | `card-cost` + `fight-start` | every card 1 AP cheaper, and a quarter off your life — the first ring with a drawback, and the first rare |
| **Warm / Cold / Static / Dirty** | `card-cost` | every card of that colour costs 1 AP less — one per colour |
| **flip x12** | `deck-built` | recolours every card of one colour as another — one for each ordered pair; see below |

**A concept ring and a form ring are not the same object** and must not be priced as one.
Striker covers 4 cards, Keen covers 12.

### Momentum — a streak that belongs to the duel *(2026-08-22, owner's call)*

**Every card gains +0.2x DMG for each turn played without a plan card, and a plan card wipes it.**
Uncommon. It scales the *duelist* rather than a colour or a form: the `card-damage` rule carries no
predicate at all, so the streak is worth the same on every card in the hand.

- **It is written as two positive rules and no negation.** One grows on every turn, one resets on a
  turn holding a plan card, and **growth is applied before resets** — so a planning turn nets zero
  rather than depending on which rule the file lists first. The grammar has no `not` and this is the
  shape that means it does not need one.
- **`turn-taken` is a new moment**, the first that is about a *turn* rather than a card, a blow or a
  fight. Its predicate is matched against the turn as a whole: the rule fires when any card of the
  turn matches it.
- **An empty turn is still a turn taken**, so a duelist chilled out of their whole turn keeps
  building. The streak is about not *planning*, not about swinging.
- **A duelist who falls mid-turn never reaches it**, since `playTurn` returns early on a death — a
  streak is a fact about turns taken and a corpse takes none.
- **The streak does not survive the fight**, and that needed a rule: `combat.KeepsGrowth` reports
  false for any ring holding a `reset-growth`, and `Session.AbsorbGrowth` skips it. Otherwise one
  good duel would bank a permanent bonus that a single plan card had once wiped.
- **It is a real argument against Plan and Prepare**, which is the interesting part: the deck's plan
  cards are how a hand is rebuilt, and this ring prices that. Whether 0.2x a turn is enough to make
  a player skip a Plan is unmeasured — `tools/balance` wears the status rings only.

### The Enflamed family — growth inside a fight *(2026-08-22, owner's call)*

**Enflamed (fire), Frostbitten (ice), Lithium (lightning), Granite (earth)**: their colour gains
**+0.1x DMG every time an attack of that colour lands**, and keeps it for as long as the ring is
worn. Uncommon.

**They are the first accumulator that moves during a fight.** Heart and the growing stat rings step
once per win, at `fight-won` — `grow-on-win`, renamed from `grow` on 2026-08-22 so both growth verbs
name their moment; these step at `attack-lands`, so the second fire attack of a duel is
already stronger than the first. That needed a second verb — `grow-on-hit` — because a verb belongs
to exactly one moment, and it needed a way home: combat grows the *duelist's* copy of the
accumulator, and `Session.AbsorbGrowth` reads it back on the win, before the screen throws that
duelist away.

- **Once per hit** *(owner's call, 2026-08-22)*, where a status is once per blow. Two fire cards in
  a hand are two steps, and a fire card that Echo seats three times is three — it counts **landings**
  rather than cards. **That is the combination it exists for**: the rings that multiply landings and
  the rings that grow per landing are meant to compound into a build, not to politely ignore each
  other. Echo plus Enflamed is +0.3x off one card.
- **A blow is paid for after it lands, never during** *(owner's call, 2026-08-22)*. The four fire
  cards of a Four of a Kind all hit at the ring's old strength and the +0.4x shows up on the next
  fire attack. A ring that strengthened the blow that grew it would mean the first attack of a fight
  already wearing its own bonus.
- **The growth is linear, and deliberately.** The step reads the effect's raw `Amount`, never
  `Amount + Grown` — a growth that grew would compound, and no growing ring in the game does.
- **A lost fight forfeits what it earned**, which needs no rule of its own: a defeat ends the run.
  **Selling forfeits it too**, by the shop's existing rule.
- **This is the first ring state that changes mid-fight**, so it is also the first thing a mid-fight
  save would have to write down. Nothing saves yet.
- **Uncapped, like every other accumulator.** +0.1x a blow across a long fight is a big number by
  the top of the tower, and nothing measures it — `tools/balance` wears the status rings only.

### Atrophy, and the ladder as a ring *(2026-08-22, owner's call)*

**Every 3 AP attack is dealt as its 2 AP version**: Lunge becomes Thrust, Cleave becomes Slash,
Smash becomes Strike. Rare.

**It is a `deck-built` swap, so it is the flip's shape applied to the other axis.** A flip changes a
card's colour as the fight's deck is dealt; Atrophy changes its *concept*, one rung down the same
form's ladder. Everything downstream — cost, damage, the hand it forms, the card face — follows
because the card genuinely is a Thrust.

- **What the player buys is a turn with more cards in it.** Three Lunges cost 9 AP and do not fit a
  6 AP turn; three Thrusts cost 6 and do. It trades damage per card for cards per turn, which is a
  hand-ladder decision rather than a damage one — a Three of a Kind of Thrusts against one Lunge and
  a Jab.
- **`combat.Neighbour` already existed**, built for worms, so the ladder is still a consequence of
  `duelist_cards.json` rather than a table written twice. A card with no rung below it is left alone.
- **`Tier` is a new predicate and it reads the *declared* cost.** A discount ring cannot move a card
  out of Atrophy's reach, which would otherwise make two worn rings switch each other off in an
  order nobody chose.
- **Two demoting rings do not chain.** `DemoteConcept` reads the card the run owns and takes the
  deepest single step, exactly as flips read the original element. A ring wanting two rungs says
  `Amount: 2`.
- **Nothing measures it**, and this one is the most likely of the new rings to be badly priced:
  `tools/handodds` measures which hands a deck can reach, and Atrophy changes that deck.

### Echo, and the one blow a turn *(2026-08-22, owner's call)*

**The Echo Ring makes the blow's first attack card land three times — full DMG, two thirds, one
third.** Uncommon.

**It is extra *terms in the sum*, not extra blows**, and that is the decision the ring forced. A
turn lands one blow: every attack card is added up, the hand multiplies the total, and the result
lands once. Three separate landings would have meant three misses to roll, three sets of defends and
three status applications — a second shape for a round. So an echo seats the lead card again behind
itself at a smaller figure, and the turn reads as *seven cards played, the first of them three
times*.

- **The echo never reaches the matcher.** `blowFor` has already run when `handEvent` adds the
  echoes, so an echoed Strike does not turn a Pair into Three of a Kind. It pays into the hand the
  real cards formed — which is also what stops one ring rewriting the hand ladder.
- **The multiplier therefore multiplies the echoes too**, since they are in the base sum. Echo is
  worth about two thirds of the lead card, times the hand — strongest in a big hand, which is the
  opposite of a ring that rescues a bad one.
- **The player watches it happen.** The echo terms are seated on the lead card, so the sum in the
  math box shows the first card's figure paying three times and each one flies out of that card.
  That was the owner's requirement, not a side effect.
- **`blow-formed` is a new moment and `echo-attack` a new verb**, and this is the first moment that
  sees a *blow* rather than a card. `MaxEchoLandings` is 5 — a width on `Event`'s hand arrays,
  which have to stay fixed for an Event to be comparable.
- **Two echo rings add landings rather than multiplying**: three and three is five, not nine.
- **A seven-term sum is wider than the box was laid out for**, so `layOutMath` now shrinks a line
  that will not fit rather than letting a figure hang off the band.
- **Nothing measures it.** `tools/balance` wears only the status rings.

**Flurry, Rend and Aftershock repeat a whole form** *(2026-08-22, owner's call)*: every stab / slash
/ crush card in the blow lands **twice, both at full damage**, uncommon. `repeat-card` is the second
verb at this moment and it is the one that does *not* diminish — an echo is a card ringing on, a
repeat is the card played again.

- **They deal exactly what Keen, Heavy and Needle deal today, and cost a tier more.** Two
  full-strength landings and one doubled landing are the same figure inside the same sum. What the
  repeat buys is **two hits instead of one**, and nothing in the game currently pays for a hit —
  statuses land once per blow and are deduplicated. **So these three are, right now, a worse buy
  than the commons they sit beside.** They are written for the hit mechanics that are coming, and
  that is a deliberate bet rather than an oversight.
- **`Lead` is a new predicate, and the only one that is not a fact about the card.** It is what lets
  one pair of verbs cover both scopes: Echo says `{"Lead": true}`, a repeat says `{"Form": "crush"}`.
  A rule setting `Lead` at any other moment is refused at load, since no other moment knows which
  card leads.
- **Repeats resolve before echoes** when a card has both, because an echo of a repeated card would
  be an echo of something that already happened twice.
- **The event's term arrays widened from 9 to 25** — every card of a legal turn, each landing up to
  `MaxEchoLandings` times. A repeat matches on form, so five crush cards is five cards landing
  twice, where an echo only ever touched one card.

### The discount rings — one per colour *(2026-08-22, owner's call)*

**Warm, Cold, Static and Dirty**: every card of one colour costs **1 AP less**, at `card-cost`. All
four common. The discount was one ring named Thrifty, matching fire and named after nothing in the
game; naming it for the *colour it warms* generalises to four and drops a word the design never
owned.

**They are the third thing a colour ring can be**, after the damage doubler and the status ring, and
the one that changes what a turn can hold rather than what it does: a 6 AP budget buying four cheap
cards instead of three is a different hand ladder, not a bigger number. **Nothing measures that** —
`tools/balance` wears the status rings and nothing else — so what a colour's discount is worth
against a colour's doubling is unknown, and reads as the bigger of the two.

**`static-ring` is the lightning discount and not the earth→lightning flip.** The flip held that key
for a few hours on 2026-08-22 and is now **Dust Storm**; the record key moved with the name.

### The colour rings

`data/rings.json` holds them, each as one `attack-lands` rule matching one colour and applying one
status. **There is no special case for them in the engine** *(2026-08-17)* — they are the plainest
thing the grammar can say, which is what the grammar was checked against. One ring is one element,
so wearing one and swinging a hand of all four colours lands one status and nothing else — which is
what makes the second and third worth buying.

| Ring | Element | What wearing it does |
|---|---|---|
| Burning Ring | fire | your fire attacks burn: 50% of your DMG at the end of each round |
| Chilling Ring | ice | your ice attacks chill: one card off the front of each of their turns |
| Shocking Ring | lightning | your lightning attacks shock: 25% chance their attack misses |
| Weighted Ring | earth | your earth attacks weigh: they deal 25% less damage |

### The flip rings — one for every ordered pair *(2026-08-22, owner's call)*

**Twelve rings, each `deck-built` / one colour in / another colour out.** Frozen Lightning was the
only one for five days; the pattern generalised to every ordered pair of the four colours, and all
twelve are **common**. The names are thematic rather than mechanical — "Permafrost" says earth into
ice without saying either word — which is a deliberate cost: the *card* has to be read to know what
it does, and the tooltip is what says it.

| dealt as → | from fire | from ice | from lightning | from earth |
|---|---|---|---|---|
| **fire** | — | Meltdown | Firestorm | Magma |
| **ice** | Frostbite | — | Frozen Lightning | Permafrost |
| **lightning** | Heat Lightning | Thundersnow | — | Dust Storm |
| **earth** | Obsidian | Glacier | Fulgurite | — |

- **A flip is what makes a colour ring worth wearing**, which is the whole point of the pair: Fire
  Ring doubles fire cards and there are only so many, so Frostbite-and-friends is how a deck is bent
  toward the colour a run has bought into. It is also how the *status* rings get fed.
- **Flips do not compose**, and that is enforced rather than emergent — `combat.FlipElement` reads
  each card's **original** element, so Frostbite (fire→ice) and Meltdown (ice→fire) worn together do
  not cascade a deck to one colour. See `TestFlipsDoNotCompose`.
- **Two flips naming the same source is the new case the twelve introduce, and last-worn wins**
  *(owner's call, 2026-08-22)*. Frostbite and Heat Lightning both claim fire; the later ring in the
  row takes it, by the same rule that orders every other multiplicative effect. Decided rather than
  merely observed — nothing warns the player and there is still no way to reorder the row, and both
  of those are accepted.
- **They are twelve of thirty-two records, and the dilution is accepted** *(owner's call,
  2026-08-22)*. The catalogue is now more than a third flips, so a common ring's ten tickets are ten
  out of a much bigger pot than they were at seventeen rings — and more rings are coming, which is
  what makes that fine. If the shelf ever does need thinning, the lever is a weight or a tier, not
  a price.

**Every colour is two rings as of 2026-08-22** *(owner's call)*. **Fire, Ice, Lightning and Earth**
are now `card-damage` doublers on their colour — the first *element* multipliers, where Keen, Heavy
and Needle multiply a form — each common and each keeping the colour's artwork. The status each used
to apply moved to a second ring — **Burning, Chilling, Shocking, Weighted** — all uncommon and all
drawing the default ring face. So a colour offers cheap damage or a dearer, rarer status, and eight
of the seventeen records are now colour rings.

**Two records and two files were renamed with it**: `frozen-ring` → `ice-ring` and `thunder-ring` →
`lightning-ring`, with `assets/ring/frozen-ring.png` → `ice-ring.png` and `thunder-ring.png` →
`lightning-ring.png`. The ring naming now matches the element names the rules use. "Frozen" and
"Thunder" are free again and may come back for something else.

**`tools/balance`'s elemental postures wear the *status* halves**, not the colour rings — what they
measure is what a status does, so the list follows the rule rather than the name.

**BURNING went from 10% to 50% of the attacker's DMG in the same call**, over the same two rounds.
That is a fivefold change to a status nothing measures — `tools/balance` plays postures and knows
nothing about rings — so it is a judgement, and a large one: at 50% over two rounds a burn is
roughly a whole extra attack, which is what the uncommon tier is meant to be paying for.

**The ring is read off the attacker, never the victim.** Your fire ring makes *your* fire attacks
burn; it does nothing about fire aimed at you. The alternative would make a ring a liability and
buying one a decision with a wrong answer.

**A run opens wearing nothing** *(owner's call, 2026-08-21)*. It wore fire, ice and lightning for
four days, which was always written down as temporary: the list existed because a ring could not
otherwise be got at all. `session.StartingRings` stays as the seat for putting one on without
playing to a shop, and ships empty. The worn set moved off the combat screen and onto the run on
2026-08-17, which is what makes a bought ring survive a fight.

**What that costs, stated rather than discovered:** a run holds 5 vitae and a base ring is 3, so
**the bare opening lasts exactly one fight** — the first shop can already afford a colour, and the
first duel is the only one fought with every element inert. That is a much shorter gap than the
first pricing draft produced, and it is the deliberate consequence of a base ring being cheap.

### The shop *(2026-08-21, owner's call; built the same day)*

**Three rings on a shelf after every fight, and the row you are wearing under them.** Both rows are
ring cards and both are clicked; the difference is which way the vitae moves. `internal/screens/shop.go`
draws it, `internal/session/shop.go` holds the rules, and neither knows what comes after the shop —
`session.PhaseShop` is a station of the run loop and `advanceRun` is what leaves.

- **A ring declares a rarity, and the rarity is the price** *(owner's call, 2026-08-22)*. This
  replaced a per-ring price. `rings.json` names one of three tiers and `data.Rarity` turns it into
  both what the shop charges and how often the shelf offers it:

  | Rarity | Price | Sells for | Draw weight |
  |---|---|---|---|
  | common | 3 | 1 | 10 |
  | uncommon | 5 | 2 | 4 |
  | rare | 7 | 3 | 1 |

  **A common ring is still 3, the base**, so the pacing below is unchanged: one of the four that
  give a colour its status is the plainest thing the grammar can say and the thing everything else
  is read against.
- **Three tiers rather than seventeen numbers.** A per-ring price could only be judged one ring at a
  time and was drifting; a tier can be read against the whole catalogue at a glance, and rebalancing
  a ring is moving it rather than inventing a figure. **What that costs, said out loud:** two rings
  in the same tier cost the same even when one is plainly stronger — the answer to that is which
  tier it belongs in, not a fourth tier.
- **Scarcity and price are deliberately different curves.** A rare ring is a tenth as likely to
  appear as a common one but only a bit over twice the price. The 3 / 5 / 7 ladder *(owner's call,
  2026-08-22)* spans exactly what the seventeen hand-written prices used to. What makes it rare is that a run mostly
  does not see it; a price tracking the odds would make it unbuyable on the one visit it turns up.
- **Every ring is `common` as of 2026-08-22**, which is the migration's starting position and not a
  judgement about any of them. The tiers are assigned by hand.
- **That is a full ring or two a fight against an income of roughly 5–10**, so **the purse stops
  binding once the five fingers are full** — around the fourth or fifth fight, after which vitae has
  nothing to buy but swaps. The first draft priced a base ring at 20 and made the whole run about
  affording one; this is deliberately the other side of that, and what it wants next is something
  else to spend on rather than dearer rings.
- **Nothing measures whether any of those numbers is right.** `tools/balance` plays postures against
  the roster and knows nothing about rings, so what a doubling of every slash card is worth in vitae
  is a judgement. Recorded as a judgement rather than dressed up as a derivation.
- **Selling pays the tier's own figure — 1, 2 or 3** *(owner's call, 2026-08-22)*. It was a quarter
  of the price rounded up, which across three prices paid an uncommon and a rare the same 2: three
  tiers is three numbers, and writing them down beats arithmetic that has to be argued with. The
  round trip still loses, and loses more the dearer the ring — a shelf you could try on for free
  would be a rerolling of your hand every visit rather than a decision.
- **Selling is the only way a ring comes off, and it is how the sixth ring is bought.** A purchase at
  five worn is refused rather than swapped, so trading is two decisions with a price between them —
  never one click that quietly throws a worn ring away.
- **A sold ring's accumulator resets to zero.** `Session.grown` is keyed by record precisely so a
  ring taken off and put back on is the *same ring*; the decision is that it is not the same
  *number*. The growth is what wearing it through fights paid for, so selling forfeits it. It is
  what stops a Heart Ring being parked in the shop between fights.
- **What is already worn is off the shelf**, rather than shown and refused: a seat spent saying
  nothing.
- **Selling out of the middle of the row changes the firing order**, since rings fire left to right
  and a re-bought one goes on at the right-hand end. That is a real cost of letting a ring come off
  and there is no re-ordering control yet — see TODO.md.
- **The shelf is its own random stream** (`seeds.ShopStock`), per fight, so a defeat and a retry walk
  into the same shop exactly as they meet the same opponent. **It is three weighted draws without
  replacement**, rather than a shuffle: each seat draws on rarity tickets and the drawn ring leaves
  the pool, so the shelf never offers the same ring twice.

### Flip rings — the element-transform ring

**A ring that maps one element onto another across the whole deck**. A
"frozen lightning" ring turns every lightning card into an ice card, so a deck holding 12 of
each now holds 24 ice and no lightning.

**This is the ring the five-of-an-element hand needs to exist.** At 12 cards per element in a
60-card deck, drawing five of one in a hand of eight is a fluke you cannot build toward. A flip
doubles the pool and turns the hand into a deck you assembled — which is the whole stated
purpose of hands.

- **It is deliberately more powerful than a discount ring**, and that is accepted. The primary
  engine-building of this game is the interaction of rings, brands and how the deck has been
  altered; rings having very different magnitudes is what makes mixing them an act of judgement
  rather than an ordering.
- **It is the same primitive as an enemy affix.** An affix already maps `basic → fire` across an
  enemy's deck (see *Enemies*). One transform mechanism, pointed at either side — build it once.
- **It is cheaper to implement than the discount ring.** A flip is a pure transform on a card's
  element and never touches `Cost()`, so it does not require the "cost becomes a property of the
  pairing" rewrite. It still needs element to reach `internal/combat` for the hand to *match* on
  it.
- **Flips do not compose.** Every flip reads the card's *original* element, so lightning→ice and
  fire→ice both land on their own sources and cannot chain. Without that rule two flips could
  cascade a deck to a single colour and the order they were bought in would change the result.
- `[?]` Whether a flip's source element can be one that another equipped flip targets — allowed
  under the no-compose rule above, but it means two rings can both feed the same colour, which is
  a 36-of-60 monochrome deck. Watch it before deciding whether the cap is the ring slots or a
  rule.

**Discounting matching cards is built** *(2026-08-17)*. Cost is a property of the **pairing**:
`Duelist.CardCost` and `Duelist.CostOf` are what the engine, the planner, the AP bar and the
over-budget check all read, and `Card.Cost()` is only the card's own printed figure. Every card face
on every screen is drawn at the wearer's cost — the hand, the deck overlay, the flights, and the
post-battle offer through `session.CardCost` — because a card showing three dashes while the budget
charges two is the screen contradicting the engine. **An enemy's queued card is priced by the
enemy**, never by the player's rings.

**Rings are drawn as cards**, in a horizontal row across the top, not necessarily spanning the
whole bar. Same size as other cards, and **no glyphs**.

**The row is on the screen at full card size, and it is what the player is actually wearing**
*(2026-08-16)*. It drew every record in `data/rings.json` up to the cap while nothing read a
ring; it now reads what the run is wearing, because a catalogue that equips itself would have put
the earth ring on the moment the file gained a fourth entry. **The row starts empty and fills as
rings are bought** — see The shop. What made the row possible is that **the vertical problem below solved itself**: the full-height Resolution pane
left the 12–46% band for a three-line feed above the hand, so the band the row needed was
already empty. The character block shrank into the top-left corner to give it the width.

- **The row spreads to fill its width and closes up as it fills**: first card flush left, last
  flush right, so three rings stand well apart and five sit shoulder to shoulder. The row runs
  to 79% because the enemy card is past 81%, which leaves about 825px — and five cards is 810
  since **the whole card set is 162x224** (a tenth off the width, 15%
  off the height, at the owner's request). At the old 180 it overlapped by 26px each, and it
  will again if the row ever narrows. Overlap rather than shrink is the accepted answer, the
  same idiom as the deck overlay — a card cannot be scaled, so a smaller ring is a different
  drawing and there is no smaller ring style.
- **The row is not a box**. No fill, no frame, no title: a pink panel around five
  pink-bordered cards read as cards trapped in a container. What marks it is a **rule under the
  cards**, the width of the row, with the fraction on its right end — and the cards align flush
  with the top of the character block rather than each sitting inset in a frame.
- **It fits vertically**, for the reason above: the full-height Resolution pane's departure
  is what freed the band.
- Dropping the glyphs is exactly what *would* free the height: the glyph column is the floor on
  card size. `[?]` Same width and shorter is the escape if the band is ever wanted back.
- `internal/cards` draws a ring already — `RingStyle`, pink border, artwork instead of glyphs —
  so the drawer question is answered: rings share the frame and colour logic and skip the glyph
  column. The screen builds it through `ringSpec`.

**The cap is written down after all, as a fraction**. `worn/5` sits on the
right-hand end of the rule under the row, the same shape as the deck pile's `left/owned` and the
AP figure. That **softens "the cap is never displayed"** above, deliberately: a number in a corner
is read without being looked for, where five slots of which two are empty frames would spend the
loudest thing in the pane on what you have not got.

**Rings are the first thing that genuinely needs `Session`.** They survive fights;
`CombatScene` state does not. The sketch dodges this by equipping everything defined, up to the
cap, every time the screen initialises.

Art note: `fire-ring.png`, `ice-ring.png`, `lightning-ring.png` and `earth-ring.png` are embedded
and drawn on the four colour rings. The four status rings have no art and draw `default-ring.png`.
Earth has none, so a fourth ring is art before it is data.

---

---

## Worms — altering the deck between fights

**A worm is a change to a card you already own.** It recolours it, removes it, or copies it. It
never invents one, and that restriction is the whole safety property: the *concept* is never
touched, so nothing a worm produces can be a card `internal/combat` has not registered.

**Offered after a won fight, on the post-battle screen.** Two worms are drawn from the catalogue
and shown as cards; pick one, pick the card it takes, **see what it would become**, and confirm.

| | |
|---|---|
| When | after a won fight, before the next room |
| Offered | **three cards**: two worms from `data/worms.json`, and **+5 Vitae** |
| Then | a hand dealt fresh off the whole run deck, at the combat hand size |
| Then | a **morph**: the card before and after, side by side, nothing committed |
| Choice | one prize, and — if it is a worm — one card |

- **Worm first, card second, and it turned round to get there** *(2026-08-17)*. The first version
  asked for a card and then offered verbs on it, which made the reward look like a property of the
  card. **What you were given for winning has to be the thing on the screen when you arrive.**
- **Two options, so the choice is a comparison.** One is an instruction; three is a menu to read
  rather than a decision to make. They are distinct by construction — the offer shuffles the
  catalogue rather than drawing from it twice.
- **The card offer is dealt off the *whole deck*, not off what the fight left in the piles.** A
  reward is about what you own.
- **Two random streams, and they are separate on purpose.** Which worms and which cards are
  drawn from different lists and change on different schedules: sharing would mean adding a worm
  to the catalogue silently rerolled which cards every fight of every run offered.
- **Nothing is committed until the take.** Back steps out of the morph to the cards and out of the
  cards to the worms, so a player who picked up the wrong worm or aimed it at the wrong card is
  never stuck with either.
- **The preview runs the real worm against a throwaway copy of the run**, never a second
  implementation of what each target does. A preview with its own arithmetic is a preview that can
  disagree with the thing it previews.
- **The result flies to the middle and is held there.** A card won and immediately lost into a
  deck of forty-eight is a reward the player never sees, which is the same reason the morph exists
  — and it *travels* rather than appearing, per CLAUDE.md's rule that cards always do. The hold
  does not start until it lands, so a slower flight is a longer look rather than a card arriving
  late to a countdown already running. **A removal has nothing to fly**: what was won is an
  absence, so the empty seat is drawn and nothing crosses the screen.
- **There is no decline, because the money is a card** *(2026-08-17)*. Declining used to be a
  button off to one side, which made it the odd thing out on a screen otherwise made of cards.
  Taking vitae instead of altering the deck is now a **choice among three with a price**, not an
  exit — and it means every path through the screen is the same path: **pick a card, watch it fly
  to the middle.**
- **The vitae card's seat never moves.** It is appended last rather than shuffled in, so a player
  who has learned where the money is can take it without reading.
- **Run-scoped, never persisted.** Two runs from the same seed may hold different decks, because
  an alteration is a *choice*: replay is a seed plus a choice log. See the `randomness` skill.

### The grammar: a target and a new value

A worm record is the card language pointed at a card that already exists — see `data/worms.json`
and `internal/session/worm.go`, which is where a record is validated.

| Target | Value | What it does |
|---|---|---|
| `element` | a colour | recolours one card |
| `remove` | — | takes one card out of the run |
| `duplicate` | — | puts a second copy of one card into the run |
| `cost` | a signed delta | changes what one card costs |
| `amount` | a percentage | scales one card's figure, whatever that figure is |
| `promote` | — | one rung up its form's ladder: Jab → Strike → Smash |
| `demote` | — | one rung down: cheaper and weaker |

**The vocabulary is closed**, the same posture the card verbs take: a new target is a Go change
plus one place applying it, never something a JSON file can assert into existence. A bad record —
an unknown target, an element the rules lack, a value on a target that reads none — **panics at
init**.

**`amount` reaches every card in the deck with one worm**, because what the figure *is* depends on
the verb: a defence percentage, points banked, cards drawn, or a damage multiplier. That is the
card language paying off — one worm, four meanings, no special cases.

**Cost and amount are per-card and the rest of a card is not.** `combat.Card` carries `CostDelta`
and `AmountPct`; `Cost()`, `Amount()` and `Damage()` are the three methods that read them, which is
where the bounds live. **Form and label are still concept-wide**, so a worm reaching for one of
those would change every copy of that card in the deck — that is the argument to make again from
scratch before adding one.

### The bounds

| | Floor | Ceiling |
|---|---|---|
| Cost | **0** | none declared; concepts run 1–3 |
| Amount | 1 | a defence is clamped **under 100** |

- **A card may be driven to 0 AP** *(owner's call, 2026-08-17)*, and that moves the game onto its
  other bound: a round is capped by cost *and* by count independently, so a free card is limited by
  `MaxActions` rather than by the budget. Taken with that in view, not as an oversight.
- **Nothing stops a blow outright, however many worms are stacked on a Defend.** `RegisterConcept`
  refuses a *concept* declaring 100 or more; `Card.Amount` clamps a *modified* one. It clamps
  rather than refusing, because a reward that silently did nothing is worse than one that hits its
  ceiling.
- **`amount` compounds rather than replaces** — 150% twice is 225% — so a second worm on the same
  card is worth taking.
- **A ladder has two ends.** A Smash cannot be promoted and a Jab cannot be demoted, and a plan
  card has no form and so no ladder at all. The screen asks `CanApply` before offering a card, so
  a worm that would do nothing is never presented as a choice.

### The card says what the card does

**Effect text reads the card, not the concept** *(2026-08-17)*. It was already a template over the
value; what changed is which value it reads. So an altered Defend prints the percentage it
actually cuts and an altered Prepare prints what it actually banks. **A card whose face disagreed
with its behaviour is the worst thing an alteration mechanic can produce**, and it is the reason
this was not deferred.

**A worm may not recolour a card to basic.** That would be a way to *lose* a colour rather than
choose one, and no attack card in the deck is drab.

### REMOVE is the strongest option, and that is accepted

Thinning a 48-card deck against a fixed hand of eight raises consistency every time. It is
deliberate rather than unnoticed: the offer is two worms out of a growing catalogue, so removal
being the best of what is on the table is only sometimes the question. **`duplicate` is the one
most likely to need a cost** — copies are the sharpest dial in the game, since four of one concept
in a turn is a Four of a Kind.

### `[?]` The deck overlay stops being able to show the deck

Rows in the deck panel cap at twelve, and `element` worms *migrate* cards between colour rows — so
building toward fire pushes that row past the cap and the panel starts hiding exactly what was
built. **Owner's call (2026-08-17): defer.** The replacement is not a bigger cap: the panel wants
counts — attacks, plans, how many of each colour — rather than every card drawn at once.

### What it needed that did not exist

**The run.** `internal/session` holds the deck now, because the combat screen rebuilds its piles
on every `Init` and `Init` is how the next fight starts — so anything held on that scene is thrown
away between rooms. It is the same hole rings, vitae and brands are blocked on.

**No card identity, and that is a consequence of *when* alteration happens.** Between fights no
pile is live, so an offer is a list of positions in the run deck and a position is unambiguous for
as long as the screen is up. Mid-fight alteration would need a real ID *and* a field on every
event, since the log rebuilds a card from what an event carries.

### The between-fight chain

Post-battle is the first of several scenes between one room and the next: **alteration**, then a
**shop** where vitae is spent, then a **room or stairway choice** between two doors. Each is an
ordinary scene in the registry rather than a mode of the combat screen. What is missing is
whatever decides the order they run in — today the chain is hard-wired, win → post-battle →
combat.

---

## Brands

**Brands alter the container; rings alter the contents**. That is the
axis, and it is what tells you which of the two a new power belongs to:

| | Brands | Rings |
|---|---|---|
| What they touch | the chassis — hand size, total discards per round, ring slots | the cards — elements, costs, and the stats that feed them |
| Removable | **never.** You brand yourself and you do not take it off | freely; five equipped, swap as you like |
| Scope | **for the run** | for the run, but re-chosen after every fight |

- **"Permanent" means for the run, not across runs**. A brand is a
  commitment made *inside* a run that cannot be undone, which is a different thing from
  meta-progression. Hand *discovery* is the profile-scoped mechanic; brands are not.
- **More actions is struck from the list.** Brands were previously recorded as granting "more
  actions"; the action cap is now permanently five and nothing raises it. A brand growing hand
  size is the nearest legal thing and is a container change, so it fits the axis.
- Otherwise still open — capacity and rule-bending, with the above as the test for what counts.

Like rings, they have **concrete definitions that never really change**, which makes them a fit
for the `data/` pattern: JSON beside a small Go loader.

---

## Vitae

The currency. Earned from fights, spent on rings. `Session` carries the purse; the post-battle
screen's third prize card and propagation are what add to it, and **the shop is what takes it out**
*(2026-08-21)* — `Session.SpendVitae` is the one place a purse goes down, and `Session.Buy` is its
only caller.

### Propagation — vitae earns interest *(2026-08-17, owner's call; built the same day)*

**After every fight, a run gains +1 vitae for every 5 it is already holding**, capped at **+5**.
So 5 held pays 1, 10 pays 2, and 25 pays the maximum 5 — holding more than 25 propagates no
faster.

- **It is a rule of the run, not a ring.** The Banker ring scales it, which means it has to exist
  on its own first — a ring may only ever bend a rule the game already has.
- **The cap is what stops it running away.** Uncapped, +1 per 5 is roughly ×1.2 a purse per
  fight, which compounds across 24 fights into a number no shop can be priced against. Capping
  the *rate* rather than the purse leaves a big purse worth having and stops the curve.
- **Rounded down**, like every other integer rule in the game.
- **The cap binds the base rate, and a ring scales what the cap produced** *(2026-08-17, owner's
  call)*. So at 25 held propagation is +5, and +10 wearing Banker. The alternative — an absolute
  cap on the figure that finally lands — would make Banker do nothing past 25 held, which is a
  ring that stops working exactly when a run can afford it.
- **Order of operations, therefore:** count the fives, clamp to +5, *then* apply every ring that
  scales propagation, left to right in worn order. Two such rings compound, like every other
  ring effect.
- **It happens in `Session.WonFight`**, before the room counter moves and before the post-battle
  screen deals its prizes: propagation is interest on what the run walked out of the fight holding,
  not on what the prize card is about to pay it.

---

## The tower

**8 floors × 3 fights.** Fixed layout, drawing no randomness — what is *in* it is random, the
shape is not. *(ideas.md's "one enemy per level" is superseded.)*

- **Every third fight is a floor boss.** Bosses are durable — high `HP`, and earth on them if
  a boss should also blunt damage — with one strong attribute. They cannot spawn enemies, which
  implies normal enemies can, a mechanic recorded nowhere else and otherwise undefined.
- **After fights 1 and 2: a choice of two doors.** After the boss: **a choice of stairwell.**
  Captured as two concepts even though the mechanic is likely the same, because one is "next
  fight on this floor" and the other is "next floor" — a real difference to hang divergence on.
- **Doors hint at what is behind them.** Cold coming off the door for an ice enemy, smoke for
  fire. This is graded reveal one level up from `concealedLabel`, and probably wants the same
  answer as the in-combat version.
- **Generate both doors, always.** Rolling only the chosen one shifts every subsequent draw in
  the run.

### The ascent curve

**Every room grows the opponent's HP and DMG by 10%, compounding** *(2026-08-17, owner's call)*.
Floor 1's outer room is the baseline and takes a record's stats unchanged; each fight after it is
10% harder than the one before. `pyramid.AscentGrowthPct` is the number, `pyramid.ScaleToFight`
is the arithmetic, and `entities.NewEnemyFrom` takes the fight index so an unscaled opponent cannot
be built by accident.

- **It compounds per *room*, not per floor.** A floor is three rooms, so a floor costs about a
  third more than the one below it and the stairway boss is harder than the inner room beside it.
- **HP and DMG only. `Actions` is left alone**, because it is the budget a *deck* is spent out of:
  growing it hands an opponent more cards rather than a harder version of its own. It stays a
  per-enemy dial, authored deliberately.
- **Integer arithmetic, and the multiplier compounds rather than the stat.** The obvious version —
  `v = v * 110 / 100` once per room — freezes every stat below 10, because integer division
  truncates `5 * 110 / 100` straight back to 5. Half the roster opens on DMG 5 or 6, so the curve
  would have done nothing to exactly the band it was added for. A fixed-point multiplier truncated
  once at the end is what fixes it, and `TestASmallStatStillClimbs` is what caught it.
- **No `math.Pow`.** A float power is not reliably identical across two machines and a stat feeds a
  duel meant to be replayable from a seed — the same rule that keeps `math/rand` out of the game.
- **Nothing caps the fight index.** The fight order is the whole 96-record roster standing in for a
  generator, so playing far enough asks for numbers the 8-floor tower never would.

**It doubles a curve that is already in the data, and that is deliberate but worth stating.**
`ValidFloors` already sorts the roster from an 80 HP / DMG 5 Giant Bat to a 400 HP / DMG 22
Bio-Titan, roughly a fivefold climb; the ascent curve multiplies on top of that, reaching about
×8.9 by floor 8's stairway. The measured cost is in the balance table below.

`[?]` Whether the curve should be flatter now that it stacks on the roster's own progression, or
whether the roster should flatten instead and let the curve carry the climb.

`[?]` What distinguishes one stairwell from another. `[?]` Whether the shop and the door choice
are one screen or two, and in which order.

[ascend.go](internal/screens/ascend.go) is a stub whose comment already describes this.

---

## Enemies

**Every enemy carries its own deck, and that is what makes it itself** *(2026-08-16, owner's
call)*. `data/enemies.json` holds a `Cards` array per record, written in the card language above:
three attacks on the 0.5x / 1x / 2x rungs plus one non-attack, named to the creature. A Clear Slime
oozes, engulfs, dissolves and congeals.

**This deleted `PlanStyle` and the four planners.** An opponent's behaviour used to be a string on
its record picking one of `brute`, `swarm`, `warden` or `tactician`, and every enemy in the game
drew from one shared list of `Attack` and `Heavy`. Three consequences, all now moot:

- A Dragon and a Slime differed by a label rather than by anything the player could read.
- **Three of the four styles were unreachable.** The warden asked for a Defend by name and the
  tactician for a Prepare; the shared list held neither, so *every* enemy fought as a brute.
- Affixes, which *transform* a deck, had almost nothing to transform.

An enemy holding four cheap copies of one card is a swarm. One holding four expensive ones is a
brute. One holding shields is a warden. The player learns a deck.

### Enemies do not form hands

**An enemy's attack cards resolve one at a time, in the order its planner chose them**
*(2026-08-17, owner's call)*. Each lands its own blow at its own face damage; no hand is read, so
there is no multiplier and no hand off an enemy's turn. `Duelist.SoloAttacks`
carries it and `resolveSoloAttacks` is the phase.

**Hands are the player's axis and an enemy has no way into it.** A hand counts copies of a
*concept*, and every enemy card in the roster is authored `basic` and `FormNone`, so what an
enemy "formed" was an accident of what its planner could afford. Now
three cards on the table mean three blows, which is a round the player can read off the table
before pressing DUEL!.

- **A defence still covers the whole turn.** Raised cards answer *every* blow of the opposing turn
  and are spent once it is over — see `applyDefends`. Spending a Defend on the first swing would
  make it nearly worthless against the only opponents that swing more than once.
- **One shock roll per turn, not one per card.** A shock is "the turn's attack misses". Rolling per
  card would change what the status means and would advance the package's one random stream a
  different number of times each round.
- **It is a flag on the duelist, never a rule about side B.** The engine has no idea which side is a
  person and must not learn: the balance tool plays both sides headlessly.
- **`[?]` Whether a boss or an affix can give an enemy hands back.** The flag is per duelist, so
  nothing in the rules forbids it.

### One planner

`PlanFor(duelist, hand)` **scores every affordable combination of the hand's attacks**, and takes
the best. It is exhaustive rather than greedy because a greedy pass cannot see that three Ooze
forming a Three of a Kind beat one Dissolve — a hand is at most seven cards, so this is 128 candidates.

**A hand-forming duelist is scored through the same `blowFor` the resolver uses**, so the plan it plays
is the plan the engine will score. **A solo attacker is scored as the sum of what it picked**, which
is the same arithmetic its phase performs — so the search is looking for the most damage the budget
buys rather than for the best combination.

**Then it spends what the attacks did not want**, on defences first and the hand's own order after.
That second pass is what keeps a non-attack card in an enemy deck from being dead content: a
planner that only maximised damage would never raise a shield, and every `Congeal` in the roster
would sit in a discard pile forever.

**`Copies` was the difficulty dial and it is a blunter one now.** Under hands, four copies of a
1 AP card was a Four of a Kind at 5x; with no hand to form, four copies is four small blows and the dial
is simply *how many cards a turn holds*. What sharpened instead is **variety**: an enemy with three
different attacks used to land only the biggest of them, because distinct concepts formed no hand
and fell through to the High Card. It now lands all three.

### Balance after the rework — measured, and not yet good

`tools/balance` plays every posture against every enemy through the real `ResolveRound`. As of
2026-08-16, against a fighter of DMG 10 / 6 AP / 60 HP wearing all four rings:

| | walls (beaten by no posture) |
|---|---|
| before this change | 12 of 96 |
| enemy decks, HP left alone | 15 of 96 |
| **enemy decks and HP doubled** | **44 of 96** |
| enemies stopped hand-forming *(2026-08-17)* | 45 of 96 |
| **the 10% ascent curve — what ships** *(2026-08-17)* | **74 of 96** |

So **the per-enemy decks cost three walls, the HP doubling twenty-nine, the hand removal one, and
the ascent curve another twenty-nine**. Floors 1–2 are untouched — floor 1's outer room is the
curve's baseline — and everything from floor 3 up is now a wall.

**The curve is measured at the shallowest slot each enemy can occupy**, the first fight of its
lowest valid floor, so those are the kindest numbers a player will ever meet it with.

**Taking hands off the enemies moved the total by one and the roster underneath it a lot**, in
both directions, which is worth knowing before reading the total as "no change". Floors 1–2 got
markedly easier — several enemies went from being beaten by two postures to being beaten by eight —
and floors 2–5 got harder, because an enemy holding three *different* attacks used to land only the
biggest of them and now lands all three. The tool is one draw rather than a distribution, so a
±1 movement in the total is inside its noise; the per-band movement is not.

**Forty-four walls is accepted, not a regression to fix** *(2026-08-16, owner's call)*. The
objection was that the doubling overshot against a player whose own ceiling has not moved. The
answer is that the ceiling is *supposed* to move, and rings are how: a duelist wearing nothing is
not the duelist those floors are priced against, and the deep tower is meant to need a build. **The
whole ascension is not expected to be winnable yet** and reading these numbers as though it should
be is what would produce the wrong tuning.

What this does mean is that **the wall count stops being a bug signal and becomes a progression
signal**, which is a different thing to measure — and the thing to measure it with is a player who
has been built up, not the bare fighter this tool uses. Two figures still deserve reading: a wall
on a *shallow* floor is the old failure and is still a failure, and the tool being one draw rather
than a distribution makes every number here softer than it looks.


### The count bound moved into the rules

`maxSelected` left the screen and became `Duelist.MaxActions()`. It had to: **the opponent's
planner obeys the action cap exactly as the player's selection does**, and a cap enforced only
by the screen was a cap the enemy ignored. A method rather than a constant, so a ring raising
it has somewhere to bite — which is what this file asked for.

---

## Randomness

The determinism rules in `CLAUDE.md` still hold.

**Lightning put randomness into combat, and it is built.** The rules pre-gated this rather than
forbidding it, and it arrived the way they required: an injected `*rand.Rand` on `ResolveRound`,
never a package global. A nil source means no rolls, which is how tests and any future preview
stay exact. It is the sixth stream and it is salted from `RunSeed` like the others; nothing
shares a source.

**`[?]` The roll is conditional, and this document said it should not be.** The design note here
required rolling on *every* attack phase and discarding the irrelevant result, on the grounds
that a conditional roll means adding or removing a status shifts every later roll in the run —
so a balance tweak invalidates every stored seed. The implementation short-circuits when the
attacker carries no shock, so the stream only advances when lightning is in play. **Nothing
depends on stored seeds yet**, which is why this is recorded rather than fixed; it has to be
settled before the save format lands, because a choice log replays through this.

**`tools/balance` is a sample, not an answer.** It plays one fixed-seed draw per matchup, so a
posture winning half the time and one winning always currently look identical. Multi-sample
reporting is open work and is what the tool needs before any number it prints can be tuned
against.

**Deck shuffles use a seed derived per encounter, not a running stream:**
`hash(runSeed, floor, fightIndex)`.

- Same run seed replays the same enemy deck.
- Different encounters shuffle differently, even for the same enemy type.
- **A derived seed does not advance**, so nothing the player does can move it. A running stream
  stays vulnerable to draw-count drift; this cannot be.

The same trick should apply to the player's deck once `Session` exists.

---

## Overlays

Two kinds, and the distinction matters because there is no Escape key and no right click:

- **Modal** — fills the screen, everything behind goes dead, requires a deliberate action to
  leave. The deck overlay today; a hand reference later. Its exit must be the brightest thing
  on screen or it is a trap.
- **Transient** — flashes over a frozen screen, takes no input, dismisses itself. The **HAND**
  splash. Needs no exit affordance precisely because it accepts nothing.

**The freeze is already built.** `dwellFor` decides how long each event holds the screen, so a
`KindHand` event with a splash-length dwell gets a frozen screen for free and the splash draws
while the playback cursor rests there. Presentation-only, so it cannot touch the outcome, and
splash length joins the pacing constants destined to become the game-speed setting.

**A hand never has to be drawn *across* rows**, because it gets a line of its own, in amber, at
the moment it forms. So does a chill. The bracket-or-join problem simply stopped existing, which
is worth recording as the pattern — **one row per slot was being asked to answer two questions at
once, and the fix was a line of prose, not a cleverer drawing.**

The **hand splash** above is still wanted and still unbuilt; a sentence makes a hand *legible*,
not *loud*. Freezing the screen for a splash-length `KindHand` remains free.

---

## Open questions

Collected from above.

- `[?]` **Nothing reads within-phase order any more**, so drag-to-reorder has no mechanical
  effect. Either it stops being presented as a decision, or something reads order again.
- `[?]` **Nothing tests what Plan is worth.** `tools/balance` deals no cards, so a wider hand is a
  wider hand of nothing and the `planning` row measures 2 AP of pure loss. Pricing it needs the sim
  to draw, which needs a seventh stream — see the entry under *Determinism*.
- `[?]` **The three attack forms differ only in which cards pair with which.** Same costs,
  same damage, no riders — enough to make a hand a choice, not enough to make a form one.
- `[?]` **Draw variance is answered by two levers now** — the Discard button and Plan — and neither
  has been priced against the other.
- `[?]` **Pair fires on most turns**, which makes the bottom rung a permanent global multiplier
  favouring whoever repeats themselves — currently the AI planners.
- `[?]` **Every same-concept hand shows all-distinct colours**, because the deck holds one copy per
  concept per colour. It no longer costs a multiplier, but it does mean a built hand always lands
  every status the player is ringed for — the colours are not a choice.
- `[?]` **The shock roll is conditional**, against a written rule that it should be unconditional.
  Settle it before the save format lands.
- `[?]` **Enemy life totals have not been retuned** against the new damage curve, and
  `tools/balance` is a single-sample tool that cannot yet answer whether they should be.
- `[?]` Duration, stacking and refresh for every status.
- `[?]` Whether ring cards may be shorter than action cards, given they have no glyphs.
- `[?]` What distinguishes one stairwell from another.
- `[?]` Whether the shop and door choice are one screen or two.
- `[?]` Whether earth becomes a floor affix.
- `[?]` How a player is shown that an attack card contributed nothing. The pane no longer draws a
  line per attack card, so a card outside the hand is silent there; the table says it, by raising
  every attack card and then lowering the ones the hand did not name. Whether that is legible at
  playback speed is unanswered.
- `[?]` Earth's green collides with `playerSwatch`. One of the two schemes has to give, and
  what is holding it off is that a border and a swatch are never seen side by side.
- `[?]` Whether the attack/plan categories are the same axis as the *role* taxonomy the
  initiate/respond model in `TODO.md` asks for, or orthogonal to it — and whether the *forms*
  are the axis that taxonomy actually wanted.
- `[?]` How enemies scale up the tower.
