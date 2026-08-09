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
  category glyphs are 22 — so ask `systems.SizeOf(kind)` and never assume `GlyphSize`.
- **The card is drawn by `internal/cards`, not by this screen.** `drawCard` builds a
  `cards.Spec`, pulls a cached image and blits it; it draws nothing itself. Change how a
  card looks there, then `go run ./tools/cardsheet` and refresh the tab — the tool and the
  screen run the same code, so the sheet cannot lie.
- **Colour: name one colour and scale it** with `systems.ColorAtStrength` — but that scales
  toward *black*, so on the off-white card surface it makes things louder, not quieter. Use
  `systems.ColorToward` against a light ground. The card's element is its **border** now,
  not its surface.
- **Widgets are hand-rolled**, `models` struct plus `systems.Update*`/`Draw*`. No toolkit.
- **Determinism**, which the deck's `rng` and the `deckSeed` placeholder live under.
- **Which of the screen's six files holds what** — the map is in `CLAUDE.md`'s Package layout
  section, and it lives there rather than here so there is only one of it. Everything below
  describes the screen, not a file; they are one package, so a symbol named here may sit in
  any of them and `Grep` over `internal/screens/` is the way to find it.

## Resolution order

**Phases, implemented 2026-08-06.** A round is **a whole turn each**: everything side A
queued resolves before side B does anything, and within a turn the categories go in order —
**prepare, attacks, defenses**. Defenses come last within a turn because the opponent moves
next, so a defense raised at the end of your turn is up when their blow arrives.

This replaced alternation, which replaced volley-per-side on 2026-07-31. The reason is
legibility: interleaving may simply not be graspable by players. See
[MECHANICS.md](../../../MECHANICS.md).

**Initiative is gone**, whole — the method, the constants, the tie-break, the clock glyph and
the `i3` in the concealed enemy label. With one contiguous turn per side there is no exchange
for a faster action to lead. `Spd` still buys action points and still never buys priority.

### What this means for the screen

- **Cross-category reordering does nothing.** A defense cannot be dragged ahead of an attack;
  the drag lands the card in a queue that is then regrouped. **Within-category order is
  preserved and is the whole of what dragging now changes** — sequence combos will match on
  it, making an ice Strike before a fire Strike a different round from the reverse.
- **`Slot.Index` is not a position in the round.** It is where the card sits in its own
  side's queue, which regrouping breaks apart. Anything asking "how far through the round are
  we" counts slots — `CombatScene.currentSlot` does, and lighting the right Resolution row
  depends on it.
- **The category is deliberately not concealed** on enemy rows. It is what decides where a row
  sits, so withholding it would make the pane unreadable rather than merely uncertain. It took
  over that job from the initiative number — see `concealedLabel`.
- **The card shows its category as a glyph** — a sword for attack, a kite shield for
  defend, an open book for prepare — in the top-left corner above the cost. It was a word
  under the name until 2026-08-09. Three states are not a quantity, which is why it was
  never a numbered badge; a 22-pixel silhouette is read before any text is.
- **Cost is a stack of dash marks**, under the category glyph, one per point. Not a numeral
  and not a badge. Costs run 1..4 and a fifth tier is a layout change, not a bigger number.

### Combos, and the one thing they changed on this screen

Landed 2026-08-07. The rules are in `internal/combat/combo.go` and the design is in
`MECHANICS.md`; two things matter to the screen.

- **`combat.MatchCombos` is what to call to preview a combo while the player plans.** It is
  the same matcher the engine uses, so a previewed combo is the combo that fires — by
  construction, not by two pieces of code agreeing. Nothing draws it yet.
- **A staggered slot is a row that never resolves**, which is the first time the pane has had
  one. `currentSlot` counts `KindStaggered` alongside `KindAction` for exactly this reason —
  one beat per slot, taken or lost — and `TestEverySlotIsEitherTakenOrStaggered` pins it.
  **The pane still draws that row as though it happened**, which is a known gap: it has no way
  to show a card struck out of the round, just as it has no way to bracket a combo.

`KindCombo` is emitted on the card that **completes** the run, so a combo line lands under the
cards that earned it. It fired on the *first* card until 2026-08-07 and read backwards. The
screen does nothing to arrange this — it replays the log in order, and the engine moved.

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

The screen reads top to bottom rather than left to right, decided 2026-08-04: **who you
are and what the round is doing** above, **the cards you are doing it with** along the
bottom. Colours identify the role and are placeholders, not a chosen palette.

| Element | Slot | Colour | Role |
|---|---|---|---|
| Character strip | 4–39% x, 2% y | green | health, discards, vitae, side by side |
| Resolution | 15–78% x, 12–46% y | pink | what the round actually did, accumulating as it plays |
| Action Flow | *built, not drawn* | pink | both queues in play order — see below |
| Caption box | hand width, 48% y | pink | your plan and its AP cost, and what to press |
| Hand | centred, 59% y | element | the cards, portrait, in one row |
| AP figure and bar | hand width, under the row | blue | the budget |
| Buttons | 95% y | — | Discard 20%, DUEL! 33%, Deck 88% |
| Enemy sprite | 88% x, 34% y | — | the opponent |

**Cards are portrait and live along the bottom.** Landscape cards in a vertical column
capped how many could be shown, and the hand is going to grow. `cardWidth`/`cardHeight` are
**flat constants — 180x264 — and must stay flat**: they used to be derived from the glyph
row, so adding a badge silently widened every card and the layout could not be reasoned
about without doing the arithmetic. Contents fit the card, never the reverse.

**`handBand()` is the single authority on the hand's horizontal extent.** The card slots
are cut out of it, the AP bar spans it and the caption box matches it, so the three cannot
drift apart when the hand size changes. A card in flight still owns its slot, which is what
stops the row sliding half a card sideways when one is lifted.

**The buttons are one strip at 95%, under the AP bar.** Discard at 20% and DUEL! at 33% sit
together, because they are the same choice — **you select a set, then decide what it was
for** — and the choice belongs next to the cards it is made from. Deck is alone at 88%; it
changes nothing and belongs nowhere near them. Discard carries one condition DUEL! does
not: a round's discards can run out.

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

**The hand persists across rounds** *(changed 2026-08-06)*. It used to empty completely every
round and deal a fresh eight, justified by "a hand kept back would let a plan be prepared once
and repeated". That conflated the queue with the hand: the *queue* still empties every round,
so no plan repeats by default, but taking away cards the player had deliberately held punished
holding them and made playing a card the only way to keep it.

The consequence to hold on to: **Discard is now the only way an unwanted card leaves your
hand.** `discardsPerRound` stopped being a convenience and became the rate at which a hand can
be steered.

**The character block replaced the fighter's sprite and health bar.** A bar says roughly
how hurt you are, and a duel decided in whole points wants the exact number, so life is a
red fraction. Discards refill each round; vitae is a fixed placeholder drawn anyway, so the
box has its real shape before the rest of the character's state is designed. The enemy
keeps its sprite and bar for now.

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

*Split 2026-08-07, then narrowed the same day.* The old single Resolution pane was renamed
**Action Flow** and a new **Resolution** took its slot. Flow was then **dropped from `Draw` as
an experiment** and Resolution widened to 15–78% to take both columns. `drawActionFlow` and
`actionFlowRows` are deliberately left in place and unwired, so restoring it is one line.

**What is given up while it is off, and it is not nothing:** the enemy's queued shape during
planning. Those `??? (attack)` rows are the tell — see the concealment section below — and
Resolution is empty until DUEL! is pressed, so nothing on screen says what the opponent is
about to do. It bites hardest against Tactician1, whose whole design is that you read
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

*2026-08-07, changed 2026-08-08.* A line is `<who> <verb> <phrase>` — **"Duelist attacks with a
heavy strike"** — and the verb is **coloured, bold and underlined**: **red for attack, blue for
defend, the row's own ink for prepare**. A round can then be scanned for what *kind* of thing
happened before any of it is read.

**The verb was a filled chip until 2026-08-08.** It was replaced because a saturated rectangle in
a pane that already carries a swatch and a sentence drew the eye to the block rather than to the
word inside it. **This is the same correction that retired the full-width highlight bar the day
before, one scale smaller** — mark the words, do not sit them on a lit shape. If a third version
is ever wanted, that is the argument to answer.

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
- **The prose lives in `internal/screens`, not `internal/combat`.** The rules package names
  actions; it does not describe them. `actionPhrases` is a map keyed by `ActionKind`, with a
  fallback so a new card reads awkwardly rather than producing a sentence with a hole in it.
- **Outcomes append to `suffix`**, after the verb, so the mark never moves as a line grows.
- **The name is said as well as coloured, deliberately.** The swatch already encodes the side,
  but a line beginning "Strike" reads as an instruction rather than a report, and with both
  sides in one list the reader would have to hold which colour is which.

**Row highlights and swatches are centred on the measured line height**, not offset from the
row top by a constant. The old `rowY-4` / `rowHeight-2` numbers were picked by eye against a
single 30px pitch and clipped the text the moment a 22px pitch existed. `text.Measure` once per
pane, centre everything on it, and any pitch works.

**Labels in the character strip are title case, not caps.** `VITAE` rendered as `VITRE` —
kubasta's uppercase A at 12px carries a diagonal that reads as an R with no lowercase around it
to set the shape. Not worth a label the player reads as a different word.

**It has no phase headings, and that is a space constraint rather than a decision.** The pane
holds nine rows between `paneFirstRow` and its bottom edge, and five player actions plus the
enemy's already reach that. Under phases the grouping reads off the order anyway. If headings
are wanted, the pane has to get taller or the rows shorter first.

Two panes now, and the count went **down** before it went back up. Chosen folded into the
palette on 2026-08-02; Enemy went the same day, because a merged Resolution already shows the
opponent's actions in a better order than a column of its own; Actions went on 2026-08-04 with
the move to the bottom, since the hand has no frame and there was nothing left for a placement
to hold. That left the 15–39% column empty and deliberately unassigned for three days —
**Action Flow is what finally claimed it**, on 2026-08-07.

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
  2026-08-06. It had to move: it is a rule, and **the opponent's planner obeys it exactly as
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
- **The AP budget is drawn twice, on purpose.** A `3/6 AP` line for the exact figure and a
  bar under it for the glanceable one, both sitting between the cards and the two buttons
  that spend them.

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

