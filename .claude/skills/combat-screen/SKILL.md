---
name: combat-screen
description: The combat screen's layout, its card/action box widget, hidden information, and the resolution-order rule the screen must obey. Load before touching any of internal/screens/combat.go, combat_deck.go, combat_hud.go, combat_actionbox.go, internal/combat, the hand, the cards, the deck overlay, the fight log, the character block, or anything about how a round is drawn or played back.
---

# The combat screen

The screen under active construction, and the reason `NewGlobalState` boots straight into
`Combat`. Everything here is a decision already made — read before proposing a change to
one, and see `TODO.md` for what is still open.

**The rules live in `internal/combat` and the screen only replays them.** `ResolveRound`
decides a whole round before a frame of playback runs. Never change those rules to make a
screen look right — if a screen contradicts the engine, say so and let the owner decide
which one is wrong. That is a game-design call and it ripples into the tests and the
balance.

**What is not here and still applies.** `CLAUDE.md` holds the rules this screen is built on
top of, and they are not repeated below:

- **The input vocabulary** — left click, drag and drop, long press. No right click, ever,
  and no keyboard outside the one seed field.
- **Glyphs are generated and cannot be resized.** Authored at the size shown, integer
  scales only, 1 is the floor. They are **not all one size** — the damage sword is 64, the
  category glyphs are 32 — so ask `systems.SizeOf(kind)` and never assume `GlyphSize`. **A
  glyph is placed by its inked bounds, not its canvas**, since none of them fills the square
  it is drawn on.
- **The card is drawn by `internal/cards`, not by this screen.** `drawCard` builds a
  `cards.Spec`, pulls a cached image and blits it; it draws nothing itself. Change how a
  card looks there, then `go run ./tools/cardsheet` and refresh the tab — the tool and the
  screen run the same code, so the sheet cannot lie.
- **Colour: name one colour and scale it** with `systems.ColorAtStrength` — but that scales
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
**prepare, attack, defenses**. Defenses come last within a turn because the opponent moves
next, so a defense raised at the end of your turn is up when their blow arrives.

**The attack phase is one blow** *(2026-08-14)*. Every attack card queued is announced with a
`KindAction`, then one `KindHand` names the hand they formed, then a single `KindDamage`
lands. Five Strikes are not five hits.

**The phase is also one line in the log, and it is the hand's.** The `KindAction`s still play —
each is a beat, and each raises its card on the table — but `logRows` writes no sentence for them.
`KindHand` carries `Base`, `Multiplier` and `Amount`, so the line reads *"HAND! Duelist
lands a Pair (20 x 1.5 = 30)"* and the damage attaches to it. **Every hand takes that line and
every line carries its multiplier, the identity included** *(2026-08-19, owner's call)* — a High
Card prints `(20 x 1 = 20)`, because hands are going to be upgradable and a term that appeared only
once the multiplier stopped being 1 would make an upgrade read as a new rule. It is also the
commonest turn in the game, so it is the line that teaches the shape. **Attack cards that build no hand contribute
nothing**, and what says so is the table: every attack card is raised as it is announced, and the
hand lowers the ones it did not name.

**None of the three paragraphs above describes an enemy's turn** *(2026-08-17)*. `Duelist.SoloAttacks`
makes attack cards resolve one at a time in queue order, each landing its own blow, and **no
`KindHand` is emitted at all**. `CombatScene.soloAttacker(side)` is the screen's single predicate
for it and two things read it:

- **The log writes a sentence per attack card**, because there is no phase line coming to carry
  them. `logRows` suppresses an attack's `KindAction` only when a hand *is* coming.
- **The table lights one card at a time**, so `noteResolved` seats `[]int{seat}` rather than
  `attackSeats`. Raising the set says "these cards are one blow", which is exactly what an enemy's
  turn is not: three cards swing three times, and the card that is up is the card that is hitting.

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
  and every raised defend answers the one blow regardless of when it went up. **The row is still
  draggable and the gesture still costs the player attention.** That is a design hole, tracked in
  `TODO.md`; do not paper over it on the screen, and do not invent a rule to justify it.
- **`Duelist.Defends` is a set, not a queue.** One card reaches it as of 2026-08-15 — Defend, at
  50% — and several compose multiplicatively. Nothing about the order they were raised in reaches
  the outcome, and **nothing reduces a blow to zero**: something always lands.
- **`Slot.Index` is not a position in the round.** It is where the card sits in its own
  side's queue, which regrouping breaks apart. Anything asking "how far through the round are
  we" counts slots — `CombatScene.currentSlot` does, and lighting the right Resolution row
  depends on it.
- **The category is deliberately not concealed** on enemy rows. It is what decides where a row
  sits, so withholding it would make the pane unreadable rather than merely uncertain. It took
  over that job from the initiative number — see `concealedLabel`.
- **The card shows its category as a glyph** — a sword for attack, a kite shield for
  defend, an open book for prepare — in the top-left corner above the cost. Three states are
  not a quantity, which is why it is not a numbered badge; a 22-pixel silhouette is read
  before any text is.
- **Cost is a stack of dash marks**, under the category glyph, one per point. Not a numeral
  and not a badge. Costs run 1..4 and a fifth tier is a layout change, not a bigger number.

### Hands, and the one thing they changed on this screen

The catalogue is `data/hands.json`, the matcher is `internal/combat/hand.go`, and the design is
in `MECHANICS.md`; these are what matter to the screen.

- **A hand is a *hand*, and it is a damage multiplier and nothing else** *(2026-08-17)*.
  `Event.Hand` is a `HandID` and `Event.Multiplier` the percent. `handName` in `prose.go`
  looks it up with `HandByID` and prints `Hand.Name` — "Two Pair" — **assembling nothing**. It used
  to join a hand to a *mix* counting the distinct colours, and to fill a `{card}` template from the
  concept that formed the hand, so one hand could print as "Duo Strike Flurry"; both axes are gone
  and a hand carries its whole name. **Exactly one fires per turn**, so there is no stacking to draw
  and no ranking to explain.
- **`Event.Hand` always names a hand** *(corrected 2026-08-19)*. A turn with an attack in it falls
  back to the catalogue's `high-card`, so `HandNone` never reaches a `KindHand` — the log had a
  branch written against the opposite belief and it had been unreachable for some time. **The High
  Card takes the hand line like any other hand**, and carries its `x 1` in both the line and the
  dialog since 2026-08-19 — **the last place it was written differently from the rest**. What is
  left that treats it specially is `matchHand` reaching it by fallback rather than by counting,
  which is structural: counting would match the one-card hand against every turn in the game, and
  the fallback picks the hardest-hitting card rather than the commonest.
- **The event carries the arithmetic, and the engine takes its damage from the same field.**
  `Base` is what the hand's own cards deal added up, and `Amount` is `Base` under the multiplier —
  the blow *before* the attacker's weight and before any defence. `resolveAttackPhase` blunts
  `Amount` rather than re-adding the sum, so the figure printed and the figure landed cannot be two
  different numbers. The gap between that and the `KindDamage` after it is exactly what the defence
  was worth.
- **The multiplier multiplies the cards** *(2026-08-18, owner's call)*. There is no third term: the
  event carried a `Swing` — one 1x attack at the attacker's DMG, *added* to the cards — until then,
  which made a hand's percent worth a fixed figure rather than a proportion. `Swing` is gone from
  `Event` and `high-card` sits at `100` rather than `0`, since a multiplier applied to the cards
  cannot be zero without deleting the blow. See MECHANICS.md.
- **`Event.HandAmounts` is what each of the hand's cards deals**, parallel to `HandCards` and to
  the same count, summing to `Base`. It exists so the hand dialog can show the sum term by term
  without the screen owning `CardDamage`, the strength scaling and every ring that touches a card's
  damage — which would be a second resolver, the thing `Base` and `Multiplier` are on the event to
  prevent.
- **A fired hand keeps its own cards raised, and the list comes from the event**.
  `Event.HandCards[:HandCardCount]` names which cards of the turn formed it. **Never derive
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
  and **every attack previews, the High Card included** *(2026-08-19, owner's call)*. A single
  attack card is a hand — the catalogue's `high-card` at the identity multiplier — so the name is
  on screen from the first attack picked; a queue of nothing but plans names nothing, `BlowFor`
  returning a blow with no cards. **This reverses the old rule**, which was that only a hand of two
  or more previewed, on the argument that HAND! over one Strike empties the word. What makes it
  safe is that the label names the *hand* rather than shouting HAND!, and the log still writes a
  lone attack as an ordinary attack sentence. **Two lines show it**: `drawPlannedHand` writes its
  name across the middle of the table, breathing, in `handNameInk`, with **what it is worth on a
  second line under it** — `1.15x DMG` — and that pair is what flies down to the hand row at DUEL!,
  so the preview and the announcement are one object. See `handBanner`.
  **`Blow.Cards` indexes the turn, not the hand**, which is why the preview goes through
  `ResolutionOrder` — a Prepare queued first resolves last, so a preview read off the hand as the
  player left it would miss the hand behind it.
- **A chilled slot is a row that never resolves.** `currentSlot` counts `KindChilled`
  alongside `KindAction` for exactly this reason — one beat per slot, taken or lost — and
  `TestEverySlotIsEitherTakenOrChilled` pins it. **The pane still draws that row as though it
  happened**, which is a known gap. Ice is the only thing that can take a slot.

### The hand dialog: the sum acted out

*`combat_mathbox.go`, 2026-08-18.* On the beat a hand fires, the blow's arithmetic is played out at
the size of the screen: the hand's name shouted beside the cards it names, then each card's own
figure flying down out of that card into the band above the hand, then the multiplier, then
the answer.

**It exists because the sum was the one number on this screen nobody could source.** The Resolution
feed printed it, correctly, in sixteen-point text on the third row of a three-row box. That is a
*record*, which is what a feed should be; what it is not is an *explanation*. A player could see
the total and could not see which card paid for which part of it, so the multiplier read as a
number the game had decided rather than one they had built.

- **It says nothing the event does not carry and computes nothing.** Every figure comes off the
  `KindHand` event — `HandAmounts`, `Multiplier`, `Amount`. **This is the
  rule to hold**: a second *drawing* of one event, never a second arithmetic. A figure it needs and
  the event has not got is a field that goes on the event.
- **It is the one thing on this screen that can stop the playback cursor.** `advancePlayback` holds
  while `mathBox.running()`, because a sum revealed a figure at a time does not fit inside one
  event's dwell and the alternative is the box racing the log. Every other animation runs on its
  own clock beside playback. **It still cannot change an outcome** — the round was decided before a
  frame of it was drawn — but it *does* change pacing, and `demoGiveUpAt` is sized against the dwell
  alone, so a longer script is worth checking against it.
- **`mathScript(e)` is the half with no geometry in it**, and it is what `combat_mathbox_test.go`
  pins: the strings, their order, which items fly, and that the line ends with the event's own total
  rather than a sum of its terms. The tests create no `ebiten.Image` and need no font — the same
  narrow exception the other screen tests take.
- **The layout is computed once, before anything is shown**, and items are revealed left to right
  into space already claimed. Laying the line out again as each item appeared would recentre the
  whole sum on every beat, so figures already on screen would crawl sideways while being read.
- **It takes its height from the band above the hand and its width from the table, and neither is
  an accident.** The depth is `mathBandHeight`, which is what the Resolution feed's collapsed box
  came to — the size the sum was laid out and looked at against. **The feed is gone and the
  constant deliberately outlived it**: changing that number is re-laying out the arithmetic, not
  tidying up after a deleted pane. While the feed was there this was computed in `handMathRect`
  rather than read off `feedRect`, because a player holding the box open grew it upward and a
  dialog that moved with it would re-lay a line of figures out from under a reader mid-flight. The
  **width** was deliberately not the feed's either: `feedRect` spanned `handBand`, which narrows as
  the hand empties, and a two-card hand gives about 330px against a widest sum of roughly 640 — so
  a centred line that does not wrap and cannot shrink would have run off both ends in exactly the
  rounds a duel is decided in. `TestTheWidestSumFitsItsBand` found that and holds it. **The same
  trap is live anywhere else that borrows `handBand` for something that is not the hand.**
- **Every hand the engine names is shouted, `HIGH CARD!` included** *(2026-08-19, owner's call)*.
  It was silent until then, on the argument above. What changed is that the name is carried by the
  banner from DUEL! onward, so silence at the hand beat would not withhold an announcement — it
  would take a word off the screen at the moment the blow lands, and take the multiplier's origin
  with it. The arithmetic plays either way, so every attack phase shows where its figure came from.
  An event naming *no* hand is still silent; nothing emits one.
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
  **It is centred on the screen and overlays the opponent's cards** *(2026-08-19, owner's call)*.
  It sat over the player's own half until then, which kept it clear of that row at the cost of
  putting the loudest word on the screen off to one side — and of asking a name at 80 points and
  growing to fit half a screen. `tableCentre` is the seat now, and the overlap is accepted rather
  than designed around: the opponent's cards have been read by the time a hand is named, and the
  alternative is shrinking the name, which is the opposite of what its size is for.
- **The name carries a second line saying what it is worth** *(2026-08-19, owner's call)*:
  `1.15x DMG`, travelling with it as one object.
  **The multiplier used to be a number the player first met when it flew out of the word**, several
  beats after the round was committed — so the ladder was something to be told about afterwards
  rather than something to play toward. `handMultiplierLine` formats it through
  `handMultiplierText`, the sum's own formatting, so the planned figure and the fired one cannot be
  two spellings; `TestTheHandNameCarriesTheMultiplierTheSumWillShow` pins that.
  **Only the name grows on the flight.** The line is written at `mathMultLineSize`, which *is*
  `mathTermSize` — the size a figure is written at in the sum — wherever it is drawn, while the
  name swells 80 → 124 around it. The gap between them stays proportional to the name, so the pair
  opens up rather than colliding, and `multLineDrop` is the one function both the drawing and the
  origin measure it with.
  **The line is not replaced by the multiplier — it *is* the multiplier, and it sets off**
  *(2026-08-19, owner's call)*. It rests under the name through every card's figure flying into the
  sum, and **the whole banner is cleared on the frame the sum's own copy leaves** —
  `mathBox.at >= mathBox.multAt`, in `advancePlayback`, where the box's clock runs. The name goes
  with the figure rather than a beat later: it has been carried down, read and spent by then, and
  a word left breathing over the hand while the sum finishes and the opponent swings back is
  saying something the round has moved past. The handoff is the damage figure's four-things-matching rule
  applied a second time — same size (`mathMultLineSize` = `mathTermSize`, and `fromScale: 1` so it
  does not grow like a card's figure), same colour, same place, same frame. **The origin is the
  `1.15` inside `1.15x DMG`, not the line's centre**, or the figure would start under the `x` and
  shift sideways on its first frame. `handMultiplierOrigin` falls back to the shouted word for a
  hand the banner never carried — an opponent's, which nothing produces today.
  **The point is that it never leaves the screen.** A preview that vanished at DUEL! and a shout
  that popped in several beats later asked the player to recognise the same word twice instead of
  watching it move — the card-flight argument applied to the one thing on this screen that is not
  a card.
  Three things follow. **The box does not shout what the banner is already saying** — otherwise the
  announcement is drawn twice and the multiplier flies out of whichever copy was drawn last;
  `showing` is that check, and a word the banner does *not* carry (an opponent's hand, which
  nothing produces today) still pops on its own. **It is raised in `startRound`**, on the last frame
  `previewAttack` can still be asked, and cleared in `endOfRound` — except on a settled duel, which
  freezes with its cards and its name up. And **it is centred on `handRowCentre`, never on
  `handBand`**, or it would drift sideways as the row narrowed under it — the same trap the sum's
  width avoids.
- **The name doubled and is bold** *(2026-08-19, owner's call)*: 80 points, wherever it is
  written, making it the biggest type on the screen. Bold is faux — the same word drawn again
  `mathBoldStep` to the right, the pane's own idiom, since `text/v2` has no synthetic bold and
  kubasta ships one weight. The step is proportional to the size and applied *after* the scale, so
  a breathing word does not pulse between bold and not.
  **`TestTheWidestHandNameFitsTheScreen` is what holds the size**: a name is not a figure —
  `FOUR OF A KIND!` is fifteen characters — and it is centred, does not wrap and cannot shrink, so
  one too wide runs off both edges at once.
  **The margin came back when the name stopped growing** *(2026-08-19)*: at 124 the longest name —
  `ELEMENTAL THREE OF A KIND!`, the three matching axes having given every rung an axis word — was
  1220 pixels of 1280, about 95% of the screen; at the one size of 80 it is around 790. The test
  still holds the end that matters, and anything that grows either the catalogue's wording or this
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
  left, the ring round the deck stack. Pink already means ring and pane chrome, which is the
  question to answer before a third pink is proposed.
- **The arithmetic doubled with the name** *(2026-08-19, owner's call)*: terms 38 → 76, operators
  30 → 60, the total 50 → 100 — it was being overwhelmed by the cards and the shout around it. The
  landing damage figure doubled with it and had to, `hitFigureSize` being `mathTotalSize` rather
  than a size of its own. Width still fits with room — the widest sum the rules can produce is
  about 830 against a 1232-wide band, and `TestTheWidestSumFitsItsBand` measures to the ink now
  rather than to the resting centres. **Depth is the constraint that is nearly spent**: a
  100-point total is 85 pixels tall against `mathBandHeight`'s 82, so it clears its neighbours but
  the next increase has to move the band, not only the type.
- **Every number is drawn in the colour of what produced it** *(2026-08-19, owner's call)*. A
  card's figure wears that card's element — `cards.BorderOf`, the same colour as the border it
  flies out of — so the sum reads as being made *of the cards* rather than handed down by the
  game; the multiplier wears `handNameInk`, the hand's own colour, which is also the banner it
  leaves; the total wears the attack ink, and the damage figure that flies on out of it wears the
  same. Operators stay faded ground ink, being the one thing on the line the game supplied rather
  than the player. **The element is read off the card in the seat, never off the event** —
  `Event.Element` is the blow's lead card and the sum has a figure per card.
  **Lightning's own colour was darkened to make this work** — `{240,205,55}` to `{214,152,12}`, in
  `cards`, so every lightning border moved with it. A bright yellow is legible on a dark ground and
  nearly invisible on the two light ones this game draws on — the off-white card surface and the
  cream screen — and a figure written straight onto the cream is where that finally showed. It is
  now the same value as `attentionYellow`: a collision rather than a shared constant, and the
  attention colour is the one that moves if they ever have to be told apart.
- **A flown figure travels and grows; an operator is stamped in place.** That difference is the
  whole grammar of the box — something that flies came off a card, something that pops is
  punctuation the game supplied.

**Within a turn the order is: prepares one at a time, every attack card announced, the hand, the
damage, then the defends.** The screen does nothing to arrange this; it replays the log in order,
and the engine decides.

Three consequences for playback. **The hand line lands after its cards are announced but before
the damage**, so a boosted figure never arrives before the reason for it, and `noteHand` has
real rows to mark because the whole queue is seated at DUEL! rather than a card at a time. **The
whole attack hand is raised by the *first* announcement** — `attackSeats` reads them off the turn,
so `firingSeats` is a list rather than one seat and the beats after it name the same set — and
`noteHand` narrows the list to what earned it. And **the hand is announced even if the blow then
misses** — the shock roll happens after the hand event, because the hand is scored off the queue
and the queue was committed at DUEL!.

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
  switch was written inherited a quarter-second flash nobody chose, `KindNegated` included. A map
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

### The blow landing, and the bar that waits for it

*`combat_hits.go`, 2026-08-18.* The damage figure travels out of wherever the blow was last seen
and into the card whose bar it empties, and **the bar holds its old figure until the number
arrives**, so the drop and the arrival are one event rather than two. It is the second half of the
hand dialog: that answered "where did that number come from", this answers "and what did it do" —
which used to be a bar dropping while the total sat in the middle of the screen with nothing drawn
between them.

- **The model has already moved.** `applyEvent` writes the new life the instant the event is
  reached, exactly as it always did, and the flight is raised afterwards — so a figure in the air is
  a ghost of something that has happened, and nothing that asks how much life is left gets a
  different answer while it is up. What lags is the *drawing*, through `shownLife`, which is a view
  over the combatant rather than a second copy of it. Same division `spendSelected` keeps: the
  animation never owns the state. `enemySpec` and `duelistSpec` take the life to draw as an
  argument for exactly this reason.
- **It stops the playback cursor too**, for the dialog's reason: a figure crossing half the screen
  does not fit inside one event's dwell, and the alternative is the bar dropping before the number
  reaches it. `combatTheatre.running` is what `advancePlayback` waits on. It changes pacing and cannot change
  an outcome.
- **Where it sets off from is a rule, not a rectangle** — `anchorBlow`. The sum line when the turn
  scored a hand, because the total is already on screen there and two figures for one blow would be
  two blows; the acting card's own seat when it did not, because a solo attacker emits no
  `KindHand` at all and every attack lands its own face damage. `soloAttacker(side)` is the
  predicate that already knows which.
- **The handoff from the sum is four things matching, and all four are deliberate**: the figure is
  the total's size (`hitFigureSize` *is* `mathTotalSize`), the total's colour, at the total's
  position, on the frame the box clears — `advancePlayback` clears a finished box at the top of the
  beat the damage lands rather than a tick after the script stops, so the last frame of the sum and
  the first frame of the flight are the same frame. Any one of the four missing and it reads as two
  numbers swapping rather than one setting off. It also does not fade *in*, for the same reason.
- **It shrinks where the sum's items grow**, and the difference is the meaning: a term flying into
  the sum comes toward the reader, a total flying into a card goes away into it.
- **What the bar holds is read off the combatant, never worked back from the event** *(2026-08-19,
  bug)*. It was `e.Life + e.Amount` — what was there, from what is left plus what was dealt — which
  is right for every blow except the one that ends a duel: `e.Life` is clamped at zero, so overkill
  makes that arithmetic return the *size of the blow*. A pair of Cleaves for 60 on an enemy holding
  30 of 90 drew `60/90` for the length of the flight and then emptied — health visibly going *up*,
  on the killing blow and nowhere else. `applyEvent` reads the life before it overwrites it and
  hands it to `noteHit`.
- **`Init` takes the whole theatre down**, which is the lesson the frozen last round taught — anything
  tidied up only by the end-of-round spend assumes every round ends in one, and a settled duel does
  not.
- **`shownLife` walks the list although there is only ever one figure owed.** The cursor holds for a
  whole flight, so a second `KindDamage` cannot be reached while the first is up — but relying on
  that would break the day the hold is shortened, and it would break as a bar showing a life nobody
  has, which is hard to attribute.

### A Prepare's points, and the AP line that waits for them

*`combat_bank.go`, 2026-08-19, owner's call.* `+2 AP` flies out of the card that banked it and into
the fighter card whose budget it raises, and **that card's AP line goes up as the figure lands**.
It is the `KindGathered` row of the choreography table and the damage figure's argument applied to the
one card in the game whose entire effect is a number changing somewhere else: until this, a Prepare
resolved with a lift in its own seat, a sentence in a log nobody had open, and a budget that
silently read two higher at the start of the next round.

- **The target is the fighter card, not the strip's AP figure**, and the theatre row moved with it.
  `3/6 AP` under the bar is *this* round's budget being spent and a Prepare does not touch it; the
  line a Prepare changes is the card's, which is the live budget with `BonusAP` in it. A figure
  landing on the strip would arrive at a total that does not move. `anchorAPFigure` stays in the
  enum with nothing pointing at it, for whatever next raises or spends the current round's budget.
- **`shownBank` is a view, exactly like `shownLife`.** The engine counts the points into
  `GatheredAP` the moment the event resolves and turns them into `BonusAP` when the round's end
  state is adopted — unchanged. What the screen adds is the same figure arriving early, so the line
  moves when the number reaches it rather than a whole opposing turn later. `endOfRound` zeroes it
  on the frame the adoption happens, or the two points would be counted twice.
- **`duelistSpec` takes the AP to draw as an argument**, for the reason it takes the life: the
  card's cache keys on the whole `Spec`, and the figure has to be able to differ from what the
  combatant would answer.
- **It stops the playback cursor**, like a landing damage figure and for the same reason. Pacing
  only; the round was decided before a frame of it was drawn.
- **It grows into place where the damage figure shrinks.** Points being *added* to a fighter arrive
  at full size on the card they join; a total flying into a card goes away into what it empties.
  That is `gestureFly`'s own description, and the hit is the documented exception to it.
- **The credit outlives its figure.** A flight is dropped when its hold expires and the points stay
  on the card until the adoption; `combat_bank_test.go` pins that, the credit landing on arrival
  rather than on launch, and `combatTheatre.clear` from `Init` taking both down.
- **An enemy banks the same way and its card has no AP line to raise**, so the figure flies to the
  enemy card and lands on nothing. That is the anchors-are-named-by-role rule biting rather than an
  oversight: the alternative is this file growing an opinion about which side is a person.

### The theatre: what travels, out of what, into what

*`combat_theatre.go`, 2026-08-18.* One table, one entry per `EventKind`, naming a source anchor, a
target anchor, a gesture and the reason. **It is the map, not the machinery** — the drawings stay
with the code that already owns each region, because a generic renderer over nine genuinely
different gestures would be more machinery than the nine gestures.

- **The rule it holds is one sentence: everything travels from the thing that caused it to the
  thing it happened to.** A figure appearing in the middle of the screen is a figure that was never
  anywhere else, so the player has to be *told* what it was instead of having watched it happen —
  the card-flight argument applied to every event. Written down as a table the rule is checkable;
  written into nine call sites it is a habit.
- **The checkable part is completeness.** `TestEveryEventKindIsChoreographed` fails when a new kind
  arrives without an entry. **That matters more since the Resolution feed went behind a button**
  *(2026-08-18)*: with no running commentary on screen, an event nobody drew is an event the
  player is only ever told about by opening the log. **A kind with no picture says so with
  `anchorNone` and a reason** — an absent entry and a deliberate silence otherwise read
  identically.
- **Anchors are named by role, never by side.** `anchorActorSeat` is the acting side's row whichever
  side that is, for the reason `SoloAttacks` is a flag on a duelist rather than a rule about
  `SideB`: the engine has no idea which duelist is a person and this screen must not grow a second
  opinion about it.
- **`Event.Ring` exists because of one row.** `KindStatus` flies out of the ring that caused it, and
  nothing on screen could say which ring that was — reading it off the card's element would be a
  second rule about something the ring grammar already decides, and wrong the first time a form
  or concept ring applied a status, both of which `RegisterRing` accepts today. `statusesFrom`
  always knew and used to throw the answer away; the first ring worn is the one credited.
- **It is deliberately not JSON** *(owner asked)*. Every anchor is a geometry function that takes
  arguments — `ringSlotAt`, `enemySeatAt`, `enemyCardRect` — and a file can only name one by string,
  so the Go table a file would need underneath it is this table. `data/*.json` is `//go:embed`ed
  too, so a beat in a file costs the same rebuild as a beat in a constant. **If the timings turn out
  to be what gets edited over and over, lift the timings out and leave the anchors** — timings are
  data, anchors are code.
- **Most rows are not drawn yet, and that is the point of writing them down.** What travels today
  is `KindDamage` and `KindHand`; the kinds marked `anchorNone` are done by being silent, and
  `KindAction`'s drawing is the card lifting in its own seat, which `tableFireLift` already does.
  Everything else — a banked point landing on the AP figure, a status flying off its ring, a chill
  acting from the badge row, a missed turn struck out — describes what the gesture should be when
  someone builds it. The log writes a sentence for each of those; none of them moves — and with
  the feed gone, a sentence in a panel is the only place they are said at all.

### What survives any model

- **`combat.ResolutionOrder` is the single authority on order.** `ResolveRound` plays what
  it returns and the table draws what it returns. Neither derives the order
  independently, which is what makes it structurally impossible for the pane to lie to the
  player about their own round. `TestResolutionOrderIsWhatResolveRoundPlays` pins it.
  **A chill is the one thing that can remove a slot** rather than reorder it, and the
  every-slot-accounted-for test above is what keeps that honest.
  **This is what made the phase change cheap** — one pure function body plus its tests, and
  both consumers followed untouched. It paid for itself exactly as predicted.
- **Ordering is a rule.** It belongs in `internal/combat`, never in a screen. A new effect
  that rearranges resolution changes `ResolutionOrder` and both consumers follow.

### Defense expiry is a rule about turns, not about order

**A defense expires at the start of its owner's next turn, not at the round boundary.** Side B
acts last, so a defense cleared at the boundary would protect B from nothing it ever faces.
Expiring at the owner's next turn means every defense covers exactly one opposing turn
whichever side raised it.

This lives in `ResolveRound`, **not** in `ResolutionOrder` — a side that queues nothing still
has a turn and still loses its guard in it, so it cannot be derived from the slot list. It is
the one place the engine's symmetry needed defending against an order that is not symmetric.

The old rule — a Guard lasting until its owner's next *action*, so an idle duelist kept it
forever — is gone, along with `TestGuardHoldsWhileItsOwnerDoesNothing`.

## Combat screen layout is scaffolding

The screen reads top to bottom rather than left to right: **who you
are and what the round is doing** above, **the cards you are doing it with** along the
bottom. Colours identify the role and are placeholders, not a chosen palette.

| Element | Slot | Colour | Role |
|---|---|---|---|
| Duelist card | 1% x, 2% y | — | who you are: name, DMG, AP, Vitae, health |
| Ring row | between the two cards, 10px below their top | pink borders on grey | what you are wearing |
| Enemy card | right edge at 99% x, 2% y | — | the opponent |
| The table | full width, under the top row | element | both queues as cards, player left, enemy right |
| The math band | table width, above the hand | — | empty at rest; the previewed hand's name, then the blow's arithmetic |
| Fight log | a dialog, 4–96% | pink | every round of the fight, behind the Log button |
| Hand | centred in what the sort column leaves, 66% y | element | the cards, portrait, in one row |
| Sort column | band's right edge, centred on the cards | slate | `$` / `T` / `E`, the hand's arrangement |
| AP bar | hand width, directly under the row | blue | the budget |
| Bottom strip | 95% y | — | AP figure, Discard, DUEL!, Log, deck pile — evenly spread |

**The top of the screen is one row of three things**: the duelist card in the
left corner, the enemy card in the right, the rings filling everything between.

- **The ring row takes both its edges from the two cards** rather than from percentages —
  `ringPaneRect` reads `duelistCardRect` and `enemyCardRect`. That replaced a hardcoded 79%, a
  percentage standing in for the position of a card it could not see, which went stale the day
  after it was written when the enemy moved to the corner.
  `TestTheTopRowIsThreeThingsThatDoNotOverlap` is what keeps it honest.
- **The rings sit on a plain grey backing and drop 10px below the two cards.** The two go
  together: with the row spanning most of the screen and a card at either end, nothing said
  where the middle began — but a backing whose top edge lands on both cards' top edges makes
  the three read as one wide object with two cards embedded in it, which is the *same* failure
  the framed pink version was retired for. The drop breaks that line.
- **A fill, never a frame.** One step darker than the screen's cream `screenGround`, no border,
  no title, no hue — a colour that meant something would compete with the five saturated pink
  borders standing on it. `ringPaneBackRect` is derived from the row and pads it by 8 against
  a 16px gap, so **the backing can never touch either fighter card**;
  `TestTheRingBackingHoldsTheWholeRowWithoutTouchingTheCards` pins that and the fact that it
  is deep enough to hold the `3/5` fraction, which belongs to the row it counts.

**Cards are portrait and live along the bottom.** Landscape cards in a vertical column
capped how many could be shown, and the hand is going to grow. `cardWidth`/`cardHeight` are
**flat constants — 162x224 — and must stay flat**: they
used to be derived from the glyph
row, so adding a badge silently widened every card and the layout could not be reasoned
about without doing the arithmetic. Contents fit the card, never the reverse.

**`handBand()` is the single authority on the hand's horizontal extent.** The card slots
are cut out of it, the AP bar spans it and the caption box matches it, so the three cannot
drift apart when the hand size changes. A card in flight still owns its slot, which is what
stops the row sliding half a card sideways when one is lifted.

**The strip at 95% is one row of five things, spaced rather than placed**: the
AP figure at the hand's left edge, Discard, DUEL!, then the Log button and the deck pile in the
corner. The two
buttons used to sit at 20% and 33%, side by side because they were the same choice made two
ways. **They are separate choices now and the spacing says so** — `buttonStripSlots` divides
what is left between the figure's column and the **Log button** into three equal gaps and puts a
button
in each of the two spaces, so the strip stays evenly spread if any of the three fixed things
moves. `TestTheButtonStripSharesItsSpaceEvenly` checks the gaps against each other rather than
against numbers, because the property wanted is the relationship, not a coordinate. Discard
still carries one condition DUEL! does not: a round's discards can run out.

**The right-hand end of that span is the Log button, not the pile** *(2026-08-18)*. It arrived
between the two, and a strip still measured to the pile would spread its buttons across a span
with a control standing in it. **The two corner controls are a pair**: both are things you open to
look at, as against Discard and DUEL!, which commit a round — which is also why the Log button
takes the sort column's slate rather than a committing colour.

**The bottom of the screen is one line**, and two of the three things on it are
now hung off the bottom edge rather than off the hand's geometry:

- **The deck pile's anchor moved** from "down from the AP bar" to "up from the bottom edge",
  `deckStackBottomInset = 10`. The bar was the constraint while the strip below it was 86
  pixels and a 54-pixel pile only just fitted; the hand came down and left slack, so measuring
  from above left the pile floating in the middle of it. Both of its corners are margins from
  the screen's own edges now, which is what it wanted to say all along.
- **The mute button uses the same inset**, so the two share a bottom edge exactly. It is chrome
  in `internal/game`, which imports this package and cannot be imported back, so the number is
  shared by *being the same number* — checked from both ends, by
  `TestTheBottomOfTheScreenIsOneLine` here and `TestTheMuteButtonSitsInTheBottomLeftCornerOnScreen` there.
- **The discard badge is four pixels lower**, because it hangs off a button strip placed as a
  percentage. Four pixels reads as one line; making it exact would mean taking the strip off
  percentages for no other reason, and the test allows the slack rather than pretending.

**The hand sits at 66% because that is what lines the AP figure up with the buttons**
. Dropping the pile freed the band it used to float in, and the row came down
into it until the action-point figure's top landed on the Discard button's top. It is a
coincidence of five constants — `handTopPct`, `cardHeight`, `apBarBelow`, `apBarHeight`,
`apFigureBelowBar` against the strip — and nothing in the code enforces it, so
`TestTheAPFigureLinesUpWithTheButtonStrip` does. **What the drop buys is height at the top**:
the bar, the figure, the cards and the band above them all measure off this row, so moving it
moves the whole lower half together and opens the space between the fighter cards and the hand.

**`apFigureReserve` is a fixed column width, not the figure's measured width.** Measuring
would move both buttons the moment the text went from `9/12 AP` to `10/12 AP`. The reserve
holds the normal figure and the `+N over` tail runs past it, into a gap hundreds of pixels
wide. The buttons read `handSize` through `handBand`, never the live hand, for the same
form of reason: `handBand` is centred, so a shorter row starts further right.

**The discards left this round are a badge on that button** — a filled disc **centred on its
bottom-right corner**, count inside, drawn by `drawDiscardsLeft` after `systems.DrawButton`
because the button blits an opaque cached face. A number you watch tick belongs on the control
that ticks it, not at the other end of the screen. Three things about it:

- **Centred on the corner, not inset inside it**, so it hangs off both edges and reads as
  attached to the button rather than printed on it. Nothing sits under that corner — the hand
  row ends well above the button strip — so the overhang costs nothing.
- **The scene draws it, not the widget.** `models.Button` is shared by every screen and holds
  one centred string; a corner-badge field would put a rule about this screen into all of them.
- **Two fill/ink pairs, not one dimmed.** The face underneath does not dim, it changes colour
  entirely — `disabledButtonColor` is flat grey and ignores `BaseColor` — so a badge tuned for
  the yellow face has nothing to say about the grey one.

**There is no Deck button any more**. The draw pile itself is the control: a
stack of card backs at the right-hand end of the strip, clicked to open the overlay and
clicked to close it, wearing a **bright yellow ring while the overlay is up**. That ring is
load-bearing, not decoration — see the overlay's "make its exit the brightest thing on screen"
rule below, which the old button satisfied for free by being lit on a dead screen. It lives in
`combat_flight.go` along with the cards that fly out of it.

**Its y is measured up from the bottom edge**, and never as a percentage. It was
measured *down from the AP bar* until then, and both anchors were chosen the same way — name
the thing that actually constrains it. The bar was that thing while the strip below it was 86
pixels and 95% of the screen height put the pile three pixels *through* the bar; the hand has
since come down and left slack there, so the bottom edge is. `cards.Stack` is small because of
the original squeeze and stayed small. `TestDeckStackClearsTheAPBarAndTheScreen` still holds
the ring against both the bar and the screen, so neither anchor can quietly stop working.

**Its x is measured in from the right edge, and the margin is where its count is written**
. `deckStackRightMargin` went 10 → 96 so the fraction sits beside the pile rather
than floating in the corner. **The count is `left / owned`** — `len(s.deck)` over `deckSize()` —
and the denominator deliberately never moves: the numerator alone says how far through the deck
the round is, and the discard is the subtraction (owned, less what is left to draw, less the
hand). It replaced `deck 45 · discard 7` on the line under the hand, which is gone.

**A won fight leaves by itself, and there is no control at all while it does** *(2026-08-19,
owner's call)*. `holdVictory` counts `victoryHoldTicks` from the last drawn event and then hands
over to the post-battle screen; `victoryPending` is the predicate, and **the DUEL! slot is not
drawn while it is true** — a `Next` standing lit under a screen that is about to change on its own
is the offer of a choice the player has not got.

- **The hold is the design and only the press was dropped.** The screen freezes a settled duel on
  purpose (see the freeze below), so leaving on the frame the last figure lands would throw that
  picture away. `victoryHoldTicks` is the one number to move if the pause reads wrong.
- **The count stops while a dialog is up.** The fight log can be opened on a won fight, and a
  screen changing out from under an open panel is reading material snatched away.
- **A defeat is untouched.** Retry keeps its button, because playing the same fight again is a
  decision and it is the player's.

**DUEL! becomes Retry when a duel is lost**, changing its own label rather than a fourth button
appearing. Same slot, same meaning — commit and move the game
forward — and a control that only ever showed up at the end of a fight would be one nobody
had learned. `duelSettled()` gates it on playback having *finished* as well as someone being
dead, because life reaches zero partway through the log and offering the exit before the
killing blow is drawn would cut the round short at its best moment.

The caption changes with it, naming the next enemy. Before this, winning left every button
dark with no way to play on short of restarting the process.

**Selection having two verbs is deliberate.** There is no discard mode and no second
gesture. One selected set, two things you can do with it, which is why the two buttons are
adjacent and why the action points come back when a card is discarded — the selection was
proposed, not spent.

**Selected is the only thing that ever leaves the hand**, by either verb. `spendSelected` is
the single movement — the selected cards go to the discard pile, the draw tops the hand back
up to `handSize` — and both Discard and the end of a round call it. Two functions doing this
would be two functions that have to agree.

### The table: two hands facing each other

*`combat_table.go`.* Pressing DUEL! lays both queues out across the middle of the
screen at full size — **the player's played cards left-aligned, the opponent's queued cards
right-aligned** — and clears them when the round ends. It is the first thing on this screen
that shows a round as a confrontation rather than as a list.

It replaced a pile of played cards in the bottom-left corner, where a card rose
out of its slot, held below the Resolution pane to be read, and stacked in the corner. That
pile was legible and it was history *only*: it grew with nothing opposite it, so the one thing
a duel is about — my cards against yours — was something the player had to assemble from a pane
of sentences.

- **Both rows come from `combat.ResolutionOrder`**, so both say what *will* happen rather than
  what was planned — a queue planned attack-first resolves prepare-first, and a row in
  selection order would be a confident picture of a round that never happens.
- **The whole queue is dealt at round start, not a card at a time.** The opponent's hand is
  known in full at that moment, so a player's row assembling itself over the next few seconds
  would be one hand against half of another. **Playback drives which cards are lit, not which
  cards exist** — `firingSeats` and `enemyFiringSeats`, written by `noteResolved`.
- **A prepare or a defend lights its own seat; the attack hand goes up all at once**
  *(2026-08-15)*. A turn lands one blow and the blow is the set, so the first attack announcement
  raises every attack card of that turn and `noteHand` then drops whichever earned nothing. They
  used to climb one per beat, which read as one attack per card — the model this replaced. Only
  one *side* is ever lit: the event that lights one clears the other. **A solo attacker climbs one
  per beat and should** *(2026-08-17)*, because one attack per card is exactly what its turn is.
- **`noteResolved` and `seatPlayedCards` count along the same walk.** The third card to resolve
  is not the third card in the hand, and two independent tallies would light the wrong one the
  first time somebody queued a defense before an attack.
  `TestSeatingWalksTheSameOrderAsPlayback` is what replaced the safety the old per-event pile
  had for free.
- **A resolving card lifts in place** (`tableFireLift`) rather than flying to the middle to be
  read. The middle is where the opponent's hand is now, so the old hold beat would send a card
  across the cards it is being played against. The lift borrows selection's gesture from the
  hand row.
- **The two rows never touch.** Five full-size cards a side do not fit a screen, so each row
  overlaps *within itself* and the pitch is derived exactly as `handPitch` is — the action cap
  is a rule a ring is expected to raise, so a pitch that only worked for five would hide a
  rendering bug behind a balance change. `TestTheTwoHandsNeverReachEachOther`.
- **The opponent's row is up during planning, and that is what the row is for**.
  It appeared only at DUEL! for one day, because `enemyActions` held *last* round's plan until
  then. **The fix was to move when the opponent commits**: `planEnemyRound` runs at the start of
  the planning phase — from `Init` for round one and from the end of playback after that — so
  the player picks their round against a hand they can see.
  **Nothing about the plan changed, only when it is shown.** `PlanFor` never sees the player's
  queue and the opponent's state does not move between one round ending and DUEL! being pressed,
  so the same cards are chosen either way. `startRound` must never re-plan; if it did, the cards
  the player chose against would not be the cards they faced.
  **Ordering that bit once**: in `Init` the plan has to come *after* the life reset — the
  function refuses to plan for a dead duelist, and a screen re-entered after a defeat still has
  a corpse on it until then, so planning first dealt the next fight an opponent with no cards.
- **The duel is open-information now, on the owner's call.** `concealEnemy` is still the screen's
  concealment predicate, and the table deliberately ignores it — with the opponent's cards face up
  there is nothing left for it to hide. **The lever is still built**: `cards.Spec.FaceDown` draws a
  back and the draw pile is a stack of them, so hiding this row again is a field rather than a
  second drawing path.
- **The opponent's cards fly in from the enemy fighter card** in the top-right corner — the
  opponent itself, and the mirror of the player's cards coming out of their hand. There is no
  enemy draw pile on screen; inventing one would be a second thing to explain, where a card
  coming out of the thing that *is* the opponent needs no caption. Same `riseTicks` and same
  `flightStaggerPer` as the player's row, and `TestBothRowsUseTheSameArrivalClock` is what stops
  a later change to one being made twice.
- **Every opponent card is elementless**, because every enemy's deck in `data/enemies.json` is
  authored `basic`. That was a fact nobody could see when an enemy card was never drawn; it is
  visible now, and the neutral grey border is the truth rather than a placeholder. It stays true
  until affixes exist: an enemy's colour does nothing without one, so a coloured enemy card would
  be a border claiming a rule.
- **A played card has not left the hand.** It leaves at the end of the round with everything
  else played, which is what keeps the Resolution pane able to narrate from `fighterActions`
  while the round is still running. `resolvedInHand` hides a *drawing*, exactly like
  `inboundTo`.

**Four things move, and they share a clock and nothing else** — `travel`. A card to or from
the draw pile, one of the player's to its seat, one of the
opponent's to theirs. The obvious unification — one struct holding a start and an end point —
was **rejected**: no mover stores its endpoints. Every one recomputes both every frame from
`slotAt`, `playedSeatAt`, `enemySeatAt` or `deckStackRect`, which is what makes a flight survive
the row re-laying out underneath it and survive a resize. What they genuinely share is a delay,
an age, a duration and an eased progress, so that is all `travel` is. The *gestures* stay
separate, because they are genuinely different drawings: the discard accelerates away lifting,
turning and shrinking; the deal scales up out of the pile and flips face up; the two table rows
travel flat.

**The animation does not own the card, and this is the rule to protect.** `spendSelected`
completes the whole logical move before it raises a single flight, so a card in the air has
already gone: the hand, the piles and the queue are correct the instant it returns. The
tempting alternative — holding a card in its slot until the flight lands — would make
"is this card in the hand" a question `planning()`, the AP budget and `handBand()` all have
to answer. What the row does instead is skip *drawing* a slot a card is still flying into
(`inboundTo`), which is a view concern and stays in `combat_flight.go`.

**The hand persists across rounds**. It used to empty completely every
round and deal a fresh eight, justified by "a hand kept back would let a plan be prepared once
and repeated". That conflated the queue with the hand: the *queue* still empties every round,
so no plan repeats by default, but taking away cards the player had deliberately held punished
holding them and made playing a card the only way to keep it.

The consequence to hold on to: **Discard is now the only way an unwanted card leaves your
hand.** `discardsPerRound` stopped being a convenience and became the rate at which a hand can
be steered.

**A settled duel does not spend the hand at all — it freezes** *(2026-08-16)*. `endOfRound`
adopts the end state and then returns if `duelSettled()`, so the last round is never cleared
away: the played cards stay on the table, the hand keeps the gaps they left, and `Init` clears
all of it when the next fight starts.

**What forced it is that the row is a measurement, not just a picture.** `handBand` is a function
of how many cards are in the hand and the AP bar spans that band — so spending the hand after the
killing blow collapsed the cards into a narrow centred huddle and dragged the bar in with them.
**The picture the player is looking at when the blow lands is the picture they should still be
looking at while the screen holds it.**

The intermediate version — refill nothing but still spend — does not work and is worth not
re-trying: a hand that loses cards reflows whether or not it is topped back up. What has to stop
is the whole end-of-round movement, which is why this is a branch in `endOfRound` rather than a
rule inside `drawHand`.

**What the freeze cost, and it is the shape of thing to look for again.** `s.theatre.resolved` used to be
emptied in exactly two places — `seatPlayedCards` at the start of a round and `spendSelected` at
the end of one — and that covered every case *only because every round ended in a spend*. With the
last one frozen, the winning hand was still seated when the next fight started: it drew over the
new table, and `resolvedInHand` blanked the hand slots it claimed, so the fresh hand came up full
of holes. **`Init` clears the table now** — `resolved` and `enemyDealt` both — which is where a
per-fight reset belonged anyway. Anything else that is only ever cleaned up by the spend is now
sitting on the same assumption.

**Both fighters are cards, in opposite corners.** The argument for it: everything the duel is
made of is a card, including both the
people playing it.

- **`cards.DuelistStyle` and `cards.EnemyStyle` are twins and must stay so.** Same footprint,
  and **the health bar and the fraction under it are at identical offsets on both** — the two
  cards face each other across the screen, and a bar at a different height on each would turn
  comparing them into an act of measurement. `TestTheTwoFighterCardsShareTheirHealthGeometry`
  pins it. Above the bar they differ, because that is where they say different things: a
  portrait on one, three stat rows on the other.
- **The duelist card holds name, DMG, AP, Vitae, bar, fraction.** DMG is `Strike.Damage(DMG)`
  asked of the rules rather than the field copied out. **That is an identity today** — the stat
  was renamed off `Str` on 2026-08-16 precisely because the middle rung of the ladder returns it
  unchanged — and the call is still worth making, because it makes the figure follow the *ladder*
  rather than the field. AP is the live budget including a banked Prepare. Vitae is still a fixed
  placeholder with no rule behind it.
- **`Spec.Stats` is a fixed array, not a slice, and that is load-bearing** — the screen's card
  cache keys on the whole `Spec`, so it has to stay comparable. `cards.MaxStatLines` is what
  the layout fits rather than headroom over it: a fourth figure lands on the health bar and
  `TestStatRowsClearTheHealthBar` fails rather than drawing it.
- **The enemy card carries the statuses standing on it**, as a row of badges along the bottom
  edge from `assets/effect/` — flame, snowflake, bolt, and a placeholder for earth. **It is the
  only thing on screen that says a status is on**: a chill takes a card off a turn not yet
  queued and a weight blunts a blow not yet swung, so without a badge either is learned only by
  being surprised by it. Twenty pixels, in the strip the life fraction leaves above the border,
  and `TestStatusBadgesClearTheHealthTextAndTheBorder` holds both ends of it. **The row is
  centred and closes up**, like the ring row.
  - **The badge appears on the beat the status lands**, not when playback finishes.
    `applyEvent` writes it from `KindStatus` — a drawing, overwritten a few frames later when
    the two duelists adopt `s.enemyAfter`. Same argument as the burn: a card that disagrees with
    the sentence beside it.
  - **The duelist card has no badge row**, and that is not an oversight of the twins rule — the
    bar and the fraction are still at identical offsets, which is where that rule bites. Nothing
    can put a status on the player: an enemy wears no rings, and a ring is what makes a status
    happen. `DuelistStyle` gains the three fields when one can.
- **The enemy names itself above its portrait**. The name sat between the
  portrait and the bar until then; every other card in the game carries its name across the
  top, and a card with its own reading order reads as a different kind of object.
- **All of it is one cached image from `internal/cards`**, health bar included, so
  `tools/cardsheet` draws a wounded fighter without reimplementing a bar. `Life`/`MaxLife` are
  on the `Spec`, so a point of damage is a new cache entry — affordable because life changes
  on damage events, not per frame. The duelist's figures work the same way.
- **Red is what is left, not what is lost.** A bar where the red grows as the enemy weakens
  says the opposite of what it means.
- **Neither card carries an element**, so both borders are the neutral mid grey. If the two
  corners ever need telling apart by colour that is one entry in `cards.Element`, not a
  change to either style.
- **There is no loose sprite anywhere any more.** `drawCombatant`, `DrawHealthBar` and
  `Combatant.Sprite` are gone, and `entities` imports no Ebitengine at all.
- **Discards left lives on the Discard button, not on the duelist card** — see
  `drawDiscardsLeft`. The card holds what is read between rounds, not what is watched during one.

**The deck overlay is a dialog, and one of two in the game** — the fight log below is the other. It fills nearly the screen,
everything behind it goes dead, and `Draw` renders the Deck button *again* on top of the
overlay so the single live control is the only one that looks live. Pressing it closes it.
There is no Escape key to fall back on and no right click, so a modal has to make its exit
the brightest thing on screen or it is a trap.

- **It draws the cards, not a table of counts.** A count cannot say which of six Strikes
  are fire, and with elements on the cards that is most of what the panel is for.
- **One grid holding both piles, discarded cards dimmed.** Sorting by kind and element
  rather than by pile means **a card does not move when it is discarded, it only dims** —
  the deck draining in place reads better than cards jumping between two lists.
- **Sorted, never in pile order.** Drawing the shuffled draw pile in order would hand the
  player their next five cards. The old counts-only version dodged this by construction; the
  sort is what replaces that protection.
- It draws a `+N more not shown` line rather than silently dropping overflow. It cannot fire
  at twenty cards, but deckbuilding will grow the deck and a panel that quietly hid cards
  would be a picture that lies about what you own.

### The fight log, and the second dialog

*`combat_log.go`, 2026-08-18.* A 44px square marked `L` beside the draw pile, sharing its bottom
edge, opening a panel with every round of this fight written out in sentences.

**It replaced the Resolution feed rather than joining it**, in two steps the same day: the log
first, then the feed removed once it was clear the log held what the feed had been holding. That
feed showed one round, cleared at the start of the next and had no scroll gesture to reach an
earlier one — so a fight's account was something the player had to have been watching. That was
survivable while it was the only thing narrating a round. It stopped being that: the table shows
both hands, cards fly to their seats, the sum is acted out and the damage figure travels into the
bar it empties. Once the pictures say what happened, a running list of sentences under them is a
second telling costing a band of the screen, and what the sentences are *good* for is the thing
the feed could not do — being read back.

- **The walk was not rewritten.** `logRows(events)` is the feed's own walk: the prose, the merging
  of an action with its outcome, the one line per slot. It knows nothing about capacity, overflow
  or which row is live — those are properties of the box, and they stay with the caller. **That is
  why a round reads here exactly as it read while it was happening.**
- **The round in progress is included only as far as playback has reached**, the same slice the
  feed drew and the same protection: the dialog can be opened mid-round, the resolved round is
  sitting in `s.log` in full, and a log built from all of it would hand the player the rest of the
  round they are watching.
- **`CombatScene.rounds` holds events, not finished lines.** Storing sentences would freeze a
  fight's account against the wording of the day it was played, and would be a second copy of
  something the events already say. A round moves into it in `startRound`, as `log` is about to
  stop being the current round. Appending at the end of playback instead would double it:
  `log` still holds the finished round through the planning phase, and `fightLogRows` reads both.
- **Headings belong to the caller.** `logRows` still ignores `KindRoundStart`/`KindRoundEnd`,
  because only the caller knows whether a round has anything before it. A heading is a row with no
  swatch and no verb, which `drawPane` centres.
- **The two dialogs are mutually exclusive rather than stacked.** Each one's button is dead while
  the other is up, so there is no way to open the second without closing the first — and each
  survives its *own* overlay, because it is the only thing that closes one. `modalUp()` is the
  single predicate every other control gates on; a button left live under a dialog is a round
  edited through a panel the player is only reading.
- **It takes the deck overlay's footprint and the Resolution pane's colours.** Two dialogs at two
  sizes would read as two kinds of thing, and the off-white ground is what makes the coloured verbs
  legible — the reason Resolution went light in the first place.
- **What it does not do yet:** the log keeps only the newest rows a full panel can hold and reports
  the rest as `... N earlier`, and it is a *fight* log rather than a run log — `Init` clears it.
  Both are in `TODO.md`. Neither is fixable inside the panel: the first wants a scroll gesture the
  input vocabulary has not got, the second wants the rounds moved onto `session.Session`.

### The round narrates itself in pictures; the log holds the sentences

*Split, then narrowed, then emptied.* The screen carried two live panes at once — one for what you
had **queued** and one for what had **happened**. The first was dropped from `Draw` on 2026-08-07
and deleted on 2026-08-21; the second was removed on 2026-08-18, when the fight log arrived to hold
what it had been holding. What draws these rows now is the log, and nothing else does.

**What is given up, and it is not nothing:** the enemy's queued shape during planning, and a
running account of the round as it plays. The first matters less than it did, since the table has
drawn the opponent's cards face up since 2026-08-12; the second is the trade the owner took — the
table, the flights, the hand dialog and the travelling damage figure narrate a round in pictures,
and the sentences are for reading back.

**The prompt went with it.** `(press DUEL!)` was a line in the feed, and nothing carries it now;
the button says DUEL! on its face, which is what has to be enough. **Nothing on screen says why a
dark DUEL! button is dark** either — the AP bar going red says something is wrong, not what to do
about it. That was already true and is now the only thing saying it.

**A hand never has to be marked across non-adjacent slots**, which was the open problem for as
long as a walking highlight down one row per slot was the only account of a round: the table keeps
the cards that earned it raised, and the log says it in words. Same for a slot a chill deleted.

**Short labels and sentences are not interchangeable**, which is why the log is a full-screen panel
rather than a column: `Strike` fits a quarter of the width and "Duelist lands a Card Three of a
Kind" does not.

Four rules survive into the log, and they are the ones to protect:

- **One line per slot, not one per event.** A busy round is 25–30 events. Merging an action with
  its outcome is presentation of events the engine already decided; it computes nothing.
- **Hands and chills get lines of their own**, because they are not something a card did.
  Folding a hand into the line of the card that happened to start it would bury the one thing
  worth reading.
- **Built only from events playback has reached**, so nothing ever spoils the rest of a round in
  progress. It was `s.log[:cursor+1]` in the feed and it is the same slice in `fightLogRows`.
- **Overflow is reported, never silent** — `... N earlier`, the same rule as the deck overlay's
  `+N more not shown`. A panel that quietly hides part of what it claims to show is a picture that
  lies.

**The caption stopped narrating long before the feed went** and there is no caption box at all
now. It used to show one event at a time, which meant the whole account of a round existed only as
a quarter-second flash.

`panePlacement` carries its own `rowHeight` because the two panes hold different things:
`paneRowHeight` (30) for card names, `paneTextRowHeight` (22) for sentences.

### The log writes sentences, and the verb is marked in the text

*A line is `<who> <verb> <phrase>` — **"Duelist attacks with a
heavy strike"** — and the verb is **coloured, bold and underlined**: **red for attack, blue for
defend, the row's own ink for prepare**. A round can then be scanned for what *kind* of thing
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
  colour for the side and for the live row, and bold alone for the live row — so the verb needs
  the combination to be unmistakable. **The consequence: the underline is no longer what marks
  the live row.** That row is now distinguished by `nowInk` plus faux-bold on `prefix`/`suffix`,
  and the verb keeps its category colour there rather than going pink with the rest.
- **Prepare has no hue and inherits the row's ink.** As a chip it needed a pale ground *and* a
  near-black foreground, because white-on-white is invisible. With no ground there is nothing for
  a pale colour to be legible against, so it takes `p.ink` — already the colour that reads on that
  pane, whichever way round it is painted. Being the category with no colour is also its
  right rank: it is the one that does nothing to the opponent. `verbInkFor` returns **zero alpha**
  to mean this, the same "use the default" convention `Button.BaseColor` uses.
- **The underline sits flush with the bottom of the measured line box**, never a constant above
  it. It used to hang under a chip of fixed height; with no chip the only thing to position it
  against is the text, and `text.Measure` reports the full line including descent — which is what
  clears the `p` in "prepares". A rule placed a few pixels above the baseline struck through it.
- **The prose lives in `internal/screens`, not `internal/combat`.** The rules package names cards;
  it does not describe them. **It is generated from the verb rather than tabulated** *(2026-08-16)*
  — `actionPhrase` and `cardEffect` in `prose.go`, switching on `Verb` and dropping the
  card's own label in as the noun. There were two hand-maintained maps, one string per concept,
  which worked for fourteen concepts and cannot work for the ~400 that per-enemy decks produce: a
  card with no entry drew a blank face. **Every phrase carries an article** so `cardPhrase` can
  slot an element into it — that is a constraint on any new wording here.
- **Outcomes append to `suffix`**, after the verb, so the mark never moves as a line grows.
- **The name is said as well as coloured, deliberately.** The swatch already encodes the side,
  but a line beginning "Strike" reads as an instruction rather than a report, and with both
  sides in one list the reader would have to hold which colour is which.

**Row highlights and swatches are centred on the measured line height**, not offset from the
row top by a constant. The old `rowY-4` / `rowHeight-2` numbers were picked by eye against a
single 30px pitch and clipped the text the moment a 22px pitch existed. `text.Measure` once per
pane, centre everything on it, and any pitch works.

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
the screen reads as two colours: green is you, yellow is them. `handSwatch` amber is the third
and marks a Resolution line that is not a card acting — it belongs to whoever formed the hand,
but the line is an announcement, and giving it a side's colour would file it in the column of
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
  It is a method on `Duelist` so a ring raising it has somewhere to bite.
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
column against the band's right edge, centred on the cards: **`$` cost, `T` type, `E` element**.
The active one latches darker than the other two.

- **Nothing here can change a round.** Cross-category order is regrouped away by
  `ResolutionOrder`, a hand is counted rather than read in sequence, and defends are a set. This
  is entirely about reading eight overlapping cards, and it is the reason the feature could be
  taken without asking what it does to the engine.
- **Cost is the default, and every mode ends with it.** Each arrangement is the deck overlay's
  own key chain — cost, form, concept, element — with one key promoted to the front, so a row
  of cards means the same thing in the hand as in the panel. Only the leading key differs.
- **The sort re-applies on every refill**, in `spendSelected` *before* anything is animated, so a
  dealt card flies to the slot it will actually occupy. A drag still works and survives until the
  next deal, at which point the sort reclaims the row.
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
  its enum as the zero value and trails on screen: the colours are what the statuses are counted
  on, and the colourless cards are the plans.
- **The cards lost width to pay for the column.** `cardBandWidth` is the band less
  `sortColumnReserve`, `handBand` centres on *that* rather than on `PctX(50)` — so the whole row
  nudged left instead of only its right edge coming in — and `handBandLeftPct` came in from 4% to
  2% to find some of the overlap back. The AP bar and the AP figure travel with it, both being
  measured off `handBand`.
- **`models.Button.Latched` and `TextSize` arrived for this** and both are deliberately general.
  A latched button takes `ButtonStateLatched`, drawn at **38% — darker than resting, not
  brighter**: hover and press own the bright end of the ramp, and an active mode lit to full
  strength read as a button the cursor was on. Disabled still wins over the latch. `TextSize`
  zero means the default 20, the same "use the default" convention `BaseColor`'s zero alpha
  uses; the sort buttons set 30, because a square carrying one character is nearly all label.

## Hidden information is gated on `DebugGameplay`

The opponent's queued actions were concealed in the enemy rows of the queued-actions pane
unless `DebugGameplay` is on. `CombatScene.concealEnemy` is the single predicate —
`!gs.DebugGameplay && s.planning()` — and anything else that becomes secret should join it
rather than growing a second rule.

**The fight log needs no concealment rule at all**, and that falls out of its design rather than
being a second decision: it is built from `s.log[:cursor+1]` for the round in progress, so it can
only ever contain events playback has already reached, and an action that has resolved is not a
secret. There is no way for it to leak the rest of the round because it has not been given it.

- **Concealment lifts once playback starts**, for the same reason.
- **Concealed rows keep their real count**, so the opponent's AP spend stays readable even
  when the actions do not. Deliberate, and recorded as open in `TODO.md`: collapsing the
  rows would hide the spend and destroy the pane's account of who acts when, which is the
  one thing that pane exists to show.
- Debug is a *view*, never a rule. `ResolveRound` never sees the flag, so turning debug on
  or off cannot change an outcome — the same constraint that applies to playback speed.

