# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Ascending Duel — a roguelike where you duel your way up a tower, collecting rings and brands of power. Written in Go with [Ebitengine v2](https://ebitengine.org/) (`github.com/hajimehoshi/ebiten/v2`). Module path: `github.com/curiousjc/ascend-duel`.

## Where things are written down

Five streams, each with one job. Reach for the right one rather than searching all of them.

| Stream | File | Read it when |
|---|---|---|
| **How to work** | `CLAUDE.md` — this file | always; it is loaded every session |
| **Procedure** | `.claude/skills/*/SKILL.md` | on trigger — see the index below |
| **What the game *is*** | [MECHANICS.md](MECHANICS.md) | designing or implementing any mechanic, before proposing a design change |
| **What to build next** | [TODO.md](TODO.md) | picking up work |
| **Unfiltered** | [ideas.md](ideas.md) | the inbox; entries get promoted into MECHANICS or TODO and struck from here |

- **`MECHANICS.md` is the design record.** Decided unless marked `[?]`. It holds the element
  set and their statuses, cards and types, hands, rings, brands, vitae, the tower, enemies,
  and the phase-based resolution experiment.
- **`TODO.md` is open work only.** Completed entries are deleted rather than archived, so it
  says what is left, not what happened. Prefer `MECHANICS.md` for "what should this do".
- **Never add an entry to `TODO.md` unless the owner asks for that specific thing to be tracked**
  *(owner's call, 2026-08-23)*. Noticing something during a change is not a reason to file it.
  Say it in the reply and let the owner decide — the list is theirs to grow, and a list that
  accumulates every observation anybody had is one nobody reads. The same goes for `[?]` entries
  in `MECHANICS.md`: an open question is filed because the owner wants it open, not because the
  work turned one up.
- When the two disagree, `MECHANICS.md` is newer and wins — say so rather than guessing.
- **Cut means deleted, not tombstoned.** When something is taken out of the design, remove
  every trace of it rather than leaving a note saying it was removed and why. These files are
  loaded into context; a record of things that do not exist is a running cost paid on every
  session, and it grows without bound because nothing ever retires a tombstone. Git history is
  the record of what used to be true. **If a removal genuinely needs to stay visible — because
  the code still has a shape only the dead mechanic explains, or the same idea keeps being
  re-proposed — ask before writing it down**, rather than deciding alone that it earns a line.

### The skill index — every skill and what trips it

**This table is the tripwire.** The individual sections further down explain *why* a skill
exists; this says only what makes you reach for one, so nothing is missed by not having read to
the bottom of a long file. **A new skill is a row here** — a skill nobody knows to load is a
skill that does not exist.

| Skill | Load it before |
|---|---|
| [`caveman`](.claude/skills/caveman/SKILL.md) | **every session, at the start** — see below; it is on by default in this repo |
| [`github-workflow`](.claude/skills/github-workflow/SKILL.md) | any `git` or `gh` command — branching, committing, pushing, opening or merging a PR, cleaning up after one, or when a merge is refused |
| [`data`](.claude/skills/data/SKILL.md) | adding a file to `data/`, adding or changing a field on one, authoring cards / enemies / rings / worms, or writing a loader |
| [`randomness`](.claude/skills/randomness/SKILL.md) | adding any roll, adding or seeding a stream, touching a salt or a seed, writing a shuffle, or deciding whether a mechanic should be random at all |
| [`combat-screen`](.claude/skills/combat-screen/SKILL.md) | touching any `internal/screens/combat*.go`, `internal/combat`, or anything about how a round is drawn or played back |
| [`rings`](.claude/skills/rings/SKILL.md) | designing or **discussing** a new ring, adding to `rings.json` or `statuses.json`, adding a moment or an effect verb, or wiring anything that reads a worn ring |

**Loading is cheap and guessing is not.** Every one of these exists because something specific
went wrong once and should not have to be rediscovered.

## Caveman mode is on in this repo

**Load the `caveman` skill at the start of every session and stay in it.** It compresses
replies in the terminal and nothing else.

- **It never touches anything persisted** — code, comments, commit messages, PR bodies,
  `MECHANICS.md`, `TODO.md`, this file. The skill's own Boundaries section says so and it is
  the reason this is safe: the design record keeps its longhand reasoning. Compression
  applies to what is typed back into the terminal.
- **It does not override the instructions that say to argue.** Raising a structural
  objection, saying where a claim came from, and saying plainly when something is unverified
  all still happen — in fewer words, not fewer times. **If terseness is eating the
  reasoning rather than the padding, say so** rather than quietly writing shorter.
- **It drops out by itself for security warnings, irreversible actions and genuine
  ambiguity**, and should also drop for design discussion, where the argument is the point.
- Default level is **full**. Ask for `lite` or `ultra` in words; the `/caveman` switcher is
  a hook that is deliberately not installed. See
  [.claude/skills/caveman/README.md](.claude/skills/caveman/README.md) for what else was
  left out and the honest token arithmetic.
- **"stop caveman" or "normal mode" turns it off for a session** without editing anything.

## Licensing and IP — read before adding dependencies or assets

The project is **source-available, not open source**, and is intended to be sold
(Steam) by its two copyright holders while the source stays public.

- **License: PolyForm Noncommercial 1.0.0** (`LICENSE`). Anyone may read/build/modify/share
  noncommercially; selling is reserved to the copyright holders. Additional Permissions at
  the top of `LICENSE` explicitly allow monetized streaming and video of gameplay.
- **Copyright holders: Justin Crosby (CuriousJC) and KingSherman1820.** They have a
  written partnership agreement; Justin can speak for both on licensing and IP, so
  there is no need to ask whether Sherman agrees.
- **`CONTRIBUTING.md` carries a contributor grant.** Outside contributions must come
  with a commercial-use grant, or merging them would leave the game unsellable. Do not
  weaken or remove that document.
- **No GPL, ever.** Every dependency must be permissive (MIT / BSD / Apache-2.0) or it
  cannot go into a product licensed this way. Check the license before adding anything
  to `go.mod`, and flag it in the PR.
- **Assets need provable licenses.** Do not add assets with unclear provenance; "found
  it online" is not sufficient for a paid release. The enemy portraits are PVGames creature
  art from the Humble *Isometric Assets Galore* bundle, whose licence permits shipping
  inside a game; the pack and its terms are in `.scratch/flat-creatures` (gitignored).
  Everything else in `assets/` is either first-party or generated at runtime.
- **Do not propose rewriting git history over the relicense.** The Apache 2.0 grant on
  commits published before it is irrevocable, and the owners have accepted that.

## Commands

```powershell
go run .            # build and launch the game window
go build .          # produce ascend-duel.exe (gitignored)
go vet ./...
gofmt -l .          # list unformatted files

go run -tags debugtrace .   # with internal/trace live: event log + trace/frame.png
go run -tags idleexit .     # closes itself after two minutes with nobody at the controls
go run -tags demoplay .     # plays a scripted round by itself, writes demo/*.png, exits
go run -tags scenario .     # a chosen set of rings, a chosen opening hand, a chosen enemy
ASCEND_DUEL_SCENARIO=seven-term-sum go run -tags scenario .   # a named one
go run ./tools/glyphsheet   # regenerate the committed glyph contact sheet
go run ./tools/sheets       # regenerate every review sheet and the index that links them
go run ./tools/cardsheet    # every card variation to PNGs + an HTML page, then refresh the tab
go run ./tools/ringsheet    # every ring to PNGs + a page grouped by rarity: art, price, text, rules
go run ./tools/wormsheet    # every worm to PNGs + a page grouped by what it changes about a card
go run ./tools/handsheet    # every rung of the hand ladder as a real hand, by ascending multiplier
go run ./tools/enemysheet   # all 96 creatures by floor band: card, stat line, whole deck
go run ./tools/bosssheet    # the 30 stairway protectors, the same way, by floor
go run ./tools/seeds        # re-check the named deck seeds, and search for new ones
go run ./tools/handodds     # how often each rung of the hand ladder can actually be built
```

**The six sheets are committed, under `docs/sheets/`** *(owner's call, 2026-08-23)*. They write
there rather than beside their own tools, and `docs/sheets/index.html` is the page a bare clone
opens to see every card, ring, worm, hand, creature and boss in the game. That reverses the older
rule that a regenerated artefact is not worth committing: the argument it left out is the
audience, since a sheet needing a Go toolchain and six remembered commands is a sheet only ever
seen by whoever just changed the thing it shows.

**The cost is history weight, so regenerate deliberately.** About 4.4 MB across 280 binary files
is rewritten by a full run, and a sheet rebuilt in a commit that changed nothing about it is pure
weight. **Three quarters of that is the two roster sheets**, which carry 126 photographic
portraits between them — so a commit touching only `rings.json` should regenerate the ring sheet
alone rather than reaching for the one command out of habit. **`go run ./tools/sheets` is the one
command** — it runs all six and rewrites the index, because six commands remembered in the right
order is how five end up current and one ends up lying. A stale sheet is worse than none: it is a
picture of a catalogue that no longer exists.

**A seed is an opening hand**, because the shuffle is deterministic. `internal/screens/seeds.go`
holds a catalogue of named seeds — `three-strikes`, `four-strikes`, `all-plans` — so a
hand that demonstrates something can be asked for by name instead of found by relaunching.
`deckSeedName` picks which one a launch deals.

**Re-run `tools/handodds` after touching the deck, and read the hand multipliers against what it
prints.** The ladder is priced off how reachable each rung is — a form pair is a 100% hand and pays
110, a concept Four of a Kind is a 0.15% hand and pays 500 — and every one of those figures is a fact
about `data/duelist_cards.json`, the hand size and the action budget. Change any of them and the ladder is
tuned against a deck that no longer exists, silently, because nothing fails. The tool measures
**reachability** rather than what the matcher picks: whether a hand of eight can afford some set
forming the rung, which is the question the player is actually answering. **It counts every card,
plans included** *(2026-08-23)* — they carry an element and a form now and join hands like anything
else, bringing no damage with them. MECHANICS.md holds the
table and the rule that turned it into multipliers.

**Re-run `tools/seeds` after touching `data/duelist_cards.json`, `startingDeck` or `handSize`.** A seed is
a fact about one particular deck; change the deck and every catalogued number silently deals
something else. The tool re-checks the catalogue before it searches and says which entries no
longer match — a change to the deck size has invalidated every entry at once before. A demo
testing a Three of a Kind against a hand with two Strikes in it is worse than no demo, because it passes.

**A rarer hand needs a bigger search, and some hands are impossible.** `four-strikes` is four of
the four Strikes in a hand of eight from 48 cards and turns up around seed 900; the default 20,000
finds it. **A hand wanting five copies of a concept cannot be dealt at all**, since no attack card
exists more than four times — so check the arithmetic before concluding either way. A hand the tool
reports as unfindable usually means the search was too short, but not always.

**Four build tags, and they compose.** Each selects a different file in its package, so one
configuration can compile while another does not. Vet and build every one you might have
broken:

```powershell
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go run -tags "debugtrace idleexit" .    # traced and self-closing: the unattended run
```

**`demoplay` is how the combat screen gets looked at without anybody sitting at it.** It plays
a scripted round or two — selection, DUEL!, playback — and writes the screen to `demo/*.png`,
then closes. It exists because the screen is the one thing `go test` cannot check: a hand line, a
marked verb, a highlight on the right row are all things you have to *see*. It is the
`tools/glyphsheet` idea applied to a live screen, and the same rule applies — a stale picture is worse than none, so regenerate rather than trust an old
capture. `demo/` is gitignored; fifty near-identical PNGs are not a diff anyone wants.

```powershell
go test ./...                                   # all tests
go test ./internal/combat -run TestName         # a single test
git commit -s                                   # sign-off, per CONTRIBUTING.md
```

Tests live in `internal/combat` — the only package that can be tested without a
window, by design. Keep it that way: rules go in `combat`, not in screens.

**"Needs no window" is not the same as "needs no display server", and CI found the
difference the hard way.** On Linux, `ebiten/internal/ui` calls `glfw.Init()` from a package
`init()`, so *linking* Ebitengine into a test binary is enough to panic on a missing
`DISPLAY` — before a single test function runs. Four of the tested packages link it:
`internal/screens` and `internal/models` directly, and `internal/cards` and `internal/music`
because their tests import `assets`, which hands back `*ebiten.Image`. `internal/combat`,
`internal/session` and the rest are genuinely clean. Both workflows therefore run the Linux test step under `xvfb-run -a`, which supplies
a throwaway X server nothing ever draws to. Windows is unaffected — Ebitengine is pure Go
there. **If a package's tests start importing `assets`, they have joined that group**; the
package's own no-Ebitengine rule still holds and is still worth holding, but it no longer
buys a display-free test run.

**What cannot be unit-tested gets a tool instead.** `internal/screens` needs a window, so anything
it decides is checked by launching the game, and the sheets under `docs/sheets/` are how the
catalogues get reviewed.

**Nothing simulates a duel.** An unwinnable enemy is invisible while playing, because losing slowly
looks exactly like losing to bad draws — so a cost, a stat line or a planner can be changed today
without anything catching what it did. `internal/combat`, `internal/decks` and `internal/pyramid`
are all free of Ebitengine, which is what would let a headless simulation be written. Keep them
that way.

## Releasing — `.github/workflows`

**CI** runs on every PR, on **Windows and Linux**, under all three build-tag configurations.
**Release** has two entrances and both produce the same release. Pushing a `v*` tag still
fires it, so *tagging is releasing*:

```powershell
git tag -a v0.1.0 -m "..."; git push origin v0.1.0
```

- **Or run it by hand from the Actions tab** — *Release* → *Run workflow*, on `main`. This is
  the normal path. **The workflow creates the tag itself**, at the commit it built, via
  `gh release create --target`, so a local tag-and-push is not needed to cut a release.
- **The manual run's `version` input is optional, and the usual way to use it is to leave it
  blank.** The `version` job then reads the highest existing `vX.Y.Z` tag off
  the remote and increments whichever part the `bump` dropdown names — `patch` by default,
  `minor` or `major` on request — so cutting a release does not mean remembering what the
  last one was. Typing a version still wins outright and ignores `bump`, which is how a
  prerelease or any other version the arithmetic would not produce gets cut. With no tags at
  all the base is `v0.0.0`, so a first `minor` release is `v0.1.0`.
  **Prereleases are excluded when scanning for the latest tag**, because `sort -V` puts
  `v1.2.0-rc1` after `v1.2.0` and bumping from an rc would skip the release it was a
  candidate for.
- **The manual path is guarded, because a typed version has no `v*` filter in front of it.**
  A `version` job runs first and fails the whole run if the branch is not `main`, if a
  supplied string is not `vMAJOR.MINOR.PATCH`, or if the resolved tag already exists — the
  last one because `gh` would otherwise attach binaries to a tag naming a different commit,
  and it is checked for a computed version as well as a typed one. The input is read
  through the environment and never interpolated into a shell script; this job is one
  `needs:` away from the write token.
- **The `version` job's output is the single source of the version**, and nothing downstream
  may read `GITHUB_REF_NAME` — it is the tag on one path and the branch name on the other, so
  the failure it prevents is a binary stamped `-X main.version=main`.
- **A tag the workflow creates cannot re-trigger it.** Pushes made with `GITHUB_TOKEN` raise
  no workflow events, so the manual run publishes once instead of looping.
- **Windows ships an `.exe`, Linux a `.tar.gz`.** Release assets carry no file permissions,
  so a bare Linux binary downloads without its execute bit and does not run. The tar keeps it.
- **Ebitengine is pure Go on Windows and cgo on Linux**, where it links against X11, GL and
  ALSA headers. Both workflows install the same apt list; if one changes the other has to.
- **Linux builds on `ubuntu-22.04`, not `latest`.** A cgo binary links against the glibc of
  the machine that built it and will not start on anything older, so the newest runner
  quietly narrows the audience. Oldest supported image is the widest reach.
- **Version is stamped at link time** — `-X main.version=<tag>` — and shown in the window
  title and on the title screen. `main.version` defaults to `"dev"`, because a plain
  `go run .` injects nothing and a build that guessed a version would be worse than one that
  admits it has none. This is what lets a bug report name a build; a filename stops
  travelling with the binary the moment it is renamed.
- **Neither `debugtrace` nor `idleexit` is ever set in a release build.** Instrumentation
  must not ship, and a game that closes itself on an idle player is a bug.
- **Only first-party actions**, and publishing goes through the `gh` CLI rather than a
  marketplace action. A job holding a write token is the last place to run unreviewed code.
  Build jobs upload artifacts; one `publish` job creates the release, because two jobs both
  calling `gh release create` is a race.
- **Action majors are pinned to whichever one actually runs on Node 24**, which is why the
  numbers look out of step: `checkout@v5`, `setup-go@v6`, `upload-artifact@v6`,
  `download-artifact@v7`. Both artifact actions shipped a major with only
  preliminary Node 24 support that still defaulted to Node 20 — v5 and v6 — so bumping by one
  leaves the deprecation warning on every run. Both workflows pin the same set; if one moves
  the other has to. Node 24 needs runner 2.327.1 or newer, which the hosted runners are.

Release notes live in `.github/release-notes/<tag>.md`. Missing ones fall back to generated
notes rather than failing a build that already succeeded.

## Git workflow — see the `github-workflow` skill

The procedure lives in
[.claude/skills/github-workflow/SKILL.md](.claude/skills/github-workflow/SKILL.md) rather
than here. **Load it before running any `git` or `gh` command** — branching, committing,
pushing, opening or merging a PR, cleaning up afterwards, or when a merge is refused.

It is a skill and not a section of this file on purpose: it is long, it is procedural, and
it only matters during the few minutes an actual git operation is happening. Most of what it
says is a specific thing that went wrong once and should not have to be rediscovered — the
review ruleset on `main` that makes `gh pr merge` fail while the protection API reports the
branch unprotected, why `git branch -d` always refuses a squash-merged branch, and why a
branch name is never reused.

The three decisions worth knowing without opening it:

- **Squash merge every PR**, so `main` reads as a list of milestones. Commit freely on the
  branch; the squash collapses them.
- **Work on `game-updates-N`**, incrementing every PR. Branch off `main`, never commit to it.
- **Leave work unstaged.** The owner reviews diffs in VS Code. Do not commit, push or open a
  PR without being asked for that specific step.

## Determinism — see the `randomness` skill

Runs will eventually be **replayable from a seed**. **Combat is stochastic as of 2026-08-14** —
lightning rolls — so the rules that protect replayability are live rather than theoretical, and
they are easy to break without noticing.

The procedure, the stream table and the argument a new roll has to make live in
[.claude/skills/randomness/SKILL.md](.claude/skills/randomness/SKILL.md). **Load it before
adding any roll, adding or seeding a stream, touching a salt or a seed, writing a shuffle, or
deciding whether a mechanic should be random at all.**

Four things stay here, because they are the tripwire — the failure is not knowing the skill
exists:

- **Never call the `math/rand` package-level functions** (`rand.Intn`, `rand.Shuffle`, …).
  They draw from a global source shared with every other caller, which makes a run
  unreproducible. Randomness comes from an explicit `*rand.Rand` carried on state.
- **Every consumer gets its own salted stream off `GlobalState.RunSeed`**, and a stream is
  only ever advanced by its own concern. Sharing one means a change to either silently rerolls
  the other. Four are live; the skill's table says which.
- **No `time.Now()` in game rules, and never let map iteration order decide anything.** Go
  randomises map order deliberately; iterate a sorted key slice.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before
  playback begins, so animation speed, a game-speed setting and any skip button may alter
  pacing and must not alter results. Same constraint as the debug flags, `internal/trace`,
  `internal/idle` and the scripted demo.

**Rewrite a random-sounding rule rather than let it in.** Lightning is the deliberate
exception, not the precedent, and a second roll needs its own argument in `MECHANICS.md` — the
skill says what the first one cost.

## The combat screen — see the `combat-screen` skill

Its layout, its card and action-box widget, its hidden information, and the resolution-order
rule the screen has to obey all live in
[.claude/skills/combat-screen/SKILL.md](.claude/skills/combat-screen/SKILL.md). **Load it
before touching any of the combat screen's files — `internal/screens/combat.go`,
`combat_deck.go`, `combat_hud.go`, `combat_actionbox.go` — or
`internal/combat`, or anything about how a round is drawn or played back.**

It is a skill because it is the screen under active construction: it grows every session
while mattering only when that screen is the work. The general UI conventions below still
apply to it and stay here.

Two things worth knowing without opening it:

- **`internal/combat` decides rounds; the screen only replays them.** Never change the rules
  to make a screen look right — say so and let the owner decide which one is wrong.
- **`combat.ResolutionOrder` is the single authority on play order**, and both `ResolveRound`
  and the table's two rows read it rather than deriving their own.

## UI: clicks and drag-and-drop only

A firm design decision, not a current limitation. These apply everywhere, the combat screen
included. The entire input vocabulary is:

- **Left click** — buttons and selection.
- **Drag and drop** — the action box, and anything else that needs ordering or moving.
- **Hover** — rest the cursor on something and a tooltip explains it *(2026-08-21)*. A card's
  damage arithmetic term by term, a ring's rule, a status badge's meaning. `models.Tooltip` and
  `systems.DrawTooltip` are the widget; the wording is `internal/screens/tips.go`.
- **Long press** — the same reveal, for a touchscreen or a controller, where there is no cursor to
  rest. **Not built**, and it is the only reason hover did not simply replace it: see MECHANICS.md
  §Hover and long press, where the record of hover being *rejected* was reversed.
- **One typed-text field in the whole game** — entering a seed to replay a run. Nothing
  else anywhere accepts keyboard input.

**No right click, ever.** There is no context menu and no secondary action. Anything
that feels like it wants one needs a different design.

- **Wanting a text field is a design smell.** Find the click or drag version instead.
  A settings value is a row of buttons or a slider, never a number you type.
### Cards fly; they never appear

**A card that changes where it is on screen travels there** *(2026-08-17)*. Drawn, discarded,
played to the table, re-sorted in the hand, won as a prize — every one of those is a journey with
a start, a duration and an eased arrival, never a card in one place on one frame and another place
on the next.

It is a rule rather than a flourish because of what a card *is* here: the same object moving
between piles that the player is tracking by eye. A card that appears in the middle of the screen
is a card that was never anywhere else, so the player has to re-read it to find out what happened
instead of having watched it happen.

- **`internal/screens/combat_flight.go` is the pattern**, and `travel` — a delay, an age, a
  duration, an eased progress — is the clock every mover shares. The post-battle screen uses the
  same one for the won card's flight to the centre.
- **Ease out, so a card leaves quickly and lands gently.** `easeOut` is what makes an arrival read
  as landing rather than as stopping.
- **A flight is raised after the model has already moved**, so it is a ghost of something that has
  happened. That is what keeps the state machine ignorant of animation — see `spendSelected`.
- **It may never change an outcome**, the same constraint as playback speed and the debug flags. A
  flight is something to look at.
- **The exception is an absence**: a removed card has nothing to fly, so the seat it would have
  landed in is drawn empty.

- **No UI toolkit dependency.** Widgets are hand-rolled following the
  `models.Button` + `systems.UpdateButton`/`DrawButton` split. Add new widgets the same
  way: a plain struct in `models`, behaviour in `systems`, owned by the scene that uses
  it.
- **ebitenui was evaluated and declined.** Everything the game needs is a *game* widget,
  which is where general-purpose toolkits are weakest, and a toolkit is one more dependency
  to licence-check against a product that will be sold. **The one trigger for revisiting is
  the seed text field** — a text input with a caret, selection and clipboard is the single
  widget genuinely cheaper to take than to build.

The action box is a *game* widget, not a UI widget: draggable action cards with live
action-point validation. General-purpose toolkits are weakest at exactly that, so
hand-rolling costs little and buys full control.

### Glyphs are mostly generated, and drawn art is the exception

[internal/systems/glyphs.go](internal/systems/glyphs.go) generates the pixel-art
glyphs, drawn at 1:1 — 64x64 for the damage sword and the runner, 32x32 for the small ones.
**Generated art has no provenance question**, which is the whole reason to prefer this pattern
for interface art in a game that will be sold.

**A `GlyphKind` has two possible backings, and `RenderGlyphAt` dispatches between them.** Most are
a generated silhouette; the four **form marks are authored PNGs** in `assets/form/`, listed in
`glyphArt` and loaded by key like the ring and effect art. A caller asks for a kind and does not
know which it got, which is what let drawn art in without a second drawing path.

It is a **generator, not a bitmap**. A glyph is a filled silhouette described by horizontal
spans; the rim is *derived* by asking which filled pixels touch empty space, and the
interior shading is *computed* from where a pixel sits across its row and down the sprite.
Nothing is hand-placed, so a shape can be nudged without repainting it.

- **Nothing in a silhouette may be thinner than about five pixels.** The derived rim takes
  one pixel off each side, so a three-pixel crossguard renders as two rows of outline
  around one row of metal and reads as a scratch. This is the main constraint the technique
  imposes and it drives every span in the file.
- **A card's corner carries a drawn form mark** *(2026-08-23)*: a spear, a sword, an axe and a
  bulb for stab, slash, crush and plan, from `assets/form/`. `cards.Form.glyph()` maps a form to
  its kind and reports `FormNone` as having none, so a ring and both fighter cards leave the slot
  empty. **The mark is tinted by the card's element** — see the card section below, which is where
  the element is said now.
- **Glyphs are the deliberate exception to the colour rule below.** They carry a five-value
  `Palette` — outline, specular, highlight, mid, shade, accent — because a bevel cannot be
  made from one colour scaled down. They are drawn untinted; a disabled card dims them by
  *alpha*, so the shading survives and only the weight changes. Tinting one toward the card
  colour would collapse it back to a flat silhouette.
- **Every glyph is *generated* in one hueless palette, `white`, and should stay that way.** The
  form marks are the exception and they are drawn art rather than generated: `cards.tintInk`
  colours one by the card's element on its way onto the face, which is where the element is said
  now (see the card section below). The palette itself stays hueless — the tint is applied to the
  rendered image, so the generator has no opinion about colour and the glyph sheet still shows
  every mark in one neutral set.
- **A hueless glyph on an off-white card loses most of its bevel, and that is accepted.**
  `Specular` is pure white and `Highlight` is `{232,236,242}`, so against the off-white
  surface the lit side of a bevel largely disappears and a glyph reads as outline plus
  shading. The near-black `Outline` carries legibility, so nothing is broken — but the
  five-value palette is mostly spent, and `cards.Surface` is one constant if that ever
  needs re-testing.
- **Glyphs are not all one size, and `systems.SizeOf(kind)` is the authority.** The damage
  sword and the runner are 64; the retired category glyphs were **22**. Never assume
  `GlyphSize` at a call site — a small glyph centred in a 64-pixel hole is the failure. **The
  card is now the exception and says so**: `Style.FormSize` names the box, and the mark is
  centred on its ink so that it fills one.
- **A glyph cannot be resized, so a smaller one is a *different drawing*.** `CardGlyphScale`
  is 1 and integer-only: the rim is derived one pixel thick, so a third-size copy of a 64px
  shape is a third-size copy of its outline with nothing inside. The 22px category glyphs
  were authored at 22, and at that size the detail budget is nearly nothing — a 5px feature
  is two pixels of rim around three of fill. **They are told apart by proportion before any
  detail is legible**: the sword narrow and vertical, the shield wide-shouldered, the book
  wider than tall. That is the design constraint to work with, not around.
- **`GlyphKind` is append-only.** The glyph cache keys on the ordinal, so inserting a kind
  mid-enum silently re-points every existing entry — the same hazard `combat.ConceptID` carries,
  which is why an ID is never serialized.
- **The card is 162x224, and it is a column and a paragraph.** The form mark sits in a 32px
  box at (10,8) — **inside the card, not hanging off the corner**, because a mark carrying
  detail loses it to the card's curve where a plain silhouette would survive the crop; under it
  the cost dashes
  make a **26px column**; and **the effect text takes everything right of that**, centred in it
  both ways, at 18pt. `blitGlyph` still clips to the rounded shape, which is what a future glyph
  will want back.
- **There is no damage badge at all.** The 64px generated sword went first — it said what the
  corner mark already says — and then the bare figure, because the text states what the card
  deals and a number beside it was the same fact multiplied out by the wielder's Strength.
  `cards.Spec` has no `Damage` field and `drawCard` takes no Strength. `systems.GlyphDamage` still
  exists and is still on the glyph sheet; nothing draws it.
- **A card's picture is a function of the card *and who is holding it*** *(2026-08-21)*. It was a
  function of the card alone for a week, and that was the bug: a slash in the hands of someone
  wearing Keen read "2x DMG" and dealt four times their DMG, because the card's multiplier and the
  ring's scaling are applied in different places. `screens.held` is the pairing — cost, DMG and
  worn rings, travelling together — and **the figure a ring has moved is written in the ring
  pink**, via `Spec.TextInk` and `Spec.TextHighlight`, which colours that run of the line and not
  the sentence around it: a pink verb would say the ring changed the card rather than the number. The
  *damage* is still not printed: the face carries the multiplier and the tooltip carries the
  arithmetic.
- **The wording is the constraint now, not the space.** The text column is ~128px — a dozen or
  so characters a line — so effect text has to be short words, and `DMG` rather than `damage`.
  `TestNoEffectTextWordIsWiderThanItsColumn` fails on a word that will not fit and
  `TestEveryCardTextFitsItsBand` on a string that wraps past the band;
  `TestLeftColumnDoesNotCollide` and `TestTheCostColumnStaysOutOfTheTextColumn` hold the column
  against its neighbours.
- **A `\n` in effect text is an authored line break** *(2026-08-23)*, honoured by
  `cards.WrapText` before the width is measured, and split back into lines by the tooltip.
  It exists because width-wrapping cannot make a *set* of cards break in the same place: the
  four elemental worms differ only in the element they name, and `FIRE` sits comfortably on
  the line where `LIGHTNING` all but fills it, so left to the measurer the four read as four
  layouts of one card. **A break can only ever add a line** — an authored line too wide for
  the band still wraps — so it is not a way past the column.
- **A glyph may be placed at a negative offset** to hang off an edge. `blitGlyph` clips it to
  the rounded silhouette via `insideRounded`, and `fadeRegion` skips transparent pixels — both
  so a corner glyph cannot fill in the transparent corner and square the card off.
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

**The sheet measures each glyph rather than assuming 64**, since they are not all one size,
and centres a small one on its own width so a row reads as different sizes rather than as
misalignment.

**The sheet draws each glyph twice: at `systems.CardGlyphScale` and enlarged.** The
actual-size row is the one that answers "can I read this" — reviewing only the enlarged row
is how a glyph comes to look acceptable in review and clunky in play. The scale constant
lives in `systems` precisely so the sheet reads the same number the card does and the preview
cannot drift from the game.

### Audio is generated too, and for the same reason

[internal/music](internal/music) plays the score. `assets/ascending.mid` is a **Standard
MIDI File — a kilobyte of notes** — and `internal/music` synthesises it to PCM once at
startup. `main.go` starts it after assets load; it loops for the whole session across
every screen. Editing the tune means editing the MIDI file. **Nothing is baked and there
is no build step.**

- **Ebitengine cannot play MIDI.** Its audio package decodes MP3, Ogg Vorbis and WAV,
  and that is all. The three ways past that were converting to Ogg offline, embedding a
  SoundFont plus a synthesiser, or generating the audio in Go.
- **The third was chosen for the glyph argument**: generated output has no provenance
  question. A SoundFont is megabytes with a licence to clear, and a rendered Ogg carries
  that same question inside it *invisibly* — which is worse, because the problem stops
  being visible in the diff. `oto` (Apache-2.0, first-party to Ebitengine) is the only
  dependency this added.
- **What it costs is fidelity.** This is an oscillator, so the score sounds like a
  chiptune. The current file is two synth basses and a drum part, so the distance is
  short — **a score wanting strings or a piano would not survive the trip, and that is
  the moment to revisit the decision rather than to keep bolting on oscillators.**
- **`smf.go` and `synth.go` may not import Ebitengine**, exactly like `internal/combat`.
  That is what makes them testable, and `music_test.go` pins the shape of the real file:
  85 notes, three channels, a 13-bar loop, and a render that is byte-identical twice.
  **Generated output fails quietly** — a synth handed a file it half-understands plays
  something, and what it plays is wrong in a way nobody notices.
- **No `math/rand`**, per the determinism rules. The drum noise is a 15-bit shift
  register seeded from each note's start frame, so two renders cannot differ.
- **Failing to open the audio device is logged, never fatal.** A machine with no sound
  card still plays the game — and `music.Available()` reports it, so the mute button
  disables itself rather than silently doing nothing.
- **Mute is a button, never a hotkey** — the input vocabulary has no keyboard. It lives in
  the game's chrome; see the section below. `SetMuted` takes the volume to zero rather than
  pausing, so the score keeps running underneath and unmuting lands where it would have got
  to rather than resuming a phrase already heard.
- **The game boots muted** — `music.muted` starts true and `Start` reads it when it opens the
  device, so the score is running but silent until the button is pressed. There is no settings
  screen to turn music off from, and music that begins on its own is the first thing a new
  player reaches for a control to stop.

### The frame: one control that belongs to no screen

[internal/game/chrome.go](internal/game/chrome.go) draws the **mute button** — a 44px
square in the bottom-left corner of every screen, carrying a generated speaker glyph.

- **It is deliberately outside "scenes own their own widgets" rather than an exception to
  it.** The score is started once in `main` and loops for the whole session across every
  screen, so the control that silences it belongs at the same level. The alternative was
  the same button on four scenes: four placements to keep in step and four callbacks into
  one package.
- **The bar for joining the frame is high, and the file says so.** Something true for the
  whole session, on every screen, owned by no scene. A frame is easy to grow by accident.
- **`state.ModalOpen` is what it cost.** A scene sets it while it has a dialog up and the
  chrome neither updates nor draws — otherwise the button would sit live on top of the deck
  overlay, whose whole design is that the control closing it is the only lit thing on
  screen. **The frame clears the flag each tick and the scene re-asserts it**, so leaving a
  screen with its overlay open cannot hide the chrome for the rest of the session.
- **Square and iconic because the corner is 52 pixels wide** on the combat screen — the hand
  band starts at x=52 and the action-point figure sits on its left edge, so a labelled
  button does not fit beside them.
- **`GlyphSound` and `GlyphMuted` are the only glyphs that are not about a card**, at a
  third size, 32px. They are generated for the same reason everything else is — no
  provenance question — and the muted one's bar is an `accent` rather than part of the
  silhouette, because a bar merged into a solid shape is only visible where it leaves it.
  The glyph says the **state**, not the action: a crossed speaker means the score is off.

### Cards: the left column carries the element, the border carries state

A card is a **constant off-white surface** (`cards.Surface`) with a **neutral grey border**, and
the element is said by **the left column — the tinted form mark and the cost ticks under it**
*(owner's call, 2026-08-23)*. The whole card is drawn by `internal/cards`. Five things follow and
are easy to re-break:

- **The border was the element from 2026-08-09 until the swap, and must not drift back.**
  `cards.borderBase` is what a border is actually drawn from and it returns the same grey for
  every element; `cards.BorderOf` still holds the element colours and is still what the mark, the
  deck panel's row labels and the arithmetic panel read. `TestTheBorderIsTheSameWhateverTheElement`
  fails if an element gets its border back. The argument for the swap: a border is the loudest
  thing on a card and it was naming the one fact the player already knows from the row the card is
  in, while the corner mark — the thing a hand is counted on — was hueless.
- **Ring keeps its pink border.** Pink was never an element; it is the "you cannot play this"
  signal, and `TestARingStillBordersPink` holds it against a change that neutralises the four.
- **The ticks are the element too, and share the border's state.** `Spec.atState` is the one
  switch carrying a colour from full strength to whatever the card's state wants — the border
  passes it the neutral grey and the ticks pass it the element, so selection, dragging and
  unaffordable move both together. A second copy of that switch is how a selected card ends up
  with a lit border and resting ticks; `TestTheTicksAndTheBorderShareOneState` fails on it.
- **The mark is tinted, not repainted.** `cards.tintInk` maps each pixel's own brightness onto a
  ramp between a dark and a light version of the element's colour, so the drawing keeps its
  outline and its bevel. A flat silhouette in the element colour throws away the interior detail
  that is the entire reason the form marks are drawn art rather than generated glyphs.
- **A near-white border on an off-white card is invisible.** `basic` is therefore a mid grey
  in `cards.BorderOf`, and a test fails if it is set to a near-white. It is also the colour every
  card's border now draws in.
- **`ColorAtStrength` is the wrong tool on a light card.** It scales toward *black*, which
  reads as quieter only against a dark ground. On
  an off-white card a border scaled down comes out darker than the surface and therefore
  *louder* than the live card beside it, which is how a pane's idle rows end up in front of
  its lit one. Use `systems.ColorToward(c, ground, pct)`, which moves a colour
  toward whatever it actually sits on. Card state is expressed as distance to the surface.
- **Cost is tick marks and the form is a corner mark**, not text and not a numeral. **The ticks
  are 16x4 as of 2026-08-23** *(owner's call)*, down from 13x8 — half the height and a quarter
  longer, so four of them stack in 31 pixels rather than 47 and the cost column ends higher up the
  face without its top edge moving. Every
  card in the game runs 1..3, the player's and every enemy's; a fourth tick grows the stack
  further down the card and is a layout change, not just a bigger number.
  `TestLeftColumnDoesNotCollide` fails rather than rendering it. **A card declares its own
  cost now** *(2026-08-16)*, so nothing stops a data file writing 5 — which is a reason to
  read this line before authoring one, not a reason for the renderer to clamp.

Rings reuse the whole format with a pink border and artwork instead of glyphs, and no cost
or category because a ring is neither played from a hand nor resolved in a round. **A ring card
names itself in one word to a line, and drops the word "Ring"** *(2026-08-21)* — the border, the
picture and the row it sits in all say "ring" already, so the noun costs the name its width and
says nothing. `data.RingData.FaceName` is the trim, `Style.NameWordPerLine` the break, and the
full name still titles every tooltip. Two lines is what the card has room for above its art;
`TestEveryRingNameFitsItsCard` fails on a ring named a word too long. **Most of
them have no artwork and draw `default-ring.png`** — `data.RingData.ArtKey` is the fallback and
`TestEveryRingDrawsSomething` fails on a key naming no file, so a blank face means art nobody
has painted rather than a name nobody spelled right.

**`go run ./tools/ringsheet` is how the catalogue gets looked at.** A run wears five and the
shelf offers three, so seeing forty-six in a launched game means playing to a shop over and
over. The sheet draws each with its price, its authored `Text` and its rules side by side —
which is also the only place the sentence a player reads can be checked against the rules that
actually fire.

**`tools/wormsheet` and `tools/handsheet` are the same idea on the other two catalogues**
*(2026-08-23)*. A worm is offered two at a time after a won fight, so the whole catalogue is five
fights away; the sheet draws all ten grouped by what each one changes about a card, with the
authored `Text` against the rule that fires, exactly as the ring sheet does. The hand sheet draws
every rung of the ladder as an *actual hand of real cards* — cheapest set the shipping deck can
form — ordered by ascending multiplier across all three axes at once, which is the comparison
`hands.json`'s axis-by-axis layout hides. **It does not sample**: the AP figure beside a rung is
what the cheapest copy costs once you hold the cards, and how often you hold them is
`tools/handodds`. Two tools reporting the same probability by different methods would be two
numbers that can disagree.

**`tools/enemysheet` and `tools/bosssheet` do it for the two opponent pools** *(2026-08-23)*. A
creature is met one at a time, three rooms to a floor, and its whole personality is a deck the
player only ever sees the played half of — so "is floor five dearer than floor four" was a
question answered by reading JSON. Each page groups its pool by floor, prints the band's HP, DMG
and AP spread in the heading, and draws every record as **one composite strip**: the opponent's
own card as the combat screen draws it, then its deck, one card per concept with the copy count
in the table under it.

- **A strip rather than a file per card**, because a file per card is about five hundred binaries
  rewritten on every full run against a hundred and twenty-six, for the same pixels. These two
  sheets are three quarters of the committed weight; see the note above about regenerating only
  what changed.
- **Both are one tool over two pools.** `tools/roster` holds the whole page and the two commands
  are a `Pool` plus a `main`, which is the deliberate exception to `tools/sheets`'s "no shared
  library" argument — the other four sheets share nothing, and these two are the *same sheet*
  read against each other. A boss sheet that had not learned about a new column would show the
  game as it was.
- **The deck size is read off `internal/decks`**, not added up from `Copies` in a template, and
  importing that package registers every concept at init — so a card naming a verb the rules do
  not have fails the sheet exactly as it fails a launch.

**It groups by rarity, and prints each tier's share of a shelf draw** *(2026-08-22)*. The tier is
the whole pricing decision — a ring is rebalanced by moving it, never by writing a number — so the
review question is "does any of these thirty commons belong a tier up", which an alphabetical list
cannot answer. The share is the tier's tickets over the catalogue's, to a tenth of a percent,
because a scarce tier rounds to `0%` and would read as unreachable.

### Colour: name one colour and scale it — but the rule is narrower than it reads

**This applies to widget *state*, not to widget *surfaces*.** A button naming crimson and
brightening toward it on press is the rule working. A button that can never have a lit top
edge and a shadowed bottom one is the rule overreaching, and it does currently overreach:
glyphs are written down as an exception when the only real problem is that a bevel needs more
than one value.

The intended direction: **buttons, cards and the resolution panes all want bevelling**, from
palettes like the ones in [glyphs.go](internal/systems/glyphs.go). When that lands, the rule
below should be rewritten as what it actually is — how a surface responds to hover, press and
disable — with the surface's own light and shade coming from a palette. Until then it governs
everything that has not been given one.

**And it assumes the thing being dimmed sits on a dark ground.** `ColorAtStrength` scales
toward black, so on a light surface it makes things louder rather than quieter — see the card
section above. `systems.ColorToward` is the light-ground counterpart and the two are not
interchangeable.

**The combat screen's ground is cream as of 2026-08-14** (`screens.screenGround`), so on that
screen `ColorAtStrength` is now the exception rather than the default, and reaching for it to
dim something drawn straight onto the table is a bug waiting to be seen. It still governs
buttons, because a button paints its own dark face and its label is white — that face is the
ground its states are scaled against, not the screen. Text written directly on the table takes
`screens.groundInk`.

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

### Two debug flags, and they are not interchangeable

`DebugPlacement` and `DebugGameplay` answer different questions and are wanted at different
times. Keep them separate.

- **`DebugPlacement`** — the grid, the rulers, the `Debug1`/`Debug2` scratch strings. About
  *where things are drawn*. Safe to leave on while playing, but off by default, so a change
  that needs the guides has to turn it on deliberately.
- **`DebugGameplay`** — perfect information, starting with the opponent's queued actions.
  About *what the player is allowed to know*. **Off by default**: with it on you are not
  playing the game, you are inspecting it, and it is easy to tune balance against a view no
  player will ever have. What it currently reveals is the combat screen's, and lives in the
  `combat-screen` skill.

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

### `internal/idle` is a fourth thing, and it is compiled out too

[internal/idle](internal/idle) closes the game after a stretch with nobody at the controls.

```powershell
go run -tags idleexit .                       # closes itself after two minutes idle
ASCEND_DUEL_IDLE_SECONDS=30 go run -tags idleexit .
go run .                                      # nothing: Tick is empty and always false
```

It exists so the game can be **launched unattended** — started to check a change, left to run,
and gone by itself rather than holding a window open for the rest of a session.

- **A build tag for the same reason as trace**, and the same two-file `_on`/`_off` shape. A
  game that quits on a player who steps away to make tea is a bug, so this must not be in a
  binary that ships. It has to stay deletable in one commit.
- **Everything is gated on window focus, cursor movement included.** That is the whole trick,
  not a nicety: an unattended run sits in the background while whoever launched it does
  something else, and a cursor crossing the desktop over an unfocused window would otherwise
  read as someone playing. The one case it exists for would be the one case it never fired in.
- It sets `ShouldClose` rather than returning `ErrClosing`, so the exit runs through the same
  path as the window's close button and there is only one way the game ends.
- **It may never change an outcome.** It closes a window; it does not touch a duel.

### `internal/scenario` is a fifth thing, and it is compiled out too

[internal/scenario](internal/scenario) plugs **a chosen set of rings, a chosen opening hand, a
chosen enemy — and a chosen screen** into a launched game.

```powershell
go run -tags scenario .                                        # the first entry in the file
ASCEND_DUEL_SCENARIO=seven-term-sum go run -tags scenario .    # a named one
go run .                                                       # nothing: every function is a zero value
```

It exists because an interaction between rings is currently a twenty-minute question. A ring is
bought from a shelf of three, a hand is dealt from a shuffled deck, and an enemy is whoever the
climb put in the room — so "does Echo actually multiply Enflamed's growth" cannot be *looked at*
without playing towards it. The rules are unit-tested; what no test can answer is what the
combination looks like on screen. It is the ring-and-hand counterpart of `deckSeedName` and
`session.StartingRings`, which each do one axis of the same job.

- **`scenarios.json` lives beside the package, not in `data/`.** Everything in `data/` is the
  game's own catalogue, loaded by every build. A scenario describes a thing being *tested*, and
  filing it with the cards would embed a debug fixture in a release binary.
- **A build tag for the reason trace and idle have one**, and the same two-file `_on`/`_off`
  shape. This hands the player a chosen hand and a chosen row of rings; it must not ship, and it
  has to stay deletable in one commit. The `//go:embed` is in the `_on` file, so an untagged build
  carries neither the fixture nor the reader.
- **It deliberately changes outcomes, unlike everything else that is compiled out.** `trace`,
  `idle`, the demo and both debug flags are views and may never alter a result. This is a
  *fixture* — which is exactly the argument for the build tag rather than a runtime flag.
- **Three call sites, each one guarded line**: `main` sets `session.StartingRings`, `Init` picks
  the enemy, `resetDeck` plugs the hand. Nothing else in the game knows the package exists.
- **The hand is dealt over the shuffle rather than through it.** The draw pile is untouched, so
  the second hand of the fight is a normal one and the fixture is only the opening.
- **A misspelled ring, card or enemy fails the launch**, at package init, before a window opens.
  A fixture that quietly tests something else is worse than a game that will not start.
- **It also opens the game on a named screen** *(owner's call, 2026-08-22)*: `"Screen": "reward"`
  or `"shop"`, with `Fight`, `Vitae` and `Life` saying what state to arrive in. A between-fights
  screen was otherwise a twenty-minute question — the reward screen's narration and the shop's
  shelf both needed a duel played to reach them, every time. It sets the run's *phase* and lets
  `screens/flow.go` decide the scene, so the run never disagrees with what is on screen.
  `reward-payout` and `shop-shelf` are the two entries.
- **Every entry carries a `Note` saying what question it answers**, printed at startup. A fixture
  whose purpose nobody remembers is a fixture that gets deleted.

## Architecture — and how to navigate it

**Every package's story lives in its own `doc.go`, and that is the navigation rubric.** This
section holds only what has to be true before you open a file: the graph, the loop, and the
tripwires. Everything else — what a package is for, what may never go in it, and the specific
thing that went wrong once — is a `go doc` away and sits beside the code it describes, so it is
edited in the same commit as the thing it explains.

```powershell
go doc ./internal/combat        # the package story
go doc ./internal/screens
go doc ./internal/session
```

**A file's own header comment sits *below* its `package` clause**, never above it — a comment
directly above `package X` is a second package comment, and `go doc` then shows whichever file the
toolchain reached first. `doc.go` is the only file whose comment goes above the clause.

### Where to read what

| Question | Read |
|---|---|
| what is this package for, what may never go in it | that package's `doc.go` |
| what does this file hold | the header comment under its `package` clause |
| what is the game supposed to *do* | [MECHANICS.md](MECHANICS.md) |
| what is left to build | [TODO.md](TODO.md) |
| how do I do X safely (git, data, rings, randomness, the combat screen) | the skill — see the index above |

### The dependency graph

**Generated from the real imports, not drawn from memory** *(2026-08-21)*. The picture that used
to be here had two arrows the code contradicted. Regenerate it rather than patch it:

```powershell
go list -f '{{.Name}}: {{join .Imports " "}}' ./... | grep curiousjc
```

| Package | imports, of ours |
|---|---|
| `seeds` `models` `assets` `idle` `trace` `music` | *nothing* |
| `data` | *nothing* |
| `scenario` | data, combat *(compiled out unless `-tags scenario`)* |
| `pyramid` | data |
| `combat` | data |
| `decks` | data, combat |
| `entities` | data, combat, pyramid |
| `session` | data, combat, pyramid, seeds |
| `state` | data, session |
| `systems` | assets, models, state |
| `cards` | systems |
| `actions` | state |
| `screens` | all of the above, plus `scenario` |
| `game` | screens, state, systems, models, music, idle, trace |
| `main` | game, session, assets, data, music |

Six facts about it that are load-bearing:

- **`data` is the bottom and must never import upward.** That is why enemy concepts are
  registered by `internal/decks` rather than handed over by `data`: enemy cards live in
  `enemies.json` beside portraits and floor bands, so the rules reading that file directly would
  cross the who-consumes-it line.
- **`seeds` imports nothing and `combat` deliberately does not import it.** The rules take an
  injected `*rand.Rand` and stay ignorant of where it came from.
- **`decks` sits above `combat` and `data` and below `screens`**, which is the whole reason it is
  a package: it is the one place allowed to turn a JSON card list into rules types, reachable
  without importing a screen. `pyramid` exists for the same reason on the other axis — the climb is
  arithmetic a headless caller needs and a screen must not own.
- **`state` importing `session` is the one documented bend**, and it is what makes `state` reach
  `combat` transitively. The rule it bends was written to stop *screen* state leaking into global
  state; a run is not screen state.
- **`cards` importing `systems` is the edge that surprises people.** A card draws generated
  glyphs, so the renderer needs the generator. Neither creates an `*ebiten.Image`, which is the
  property that actually matters — it is what lets `tools/cardsheet` render with no window.
- **Nothing above `screens` knows a scene exists except `game`**, which holds the registry. That
  is what makes adding a screen a local change.

### The game loop, in code

`main.go` builds the `game.Game`, loads assets/fonts/data once, builds the run, then hands control
to `ebiten.RunGame`. It does **not** wire up widgets — scenes build their own, and the one control
belonging to no scene is built by `game` itself.

`internal/game` then drives `Update` / `Draw` / `Layout` at a fixed 1280x960 internal resolution,
picking the active scene out of one registry. `internal/screens/scene.go` is the `Scene` contract;
`Init` may run more than once, because a screen can be re-entered.

### The run loop, in play

**The run owns where it is, and one file moves it on.** A scene that has finished calls
`screens.advance`; nothing names its successor.

```
fight  →  reward  →  shop  →  choice  →  fight ...
```

- **`session.Phase` is the station** — see `internal/session/flow.go`, which holds the order.
- **`screens.phaseScreens` is which scene draws it** — see `internal/screens/flow.go`. A phase with
  no scene registered is walked past rather than drawn blank, which is why the loop could name the
  shop and the room choice before either existed. The shop landed on 2026-08-21 and cost exactly
  the three edits below; the room choice is still walked past.
- **Adding a screen is therefore three edits**: a phase in `session/flow.go`, an entry in
  `screens/flow.go`, and an entry in the registry in `internal/game`. No existing scene changes.

### The three rules a change most often breaks

- **`internal/combat` decides rounds; a screen only replays them.** Never change a rule to make a
  screen look right — say which of the two is wrong and let the owner decide. That is a
  game-design call and it ripples into the tests and the balance.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before
  playback begins, so animation speed, a game-speed setting, a dialog that pauses the cursor, the
  debug flags, `internal/trace`, `internal/idle` and the scripted demo may all alter pacing and
  none of them may alter results.
- **Working state belongs to the narrowest thing that needs it.** One screen reads it → the scene.
  It has to outlive a fight → `internal/session`. Every screen genuinely needs it →
  `state.GlobalState`. Nothing else earns a place in global state.

### What is *not* in a package doc, because it is a tripwire

- **Never call the `math/rand` package-level functions.** See the `randomness` skill.
- **A new `EventKind` needs a choreography entry**, or `internal/screens` fails a test. An event
  with no picture and an event whose picture was forgotten otherwise look identical.
- **`ConceptID`, `StatusID`, `GlyphKind` and `Element` are append-only**, and none may be
  serialized. Arrays and caches are indexed by the ordinal, so inserting one mid-enum silently
  re-points everything already stored.
- **Re-run `tools/handodds` and `tools/seeds` after touching the deck.** Both measure facts about
  one particular deck, and nothing fails when they go stale.

### Drawing idioms

- Sprites are drawn via `colorm.DrawImage` so a `colorm.ColorM` can tint/hue-shift them; buttons and shapes use `vector.DrawFilled*` into a scratch `ebiten.NewImage`.
- Positioning convention: translate by `-w/2, -h/2` first to center the origin, then translate to the target coordinate. Buttons store `ScreenX`/`ScreenY` as their *center*, and both `UpdateButton` (hit testing) and `DrawButton` re-derive the top-left from it.
- **Rounded rectangles are done two ways, and that is a known inconsistency.** Health bars
  draw an opaque mask and composite it with `ebiten.BlendSourceIn` — see `DrawHealthBar`
  and `CreateRoundedRecMask` in [combat_hud.go](internal/screens/combat_hud.go). **Cards
  cannot use that**: it takes an `*ebiten.Image`, its body is `vector.DrawFilledCircle`,
  and `BlendSourceIn` is a GPU blend mode, none of which exist without a graphics context —
  and `internal/cards` must render without one so the review tool can call it. Cards
  therefore rasterise their corners in plain Go (`cards/shape.go`), hard-edged, because the
  glyphs on them are 1:1 pixel art. Migrating health bars onto that path would collapse the
  two; it has not been done.

## Art

**`assets/` is grouped by what a file is for**: `game/` (fonts, title screens), `enemy/`,
`ring/`, `effect/`, `sounds/`. The `//go:embed` paths are relative to `embed.go`, so
refiling something is one line there and nothing anywhere else.

**A map key is not tied to a file path.** Keys are the lookup names used across the game and
`data/*.json` writes them down; tying one to a path would mean a data migration every time a
file was refiled. A named asset is three edits: the file, an `//go:embed` var, and a map entry.

**The enemy portraits are the exception, and they are a *family* rather than named
assets.** There are 96 of them, so `//go:embed enemy/*-portrait.png` pulls
the directory in as an `embed.FS` and `LoadImageData` walks it, keying each by filename
stem — `enemy/ogrewarlord-portrait.png` is `ogrewarlord-portrait`, which is what
`data/enemies.json` writes in its `Portrait` field. **The consequence is exactly what the
three-edit rule protects against: a portrait's key is tied to its filename**, so renaming
one means editing the JSON. That is the price of not hand-maintaining 192 lines nobody
could review. Reach for the glob only when a *set* of files is being added; a one-off asset
still gets its own var.

**The thirty boss portraits are a second family, in `assets/boss/`**, globbed the same way and
keyed by stem — so `boss/bayaz-boss.png` is `bayaz-boss`, which is what `data/bosses.json` writes.
The `-boss` suffix is load-bearing: both families land in one flat map, and a boss whose key
collided with a creature's would silently draw that creature.
`TestNoBossPortraitIsAnEnemyPortrait` fails on it.

They are handed out as **bytes, not `*ebiten.Image`** — they are drawn into a card by
`internal/cards`, which has no graphics context, and decoding 96 at startup would cost
~20 MB of resident memory for pictures most runs never show.

**`assets/effect/` is the status badges**, drawn as a centred row along the bottom of the enemy
card by `internal/cards` — so they go through `LoadImageData` as bytes, exactly like the ring art
and for the same reason. `effectKeys` in `internal/screens/card_art.go` maps an element to its
badge; `default-effect.png` is the fallback, and `TestEveryStatusElementHasABadge` fails rather
than letting a shipped element quietly draw it. **The table is keyed by element and not read off
a ring**, because a badge belongs to the status: a status arriving by an affix or a boss rule has
to draw the same picture.

**Nothing in the game draws a loose sprite.** There are no creature sprites in `assets/`;
`Combatant` has no `Sprite` field and `entities` imports no Ebitengine at all. **Both duelists
are cards**, in opposite corners, and both state their life the same way — a bar over a
fraction, at identical offsets on the two styles so the pair can be compared across the
screen without measuring. **The enemy's carries a badge row under its fraction and the
player's does not**, which is not a break of that rule: an enemy wears no rings, so nothing can
put a status on the player to draw.

**The full animation sheets stay in `.scratch/flat-creatures`** (gitignored) — that folder's
README documents the grid, the frame table and the facing order — so animating enemies later
means going back for them rather than starting over.
