# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both.

Everything here is decided unless marked `[?]`. Read this before proposing a design change,
and before implementing anything that touches a rule.

**Running code:** cards and categories, phase resolution, combos, the 12-concept / 60-card
deck, and the elements and their statuses inside `internal/combat`. Those sections say so.
Everything else here is design that nothing implements yet.

**Combos are data** as of 2026-08-14 — `data/combos.json`, read by `internal/combat`, which is
the one edge from the rules into that package.

**A turn resolves one attack, and combos are how it is scored** *(2026-08-14)*. This is the
largest rules change the game has taken and it reaches into almost every section below: attack
cards are read as a *set* rather than played one at a time, the hand and the element makeup they
form are two multipliers that **add**, defends became percentage reductions instead of
negations, and lightning went back to being a roll. Where an older rule survives it is because
it was re-decided, not because it was left alone.

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

**Damage reduction is percentages all the way down, and no attribute is one of them.** Three
things cut a blow and they are all the same shape: the **earth status** blunts what the attacker
deals, the four **defend cards** each take a percentage off what arrives, and `guardDivisor`
halves whatever is left. A durable combatant is one with high `Con` or earth on it. Anything
that should reduce damage extends one of those three rather than arriving as a fourth system —
two mechanics quietly stacking is the failure to avoid.

**They compose multiplicatively and in a fixed order**: concept damage, the combo multiplier,
the attacker's weight, then every raised defend, then the guard. Weight sits on the attacker's
side of that line because it says how hard they can still swing; everything after it happens to
a blow that has already been blunted.

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
| | Feint | 3 | `str`, and strips the target's **strongest raised defend** without triggering it |
| | Heavy | 4 | `str × 2` |
| **defend** | Brace | 1 | Takes **50%** off the incoming blow |
| | Dodge | 2 | Takes **75%** off |
| | Riposte | 3 | Takes **75%** off and hits back for `str/2` |
| | Retreat | 4 | Takes **100%** off — stops the blow dead |

**An attack card's damage figure is what it contributes to a hand, not a blow it lands on its
own.** A turn resolves one attack; see *Combos* below for how the figure is assembled.

**Every card carries its effect in words on its face**, verb first, filling the card beside the
cost column. Six of the twelve could not be understood from a name, a cost and a damage figure.
The wording is `cardEffects` in `internal/screens`, beside the prose the Resolution feed uses:
the rules package names actions and never describes them. **Short words are a hard constraint** —
the column is about a dozen characters wide — and two tests hold the wording to it.

### Every defend answers the blow, and they multiply

**The defend cards a duelist has up are a set, not a queue.** The opponent's turn produces one
attack, so *every* raised card meets it and each takes its percentage off what is left: a Dodge
and a Brace together leave `25% × 50%` = an eighth of the blow. **Order is not read**, and every
card is spent on the one attack it answered. They all expire together at the start of their
owner's next turn.

This replaced a raise-order queue that had stood for a day, which itself replaced a fixed
precedence. What broke both of them is one attack per turn: "which of my defences meets which
blow" is a question with no content when there is only ever one, and negation is intolerable
when the thing being negated is a whole hand — a 2 AP Dodge cancelling a 10 AP Onslaught.
Three things follow, and two of them are losses:

- **Percentages keep the four cards distinct at their four prices.** Under negation, "halve the
  first of five" and "stop three of five" both collapse to "the blow", so Brace and Retreat lost
  their reason to exist. As reductions they stay priced by tier and stack sensibly.
- **Retreat still reaches 100%.** The most expensive defend in the game should be able to stop a
  turn dead, and at 4 of a 6-point budget it is most of a round spent doing nothing else. It is
  the only card that reaches zero, which is what it is for.
- **Within-phase ordering now means nothing anywhere in the game**, and drag-to-reorder has lost
  its last mechanical job. See *Combos* — counted hands do not read order either. Dragging is
  currently arrangement, not decision, and `TODO.md` carries what would give it one back.
- **Feint strips the strongest raised card**, not the first, since there is no first. That is
  also what keeps it worth 3 AP: stripping whichever card happened to be cheapest would make it
  a 3-point attack that removes a 1-point Brace. It fires once however many Feints are in the
  hand, and it does not touch Guard, which is a prepare rather than a defend card.

Type is a property of the *concept*, not an independent axis — a fire Guard and a basic Guard
are both prepares.

**The grid is a design brief, not just a count.** An empty cell states a spec before anything
has a name — "a 3-cost attack" is something to solve. What it must not become is a cost ladder
where each tier is the one below it with a bigger number: that is twelve cards and three
decisions, and it is the trap `Dodge`/`Riposte` already half fell into. **Every tier differs in
kind.** Feint attacks the defence layer rather than the health total; Sift operates on the deck
and not on the arithmetic at all; Riposte answers back where Dodge only absorbs.

**`[?]` The defend column now fails that rule twice, and reductions are why.** Brace 50, Dodge
75, Riposte 75, Retreat 100 is a cost ladder with a bigger number at each rung, which is exactly
what the grid is supposed to prevent. Two of the four still have a second clause — Riposte hits
back, Retreat reaches zero — but **Brace is now purely a smaller Dodge**, where it used to be
partial against something binary. That distinction was real and the repricing spent it.

This is the price of the change and it was taken knowingly: negation could not survive one
attack per turn, and a column of four coherent percentages is better than a column of four cards
that all read as "the blow does not happen". What would earn the cells back is riders rather
than percentages — clearing a status as you give ground, a Brace that pays something for taking
the hit — and that is the shape to reach for before touching the numbers.

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

**There is no spend order any more.** Every raised defend answers the one blow — see *Every
defend answers the blow* above — so the old "Ripostes before Dodges" precedence describes
nothing. The counter fires **after** the blow has landed rather than instead of it, which is
what a Riposte reducing rather than negating implies: the attacker connects for a quarter and
takes `str/2` back.

`[?]` **Dodge and Riposte are a tier, not a choice.** Both now take 75%, and Riposte adds a
counter for one more point — so it is exactly Dodge plus a Jab, priced exactly as the two
together (2 + 1 = 3), and strictly dominates Dodge whenever it is affordable. The repricing made
this *worse*, not better: the two used to differ only in the counter, and now they differ only
in the counter while also carrying the same number. What stops it being redundant is the cap on
actions per round — one card doing two jobs beats two cards. Being played with deliberately
before changing anything; the obvious fix is to move one of the two percentages.

**Feint narrows this without closing it**. Attacking into a Riposte costs the attacker `str/2`
in counter-damage; a Feint strips the Riposte and takes no counter, so Riposte's punish clause
can be defused and Dodge's cannot — there is nothing on a Dodge to defuse. That makes the two
differ in what an opponent can do *about* them, which is weaker than differing in what they do,
so the `[?]` stands. A Feint takes the **strongest** card up — and since Dodge and Riposte are
now the same percentage, which of the two it takes falls to the tie-break (the earlier entry)
rather than to anything the player decided. One more reason to move one of those numbers.

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

#### The trigger: the colours in the hand that formed

**Decided, and rewritten by one blow per turn.** Statuses are applied by the **mix** — one status
per distinct non-basic colour among the cards that formed the attack. A mono hand lands one, a
rainbow lands four, and a drab hand lands none. A prepare or a defend carries its element for the
mix and for the ring discount and applies nothing itself; the alternative — every card applying
its status — makes a 1-AP Gather as good a delivery as a 1-AP Jab and turns the prepare phase
into the status engine.

Three consequences, all of them changes from the per-card version:

- **A colour is counted once however many cards carry it.** Two ice Strikes and an ice Jab are
  mono ice and land one chill, where three separate ice hits used to land three. Status volume
  moved from "how many coloured cards" to "how many *different* coloured cards", which is the
  same axis the mix multiplier already pays for.
- **Cards outside the hand carry no colour at all.** `Strike, Jab, Strike` in fire, ice, fire is
  a fire Strike Pair — mono, one burn — and the ice Jab contributes neither damage nor a chill.
- **The status lands because the hand formed, not because the blow hurt.** A hand reduced to
  zero by a Retreat still connected, and making the status conditional on the final figure would
  let a defensive card silently un-apply an element the attacker had already paid for.

The cost, stated: **element is mechanically inert on eight of the twelve concepts** until rings
land.

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

#### Lightning is a roll, and it is the only one in the rules

**A shock is a chance the turn's attack misses: 25% per stack, capped at 75%.** One stack is
spent per attack phase whether or not the roll lands — a shock that only burned itself on a
success would be a guarantee wearing a probability's clothes.

**This reverses the deterministic version taken two days earlier**, and one blow per turn is
what forced it. A certain miss used to delete one attack out of several; now it deletes the
whole turn, so a 1 AP lightning Jab could erase a 10 AP Onslaught outright. The alternatives
considered were breaking the hand or cutting the multiplier; a roll was chosen because lightning
should feel unreliable, which is a design reason rather than a balance one.

**The cap is what stops the roll becoming the old rule by another route.** Without it four
lightning hits would be a certain miss again, and a defence that always works is exactly what
one blow per turn makes intolerable.

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
- It breaks the rule combos otherwise follow — *what you committed to cannot be silently undone*.
  Lightning is the deliberate exception, and it is the only one.

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
  so the order is: the hand's own cards, the combo multiplier, the attacker's weight, then every
  raised defend, then the guard. A Riposte's counter is outside it, exactly as it is outside the
  combo multiplier — that is the defender hitting back, not a card the weighted duelist played.
  **Rounding is toward zero**, matching `guardDivisor`, the defend reductions and `scaleDamage`,
  so it is predictable from the reductions already in the game.
- **Statuses got a home**, and it is `Duelist.Statuses [ElementCount]Status` — an array indexed
  by element, not four named fields. That is what makes *"consume the status this element
  applies"* expressible and is the difference between a system and four ad-hoc fields. The
  price: **`Element` is append-only**, like `ActionKind` and `GlyphKind`. `Guarded` and the
  defend set stay where they are — they are card effects, and filing them in a table indexed by
  colour would say they were not.

#### The balance numbers are gone and have not been retaken

`tools/balance` carries four element postures — all-out in a colour, same concepts and same
6 AP, so a coloured row read against `all-out` is what the element is worth. **Every figure it
has ever produced was measured against the multi-blow model and none of them survive the
rewrite.** Damage now runs through a hand multiplier, defends reduce instead of negating, and
lightning rolls, so the old table would be a picture that lies — the same reason a stale glyph
sheet is worse than none. It is deleted rather than annotated.

What has to be re-measured before anything is tuned, and roughly what to expect:

- **Damage is much larger and enemy HP has not moved.** Two Strikes at Str 10 is `20 + 10×1.5`
  = 35 where it used to be 20; a rainbow Onslaught is `10×20 = 200` on top of its cards. Retuning
  enemy life totals is the owner's call and is expected, not a bug report.
- **Shock's `[?]` is reopened, not answered.** It used to beat the entire roster by cancelling
  attacks for free; a 25%-per-stack roll capped at 75% is a different card and its price is
  unknown. `shockMissPct` and `shockMissCapPct` are the levers now, not `shockPerHit`.
- **The tool is a distribution now.** One fixed-seed sample per matchup is reproducible but is
  one draw; a posture that wins 51% of the time and one that wins 100% currently look identical.
- **`[?]` Every element beats plain, and the mix multiplier widened the gap.** A fire Strike
  costs what a Strike costs and does strictly more, and now a *mixed* pair of colours pays 200
  on top where a plain one pays nothing. The 12 basic cards in a 60-card deck were the worst 12
  before this change and are further behind after it. That is a consequence of cost being per
  concept, which is deliberate and is what the ring discount is designed around. Worth deciding
  whether basic is a *cheaper* card or simply the thing an affix transforms.

---

## Resolution — phases

**Implemented.** Chosen as an experiment and built the next day; it
is what ships. Still open to reconsideration, but it is the model now, not a proposal.

A round is **a whole turn each**. Everything one side queued resolves before the other side
does anything, and within a turn the categories go in order:

1. that side's **prepares**, one card at a time
2. that side's **attack** — *one* blow, whatever it was assembled from
3. that side's **defenses**, all raised together
4. then the other side, the same way

Defenses come last *within a turn* because the opponent moves next, so a defense raised at the
end of your turn is up when their blow arrives.

**The attack phase is a single event, and that is the 2026-08-14 change.** Every attack card
queued is announced, the hand they form is announced, and then one figure of damage lands. The
prepares are now the only cards that still resolve one at a time, because each does something to
its own duelist rather than contributing to a shared blow.

**And the phase says one thing, not one thing per card.** The announcements still happen — each is
a beat, and the screen raises the card that made it — but the *sentence* is the hand's: "COMBO!
Duelist lands a Duo Strike Pair (20 + 10 x 3.5 = 55), 55 damage". Five lines saying a Strike was
swung describe five blows, which is exactly the reading this rule was written to end. **A blow that
forms no hand still gets its own ordinary sentence**, because a lone Strike is not a combo and
announcing one over every attack would empty the word.

**Prepares run before the hand is worked out.** Nothing a prepare does reaches the attack today —
Gather and Ritual bank for next round, Guard answers the *opponent's* blow, Sift is the screen's
business — so this is a legibility choice rather than a mechanical one. It is the order the
phases are named in, and a prepare that did feed the hand would need no rule change to do so.

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

### The catalogue is data

*`data/combos.json`, with `data/combos_data.go` holding its shape and
`internal/combat/combo_table.go` turning it into rules. The vocabulary and the matcher are in
`combo.go`.*

**`data` holds the shape, `internal/combat` holds the meaning**, which is the division
`CheckCostTiers` already draws for the deck lists. The file can say `"scope": ["attack"]` and
`"expand": "attack-cards"`; only the rules can say what an attack *is*, so that is where the
names are resolved (`ParseCategory`, `ParseAction`) and where a malformed catalogue is refused.

**This is the one thing in `data/` that the rules themselves read**, and it is why
`internal/combat` imports that package at all. Everything else there is consumed by `screens`,
`decks` or `entities` — layers *above* the rules — so the rules never needed it. That is the
line to hold if a sixth list is proposed: **ask who reads it, not whether it is data.** `data`
imports nothing but the standard library, so the edge costs `internal/combat` neither its
testability nor its freedom from Ebitengine.

**A malformed catalogue panics at package init**, exactly as a deck whose declared cost tiers
disagree with the rules does. A combo silently dropped is a balance change nobody made.

### The pattern: two axes, multiplied at load

Every combo is the intersection of a **hand** and a **mix**, and both are read off the cards that
formed one attack.

- The **hand** counts copies of a concept — pair, two pair, flurry, full house, barrage,
  onslaught. That is exactly what a poker hand counts, so it wears poker's names honestly.
- The **mix** counts **distinct non-basic colours** in whatever hand formed — drab, mono, duo,
  trio, rainbow. Basic is not a colour and never counts in either direction, so two basic Strikes
  and an ice Strike show one colour and are mono.

Six hands by five mixes is **27 combos, authored as 11 numbers**. Writing the grid out would be
27 sets of values nobody could keep consistent; writing the two axes is six lines and five.

**Exactly one hand and exactly one mix apply**, which is what retired the family/tier machinery
the catalogue used to carry. A hand wins on its multiplier — five Strikes are an Onslaught rather
than also the pair and the flurry inside it — and the mixes name an *exact* colour count and
partition every possible hand, so no two can both be true. Nothing needs ranking against anything.

**A lone attack forms no hand and still has a mix**, because one card is still one colour. That
is the fallback: when nothing counts, the single hardest-hitting attack card is the blow, ties
going to the card queued first.

**Attack cards outside the hand contribute nothing.** `Strike, Jab, Strike` is a Strike Pair; the
Jab is announced, is not in the hand, adds no damage and carries no colour. That is a stated rule
rather than a consequence — it is what makes *choosing a shape* pay more than throwing everything
you drew.

### Damage: one blow, and the multipliers add

A turn deals damage **once**, in the attack phase, and the figure is:

```
(damage of each card in the hand)  +  DMG × (hand multiplier + mix multiplier)
```

`DMG` is what one Strike deals at this duelist's strength — the number on the fighter card. So a
pair of Strikes at Str 10 is `10 + 10 + 10×1.5` = **35**, and a duo pair of Strikes is
`10 + 10 + 10×(1.5+2.0)` = **55**.

**The multipliers add rather than compose**, decided deliberately: 1.5× and 2× is 3.5×, not 3×.
Multiplying them stacks a ladder on a ladder and puts a rainbow Onslaught at 20× before its own
cards are counted; adding keeps the top of the table at a few hundred damage instead of several
thousand, and it is arithmetic a player can do in their head while planning.

| Hand | Cards | Multiplier | Besides damage |
|---|---|---|---|
| **`<Card>` Pair** | `[2]` | ×1.5 | — |
| **Two Pair** | `[2,2]` | ×1.75 | — |
| **`<Card>` Flurry** | `[3]` | ×2 | opponent loses one action |
| **Full House** | `[3,2]` | ×3 | opponent loses one action |
| **`<Card>` Barrage** | `[4]` | ×5 | opponent loses two actions |
| **`<Card>` Onslaught** | `[5]` | ×10 | opponent loses their whole turn |

| Mix | Colours | Multiplier | Besides damage |
|---|---|---|---|
| **Drab** | 0 | — | — |
| **Mono** | 1 | — | that colour's status |
| **Duo** | 2 | ×2 | both statuses |
| **Trio** | 3 | ×5 | all three statuses |
| **Rainbow** | 4 | ×10 | all four statuses |

**Mono pays in status alone, and that is the point of the bottom rung.** One colour is what an
ordinary hand looks like; the element is already worth having for what it does to the victim, so
paying a multiplier on top would make plain cards worse than they already are for no design gain.

**Every hand is attack-scoped.** Prepares and defends carry colours for the ring discount and
nothing else — they are not counted, and a Gather cannot join a rainbow. That is what makes the
top of the table an offence you spent a whole turn assembling rather than something a cheap card
can be padded into.

**Stagger is paid on forming the hand, not on connecting.** A shock that makes the blow miss does
not undo a stagger the player assembled five cards to earn: the hand is scored off the queue, and
the queue was committed when DUEL! was pressed. The same applies to banked AP.

**The reward vocabulary stays deliberately small and closed** — a damage multiplier, banked
action points, and stagger. Adding a combo is one entry in `combos.json`. Adding a new *kind* of
reward is a field on `Effect` plus one place applying it, and that cost is charged on purpose,
because a reward vocabulary that grows without limit is one no player can hold in their head.

### What the two axes cost, recorded honestly

- **Counted matching only.** A hand reads the turn as a set, so a Jab between two Strikes does
  not break the pair. The **run** match kind — N consecutive cards, which a sequence combo needs
  — was in the schema and is **gone**, dropped in the rewrite. Nothing had ever used it. The
  consequence is below, under *Sequences*.
- **A hand cut short still pays out.** Nothing can interrupt it — a turn's attacks resolve as one
  event — so this is now true by construction rather than by rule.
- **The bottom rung fires constantly.** Any two copies of one attack is a pair, which is most
  turns, and the AI planners repeat a card far more readily than a human building a mixed hand
  does. That was measured as a live problem on the old model and the rewrite has not addressed
  it; it only changed the number. `[?]` **Watch whether Pair should pay a multiplier at all**, or
  whether the ladder should start at Two Pair.
- **`[?]` `Copies: 1` makes most of the grid unreachable.** The starting deck ships one card per
  concept per colour, so every same-concept hand is forced to all-distinct elements — a pair is
  always duo, an Onslaught is always rainbow, and **drab and mono are unreachable at every rung
  above one card**. 13 of the 27 cells cannot currently be dealt. This is expected to resolve
  when the deck changes over a run, and is recorded so the grid is not tuned against a deck that
  cannot show most of it.
- **Poker's ranking does not transfer to this deck**, and the multipliers above are first
  drafts. Poker's ordering comes from 52 cards, 4 suits and 13 ranks; here a rank has 5 copies
  and a suit has 12. `tools/seeds` already searches real hands and is the place to measure
  before tuning further.

### The catalogue's shape

`data/combos.json` holds two lists and nothing else.

**Hands** carry a key, an ID, a name, `groups`, a `scope`, a `multiplier` in percent, an
`expand` mode and an `effect`. `groups` naming *distinct* concepts is why `[3,2]` is a full house
and can never be satisfied by five of one card. `staggerAll` is a bool in the file rather than the
`-1` the rules use, because a file saying `"stagger": -1` is one nobody can read.

**Mixes** carry a key, an ID, a name, an exact `colours` count and a `multiplier`. The loader
**refuses a catalogue with a gap in the colour counts** — every count from zero to four needs a
mix, or a turn could form a hand the engine cannot name.

**One entry per card, generated rather than written out.** The four single-group rungs carry
`"expand": "attack-cards"` and become one hand per attack card — **Jab, Strike, Feint and Heavy**
— so 16 hands from four lines, and a new attack card gets the whole ladder by existing. **Two
Pair and Full House are generic**, because naming them per card-pair is 6 and 12 near-identical
entries nobody can hold in their head. 22 hands in total.

**Per card, not per category, for the rungs that expand** *(decided after a category-wide version
shipped first)*. A category-wide "three attacks" made a Jab count the same as a Heavy and sent the
reward to whatever you happened to draw. Naming a combo for the card it is built on is what makes
it worth *building toward*, and it leaves room for the effects to differ per card later.

**Straights are dropped rather than invented** — the concepts have no natural order to be
consecutive in.

**Combo IDs are written in the file, and the hazard they carry is unchanged.** An expanded
entry's ID is the base in `combos.json` plus the card's enum value, so **appending** an attack
card is free and **inserting** one mid-enum still shifts every ID above it. Harmless only because
**no profile exists yet**. `[?]` Resolve it before a profile ships — most likely by giving
`ActionKind` an explicit stable ID for combo purposes.

### Stagger

**Stagger takes actions off the front of the victim's next turn**, which under phases is their
prepare phase — so being staggered costs a Gather before it costs an attack, and leaves you
poorer next round as well as slower now. The action points are **not** refunded: stagger is tempo
*and* economy.

**It is symmetric, and one asymmetry falls out of phases.** Side A takes its whole turn first, so
a combo A forms bites B in the same round; the identical combo formed by B lands when A has
already acted and carries to the round after. That is why `Duelist.Staggered` persists across the
boundary — it makes the rule one rule (*a staggered duelist loses actions from its next turn,
whenever that is*) rather than two spelled differently per side.

**A stagger deletes cards before the hand is matched**, so a staggered duelist cannot combo back
with a turn it never took. That ordering is why the hand is worked out *inside* a turn rather than
at the top of the round.

What keeps the top of the ladder rare is the deck and the budget: three Strikes is exactly 6 AP,
a starting fighter's entire budget, and **five Strikes is 10 AP**, reachable only by spending a
whole round on Gathers. That trade is the combo working as intended.

### Sequences — the capability the rewrite dropped

**There are no ordered combos, and there is no longer a way to write one.** The schema's `run`
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
a reward kind that *consumes* a status, which the `Effect` vocabulary has never had.

### `[?]` Whether the enemy draws on this at all

**Open, and deliberately left open rather than settled early.** Combos are symmetric today, and
the rewrite made that symmetry cost more: a swarm queueing four Jabs no longer lands four small
hits, it forms a **Jab Barrage** — five times damage and two staggers — for 4 AP.

**The four planners were not rewritten for one blow per turn and the change inverts them.** A
swarm builds a hand by accident; a brute's single Heavy forms nothing at all and pays no
multiplier. The shape the roster treats as the crude one is now the strong one, which nobody
designed. `TODO.md` carries it.

It is left standing on purpose, to watch how it balances out. **It is not settled that enemies
share the player's combo table, or even the player's cards.** An enemy built from a deck and an
affix could plausibly have attacks the player never sees and combos of its own, in which case
the symmetry here is a temporary convenience rather than a rule.

The pricing note worth keeping either way: every costing in this document reasons from the
player's budget — three Strikes is 6 AP, five is 10 — and a swarm gets four Jabs for 4 AP
because `Jab` costs 1. **A combo counting cards is priced by whoever has the cheapest cards**,
and a multiplier on top of that counting makes the gap wider than it was.

### Requirements

- **Combos are rules and live in `internal/combat`**, matching on the resolved cards. The
  screen must never derive one; that is what makes the Resolution pane structurally incapable
  of lying about the round.
- **A `KindCombo` event** carrying what fired. *Done.* It carries a `HandID`, a `MixID`, the
  combined multiplier and the list of cards that formed the hand, and the screen looks the two
  names up with `HandByID`/`MixByID` — so a combo renamed is renamed once.
- **The combo event carries its own card list, not a span.** A counted hand is not contiguous —
  Two Pair can be two cards, a card that earned nothing, and two more — so the screen brackets
  what the engine names and never derives it from a pattern length.
- **`KindStaggered` counts as a slot in playback** even though nothing happened, or the
  Resolution pane's highlight runs a row short for the rest of the round.
- **A place to browse combos** — a reference the player can return to. Probably belongs with the
  profile rather than inside a duel. `Hands()` and `Mixes()` exist for it to read.
- **The attack phase writes one line, and it is the combo's.** *Done.* Attack cards no longer draw
  a row each — a turn of five Strikes read as five actions and one figure, which is the model the
  pane was contradicting. The line carries the arithmetic (`20 + 10 x 3.5 = 55`) off the event, so
  the sum shown is the sum used, and the damage attaches to it. **What is still not drawn** is a
  row that a stagger deleted; that one stands.
- **A preview of the hand while planning is wanted and does not exist.** `AttackFor` is exported
  and is the same function the engine uses, so a previewed combo would be the combo that fires by
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
  built — Onslaught wants five of one attack, Rainbow wants four colours in one hand — and every
  one of them needs the five to be a constant the player can plan against for the life of a run.
  The catalogue loader enforces it directly: **a hand asking for more cards than a turn can hold
  is refused at package init.**
- **A growable cap would dilute every shape as it grew.** An Onslaught is an all-in commitment at
  a cap of five and routine at a cap of seven. The combos would quietly get cheaper every time
  capacity went up, which is the opposite of a reward for building toward them.
- **It is still a method, and still should be.** Rings and brands need somewhere to bite for
  everything *else* they do, and a method that reads the duelist costs nothing. What changed is
  that this particular lever is off the table: **no ring, brand or combo raises `MaxActions`.**

The consequence for prepare cards: a 4-AP prepare cannot buy action slots, so Ritual buys
points instead (+6). An earlier draft of Ritual granted +2 AP and +2 slots specifically to reach
six- and seven-card combo hands; that is exactly the dilution above and it was cut.

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
That is not a fault in Dodge's cost: a defence was priced against how many attacks arrived, so
an opponent's *shape* is what the player is really buying answers to, and one shape means one
answer.

**The styles now differ in what they build, not in how many blows they land.** With one attack
per turn, "as many attacks as the round allows" is a swarm assembling a *hand* — four Jabs is a
Jab Barrage, not four hits — and a brute's single Heavy forms nothing at all. That inverts the
old reading: the swarm is now the dangerous shape and the brute is the one giving up a
multiplier. **Nothing in the planners was rewritten for this**, so what each style is worth is
an open measurement rather than a designed outcome.

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

- `[?]` **Nothing reads within-phase order any more**, so drag-to-reorder has no mechanical
  effect. Either it stops being presented as a decision, or something reads order again.
- `[?]` **The defend column is a cost ladder with a bigger number at each rung**, which the
  concept grid exists to prevent. Brace is a smaller Dodge; Dodge and Riposte carry the same
  percentage.
- `[?]` **Pair fires on most turns**, which makes the bottom rung a permanent global multiplier
  favouring whoever repeats themselves — currently the AI planners.
- `[?]` **13 of the 27 hand/mix cells cannot be dealt** from the starting deck, because one copy
  per concept per colour forces every same-concept hand to distinct elements.
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
  every attack card and then lowering the ones the combo did not name. Whether that is legible at
  playback speed is unanswered.
- `[?]` Earth's green collides with `playerSwatch`. One of the two schemes has to give, and
  what is holding it off is that a border and a swatch are never seen side by side.
- `[?]` Whether the prepare/attack/defend types are the same axis as the *role* taxonomy the
  initiate/respond model in `TODO.md` asks for, or orthogonal to it.
- `[?]` How enemies scale up the tower.
