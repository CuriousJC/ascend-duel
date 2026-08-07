# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

Captured 2026-08-05 in one session; most of it is still unimplemented. **Cards, categories and
phase resolution landed on 2026-08-06** and those sections describe running code — they say so.
Everything else is still design.

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

**Implemented 2026-08-06 as `combat.Category`.** Every card is exactly one of three types, and
the type decides which phase of a turn it resolves in — see *Resolution* below. This is the
axis the whole round is built on.

| Type | Concepts | AP | Effect |
|---|---|---|---|
| **setup** | Prepare | 1 | Banks +2 AP for the next round |
| | Guard | 3 | Halves every attack in the opponent's next turn |
| **attack** | Strike | 2 | `str` |
| | Heavy | 4 | `str × 2` |
| **defend** | Dodge | 2 | Negates the first incoming attack |
| | Riposte | 3 | Negates the first incoming attack and hits back for `str/2` |

Type is a property of the *concept*, not an independent axis — a fire Guard and a basic Guard
are both setup.

**Guard is setup, not defend.** It moved on 2026-08-06 and went 2 → 3 AP with the change: it
does not answer one blow, it dampens a whole turn, and that is a thing you put up before the
exchange rather than a reaction inside it. Riposte is **defend** despite being a
counter-attack.

**Parry was dropped** the same day, before it was built. Dodge, Riposte and Guard already
cover cheap-precise, expensive-punishing and broad-dampening; a fourth defence had no job
left. It can come back.

The lopsided 1/2/4 split is gone with it, and so is the two-thirds-defensive theory it
carried. Six concepts split evenly 2/2/2 — 10 setup / 10 attack / 10 defend — which is a
consequence of the concept list rather than a target.

**Ripostes are spent before Dodges** when both are up. Both negate completely, so spending
the one that hits back first costs nothing, and it lands the counter early enough to kill the
attacker mid-turn.

`[?]` **Dodge and Riposte are a tier, not a choice.** Riposte is exactly Dodge plus a Quick,
priced exactly as the two together (2 + 1 = 3), so it strictly dominates Dodge whenever it is
affordable — Riposte is only *better* than Dodge, never *different*. What stops it being
redundant is the cap on actions per round: one card doing two jobs beats two cards. Being
played with deliberately before changing anything.

### Concepts and deck composition

**A concept ships as five cards: basic, plus one per primary element.** That is the rule for
adding concepts, not just a description of the starting deck — a new concept arrives as a set
of five.

Six concepts × five = **30 cards**, implemented. A hand of eight against that makes the draw a
real decision. Every concept named here now exists; the deck is complete rather than a
fragment of a longer list.

`Quick` exists in `internal/combat` as an `ActionKind` costing 1 for `str/2`, is in no deck,
and is not a concept. It remains homeless — but Riposte's counter-damage is deliberately
defined as the same figure, so it is at least a named quantity now.

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

## Resolution — phases

**Implemented 2026-08-06.** Chosen as an experiment on 2026-08-05 and built the next day; it
is what ships. Still open to reconsideration, but it is the model now, not a proposal.

A round is **a whole turn each**. Everything one side queued resolves before the other side
does anything, and within a turn the categories go in order:

1. that side's **setups**
2. that side's **attacks**
3. that side's **defenses**
4. then the other side, the same way

Defenses come last *within a turn* because the opponent moves next, so a defense raised at the
end of your turn is up when their blow arrives.

**Why:** the interleaving may not be possible for players to grok. That is the whole reason.
It also simplifies — actions are gathered into their categories inside `ResolutionOrder`, and
hidden information survives untouched, because that is a single pure function which both
`ResolveRound` and the Resolution pane read.

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

Removed 2026-08-06, whole. With one contiguous turn per side there is no exchange for a faster
action to lead, so initiative was a number on every card reporting a distinction the resolver
had stopped making. `Spd` still buys action points and still never buys priority.

Ordering *within* a category is queue order, which is what drag-to-reorder now moves and what
sequence combos will match on. See `TODO.md` for what would have to be true to bring
initiative back.

### What this cost, recorded honestly

- **It reverses a decision made 2026-07-31**, when volley-per-side was replaced by alternation
  on the grounds that *"two monolithic volleys gave the player nothing to manipulate."*
  Phase-based is not the same as volley-per-side — it groups by category within a turn — but
  it is closer to the rejected thing than to what alternation was.
- **Cross-phase reordering means nothing.** A defense cannot be dragged ahead of an attack.
  **Within-phase ordering still matters**, and matters more than before — see combos. That is
  the whole of what dragging a card along the row now does.
- **Guard persistence dissolved**, as predicted. The old *"a raised Guard lasts until its
  owner's next action"* and the deliberate quirk that an idle duelist kept its guard are both
  gone, along with `TestGuardHoldsWhileItsOwnerDoesNothing`.
- **It changes what makes stagger rare** — see below.

`ResolutionOrder` being one pure function is what made this cheap, exactly as hoped: one
function body plus its tests, and both consumers followed without being touched. Three other
candidate models are recorded in `TODO.md`; this is a fourth.

---

## Combos

**This is where the game is meant to be.** Throwing whatever you drew at the opponent works;
*choosing a shape* and building a deck toward it is meant to work better. Combos are the
mechanism that pays for that choice, and they are expected to grow to dozens of entries — so
they are a **framework with one pattern**, not a pile of special cases.

Combos are **discovered**, not given, and discovery persists on the **profile** — part of the
roguelike unlock structure, not the run. No profile exists yet, so every combo is currently
live; when one does, discovery gates the *table* and nothing else changes.

### The pattern: cards in, one of three rewards out

*Framework implemented 2026-08-07 in `internal/combat/combo.go`.*

Every combo is the same shape. **A run of cards that must appear consecutively in your own
turn**, and an `Effect` saying what forming it buys. A step matches either an exact card
(`Card(Strike)`) or any card of a category (`AnyOf(CategoryAttack)`), which is what lets "three
attacks" mean three attacks rather than three Strikes.

The reward vocabulary is **deliberately small and closed**:

| Effect | Means | Arrives |
|---|---|---|
| `DamageNum`/`DamageDen` | a damage multiplier, as a fraction | rest of that turn |
| `BankAP` | action points banked | the round after |
| `Stagger` | the opponent loses actions | their next turn |

Adding a combo is one entry in `comboTable`. Adding a new *kind* of reward is a field on
`Effect` and one place that applies it — the cost the framework deliberately charges, because a
reward vocabulary that grows without limit is one no player can hold in their head.

Four rules that make it work, each of which had an alternative:

- **Matching is on the resolved order, never the queue.** Phases regroup a queue by category,
  so `Strike, Dodge, Strike, Strike` resolves as three consecutive attacks and *does* combo.
  Matching the queue would let the Resolution pane show one thing and the engine score another.
- **Matching is on cards used, never on what they achieved.** The combo is known before the
  round is played, which is what lets a multiplier boost the cards that formed it rather than
  arriving after they have landed. It also means the opponent's defenses cannot silently
  invalidate a plan the player already committed to.
- **Effects come into force at the combo's first card**, for the rest of that turn — and
  `KindCombo` is emitted there, so the screen narrates cause before effect. Firing on the card
  that *completes* the run was rejected: a multiplier could then only ever boost cards after
  the combo.
- **Longest run first, and a card forms at most one combo.** Otherwise three attacks would
  score a Flurry at every position it fits, and five would never reach Onslaught.

`MatchCombos` is exported and is what the screen calls while the player is still planning, so a
previewed combo is the combo that fires by construction rather than by two pieces of code
agreeing.

### Count-based: the flurry/onslaught family

**One pair per attack card, generated rather than written out.** A new attack card gets its own
Flurry and Onslaught by existing — the family is a shape the game keeps, not a list somebody
has to remember to extend.

| Run | Name | Effect |
|---|---|---|
| 3 of one attack card | **`<Card>` Flurry** | opponent loses one action |
| 5 of one attack card | **`<Card>` Onslaught** | opponent loses their whole turn |

So today: Strike Flurry, Strike Onslaught, Heavy Flurry, Heavy Onslaught, Quick Flurry, Quick
Onslaught. **Heavy Onslaught is five Heavies at 4 AP each — 20 points, and near enough
impossible.** It is a rule anyway, deliberately, so that engine-building has something absurd
to aim at.

**Per card, not per category** *(decided 2026-08-07, after a category-wide version shipped
first)*. `AnyOf(CategoryAttack)` made any three attacks combo, so a Quick counted the same as a
Heavy and the reward went to whatever you happened to draw. Naming a combo for the card it is
built on is what makes it worth *building toward*: three Strikes is a deck you assembled. It
also leaves room for the effects to differ per card later — a Heavy Flurry has every reason to
hit harder than a Quick one, and a category-wide combo could never say so.

**IDs are derived from the card** (`FlurryID(a)`, `OnslaughtID(a)`), not from a position in a
list. Discovery persists on the profile, so an ID that shifted when a card was inserted would
silently re-lock combos the player had already found.

**"Unopposed" is gone from the name and from the rule** *(2026-08-07)*. It was written under
alternation, where the opponent could interleave and break a streak. Under phases every attack
you queue is consecutive by construction, so three attacks *is* three in a row and there is no
way for the opponent to interrupt. Both `[?]` questions this carried — whether a Guard resets
the streak, whether a zero-damage hit counts — **can never fire and are struck** rather than
left open.

What keeps it rare is the deck and the budget: three Strikes is exactly 6 AP, Fighter1's entire
budget, and **five Strikes is 10 AP**, reachable only by spending a whole round on Prepares.
That trade is the combo working as intended.

**Stagger takes actions off the front of the victim's next turn**, which under phases is their
setup phase — so being staggered costs a Prepare before it costs an attack, and leaves you
poorer next round as well as slower now. The action points are **not** refunded: stagger is
tempo *and* economy.

**It is symmetric, and one asymmetry falls out of phases.** Side A takes its whole turn first,
so a combo A forms bites B in the same round; the identical combo formed by B lands when A has
already acted and carries to the round after. That is why `Duelist.Staggered` persists across
the boundary — it makes the rule one rule (*a staggered duelist loses actions from its next
turn, whenever that is*) rather than two spelled differently per side.

**A stagger deletes cards before combos are matched**, so a staggered duelist cannot combo back
with a turn it never took.

### Sequence-based

An ordered pair of elements, where **order changes the result**. Not yet built; the framework
above matches them without extension, since `Card()` steps already express an exact sequence.

| Sequence | Name *(placeholder)* | Effect |
|---|---|---|
| ice Strike → fire Strike | Burnt Icecube | doubles the DoT for that round |
| fire Strike → ice Strike | Extinguishing Strike | fires the full DoT as one critical hit |

**This is what earns drag-to-reorder its place back** after phase resolution took cross-phase
ordering away. Same two cards, opposite order, different mechanic.

Extinguishing Strike **consumes a status to convert it** — the first mechanic that spends a
status rather than applying or expiring one. Likely a family of its own, and the first that
needs an `Effect` field beyond the three above.

### `[?]` Whether the enemy draws on this at all

**Open, and deliberately left open on 2026-08-07 rather than settled early.** Combos are
symmetric today: `Swarm1` forms a Quick Flurry off four Quicks every round and staggers the
player for it. `tools/balance` says that makes it **unbeatable by all three postures** — see
`TODO.md`, which is where the fix is tracked.

It is left standing on purpose, to watch how it balances out. **It is not settled that enemies
share the player's combo table, or even the player's cards.** An enemy built from a deck and an
affix could plausibly have attacks the player never sees and combos of its own, in which case
the symmetry here is a temporary convenience rather than a rule.

The pricing note worth keeping either way: every costing in this document reasons from the
player's budget — three Strikes is 6 AP, five is 10 — and `Swarm1` gets four Quicks for 4 AP
because `Quick` costs 1. **A combo counting cards is priced by whoever has the cheapest cards.**

### Requirements

- **Combos are rules and live in `internal/combat`**, matching on the resolved order. The
  screen must never derive one; that is what makes the Resolution pane structurally incapable
  of lying about the round.
- **A `KindCombo` event** carrying which combo fired. *Done.* It carries the `ComboID` and the
  screen looks the name up with `ComboByID`, so a combo renamed is renamed once.
- **`KindStaggered` counts as a slot in playback** even though nothing happened, or the
  Resolution pane's highlight runs a row short for the rest of the round.
- **A place to browse discovered combos** — a reference the player can return to. Probably
  belongs with the profile rather than inside a duel. `Combos()` exists for it to read.
- `[?]` The Resolution pane still cannot draw "these rows together did a thing", and it now
  also cannot show that a row was staggered out.

---

## A round is bounded twice

**By cost and by count, independently and on purpose.**

- **AP budget** gates what can be afforded.
- **A hard cap on actions per round** gates how much can happen at all, and holds even when
  everything is free.

**Done 2026-08-06.** The cap is five, and it is `Duelist.MaxActions()` beside `ActionPoints()`
rather than the screen's old `maxSelected` constant. It moved for a concrete reason as well as
a tidy one: **the opponent's planner has to obey it exactly as the player's selection does**,
and a cap enforced only by the screen was a cap the enemy ignored. It is a method rather than a
constant so a brand or ring raising it has somewhere to bite without touching a call site.

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
the queue type, `ResolveRound`, `ResolutionOrder` and every planner all grow it — plus the
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
  damage, cost or category, so either that generalises or rings get their own drawer sharing
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

### Styles — implemented 2026-08-06, and a step short of decks

`PlanGreedy` is gone. `combat.PlanStyle` replaced it with four behaviours, each a pure
function of a `Duelist`, chosen by a `PlanStyle` string on the data record:

| Style | Plans | Answers it badly | Answers it well |
|---|---|---|---|
| **brute** | biggest attack affordable — few, heavy blows | guarding | dodging |
| **swarm** | as many attacks as the round allows | dodging, guarding | racing it |
| **warden** | Guard, then attacks — halves what it takes | — | overwhelming it |
| **tactician** | banks with Prepare, then unloads a spike | guarding | reading the tell |

**Why this exists.** With one enemy that spent its whole budget on two big swings, two
defensive cards bought total immunity — a duel ran three rounds taking 0, 0 and 2 damage.
That is not a fault in Dodge's cost. **"Negates one attack" is priced against how many
attacks arrive**, so an opponent's *shape* is what the player is really buying answers to,
and one shape means one answer.

- **A style is not a deck.** It is the behaviour that will eventually plan *from* one, so it
  can stay when decks arrive rather than being replaced by them. Baseline decks and affixes
  above are still unbuilt.
- **The tactician's tell is the concealment scheme working.** A concealed row still shows its
  category, so a round of `??? (setup) ??? (setup)` says a spike is coming without saying
  what. That was not designed for this and turned out to be exactly what it needed.
- **`tools/balance` plays every posture against every enemy** through the real `ResolveRound`
  and prints who wins. It was written because the first roster shipped an unwinnable enemy:
  Warden1 halving all incoming damage at 120 life was a 24-round grind against a fighter who
  dies in 10. **Run it after touching any cost, stat line or planner.**
- `[?]` The roster is four enemies fought in a fixed order, with the screen advancing on a
  win. That is scaffolding standing in for the tower and it should be deleted when the tower
  arrives, not extended.

### The count bound moved into the rules

`maxSelected` left the screen and became `Duelist.MaxActions()`. It had to: **the opponent's
planner obeys the action cap exactly as the player's selection does**, and a cap enforced only
by the screen was a cap the enemy ignored. A method rather than a constant, so a ring raising
it has somewhere to bite — which is what this file asked for.

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
