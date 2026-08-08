# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

Captured 2026-08-05 in one session; most of it is still unimplemented. **Cards, categories and
phase resolution landed on 2026-08-06**, **combos on 2026-08-07**, and **the full 12-concept /
60-card deck on 2026-08-08** — those sections describe running code and say so. Everything else
is still design.

**Elements are the next piece of work, and three things wait on it.** Ring discounts, flip rings
and the five-of-a-colour combo all need `element` to cross from the screen into
`internal/combat`; none of them can be built until it does. That is the single highest-leverage
change left in this document.

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

**The concept set is a 3x4 grid: three types by four cost tiers, filled** *(decided and built
2026-08-08)*. Twelve concepts, and the grid is the reason there are twelve rather than however
many somebody thought of.

| Type | 1 AP | 2 AP | 3 AP | 4 AP |
|---|---|---|---|---|
| **prepare** | Gather | Sift | Guard | Ritual |
| **attack** | Jab | Strike | Feint | Heavy |
| **defend** | Brace | Dodge | Riposte | Mirror |

| Type | Concept | AP | Effect |
|---|---|---|---|
| **prepare** | Gather | 1 | Banks +2 AP for the next round |
| | Sift | 2 | Two extra cards leave the hand at random at the round boundary, before it refills |
| | Guard | 3 | Halves every attack in the opponent's next turn |
| | Ritual | 4 | Banks +5 AP for the next round |
| **attack** | Jab | 1 | `str/2`, minimum 1 |
| | Strike | 2 | `str` |
| | Feint | 3 | `str`, and strips one pending Riposte or Dodge **without triggering its counter** |
| | Heavy | 4 | `str × 2` |
| **defend** | Brace | 1 | Halves the next single incoming attack, then is spent |
| | Dodge | 2 | Negates the first incoming attack |
| | Riposte | 3 | Negates the first incoming attack and hits back for `str/2` |
| | Mirror | 4 | Negates **every** attack in the opponent's next turn and reflects each one's damage back at them |

Type is a property of the *concept*, not an independent axis — a fire Guard and a basic Guard
are both prepares.

**The grid is a design brief, not just a count.** An empty cell states a spec before anything
has a name — "a 3-cost attack" is something to solve. What it must not become is a cost ladder
where each tier is the one below it with a bigger number: that is twelve cards and three
decisions, and it is the trap `Dodge`/`Riposte` already half fell into. **Every tier differs in
kind.** Feint attacks the defence layer rather than the health total; Mirror scales with what
the opponent committed rather than with your own strength; Sift operates on the deck and not on
the arithmetic at all; Brace is partial where Dodge is binary.

**Ritual and Gather bank at the same rate on purpose** — 1 AP for +2 and 4 AP for +5 are both
net +1 per point. Ritual is not a better Gather, it is a Gather that does not eat your round:
four Gathers bank more (+8) but consume four of five action slots, while Ritual banks +5 and
leaves four slots to fight with. The difference is *slots*, which is the same reason Riposte is
not redundant against Dodge.

**Sift is the one concept whose effect is not a rule.** It manipulates the deck, and the deck
deliberately lives on the scene rather than in `internal/combat` — so `Sift` is an `ActionKind`
with a cost, a category and no engine effect, and the screen does the work at the round
boundary. That is a real division rather than an unimplemented card: the rules package owns
what a round does, not what you are holding. It also makes Sift the only concept `tools/balance`
cannot see.

**Sift and the Discard button are not the same thing.** Discard is *steering* — four a round,
you choose what leaves. Sift is *throughput* — more cards flow past you and you do not choose
which. It can eat a card you wanted, which is what makes it cost something beyond its 2 AP.

**Guard is a prepare, not a defend.** It moved on 2026-08-06 and went 2 → 3 AP with the change: it
does not answer one blow, it dampens a whole turn, and that is a thing you put up before the
exchange rather than a reaction inside it. Riposte is **defend** despite being a
counter-attack.

**Parry was dropped** the same day, before it was built. Dodge, Riposte and Guard already
cover cheap-precise, expensive-punishing and broad-dampening; a fourth defence had no job
left. It can come back.

The lopsided 1/2/4 split is gone with it, and so is the two-thirds-defensive theory it
carried. Six concepts split evenly 2/2/2 — 10 prepare / 10 attack / 10 defend — which is a
consequence of the concept list rather than a target.

**Ripostes are spent before Dodges** when both are up. Both negate completely, so spending
the one that hits back first costs nothing, and it lands the counter early enough to kill the
attacker mid-turn.

`[?]` **Dodge and Riposte are a tier, not a choice.** Riposte is exactly Dodge plus a Jab,
priced exactly as the two together (2 + 1 = 3), so it strictly dominates Dodge whenever it is
affordable — Riposte is only *better* than Dodge, never *different*. What stops it being
redundant is the cap on actions per round: one card doing two jobs beats two cards. Being
played with deliberately before changing anything.

**Feint narrows this without closing it** *(2026-08-08)*. Attacking into a Riposte normally
costs the attacker `str/2` in counter-damage; a Feint strips the Riposte and takes no counter,
so Riposte's punish clause is now something that can be defused and Dodge's cannot be — there
is nothing on a Dodge to defuse. That makes the two differ in what an opponent can do *about*
them, which is weaker than differing in what they do, so the `[?]` stands. Feint strips
Ripostes before Dodges, matching the order they are spent in.

### Concepts and deck composition

**A concept ships as five cards: basic, plus one per primary element.** That is the rule for
adding concepts, not just a description of the starting deck — a new concept arrives as a set
of five.

Twelve concepts × five = **60 cards**, implemented 2026-08-08. A hand of eight against that is
13% of the deck, against 27% when the deck was 30 — **doubling the deck halved consistency**,
which is exactly why Sift exists and why `discardsPerRound` is now a number to watch rather
than a generous placeholder.

**The deck list is data.** `data/cards.json` holds the twelve concepts and the elements each
ships in, loaded beside `combatants.json`; `startingDeck` is built from it. Cost, category and
damage stay in `internal/combat` — the dependency direction forbids the rules package reading
`data`, and cost is about to stop being a property of the card anyway (see *Rings*). The JSON
carries the cost tier as **documentation with a check**: the loader asserts every declared tier
against `ActionKind.Cost()` and fails loudly rather than letting two sources of truth drift.

`Quick` was renamed **Jab** on 2026-08-08 and given its five cards. It had been an `ActionKind`
with a cost and damage and no concept — "homeless" — because which of the rules' actions appear
in a deck was treated as a deckbuilding question. Filling the 1-AP attack cell answered it.
Riposte's counter-damage is still defined as the same figure, so "hits back for a Jab" is now a
sentence rather than a coincidence.

**52 was considered and rejected.** The playing-card instinct is real but it argues for 13 ranks
x 4 suits, and the fifth "suit" here is `basic` — which this document calls the absence of an
element, not a colour of its own. With `basic` a variant the deck lives on multiples of five and
52 is unreachable; without it, 13 concepts x 4 elements hits 52 exactly but every card carries
an element and nothing is plain. The grid decides it instead: 12 concepts is what three types by
four tiers produces, and 60 is what that costs.

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

**Poison has no cards and never did** — corrected 2026-08-08. Two places said otherwise (the
colour table above, and a comment in `combat_deck.go` claiming poison was in the starting deck
"because it predates the split"). `primaries` has only ever held basic, fire, ice, lightning and
earth, so `conceptDeck` has never built a poison card. The constant and its green exist and are
unused, which is fine; the claim that they were dealt was not.

### Colour

| Element | Colour |
|---|---|
| basic | near-white |
| fire | orange |
| ice | medium blue |
| lightning | yellow |
| earth | brown |
| poison | green *(reserved; no poison card is dealt — see below)* |

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

1. that side's **prepares**
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
- **A combo fires on the card that completes its run**, and its effects last for the rest of
  that turn. It read backwards the other way round — "COMBO!" above three strikes that had not
  happened yet — and, more importantly, **a run cut short still paid out**: matching happens up
  front, so a Riposte killing the attacker on the second of three strikes fired the combo off
  cards that never resolved. Completion-firing makes that impossible by construction.
  The cost: **a damage multiplier can only boost what follows the run**, not the cards that
  earned it. Nothing shipping uses one yet.
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

So today, one pair per attack card: **Jab, Strike, Feint and Heavy** — eight combos. **Heavy
Onslaught is five Heavies at 4 AP each — 20 points, and near enough impossible.** It is a rule
anyway, deliberately, so that engine-building has something absurd to aim at.

Two of those eight arrived by the family working as designed rather than by anyone adding them:
**Feint got a pair by being an attack card**, and **Jab's pair became reachable** the moment Jab
entered the deck, having been generated but undrawable while `Quick` was in no deck at all.

**Combo IDs shifted on 2026-08-08 and this was the last cheap moment for that.** `FlurryID` is
`comboFlurryBase + ComboID(a)`, derived from the card's raw enum value, and the five new concepts
had to be inserted *in category order* — the deck overlay sorts on that raw value to group the
piles the way a turn resolves. So Strike moved from 3 to 5 and its combos from 103/203 to
105/205. Harmless only because **no profile exists yet**, which is the very thing the derived-ID
scheme was designed to protect. Once discovery persists, inserting a concept mid-enum silently
re-locks combos the player has found, and the enum's category ordering and the ID stability
become a genuine conflict. `[?]` Resolve it before a profile ships — most likely by giving
`ActionKind` an explicit stable ID for combo purposes and letting the enum order serve the sort.

**Per card, not per category** *(decided 2026-08-07, after a category-wide version shipped
first)*. `AnyOf(CategoryAttack)` made any three attacks combo, so a Jab counted the same as a
Heavy and the reward went to whatever you happened to draw. Naming a combo for the card it is
built on is what makes it worth *building toward*: three Strikes is a deck you assembled. It
also leaves room for the effects to differ per card later — a Heavy Flurry has every reason to
hit harder than a Jab one, and a category-wide combo could never say so.

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
budget, and **five Strikes is 10 AP**, reachable only by spending a whole round on Gathers.
That trade is the combo working as intended.

**Stagger takes actions off the front of the victim's next turn**, which under phases is their
prepare phase — so being staggered costs a Gather before it costs an attack, and leaves you
poorer next round as well as slower now. The action points are **not** refunded: stagger is
tempo *and* economy.

**It is symmetric, and one asymmetry falls out of phases.** Side A takes its whole turn first,
so a combo A forms bites B in the same round; the identical combo formed by B lands when A has
already acted and carries to the round after. That is why `Duelist.Staggered` persists across
the boundary — it makes the rule one rule (*a staggered duelist loses actions from its next
turn, whenever that is*) rather than two spelled differently per side.

**A stagger deletes cards before combos are matched**, so a staggered duelist cannot combo back
with a turn it never took.

### `[?]` Count-based on element: the five-of-a-colour combo

**Five cards of the same element doubles your action points next round** *(added 2026-08-08, not
built)*. The first combo that counts an element rather than a card.

**It is an all-in round by construction.** The action cap is five, permanently, so five
same-element cards *is* your entire turn — there is no room for an off-colour card. And because
matching runs on the resolved order rather than the queue, a single off-element card can land in
the middle of the sequence after phases regroup it and break the run outright. Nothing else in
the game asks for a whole turn of one colour, which is what makes it worth a doubling.

Two things stand between this and existing, and the first is the more important:

- **`internal/combat` cannot see elements at all.** `ResolveRound` takes `[]ActionKind`; element
  lives on the screen's `actionCard`. A `Step` matches an exact card or a category and there is
  no element predicate to add one to. This is the *same* prerequisite ring discounts need — see
  *Rings* — so **one piece of work unblocks ring discounts, flip rings and this combo together**,
  and that is the argument for doing it next.
- **"Double your AP" is a new reward kind, not a table entry.** `BankAP` is flat and additive;
  doubling is multiplicative. Adding it is a field on `Effect` plus one place applying it — the
  cost the framework charges on purpose. `[?]` **Whether to pay it.** At the current 6 AP a
  doubling *is* `BankAP: 6`, which is free and needs no framework change; the two only diverge
  once `Spd` or a ring moves the base — and that divergence is arguably the reason to want the
  real multiplier, since a reward that scales with investment rewards building toward it.

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
symmetric today: `Swarm1` forms a Jab Flurry off four Jabs every round and staggers the
player for it. `tools/balance` says that makes it **unbeatable by all three postures** — see
`TODO.md`, which is where the fix is tracked.

It is left standing on purpose, to watch how it balances out. **It is not settled that enemies
share the player's combo table, or even the player's cards.** An enemy built from a deck and an
affix could plausibly have attacks the player never sees and combos of its own, in which case
the symmetry here is a temporary convenience rather than a rule.

The pricing note worth keeping either way: every costing in this document reasons from the
player's budget — three Strikes is 6 AP, five is 10 — and `Swarm1` gets four Jabs for 4 AP
because `Jab` costs 1. **A combo counting cards is priced by whoever has the cheapest cards.**

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
and a cap enforced only by the screen was a cap the enemy ignored.

**The cap is five permanently, and nothing may ever raise it** *(decided 2026-08-08)*. This
reverses the reason it was made a method — "so a brand or ring raising it has somewhere to bite"
— and the reversal is the point:

- **A fixed five is what makes hand concepts possible.** Poker hands exist *because* you always
  hold exactly five; that is what lets "a flush" be a permanent, learnable, nameable thing
  rather than a coincidence of how big your hand happened to get. The game is building toward
  named five-card shapes — three of one attack, five of one element — and every one of them
  needs the five to be a constant the player can plan against for the life of the run.
- **A growable cap would dilute every shape as it grew.** "Five of one element" is an all-in
  commitment at a cap of five and routine at a cap of seven. The combos would quietly get
  cheaper every time capacity went up, which is the opposite of a reward for building toward
  them.
- **It is still a method, and still should be.** Rings and brands need somewhere to bite for
  everything *else* they do, and a method that reads the duelist costs nothing. What changed is
  that this particular lever is off the table: **no ring, brand or combo raises `MaxActions`.**

The consequence for prepare cards: a 4-AP prepare cannot buy action slots, so Ritual buys
points instead (+5). An earlier draft of Ritual granted +2 AP and +2 slots specifically to reach
six- and seven-card combo runs; that is exactly the dilution above and it was cut.

Discounts **can take a card to free**, which is what makes the count bound load-bearing rather
than incidental — and with the cap frozen, a discount ring's ceiling is five free cards rather
than an ever-widening round.

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
- **No ring changes how many cards can be played.** `MaxActions` is frozen at five — see *A
  round is bounded twice*. A ring may make five cards cheaper, never make it six.

### Flip rings — the element-transform ring

**A ring that maps one element onto another across the whole deck** *(added 2026-08-08)*. A
"frozen lightning" ring turns every lightning card into an ice card, so a deck holding 12 of
each now holds 24 ice and no lightning.

**This is the ring the five-of-an-element combo needs to exist.** At 12 cards per element in a
60-card deck, drawing five of one in a hand of eight is a fluke you cannot build toward. A flip
doubles the pool and turns the combo into a deck you assembled — which is the whole stated
purpose of combos.

- **It is deliberately more powerful than a discount ring**, and that is accepted. The primary
  engine-building of this game is the interaction of rings, brands and how the deck has been
  altered; rings having very different magnitudes is what makes mixing them an act of judgement
  rather than an ordering.
- **It is the same primitive as an enemy affix.** An affix already maps `basic → fire` across an
  enemy's deck (see *Enemies*). One transform mechanism, pointed at either side — build it once.
- **It is cheaper to implement than the discount ring.** A flip is a pure transform on a card's
  element and never touches `Cost()`, so it does not require the "cost becomes a property of the
  pairing" rewrite. It still needs element to reach `internal/combat` for the combo to *match* on
  it.
- **Flips do not compose.** Every flip reads the card's *original* element, so lightning→ice and
  fire→ice both land on their own sources and cannot chain. Without that rule two flips could
  cascade a deck to a single colour and the order they were bought in would change the result.
- `[?]` Whether a flip's source element can be one that another equipped flip targets — allowed
  under the no-compose rule above, but it means two rings can both feed the same colour, which is
  a 36-of-60 monochrome deck. Watch it before deciding whether the cap is the ring slots or a
  rule.

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

**Brands alter the container; rings alter the contents** *(decided 2026-08-08)*. That is the
axis, and it is what tells you which of the two a new power belongs to:

| | Brands | Rings |
|---|---|---|
| What they touch | the chassis — hand size, total discards per round, ring slots | the cards — elements, costs, and the stats that feed them |
| Removable | **never.** You brand yourself and you do not take it off | freely; five equipped, swap as you like |
| Scope | **for the run** | for the run, but re-chosen after every fight |

- **"Permanent" means for the run, not across runs** — confirmed 2026-08-08. A brand is a
  commitment made *inside* a run that cannot be undone, which is a different thing from
  meta-progression. Combo *discovery* is the profile-scoped mechanic; brands are not.
- **More actions is struck from the list.** Brands were previously recorded as granting "more
  actions"; the action cap is now permanently five and nothing raises it. A brand growing hand
  size is the nearest legal thing and is a container change, so it fits the axis.
- Otherwise still open — capacity and rule-bending, with the above as the test for what counts.

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
| **tactician** | banks with Gather, then unloads a spike | guarding | reading the tell |

**Why this exists.** With one enemy that spent its whole budget on two big swings, two
defensive cards bought total immunity — a duel ran three rounds taking 0, 0 and 2 damage.
That is not a fault in Dodge's cost. **"Negates one attack" is priced against how many
attacks arrive**, so an opponent's *shape* is what the player is really buying answers to,
and one shape means one answer.

- **A style is not a deck.** It is the behaviour that will eventually plan *from* one, so it
  can stay when decks arrive rather than being replaced by them. Baseline decks and affixes
  above are still unbuilt.
- **The tactician's tell is the concealment scheme working.** A concealed row still shows its
  category, so a round of `??? (prepare) ??? (prepare)` says a spike is coming without saying
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

~~`[?]` **The Resolution pane has no way to draw "these rows together did a thing."**~~
**Resolved 2026-08-07, by splitting the pane in two rather than by solving it.** The old
Resolution pane was renamed **Action Flow** — one row per slot, a walking highlight, the plan —
and a new **Resolution** pane took the wide slot and the job of saying what actually happened,
in sentences, accumulating rather than flashing.

A combo therefore never has to be drawn *across* rows: it gets a line of its own, in amber, at
the moment it forms. So does a stagger. The bracket-or-join problem simply stopped existing,
which is worth recording as the pattern — **the pane was being asked to answer two questions at
once, and the fix was a second pane, not a cleverer drawing.**

The **COMBO splash** above is still wanted and still unbuilt; the pane makes a combo *legible*,
not *loud*. `dwellFor` freezing the screen for a splash-length `KindCombo` remains free.

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
- `[?]` Whether the prepare/attack/defend types are the same axis as the *role* taxonomy the
  initiate/respond model in `TODO.md` asks for, or orthogonal to it.
- `[?]` Whether armour exists, and whether it is a stat or is just what earth already does.
- `[?]` How enemies scale up the tower.
