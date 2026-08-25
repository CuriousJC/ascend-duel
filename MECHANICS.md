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
multipliers made every fight far shorter than the numbers read. **Nothing has measured the result**
— see *Enemies* below.

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

**The attack set is a 3x5 ladder: three forms by five cost tiers, filled**, and the tiers are
identical across the forms. A form is *which* pair you are building toward, never a stronger
or weaker way to build one.

**The middle three rungs are the deck; the two ends ship at zero copies** *(owner's call,
2026-08-24)*. A run opens holding 1/2/3 AP cards and can never buy anything else, so the only way
to hold a Poke or an Impale is a **Shrink** or a **Grow** worm walking a card off the end of the
three. They are real registered concepts all the same, because `combat.Neighbour` derives the
ladder from the registry — a rung that does not exist is a rung a worm cannot step onto, which is
what used to make Shrink dead on every 1 AP card and Grow dead on every 3 AP one.

| Form | 0 AP · 0.25× | 1 AP · 0.5× | 2 AP · 1× | 3 AP · 2× | 4 AP · 4× |
|---|---|---|---|---|---|
| **stab** | Poke | Jab | Thrust | Lunge | Impale |
| **slash** | Nick | Cut | Slash | Cleave | Sever |
| **crush** | Tap | Bash | Strike | Smash | Pulverize |

| Form | Concept | AP | Effect |
|---|---|---|---|
| **stab** | Poke / Jab / Thrust / Lunge / Impale | 0 / 1 / 2 / 3 / 4 | Stabs for `DMG/4` / `DMG/2` (both min 1) / `DMG` / `DMG × 2` / `DMG × 4` |
| **slash** | Nick / Cut / Slash / Cleave / Sever | 0 / 1 / 2 / 3 / 4 | Slashes for the same five figures |
| **crush** | Tap / Bash / Strike / Smash / Pulverize | 0 / 1 / 2 / 3 / 4 | Crushes for the same five figures |
| **plan** | Prepare | 1 | Banks +2 AP for the next round |
| | Plan | 2 | Draws **2 extra cards** into the next round's hand |
| | Defend | 3 | Takes **50%** off the blow aimed at you next |

**Nine attack concepts × four colours = 36 cards; three plans × four copies = 12.** A **48-card
deck** — the six zero-copy rungs are in the file and not in the pile. **No card in the player's deck
is drab** *(2026-08-15)*: every attack ships in one of the
four primary elements and the only basic cards are the plans, because nothing a plan does is
elemental and a coloured Defend would be a colour that meant nothing.

**A 0 AP card is bounded by the count rather than the cost**, which is the shift `minCardCost`
already took deliberately when Whetworm could drive a card to free: a turn is capped at
`MaxActions` cards however cheap they are. **A 4 AP card is the first single card that beats a
whole cheap turn**, at 2× a Lunge for 1.33× the price — the reason it is a worm's prize rather
than something a run can stock.

**The plans sit on the dealt 1/2/3 ladder as the attacks do**, and there is no plan at either
end — the two outer rungs are an attack's ladder only. What
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

**The three forms cost the same and hit the same, and differ only in which cards pair with which.**
That is enough to make a hand a *choice* — holding two Cleaves is not the same as holding a Cleave
and a Smash — and it is deliberately the whole of it.

**Every card carries its effect in words on its face**, verb first, filling the card beside the
cost column. The attack text names the form's verb — "Stabs for 2x DMG" — rather than opening
"Deal" on all nine, so the corner mark is not carrying the distinction alone. The wording is
`cardEffects` in `internal/screens`, beside the prose the fight log uses: the rules package
names actions and never describes them. **Short words are a hard constraint** — the column is about
a dozen characters wide — and two tests hold the wording to it.

**The corner mark is drawn art** *(2026-08-23)*: a spear for stab, a sword for slash, an axe for
crush, a bulb for plan, in `assets/form/`. **It is tinted by the card's element** rather than
repainted, so the drawing keeps its outline and its bevel — which is where the element is said
now, the border having stopped saying it the same day.

### Every raised plan answers the blow, and they multiply

**The plan cards a duelist has up are a set, not a queue.** The opponent's turn produces one
attack, so *every* raised card meets it and each takes its percentage off what is left. **Order is
not read**, and every card is spent on the one attack it answered. They all expire together at the
start of their owner's next turn.

**Defend halves, and it is the only card that reduces a blow at all** *(2026-08-15)*. Three points
of a four-to-six point budget is most of a round, which is what a halving is meant to cost.
Multiplying rather than adding is what stops several cards reaching past zero by accident: two
Defends take three quarters and a third takes seven eighths, a curve that never arrives.

**The plan form is three cards where the attack ladder is nine dealt**, and it is deliberately not
a grid of its own. Prepare is the cheapest card in the game and Defend the dearest; what sits
between them is one card rather than a rung, because a grid filled with cards that differ only by a
number is the trap this deck was rebuilt to avoid.

### Concepts and deck composition

**An attack concept ships as five cards: one per primary element.** That is the rule for adding an
attack, not just a description of the starting deck. **A plan ships in the same five**, because a
plan carries a colour for the ring discount and for the hand axis even though nothing it does is
elemental.

45 + 15 = **60 cards** *(2026-08-25, up from 48 when arcane landed)*. A hand of eight against that
is 13% of the deck, against 17% at 48 and 27% when the deck was 30. What answers draw variance is
the Discard button and Plan.

**Five copies of a concept is the ceiling of the *starting deck*, and it shapes the hand table.**
No attack concept ships more than five times, so **a Card Four of a Kind necessarily shows four of
the five colours** — copies of a concept are all different elements, so it is also the hand that
lands most of the statuses the player is ringed for. **A Card Five of a Kind became dealable on
2026-08-25** and is the whole colour set in one concept; it was reachable only after a `duplicate`
worm while the deck held four copies. See the reachability table below.

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
- **A card does not say who it lands on; its verb does.** An attack lands on the opponent and
  everything else on its own duelist, and there is no field to disagree with — so a card cannot be
  aimed anywhere the rules do not resolve, and nothing has to be validated to stop it.
- **Validation replaced the cross-check.** `CheckCostTiers` compared a declared cost against the
  rules and had nothing left to compare once the file became the rules. `combat.RegisterConcept`
  refuses an unknown verb, a defence of 100% or more, a zero amount, and the unbuilt half of the
  grid. A bad record panics at init.

**A card never names a status, and that is load-bearing.** See *Elements* — what a colour does is
decided by the source of that colour on the card's owner, and a ring may later decide *which* fire
a fire card applies. A card that named its own status would be deciding something that is not its
to decide.

**52 was considered and rejected**, and the arithmetic has changed twice since without changing the
answer. The playing-card instinct argues for 13 ranks × 4 suits; the ladder decides it instead, and
nine *dealt* attack concepts by however many colours exist is what three forms by three dealt tiers
produces. `basic` is still the absence of an element rather than a suit, and arcane is a suit rather
than a variant of nothing.

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
built. A press becomes a three-way decision the day it lands — move past `dragThreshold` is a drag,
held past a tick count without moving is a long press, released before either is a click that
toggles selection. The distance and time thresholds must not fight each other.

---

## Elements

### The set

| Tier | Elements |
|---|---|
| **primary** | ice, fire, lightning, earth, arcane |

**There are five elements and no more** *(arcane added by the owner, 2026-08-25)*. Every one of
them has cards, a colour and a status, so an element is a complete thing rather than a name waiting
for rules. **Arcane arrived with all three and nothing was waived**: twelve cards, purple, and
WEAKENED. Anything wanting a sixth has to do the same.

**What a fifth colour cost, so a sixth can be priced before it is proposed.** None of it was the
cards; the cards were one line of JSON.

- **Every elemental hand got harder and the whole ladder was re-derived.** A colour is now one in
  five rather than one in four, so an Elemental Four of a Kind fell from 6.9% to 3.7% reachable.
  See the reachability table.
- **A Card Five of a Kind became dealable**, at about one hand in 22,000, because a concept ships
  one card per colour and there are now five. The rung had no probability behind it since it was
  written; it has one now.
- **The deck overlay ran out of room.** Five rows of cards plus the tally band did not fit the
  modal, and the fix came out of three places at once — see `internal/screens/deckpanel.go`. The
  card itself could not shrink: the form mark is pixel art on a 32px canvas.
- **Twelve rings, not one.** Each colour carries four of its own — damage, status, discount, growth
  — and the flip rings are a full cross-product, which went from 12 to 20.
- **Every catalogued deck seed had to be re-checked**, and two of the five were re-found.

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
| arcane | purple |

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
| **arcane** | weakened | the victim takes double damage from everything, burn ticks included |

**WEAKENED is a different shape from the other four, and that is worth knowing before a sixth is
designed** *(owner's call, 2026-08-25)*. Every other status modifies what its carrier *does* — how
hard they swing, how often they connect, how many cards their turn holds — so it is read off
whoever is acting. WEAKENED modifies what its carrier *takes*, so it is read off whoever is being
acted upon, and it is the first thing in the damage pipeline to be read off the victim. Two
consequences follow and both are intended:

- **A ring applying it is worth more against a slow opponent than a fast one**, because the value
  is in the blows that land during its two rounds rather than in the ones you throw.
- **It amplifies a burn tick as well as a blow.** A tick is damage the carrier takes, and exempting
  it would have made the rule *"damage, except the kind that arrives at the end of the round"* —
  which is a sentence no card face can carry. Fire plus arcane is therefore the sharpest pair of
  statuses in the game, and it is a build rather than an oversight.
- **It does not stack and it is capped**, like everything else here: a second application refreshes
  the clock, and `combat.maxAmplifyPct` holds any sum at 300. Amplification is the one percentage
  with no natural ceiling — a miss chance and a weight both stop at *nothing reduces a blow to
  zero*, and this one stops nowhere at all.

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

Per-element tuning is one constant each away, and **nothing measures what moving one does**.

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
- The stream advances **per attack phase**, so a change early in a duel reshuffles every roll
  after it. That is the cost `MECHANICS.md` predicted when it argued against the roll; it is
  real and it is now paid.
- It breaks the rule hands otherwise follow — *what you committed to cannot be silently undone*.
  Lightning is the deliberate exception, and it is the only one.

#### Ice, fire, earth and arcane in detail

- **Ice takes a card, not a point.** A chilled duelist loses a card off the front of its turn,
  and the front of a turn is its attacks — so ice costs a swing, and it is felt after the player
  has committed rather than while they are still choosing.
  - **It is the only thing in the game that takes an action.** The chill is read straight off the
    status and `Duelist` carries no separate counter. The action points are **not** refunded: a
    chill is tempo *and* economy.
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
  - **A burn is state that outlives an action.** `KindBurned` fires from `endRound`, side A then
    side B, and the screen's `applyEvent` reads it alongside `KindDamage` because a burn changes a
    life total with nobody acting. **A burn can kill**, and produces a
    `KindDefeated` when it does.
- **Earth applies attacker-side, before any defence.** Weight says how hard you can still swing,
  so the order is: the hand's own cards, the hand multiplier, the attacker's weight, then every
  raised plan card. Everything the defender does therefore happens to a blow that has already been
  blunted. **Rounding is toward zero**, matching the defend reductions and `scaleDamage`, so it is
  predictable from the reductions already in the game. **It is 25%**, because a smaller cut that
  cannot stack is a status nobody notices landing.
- **Arcane applies victim-side, after the weight and before any defence.** Vulnerability says how
  hard *this body* takes a blow, so the order is: the hand's own cards, the hand multiplier, the
  attacker's weight, the target's vulnerability, then every raised plan card. A card raised in
  answer to the blow is therefore spent on the figure both statuses have already produced.
  **Rounding is toward zero**, matching blunt and every other percentage, so the two halves of one
  sum round the same way. **It is 100%** — double — and unlike the other four it is capped
  centrally as well as per record, at `combat.maxAmplifyPct`.
  - **It reaches the burn tick too**, in `endRound` rather than in the attack phase, which is the
    one place a status modifies damage nobody threw.
- **Statuses live in `Duelist.Statuses [MaxStatuses]Status`** — an array indexed by **status**,
  not by element and not as named fields. That is what makes *"consume the status this card
  applies"* expressible and is the difference between a system and a handful of ad-hoc fields. The
  price: **`StatusID` is append-only and the file decides the order**, the hazard `Element` and
  `GlyphKind` also carry. The raised-defence set stays where it is — those are card effects, and
  filing them in this table would say they were not.

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

There is no initiative. With one contiguous turn per side there is no exchange for a faster action
to lead, so a number on every card would report a distinction the resolver does not make. `Spd`
buys action points and never buys priority.

**Order within a category is queue order, and one thing reads it**: `groupsOf` fills
largest-count-first and breaks a tie by whose first card was played first, so the lead card — the
one that names the hand and carries its element — is chosen by where the player put it. Nothing
else does. A defense cannot be dragged ahead of an attack, a counted hand reads the turn as a set,
and defends compose without an order.

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
roguelike unlock structure, not the run. **The profile exists as of 2026-08-25 and does not gate
this yet** — `profile.Profile.HandsDiscovered` is the field waiting for it, and every hand is still
live. Gating the *table* is the whole of the change when it lands; nothing else moves.

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

**Every card in the turn is counted, and that is the matcher's rule rather than the catalogue's**
*(2026-08-17, widened 2026-08-23)*. An entry used to name the categories it counted and could never
change what was counted, so the field only invited an entry to claim otherwise. What is left out is
decided by the axis — a card with no value on it — and by nothing else. **A Prepare joins a hand**
and brings no damage into it.

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

### What the axis costs

- **Counted matching only.** A hand reads the turn as a set, so a Jab between two Strikes does
  not break the pair, and no hand can ask for an *ordered* run of cards.
- **A hand cut short still pays out.** Nothing can interrupt it — a turn's attacks resolve as one
  event — so this is true by construction rather than by rule.
- **The bottom rung fires constantly**, and as of 2026-08-19 it is priced as such: the form and
  elemental pairs are 98–99% hands paying 110, which is a floor rather than a reward. The open
  question of whether the ladder should start at Two Pair is **answered by pricing instead** — a
  near-certain rung pays near the identity, so it costs nothing to leave it in and it keeps the
  bottom of the ladder legible.
- **Poker's ranking does not transfer to this deck, and the ladders are now priced off measured
  rarity rather than off poker** *(2026-08-19)*. Poker's ordering comes from 52 cards, 4 suits and
  13 ranks; here a concept has 4 copies and a colour and a form 12 each since the plans were
  coloured, and the turn is bounded by AP rather than by the draw. See *The multipliers come from how often a hand can actually be built*
  above for the model and the table.
- **A turn's mismatched attacks sum**, rather than the biggest one landing alone, so a hand is
  worth more the dearer its cards are: at DMG 10 four Lunges are **400** where four Jabs are
  **100** and three Bashes are **30**. **Nothing on the enemy side is tuned against that** — the
  ladder, the ascent curve and the roster are independent, and the ladder is one file.

### The catalogue's shape

`data/hands.json` holds one list of **nineteen** entries: six rungs on each of three axes, plus the
one High Card they all fall back to. **A hand carries a key, an ID, a name, a `match`, `groups` and
a `multiplier` in percent** — nothing else. `groups` naming *distinct values on the hand's own axis*
is why `[3,2]` is a full house and can never be satisfied by five cards sharing one value.

**`match` is required and never defaulted.** An entry that landed on the wrong axis by omission
would be a balance change nobody made, so a missing or unknown one is refused at init like any other
malformed record. Two further refusals live beside it: a hand wanting more cards than a turn holds,
and one wanting more groups than its axis has values — four forms and four elements reach a blow, so
a five-group form hand is a rung nobody could climb and would otherwise fail silently. It was
*three* forms until 2026-08-23, when `plan` joined them.

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

**Plans carry an element and join hands** *(owner's call, 2026-08-23)*. Every one of the forty-eight
cards is now one of the four colours — the three plans ship one copy per colour where they used to
ship four basic copies, so the deck size did not move — and the matcher counts them like anything
else. A hand is **what you played, not what you hit with**.

What that changed, in order of how much it matters:

- **The element axis went from nine cards a colour to twelve**, which is exactly as wide as the form
  axis. The two ladders are now priced identically at every rung, because they are now equally hard.
- **`plan` became a fourth countable form.** Any two plans are a Form Pair regardless of concept or
  colour, and twelve of forty-eight cards carry it.
- **A plan brings no damage into the hand it joins.** `Card.Damage` is zero for every verb that is
  not an attack, so the multiplier multiplies the attacks that are in there with it — a fire Prepare
  beside two fire Strikes turns a Card Pair into an Elemental Three of a Kind and pays it on the two
  Strikes' damage. That is the whole of what the change buys.
- **A plan's colour arms a status.** `elementsOf` reads the formed hand, so a fire Prepare shows fire
  and lands a burn on a turn with no fire attack in it. That is the sharper half of the same
  decision.
- **A hand of nothing but plans is real and lands nothing**, which is the accepted cost — see the
  decision below the table.

The three ladders are **not** the same numbers, and as of 2026-08-25 no two of them are. The
starting deck is 60 cards — **5 per concept, 15 per form, 12 per element** — dealt into a hand of
eight against a 6 AP, 5-card turn, and that arithmetic is what the ladder is priced against rather
than poker's.

Reachability, from a two-million-hand simulation of round one — *can this turn afford some set
forming this rung* — with the multiplier each was given:

| Rung | concept | | form | | element | |
|---|---|---|---|---|---|---|
| | reach | pays | reach | pays | reach | pays |
| Pair | 92.4% | 115 | 100% | 110 | 100% | 110 |
| Two Pair | 19.9% | 210 | 52.2% | 145 | 43.3% | 155 |
| Three of a Kind | 11.3% | 250 | 76.1% | 150 | 60.1% | 160 |
| Full House | 0.95% | 415 | 5.9% | 290 | 3.1% | 335 |
| Four of a Kind | 0.27% | 500 | 7.7% | 295 | 3.7% | 340 |
| Five of a Kind | 0.004% | 785 | 0.093% | 575 | 0.019% | 680 |

The rule that produced them is unchanged: **`100 + K x ln(1/P)`, floored at 110, rounded to five,
then forced to climb within each ladder.** **K is 67.7 where it was 61.2**, because the constant is
*defined* by anchoring the rarest measurable hand — a concept Four of a Kind — at the 500 it already
carried. Re-deriving the constant rather than keeping it is what holds the concept ladder
recognisably where it was through a change that moved everything under it.

**The form and element ladders came apart, and that is the headline** *(2026-08-25)*. They were
priced identically at every rung on 2026-08-23 because they were then equally hard — twelve cards
share a form and twelve share a colour. Arcane broke the symmetry in both directions at once: a
form now gathers fifteen cards where it gathered twelve, and a colour still gathers twelve out of a
deck a quarter bigger. So **elemental hands got harder and form hands got easier**, and the element
ladder now pays more than the form ladder at every rung above the pair. An Elemental Four of a Kind
went from 6.9% to 3.7% reachable and from 285 to 340.

**Nothing measures whether the re-pricing lands.** The curve says what a rung is worth relative to
the others; it has never said whether the whole ladder is worth the right amount, and there is
still no simulation of a duel to ask.

**One number in the table is judgement rather than measurement, and it is the same one as before:**

- **The concept Pair keeps 115 rather than the 110 the floor gives it.** At 92.4% the curve floors
  out, and letting it sit at 110 would make the narrowest axis pay exactly what the two wide ones do
  at the bottom rung — the nesting problem the ladder exists to avoid.

**Card Five of a Kind stopped being an estimate** *(2026-08-25)*, which leaves every other row in
the table straight off the tool. It was 745 by judgement for as long as the deck shipped four copies
of a concept and nothing could deal a fifth — the rung existed for the `duplicate` worm. Arcane made
a concept five cards, so the rung is dealable at about one hand in 22,000 and is priced off its own
number at 785. It is still the rarest thing a round-one hand can build, by an order of magnitude.

**The element five-of-a-kind stopped being an estimate on 2026-08-23**, for a different reason: a
colour gained three plans, so five cards of one colour came down to 6 AP and the rung became
measurable at the real budget.

**The five-of-a-kind row is the one that is not all measurement** *(2026-08-19)*, and it is written
out because a number that came from somewhere else must not read as one that came from the tool:

- **Both wide five-of-a-kind rungs are now straight off the curve**, at 0.049% and 0.047% — one turn
  in about two thousand, and still the rarest thing a round-one hand can build on either axis.
- **Card Five of a Kind is the one row with no probability behind it**, and the thing to re-derive
  first if the `duplicate` worm ever makes five copies common.

Three things fall out of it and are worth keeping:

- **A near-certain hand pays near the identity.** The form and elemental pairs are 98–99% hands, so
  they are a floor rather than a reward — what they buy is the *sum of both cards*, which is
  already most of the change.
- **The narrower axis pays more at every rung**, which is what stops the nesting from making the
  concept ladder dead content: a card hand is always also a form hand, so if the form rung paid the
  same, nobody would ever have a reason to build the narrower one.
  `TestANarrowerAxisPaysMore` holds it.
- **The ladders cross, and that is intended.** A form Three of a Kind (145) pays less than a card
  Two Pair (200) though it uses fewer cards, because it is eight times as easy to build.
- **The form and element ladders are the same numbers as of 2026-08-23**, because the two axes are
  now the same width — twelve cards share the commonest value on each. They are kept as separate
  entries rather than merged: the *tie-break* still prefers form, so an elemental hand is only ever
  named when no form hand of the same rung is live, and what an elemental hand carries into
  `elementsOf` is different. **If the two are meant to feel different, the thing to change is the
  deck rather than the file** — pricing them apart when they are equally hard would be a number
  with nothing behind it.

**The best hand is chosen on multiplier, not on what it would deal, and that is now a decision**
*(owner's call, 2026-08-23)*. It was an open question while the two could only disagree by a little;
plans joining hands made them disagree by everything, and the answer is to leave the matcher alone.

The case that forced it: a turn of `Strike + two plans` forms a **Form Pair at 110 on zero damage**,
because any two plans share `FormPlan` and the pair beats the High Card's 100 — so the Strike is
announced and lands nothing, where the Strike alone would have landed its face damage. Playing plans
beside a single attack can cost you the blow, and **reading the board to avoid that is part of the
game** rather than a bug to design out.

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
whole round on Prepares. **Five Strikes is dealable as of 2026-08-25** — the deck holds five, one
per colour — which is what turned the concept five-of-a-kind from a worm's rung into the rarest
measurable hand in the game. Being dealable and being affordable are still two different questions,
which is why the wide five-of-a-kind rungs are the cheapest cards of a form or a colour rather than
of a concept.

**A colour's cheapest five now includes a plan** *(2026-08-23)* — fire Jab, Cut, Bash and Prepare at
1 AP each plus a fire Thrust at 2 is **6 AP**, which is a plain round's whole budget and the first
time an elemental five-of-a-kind has been affordable without banking. It used to be 7 AP, because a
colour held one card per form per tier and nothing cheaper. The Prepare pays nothing into the 5.65x;
what it does is take the place of the second 2 AP attack the hand used to need, so it is a rung the
plans *open* rather than a rung they win. `go run ./tools/handsheet` draws it.

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
- **Nothing measures any of this**, so **a ring's balance is unknown** — say so rather than guessing
  at a multiplier.

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
| **Burning / Chilling / Shocking / Weighted / Weakening** | `attack-lands` | the five colours' status rings, split off on 2026-08-22 and priced uncommon |
| **Fire / Ice / Lightning / Earth / Arcane** | `card-damage` | doubles every card of that colour — *element* multipliers, where Keen/Heavy/Needle are form ones |
| **Storm** | `attack-lands` | lightning shocks *and* chills |
| **Keen / Heavy / Needle** | `card-damage` | doubles **every** slash / crush / stab card in the turn |
| **Striker** | `card-damage` | doubles every Strike — a concept ring, 5 cards where a form covers 15, and priced accordingly |
| **Banker** | `fight-won` | a second +1 vitae per 5 held, on top of propagation |
| **Soul Taker** | `prizes-dealt` | the vitae prize card pays +10 rather than +5. A **flat** +5, not a scaling |
| **Hungry** | `prizes-dealt` | two post-battle choices instead of one |
| **stat rings** | `fight-start` | +10 DMG, +25 HP — and growing variants that gain per fight |
| **Momentum** | `card-damage` + `turn-taken` | every card gains +0.2x DMG per turn with no plan card in it; a plan card wipes the streak |
| **Enflamed / Frostbitten / Lithium / Granite / Unravelled** | `card-damage` + `attack-lands` | their colour gains +0.1x DMG per landed hit of that colour, and keeps it while worn |
| **Echo** | `blow-formed` | the blow's first attack card lands three times: full, 2/3, 1/3 |
| **Flurry / Rend / Aftershock** | `blow-formed` | every stab / slash / crush card lands **twice**, both at full DMG |
| **Atrophy** | `deck-built` | every 3 AP attack is dealt as its 2 AP version |
| **Onslaught** | `card-cost` + `fight-start` | every card 1 AP cheaper, and a quarter off your life — the first ring with a drawback, and the first rare |
| **Warm / Cold / Static / Dirty / Eerie** | `card-cost` | every card of that colour costs 1 AP less — one per colour |
| **flip x20** | `card-drawn` | recolours a card of one colour as another **as it is drawn** — one for each ordered pair; see below |

**A concept ring and a form ring are not the same object** and must not be priced as one.
Striker covers 5 cards, Keen covers 15.

**Arcane brought twelve rings, not one** *(2026-08-25)*, and that is the number to expect from a
sixth colour rather than the one a card list suggests. Four are the colour's own seats in families
that already existed — **Arcane** (2x DMG), **Weakening** (applies WEAKENED), **Eerie** (1 AP off),
**Unravelled** (grows +0.1x per landed arcane hit). The other eight are the flip cross-product,
which is quadratic: **Witchfire / Runefrost / Spellbolt / Leyline** turn fire, ice, lightning and
earth into arcane, and **Hexfire / Hexfrost / Hexbolt / Hexstone** turn arcane into each of them.
Twelve ordered pairs became twenty. **A sixth colour would bring fourteen**, ten of them flips.

**Weakening is the strongest of the five status rings and is priced the same as the others.** That
is deliberate rather than unexamined: WEAKENED doubles everything the target takes for two rounds,
where a weight blunts a quarter and a chill takes one card, so its tier is the thing to move first
if the arcane build turns out to dominate. Nothing in the repo measures what a ring does to a duel,
so the price is judgement — see the rings skill.

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
  a player skip a Plan is unmeasured, like every other ring.

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
  the top of the tower, and nothing measures it.

### Atrophy, and the ladder as a ring *(2026-08-22, owner's call)*

**Every 3 AP attack is dealt as its 2 AP version**: Lunge becomes Thrust, Cleave becomes Slash,
Smash becomes Strike. Rare.

**It is the flip's shape applied to the other axis, at the other moment.** A flip changes a card's
colour as that card is *drawn*; Atrophy changes its *concept* as the fight's draw pile is built, one
rung down the same form's ladder. The two moments matter for how the deck panel shows a card and for
nothing else in play — see the flip rings below. Everything downstream — cost, damage, the hand it forms, the card face — follows
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
- **Nothing measures it**, like every other ring.

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
cards instead of three is a different hand ladder, not a bigger number. **Nothing measures that**, so
what a colour's discount is worth against a colour's doubling is unknown, and reads as the bigger of
the two.

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

**Twenty rings, each `card-drawn` / one colour in / another colour out.** Frozen Lightning was the
only one for five days; the pattern generalised to every ordered pair, and all twenty are **common**.
**It is a cross-product, so it grows quadratically**: four colours were twelve rings and five are
twenty. That is the cost line to read before proposing a sixth colour — it would be thirty. The names are thematic rather than mechanical — "Permafrost" says earth into
ice without saying either word — which is a deliberate cost: the *card* has to be read to know what
it does, and the tooltip is what says it.

| dealt as → | from fire | from ice | from lightning | from earth | from arcane |
|---|---|---|---|---|---|
| **fire** | — | Meltdown | Firestorm | Magma | Hexfire |
| **ice** | Frostbite | — | Frozen Lightning | Permafrost | Hexfrost |
| **lightning** | Heat Lightning | Thundersnow | — | Dust Storm | Hexbolt |
| **earth** | Obsidian | Glacier | Fulgurite | — | Hexstone |
| **arcane** | Witchfire | Runefrost | Spellbolt | Leyline | — |

**The eight arcane names are placeholders the owner may replace** *(2026-08-25)*. They follow a
convention the original twelve do not: everything *out of* arcane is a Hex-, and everything *into*
it is a spell word grafted onto the source's phenomenon. That was chosen because sixteen thematic
one-off names is more than a player can hold, and a legible half-convention is worth more than
another eight inventions.

**They fire as a card is drawn, not as the deck is built** *(owner's call, 2026-08-24)*. Every one of
them is worded "every X card is dealt as a Y card", and the dealing is the draw. The cards a fight
plays are identical either way — a flip is unconditional over a colour, so recolouring the whole pile
once and recolouring each card on its way out produce the same hand — so what this buys is not an
outcome but a **place**: the draw pile holds the deck the run owns, and the alteration is something
that happens to a card, at a moment, on its way to the hand. That is the shape the next kind of
alteration will need, and it is what the deck panel's ALTERATIONS toggle is a picture of.

**A drawn card does not remember what it was.** It carries the colour it became; a `card-damage` ring
keyed on ice fires on a card that is ice *now*, and never on one that used to be. That was the
owner's call and it is what stops an alteration turning every later rule into a question about
history. What the original is still reachable from is the card's **identity** — every card a run owns
carries an ID, so the deck panel can show either face of a card wherever it is sitting. **No rule may
read that ID**; it is a handle for the screens.

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
- **They are twenty of fifty-eight records, and the dilution is accepted** *(owner's call,
  2026-08-22, unchanged by arcane)*. The catalogue is now more than a third flips, so a common ring's ten tickets are ten
  out of a much bigger pot than they were at seventeen rings — and more rings are coming, which is
  what makes that fine. If the shelf ever does need thinning, the lever is a weight or a tier, not
  a price.

**Every colour is two rings as of 2026-08-22** *(owner's call)*. **Fire, Ice, Lightning, Earth and
Arcane** are now `card-damage` doublers on their colour — the first *element* multipliers, where Keen, Heavy
and Needle multiply a form — each common and each keeping the colour's artwork. The status each used
to apply moved to a second ring — **Burning, Chilling, Shocking, Weighted, Weakening** — all uncommon
and all drawing the default ring face. So a colour offers cheap damage or a dearer, rarer status, and
ten of the fifty-eight records are now colour rings.

**Two records and two files were renamed with it**: `frozen-ring` → `ice-ring` and `thunder-ring` →
`lightning-ring`, with `assets/ring/frozen-ring.png` → `ice-ring.png` and `thunder-ring.png` →
`lightning-ring.png`. The ring naming now matches the element names the rules use. "Frozen" and
"Thunder" are free again and may come back for something else.

**BURNING went from 10% to 50% of the attacker's DMG in the same call**, over the same two rounds.
That is a fivefold change to a status nothing measures, so it is a judgement, and a large one: at 50% over two rounds a burn is
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
- **Nothing measures whether any of those numbers is right.** Nothing in the repo measures what a
  ring does to a duel, so what a doubling of every slash card is worth in vitae is a judgement. Recorded as a judgement rather than dressed up as a derivation.
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
  and a re-bought one goes on at the right-hand end. That is a real cost of letting a ring come off,
  and **there is no re-ordering control**: the one thing a player cannot choose is the order two
  rings apply in.
- **The shelf is its own random stream** (`seeds.ShopStock`), per fight, so a defeat and a retry walk
  into the same shop exactly as they meet the same opponent. **It is three weighted draws without
  replacement**, rather than a shuffle: each seat draws on rarity tickets and the drawn ring leaves
  the pool, so the shelf never offers the same ring twice.

## Worms — altering the deck between fights

**A worm is a change to a card you already own.** It recolours it, removes it, or copies it. It
never invents one, and that restriction is the whole safety property: the *concept* is never
touched, so nothing a worm produces can be a card `internal/combat` has not registered.

**Offered after a won fight, on the post-battle screen.** Two worms are drawn from the catalogue
and shown as cards; pick one, pick the card it takes, **see what it would become**, and confirm.

| | |
|---|---|
| When | after a won fight, before the next room |
| First | the win is **read out**: interest, a tenth of the life kept, what the room pays |
| Offered | **two cards**: two worms from `data/worms.json` |
| Then | a hand dealt fresh off the whole run deck, at the combat hand size |
| Then | a **morph**: the card before and after, side by side, nothing committed |
| Choice | one worm and one card, or **Let them escape** |

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
- **The screen opens by narrating the payout** *(owner's call, 2026-08-22)*. Four sentences type
  themselves out at the game's speed, and the figure each one names then **flies to the duelist
  card** and lands in the purse. A win used to change the purse silently while the fight was
  ending, so what a win is worth was arithmetic nobody ever saw. **A click skips to the end**, and
  the fast path pays through the same claims as the slow one — presentation may never change an
  outcome.
- **Your build is on screen throughout**: the duelist card in its usual corner and the worn rings
  beside it, so a worm is chosen against the thing it would be changing, and the purse the payout
  lands in is visible while it climbs.
- **The vitae card is gone and taking neither worm is a button again** *(owner's call,
  2026-08-22)*. A third card paying +5 was charging for something the win now pays by itself, and
  the offer is the two creatures the prose says are fleeing. **The offer is free**, so walking away
  costs nothing and does not need to look like a card.
- **The worms fly in from the sides** once the last sentence lands, because the sentence before
  them says two creatures are fleeing — a card already sitting there would contradict it.
- **Run-scoped, never persisted.** Two runs from the same seed may hold different decks, because
  an alteration is a *choice*: replay is a seed plus a choice log. See the `randomness` skill.

### The grammar: a target and a new value

A worm record is the card language pointed at a card that already exists — see `data/worms.json`
and `internal/session/worm.go`, which is where a record is validated.

| Target | Value | What it does |
|---|---|---|
| `element` | a colour | recolours one card — **one worm per colour, five since 2026-08-25** |
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

**The recolour worms are one per colour, and the set has to stay complete** *(owner's call,
2026-08-25)*. **Hex Worm** landed with arcane for that reason: four recolour worms against five
colours is a hole a player can see — every other colour can be built toward out of the post-battle
offer and one cannot. The catalogue goes eleven, so the two seats a fight offers are drawn from a
slightly bigger pot and every individual worm is a little rarer; that dilution is the same one the
flip rings took and it is accepted for the same reason.

### REMOVE is the strongest option, and that is accepted

Thinning a 60-card deck against a fixed hand of eight raises consistency every time — **more so
since arcane**, which added twelve cards without adding a card to the hand. It is
deliberate rather than unnoticed: the offer is two worms out of a growing catalogue, so removal
being the best of what is on the table is only sometimes the question. **`duplicate` is the one
most likely to need a cost** — copies are the sharpest dial in the game, since four of one concept
in a turn is a Four of a Kind.

### Where the deck lives, and why a card has no identity

**`internal/session` holds the deck**, because the combat screen rebuilds its piles on every
`Init` and `Init` is how the next fight starts — so anything held on that scene is thrown away
between rooms.

**No card identity, and that is a consequence of *when* alteration happens.** Between fights no
pile is live, so an offer is a list of positions in the run deck and a position is unambiguous for
as long as the screen is up. Mid-fight alteration would need a real ID *and* a field on every
event, since the log rebuilds a card from what an event carries.

### The between-fight chain

Post-battle is the first of several scenes between one room and the next: **alteration**, then a
**shop** where vitae is spent, then a **room or stairway choice** between two doors. Each is an
ordinary scene in the registry rather than a mode of the combat screen, and **`session.Phase` is
what decides the order** — see `internal/session/flow.go` for the chain and
`internal/screens/flow.go` for which scene draws each station. The room choice has no scene yet
and is walked past.

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

The currency. Earned from fights, spent on rings. `Session` carries the purse; **winning a fight is
the only thing that adds to it** and **the shop is what takes it out** *(2026-08-21)* —
`Session.SpendVitae` is the one place a purse goes down, and `Session.Buy` is its only caller.

**Vitae is crimson wherever it is written** *(owner's call, 2026-08-22)* — the purse on the duelist
card and the word itself in the reward screen's prose. It is the run's only currency and now the
only red on a cream screen, so a figure in that colour says "money" before it is read.

### What a win pays *(owner's call, 2026-08-22)*

Three separate things, decided by `Session.WonFight` and handed over one at a time by the reward
screen as it narrates them. See `internal/session/spoils.go`.

| Part | Figure |
|---|---|
| **Interest** | propagation, below — on the purse as it stood when the fight ended |
| **The life you kept** | **a tenth of the life remaining, rounded down**: 65 left pays 6 |
| **The room** | **3** outer, **4** inner, **5** stairway (the floor's boss), flat for the whole climb |

- **A share of the life *remaining*, not of the maximum.** It is a reward for fighting well rather
  than a rebate, and a ring that raises max life pays out more here indirectly — which is intended.
  A win on nine life pays nothing from this part.
- **The room award does not scale with the floor.** What makes a later fight worth more is the life
  you manage to keep in it.
- **Deciding and paying are separate.** The figures are frozen when the fight ends, so nothing
  about the payout depends on when the player clicks; `Session.Advance` claims whatever was never
  narrated, which is what makes a win pay in full even when no screen reads it out.
- **Soul Taker moves the room award** *(retargeted 2026-08-22)*, turning 3/4/5 into 8/9/10. It used
  to pay the vitae prize card, which no longer exists — same `prizes-dealt` moment, same flat
  addition, landing on the one figure a win always pays.

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
- **It is decided in `Session.WonFight`**, before the room counter moves and before either award
  above: interest is on what the run walked out of the fight holding, not on what the win is about
  to pay it. **The figure is decided there and arrives on the reward screen**, when the sentence
  naming it has been read.

---

## The tower

**8 floors × 3 fights.** Fixed layout, drawing no randomness — what is *in* it is random, the
shape is not. *(ideas.md's "one enemy per level" is superseded.)*

- **Every third fight is a floor boss, and the bosses are their own catalogue** *(2026-08-23)*.
  `data/bosses.json` holds thirty named stairway protectors, three or four authored per floor, and
  a run draws one for each floor when the climb is rolled — so a floor's stairway is a face the
  player can be told about rather than a creature from the same roster with bigger numbers. They
  never stand in an outer or inner room, and a stairway does not consume a roster entry.
- **A boss is pitched above the floor it guards**: more HP than the toughest enemy that can appear
  there and more DMG than the hardest hitter, on top of the ascent curve, which scales it like
  anything else. Bosses are durable — high `HP`, and earth on them if a boss should also blunt
  damage — with one strong attribute. They cannot spawn enemies, which implies normal enemies can,
  a mechanic recorded nowhere else and otherwise undefined.
- `[?]` Whether a boss should carry an affix of its own by default, and whether the several bosses
  authored for one floor should differ in shape rather than only in name and picture — today their
  decks are one template at five rungs.
- **After fights 1 and 2: a choice of two doors.** After the boss: **a choice of stairwell.**
  Captured as two concepts even though the mechanic is likely the same, because one is "next
  fight on this floor" and the other is "next floor" — a real difference to hang divergence on.
- **Doors hint at what is behind them.** Cold coming off the door for an ice enemy, smoke for
  fire — the shape of what is coming without its name.
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
×8.9 by floor 8's stairway. **What that costs a player is unmeasured.**

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
  person and must not learn — a headless simulation plays both sides.
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

### The deep tower is meant to need a build *(2026-08-16, owner's call)*

Per-enemy decks, doubled enemy HP, enemies no longer forming hands and the 10% ascent curve all
landed on top of each other and **none of them was absorbed by a retune**, which put the deep floors
out of reach of a duelist wearing nothing. **That is the intent rather than a regression**: the
player's ceiling is *supposed* to move and rings are how, so a bare fighter is not who those floors
are priced against and **the whole ascension is not expected to be winnable yet**.

**A wall on a *shallow* floor is a different thing**, and is still a failure — the player has bought
nothing by then.

### The count bound moved into the rules

`maxSelected` left the screen and became `Duelist.MaxActions()`. It had to: **the opponent's
planner obeys the action cap exactly as the player's selection does**, and a cap enforced only
by the screen was a cap the enemy ignored. A method rather than a constant, so a ring raising
it has somewhere to bite — which is what this file asked for.

---

## The profile — what survives a run

**A run dies; a profile does not.** The tower is the run — the deck, the purse, the worn rings, the
room you are in — and the profile is the thin layer that outlives it: whether the tutorial has been
watched, what has been achieved, what has been unlocked. Standard roguelike shape, and the reason
the two are separate files on disk rather than one.

**Two files, in the platform's config directory** *(owner's call, 2026-08-25)* — `profile.json` and
`run.json`, under `os.UserConfigDir()`. Never beside the executable: the game is meant to be sold on
Steam, which installs into a tree a normal user process cannot write to, and a per-executable
directory is also per-install rather than per-user. `internal/profile`'s doc comment holds the full
argument.

**Achievements and unlocks are different animals, and the file keeps them apart.** An *unlock*
changes the game — it is an input to the rules, so something in the rules reads it. An *achievement*
changes nothing; it is a record, and it is the thing that will eventually be mirrored to Steam,
which makes its key an external contract that cannot be renamed once shipped. They share a file and
never a field.

**`first-steps` is the first achievement: defeat an enemy.** It fires on every win and the profile
keeps one, so it means the first enemy the player ever beats rather than the first of a run — a
player who loses room one fifty times gets it on the fifty-first. Nothing shows it yet; see TODO.md.

**A run is saved at every phase transition and never inside a duel** *(owner's call, 2026-08-25)*.
Between stations the run is quiescent — no piles dealt, no queued actions, no hidden hand — so the
snapshot is a dozen fields rather than the whole combat screen's working state. The cost is stated
rather than discovered: quitting mid-duel loses that duel and resumes at the top of the room, which
is what `Retry` already does after a defeat.

**Resuming is not replaying, and the distinction is load-bearing.** A run is *not* replayable from
its seed alone — a deck edit is a choice, so replay would need a seed plus a choice log — but
resuming does not need the path, only the state. So the snapshot is state, and it may never be used
as a replay. See the Randomness section below, which is unchanged by this.

**The climb is not saved; it is rebuilt from the run code.** The fight order is a function of the
run seed, so storing the seed keeps one answer to "who stands in room four". **This stops being true
the day the room-choice screen lets a player pick what is ahead**, at which point the climb becomes a
choice and has to be written down. `TestTheClimbIsRebuiltFromTheSeed` is what fails on that day.

**Nothing about the profile is ever fatal.** A missing file is a new player, a corrupt file is a new
player, an unwritable directory is a session whose progress is not recorded — the same rule the audio
device is under. A game that refused to launch over a save file would be a worse bug than any it
could prevent. A file written by a newer build is read but never written over, which is the one
mistake that cannot be repaired afterwards.

**What is written down is a name, never a number.** Every ordinal in this game is append-only and
index-shaped — `ConceptID`, `Element`, `StatusID`, `GlyphKind`, `Phase` — so an ordinal in a file
that outlives its build is an ordinal that will eventually mean something else.

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

- `[?]` **Every same-concept hand shows all-distinct colours**, because the deck holds one copy per
  concept per colour. It no longer costs a multiplier, but it does mean a built hand always lands
  every status the player is ringed for — the colours are not a choice.
- `[?]` **The shock roll is conditional**, against a written rule that it should be unconditional.
  Settle it before the save format lands.
- `[?]` Duration, stacking and refresh for every status.
- `[?]` Whether ring cards may be shorter than action cards, given they have no glyphs.
- `[?]` What distinguishes one stairwell from another.
- `[?]` Whether the shop and door choice are one screen or two.
- `[?]` Whether earth becomes a floor affix.
- `[?]` Earth's green collides with `playerSwatch`. One of the two schemes has to give, and
  what is holding it off is that a border and a swatch are never seen side by side.
- `[?]` How enemies scale up the tower.
