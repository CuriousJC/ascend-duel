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

## Architecture

### Ebitengine game loop

`main.go` builds the `game.Game`, loads assets/fonts/data once, wires up the title-screen buttons, then hands control to `ebiten.RunGame`. Ebitengine then drives three methods on [game.go](internal/game/game.go):

- `Update()` — 60 TPS logic tick. Advances counters, reads the mouse, and dispatches to the active screen's `Update*Screen`. Returning a non-nil error quits the game; that is how `ShouldClose` exits (window close is intercepted via `SetWindowClosingHandled(true)`).
- `Draw(screen)` — per-frame rendering. Dispatches to the active screen's `Draw*Screen`, then overlays debug info last if `ActiveDebug`.
- `Layout(w, h)` — called on resize; recomputes the cached thirds/quarters/halfway coordinates used for positioning everywhere else.

### GlobalState is the spine

[internal/state/global_state.go](internal/state/global_state.go) defines one `GlobalState` struct threaded by pointer (`gs`) into every screen, system, and action. There is no ECS and no dependency injection — if a component needs something, it lives on `GlobalState`. Adding a new screen, entity, or asset map generally means adding a field here.

Key conventions on `GlobalState`:
- `ActiveScreen` (`Title`/`Ascend`/`Combat`/`Credits`) selects which screen runs. Both the `Update` and `Draw` switches in `game.go` must be updated together when adding a screen.
- `NewScreen bool` is the one-shot init flag: a screen's `Update` checks it, runs `Init*Screen`, then clears it. Actions that change screens set it back to `true`.
- Layout fields (`HalfwayX`, `FirstThirdY`, `ThirdQuarterX`, …) are recomputed in `Layout` and are the intended way to place things — avoid hardcoded pixel coordinates.
- `Debug1`/`Debug2` are free-form strings printed by `DrawDebugInfo`; scratch tracing goes there.

Note: `NewGlobalState()` currently starts on `Combat`, not `Title` — a development shortcut, not the intended flow.

### Package layout and its layering

- `assets/` — `//go:embed`s every image and font into the binary and exposes `LoadAssets()` / `LoadFonts()`, returning `map[string]*ebiten.Image` and `map[string]*text.GoTextFaceSource`. **A new asset needs three edits: the file, an `//go:embed` var, and a map entry in the loader.** The map key is the lookup name used everywhere else (e.g. `gs.Assets["spritesheet_png"]`, `gs.Fonts["kubasta"]`).
- `data/` — `//go:embed`s `combatants.json` and unmarshals it into `map[string]CombatantData` keyed by `CombatantRecord`. This is the pattern for all static game data: JSON next to a small Go loader. `SpriteSheet` in the JSON must match an `assets` map key, and `SpriteRect` is `[x0, y0, x1, y1]` used with `SubImage` to slice the sheet.
- `internal/models/` — plain data structs with no behaviour (`Button`). Constructors only.
- `internal/systems/` — the behaviour for models, split as `Update*` and `Draw*` free functions taking `(gs, ...)`. `models.Button` + `systems.UpdateButton`/`DrawButton` is the reference example of this model/system split; follow it for new widgets.
- `internal/entities/` — game-world actors (`Combatant`), hydrated from `data` records at screen init.
- `internal/screens/` — one file per screen, each exporting `Init*Screen` / `Update*Screen` / `Draw*Screen`. Screens own their composition and call into `systems`.
- `internal/actions/` — button `OnClick` callbacks. They take `gs` and mutate it (change screen, set `ShouldClose`); they never draw. Actions are bound at button construction in `main.go` via closures over `g.GlobalState`.

Dependency direction: `main` → `game` → `screens` → `systems`/`entities` → `models`/`state`/`data`/`assets`. Nothing lower reaches back up.

### Drawing idioms

- Sprites are drawn via `colorm.DrawImage` so a `colorm.ColorM` can tint/hue-shift them; buttons and shapes use `vector.DrawFilled*` into a scratch `ebiten.NewImage`.
- Positioning convention: translate by `-w/2, -h/2` first to center the origin, then translate to the target coordinate. Buttons store `ScreenX`/`ScreenY` as their *center*, and both `UpdateButton` (hit testing) and `DrawButton` re-derive the top-left from it.
- Rounded rectangles are done by drawing an opaque mask then compositing with `ebiten.BlendSourceIn` — see `DrawHealthBar` and `CreateRoundedRecMask` in [combat.go](internal/screens/combat.go).

## Art

Sprites come from the free Tyrian graphics set (see README). `assets/tyrian_graphics/` holds the raw source sheets; only the consolidated sheets referenced by `embed.go` are actually compiled in.
