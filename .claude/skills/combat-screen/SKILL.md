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
**attack, then everything else**. Defences come last within a turn because the opponent moves
next, so a shield or a guard raised at the end of your turn is up when their blow arrives.

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
  and every raised defend answers the one blow regardless of when it went up. **What order still
  decides is the hand's tie-break** — `groupsOf` breaks a tie by whose first card was played first,
  so the lead card that names the hand and carries its element is chosen by where the player put it.
  Do not paper over the rest on the screen, and do not invent a rule to justify it.
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
- **The card shows its form as a corner mark** — a spear, a sword, an axe, a shield — in the
  top-left corner above the cost, tinted by the card's element. A form is not a quantity, which is
  why it is not a numbered badge; a 32-pixel drawing is read before any text is.
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
  on screen from the first attack picked; a queue of nothing but shields names nothing, `BlowFor`
  returning a blow with no cards. **This reverses the old rule**, which was that only a hand of two
  or more previewed, on the argument that HAND! over one Strike empties the word. What makes it
  safe is that the label names the *hand* rather than shouting HAND!, and the log still writes a
  lone attack as an ordinary attack sentence. **Two lines show it**: `drawPlannedHand` writes its
  name across the middle of the table, breathing, in `handNameInk`, with **what it is worth on a
  second line under it** — `1.15x DMG` — and that pair is what flies down to the hand row at DUEL!,
  so the preview and the announcement are one object. See `handBanner`.
  **`Blow.Cards` indexes the turn, not the hand**, which is why the preview goes through
  `ResolutionOrder` — a Ward queued first resolves last, so a preview read off the hand as the
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

**Within a turn the order is: every attack card announced, the hand, the damage, then the defend
cards one at a time.** The screen does nothing to arrange this; it replays the log in order,
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

### The shield row on the duelist card

*`combat_shields.go` and `card_art.go`, 2026-08-31.* The player's three defend cards raise shields, and
**one pip per shield is drawn in the seat the enemy card's status badges occupy** — same offsets,
same box, so the two fighter cards stay twins. The pip is `assets/form/defend.png`, the mark the
cards themselves carry, so what was raised and what is standing are the same picture.

- **`shownShields` is a view, exactly like `shownLife`.** `Duelist.Shields` is not
  written until the round's end state is adopted, so a row reading the model would fill a whole
  opposing turn after the card that filled it and empty a whole turn after the attack that ate it.
- **All three changes to the count are announced**: `KindRaised` when a card goes down,
  `KindBlocked` each time one eats an attack, `KindExpired` when the unspent ones lapse. The last of
  those exists *for this row* — without it the pips would keep drawing a defence the engine had
  already taken away.
- **It falls back to the model when no event this round has spoken**, which is what makes the
  planning phase right: a shield raised at the end of the last round is standing while the player
  builds this one, and nothing has fired yet.
- **The engine caps a duelist at five shields**, which is exactly what the row can draw. That is not
  a coincidence and not a clamp in the screen — see `Duelist.raiseShields`, which takes the cap for
  its own reason and this row inherits it.

### The log writes sentences, and the verb is marked in the text

*A line is `<who> <verb> <phrase>` — **"Duelist attacks with a
heavy strike"** — and the verb is **coloured, bold and underlined**: **red for attack, blue for
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
  colour for the side and for the live row, and bold alone for the live row — so the verb needs
  the combination to be unmistakable. **The consequence: the underline is no longer what marks
  the live row.** That row is now distinguished by `nowInk` plus faux-bold on `prefix`/`suffix`,
  and the verb keeps its category colour there rather than going pink with the rest.
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

- **Sorting a queued hand re-prices it** *(owner's call, 2026-08-26)*. Cross-category order is still
  regrouped away by `ResolutionOrder` and a hand is still counted rather than read in sequence, but a
  growing ring now steps between the cards of one blow — so the order of the queue decides what the
  cards are worth. The buttons stay live and a bad sort can cost damage; that is the intent, not an
  oversight. This paragraph used to say the opposite and it is the one thing to unlearn about the
  file.
- **Cost is the default, and every mode ends with it.** Each arrangement is the deck overlay's
  own key chain — cost, form, concept, element — with one key promoted to the front, so a row
  of cards means the same thing in the hand as in the panel. Only the leading key differs.
- **The sort re-applies on every refill**, in `spendSelected` *before* anything is animated, so a
  dealt card flies to the slot it will actually occupy. A drag still works and survives until the
  next deal, at which point the sort reclaims the row.
- **Every figure in the hand dialog's sum comes from a card, and that card shakes as it is written**
  *(owner's call, 2026-08-26)*. A card's damage flies out of the played card, a ring's multiplier out
  of that ring's card, and an echo's extra term shakes the ring that bought the landing even though
  it puts no figure on the line. The box runs its items strictly one at a time, so putting the ring
  figures *in* the script is the whole of the sequencing — there is no second clock. `mathItem`'s
  `ringSeat` / `cardSeat` / `shakeRings` are the marks, and `handMathBox.shaking` is what the screen
  reads each tick.
- **Sideways, never a jump.** Vertical is spoken for twice already — a selected card lifts in the
  hand, and a card that built the hand lifts on the table for the whole blow. The shake says "this
  one is paying *now*", so it needed a direction nothing else uses.
- **The drag runs on the shared controller in `carddrag.go`** *(2026-08-26)*, which the worn ring
  row also uses on all three screens that draw it. The hand's adapter is `handRow`; it really does
  remove the card from `s.hand`, where the ring row's removes nothing and lets the run stay the
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
  when the actions do not. Deliberate: collapsing the rows would hide the spend and destroy the
  pane's account of who acts when, which is the one thing that pane exists to show.
- Debug is a *view*, never a rule. `ResolveRound` never sees the flag, so turning debug on
  or off cannot change an outcome — the same constraint that applies to playback speed.

