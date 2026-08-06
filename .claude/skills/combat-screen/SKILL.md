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

> **Being replaced.** Phase-based resolution was chosen on 2026-08-05 as **an experiment, and
> the direction to head in** — see [MECHANICS.md](../../../MECHANICS.md). Alternation is what
> ships today and everything below describes it accurately; it is not the design being aimed
> at. Do not treat it as settled, and do not build new mechanics on the assumption that it
> survives.
>
> The intended model: a round resolves in phases — the player's **preparations**, then
> **attacks**, then **defenses**, then the enemy. Defenses front-load because the enemy goes
> last, so a defense is up when the blow arrives. The reason is legibility: interleaving may
> simply not be graspable by players.
>
> What that changes here: **cross-phase reordering stops meaning anything** (a defense cannot
> be dragged ahead of an attack), while **within-phase ordering matters more**, because
> sequence combos make an ice Strike before a fire Strike a different round from the reverse.
> And **Guard persistence dissolves** — the last bullet below stops describing anything if
> every defense resolves before every enemy attack.

### What ships today

A round resolves the two queues **alternately**, one action each, with the longer queue
acting alone once the other empties. This replaced volley-per-side on 2026-07-31.

Within one exchange, **the faster action lands first**: lower `ActionKind.Initiative()`
wins, and side A takes a tie. Initiative is a lever wholly separate from cost — cost
decides what a plan may *contain*, initiative decides *when* its pieces happen — and it is
separate from `Spd`, which buys action points and never priority.

The loop alternation was built for: **the player chooses their actions, then alters the
resolution order.** Dragging a card to a different slot changes which of the opponent's
actions it contests. Under phases that specific payoff goes away and combos replace it.

### What survives either model

- **`combat.ResolutionOrder` is the single authority on order.** `ResolveRound` plays what
  it returns and the Resolution pane draws what it returns. Neither derives the order
  independently, which is what makes it structurally impossible for the pane to lie to the
  player about their own round. `TestResolutionOrderIsWhatResolveRoundPlays` pins it.
  **This is also what makes the phase change cheap** — one pure function body plus its tests,
  and both consumers follow.
- **Ordering is a rule.** It belongs in `internal/combat`, never in a screen. A new effect
  that rearranges resolution changes `ResolutionOrder` and both consumers follow.

### Tied to alternation, and expected to go

- A raised Guard lasts until its owner's next action, so it covers every opposing action
  in between, across a round boundary if it was queued last. A duelist who queues
  nothing therefore keeps its guard — pinned by `TestGuardHoldsWhileItsOwnerDoesNothing`.
  Phases make this vacuous.

## Combat screen layout is scaffolding

The screen reads top to bottom rather than left to right, decided 2026-08-04: **who you
are and what the round is doing** above, **the cards you are doing it with** along the
bottom. Colours identify the role and are placeholders, not a chosen palette.

| Element | Slot | Colour | Role |
|---|---|---|---|
| Character block | 4% x, 12% y | green | life, discards, vitae |
| Resolution | 45–78% x, 12–46% y | pink | both queues interleaved in play order |
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
pane that has to grow: once exchanges have structure — an initiator and a response — it
has to draw that rather than a flat list of rows.

One pane now. Chosen folded into the palette on 2026-08-02, Enemy went the same day because
an interleaved Resolution already shows the opponent's actions in a better order than a
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

