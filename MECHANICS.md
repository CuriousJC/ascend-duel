# MECHANICS.md

**What the game is.** `TODO.md` is what to build; this is what it should be when built.
`ideas.md` is the unfiltered inbox that feeds both. Read this before proposing a design change,
and before implementing anything that touches a rule.

**Everything here is decided unless marked `[?]`.** A `[?]` is an open question, wherever it
appears. The **Status** column below says whether a section describes running code or a design
nothing has built yet; inside a section, a rule is running code unless it says otherwise.

## A run, end to end

The title screen starts a run or resumes the one on disk. A run climbs a tower of **eight floors
of three rooms**, and every third room is a named stairway boss.

**A room is a duel.** Both duelists are cards facing each other across a table. Each round you
are dealt a hand of eight from your deck and given an action-point budget; you queue up to five
cards you can pay for, in an order you choose, and press DUEL!. Your attack cards are read as a
**set** — what they agree on (concept, form or element) names a **hand** off a ladder wearing
poker's names, and that hand's multiplier scales **every hit**. **Each card lands its own hit**,
worked out on its own: the card's damage, anything the relics add, times the hand — a defense's
hit deals nothing of its own but still lands its color.
Any defend cards in the turn go up first and raise **shields**; one shield eats one incoming
hit whole. Then the creature swings, card by card, each hit its own. A duel is over when
someone dies or when the **fifth round** ends, which kills whoever is still standing.

**Winning pays vitae and an essence.** An essence permanently edits one card of your deck. Vitae is
spent in a shop on a **relic** — worn, five at a time, and each one bends a rule — or on a **stone**
that raises one rung of the hand ladder, a **potion** that changes the duelist, a **rune** that
alters cards mid-fight, or a **sealed good**, which is paid for and *then* opened. Then the next
room.

Wounds carry from room to room and only a stairway heals them, so a floor is an attrition budget.
The run is written to disk between rooms and every run has a six-character code that replays it.

## Contents

| Section | What it settles | Status |
|---|---|---|
| [The thrust](#the-thrust) | the principle every rule is measured against | principle |
| [Attributes and scaling](#attributes-and-scaling) | DMG, Actions, HP, and what may cut a hit | built |
| [Cards](#cards) | forms, the cost ladder, shields, the deck, the card language | built |
| [Elements](#elements) | the five colors, their statuses, where a status comes from, and a creature's own element | built |
| [Resolution — phases](#resolution--phases) | what order a round resolves in | built |
| [Hands](#hands) | the ladder, its axes, and how every hit is multiplied | built |
| [A round is bounded twice](#a-round-is-bounded-twice) | the action budget and the five-card cap | built |
| [The round limit](#the-round-limit--a-duel-is-bounded-too) | a duel is five rounds long | built |
| [Relics](#relics) | the grammar, the catalog, the shop | built |
| [Stones](#stones--altering-the-hand-ladder) | raising one hand shape, on every axis, for one run | built |
| [Essences](#essences--altering-the-deck-between-fights) | editing the deck between fights | built |
| [Runes](#runes--altering-the-deck-during-a-fight) | editing the deck inside a fight, and the riders a card can carry | built |
| [Brands](#brands) | permanent changes to the chassis | **designed, not built** |
| [Vitae](#vitae) | the currency and what a win pays | built |
| [The tower](#the-tower) | eight floors, three rooms, the ascent curve | built except the room choice |
| [Enemies](#enemies) | the roster, the planner, what a creature may do | built |
| [The profile](#the-profile--what-survives-a-run) | what outlives a run, and the menus around it | built |
| [Achievements](#achievements) | the three triggers and the closed vocabulary | built |
| [Randomness](#randomness) | what is rolled and what may never be | built |
| [The ledger](#the-ledger--the-runs-account-of-itself) | the run's account of itself | built |
| [Overlays](#overlays) | the panels that cover the table | built |
| [Open questions](#open-questions) | every `[?]` in one place | — |

---

## The thrust

**The primary thrust of the game is building a deck and an engine that bend the rules in a
way that lets the player win.**

Relics and brands are rule-modifiers first and stat-boosters second — more actions, cheaper
cards, free cards, and stats too, since a stat is just another rule to bend. Every constant
below is a candidate for something to bend.

The consequence for the code: **rules cannot stay `const`**. A card's cost, damage and shield
count are fields on its record, so retuning one is a file edit. What is still a compile-time
constant is the actions-per-round cap and the status magnitudes, read by functions with no access
to the run. Bending either needs a **carrier** — a modifier set passed alongside the duelists that
`internal/combat` reads instead of the constants, making cost a function of the card *and* that
carrier, the way `Damage` is already a function of the action and the wielder.

Attributes do **not** need this. `DMG`, `Actions` and `HP` are already fields on `Duelist`, and
`ResolveRound` takes duelists by value, so a relic granting `+5 DMG` just hands it a different
duelist. Base values live in `data/duelists.json` and the motif files under `data/motifs/`, and are expected to
move with playtesting. A creature's are in its motif file under `data/motifs/`.

---

## Attributes and scaling

**Three stats, and every one of them is the number it sounds like**: `DMG`,
`Actions` and `HP` on `Duelist`, the player's straight out of `data/duelists.json` and a
creature's out of its motif file under `data/motifs/`, grown by the ascent curve to the fight it is
met at. Life is HP. The action-point budget is `Actions`. Damage is
`DMG × the card's own multiplier ÷ 100`.

**Damage reduction is a percentage on the attacker and a percentage on the target, and no
attribute is either of them.** The **earth status** blunts what its carrier deals out;
amplification raises what its carrier takes in. A durable combatant is one with high `HP`, or with
earth standing on whatever is hitting it. Anything that should reduce damage extends one of those
rather than arriving as a third system — two mechanics quietly stacking is the failure to avoid.
**Nothing shaves a fraction off an incoming hit**: a shield eats one hit whole, and that is the
whole of the defending side.

**The terms of a hit compose in a fixed order**: the concept's damage plus any flat bonus, the hand
multiplier, the attacker's weight, then the target's vulnerability. Weight sits on the attacker's
side of that line because it says how hard they can still swing; everything after it happens to a
hit that has already been blunted.

**Nothing subtracts from the action budget, so it has no floor.** Every term is non-negative. A
future subtraction brings its own floor with it.

`[?]` **Nothing has measured the roster against a hit per card.** Enemy HP and damage are tuned
by hand, no simulation exists, and losing to a floor that cannot be beaten looks exactly like
losing to bad draws — see *Enemies* below.

`[?]` **How enemies scale up the tower.** Enemies are fully-specified records with no level
term. A level multiplier — damage `level × 10`, speed `level × 5` — is the shape that has been
suggested. Nothing scales with floor today.

---

## Cards

### Forms and types

**Two axes, and they are not the same one**. `combat.Category` says *when* a card
resolves and has two values; `combat.Form` says *what kind of card it is* and has four. Category
is the coarser and is derivable from the form — everything outside Defend is an attack — so the
**form is what a card puts on its face** and what a hand is counted on.

**The attack set is a 3x5 ladder: three forms by five cost tiers, filled**, and the tiers are
identical across the forms. A form is *which* pair you are building toward, never a stronger
or weaker way to build one.

**The middle three rungs are the deck; the two ends ship at zero copies**. A run opens holding
1/2/3 AP cards and can never buy anything else, so the only way to hold a Poke or an Impale is a
**Debase** or a **Exalt** essence walking a card off the end of the three. They are real
registered concepts all the same, because `combat.Neighbor` derives the ladder from the
registry, and a rung that does not exist is a rung an essence cannot step onto — without the
ends, Debase is dead on every 1 AP card and Exalt on every 3 AP one.

| Form | 0 AP · 0.25× | 1 AP · 0.5× | 2 AP · 1× | 3 AP · 2× | 4 AP · 4× |
|---|---|---|---|---|---|
| **stab** | Poke | Jab | Thrust | Skewer | Impale |
| **slash** | Nick | Cut | Slash | Cleave | Sever |
| **crush** | Tap | Thump | Bash | Smash | Pulverize |

| Form | Concept | AP | Effect |
|---|---|---|---|
| **stab** | Poke / Jab / Thrust / Skewer / Impale | 0 / 1 / 2 / 3 / 4 | Stabs for `DMG/4` / `DMG/2` (both min 1) / `DMG` / `DMG × 3` / `DMG × 5` |
| **slash** | Nick / Cut / Slash / Cleave / Sever | 0 / 1 / 2 / 3 / 4 | Slashes for the same five figures |
| **crush** | Tap / Thump / Bash / Smash / Pulverize | 0 / 1 / 2 / 3 / 4 | Crushes for the same five figures |
| **defend** | Flinch | 0 | Raises **1 shield** |
| | Brace | 1 | Raises **1 shield** |
| | Block | 2 | Raises **2 shields** |
| | Guard | 3 | Raises **3 shields** |

**The 3 AP attacks pay triple.** At double they would be a point dearer than the 2 AP card for
exactly the same damage per point, which is no reason to play one; at triple they buy a figure the
budget cannot reach by spending the same points on cheaper cards.

**The defend ladder is four rungs and the dealt two are the middle**.
`Flinch` at 0 AP and `Guard` at 3 AP ship at zero copies exactly as the outer attack rungs do, so
the deck opens on Brace and Block and the two ends are somewhere an essence can walk a card to.

**Nine attack concepts × five colors = 45 cards; two defenses × five colors = 10.** A **55-card
starting deck** — the zero-copy rungs are in the file and not in the pile. **No card in the
player's deck is drab**: every card ships in one of the five elements, the defenses included.

**A 0 AP card is bounded by the count rather than the cost**, which is the shift `minCardCost`
already took deliberately when a Hone could drive a card to free: a turn is capped at
`MaxActions` cards however cheap they are. **A 4 AP card pays 5x, 1.67× a Skewer for 1.33× the
price**, so it is the one rung that pays more per AP than the rung below it, and it buys a single
figure a five-card turn cannot otherwise reach — which is why it is an essence's prize rather than
something a run can stock.

**The defenses sit on the same ladder the attacks do**, 0 through 3 AP. **The price is the count**:
one AP buys one shield, and that is the whole of the pricing decision. It is the flattest rung in
the game on purpose — the attack tiers buy 0.25x, 0.5x, 1x, 3x and 5x, where the defend tiers buy
one, two and three — because a shield is *a hit you do not take* rather than a figure, and a curve
on it would make the top card the only one worth holding.

**Flinch raises a shield for nothing, and that is the floor rather than a mistake**. A shield
eats a whole hit, so there is no fraction of one to fall to: where Poke is a Jab at a quarter
of the damage, Flinch is a Brace at none of the cost. What bounds it is the count — a turn plays
at most `MaxActions` cards however cheap they are — rather than the budget, which is the same
shift `minCardCost` took when a Hone could drive a card to free. **The duelist's own shield cap
is not part of that bound any more**; see §Shields.

**`combat.Neighbor` walks this ladder too**, so an Exalt promotes a Brace and a Debase demotes a
Guard. A free shield changes how many hits a run takes for the rest of the tower, and that is
something a run is **allowed** to build toward: ten defenses shrunk to Flinches is five free
shields a turn, and nothing takes the fifth away. `Neighbor` matches on the *verb* rather than
being pinned to attacks, so the two ladders can never step onto each other.

**`Bash` is the 1× reference the ladder is written against**, and that is why the crush form
holds the name: `DMG` on the fighter card is `Bash.Damage(DMG)`, so the figure the player reads
is what one middle-rung card deals. Nothing stops that reference moving to another form's middle
rung; it is one constant. **It is not a term in the damage formula** — a hand's multiplier
applies to the cards' own hits, never to a reference swing added on top, so `DMG` reaches a hit
only through its card.

**The opponent has two cards of its own and they belong to no form** — `Attack` (2 AP, `DMG`)
and `Heavy` (3 AP, `DMG × 2`), priced against the player's tiers. `FormNone` is a real answer
rather than a fallthrough: forms are the *player's* deck axis, and an enemy card claiming to be
a crush would be claiming membership of a deck the player can build hands against. They draw
with a blank corner.

**The three forms cost the same and hit the same, and differ only in which cards pair with which.**
That is enough to make a hand a *choice* — holding two Cleaves is not the same as holding a Cleave
and a Smash — and it is deliberately the whole of it.

**Every card carries its effect in words on its face**, verb first, filling the card beside the
cost column. The attack text names the form's verb — "Stabs for 2x DMG" — rather than opening
"Deal" on all nine, so the corner mark is not carrying the distinction alone. The wording is
`cardEffects` in `internal/screens`, beside the prose the fight log uses: the rules package
names actions and never describes them. **Short words are a hard constraint** — the column is about
a dozen characters wide — and two tests hold the wording to it.

**The corner mark is drawn art**: a spear for stab, a scimitar for slash, a club for crush and a
shield for defend, in `assets/form/`. **It is authored once per form per element and nothing is
tinted** — the hue is in the drawing, and the corner mark plus the cost ticks under it are where a
card says its element. The border does not.

### Shields

**A shield eats one incoming hit, whole**. No damage and no partial
figure: the hit lands nothing at all, and the feed says so in a line of its own because there is
no damage line for it to hang off. Brace and Block raise one and two shields for one and two AP;
Flinch and Guard are the ends of the ladder and ship at zero copies.

**The point is that the player decides how many hits they take.** A creature turn is a known
number of attacks, so a shield turns "how much is this going to hurt" from an estimate into
arithmetic. A percentage cannot: a fraction of an unknown figure is still unknown.

#### A shield eats the heaviest hit, not the first

A turn is several discrete hits of different sizes. **What a shield costs to raise does not vary
with the opponent's queue, so what it is worth must not either** — a shield spent on whichever card
happened to be queued first is worth two damage against a Giant Bat opening with a Nip and ten
against the same three cards in the other order.

A shield takes the biggest hit of the turn, a second shield the next biggest, down the order.
Ties go to the earliest, so the choice is a function of the turn and nothing else. **The rule is
the same whichever kind of attacker is swinging** — `combat.shieldedHits` is the one ranking — and
for a hand-forming attacker every landing is a hit to rank, an echo's included.

**It is decided at the top of the creature's turn, before any of it resolves**, which is the part
with a consequence. The ranking is a snapshot: a status landed by an early card amplifies the ones
after it, so a shield can be provably not-optimal in hindsight. That is paid knowingly, and what it
buys is that the whole exchange is *showable* — every broken attack can be struck and cracked before
the creature swings, rather than one card silently doing nothing at a time. Re-ranking as the turn
resolved would win a little damage and cost the player any way of watching it happen.

**The matching element goes first** *(owner's call)*. Every shield carries the element of the card
that raised it, and a shield takes the heaviest hits **of its own element** before anything else;
only the shields left over after that take the heaviest of what remains, whatever its element. So
one ice shield against an ice Nip and a fire Drain eats the Nip. That is the trade the player made
by raising ice against an ice creature: the shield is spent where it banks an action point — see
§A shield of the hit's own element banks an action point — rather than where it saves the most
life. Ties still go to the earliest.

**Ranked on the card's own damage, which is the whole of the arithmetic.** Everything downstream —
the attacker's weight, the target's vulnerability, the hand's multiplier — is one multiplier applied
identically to every hit in the turn, so none of them can reorder two. See `combat.shieldedSlots`.

**This is a straight buff to shields, and it scales with how spiky a creature's deck is.** A
swarm of identical small attacks is unaffected; a deck with one big card in it is now much
easier to blunt. Nothing simulates a duel, so no test catches what that does to a floor.

#### A shield of the hit's own element banks an action point

**A shield that eats a hit of its own element buys its owner one action point for their next turn.**
An ice Brace eating an ice goblin's Bash is a block *and* a point; a fire Brace eating the same Bash
is a block and nothing else. Basic is no element and matches nothing.

- **The next turn only.** `Duelist.Surge` is spent by the start of its owner's own turn, the same
  moment the shields lapse, so the points buy exactly the turn after the blocks and nothing later.
  They never outlive a fight.
- **No cap.** Five ice shields eating five ice hits is five more points. What bounds it is how many
  shields a turn can pay for and how many hits a creature throws; the five-card cap on a turn still
  holds, so a big surge buys dearer cards rather than more of them.
- **It is the mirror of the fizzle** — see §A creature's own element. The duelist has no element
  for a creature's hit to fizzle on, so matching a creature's element pays the player on defense
  where it costs them on offense.
- **The screen says it twice, as placeholders**: `+1 AP` over each card a matched shield breaks,
  and `(+N surge)` beside the budget while the player plans the turn it pays for.

#### The turn reads: shields, then the hand, then the hits

The defend phase resolves first — see `combat.Categories` — and the screen plays it as **one
gesture**: every shield in the turn goes up together, pips out of their own cards into the row
along the bottom of the duelist card, with **no card lifting off the table**. Only then is the
hand announced, with the cards that formed it raised, and only then are the hits worked out — a
line of arithmetic under every card of the turn, all of them at once. A defense's line usually
comes to 0.

**A lift says "this card is acting now", so exactly one thing in the turn may use it.** The defend
phase is one gesture for the whole set — the shield break's rule and the deal cascade's — because
three pauses over one phase say three things about cards where the phase says one thing about the
turn. Nothing lifts on a card's own announcement; the raise belongs to the hand's, and it ends
there.

**The rung is raised and the hits are thrown, and they are different sets.** On a turn of two
shields and one attack the Pair is the two shields, so they are what stands up; every card throws
a hit, and the attack's is the one that hurts while being in no part of the hand, so it stays down. `Blow.Rung` against `Blow.Cards` is
the distinction in the rules and `Event.RungCards` against `Event.HandCards` is how it reaches the
screen. **A row held up through the arithmetic is the announcement still being made while the thing
it announced is being read out**, so the cards go back down when the hits start.

**The No Hand is announced and raises nothing.** It is a rung and the word says what happened, but
it is the turn that made no hand — its `Blow.Rung` is a single card picked by damage rather than by
counting, and standing that card up says it did something while every other attack in the turn
lands beside it unraised. `Blow.BuiltAHand` is the predicate and it gates the lift alone.

**The creature's turn is the exception to all of it.** It names no hand, so its card lifting one at
a time is the only thing saying which hit is landing.

**None of it reaches the rules.** The order the events resolve in is unchanged, `ResolveRound`
decided the whole round before a frame was drawn, and the bundle is the screen reading forward in a
log it already holds.

**They last exactly the turn after they were played**: raised at the end of your turn, standing
through the opponent's whole turn, gone before you act again. That is the schedule a raised guard is
already on, and `expireDefenses` is the one function that says when. **An unspent shield lapses**,
and is announced when it does — a stockpile carried through quiet rounds and cashed at a boss is
the banking mechanic these cards replaced.

**One card raises at most five; a duelist holds as many as the turn can pay for.** The five is
`MaxActions` — a card promising a sixth shield would be promising one its own turn can never see
thrown at it — and `RegisterConcept` refuses a concept declaring more.

**The duelist's total is bounded by the action budget alone.** Three Guards in a turn is nine
shields and is meant to be. Clamping the total to five would price a shield at what it stops
rather than at what it cost, and a player paying six AP for two Guards has bought six shields
whether or not the next turn throws six hits.

**The pip row on the duelist card holds six**, which is what fits the bottom band at the current
pitch, and it is a *readout* limit rather than a rule — `screens.maxShieldPips`, not
`combat.MaxShields`. A duelist standing behind ten draws a full row of six and the true count is
on the engine. **A row that says what a big count actually is has not been designed**; a seventh
pip is a redesign of the band rather than a bigger number.

#### Only the player raises shields

**Creatures raise nothing at all.** Every creature deck is pure attack, and every creature is a
solo attacker whose turn resolves card by card with a hit each, so a count buys one hit out of
several — a real decision about how much of a turn to absorb.

**A shield facing a hand-forming attacker is on the same terms**: the player's turn is a hit per
landing, so a shield would eat the heaviest of them and leave the rest. Nothing produces that today,
since no creature raises one, and `VerbShield` would work on either side.

**A round's action points are the stat on the card, plus the surge the last creature turn's
matched blocks banked, and nothing else**. `Duelist.ActionPoints()` is the whole of a turn's
budget: nothing grants extras mid-round and no card draws another. **What that buys is a budget
the player can plan a whole fight against**, with the one bonus on top being something they earned
by the shields they chose and can see coming — a creature's element is on its card. Creatures raise
no shields, so their budget is the figure printed on their card.

**What it costs, stated:** a creature spends every turn swinging, so the roster is modestly
stronger and considerably more predictable. **Every creature deck is pure attack**, and none
falls below three distinct concepts.

#### The break on the card

**A blocked attack is drawn as a broken window over the card that will not fire**, and the
shield that killed it crosses the table to do it — one beat between the player's turn and the
creature's, every blocked attack at once.

**A shield spent invisibly is a rule the player pays for on every defend card and never watches
work** — the creature's card comes up, nothing happens, and a line in the feed says so after the
fact. The break is also the only thing that makes the "heaviest, not first" rule above legible.

**The break is two drawings of one thing and it stays after the animation.** The card wears the
break for the rest of the round, because the table is the account of what the turn was and a card
that never fired has to still say so once the turn is over.

**It is the first of a general mechanism.** A *mark* is a picture drawn over a finished card face
about that card's situation, as against an *upgrade*, which is what the card permanently is and
which takes the left column rather than covering anything. See `internal/cards/mark.go`, which holds
the split between the settled mark and the animation that puts it there, and
`internal/screens/combat_shatter.go`.

**Marks are a set.** A card can be several things at once, so `cards.Mark` is a bitmask rather
than one value, and the order they are painted in belongs to the renderer rather than to whoever
asks. The second mark is **the tutorial's highlight**: a card the lesson is pointing at is
washed in the tutorial's red rather than having a red frame drawn round it. A frame outside a
card is a thing near the card; a tinted card is the card answering — and where a step names
several cards, a frame each is a row of loose rectangles while a tint each is just the cards.
Relics and essences are expected to take marks too.

### Concepts and deck composition

**An attack concept ships as five cards: one per primary element.** That is the rule for adding an
attack, not just a description of the starting deck. **A defense ships in the same five**, because
it carries a color for the relic discount and for the hand axis even though nothing it does is
elemental.

45 + 10 = **55 cards**, and a hand of eight is 15% of that. **The hand is a constant eight**;
nothing in the deck draws extra cards, and the only answer to a bad draw is the Discard button.

### The deck is a starting position, not the game's deck

**Every count on this page describes the deck a run opens with, and a run spends itself changing
it**. This is the single easiest thing to forget when reasoning about
hands, costs or reachability: the 55-card grid above is where the player *begins*, and by floor
three it may be a different deck in size, in color and in what it costs to play. A figure derived
from the starting composition — a rung's odds, the cheapest way to build a hand, how many cards
share an element — is a fact about round one of fight one and about nothing else.

**Three mechanisms change it, at three different lifetimes.** Keeping them apart is what stops "my
deck is half ice" from being confused with "my deck contains more ice cards".

- **An essence edits the run's deck itself, permanently.** A won fight offers two; each names a
  card and an aspect — `element`, `remove`, `duplicate`, `cost`, `amount`, `promote`, `demote`.
  After a `remove` the card is gone from every pile and the deck is genuinely smaller; after
  Exalt or Debase it is a different concept from then on. Nothing takes an essence back.
- **A relic can rewrite the deck as a fight's deck is built** — the `deck-built` moment. Atrophy
  steps every 3 AP attack one rung down its own form's ladder, so a Skewer is dealt as a Thrust for
  the whole fight. It lasts as long as the relic is worn and no longer.
- **A relic can also rewrite a card as it is dealt** — the `card-drawn` moment, which leaves even
  the fight's deck alone. Frozen Lightning and Frozen Orb both `set-element` to ice, which is why a
  run wearing the pair reads as zero lightning and zero arcane; take them off and the colors come
  back. `card-cost` relics are the same shape applied to the price rather than the color.

**The deck panel shows the *effective* deck** — what the run will actually be dealt, with every one
of the three applied — which is the number to trust when reasoning about a live run, and is not the
composition any tool prints.

**Every axis a hand is scored on can be moved, so a build can manufacture any of them.**
Elements are the loudest case: two common relics fold two colors into a third, and a deck that
is half one element makes an Elemental Five of a Kind an ordinary turn rather than the hand the
round-one simulation reports at a 0.29% score. **Cost moves too**: a relic or an essence taking
a point off a card changes which cost tier it sits in, and no rung counts cost today, so what
that moves is what a turn can *afford* rather than what it forms. **Concepts and forms move
too** — promote and demote walk a card along its form's ladder, so Debase and Atrophy both turn
dear cards into copies of the cheap card below them, which is a Card Two Pair the starting deck
could not deal. Only the *form* survives a rung change, since a ladder is one form's. **All of
it is the intended shape of a build, not a leak**: the ladder is priced against the starting
deck on purpose, and out-earning that price is what relics and essences are for.

**So read `tools/handodds` and `tools/handsheet` as describing fight one.** Both deal from the
shipping deck with no relics worn, which is the only composition that can be stated without naming a
particular run. Neither says anything about a deck that has been played with.

**Five copies of a concept is the ceiling of the *starting deck*, and it shapes the hand table.**
No attack concept ships more than five times, so **a Card Four of a Kind necessarily shows four of
the five colors** — copies of a concept are all different elements, so it is also the hand that
lands most of the statuses the player is reliced for. **A Card Five of a Kind is the whole color
set in one concept**, dealable from the starting deck. See the reachability table below.

**The deck list is data.** `data/duelist_cards.json` holds every concept the player's deck is
built from, the form each declares, the elements each ships in, and how many copies.
`startingDeck` is built from it.

### The card language

**A card carries its own rules, and it is a record rather than an enum value.** A closed enum
with a switch per property holds a dozen player cards and cannot hold three or four bespoke cards
for each of ninety-six creatures — hundreds of concepts, each wanting its own multiplier and its
own name.

Seven fields, and the player's cards are written in the same language as every creature's:

| Field | Means |
|---|---|
| `Label` | what the card face says, and — scoped by its owner — the rules identity |
| `Verb` | **attack · shield**. A closed vocabulary; a third is a Go change |
| `Amount` | read against the verb: percent of DMG, or shields raised |
| `Cost` | action points |
| `Form` | stab / slash / crush / defend, or none — the player's deck axis |
| `Elements` | which colors the concept ships in |
| `Copies` | how many of each |

- **There is no `Category` column.** Which phase a card is in falls out of the verb, and
  carrying both would let a file say a card is an attack that raises shields.
- **`Elements` and `Copies` are two axes and neither substitutes for the other.** The player's
  attacks ship one per color and its defenses one per color — the same cards reached along
  different axes. An enemy's cards are all `basic`, so `Copies` carries its whole deck size.
- **A key is scoped to its owner** (`ClearSlime1.Engulf`). Forty creatures want a card called
  `Bite` and they do not all want it at the same multiplier; the label collides freely, the key
  must not.
- **A card does not say who it lands on; its verb does.** An attack lands on the opponent and
  everything else on its own duelist, and there is no field to disagree with — so a card cannot be
  aimed anywhere the rules do not resolve, and nothing has to be validated to stop it.
- **The file is the rules, so validation is all there is.** `combat.RegisterConcept` refuses an
  unknown verb, a zero amount, a shield count above `MaxActions`, and the unbuilt half of the
  grid. A bad record panics at init. Nothing cross-checks a declared cost against a tier, because
  there is no second place a cost is written down.

**A card never names a status, and that is load-bearing.** See *Elements* — what a color does is
decided by the source of that color on the card's owner, and a relic may later decide *which* fire
a fire card applies. A card that named its own status would be deciding something that is not its
to decide.

**The deck is not 52 cards and the playing-card instinct is the wrong one.** 13 ranks × 4 suits
is a shape borrowed from a game whose hands this one only names; the ladder decides the size
instead, and nine *dealt* attack concepts by however many colors exist is what three forms by
three dealt tiers produces. `basic` is the absence of an element rather than a suit, and arcane is
a suit rather than a variant of nothing.

### Hover and long press

**Hover explains, and long press is the same reveal on a touchscreen.** What a player needs from
a card is not a bigger picture of it but the arithmetic behind its figure, so neither gesture
magnifies anything.

**Resting the cursor on something explains it.** A card gives the whole damage chain term by
term — your DMG, the card's own multiplier, every relic that matches, the result, and a line
saying the hand multiplier comes after; a relic gives its authored line from `relics.json` and
where it fires in the worn order; a fighter card gives its figures and every status standing on
it, which is the only place a badge can be read.

- **It arrives after a dwell of a beat and a half**, about six tenths of a second, so a cursor
  crossing the hand on its way to DUEL! does not strobe eight panels. A panel that appears before
  you have decided to want it reads as a flicker following the mouse rather than as an answer. The
  dwell is a proportion of the game's one speed, like everything else that moves, so the speed
  setting carries it.
- **The hand explains itself only while the queue can be edited.** A played card is still in the
  hand model during playback while being drawn on the table, so its old seat would otherwise answer
  for a card that had visibly flown away.
- **The panel is placed beside the thing, not under the cursor**, so it never covers what it is
  about and does not slide around inside one card.
- **Nothing in it recomputes a rule.** Every figure comes off the same walk the engine compounds,
  `combat.RelicContributionsAt`. A tooltip doing its own arithmetic would be a second implementation
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

**There are five elements and no more**. Every one of
them has cards, a color and a status, so an element is a complete thing rather than a name waiting
for rules. **A color ships with all three or it does not ship**: a card in every dealt concept, a
hue of its own, and a status.

**What a sixth color costs, so it can be priced before it is proposed.** None of it is the cards
— the cards are one line of JSON. It is everything downstream:

- **Every elemental hand gets harder, and the whole ladder has to be re-derived.** A color would
  become one in six, so the reachability of every elemental rung moves and the multipliers priced
  against it are wrong until `tools/handodds` is re-run. See the reachability table.
- **The of-a-kind ceiling moves with it.** A concept ships one card per color, so the top card
  rung is however many colors there are — a sixth makes a Card Five of a Kind commoner and adds a
  rung above it that nothing has priced.
- **The deck overlay runs out of room.** A row of cards per color plus the tally band has to fit
  the modal, and the card itself cannot shrink: the form mark is drawn on a 32px canvas.
- **Four relics, not one**, since each color carries a damage, a status, a discount and a growth
  relic — **plus the flip relics, which are a full cross-product** of ordered pairs and grow
  quadratically.
- **Every cataloged deck seed has to be re-checked**, because the shuffle deals a different deck.

`basic` is the absence of an element, not a sixth color.

### Color

| Element | Color |
|---|---|
| basic | mid gray |
| fire | orange |
| ice | medium blue |
| lightning | yellow |
| earth | green |
| arcane | purple |

`cards.BorderOf` is the live table; this one says what the colors are *for*. Basic is a mid gray
because the card surface is off-white and a near-white border on it is invisible.

**Every word that names an element is written in that element's color**. An essence reading
"CARD BECOMES ARCANE" sets ARCANE in the arcane purple; a relic reading "Fire attacks BURN and
CHILL the target." sets three words across two colors. It reaches the card faces, the tooltips
and the fight log, because the words are the same words wherever they are read.

- **A status is written in the color of the element it belongs to**, so BURNING is fire orange
  wherever it appears. `statuses.json` carries an `Element` and a `Verb` — BURN against BURNING —
  and a status shipping without either fails a test rather than going quietly uncolored while
  every other one is lit.
- **No new colors.** The vocabulary is `cards.BorderOf` exactly, which is what the deck panel's
  row labels and the arithmetic panel already read — so a color cannot drift between a card and
  the sentence describing it. Lightning and earth read thinnest as text, lightning
  having already been darkened once to survive the off-white card; if either becomes unreadable the
  answer is to move `BorderOf`, which moves the whole set together, rather than to open a
  text-only variant.
- **Bold was asked for and deferred** — there is no bold weight of the card font in `assets/`, and
  faking one by overdrawing is the smudge the run-splitting drawing exists to avoid.

One collision is live: **the player's green swatch sits near earth's green**. "Green is you,
gray is them" is a screen-wide rule and an element breaks it. What holds it together for now is
that the two are never seen side by side — a swatch is a square in a pane row, a border is the
edge of a card — so the fix is deferred rather than done. Either the sides stop being
color-coded or earth takes a green far enough from `playerSwatch` to read as a different idea.

### Statuses

*Implemented in `internal/combat/status.go`.* Each element has a status it applies **to whoever
took the hit** — **and only if the attacker is wearing that element's relic**:

**`data/statuses.json` is the catalog and every figure here comes off it**, so read the file
rather than this table when a number matters. All five last **two rounds** and none of them
stacks.

| Element | Status | Verb | What it does |
|---|---|---|---|
| **fire** | BURNING | BURN | ticks 50% of the attacker's DMG at the end of each round it survives, minimum 1, frozen at the DMG that lit it |
| **ice** | CHILLED | CHILL | one card off the front of every turn it outlives |
| **lightning** | SHOCKED | SHOCK | 25% chance each of the victim's hits misses, rolled every hit |
| **earth** | WEIGHTED | WEIGH | the victim deals 25% less damage |
| **arcane** | WEAKENED | WEAKEN | the victim takes 100% more damage from everything, burn ticks included |

**WEAKENED is a different shape from the other four, and that is worth knowing before a sixth is
designed**. Every other status modifies what its carrier *does* — how
hard they swing, how often they connect, how many cards their turn holds — so it is read off
whoever is acting. WEAKENED modifies what its carrier *takes*, so it is read off whoever is being
acted upon, and it is the first thing in the damage pipeline to be read off the victim. Two
consequences follow and both are intended:

- **A relic applying it is worth more against a slow opponent than a fast one**, because the value
  is in the hits that land during its two rounds rather than in the ones you throw — **the later
  hits of the very turn that lands it included**, since a status lands between hits.
- **It amplifies a burn tick as well as a hit.** A tick is damage the carrier takes, and exempting
  it would have made the rule *"damage, except the kind that arrives at the end of the round"* —
  which is a sentence no card face can carry. Fire plus arcane is therefore the sharpest pair of
  statuses in the game, and it is a build rather than an oversight.
- **It does not stack and it is capped**, like everything else here: a second application refreshes
  the clock, and `combat.maxAmplifyPct` holds any sum at 300. Amplification is the one percentage
  with no natural ceiling — a miss chance and a weight both stop at *nothing reduces a hit to
  zero*, and this one stops nowhere at all.

**Statuses are off by default, and the relic is what switches one on**. A
bare fire attack is a plain attack with a red border: it forms hands exactly as any other
card does and it leaves no status behind. What an element does on its own is the fizzle — see
§A creature's own element — which is about the target rather than the card. The worn relics are read off the **attacker**, per hit,
before anything is applied.

**Why a relic pays for it.** A status given away free leaves its element's relics with nothing
to *be* — each one has to invent a second mechanic to sell, because the thing the color does is
already happening. Charging a relic for it makes the element set a hand axis on its own terms and
makes a relic the thing that turns a color into a rule. It also gives the loot a shape: what a
relic buys is legible in one line of card text, and a second and third relic are worth buying
because one relic is one element.

**Enemies never wear relics.** The zero value is what an enemy is hydrated with and nothing sets
it, so an enemy's colors are inert by construction rather than by a rule written down somewhere
else.

### A creature's own element

**A creature is dealt one element with its floor, and it is that element's own.** It decides the
creature's picture and the colour of every card in its deck, and it is a rule: **a hit of the
creature's own element fizzles**. An ice Bash thrown at an ice goblin lands nothing at all — a 200
damage ice hit is 0 — while the fire Bash beside it lands as it always did.

- **Per hit, not per hand.** Every card of a turn lands its own hit, so a fizzle wastes exactly the
  hits of the matching element and nothing else. See §Damage: a hit per card, one multiplier.
- **The card still forms the hand.** The hand is read off the turn before a hit is thrown, so two
  ice Bashes at an ice goblin are still a pair; what fizzles is the damage.
- **A fizzle wastes the whole hit.** No damage, no drain, no status, and no growing relic steps on
  it — everything a relic would have done on that hit goes with it. It is decided before the shock
  roll, since it is the target's nature rather than luck.
- **A wildcard never fizzles** *(owner's call)*. It counts as every element when a hand is formed,
  and a card that matched every creature's element would be wasted against all of them.
- **Basic never fizzles, and the duelist has no element**, so the rule runs one way: a creature's
  hits always land on the player. The mirror is on defense — a shield of the creature's element
  banks an action point when it eats one of its hits. See §A shield of the hit's own element banks
  an action point.

**It is the one thing an element does on its own**, without a relic. Everything else a color does
is switched on by something the player is wearing.

The fight log says `fizzles - its own element` on the hit's line and the hit's arithmetic line
takes `FIZZLE` where a shock takes `MISS`. `combat.fizzles` is the rule and `KindFizzled` the event.

### One rule, two sources — the intersection

**An element does something only where a card's color meets a source of that color on its
owner.** The player's source is a **relic**. An enemy's is an **elemental affix** — its own, or the
floor's. Neither side gets statuses free; both get them at an intersection.

What this buys is that `Duelist.Relics` turns out to be the general mechanism rather than the
player's half of one: an affix sets flags in the same array. Nothing new is needed for it, and the
name is what should eventually change rather than the machinery.

**A card still never names a status**, and that is the reason the rule is worth stating this way. A
relic may later confer *which* fire a fire card applies — different relics, different burns — so the
decision belongs to the source and not to the card. See *The card language*.

**Enemy statuses are blocked on affixes, which do not exist.** Every enemy card is authored
`basic`, so today the whole element system still runs in one direction only. Coloring an enemy
card before an affix can gate it would hand it a free status, which is exactly what this rule
forbids.

**A run opens wearing no relics at all**, so **every element is inert
until the first one is bought**: an ice Bash is a plain Bash with a blue border. That is what
makes the shop the first thing a run saves for. `session.StartingRelics` is the seat for putting one
on without playing to a shop — the relic counterpart of `deckSeedName` — and it ships empty.

**A status shows as a badge along the bottom of the enemy card**, from
`assets/effect/`. It is the only place a standing status is stated, and it has to be: two of the
four bite something the player has not done yet — a chill takes a card off a turn not yet queued,
a weight blunts a hit not yet swung — so without a badge they are learned by being surprised.
The row is centered and closes up as it fills. Earth's art is a placeholder. **The player's card
carries no badges**, because nothing can put a status on the player: the enemy wears no relics.

**Element is a rules type.** `combat.Card` is a concept plus an element, and it is the one card
type the hand, the queue and the round all use — `screens.actionCard` is an alias for it, so a
card is never converted between the screen's idea of one and the engine's.

**Cost is a property of the pairing**. `Card.Cost()` is the card's own printed
figure and `Duelist.CardCost` is what it costs the duelist holding it, discounts included — which
is what everything that spends or checks a budget reads.

#### The trigger: every hit that connects

**A status is applied by a hit, off that hit's card.** The relics match against the card that threw
the hit, and each `apply-status` they fire lands once for that hit — so a turn of three fire hits
lands a burn three times, each refreshing the last. An all-basic hand lands nothing, because no
elemental rule matches a colorless card. **A defend card throws a hit like any other and lands its
status** *(owner's call)* — one rule, every card hits, with nothing special for a verb. The cost is
taken knowingly: a 1-AP Brace is as good a status delivery as a 1-AP Jab, and buys a shield too.

**This is the whole of what color does to a hit.** An element earns its keep by what it leaves on
the victim, never by hitting harder — there is no damage multiplier anywhere on the element axis.
Three consequences:

- **A color lands once per hit that carries it.** Nothing stacks, so two ice hits and an ice hit
  more are one chill refreshed three times; what more hits buy is more chances past a shock and a
  shield, and a status that lands in time to shape the hits after it.
- **Every card carries its color, in the hand or not.** `Bash, Jab, Bash` in fire, ice, fire is a
  fire Pair, and the ice Jab's hit lands a chill anyway; a fire Brace beside them burns.
- **A status lands because the hit connected, not because it hurt.** A hit blunted to nothing still
  connected, and making the status conditional on the final figure would let the target's own
  statuses silently un-apply an element the attacker had already paid for. **A miss and a block land
  nothing** — the hit never connected.

The cost, stated: **element is mechanically inert on a defend card**, which still carries one for
the relic discount and for the hand axis.

**Magnitude is per hit, not per card.** A fire Jab and a fire Smash apply the same burn, so the
cheapest attack in the deck is the cheapest status delivery. The concept ladder prices damage;
the element ladder does not exist. Making status scale with the card is a second axis and a
design change. **Fire is the one that scales, and it scales off the *duelist*** — a share of
DMG, frozen when it lands — which is a different axis from the card and does not reopen this one.

#### One lifecycle, learned once

**Nothing stacks; a second hit resets the clock, and everything clears at the end of the round
after the one that applied it.** **How long is authored per record** — `Rounds` in
`data/statuses.json`, read into `combat.Status.Rounds` — and every status in the file writes 2
today. **It cannot be 1**, and that is a rule rather than a preference: side B acts second, so a
status B applied would expire before it ever bit anything. `RegisterStatus` refuses a record that
lasts no rounds at all; the floor of 2 is not enforced, so an author setting 1 gets a status that
never fires.

**Nothing stacks, and that is the base rule so a relic has somewhere to go.** Adding amounts
makes a status something to pile on rather than something to keep up. **Every hit that connects
lands its card's statuses**, so four fire hits refresh one burn four times rather than stacking
four. A relic that *does* stack is a relic someone can design.

**A status lands between hits, so it can reach the hits after it.** Vulnerability landed by the
second hit of a turn amplifies the third. *(Owner's call)*: it makes the order of the cards matter
more. Nothing has measured it.

**The ceiling is `combat.maxStatusPct` — 99.** One number holding *any* summed percentage short
of one, so nothing misses every time and nothing stops a hit outright. Two registration checks
hold the same line for a single record, which is what makes a catalog edit unable to reach it
either.

Per-element tuning is one constant each away, and **nothing measures what moving one does**.

#### Lightning is a roll, and it is the only one in the rules

**A shock is a 25% chance each hit misses, rolled on every hit the shock outlives** — once per
landing, so a five-card turn is five rolls and an echoed card rolls for each of its landings.
Nothing is consumed by a roll: with no stacks to wear down, a shock that spent itself on contact
would be a two-round status that reliably lasted one attack — a duration doing no work.

**A shock is a chance and never a certainty.** A certain miss would delete every hit of the
opposing turn, so a 1 AP lightning Jab would erase an 8 AP Four of a Kind outright for a point. A
roll is what was chosen, because lightning should *feel* unreliable — a design reason rather than a
balance one. With nothing stacking, the ceiling is the number itself.

**Rolling per hit rather than per turn keeps the average and narrows the spread.** A shocked
duelist loses about a quarter of each turn instead of a quarter of their turns outright; the
expected damage is the same and a turn wiped out entirely becomes rare. *(Owner's call.)*

**What it costs, accepted rather than argued away:**

- `internal/combat` is no longer pure integer arithmetic. It takes an injected `*rand.Rand` on
  `ResolveRound` — never a package global, per the determinism rules — and a nil source means
  "no rolls", which is how tests and previews stay exact.
- The stream advances **per hit**, so a change early in a duel reshuffles every roll after it —
  and so does any change to how many hits a turn throws, an echo relic included. That cost is real
  and is accepted.
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
- **Fire scales with the attacker.** A burn ticks for a share of the DMG of whoever lit it —
  `Amount` in `data/statuses.json`, read once and frozen onto the victim, so a duelist whose DMG
  changes later does not retroactively burn harder. It floors at 1, the same rule the cheapest
  attack card follows, so a duelist with very little DMG still lights a burn that does something.
  - **A burn is state that outlives an action.** `KindBurned` fires from `endRound`, side A then
    side B, and the screen's `applyEvent` reads it alongside `KindDamage` because a burn changes a
    life total with nobody acting. **A burn can kill**, and produces a
    `KindDefeated` when it does.
- **Earth applies attacker-side.** Weight says how hard you can still swing, so the order within a
  hit is the card's own term, the hand multiplier, then the attacker's weight. **Rounding is toward zero**,
  matching `scaleDamage` and every other percentage. **It is 25%**, because a smaller cut that
  cannot stack is a status nobody notices landing.
- **Arcane applies victim-side, after the weight.** Vulnerability says how hard *this body* takes
  a hit, so it is the last term: the card's own term, the hand multiplier, the attacker's weight,
  the target's vulnerability. **Rounding is toward zero**, so the two halves of one hit round the
  same way. **It is 100%** — double — and unlike the other four it is capped centrally
  as well as per record, at `combat.maxAmplifyPct`.
  - **It reaches the burn tick too**, in `endRound` rather than in the attack phase, which is the
    one place a status modifies damage nobody threw.
- **Statuses live in `Duelist.Statuses [MaxStatuses]Status`** — an array indexed by **status**,
  not by element and not as named fields. That is what makes *"consume the status this card
  applies"* expressible and is the difference between a system and a handful of ad-hoc fields. The
  price: **`StatusID` is append-only and the file decides the order**, the hazard `Element` and
  `ConceptID` also carry. Standing shields stay off this table — a shield is a card effect, and
  filing it here would say it was a status.

## Resolution — phases

A round is **a whole turn each**. Everything one side queued resolves before the other side
does anything, and within a turn the two categories go in order:

1. that side's **defenses** — every shield the turn paid for, raised in one gesture
2. that side's **attack** — one hand read off the attack cards, then a hit per card
3. then the other side, the same way

`combat.Categories` is the order and it is the single authority on it.

**Within-turn order decides nothing about what a defense protects.** `expireDefenses` runs at the
start of a side's *own* turn, so a shield raised anywhere in your turn is standing through the
opponent's either way, and your attacks are aimed at them rather than at you. What the order
decides is what the turn *reads* as: raise the guard, then swing.

**The one thing it genuinely moves is a growing relic**, which steps between the hits of one turn:
a `grow-per-card` accumulator counts the defends before the attacks score. Nothing simulates a
duel, and no test goes red for it.

**The combat screen lays a turn out in exactly this order**, left to right, with a gap at the
boundary — the row on the table reads defenses, break, attacks. That is not decoration: it is the
round's two phases made visible in the one place the round is a picture rather than a list.

**The attack phase is one hand and a hit per card.** Every attack card queued is announced, the
hand they form is announced, and then every card of the turn lands its own hit. The defenses resolve on
their own, each doing something to its own duelist.

**The run's account writes the hand once and the hits under it.** The hand's line names the rung;
under it is a line per hit — the card that threw it, its arithmetic, what became of it — and the
total of every hit. **A turn that forms no rung still takes that same heading**: the No Hand is a
rung, and naming it says the true thing rather than dressing a turn that agreed about nothing as an
achievement.

**The hand dialog acts every hit out at once on the beat the hand fires.** Under each card
on the table its hit is worked out figure by figure — the DMG flying off the duelist, the card's
multiplier off the card, each relic's factor off the relic, the hand's multiplier
off its name — and the moment a line finishes its figure flies into the target, which loses that
much as it lands. The lines run in parallel and do not wait for each other, so a simple hit lands
first. **It says nothing the event does not carry and computes nothing**, and the one thing it
changes is pacing — playback holds while it runs. See the `combat-screen` skill.

**There are two categories and no third.** Nothing resolves before the defenses, and a card
that *fed* the hand rather than contributing to it would need the phase order reopened — which is
an argument to make rather than a thing to quietly reorder.

**Why a whole turn each rather than an interleave:** an exchange that alternates card for card
is not something a player can hold in their head while queueing five of them. It also simplifies
the code — cards are gathered into their categories inside `ResolutionOrder`, one pure function
that both `ResolveRound` and the table's two rows read, so hidden information survives untouched.

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

### There is no initiative

With one contiguous turn per side there is no exchange for a faster card to lead, so a speed
number on a card would report a distinction the resolver does not make. `Actions` buys budget and
never buys priority.

**Order within a category is queue order, and two things read it**.
`groupsOf` fills largest-count-first and breaks a tie by whose first card was played first, so the
lead card — the one that names the hand and carries its element — is chosen by where the player put
it. And **a growing relic steps between the hits of a turn**, so the order the attacks are queued
in decides what each of them is worth: the first fire hit is counted bare, steps the relic, and the
second is counted at the bigger multiplier. See *Growing relics step between hits* below.

**Category order is fixed whatever order the queue is in** — a card cannot be dragged out of its
phase. A counted hand still reads the turn as a set for the purpose of *naming* it, and the
shields compose without an order among themselves.

---

## Hands

**This is where the game is meant to be.** Throwing whatever you drew at the opponent works;
*choosing a shape* and building a deck toward it is meant to work better. Hands are the
mechanism that pays for that choice.

**A hand is a damage multiplier and nothing else**. It buys no
status, no action points and no action off the opponent's turn. Statuses come from **elements
and the relics that arm them**, and that split is the whole reason this section is now short:
there is one axis, one number per rung, and one place to look for what a hand is worth.

Hands are **discovered**, not given, and discovery persists on the **profile** — part of the
roguelike unlock structure, not the run. `[?]` **Discovery is not enforced** —
`profile.Profile.HandsDiscovered` is the field waiting for it and every hand is live. Gating the
*table* is the whole of the change when it lands; nothing else moves.

### The catalog is data

*`data/hands.json`, with `data/hands_data.go` holding its shape and
`internal/combat/hand_table.go` turning it into rules. The vocabulary and the matcher are in
`hand.go`.*

**`data` holds the shape, `internal/combat` holds the meaning**, which is the division
`RegisterConcept` already draws for the deck lists. The file says how many copies of a card a rung
wants and what it pays; only the rules can say what a turn is and how wide one can be, so that is
where a malformed catalog is refused.

**This is the one thing in `data/` that the rules themselves read**, and it is why
`internal/combat` imports that package at all. Everything else there is consumed by `screens`,
`decks` or `entities` — layers *above* the rules — so the rules never needed it. That is the
line to hold if a seventh list is proposed: **ask who reads it, not whether it is data.** `data`
imports nothing but the standard library, so the edge costs `internal/combat` neither its
testability nor its freedom from Ebitengine.

**A malformed catalog panics at package init**, exactly as a malformed card record does. A hand
silently dropped is a balance change nobody made.

### The pattern: four axes, and the of-a-kind rungs wear poker's names

A hand counts **cards that agree** in the set that formed one attack — which is exactly what a
poker hand counts, so the rungs wear poker's names honestly: Pair, Two Pair, Three of a Kind, Full
House, Four of a Kind.

**The bottom of the ladder is `No Hand`**, and it is named for what it is: the turn whose cards
agreed about nothing. It is a rung in every other respect — an entry in `hands.json`, a multiplier
of its own, raisable by a stone, nameable by a relic — and it is announced like any other. What it
does not get is the lift, since a rung no cards made has no cards to raise. See §Shields.

**What they have to agree *on* is the hand's own axis**, and there are
four:

| Axis | Cards agree on | Two that form a pair | Two that do not |
|---|---|---|---|
| `concept` | the same card | ice Thump + fire Thump | Thump + Bash |
| `form` | stab, slash, crush or defend | Thump + Smash (both crush) | Thump + Thrust |
| `element` | fire, ice, lightning, earth or arcane | ice Thump + ice Thrust | ice Thump + fire Thump |
| `cost` | the same action points | Thump + Cut (both 1 AP) | Thump + Thrust |

**`cost` is the fourth axis.** It is genuinely
orthogonal to the other three in the player's deck — every cost tier holds three or four forms, and
every form spans all three costs — so a hand counting cost is not a hand counting form under
another name. **Every card carries a cost, so this axis has no absence**, unlike `FormNone` and
`Basic`.

**The of-a-kind ladder exists once per axis on the first three above the Pair**, as its own
catalog entry rather than as one entry with three readings — so a Card Three of a Kind and an
Elemental Three of a Kind can be priced apart, which they have to be: one wants three copies of a
five-copy concept and the other three of eleven cards sharing a color. **The Pair is the one rung
that is a single entry read three ways** — see *The pairs are one rung* below, which is also where
the argument for keeping the rest apart is written down.

**The axes are not parallel, and the nesting is the thing to hold onto.** A concept fixes a form,
so **every card hand is also a form hand**; element is independent of both, which is why an ice
Thump beside a fire Thump is a card hand and no kind of elemental one. That asymmetry is what the
tie-break and the multiplier ordering below both exist to answer.

**A card with no value on an axis matches nothing on it.** `FormNone` and `Basic` are absences
rather than values, so an enemy's formless colorless deck cannot build a form or an elemental hand
at all — its whole ladder is the concept axis, which is what its `Copies` field was always buying.
The player's defenses carry a color like everything else, and it is inert for the same reason.

**Exactly one hand still applies, and a tie goes to the narrowest axis.** Two Thumps and two
Cleaves satisfy the Card Two Pair and the Form Two Pair at once; the narrower one is what the
player aimed at, so `concept` beats `form` beats `element` beats `cost` whenever the multipliers
are level. `combat.Axis` is written in that order for exactly this reason and is never
serialized, so the order is free to mean something.

**Exactly one hand applies.** It wins on its multiplier — four Bashes are a Four of a Kind rather
than also the pair and the trips inside it — so a turn produces one hand with no ranking
machinery beyond that comparison.

**A lone attack forms no hand.** That is the fallback: when nothing counts, the turn is the No Hand,
named after its hardest-hitting attack card, ties going to the card queued first.

**The fallback rarely fires, and that is what having four axes buys.** Two attacks need only
share a form or a color to count together, so Smash + Bash at DMG 10 lands **33** as a Form Pair
rather than the Smash's **20** alone — and none of that comes from the multiplier being generous,
since 1.1x of two cards beats 1.0x of one. Two attacks that agree on nothing at all are the rare
case, and the No Hand is what names it.

**Every attack the turn paid for lands.** `Bash, Jab, Bash` is a Pair and the Jab makes no part of
it; the Jab lands its own hit and carries its color anyway, at the Pair's multiplier. What makes
*choosing a shape* pay more is the multiplier over every hit rather than an attack being deleted
for disagreeing. See §Damage: a hit per card, one multiplier.

**Every card in the turn is counted, and that is the matcher's rule rather than the catalog's.**
A rung cannot name the categories it counts; what is left out is decided by the axis — a card with
no value on it — and by nothing else. **A Brace joins a hand** and brings no damage into it.

### Damage: a hit per card, one multiplier

**Every card of the turn lands its own hit**, in the attack phase, and each hit is built in four
steps, always in this order *(owner's call, 2026-09-26)*:

```
DUELIST  →  CARD  →  CARD RELICS  →  HAND  →  HAND RELICS
```

1. **DUELIST** — the DMG every hit of the turn is swung at: the duelist's own DMG, plus what the
   turn's **duelist upgrades** raise it by — the riders on the cards kept back, a rung relic's
   `add-hand-dmg`, the jars, the purse.
2. **CARD** — that DMG times the card's own multiplier, then the card's **card upgrades**: its own
   played riders, a flat `+N` first and then a percentage. They touch this card's hit and nothing
   else.
3. **CARD RELICS** — every relic that prices this card, in worn order.
4. **HAND** — the hand's multiplier.
5. **HAND RELICS** — every relic that multiplies the hand, `scale-hand-damage`: Paired Rings,
   Triplicate Rings and the rest of the family, and Dual Wield. They reach every hit of a hand they
   name, as the hand does.

then the attacker's weight and the target's vulnerability, which are statuses on the two duelists
rather than part of what was built.

**Every damage increase is one of two kinds, and which kind decides where it goes.** A **duelist
upgrade** raises the DMG at step 1 and so reaches every hit of the turn; a **card upgrade** changes
one card at step 2 and reaches that card's hit alone. The hand multiplier reaches every hit too,
like a duelist upgrade, but it is the hand's and it comes after the card; the hand relics
multiply it, and they come last. A rider fires on a card that is
**played** — it does not have to make the hand — and a rider on a card **kept back** is a duelist
upgrade, since that card has no hit of its own to raise.

`combat.blowDMG` is step 1, `combat.Duelist.CardDamage` is steps 2 and 3, and `combat.strike` puts
the five together. **Multiplying is multiplying**, so the order changes a figure only by where a
rounding lands; what it decides is how a hit's working reads, and the hand dialog and the ledger
write the steps in this order, a row per step. **Each hit is worked out and rounded on
its own**, so a turn's total is the sum of rounded hits. A pair of Skewers at DMG 10 is two hits of
`30 × 1` — **60** in all.

**A hit is a landing, not a card.** A card lands once, plus once for every extra landing a relic
buys it — an echo's ladder, a form repeat — so a five-card turn is at least five hits and can be
more, and every one of them is multiplied by the hand.

**Nothing about the attack phase is singular except the hand that names it.** *(Owner's call.)*
Everything else happens to each hit:

- **A duelist upgrade reaches every hit, a card upgrade only its own.** See the five steps above.
- **A shock rolls per hit.** See *Lightning is a roll*.
- **A shield eats one hit**, the heaviest first.
- **A status lands per hit that connects**, off that hit's card.
- **A drain takes its share of each hit that landed.**
- **A growing relic steps on every hit that connects**, and each hit is priced at the figure the
  hits before it left. A miss or a block pays no relic.
- **Hits stop at a death.** A hit after the one that killed is worked out on screen and not thrown.

**Every card the turn played throws a hit, whether or not it made the hand** *(owner's call)*. An
action point spent on an attack buys a swing: `Bash, Jab, Bash` is a Pair the Jab makes no part of,
and the Jab lands anyway, at the Pair's rate.

**A defense throws a hit too, and its card deals nothing.** Its hit is its own card upgrades, then
its relics, times the hand — usually 0 — and it can miss and lands its card's statuses like any other. **A hit of nothing
spends nothing of the target's** *(owner's call)*: no shield eats it and it does not clear the
target's defenses, or a turn of shields would strip an opponent's for free. A hit a card upgrade
lifted above 0 is a hit like any other.

**The floor of the ladder is every attack at the identity**, so dumping action points pays real
damage and building a rung pays more again by exactly the multiplier. **Nothing re-prices
`hands.json` automatically**: `tools/handodds` measures how often a rung can be *built* rather than
what a turn comes to, so a change to what a hit is worth will not show up there.

**`data/hands.json` is the ladder and the figures live there**, one entry per rung with its
multiplier in percent. They are priced off measured reachability rather than authored by feel, so
the file moves whenever `tools/handodds` is re-run against a changed deck — read it rather than a
figure written down here. The *shape* is fixed: the Pair at the identity, then the of-a-kind rungs
on each of `concept`, `form` and `element`, and the concept axis dearest at every rung because a
concept ships the fewest copies.

**The multiplier multiplies the cards, and there is no third term.** A percent applied to a
separate reference swing would buy a *fixed figure rather than a proportion* — at DMG 10 a Four of a
Kind worth +50 whether built from four Jabs dealing 5 each or four Skewers dealing 20 each, which is
2.5× the base in the first case and 0.6× in the second, so the ladder would pay least to the decks
that had climbed furthest. Two things follow:

- **The ladder is tunable from `data/hands.json` alone.** The percent applies to a figure the file's
  reader can see, so an entry means what it says.
- **A hand is worth more on bigger cards, in proportion.** A Pair of Skewers beats a Pair of Jabs by
  exactly the 4× the cards themselves are apart.

**The No Hand pays the identity.** `×1` rather than `×0`, since ×0 would be an attack phase that
dealt nothing; it means *no multiplier*, and every attack in the turn lands its own face damage. It
is in `hands.json` with a name and an ID so the log can say what happened on the turn that happens
most often — **a turn the engine could not name is the one failure this model can have**, which is
why the loader panics without it. The hardest-hitting card is what the rung is **named** after.

**It is fallen back to rather than matched.** Counting is the wrong way to pick it — `matchCountOf`
fills groups largest-count-first and would hand back whichever concept appeared most, not the card
that hits hardest — so `matchHand` skips every one-card hand and `biggestAttack` answers the
question on damage.

**Color buys statuses and no damage.** Each hit lands the status of its card's element, gated on the
attacker wearing that element's relic; basic is not a color and never counts. **An attack that made
no hand still lands its hit, so it still burns** — a card the player watched land and leave nothing
behind would read as a bug rather than as a rule.

**The rung is kept separately.** `Blow.Cards` is the scoring set and `Blow.Rung` is the cards that
made the hand. Two readers ask the narrower question: `RiderScaleInCombo` — the `DMG IF IT SCORES`
upgrade, whose whole subject is membership — and the screen's lift, which says which cards made the
rung.

`[?]` **What a hit per card opens.** A hit is the unit an element can be judged on, so an ice hit
against an ice creature *healing* it rather than harming it is now a rule with somewhere to live.
Not built.

### What the axis costs

- **Counted matching only.** A hand reads the turn as a set, so a Jab between two Bashes does
  not break the pair, and no hand can ask for an *ordered* run of cards.
- **The rung chooses what the swing is multiplied by, never what swings.** Deciding which four of
  five attacks to build around is the turn's decision; deciding which of them are allowed to hit is
  not.
- **A hand cut short still pays out.** Nothing can interrupt the hand — it is read before any hit
  is thrown — so every hit carries its multiplier however the hits before it went.
- **The bottom rung fires constantly and is priced as such.** The Pair is a near-certain hand
  paying the identity, which is a floor rather than a reward. Whether the ladder should start
  higher is **answered by pricing instead** — a near-certain rung pays near the identity, so it
  costs nothing to leave in and it keeps the bottom of the ladder legible.
- **Poker's ranking does not transfer to this deck, and the ladders are priced off measured
  rarity rather than off poker.** Poker's ordering comes from 52 cards, 4 suits and 13 ranks;
  here a concept ships five copies while a color spans eleven cards, and the turn is bounded by
  the action budget rather than by the draw. See *The multipliers come from how often a hand can
  actually be built* for the model and the table.
- **A turn's mismatched attacks sum**, rather than the biggest one landing alone, so a hand is
  worth more the dearer its cards are: at DMG 10 four Skewers are **400** where four Jabs are
  **100** and three Thumps are **30**. **Nothing on the enemy side is tuned against that** — the
  ladder, the ascent curve and the roster are independent, and the ladder is one file.

### The catalog's shape

`data/hands.json` holds one list: the of-a-kind rungs on each of three axes, the merged Pair,
the Elementalist, plus the one No Hand they all fall back to. **A hand
carries a key, an ID, a name, a `match`, `groups` and a `multiplier` in percent** — nothing
else. `groups` naming *distinct values on the hand's own axis* is why `[3,2]` is
a full house and can never be satisfied by five cards sharing one value.

### The pairs are one rung

**There is one Pair**, keyed `pair`, written `"match": "any"`, and read on **concept, form or
element — whichever the turn satisfies**. Three per-axis pairs would be three rungs, three stones
and three relics describing the same two cards, and a player forming a pair does not care which
axis let them.

**It pays 1x**, the identity, which is what the No Hand pays too: a pair is the floor of the ladder
rather than a reward, so what it buys is a *name* for a turn that agreed about something and a peg
for a stone and a relic to hang off. The loader allows a multi-card rung *at* 100 and refuses one
below it — a rung under the identity would pay a player less for building more.

**A pair is certain rather than likely.** A hand of eight over four forms cannot avoid one, so
`tools/handodds` scores it at 100% and the No Hand is essentially unreachable in round one. That
is the right shape for a floor: the ladder starts where every turn already is.

`combat.Hand.Axes` is where the list lives and `combat.Hand.On` is how one reading is taken.

**`Match` is the narrowest axis a merged rung names, and a *formed* hand reports the axis that
satisfied it** — so the hand can always say how it was made even though the rung could have been
made three ways.

**`match` is required and never defaulted.** An entry that landed on the wrong axis by omission
would be a balance change nobody made, so a missing or unknown one is refused at init like any other
malformed record. Two further refusals live beside it: a hand wanting more cards than a turn holds,
and one wanting more groups than its axis has values — a five-group form hand is a rung nobody
could climb and would otherwise fail silently.

**A hand names one axis to count on, not one per group.** A mixed hand — three ice cards *and* a
pair of Thumps — is deliberately not expressible; reopening it is a schema change and should be
argued for here first. **`"match": "any"` is not that door**: a merged rung is the *same* groups
read on one axis at a time, and whichever reading it satisfies it satisfies whole.

### Counting difference, not copies

**One rung counts what a set of cards has in common by being all different** rather than all the
same, and the grammar already half-expressed it: `[1,1,1,1,1]` is five groups of one card, which is
five distinct values on the hand's own axis.

| Rung | Axis | Shape | What it asks for |
|---|---|---|---|
| Elementalist | `element` | `[1,1,1,1,1]` | five cards, all five colors |

**It is the only rung that counts difference, and there is no ladder under it.** A rung asking
for three or four colors out of five scores at or beside the identity, and a rung the curve has
nowhere to put is a rung nobody can aim at.

**The grammar is three axes and nothing else**: `concept`, `form`,
`element`, plus `any` for a rung read on whichever of them the turn satisfies. A hand says what its
cards must *agree* on and there is no way to say what they must differ on, beyond `[1,1,1]` on the
hand's own axis. Anything wanting more than that is a schema change and belongs in the paragraph
above.

**Keys carry their axis and the names are long, for now**.
`concept-two-pair`, `form-two-pair`, `element-two-pair`; on screen, **Card Two Pair**, **Form Two
Pair**, **Elemental Two Pair**. **The Pair is the exception** — it counts on all three, so its key
names none of them. The file's word is `concept` and the player's is *Card*, which is
the one deliberate mismatch — a player has never heard of a concept. The longest name is
`ELEMENTAL THREE OF A KIND!`; it measured 1220 pixels of a 1280-wide screen while the name was
shouted at 124 points, and about 790 since the name settled at one size of 80.
`TestTheWidestHandNameFitsTheScreen` is what fails if a name or a type size grows past the screen.

**Adding a rung is one entry.** There is no reward vocabulary to extend, which is the point of the
narrowing: the only things a new entry can say are which axis it counts on, what shape it wants, and
what that shape is worth.

### The multipliers come from how often a hand can actually be built

**Defenses carry an element and join hands.** Every one of the fifty-five cards is one of the
five colors, defenses included, and the matcher counts them like anything else. A hand is **what
you played, not what you hit with**. Five things follow:

- **The element axis is eleven cards a color**, against fifteen for an attack form and ten for
  defend, so the two ladders are close but not identical and are priced apart.
- **`defend` is a fourth countable form.** Any two defenses are a Pair regardless of concept or
  color, and ten of the fifty-five cards carry it.
- **A defense brings no damage into the hand it joins.** `Card.Damage` is zero for every verb
  that is not an attack, so the multiplier multiplies the attacks that are in there with it — a
  fire Brace beside two fire Bashes turns a Pair into an Elemental Three of a Kind and pays it
  on the two Bashes' damage. That is the whole of what the change buys.
- **A defense's color counts toward the hand and lands its status.** Every card throws a hit,
  and a hit lands its card's statuses — so a fire Brace makes a fire hand and burns.
- **A hand of nothing but shields is real, is scored, and deals nothing** — which is the
  accepted cost, see the decision below the table. It is named and multiplied like any other rung,
  and every hit it throws is worth nothing but its flat bonuses. Leaving it unscored would make the
  shield build the one hand the ladder cannot see, and a relic or an authored card should be able
  to reward it.
- **A hit of nothing spends nothing of the target's** — no shield eaten, no defense cleared —
  because a shield build stripping an opponent's defenses for free is an attack in everything but
  the arithmetic. It still rolls its shock and lands its statuses.

The ladders are **not** the same numbers, and no two of them are. The starting deck is 55 cards —
**5 per concept, 15 per attack form and 10 for defend, 11 per element, 20 at the commonest cost** —
dealt into a hand of eight against a 6 AP, 5-card turn, and that arithmetic is what the ladder is
priced against rather than poker's.

#### Two questions, not one

*Reachable* is two questions wearing one number, and keeping them apart is what stops the ladder
carrying inversions:

- **Dealt** — does the hand hold the cards for the rung at all, whatever they cost.
- **Playable** — dealt *and* affordable inside the round's action points.

They come apart hard on the five-card rungs. A **Form Full House is dealt in 93.3% of hands and
payable in 8.5%**: the shuffle hands it to you constantly and the round cannot pay for it. On most
of the concept axis they are the same number.

**Neither one alone can price the ladder.** Pricing on the deal puts that Form Full House below a
two-card pair, so nobody would ever build it. Pricing on playability treats a rung you are dealt
every hand and can rarely afford as though you had never seen it — when in fact it is the one kind
of hand you can *plan toward*, by holding cards, buying a cheaper copy or wearing a discount.

**So the score is the geometric mean of the two**, `√(dealt × playable)`. It collapses to
playability wherever the budget never bites, and lifts a rung the deal offers often but the round
cannot pay for by exactly half the distance in the log. An arithmetic mean would be dominated by
whichever number happened to be large.

**The curve is then one line: `110 + 168 × log10(100/score)`.** 110 is what a rung scoring 100%
pays — the identity plus a token, where the commonest rungs already sat. 168 is what one factor of
ten in rarity buys, and it is **a fixed constant rather than a curve fitted to the rarest rung in
the sample**, so adding a rarer hand cannot silently reprice every hand below it. It is the number
that puts the rarest rung of the shipped ladder at the 785 it was already tuned to.

`go run ./tools/handodds -price` prints what the curve would charge beside what the file charges,
and marks every row where they differ. **The Pair is the one marked row and it is deliberate**
: the curve would charge 110 for a 100% hand and the file charges 100,
because the Pair is the ladder's floor rather than a reward. Every other rung matches, so any second
mark is a real signal rather than accumulated drift.

From a two-million-hand simulation of round one:

| Rung | Axis | Dealt | Playable | Score | Pays |
|---|---|---|---|---|---|
| Pair | any | 100% | 100% | 100% | 100 |
| Form Three of a Kind | form | 95.7% | 80.0% | 87.5% | 120 |
| Form Two Pair | form | 97.5% | 61.7% | 77.6% | 129 |
| Elemental Three of a Kind | element | 79.7% | 65.3% | 72.2% | 134 |
| Elemental Two Pair | element | 96.9% | 52.8% | 71.5% | 134 |
| Card Two Pair | concept | 57.3% | 26.6% | 39.0% | 179 |
| Form Full House | form | 93.3% | 8.5% | 28.2% | 202 |
| Form Four of a Kind | form | 41.4% | 10.1% | 20.4% | 226 |
| Elemental Full House | element | 76.5% | 4.7% | 18.9% | 232 |
| Card Three of a Kind | concept | 19.8% | 14.5% | 16.9% | 240 |
| Elementalist | element | 38.4% | 4.8% | 13.5% | 256 |
| Elemental Four of a Kind | element | 21.0% | 5.0% | 10.2% | 276 |
| Card Full House | concept | 12.0% | 1.4% | 4.2% | 342 |
| Form Five of a Kind | form | 8.6% | 0.14% | 1.10% | 439 |
| Card Four of a Kind | concept | 1.1% | 0.39% | 0.64% | 479 |
| Elemental Five of a Kind | element | 2.7% | 0.03% | 0.29% | 537 |
| Card Five of a Kind | concept | 0.016% | 0.006% | 0.009% | 787 |

**Every row is measurement.** Nothing in the table is hand-set and nothing is forced to climb
within its ladder: the curve is monotone in the score by construction.

**Poker's shape order does not survive this deck, and the ladder does not pretend it does.**
Poker's ordering comes from thirteen ranks; here the form axis has four values and the element
axis five, so **spreading is harder than stacking**: a Form Two Pair wants two distinct forms and
four cards and is rarer than a Form Three of a Kind, which wants one value and three. Forcing two
pair below three of a kind on every axis would pay the commoner hand more, on both wide axes, at
both the two-pair and the full-house rung.

**What replaced the shape order is containment.** A rung whose groups *dominate* another's on the
same axis contains it and must pay more: `[3]` contains `[2]`, `[3,2]` contains both `[3]` and
`[2,2]`, `[5]` contains `[4]`. `[2,2]` and `[3]` contain neither, so the file is free to price them
in whatever order the measurements say. `TestTheLadderClimbs` checks exactly that and nothing more.

**Nothing measures whether the re-pricing lands.** The curve says what a rung is worth relative to
the others; it has never said whether the whole ladder is worth the right amount, and there is
still no simulation of a duel to ask.

Three things fall out of it and are worth keeping:

- **A certain hand pays the identity.** The Pair is a 100% hand, so it is a floor rather than a
  reward — what it buys is the *sum of both cards*, which is already the whole of the change. It is
  the one rung priced under the curve on purpose; a rung the curve has nowhere to put is a rung
  nobody can aim at, and the honest thing is to charge nothing for it.
- **A card hand pays more than the form hand inside it.** A concept fixes a form, so any set sharing
  a concept also shares a form — if the form rung paid the same, nobody would ever have a reason to
  build the narrower one. `TestACardHandPaysMoreThanTheFormHandInsideIt` holds it.
- **That does not extend to element**. A concept does *not* fix an
  element — a fire Jab and an ice Jab are one concept and two colors — so there is no containment
  between those axes and no reason the card rung must outpay the elemental one. The measurements say
  an Elemental Two Pair is rarer than a Form Two Pair, and it is priced above it.
- **The ladders cross, and that is intended.** A Form Full House pays 202 against a Card Two Pair's
  179 though it counts on the wider axis, because at a 28.2% score it is genuinely the harder hand.
  Rung shape and axis width are both inputs to rarity; neither is the answer on its own.

**The best hand is chosen on its multiplier, never on what it would deal.** The matcher does not
look at damage, and the case that makes that visible is a turn of `Bash + two shields`: any two
defenses share `FormDefend`, so it forms a **Pair on zero damage** and the Bash is announced
inside a hand whose own two cards deal nothing. The Pair paying the identity means the player
loses nothing to the multiplier — what can still cost them is which cards the rung is made of. **Reading the board to avoid that is part of the game** rather than a bug to design out.

**Hand IDs are written in the file, never derived.** There is **one ID per catalog key**, so
reordering the cards cannot renumber a hand a player has already found — which matters the day
profile discovery starts recording them. **They are banded by axis**: 1 for the No Hand, 10 for
the merged Pair, 11–15 concept, 21–25 form, 31–38 element. The gaps are deliberate, so a new rung
lands in its own band without moving anything. **Once a profile records them, they freeze.**

**Straights are dropped rather than invented** — the concepts have no natural order to be
consecutive in, and the grammar has no notion of consecutiveness for one to be written in.

What keeps the top of the ladder rare is the deck and the budget: three Bashes is exactly 6 AP,
a starting fighter's entire budget, and **five Bashes is 10 AP**, reachable only by spending a
whole round on shields. **Five Bashes is dealable** — the deck holds five, one per color — which
is what makes the concept five-of-a-kind the rarest measurable hand in the game rather than a rung
only an essence could reach. Being dealable and being affordable are still two different questions,
which is why the wide five-of-a-kind rungs are the cheapest cards of a form or a color rather than
of a concept.

**A color's cheapest five includes a defense** — fire Jab, Cut, Thump and Brace at 1 AP each
plus a fire Thrust at 2 is **6 AP**, a plain round's whole budget, which is what makes an
elemental five-of-a-kind affordable at all. The Brace pays nothing into the multiplier; what it
does is take the place of a second 2 AP attack, so it is a rung the defenses *open* rather than
one they win. `go run ./tools/handsheet` draws it.

### Requirements

- **Hands are rules and live in `internal/combat`**, matching on the resolved cards. The
  screen must never derive one; that is what makes the written account structurally incapable
  of lying about the round.
- **A `KindHand` event** carrying what fired. *Done.* It carries a `HandID`, the multiplier and
  the list of cards that formed the hand, and the screen looks the name up with `HandByID` — so a
  hand renamed is renamed once.
- **The hand event carries its own card lists, not a span.** A counted hand is not contiguous —
  Two Pair can be two cards, a card that earned nothing, and two more — so the screen raises what
  the engine names and never derives it from a pattern length.
- **`KindChilled` counts as a slot in playback** even though nothing happened, or the
  log runs a row short for the rest of the round.
- **A place to browse hands** — a reference the player can return to. Probably belongs with the
  profile rather than inside a duel. `Hands()` exists for it to read.
- **The attack phase writes the hand's line and a line per hit.** Attack cards write no act of
  their own: a turn of five Bashes is the hand's heading and five hits under it, each carrying its
  arithmetic off the event, so the figure shown is the figure used and the hit's outcome attaches
  to it. **The hand dialog carries the same arithmetic at the size of the screen**; the lines stay
  because they are the record and the dialog is the moment. **What is still not drawn** is a row
  that a chill deleted.

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

**The cap is five permanently, and nothing may ever raise it.** Three reasons:

- **A fixed five is what makes hand concepts possible.** Poker hands exist *because* you always
  hold exactly five; that is what lets "a full house" be a permanent, learnable, nameable thing
  rather than a coincidence of how big your hand happened to get. The named five-card shapes are
  built — a Full House wants five cards in a 3-and-2 shape — and every one of them needs the five
  to be a constant the player can plan against for the life of a run.
  The catalog loader enforces it directly: **a hand asking for more cards than a turn can hold
  is refused at package init.**
- **A growable cap would dilute every shape as it grew.** A Four of a Kind is an all-in commitment
  at a cap of five and routine at a cap of seven. The hands would quietly get cheaper every time
  capacity went up, which is the opposite of a reward for building toward them.
- **It is a method anyway, and should be.** Relics and brands need somewhere to bite for
  everything *else* they do, and a method that reads the duelist costs nothing. This particular
  lever is simply off the table: **no relic, brand or hand raises `MaxActions`.**

A card that wanted to buy capacity could therefore only buy *points*, never slots — anything
reaching for six- and seven-card hands is the dilution above and is refused.

Discounts **can take a card to free**, which is what makes the count bound load-bearing rather
than incidental — and with the cap frozen, a discount relic's ceiling is five free cards rather
than an ever-widening round.

---

## The round limit — a duel is bounded too

**Every fight lasts at most five rounds. A duelist still standing when the fifth ends dies.**

The round above is bounded by cost and by count; this bounds the *duel*. It applies to every fight
in the tower — the two ordinary rooms and the stairway protector alike, one number for all of
them — and it is the reason a duel is a race rather than a siege.

**What it is for.** Without it the correct play against anything dangerous is to stall: raise
shields, hold cards back, and win on attrition against a creature whose deck cannot out-scale a
defended turn. Attrition is the one strategy that gets *stronger* the worse the matchup is, which
is backwards. A clock makes damage the thing every build has to solve, and it prices defense
honestly: a shield buys a round, and rounds are now finite.

**Timing out is a death, not a loss on points.** The duelist's life goes to zero and the run ends
through the same door a killing blow uses — there is no retry, per the roguelike rule. What the
clock takes is announced (`KindTimeUp`) before the fall, so the account of the fight says what
happened rather than showing a duelist dying to nothing.

**The rules own it, not the screen.** `combat.Duelist.RoundLimit` is what the resolver checks, at
the end of the round and after every other way the round could have finished:

- **A duelist who killed their opponent on the final round has beaten the clock.** The check runs
  only on a fight still standing on both sides, so a win on round five is a win.
- **A duelist who died to the final blow died to the blow.** The clock never fires over a body,
  the same rule the burn tick keeps.
- **Zero is no clock at all**, which is what every creature carries and what a bare duelist in a
  test carries. That is deliberately not "the default": a default of five in the rules would put
  every headless caller on a timer it was never written against, and the safe direction for a rule
  nobody asked for is off.

**The number belongs to the run.** `session.Session.RoundLimit` is what a fight is actually on,
seeded from `combat.DefaultRoundLimit` and carried to the fighter by `Equip` — the same seat the
relics and the stones arrive in. It is saved with the run; a save written before the clock existed
resumes onto the default rather than onto no clock.

**A relic may move it, and it moves the fight rather than the run.** `adjust-round-limit` names a
signed number of rounds at `fight-start`, and `combat.RoundLimitFor` sums the worn set's deltas over
the run's own number each time a fighter is put together — so selling the relic hands the rounds
straight back, where a relic writing to the run would leave the whole climb moved. Hermes is the
first record: every card 1 AP cheaper, every fight two rounds shorter.

- **A delta, never a figure.** A relic naming three rounds outright could not be mixed with one
  that buys rounds — whichever was read last would simply win, and which that was would depend on
  nothing the player can see. Deltas sum, so a relic taking two and a relic giving one leave a
  fight one round shorter and both sentences stay true.
- **Summing is what makes worn order irrelevant**, which is the one place a relic verb steps
  outside left-to-right compounding. Addition commutes; a figure would not, and a drawback a second
  relic could cancel by sitting to its right is not a drawback.
- **Never below one.** Zero is no clock at all in the rules, so a stack of drawbacks reaching it
  would take the mechanic off the fight rather than tighten it — the direction
  `session.SetRoundLimit` already clamps. A relic moving the clock by zero is refused at load.
- **A fight already on no clock stays on none.** Creatures and every bare duelist in a test carry a
  zero, and a delta off an unlimited fight is still unlimited.

**The player watches it fill.** A five-cell bar under the tower place, one cell per round spent,
with the last one taking the game's one red as it lights. It is a picture of the round counter and
decides nothing — the clock is checked inside the resolved round, per the rule that presentation
may never change an outcome. The tutorial names it before the first duel, because a timer that
killed without having said so would be the worst kind of hidden rule.

**The cost, stated rather than discovered.** A hard cap turns every fight into a damage check, and
nothing in this repo simulates a duel — so a floor where creature HP has outrun what a run can
build is unwinnable and no test goes red. That is the thing to watch as the tower scales, and it is
an argument for a headless duel simulator rather than against the clock.

---

## Relics

- **Bought after every fight, with vitae.** *(see The shop, below)*
- **Five at once**, until brands expand capacity.
- **The cap is never displayed.** It surfaces naturally when you try to buy a sixth.
- **No relic changes how many cards can be played.** `MaxActions` is frozen at five — see *A
  round is bounded twice*. A relic may make five cards cheaper, never make it six.
- **Relics are the duelist's only**. An enemy wears none; affixes are the
  enemy-side counterpart.

### A relic is written in a grammar

**Every relic is data, in a `When` / `If` / `Then` grammar**, and it is **built**:
`data/relics.json` is written in it, `internal/session` parses it, and `internal/combat/relic.go`
holds the vocabulary and refuses a rule that misuses it. The full vocabulary, the code seat each
moment lands on, and the questions to put to a new relic idea live in
[.claude/skills/relics/SKILL.md](.claude/skills/relics/SKILL.md); this is the argument for the
shape.

**A relic is the only collected thing that is never played.** A card resolves in the turn you
queued it, an essence fires when you pick it, a hand is scored when the attack phase runs — each
already knows *when*. A relic waits, so it says so itself, and that is the third part the card
language does not need.

- **A relic holds a *list* of rules.** Forced by the growing stat relics, which accumulate at one
  moment and apply at another; it generalizes to any relic wanting two.
- **`Then` is a list too**, which is what buys a lightning relic that shocks *and* chills with no
  new vocabulary.
- **Seven moments, and only four are in `internal/combat`.** The other three fire in `session`
  and on the post-battle screen, which is what makes a relic a **run** concept the rules consult
  rather than a combat one. `relics.json` is therefore parsed in `internal/session`, beside the
  essences and for the same reason.
- **Relics fire left to right, in worn order.** A determinism rule, not a preference: multiplicative
  effects are order-sensitive, and worn order is the only order the player can see. **Compounding
  is intended** — two slash relics are ×4.
- **A relic may only bend a rule the game already has.** Banker scales vitae propagation, so
  propagation had to be designed first. This is the test to apply to any new relic.

#### Growing relics step between hits

**A `grow-on-hit` relic steps on every hit that connects**, and each hit is counted at the
accumulator the one before it left. Stepping once after the whole turn would count every card of
the turn at the figure the relic opened with, which is the same relic paying for one card.

- **The order of the queue is therefore a rule**, and it is the one thing to unlearn from any
  reading of a hand as an unordered set. The card that goes first pays for the card behind it, so
  the sort buttons and the hand's drag are not presentation. Paying attention to the order is the
  point.
- **Hits, not cards.** A card an echo or a repeat lands three times steps the relic three times,
  and each of those hits is itself counted at the figure the previous one left — so an echo ladder
  compounds inside itself.
- **The shape is settled per card and the figures are asked per landing.** How many times a card
  lands is fixed when the card is reached; what each landing is worth is not. See
  `combat.LandingShape`.
- **A miss or a block pays nothing.** A hit that never connected leaves the relic where it was, and
  the hits after it are counted at the figure it left.
- **A relic's figure belongs to the hit's arithmetic, never to the card face.** A card says what the *card*
  does and nothing else — `1x DMG` whatever is on the fingers — because a growing relic's
  multiplier depends on where in the turn the card is counted and no printed figure could be
  right in every queue position. The flat relics go the same way: a fire relic doubling every
  fire card would be invisible in the term's figure with nothing accounting for it. So each hit's
  arithmetic is where every relic is read. **Every figure flies out of the card that produced it and
  that card shakes as it lands**, one figure at a time within a hit and every hit at once — the
  card's damage from the played card, each relic's multiplier from its own relic, and an echo's
  extra hit shaking the relic that bought the landing even though it puts no figure on the line.
  The relic's badge steps as the hits are counted. `combat.CardScaleBySeat` is the multiplier per worn
  seat, `combat.LandingSeats` is who bought an extra landing, and `Event.HandRelicScale` /
  `Event.HandLanding` / `Event.HandGrown` carry them.
- **Cost is the exception and stays on the face.** A discount is not order-dependent, and a card
  face disagreeing with the AP bar is the failure that rule has always existed to prevent.

**What the grammar cost, and every item was real work:**

- `Duelist.Relics` was `[ElementCount]bool`, which a form multiplier had no element to be a bit
  under. It is a fixed array of `WornRelic` — a `RelicID` and its accumulator — plus a count,
  which is the shape the defend set already used and the reason a duelist is still comparable.
- `Duelist.Statuses` was indexed by element and is indexed by **status** — see below.
- **Growing relics hold state**, the first relic thing that does, and the first that must be
  **serialized**: an accumulator on `Session`, keyed by `RelicRecord`, which is why the record key
  is the identity rather than an index. **Uncapped, by decision** — a +5 HP relic is +100 by the
  top of the tower and that is the intent. **One numeric effect per growing relic**, so the
  accumulator never has to say which of two it feeds.
- **Nothing measures any of this**, so **a relic's balance is unknown** — say so rather than
  guessing at a multiplier.

### Statuses are their own collection, not a property of an element

**A status is data**, in `statuses.json`: a key, a name, a verb, a badge, one of a closed set of
effect kinds (`damage-over-time`, `lose-actions`, `miss-chance`, `damage-reduction`,
`damage-amplification`), an amount and a duration.

**Fully decoupled — fire does not burn on its own.** A status that came free with its color would
leave that color's relics with nothing to be, and a *second* fire status arriving on a different
relic later is only possible while the first was never inherent to the color.

**`Duelist.Statuses` is indexed by status, not by element**, and its width is `MaxStatuses` — an
array width rather than a design cap, which registration refuses to grow past because a duelist
has to stay comparable. `cards.MaxEffects` is the badge row's width and
`TestTheCardHoldsAsManyEffectsAsThereAreStatuses` is what turns authoring one more status into a
visible layout decision rather than a silently clipped row. The badge lookup is
`screens.statusBadges`, read straight off each record's `Badge`. **`StatusID` is append-only**,
carrying the same hazard `Element` does — with the file, not the enum, deciding the order.

### The relic shapes

**This is not the catalog** — `data/relics.json` is, and `docs/sheets/relicsheet/index.html` is
where it is read, with every record's art, price, authored line and resolved rules side by side.
Nor are these rarities: see *Rarity is the price dial*, below.

**What this table is** is one row per *shape*, and each row is the argument for why that shape
exists. A new relic that matches a row here is a sibling and should be priced like one; a new
relic that matches none of them is a new shape and needs its own argument.

| Shape | Moment | Does |
|---|---|---|
| **Burning / Chilling / Shocking / Weighted / Weakening** | `attack-lands` | the five colors' status relics — one per color, and the thing that arms an element at all |
| **Fire / Ice / Lightning / Earth / Arcane** | `card-damage` | doubles every card of that color — *element* multipliers, where Keen/Heavy/Needle are form ones |
| **Storm** | `attack-lands` | lightning shocks *and* chills |
| **Keen / Heavy / Needle** | `card-damage` | doubles **every** slash / crush / stab card in the turn |
| **Striker** | `card-damage` | doubles every Bash — a concept relic, 5 cards where a form covers 15, and priced accordingly |
| **Banker** | `fight-won` | a second +1 vitae per 5 held, on top of propagation |
| **Soul Taker** | `prizes-dealt` | the vitae prize card pays +10 rather than +5. A **flat** +5, not a scaling |
| **Hungry** | `prizes-dealt` | two post-battle choices instead of one |
| **stat relics** | `fight-start` | +10 DMG, +25 HP — and growing variants that gain per fight |
| **Momentum** | `card-damage` + `turn-taken` | every card gains +0.2x DMG per turn with no defend card in it; a defend card wipes the streak |
| **Enflamed / Frostbitten / Lithium / Granite / Unravelled** | `card-damage` + `attack-lands` | their color gains +0.1x DMG per landed hit of that color, and keeps it while worn |
| **Echo** | `blow-formed` | the blow's first attack card lands three times: full, 2/3, 1/3 |
| **Flurry / Rend / Aftershock** | `blow-formed` | every stab / slash / crush card lands **twice**, both at full DMG |
| **Atrophy** | `deck-built` | every 3 AP attack is dealt as its 2 AP version |
| **Onslaught** | `card-cost` + `fight-start` | every card 1 AP cheaper, and a quarter off your life — the drawback shape |
| **Warm / Cold / Static / Dirty / Eerie** | `card-cost` | every card of that color costs 1 AP less — one per color |
| **flip x20** | `card-drawn` | recolors a card of one color as another **as it is drawn** — one for each ordered pair; see below |

**A concept relic and a form relic are not the same object** and must not be priced as one.
Striker covers 5 cards, Keen covers 15.

**A color is worth about a dozen relics, not one**, and that is the number to expect when one is
proposed. Four are the color's own seats in families that already exist — the damage relic, the
status relic, the discount and the growing one — and the rest are the **flip cross-product**,
which is quadratic in the number of colors. A sixth color would bring fourteen relics, ten of them
flips.

**Weakening is the strongest of the five status relics and is priced the same as the others.** That
is deliberate rather than unexamined: WEAKENED doubles everything the target takes for two rounds,
where a weight blunts a quarter and a chill takes one card, so its tier is the thing to move first
if the arcane build turns out to dominate. Nothing in the repo measures what a relic does to a duel,
so the price is judgment — see the relics skill.

### Momentum — a streak that belongs to the duel

**Every card gains +0.2x DMG for each turn played without a defend card, and a defend card wipes
it.**
Uncommon. It scales the *duelist* rather than a color or a form: the `card-damage` rule carries no
predicate at all, so the streak is worth the same on every card in the hand.

- **It is written as two positive rules and no negation.** One grows on every turn, one resets
  on a turn holding a defend card, and **growth is applied before resets** — so a defending turn
  nets zero rather than depending on which rule the file lists first. The grammar has no `not`
  and this is the shape that means it does not need one.
- **`turn-taken` is a new moment**, the first that is about a *turn* rather than a card, a hit or a
  fight. Its predicate is matched against the turn as a whole: the rule fires when any card of the
  turn matches it.
- **An empty turn is still a turn taken**, so a duelist chilled out of their whole turn keeps
  building. The streak is about not *defending*, not about swinging.
- **A duelist who falls mid-turn never reaches it**, since `playTurn` returns early on a death — a
  streak is a fact about turns taken and a corpse takes none.
- **The streak does not survive the fight**, and that needed a rule: `combat.KeepsGrowth` reports
  false for any relic holding a `reset-growth`, and `Session.AbsorbGrowth` skips it. Otherwise one
  good duel would bank a permanent bonus that a single defend card had once wiped.
- **What it prices is taking a hit**, which is the question it puts to the player on every turn:
  swing into the next blow, or spend the turn defending and lose the streak. Whether 0.2x a turn
  is enough to make a player eat an attack is unmeasured, like every other relic.

### The Enflamed family — growth inside a fight

**Enflamed (fire), Frostbitten (ice), Lithium (lightning), Granite (earth)**: their color gains
**+0.1x DMG every time an attack of that color lands**, and keeps it for as long as the relic is
worn. Uncommon.

**They are the accumulator that moves during a fight.** Heart and the growing stat relics step
once per win, at `fight-won` — the `grow-on-win` verb; these step at `attack-lands`, so the
second fire attack of a duel is already stronger than the first. That needed a second verb —
`grow-on-hit` — because a verb belongs to exactly one moment, and it needed a way home: combat
grows the *duelist's* copy of the accumulator, and `Session.AbsorbGrowth` reads it back on the
win, before the screen throws that duelist away.

- **Once per hit.** Two fire cards in a hand are two steps, and a fire card that Echo lands three
  times is three — it counts **hits** rather than cards. **That is the combination it exists for**:
  the relics that multiply landings and the relics that grow per hit are meant to compound into a
  build, not to politely ignore each other. Echo plus Enflamed is +0.3x off one card.
- **A hit is paid for after it lands, never during**. The first fire hit of a fight lands at the
  relic's opening strength and the second at the stepped one; a relic that strengthened the hit
  that grew it would mean the first attack of a fight already wearing its own bonus.
- **The growth is linear, and deliberately.** The step reads the effect's raw `Amount`, never
  `Amount + Grown` — a growth that grew would compound, and no growing relic in the game does.
- **A lost fight forfeits what it earned**, which needs no rule of its own: a defeat ends the run.
  **Selling forfeits it too**, by the shop's existing rule.
- **It is relic state that changes mid-fight**, which is the thing a mid-fight save would have to
  write down. A run is only ever snapshotted between phases, so nothing writes it.
- **Uncapped, like every other accumulator.** +0.1x a hit across a long fight is a big number by
  the top of the tower, and nothing measures it.

### Atrophy, and the ladder as a relic

**Every 3 AP attack is dealt as its 2 AP version**: Skewer becomes Thrust, Cleave becomes Slash,
Smash becomes Bash.

**It is the flip's shape applied to the other axis, at the other moment.** A flip changes a
card's color as that card is *drawn*; Atrophy changes its *concept* as the fight's draw pile is
built, one rung down the same form's ladder. The two moments matter for how the deck panel shows
a card and for nothing else in play — see the flip relics below. Everything downstream — cost,
damage, the hand it forms, the card face — follows because the card genuinely is a Thrust.

- **What the player buys is a turn with more cards in it.** Three Skewers cost 9 AP and do not
  fit a 6 AP turn; three Thrusts cost 6 and do. It trades damage per card for cards per turn,
  which is a hand-ladder decision rather than a damage one — a Three of a Kind of Thrusts
  against one Skewer and a Jab.
- **`combat.Neighbor` already existed**, built for essences, so the ladder is still a
  consequence of `duelist_cards.json` rather than a table written twice. A card with no rung
  below it is left alone.
- **`Tier` is a new predicate and it reads the *declared* cost.** A discount relic cannot move a
  card out of Atrophy's reach, which would otherwise make two worn relics switch each other off
  in an order nobody chose.
- **Two demoting relics do not chain.** `DemoteConcept` reads the card the run owns and takes the
  deepest single step, exactly as flips read the original element. A relic wanting two rungs says
  `Amount: 2`.
- **Nothing measures it**, and this one is the most likely of the new relics to be badly priced:
  `tools/handodds` measures which hands a deck can reach, and Atrophy changes that deck.

### Echo, and a hit per landing

**The Echo Ring makes the turn's first attack card land three times — full DMG, two thirds, one
third.** Uncommon.

**Each landing is a hit of its own.** The lead card is seated again behind itself at a smaller
figure, twice, and each of those is a hit with everything a hit carries: the hand's multiplier,
every flat bonus, its own shock roll, its own statuses, a shield to eat it. The turn reads as
*seven hits off five cards, the first card three times*.

- **The echo never reaches the matcher.** The hand is read before the landings are laid out, so
  an echoed Bash does not turn a Pair into Three of a Kind — which is also what stops one relic
  rewriting the hand ladder.
- **The multiplier multiplies every echo.** Echo is worth about two thirds of the lead card, plus
  two more servings of every flat bonus, all times the hand — strongest in a big hand, which is the
  opposite of a relic that rescues a bad one.
- **The player watches it happen.** An echoed card's hits are three lines of arithmetic stacked
  under it, each flying its own figure out. *(Owner's call.)*
- **`blow-formed` is the moment that sees the turn's attacks as a set**, and `echo-attack` its verb.
  `MaxEchoLandings` is 5 — a width on `Event`'s hand arrays, which have to stay fixed for an Event
  to be comparable.
- **Two echo relics add landings rather than multiplying**: three and three is five, not nine.
- **Nothing measures it**, like every other relic.

**Flurry, Rend and Aftershock repeat a whole form**: every stab / slash
/ crush card of the turn lands **twice, both at full damage**, uncommon. `repeat-card` is the second
verb at this moment and it is the one that does *not* diminish — an echo is a card ringing on, a
repeat is the card played again.

- **They deal what Keen, Heavy and Needle deal on the card, and more once anything flat is in
  play.** Two full-strength landings and one doubled landing are the same card term; what the
  repeat buys is **two hits instead of one**, and a hit is now what pays — a flat bonus joins each,
  a status lands on each, a drain takes a share of each and a growing relic steps on each. What
  that makes them worth against the commons beside them is unmeasured.
- **`Lead` is a new predicate, and the only one that is not a fact about the card.** It is what
  lets one pair of verbs cover both scopes: Echo says `{"Lead": true}`, a repeat says `{"Form":
  "crush"}`. A rule setting `Lead` at any other moment is refused at load, since no other moment
  knows which card leads.
- **Repeats resolve before echoes** when a card has both, because an echo of a repeated card would
  be an echo of something that already happened twice.
- **The event's term arrays are 25 wide** — every card of a legal turn, each landing up to
  `MaxEchoLandings` times. A repeat matches on form, so five crush cards is ten hits, where an echo
  only ever touches one card.

### The discount relics — one per color

**Warm, Cold, Static, Dirty and Eerie**: every card of one color costs **1 AP less**, at
`card-cost`, one relic per color. Each is named for the *color it warms* rather than for the
discount, which is what lets the family cover the whole set without repeating a word.

**They are the third thing a color relic can be**, after the damage doubler and the status
relic, and the one that changes what a turn can hold rather than what it does: a 6 AP budget
buying four cheap cards instead of three is a different hand ladder, not a bigger number.
**Nothing measures that**, so what a color's discount is worth against a color's doubling is
unknown, and reads as the bigger of the two.

### The color relics

`data/relics.json` holds them, each as one `attack-lands` rule matching one color and applying one
status. **There is no special case for them in the engine** — they are the plainest
thing the grammar can say, which is what the grammar was checked against. One relic is one element,
so wearing one and swinging a hand of all four colors lands one status and nothing else — which is
what makes the second and third worth buying.

| Relic | Element | What wearing it does |
|---|---|---|
| Burning Ring | fire | your fire attacks BURN: a share of your DMG at the end of each round |
| Chilling Ring | ice | your ice attacks CHILL: one card off the front of each of their turns |
| Shocking Ring | lightning | your lightning attacks SHOCK: a chance their attack misses |
| Weighted Ring | earth | your earth attacks WEIGH: they deal less damage |
| Weakening Ring | arcane | your arcane attacks WEAKEN: they take more damage from everything |

The figures are `data/statuses.json`'s, not this table's.

### The flip relics — one for every ordered pair

**One relic for every ordered pair of colors**, each `card-drawn` / one color in / another color
out. **It is a cross-product, so it grows quadratically**: five colors is twenty relics and a
sixth would be thirty. That is the cost line to read before proposing one. The names are
thematic rather than mechanical — "Permafrost" says earth into ice without saying either word —
which is a deliberate cost: the *card* has to be read to know what it does, and the tooltip is
what says it.

| dealt as → | from fire | from ice | from lightning | from earth | from arcane |
|---|---|---|---|---|---|
| **fire** | — | Meltdown | Firestorm | Magma | Burning Orb |
| **ice** | Frostbite | — | Frozen Lightning | Permafrost | Frozen Orb |
| **lightning** | Heat Lightning | Thundersnow | — | Dust Storm | Charged Orb |
| **earth** | Obsidian | Glacier | Fulgurite | — | Stone Orb |
| **arcane** | Burning Mana | Frozen Mana | Electrified Mana | Enchanted Earth | — |

**The arcane names follow a convention the other twelve do not.** Everything *out of* arcane is
an **Orb** and everything *into* it is a **Mana**, each
qualified by the color at the other end of the flip — Burning Orb is arcane dealt as fire, Burning
Mana is fire dealt as arcane. Enchanted Earth is the one that breaks the second half of the
pattern, because "Earthen Mana" says the direction backwards. That half-convention is deliberate:
sixteen thematic one-off names is more than a player can hold, and a name that says which way the
flip runs is worth more than another eight inventions.

**They fire as a card is drawn, not as the deck is built**. Every one of
them is worded "every X card is dealt as a Y card", and the dealing is the draw. The cards a fight
plays are identical either way — a flip is unconditional over a color, so recoloring the whole pile
once and recoloring each card on its way out produce the same hand — so what this buys is not an
outcome but a **place**: the draw pile holds the deck the run owns, and the alteration is something
that happens to a card, at a moment, on its way to the hand. That is the shape the next kind of
alteration will need, and it is what the deck panel's ALTERATIONS toggle is a picture of.

**A drawn card does not remember what it was.** It carries the color it became; a `card-damage`
relic keyed on ice fires on a card that is ice *now*, and never on one that was ice before the
flip. That is what stops an alteration turning every later rule into a question about history.
What the original is still reachable from is the card's **identity** — every card a run owns
carries an ID, so the deck panel can show either face of a card wherever it is sitting. **No
rule may read that ID**; it is a handle for the screens.

- **A flip is what makes a color relic worth wearing**, which is the whole point of the pair:
  Fire Relic doubles fire cards and there are only so many, so Frostbite-and-friends is how a
  deck is bent toward the color a run has bought into. It is also how the *status* relics get
  fed.
- **Flips compose, and the cascade is the point**. `combat.FlipSteps` is
  the walk: each worn relic, in worn order, reads **what the flip before it left behind**. Frozen
  Lightning (lightning→ice) and Glacier (ice→earth) worn in that order deal a lightning card as
  earth, through ice, and a run wearing both holds no lightning and no ice at all.
  **Funnelling a deck is the intent rather than the hazard**: two relics that each claim to touch
  one color can walk a whole deck into one, and a build that takes two uncommons and the right
  worn order is a build worth having.
  **What a player reads to keep track of it is the deck panel's alterations view**, which is the
  answer to "so what am I actually holding", and the deal on the combat screen, which plays a beat
  per ring so the cascade is watched rather than deduced — see `screens/combat_deal.go`.
  **Worn order is load-bearing and the row cannot be reordered.** See `TestFlipsCompose` and
  `TestFlipStepsNameEveryRingThatTouchedTheCard`.
- **A card may not take the cascade twice.** It chains *within* one draw and must not chain across
  two: a card that has been through the hand and the discard is wearing a color a relic made, so
  the draw pile holds cards as the run owns them and the discard is restored on its way back in.
  `screens.restoreToDeck` is what pays for that, and
  `TestARedrawnCardDoesNotTakeTheCascadeTwice` is the tripwire.
- **Two flips naming the same source is still last-worn-wins**. Frostbite and Heat Lightning
  both claim fire; the later relic in the row takes it, by the same rule that orders every other
  multiplicative effect. Under the cascade that is the *same* rule read one step at a time — the
  first ring recolors the card and the second is then looking at a card of a different color, so
  "last wins" and "each reads the one before" only differ when the two name the same source.
  Nothing warns the player and there is no way to reorder the row; both accepted.
- **The flips are a large share of the catalog and the dilution is accepted.** A common relic's
  ten tickets compete against every other common, flips included, so a family this size makes any
  one relic rarer on the shelf. If the shelf needs thinning the lever is a weight or a tier, never
  a price.

**Every color is two relics.** **Fire, Ice, Lightning, Earth and Arcane** are `card-damage`
doublers on their color — *element* multipliers, where Keen, Heavy and Needle multiply a form —
and **Burning, Chilling, Shocking, Weighted, Weakening** are the status relics beside them, a tier
dearer. So a color offers cheap damage or a dearer, rarer status, and every record key matches the
element name the rules use.

**A burn is priced to be worth roughly a whole extra attack** over its two rounds, which is what
the uncommon tier is meant to buy. Nothing measures it, so that is a judgment.

**The relic is read off the attacker, never the victim.** Your fire relic makes *your* fire attacks
burn; it does nothing about fire aimed at you. The alternative would make a relic a liability and
buying one a decision with a wrong answer.

**A run opens wearing nothing.** `session.StartingRelics` is the seat for putting one on without
playing to a shop and ships empty. The worn set belongs to the run rather than to the combat
screen, which is what makes a bought relic survive a fight.

**What that costs, stated rather than discovered:** a run holds 5 vitae and a base relic is 3, so
**the bare opening lasts exactly one fight** — the first shop can already afford a color, and the
first duel is the only one fought with every element inert. That is a much shorter gap than the
first pricing draft produced, and it is the deliberate consequence of a base relic being cheap.

### The shop

**Three relics on a shelf after every fight, and the row you are wearing under them.** Both rows
are relic cards and both are clicked; the difference is which way the vitae moves.
`internal/screens/shop.go` draws it, `internal/session/shop.go` holds the rules, and neither
knows what comes after the shop — `session.PhaseShop` is a station of the run loop and
`advanceRun` is what leaves.

**What a visit puts up, in panes:**

| Pane | Holds | Rerolls |
|---|---|---|
| relics | three off the catalog, drawn on rarity tickets without replacement | yes |
| sealed packs | **two of the three** — a bag of rocks, a vial of essence, a sack of runes | yes |
| potions | the whole of `data/potions.json`, the same vessels every visit | no — a reroll would offer what is already offered |
| brand | one seat, **dim and unclickable**, because §Brands is unbuilt | no |

- **Two packs out of three is what makes the pack pane a question.** All three on the shelf every
  visit asks nothing: the only decision is what the purse can cover. Which two you meet is a roll
  on `seeds.PackOffer`, and all three catalogs stay reachable across a run.
- **A reroll is per pane and it escalates** — 2 vitae, then 4, then 8, doubling within a visit and
  starting again at the next shop. The relics and the packs each have their own button and their
  own count, so pressing one does not make the other dearer.
- **A reroll advances the visit's cursor rather than seeding a second stream.** The scene holds
  its `*rand.Rand`s from `Init` and every deal draws from them, which is what keeps a replayed run
  exact however many times the button is pressed.
- **A potion is the one thing in the shop that changes the duelist.** It is drunk on the spot and
  moves one of the three figures a fighter is made of — `internal/session/potion.go` owns what
  each one does and refuses a record the rules cannot apply. **One click and no confirm**, the
  shelf's rule rather than the worn row's: the price is on the card, the purse cannot go into
  debt, and what it does is visible on the duelist card immediately.

- **A relic declares a rarity, and the rarity is the price.** `relics.json` names one of three
  tiers and `data.Rarity` turns it into both what the shop charges and how often the shelf offers
  it:

  | Rarity | Price | Sells for | Draw weight |
  |---|---|---|---|
  | common | 3 | 1 | 10 |
  | uncommon | 5 | 2 | 4 |
  | rare | 7 | 3 | 1 |

  **A common relic is 3, and that is the base everything else is read against** — the plainest
  thing the grammar can say, priced at what a first shop can afford.
- **Three tiers rather than a number per relic.** A per-relic price can only be judged one relic
  at a time and drifts; a tier is read against the whole catalog at a glance, and rebalancing a
  relic is moving it rather than inventing a figure. **What that costs, said out loud:** two relics
  in the same tier cost the same even when one is plainly stronger — the answer to that is which
  tier it belongs in, not a fourth tier.
- **Scarcity and price are deliberately different curves.** A rare relic is a tenth as likely to
  appear as a common one but only a bit over twice the price. What makes it rare is that a run
  mostly does not see it; a price tracking the odds would make it unbuyable on the one visit it
  turns up.
- **Tiers are assigned by hand**, per record, and the relic sheet is where the spread is read.
- **That is a full relic or two a fight against an income of roughly 5–10**, so **the purse stops
  binding once the five fingers are full** — around the fourth or fifth fight, after which vitae
  has nothing to buy but swaps. What answers that is something else to spend on rather than
  dearer relics: the sealed goods, the stones and the potions below.
- **Nothing measures whether any of those numbers is right.** Nothing in the repo measures what
  a relic does to a duel, so what a doubling of every slash card is worth in vitae is a
  judgment. Recorded as a judgment rather than dressed up as a derivation.
- **Selling pays the tier's own figure — 1, 2 or 3**, written down rather than derived from the
  price by arithmetic that has to be argued with. The round trip loses, and loses more the dearer
  the relic — a shelf you could try on for free
  would be a rerolling of your hand every visit rather than a decision.
- **Selling is the only way a relic comes off, and it is how the sixth relic is bought.** A
  purchase at five worn is refused rather than swapped, so trading is two decisions with a price
  between them — never one click that quietly throws a worn relic away.
- **A sold relic's accumulator resets to zero.** `Session.grown` is keyed by record precisely so a
  relic taken off and put back on is the *same relic*; the decision is that it is not the same
  *number*. The growth is what wearing it through fights paid for, so selling forfeits it. It is
  what stops a Heart Ring being parked in the shop between fights.
- **What is already worn is off the shelf**, rather than shown and refused: a seat spent saying
  nothing.
- **Selling out of the middle of the row changes the firing order**, since relics fire left to
  right and a re-bought one goes on at the right-hand end. That is a real cost of letting a
  relic come off, and **there is no re-ordering control**: the one thing a player cannot choose
  is the order two relics apply in.
- **The shelf is its own random stream** (`seeds.ShopStock`), per fight, so a defeat and a retry
  walk into the same shop exactly as they meet the same opponent. **It is three weighted draws
  without replacement**, rather than a shuffle: each seat draws on rarity tickets and the drawn
  relic leaves the pool, so the shelf never offers the same relic twice.

### The sealed packs: a bag of rocks, a vial of essence, a sack of runes

**All three take the same shape**: **5 vitae**, **four of something inside**, and the player keeps
**exactly one of the four** — the other three are gone. Each draws on its own per-fight stream
(`seeds.BagStock`, `seeds.VialStock`, `seeds.SackStock`), and a visit offers two of the three.

- **What is bought is the choice, not the thing.** A relic is read and then paid for; a good is paid
  for and then read. That is the whole design, and it is why neither card names its contents: the
  face says the shape of the offer ("4 stones, keep 1") and nothing about which four.
- **A bag holds four stones; a vial holds four essences.** See the stones section below. The
  vial is the reward screen's mechanic bought rather than won — pick one of four, then pick the
  card it eats — and what five vitae buys over the free offer of two is twice the choice, at the
  shop rather than at the end of a fight.
- **One of whichever two are offered, per visit, restocked next fight.** It bounds what a rich
  run does in one stop and keeps the shop a short offer rather than a vending machine, which is
  the argument the three-ring shelf is already under.
- **They are the answer to "vitae has nothing to buy but swaps"**, which the shop section above
  records as the thing it wanted next. A run with five fingers full now has somewhere for its purse
  to go that is not a relic it will sell back at a loss.
- **Each draws from its own seeded stream** — `seeds.BagStock` and `seeds.VialStock`, both per
  fight. The vial's is deliberately not the reward screen's `EssenceOffer`: sharing would make
  the shop's four a function of which two had just been offered free, so buying the vial could
  guarantee — or rule out — the pair the player had turned down. A rule nobody designed,
  arriving out of an implementation detail.
- **The way out without a card is SKIP, and there is no close button.** Every other modal in the
  game is a look at something; this one stands between a purchase and what it bought, so leaving
  forfeits the vitae. That is the player's call to make *(owner's call, 2026-09-26)*, and it is a
  labelled button at the bottom of the screen rather than the red X that means "close" everywhere
  else. Every card is an exit too.
- **The row is five seats rather than two rows, and the screen decided that.** A relic is worn and a
  good is opened, so two rows would have read better — but the shop is 960 tall with a build band,
  two sentences of narration and the Leave button at 88%, and a card is 224. There is room for one
  row of cards, not two.

### Rarity is the price dial, and it is the only one

**`docs/sheets/relicsheet/index.html` is the catalog** — art, price, authored text and resolved
rules for every relic, grouped by family — and it is the only place that can be current. A table
of rows in a file loaded every session is a cost paid forever and a figure that rots silently.

**The families, which is the map a new relic is placed on.** Each is a cell in the coverage grid
that `.claude/skills/relics/coverage.py` reports; an empty cell is a relic waiting to be written,
and a proposal landing in a full one is a sibling.

| Family | Tier | What it is |
|---|---|---|
| **concept relics** | common | one per attack card, `scale-damage 200` — Striker's shape, five cards wide |
| **form cost relics** | rare | the form counterparts of Warm's color family |
| **form status / growth** | uncommon | a status relic and a growing relic per attack form |
| **tier relics** | rare / uncommon | demote a whole tier, or pay for holding one |
| **rung relics, flat** | common | one per rung, `add-hand-dmg`, the bonus derived from the rung's multiplier |
| **rung relics, multiplying** | uncommon / rare | `scale-hand-damage` on one rung; **the top two rungs are rare**, because 4x on a Four of a Kind makes every hit twentyfold |
| **double-status** | rare | one per unordered status pair, triggered by an element holding one of the two |
| **element repeats** | uncommon | the color half of Flurry / Rend / Aftershock |
| **held-card relics** | common | DMG per matching card **kept back**, one per color and one per form |

**Four tier rules, and the reasoning is worth more than the assignments:**

- **Every card-cost reducer is rare.** A discount is worth a fraction of a turn every turn,
  forever; nothing else at common compounds like that.
- **Every flip relic is uncommon.** A flip is what makes a mono-color build reachable at all, and
  the color payoffs it feeds are uncommon already, so a common flip undersells itself.
- **A demotion is a discount written the other way round**, and belongs in the cost family's tier.
  The narrow version being cheaper than the wide one is the inconsistency to check for.
- **Flat, self-capping and unable to compound is what common means.** A relic with a dead case
  the player cannot steer away from — being attacked and having to defend — is commoner still
  than one whose dead case is a build they chose.

**The weights invert what "adding a common" means, and this is the thing to know before adding
another relic.** At 10 / 4 / 1 tickets, **anything added at common devalues every rare in the
game**, because it grows the denominator every tier's share is taken over. So the number that
matters when a batch lands is each tier's *share of a shelf draw*, not how many records it holds
— see the header of the relic sheet, which prints the three shares to a tenth of a percent.

### A rung relic is a second multiplier, never a bigger hand

`scale-hand-damage` scales **every hit**, after the ladder's own multiplier — the HAND RELICS
step.
`Event.Multiplier` stays the rung's figure and a relic may not touch it.

**Folding the two together would say the *hand* changed** — a Pair under Pairing reading as
2.3x — when what changed is that the player is wearing a relic. The banner, the hand row and the
hits all show the rung actually built, and the relic's figure is drawn as its own factor in every
hit, in the pane's pink, flying out of the relic that paid.

### A flat rung relic is base damage, not a term

`add-hand-dmg` raises the **DMG the hand is swung at**, for the length of one turn. A duelist on
14 wearing the Twinned Ring swings a Pair at 16, so every card in the hand grows by its own
multiplier — a 1x card by 2, a 0.5x card by 1 — and the raise is inside every hit before the ladder
multiplies anything.

**It is not a flat term added after the cards are counted.** A flat term is worth the same 2
whether the Pair is two Jabs or two Skewers; a relic that raises DMG is worth more to a hand that
is worth more, which is the relationship every other damage relic in the catalog has.

**It is therefore not drawn as a term.** `Event.HandBonus` is still on the event, to be *said*
rather than added: the ledger writes `Twinned Ring (Pair) +2 DMG` above the hits it raised, and
the hand dialog shakes the relic as the first DMG figure flies past it. A screen that also wrote it
as a term would print a hit over its own total — and a relic folded silently into figures the
game already shows is a relic the player cannot tell apart from a better hand.

So a rung relic can be written either as base damage the duelist gained or as a multiplier laid over
the hand. The two are `add-hand-dmg` and `scale-hand-damage`, and neither one touches
`Event.Multiplier`.

**It is the one-turn counterpart of `add-dmg`, and the names are `dmg` on both for that reason.**
`add-dmg` fires at `fight-start`, is unconditional, and is added to the duelist in
`session.Equip` for the whole duel — Might's +10 is on every card of every turn. `add-hand-dmg`
raises the same stat for one turn and only when the rung it names was satisfied. They stack
additively, and the strictly stronger shape per point is the unconditional one: a rung relic has to
be earned before it pays, which is what lets its figure be bigger at the same rarity.

### A relic reads what the blow *satisfied*, not what the ladder named it

**The ladder names a blow after one rung — the best-paying one — and a relic is not asked that
question.** A blow carries every rung its cards satisfy (`combat.Blow.Satisfied`) and a rule
naming a rung fires if that rung is in the set.

**What that prevents:** four Cuts are a Card Four of a Kind *and* a Form Four of a Kind, and the
ladder pays the better of the two — so the blow is named on the concept axis, and a Form Four of
a Kind relic would sit still through a turn that plainly was one. A player who does the harder
thing must not switch off the easier thing's relic.

**The ladder is therefore cumulative downward as well as sideways**, and that half is deliberate
rather than a side effect: four Cuts also satisfy both three-of-a-kinds and the Pair, so every
one of those relics pays. The consequence to hold in view is that **the Pair family
is now near-unconditional** — a Pair relic pays on essentially every multi-card hand — while still
being priced as a conditional one. Nothing measures that, on the same terms as every other price in
the shop section.

**The No Hand is not a rung a bigger hand satisfies.** It is the fallback for a turn that formed
nothing, matched by which attack hits hardest rather than by counting, so it is in the set only when
it *is* the blow. A No Hand relic stays a relic about turns that built nothing.

**Only `Hand` moves damage; the satisfied set moves relics.** `Event.Multiplier`, the banner and the
hand row all still show the one rung the blow was paid as — a hand that quietly listed five rungs on
screen would be a worse reading of the same turn.

**A sentence covering several rungs is one rule, never one rule each.** `"Hands": [...]` in
`relics.json` names a set and fires once. The four multiplier rings — Triplicate, Fulsome,
Quartered, Perfected — were written as three rules apiece, one per axis, which under this change
fired once per axis the blow satisfied: Quartered Rings paid 4x twice on four identical cards
and dealt 16x, against its own printed "Every Four of a Kind deals 4x DMG." `Hand` and `Hands`
are the same predicate and a record setting both is refused at load.

### The defensive half of the game has one relic, and no verb can reach it

**Nothing in the vocabulary names a shield.** `CardDamage` returns zero immediately for a card
that deals none, so a `card-damage` rule on a defend concept is a record that can never fire —
which is why the defend concepts have no concept relic, and why **Braced** reaches shields
sideways, through cost, rather than head on.

`[?]` It is recorded as a **gap rather than a decision**: almost every relic in the catalog is
about attacking, in a game whose one defensive mechanic — a shield eating a whole hit — is among
its strongest. Filling it means a verb that raises, keeps or spends a shield, and that has not
been designed.

## Stones — altering the hand ladder

**An essence alters a card; a stone alters a hand.** One stone raises **every rung of one shape**
— a Three of a Kind counted on the card, on the form and on the element all at once — and each of
them by **a tenth of the figure `hands.json` writes for that rung**, for the rest of the run.

- **A stone buys a hand, not one reading of it.** A Card Three of a Kind, a Form Three of a Kind and
  an Elemental Three of a Kind are one idea read on three axes, and a player choosing a rock is
  choosing to lean on Three of a Kind rather than on one axis of it. So the stone names a *shape*
  and every axis' rung of that shape moves together.
- **The multipliers stay apart.** Each rung still pays its own catalog figure and moves by a tenth
  of *that*, so one Jasper adds 24 to the Card Three of a Kind, 12 to the Form and 13 to the
  Elemental — the ladder's pricing of how hard each axis is to land survives the stone.
- **A shape is the rung's `groups` with the axis left out** — `[3]`, `[3, 2]`, `[4]` — and a stone
  writes its `Groups` the same way. `combat.Hand.Shape` derives it rather than a field declaring
  it, so a rung cannot be filed under a shape its groups disagree with. The Pair is already one
  rung on every axis, and the No Hand and the Elementalist have no siblings, so each of those is a
  shape of one rung.
- **There is a stone for every shape, and one only**, and `data/stones.json` is refused at load if
  one is missing or doubled — a shape with no stone is a set of rungs that can never be raised,
  and nothing would fail.
- **The count is still kept per rung.** A stone lands on each rung of its shape, so siblings always
  carry the same level; keeping them apart is what leaves the resolver, the hands panel and
  `run.json` reading one number per rung.
- **A stone is a *level*, and a run keeps two counters against every rung.** The level is how
  many stones stand on the hand, and it moves what the hand pays; the
  *plays* are how many times the run has actually formed it, and they move nothing at all. They are
  deliberately not derived from one another — buying a rock and landing a Full House are different
  achievements, and one figure could not say which had happened. See *Play counts* below.
- **Ten percent of the base, per stone, additive, floored.** Card Two Pair is 179, so a stone is
  worth 17 and two stones are worth 34 — never 17.9 rounded up, and never a tenth of the number the
  previous stone produced. The tenth stone is worth exactly what the first was. Integer arithmetic
  throughout, like the rest of the damage path. **The Pair's 100 makes its stone worth 10**, which
  is the cheapest step on the ladder and the only one starting from the identity.
- **A stone is spent from the consumables pane, mid-fight, beside the runes.** It needs nothing
  selected, because the shape it raises is written on the record, and it takes effect on the duelist
  standing there rather than at the next fight.
- **It belongs to the run, not to the profile.** Stones are gone when the run is — the same
  lifetime as relics, essences and the deck — and they are written into `run.json` by hand
  *key*, so a rung this build has not got refuses the resume rather than being dropped.
- **The bump rides on the duelist, never on the catalog.** `combat.handTable` is package state
  shared by every fight, every review tool and every test, so a run raising a rung in place would
  raise it for the enemy planner and for `tools/handsheet`. A duelist carries a count per rung and
  the ladder is read *through* it; a duelist with no stones reads the shipped table untouched.
- **The hands panel shows what a hand pays this run**, with a raised figure written in the relic
  pink — the same color a relic-moved figure takes on a card. The shared reading is *something
  you bought moved this number*; a second hue for the second source would be two colors to learn
  one fact.
- **A stone has no rarity and the bag is a flat draw.** Every rung is worth a tenth of itself, so a
  Five of a Kind stone is not a better rock than a Pair stone — it is a rock for a hand you may
  never build. Weighting them would be pricing the *hand*, which the ladder already does.
- **A shape stone is worth more rungs than a single-rung one.** Jasper moves three rungs and the
  Pair's Agate moves one; that is the Pair being already merged rather than a price, and nothing
  measures whether it wants correcting.
- **Nothing measures whether a tenth is the right number**, on the same terms as every price in the
  shop section above. What it is worth in practice depends on which rung a run keeps hitting, which
  `tools/handodds` measures for the shipped deck and not for a run that has been distilling it.

### The pouch — carrying stones

**Using a stone is not the same as owning one.** A run carries stones and decides later which rungs
it raises, which is what makes a stone a consumable rather than a prize.

| | |
|---|---|
| Held | in the run's **pouch**, uncapped, across fights and across a save |
| Spent | from the **consumables pane**, on the combat screen, beside the runes. The shop's `S` panel also uses one, and sells one for `StoneSalePrice` |
| Arrives from | the shop's shelf. **A bag of rocks and a rock shower both apply what they draw on the spot** |

**A stone that arrives already decided is applied where it lands.** The bag of rocks is a pick
out of four and the shower is a flat draw of several, and in both the rung is
settled by the time the player sees it — so a pouch filling with rocks nobody chose is an inventory
chore in front of a decision that has already been made. What the pouch is for is a stone the run
*bought*, which is a decision still to make.

- **The pouch is a list of keys and `stones` is a map of counts**, which is the opposite shape for
  the opposite reason. The counts are what the ladder reads and two Agates there are genuinely one
  number; the pouch is a row of cards to click, and two Agates in it are two separate decisions.
- **The consumables pane holds every kind of carried thing**, and `session.Consumable` is the one
  vocabulary it reads — a kind, the record, and where in its own list it came from. Runes lead the
  row because a rune has to be *aimed* and the hand's selection is read toward it; a stone stands
  wherever it stands. **A new consumable is a kind, a picture and a sentence**: see
  `screens.drawConsumableCard` and `screens.consumableTipLines`, which are the whole of what the
  screen learns about one.
- **Only the runes reorder.** The row is the sack then the pouch, so a seat past the last rune is
  not a sack position; a drag across the join would reorder by a number meaning something else. A
  stone has nothing to be before or after.
- **Selling pays 5, which is what a whole bag costs.** Deliberately generous rather than tuned:
  selling exists so a rung you will never build is worth something, and a price that made selling
  pointless would leave the pouch full of rocks nobody wants. One number, `StoneSalePrice`, and the
  obvious thing to move first if the pouch turns out to be a vitae fountain.
- **Selling is the shop's and nowhere else's**, because selling is a trade and the shop is the only
  screen that trades. Using is on both, since a rung is worth raising in front of the hand that
  wants it and also worth raising while shopping for the next one.
- **A panel, not a row.** The shelf ends at y=684 and Leave is centered at 845; a card row wants 224
  of the 160 between them. So the pouch is the shop's third corner toggle beside `D` and `C`.
- **Spending asks twice**, on the worn row's own argument: a stone armed by a click puts two tabs
  under it, because a rung raised cannot be lowered and a sale cannot be undone.
- **The pouch is snapshotted separately from the placed counts.** A carried stone is a decision
  still to make and a placed one is a decision already made, and folding the two would lose that.

### Play counts — the run's other number against a rung

**A run keeps two counters against every hand, and they say different things.** The *level* is how
many stones stand on the rung, it is bought, and it moves what the rung pays. The *plays* are how
many times the run has formed the rung, they are earned, and they move nothing at all.

- **Two counters rather than one derived from the other.** Buying a rock and landing a Full House
  are different achievements, and a single figure could not say which had happened. A rule that
  wanted playing a hand to *improve* it would be a mechanic, and it would go through stones.
- **The rules never see it.** `combat.Duelist` carries stone counts because the resolver has to read
  them through `HandTable`; it carries no play counts, because a hand pays what it pays however
  often it has been formed. The tally lives on the run — `internal/session/play.go`.
- **Counted off the resolved round, never off the playback.** The combat screen reads the event log
  the instant `ResolveRound` hands it over, exactly as it takes the round's vitae — so a player who
  leaves the screen mid-animation cannot change the count. Presentation may never change an
  outcome, and a tally is an outcome.
- **Every `KindHand` counts, the No Hand included.** A turn with an attack in it always forms a
  hand, so the fallback is a rung the player built as much as any other, and a ladder whose
  commonest rung was the one stuck at zero would read as broken rather than as deliberate.
- **The player's side only.** A creature has no run behind it.
- **Saved, and dropped rather than refused on a rung this build has not got.** That is the opposite
  of what a stone gets: a stone is something the player paid for, and losing one changes what the
  run pays. A tally is a statistic, and refusing to open a save over one would be the machinery
  mattering more than the game.
- **The hands panel is where both are read.** A rung the run has neither played nor raised carries
  no annotation at all — every rung reading `PLAYED 0` is noise around the two or three the
  player is actually working on.


## Essences — altering the deck between fights

**An essence is a change to a card you already own.** It recolors it, removes it, or copies it. It
never invents one, and that restriction is the whole safety property: the *concept* is never
touched, so nothing an essence produces can be a card `internal/combat` has not registered.

**Offered after a won fight, on the post-battle screen.** Two essences are drawn from the catalog
and shown as cards; pick one, pick the card it takes, **see what it would become**, and confirm.

| | |
|---|---|
| When | after a won fight, before the next room |
| Beside it | the win **reading itself out** in the first third of the width: interest, a tenth of the life kept, what the room pays |
| Offered | **two cards**: two essences from `data/essences.json`, in the two thirds beside the payout |
| Then | a hand dealt fresh off the whole run deck, at the combat hand size |
| Then | a **morph**: the card before and after, side by side, nothing committed |
| Choice | one essence and one card, or **Let them escape** |

- **Essence first, card second.** Asking for a card and then offering verbs on it makes the
  reward look like a property of the card. **What you were given for winning has to be the thing
  on the screen when you arrive.**
- **Two options, so the choice is a comparison.** One is an instruction; three is a menu to read
  rather than a decision to make. They are distinct by construction — the offer shuffles the
  catalog rather than drawing from it twice.
- **The card offer is dealt off the *whole deck*, not off what the fight left in the piles.** A
  reward is about what you own.
- **Two random streams, and they are separate on purpose.** Which essences and which cards are
  drawn from different lists and change on different schedules: sharing would mean adding an essence
  to the catalog silently rerolled which cards every fight of every run offered.
- **Nothing is committed until the take.** Back steps out of the morph to the cards and out of
  the cards to the essences, so a player who picked up the wrong essence or aimed it at the
  wrong card is never stuck with either.
- **The preview runs the real essence against a throwaway copy of the run**, never a second
  implementation of what each target does. A preview with its own arithmetic is a preview that can
  disagree with the thing it previews.
- **The result flies to the middle and is held there.** A card won and immediately lost into a
  deck of dozens is a reward the player never sees, which is the same reason the morph exists
  — and it *travels* rather than appearing, per CLAUDE.md's rule that cards always do. The hold
  does not start until it lands, so a slower flight is a longer look rather than a card arriving
  late to a countdown already running. **A removal has nothing to fly**: what was won is an
  absence, so the empty seat is drawn and nothing crosses the screen.
- **The payout narrates itself beside the offer**. The sentences type
  out at the game's speed in the first third of the width, and the figure each one names then
  **flies to the duelist card** and lands in the purse. It gates nothing: the essences are up from
  the first frame and are clickable while it reads. **A click on neither row skips to the end**,
  and taking an essence or walking away claims whatever is left unread — the fast path pays through
  the same claims as the slow one, because presentation may never change an outcome.
- **Two columns rather than two screens**. A win is one thing, and the
  payout and what it is offering are read against each other: what an essence is worth is judged
  with the purse it landed in on screen. The split is one number both columns derive from, so
  neither can be centered into the other.
- **One edge runs across the band**. The payout's last line, the two
  essences and **Let them escape** all end on it, so the two columns read as one row rather than as
  two things that happen to be side by side. The payout is therefore laid out from the bottom up: a
  fight paying no interest is a sentence shorter, and a block hung from the top would float by
  exactly that sentence.
- **The essences touch, and the way out stands off the end of them**.
  The pair is the question and is compared across, so nothing stands between them; the answer that
  is not a card sits where it cannot be read as a third one.
- **Your build is on screen throughout**: the duelist card in its usual corner and the worn
  relics beside it, so an essence is chosen against the thing it would be changing, and the
  purse the payout lands in is visible while it climbs.
- **Taking neither essence is a button, not a third card.** The offer is the two creatures the
  prose says are fleeing, and a third card paying vitae would charge for something the win pays
  by itself. **The offer is free**, so walking away costs nothing and must not look like a card.
- **The essences fly in from the sides** as the screen opens, because the payout beside them says
  two creatures are fleeing — a card already sitting there would contradict it.
- **Run-scoped, never persisted.** Two runs from the same seed may hold different decks, because
  an alteration is a *choice*: replay is a seed plus a choice log. See the `randomness` skill.

### The grammar: a target and a new value

An essence record is the card language pointed at a card that already exists — see
`data/essences.json` and `internal/session/essence.go`, which is where a record is validated.

| Target | Value | What it does |
|---|---|---|
| `element` | a color | recolors one card — one essence per color |
| `remove` | — | takes one card out of the run |
| `duplicate` | — | puts a second copy of one card into the run |
| `cost` | a signed delta | changes what one card costs |
| `amount` | a percentage | scales one card's figure, whatever that figure is |
| `promote` | — | one rung up its form's ladder: Thump → Bash → Smash |
| `demote` | — | one rung down: cheaper and weaker |

**The vocabulary is closed**, the same posture the card verbs take: a new target is a Go change
plus one place applying it, never something a JSON file can assert into existence. A bad record —
an unknown target, an element the rules lack, a value on a target that reads none — **panics at
init**.

**`amount` reaches every card in the deck with one essence**, because what the figure *is*
depends on the verb: a defense percentage, shields raised, or a damage multiplier. That is the
card language paying off — one essence, four meanings, no special cases.

**Cost and amount are per-card and the rest of a card is not.** `combat.Card` carries `CostDelta`
and `AmountPct`; `Cost()`, `Amount()` and `Damage()` are the three methods that read them, which is
where the bounds live. **Form and label are still concept-wide**, so an essence reaching for one of
those would change every copy of that card in the deck — that is the argument to make again from
scratch before adding one.

### The bounds

| | Floor | Ceiling |
|---|---|---|
| Cost | **0** | none declared; the dealt concepts run 1–3 |
| Amount | 1 | a shield card is clamped at `MaxActions` shields |

- **A card may be driven to 0 AP**, and that moves the game onto its
  other bound: a round is capped by cost *and* by count independently, so a free card is limited by
  `MaxActions` rather than by the budget. Taken with that in view, not as an oversight.
- **No card can promise more shields than a turn can throw attacks.** `RegisterConcept` refuses
  a *concept* declaring more than `MaxActions`; `Card.Amount` clamps a *modified* one. It clamps
  rather than refusing, because a reward that silently did nothing is worse than one that hits
  its ceiling.
- **`amount` compounds rather than replaces** — 150% twice is 225% — so a second essence on the same
  card is worth taking.
- **A ladder is a ring**. The top rung promoted lands on the bottom and
  the bottom demoted lands on the top, so there is no card an Exalt or a Debase cannot reach.
  `combat.NeighborWrapping` is the essence's door and `combat.Neighbor` — which stops at both ends —
  is still what a relic demoting a card as it is dealt reads: a relic aimed at nothing, and a wrap
  there would turn one that weakens a hand into one that hands it the top rung.
- **The two ladders never meet.** A defend card is not an attack, so it walks its own rung list and
  wraps inside it; a card with no form has no ladder at all.

### The card says what the card does

**Effect text reads the card, not the concept.** The text is a template over the card's own
value, so an altered Brace prints the shields it actually raises and an altered attack prints the
multiplier it actually swings at. **A card whose face disagrees with its behavior is the worst
thing an alteration mechanic can produce.**

**An essence may not recolor a card to basic.** That would be a way to *lose* a color rather than
choose one, and no attack card in the deck is drab.

**The recolor essences are one per color, and the set has to stay complete.** A color with no
recolor essence is a hole a player can see: every other color can be built toward out of the
post-battle offer and one cannot. **A new color therefore brings an essence with it**, and the
dilution — two seats drawn from a slightly bigger pot — is the same one the flip relics take and
is accepted for the same reason.

### REMOVE is the strongest option, and that is accepted

Thinning a 55-card deck against a fixed hand of eight raises consistency every time, and the
wider the deck the more one removal is worth. It is deliberate rather than unnoticed: the offer
is two essences out of a growing catalog, so removal being the best of what is on the table is
only sometimes the question. **`duplicate` is the one most likely to need a cost** — copies are
the sharpest dial in the game, since four of one concept in a turn is a Four of a Kind.

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

## Runes — altering the deck *during* a fight

**A rune is a consumable you carry into a duel and spend between its turns.** It is bought
from the shop in a **sack of runes**, held in the run, and spent from a board piece on the
combat screen — clicked in the **consumables pane** on the top row, aimed with the hand's own
selection. See the section below.

**They overlap with essences on purpose**. Several runes do what an essence already
does. What makes them a different thing is *when* you spend one: an essence is won after a fight and
applied on the spot, and a rune is carried and spent in the gap between one turn and the next.
The overlap is accepted rather than designed around; the two catalogs are separate files with
separate grammars.

| | |
|---|---|
| Bought | the shop's **sack of runes** — 5 vitae, holds 4, keep 1, offered on visits where the pack roll puts it up |
| Held | in the run's sack, **two at a time**, across fights and across a save |
| Spent | on the combat screen, **between turns only** — select the cards, then click the rune |
| Targets | cards **in the hand**, by identity, one or two of them |
| Lasts | the rest of the run |

### A graft makes the left card the right card whole

**"LEFT CARD BECOMES RIGHT CARD" is not a partial verb.** Everything the right card is travels:
concept, element, form override, the essence-written cost and damage deltas, and the riders.
Copying the concept alone would graft a fire Cut onto an ice Jab and produce an ice Cut — a card
whose name says it became the right-hand card and whose color says it did not.

- **The identity does not travel.** `combat.Card.ID` says *which* card this is rather than what it
  is, so the run holds the same cards it held before, each findable by the handle it has always had.
- **The upgrade travels with it, and that is a real balance consequence.** A graft is a way to
  duplicate a wildcard, or a gold card — the most valuable thing a card can carry. It is what the
  card promises, and the alternative was a carve-out for one rider that nobody could predict from
  reading the card. The wash follows for free, because `screens.upgradeOf` derives it from the rider
  rather than it being copied separately.
- **A rune that changes the *run* rather than a card cannot be grafted**, by construction. Hoard
  and Cairn take no targets, so there is nothing on a card for the graft to find.
- **The offer refuses only a pair the graft would not change**, comparing everything the apply
  copies. Refusing any pair sharing a *concept* would make two colors of one card illegal, which
  is the pick a player reaching for this most obviously wants.

### The sack holds two, and the top row says so

**A run carries at most two runes**, and the count is drawn where the worn relics' is: the top
row of every screen that shows a build is now **two panes** — `worn/5` relics on the left, `held/2`
runes on the right.

- **The cap came from the pane and not the other way round.** A row drawn as `n/2` has to be a rule
  or it is a lie the first time a third rune arrives. `session.MaxHeld` is that rule, and it
  reads the same way `combat.MaxWornRelics` does — one number, read by the screen rather than
  restated in it.
- **What it buys is that the third purchase is a decision.** An uncapped consumable is one a rich
  run hoards rather than spends; with two seats, a sack bought while both are full is a rune
  you have to spend one to make room for.
- **The shop's sack seat goes dim when the sack is full**, rather than taking five vitae for a
  rune that would be refused. It is the same courtesy an unaffordable good already gets, and it
  is the control the player actually meets — `Hold` refusing is the belt behind it.
- **Only the sack needs it.** A stone is spent in the dialog that opened the bag and an essence
  in the dialog that opened the vial, so neither hands the run something it has no room for.
- **The pane is where a rune is visible at all**, and it is on every screen that shows a build —
  so what a run is carrying is readable on the screens where it decides what to carry, not only
  mid-duel.
- **The whole row packs at one pitch, and it overlaps**. The combat row spans 1443
  pixels between the two fighter cards; five relic seats and two consumable seats do not fit at full
  size there and no gutter arithmetic makes them. So the pitch is *solved* for the span rather than
  chosen — 201 against a 203-pixel card, a two-pixel overlap — and **both panes take the same one**,
  which is what makes the line read as one row divided rather than two rows at two spacings. A row
  closes up rather than shrinking a card, because a smaller relic is a different drawing.
  **The shop and the reward screen pay nothing**: there is no opponent card, the span is 1662, and
  the pitch hits its cap at 229 before it hits the span, so nothing overlaps there.
- **A fixture may still plant more than two.** `session.StartingRunes` goes past the cap on
  purpose, exactly as a scenario's hand may be longer than the game's own — four fixtures walk six
  runes through the dialog. The pane draws the first two seats and the count reports the honest
  number, so an over-full sack looks like what it is.

### Select the cards, then click the consumable

**Aiming a consumable is select-then-click**: select the cards in the hand, then click the rune in
the consumables pane. The essence offer follows the same order, so there is one way to point a
consumable at a card anywhere in the game, and no modal, no two-stage prompt and no second drawing
of the hand row anywhere in it.

- **A consumable is clickable exactly when the selection is what it needs**, and dim otherwise —
  the same state an unaffordable card on the shop shelf takes. That predicate is the whole of what
  makes the reversed order readable: the player is never asked whether they have picked the right
  cards, because the consumable that wants them is the one that is lit.
- **Exactly, never at least.** A rune that ate two cards out of a selection of five would be
  choosing for the player which two, and nothing on screen could say which it picked.
- **The order of a selection is the order of the row.** A rune naming a first and a second
  target — Clone — reads them left to right, and the player reorders by dragging, exactly as they
  reorder the round. There is no separate click order to learn.
- **What it costs, said out loud.** On the combat screen a selected card is also a card queued for
  the round, so one gesture now carries two meanings. The objection was raised and overruled; the
  lit-when-legal rule is what makes it workable, and the cost is that a player who has selected a
  turn's worth of cards may find a rune lit that they did not mean to think about.
- **A rock shower's stones fly to the duelist card** rather than putting a receipt up to be
  dismissed. They are in the pouch before the first frame is drawn — the flight is a picture of
  where they went, not a confirmation, and it cannot change an outcome.
- **The reward screen is one stage, not two.** Both rows are on screen: select the card, click
  the essence. Legality follows the order, so the *cards* are all selectable and the *essences*
  go dim.
- **A tutorial anchor here has to cover both rows.** The lit square is also the one legal click,
  so a step lighting only the essences is a lock-up when taking one needs a selection first.

### An essence can be carried into a fight

**An essence is a consumable now, beside a rune and a stone.** It is still what a won fight offers
and still what a vial sells, and taking one there still spends it on a card there — that is the
ordinary way to meet one. What is new is the **satchel**: an essence carried into a duel, drawn in
the consumables pane, and aimed at a card in the hand.

**Why it can be.** An essence edits the run's deck and so does a rune; `resyncHandFromRun` is
what makes either legible mid-fight, and the hand morphs are raised off the difference between
the hand before and after — so no essence needs a picture of its own. The gate is `planning()`,
the rule every consumable is under: `ResolveRound` decides a whole round before a frame of it is
drawn, so a card altered during playback would show a face disagreeing with a blow already
computed.

- **One card, and no record may say otherwise.** A rune's record says how many cards it eats; an
  essence's does not, because the count is a fact about the mechanic. What moves it is a relic —
  see *An essence takes one card, and a relic widens it* below. It is lit when exactly that many
  cards are selected, dim otherwise, which is the predicate the whole pane reads.
- **Aimed by identity, never by deck position.** A fight holds copies of the run's cards across
  three piles, so the card in the hand is found by `combat.Card.ID`. `Session.CanApplyTo` and
  `Session.ApplyTo` are the two doors; `Session.CanApply` and `Apply` still take a position and are
  still what the reward screen and the vial use.
- **A copy joins the hand it was copied from.** `spawn` spent between fights only has to put the
  card in the deck; spent mid-duel it has to reach the hand or it reads as a dud. `ApplyTo` hands it
  over through `Session.Duplicated`, which is the duplicate rune's own handover.
- **There is no cap.** `MaxHeld` is two because the pane draws `held/2` and a fraction has to be a
  rule. The satchel is counted with the pouch, which has never had one.
- **Carried in acquisition order, and written down.** `RunSnapshot.Satchel` sits beside `Held` and
  `Pouch`; an essence the catalog no longer holds refuses a resume rather than being dropped, which
  is what a carried rune and a carried stone both do.
- **Nothing puts one in the satchel yet except a fixture.** `scenarios.json` takes `"Essences"`, and
  the `ladder-wrap` record is ten of them. Where a run *acquires* a carried essence — a shop seat, a
  sealed good, a reward that offers keep-or-spend — is an open question and a catalog decision.

### A sealed good is a screen

**Opening a bag of rocks, a sack of runes or a vial of essence takes the whole screen**, with the
build band up, the draw pile in the corner and the deck panel a click away. What is on it is a
decision about the build — which rung to raise, which rune to carry, which essence to spend and on
which cards — and a panel covering the screen to ask that was covering the answer.

- **The deck panel is on every between-fights screen**, not just the shop: the reward screen
  aims an essence at a card off a deck of fifty-odd while showing eight of them, which was the place
  the deck mattered most and could not be read. `deckpile.go` is the pile all three draw, and each
  screen owns its own toggle and its own click.
- **The ways out are taking a card or pressing SKIP**, which takes nothing and forfeits what the
  good cost — the reward screen's LET THEM ESCAPE on the same terms. The chrome stands down, as it
  does on the reward screen.
- **It is not a station of the run.** The run is standing in the shop the whole time and is standing
  there when the good is finished with. See CLAUDE.md §Five screens that are not stations of a run.

### An essence takes one card, and a relic widens it

**One card is the mechanic, and the number is not on any record.** Every essence in the catalog
reaches exactly as far as every other, so what an essence *is* stays a single idea — a change to a
card you already own — and a record that could ask for three would be a second dial hidden inside a
catalog nobody prices.

**What moves the number is a relic**, at the `essence-spent` moment: `adjust-essence-targets` moves
it, and `Session.EssenceTargets` is the one seat every spend site asks through. Cloud Necklace is
the first, at +1.

**A card at a time, and every delta sums.** Worn order decides nothing because addition commutes —
the argument `adjust-round-limit` is already under — so two Cloud Necklaces are three cards. What
that buys is a step the essence catalog can be priced against: a doubling would make a second copy
worth four cards and a third worth eight, which is a curve nothing else in the game is on.

- **Never below one card.** A scaling that rounded away would take the essence mechanic off the run
  rather than making it meaner — the clamp `SetRoundLimit` and `SetRelicSlots` are both under.
- **Never more cards than the screen is offering.** A run wanting four cards out of a deck of three
  would leave every essence in the game unclickable, so each spend site clamps to the row in front
  of the player: the reward offer, the vial's offer, the hand. A consumable that can never be
  clicked is worse than one that reaches less far than it promised.
- **The reach is a ceiling, never a quota**. A player wearing a Cloud
  Necklace may spend an essence on one card. Being made to spend the whole reach would turn a relic
  that gives you more into a relic that takes the small move away — and there is nothing ambiguous
  about a short selection, because **every card selected is a card the essence lands on**. That is
  what `consumableTarget.fewest` opens, and it is the one consumable in the game with a range: a
  rune names its cards, so a selection of the wrong size leaves it unclickable.
- **What the tooltip says is what this click would do.** With cards picked it counts them, and with
  nothing picked it says how far the essence can go — a panel promising two cards to a player who
  has picked one is describing a click they are not making. `screens.essenceReach` is the one
  answer all three spend sites read.
- **All or nothing.** Every card is checked before any of them is changed, so an essence that lit up
  cannot land on two cards and refuse the third. `Session.CanApplyToAll` is the question and
  `Session.ApplyToAll` is the answer, and they ask the same thing.
- **The deck is edited from the back forwards.** A removal shifts every position above it and leaves
  everything below it alone, so a descending walk is what makes several targets in one spend safe.
  The preview walks it the same way, or the picture would be of a deck the run never reached.
- **A card named twice is refused.** Neither the offer row nor the hand can produce one; a spend
  that changed one card twice would be an essence quietly doing half of what the player was
  shown.
- **The order of a selection is the order of the row.** The settled cards land left to right in the
  order they were sitting in, whatever order they were clicked in — the rule every consumable on the
  combat screen is already under.
- **One beat for all of them.** Every card a spend took changes at once. Four dissolves in sequence
  would be four pauses over a row the player is waiting to read, and what the beat says is one thing
  about the essence rather than one thing about each card.
- **A full selection replaces its oldest card.** At the cap there is nothing a further click could
  add, and a row that ignored it would leave a player who picked the wrong card having to work out
  that they must deselect one first. With one target that reads as the pick simply moving.

**The essence says how far it reaches, in its own tooltip**. CARD
BECOMES FIRE is what one card reads; two is 2 CARDS BECOME FIRE, DESTROY CARD is DESTROY 2 CARDS,
and COPY CARD is COPY CARD TWICE. Nothing else on the screen says the number — the row lights the
essences only once it has enough cards, which says *when* and never *how many*.

- **The catalog is authored for one card and the wider lines are derived.** `Essence.TextAt` is the
  derivation and `data/essences.json` is untouched: a plural string beside `Text` would not be one
  string but one per reach, so fifteen records would each carry a sentence per count, all saying the
  same thing a different way.
- **Two authored shapes, and they are rewritten differently.** `CARD <verb> ...` is a sentence about
  the cards, so the count leads and the verb agrees with it; `<verb> CARD` is a sentence about the
  doing, so the count lands on what is being done to.
- **A copy is counted in times rather than in cards**, which is read off the essence's *target*
  rather than off its wording — so a second duplicating essence gets the same treatment without
  anybody noticing it needed it.
- **A line fitting neither shape is printed as it was written.** Mangling is the worse failure: a
  sentence nobody can read is harder to spot than one that has not learned to count, and
  `TestEveryEssenceSaysHowFarItReaches` fails on a record the rewrite cannot reach.
- **The review sheets show the catalog as shipped**, which is one card. A sheet is a picture of what
  was authored, not of a run that has bought a relic — the rule `tools/handodds` and
  `tools/handsheet` are already under for stones.
- **The consumables pane is the one place the count is not clamped to a row.** An essence carried
  past a between-fights screen is spent in a fight that has not been dealt, so there is no hand to
  clamp against and what the pane says is what the essence will do.

### Nothing is greyed out for being pointless

**A pick that would change nothing is legal, and the burden is the player's.** Painting a lightning
card lightning, grafting a card onto its own twin, writing an upgrade a card already carries — each
of those wastes the consumable, and each is wasted by a player who chose that card over every other
card in front of them.

**Refusing a pointless pick is a card sitting dead under the cursor while the screen declines to
say why.** A player who cannot tell an illegal pick from a pointless one cannot learn either, and
a mechanic that quietly edits the set of things you may click is teaching a grammar nobody wrote
down.

**What is still refused is a pick the rules cannot resolve**, which is a different question:

- the wrong number of targets, which is what stops a two-card rune being spent on one
- the same card named twice, which would spend a two-card rune on one card for double the effect
- a card or a deck index that is not there
- a chimera with nothing behind it to copy — there is no effect to resolve, so there is nothing to
  waste

`Session.CanApply` and `Session.CanApplyRune` are the two doors, and they now answer that narrower
question. **The consumables pane still goes dim on the count**, which is the selection being the
wrong shape rather than the pick being a poor one.

### Why between turns, and never inside one

`ResolveRound` decides a whole round before a frame of playback runs, so a card altered
mid-playback would put a face on screen that disagrees with a blow already computed. The board
piece is gated on `planning()`, which is the screen's existing predicate for "the player may edit
the queue". **This is the same constraint that governs playback speed, the debug flags and every
card in flight**, arriving at the one mechanic that genuinely wanted to break it.

**Card identity is what makes it possible at all.** A rune's targets are `combat.Card.ID`s
rather than deck positions. That matters more here than it does
for an essence: a rune may name two cards, and it is spent while a hand, a draw pile and a discard
pile are all live holding copies of the same cards, so a position would be meaningless by the time
the player confirmed.

### The grammar: a target, a count and a value

A rune record is the essence record's shape plus the two fields an essence never needed — **how many
cards it takes**, and **what class of change it makes**. See `data/runes.json` and
`internal/session/rune.go`, where a record is validated.

| Target | Change | Value | Count | What it does |
|---|---|---|---|---|
| `rider` | **upgrade** | a figure, plus a `Rider` name | 1–2 | writes the card's one upgrade |
| `remove` | normal | — | 1–2 | takes cards out of the run |
| `vitae` | normal | a figure | **0** | fills the purse and touches no card |
| `duplicate` | normal | — | 1–2 | copies a card — **and the copy joins the dealt hand** |
| `element` | normal | an element name | 1–2 | recolors cards |
| `form` | normal | a form name | 1–2 | changes what cards **count as** on the form axis |
| `stones` | normal | how many | **0** | puts that many random stones in the run's **pouch** |
| `clone` | normal | — | **2** | the first card picked becomes the second |
| `chimera` | normal | — | **0** | fires the run's last rune again |

### Normal and upgrade: the two classes of change

**A card is a form, an element and an action — and then one upgrade.** The first three compose
freely: a Jab painted fire and reformed to crush is all three at once, and a rune that moves one
of them has no opinion about the others. The fourth is different. **A card carries exactly one
upgrade, and writing it discards whatever was there** — `combat.MaxCardRiders` is 1, and
`Card.SetRider` replaces rather than stacks.

**So a Siphon on a golden card leaves a card that heals and has forgotten it was ever gold**, and a
Bulwark on that same card leaves a Guard that still heals. That is the whole distinction, and it is
worth naming because it is **invisible in the effect**: Bulwark and Golden both read as "a rune
changed my card", and what separates them is what the *next* rune does.

- **`Change` is a required field on every record**, `normal` or `upgrade`, from a closed vocabulary.
- **It is authored rather than derived, and the loader refuses a record that disagrees with its own
  target.** Every `rider` rune is an upgrade and nothing else is, so it *could* have been
  computed — and a computed field says nothing, where an authored one is a claim the record makes
  and the loader checks. Same posture as `Match` in the tutorial script: the thing the author meant,
  written where the author is looking.
- **No rider pick is illegal.** The same upgrade twice writes what the card already carries and
  wastes the rune; a different one replaces it outright. See §Nothing is greyed out.
- **Riders do not stack**, and that is a deliberate bound on the rune economy: two Siphons on
  one card would be twice the heal for no more thought.

**Three of the targets are worth saying twice:**

- **`duplicate` is the mid-fight `spawn`, and the copy has to reach the hand.** An essence copying a
  card only has to put it in the deck, because it is spent between fights. A rune is spent in
  the middle of one, and the fight's piles were dealt before the copy existed — so a copy that went
  only into the run would not be playable until the *next* fight and would read as a dud. The copy
  is a new card with a new identity, arrives unselected, and `Session.Duplicated` is the handover.
- **`form` is an override on the card, not a replacement of the concept.** A Brace told to be a
  crush is still a Brace: it still shields, and it now counts as a crush when the hand is
  matched. **A defend card is a legal target and that is the point** — it produces a card that
  shields and matches on an attack axis, which nothing in the catalog does.
  `combat.Card.FormOverride` is the field and `Card.Form` is the one chokepoint that reads it.
- **`stones` is the first thing in the game that rolls while it is being *spent*.** Every other
  draw decides what a shelf is offering and is a function of the fight; a run may carry three rock
  showers and spend all three in one fight, so the fight index alone would hand out the same three
  stones each time. `seeds.StoneShower` is the stream and **the number of stones the run has already
  placed is mixed in** — a figure the snapshot already carries, so a resumed run rolls what it would
  have rolled. **The source is the caller's**, because the run does not know its own seed;
  `ApplyRuneRolling` is refused outright without one rather than falling back to a default draw.
- **They go into a pouch, not onto the ladder** — see §The pouch above. The dialog stays up
  afterwards as a *receipt* rather than an offer, and any click dismisses it.
- **`clone` is the one target whose two seats are not interchangeable.** Every other rune
  treats its targets as a set; this one is directional — first pick changes, second pick is the
  template — so the picker's click order is a rule rather than a detail. It copies the concept and
  keeps the first card's identity, riders and modifiers.

### Gold and silver: the second roll in the game, and it is on a card now

**Golden and Silver are upgrades, and they gamble every time their card is played.** A **gold**
card rolls a d5: on a 1 the run gains **+1 DMG** permanently, on a 2 it gains **+5 max life**,
and on a 3, 4 or 5 nothing happens. A **silver** card rolls the same die and pays **+10 vitae**
on a 1 and nothing otherwise. Both are `rider` runes, both take one card, and the `Value` is the
denominator.

**The luck lives on the card rather than in the consumable.** What the rune buys is what a card
permanently *became*, so it rides through the shuffle and rolls again on every play, for the rest
of the run — which is why the denominator is the dial: a cheap starting card is played dozens of
times a run.

**One roll, three outcomes — two independent rolls were the alternative and were declined.** Two d5s
would have paid something 36% of the time and *both* 4% of the time, and a headline outcome that
rare is one most runs never see while the runs that do see it price the card off it for ever. One
die with a losing face is a gamble a player can hold in their head.

**A roll needs its own argument and this is it.** The `randomness` skill is explicit that
lightning is the exception rather than the precedent: certainty is usually the better game as
well as the cheaper code. The exception here is that **the card's whole subject is luck**. Every
other random-sounding rule in the game had a deterministic rewrite that was at least as good;
this one does not, because a metal that always paid is a purchase, and the catalog is already
full of purchases.

- **The roll is in `internal/combat`, because the card is played there.** It is the second thing in
  that package that rolls, and it takes **its own injected source** — `combat.Sources` is a struct
  with a `Roll` for the shock and a `Luck` for the gamble, never one field. Sharing would make every
  shock in a run a function of how many gold cards were played, and every gamble a function of how
  often the player was shocked.
- **`seeds.LuckRoll` is the stream**, per fight, and it is a **live cursor** rather than a seed
  plus a counter: a roll that happens inside a resolved round has the round's own sequence to
  advance, so nothing on the run has to tally how many rolls have been taken.
- **What it grants is permanent and run-level**, so it lands on `dmgBonus` and `lifeBonus` — the two
  figures a potion moves. **The rules move both halves**: the fighting duelist, so the point of DMG
  is worth something for the rest of the fight, *and* the run, through `KindGrantedDMG` and
  `KindGrantedLife` read off the resolved log by `screens.settleGrants`. Doing only the first would
  be a bonus that evaporated at the round boundary; only the second, one the player could not use
  until the next fight.
- **Silver needs no grant event.** Vitae already travels out of a resolved round as the difference
  between the purse the duel opened with and the one it closes with, so a silver card steps
  `Duelist.Vitae` and announces a `KindVitae` like any other payment.
- **`LuckOutcomes` is 3 and a record naming fewer faces is refused at load**, because a die with no
  losing face is a different card and the mistake is one a number in a JSON file could make quietly.
- **The payouts are Go constants rather than record fields.** A record has one `Value` and gold
  needs two figures, and **the odds are the interesting dial** — a richer metal is written by
  moving the denominator, not by paying more per hit.

**What nothing catches is the balance.** A card that gambles on every play is worth however
often the player plays it, which on a cheap card in the starting deck is dozens of times a run.
Nothing in the repo simulates a run, so the dial to move is the denominator in `data/runes.json`
and no code changes.

### `chimera` fires the last one again

**It carries no effect of its own.** What it costs, how many cards it names and what it does to them
all come from the rune it is copying, resolved through `Session.Echoes` before anything reads
it. A chimera behind an Embermark asks for two cards and paints them fire; a chimera behind a Hoard
asks for none.

- **The memory is the run's, not the fight's.** A chimera carried out of one duel and into the next
  still copies what was spent in the first. `Session.lastRune` is the record key and it is
  **saved with the run**, so a resume does not forget.
- **It refuses only on a run that has spent nothing.** There is no effect to copy, and a consumable
  that landed and did nothing is something bought and taken away. The consumables pane draws it dim,
  the same courtesy an unaffordable relic gets.
- **The targets are picked again rather than inherited.** The copied rune's cards are long gone
  from the hand by the time a chimera is spent — a different turn, sometimes a different fight — and
  re-firing against the same identities would be a no-op wherever the effect was idempotent, which
  is most of the catalog.
- **A chimera never becomes the thing to copy.** `rememberRune` records the *resolved* record, so
  two chimeras in a row both fire the rune behind them rather than the second copying the first
  into nothing.
- **A remembered record the catalog no longer holds is forgotten rather than refused**, which is
  the one place `Resume` is lenient and is deliberate: a *held* rune is a thing the player owns
  and would notice going missing, where this is a memory of one already spent. The worst it costs is
  a chimera with nothing to copy, which is a state the mechanic already has a rule for.
- **The card's face says what it would fire** — `COPIES / HOARD` — because its authored line cannot.
  A card whose whole subject is a rune named somewhere else is one the player would otherwise
  have to remember the answer to.

**The vocabulary is closed**, the posture every other one in the game takes. A bad record — an
unknown target, a rider the rules lack, a concept this build has not registered, a `vitae` asking
for a card — **panics at init**.

**A rune that changes what a card is keeps the card's identity, and so keeps its riders.** A card
the player has already spent two runes on stays the card they invested in. Minting a fresh
identity would take the investment with it — `clone` is the target this is about today.

**`MaxRuneTargets` is 2**, because the picker shows the targets side by side and a picker that
scrolled would be a menu to read rather than a decision to make — the two-essence offer's argument.

### Riders: a rule carried by one card

**A relic waits on a finger and fires for every card that matches it; a rider is the same idea
aimed the other way.** It belongs to one card of the run, travels with it through the shuffle, the
hand and the discard, and fires only when that card is played.

| Rider | Fires | Does |
|---|---|---|
| `heal-on-play` | as the card is played, **after a chill has taken what it takes** | restores life, capped at full |
| `shield-on-play` | as the card is played | raises shields, through the same cap a Guard is under |
| `damage-on-play` | the turn the card is played into | **adds to that card's own hit** |
| `damage-in-hand` | every turn the card is **kept back** | adds to the duelist's DMG for every hit of that turn |
| `scale-in-hand` | every turn the card is **kept back** | scales the duelist's DMG, as a percentage |
| `vitae-in-hand` | every turn the card is **kept back** | pays vitae — **announced, never applied** |
| `scale-in-combo` | as the card is played | **scales that card's own hit**, as a percentage |

**The riders add two ideas nothing else in the game has:**

- **A played card's damage rider is a card upgrade** *(owner's call, 2026-09-26)*: step 2 of the
  five in *Damage: a hit per card, one multiplier* — after the card's own multiplier and before any
  relic — and no other hit of the turn sees it. It fires on any card that is played, whether or not
  it made the hand. Flat first, then the percentage, so a +10 and a doubling on one card come to
  `(DMG × card + 10) × 2`. The hand dialog writes each as a row of its own in that card's line —
  `+ 10`, `× 2` — in the rider's tint, flying out of the card.
- **A kept-back card's damage rider is a duelist upgrade**: step 1, raising the DMG every hit of
  the turn is swung at, since the card is not in the turn to own a hit of its own. `Card.Damage`
  is linear in DMG, so every hit grows by the same proportion. `combat.blowDMG` is that rule, and
  the DMG is **put back before the duelist is returned** — a bonus left standing would silently be
  a permanent upgrade.
- **The four in-hand riders are the first mechanic that rewards *not* playing a card.** They pay on
  every turn the card is still being held, so a card dealt on the first turn and kept for three
  has paid three times, and nothing is ever spent. `ResolveRoundHolding` is what tells the
  resolver what a side did *not* play; `ResolveRound` delegates to it with nothing held.
- **The rules hold the purse for the length of a round, and the run holds it the rest of the
  time.** `combat.Duelist.Vitae` is seeded from the run at the top of each
  round and stepped as the round pays; the combat screen then hands the run the **difference**. It
  reads that off the **resolved duelist**, not off the playback, so how fast a round is drawn cannot
  change what the player is paid. `KindVitae` is still emitted, but it is the feed's line rather
  than the payment — summing those events to move a purse is the old way and would now double-pay.
  The rules got a purse because a relic wanted to read one: see Rampant, which raises the duelist's DMG per vitae
  held and would otherwise price a turn-three hit at turn-one rates.

- **The vocabulary is a Go enum in `internal/combat`, not a data record.** Everything else a
  rune does happens to the run; a rider is the one thing read while a round resolves, and that
  package is at the bottom of the graph and reads no JSON. The *amount* rides on the card, so the
  rules need no lookup table and nothing to keep in step with a catalog.
- **A card carries one rider — `MaxCardRiders` is 1**, because a rider
  *is* the card's one upgrade. See §Normal and upgrade above, which is where the grammar it belongs
  to is written down.
- **`Card.Riders` is a fixed array, and it has to be.** `combat.Card` must stay comparable — the
  screen's face cache and `TestRoundIsDeterministic` both depend on it — so a slice would end
  both. Same constraint that made `Duelist.Relics` an array. It stayed an array when the count came
  down to one, because a seat is what makes "no upgrade" the zero value rather than a case.
- **Last one wins, and nothing stacks.** `Card.SetRider` replaces, so a second rune on a card is
  one upgrade forgetting the other rather than two ten-point heals adding to twenty life.
- **A card a chill ate heals nothing**, which is why riders fire after the chill and before the
  hits. A rider on the front card of a turn is exposed to the one thing that can delete it.
- **A heal that restores nothing is silent.** The cap is applied first and the event carries what
  actually landed, so the log never reports life that the bar cannot show.

### The card says what the card carries

Effect text reads the card, so an upgraded card prints an extra line — `+10 LIFE`, `GOLD` — under
its own. **It is not written in the relic pink**: that color means "a relic did this" everywhere
else, and a rune is not a relic.

**Every rider is visible in three places**: the line on the face, the wash over the card, and
the tooltip. None is redundant with the others — the color carries at a glance across
a row of eight, the face's line answers "what does that mean" without a hover, and the tooltip
carries the figures neither has room for. See §An upgrade is painted on the card, and §The tooltip.

### The tooltip is a stat block first

A card's tooltip opens with what the card **is**, in four lines or so:

```
FIRE JAB
1 AP
5 DMG
+10 HEAL ON PLAY
```

The title is the element and the name; then the AP the *holder* pays, the figure the card is
worth in the unit its verb is measured in — DMG for an attack, `1 SHIELD` for a shield — and then
a line per thing the upgrade adds.

**The arithmetic goes underneath the block, and only when a relic or an essence has actually
moved something.** A card nothing has touched derives to itself, and three lines saying so is a
panel that trains the player not to read it.

- **The block's figure carries the relics.** `screens.cardTip` hands `carddesc` the compounded relic
  scale, so the headline number is what the card will deal rather than its bare worth over a chain
  ending in a bigger one. The chain therefore prints **no total** — the block already stated it, and
  two copies of one number is a pair that can disagree.
- **The wording lives in `internal/carddesc`, which is windowless.** That is what lets
  `tools/upgradesheet` print the same strings the game shows rather than a hand-written snapshot of
  them — the trade `tools/cardsheet` makes for card names, taken the other way because the sheet's
  whole job here is "does this read right".
- **Gold gets two lines and every other upgrade gets one.** Its payouts are mutually exclusive, and
  one line joining them with "or" reads as a card that pays both.
- **The element word in the title is colored**, like every other element word in the game. A
  card's title is `FIRE JAB`, so a plain-string title would be the one place that rule does not
  reach and the place it matters most. `Title` and `Lines` are both `TipLine` and go through one
  drawing.
- **A wildcard is CHROMATIC, not the element it happens to be.** The card still *is* an arcane
  Skewer — it burns as one, it is drawn from the arcane row, `Blow.Elements` reports arcane —
  but the title says what the player is holding, and `ARCANE SKEWER` over a line reading `COUNTS
  AS EVERY ELEMENT` is a panel contradicting itself in two lines. **CHROMATIC takes no color**:
  the wheel has none left for "all of them", and writing it in one of the five would claim the
  one thing the word exists to deny.

### The wildcard

`RiderWildElement` makes one card count as **every element at once** when a hand is formed. It is
attached by the **Motley** rune and it is the eighth rider kind.

**The card keeps its own element and everything else goes on reading it.** A wild fire Bash is
still a fire Bash: it lands a burn, it sits in the fire row of the deck panel, and `Blow.Elements`
reports fire for it. One question changes — what it counts as on the element axis — and the answer
is "whatever the hand needs".

**It is the first rider read while the hand is *matched* rather than while the turn resolves.**
Every other rider fires after the hand is already decided; this one is inside `matchCountOf`, which
is what made it a change to the matcher rather than another case in `playRiders`.

Four rules, and each is a thing that is easy to get wrong:

- **Element only.** `Card.Wild` takes an axis and answers for one. A wildcard that widened the
  concept or form axes as well would make the whole ladder a single rung, so a wildcard on another
  axis is a *new rider kind* rather than a widening of this one.
- **It tops a group up; it never seeds one.** A tally is a candidate if its own members plus the
  unspent wildcards can reach the group's size, and the ranking is on its *own* members — so a
  wildcard joins whichever element already has the most of itself, which is the reading a player
  makes looking at the row.
- **It is spent once.** Two groups of a Full House cannot both have the same wildcard, which is
  what keeps a rung out of reach of a turn that has not got the cards for it.
- **A turn of nothing but wildcards is a group.** They agree with each other, so refusing would be
  the matcher saying that cards which match everything match nothing.

**It carries no amount, and the vocabulary says so.** `RiderKind.CarriesAmount` is what
`internal/session` asks before demanding a figure off a rider rune's record, so a `Value` on a
Motley is refused rather than being a number nothing reads.

**This is a balance lever and it was taken as one.** One wildcard turns any three-of-an-element
into a four, and the elemental rungs are high on the ladder — so what a run pays for a Motley is
the number to watch, and that number is the rune's place in `data/runes.json` rather than
anything in the rules.

### An upgrade is painted on the card, and where is still open

**A card has a form, an element and an action, and then one upgrade.** There are ten of them, one
per rider kind, and every one draws: a card the run has altered says so from across the table.

**The card goes gold and the border does not**. `wash-face` is what
`cards.DefaultUpgradeStyle` names: every pixel inside the border ring is pulled toward the
upgrade's ink and the ring itself is left exactly as it was.

**What settled it is that the border is already saying something.** It carries the card's *state* —
resting, selected, unaffordable, being dragged — in a wash away from the neutral gray, so an upgrade
painted over it would be a second thing in the one place the card says the first. Keeping them apart
is what lets a queued gold card read as queued *and* gold rather than as one of the two winning. It
also keeps the card's outline against the table, which is what tells eight cards in a row apart
before any of them is read.

**Two other answers are kept as a review knob**, drawn by `go run ./tools/upgradesheet` beside the
default: `border` paints the 3px ring and nothing else — the quietest answer, at the cost of the
state signal it would be sharing the ring with — and `wash` takes the whole card including the
border, which nobody misses and nothing on the face escapes. It is the shape `TintMode` had, for the
same reason: how loud an upgrade should be is not a question anybody wins by arguing.

- **A relic card is the one card this must never touch**, and it does not: a relic carries no rider,
  so its pink is never washed.
- **`UpgradeBorderPct` is 80 rather than 100**, for the `border` style, so a fifth of the state
  color still shows through the ink.

**It never takes the left column.** That column states the element, and nine of the ten upgrades
have nothing to do with the element — a left column in gold is the element slot saying something
that is not about the element.

- **`systems.Upgrade` is the vocabulary** and it is *presentation*: something visible has happened
  to this card, and here is what to paint it with. `internal/screens` is where a rider becomes one,
  on exactly the terms `Spec.TextInk` is where a relic becomes a color — neither `internal/cards`
  nor `internal/systems` learns what a rider is.
- **An upgrade is painted into the face; a mark is painted over it.** Under the `wash` style
  both cover the whole card, so the drawing does not tell them apart — what does is ownership.
  An upgrade is what the card permanently *is*; a `cards.Mark` is the card's situation. A
  shattered gold card reads as gold and broken, in that order, and the order is fixed in
  `Render` so one pair of facts draws one way.
- **Every upgrade is an ink, and nine of the ten are one flat color.** The tenth is the wildcard,
  whose wash is the five element colors in bands — and a vocabulary where one entry is a picture
  and nine are colors would be two mechanisms with a `switch` between them. An ink holds both: the
  authored PNG for the one that needs a picture, a generated square for the rest, and one sampling
  path that never asks which it got.
- **The wildcard is the one upgrade that leaves the form mark hueless.** The left column exists to
  state the element; a wildcard's element is still what the card *is*, but what it *counts as* is
  every element at once, so a column stating one of them states the less useful half of the truth.
  Every other upgrade leaves the element's tint alone, because none of them is about the element.
- **Eight of the ten colors are placeholders and they are standing on a full wheel**. Hue is
  spent — five elements, the relic pink, the two verbs, the two duelists, the ground — so what
  is there is picked to be *told apart* rather than to mean anything. Gold and silver are the
  exception: they are metals, and they are what the mechanic is called. `go run
  ./tools/upgradesheet` is the page to retune them against.
- **The wash is 55% of the way toward the ink, and the two metals are a *sheen* rather than a
  flat color.** Both of those are the card's own surface pushing back: it is a pale warm
  neutral, so a gentle wash of gold moves the hue a little and the lightness not at all, and the
  card comes out as warm paper. 22, 34 and 40 were all tried and all read as cream. What makes
  metal read as metal is a light running across it, so gold and silver are generated as a
  diagonal band — dark shoulders, a bright crest — through the same ink mechanism the wildcard's
  picture uses.
- **The eight flat placeholders are loud at that strength**, which is the right direction for a
  placeholder to be wrong in. One of them — the heal's rose — sits close to the relic pink, which is
  exactly the kind of collision the "hue is spent" note predicts and the reason these are marked
  temporary rather than settled.

### Targets come out of the hand *(taken while building it, and the one most worth revisiting)*

The picker offers the **hand**, not the whole deck. Two arguments for it: a mid-fight consumable
aimed at the card you are about to play is a decision about *this* turn, where one aimed at a card
somewhere in a pile of forty is the reward screen's decision taken in a worse place; and the hand
is already on screen. **What it costs is reach** — a card in the draw pile cannot be touched until
it is drawn.

### The sack is the pack whose contents are carried

The sack takes the bag's and the vial's shape exactly, and differs in the one way that matters:
**a stone and an essence are spent the moment they are chosen, and a rune goes into the sack to
be spent later.** That is what `MaxHeld` is a cap on, and it is why the sack seat goes dim when
the sack is full rather than taking five vitae for a rune that would be refused.

### Run-scoped, and saved

The sack and every rider are written into `run.json`, **by name and never by ordinal** — the
rule every other vocabulary in a snapshot is under. A rune the catalog no longer holds, or a
rider the rules lack, is **refused on resume rather than dropped**: a run that came back one
consumable lighter is a run the player would have to work out had changed.

---

## Brands

**Nothing implements brands.** There is no `data/brands.json`, no loader and no screen; what
follows is the decided shape, and it is the one section of this file that describes a mechanic
the game does not have.

**Brands alter the container; relics alter the contents.** That is the axis, and it is what tells
you which of the two a new power belongs to:

| | Brands | Relics |
|---|---|---|
| What they touch | the chassis — hand size, total discards per round, relic slots | the cards — elements, costs, and the stats that feed them |
| Removable | **never.** You brand yourself and you do not take it off | freely; five equipped, swap as you like |
| Scope | **for the run** | for the run, but re-chosen after every fight |

- **"Permanent" means for the run, not across runs.** A brand is a commitment made *inside* a
  run that cannot be undone, which is a different thing from meta-progression. Hand *discovery*
  is the profile-scoped mechanic; brands are not.
- **A brand may not grant actions.** The action cap is permanently five and nothing raises it —
  see *A round is bounded twice*. Growing the **hand** is the nearest legal thing, and it is a
  container change, so it fits the axis.
- `[?]` Everything else is open — capacity and rule-bending, with the above as the test for what
  counts.

Like relics, they have **concrete definitions that never really change**, which makes them a fit
for the `data/` pattern: JSON beside a small Go loader.

---

## Vitae

The currency. Earned from fights, spent in the shop. `Session` carries the purse; **winning a fight
is the only thing that adds to it** and **the shop is what takes it out** —
`Session.SpendVitae` is the one place a purse goes down. Its callers are `Session.Buy` and
`Session.BuyGood`, the sealed goods.

**Vitae is crimson wherever it is written** — the purse on the duelist
card and the word itself in the reward screen's prose. It is the run's only currency and now the
only red on a light screen, so a figure in that color says "money" before it is read.

### What a win pays

Three separate things, decided by `Session.WonFight` and handed over one at a time by the reward
screen as it narrates them. See `internal/session/spoils.go`.

| Part | Figure |
|---|---|
| **Interest** | propagation, below — on the purse as it stood when the fight ended |
| **The life you kept** | **a tenth of the life remaining, rounded down**: 65 left pays 6 |
| **The room** | **3** outer, **4** inner, **5** stairway (the floor's boss), flat for the whole climb |

- **A share of the life *remaining*, not of the maximum.** It is a reward for fighting well rather
  than a rebate, and a relic that raises max life pays out more here indirectly — which is intended.
  A win on nine life pays nothing from this part.
- **The room award does not scale with the floor.** What makes a later fight worth more is the life
  you manage to keep in it.
- **Deciding and paying are separate.** The figures are frozen when the fight ends, so nothing
  about the payout depends on when the player clicks; `Session.Advance` claims whatever was never
  narrated, which is what makes a win pay in full even when no screen reads it out.
- **Soul Taker moves the room award**, turning 3/4/5 into 8/9/10 — a flat addition at
  `prizes-dealt`, landing on the one figure a win always pays.

### Propagation — vitae earns interest

**After every fight, a run gains +1 vitae for every 5 it is already holding**, capped at **+5**.
So 5 held pays 1, 10 pays 2, and 25 pays the maximum 5 — holding more than 25 propagates no
faster.

- **It is a rule of the run, not a relic.** The Banker relic scales it, which means it has to exist
  on its own first — a relic may only ever bend a rule the game already has.
- **The cap is what stops it running away.** Uncapped, +1 per 5 is roughly ×1.2 a purse per
  fight, which compounds across 24 fights into a number no shop can be priced against. Capping
  the *rate* rather than the purse leaves a big purse worth having and stops the curve.
- **Rounded down**, like every other integer rule in the game.
- **The cap binds the base rate, and a relic scales what the cap produced**. So at 25 held
  propagation is +5, and +10 wearing Banker. The alternative — an absolute cap on the figure
  that finally lands — would make Banker do nothing past 25 held, which is a relic that stops
  working exactly when a run can afford it.
- **Order of operations, therefore:** count the fives, clamp to +5, *then* apply every relic that
  scales propagation, left to right in worn order. Two such relics compound, like every other
  relic effect.
- **It is decided in `Session.WonFight`**, before the room counter moves and before either award
  above: interest is on what the run walked out of the fight holding, not on what the win is about
  to pay it. **The figure is decided there and arrives on the reward screen**, when the sentence
  naming it has been read.

---

## The tower

**8 floors × 3 fights.** Fixed layout, drawing no randomness — what is *in* it is random, the
shape is not.

### A floor is a motif and an element

**The tower picks one whole motif and one of the five elements per floor**, and the floor's three
rooms are three records of that motif dealt as that element. A fire goblin floor is three goblins
in fire — so what the player walked into is something they can plan against, rather than three
unrelated creatures who happen to share a corridor.

- **A motif is a file**, `data/motifs/<motif>.json`, holding every creature that can stand in one
  of its three rooms: outer chamber, inner chamber, stairway.
- **A motif is never fought twice in one run.** Each floor strikes its theme off before the next
  is rolled, so a climb is a tour of the roster rather than a shuffle of it.
- **A motif carries the band of floors it may theme**, at the file level rather than per record: a
  motif whose creatures were valid on floors 1 to 3 and whose boss was valid on 4 to 6 could never
  theme a floor at all.
- **Every chamber of every motif can be dealt at least two ways.** For the outer and inner rooms
  and each of the five elements there are at least two records that fit, so a chamber is a pool
  rather than a fixed set. **The stairway needs only one**, because a boss is a name rather than a
  room the climb fills: a motif may field one boss per element and each is the fight that floor is
  remembered by. The loader refuses a motif that cannot reach either figure — a floor the generator
  can offer and then fail to build is worse than one that never existed.
- **A record is dealt as exactly one element and its whole deck takes it.** There is no element
  anywhere on a creature's card: the colour belongs to the creature, the way a duelist's Jab is a
  concept that ships in five colours. Today the element marks the attacks and picks the picture,
  and `[?]` what else it should do — a status on hit, a resistance, something the floor does to the
  *player* — is open.
- **A record carries one picture per element it can be dealt as.** A fire goblin serf and an ice
  goblin serf are two drawings of one creature.

- **The stairway is the floor's third room and the boss is a record of the same motif.** It is a
  face the player can be told about, tiered above the two rooms below it and further along the
  ascent curve than either, but it is not a separate catalog: a goblin floor ends on a goblin.
- `[?]` **A boss has no advantage of its own yet.** What separates it from the creatures below it
  is its place on the curve and its own base stat line. One advantage per boss, drawn from a pool
  the record carries out of a closed vocabulary, is the decision still to make. See TODO.md.
- `[?]` Whether an inner-chamber creature should differ from an outer one by anything other than
  its place on the curve.

### The ascent curve

**Every fight is harder than the one before it, and the step is the fight rather than the floor.**
A creature's `HP` and `DMG` are **step-zero quantities** — what it is worth in the very first room
of the tower, whatever floor it is actually met on — and the curve puts it where it stands:

```
step = (floor - 1) * 3 + room          room: outer 0, inner 1, stairway 2
```

- **Two rates, not one.** `data/tower.json` holds `HPGrowth` and `DMGGrowth` in basis points, so
  how fast a creature's life outruns the player's damage is a separate dial from how fast its
  blows outrun the player's life.
- **Stepping per fight is what makes the ordering free.** A floor's boss is harder than its own
  inner chamber, and the next floor's outer chamber is harder than that boss, with no constraint
  between two separate numbers to get wrong.
- **So a late-band creature is not written as a high stat line.** It is written as the multiple of
  its neighbours it is meant to be. That also keeps the ratio between two motifs fixed however the
  curve is retuned.
- **`Actions` never scales.** Growing the budget would hand a high-floor creature more cards rather
  than a harder version of its own.
- **Nothing caps it.** The tower has a configured height and the climb wraps past it; the curve
  keeps counting, which is what makes the endless tower a number rather than a rewrite.
- **After fights 1 and 2: a choice of two doors.** After the boss: **a choice of stairwell.**
  Captured as two concepts even though the mechanic is likely the same, because one is "next
  fight on this floor" and the other is "next floor" — a real difference to hang divergence on.
- **Doors hint at what is behind them.** Cold coming off the door for an ice enemy, smoke for
  fire — the shape of what is coming without its name.
- **Generate both doors, always.** Rolling only the chosen one shifts every subsequent draw in
  the run.

### Life between fights, and what a stairway is worth

**A wound is carried from room to room, and only a boss takes it away.** A floor is an
attrition budget of three rooms. Opening every duel at full life would make damage a fact about
one round and never about the climb, and the only thing a bad fight would cost is the tenth of
life-left the payout pays.

- **Beating the floor's stairway protector heals to full**, and it is the only thing that does. No
  card, no relic and no room between fights returns life outside a duel. That is what makes the
  third room of a floor the one worth arriving at holding something back.
- **It also raises the ceiling by a third, compounding.** Each stairway is 33% more body than the
  run already had, not 33% of the body it started with — so a run standing on floor eight, seven
  stairways up, is carrying about seven and a half times the life it opened with. The ascent curve
  grows the opponent by 10% a *room*, which is a little over twice that across the same climb, so
  the two are pulling in the same direction and the boss bonus is the player's half of it.
- **The ceiling grows, the wound does not scale with it.** Forty points taken on floor two are
  forty points on floor six — worth much less against a bigger body, which is deliberate: the
  reward for climbing is that the early rooms of a floor stop being able to end you.
- **A defeat ends the run**, so nothing carries a wound past the bottom of the tower. There is no
  state where a run is alive and unable to start a fight; a wound deeper than the ceiling — only
  reachable by selling the relic that was holding the ceiling up — starts the fight on one life
  rather than on a corpse.
- **The rounding is down at every step.** A ceiling is a whole number of hit points, so 100 becomes
  133, then 176, then 234 rather than the 235 the arithmetic in the round would give.
- **The opponent is always whole.** A creature met in a room has not fought anybody; only the
  player carries anything between fights.

`session.Session` stores **the wound and the count of bosses beaten**, not a life total and a
ceiling. The ceiling is rebuilt from the record every fight and then moved by whatever is worn, so
a stored total would mean a different fraction of it the moment a relic changed hands, and a stored
multiplier is a second copy of a fact the count already carries. See `internal/session/life.go`.

### The ascent curve

**Every room grows the opponent's HP and DMG by 10%, compounding**.
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
- **Nothing caps the fight index.** The fight order is the whole roster standing in for a
  generator, so playing far enough asks for numbers the eight-floor tower never would.

**It doubles a curve that is already in the data, and that is deliberate but worth stating.**
`ValidFloors` sorts the roster from the weakest floor-one creature to the strongest at the top,
which is several times the climb on its own; the ascent curve multiplies on top of that. **What
that costs a player is unmeasured** — read the floor bands off `go run ./tools/motifsheet` rather
than from a figure written here.

`[?]` Whether the curve should be flatter now that it stacks on the roster's own progression, or
whether the roster should flatten instead and let the curve carry the climb.

`[?]` What distinguishes one stairwell from another. `[?]` Whether the shop and the door choice
are one screen or two, and in which order.

[ascend.go](internal/screens/ascend.go) is a stub whose comment already describes this.

---

## Enemies

**Every creature carries its own deck, and that is what makes it itself.** Each record under
`data/motifs/` holds a `Cards` array, written in the card language above — attacks named to the
creature, at that creature's own rungs, and **never fewer than three distinct concepts**. A Clear
Slime oozes, engulfs and dissolves.

**Two creatures of one motif hold different cards.** They are two different fights rather than one
fight at two weights, which is what makes a floor's pool worth having.

**Every creature deck is pure attack.** A creature raises no shields and blunts nothing, so its
whole personality is which blows come round how often: four cheap copies of one card is a swarm,
four expensive ones is a brute, a spiky deck with one big card in it is the one a shield hurts
most. The player learns a deck. `go run ./tools/motifsheet` is where one is read.

### Enemies do not form hands

**An enemy's attack cards resolve one at a time, in the order its planner chose them**
. Each lands its own hit at its own face damage; no hand is read, so
there is no multiplier and no hand off an enemy's turn. `Duelist.SoloAttacks`
carries it and `resolveSoloAttacks` is the phase.

**Hands are the player's axis and an enemy has no way into it.** A hand counts copies of a
*concept*, and every creature card is `FormNone`, so what an enemy "formed" was an accident of
what its planner could afford. Now
three cards on the table mean three hits, which is a round the player can read off the table
before pressing DUEL!.

- **The player's shields answer the whole turn.** A shield stands through the opposing turn and
  eats the heaviest hit in it rather than the first — see §Shields. Spending one on whichever
  card came round first would make it worth least against exactly the opponents that swing most.
- **A shock rolls once per hit**, exactly as it does against a hand-forming attacker — see
  *Lightning is a roll*.
- **It is a flag on the duelist, never a rule about side B.** The engine has no idea which side is a
  person and must not learn — a headless simulation plays both sides.
- **`[?]` Whether a boss or an affix can give an enemy hands back.** The flag is per duelist, so
  nothing in the rules forbids it.

### One planner

`PlanFor(duelist, hand)` **scores every affordable combination of the hand's attacks**, and
takes the best. It is exhaustive rather than greedy because a greedy pass cannot see that three
Ooze forming a Three of a Kind beat one Dissolve — a hand is at most seven cards, so this is 128
candidates.

**A hand-forming duelist is scored through the same `blowFor` the resolver uses**, so the plan
it plays is the plan the engine will score. **A solo attacker is scored as the sum of what it
picked**, which is the same arithmetic its phase performs — so the search is looking for the
most damage the budget buys rather than for the best combination.

**Then it spends what the attacks did not want**, in the hand's own order. That second pass is
what keeps a card a damage-maximizing pass would never reach from being dead content.

**`Copies` is the difficulty dial and it is a blunt one.** With no hand to form, four copies of a
1 AP card is four small hits, so the dial is simply *how many cards a turn holds*. The sharper
one is **variety**: a creature with three different attacks lands all three, and how spiky that
set is decides how much one shield takes off the turn.

### The deep tower is meant to need a build

Per-enemy decks, the roster's own HP curve and the 10% ascent curve compound, and none of them
is absorbed by a retune — so the deep floors are out of reach of a duelist wearing nothing.
**That is the intent rather than a regression**: the
player's ceiling is *supposed* to move and relics are how, so a bare fighter is not who those floors
are priced against and **the whole ascension is not expected to be winnable yet**.

**A wall on a *shallow* floor is a different thing**, and is still a failure — the player has bought
nothing by then.

### The count bound is the rules', not the screen's

`Duelist.MaxActions()` is the cap, and **the opponent's planner obeys it exactly as the player's
selection does**. A cap enforced only by the screen is a cap the enemy ignores.

---

## The profile — what survives a run

**A run dies; a profile does not.** The tower is the run — the deck, the purse, the worn relics, the
room you are in — and the profile is the thin layer that outlives it: whether the tutorial has been
watched, what has been achieved, what has been unlocked, **and what the player has chosen about the
program** — how loud the score is and how fast the game moves. Standard roguelike shape, and the
reason the two are separate files on disk rather than one.

**Two files, in the platform's config directory** — `profile.json` and
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
player who loses room one fifty times gets it on the fifty-first. It is one record in
`data/achievements.json` — see the next section.

**A run is saved at every phase transition and never inside a duel**.
Between stations the run is quiescent — no piles dealt, no queued actions, no hidden hand — so the
snapshot is a dozen fields rather than the whole combat screen's working state. The cost is stated
rather than discovered: quitting mid-duel loses that duel and resumes at the top of the room.

### The title screen, and giving up

**The game boots to a menu again, and the menu is where a run is decided.** It booted straight into
a duel for as long as the combat screen was the thing under construction, and a run existed before
anybody had been asked anything — which was fine until there was a question worth asking. There is
now: **New Run** or **Continue**.

- **Continue is dead when there is nothing to continue**, rather than absent. A menu whose entries
  come and go between launches is one that has to be re-read every time.
- **New Run asks first, and only when it would destroy something.** A player on a clean install gets
  a run; a player forty rooms up gets a question.
- **A pinned seed is not rerolled by New Run.** A pin is a debugging session where the same tower in
  the same order is the whole point, and a menu button must not undo what the source set. The
  tutorial still outranks both — a taught run is dealt the script's code, because the lesson
  promises the player the hand they are holding.

**Abandon Run is how a climb ends early**, on the settings screen, below a rule, in the
destructive red, behind a confirm. Without it there is no way to give up — quitting means
closing the window, and the next launch resumes exactly where it left off. It is filed under the
program's screen rather than given a corner of its own because that is the one screen reachable
from everywhere, which is what a "give up" control has to be — and the placement is why it is
set so far apart from the two bars.

**A death ends the run, and there is no retry.** A defeat that put the same opponent straight
back up is not what a roguelike is. The duelist falling ends the climb exactly as giving up does
— same function, same deleted file, same trip back to the title — and the button in the DUEL!
slot says **End Run** rather than offering a choice. The press is the player deciding they have
looked long enough, not a decision about whether to die; the screen holds its last picture until
they make it.

**The run's ledger is readable on that screen before the button is pressed**, since it is chrome and
the run still exists until the press. That is the account of the climb that just ended, which is the
one moment it is most worth having.

### The end-of-run splash

**A finished run gets a page saying what it came to**, before the player is put back on the title —
whether it ended in a death or was given up from the settings screen. Both are the same event and
both land here; the only difference is the sentence at the top.

What it shows: **floor reached, enemies defeated, damage dealt** — the three the run is judged on —
then rooms entered, rounds fought and unspent vitae, quieter. And **the run code, in a box, at three
times the size of anything else on the page.**

**The seed is the reason the page earns its place.** Everything else is a number about a run that is
over; the code is the one thing still useful afterwards — it deals the whole tower again, and it is
how a run can be handed to somebody or named in a bug report. Before this it went to the log at
launch and nowhere a player could ever see.

**The code is also in the bottom-right corner of the settings screen**, quiet and captioned, drawn
only while a run is in progress. The splash is a page you see once and only after the fact; the cog
is on every screen, so that corner is where the answer is always two clicks away. Nothing is drawn
there with no run standing, because the code would name a tower the next New Run is about to reroll.

**There is no "run it again" button, and there should not be.** The seed being on screen is what
makes a run repeatable; a button that dealt it again would be a retry with a longer name.

**The summary is taken before the run is destroyed** and held on the global state, so the splash
draws numbers rather than holding a finished run open — a `Session` still alive behind that page
would be a run that is over and still resumable.

### Two menu screens

**Achievements** and **Credits** hang off the title menu. Neither is a station of a run: they read
the profile and a list of names, they never touch `session.Phase`, and each records where the player
came from so Back works from anywhere — the same shape the settings screen has.

- **Achievements lists what has not been earned as well as what has**, grayed. A page showing
  only what you already have says nothing on the day you most want to read it.
- **Credits names both copyright holders, the two dependencies, the portrait source and the
  license.** It is partly a page to read and partly a page to be *correct*: the project is
  source-available and meant to be sold, so the attribution has to exist somewhere the player can
  see it.

Both are deliberately simple. When either outgrows the screen it takes a `models.Scrollbar`,
which already exists.

**Resuming is not replaying, and the distinction is load-bearing.** A run is *not* replayable from
its seed alone — a deck edit is a choice, so replay would need a seed plus a choice log — but
resuming does not need the path, only the state. So the snapshot is state, and it may never be used
as a replay. See the Randomness section below, which is unchanged by this.

**The climb is not saved; it is rebuilt from the run code.** The fight order is a function of
the run seed, so storing the seed keeps one answer to "who stands in room four". **This stops
being true the day the room-choice screen lets a player pick what is ahead**, at which point the
climb becomes a choice and has to be written down. `TestTheClimbIsRebuiltFromTheSeed` is what
fails on that day.

**Nothing about the profile is ever fatal.** A missing file is a new player, a corrupt file is a
new player, an unwritable directory is a session whose progress is not recorded — the same rule
the audio device is under. A game that refused to launch over a save file would be a worse bug
than any it could prevent. A file written by a newer build is read but never written over, which
is the one mistake that cannot be repaired afterwards.

**What is written down is a name, never a number.** Every ordinal in this game is append-only and
index-shaped — `ConceptID`, `Element`, `StatusID`, `Phase` — so an ordinal in a file
that outlives its build is an ordinal that will eventually mean something else. **The stones are the
newest case**: a run's raised rungs are saved by hand *key*, never by the seat the
count actually sits in, because a seat is a position in the catalog this build happened to load.

### Settings

**Two numbers, on the profile rather than in a file of their own**: `musicVolume`, 0 to 1, and
`speed`, a multiplier between 0.5x and 2x on the game's one clock. The profile is already the
per-user file the game writes, and a second one would double the migration policy above for two
numbers.

**They live on a settings screen reached from a cog in the corner of every screen.** Both are
drag bars — CLAUDE.md's rule is that a settings value is a row
of buttons or a slider and never a number typed in, and a bar is the honest shape for "anywhere
along here".

**There is no mute, and zero on the music bar is the only silence there is.** A latch and a bar
are two controls over one number that then have to be kept from disagreeing. What the corner
gives up is one-click silence; what it buys is somewhere to put the game speed.
**A new player still boots silent** — a fresh profile is at zero, because music that begins on its
own is the first thing a new player reaches for a control to stop.

**The speed setting may never change an outcome, and that is what makes it safe to offer.** A whole
round is resolved before playback begins, so scaling the beat moves pictures and nothing else — the
same constraint the debug flags, `internal/trace` and the scripted demo are under. It scales
`clock.go`'s single beat, which every duration in the game is a fraction of; a duration written as a
raw number rather than as a fraction of it is one the setting cannot reach, which is what
`TestNoClockIsWrittenAsARawNumber` has been guarding since before there was a setting.

**A sounds bar is expected and is deliberately not there yet.** There is no sound system, and a
slider setting a number nothing reads would be a control that lies about what it does.

## Achievements

**An achievement is a record of something the player did, and it changes nothing.** That is the
line `internal/profile` has drawn since it was written: an *unlock* is an input to the rules and
something in the rules reads it; an achievement is a note. **A relic behind an achievement
therefore reads the unlock, never the award** — the record *grants* an unlock key, which is two
keys rather than one, and that is what lets an achievement be reworded or retired without
orphaning the thing it opened. Nothing is gated on one yet; the bridge is a field on the record
so the day one is, it is a line of JSON.

**The catalog is `data/achievements.json`**, and it is a loader rather than a Go table because a
record has to say *what earns it* — a name and a sentence would not have needed one.

### Three kinds of trigger, because there are three kinds of achievement

The list looks heterogeneous and is not. It is three families, and only one of them is situational:

- **A turn shape** — what the player put on the table together. Spectrum (four elements at once),
  Elementalist (five), Weaponmaster (three attack forms), Arsenal (three attack forms and a
  defense), Prism (one form or one card, in all five colors), and the two ends of the cost ladder
  — Tiny But Fierce (five free attacks of one card) and Godslayer (five 4 AP ones).
  **This family is pure grammar**, and it is where a shape the hand ladder cannot *price* belongs:
  worth naming, not worth paying for.
- **A lifetime count** — three hundred slashing cards, two hundred Bashes. A tally on the profile,
  not a predicate over anything the process is holding.
- **A named moment** — a duel won, the tutorial finished, the fifth floor reached, a card altered
  into a Flinch, ten shields standing at once. The only family that costs a line of Go each, and
  deliberately the short one.

**The turn family reads the turn, not the hand.** A hand counts the cards that scored it and leaves
the rest out; these are about what was played together, which is why Arsenal can ask for a defense
beside three attack forms — something no rung on the ladder can say, because a hand can only state
what its cards must *agree* on.

**Thresholds are at-least, everywhere**. A five-element turn earns Spectrum as well
as Elementalist, arriving on floor six earns the fifth-floor row, and standing behind eleven shields
earns the row that asked for ten. The alternative makes a player who jumped a step permanently miss
it, which reads as a bug in the page.

### A clause may filter on cost

**`Cost` narrows a clause's selection to cards of exactly that AP**, on top of `Of`, and it exists
for the two ends of the promote ladder. Every attack ladder in `duelist_cards.json` runs five rungs
from 0 AP to 4 AP with the two ends shipped at zero copies, so a run reaches them only by promoting
or demoting — which makes "five 4 AP attacks of one card" a statement about a deck the player
*built*, and the reason those two achievements are worth naming at all.

**It reads `Card.Cost()` and not the concept's figure.** An essence's `CostDelta` is part of
what the turn cost, so five Skewers an Exalt pushed to 4 AP count and five Impales a Hone made
cheap do not. The achievement is about what was paid.

**Zero is a filter and not an absence**, which is why the field is a pointer in the JSON struct —
the free rung is exactly the thing Tiny But Fierce is about, and an int could not say "the free
ones" without also saying "any".

### Two of them are unreachable today, and that is deliberate

**Godslayer is five 4 AP cards against a six AP budget** — twenty points out of six. Nothing in the
game grants AP on that scale, so it cannot be earned until something does; a brand is the likeliest
door, since brands alter the container. It is written now because the *pattern* is what a brand
would have to be judged against.

**This is the one place the reachability rule below is knowingly bent**, and it is bent in the half
that is safe: `TestEveryShippedAchievementIsReachable` proves the pattern is satisfiable by some
turn of five cards, which is what stops a row that no turn could ever match at any price. It says
nothing about the budget, and the test now says so out loud.

**Tiny But Fierce is reachable, and is the pair to it.** Five Pokes cost nothing at all, and five
cards is exactly `MaxActions` — so it is the turn that hits the *count* bound rather than the
budget, which is the shift Flinch and `minCardCost` both made. Getting there is five Shrinks over a
run, which is a long grind and not a special case.

**Invulnerable needed the duelist shield cap lifted** — see §Shields, where the clamp that made ten
impossible came off on the same day.

### What a record says, and what it says twice

**Two pieces of prose, in opposite tenses.** `How` is what you must do and is legible while the row
is still locked — which is the entire reason the achievements page lists what has not been earned.
`Said` is what the game says once it has happened. One line would have to be both.

**Every `Said` line is shown at once.** Picking one of several would be a roll, and a roll owes its
own salted stream and its own argument; several lines together cost neither.

### The toast, and when the tallies are settled

**An achievement announces itself and is clicked out of.** Nothing announced one until now — the
record was visible on a screen the player had to think to open. It is the confirm dialog's shape
with one answer instead of two: a small centered box, on confirm.go's argument that a notice the
size of a page reads as something having gone wrong. **It is drawn by the frame rather than by a
scene**, because an achievement can land during a duel, on the post-battle screen, or on the
transition between them. Like every dialog it freezes pacing and cannot change an outcome.

**The queue is drained one box at a time.** A five-element turn earns three achievements together,
and a single box would have to pick one name to put at the top.

**Counters are held in memory and settled when a duel ends**. The alternative was a
disk write per card played, against a file the rest of the game writes at a handful of named
moments. What that costs is stated rather than discovered: a crash mid-duel loses that duel's
tallies and nothing else. **A lost duel settles them too** — what a defeat costs is the climb, not
the record of what was swung on the way up.

**Counters are per concept and per form, and never per concept and element**. Five
colors of twelve concepts is sixty tallies to say what twelve say. Both axes are counted because
the two questions are genuinely different: "how many slashing cards" is the form and "how many
Bashes" is the concept.

### The failure this shape exists to prevent

**An achievement nobody can earn looks exactly like one nobody has earned yet.** No amount of
playing tells the two apart, and nothing fails. So every word a record may write is a closed
vocabulary refused at load: a trigger kind, a clause mode, an axis, a moment name, and a counter
name — the last checked against the forms and the player's own concepts, because a misspelled tally
is a row that sits locked forever while the game plays happily on. `internal/achieve` is where all
of that is refused, and `TestEveryShippedAchievementIsReachable` builds an actual turn for every
pattern in the file.


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

## The ledger — the run's account of itself

**Every fight of the run, kept and readable, with the working under every blow.** It replaced the
fight log, which held one fight, was thrown away by the next `Init`, and had no way to reach its
own earlier rows.

What it is for is two questions at two scales, and the design answers both on one panel:

- **"What just happened?"** — every hit written as the arithmetic it was: each landing, what the card was
  worth, which relic multiplied it and which relic bought the extra landing, and what the hand's
  multiplier did to it, and what became of it. The hand dialog acts this out while the hits land and
  then it is gone; the ledger is where it keeps.
- **"How did my run go, and where did it go wrong?"** — every fight as one line: floor, opponent,
  won or lost, in how many rounds, for how much damage. Clicking one opens it.

The rules that hold it up:

- **The fight in progress is expanded; every finished fight is folded to its heading.** A
  thirty-fight climb is thousands of lines and a panel opening onto all of them is a scrollbar
  with nothing to aim at.
- **A round is written down when it ends, never while it plays.** So the panel cannot show a round
  the player is still watching, which is the concealment rule it never needed to be given.
- **It stores sentences, not events.** A `combat.Event` is about 2.5 KB — three 25x5 arrays,
  almost all zero on anything that is not a hand — so a run of them would be megabytes to say what
  a few hundred kilobytes of prose says. The events are still the source and the words are the
  same ones the log always wrote.
- **It says nothing the events do not.** Every figure comes off the `KindHand` event, which is why
  those fields are on the event at all; the panel is a second *drawing* of one event and never a
  second arithmetic.
- **It is saved with the run**, so "how did my whole climb go" survives a resume. It is the one
  part of a snapshot that is prose rather than state, and therefore the one part that may never
  fail a resume: an unrecognized voice draws plain rather than costing the player the run.
- **It is reachable everywhere**, from a button beside the cog rather than from the combat screen.
  It is chrome, not a screen — see CLAUDE.md, and note that a screen could not have done it:
  navigating away from a duel and back re-deals it.
- **It is colored like the screen it accounts for**: a figure in its card's element, a relic's
  multiplier in the relic pink, a verb in its category's color. The hand itself is *marked* rather
  than colored — bold and underlined — because hue belongs to the elements and there is none left
  that is not a near-collision. See CLAUDE.md.
- **Scrolling is a dragged scrollbar**. The input vocabulary is clicks, drags and
  hover; the wheel stays out of the game.

### The gap between two fights is part of the account

**Under each fight's rounds sits an `After` block: what the player did with what that fight paid
them.** The card taken and the card cut on the reward screen, the essence spent on one of them, the
relic bought or sold, the stone spent and the rung it raised, and what the purse gained and paid.

- **It hangs off the fight rather than sitting between two of them.** The panel folds by fight, so
  a block of its own would be a third thing to fold and a heading belonging to no record. Filed
  under the duel it followed, it opens and closes with it — which is how the player remembers it:
  *after fight one*.
- **It is read off the run rather than announced by the screens.** A call beside every commit is a
  list a new mechanic gets left off silently, since a missing announcement and a deliberate silence
  are the same absence. The run is watched instead and whatever moved is worded — the rule
  `internal/screens/combat_handmorph.go` already follows, where what a rune did is read off the
  card faces so that no rune has a case in the drawing.
- **Cards are told apart by `combat.Card.ID`.** That is what makes "this card became that one"
  sayable at all: counting faces cannot distinguish a card altered from a card cut with another
  taken in its place, and would report one act as two lines.
- **Nothing is recorded during a fight.** A rune spent mid-round is an event of that round and
  belongs to the round's own lines.

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

Every `[?]` above, in one place. Each is a decision nobody has taken, not a thing nobody has
written down.

**Balance, and the reason all of it is open:** *nothing in this repo simulates a duel.* An
unwinnable floor looks exactly like a run of bad draws, no test goes red, and every price and
every stat below is therefore a judgment.

- `[?]` **Nothing has measured the roster against a hit per card.** Enemy HP and DMG are tuned
  by hand, and every flat bonus, status, drain and shock now scales with how many hits a turn
  throws.
- `[?]` **How enemies scale up the tower.** Records carry no level term; the ascent curve is the
  only thing that scales one.
- `[?]` **Whether the ascent curve should be flatter**, now that it compounds on top of the
  roster's own progression — or whether the roster should flatten and let the curve carry it.
- `[?]` **Whether a Pair relic is priced right.** A turn satisfies every rung its cards reach, so
  the Pair family is near-unconditional while still being priced as a conditional one.

**Mechanics that exist and are unfinished:**

- `[?]` **Hand discovery is not enforced.** `profile.Profile.HandsDiscovered` is the field waiting
  for it and every rung is live.
- `[?]` **Almost every relic is about attacking.** The one defensive mechanic is among the game's
  strongest and no verb reaches it; filling the gap means a verb that raises, keeps or spends a
  shield.
- `[?]` **Where a run acquires a *carried* essence** — a shop seat, a sealed good, a reward that
  offers keep-or-spend. Only a fixture puts one in the satchel today.
- `[?]` **Brands are designed and unbuilt**, beyond the container/contents axis and the rule that
  none of them may grant actions.
- `[?]` **Long press is unbuilt**, and it is the whole of the touchscreen and controller story for
  the reveal hover already gives.
- `[?]` **Enemy statuses are blocked on affixes, which do not exist.** Every creature card is
  authored `basic`, so the element system runs in one direction only.

**Tower and rooms:**

- `[?]` **What distinguishes one stairwell from another.**
- `[?]` **Whether the shop and the door choice are one screen or two**, and in which order.
- `[?]` **Whether a boss carries an affix by default**, and whether several bosses on one floor
  should differ in shape rather than only in name and picture.
- `[?]` **Whether a boss or an affix may give an enemy hands back.** The flag is per duelist, so
  nothing in the rules forbids it.

**Determinism and color:**

- `[?]` **The shock roll is conditional**, against a written rule that it should be
  unconditional — the stream only advances when lightning is in play, so a balance tweak moves
  every later roll. Settle it before the save format depends on a stored seed.
- `[?]` **Earth's green sits next to the player's green swatch.** One of the two schemes has to
  give; what holds it off is that a border and a swatch are never seen side by side.
- `[?]` **Eight of the ten upgrade inks are placeholders** on a wheel with no hue left to claim.
- `[?]` **The relic art is pixel art and the other catalogs are not.** Closing it means
  regenerating one side or the other.
