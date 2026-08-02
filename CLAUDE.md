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

### Colour: name one colour and scale it

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

### Combat screen panes are scaffolding

The combat screen reads **fighter / palette / resolution / enemy**: two panes with the
duelists as bookends and the round between them. Colours identify the role and are
placeholders, not a chosen palette.

| Element | Slot | Colour | Role |
|---|---|---|---|
| Fighter sprite | 9% | — | the player |
| Actions | 18–38% | green | available cards, the AP budget, then the queue |
| Resolution | 45–78% | pink | both queues interleaved in play order |
| Enemy sprite | 88% | — | the opponent |

**The Actions pane stacks two zones in one column.** Available cards from `availableTop`,
the AP text and bar acting as the divider, then the queued cards from `chosenTop` — and
`chosenZone` is the lower one, which is the drop target. A card released anywhere else,
including on the available cards above, is discarded. Card geometry keys off a zone offset
rather than a pane, which is what `cardSlot(gs, top, i)` takes.

**The Resolution pane is the centrepiece and gets the width to prove it.** It is the only
pane that has to grow: once exchanges have structure — an initiator and a response — it
has to draw that rather than a flat list of rows.

Two panes rather than four, decided 2026-08-02. Chosen folded into the palette because one
column can hold both zones; Enemy went entirely because an interleaved Resolution already
shows the opponent's actions in a better order than a column of its own. The player's rows
carry the palette's green and the opponent's carry `enemySwatch` yellow, so the screen
reads as two colours: green is you, yellow is them. Do not treat the widths or colours as
settled.

### The action box

[combat_actionbox.go](internal/screens/combat_actionbox.go) is the drag-and-drop queue,
and the reference for building a *game* widget: state on the scene, hand-rolled hit
testing, no toolkit. Drag from the available zone to queue, drag within the queue zone to
reorder, drag out of the queue zone to discard.

- **`planning()` is the single predicate** for "the player may edit the queue" — derived
  from `cursor >= len(log)` plus both duelists alive, not stored. Drag and the DUEL!
  button both gate on it, so they can never disagree.
- **The action-point budget is enforced at pick-up.** A card the remaining points will
  not cover cannot be lifted and draws dimmed. Accepting a drag and then bouncing the
  drop is a worse conversation than never letting the card leave.
- **A card dragged out of the queue leaves it on pick-up**, not on drop, so the gap
  closes under the cursor and the drop index is measured against the real list.
  Dropping outside the pane is therefore how an action is removed.
- The in-flight card is drawn last in `Draw`, so it rides over the panes it crosses.
- **The AP budget is drawn twice, on purpose.** A `3/6 AP` line for the exact figure and a
  bar under it for the glanceable one. A card in flight from the palette adds a dimmer
  segment ahead of the fill, so the bar answers "does this fit" before the card lands —
  it has not joined the queue yet and would otherwise not move the bar at all.

### Hidden information is gated on `ActiveDebug`

The opponent's queued actions are concealed in both the Enemy pane and the enemy rows of
the Resolution pane, unless `ActiveDebug` is on. `CombatScene.concealEnemy` is the single
predicate — `!gs.ActiveDebug && s.planning()` — and anything else that becomes secret
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
- `Draw(screen)` — per-frame rendering. Returns early while `NewScreen` is set, so a scene is never drawn before its `Init` has run; then calls the scene's `Draw` and overlays debug info last if `ActiveDebug`.
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
- **`NewGlobalState` boots into `Combat`, not `Title`, on purpose.** The combat screen is the one under construction and clicking through the title every run costs a step in a loop that runs often. `ActiveDebug` in `main.go` is on by default for the same reason. Leave both unless asked; put `ActiveScreen` back to `Title` once combat stops being the active work, and flag it when a change needs the title screen to be seen.
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
