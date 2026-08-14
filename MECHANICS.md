# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

**Running code:** cards and categories, phase resolution, combos, the 12-concept / 60-card
deck, and the elements and their statuses inside `internal/combat`. Those sections say so.
Everything else here is design that nothing implements yet.

**Next up: combos as data** — a `combos.json` catalogue reaching the rules through a bridge
package, so the set can be seen and grown in one file.

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
duelist. Base values live in `data/duelists.json` and `data/enemies.json`, and are expected to
move with playtesting.
What is still frozen is the *conversion* — `LifePerCon`, `baseActionPoints`, `speedPerPoint` —
and those are the bigger balance levers.

---

## Attributes and scaling

**Implemented today:** `Con`, `Str`, `Spd` on `Duelist`. Life is `Con × LifePerCon` (5). Action
points are `4 + Spd/10`, minimum 1. Damage comes from the *action* via `ActionKind.Damage(str)`,
not from an attribute of its own. Base values are per-combatant data in `data/duelists.json` and `data/enemies.json`.

**Damage reduction has exactly two sources, and no attribute is one of them.** The **earth
status** blunts what the opponent deals by a percentage, and `guardDivisor` halves a blow
against a raised guard. A durable combatant is one with high `Con` or earth on it. Anything
that should reduce damage extends one of those two rather than arriving as a third system —
two mechanics quietly stacking is the failure to avoid.

`[?]` **How enemies scale up the tower.** Enemies are fully-specified records with no level
term. A level multiplier — damage `level × 10`, speed `level × 5` — is the shape that has been
suggested. Nothing scales with floor today.

---

## Cards

### Types

**Implemented as `combat.Category`.** Every card is exactly one of three types, and
the type decides which phase of a turn it resolves in — see *Resolution* below. This is the
axis the whole round is built on.

**The concept set is a 3x4 grid: three types by four cost tiers, filled.** Twelve concepts,
and the grid is the reason there are twelve rather than however many somebody thought of.

| Type | 1 AP | 2 AP | 3 AP | 4 AP |
|---|---|---|---|---|
| **prepare** | Gather | Sift | Guard | Ritual |
| **attack** | Jab | Strike | Feint | Heavy |
| **defend** | Brace | Dodge | Riposte | Retreat |

| Type | Concept | AP | Effect |
|---|---|---|---|
| **prepare** | Gather | 1 | Banks +2 AP for the next round |
| | Sift | 2 | Two extra cards leave the hand at random at the round boundary, before it refills |
| | Guard | 3 | Halves every attack in the opponent's next turn |
| | Ritual | 4 | Banks +6 AP for the next round |
| **attack** | Jab | 1 | `str/2`, minimum 1 |
| | Strike | 2 | `str` |
| | Feint | 3 | `str`, and strips the target's **first pending defend card** without triggering it |
| | Heavy | 4 | `str × 2` |
| **defend** | Brace | 1 | Halves the next single incoming attack, then is spent |
| | Dodge | 2 | Negates the next incoming attack |
| | Riposte | 3 | Negates the next incoming attack and hits back for `str/2` |
| | Retreat | 4 | Negates the next **three** incoming attacks |

**Every card carries its effect in words on its face**, verb first, filling the card beside the
cost column. Six of the twelve could not be understood from a name, a cost and a damage figure.
The wording is `cardEffects` in `internal/screens`, beside the prose the Resolution feed uses:
the rules package names actions and never describes them. **Short words are a hard constraint** —
the column is about a dozen characters wide — and two tests hold the wording to it.

### Defends answer in the order they were raised

**The defend cards a duelist has up are a queue, not a pool.** The first one raised answers the
next incoming attack, whatever it is — Dodge and Retreat stop it dead, Riposte stops it and hits
back, Brace halves it and lets the rest through to any Guard. They all expire together at the
start of their owner's next turn.

This replaced a fixed precedence (Ripostes spent before Dodges, Braces only once nothing had
negated), which existed because four independent counters had no order to read. Three things
follow:

- **Order within the defend phase is a real decision**, and drag-to-reorder is how it is made.
  Leading with a Brace against a swarm spends the cheap card on the first blow and keeps the
  Dodge for the one that matters.
- **Feint has something well-defined to strip**: whichever card is at the front. Leading with a
  Brace baits a 3-point attack into stripping a 1-point card.
- **A Riposte queued last answers late**, so its counter-damage no longer lands as early in the
  opponent's turn as possible and can miss a kill it would once have made. That is the price of
  the queue, and it is paid deliberately.

**Retreat is the ordering rule's stress case.** Three negations on one card, and a Feint takes
one *charge* rather than the card — otherwise a 3 AP attack would beat a 4 AP defence outright.

Type is a property of the *concept*, not an independent axis — a fire Guard and a basic Guard
are both prepares.

**The grid is a design brief, not just a count.** An empty cell states a spec before anything
has a name — "a 3-cost attack" is something to solve. What it must not become is a cost ladder
where each tier is the one below it with a bigger number: that is twelve cards and three
decisions, and it is the trap `Dodge`/`Riposte` already half fell into. **Every tier differs in
kind.** Feint attacks the defence layer rather than the health total; Sift operates on the deck
and not on the arithmetic at all; Brace is partial where Dodge is binary.

**`[?]` Retreat is the weakest cell against that rule.** Three negations for four points is
Dodge with a bigger number, where every other tier changes what the card *is*. It was chosen
knowingly, on the grounds that the 4-cost defend it replaced — a card that negated a whole turn
and reflected every blow — was strong against everything and therefore mispriced rather than
interesting. Retreat is priceable; whether it earns the cell is open, and the answer is probably
a rider that makes volume mean something (retreating out of a status, giving ground) rather than
a fourth number.

**Ritual banks at a better rate than Gather** — 1 AP for +2 against 4 AP for +6, net +1 per
point against net +2. It sells rate *and* slots: four Gathers bank more (+8) but consume four of
five action slots, while one Ritual banks +6 and leaves four slots to fight with. It was +5 —
exactly Gather's rate — until 2026-08-14, on the theory that a 4-cost prepare should sell only
slots; at that price it was a card nobody had a reason to open with.

**Sift is the one concept whose effect is not a rule.** It manipulates the deck, and the deck
deliberately lives on the scene rather than in `internal/combat` — so `Sift` is an `ActionKind`
with a cost, a category and no engine effect, and the screen does the work at the round
boundary. That is a real division rather than an unimplemented card: the rules package owns
what a round does, not what you are holding. It also makes Sift the only concept `tools/balance`
cannot see.

**Sift and the Discard button are not the same thing.** Discard is *steering* — four a round,
you choose what leaves. Sift is *throughput* — more cards flow past you and you do not choose
which. It can eat a card you wanted, which is what makes it cost something beyond its 2 AP.

**Guard is a prepare, not a defend**, at 3 AP: it does not answer one blow, it dampens a whole
turn, and that is a thing you put up before the exchange rather than a reaction inside it.
Riposte is **defend** despite being a counter-attack.

**There is no Parry**, and it was dropped before it was built. Dodge, Riposte and Guard
already cover cheap-precise, expensive-punishing and broad-dampening; a fourth defence had no
job left. It can come back.

The three types split evenly 2/2/2 — 10 prepare / 10 attack / 10 defend — which is a
consequence of the concept list rather than a target.

**Ripostes are spent before Dodges** when both are up. Both negate completely, so spending
the one that hits back first costs nothing, and it lands the counter early enough to kill the
attacker mid-turn.

`[?]` **Dodge and Riposte are a tier, not a choice.** Riposte is exactly Dodge plus a Jab,
priced exactly as the two together (2 + 1 = 3), so it strictly dominates Dodge whenever it is
affordable — Riposte is only *better* than Dodge, never *different*. What stops it being
redundant is the cap on actions per round: one card doing two jobs beats two cards. Being
played with deliberately before changing anything.

**Feint narrows this without closing it**. Attacking into a Riposte normally
costs the attacker `str/2` in counter-damage; a Feint strips the Riposte and takes no counter,
so Riposte's punish clause is now something that can be defused and Dodge's cannot be — there
is nothing on a Dodge to defuse. That makes the two differ in what an opponent can do *about*
them, which is weaker than differing in what they do, so the `[?]` stands. Feint strips
Ripostes before Dodges, matching the order they are spent in.

### Concepts and deck composition

**A concept ships as five cards: basic, plus one per primary element.** That is the rule for
adding concepts, not just a description of the starting deck — a new concept arrives as a set
of five.

Twelve concepts × five = **60 cards**, implemented. A hand of eight against that is
13% of the deck, against 27% when the deck was 30 — **doubling the deck halved consistency**,
which is exactly why Sift exists and why `discardsPerRound` is now a number to watch rather
than a generous placeholder.

**The deck list is data.** `data/duelist_cards.json` holds the twelve concepts and the elements each
ships in, loaded beside `data/duelists.json`; `startingDeck` is built from it. Cost, category and
damage stay in `internal/combat` — the dependency direction forbids the rules package reading
`data`, and cost is about to stop being a property of the card anyway (see *Rings*). The JSON
carries the cost tier as **documentation with a check**: the loader asserts every declared tier
against `ActionKind.Cost()` and fails loudly rather than letting two sources of truth drift.

`Quick` was renamed **Jab** and given its five cards. It had been an `ActionKind`
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

**Poison has no cards and never did**. Two places said otherwise (the
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

*Implemented in `internal/combat/status.go`.* Elements are **mechanical**, as
always intended. Each applies a status **to whoever took the blow**:

| Element | Status | What it does |
|---|---|---|
| **ice** | chill | −1 AP per stack off the victim's next budget |
| **lightning** | shock | the victim's next attack misses outright, one per stack |
| **fire** | burn | 2 damage per stack at the end of each round it survives |
| **earth** | weight | the victim deals 10% less damage per stack, capped at 50% |

**Element crossed into `internal/combat` the same day**, which is what this section had been
waiting on and what unblocked ring discounts and the flip ring with it.
`combat.Element` is a rules type, `combat.Card` is a concept plus an element, and `[]Card`
replaced `[]ActionKind` through `ResolveRound`, `ResolutionOrder`, `Slot`, `PlanFor`, `CostOf`
and every planner. The screen's own `element` type and its `actionCard` struct are gone —
`actionCard` is an alias for `combat.Card`, so the hand, the queue and the round are one type
and a card is never converted between them.

**Cost is now a method on the card** (`Card.Cost()`), delegating to the concept. Nothing
discounts anything yet; that is the seat the ring discount sits in, cut while everything else
was moving.

#### The trigger: a landed attack, and nothing else

**Decided.** A prepare or a defend carries its element for combos and for the ring
discount and applies no status. The alternative — every card applying its status — makes a 1-AP
Gather as good a delivery as a 1-AP Jab and turns the prepare phase into the status engine.

The cost, stated: **element is mechanically inert on eight of the twelve concepts** until rings
land. And **the status lands because the blow connected, not because it hurt** — a Guard halves
a hit and the hit still landed, so it still chills. A Dodge, Riposte or Retreat stops the blow
dead and nothing is applied.

**Magnitude is per hit, not per card.** A fire Jab and a fire Heavy apply the same burn, so the
cheapest attack in the deck is the cheapest status delivery. The concept ladder prices damage;
the element ladder does not exist. Making status scale with the card is a second axis and a
design change.

#### One lifecycle, learned once

**Amount stacks, duration refreshes, everything clears at the end of the round after the one
that applied it.** `statusDuration` is 2 round-ends and it is one number for all four
deliberately. It cannot be 1: side B acts second, so a status B applied would expire before it
ever bit anything.

Per-element tuning is one constant each away. Run `tools/balance` before moving one.

#### Lightning is deterministic, and that is a change to this document

**"A chance to miss" is gone; a shock makes the next attack miss outright**. A
roll would need an injected `*rand.Rand` on `ResolveRound` — a sixth determinism stream,
advanced per attack, so any change to round one reshuffles every roll after it — and it would
end `internal/combat` being pure integer arithmetic, which is what makes it testable and what
makes `tools/balance` exact rather than a distribution. A certain miss also matches the rule
combos already follow: **what you committed to cannot be silently undone.**

What it gives up is the tension of a swing that might not land. Recorded rather than hidden.

#### The rest, and what each cost

- **Ice is the AP element in both directions.** The ice *ring* discounts your ice cards; the
  ice *status* cuts the enemy's budget. Same element, opposite targets, deliberate. It is read
  in `ActionPoints()` rather than subtracted when it lands, which is what makes it bite the
  round *after* the blow — the budget for the round in progress was committed before the attack
  resolved. The existing floor of 1 AP still holds however cold it gets.
- **Fire needed state that outlives an action**, and got it. `KindBurned` fires from `endRound`,
  side A then side B, and the screen's `applyEvent` reads it alongside `KindDamage` because a
  burn changes a life total with nobody acting. **A burn can kill**, and produces a
  `KindDefeated` when it does.
- **Earth applies attacker-side, before any defence.** Weight says how hard you can still swing,
  so the order is: concept damage, combo multiplier, the attacker's weight, then the defender's
  brace and guard. A Riposte's counter is outside it, exactly as it is outside the combo
  multiplier — that is the defender hitting back, not a card the weighted duelist played.
  **Rounding is toward zero**, matching `guardDivisor` and `scaleDamage`, so it is predictable
  from the reductions already in the game.
- **Statuses got a home**, and it is `Duelist.Statuses [ElementCount]Status` — an array indexed
  by element, not four named fields. That is what makes *"consume the status this element
  applies"* expressible, which Extinguishing Strike needs and which is the difference between a
  system and four ad-hoc fields. The price: **`Element` is append-only**, like `ActionKind` and
  `GlyphKind`. `Guarded` and the defend queue stay where they are — they are card effects, and
  filing them in a table indexed by colour would say they were not.

**Poison has lost its obvious job.** Fire is the damage-over-time element now. Poison, force
and hunger have no statuses.

#### What the balance tool says, and it is not comfortable

`tools/balance` carries four element postures — all-out in a colour, same concepts
and same 6 AP, so a coloured row read against `all-out` is what the element is worth. Enemies
beaten, out of 96:

| shocking | dodging | retreating | chilling | weighting | burning | all-out | feinting | guarding | bracing | ritual |
|---|---|---|---|---|---|---|---|---|---|---|
| 96 | 93 | 86 | 79 | 66 | 57 | 50 | 39 | 28 | 28 | 2 |

*(Measured 2026-08-14, after Retreat replaced the 4-cost reflecting defend and Ritual went to
+6. Two enemies — Clear Pod and Clear Slime — lose to everything; nothing is a wall.)*

- **`[?]` Shock is priced wrong and the number is one constant.** It now beats the entire
  roster, and it does so while dealing full damage, where a defensive posture gives up its turn
  to do less. Two lightning attacks a round is a free negation of most enemy turns on top of a
  full offence. `shockPerHit` is the lever; a shock that needed two hits to spend, or that only
  stopped the victim's *first* attack, are the two candidates.
- **`[?]` Every element beats plain.** A fire Strike costs what a Strike costs and does strictly
  more, so the 12 basic cards in a 60-card deck are strictly the worst 12. That is a
  consequence of cost being per concept, which is deliberate and is what the ring discount is
  designed around — but it means `basic` currently has no reason to be in a deck at all.
  Worth deciding whether basic is a *cheaper* card or simply the thing an affix transforms.

---

## Resolution — phases

**Implemented.** Chosen as an experiment and built the next day; it
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

There is no initiative. With one contiguous turn per side there is no exchange for a faster
action to lead, so initiative was a number on every card reporting a distinction the resolver
had stopped making. `Spd` still buys action points and still never buys priority.

Ordering *within* a category is queue order, which is what drag-to-reorder now moves and what
sequence combos will match on. See `TODO.md` for what would have to be true to bring
initiative back.

### What this cost, recorded honestly

- **It reverses an earlier decision**, which replaced volley-per-side with alternation
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

*Framework implemented in `internal/combat/combo.go`.*

Every combo is the same shape. **A run of cards that must appear consecutively in your own
turn**, and an `Effect` saying what forming it buys. A step matches an exact concept
(`Exactly(Strike)`), any concept of a category (`AnyOf(CategoryAttack)`) — which is what lets
"three attacks" mean three attacks rather than three Strikes — or **any card of one element**
(`OfElement(Ice)`). Any of the three can be pinned to a colour as well:
`Exactly(Strike).WithElement(Ice)` is an ice Strike and nothing else.

**The element axis arrived with elements themselves**, and `Exactly` is what `Card`
was called before the `Card` *type* took the name. `basic` is a colour a step may name — the
constraint is a separate flag rather than "element unset", because an all-plain run is a
legitimate pattern that `element: 0` could not distinguish from not asking.

**The flurry family stays colour-blind**, deliberately: three Strikes is a Strike Flurry however
they are painted. Requiring one colour would silently make every flurry in the game an element
combo, against a deck holding five Strikes of each.

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

**Combo IDs shifted and this was the last cheap moment for that.** `FlurryID` is
`comboFlurryBase + ComboID(a)`, derived from the card's raw enum value, and the five new concepts
had to be inserted *in category order* — the deck overlay sorts on that raw value to group the
piles the way a turn resolves. So Strike moved from 3 to 5 and its combos from 103/203 to
105/205. Harmless only because **no profile exists yet**, which is the very thing the derived-ID
scheme was designed to protect. Once discovery persists, inserting a concept mid-enum silently
re-locks combos the player has found, and the enum's category ordering and the ID stability
become a genuine conflict. `[?]` Resolve it before a profile ships — most likely by giving
`ActionKind` an explicit stable ID for combo purposes and letting the enum order serve the sort.

**Per card, not per category** *(decided after a category-wide version shipped
first)*. `AnyOf(CategoryAttack)` made any three attacks combo, so a Jab counted the same as a
Heavy and the reward went to whatever you happened to draw. Naming a combo for the card it is
built on is what makes it worth *building toward*: three Strikes is a deck you assembled. It
also leaves room for the effects to differ per card later — a Heavy Flurry has every reason to
hit harder than a Jab one, and a category-wide combo could never say so.

**IDs are derived from the card** (`FlurryID(a)`, `OnslaughtID(a)`), not from a position in a
list. Discovery persists on the profile, so an ID that shifted when a card was inserted would
silently re-lock combos the player had already found.

**"Unopposed" is gone from the name and from the rule**. It was written under
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

**Five cards of the same element doubles your action points next round** *(not
built)*. The first combo that counts an element rather than a card.

**It is an all-in round by construction.** The action cap is five, permanently, so five
same-element cards *is* your entire turn — there is no room for an off-colour card. And because
matching runs on the resolved order rather than the queue, a single off-element card can land in
the middle of the sequence after phases regroup it and break the run outright. Nothing else in
the game asks for a whole turn of one colour, which is what makes it worth a doubling.

Two things stood between this and existing. **The first is done** — elements
are in `internal/combat`, `OfElement(e)` is a `Step`, and five of them in a row is the whole
pattern. What is left is the second:

- **`internal/combat` can see elements.** `ResolveRound` takes `[]Card`, and
  `TestOfElementMatchesAnyConceptOfOneColour` already drives the exact five-in-a-row shape
  through the matcher against a table of its own.
- **"Double your AP" is a new reward kind, not a table entry.** `BankAP` is flat and additive;
  doubling is multiplicative. Adding it is a field on `Effect` plus one place applying it — the
  cost the framework charges on purpose. `[?]` **Whether to pay it.** At the current 6 AP a
  doubling *is* `BankAP: 6`, which is free and needs no framework change; the two only diverge
  once `Spd` or a ring moves the base — and that divergence is arguably the reason to want the
  real multiplier, since a reward that scales with investment rewards building toward it.

### Sequence-based

An ordered pair of elements, where **order changes the result**. Not yet built, and **nothing
stands in the way of building them**: `Exactly(Strike).WithElement(Ice)`
followed by the same with `Fire` is Burnt Icecube's whole pattern, and
`TestASequenceComboReadsTheOrderTheCardsWereQueuedIn` already pins that the reverse order does
not match it. What each *does* is still a reward kind the `Effect` vocabulary has not got.

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

**Open, and deliberately left open rather than settled early.** Combos are
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

**Done.** The cap is five, and it is `Duelist.MaxActions()` beside `ActionPoints()`
rather than the screen's old `maxSelected` constant. It moved for a concrete reason as well as
a tidy one: **the opponent's planner has to obey it exactly as the player's selection does**,
and a cap enforced only by the screen was a cap the enemy ignored.

**The cap is five permanently, and nothing may ever raise it**. This
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

**A ring that maps one element onto another across the whole deck**. A
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

**Discounting matching cards is why element had to cross into `internal/combat`, and it has**
. Cost stops being a property of the concept and becomes a property of the
pairing, and the seat is already cut: `Card.Cost()` is a method on the card, `CostOf` takes
`[]Card`, and the queue type, `ResolveRound`, `ResolutionOrder` and every planner already carry
the element. Nothing discounts anything yet — a discount needs an equipped ring, which needs
`Session` — but the rewrite this entry warned about was paid for with the elements work and is
not owed twice. What is left is the screen's AP bar and over-budget check reading a discounted
cost rather than a bare one.

**Rings are drawn as cards**, in a horizontal row across the top, not necessarily spanning the
whole bar. Same size as other cards, and **no glyphs**.

**The row is on the screen, at full card size, and it is a sketch** — nothing
buys, equips or reads a ring, and `data/rings.json` holds the three that have art. What made it
possible is that **the vertical problem below solved itself**: the full-height Resolution pane
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

Art note: `fire-ring.png`, `frozen-ring.png` and `thunder-ring.png` are embedded and now drawn.
Earth has none, so a fourth ring is art before it is data.

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

- **Every third fight is a floor boss.** Bosses are durable — high `Con`, and earth on them if
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

- **Baseline decks are data**, alongside the `data/enemies.json` records — `data/enemy_cards.json` holds them.
- **Affixes are renamed to match the elements**: `hot`/`cold`/`charged` become `fire`/`ice`/
  `lightning`. Two vocabularies for the same things was a collision waiting to happen.
- **`undying` is parked**, not deleted. Revisit later.
- `[?]` Earth has no affix — either it joins the list or there is a reason it cannot be a floor
  theme.

**`PlanGreedy` stops working as written.** It plans from a `Duelist` and a fixed set of four
actions; with a deck it has to draw a hand and plan from that, which subjects the enemy to the
same "what did I draw" pressure the player faces.

### Styles — implemented, and a step short of decks

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

**Two panes, split rather than solved.** **Action Flow** is one row per slot, a walking
highlight, the plan; **Resolution** takes the wide slot and the job of saying what actually
happened, in sentences, accumulating rather than flashing.

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
- `[?]` How enemies scale up the tower.
