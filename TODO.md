# TODO

Working notes for ascend-duel. Findings and reasoning live in `analysis.md`;
this file is just the running list.

**The design record moved to [MECHANICS.md](MECHANICS.md) on 2026-08-05.** That file is what
the game *is*; this one is what to build. Entries below still carry the reasoning behind
things already implemented, which is worth keeping — but **when the two disagree,
`MECHANICS.md` is newer and wins.** Say so rather than guessing.

A large batch of design was captured on 2026-08-05 and is *only* in `MECHANICS.md`: the
element set and their statuses, the card types, combos, rings, brands, vitae, the tower's
doors and stairwells, enemy decks and affixes, and the phase-based resolution experiment. None
of it has tasks here yet.

Status: `[ ]` open · `[x]` done · `[~]` in progress · `[?]` needs a decision

---

## Now — quick wins, independent of any design decision

- [ ] **A mute button.** `internal/music` landed 2026-08-09 and the score loops across
      every screen with **no way to turn it off**, which is not shippable. It cannot be a
      hotkey — the input vocabulary has no keyboard — so it is an on-screen speaker
      toggle, built the ordinary way: a struct in `models`, `UpdateMute`/`DrawMute` in
      `systems`. Two open questions before building it:
      - **Where does it live?** A corner of every screen means every scene owns one, or
        it gets drawn centrally after the scene like the debug overlay is. The second is
        less code and puts a widget outside the "scenes own their own widgets" rule; the
        first is consistent and repeated four times.
      - **Does it want to be a volume slider instead?** A slider is inside the input
        vocabulary (drag), and settings values are "a row of buttons or a slider, never
        a number you type". But there is no settings screen yet to put one on, and a
        binary toggle is what an unmutable loop actually needs today.
- [ ] **The score's loop point is rounded, not authored.** `loopTicks` rounds the last
      note-off to the nearest bar, which for `ascending.mid` trims 60 ticks (about 62ms)
      of a drum tail past bar 13. That is inaudible and the tail is folded back over the
      start anyway, but the rounding is a *guess at intent*. If a future score wants a
      loop that is not its full length — an intro bar played once, say —
      `audio.NewInfiniteLoopWithIntro` already supports it and the loop point would need
      to come from the file (a marker meta-event) rather than from arithmetic.
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
- [x] **One dwell for every event** *(2026-08-06)*. `dwellFor` and its three-way switch are
      gone; `eventDwellTicks` (0.75s) is the single number and `advancePlayback` reads it
      directly, so there is no per-kind table to add a case to.
      - **The switch had a `default` arm and the default was the shortest of the three.**
        Every event kind added after it was written silently inherited a quarter-second
        flash. `KindNegated` did exactly that on arrival: a Dodge cancelling a Heavy — about
        the most consequential thing in a round — went past faster than the round-start
        beat. The bug was the shape, not the numbers, so the shape went.
      - Costs a longer round to watch, and a round-start marker now holds as long as a
        killing blow. Accepted: one constant is the price of never having a new event kind
        quietly pick its own pacing. It is still the constant game-speed will scale.
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
        - [~] **Colour was deliberately unspent, and is now being spent.** One hueless
              `white` palette for all three, so that an element or block type could land on
              colour later and mean something on arrival. Elements arrived 2026-08-04 — see
              the entry below. What is still open is whether the *glyph* takes an element
              palette or only the card surface does.
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
- [x] **Segment the AP bar** *(2026-08-07)*. One cell per action point instead of a
      continuous fill. Action points are whole numbers spent in ones and twos, and a smooth
      bar made the player read the `3/6 AP` line to find the remainder — which is the small
      text the bar exists to save them from. Three lit cells and three dark ones says "three
      left" without a number.
      - The cells make the budget boundary draw itself, so the white tick that used to mark
        it is gone: where blue meets red *is* the edge of what can be afforded.
      - Falls back to one unbroken bar below `apBarMinCell`. A big enough AP bonus would
        make cells thinner than a couple of pixels, and stripes that fine are a smear rather
        than a count.
- [x] **Stamp the version into the binary** *(2026-08-07)*. `-X main.version=<tag>` at link
      time, shown in the window title and on the title screen; defaults to `"dev"` because a
      plain `go run .` injects nothing.
      - **The point is that a bug report can name a build.** The filename carries the version
        but stops travelling with the binary the moment it is renamed, or when a screenshot
        is all you have. The window title is in any screenshot of the window.
      - The title screen is skipped on boot while combat is the screen under construction, so
        the window title is the one that actually gets seen today.
- [x] **CI and a release pipeline, and v0.1.0 shipped** *(2026-08-06/07)*. The repo had
      neither. `ci.yml` gates every PR; `release.yml` fires on a `v*` tag and publishes.
      **v0.1.0 is live** with a Windows exe, verified by downloading the published asset back
      and running it.
      - See the *Releasing* section of `CLAUDE.md` for the shape and the reasoning.
      - **Tag-triggered, not release-on-merge.** Considered and declined on 2026-08-07:
        continuous delivery was the better default for a prototype whose players are friends,
        but the owner chose to keep releases deliberate. The version stamp that CD would have
        *required* was taken anyway, since it is worth having either way.
- [ ] **Verify the Linux build actually works.** **Nothing has ever run it.** The workflows
      assert Linux is supported and that claim is untested — the first CI run on a PR is the
      real check, and the apt dependency list is the most likely thing to be wrong.
      - It **cannot be checked locally from Windows**: `GOOS=linux go vet` fails inside
        Ebitengine's OpenGL driver because cross-compiling disables cgo and the Linux driver
        is cgo-gated. That is the same reason the runner needs the X11/GL/ALSA headers.
      - Linux was added to *CI* rather than only to *release* precisely for this: better a
        red PR than a release that dies halfway with the Windows half already published.
      - **v0.1.0 will never gain a Linux binary** — it is already cut. A Linux download needs
        a new tag, and that tag needs its own notes file (v0.1.0's says "Windows x64 only").
      - Sherman has a GUI Linux box and is the intended tester; the owner's is headless.
- [x] **Linux release builds** *(2026-08-07)*. Both CI and release now cover Windows and
      Linux; `ubuntu-22.04` rather than `latest`, because a cgo binary links against its
      build machine's glibc and will not start on anything older.
      - Ebitengine is pure Go on Windows and **cgo on Linux**, linking X11, GL and ALSA. Both
        workflows install the same apt list and have to be changed together.
      - **Shipped as `.tar.gz`.** GitHub release assets carry no file permissions, so a bare
        binary downloads without its execute bit and will not run.
      - Release restructured into build jobs that upload artifacts plus one `publish` job, so
        two jobs cannot race on `gh release create`.
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
- [x] **Portrait cards along the bottom, with the buttons underneath.** Decided and landed
      2026-08-04. The action cards are playing-card rectangles — taller than wide — laid out
      in a row across the bottom, with Discard / DUEL! / Deck below them. The reason is
      capacity: a horizontal row of portrait cards shows more at once than a vertical column
      of landscape ones, and the hand is going to grow.
      - **The plan was to invert the derivation — key `cardHeight` off the glyph column —
        and that was rejected on sight.** The card size is now two flat constants, 180x264.
        "I need static card sizes to understand what's going on": a derived size means adding
        a badge silently resizes every card in the game, and a layout you cannot read off the
        source is a layout you cannot reason about. Contents fit the card, never the reverse.
      - **The glyphs stack vertically down the left edge**, the way a real card puts its rank
        and suit in a column at the corner, with each number beside its glyph rather than
        across it. There is room now, and a numeral in open card beats one fighting a bevel.
      - **`selectedNudge` rotated with the layout.** A selected card lifts *up* out of the
        row rather than sliding right, and drag-to-reorder is horizontal.
      - **`handBand()` became the single authority on the row's width** — card slots are cut
        out of it, the AP bar spans it, the caption box matches it. A card in flight keeps its
        slot so the row does not slide sideways when one is lifted.
      - **This vacated the 15–39% column entirely** and it is still empty, deliberately. The
        rest of the layout moved with it: Resolution to 12–46%, the caption into a box of its
        own at 48%, the AP budget under the hand rather than over it, buttons to 95%.
- [x] **The character block, replacing the fighter's sprite and health bar.** Landed
      2026-08-04. Life as a red fraction, discards left this round, and vitae. A bar says
      roughly how hurt you are; a duel decided in whole points of damage wants the number.
      - **Discards are capped at 4 per round and refill with the hand.** One press costs one
        however many cards were selected, which is what makes the size of the selection a
        decision rather than a formality. It also closes the hole recorded under the Discard
        button entry above — discarding is no longer free and unlimited.
      - **Vitae reads a fixed 5 and has no rule behind it.** Drawn anyway so the block has its
        real shape now rather than being retrofitted into a box already sized without it. It
        moves to `Session` state when that exists.
      - The enemy keeps its sprite and health bar. Only the player converted.
      - [?] The block is the only thing in the vacated left column, and the screen leans
            left-heavy above the hand. Unresolved along with what fills 15–39%.
- [x] **Elements, as colour only to begin with.** Landed 2026-08-04 — and **the "colour only"
      half was superseded the next day.** Elements are mechanical: each applies a status, and a
      matching ring discounts its cards. The set changed too — earth joined as a primary, poison
      became secondary and leaves the deck. **See `MECHANICS.md` for the current design**; what
      follows is the record of the cosmetic first pass, which is still accurate about what the
      code does today.
      - What the successor makes due: the type must cross into `internal/combat`, exactly as the
        note further down predicted. That prediction is the useful part of this entry now.
      carries an element, and the element is what colours it: **fire orange, ice medium
      blue, lightning yellow, poison dark green, everything else white.** Lightning and
      poison were added to the original fire/ice pair the same day, before either had been
      seen on screen — four colours turn over faster in a five-card hand than two, and
      looking at them is the whole point of this pass.
      - **Matched pairs, one Strike and one Guard per element.** 8 of the 20 cards are
        coloured, so a hand averages two. The pairing is deliberate: a hand that comes up two
        colours is offering a choice between them rather than an accident of the shuffle.
      - **White is the absence of an element, not a fifth colour.** A plain card makes no
        claim, which is what leaves the coloured ones free to catch the eye.
      - **This is what the reserved colour was reserved for.** The glyphs were given one
        hueless palette on 2026-08-03 specifically so that colour would still be free to mean
        something when elements arrived. It has arrived.
      - [?] **Yellow now means two things and green means two things.** Lightning is yellow
            and `enemySwatch` is yellow; poison is dark green and `playerSwatch` — the
            player's Resolution rows and the drop indicator — is green. The two never share a
            region of the screen today, so nothing is currently ambiguous, but "green is you,
            yellow is them" is written down as a screen-wide rule and this quietly breaks it.
            Either the sides stop being colour-coded or the elements avoid those two hues.
      - [?] **White-on-light text.** The card name and the glyph numerals are drawn near
            white, which is fine on a 45%-strength surface and thin on a selected one at 65%
            — worst on white and lightning. Wants either a per-element ink or a darker
            selected fill.
      - **Cosmetic this pass, mechanical soon** — stated explicitly, so this does not read
        later as a placeholder nobody came back to. Painting them first is deliberate: the
        point is to look at the screen and get a feel for it before the rule is designed.
      - **It still forces cards to become instances.** A card today *is* a
        `combat.ActionKind`, so two Strikes are the same value and are indistinguishable. An
        element makes them differ, which means the deck, hand and discard all become slices
        of a struct — `{Kind, Element}` — even while the element does nothing at all.
        - Keep that struct **on the scene**, not in `internal/combat`, for as long as the
          element is paint. The moment it changes damage or resolution order it has to cross
          over, and then `ResolveRound` and `ResolutionOrder` both grow it and every test in
          the package changes shape. That is the cost to weigh when the mechanic is designed;
          it is cheap now and gets less so with everything built on top of the current
          signatures.
      - **`palettePane.color` green went away with this**, as planned. It was vestigial —
        the colour of a pane deleted on 2026-08-02, still filling cards that no longer sat
        inside anything. What survives of it is `playerSwatch`, which marks the player's side
        in the Resolution pane and nothing else.
      - [?] Whether the *glyph* takes the element palette too, or only the card surface. A
            full five-value orange palette is not the flat tint the glyph rules forbid, so it
            is available — but colouring both may be saying it twice.
- [ ] **AP cost as dots on each action card — superseded, revisit.** The plan was dots
      rather than a numeral, lining up against a per-point segmented AP bar so a 4-cost
      card visibly is most of a 6-point budget. The glyph row landed first and puts a
      numeral on a runner instead.
      - Dots and a glyph are not obviously compatible — three pips inside a 32px glyph is
        cramped, and a glyph plus dots beside it is saying it twice again.
      - The segmented-bar half of the idea survives regardless and is still worth having.
- [x] **Stop allocating in `Draw`** *(2026-08-06)*. `DrawButton` made a new `ebiten.Image`
      per button per frame and `DrawHealthBar` made two per bar per frame — roughly 300 GPU
      textures a second to draw pictures that change on a click or a hit. *(analysis §4)*
      - **Buttons repaint on change, not per frame.** `Button.Image` was already allocated at
        the right size and used only for hit-test bounds; it is now what gets painted.
        `Painted`/`PaintedState`/`PaintedText`/`PaintedColor` record what the cached face
        holds, and a button's face is a function of exactly those plus its size, so comparing
        them is a complete staleness test.
      - `needsPaint` also compares `Image.Bounds()` against `Width`/`Height`. Not paranoia:
        the image is allocated once in `NewButton`, so changing the size afterwards would
        otherwise stretch or clip the cached face with no clue why.
      - **The health bar composites through two package-level scratch images** instead of
        allocating a pair each frame. Safe because Ebitengine draws from one goroutine, both
        images are fully overwritten at the top of every call, and each call finishes
        compositing into the screen before it returns — so two bars in one frame take turns.
      - Allocated lazily rather than at package init: `ebiten.NewImage` before the game loop
        is running is a rule worth not testing.
      - Verified by comparing captured frames before and after — pixel-identical.

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
      - **Superseded on 2026-08-06 by phases.** Alternation, initiative and the
        until-you-act guard rule are all gone; see the entry below.
- [x] **Phase resolution, categories, and three new concepts** *(2026-08-06)*. The round is
      now **a whole turn each**: everything side A queued resolves in category order —
      prepares, then attacks, then defenses — and only then does side B begin. `ResolutionOrder`
      is still the single authority and both consumers followed for free, which is exactly
      what that split was for. One function body plus its tests, as predicted.
      - **`combat.Category`** is the new organising axis: `CategoryPrepare` /
        `CategoryAttack` / `CategoryDefend`, a property of the action rather than an
        independent choice. `Categories()` is both the phase order and the iteration order.
      - **Costs are now** Gather 1, Guard 3, Quick 1, Strike 2, Heavy 4, Dodge 2,
        Riposte 3. `ActionKind` is declared in category order so the deck overlay's sort
        groups the piles the way a turn resolves them.
      - **Guard moved from defend to prepare** and went 2 → 3 AP: it halves *all* incoming
        damage for the opposing turn rather than blocking one blow. Parry was dropped
        before it was built.
      - **Gather** costs 1 and banks +2 AP for the *next* round. It stacks within a round
        (two Gathers are +4) and is replaced rather than added to at the boundary, so
        preparing every round is a flat +2 and cannot compound. Stacking is deliberate:
        it is what puts a five-attack round in reach without a ring discount, which
        `MECHANICS.md` previously called unreachable. Accepted as a good trade.
      - **Dodge** (2 AP) negates the first incoming attack. **Riposte** (3 AP) negates one
        and hits back for half a Strike. Ripostes are spent before Dodges — both negate
        completely, so spending the one that hits back first is free, and its counter can
        kill the attacker mid-turn.
      - **Defenses expire at the start of their owner's next turn, not at the round
        boundary.** This is the rule the whole change turns on: side B acts last, so a
        defense cleared at the boundary would protect B from nothing it ever faces. The
        expiry point is a fact about *turns* and lives in `ResolveRound`, not in
        `ResolutionOrder`. An idle duelist now loses its guard — the deliberate quirk
        `TestGuardHoldsWhileItsOwnerDoesNothing` used to pin is gone with alternation.
      - **Two new event kinds**, `KindGathered` and `KindNegated`. Riposte's counter is a
        plain `KindDamage` from the defender's side, so the health bars need no new case.
      - The screen's `currentAction` became **`currentSlot`**, counting position in the
        resolution order rather than a side plus a queue index. `Slot.Index` is where a
        card sits in the player's queue and that is no longer where it lands in the round.
- [ ] **Revisit whether an initiative system makes sense for resolution.** Removed wholesale
      on 2026-08-06 — the `Initiative()` method, the four constants, the tie-break in
      `ResolutionOrder`, the clock glyph and the `i3` in the concealed enemy label.
      - **Why it went:** with one contiguous turn per side there is no exchange for a
        faster action to lead. It was a number on every card reporting a distinction the
        resolver had stopped making, and it was occupying the third glyph slot.
      - **What it would need to come back:** somewhere for it to bite. The obvious
        candidates are ordering *within* a phase (currently queue order, which is free and
        legible), or a partial interleave where one designated fast action jumps a phase.
        Both reintroduce the legibility problem phases were adopted to solve, so the bar is
        that it buys something combos and category ordering do not.
      - The card has room: the glyph column now holds two badges in a space sized for
        three, and `deckCardStyle` shrank 50px on the strength of that. Bringing initiative
        back costs the deck overlay its third grid row.
- [ ] **Defenses target a specific incoming attack** *(design captured 2026-08-06)*. The
      resolution order is right now; what is missing is that a defense is currently a pool
      rather than a choice. **Guard stays untargeted** — it covers you entirely, which is
      exactly why it is a prepare. **Dodge and Riposte should each name the enemy attack they
      answer**: dodge the first and riposte the second, or riposte the first and dodge the
      second, and those are different rounds.
      - **This is what replaces initiative as the ordering lever**, and it is a better one.
        Initiative decided *when* your action happened; this decides *what it happens to*,
        which is a decision the player makes rather than a number they read off a card.
      - Today `Ripostes` and `Dodges` are plain counters on `Duelist` and negation is spent
        Riposte-first by a fixed rule. That rule exists only because nothing can express a
        preference. Targeting replaces it — the fixed order becomes the fallback for an
        untargeted defense, if untargeted defenses survive at all.
      - **The hard part is the UI, not the rules.** Engine-side this is a target field on
        the queued action and a lookup at negation time. Screen-side the player has to point
        at an enemy attack that is **concealed while planning** — you know the enemy has two
        actions and their categories, not what they are. Targeting "their second attack" is
        therefore targeting a slot, not a card, which may actually be the cleaner design.
      - Input vocabulary allows exactly click and drag. Dragging a Dodge onto an enemy row
        in the Resolution pane is the obvious gesture and would give that pane its first
        interactive job. No right click, no keyboard.
      - **Open:** what happens when the targeted attack does not arrive — the enemy queued
        fewer actions than you predicted, or died first. Wasted, or does it fall back to the
        next attack? Wasted is more honest and makes prediction matter; falling back makes
        targeting a free upgrade over the current pool.
      - Blocked on nothing, but worth doing *after* enemy variety: targeting an opponent
        that always throws the same two attacks is a decision with one right answer.
- [~] **Enemies that fight in genuinely different shapes.** *(styles landed 2026-08-06; decks
      still open)* `PlanGreedy` is gone, replaced by `combat.PlanStyle` and four planners
      selected by a `PlanStyle` string on the data record — so enemy behaviour is data and the
      roster is tunable without touching Go.
      - **brute** biggest attack affordable · **swarm** as many attacks as the round allows ·
        **warden** Guard then attacks · **tactician** banks with Gather then unloads.
      - Roster is Monster1, Swarm1, Warden1, Tactician1 in `combatants.json`. The combat screen
        walks them in order, advancing on a win — **scaffolding for the tower, to be deleted
        rather than extended.** It also fixed a genuine dead end: winning used to leave every
        button dark with no way to play on short of restarting the process.
      - **The immunity problem is fixed.** Dodging beat every fight before; it now loses to
        swarm, which is the point — "negates one attack" is priced against how many attacks
        arrive. Each enemy has a different right answer: brute wants dodging, swarm wants
        racing, tactician wants reading the tell, warden wants overwhelming.
      - The tactician's tell is the concealment scheme paying off by accident: a concealed row
        still shows its category, so `??? (prepare) ??? (prepare)` warns of a spike without saying
        what it is.
      - **Still open, and the reason this is `[~]`:** `MECHANICS.md` decides enemies get a
        *deck* and that an affix transforms it, which subjects them to the same "what did I
        draw" pressure the player faces. That needs its own shuffle stream. A deck-driven
        planner arrives as one more style beside these rather than replacing the idea.
      - `[?]` Four sprites are placeholder crops off one Tyrian sheet and three of them are
        near-identical. Fine for now, wrong for a release.
- [x] **`tools/balance`** *(2026-08-06)*. Prints what every enemy does to the fighter across
      three postures, playing whole duels through the real `ResolveRound`.
      - **Written because an unwinnable enemy shipped and nobody could see it.** Warden1 halves
        everything it takes, and at 120 life that was a 24-round grind against a fighter who
        dies in 10. Retuned to 70 life and Str 9 on the strength of the table.
      - The first version divided life by the last round's damage and lied twice — it read a
        killing round as the steady state, and flattened the tactician's rhythm into whichever
        half it sampled. Playing the duel out costs nothing and cannot be wrong about a rule.
      - **Run it after touching any cost, stat line or planner.** What it cannot model is the
        draw, so read it as the best case for each posture.
- [x] **`internal/idle`, the unattended-run timer** *(2026-08-06)*. `go run -tags idleexit .`
      closes the game after two minutes with nobody at the controls; `ASCEND_DUEL_IDLE_SECONDS`
      overrides it. Bootstrapping so the game can be launched to check something and left.
      - Build tag and the same `_on`/`_off` two-file shape as `internal/trace`, for the same
        reason: a game that quits on a player who steps away is a bug, so not a byte of it may
        reach a shipped binary.
      - **Everything is gated on window focus, cursor movement included** — an unattended run
        sits in the background, and a cursor crossing the desktop over an unfocused window
        would otherwise read as someone playing. The one case it exists for would be the one
        case it never fired in.
- [ ] **Enemies get decks, and affixes transform them.** The half of the enemy model that
      styles did not deliver. `MECHANICS.md` decides it: an enemy has a deck smaller than the
      player's, and an affix *transforms* it rather than adding to it — a brute has basic
      attacks, a fire brute on a fire floor has all fire attacks.
      - A planner would draw a hand and plan from *that*, which subjects the opponent to the
        same "what did I draw" pressure the player faces. Styles plan from a budget instead,
        so today's enemies always play their best round.
      - **Needs a shuffle stream of its own** — see the four-stream rule in `CLAUDE.md`. This
        is the first thing since the player's deck to need randomness at all.
      - `AvailableAffixes` is in `combatants.json` and read by nothing.
      - `[?]` Earth has no affix; whether it can be a floor theme is still open.
- [ ] **Guard versus Dodge, with data this time.** `tools/balance` says the *guarding* posture
      (Guard + Strike, 5 of 6 AP) loses to three of the four enemies and only narrowly beats
      the warden. Guard costing 3 eats most of a round's budget and leaves one Strike behind
      it, so a broad halving does not pay for itself.
      - This is the question deferred on 2026-08-06 when Guard was set at 3, now with numbers
        rather than a single opponent behind it. Candidates: Guard to 2, or the budget grows,
        or halving becomes something stronger.
      - Do not tune it against these four alone — the whole lesson of the enemy work is that a
        defensive card's worth is set by the shapes it faces, and four is still a small sample.
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
      - **Flow, decided:** build a plan → DUEL! → watch playback → plan again. The *queue*
        empties every round, so no plan is ever repeated by default.
      - **The hand no longer empties with it** *(2026-08-06)*. Only the cards actually played
        go to the discard; whatever is left sits where the player left it and the draw tops
        the hand back up. See the deck entry below for why the original rule was wrong.
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
- [x] **The combo framework, and stagger as its first combo** *(2026-08-07)*. Combos are
      where the game is meant to be — throwing what you drew at the enemy works, choosing a
      shape and building toward it should work better — so this landed as a **framework built
      for dozens of entries**, not as one mechanic. `internal/combat/combo.go`, 22 tests.
      **See `MECHANICS.md` for the design**; what follows is only what it cost to build.
      - **One pattern: a run of cards in, one of three rewards out** — damage multiplier,
        banked AP, or an alteration to the opponent. Adding a combo is one table entry;
        adding a new *reward kind* is a field on `Effect` and one place applying it. That
        second cost is charged deliberately, so the vocabulary stays small enough to learn.
      - **The flurry/onslaught family is generated, one pair per attack card**: Strike Flurry,
        Strike Onslaught, Heavy Flurry, Heavy Onslaught, Quick Flurry, Quick Onslaught. A new
        attack card gets its own pair by existing, which is the point — deckbuilding will add
        many, and a hand-written table is one that goes stale.
      - **Per card rather than per category**, changed the same day after the category version
        shipped first. `AnyOf(CategoryAttack)` meant any three attacks combo'd, so a Quick was
        worth a Heavy and the reward went to whatever you drew. Three Strikes is a deck you
        assembled. It also leaves room for a Heavy Flurry to hit harder than a Quick one later,
        which a category-wide combo could never express.
      - **Heavy Onslaught is 20 AP and near enough impossible.** Kept deliberately, so
        engine-building has something absurd to aim at.
      - Combo IDs derive from the card, not from a list position, so inserting a card cannot
        renumber combos a profile has already unlocked.
      - **"Unopposed" is struck from the name and the rule.** It was written under alternation.
        Under phases every attack you queue is consecutive by construction, so nothing can
        interrupt a streak — which also **kills both `[?]` questions this item carried**
        (does a Guard reset it; does a zero-damage hit count). Neither can ever fire.
      - **The two open questions were closed, and here is which way.** The lost action comes
        off the *front* of the victim's next turn, which under phases is their prepare phase — so a
        stagger costs a Gather before it costs an attack. Its action points are **not**
        refunded, making stagger tempo *and* economy.
      - **`Duelist.Staggered` persists across the round boundary**, and that is forced rather
        than chosen: side B acts last, so a combo B forms has no turn left to bite in. Holding
        it on the duelist is what keeps this one rule instead of two.
      - **Combos match what survives a stagger, not the queue**, or a staggered duelist could
        combo back with a turn it never took.
      - `resolveRound` takes the table as a parameter so tests can drive a synthetic combo
        through the whole engine — the multiplier and banked-AP paths would otherwise have
        shipped without ever having been run, since both live combos use stagger.
      - **Screen cost was one line and one bug.** `currentSlot` counted `KindAction`, and a
        staggered slot produces none, so the Resolution highlight would have run a row short
        for the rest of the round. It counts `KindStaggered` too now: one beat per slot,
        taken or lost.
- [x] **The Resolution pane, and the caption stops narrating** *(2026-08-07)*. The old
      Resolution pane became **Action Flow** (the plan) and a new **Resolution** took its slot
      (the record, accumulating as the round plays). See the `combat-screen` skill.
      - **This is how combos became visible at all.** Before it, the whole account of a round
        existed only as a quarter-second caption flash — a combo forming was unreadable, and a
        Guard halving a Heavy went past before it could be noticed.
      - **It retired the `[?]` about drawing a combo across non-adjacent rows** by splitting
        the pane rather than solving it. Worth keeping as a pattern: the pane was being asked
        two questions at once and the answer was a second pane, not a cleverer drawing.
      - **Action Flow is currently built and not drawn**, and Resolution has both columns. One
        line to restore. What that costs while it is off: the enemy's queued shape during
        planning, which is the tell.
      - Lines are sentences with the verb coloured, bold and underlined in the text — red attack,
        blue defend, no hue for prepare. It was a filled chip until 2026-08-08; the block drew
        the eye instead of the word, the same reason the highlight bar went. Prose lives in
        `screens`, never in `combat`.
- [x] **Rename: the `setup` category is `prepare`, and the `Prepare` card is `Gather`**
      *(2026-08-07)*. One word meaning both a category and a card in it was the confusion; now
      `Gather` is a card of category `prepare`, beside `Guard`.
- [x] **Named deck seeds and `tools/seeds`** *(2026-08-07)*. The shuffle is deterministic, so
      **a seed is an opening hand**. `internal/screens/seeds.go` catalogues them by name and
      `deckSeedName` picks which one a launch deals.
      - **Written because the demo could not click a combo.** The old fixed deal held no
        three-of-a-kind, so seeing a Flurry meant writing the queue directly — which tested the
        pane but not the path to it. `strike-flurry` guarantees three Strikes and the demo now
        selects them through `toggle` like a player.
      - **The re-check is the half that matters.** A seed is a fact about one particular deck;
        change `startingDeck` or `handSize` and every number silently deals something else. The
        tool re-deals each catalogued seed and reports the ones that no longer match. It
        rejected two guessed numbers on its first run.
      - `[?]` Catalogued seeds are compile-time only. The planned run-seed text field is the
        real version of this, and when `Session` lands the catalogue should feed *it* rather
        than a package var.
- [x] **`-tags demoplay`, the scripted demo** *(2026-08-07)*. The game plays a scripted round
      or two with nobody at the controls and writes `demo/*.png`, then closes.
      - **Written because the combat screen is the one thing nothing else can check.**
        `go test` needs no window and `tools/balance` judges rules, not pixels; a combo line, a
        verb chip and a highlight on the right row all have to be *seen*. It is
        `tools/glyphsheet`'s idea applied to a live screen.
      - Two files, `_on`/`_off`, behind a build tag, deletable in one commit — the same shape
        and the same reasoning as `internal/trace` and `internal/idle`. A game that plays
        itself must not ship.
      - It presses the real buttons (`toggle`, `startRound`). The one place it reaches past
        input is the queue: the fixed `deckSeed` deal holds no three-of-a-kind, so a combo
        could never be made to fire by clicking, and a combo firing is what needs looking at.
      - **It found two things immediately**: `VITAE` rendering as `VITRE` in caps, and the row
        highlight clipping at the Resolution pane's tighter pitch.
      - `[?]` It scripts one fixed pair of rounds. A flag for which plan, or a second script
        for the losing case, is the obvious next step and is not built.
- [ ] **Swarm1 is unbeatable.** *(Caused by combos, 2026-08-07. Left standing deliberately, to
      watch how it balances out; this is the entry that says it is known and not forgotten.)*
      `tools/balance` reports Swarm1 beating **all three postures**, which is the tool's own
      "wall" condition. Before combos it lost to `all-out` in two rounds.
      - **Why:** it queues `Strike + Quick + Quick + Quick + Quick`, and four Quicks in a row is
        a Quick Flurry every single round. The player is staggered out of an action each round
        and never gets ahead.
      - **A combo counting cards is priced by whoever has the cheapest cards.** Every costing in
        `MECHANICS.md` reasons from the player's budget — three Strikes is 6 AP — while Swarm1
        buys four Quicks for 4. Nobody costed the combo against a cheap attacker.
      - Per-card matching already softened this: under the category-wide version it formed a
        full Onslaught and the fighter dealt **zero** damage from round two. Worth knowing that
        the same change also un-broke Tactician1, which the category version had made
        unbeatable too.
      - Candidates, none chosen: price a run by AP rather than by card; give stagger a
        cooldown; retune the swarm; or **give enemies their own cards and their own combos**,
        which is the option `MECHANICS.md` now flags as genuinely open — it is not settled that
        the opponent draws on the player's combo table at all.
      - **The deeper cause is that enemies plan from a budget and always play their optimal
        round** — see the enemy-decks entry. A deterministic optimiser forms the same combo
        every round forever, which is the condition a combo system punishes hardest. Enemy
        decks may fix this without touching a combo rule.
      - **All four candidates were implemented and measured on 2026-08-08, and reverted. None is
        chosen — this is evidence, not a decision.** Each was built in turn, `go run ./tools/balance`
        recorded, then rolled back before the next. Baseline for comparison: Swarm1 loses no
        posture (the wall); Monster1 1/3; Tactician1 2/3; Warden1 3/3.

        | candidate | Swarm1 after | elsewhere | tests |
        |---|---|---|---|
        | price a run by AP | fixed, 1/3 | **Tactician1 2/3 → 1/3** | 3 edits needed |
        | stagger cooldown | fixed, 1/3 | **nothing changes** | pass, unmodified |
        | retune the swarm | fixed at ≤50 life | cause untouched | pass, unmodified |
        | enemies keep their own combos | fixed, 1/3 | nothing changes | **2 fail** |

        - **Pricing by AP makes expensive cards *easier* to combo, which is backwards.** A Heavy
          Flurry becomes `ceil(6/4)` = **2 cards** instead of 3, so Tactician1's `Heavy+Heavy+Quick`
          gains a combo it could never reach and beats `all-out`. It also silently falsifies the
          `heavy-flurry` seed's stated rationale while `tools/seeds` keeps passing, because that
          tool pins the *deal* and not the combo.
        - **A stagger cooldown fixes Swarm1 and leaves every other enemy identical line for line**,
          with no test churn. Cost: one more rule to learn, one more piece of hidden duelist state
          the screen does not show, and the win is narrow at 12/60 life.
        - **Retuning cannot fix the cause, and the sweep proves it.** Every budget from 5 to 10 AP
          is still a wall: `planSwarm` fills with one card then upgrades along the plan, so it
          always emits a homogeneous run of at least three — `Q×5`, `S+Q×4`, `S×2+Q×3`, `S×3+Q×2`,
          `S×4+Q`, `S×5`. Only 4 AP breaks it, by being too weak to win, and that still forms a
          Quick Flurry. Dropping Constitution to 10 (50 life) also breaks it while leaving the
          every-round stagger completely intact. **This is the "deeper cause" above, demonstrated.**
        - **Enemies keeping their own combos costs two pinned properties**, not a tuning number:
          `TestSideBsFlurryLandsInTheFollowingRound` and `TestStaggerIsSymmetric` both fail and
          can only be *deleted*, because what they pin is the symmetry this removes by design. It
          cannot be judged on a balance table — the table likes it.
        - **Stagger cooldown and a retune compose**, if the 12/60 win is too tight to accept alone.
- [ ] **Warden1 is free, which is the wall condition's mirror image.** *(Noticed 2026-08-08.)*
      `tools/balance` has it losing to **all three** postures — its own stated "free" condition —
      in the baseline and under all four Swarm1 candidates above, so nothing proposed there
      touches it. Recorded because the tool was written to catch exactly this and the opposite
      case got an entry while this one did not. Warden1 appears above only as the enemy that was
      once *unwinnable*; this is the other end of the same tuning problem.
- [ ] **Procedurally generated enemies.** *(Cut 2026-08-07.)* The roster is four hand-written
      records in `combatants.json` and the combat screen walks them in order. That is
      scaffolding. An enemy should be **generated** from the floor, so the tower can be endless
      and a seed can reproduce it.
      - **Assembled from parts, not rolled from scratch.** The pieces that already exist or are
        already decided: a **stat line** scaled by floor depth; a **deck** (see enemy decks);
        an **affix** that transforms that deck rather than adding to it; a **sprite**; and a
        **personality** — what `PlanStyle` is the first crude version of.
      - **Personality is the part that is furthest out**, and deliberately so. Today it is one
        of four planner functions chosen by a string. What it wants to become is a set of
        leanings — how readily it defends, whether it banks, whether it plays toward a combo —
        that a generator can dial rather than pick from a list of four.
      - **Needs its own randomness stream**, per the four-stream rule in `CLAUDE.md`. Enemy
        selection is already named as one of the four; this is what will draw on it.
      - **Blocked on nothing, but pointless before enemy decks**, which decide the shape of the
        thing being generated. Generating stat lines alone would just be `combatants.json` with
        extra steps.
      - The four current records become **seeds for the generator or test fixtures**, not the
        roster. `AvailableAffixes` in the JSON is read by nothing and is waiting for this.
- [?] **Ordering model — four candidates, one implemented, and a fourth now chosen to try.**
      Contested slots is what ships today. `ResolutionOrder` is a single pure function, so
      swapping between any of these is one function body plus its tests.
      - **Phases — chosen 2026-08-05 as an experiment and as the direction.** Preparations,
        then attacks, then defenses, then the enemy. Defenses front-load because the enemy
        goes last. Chosen on legibility grounds: interleaving may not be graspable by players.
        **See `MECHANICS.md` for the full entry and its costs** — cross-phase reordering stops
        meaning anything, Guard persistence dissolves, and stagger's rarity has to come from
        elsewhere. The three below are the alternatives it was chosen over, kept because the
        experiment may not survive contact.
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
        - ~~The whole hand discards at the end of a round and five are drawn fresh.~~
          **Reversed 2026-08-06: only what was played leaves.** The original rule was
          justified by "keeping cards back would let a plan be prepared once and repeated",
          which conflated the *queue* with the *hand*. The queue does still empty every
          round, so no plan repeats by default — but taking away cards the player had
          deliberately held punished holding them, and the only way to keep a card was to
          spend it. Hand retention is the deckbuilder lever the entry already suspected it
          was; it turned out not to fight the no-repeat decision at all.
        - It also gave Discard a job it did not have. A card you never want now stays in
          hand until you throw it out, where before the round boundary cleared it for free.
        - `spendSelected` is the single movement, shared by Discard and by the end of a
          round. Both are "the selected cards leave and the hand refills", so they are one
          function rather than two that have to agree.
        - The spend happens when *playback* finishes, not when the round resolves.
          `endRoundHand` rebuilds `fighterActions`, and the Resolution pane draws
          `fighterActions` to narrate the round, so spending early empties the pane
          mid-round. The ordering in `advancePlayback` is load-bearing.
        - ~~`Quick` is defined in the rules but absent from the deck.~~ **Resolved
          2026-08-08:** `Quick` was renamed **Jab** and given its five cards when the
          concept grid was filled. The split it illustrated is still real — which of the
          rules' actions appear in a deck is a deck-building question — but `Sift` is the
          card that carries it now, being the one concept whose effect is not a rule at all.
        - The 10/6/4 composition and the hand size of five are first guesses, unplayed.
          **Both superseded:** the deck is 12 concepts x 5 elements = 60 built from
          `data/cards.json`, and the hand is 8. The hand was sized against a 30-card deck
          and deliberately left alone when the deck doubled — see the note on `handSize`.
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
- [ ] **Price Mirror. `tools/balance` says it is broken.** Added 2026-08-08 as the 4-AP
      defend: negates every attack in the opponent's next turn and reflects each blow's
      damage back. The `mirroring` posture **beats all four shipped enemies and finishes
      every one of them on 60/60 life** — the "strong against everything" condition that
      means a card is mispriced rather than good.
      - It is also the *only* posture that beats `Swarm1`, an enemy recorded as unbeatable
        by all three original postures. So it papers over a known balance hole by being
        overpowered instead of by being right, and fixing Mirror re-opens that hole.
      - The lever is one constant: `mirrorReflectNum/mirrorReflectDen` in
        [combat.go](internal/combat/combat.go), left at 1/1 because full reflection is the
        card as designed. Two candidates, in order: **halve the reflection** (keeps its
        character — it still scales with what the opponent committed), or **cap what it
        stops** ("negates and reflects the first two attacks" is a different card but a
        priceable one).
      - [?] **Does giving enemies more attacks fix it instead?** Worth investigating, and
        worth knowing the arithmetic points the other way first: **enemies already have the
        same 5-action cap.** `MaxActions()` returns `baseMaxActions` for every duelist, so
        what limits an enemy is its AP budget, not its action count — `Swarm1` at 6 AP
        already throws five (`Strike+Jab+Jab+Jab+Jab`), while `Monster1` at 5 AP throws two.
        And Mirror is *most* devastating against `Swarm1`: it wins there in 3 rounds, the
        fastest of the four, because a reflection scales with how many blows arrive. More
        enemy attacks therefore makes Mirror **stronger**, not weaker.
        - The version of the intuition that might survive: raising enemy *AP* would make
          Guard and Brace better too, since both scale with attack count in the same
          direction, which could narrow Mirror's lead over them without touching Mirror.
          That is a real experiment — add AP to the roster in `combatants.json` and re-run
          `tools/balance`. Cheap, because it is a data change.
        - What it cannot fix: Mirror is currently *unconditionally* better than Dodge and
          Guard in every matchup. A change that makes all three better leaves the ordering
          intact.
- [ ] **Rethink the deck overlay — it now shows less than half the deck.** The grid is
      8 columns x 3 rows = 24 slots; a 60-card deck puts up to 52 cards outside an 8-card
      hand, so `drawPileGrid` draws 24 and prints "+28 more not shown" on every look.
      - **The overflow line firing is the design working.** It was written when the deck was
        30 and this could not happen, precisely so that a grown deck would produce a visible
        shortfall rather than a panel that quietly lied about what you own. What it is not
        is an answer.
      - **It cannot be fixed with a bigger grid.** `GlyphSize` is 64, `CardGlyphScale` is
        already 1, integer scales only, and the two-glyph column is the floor on a card's
        height. `deckCardStyle` is 138x186 and most of what is left to cut is padding.
      - Options, none chosen:
        - **Group identical cards under a count.** Twelve concepts x five elements is 60
          cards but only 60 *distinct* ones at Copies 1 — so this only helps once duplicates
          exist. It would help a lot then.
        - **One tile per concept, elements as five pips.** Twelve tiles fit trivially, and
          the element breakdown is what a count could never say — the original reason the
          overlay draws cards rather than a table. This is the strongest lead.
        - **Paging.** Honest and cheap, but it stops being "the whole deck is one look",
          which was the stated point of the panel.
        - **A smaller non-card representation** — swatch grid, one row per concept. Loses the
          glyphs, which may be fine here: the overlay is inventory, not a thing you play from.
      - Whatever wins, keep the property that made the current panel good: **a card does not
        move when it is discarded, it only dims.** Watching the deck drain in place is why
        both piles share one grid.

### Loose ends from the 2026-08-08 concept work

Five decisions and gaps that came out of filling the concept grid and were not captured
anywhere at the time. Listed because a rule that exists only in the code is a rule the next
design conversation will contradict without noticing.

- [?] **What a Feint does when a second negation sits behind the one it strips.** Reading the
      path in `resolveAttack`: the strip decrements one, then falls through to the ordinary
      Riposte and Dodge checks, so **two Dodges beat a Feint and one does not.** That is a
      coherent rule and nobody chose it — it is a consequence of where the strip was placed.
      - Decide it, then pin it with a test. The alternative reading is that a Feint's damage
        bypasses the negation layer entirely once it has stripped something, which is stronger
        and makes stacked defences worthless against it.
      - Not currently tested either way, which is the actual defect here.
- [?] **Does Sift stack?** It does today: `siftsResolved() * siftExtraDiscards`, so two Sifts
      throw four extra cards away. Gather's within-round stacking is a documented deliberate
      choice; Sift's is just what the loop does. Two Sifts is 4 AP of a 6 AP round to churn
      six cards, which may be fine — but it should be a decision.
- [ ] **Price Sift, and build something that can measure it.** 2 AP replacing seven of eight
      cards largely erases the consistency cost of a 60-card deck, which is a lot for the
      cheapest prepare after Gather.
      - **`tools/balance` structurally cannot see it.** Sift's effect is on the hand, the hand
        is on the scene, and the tool has no deck — so this is the one concept in the game
        with no instrument pointed at it. Either the tool grows a deck (which means moving
        draw into `combat`, see the deckbuilder entry) or something else measures draw
        variance. The second is probably a small harness over `screens.OpeningHand`, which
        already deals real hands headlessly.
- [ ] **Move four interaction rules from code comments into `MECHANICS.md`.** All four are
      decided, implemented and tested; all four are recorded in the wrong file, and MECHANICS
      is supposed to be what the game *is*. A designer reading it today would not learn any
      of them:
      - Brace and Guard **both** apply, quartering a blow.
      - Mirror is checked **before** Dodge and Riposte, so it never lets a cheaper negation be
        spent on a blow it was going to stop for free.
      - Mirror reflects **post-combo-multiplier and pre-guard** — it returns what the attacker
        committed, which is what makes it a read rather than a damage reduction.
      - Feint's strip is **unconditional** and fires even behind a Mirror, deliberately, so the
        card has no hidden interaction with something the player cannot see.
- [ ] **The demo has never shown a new card resolving.** `demoSeedName` is `strike-flurry` and
      `demoClickRun` is `Strike`, so the scripted round plays Strikes and a Riposte. The
      narration written for Feint, Mirror, Brace and Sift — the "reflects N" line, the "strips
      their riposte" line, "braced" — has never appeared on screen, and the demo is the only
      thing that looks at the screen without a person sitting there.
      - Point it at a hand that plays a Mirror into a multi-attack enemy turn. `all-categories`
        (seed 6) deals eight distinct concepts including Mirror and Brace; `Swarm1` throws five
        attacks, which is the turn that makes a reflection worth watching.
      - `demoClickRun` has to agree with the seed or the click phase silently selects fewer
        cards and nothing forms. That pairing is what `tools/seeds` exists to keep honest.

### Cards and piles — presentation

- [ ] **Cards must explain themselves. Long press explains the mechanic.** Today a card shows
      its name, category word, damage and cost, and everything else has to be inferred — that
      Riposte hits back, that Guard covers a whole turn rather than one blow, that Mirror
      reflects. Six of the twelve concepts cannot be understood from the card at all.
      - **This is long press, not hover** — confirmed 2026-08-08 against the recorded
        vocabulary. `MECHANICS.md` §Long press already assigns "explains" to long press and
        records that hover was considered and rejected; CLAUDE.md's input vocabulary has no
        hover in it. The split to preserve if hover ever does return is **hover un-occludes,
        long press explains**, so this task is the second half of that sentence.
      - The gesture already has a designed shape: a press is a three-way decision — past
        `dragThreshold` is a drag, held past a tick count without moving is a long press,
        released before either is a click that toggles selection. **The distance and time
        thresholds must not fight each other.**
      - Needs the effect text to live somewhere. `actionPhrases` in
        [combat_panes.go](internal/screens/combat_panes.go) is prose for a *sentence about a
        round*, not a rules description, so this is a second table — and the rule it describes
        lives in `internal/combat`, which must not grow UI strings. Same shape as
        `actionPhrases`: the screen describes, the rules name.
- [ ] **Draw and discard animation, with both piles on screen.** Cards should visibly travel:
      out of the draw pile into the hand, and out of the hand into the discard. Physical
      movement, not a count changing.
      - **Discard pile on the left, showing its cards.** Draw pile further right on the same
        horizontal axis, drawn at **half a button — 69x25**, buttons being 138x50.
      - **The draw pile cannot show faces.** It is shuffled and ordered, so drawing its
        contents would hand the player their next cards and make the shuffle pointless — the
        same reason `drawPileGrid` sorts rather than showing pile order. A stack
        representation, not cards. The discard is public information and may show its cards.
      - Layout is the hard part and is not solved: the bottom band already runs 566→937 and
        holds eight overlapping 180x264 cards plus three 138x50 buttons, and rings are recorded
        as wanting a row that *already* does not fit vertically. This competes with that.
      - **It must not become a per-event dwell table.** Playback pacing is deliberately one
        constant, `duelTicksPerEvent`, because a switch with a `default` arm silently gave every
        new event kind the shortest timing — see the history note in
        [combat.go](internal/screens/combat.go). Card movement is not an event and should be
        animated on its own clock rather than by adding cases to that one.
      - Presentation only: **it may never change an outcome.** Same constraint as trace, idle
        and the debug flags.
      - Deliberately under-specified beyond the pile placement above, per the owner: worth
        doing properly when it is picked up rather than designed in advance here.

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
- [ ] **Profile — persistent unlocks across runs.** Wanted 2026-08-04, for the standard
      roguelike shape where the tower is the run and the profile is what survives it.
      - **A different lifetime from `Session`**, the per-run state in the split above.
        `Session` holds the current floor and the rings collected this climb, and dies with
        the run. A profile spans every run and is the only thing that does.
      - **It is the first thing in the game that must be serialized as real state.** The save
        format decided below is a seed plus a choice log, which works precisely because a run
        is replayable from its inputs. Accumulated unlocks are not derivable from anything —
        they are the residue of runs already thrown away — so a profile needs an actual file
        format with a version stamp and does not get the replay trick for free.
      - **One profile, implicit, for now.** No slot picker and no naming, because naming
        needs a text field and the one text field in the game is spoken for by the seed.
      - What actually unlocks is undecided: cards for the starting deck, enemies in the pool,
        floors, whole alternate decks. Worth answering alongside the loot loop, since an
        unlock and a reward are the same object with different lifetimes.
- [ ] **Several profiles, and the second text screen.** Explicitly a later problem, split out
      from the entry above on 2026-08-04 so that "one profile for now" does not quietly
      become "one profile forever". Multiple profiles need naming, naming needs typing, and
      typing makes the one-text-field rule in `CLAUDE.md` into two.
      - That is a rule change rather than a feature. Revisit the hand-rolled-UI decision
        under "Open decisions" at the same time — its `[?]` trigger is precisely "the seed
        text field turns out to be painful", and a second field doubles the exposure.
      - Numbered slots picked from a list would dodge the text field entirely, at the cost of
        "Profile 2" meaning nothing to the player.
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
- [ ] **Sign the Windows binary so Defender stops saying "Unknown publisher."** Raised
      2026-08-08 after downloading the v0.1.1 `.exe`. Not urgent and **not cheap** — this
      entry exists mainly so the cost is written down before someone assumes it is a
      workflow tweak.
      - **Signing changes the name in the dialog, not the dialog.** SmartScreen gates on
        *reputation*, which a brand-new certificate does not have. A signed build shows
        "Justin Crosby" instead of "Unknown publisher" and still warns until enough clean
        downloads accrue. Only an EV certificate has historically carried instant
        reputation, and even that is reported to reset on renewal.
      - **A `.pfx` in GitHub secrets is not an option.** CA/Browser Forum rules since
        June 2023 require the private key on FIPS 140-2 Level 2 hardware or a cloud HSM,
        so CI signing means a hosted signing service, not a file.
      - **Azure Artifact Signing** (renamed from Trusted Signing) is the cheap path at
        ~$9.99/month and is built for CI. Eligibility is the catch: it was restricted to
        US/Canada organisations with three years of verifiable history, with individual
        signup coming and going through preview, and there are reports of the role
        assignment needing a paid Entra ID tier on top. **Check current eligibility before
        costing this** — the answer has moved repeatedly.
      - **It would put signing credentials in the release workflow**, which is the one job
        already holding `contents: write`. Weigh that against the repo's rule that the
        publish job is the last place to widen access.
      - `[?]` **This may be the wrong problem to spend money on.** The game is intended to
        sell through Steam, which delivers its own signed client and does not put the
        binary through this path. If the GitHub releases stay a developer-and-playtester
        convenience, "Unknown publisher" may be acceptable indefinitely. Decide whether
        the download page is ever a *player-facing* distribution channel first; that
        answer, not the price, is what settles this.

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
