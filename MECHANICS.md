# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

Captured 2026-08-05 in one session. Nothing below is implemented yet except where noted.

---

## The thrust

**The primary thrust of the game is building a deck and an engine that bend the rules in a
way that lets the player win.**

Rings and brands are rule-modifiers first and stat-boosters second — more actions, cheaper
cards, free cards, and stats too, since a stat is just another rule to bend. Every constant
below is a candidate for something to bend.

The consequence for the code: rules cannot stay `const`. `baseActionPoints`, `speedPerPoint`,
the per-action costs, `guardDivisor` and the actions-per-round cap are compile-time constants
today, read by functions with no access to the run. They need a **carrier** — a modifier set
passed alongside the duelists that `internal/combat` reads instead of the constants. Cost
becomes a function of the card *and* that carrier, the way `Damage` is already a function of
the action and the wielder.

Attributes do **not** need this. `Con`, `Str` and `Spd` are already fields on `Duelist`, and
`ResolveRound` takes duelists by value, so a ring granting `+5 Str` just hands it a different
duelist. Base values live in `data/combatants.json` and are expected to move with playtesting.
What is still frozen is the *conversion* — `LifePerCon`, `baseActionPoints`, `speedPerPoint` —
and those are the bigger balance levers.

---

## Attributes and scaling

**Implemented today:** `Con`, `Str`, `Spd` on `Duelist`. Life is `Con × LifePerCon` (5). Action
points are `4 + Spd/10`, minimum 1. Damage comes from the *action* via `ActionKind.Damage(str)`,
not from an attribute of its own. Base values are per-combatant data in `combatants.json`.

**`ideas.md` proposed a different model** — attributes are *speed*, *damage* and *armour*, each
scaling with level: damage `level × 10`, speed `level × 5`, armour likewise. Two conflicts with
what exists, both unresolved:

- **There is no armour, anywhere.** `Duelist` has no such field and nothing reads one. The only
  damage reduction in the game is `guardDivisor`, which halves a blow against a raised guard.
- **Nothing scales with floor.** Enemies are fully-specified records; there is no level term.

`[?]` **Whether armour arrives at all, and if so whether it is a stat or a status.** Worth
noticing before deciding: **earth is already armour by another name** — "blunts the damage the
opponent deals, by a percentage" is exactly what an armour stat does. If armour lands as a
separate stat, the two need to be one system rather than two mechanics that quietly stack.

`[?]` **How enemies scale up the tower**, given the records are flat and the ideas note assumed
a level multiplier.

**This matters for bosses**, which are recorded below as having "high armour" — a rule resting
on a stat that does not exist.

---

## Cards

### Types

Every card is exactly one of three types:

| Type | Concepts |
|---|---|
| **setup** | Prepare |
| **attack** | Strike, Heavy |
| **defend** | Guard, Parry, Dodge, Riposte |

Type is a property of the *concept*, not an independent axis — a fire Guard and a basic Guard
are both defend.

Riposte is **defend**, despite being a counter-attack.

The split is deliberately lopsided: 1 setup, 2 attack, 4 defend, so a starting deck is
5 setup / 10 attack / 20 defend. Two thirds defensive in a game won by reducing the opponent
to zero, on the theory that defends convert into damage.

### Concepts and deck composition

**A concept ships as five cards: basic, plus one per primary element.** That is the rule for
adding concepts, not just a description of the starting deck — a new concept arrives as a set
of five.

Seven concepts × five = **35 cards**. A hand of eight against that makes the draw a real
decision.

`Quick` exists in `internal/combat` as an `ActionKind`, is in no deck, and is not a concept.
It remains homeless.

### Long press

Long press on a card reveals the whole card, un-occluded by the overlap. This is what the
recorded input vocabulary already assigns to long press.

Hover was considered and rejected in favour of it. The distinction to preserve if hover ever
returns: **hover un-occludes, long press explains.**

A press therefore becomes a three-way decision — move past `dragThreshold` is a drag, held
past a tick count without moving is a long press, released before either is a click that
toggles selection. The distance and time thresholds must not fight each other.

---

## Elements

### The set

| Tier | Elements |
|---|---|
| **primary** | ice, fire, lightning, earth |
| **secondary** | poison, force, hunger |

Only primaries get cards. Where the secondaries appear — rings, enemies, brands, higher-tier
cards — is `[?]` and not assumed.

`basic` is the absence of an element, not a fifth colour. It replaced `none`/`plain` in the
code's naming.

### Colour

| Element | Colour |
|---|---|
| basic | near-white |
| fire | orange |
| ice | medium blue |
| lightning | yellow |
| earth | brown |
| poison | green *(currently in the deck; leaves when poison becomes secondary)* |

Two collisions are live and unresolved: **lightning yellow is also `enemySwatch`**, and
**poison/earth green-brown sits near `playerSwatch`** — "green is you, yellow is them" is
written down as a screen-wide rule and elements break it. Either the sides stop being
colour-coded or the elements avoid those hues.

### Statuses

Elements are **mechanical**, as always intended. Each applies a status **to the opponent**:

| Element | Status |
|---|---|
| **ice** | reduces the enemy's AP |
| **lightning** | introduces a chance to miss |
| **fire** | damage over time — persists a set duration, lands **at end of round** |
| **earth** | blunts the damage the opponent deals, by a percentage |

Duration, stacking and refresh are `[?]` for all four.

Notes on each:

- **Ice is the AP element in both directions.** The ice *ring* discounts your ice cards; the
  ice *status* cuts the enemy's budget. Same element, opposite targets, deliberate.
- **Fire needs state that outlives an action.** `ResolveRound` today only produces damage as a
  consequence of an action; a DoT ticks at a point in the round owned by nobody's action and
  persists across the boundary. New event kind, and `advancePlayback` grows a case.
- **Earth is the first percentage** in a package documented as pure integer arithmetic.
  Workable — `guardDivisor` already halves with integer division — but the rounding rule has
  to be stated, not left to `/`.
- **Statuses need a home.** `Duelist` has exactly one today, `Guarded bool`. Four statuses
  with amounts and durations is a status *system*, and combos both read and consume them.

**Poison has lost its obvious job.** Fire is the damage-over-time element now. Poison, force
and hunger have no statuses.

---

## Resolution — the experiment

**Decided 2026-08-05 as an experiment, and as the direction to head in.** Not a placeholder,
not settled either: try it, then reconsider.

A round resolves in **phases**, not by interleaving:

1. the player's **preparations**
2. the player's **attacks**
3. the player's **defenses**
4. then the enemy, with their attacks

Defenses are front-loaded on the assumption that the enemy goes last, so a defense is up when
the blow arrives.

**Why:** the interleaving may not be possible for players to grok. That is the whole reason.
It also simplifies — actions can be gathered into their categories inside resolution, and
hidden information survives untouched, because `ResolutionOrder` is a single pure function
that both `ResolveRound` and the Resolution pane read.

### What this costs, recorded honestly

- **It reverses a decision made 2026-07-31**, when volley-per-side was replaced by alternation
  on the grounds that *"two monolithic volleys gave the player nothing to manipulate."*
  Phase-based is not the same as volley-per-side — it groups by category rather than by side —
  but it is closer to the rejected thing than to what ships.
- **Cross-phase reordering stops meaning anything.** A defense cannot be dragged ahead of an
  attack. **Within-phase ordering still matters**, and matters more than before — see combos.
- **Guard persistence dissolves.** *"A raised Guard lasts until its owner's next action"* stops
  describing anything if every defense resolves before every enemy attack, and
  `TestGuardHoldsWhileItsOwnerDoesNothing` stops testing anything.
- **It changes what makes stagger rare** — see below.

`ResolutionOrder` being one pure function is what makes this cheap: one function body plus its
tests. Three other candidate models are recorded in `TODO.md`; this is a fourth.

---

## Combos

Combos are **discovered**, not given, and discovery persists on the **profile** — part of the
roguelike unlock structure, not the run.

They come in two shapes, which need different matching:

### Count-based

**Unopposed stagger** *(decided 2026-08-04, recorded in `TODO.md`)*. Three attacks in a row
without the opponent answering staggers them and costs them an action. Five knocks out their
whole round.

- The original rationale was that **alternation made this rare**. Under phase resolution it is
  not rare for that reason — but it stays rare for others, and those are the reasons now:
  **five Strikes in a 35-card deck means P(3+ in a hand of eight) ≈ 7%**; three Strikes is
  exactly 6 AP, Fighter1's entire budget; and **five Strikes is 10 AP, unreachable without
  ring discounts.**
- That last point is the design working as intended: the five-combo is something you *build
  toward*, not something you draw into. It is gated behind engine-building, which is the
  thrust of the game.
- Later enemies are expected to absorb mechanics like this.

### Sequence-based

An ordered pair of elements, where **order changes the result**:

| Sequence | Name *(placeholder)* | Effect |
|---|---|---|
| ice Strike → fire Strike | Burnt Icecube | doubles the DoT for that round |
| fire Strike → ice Strike | Extinguishing Strike | fires the full DoT as one critical hit |

**This is what earns drag-to-reorder its place back** after phase resolution took cross-phase
ordering away. Same two cards, opposite order, different mechanic.

Extinguishing Strike **consumes a status to convert it** — the first mechanic that spends a
status rather than applying or expiring one. Likely a family of its own.

### Requirements

- **Combos are rules and live in `internal/combat`**, matching on the resolved order. The
  screen must never derive one; that is what makes the Resolution pane structurally incapable
  of lying about the round.
- **A `KindCombo` event** carrying which combo fired and which slots formed it.
- **A place to browse discovered combos** — a reference the player can return to. Probably
  belongs with the profile rather than inside a duel.

---

## A round is bounded twice

**By cost and by count, independently and on purpose.**

- **AP budget** gates what can be afforded.
- **A hard cap on actions per round** gates how much can happen at all, and holds even when
  everything is free.

The cap is five today (`maxSelected`, currently in the screen). It is a **rule**, not a UI
detail, and belongs alongside `ActionPoints()` — and it should stop being a constant, since
brands and rings raising it is an obvious reward.

Discounts **can take a card to free**, which is what makes the count bound load-bearing rather
than incidental.

---

## Rings

- **Bought after every fight, with vitae.**
- **Five at once**, until brands expand capacity. *(ideas.md's "extra fingers bought from a
  shop" is superseded.)*
- **The cap is never displayed.** It surfaces naturally when you try to buy a sixth.
- **A ring per element discounts cards of that element** — an ice ring makes an ice Strike cost
  one less. Not a budget increase; a per-card discount. This is the expensive branch and it is
  the chosen one.
- Rings also boost stats and bend other rules.

**Discounting matching cards is why element must cross into `internal/combat`.** Cost stops
being a property of the card and becomes a property of the pairing, so `Cost()`, `CostOf()`,
the queue type, `ResolveRound`, `ResolutionOrder` and `PlanGreedy` all grow it — plus the
screen's AP bar, over-budget check and caption. This cost was written down in advance in the
elements entry and is now due.

**Rings are drawn as cards**, in a horizontal row across the top, not necessarily spanning the
whole bar. Same size as other cards, and **no glyphs**.

- Five at 180px wide is 948 including gaps — fits horizontally.
- **It does not fit vertically.** A hand-size card is 264 tall, the bottom band already runs
  566→937, and that leaves ~270px for the character block, Resolution pane, caption and enemy —
  where the Resolution pane alone is currently 326.
- Dropping the glyphs is exactly what *would* free the height: the glyph column is the floor on
  card size. `[?]` Same width and shorter is the obvious escape, but "same size" was stated.
- `drawCard` takes an `actionCard` and builds a glyph column from `badgesFor`. A ring has no
  damage, initiative or cost, so either that generalises or rings get their own drawer sharing
  the frame and colour logic.

**Rings are the first thing that genuinely needs `Session`.** They survive fights;
`CombatScene` state does not.

Art note: `fire-ring.png`, `frozen-ring.png` and `thunder-ring.png` are already embedded and
unused. Earth has none.

---

## Brands

Brands of power expand capacity and bend rules — more rings, more actions, cheaper or free
cards, stat boosts. Otherwise undefined.

Like rings, they have **concrete definitions that never really change**, which makes them a fit
for the `data/` pattern: JSON beside a small Go loader.

---

## Vitae

The currency. Earned from fights, spent on rings.

It is already on screen in the character block, reading a fixed `5` as a placeholder. It moves
to `Session` when that exists.

---

## The tower

**8 floors × 3 fights.** Fixed layout, drawing no randomness — what is *in* it is random, the
shape is not. *(ideas.md's "one enemy per level" is superseded.)*

- **Every third fight is a floor boss.** Bosses have high armour and one strong attribute, and
  cannot spawn enemies — which implies normal enemies can, a mechanic recorded nowhere else and
  otherwise undefined.
- **After fights 1 and 2: a choice of two doors.** After the boss: **a choice of stairwell.**
  Captured as two concepts even though the mechanic is likely the same, because one is "next
  fight on this floor" and the other is "next floor" — a real difference to hang divergence on.
- **Doors hint at what is behind them.** Cold coming off the door for an ice enemy, smoke for
  fire. This is graded reveal one level up from `concealedLabel`, and probably wants the same
  answer as the in-combat version.
- **Generate both doors, always.** Rolling only the chosen one shifts every subsequent draw in
  the run.

`[?]` What distinguishes one stairwell from another. `[?]` Whether the shop and the door choice
are one screen or two, and in which order.

[ascend.go](internal/screens/ascend.go) is a stub whose comment already describes this.

---

## Enemies

**Enemies have a deck**, smaller than the player's. This answers an open question in `TODO.md`
in the affirmative.

**An affix transforms the deck rather than adding to it.** A brute has basic attacks; a *fire*
brute on a fire floor has all fire attacks. The deck stays one list and the affix maps
`basic → fire` across it. This is cheaper than the recorded "affixes become cards shuffled in".

- **Baseline decks are data**, alongside the existing `combatants.json` records.
- **Affixes are renamed to match the elements**: `hot`/`cold`/`charged` become `fire`/`ice`/
  `lightning`. Two vocabularies for the same things was a collision waiting to happen.
- **`undying` is parked**, not deleted. Revisit later.
- `[?]` Earth has no affix — either it joins the list or there is a reason it cannot be a floor
  theme.

**`PlanGreedy` stops working as written.** It plans from a `Duelist` and a fixed set of four
actions; with a deck it has to draw a hand and plan from that, which subjects the enemy to the
same "what did I draw" pressure the player faces.

---

## Randomness

The determinism rules in `CLAUDE.md` still hold. Two additions from this session:

**Lightning puts randomness into combat**, which the rules pre-gate rather than forbid: it
arrives as an injected `*rand.Rand` on `ResolveRound`, never a global, and
`TestRoundIsDeterministic` changes shape — becoming stronger, since same seed plus same inputs
must produce the same log.

**Roll unconditionally per attack, not only when lightning is present.** A conditional roll
means adding or removing a status shifts every later roll in the run, so a balance tweak
invalidates every stored seed. Rolling always and ignoring the irrelevant result costs nothing.

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
  leave. The deck overlay today; a combo reference later. Its exit must be the brightest thing
  on screen or it is a trap.
- **Transient** — flashes over a frozen screen, takes no input, dismisses itself. The **COMBO**
  splash. Needs no exit affordance precisely because it accepts nothing.

**The freeze is already built.** `dwellFor` decides how long each event holds the screen, so a
`KindCombo` event with a splash-length dwell gets a frozen screen for free and the splash draws
while the playback cursor rests there. Presentation-only, so it cannot touch the outcome, and
splash length joins the pacing constants destined to become the game-speed setting.

`[?]` **The Resolution pane has no way to draw "these rows together did a thing."** It is one
row per slot with a single highlight walking down. A sequence combo spans two or more slots
that need not be adjacent. Bracket them, join them, or collapse them for the splash — undecided,
and worth solving before the pane grows further.

---

## Open questions

Collected from above.

- `[?]` Where secondary elements (poison, force, hunger) appear at all.
- `[?]` Duration, stacking and refresh for every status.
- `[?]` Whether ring cards may be shorter than action cards, given they have no glyphs.
- `[?]` What distinguishes one stairwell from another.
- `[?]` Whether the shop and door choice are one screen or two.
- `[?]` Whether earth becomes a floor affix.
- `[?]` How the Resolution pane draws a combo spanning non-adjacent rows.
- `[?]` Element colours collide with the side colours (`playerSwatch` green, `enemySwatch`
  yellow). One of the two schemes has to give.
- `[?]` Whether the setup/attack/defend types are the same axis as the *role* taxonomy the
  initiate/respond model in `TODO.md` asks for, or orthogonal to it.
- `[?]` Whether armour exists, and whether it is a stat or is just what earth already does.
- `[?]` How enemies scale up the tower.
