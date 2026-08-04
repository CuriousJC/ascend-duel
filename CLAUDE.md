# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Ascending Duel — a roguelike where you duel your way up a tower, collecting rings, brands of power, and pets. Written in Go with [Ebitengine v2](https://ebitengine.org/) (`github.com/hajimehoshi/ebiten/v2`). Module path: `github.com/curiousjc/ascend-duel`.

`ideas.md` holds informal design notes (bosses, attributes, floors, rings) that are not yet implemented. `TODO.md` is the running work list and the record of decisions already made — read it before proposing design changes.

## Licensing and IP — read before adding dependencies or assets

The project is **source-available, not open source**, and is intended to be sold
(Steam) by its two copyright holders while the source stays public.

- **License: PolyForm Noncommercial 1.0.0** (`LICENSE`), relicensed from Apache 2.0 on
  2026-07-31. Anyone may read/build/modify/share noncommercially; selling is reserved
  to the copyright holders. Additional Permissions at the top of `LICENSE` explicitly
  allow monetized streaming and video of gameplay.
- **Copyright holders: Justin Crosby (CuriousJC) and KingSherman1820.** They have a
  written partnership agreement; Justin can speak for both on licensing and IP, so
  there is no need to ask whether Sherman agrees.
- **`CONTRIBUTING.md` carries a contributor grant.** Outside contributions must come
  with a commercial-use grant, or merging them would leave the game unsellable. Do not
  weaken or remove that document.
- **No GPL, ever.** Every dependency must be permissive (MIT / BSD / Apache-2.0) or it
  cannot go into a product licensed this way. Check the license before adding anything
  to `go.mod`, and flag it in the PR.
- **Assets need provable licenses.** The Tyrian placeholder art is a known release
  blocker — informal blog-post permission, not a license. Do not add assets with
  unclear provenance; "found it online" is not sufficient for a paid release.
- The pre-relicense Apache 2.0 grant on already-published commits is irrevocable and
  the owners have accepted that. Do not propose rewriting history over it.

## Commands

```powershell
go run .            # build and launch the game window
go build .          # produce ascend-duel.exe (gitignored)
go vet ./...
gofmt -l .          # list unformatted files

go run -tags debugtrace .   # with internal/trace live: event log + trace/frame.png
```

Vet and build **both** configurations when touching anything traced — the tag selects a
different file in `internal/trace`, so one can compile while the other does not:

```powershell
go vet ./...; go vet -tags debugtrace ./...
```

```powershell
go test ./...                                   # all tests
go test ./internal/combat -run TestName         # a single test
git commit -s                                   # sign-off, per CONTRIBUTING.md
```

Tests live in `internal/combat` — the only package that can be tested without a
window, by design. Keep it that way: rules go in `combat`, not in screens.

## Git workflow

- **Squash merge every PR.** One commit on `main` per PR, so the history reads as a
  list of milestones. The number of commits on the branch does not matter — write
  them freely and let the squash collapse them.
- Because of squashing, `git branch -d` will *always* refuse a merged feature branch:
  the squash creates a new commit, so the branch tip is never an ancestor of `main`.
  Confirm the content landed (`git diff main <branch>` returns nothing), then `-D`.
- Branch off `main`; never commit directly to it.
- **Work on `game-updates-N`, not a branch per feature.** Decided 2026-08-03, numbered from
  2026-08-04. This is a one-author project, so branch names buy no coordination — and because
  every PR is squash merged, **the branch name never reaches `main`'s history at all**. The
  PR title becomes the commit. Naming ceremony is pure overhead; the PR description is the
  thing that lasts.
- **The number increments once per PR, and a name is never reused.** The squash rewrites the
  commit, so a reused branch diverges from its own remote — one ahead, one behind, identical
  content — and the next push is refused until it is force-pushed. A fresh number sidesteps
  that entirely. `game-updates` ran through #19; `game-updates-2` starts after it.
  - The alternative considered and not taken: GitHub's *Automatically delete head branches*
    setting, which lets one name be reused forever with no per-PR step. Worth revisiting if
    the numbering starts to grate.
- **A PR may cover several unrelated things, and that is fine.** What is not fine is a
  branch that stays open across many sessions. `add-deck` became untrackable because of how
  long it ran, not because of what it was called — by the end it held the deck, the playback
  pacing, the button row, a debug-flag split and a glyph generator.
- **The cadence rule that replaces naming: land when you can still describe it honestly.**
  If the PR description will not fit one clear paragraph without turning into a list of
  unrelated headings, the branch has been open too long. Land it and start the next.
- **Starting the next branch after a merge:**

  ```powershell
  git checkout main; git pull; git checkout -b game-updates-3
  ```

  Then delete the merged branch locally, and its remote with
  `git push origin --delete game-updates-2`. Confirm the content landed first — with a
  squash, `git diff main <branch>` returning nothing is the check, not `git branch -d`
  succeeding.
- **Say so out loud before switching branches**, and never do it as a side effect of some
  other task.
- `git checkout main` **before** `git pull`. Pulling while on a feature branch drags
  `main`'s history onto that branch and causes confusion.
- Commits use the GitHub noreply identity, not a personal address.
- The owner reviews diffs in VS Code, so leave work **unstaged** unless asked to
  commit. Do not push or open a PR without being asked for that step.

## Determinism — a planned feature that constrains code written now

Runs will eventually be **replayable from a seed**: the same tower, enemies and rolls,
so the player can retry and make different choices. Nothing is stochastic yet, which is
exactly why these rules are cheap to follow — retrofitting determinism is expensive, so
do not write code that forecloses it.

- **Never call the `math/rand` package-level functions** (`rand.Intn`, `rand.Float64`,
  `rand.Shuffle`, …). They draw from a global source shared with every other caller,
  which makes a run unreproducible. Randomness comes from an explicit `*rand.Rand`
  carried on state and seeded once per run.
- **Four separate streams: enemy selection, loot offers, floor offers, card shuffles.**
  Never share one source between them, or a change to loot generation silently rerolls
  every enemy in the tower. A stream is only ever advanced by its own concern. Tower
  layout is fixed (8 floors × 3 fights, endless later) and draws no randomness.
- **The card shuffle is the only stream that exists yet**, and it lives on `CombatScene`
  as `rng`, seeded in `Init` from the `deckSeed` constant. That constant is a placeholder
  for the per-run seed: every launch deals the same opening hand, which is what makes a
  layout problem reproducible while the screen is being built. When `Session` state lands,
  the seed reads from there and nothing else about the deck code changes.
- **The deck lives on the scene, not in `internal/combat`.** Keeping the shuffle out of
  the rules package is what preserves its purity, its tests and the headless balance sim.
  Moving draw into `combat` is a real option later, but it has to arrive as an injected
  source parameter on `ResolveRound` and it changes `TestRoundIsDeterministic` — see the
  deckbuilder entry in `TODO.md` before doing it.
- **Do not pre-roll randomness into fixed-size slices.** A seeded `*rand.Rand` already is
  an infinite deterministic list, and the planned endless tower gives no worst case to
  size an array against. Rerolls simply advance the cursor.
- **Never let map iteration order affect an outcome.** Go deliberately randomises it.
  `gs.Combatants` is a map — iterate a sorted key slice if a choice depends on order.
- **No `time.Now()` in game rules.** Wall-clock decisions cannot be replayed. Tick
  counters are fine; they are part of the simulation.
- **`internal/combat` is pure integer arithmetic with no randomness and no clock.**
  `TestRoundIsDeterministic` pins this. If randomness ever enters combat it arrives as
  an injected source parameter, never a global.
- **Presentation may never change outcomes.** `ResolveRound` decides a whole round
  before playback begins, so animation speed, the planned game-speed setting, and any
  skip button are free to alter pacing and must not alter results.

## Resolution order is a player lever, not just an output

A round resolves the two queues **alternately**, one action each, with the longer queue
acting alone once the other empties. This replaced volley-per-side on 2026-07-31 — see the
entry in `TODO.md` for the full reasoning.

Within one exchange, **the faster action lands first**: lower `ActionKind.Initiative()`
wins, and side A takes a tie. Initiative is a lever wholly separate from cost — cost
decides what a plan may *contain*, initiative decides *when* its pieces happen — and it is
separate from `Spd`, which buys action points and never priority.

The intended loop is: **the player chooses their actions, then alters the resolution
order.** That is why two monolithic volleys were wrong; they gave the player nothing to
manipulate. Dragging a card to a different slot changes which of the opponent's actions it
contests, and therefore whether it beats that action or answers it.

- **`combat.ResolutionOrder` is the single authority on order.** `ResolveRound` plays what
  it returns and the Resolution pane draws what it returns. Neither derives the order
  independently, which is what makes it structurally impossible for the pane to lie to the
  player about their own round. `TestResolutionOrderIsWhatResolveRoundPlays` pins it.
- **Ordering is a rule.** It belongs in `internal/combat`, never in a screen. A new effect
  that rearranges resolution changes `ResolutionOrder` and both consumers follow.
- A raised Guard lasts until its owner's next action, so it covers every opposing action
  in between, across a round boundary if it was queued last. A duelist who queues
  nothing therefore keeps its guard — deliberate, pinned by
  `TestGuardHoldsWhileItsOwnerDoesNothing`, and worth revisiting during balancing.

## UI: clicks and drag-and-drop only

A firm design decision, not a current limitation. The entire input vocabulary is:

- **Left click** — buttons and selection.
- **Drag and drop** — the action box, and anything else that needs ordering or moving.
- **Long press** — reveal further information about the thing under the cursor. Not
  built yet.
- **One typed-text field in the whole game** — entering a seed to replay a run. Nothing
  else anywhere accepts keyboard input.

**No right click, ever.** There is no context menu and no secondary action. Anything
that feels like it wants one needs a different design.

- **Wanting a text field is a design smell.** Find the click or drag version instead.
  A settings value is a row of buttons or a slider, never a number you type.
- **No UI toolkit dependency.** Widgets are hand-rolled following the
  `models.Button` + `systems.UpdateButton`/`DrawButton` split. Add new widgets the same
  way: a plain struct in `models`, behaviour in `systems`, owned by the scene that uses
  it.
- ebitenui was evaluated and declined — see the entry in `TODO.md` for the reasoning and
  the repo data behind it. Do not reach for it without revisiting that.

The action box is a *game* widget, not a UI widget: draggable action cards with live
action-point validation. General-purpose toolkits are weakest at exactly that, so
hand-rolling costs little and buys full control.

### Glyphs are generated, not drawn — and not scaled from one colour

[internal/systems/glyphs.go](internal/systems/glyphs.go) generates the 64x64 pixel-art
glyphs on the action cards, drawn at 1:1. **Generated art has no provenance question**, which is exactly
the problem making the Tyrian set a release blocker, so this is the pattern to prefer for
interface art.

It is a **generator, not a bitmap**. A glyph is a filled silhouette described by horizontal
spans; the rim is *derived* by asking which filled pixels touch empty space, and the
interior shading is *computed* from where a pixel sits across its row and down the sprite.
Nothing is hand-placed, so a shape can be nudged without repainting it.

- **Nothing in a silhouette may be thinner than about five pixels.** The derived rim takes
  one pixel off each side, so a three-pixel crossguard renders as two rows of outline
  around one row of metal and reads as a scratch. This is the main constraint the technique
  imposes and it drives every span in the file.
- **Glyphs are the deliberate exception to the colour rule below.** They carry a five-value
  `Palette` — outline, specular, highlight, mid, shade, accent — because a bevel cannot be
  made from one colour scaled down. They are drawn untinted; a disabled card dims them by
  *alpha*, so the shading survives and only the weight changes. Tinting one toward the card
  colour would collapse it back to a flat silhouette.
- **Colour was kept unspent, and elements spent it — on the card, not the glyph.** There is
  still one palette, `white`, with no hue at all, and all three glyphs use it. What carries
  the element is the card *surface*: fire orange, ice blue, lightning yellow, poison green,
  and near-white for no element at all. Holding the glyphs hueless from 2026-08-03 is
  exactly what made that possible a day later, and they should stay that way — a coloured
  glyph on a coloured card says it twice and leaves nothing for the next distinction.
- **A glyph cannot be made smaller, and that sets the floor on a card's size.** `GlyphSize`
  is 64 and `CardGlyphScale` is 1: authored at the size it is shown, integer scales only, and
  1 is already the floor. Three glyphs therefore need 3x64 plus gaps down and 64 plus a
  numeral across, whatever else a card does. A "smaller" card is one with less padding and
  smaller text around an identical column — which is why the deck overlay's card is 138x236
  against the hand's 180x264. Do not reach for a fractional scale; it drops pixels out of a
  rim that is one pixel thick.
- `RenderGlyph` returns a plain Go image and is free of Ebitengine on purpose — creating an
  `*ebiten.Image` needs a graphics context, and the review tool has no window. `Glyph`
  wraps and caches it for the game.

**`go run ./tools/glyphsheet` writes `tools/glyphsheet/glyphs.png`** — every glyph by every
palette, so the art can be reviewed by opening a file rather than by launching the game and
hunting for a card that uses it. **The sheet is committed on purpose**: GitHub renders image
diffs side by side, so a change to a silhouette shows up in review as a picture. Regenerate
it whenever the glyph code changes — a stale sheet is worse than none, because it is a
picture that lies.

The output sits beside the tool that makes it rather than in a shared directory, so the pair
moves and is understood together. It is deliberately **not** in `assets/`: everything there
is `//go:embed`ed and loaded at runtime, and the sheet is a picture *of* generated art, not
an input to it. Filing it as an asset would imply the game reads it, which is the opposite
of the property that makes generating glyphs worth doing.

**The sheet draws each glyph twice: at `systems.CardGlyphScale` and enlarged.** The
actual-size row is the one that answers "can I read this". The scale constant lives in
`systems` precisely so the sheet reads the same number the card does and the preview cannot
drift from the game — an earlier version showed only the enlarged row, and the glyphs duly
looked acceptable in review and clunky in play.

### Colour: name one colour and scale it — but the rule is narrower than it reads

**This applies to widget *state*, not to widget *surfaces*.** A button naming crimson and
brightening toward it on press is the rule working. A button that can never have a lit top
edge and a shadowed bottom one is the rule overreaching, and it does currently overreach:
glyphs had to be written down as an exception when the only real problem was that a bevel
needs more than one value.

The intended direction, stated 2026-08-03: **buttons, cards and the resolution panes all
want bevelling eventually**, from palettes like the ones in
[glyphs.go](internal/systems/glyphs.go). When that lands, the rule below should be rewritten
as what it actually is — how a surface responds to hover, press and disable — with the
surface's own light and shade coming from a palette. Until then it still governs everything
that has not been given one.



A widget names the colour it wants at **full strength**, and its other states are
scaled down from that with `systems.ColorAtStrength`. `models.Button.BaseColor` is the
reference case: the button rests at 65%, hovers at 82% and reaches the named colour at
100%, so pressing it lights it up to exactly the colour in the source.

- **Scale a colour, never add to it.** Adding a fixed step to every channel walks a
  saturated colour toward white — crimson hovering to a washed-out pink — and a channel
  already near 255 has nowhere to go. Scaling holds the hue.
- A zero-alpha colour means "use the default", so widgets that never pick one are
  unaffected.
- Disabled deliberately ignores the widget's colour. A disabled control should read as
  unavailable first and as itself second.

### Combat screen layout is scaffolding

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

### The action box

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

### Two debug flags, and they are not interchangeable

`ActiveDebug` split into `DebugPlacement` and `DebugGameplay` on 2026-08-02, because they
answer different questions and are wanted at different times.

- **`DebugPlacement`** — the grid, the rulers, the `Debug1`/`Debug2` scratch strings. About
  *where things are drawn*. Safe to leave on while playing, and on by default because the
  combat screen is still being laid out.
- **`DebugGameplay`** — perfect information, starting with the opponent's queued actions.
  About *what the player is allowed to know*. **Off by default**: with it on you are not
  playing the game, you are inspecting it, and it is easy to tune balance against a view no
  player will ever have.

Neither may ever change an outcome. Both are views, the same constraint that applies to
playback speed — `ResolveRound` never sees either flag.

Both are set once in `main.go`; there is no runtime toggle, because a hotkey would need the
keyboard and the input vocabulary does not have one. **Both default to off.**

### `internal/trace` is a third thing, and it is compiled out

[internal/trace](internal/trace) writes a running account of what the game did — layout
rectangles, resolved rounds, clicks and drags — and periodically captures the screen to
`trace/frame.png`. It exists so a problem can be diagnosed from output rather than from
someone describing what they saw, or taking screenshots by hand.

```powershell
go run -tags debugtrace .     # traced
go run .                      # nothing: every trace function is empty
```

- **A build tag, not a runtime flag, and that is the point.** The two debug flags above are
  *views* a player could conceivably be given. This is instrumentation for whoever is
  building the game and it must not be in a binary that ships. `go build .` carries none of
  it — no strings, no PNG encoder, no file writes.
- **It must stay deletable in one commit.** That property is what makes it acceptable in a
  product that will be sold. If trace calls spread thinly through the screens, it is gone.
- **`internal/combat` may never import it.** trace imports Ebitengine, and the rules package
  not importing Ebitengine is exactly what makes it testable without a window. The *screen*
  traces the event log `ResolveRound` hands back; combat itself stays clean.
- **It may never change an outcome**, the same constraint as the debug flags and playback
  speed. `ResolveRound` neither sees it nor calls it.
- **Guard call sites that build their arguments** with `if trace.Enabled()`. The no-op
  functions cost nothing, but Go still evaluates what is passed to them.
- Lines carry the **simulation tick**, not a wall clock, so a trace lines up with a replay of
  the same seed. Captures are throttled to one every two seconds: `ReadPixels` is a
  GPU-to-CPU readback that stalls the frame it happens on.
- The layout dump re-runs whenever the **hand size** changes, since the whole bottom band is
  a function of that number. `tracedHand` watches it, so no call site has to remember.

### Hidden information is gated on `DebugGameplay`

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

## Architecture

### Ebitengine game loop

`main.go` builds the `game.Game`, loads assets/fonts/data once, then hands control to `ebiten.RunGame`. It does **not** wire up widgets — scenes build their own. Ebitengine then drives three methods on [game.go](internal/game/game.go):

- `Update()` — 60 TPS logic tick. Advances counters, reads the mouse, runs the active scene's `Init` if `NewScreen` is set, then returns the scene's `Update`. Returning a non-nil error quits the game; `ShouldClose` becomes `game.ErrClosing`, which `main` treats as a clean exit (window close is intercepted via `SetWindowClosingHandled(true)`).
- `Draw(screen)` — per-frame rendering. Returns early while `NewScreen` is set, so a scene is never drawn before its `Init` has run; then calls the scene's `Draw` and overlays debug info last if `DebugPlacement`.
- `Layout(w, h)` — returns the fixed 1280x960 internal resolution and records it on `GlobalState`, which is what `PctX`/`PctY` read to place things.

### Scenes own their own state

[internal/screens/scene.go](internal/screens/scene.go) defines the `Scene` interface — `Init` / `Update` / `Draw`. Each screen is a struct implementing it, registered once in `NewGame`'s `scenes` map. There is one registry rather than parallel `Update` and `Draw` switches, which used to be able to drift apart.

**A screen's working state belongs on its scene, not on `GlobalState`.** `CombatScene` owns the combatants, the queued action sets, the resolved event log, the playback cursor and its DUEL! button. Adding per-screen state means adding a field to the scene.

Scenes also build their own widgets in `Init` and wire them to their own methods (`models.NewButton(..., s.startRound)`). Nothing outside the scene needs to know it has a button.

`Init` may run more than once — a screen can be re-entered — so build expensive things behind a nil check and reset per-visit state unconditionally.

### GlobalState is what is genuinely shared

[internal/state/global_state.go](internal/state/global_state.go) is threaded by pointer (`gs`) into every scene and system, and carries only what is actually global: input, timing, layout, loaded resources, `ActiveScreen`, `NewScreen`, `ShouldClose`, and debug scratch. It imports nothing from `combat`, `entities` or `models`.

Key conventions:
- `ActiveScreen` (`Title`/`Ascend`/`Combat`/`Credits`) selects the scene. Adding a screen means adding an `ActiveScreen` constant and one entry in the `scenes` map.
- **`NewGlobalState` boots into `Combat`, not `Title`, on purpose.** The combat screen is the one under construction and clicking through the title every run costs a step in a loop that runs often. Leave it unless asked; put `ActiveScreen` back to `Title` once combat stops being the active work, and flag it when a change needs the title screen to be seen. **Both debug flags are off by default in `main.go`** — `DebugPlacement` used to default on while the screen was being laid out and no longer does, so a change that needs the grid or the rulers has to turn it on deliberately.
- `NewScreen bool` is the one-shot init flag, consumed centrally in `game.Update`. Actions that change screens set it back to `true`; scenes never touch it.
- `gs.PctX(pct)` / `gs.PctY(pct)` are the intended way to place things — avoid hardcoded pixel coordinates. They replaced a dozen cached fields for halves, thirds and quarters, which could not express 40% and could not be extended to without inventing a field name per fraction. **Percentages anchor a group; offsets within a group stay in pixels** (see the title menu), and sizes are never percentages. The debug overlay rules the screen at the halves/thirds/quarters and ticks every 10% along the top and left edges, so a position can be read straight off the screen.
- `Debug1`/`Debug2` are free-form strings printed by `DrawDebugInfo`; scratch tracing goes there.

### Package layout and its layering

- `assets/` — `//go:embed`s every image and font into the binary and exposes `LoadAssets()` / `LoadFonts()`, returning `map[string]*ebiten.Image` and `map[string]*text.GoTextFaceSource`. **A new asset needs three edits: the file, an `//go:embed` var, and a map entry in the loader.** The map key is the lookup name used everywhere else (e.g. `gs.Assets["spritesheet_png"]`, `gs.Fonts["kubasta"]`).
- `data/` — `//go:embed`s `combatants.json` and unmarshals it into `map[string]CombatantData` keyed by `CombatantRecord`. This is the pattern for all static game data: JSON next to a small Go loader. `SpriteSheet` in the JSON must match an `assets` map key, and `SpriteRect` is `[x0, y0, x1, y1]` used with `SubImage` to slice the sheet.
- `internal/models/` — plain data structs with no behaviour (`Button`). Constructors only.
- `internal/systems/` — the behaviour for models, split as `Update*` and `Draw*` free functions taking `(gs, ...)`. `models.Button` + `systems.UpdateButton`/`DrawButton` is the reference example of this model/system split; follow it for new widgets.
- `internal/entities/` — game-world actors (`Combatant`, embedding `combat.Duelist`), hydrated from `data` records at scene init.
- `internal/combat/` — the duel rules. **No Ebitengine import, ever.** `ResolveRound` returns an event log plus the end state; the screen replays it and never computes an outcome. This is the only package with tests, because it is the only one that needs no window. **Never change these rules to make a screen look right** — if a screen contradicts the engine, say so and let the owner decide which one is wrong. That is a game-design call, and it ripples into the tests and the balance.
- `internal/screens/` — one `Scene` implementation per screen, owning its own state and widgets, calling into `systems` to draw them.
- `internal/actions/` — callbacks that act on the game as a whole: change screen, quit. They take `gs` and mutate it; they never draw. **Callbacks touching only one screen's state do not go here** — those are methods on the scene that owns the state.

Dependency direction: `main` → `game` → `screens` → `systems`/`entities`/`actions` → `models`/`state`/`combat`/`data`/`assets`. Nothing lower reaches back up. `state` sits near the bottom and must stay there — if it starts importing `entities` or `models` again, screen state has leaked back into it.

### Drawing idioms

- Sprites are drawn via `colorm.DrawImage` so a `colorm.ColorM` can tint/hue-shift them; buttons and shapes use `vector.DrawFilled*` into a scratch `ebiten.NewImage`.
- Positioning convention: translate by `-w/2, -h/2` first to center the origin, then translate to the target coordinate. Buttons store `ScreenX`/`ScreenY` as their *center*, and both `UpdateButton` (hit testing) and `DrawButton` re-derive the top-left from it.
- Rounded rectangles are done by drawing an opaque mask then compositing with `ebiten.BlendSourceIn` — see `DrawHealthBar` and `CreateRoundedRecMask` in [combat.go](internal/screens/combat.go).

## Art

Sprites come from the free Tyrian graphics set (see README). `assets/tyrian_graphics/` holds the raw source sheets; only the consolidated sheets referenced by `embed.go` are actually compiled in.
