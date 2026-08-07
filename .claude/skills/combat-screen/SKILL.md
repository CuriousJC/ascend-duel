---
name: combat-screen
description: The combat screen's layout, its card/action box widget, hidden information, and the resolution-order rule the screen must obey. Load before touching internal/screens/combat.go, combat_actionbox.go, internal/combat, the hand, the cards, the deck overlay, the Resolution or caption panes, the character block, or anything about how a round is drawn or played back.
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
- **Glyphs are generated, and a glyph cannot be scaled below 1.** `GlyphSize` is 64 and the
  art is authored at the size it is shown, so the glyph column is the *floor* on a card's
  size rather than something that scales with it. A smaller card gets less padding and
  smaller text around an identical column.
- **Colour: name one colour and scale it** with `systems.ColorAtStrength`, never add to it.
- **Widgets are hand-rolled**, `models` struct plus `systems.Update*`/`Draw*`. No toolkit.
- **Determinism**, which the deck's `rng` and the `deckSeed` placeholder live under.

## Resolution order

**Phases, implemented 2026-08-06.** A round is **a whole turn each**: everything side A
queued resolves before side B does anything, and within a turn the categories go in order —
**setup, attacks, defenses**. Defenses come last within a turn because the opponent moves
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
- **The card shows its category as a word**, under the name, where the initiative badge used
  to be a number. Three states are not a quantity, so a badge with no number beside it would
  read as a badge missing one.

### What survives any model

- **`combat.ResolutionOrder` is the single authority on order.** `ResolveRound` plays what
  it returns and the Resolution pane draws what it returns. Neither derives the order
  independently, which is what makes it structurally impossible for the pane to lie to the
  player about their own round. `TestResolutionOrderIsWhatResolveRoundPlays` pins it.
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
| Character block | 4% x, 12% y | green | life, discards, vitae |
| Resolution | 45–78% x, 12–46% y | pink | both queues in play order: your whole turn, then theirs |
| Caption box | hand width, 48% y | pink | what the round is doing right now |
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

**The Resolution pane is the centrepiece and gets the width to prove it.** It is the only
pane that has to grow: it has to draw a combo spanning slots that need not be adjacent, and
one row per slot with a single walking highlight has no way to say "these together did a
thing".

**It has no phase headings, and that is a space constraint rather than a decision.** The pane
holds nine rows between `paneFirstRow` and its bottom edge, and five player actions plus the
enemy's already reach that. Under phases the grouping reads off the order anyway. If headings
are wanted, the pane has to get taller or the rows shorter first.

One pane now. Chosen folded into the palette on 2026-08-02, Enemy went the same day because
a merged Resolution already shows the opponent's actions in a better order than a
column of its own, and Actions went on 2026-08-04 with the move to the bottom — the hand has
no frame, so there was nothing left for a placement to hold. The player's rows carry
`playerSwatch` green and the opponent's carry `enemySwatch` yellow, so the screen reads as
two colours: green is you, yellow is them. Do not treat any of it as settled — the 15–39%
column the Actions pane vacated is deliberately still empty.

## The action box

[combat_actionbox.go](internal/screens/combat_actionbox.go) is the hand and its
drag-to-reorder, and the reference for building a *game* widget: state on the scene,
hand-rolled hit testing, no toolkit. Click a card to select it into the round's queue, click
it again to take it out, drag sideways to move it along the row.

- **`planning()` is the single predicate** for "the player may edit the queue" — derived
  from `cursor >= len(log)` plus both duelists alive, not stored. Drag and the DUEL!
  button both gate on it, so they can never disagree.
- **The action-point budget is enforced at selection.** A card the remaining points will
  not cover cannot be selected and draws dimmed. Accepting the click and then refusing it is
  a worse conversation than never letting it happen.
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

The opponent's queued actions are concealed in both the Enemy pane and the enemy rows of
the Resolution pane, unless `DebugGameplay` is on. `CombatScene.concealEnemy` is the single
predicate — `!gs.DebugGameplay && s.planning()` — and anything else that becomes secret
should join it rather than growing a second rule.

- **Concealment lifts once playback starts.** An action that has already resolved is not a
  secret, and the Resolution pane still has to narrate the round.
- **Concealed rows keep their real count**, so the opponent's AP spend stays readable even
  when the actions do not. Deliberate, and recorded as open in `TODO.md`: collapsing the
  rows would hide the spend and destroy the pane's account of who acts when, which is the
  one thing that pane exists to show.
- Debug is a *view*, never a rule. `ResolveRound` never sees the flag, so turning debug on
  or off cannot change an outcome — the same constraint that applies to playback speed.

