# TODO

Working notes for ascend-duel. Findings and reasoning live in `analysis.md`;
this file is just the running list.

Status: `[ ]` open · `[x]` done · `[~]` in progress · `[?]` needs a decision

---

## Now — quick wins, independent of any design decision

- [x] **Edge-triggered clicks.** `UpdateButton` used `ebiten.IsMouseButtonPressed`
      (level-triggered), firing `OnClick` every tick the mouse was held. Now uses
      `inpututil.IsMouseButtonJustPressed` / `JustReleased`, arming on press and firing
      on release inside bounds so dragging off cancels. *(analysis §5)*
      - [x] Verified in the running game after flipping the start screen to `Title`:
            one `Settings Button Clicked!` per click in stdout, where the old
            level-triggered code printed once per tick held.
- [x] **Extract `newCombatantFrom(gs, record)`.** Two near-identical init blocks in
      `InitCombatScreen`; the `SpriteRect` bug came from exactly this duplication.
      *(analysis §7)* Landed as `entities.NewCombatantFrom(data, sheet)` plus a small
      `combatantFromRecord(gs, key)` helper in `combat.go` — `entities` can't take a
      `*GlobalState` because `state` already imports `entities`, so the screen does the
      map lookups and the entity does the construction. `NewCombatant()` removed: it
      built an invalid combatant and had nothing left calling it.
- [x] **Move button positioning out of `Draw`.** `DrawTitleScreen` writes `ScreenX/Y`
      that `Update` then hit-tests against — stale on frame 1 and whenever Ebiten skips
      a `Draw`. *(analysis §3)* Moved to a new `InitTitleScreen`, run once from
      `UpdateTitleScreen` off the `NewScreen` flag. Safe to compute once now that the
      internal resolution is fixed.
- [x] **Screen-transition nil deref.** Latent bug surfaced by flipping the start screen
      to `Title`: an action sets `ActiveScreen`+`NewScreen` mid-`Update`, then `Draw`
      runs on the *new* screen before its `Init` ever has — `DrawCombatScreen` hit a nil
      `gs.Fighter`. `Game.Draw` now returns early while `NewScreen` is set, costing one
      skipped frame per transition. Was invisible while the game booted into Combat,
      because Ebiten's first `Update` always beat the first `Draw`.
- [x] `SettingsButtonAction` sets `NewScreen` without changing screen — removed; with the
      `Draw` guard it would now cost a skipped frame and a pointless re-`Init`.
- [x] **Fixed internal resolution.** `Layout` returns a constant 1280x960 and lets
      Ebiten letterbox, making every absolute coordinate safe. *(analysis §8, decided)*
      `game.ScreenWidth`/`ScreenHeight` are now the single source — `main.go`'s private
      duplicate pair is gone and `SetWindowSize` reads the same constants.
      Resizing enabled in `main.go` via `SetWindowResizingMode`, so the letterbox path
      actually runs.
      - [x] Verified on screen: the scene scales as a unit and letterboxes, and the
            layout no longer reflows on resize.
- [x] **AP bar in the Chosen pane.** The budget was a `3/6 AP` text line; it is now that
      line plus a bar underneath, filling in the pane's blue as the queue is built. The
      number answers "exactly how much", the bar answers "how much room is left" without
      being read. `drawAPBar` in [combat_actionbox.go](internal/screens/combat_actionbox.go).
      - A card lifted off the palette draws its cost as a dimmer segment ahead of the fill,
        because it has not joined the queue yet and would otherwise not move the bar until
        it landed. The bar answers "does this still fit" while the card is in the air.
      - A card lifted *out* of the queue already leaves it on pick-up, so that direction
        needs no special case — the fill drops the moment it is grabbed.
- [x] **Discard and Deck buttons, and the button row.** Discard at 20% and DUEL! at 33%
      sit together directly under the hand, because they are the same choice and it belongs
      next to the cards it is made from. Deck is alone at 88%. All three share the 80–90%
      band so the row reads as one strip.
      - **Deck** is a dialog. It fills almost the whole screen, everything else on the
        screen goes dead behind a scrim, and the Deck button is redrawn on top of the
        overlay so the one control that still works is the one that still looks like it
        does. Pressing it again closes it. No other close affordance, because there is no
        keyboard and no right click to reach for.
        - Both piles, counted by kind. **Never in order**: the draw pile is shuffled, and
          listing it in sequence would hand the player their next five cards and make the
          shuffle pointless. The discard is shown beside it because a reshuffle folds it
          back in, so "what is left" honestly means both. Every kind is listed even at
          zero, which is how Quick's absence from the deck is visible rather than implied.
      - **Discard** sends the selected cards to the discard pile and deals the hand back
        up to five.
      - [?] **Discard is free and effectively unlimited, which is a hole.** Selection is
            AP-gated, so one press can only throw out a budget's worth — but discarding
            returns those points, so pressing it repeatedly cycles the entire deck for
            nothing. Filtering that hard should cost something: a once-per-round limit, an
            AP charge, or drawing back one fewer than was thrown.
      - **Selection doing two jobs is the design, decided 2026-08-02.** You select a set
        and then choose what it was for: DUEL! plays it, Discard throws it away. There is
        no discard mode and no second gesture, and there should not be — one selection with
        two verbs is why the two buttons sit together, and it means every card you pick up
        is a question you have to answer rather than a slot you have filled.
      - That the action points come back when a card is discarded follows from this rather
        than being a leak: the selection was never spent, only proposed.
- [x] **Halve the playback dwell.** `actionDwellTicks` went from three seconds to one and
      a half. At three seconds a round of six actions took twenty seconds to watch and the
      gap between a move and its consequence read as the game hesitating. This is the
      constant the game-speed setting below will scale.
- [x] **Split the debug flag in two.** `ActiveDebug` became `DebugPlacement` (grid, rulers,
      scratch strings — about where things are drawn) and `DebugGameplay` (perfect
      information — about what the player is allowed to know).
      - Placement defaults **on**, gameplay defaults **off**, so the game plays as intended
        out of the box. Tuning balance against a view no player will ever have is the
        specific mistake the split exists to prevent.
      - Neither may change an outcome. `ResolveRound` never sees either flag.
      - Still no runtime toggle for either; a hotkey needs the keyboard the input
        vocabulary does not have.
- [x] **Procedural pixel-art glyphs on the cards.** `internal/systems/glyphs.go` generates
      three 64x64 glyphs — a sword for damage, a clock for initiative, a runner for
      action-point cost — each with its number written across it.
      - **Drawn in code rather than sourced**, which sidesteps the licensing problem that
        makes the Tyrian art a release blocker. Pixels written here have no provenance
        question at all, and that argues for doing more of the UI art this way.
      - They **replaced** the `init 3` line and the bare cost numeral. Two of the three
        numbers were already text and saying everything twice reads worse than once.
      - Damage is omitted when it is zero, so a Guard shows two glyphs rather than a sword
        reading 0. Needed `ActionKind.Damage` exported, since damage depends on the
        wielder's `Str` and the card has to ask.
      - **Rewritten as a generator on 2026-08-03**, after the first version — a hand-typed
        16x16 character map with one colour and one alpha shade — turned out to be flat and
        unresizable. Both limits were in the authoring, not in the idea:
        - A glyph is now a filled silhouette described by horizontal spans. The rim is
          *derived* from the silhouette and the shading is *computed* from position, so
          nothing is hand-placed and a shape can be nudged without repainting it.
        - Ported from the approach in the Python shield generator: shape table, derived
          outline, position-driven shading, named palette roles.
        - **Nothing may be thinner than about five pixels**, because the derived rim eats
          one from each side. The first pass had a three-row crossguard, which rendered as
          two rows of outline around one row of metal. This is the technique's real
          constraint and it sets every span in the file.
        - Five-value palettes, and glyphs are the deliberate exception to the colour rule —
          a bevel cannot be made from one colour scaled down. Drawn untinted; disabled
          cards dim by alpha so the shading survives.
        - [?] **Colour is deliberately unspent.** One hueless `white` palette for all three, so
              that an element or block type can land on colour later and mean something on
              arrival. This is the decision to revisit first when elements are designed.
      - [x] **`go run ./tools/glyphsheet`** writes `tools/glyphsheet/glyphs.png` — every
            glyph by every palette, reviewable by opening a file instead of launching the
            game. Committed on purpose so GitHub renders it as an image diff in review.
            Regenerate whenever the glyph code changes; a stale sheet is a picture that lies.
            - Output lives beside the tool rather than in a shared directory, so the pair
              moves together. Not in `assets/`, which is for `//go:embed`ed files the game
              loads — this is a picture *of* generated art, not an input to it.
            - `tools/` rather than `cmd/`: the convention that `cmd/` holds a module's
              shipped binaries does not fit a dev tool sitting beside a game whose own
              `main.go` is at the root, and moving the game under `cmd/` to satisfy the
              convention was explicitly rejected.
            - **Draws every glyph twice, at `systems.CardGlyphScale` and enlarged.** The
              scale constant lives in `systems` so the sheet reads the same number the card
              does and cannot drift. The first version showed only the enlarged row, which
              is how the glyphs came to look fine in review and clunky in play.
      - [?] **The bevel barely registers at card size.** Seen honestly at 2x, the five-value
            shading that the palette rewrite bought is almost invisible; the clock survives
            best because it is a solid disc, and the sword and figure read as flat
            silhouettes. The work pays off at inspection size and not much at the size that
            matters.
            - The cause is authoring at 32 and doubling: the game gets chunky 2x2 blocks,
              not 64px art. Authoring the shapes at 64 and drawing them 1:1 would buy real
              detail at the size actually used. `disc()` is parametric so the clock scales
              for free; the sword and figure are span tables and would need redrawing.
            - The alternative is to accept it and design *for* 64px — fewer, heavier forms
              rather than detail that dissolves.
      - The iteration that fixed the figure came from *rendering it and looking at it* —
        detached head reads as a hat, head joined to shoulders reads as a blob, head on a
        narrow neck reads as a head. Authoring pixel art without seeing the output is the
        real limitation here, and the sheet tool is the fix.
- [ ] **Bevel the widgets, not just the glyphs.** Stated 2026-08-03: buttons, cards and the
      resolution panes all want the same treatment the glyphs got — a palette with an
      outline, a lit edge and a shadowed one, rather than a single colour scaled up and
      down. The "name one colour and scale it" rule is really about how a surface responds
      to hover, press and disable, and it has been doing duty as a rule about what a surface
      may look like, which is further than it needs to go.
      - `systems.Palette` already exists and is the obvious thing to widen to.
      - Do the buttons first: they have three states to show, so the payoff is visible
        immediately and the state-versus-surface split gets tested by something real.
      - The panes are the least urgent and the largest areas, so a heavy bevel there will
        read as chrome. Worth doing last and lightly.
- [x] **Bigger cards, and double-size glyphs.** Cards went from 130x68 to 224x88 and the
      glyphs draw at 2x. `cardWidth` is now *derived* from the glyph row rather than the row
      being fitted into it, so changing a gap widens the card instead of silently
      overlapping. The palette column widened to 15–39% to hold them.
      - **Integer scaling only.** 32px glyphs were being drawn 1:1 in internal coordinates,
        but the whole 1280x960 frame is letterboxed into the window, so at anything other
        than the native size every pixel was already being resampled at a fractional ratio.
        Doubling first means the art survives that better. Never scale the glyphs by 1.5.
      - `selectedNudge` became a fixed 28px rather than 30% of the card width, which grew
        with the card and started pushing selected cards toward the next pane.
- [ ] **AP cost as dots on each action card — superseded, revisit.** The plan was dots
      rather than a numeral, lining up against a per-point segmented AP bar so a 4-cost
      card visibly is most of a 6-point budget. The glyph row landed first and puts a
      numeral on a runner instead.
      - Dots and a glyph are not obviously compatible — three pips inside a 32px glyph is
        cramped, and a glyph plus dots beside it is saying it twice again.
      - The segmented-bar half of the idea survives regardless and is still worth having.
- [ ] **Stop allocating in `Draw`.** `DrawButton` makes a new `ebiten.Image` per button
      per frame (180/sec); `DrawHealthBar` makes two per bar per frame (240/sec).
      `Button.Image` already exists and is only used for bounds — render into it on
      state change instead. *(analysis §4)*

## Next — where the game actually starts

- [x] **`package combat` as a pure core.** No ebiten import.
      `ResolveRound(a, b Duelist, aActions, bActions []ActionKind, round int)` returns an
      ordered event log for **one round** plus the state both sides end in; the combat
      screen holds the log, a playback cursor and a tick timer, applies one event at a
      time, then adopts the returned duelists when the cursor catches up. 19 tests, the
      first in the repo. `entities.Combatant` embeds `combat.Duelist`, which keeps the
      sprite out of the rules and leaves `gs.Fighter.Str` reading the same as before.
      - Rewriting the turn model from the timeline version touched this file, its tests,
        and ~20 lines of screen glue. No drawing code moved. That is the payoff of the
        split, and it is worth remembering the next time the rules change.
- [x] **Define `Action`.** `ActionKind` is Strike / Guard / Heavy / Quick, costing
      1 / 2 / 2 / 4 action points. A duel is a sequence of rounds: both sides spend an
      AP budget on a set, the two sets resolve **alternately** — one of A's, one of B's —
      then control returns to re-plan. `Spd` buys budget (`AP = 4 + Spd/10`), not turn
      frequency. All integer arithmetic, so rounds are exactly reproducible.
      - Side A takes the first turn, and the screen maps the player to A. A Guard the
        player places is therefore up for the enemy's reply.
      - Whichever queue is longer keeps acting alone once the other empties, so a speed
        advantage is spent at the tail of the round.
      - **Alternating replaced volley-per-side on 2026-07-31**, when the resolution pane
        made the old order visible and it read wrong. The point is that resolution order
        is a thing the player can see and, later, manipulate — speed effects and other
        tweaks need somewhere to bite, and two monolithic volleys gave them none.
        `KindVolleyStart` went with it; there are no volleys to start.
      - `Guarded` persists on the `Duelist` and clears at the start of its owner's next
        *action*, so it covers every opposing action in between, across a round boundary
        if the guard was queued last. A duelist who queues nothing therefore keeps its
        guard — deliberate, pinned by `TestGuardHoldsWhileItsOwnerDoesNothing`, and
        harmless because standing still deals no damage.
      - Superseded an earlier speed-timeline model where a fast duelist took whole
        extra turns. That is wrong for a game you re-plan every round: extra turns are
        invisible when you only ever plan one.
- [x] **DUEL! button.** `DuelButtonAction` resolves **one round** and hands the screen a
      log to replay; control returns to the player to re-plan. It ignores presses during
      playback and once someone is down. A caption line reports what playback is showing,
      and between rounds shows the queued plan and its AP cost.
- [x] **Decide Constitution → HP.** `NewCombatantFrom` sets
      `MaxLife = Con * entities.LifePerCon` and starts `CurrentLife` full;
      `DrawHealthBar` scales the red fill by `CurrentLife/MaxLife` over a darker track,
      and playback of the resolved log drives it. `LifePerCon = 5` survived the rebalance
      and now gives Fighter 60 and Monster 100 — the lopsided 60-vs-160 spread was a data
      problem, not a constant problem, and was fixed in `combatants.json`.
- [x] **Drag-and-drop for the action box.** Built hand-rolled in
      [combat_actionbox.go](internal/screens/combat_actionbox.go). Drag a card out of the
      green Player palette into the blue Chosen pane to queue it, drag a chosen card to a
      new position to reorder it, drag one out of the pane to discard it. The Resolution
      pane updates live, because it reads `s.fighterActions` directly.
      - Hand-rolled, no toolkit — decided, see "Open decisions". Drag state lives on
        `CombatScene` alongside everything else it owns.
      - **Flow, decided:** build a plan → DUEL! → watch playback → build a *fresh* plan →
        DUEL! again. The queue empties every round; nothing carries over. Makes each
        round a real decision rather than a default nobody revisits.
      - `defaultFighterPlan` is deleted — it only ever stood in for the player.
      - The planning/playback phase stays derived from `cursor >= len(log)` rather than
        becoming its own field, so there is one source of truth. `planning()` is the one
        predicate; drag and DUEL! both gate on it.
      - DUEL! is disabled while the queue is empty — first real use of the existing
        `models.ButtonStateDisabled`. An empty plan is mechanically legal and
        `ResolveRound` handles it, but it means standing still while being hit.
      - Under-spending the budget is allowed; forbidding it would add a rule for no gain.
      - **The budget is enforced at pick-up, not at drop.** An action the remaining points
        will not cover cannot be lifted off the palette at all, and draws dimmed. Letting
        a card be dragged and then bounced is a worse conversation than never letting it
        leave. Re-checked on drop anyway, since the two are separated by time.
      - **A card lifted out of the queue leaves it immediately** rather than on drop, so
        the gap closes under the cursor and the insertion index is measured against the
        list the card actually lands in. Dropping outside the pane is therefore the
        removal gesture, with no separate delete affordance to find.
      - **No "repeat last plan" shortcut.** Explicitly not wanted: every round being a
        real decision is where the balance and design work lives, and a repeat button
        undercuts that. Revisit only if playtesting proves it tedious — do not add it
        pre-emptively.
- [x] **Hide the enemy's plan, behind `DebugGameplay`.** The yellow pane and the enemy rows
      of the Resolution pane showed the opponent's queued actions while the player built
      theirs — perfect information, and only ever there so the pane had content. Both now
      render `???` unless `DebugGameplay` is on, so the plan stays readable while debugging
      alongside the layout guides and is invisible in normal play. `CombatScene.concealEnemy`
      is the single predicate: `!gs.DebugGameplay && s.planning()`.
      - Concealment lifts once playback starts. An action that has already happened is not
        a secret, and the Resolution pane still has to narrate the round.
      - [?] **What it still leaks: the row count.** A concealed queue occupies its real
            number of rows, so the opponent's AP spend is readable even when the actions
            are not — and against `PlanGreedy` that is most of the tell. Deliberate:
            collapsing the rows would hide the spend but destroy the Resolution pane's
            account of who acts when, and that alternation is a rule the player is meant
            to read and eventually manipulate. Settle this with the wider question below.
      - [?] **The wider question, not yet answered:** which information is hidden and which
            is random. Those are different levers — hidden-but-deterministic rewards
            reading the opponent, random rewards hedging — and the interesting play is in
            choosing per mechanic rather than applying one rule to everything. Note that
            hidden information is listed under "is this seed winnable?" as a property that
            would break solvability, so this decision has a cost recorded elsewhere.
      - There is no runtime toggle for `DebugGameplay`; it is set once in `main.go`. A hotkey
        would be the obvious convenience and collides with the no-keyboard rule in
        `CLAUDE.md`, so it needs a decision rather than a keystroke.
- [x] **Initiative on actions, contesting the paired slot.** Every `ActionKind` has an
      `Initiative()` — Quick 1, Guard 2, Strike 3, Heavy 5 — and within an exchange the
      faster action lands first, side A taking a tie. Alternation is unchanged; what
      initiative decides is who leads each pairing.
      - **Why contested slots and not a global initiative sort.** Sorting every action from
        both sides into one pool is the purer reading of "how quickly you can act", and it
        makes dragging a card to a different position *do nothing* — the sort would already
        have decided. It also reproduces the speed-timeline model rejected above, where a
        fast duelist takes several actions in a row. Contesting the paired slot keeps the
        alternation guarantee and makes position the decision.
      - `combat.ResolutionOrder(a, b) []Slot` is now the single authority on order.
        `ResolveRound` plays it, the Resolution pane draws it, and
        `TestResolutionOrderIsWhatResolveRoundPlays` pins that they agree. Any future
        reordering effect changes that one function.
      - Guard at 2 is deliberately faster than the Strike at 3 it answers, so guarding in
        the same exchange beats the blow rather than arriving after it.
      - `Spd` still buys action points and never priority. Two separate levers:
        `TestSideAActsBeforeSideB` keeps a duelist 500x faster from leading an exchange.
      - Cards show `init N` under the name. Without it the Resolution pane reorders for
        reasons the player cannot see.
      - [?] **Initiative values are a first guess, not balanced.** 1/2/3/5 across
            Quick/Guard/Strike/Heavy is spaced to make every pairing decisive rather than
            tuned against the AP costs of 1/2/2/4. Whether Heavy should be beatable by
            *everything* is exactly the sort of thing the headless balance sim should
            answer.
- [?] **Ordering model — three candidates, one implemented, none settled.** Contested slots
      is what ships today. Two alternatives came out of the 2026-08-02 discussion and both
      are live options; `ResolutionOrder` is a single pure function, so swapping between
      them is one function body plus its tests.
      - **Contested slots (built).** Queues alternate; initiative decides who leads each
        pairing. Every action of yours meets one of theirs — "every ask gets an answer".
        The cost: a fast action placed late still resolves late, because initiative never
        lets an action jump to an earlier exchange. Quick means "wins its exchange", not
        "happens early".
      - **Wind-up time.** Initiative is how long an action takes to come out, accumulating
        down the queue; actions resolve in time order across both sides. Gives the "land
        three fast hits before the slow enemy connects" feel, and reordering still bites
        because your queue is consumed in order. Costs the pairing symmetry entirely.
        Distinct from the rejected speed-timeline model: tempo is bought with card choice,
        not accrued from `Spd`, and AP still caps the action count.
      - **Initiate / respond.** The richest and the biggest change. An exchange is one
        initiator's action plus the opponent's response *if they queued one*; whoever's
        next card is faster initiates, and a card that cannot respond is not consumed —
        it waits. That waiting is what lets a fast plan land several blows against a slow
        one while keeping action-and-answer where both sides planned for it.
        - Needs a taxonomy that does not exist: **role** (can it initiate, respond, or
          both), **response timing** (does the answer land before or after the blow),
          **consumption** (does answering spend the card), **effect** (modify, negate,
          counter). Today's four sit as initiate-only except Guard, which is respond-only.
        - The payoff falls out of response timing for free: let a response resolve first
          only when its initiative is *lower*, and Guard at 2 blocks Strike (3) and Heavy
          (5) but is too slow against Quick (1). **Quick becomes the guard-breaker** from
          numbers already on the cards, and a defensive card's initiative becomes the
          single number saying what it is for.
        - It also dissolves the Guard-persistence oddity. A Guard consumed by the attack
          it answers has no "lasts until its owner acts again" rule to reason about, and
          `TestGuardHoldsWhileItsOwnerDoesNothing` stops describing anything.
        - **Deadlock is a real hazard.** If cards are spent only by initiating or
          responding, two sides holding nothing but responses have nobody to initiate and
          nothing to resolve — an unbounded round. Needs an explicit rule (discard
          unanswered responders at round end, or let a responder initiate as a no-op) and
          a test, because it will not show up until the AI learns to guard.
        - Open besides that: is a too-slow response wasted or does it carry to the next
          attack; do unanswerable cards wait forever or get skipped; should responses cost
          less AP than attacks.
      - Build the palette/resolution layout before choosing. All three produce an
        interleaved order, so the layout is not wasted on any of them, and dragging cards
        around is a better way to feel the difference than reasoning about it.
- [ ] **Graded reveal of the enemy's actions — the design this is heading toward.** A
      concealed action currently shows `??? (i3)`: initiative always leaks, the name never
      does. That is one cut, chosen because hiding initiative makes the Resolution pane
      unreadable — the player could not tell why the rows sit in that order.
      - The proposal worth building out: reveal *categories* rather than identities. Does
        it damage? Does it apply a status? How fast is it? The player reads the shape of
        the opponent's round and plans against it without knowing the specific card.
      - This is the concrete form of the hidden-vs-random question above. Hidden but
        graded is a third thing from either, and probably the most interesting: it rewards
        reading without punishing with pure guesswork.
      - Wants a reveal level per action rather than the current boolean, and something on
        the enemy side that decides how much leaks — an affix, a ring, a floor property.
- [ ] **Deckbuilder — the direction the action box is growing into.** Decided 2026-08-02.
      Actions stop being four verbs and become a deck of card *instances*: play a card, it
      goes to a discard pile, and thinning the deck is a reward the player can be offered.
      Think dozens of cards rather than four, though not a full 52.
      - **Decided — the screen.** Four columns: **fighter / palette / resolution / enemy**,
        with the duelists as bookends and the round between them. The Chosen pane and the
        Enemy pane both go away. Chosen because the palette can hold the ordered queue
        itself; Enemy because an interleaved Resolution already shows their actions in a
        better order than a separate column does. **The Resolution pane becomes the
        centrepiece of the scene** and inherits the freed width, which it needs — with a
        response model it has to draw exchange structure, not a flat list of rows.
      - **Decided — the interaction.** Click a card in the palette to select it, drag to
        order the selected ones, and the palette order falls through into the Resolution
        pane. The action-point budget gates *selection*: a card the remaining points will
        not cover cannot be selected, which is today's pick-up rule moved to the click.
        - Splitting the gestures this way is what makes it safe. Dragging into an
          interleaved list means the drop position is not where the card lands — your
          index is measured against your own queue, and the opponent's actions
          re-interleave around it. Dragging only ever *reorders* cards that already
          exist, and adding is a click somewhere else entirely.
        - The Resolution pane must re-interleave live under the cursor while dragging, or
          the reorder is guesswork. `ResolutionOrder` is a pure function over two slices,
          so this is a per-frame call with the hypothetical placement, not a cache.
        - Removal becomes clicking a card off in the palette. Today's gesture is dragging
          it outside the pane, which nothing on screen suggests.
      - **Decided — selection clears every round, palette order may persist.** Persisting
        the arrangement is fine and probably good; persisting the *selection* rebuilds the
        "repeat last plan" shortcut that is explicitly rejected above.
      - **Thinning and drawing are the same decision.** A reward for slimming the deck
        means nothing if the whole deck is visible every round — that is just a longer
        palette. Committing to thinning commits to a hand, a draw, a discard and a
        reshuffle, and those arrive together rather than one at a time.
      - **This puts randomness inside `internal/combat` for the first time.** A shuffle is
        a fourth seeded stream alongside enemy selection, loot offers and floor offers.
        Consistent with the determinism rules — it arrives as an injected source, never a
        global — but `ResolveRound` grows a source parameter and
        `TestRoundIsDeterministic` changes shape to seed it explicitly.
      - **Open, and needed before any of this is buildable:**
        - Hand size, deck size, cards drawn per round, and what happens when the deck runs
          out mid-fight. All four interact with the AP budget.
        - Whether `AP = 4 + Spd/10` survives unchanged once it is gating selection from a
          hand rather than from a fixed palette of four.
        - Whether the enemy has a deck too. If it does, affixes become cards shuffled into
          it and "this is a cold floor" stops being a stat modifier and becomes a literal
          deck edit — which fits the enemy model below far better than what is planned
          there now.
        - Whether the deck is per-run or per-fight, and whether the discard reshuffles
          within a fight or only between them.
      - **First slice landed 2026-08-02.** A 20-card starting deck — 10 Strike, 6 Guard,
        4 Heavy — with a hand of five, a discard pile, and a reshuffle when the draw pile
        runs dry. Click a card to select it, drag to reorder, and the queue is derived from
        the hand by `syncQueue` rather than stored twice.
        - The whole hand discards at the end of a round and five are drawn fresh. Keeping
          cards back would let a plan be prepared once and repeated, which the "no repeat
          last plan" decision above rules out. Worth revisiting if it plays badly — hand
          retention is a real deckbuilder lever, just one that fights that decision.
        - The discard happens when *playback* finishes, not when the round resolves.
          `discardHand` rebuilds `fighterActions`, and the Resolution pane draws
          `fighterActions` to narrate the round, so discarding early empties the pane
          mid-round. The ordering in `advancePlayback` is load-bearing.
        - `Quick` is defined in the rules but absent from the deck. Which of the actions
          the rules permit actually appear is a deck-building question, and that split is
          the point of having a deck at all.
        - The 10/6/4 composition and the hand size of five are first guesses, unplayed.
        - **Still missing before this is a deckbuilder rather than a card-shaped hand:**
          nothing adds or removes cards, so there is no building and no thinning reward.
          That needs the loot loop, which needs the tower.
- [ ] **Real opponent AI.** `combat.PlanGreedy` buys the biggest attack it can afford and
      never guards. Deterministic and fine for testing, but not a fight.
- [x] **Wire `Str` / `Spd` / `Con`.** `Con` feeds `MaxLife`, `Str` sets action damage,
      `Spd` sets turn frequency. All three are live.
- [x] **Rebalance `combatants.json` — measured, not guessed.** Was Fighter
      Str10/Spd11/Con12 vs Monster Str1/Spd31/Con32, which gave 16 rounds against a
      monster that removed 15 life total. Now Fighter **Str10/Spd20/Con12** (60 life,
      6 AP) vs Monster **Str10/Spd15/Con20** (100 life, 5 AP). Verified headlessly:
      - `Guard + Strike + Strike` **wins** on round 5, fighter ends on 12/60.
      - `Heavy + Strike` **loses** on round 3, monster still on 10/100.
      - The greedy plan deals more damage per round and still loses the race, so the
        plan choice decides the fight. That is the balance point worth protecting when
        these numbers change again.
      - ~1.8s of playback per round.
      - Raising Monster Str to 10 also removes the degenerate case where Guard halved a
        1-damage hit to nothing.
      - [?] Still open: should Guard subtract flat armour rather than halving? `ideas.md`
            lists armour as a core attribute alongside speed and damage, so halving may
            be the wrong long-term shape even though it now behaves.

## Later

- [ ] **Game speed setting.** User-facing options: *very slow · slow · normal · fast ·
      very fast*, scaling how quickly the duel event log plays back. Ship "normal" only
      to begin with, but route it through a setting rather than a constant so the other
      four are a data change later.
      - Today the pacing is `duelTicksPerEvent = 8` in
        [combat.go](internal/screens/combat.go) — one constant, one caller, so this is
        cheap right now and gets steadily more expensive as animation and sound land and
        each grows its own timing constant.
      - Speed must scale *presentation only*. `combat.ResolveRound` already decided the
        whole round before playback starts, so speed can never change an outcome — worth
        protecting, since "fast mode plays differently" is a classic bug in this shape of
        game.
      - Belongs to whatever settings screen eventually backs `SettingsButtonAction`,
        which currently only prints.
- [ ] **Seeded randomness for replayable runs.** Goal: record a seed with a run so the
      same enemies and offers can be replayed while the player makes different choices.
      Nothing is stochastic yet, so there is no work to do — but see the determinism
      rules in `CLAUDE.md`, which exist to keep this cheap.
      - **Three separate streams, decided:** enemy selection, loot offers, floor offers.
        Tower layout is *not* a stream — it is fixed at 8 floors of 3 fights.
      - Separate streams mean adding a random call in one system cannot shift another
        system's results. Without that, tweaking loot generation silently rerolls every
        enemy in the tower, and balance testing becomes impossible to reason about.
      - Show the seed somewhere and allow entering one, or the feature is invisible.
      - `internal/combat` is currently pure integer arithmetic with no randomness at all,
        and `TestRoundIsDeterministic` pins that. Keep it that way — if randomness enters
        combat, it arrives as an injected source, not a global.
- [ ] **Don't pre-roll into a fixed array — keep a seeded stream per concern.** A
      `*rand.Rand` seeded once *is* an infinite deterministic list; a pre-generated slice
      is just the first N entries of it, and N has to be guessed. The endless tower has
      no worst case to size against, so any N is eventually wrong.
      - **Rerolls advance the cursor**, which is exactly the intended behaviour: reroll
        and you get the next offer down the list. No separate reroll stream needed.
      - Replay stays exact because the *list* is fixed by the seed. Identical choices
        consume identical draws; different choices land at a different position in the
        same list. That is the property worth having, and it survives rerolls.
      - Materialize a window of a stream into a slice only when something needs to
        *inspect* it — a balance sim or a test — not as the storage model.
      - The one discipline this needs: a stream is only ever advanced by its own
        concern. Never borrow the loot stream to pick an enemy.
- [x] **`Scene` interface** replacing the two parallel switches in `game.go`, plus the
      per-screen half of the `GlobalState` split — the same trigger fired both, and they
      were one piece of work. *(analysis §1, §2)*
      - `Scene` is `Init`/`Update`/`Draw`; scenes are registered once in a map, so the
        `Update` and `Draw` paths can no longer drift apart.
      - Scenes own their state *and* build their own widgets, wiring them to their own
        methods. `main.go` no longer constructs buttons for screens it knows nothing
        about, and `DuelButtonAction` became `CombatScene.startRound`.
      - `NewScreen` is consumed centrally in `game.Update`; scenes never touch it.
      - `GlobalState` went from 34 fields to 20 and no longer imports `combat`,
        `entities` or `models`.
- [ ] **Split the rest of `GlobalState`** into `Resources` (assets/fonts/data,
      read-only), `Layout`, and `Session` (run progress). Deferred: `Session` has nothing
      to hold until the tower loop exists, and the remaining fields are not crowding
      anything. The seed streams will live in `Session` when it lands.
- [ ] **Ascend / tower loop.** `ascend.go` is a bare `package screens`; `Ascend` and
      `Credits` are empty cases in both switches. Structure decided:
      - **8 floors, 3 fights each — 24 fights to the top.** The layout is fixed, not
        generated. Only the enemies and the offers are random.
      - **A binary loot choice after every fight.** Two options, pick one.
      - **A binary floor choice after the last fight on a floor**, on top of that fight's
        loot choice.
      - **Floor 8 ends the run** for the first version — 7 floor choices, no offer at the
        top.
      - Floor choices steer **enemy affixes and behaviour** — "this is a cold floor",
        "this is a fire floor" — plus whatever other levers exist by then. The specific
        options are undecided; the mechanism is the part that matters.
      - Run progress (current floor, current fight, collected rings/brands/pets) is the
        `Session` state in the `GlobalState` split below — this is the feature that
        forces that refactor.
- [ ] **Save format: seed plus choice log, not serialized state.** Falls out of seeding
      for free, and only stays free if nobody builds save/load the other way first.
      - A run is fully described by its seed and the ordered list of **every player
        input**, which is more than the loot and floor picks:
        - **The action set queued each round.** ~5 rounds x 24 fights, so this is the
          bulk of the log, not a footnote.
        - Which of the two loot offers was taken, per fight.
        - Which of the two floor offers was taken, per floor.
        - Every reroll — it is a decision *and* it advances a stream, so omitting it
          desyncs everything after it.
      - A few KB rather than a few dozen bytes, and it grows with duel length rather
        than being fixed size. Still trivial. It survives every change to the shape of
        in-memory state, and doubles as a replay file and a reproducible bug report.
      - Recording action plans is what makes hand-editing a save interesting: loot picks
        only answer "what if I took the other ring", where plans answer "what if I had
        guarded on round 3". It is also what makes a "this seed is winnable" claim
        checkable — a proof is just a choice log that replays to a win.
      - **Serialize action names, not `iota` ordinals.** `ActionKind` is `iota`-based, so
        inserting a new action anywhere but the end silently reinterprets every existing
        log — a saved `Guard` becomes whatever now sits at 1, with no error. Same applies
        to any other enum that reaches the save file.
      - Serializing live state instead means a migration every time state changes — the
        refactor this whole set of decisions exists to avoid.
      - Cost: loading replays the run to reach the current point. Trivial here, since
        combat is pure integer arithmetic and a whole duel resolves in microseconds.
      - Caveat: this only holds while the rules are stable. A balance change invalidates
        old saves, so the format needs a rules-version stamp and a plan for what happens
        when it does not match.
- [ ] **Watch: "is this seed winnable?" as a solvable question.** Not a feature to build
      now — a *property to avoid destroying*. Deterministic combat plus deterministic
      streams plus a bounded choice space means a run is in principle searchable.
      - What it would give: guaranteeing a daily seed is beatable, difficulty grading a
        seed by how narrow its winning lines are, and finding degenerate loot combos
        without playing thousands of runs.
      - Feasibility is unclear and worth being honest about. 24 binary loot choices and 7
        binary floor choices is only ~2^31 paths, which pruning handles easily — but the
        *combat plan* each fight is also a choice, and the number of ways to spend an AP
        budget multiplies that out fast. Fixing a plan policy makes it tractable and
        answers a weaker question: winnable *by this policy*.
      - What would kill it, and therefore what to weigh decisions against:
        - Hidden information the solver cannot see.
        - Randomness resolved *during* a fight from an unseeded source. Amended
          2026-08-02: a *seeded* shuffle is not this. The deck order is fixed by the seed,
          so a solver exploring a line of play knows its draws exactly and replay stays
          exact — the branching factor grows but the question stays well-posed. The
          deckbuilder direction therefore costs less here than this entry first implied.
          What would genuinely kill it is randomness drawn from a global or a clock, which
          the determinism rules already forbid.
        - Unbounded state — anything that makes a position not comparable to another.
        - Real-time or reflex elements.
      - None of those are on the roadmap, so the property is currently free. Re-read this
        before adding anything that breaks one.
- [ ] **Endless tower (after the 8-floor version works).** Keep climbing until the curve
      stops you, rather than a fixed summit. Scaling probably exponential.
      - Design the floor loop so 8 is a *configured stop*, not a baked-in constant, or
        this becomes a rewrite instead of a setting.
      - Exponential scaling wants a sanity check on integer range and on the health bar:
        `DrawHealthBar` scales by `CurrentLife/MaxLife` so it copes, but a four-digit
        damage number will not fit the current caption.
      - The interesting design question is what actually stops you. Enemy stats
        outrunning yours, or a resource that runs down?
- [ ] **Enemy model: one archetype, scaled and affixed.** Enemies are essentially the
      same creature with main stats growing by depth, plus affixes that may stack. This
      contradicts how `combatants.json` is shaped today — a flat list of fully-specified
      records — so the data wants to become a base statline, a scaling rule, and a pool
      of affixes to draw from.
      - `AvailableAffixes` already anticipates this and is still unread.
      - Affixes must compose. Two on one enemy is the normal case, not an edge case.
      - Floor choices feed this directly: "a cold floor" biases which affixes appear.
- [ ] **Headless balance sim for the difficulty curve.** The curve gets playtested a lot,
      and most of that testing should not require playing.
      - `internal/combat` has no Ebitengine dependency precisely so this is possible:
        run thousands of duels across floor depths and plot where the player loses.
      - Needs the enemy scaling rule and a plausible player-plan model first, so it is
        downstream of the two items above — but it is the reason to keep combat pure.
- [ ] **Affixes.** `cold` / `hot` / `charged` / `undying` are in `combatants.json` and
      never read. Ring and effect art is partly present.
- [ ] **Replace Tyrian placeholder art — now a release blocker, not a cosmetic swap.**
      Technically still easy: the `SpriteSheet` + `SpriteRect` indirection means it's a
      data change. Keep it that way — no identifiers or packages named for it, no
      assumptions baked in about its palette or rect conventions.
      - The reason it got promoted: the Tyrian set has no formal license, only "use and
        abuse them as desired" on a 2007 Lost Garden blog post. That is fine for a public
        hobby repo and thin for a paid release. **This has to be resolved before the game
        is sold anywhere.**
      - Applies to everything under `assets/tyrian_graphics/` plus the two consolidated
        sheets actually embedded (`tyrian_monster_sprites.png`, `tyrian_ship_sprites.png`).

## Housekeeping

- [x] `data.Combatants` package var is declared and never assigned — dead, and a trap.
      Removed; `LoadCombatants()` returns the map and `GlobalState.Combatants` holds it.
- [x] `Update*Screen` returns `error`; callers discard it. Now propagated out of
      `Game.Update`. Ebitengine stops the loop on any non-nil error from `Update`, so a
      screen error is fatal by design — the only sensible reading of an error a screen
      cannot handle itself.
- [x] `Life_max` / `Life_current` → `MaxLife` / `CurrentLife` (Go naming).
- [x] `CreateRoundedRecMask` left edge is `radius+width` wide, should be `radius`. Works
      only because it's clipped at x=0. Fixed — no visual change, since the overdraw was
      always clipped away; it would have corrupted the mask the moment the function was
      called with a non-zero x.
- [x] Quitting exits status 1 and logs `Closing` like a crash. Now `game.ErrClosing`, a
      sentinel, with an `errors.Is` check in `main` — a deliberate quit exits 0 silently
      and anything else still goes to `log.Fatal`.
- [x] `.gitattributes` with `* text=auto eol=lf` — every file used to fail `gofmt -l` on
      line endings alone, making the check useless. PNG and TTF marked binary so they are
      never converted. The existing working tree was normalized with `gofmt -w`; the file
      only governs what git writes on checkout, so it would not have fixed files already
      on disk. `gofmt -l .` is now clean and worth running.
- [x] Embedded-asset drift: `thunder-ring.png` and the three `*-effect.png` files existed
      in `assets/` but were not embedded. All four now embedded and mapped
      (`thunderring_png`, `fireeffect_png`, `frozeneffect_png`, `thundereffect_png`),
      ready for the affix work.
- [x] Local `main` branch was 47 commits stale — fast-forwarded, and the merged
      `dad-work` / feature branches deleted locally and on the remote. `origin` now
      carries `main` only.

## Open decisions

- [x] **Hand-rolled UI vs [ebitenui](https://github.com/ebitenui/ebitenui) — decided:
      hand-roll everything.** Not part of Ebitengine; a separate community project.
      - **The design decision that settles it:** every interaction in this game is a
        click or a drag-and-drop, and there will be exactly **one** text input in the
        whole game — the seed field. Recorded as a firm rule in `CLAUDE.md`.
      - Drag-and-drop was the wrong trigger for this decision. The action box is a *game*
        widget — draggable cards with live AP validation — and general-purpose toolkits
        are weakest at bespoke game widgets. Roughly 200–300 lines hand-rolled, using hit
        testing and `inpututil` handling the codebase already has.
      - Repo data as of 2026-07-31, via the GitHub API: created 2020, last push
        2026-04-22, 928 stars, 74 forks, **73 open issues**, MIT, not archived. Releases
        v0.7.3 (Mar 2026), v0.7.2 (Sep 2025), v0.7.0 (Aug 2025). Contributors:
        mcarpenter622 322 commits, blizzy78 151, mat007 42, then a long tail.
      - What that means: two people are effectively the project, it is still pre-1.0
        after six years (so API churn is expected, not just abandonment risk), and a
        non-critical issue would likely sit. Fine for a hobby dependency, poor for
        something load-bearing in a product being sold.
      - It is also retained-mode — it owns a widget tree and its own event dispatch —
        which is a second UI architecture running alongside the `Scene` design rather
        than a layer that slides out cleanly.
      - [?] Revisit **only** if the seed text field turns out to be genuinely painful.
            Even then it is the easiest possible text input: short, ASCII, fixed
            charset, no selection or clipboard or IME needed. `ebiten.AppendInputChars`
            plus backspace handling is plausibly ~60 lines, so this trigger may never
            actually fire. If it does, confine any toolkit to whole screens and keep its
            types out of scene fields and signatures.
- [?] **Title hue shift.** `title.go` calls `ChangeHSV(1, 1, 1)`; `hueTheta` is *radians*,
      so that's a ~57° rotation, not identity. Source PNG is warm gold/amber. If the
      title looks greener on screen than the file, it's unintended — identity is
      `ChangeHSV(0, 1, 1)`.
- [x] **Start screen.** Flipped from the `Combat` dev shortcut back to `Title`. Combat is
      one click away and is now worth navigating to. Flipping it is also what surfaced
      the screen-transition nil deref above.

## Licensing (for an eventual Steam release)

Model: source stays public, nobody else may commercialise it. Justin and Sherman
have a signed agreement covering the relicense.

- [x] **Relicensed from Apache 2.0 to PolyForm Noncommercial 1.0.0.** Source-available,
      not open source. Read / build / modify / share for noncommercial purposes; selling
      is reserved to the copyright holders. README states the terms in plain English.
- [ ] **Put Sherman's legal name in `LICENSE`.** The Required Notice currently names the
      GitHub handle `KingSherman1820`. Deliberately deferred — a written partnership
      agreement covers the two of them — but a copyright notice naming only a handle is
      weak if it ever has to be enforced.
- [x] **Contributor grant — `CONTRIBUTING.md`.** Under a noncommercial licence an outside
      contributor keeps copyright and has *not* granted the right to sell their work, so
      merging unlicensed contributions would leave the game unsellable. Contributors now
      grant a perpetual, royalty-free, commercial-use-and-relicense licence, signalled by
      `git commit -s` (DCO convention). Landed before the repo has outside PRs, which is
      the only cheap time to do it.
- [x] **Streaming and video explicitly permitted, including monetized.** Additional
      Permissions section at the top of `LICENSE`, restated in the README. Keeps the
      standard PolyForm text unmodified rather than editing the licence body.
- [ ] **`THIRD-PARTY-NOTICES` file.** Apache-2.0 and BSD deps may sit inside a
      restricted-licence product, but only if their notices and attributions travel with
      the binary. Needed for a Steam build, not for the repo.
- [ ] Contact address for licensing enquiries. Deferred deliberately; anonymous is fine
      for now, and `CONTRIBUTING.md` points people at issues instead. Use a purpose-made
      address rather than a personal one when it happens.
- [ ] Get thirty minutes of actual legal review before relying on any of this. The
      licence is standard and well drafted; the contributor grant in `CONTRIBUTING.md`
      is a reasonable draft written by a non-lawyer.
- [x] Ebitengine — Apache 2.0. Permissive, so it can be included in a
      noncommercially-licensed product; notices must be retained.
- [x] All transitive deps — BSD (`golang.org/x/*`) or Apache. **No GPL anywhere** — this
      is what makes the relicense possible at all.
- [x] `Kubasta.ttf` — CC0, per the author's own FontStruct page.
- [ ] Confirm FiraSans and RobotoFlex (expected OFL / Apache — low risk).
- [ ] Tyrian art — see the release blocker under "Later". No formal licence, and that
      must be resolved before the game is sold.

Note: the Apache 2.0 grant on everything published before this change is irrevocable.
Anyone who already had the code keeps commercial rights to those snapshots. Accepted
deliberately — there was not enough there to matter — so no history rewrite. The new
licence governs from here forward.
