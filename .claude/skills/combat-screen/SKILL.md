---
name: combat-screen
description: The combat screen's layout, its card/action box widget, hidden information, and the resolution-order rule the screen must obey. Load before touching any of internal/screens/combat.go, combat_deck.go, combat_hud.go, combat_actionbox.go, internal/combat, the hand, the cards, the deck overlay, the ledger, the character block, or anything about how a round is drawn or played back.
---

# The combat screen

**The drawing layer split off on 2026-09-17.** Everything the combat screen draws *through* — the
table, the clock, the movers, the card faces, the panels belonging to no screen, the prose — is
`internal/ui` now, and the scene itself is still `internal/screens`. **A symbol named anywhere below
may be in either**, and many are exported now where this file writes them in lower case; grep both
packages rather than trusting the spelling here. The rule the boundary states is that nothing in
`internal/ui` may reach into a scene.


The screen under active construction. Everything here is a decision already made — read before
proposing a change to one, and see `TODO.md` for what is still open.

**The game boots to the title screen**, and has since 2026-09-03; this file used to say the opposite,
from the months when booting straight into a duel was right because the duel was the only screen.

**The rules live in `internal/combat` and the screen only replays them.** `ResolveRound`
decides a whole round before a frame of playback runs. Never change those rules to make a
screen look right — if a screen contradicts the engine, say so and let the owner decide
which one is wrong. That is a game-design call and it ripples into the tests and the
balance.

**What is not here and still applies.** `CLAUDE.md` holds the rules this screen is built on
top of, and they are not repeated below:

- **The input vocabulary** — left click, drag and drop, long press. No right click, ever,
  and no keyboard outside the one seed field.
- **All interface art is authored** *(2026-09-16)*. The silhouette generator in
  `internal/systems` is deleted; a picture is a PNG fetched by asset key through
  `systems.ArtMark`, which reduces a 256px source to whatever size is asked for. The form marks
  and the cost ticks are one drawing per form per element plus a neutral set — see
  `cards.MarkArtKey`, which is total, so a blank corner is a missing file rather than a card with
  no element. Nothing is tinted: the hue is in the drawing.

- **The card is drawn by `internal/cards`, not by this screen.** `drawCard` builds a
  `cards.Spec`, pulls a cached image and blits it; it draws nothing itself. Change how a
  card looks there, then `go run ./tools/cardsheet` and refresh the tab — the tool and the
  screen run the same code, so the sheet cannot lie.
- **Color: name one color and scale it** with `systems.ColorAtStrength` — but that scales
  toward *black*, so on the off-white card surface it makes things louder, not quieter. Use
  `systems.ColorToward` against a light ground. The card's element is its **border** now,
  not its surface.
- **Widgets are hand-rolled**, `models` struct plus `systems.Update*`/`Draw*`. No toolkit.
- **Determinism.** Three of the screen's sources are seeded in `Init` and all three come off
  `RunSeed`: the deck's `rng` and the opponent's `enemyPile` from `shuffleSeeds` (salted per
  side and per fight), and `combatRNG` for the engine's lightning roll from
  `RunSeed ^ combatSalt`. Separate streams, and they must stay separate. `deckSeed` pins the
  two shuffles for debugging — both of them, never one.
- **Which of the screen's files holds what** — the map is the package doc,
  `go doc ./internal/screens`, and it lives beside the code rather than here so there is only
  one of it. Everything below describes the screen, not a file; they are one package, so a symbol
  named here may sit in any of them and `Grep` over `internal/screens/` is the way to find it.

## Resolution order

**Phases.** A round is **a whole turn each**: everything side A
queued resolves before side B does anything, and within a turn the categories go in order —
**the defenses, then the attacks**; `combat.Categories` is the order. A shield raised anywhere in
your turn is standing through the opponent's whole turn either way, since `expireDefenses` runs at
the start of your *own* next turn — so what the order buys is how the turn reads: raise the guard,
then swing.

**The attack phase is one hand and a hit per landing.** Every attack card queued is announced with
a `KindAction`, then one `KindHand` names the hand they formed and carries every hit's arithmetic,
then each hit lands as its own `KindMissed`, `KindBlocked` or `KindDamage`, carrying `Slot` (the
card) and `Hit` (its term on the hand event). Five Bashes are five hits under one Four of a Kind.

**The run's account writes the hand once and a line per hit under it.** The `KindAction`s still
play — each is a beat, and each raises its card on the table — but they write no record. Each hit's
line carries its own working, the identity multiplier included *(owner's call)*, because hands are
going to be upgradable and a term that appeared only once the multiplier stopped being 1 would
make an upgrade read as a new rule; the hit's outcome attaches to that line by its `Hit`, whatever
was written after it. **Every card the turn played throws a hit** *(owner's call)*, whether
or not it made the rung. See MECHANICS.md §Damage: a hit per card, one multiplier.

**None of the paragraphs above describes an enemy's turn.** `Duelist.SoloAttacks` makes attack
cards resolve one at a time in queue order, each landing its own hit at face damage, and **no
`KindHand` is emitted at all** — so its damage events fly one at a time out of the lit card, through
`noteHit`, rather than out of a line of arithmetic. `CombatScene.soloAttacker(side)` is the screen's single predicate
for it and two things read it:

- **The log writes a sentence per attack card**, because there is no phase line coming to carry
  them. `logRows` suppresses an attack's `KindAction` only when a hand *is* coming.
- **The table lights one card at a time, and a creature is the only thing that lights one at all.**
  `noteResolved` seats `[]int{seat}` for a solo attacker and nothing for anybody else — a duelist's
  cards are raised by the hand's announcement, which a creature never gets. Raising a set says
  "these cards made the hand", which an enemy's turn has none of: three cards swing three times,
  and the card that is up is the card that is hitting.

Everything else about playback is unchanged — one `KindAction` per slot, so `currentSlot` still
counts beats the same way.

Phases replaced alternation, which replaced volley-per-side. The reason is
legibility: interleaving may simply not be graspable by players. See
[MECHANICS.md](../../../MECHANICS.md).

**Initiative is gone**, whole — the method, the constants, the tie-break, the clock glyph and
the `i3` in the concealed enemy label. With one contiguous turn per side there is no exchange
for a faster action to lead. `Spd` still buys action points and still never buys priority.

### What this means for the screen

- **Dragging a card changes nothing the engine can see** *(2026-08-14)*. Cross-category
  reordering never did — the drag lands the card in a queue that is then regrouped — and the two
  things that read *within*-category order are both gone: hands are counted, so a turn is a set,
  and shields are ranked by the hit they would eat rather than by when they went up. **What order still
  decides is the hand's tie-break** — `groupsOf` breaks a tie by whose first card was played first,
  so the lead card that names the hand and carries its element is chosen by where the player put it.
  Do not paper over the rest on the screen, and do not invent a rule to justify it.
- **`Duelist.Shields` is a count per element, not a queue.** Which hits they eat is decided at the
  top of the attacker's turn by `combat.shieldedHits` — each shield its own element's heaviest hit
  first, then the heaviest of what is left — so the order the shields went up in reaches no outcome.
  **The element is a rule**: a matched block banks an action point (`Event.Surged`,
  `Duelist.Surge`), so a `KindBlocked` carries **the shield's** element in `Element` and the row
  spends that pip (`ui.ShieldRow.Spend`). **Nothing reduces a hit to zero by arithmetic** — a
  shield eats a whole hit or it does not, and a fizzle (`KindFizzled`, a hit of a creature's own
  element) wastes a whole hit rather than zeroing a figure. It is drawn like a miss: `FIZZLE` on
  the hit's line.
- **`Slot.Index` is not a position in the round.** It is where the card sits in its own
  side's queue, which regrouping breaks apart. Anything asking "how far through the round are
  we" counts slots — `CombatScene.currentSlot` does, and lighting the right Resolution row
  depends on it.
- **The category is deliberately not concealed** on enemy rows. It is what decides where a row
  sits, so withholding it would make the pane unreadable rather than merely uncertain. It took
  over that job from the initiative number — see `concealedLabel`.
- **The card shows its form as a corner mark** — a spear, a sword, an axe, a shield — in the
  top-left corner above the cost, tinted by the card's element. A form is not a quantity, which is
  why it is not a numbered badge; a 32-pixel drawing is read before any text is.
- **Cost is a stack of dash marks**, under the category glyph, one per point. Not a numeral
  and not a badge. Costs run 1..4 and a fifth tier is a layout change, not a bigger number.

### Hands, and the one thing they changed on this screen

The catalog is `data/hands.json`, the matcher is `internal/combat/hand.go`, and the design is
in `MECHANICS.md`; these are what matter to the screen.

- **A hand is a *hand*, and it is a damage multiplier and nothing else** *(2026-08-17)*.
  `Event.Hand` is a `HandID` and `Event.Multiplier` the percent. `handName` in `prose.go`
  looks it up with `HandByID` and prints `Hand.Name` — "Two Pair" — **assembling nothing**. It used
  to join a hand to a *mix* counting the distinct colors, and to fill a `{card}` template from the
  concept that formed the hand, so one hand could print as "Duo Bash Flurry"; both axes are gone
  and a hand carries its whole name. **Exactly one fires per turn**, so there is no stacking to draw
  and no ranking to explain.
- **`Event.Hand` always names a hand** *(corrected 2026-08-19)*. A turn with an attack in it falls
  back to the catalog's `no-hand`, so `HandNone` never reaches a `KindHand` — the log had a
  branch written against the opposite belief and it had been unreachable for some time. **The High
  Card takes the hand line like any other hand**, and carries its `x 1` in both the line and the
  dialog since 2026-08-19 — **the last place it was written differently from the rest**. What is
  left that treats it specially is `matchHand` reaching it by fallback rather than by counting,
  which is structural: counting would match the one-card hand against every turn in the game, and
  the fallback picks the hardest-hitting card rather than the commonest.
- **The event carries every hit's arithmetic, and the engine lands the same figures.**
  `HitAmounts[i]` is hit i before the attacker's weight and the target's vulnerability, and the
  hit's own `KindDamage` is that figure after them — so the figure a line prints and the figure the
  hit landed cannot come from two different sums. `Amount` on the hand event is every hit added up.
  A shield does not appear in that gap at all: it removes a whole hit up front, and `KindBlocked` is
  what says so.
- **The multiplier multiplies the cards, and there is no third term** *(owner's call)*. `no-hand`
  therefore sits at `100` rather than `0`: a multiplier applied to the cards cannot be zero without
  deleting every hit. See MECHANICS.md.
- **`Event.HandAmounts` is what each hit's card deals**, parallel to `HandCards`. It exists so a
  hit's line can show the card's term without the screen owning `CardDamage`, the strength scaling
  and every relic that touches a card's damage — which would be a second resolver.
- **A fired hand keeps its own cards raised, and the list comes from the event**.
  `Event.RungCards[:RungCardCount]` names which cards of the turn formed it. **Never derive
  that from the hand's group sizes, and never assume the cards are adjacent.** A counted hand is
  not contiguous — Two Pair is two cards, a card that earned nothing, and two more — which is why
  the event carries a list rather than a start and a length. `noteHand` narrows the raised set on
  whichever row it belongs to, so the opponent's hand says it the same way.
  **The yellow bracket round those cards is gone** *(2026-08-19, owner's call)*, in the hand row
  and on both table rows. What is left saying which cards earned the hand is the lift, and in the
  hand row nothing says it at all — the name is the whole of the feedback while planning. That is
  the trade to know about before adding a third way to mark a set of cards.
- **`combat.BlowFor` previews the hand while the player plans** *(2026-08-15)*. It is the same
  function the resolver uses, so a previewed hand is the hand that fires by construction rather
  than by two pieces of code agreeing. `previewAttack` calls it on `ResolutionOrder(queue, nil)`
  and **every attack previews, the No Hand included**. A single attack card is a rung — the
  catalog's `no-hand` at the identity multiplier — so the name is on screen from the first attack
  picked, and it names the turn honestly rather than dressing it as an achievement. A queue of
  nothing but shields names nothing, `BlowFor` returning a blow with no cards.
  **Two lines show it**: `drawPlannedHand` writes its
  name across the middle of the table, breathing, in `handNameInk`, with **what it is worth on a
  second line under it** — `1.15x DMG` — and that pair is what flies down to the hand row at DUEL!,
  so the preview and the announcement are one object. See `handBanner`.
  **`Blow.Cards` indexes the turn, not the hand**, which is why the preview goes through
  `ResolutionOrder` — a Brace queued first resolves last, so a preview read off the hand as the
  player left it would miss the hand behind it.
- **A chilled slot is a row that never resolves.** `currentSlot` counts `KindChilled`
  alongside `KindAction` for exactly this reason — one beat per slot, taken or lost — and
  `TestEverySlotIsEitherTakenOrChilled` pins it. **The pane still draws that row as though it
  happened**, which is a known gap. Ice is the only thing that can take a slot.

### The hand dialog: every hit worked out under its card

*`combat_mathbox.go`.* On the beat a hand fires, **every hit's arithmetic is played out at once**, a
line under each card on the table, defenses included: the DMG flying off the duelist card, the card's own
multiplier off the card, each relic's factor off its relic, the hand's multiplier
off the banner, then the hit's figure. A card that lands several times — an echo, a form repeat —
stacks its lines under itself. Every line but the first carries its `+` on the left rather than on
a row of its own *(owner's call)*.

**The lines run in parallel and do not wait for each other** *(owner's call)*. Within a line the
items still arrive one at a time; a line with fewer terms finishes first and is **thrown** the
moment it does — see `throwColumn` below. A toggle for how this shows is wanted later and does not
exist.

- **It says nothing the event does not carry and computes nothing.** Every figure comes off the
  `KindHand` event — `HandAmounts`, `HitAmounts`, `Multiplier`. **This is the rule to hold**: a
  second *drawing* of one event, never a second arithmetic. A figure it needs and the event has
  not got is a field that goes on the event.
- **What became of each hit is read ahead in the log.** `hitOutcomes` walks forward from the hand
  event over the hits it threw and hands each line its own outcome's position. A finished line
  that **landed** flies its figure into the target (`throwColumn` moves the target's life at that
  moment and marks the event walked), a **miss** says MISS over the line, a **block** says BLOCKED,
  and a line with no outcome — the target fell to an earlier hit — fades where it stands.
  **This is the one place the screen reads ahead of the cursor besides the shield break**, and it
  is confined to the hits directly after the hand.
- **Events it has already shown are walked past.** `combatTheater.walked` holds the log positions
  a line drew, and `advancePlayback` steps over them without a beat, so a hit is never drawn twice.
  It is cleared at `startRound`, because a position means nothing in the next round's log.
- **It is the one thing on this screen that can stop the playback cursor.** `advancePlayback` holds
  while `mathBox.Running()`, and the flights it throws hold the cursor after it. **It still cannot
  change an outcome** — the round was decided before a frame of it was drawn — but it *does* change
  pacing, and `demoGiveUpAt` is sized against the dwell alone, so a longer script is worth checking
  against it.
- **`hitScript(e, i, first)` is the half with no geometry in it**, and it is what
  `combat_mathbox_test.go` pins: the strings, their order, which items fly, and that each line ends
  with the event's own figure for that hit. The tests create no `ebiten.Image`.
- **Each line is laid out once, before anything is shown**, and revealed left to right into space
  already claimed, centered on its card's column, a `mathLinePitch` per line below the card's bottom
  edge. **Nothing is shrunk** *(owner's call)*: at full type size neighboring lines overlap and a
  long one runs off the screen, which is expected and is the next thing to design.
- **Shakes and signals are read off what started this frame**, not off one cursor. `takeShakes` and
  `takeSignalSeats` each drain their own list of items the box started, because two readers drain
  them on different schedules and several lines start on one frame.
- **Every rung the engine names is shouted, `NO HAND!` included.** The bottom of the ladder is a
  rung like any other — a multiplier, a stone that raises it, a relic that names it — and the word
  says what the turn was. **What the No Hand does not get is the lift**: the announcement raises the
  cards that *made* the rung and that turn made none, so `builtARung` gates the raise and nothing
  else. A card standing up for having done nothing is worse than a still table.
  An event naming *no* hand is silent; nothing emits one.
- **The hand's name is one word with two homes, and it travels between them** *(2026-08-19,
  owner's call)*. `handBanner` holds it: the planning seat is the middle of **the whole table**,
  and at DUEL! it flies *down* into the hand row while the cards fly *up* to the table, coming up
  to full alpha as it goes. It rests there for the rest of the round.
  **It does not grow on the way** *(2026-08-19, owner's call)*. The name was 80 points proposed and
  124 shouted, swelling on the flight, on the split that a preview proposes and an announcement
  records — worth having while the two were separate drawings, and not once the word travels: a
  name that swells while it moves is a second thing happening to it, and the journey plus the
  alpha already say it is committed. `mathNameSize` is now the one size the hand's name is written
  at anywhere, the box's own shout included.
  **It is centered on the screen and overlays the opponent's cards** *(2026-08-19, owner's call)*.
  It sat over the player's own half until then, which kept it clear of that row at the cost of
  putting the loudest word on the screen off to one side — and of asking a name at 80 points and
  growing to fit half a screen. `tableCenter` is the seat now, and the overlap is accepted rather
  than designed around: the opponent's cards have been read by the time a hand is named, and the
  alternative is shrinking the name, which is the opposite of what its size is for.
- **The name carries a second line saying what it is worth** *(2026-08-19, owner's call)*:
  `1.15x DMG`, traveling with it as one object.
  **The multiplier used to be a number the player first met when it flew out of the word**, several
  beats after the round was committed — so the ladder was something to be told about afterwards
  rather than something to play toward. `handMultiplierLine` formats it through
  `handMultiplierText`, the lines' own formatting, so the planned figure and the fired one cannot be
  two spellings; `TestTheHandNameCarriesTheMultiplierTheLinesWillShow` pins that.
  **Only the name grows on the flight.** The line is written at `mathMultLineSize`, which *is*
  `mathTermSize` — the size a figure is written at in a line — wherever it is drawn, while the
  name swells 80 → 124 around it. The gap between them stays proportional to the name, so the pair
  opens up rather than colliding, and `multLineDrop` is the one function both the drawing and the
  origin measure it with.
  **The line is not replaced by the multiplier — it *is* the multiplier, and it sets off**
  *(owner's call)*. It rests under the name through the start of every line, and **the whole banner
  is cleared on the frame the first line's copy leaves** — `mathBox.multStarted`, in
  `advancePlayback`, where the box's clock runs. Every line's multiplier flies out of the same spot.
  The name goes with the figure rather than a beat later: it has been carried down, read and spent
  by then. The handoff is the damage figure's four-things-matching rule
  applied a second time — same size (`mathMultLineSize` = `mathTermSize`, and `fromScale: 1` so it
  does not grow like a card's figure), same color, same place, same frame. **The origin is the
  `1.15` inside `1.15x DMG`, not the line's center**, or the figure would start under the `x` and
  shift sideways on its first frame. `handMultiplierOrigin` falls back to the shouted word for a
  hand the banner never carried — an opponent's, which nothing produces today.
  **The point is that it never leaves the screen.** A preview that vanished at DUEL! and a shout
  that popped in several beats later asked the player to recognize the same word twice instead of
  watching it move — the card-flight argument applied to the one thing on this screen that is not
  a card.
  Three things follow. **The box does not shout what the banner is already saying** — otherwise the
  announcement is drawn twice and the multiplier flies out of whichever copy was drawn last;
  `showing` is that check, and a word the banner does *not* carry (an opponent's hand, which
  nothing produces today) still pops on its own. **It is raised in `startRound`**, on the last frame
  `previewAttack` can still be asked, and cleared in `endOfRound` — except on a settled duel, which
  freezes with its cards and its name up. And **it is centered on `handRowCenter`, never on
  `handBand`**, or it would drift sideways as the row narrowed under it.
- **The name doubled and is bold** *(2026-08-19, owner's call)*: 80 points, wherever it is
  written, making it the biggest type on the screen. Bold is faux — the same word drawn again
  `mathBoldStep` to the right, the pane's own idiom, since `text/v2` has no synthetic bold and
  kubasta ships one weight. The step is proportional to the size and applied *after* the scale, so
  a breathing word does not pulse between bold and not.
  **`TestTheWidestHandNameFitsTheScreen` is what holds the size**: a name is not a figure —
  `FOUR OF A KIND!` is fifteen characters — and it is centered, does not wrap and cannot shrink, so
  one too wide runs off both edges at once.
  **The margin came back when the name stopped growing** *(2026-08-19)*: at 124 the longest name —
  `ELEMENTAL THREE OF A KIND!`, the three matching axes having given every rung an axis word — was
  1220 pixels of 1280, about 95% of the screen; at the one size of 80 it is around 790. The test
  still holds the end that matters, and anything that grows either the catalog's wording or this
  size trips it.
- **Both names breathe** — `mathBreath`, a slow ±6% swell read off `gs.Count`. It is on the free
  clock rather than on a script's, because the preview has no clock at all and the shout's own
  finishes while the word is still up. For the shout it *multiplies* the pop rather than taking
  over from it, so there is no step in the middle of the only thing moving.
- **The name is pink, and the multiplier is pink with it** *(2026-08-19, owner's call)*.
  `handNameInk` — planned name, shouted name and the multiplier that flies out of it. It was
  `attentionYellow` until lightning's border took that same darkened yellow, at which point the
  loudest word on the screen was wearing a hue that also means "this card is lightning". **The
  multiplier follows the word rather than staying behind**: it flies *out* of it, and a figure
  leaving a pink word in yellow reads as a second thing appearing. `attentionYellow` has one user
  left, the ring round the deck stack. Pink already means relic and pane chrome, which is the
  question to answer before a third pink is proposed.
- **The arithmetic is set large** *(owner's call)*: terms at 76, operators at 60, a hit's figure at
  100. The landing damage figure is the same size, `hitFigureSize` being `mathTotalSize` rather than
  a size of its own. **Neither width nor depth is held** — see the layout bullet above.
- **Every number is drawn in the color of what produced it** *(2026-08-19, owner's call)*. A
  card's figure wears that card's element — `cards.BorderOf`, the same color as the border it
  flies out of — so a hit reads as being made *of its card* rather than handed down by the
  game; the multiplier wears `handNameInk`, the hand's own color, which is also the banner it
  leaves; the total wears the attack ink, and the damage figure that flies on out of it wears the
  same. Operators stay faded ground ink, being the one thing on the line the game supplied rather
  than the player. **The element is read off the card in the seat, never off the event** —
  `Event.Element` is the hand's lead card and every line has its own.
  **Lightning's own color was darkened to make this work** — `{240,205,55}` to `{214,152,12}`, in
  `cards`, so every lightning border moved with it. A bright yellow is legible on a dark ground and
  nearly invisible on the two light ones this game draws on — the off-white card surface and the
  light screen — and a figure written straight onto the cream is where that finally showed. It is
  now the same value as `attentionYellow`: a collision rather than a shared constant, and the
  attention color is the one that moves if they ever have to be told apart.
- **A flown figure travels and grows; an operator is stamped in place.** That difference is the
  whole grammar of the box — something that flies came off a card, something that pops is
  punctuation the game supplied.

**Within a turn the order is `combat.Categories()`: the defend cards first, then the attacks —
announced, then the hand, then the hits.** The screen does nothing to arrange this; it replays
the log in order, and the engine decides. Both the resolver and the table's two rows read
`ResolutionOrder` rather than deriving an order of their own.

Three consequences for playback. **The hand line lands after its cards are announced but before
any hit**, so a boosted figure never arrives before the reason for it, and `noteHand` has real rows
to mark because the whole queue is seated at DUEL! rather than a card at a time.

**The rung is raised by the hand's announcement and by nothing else.** `noteHand` lifts
`Event.RungCards` — the cards that *made the hand* — rather than `HandCards`, which is every hit the
turn throws and would stand up a card that is in no part of the rung. `firingSeats` is a list
rather than one seat because a counted hand is not contiguous. Then `advancePlayback` puts them back
down the moment the lines start, and the lines walk `HandCards`: **the rung is raised, the hits are
thrown.**

**A duelist's turn is three moments and each has exactly one gesture: the shields fire as a bundle
with nothing lifted, the hand is announced with its cards up, and the lines run with the table at
rest.** Nothing else on that turn may lift a card — a second gesture ahead of the announcement
reads as whichever card it lifted having gone first.

And **the hand is announced even if every hit then misses** — the shock rolls per hit, after the
hand event, because the hand is scored off the queue and the queue was committed at DUEL!.

### Pacing: one speed, and a table of proportions

*2026-08-19, owner's call, from playing it.* **`beatTicks` in `clock.go` is the game's one speed
— 25 ticks, five twelfths of a second** — and `eventDwells` is a multiplier per event kind, read
through `eventDwell`. Everything is `1` today: the beats were being tuned against a dwell that was
itself wrong, so the speed came down and the proportions were flattened to see what that alone
does.

- **Two questions, two edits.** "Playback is too slow" is the constant; "a chill should hold longer
  than a card firing" is a multiplier. Written as durations the two could not be asked separately —
  every retune of the speed meant re-deriving every entry, and an entry that had drifted out of
  proportion looked exactly like one that had been chosen. **A row that is not 1 needs a sentence
  saying why**, the way the choreography table's rows carry a reason.
- **75 → 25 is the feed leaving, not impatience.** The dwell went *up* to 75 on 2026-08-07 because
  the Resolution feed had made every beat a sentence to read. The feed went behind a button on
  2026-08-18 and the round narrates itself in pictures now, so the reason for the long beat left
  with it.
- **Per-kind pacing is back, and the shape it comes back in is the point.** It was a `switch` with
  a `default` arm once, and the default was the shortest dwell — so every kind added after that
  switch was written inherited a quarter-second flash nobody chose. A map
  with an entry per kind, `TestEveryEventKindHasADwell` failing when a kind is added, and a missing
  entry falling back to the *plain* beat rather than to nothing.
- **The dwell is keyed on the event behind the cursor**, not the one at it: the cursor names the
  event about to arrive, and what is being held is the one already on screen. `dwellForCurrent` is
  that offset, in one place.
- **Every other clock on the screen is a fraction of the same speed** *(2026-08-19, owner's call)*.
  `beat(num, den)` is how they are written: each beat of the hand dialog, the damage figure's
  flight and its hold, the banner's journey, and every card flight, deal, stagger and slide. The
  fractions reproduce the numbers those fifteen constants held at a speed of 25, so introducing it
  changed nothing — what it changes is that they move together. Before it, cutting the speed sped
  the round's *account* up and left every animation in it exactly as slow, so a round was paced by
  whichever of the two happened to be longer. `beat` never returns less than a tick.
- **The one clock that is not tied to it is `mathBreathTicks`**, and the line is worth holding:
  everything `beat` scales is a *duration between two things*, and the breath is an idle
  oscillation on a word that is mostly on screen while nothing is playing back at all. Tying it
  would make the label breathe faster the faster a round is watched.
- **Cards move during planning too**, which is the thing to know before turning the speed down: a
  discard leaving the hand is on the same clock as the round, and no round is playing while it
  happens. That is the trade for a single speed; the alternative is a second constant for movement.
- **`demoGiveUpAt` has a flat term as well as a multiple of the speed**, since the safety net has
  to outlast dialogs that no longer shrink in proportion to it.

### A hand arrives: deal, cascade, sort

*`combat_deal.go`, 2026-09-15.* **Every hand in a fight comes in the same way.** Cards fly out of
the pile left to right in **pile order**, the flip cascade plays **one beat per worn ring** over
them, and the row **sorts itself last**. The opening hand used to be filled and sorted with nothing
on screen; a refill flew. Two ways of doing one thing, and the one the player meets first said
nothing.

- **The sort moved to the end, reversing `spendSelected`'s old rule.** It sorted before anything
  was animated so a dealt card flew straight to its final slot. That is one journey per card and a
  hand that never shows what the shuffle gave you.
- **The hand holds the finished cards from the first frame**; the deal owns the **faces**. So a card
  clicked while it is still wearing its pile face is the card the engine will score —
  `shownLife`'s division, applied to a card rather than to a bar. The exception is the sort, which
  really does reorder the row, on a hand with no round in flight.
- **The flips chain now** *(owner's call)* — `combat.FlipSteps` is the walk, each ring reading what
  the ring before it left. Two rings is two beats. See MECHANICS.md §The flip relics.
- **A ring that touches nothing in this hand is not a beat**, and the ring that *is* firing
  **toasts** — `relicToast`, which is the rattle, a tilt one way then the other, and the lit
  border, all three off one `travel`. The hits' own toast, on a second clock, since one is playback
  and the other is a hand arriving. **A resting relic is blitted and a toasting one is flown**,
  because a turn puts the card off the pixel grid.
- **`tickDeal` is driven by the scene, not by `combatTheater.tick`.** Its stages hand over to each
  other and the handovers need the scene: the cascade reads the run's worn relics and the sort
  rewrites the hand.
- **`dealtTo` is the fourth suppression** on the hand row, beside `inboundTo`, `resolvedInHand` and
  `slidingTo`, and it is checked after the first three for the same reason the morph is checked
  last.

### Shields break the attacks they ate, and a card can be marked

*`combat_shatter.go` and `internal/cards/mark.go`, 2026-09-08.* A shield eats the creature's
hit of its own element first and then its **heaviest** — never simply its first —
`combat.shieldedSlots`, decided at the top of the creature's turn — and the screen plays that as
**one beat between the two turns**: every blocked pip flies out of the duelist card into the attack
card it kills, wearing the element of the shield the engine spent, the break opens across that
card's face, the round holds, then the creature swings with what is left. **A matched block writes
`+1 AP` over the card it broke** — `drawSurgeNote`, a placeholder — and the banked points show as
`(+N surge)` beside the AP figure while the player plans the turn they pay for.

- **The rules decided it before a frame was drawn, which is the only reason it is drawable.** The
  mask is chosen up front rather than as each card arrives, so the whole exchange is known at the
  boundary. Re-ranking as the turn resolved would win a little optimality and cost the player any
  way of watching it happen — see MECHANICS.md §Shields, which records that trade.
- **A *mark* is the third way to say something on a card, and it is not an upgrade.** The face is
  what the card is; an upgrade is what the run permanently made it and takes the left column; a mark
  is the card's situation and covers the face. `cards.Mark` is a **bitmask** — a card can be broken
  and pointed at at once — and `cards.drawMark` owns the order they compose in. Append-only, and
  worse to insert into than an ordinal enum: claiming a bit in the middle changes every existing
  value's meaning.
- **Two halves, one geometry.** The settled mark is baked into the cached card image; the arrival
  changes every frame and would blow that cache, so `internal/cards` exports the crack geometry and
  the screen strokes the same lines on the GPU. Same seed both sides, so the frame the animation
  ends on and the frame the mark starts are the same picture.
- **`advanceBreaks` is a copy of `advance` and has to be.** The generic one ticks a mover and drops
  it in the same pass, so a break has exactly one frame in which it is both finished and still in
  the list — noting them outside that loop saw nothing, the mark never landed, and the web appeared
  and vanished.
- **A break lives inside its own round.** `seatEnemyCards` drops the marks, because the opponent's
  row is re-planned the instant a round ends and a seat number stops meaning what it meant. Anything
  wanting to *point* at a break has to do it during the round.
- **The pips leave the shield row as they fly**, several beats before `KindBlocked` says so — the
  same predict-then-correct the raises use.
- **`Event.Slot` is the index into the turn as it resolved**, the convention `HandCards` already
  uses, and it inherits that convention's known gap: a chilled card is trimmed off the front before
  the indices are handed out while the table row still draws it. Nothing can chill a creature today.
- **It holds the playback cursor** — `combatTheater.running` — which is pacing and is allowed. The
  hold after the break is the longest single one on this screen, deliberately: the round has three
  acts and the middle one had no beat of its own.

### A hit landing, and the bar that waits for it

*`combat_hits.go`.* A hit's figure travels out of wherever it was worked out and into the card whose
bar it empties, and **the bar holds what it had until the number arrives**, so the drop and the
arrival are one event rather than two. It is the second half of the hand dialog: that answers
"where did that number come from", this answers "and what did it do".

- **Several fly at once.** A hand-forming turn's lines finish in whatever order their arithmetic
  allows and each is thrown the moment it does, so up to a turn's worth of figures can be in the air
  together and the bar drops by each as it lands.
- **The model has already moved.** The life is written before the figure is raised — by
  `throwColumn` for a hand's hit, the moment its line finishes; by `applyEvent` for a solo
  attacker's, when the event is reached — so a figure in the air is a ghost of something that has
  happened. What lags is the *drawing*, through `shownLife`, which is a view over the combatant
  rather than a second copy of it. `enemySpec` and `duelistSpec` take the life to draw as an
  argument for exactly this reason.
- **`shownLife` is the model plus every figure still in the air, capped at the most any of them
  was holding.** The sum of what is owed is what the bar has yet to lose, whatever order the
  figures land in; the cap is what keeps a killing hit's overkill from drawing a bar above the life
  that was there.
- **What a figure holds is read off the combatant, never worked back from the event.** `e.Life` is
  clamped at zero, so `e.Life + e.Amount` is the *size of the hit* on a killing blow and the bar
  would jump up for the length of a flight. The caller reads the life before it overwrites it.
- **It stops the playback cursor too**: a figure crossing half the screen does not fit inside one
  event's dwell, and the alternative is the bar dropping before the number reaches it.
  `combatTheater.Running` is what `advancePlayback` waits on. It changes pacing and cannot change
  an outcome.
- **Where it sets off from is a rule, not a rectangle** — `anchorBlow`. A hand's hit leaves its own
  line's figure (`hitFlight.from`, the one mover that stores a point, because a line is laid out once
  and never moves); a solo attacker's leaves the lit card's seat, because it emits no `KindHand` and
  every attack lands its own face damage. `soloAttacker(side)` is the predicate that knows which.
- **The handoff from a line is four things matching, and all four are deliberate**: the figure is
  the total's size (`hitFigureSize` *is* `mathTotalSize`), the total's color, at the total's
  position, and the line stops drawing its total on the frame the figure sets off (`spent`). Any one
  of the four missing and it reads as two numbers swapping rather than one setting off. It also does
  not fade *in*, for the same reason.
- **It shrinks where a line's items grow**, and the difference is the meaning: a term flying into a
  line comes toward the reader, a figure flying into a card goes away into it.
- **`Init` takes the whole theater down**, which is the lesson the frozen last round taught — anything
  tidied up only by the end-of-round spend assumes every round ends in one, and a settled duel does
  not.

### A card that fires says so: the signal widget

*`combat_signal.go`, 2026-09-10.* A rider firing used to be **completely silent**. A golden card
came up on its one face in five, the run gained a permanent point of DMG, and the screen drew
nothing at all — no figure, no mark, no line. The choreography table had said `KindGrantedDMG`,
`KindGrantedLife` and `KindHealed` flew from the acting seat to the acting card since the day those
kinds landed, and nothing ever drew one. This is that promise kept, plus the vocabulary the table
had no word for.

**Two halves, and the burst is the new one.**

- **The burst** is the firework: rays thrown out of the card that fired. It is an *emphasis at the
  source*, not a journey, which is why it is deliberately **not a `gesture` in the theater table** —
  it composes with whatever row that table already has, and that is what lets the next thing wanting
  fireworks (a scored card, a relic firing) reuse it without the table growing a row per decoration.
  A `gestureBurst` is worth adding the day something bursts and sends nothing anywhere.
- **The flight** is the figure traveling into the figure it changed: the DMG row, the VITAE row or
  the health bar. The rows come off `cards.DuelistStyle`'s own `StatsTop`/`StatRowPitch` rather than
  from constants typed here, so a card that re-lays out moves the target with it.

**A signal is drawn in the tint of the rider that threw it** — `upgradeForRider` into
`systems.UpgradeTint`, the same table the card's own face is washed with. The wheel is full, so a
hue of its own would be claiming one; and the burst then matches the card it comes out of, so the
two read as one object rather than as a card and an effect near it. Gold sparks for a gold card,
silver for silver, sparks and figure alike — **one card, one color, burst to landing**.

**`RiderVitaeInHand` is the one exception, and it is the game's rather than this widget's.**
`vitaeInk` is the crimson vitae is written in everywhere it is written, and it is the only red on
the table precisely so a figure in it says "money" before it is read; that rider's whole subject is
vitae, so its placeholder blue-gray was the one tint saying the wrong thing. **Silver is deliberately
not swept up in it** although it also pays the purse: what a silver card says is that the *metal*
came up, and the row it lands on is already crimson without the figure agreeing. The core of a burst
is lifted only a quarter toward white for the same reason — at more than half, a pale tint like
silver's put a white disc back in the middle of the thing that had just stopped being one.

**When they fire took the real decision.** Riders resolve *before* the attack phase — `playTurn`
runs chill, then riders, then the hits — so every one of these events sits in the log ahead of
`KindHand`. Drawing them where they sit would put four fireworks up before the hand was named. So
the screen defers *(owner's call)*:

- **Every held card signals first, and the lines do not begin until they have landed** *(owner's
  call)*. `startHandMath` calls `releaseHeldSignals` after building the box and before it has run a
  frame; `advancePlayback` freezes the box while any signal is up, so the held figures fly to the
  duelist card and only then do the hits start being worked out. Together, because what the turn
  kept back is one fact about the turn rather than several about cards.
- **Then each played card signals as its own figure sets off**, released by
  `handMathBox.takeSignalSeats` on the beat that card's term starts in its line. The lines start
  together, so the cards' signals mostly leave together.
- **The resolver was not reordered to achieve this.** A heal arriving before the hits is a rules
  decision with its own argument in `playRiders`. The screen owns when it draws; the log owns what
  happened.
- **A turn that never scores flushes at the boundary** — the acting side changing, or the round
  ending. A hand of nothing but defenses forms no hand, so there is no sequence to hang anything on;
  `noteShieldRaise` keeps the same fallback for pips.

**Every signal holds the playback cursor** *(owner's call: every signal of a card firing holds)*,
including inside the lines — the box waits on `running(theater.signals)` before its next term.

**`signalShown` is `shownLife`'s idea pointing the other way.** A damage figure lands on a life the
model has already spent, so the drawing lags. A grant lands on a figure the screen's copy of the
duelist does not hold until `endOfRound`, so the drawing *leads*: the tally grows when a figure
arrives and `theater.adopted()` drops it on the frame the authoritative duelists are taken up.
`duelistSpec` therefore takes DMG and MaxLife as arguments too, for the reason it already took life.

**`combat.Event.Rider` was added for this** *(2026-09-10)*, on `Event.Relic`'s argument: the thing
that caused this is something the player can see and nothing else on the event could name it. Two
riders emit `KindVitae` — a played `RiderSilver` and a held `RiderVitaeInHand` — and the fight log
printed **"kept back for N vitae" over a card that had just been played** until the field existed.
`Event.Slot` is now set on rider events too, so a signal knows its seat rather than guessing from
the firing list.

**`riderDraws` is the tripwire**, and it is the choreography table's argument one layer down: every
rider kind says whether it reaches the log, as which kind, and — when it draws nothing — why. An
absent entry and a deliberate silence read identically in a switch;
`TestEverySignalRiderIsAccountedFor` walks `combat.RiderKinds()` against the table *and* against
`noteSignal`, so the two cannot drift.

**The burst pattern is derived from the rider and the seat, never rolled** — the crack pattern's
rule and the dissolve's, and the explicit exception the `randomness` skill records.

**What is still missing:** gold, silver and the heal write **no line in the fight log**, and the
vitae line only appears when some earlier row is open — `attach` drops a line when `cur` is -1, and
rider events arrive before anything on that turn has opened a row. Fixing it means deciding whether
these `announce` their own line, which is a change to the log's shape rather than to this widget.
See `internal/scenario`'s `signals` entry for looking at any of it.

### The round timer under the tower place

*`combat_hud.go`, 2026-09-06.* Every fight lasts five rounds and the duelist still standing at the
end of the fifth dies — see MECHANICS.md §The round limit, and `internal/combat/clock.go`, which is
what actually ends it. The screen draws a **five-cell bar under the two tower lines**, one cell per
round, filled for the rounds already spent.

- **The clock is a rule and this is a readout.** `s.round` is the count and the bar is a picture of
  it; the limit is checked inside the resolved round, so pausing, the speed slider and a dialog over
  the cursor cannot buy a player a sixth round. Same constraint as everything else on this screen.
- **The limit is read off the fighter, never off the run.** `session.Session.RoundLimit` reaches a
  duel through `Equip`, and a bar reading the run directly would keep drawing five while the duelist
  fought to whatever a relic had moved it to. A screen with no fighter draws no bar.
- **Segments rather than a sliding fill.** A round is a discrete thing the player spends, so what
  the bar has to say is a count — three dark, two left — not a proportion. It also survives a limit
  a relic has moved with nothing to rescale: six cells is six rounds.
- **The last cell takes `modalCloseColor`, and only once the fight reaches it.** There is no hue
  left to claim, so this is not claiming one: it is the existing meaning of the game's one red —
  "this ends something" — arriving at the moment it becomes true.
- **There are 23 pixels between the tower lines and the table row and the bar spends 20.**
  `TestTheRoundTimerFitsUnderTheTowerLines` is what holds that; the column is the duelist card's,
  which is why the readout is a bar and not a sentence.
- **It arrives with a tooltip**, which is deliberately unlike every other figure written straight
  onto the table — a timer that killed without having said what it was would be the worst kind of
  hidden rule. The tutorial names it before the first duel for the same reason.
- **`KindTimeUp` is the event.** It moves the life bar like a burn does (nobody acted, so nothing
  else would), pops on the card whose bar it empties rather than flying from a seat there is none of,
  and opens its own line in the feed with the fall following it.

### The shield row on the duelist card

*`combat_shields.go` and `card_art.go`.* The player's defend cards raise shields, and
**one pip per shield is drawn in the seat the enemy card's status badges occupy** — same offsets,
same box, so the two fighter cards stay twins. The pip is `assets/form/defend.png`, the mark the
cards themselves carry, so what was raised and what is standing are the same picture.

- **`shownShields` is a view, exactly like `shownLife`.** `Duelist.Shields` is not
  written until the round's end state is adopted, so a row reading the model would fill a whole
  opposing turn after the card that filled it and empty a whole turn after the attack that ate it.
- **All three changes to the count are announced**: `KindRaised` when a card goes down,
  `KindBlocked` each time one eats an attack, `KindExpired` when the unspent ones lapse. The last of
  those exists *for this row* — without it the pips would keep drawing a defense the engine had
  already taken away.
- **It falls back to the model when no event this round has spoken**, which is what makes the
  planning phase right: a shield raised at the end of the last round is standing while the player
  builds this one, and nothing has fired yet.
- **The engine caps a duelist at five shields**, which is exactly what the row can draw. That is not
  a coincidence and not a clamp in the screen — see `Duelist.raiseShields`, which takes the cap for
  its own reason and this row inherits it. **`combat.MaxShields` exports it** for the one caller that
  has to predict against it, below.
- **The whole defend phase is one gesture** *(owner's call, 2026-09-17)*. Every shield in the turn
  goes up together: the first `KindRaised` reached flies its own pips and every raise behind it in
  the same phase, all on one frame, out of their own cards and into the pip row along the bottom of
  the fighter card — a pip joins a row, where a damage figure hits a card. The raises behind it
  arrive at their own beats with nothing to do. **It is the shield break's rule and the deal
  cascade's**: what the defend phase says is one thing about the turn, not three things about cards.
  `noteShieldRaise` is the bundle and `raisesInPhase` is the walk. **The pips are the phase's and
  never a hit's** — a defense's hit line comes to 0 and flies nothing; its shields went up a
  phase earlier.
- **A raise names its own card, in `Event.Slot`**, which is the only thing the pips have to leave
  from: nothing is lit during the defend phase, so a seat read off the lit card would have no
  answer. `noteShields` reads the same field for the element a pip lands wearing.
- **The count comes from the announcement, and the row waits for the pips.** A raise carries what is
  standing after its own card in `Life`, which the row takes when that flight lands; `noteShieldRaise`
  swallows the event either way, including the bundled raises that flew beats ago, **because letting
  one through to `noteShields` fills the row while the pips filling it are still crossing the
  screen**. The row records the seat, so one card's pips can never fly twice — **and forgets it at
  the end of the round**, because a seat is a position in one round's table — kept across the
  boundary it gags the next round's defense in the same seat, which then flies nothing and lands
  with no color.
- **The row is one list, and the count is its length** *(2026-09-02)* — `shield_row.go`. It was
  four parallel structures for a day: a count, a color per pip, a "has anything spoken" flag and
  the set of seats. **Every shield bug in that day was two of them disagreeing** — a count ahead of
  the colors drew a white pip, a color list trimmed by something that had taken no shield away
  lost a pip's element, a set of seats outliving its round stopped a flight happening at all. A pip
  *is* its color now, so the commonest failure is unrepresentable rather than repaired in two
  places. **The rule the type carries**: a raise may only raise, and only a block or an expiry may
  lower — a raise is announced a phase after the pips it describes have landed and names what is
  standing after *its own* card, so the first of two raises is smaller than what is already drawn.
- **A defense lifts nothing at all.** `noteResolved` switches on the card's kind: a defense raises
  no seats, a solo attacker's card raises its own, an attack raises nothing. A lift says "this card
  is acting now" and the bundle says it for every shield at once, so lifting them one by one is
  three beats of card movement in front of a hand that has not been named yet. **The argument to
  answer before changing this is which single gesture says the defense happened.**
- **A flight whose seat the row no longer holds still flies, from the row's first card.** It used to
  draw nothing while `landShields` paid the pips in anyway, which is a pip appearing without having
  crossed the screen — the one failure this whole gesture exists to prevent.

### The log writes sentences, and the verb is marked in the text

*A line is `<who> <verb> <phrase>` — **"Duelist attacks with a
heavy strike"** — and the verb is **colored, bold and underlined**: **red for attack, blue for
defend, the row's own ink for everything else**. A round can then be scanned for what *kind* of thing
happened before any of it is read.

**The hand line is the exception and has no verb.** It is an announcement — amber swatch,
`HAND!` in front — and it is the one announcement that takes outcomes, because the attack cards
that would otherwise have carried them write no lines. Which side owns the current line is tracked
in `curSide` rather than read back off the row's swatch, or the amber would make every hit look
like it belonged to the other duelist.

**The verb is marked, never chipped or barred.** A saturated rectangle in a pane that already
carries a swatch and a sentence draws the eye to the block rather than to the word inside it —
which is why neither a filled chip behind the verb nor a full-width highlight bar behind the row
survived. **Mark the words, do not sit them on a lit shape.** If a third version is ever wanted,
that is the argument to answer.

- **`paneRow` is three runs**, `prefix`/`verb`/`suffix`, not one string. The verb has to be its
  own run so it can be measured, tinted and underlined independently; slicing it back out of a
  finished sentence would be worse. Rows that are not sentences put everything in `prefix`.
- **All three marks, on every row, always.** One alone would be ambiguous — the pane already uses
  color for the side and for the live row, and bold alone for the live row — so the verb needs
  the combination to be unmistakable. **The consequence: the underline is no longer what marks
  the live row.** That row is now distinguished by `nowInk` plus faux-bold on `prefix`/`suffix`,
  and the verb keeps its category color there rather than going pink with the rest.
- **The underline sits flush with the bottom of the measured line box**, never a constant above
  it. It used to hang under a chip of fixed height; with no chip the only thing to position it
  against is the text, and `text.Measure` reports the full line including descent — which is what
  clears a descender. A rule placed a few pixels above the baseline struck through one.
- **The prose lives in `internal/screens`, not `internal/combat`.** The rules package names cards;
  it does not describe them. **It is generated from the verb rather than tabulated** *(2026-08-16)*
  — `actionPhrase` and `cardEffect` in `prose.go`, switching on `Verb` and dropping the
  card's own label in as the noun. There were two hand-maintained maps, one string per concept,
  which worked for fourteen concepts and cannot work for the ~400 that per-enemy decks produce: a
  card with no entry drew a blank face. **Every phrase carries an article** so `cardPhrase` can
  slot an element into it — that is a constraint on any new wording here.
- **Outcomes append to `suffix`**, after the verb, so the mark never moves as a line grows.
- **The name is said as well as colored, deliberately.** The swatch already encodes the side,
  but a line beginning "Bash" reads as an instruction rather than a report, and with both
  sides in one list the reader would have to hold which color is which.

**Row highlights and swatches are centered on the measured line height**, not offset from the
row top by a constant. The old `rowY-4` / `rowHeight-2` numbers were picked by eye against a
single 30px pitch and clipped the text the moment a 22px pitch existed. `text.Measure` once per
pane, center everything on it, and any pitch works.

**Caps in kubasta are a size question, not a ban.** The character strip shouted HEALTH /
DISCARDS / VITAE at 12px and `VITAE` rendered as `VITRE` — the uppercase A carries a diagonal
that reads as an R with no lowercase around it to set the shape. **The duelist card's `DMG`
and `AP` are caps at 17px and read correctly**, checked on the contact sheet rather than
assumed. So: title case below about 14px, and look at anything set in caps before shipping it.

**It has no phase headings, and that is a space constraint rather than a decision.** The pane
holds nine rows between `paneFirstRow` and its bottom edge, and five player actions plus the
enemy's already reach that. Under phases the grouping reads off the order anyway. If headings
are wanted, the pane has to get taller or the rows shorter first.

Two panes, and three others were folded away to get there. Chosen folded into the palette;
Enemy went because a merged Resolution already shows the opponent's actions in a better order
than a column of its own; Actions went with the move to the bottom, since the hand has no frame
and there was nothing left for a placement to hold. **The queued-actions pane claimed the 15–39% column
those left empty.**

The player's rows carry `playerSwatch` green and the opponent's carry `enemySwatch` yellow, so
the screen reads as two colors: green is you, yellow is them. `handSwatch` amber is the third
and marks a Resolution line that is not a card acting — it belongs to whoever formed the hand,
but the line is an announcement, and giving it a side's color would file it in the column of
squares where every entry is a card that resolved.

## The action box

[combat_actionbox.go](internal/screens/combat_actionbox.go) is the hand and its
drag-to-reorder, and the reference for building a *game* widget: state on the scene,
hand-rolled hit testing, no toolkit. Click a card to select it into the round's queue, click
it again to take it out, drag sideways to move it along the row.

- **`planning()` is the single predicate** for "the player may edit the queue" — derived
  from `cursor >= len(log)` plus both duelists alive, not stored. Drag and the DUEL!
  button both gate on it, so they can never disagree.
- **What is enforced at selection is the *count*, not the budget.** Selection stops at
  `s.fighter.MaxActions()` cards and a card past that draws dimmed; cost is deliberately not
  checked, because selection is also how a card is picked for the discard pile and a hand you
  could not afford would be a hand you could not throw away. The budget is enforced one step
  later, at DUEL!, which goes dead while the selection is over it.
- **The cap lives in `internal/combat`, not here.** It was `maxSelected` on this screen until
  It had to move: it is a rule, and **the opponent's planner obeys it exactly as
  the player's selection does** — a cap enforced only by the screen was one the enemy ignored.
  It is a method on `Duelist` so a relic raising it has somewhere to bite.
- **A press is not a drag until the cursor moves past `dragThreshold`.** Without it every
  click jitters into a one-pixel reorder and selecting a card is a coin toss. The card
  leaves the row at that moment rather than on release, so the gap closes under the cursor
  and the drop index is measured against the row the card lands in.
- **Dropping outside the band puts the card back.** There is no discard gesture — clicking a
  card off is how it leaves the queue, and that is visible on screen in a way that dragging
  into empty space never was.
- **Selection lifts a card *up* out of the row**, and reordering is horizontal. Both rotated
  with the layout; selection is the only state a card carries, so it gets a whole axis to
  itself rather than a tint competing with the affordability dimming.
- The in-flight card is drawn last in `Draw`, so it rides over everything it crosses.
- **The AP budget is still drawn twice, but not stacked**. The `3/6 AP` figure
  and the pile counts used to share a line of small text wedged between the cards and the bar.
  That line is gone; **the figure moved down onto the button line**, left-aligned under the
  left end of the bar, and the pile counts moved to the pile. What was wrong with the figure
  was where it sat, not that it was written down — the bar answers "how much room is left"
  without being read, and the figure answers "exactly".
- **The row sits directly on the bar.** Losing the text line freed 22 pixels and the *cards*
  took them, moving down from 59% to 61% rather than the bar moving up — the strip below the
  bar held the deck stack, whose top was measured from it.
- **And down again to 66%**, once the pile stopped being measured from the bar
  at all. The row falls exactly where the AP figure's top meets the Discard button's top;
  see the bottom-strip section above and `TestTheAPFigureLinesUpWithTheButtonStrip`. The
  bar, the figure, the cards and the band above them all measure off `handTopPct`, which is
  why one constant moves the whole lower half of the screen.

### Sorting the hand

*[combat_sort.go](internal/screens/combat_sort.go), 2026-08-16.* Three 44px square buttons in a
column against the band's right edge, centered on the cards: **`$` cost, `T` type, `E` element**.
The active one latches darker than the other two.

- **Sorting a queued hand re-prices it** *(owner's call)*. Cross-category order is still regrouped
  away by `ResolutionOrder` and a hand is still counted rather than read in sequence, but a growing
  relic steps between the hits of one turn — so the order of the queue decides what the cards are
  worth. The buttons stay live and a bad sort can cost damage; that is the intent, not an
  oversight.
- **Cost is the default, and every mode ends with it.** Each arrangement is the deck overlay's
  own key chain — cost, form, concept, element — with one key promoted to the front, so a row
  of cards means the same thing in the hand as in the panel. Only the leading key differs.
- **The sort re-applies on every refill**, in `spendSelected` *before* anything is animated, so a
  dealt card flies to the slot it will actually occupy. A drag still works and survives until the
  next deal, at which point the sort reclaims the row.
- **Every figure in a hit's line comes from a card, and that card shakes as it is written**
  *(owner's call)*. A card's damage flies out of the played card, a relic's multiplier out of that
  relic's card, and an echo's extra hit shakes the relic that bought the landing even though it puts
  no figure on the line. Each line runs its items one at a time, so putting the relic figures *in*
  the script is the whole of the sequencing within a hit. `mathItem`'s `relicSeat` / `cardSeat` /
  `shakeRelics` are the marks, and `handMathBox.takeShakes` is what the screen drains each tick.
- **Sideways, never a jump.** Vertical is spoken for twice already — a selected card lifts in the
  hand, and a card that built the hand lifts on the table while the hand is announced. The shake says "this
  one is paying *now*", so it needed a direction nothing else uses.
- **The drag runs on the shared controller in `carddrag.go`** *(2026-08-26)*, which the worn relic
  row also uses on all three screens that draw it. The hand's adapter is `handRow`; it really does
  remove the card from `s.hand`, where the relic row's removes nothing and lets the run stay the
  authority.
- **`sortHand` returns the permutation it applied** — for each new position, the index that card
  came from — and that is why it sorts a slice of indices and rebuilds rather than sorting the
  cards in place. Two identical cards cannot be told apart after the fact by looking at them, and
  a card sliding to its new place has to know where it set off from. `spendSelected` reads the
  same list to tell a survivor from a card fresh out of the pile: past `dealt` it flies in, below
  it, it slides.
- **A sorted card slides, it does not pop.** `handSlide` is the fourth mover in
  `combat_flight.go` and stores no coordinates, like the other three — but it carries a row size
  at *each* end, because a survivor at the end of a round leaves one row and lands in a
  differently sized one. The gesture is flat, full size, no flip or spin: the other three cross
  the screen, this one shuffles a few inches. `slidingTo` blanks the row's own copy exactly as
  `inboundTo` does, and `addSlide` drops any earlier slide claiming the same slot so pressing a
  second mode mid-flight cannot draw one card twice.
- **`sortHand` resyncs the queue, and that is load-bearing.** The list is the authority on the
  queue's *order* as well as its membership and `handIndexForQueue` is the inverse of that walk,
  so a hand rearranged under a stale `fighterActions` leaves the hand preview naming a hand the
  cards at those positions do not make.
- **`sortMode` is the one field `Init` does not reset.** A reading preference is not a fact about
  a duel, and snapping back to cost every fight would make it something the player re-presses.
- **All three go dead outside `planning()`** — a resolved card is drawn from the hand slot it
  flew out of, so rearranging mid-round would light the wrong card on the table.
- **`elementRank` and `categoryRank` are written out**, like `formRank`. `combat.Basic` leads
  its enum as the zero value and trails on screen: the colors are what the statuses are counted
  on, and the colorless cards are the plans.
- **The cards lost width to pay for the column.** `cardBandWidth` is the band less
  `sortColumnReserve`, `handBand` centers on *that* rather than on `PctX(50)` — so the whole row
  nudged left instead of only its right edge coming in — and `handBandLeftPct` came in from 4% to
  2% to find some of the overlap back. The AP bar and the AP figure travel with it, both being
  measured off `handBand`.
- **`models.Button.Latched` and `TextSize` arrived for this** and both are deliberately general.
  A latched button takes `ButtonStateLatched`, drawn at **38% — darker than resting, not
  brighter**: hover and press own the bright end of the ramp, and an active mode lit to full
  strength read as a button the cursor was on. Disabled still wins over the latch. `TextSize`
  zero means the default 20, the same "use the default" convention `BaseColor`'s zero alpha
  uses; the sort buttons set 30, because a square carrying one character is nearly all label.

## The tutorial's footprint on this screen

*`combat_tutorial.go`; the machinery is `internal/tutorial` and `internal/screens/tutorial.go`.*
Two things landed here on 2026-09-08 that a change to this screen can break silently:

- **A step waiting for NEXT holds the round where it is.** `tutorial.Run.HoldsRound`, read once in
  `advancePlayback`. It exists for the shield step, which is the first in the lesson to land inside
  a playing round. **Only a NEXT step holds** — that is what stops a step waiting on an *outcome*
  from stopping the thing it is waiting for. Pacing, never a rule.
- **A card the lesson points at is tinted, not framed.** `tutorial.Anchor.NamesCards` is the closed
  table; `screens.marksFor` reads `gs.InputFocus` — the same list the spotlight is handed — so the
  lit card and the clickable card are one card by construction. It matches seats by **equality, not
  containment**, because the hand row overlaps when it is full.
- **`gs.InputFocus` is a *list* of rectangles**, and that is load-bearing rather than tidiness. It
  was one, and one is a bounding box: an anchor naming a set of cards could only hand over the box
  round them, so a card sitting between two of them was lit and clickable while not being named. The
  taught four sit at seats 0, 1, 2 and 4 under the default cost sort. See CLAUDE.md.
- **An anchor names what a step is asking for, not what it is about.** `matching-cards` and
  `matching-cards-left` are two anchors over one set for that reason.

## Hidden information is gated on `DebugGameplay`

The opponent's queued actions were concealed in the enemy rows of the queued-actions pane
unless `DebugGameplay` is on. `CombatScene.concealEnemy` is the single predicate —
`!gs.DebugGameplay && s.planning()` — and anything else that becomes secret should join it
rather than growing a second rule.

**The ledger needs no concealment rule at all**, and that falls out of its design rather than
being a second decision: a round is written into the run's account when it *ends*, so the panel
can only ever hold rounds that have finished playing. The round being watched is not in it yet, so
there is nothing for it to leak. *(The fight log it replaced got the same property a different
way, by being built from `s.log[:cursor+1]`.)*

- **Concealment lifts once playback starts**, for the same reason.
- **Concealed rows keep their real count**, so the opponent's AP spend stays readable even
  when the actions do not. Deliberate: collapsing the rows would hide the spend and destroy the
  pane's account of who acts when, which is the one thing that pane exists to show.
- Debug is a *view*, never a rule. `ResolveRound` never sees the flag, so turning debug on
  or off cannot change an outcome — the same constraint that applies to playback speed.

