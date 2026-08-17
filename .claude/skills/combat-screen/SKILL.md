---
name: combat-screen
description: The combat screen's layout, its card/action box widget, hidden information, and the resolution-order rule the screen must obey. Load before touching any of internal/screens/combat.go, combat_deck.go, combat_panes.go, combat_hud.go, combat_actionbox.go, internal/combat, the hand, the cards, the deck overlay, the Resolution or caption panes, the character block, or anything about how a round is drawn or played back.
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
- **Which of the screen's six files holds what** — the map is in `CLAUDE.md`'s Package layout
  section, and it lives there rather than here so there is only one of it. Everything below
  describes the screen, not a file; they are one package, so a symbol named here may sit in
  any of them and `Grep` over `internal/screens/` is the way to find it.

## Resolution order

**Phases.** A round is **a whole turn each**: everything side A
queued resolves before side B does anything, and within a turn the categories go in order —
**prepare, attack, defenses**. Defenses come last within a turn because the opponent moves
next, so a defense raised at the end of your turn is up when their blow arrives.

**The attack phase is one blow** *(2026-08-14)*. Every attack card queued is announced with a
`KindAction`, then one `KindCombo` names the hand and mix they formed, then a single `KindDamage`
lands. Five Strikes are not five hits.

**The phase is also one line in the feed, and it is the combo's.** The `KindAction`s still play —
each is a beat, and each raises its card on the table — but the pane draws no sentence for them.
`KindCombo` carries `Base`, `Swing`, `Multiplier` and `Amount`, so the line reads *"COMBO! Duelist
lands a Duo Strike Pair (20 + 10 x 3.5 = 55)"* and the damage attaches to it. **A blow that formed
no hand writes an ordinary attack sentence instead**, from the card the engine names on the event —
"COMBO!" over a lone Strike would empty the word. **Attack cards that build no hand contribute
nothing**, and what says so is the table: every attack card is raised as it is announced, and the
combo lowers the ones it did not name.

**None of the three paragraphs above describes an enemy's turn** *(2026-08-17)*. `Duelist.SoloAttacks`
makes attack cards resolve one at a time in queue order, each landing its own blow, and **no
`KindCombo` is emitted at all**. `CombatScene.soloAttacker(side)` is the screen's single predicate
for it and two things read it:

- **The feed writes a sentence per attack card**, because there is no phase line coming to carry
  them. `resolutionLines` suppresses an attack's `KindAction` only when a combo *is* coming.
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

### Combos, and the one thing they changed on this screen

The catalogue is `data/combos.json`, the matcher is `internal/combat/combo.go`, and the design is
in `MECHANICS.md`; these are what matter to the screen.

- **A combo is a *hand* and a *mix*, and the event carries both.** `Event.Hand` is a `HandID`,
  `Event.Mix` a `MixID`, and `Event.Multiplier` the combined percent. `comboName` in
  `combat_panes.go` looks the two up with `HandByID`/`MixByID` and joins them — "Duo Strike Pair"
  — dropping the mix when it has no colours. **Exactly one of each fires per turn**, so there is
  no stacking to draw and no ranking to explain.
- **`Event.Hand == combat.HandNone` means a lone attack that formed nothing**, which the pane
  writes as an ordinary attack sentence rather than as a combo. It still carries a mix, because
  one card is one colour, and it carries the card it led with in `Action`/`Element` so the pane
  has something to name.
- **The event carries the arithmetic, and the engine takes its damage from the same field.**
  `Base` is what the hand's own cards deal, `Swing` is one Strike at this attacker's strength, and
  `Amount` is what they come to under the multiplier — the blow *before* the attacker's weight and
  before any defence. `resolveAttackPhase` blunts `Amount` rather than re-adding the sum, so the
  figure printed and the figure landed cannot be two different numbers. The gap between that and
  the `KindDamage` after it is exactly what the defence was worth.
- **A fired combo brackets its own cards, and the list comes from the event**.
  `Event.ComboCards[:ComboCardCount]` names which cards of the turn formed it. **Never derive
  that from the hand's group sizes, and never assume the cards are adjacent.** A counted hand is
  not contiguous — Two Pair is two cards, a card that earned nothing, and two more — which is why
  the event carries a list rather than a start and a length. Both rows bracket, through one
  `drawComboBracket` taking positions, so the opponent's hand is ringed by the same code.
- **`combat.BlowFor` previews the hand while the player plans** *(2026-08-15)*. It is the same
  function the resolver uses, so a previewed combo is the combo that fires by construction rather
  than by two pieces of code agreeing. `previewAttack` calls it on `ResolutionOrder(queue, nil)`
  and **only a formed hand previews** — `HandNone` is a lone attack, and COMBO! over one Strike
  empties the word. Two things show it: `drawComboPreview` rings the cards in the *hand row*
  through the same `drawComboBracket` the table uses, and the Resolution feed carries a line.
  **`Blow.Cards` indexes the turn, not the hand** — `previewHandSlots` translates through
  `handIndexForQueue`, or a Prepare queued first would ring the wrong cards.
- **A staggered slot is a row that never resolves.** `currentSlot` counts `KindStaggered`
  alongside `KindAction` for exactly this reason — one beat per slot, taken or lost — and
  `TestEverySlotIsEitherTakenOrStaggered` pins it. **The pane still draws that row as though it
  happened**, which is a known gap.

**Within a turn the order is: prepares one at a time, every attack card announced, the combo, the
damage, then the defends.** The screen does nothing to arrange this; it replays the log in order,
and the engine decides.

Three consequences for playback. **The combo line lands after its cards are announced but before
the damage**, so a boosted figure never arrives before the reason for it, and `noteCombo` has
real rows to mark because the whole queue is seated at DUEL! rather than a card at a time. **The
whole attack hand is raised by the *first* announcement** — `attackSeats` reads them off the turn,
so `firingSeats` is a list rather than one seat and the beats after it name the same set — and
`noteCombo` narrows the list to what earned it. And **a hand pays its stagger even if the blow then misses** — the shock roll happens after
the combo event, because the hand is scored off the queue and the queue was committed at DUEL!.

### What survives any model

- **`combat.ResolutionOrder` is the single authority on order.** `ResolveRound` plays what
  it returns and the Action Flow pane draws what it returns. Neither derives the order
  independently, which is what makes it structurally impossible for the pane to lie to the
  player about their own round. `TestResolutionOrderIsWhatResolveRoundPlays` pins it.
  **Stagger is the one thing that can remove a slot** rather than reorder it, and the
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
| Resolution | 15–78% x, 12–46% y | pink | what the round actually did, accumulating as it plays |
| Action Flow | *built, not drawn* | pink | both queues in play order — see below |
| Caption box | hand width, 48% y | pink | your plan and its AP cost, and what to press |
| Hand | centred in what the sort column leaves, 66% y | element | the cards, portrait, in one row |
| Sort column | band's right edge, centred on the cards | slate | `$` / `T` / `E`, the hand's arrangement |
| AP bar | hand width, directly under the row | blue | the budget |
| Bottom strip | 95% y | — | AP figure, Discard, DUEL!, deck pile — evenly spread |

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

**The strip at 95% is one row of four things, spaced rather than placed**: the
AP figure at the hand's left edge, Discard, DUEL!, and the deck pile at the right. The two
buttons used to sit at 20% and 33%, side by side because they were the same choice made two
ways. **They are separate choices now and the spacing says so** — `buttonStripSlots` divides
what is left between the figure's column and the pile into three equal gaps and puts a button
in each of the two spaces, so the strip stays evenly spread if any of the three fixed things
moves. `TestTheButtonStripSharesItsSpaceEvenly` checks the gaps against each other rather than
against numbers, because the property wanted is the relationship, not a coordinate. Discard
still carries one condition DUEL! does not: a round's discards can run out.

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
the bar, the figure, the cards and the Resolution feed all measure off this row, so moving it
moves the whole lower half together and opens the band between the fighter cards and the feed.

**`apFigureReserve` is a fixed column width, not the figure's measured width.** Measuring
would move both buttons the moment the text went from `9/12 AP` to `10/12 AP`. The reserve
holds the normal figure and the `+N over` tail runs past it, into a gap hundreds of pixels
wide. The buttons read `handSize` through `handBand`, never the live hand, for the same
family of reason: `handBand` is centred, so a shorter row starts further right.

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

**DUEL! becomes the way onward when the duel ends**, changing its own label to Next or Retry
rather than a fourth button appearing. Same slot, same meaning — commit and move the game
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
  raises every attack card of that turn and `noteCombo` then drops whichever earned nothing. They
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
- **The duel is open-information now, on the owner's call.** `concealEnemy` still governs the
  Action Flow pane, but with the opponent's cards face up on the table there is nothing left for
  it to hide. **The lever is still built**: `cards.Spec.FaceDown` draws a back and the draw pile
  is a stack of them, so hiding this row again is a field rather than a second drawing path.
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
away: the played cards stay on the table, the hand keeps the gaps they left, the queue stays in
the Resolution feed, and `Init` clears all of it when the next fight starts.

**What forced it is that the row is a measurement, not just a picture.** `handBand` is a function
of how many cards are in the hand, the AP bar spans that band, and the feed's bottom edge comes
off the same row — so spending the hand after the killing blow collapsed the cards into a narrow
centred huddle and dragged the bar and the feed in with them. **The picture the player is looking
at when the blow lands is the picture they should still be looking at when they press Next.**

The intermediate version — refill nothing but still spend — does not work and is worth not
re-trying: a hand that loses cards reflows whether or not it is topped back up. What has to stop
is the whole end-of-round movement, which is why this is a branch in `endOfRound` rather than a
rule inside `drawHand`.

**What the freeze cost, and it is the shape of thing to look for again.** `s.resolved` used to be
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

**The deck overlay is a dialog, and the only one in the game.** It fills nearly the screen,
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

### Action Flow is built and currently not drawn

*Split, then narrowed.* The old single Resolution pane was renamed
**Action Flow** and a new **Resolution** took its slot. Flow was then **dropped from `Draw` as
an experiment** and Resolution widened to 15–78% to take both columns. `drawActionFlow` and
`actionFlowRows` are deliberately left in place and unwired, so restoring it is one line.

**What is given up while it is off, and it is not nothing:** the enemy's queued shape during
planning. Those `??? (attack)` rows are the tell — see the concealment section below — and
Resolution is empty until DUEL! is pressed, so nothing on screen says what the opponent is
about to do. It bites hardest against a tactician, whose whole design is that you read
`??? (prepare) ??? (prepare)` and brace.

The rest of this section describes the split as designed, and still applies if Flow comes back.

- **Action Flow** is what you **queued**, in play order — live while you plan, before anything
  has happened. A prediction, and what drag-to-reorder edits.
- **Resolution** is what actually **happened** — empty until DUEL! is pressed, filling as the
  round plays back. A record.

Showing the round twice is only worth the space because of that split, and it is what retired
the open problem of how one pane could be both. **Action Flow never learned to bracket a combo
across non-adjacent slots and no longer has to**, because Resolution says it in words. Same for
a slot a stagger deleted: Flow still draws it as a row, Resolution reports it lost.

**The narrow column and the wide one are not interchangeable.** Flow rows are short labels
(`Strike`, `??? (attack)`) and fit the 15–39% column the Actions pane vacated. Resolution rows
are sentences, so it keeps the wide middle slot — which is what the pane billed as the
centrepiece should have anyway.

- **One line per slot, not one per event.** A busy round is 25–30 events against a dozen rows.
  Drawing the log verbatim would need a scrollback, and **there is no scroll gesture** — no
  wheel convention, no keyboard, no right click. Merging an action with its outcome is
  presentation of events the engine already decided; it computes nothing.
- **Combos and staggers get lines of their own**, because they are not something a card did.
  Folding a combo into the line of the card that happened to start it would bury the one thing
  the pane was added to show.
- **Built only from events playback has reached** (`s.log[:cursor+1]`), so the pane says exactly
  as much as the player has been shown and never spoils the rest of the round.
- **It clears every round**, because there is no way to scroll back to an earlier one.
- **Before a round resolves it holds `planningLines` instead**: `(press DUEL!)`, and above it the
  combo the current selection has already formed. **The pane records rather than proposes, and
  this is the standing exception rather than a new one** — the prompt was already a proposal,
  because the caption box that used to carry it is gone and nothing else on the screen has room.
  It costs nothing the rule protects: the pane is empty while planning, so nothing is said twice,
  and the record replaces both lines the moment DUEL! is pressed. **It carries no arithmetic** —
  `Base` and `Swing` are the resolver's, against a strength and a shock roll that have not
  happened, and a figure the round then contradicted would be worse than no figure.
- Overflow draws `... N earlier` rather than dropping lines silently — the same rule as the deck
  overlay's `+N more not shown`. A panel that quietly hides part of what it claims to show is a
  picture that lies.
- `resolutionCapacity` is **derived** from the band and the pitch, so changing either cannot
  leave a constant behind claiming a capacity the pane does not have.

**The caption stopped narrating.** It used to show one event at a time, which meant the whole
account of a round existed only as a quarter-second flash — a combo forming was unreadable and
a block that halved a Heavy went past before it could be noticed. It now returns `""` during
playback and keeps its other job: the plan line, its AP cost, and what to press when a fight
ends. **One job each — the pane records, the caption proposes.** Letting both narrate would put
the newest line on screen twice, which is the thing the pane was added to fix.

`panePlacement` carries its own `rowHeight` because the two panes hold different things:
`paneRowHeight` (30) for card names, `paneTextRowHeight` (22) for sentences.

### Resolution writes sentences, and the verb is marked in the text

*A line is `<who> <verb> <phrase>` — **"Duelist attacks with a
heavy strike"** — and the verb is **coloured, bold and underlined**: **red for attack, blue for
defend, the row's own ink for prepare**. A round can then be scanned for what *kind* of thing
happened before any of it is read.

**The combo line is the exception and has no verb.** It is an announcement — amber swatch,
`COMBO!` in front — and it is the one announcement that takes outcomes, because the attack cards
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
  pane, dark on Resolution and light on Action Flow. Being the category with no colour is also its
  right rank: it is the one that does nothing to the opponent. `verbInkFor` returns **zero alpha**
  to mean this, the same "use the default" convention `Button.BaseColor` uses.
- **The underline sits flush with the bottom of the measured line box**, never a constant above
  it. It used to hang under a chip of fixed height; with no chip the only thing to position it
  against is the text, and `text.Measure` reports the full line including descent — which is what
  clears the `p` in "prepares". A rule placed a few pixels above the baseline struck through it.
- **The prose lives in `internal/screens`, not `internal/combat`.** The rules package names cards;
  it does not describe them. **It is generated from the verb rather than tabulated** *(2026-08-16)*
  — `actionPhrase` and `cardEffect` in `combat_panes.go`, switching on `Verb` and dropping the
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
and there was nothing left for a placement to hold. **Action Flow claimed the 15–39% column
those left empty.**

The player's rows carry `playerSwatch` green and the opponent's carry `enemySwatch` yellow, so
the screen reads as two colours: green is you, yellow is them. `comboSwatch` amber is the third
and marks a Resolution line that is not a card acting — it belongs to whoever formed the combo,
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
  bar, the figure, the cards and the Resolution feed all measure off `handTopPct`, which is
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
  own key chain — cost, family, concept, element — with one key promoted to the front, so a row
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
  so a hand rearranged under a stale `fighterActions` leaves the combo preview bracketing
  whichever cards sit at the old positions.
- **`sortMode` is the one field `Init` does not reset.** A reading preference is not a fact about
  a duel, and snapping back to cost every fight would make it something the player re-presses.
- **All three go dead outside `planning()`** — a resolved card is drawn from the hand slot it
  flew out of, so rearranging mid-round would light the wrong card on the table.
- **`elementRank` and `categoryRank` are written out**, like `familyRank`. `combat.Basic` leads
  its enum as the zero value and trails on screen: the colours are what a mix is counted on, and
  the drab cards are the plans.
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

The opponent's queued actions are concealed in the enemy rows of the **Action Flow** pane
unless `DebugGameplay` is on. `CombatScene.concealEnemy` is the single predicate —
`!gs.DebugGameplay && s.planning()` — and anything else that becomes secret should join it
rather than growing a second rule.

**Resolution needs no concealment rule at all**, and that falls out of its design rather than
being a second decision: it is built from `s.log[:cursor+1]`, so it can only ever contain
events playback has already reached, and an action that has resolved is not a secret. There is
no way for it to leak the rest of the round because it has not been given the rest of the round.

- **Concealment lifts once playback starts**, for the same reason.
- **Concealed rows keep their real count**, so the opponent's AP spend stays readable even
  when the actions do not. Deliberate, and recorded as open in `TODO.md`: collapsing the
  rows would hide the spend and destroy the pane's account of who acts when, which is the
  one thing that pane exists to show.
- Debug is a *view*, never a rule. `ResolveRound` never sees the flag, so turning debug on
  or off cannot change an outcome — the same constraint that applies to playback speed.

