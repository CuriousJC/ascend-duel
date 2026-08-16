# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Ascending Duel — a roguelike where you duel your way up a tower, collecting rings, brands of power, and pets. Written in Go with [Ebitengine v2](https://ebitengine.org/) (`github.com/hajimehoshi/ebiten/v2`). Module path: `github.com/curiousjc/ascend-duel`.

## Where things are written down

Five streams, each with one job. Reach for the right one rather than searching all of them.

| Stream | File | Read it when |
|---|---|---|
| **How to work** | `CLAUDE.md` — this file | always; it is loaded every session |
| **Procedure** | `.claude/skills/*/SKILL.md` | on trigger — git and GitHub work, the combat screen |
| **What the game *is*** | [MECHANICS.md](MECHANICS.md) | designing or implementing any mechanic, before proposing a design change |
| **What to build next** | [TODO.md](TODO.md) | picking up work |
| **Unfiltered** | [ideas.md](ideas.md) | the inbox; entries get promoted into MECHANICS or TODO and struck from here |

- **`MECHANICS.md` is the design record.** Decided unless marked `[?]`. It holds the element
  set and their statuses, cards and types, combos, rings, brands, vitae, the tower, enemies,
  and the phase-based resolution experiment.
- **`TODO.md` is open work only.** Completed entries are deleted rather than archived, so it
  says what is left, not what happened. Prefer `MECHANICS.md` for "what should this do".
- When the two disagree, `MECHANICS.md` is newer and wins — say so rather than guessing.
- **Cut means deleted, not tombstoned.** When something is taken out of the design, remove
  every trace of it rather than leaving a note saying it was removed and why. These files are
  loaded into context; a record of things that do not exist is a running cost paid on every
  session, and it grows without bound because nothing ever retires a tombstone. Git history is
  the record of what used to be true. **If a removal genuinely needs to stay visible — because
  the code still has a shape only the dead mechanic explains, or the same idea keeps being
  re-proposed — ask before writing it down**, rather than deciding alone that it earns a line.

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
go run ./tools/balance      # what all 96 enemies do to the fighter, one line each
go run ./tools/balance -v OgreWarlord   # one enemy, round by round
go run ./tools/glyphsheet   # regenerate the committed glyph contact sheet
go run ./tools/cardsheet    # every card variation to PNGs + an HTML page, then refresh the tab
go run ./tools/seeds        # re-check the named deck seeds, and search for new ones
```

**A seed is an opening hand**, because the shuffle is deterministic. `internal/screens/seeds.go`
holds a catalogue of named seeds — `strike-flurry`, `strike-barrage`, `all-plans` — so a
hand that demonstrates something can be asked for by name instead of found by relaunching.
`deckSeedName` picks which one a launch deals.

**Re-run `tools/seeds` after touching `data/duelist_cards.json`, `startingDeck` or `handSize`.** A seed is
a fact about one particular deck; change the deck and every catalogued number silently deals
something else. The tool re-checks the catalogue before it searches and says which entries no
longer match — a change to the deck size has invalidated every entry at once before. A demo
testing a Flurry against a hand with two Strikes in it is worse than no demo, because it passes.

**A rarer hand needs a bigger search, and some hands are impossible.** `strike-barrage` is four of
the four Strikes in a hand of eight from 48 cards and turns up around seed 900; the default 20,000
finds it. **A hand wanting five copies of a concept cannot be dealt at all**, since no attack card
exists more than four times — so check the arithmetic before concluding either way. A hand the tool
reports as unfindable usually means the search was too short, but not always.

**Three build tags, and they compose.** Each selects a different file in its package, so one
configuration can compile while another does not. Vet and build every one you might have
broken:

```powershell
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...; go vet -tags demoplay ./...
go run -tags "debugtrace idleexit" .    # traced and self-closing: the unattended run
```

**`demoplay` is how the combat screen gets looked at without anybody sitting at it.** It plays
a scripted round or two — selection, DUEL!, playback — and writes the screen to `demo/*.png`,
then closes. It exists because the screen is the one thing `go test` cannot check and
`tools/balance` cannot either: a combo line, a marked verb, a highlight on the right row are all
things you have to *see*. It is the `tools/glyphsheet` idea applied to a live screen, and the
same rule applies — a stale picture is worse than none, so regenerate rather than trust an old
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
`DISPLAY` — before a single test function runs. Three of the four tested packages link it:
`internal/screens` directly, and `internal/cards` and `internal/music` because their tests
import `assets`, which hands back `*ebiten.Image`. Only `internal/combat` is genuinely
clean. Both workflows therefore run the Linux test step under `xvfb-run -a`, which supplies
a throwaway X server nothing ever draws to. Windows is unaffected — Ebitengine is pure Go
there. **If a package's tests start importing `assets`, they have joined that group**; the
package's own no-Ebitengine rule still holds and is still worth holding, but it no longer
buys a display-free test run.

**What cannot be unit-tested gets a tool instead.** `internal/screens` needs a window, so
anything it decides is checked by launching the game; anything the *rules* decide about
balance is checked by `tools/balance`, which plays whole duels through the real
`ResolveRound` and prints who wins. It exists because an unwinnable enemy shipped and was
invisible — losing slowly looks exactly like losing to bad draws. Run it after touching a
cost, a stat line, or a planner.

**It is a sample rather than an exact answer as of 2026-08-14**, because combat rolls for
lightning. It seeds one fixed source (`balanceSeed`) per run so a result is reproducible, but a
posture that wins half its duels and one that wins all of them currently print the same line.
**Read a single run as one draw**, and treat multi-sample reporting as the next thing the tool
needs before its numbers can be tuned against.

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

## Determinism — a planned feature that constrains code written now

Runs will eventually be **replayable from a seed**: the same tower, enemies and rolls,
so the player can retry and make different choices. **Combat is stochastic as of 2026-08-14** —
lightning rolls — which is exactly the case these rules were written to survive, so follow them
rather than treating the first roll as permission for the second.

- **Never call the `math/rand` package-level functions** (`rand.Intn`, `rand.Float64`,
  `rand.Shuffle`, …). They draw from a global source shared with every other caller,
  which makes a run unreproducible. Randomness comes from an explicit `*rand.Rand`
  carried on state and seeded once per run.
- **Six separate streams: enemy selection, loot offers, floor offers, the combat roll, and
  *two* card shuffles — the player's and the opponent's.** Never share one source between them, or a
  change to loot generation silently rerolls every enemy in the tower. A stream is only
  ever advanced by its own concern. Tower layout is fixed (8 floors × 3 fights, endless
  later) and draws no randomness.
- **The two card shuffles are separate, and must stay separate.** Sharing would make the
  player's opening hand a function of how many cards the opponent happened to draw, so
  **every entry in `seeds.go` would break the first time an enemy deck was retuned** — and a
  named hand has to stay a fact about the player's deck alone. That is the shape of argument
  to apply if a seventh stream is proposed: ask what it would silently reroll.
- **Both shuffles exist, and both read the run seed as of 2026-08-15.** The player's lives on
  `CombatScene` as `rng`; the opponent's lives on `decks.EnemyPile`. `CombatScene.shuffleSeeds`
  is the one place either is chosen: `RunSeed ^ playerDeckSalt ^ fight` and
  `RunSeed ^ enemyDeckSalt ^ fight`, where `fight` is `(fightIndex+1) * fightStride`. So every
  fight deals fresh cards, and a defeat and a retry deal that fight again rather than a new
  one — the same property the enemy roster has, and for the same reason: nothing re-rolls a run
  until `Session` exists.
- **`deckSeed` is the pin, and pinning is all-or-nothing.** Non-zero fixes the player's shuffle
  to a catalogued hand and the opponent's to `decks.EnemySeed`; zero — the default, written as
  an empty `deckSeedName` — rolls both. It pins both sides because half a reproducible duel is
  worse than none: the hand looks right and the fight still differs. `tools/balance` and the
  `demoplay` build are the callers that want the pin.
- **The per-run seed is `GlobalState.RunSeed`.** `main` sets it once — from `fixedRunSeed` if
  that constant is non-zero, otherwise from the clock — and logs it. **Reading the clock there
  is not a breach of "no `time.Now()` in game rules"**: choosing a seed is the one place a run
  is allowed to be unpredictable, and it happens once, outside the rules, in main.
  `fixedRunSeed` is the debugging toggle, the counterpart of `deckSeed`. **Four streams are
  live off it** — enemy selection (`RunSeed ^ enemySelectSalt`, shuffled *within each floor
  band* so a run opens on a different opponent without a floor-eight enemy ever being fight
  one), the combat roll, and the two card shuffles. A consumer that starts reading `RunSeed`
  must salt its own source, never share one.
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
- **`internal/combat` has no clock and exactly one roll.** It is otherwise integer arithmetic,
  and `TestRoundIsDeterministic` pins that a nil source resolves identically every time.
- **The one roll is lightning, and it arrived the way this section requires.** A shock is a 25%
  chance per stack that the turn's attack misses, capped at 75%; the source is a `*rand.Rand`
  parameter on `ResolveRound` and a nil one means no rolls. **This reverses the deterministic
  version taken two days earlier**, because one blow per turn turned a certain miss into a 1 AP
  card deleting a 10 AP hand. `MECHANICS.md` holds the argument.
- **What it cost is exactly what this file predicted it would**, and both are now paid:
  `tools/balance` is a distribution rather than an exact answer, and the stream advances per
  attack phase, so a change early in a duel reshuffles every roll after it.
- **Still rewrite a random-sounding rule rather than let it in.** Lightning is the deliberate
  exception, not the precedent — it was taken because unreliability is what lightning *is*, and
  the alternatives (breaking the hand, cutting the multiplier) were weighed and written down.
  **Certainty is often the better game as well as the cheaper code**, and it matches the rule
  combos otherwise follow, that what you committed to cannot be silently undone. A second roll
  needs the same argument made again from scratch.
- **Presentation may never change outcomes.** `ResolveRound` decides a whole round
  before playback begins, so animation speed, the planned game-speed setting, and any
  skip button are free to alter pacing and must not alter results.

## The combat screen — see the `combat-screen` skill

Its layout, its card and action-box widget, its hidden information, and the resolution-order
rule the screen has to obey all live in
[.claude/skills/combat-screen/SKILL.md](.claude/skills/combat-screen/SKILL.md). **Load it
before touching any of the combat screen's files — `internal/screens/combat.go`,
`combat_deck.go`, `combat_panes.go`, `combat_hud.go`, `combat_actionbox.go` — or
`internal/combat`, or anything about how a round is drawn or played back.**

It is a skill because it is the screen under active construction: it grows every session
while mattering only when that screen is the work. The general UI conventions below still
apply to it and stay here.

Two things worth knowing without opening it:

- **`internal/combat` decides rounds; the screen only replays them.** Never change the rules
  to make a screen look right — say so and let the owner decide which one is wrong.
- **`combat.ResolutionOrder` is the single authority on play order**, and both `ResolveRound`
  and the Resolution pane read it rather than deriving their own.

## UI: clicks and drag-and-drop only

A firm design decision, not a current limitation. These apply everywhere, the combat screen
included. The entire input vocabulary is:

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
- **ebitenui was evaluated and declined.** Everything the game needs is a *game* widget,
  which is where general-purpose toolkits are weakest, and a toolkit is one more dependency
  to licence-check against a product that will be sold. **The one trigger for revisiting is
  the seed text field** — a text input with a caret, selection and clipboard is the single
  widget genuinely cheaper to take than to build.

The action box is a *game* widget, not a UI widget: draggable action cards with live
action-point validation. General-purpose toolkits are weakest at exactly that, so
hand-rolling costs little and buys full control.

### Glyphs are generated, not drawn — and not scaled from one colour

[internal/systems/glyphs.go](internal/systems/glyphs.go) generates the pixel-art
glyphs on the action cards, drawn at 1:1 — 64x64 for the damage sword and the runner, 32x32
for the three category glyphs. **Generated art has no provenance question**, which is the
whole reason to prefer this pattern for interface art in a game that will be sold.

It is a **generator, not a bitmap**. A glyph is a filled silhouette described by horizontal
spans; the rim is *derived* by asking which filled pixels touch empty space, and the
interior shading is *computed* from where a pixel sits across its row and down the sprite.
Nothing is hand-placed, so a shape can be nudged without repainting it.

- **Nothing in a silhouette may be thinner than about five pixels.** The derived rim takes
  one pixel off each side, so a three-pixel crossguard renders as two rows of outline
  around one row of metal and reads as a scratch. This is the main constraint the technique
  imposes and it drives every span in the file.
- **Nothing on a card is a glyph as of 2026-08-15.** The three category glyphs said which phase a
  card resolved in, and the deck rework replaced that fact with a *family* — stab, slash, crush,
  plan — which has no art yet. A card's corner therefore carries an **uppercase letter**: S, **D**
  for slash (Stab took the S), C, P. `cards.Family.glyph()` is the seat the art goes back into and
  returns nothing for every family today; `internal/systems` is untouched and the glyph sheet
  still renders everything. The paragraphs below describe the generator, which is still the
  pattern for interface art and is still what the mute button uses.
- **Glyphs are the deliberate exception to the colour rule below.** They carry a five-value
  `Palette` — outline, specular, highlight, mid, shade, accent — because a bevel cannot be
  made from one colour scaled down. They are drawn untinted; a disabled card dims them by
  *alpha*, so the shading survives and only the weight changes. Tinting one toward the card
  colour would collapse it back to a flat silhouette.
- **Every glyph uses one hueless palette, `white`, and they should stay that way.** Colour
  means "element", and the element is carried by the card's *border* (see the colour section
  below). A coloured glyph on a coloured card says it twice and leaves nothing for the next
  distinction.
- **A hueless glyph on an off-white card loses most of its bevel, and that is accepted.**
  `Specular` is pure white and `Highlight` is `{232,236,242}`, so against the off-white
  surface the lit side of a bevel largely disappears and a glyph reads as outline plus
  shading. The near-black `Outline` carries legibility, so nothing is broken — but the
  five-value palette is mostly spent, and `cards.Surface` is one constant if that ever
  needs re-testing.
- **Glyphs are not all one size, and `systems.SizeOf(kind)` is the authority.** The damage
  sword and the runner are 64; the retired category glyphs were **22**. Never assume
  `GlyphSize` at a call site — a small glyph centred in a 64-pixel hole is the failure. **The
  card is now the exception and says so**: `Style.FamilySize` names the box, because a letter has
  no intrinsic size and centring by ink is what makes it fill one.
- **A glyph cannot be resized, so a smaller one is a *different drawing*.** `CardGlyphScale`
  is 1 and integer-only: the rim is derived one pixel thick, so a third-size copy of a 64px
  shape is a third-size copy of its outline with nothing inside. The 22px category glyphs
  were authored at 22, and at that size the detail budget is nearly nothing — a 5px feature
  is two pixels of rim around three of fill. **They are told apart by proportion before any
  detail is legible**: the sword narrow and vertical, the shield wide-shouldered, the book
  wider than tall. That is the design constraint to work with, not around.
- **`GlyphKind` is append-only.** The glyph cache keys on the ordinal, so inserting a kind
  mid-enum silently re-points every existing entry — the same hazard `MECHANICS.md` records
  for the concept enum and its combo IDs.
- **The card is 162x224, and it is a column and a paragraph.** The family mark sits in a 32px
  box at (10,8) — **inside the card, not hanging off the corner**, because a clipped letter
  reads as a rendering fault where a clipped silhouette reads as itself; under it the cost dashes
  make a **26px column**; and **the effect text takes everything right of that**, centred in it
  both ways, at 18pt. `blitGlyph` still clips to the rounded shape, which is what a future glyph
  will want back.
- **There is no damage badge at all.** The 64px generated sword went first — it said what the
  corner mark already says — and then the bare figure, because the text states what the card
  deals and a number beside it was the same fact multiplied out by the wielder's Strength.
  `cards.Spec` has no `Damage` field and `drawCard` takes no Strength, so **a card's picture is
  a function of the card alone**. `systems.GlyphDamage` still exists and is still on the glyph
  sheet; nothing draws it.
- **The wording is the constraint now, not the space.** The text column is ~128px — a dozen or
  so characters a line — so effect text has to be short words, and `DMG` rather than `damage`.
  `TestNoEffectTextWordIsWiderThanItsColumn` fails on a word that will not fit and
  `TestEveryCardTextFitsItsBand` on a string that wraps past the band;
  `TestLeftColumnDoesNotCollide` and `TestTheCostColumnStaysOutOfTheTextColumn` hold the column
  against its neighbours.
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

### Cards: the border carries the element, not the surface

A card is a **constant off-white surface** (`cards.Surface`) with a **thick coloured border**
carrying the element, and the whole card is drawn by `internal/cards`. Three things follow
and are easy to re-break:

- **A near-white border on an off-white card is invisible.** `basic` is therefore a mid grey
  in `cards.BorderOf`, and a test fails if it is set to a near-white.
- **`ColorAtStrength` is the wrong tool on a light card.** It scales toward *black*, which
  reads as quieter only against a dark ground. On
  an off-white card a border scaled down comes out darker than the surface and therefore
  *louder* than the live card beside it, which is how a pane's idle rows end up in front of
  its lit one. Use `systems.ColorToward(c, ground, pct)`, which moves a colour
  toward whatever it actually sits on. Card state is expressed as distance to the surface.
- **Cost is dash marks and the family is a corner mark**, not text and not a numeral. Costs run
  1..4 — the attack ladder uses 1..3 and the two defences sit at 4; a fifth tier grows the
  dash stack further down the card and is a layout change, not just a bigger number.
  `TestLeftColumnDoesNotCollide` fails rather than rendering it.

Rings reuse the whole format with a pink border and artwork instead of glyphs, and no cost
or category because a ring is neither played from a hand nor resolved in a round. Nothing
in the game builds one yet — `tools/cardsheet` is the only place they exist.

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

## Architecture

### Ebitengine game loop

`main.go` builds the `game.Game`, loads assets/fonts/data once, then hands control to `ebiten.RunGame`. It does **not** wire up widgets — scenes build their own, and the one control that belongs to no scene is built by `game` itself (see the frame, below). Ebitengine then drives three methods on [game.go](internal/game/game.go):

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

- `assets/` — `//go:embed`s every image and font into the binary and exposes `LoadAssets()` / `LoadFonts()`, returning `map[string]*ebiten.Image` and `map[string]*text.GoTextFaceSource`. **A new asset needs three edits: the file, an `//go:embed` var, and a map entry in the loader.** The map key is the lookup name used everywhere else (e.g. `gs.Assets["giantrat_png"]`, `gs.Fonts["kubasta"]`), and it is **independent of where the file sits** — see the Art section. There are two extra loaders for callers that cannot take an Ebitengine type: `LoadFontData()` and `LoadImageData()` hand back raw bytes, because `internal/cards` and `tools/cardsheet` render without a graphics context. `gs.FontData` carries the fonts for exactly that reason.
- `data/` — JSON next to a small Go loader, which is the pattern for all static game data. Six files:

  | File | Loader | Holds |
  |---|---|---|
  | `duelists.json` | `LoadDuelists` | who the player can be, and their card back |
  | `enemies.json` | `LoadEnemies` | 96 opponents: stats, style, portrait, valid floors |
  | `duelist_cards.json` | `LoadDuelistCards` | the player's deck list |
  | `enemy_cards.json` | `LoadEnemyCards` | what an opponent draws from |
  | `rings.json` | `LoadRings` | the rings that exist: name, art key, element, one line of text |
  | `combos.json` | `LoadCombos` | the two combo axes: six hands, five element mixes |

  **`combos.json` is the only one the rules read.** `internal/combat` imports `data` for it and for nothing else; the other five are consumed by `screens`, `decks` or `entities`. `data` holds the shape and `combat` holds the meaning — the file names a category as a string and the rules resolve it with `ParseCategory`, the same division `CheckCostTiers` draws for the deck lists. A catalogue naming something the rules do not have panics at init rather than loading a combo that can never fire.

  **`rings.json` has its first rule as of 2026-08-16, and it is the statuses.** A ring is what
  makes its element do anything: an attack applies a status only if its owner wears that
  element's ring, so an unringed fire Strike is a plain Strike with a red border. `Element` is
  the field that carries it — parsed in `internal/screens` with `combat.ParseElement` and set as
  a flag on `combat.Duelist.Rings`, because `internal/combat` may not read this file. A name the
  rules do not have is logged rather than dropped.

  **What is worn is `startingRings` in `internal/screens`, not the file.** Four rings exist and
  the player starts in three — earth is left off so a launch tests the gate as well as the
  statuses. The row used to draw every record up to the cap, which would have equipped the fourth
  ring the moment the file gained one. It is temporary and is the counterpart of `deckSeedName`:
  buying and equipping needs `Session`. **Do not grow a rules vocabulary in this file ahead of
  the rules** — the discount and the flip are still unwritten and `Card.Cost()` is still the seat
  the discount sits in. `data.RingOrder` is the sorted walk, per the determinism rules.

  **The duelists and the enemies are separate files because their fields do not overlap** — an enemy has a plan style, a portrait and an affix pool; a duelist has a card back and, eventually, a deck. One struct would make every field optional and none of them mean anything.

  `PlanStyle` names how an enemy fights and is parsed by `combat.ParsePlanStyle`, falling back to brute — **enemy behaviour is data, so the roster is tunable without touching Go.** `ValidFloors` is `[lowest, highest]` against the planned 8-floor tower, so a Dragon is not on floor one; nothing generates floors yet, so today it only sorts the fight order. `data.EnemyOrder` is the sorted walk — never range the map, per the determinism rules.

  - **The two card lists are one shape, tuned separately.** Concepts, the family and category each declares, the elements each ships in, and how many copies. `startingDeck` is built from the duelist list, so **deck size is a consequence of a file you can read** — 9 attacks × 4 colours plus 3 plans × 4 copies = **48**. **No player card is drab except the plans** *(2026-08-15)*: attacks are always coloured, and the plans are basic because nothing they do is elemental. One consequence worth knowing before touching the combo table — four copies is the ceiling, so a Barrage is the top of the hand ladder and is always a Rainbow. Enemy cards are `Attack` and `Heavy` and nothing else, all `basic` — not because the colour is thrown away (it is read and carried) but because MECHANICS.md has affixes *transforming* a basic deck into an element, so a colour typed into that file would pre-empt a mechanic that does not exist. **Every plan style therefore collapses to brute**, since the warden asks for a Defend by name and the tactician for a Prepare.
  - **Cost, category, family and damage are deliberately *not* in either**; they are rules and live in `internal/combat`, which cannot import `data`. **So a separate enemy file cannot yet give an enemy Strike a different cost** — that is a rules change, not a data one. The JSON's `CostTier`, `Category` and `Family` are documentation *with a check*: `data.CheckCostTiers` asserts all three against `ActionKind.Cost()`/`.Category()`/`.Family()` and both deck builders **panic at package init** on any disagreement. A deck quietly five cards short is a balance change nobody made, so it fails on launch instead. Concept names are joined to the rules by `combat.ParseAction`, which reports failure rather than falling back.
- `internal/decks/` — **the opponent's deck, and the only package between `data` and `internal/combat`**. It exists so the combat screen and `tools/balance` share one enemy deck: the balance tool plays whole duels headlessly and cannot import `internal/screens`, which links Ebitengine. **No Ebitengine here, ever**, for that reason. `EnemyPile` is the three piles plus a shuffle; **the enemy's hand does not persist between rounds**, unlike the player's, because a style only takes attacks plus a Defend or Prepare and everything else would accumulate until the hand locked up — which it did. The player's hand may persist because Discard exists, and that is the lever an enemy has not got.
- `internal/models/` — plain data structs with no behaviour (`Button`). Constructors only.
- `internal/systems/` — the behaviour for models, split as `Update*` and `Draw*` free functions taking `(gs, ...)`. `models.Button` + `systems.UpdateButton`/`DrawButton` is the reference example of this model/system split; follow it for new widgets.
- `internal/entities/` — game-world actors (`Combatant`, embedding `combat.Duelist`), hydrated from `data` records at scene init.
- `internal/idle/` — the unattended-run timer, behind the `idleexit` build tag. Two files, `_on`/`_off`, exactly like `internal/trace`.
- `internal/cards/` — **draws an action card into a plain Go image, and creates no
  Ebitengine images.** Same pattern and same reason as `systems.RenderGlyph`: it is what
  lets `tools/cardsheet` render a card with no window. `Spec` is plain data (name,
  category, cost, damage, element, optional artwork, state) rather than a
  `combat.ActionKind`, so the sheet can draw combinations the rules cannot produce — a
  border colour nothing uses, a ring. `Style` is the geometry: `Hand`, `Mini` (half size,
  the deck overlay), `Stack` (the draw pile's back), `RingStyle`, and the two fighter cards
  `EnemyStyle` and `DuelistStyle` — twins, and the health bar has to stay at the same offsets
  on both. **`Spec` must stay comparable**, because the screen's cache keys on the whole
  struct; that is why `Stats` is a fixed array. Text is set with `golang.org/x/image` because
  Ebitengine's `text/v2` needs an `*ebiten.Image`; both the game and the tool go through
  this, so the sheet cannot drift from what is drawn. `internal/screens/card_art.go` is the
  bridge and holds the cache — rendering writes every pixel in Go and is far too slow to do
  per frame.
- `internal/music/` — the score, **synthesised at startup from a MIDI file**. See the section below; the short version is that `smf.go` and `synth.go` are pure arithmetic and tested, and only `music.go` touches Ebitengine's audio.
- `internal/screens/combat_demo_{on,off}.go` — the scripted-demo driver, behind `demoplay`. Same two-file shape, and it lives beside the screen it drives rather than in a package of its own because it reaches into that screen's own methods (`toggle`, `startRound`). It holds its script in package state so `combat.go` gains only two call sites. **It may never change an outcome**, the same constraint as trace and idle.
- `internal/combat/` — the duel rules, **the elements and their statuses, the opponent's planners, and the combo table**. **No Ebitengine import, ever.** **`combat.Card` is a concept plus an element and is the unit the whole package deals in** — `[]Card` through `ResolveRound`, `ResolutionOrder`, `Slot`, `PlanFor` and `CostOf`, which is what let the screen's own `element` type and card struct be deleted rather than mapped. `status.go` holds the four statuses; they share one lifecycle on purpose and `Duelist.Statuses` is an array indexed by element, which makes **`Element` append-only** the same way `ActionKind` and `GlyphKind` are. **A status only happens if the attacker wears that element’s ring** *(2026-08-16)*, read off `Duelist.Rings` — the same array shape, and the seat the ring discount will sit beside. Nothing stacks: a second hit resets the clock. An enemy wears no rings, so an enemy’s colours are inert by construction. **A planner takes the hand it was dealt** — `PlanFor(style, duelist, hand)` — so a style is how a hand is *played*, not what is played, and a brute that draws no Heavy does not swing one. The shuffle that produced the hand stays outside this package, in `internal/decks`, which is what keeps the rules free of randomness and of a clock. `ResolveRound` returns an event log plus the end state; the screen replays it and never computes an outcome. It is tested because it needs no window — and that property, not the package name, is the rule. `internal/music` and `internal/cards` are tested for the same reason. `internal/screens` has three small tests too, which is a **deliberate narrow exception**: they compare constants and walk switch statements, create no `ebiten.Image`, and run headless. They exist because they guard cross-package invariants a compiler cannot see — the card footprint against the renderer, the element and category mappings, the deck row's sort and geometry. Do not read them as licence to test the rest of the screen, and do not reach for a window to keep one alive. **Two categories, four families** *(2026-08-15)*. `Category` is attack/plan and says *when* a card resolves; `Family` is stab/slash/crush/plan and says what kind of card it is. The attack set is a 3x3 ladder — three families by three tiers at 1/2/3 AP for 0.5x/1x/2x damage, identical across the families — plus three plans on the same 1/2/3 ladder: Prepare (1 AP, banks 2), Plan (2 AP, widens next round's hand by two) and Defend (3 AP, halves the blow). **Nothing reduces a blow to zero, and that is a rule** — a turn lands one figure however many cards made it, so total negation would be a whole opposing turn deleted by one card. `FamilyNone` is a real answer and belongs to the opponent's two cards, `Attack` and `Heavy`. **A turn resolves exactly one attack, and combos are how it is scored** *(2026-08-14)*. `resolveAttackPhase` announces every attack card, then `BlowFor` reads them as a *set* and returns one blow: `Σ Damage(cards in the hand) + Strike.Damage(DMG) × (hand + mix)`, where the two multipliers **add**. Attack cards that build no hand are announced and contribute nothing. **The catalogue is two axes, not a list**: `data/combos.json` holds five *hands* (copies of a concept — pair through barrage) and five *mixes* (distinct non-basic colours — drab through rainbow), `combo_table.go` turns them into rules, and `combo.go` holds the vocabulary and the matcher. Exactly one hand and exactly one mix apply, which is what retired the family/tier machinery: a hand wins on its multiplier, and the mixes name exact colour counts that partition every hand. **Matching is counted only** — the `run` match kind was dropped, so nothing in the game reads card order any more. One closed reward vocabulary (damage multiplier, banked AP, stagger). **Adding a combo is one entry in the JSON**; adding a *reward kind* is a field on `Effect` plus one place applying it, and that cost is charged on purpose. **This is the one file in `data/` that the rules themselves read**, and the only reason `internal/combat` imports that package — see the layering note below. A malformed catalogue panics at init, exactly as a mis-declared cost tier does — including a gap in the mixes' colour counts, and including a missing `high-card` entry, since a hand the engine cannot name is the one failure this model can produce. **The High Card is the one-card hand at no multiplier**: when nothing was built, the hardest-hitting attack card is the blow and what lands is its face damage. It is fallen back to rather than matched, because counting picks the commonest concept and not the biggest card. **Defends reduce rather than negate** and compose multiplicatively, order unread; **lightning is a roll**. **The rules cannot draw a card** — there is no deck in the package — so Plan records `Duelist.BonusDraw` and emits `KindDrew`, and the screen's `handTarget` honours it on the next refill. It is assigned rather than added to at the round boundary, so a Plan widens exactly one hand. See `MECHANICS.md`. **Never change these rules to make a screen look right** — if a screen contradicts the engine, say so and let the owner decide which one is wrong. That is a game-design call, and it ripples into the tests and the balance.
- `internal/screens/` — one `Scene` implementation per screen, owning its own state and widgets, calling into `systems` to draw them.
- **The combat screen is nine files.** They are one package and Go does not care where a declaration sits, so these are *reading* boundaries — the point is that an edit does not start by finding your place in 2,000 lines. Grouped by what a change is usually about:
  - `combat.go` — the scene: `CombatScene`, `Init`, `Update`, `Draw`, `startRound`, playback (`advancePlayback`, `applyEvent`, `currentSlot`), the caption text, `nextFight`, and the trace layout dump.
  - `combat_deck.go` — the cards and the piles: `actionCard`, `buildStartingDeck` (which reads `data/duelist_cards.json`), the deck seed, the shuffle and draw, `spendSelected`, and the deck overlay. **`actionCard` is an alias for `combat.Card`** — elements are rules, so the hand, the queue and the round are one type and a card is never converted between them.
    **The overlay shows every card you own**, in four colour rows plus a row of plans, at
    `cards.Mini` overlapped so all but six pixels of each shows. Two rules govern it and
    both have been broken once: *a card does not move when it is played, it only dims* — so
    the hand is included rather than excluded, and availability is the **last** sort key,
    never the first — and the pitch is a constant sized for a full row, never derived from
    how many cards are currently in one. Rows sort stab, slash, crush, plan, cheapest first;
    `familyRank` is written out rather than read off the enum, because the enum's order is what
    an expanded combo ID is derived from and that is a rule. It sorts on **family** rather than
    category because category has two values now, and sorting by it would put nine cards in one
    undifferentiated block.
  - `combat_panes.go` — Action Flow and Resolution: the placements and colours, `paneRow`, `drawPane`, `resolutionLines`, and the prose that turns an event into a sentence.
  - `combat_hud.go` — everything around the round: the two fighter cards, `drawBox`, and the discards badge. **Both duelists are cards**, in opposite top corners, each holding name / DMG / AP / Vitae over a health bar and a fraction. `duelistCardRect` and `enemyCardRect` are the one place each geometry is written, and the ring row takes both of its edges from them.
  - `combat_rings.go` — **the ring row**: full-size `cards.RingStyle` cards from `data/rings.json`, a rule under them running the row's width, and the cap written as `worn/5` on that rule's right end. **It is what the player is wearing** *(2026-08-16)*: `startingRings` names the worn set and `equipRings` turns each one’s element into a flag on `combat.Duelist.Rings`, which is what makes that element’s status fire. Nothing *buys* or unequips a ring yet. It holds the 12–46% band, which is what pays for full-size ring cards. **Its width is what the two fighter cards leave** — `ringPaneRect` reads `duelistCardRect` and `enemyCardRect` rather than a percentage, so the right edge cannot go stale when a card moves. Two things it does deliberately: **a fill, never a frame** — a plain grey backing one step lighter than the screen, no border, no title, no hue, because a framed row reads as cards trapped in a panel while a bare row leaves nothing saying where the middle begins; and **the row drops 10px below the two cards** so the three do not share a top line and read as one wide object. The backing must never reach either card — `TestTheRingBackingHoldsTheWholeRowWithoutTouchingTheCards`. And **the pitch is a function of how many rings are worn**, first card flush left and last flush right, so three stand apart and five close up and overlap by ~26px. Overlap rather than shrink, because a card cannot be scaled and there is no ring style below this one.
  - `combat_actionbox.go` — the hand and its drag-to-reorder.
  - `combat_sort.go` — **how the hand is arranged, and the three square buttons that choose
    it**: `$` cost, `T` type, `E` element, stacked in a column against the band's right edge and
    centred on the cards. **Cost is the default and every mode ends with it** — each is the
    deck overlay's own key chain with one key promoted to the front, so a row of cards means
    the same thing in the hand as in the panel. Three things about it: **the sort re-applies on
    every refill**, so a drawn card lands where it belongs rather than on the right-hand end and
    a drag survives only until the next deal; **`sortMode` is the one field `Init` does not
    reset**, because it is a reading preference rather than a fact about a duel; and **it cannot
    change an outcome**, since nothing in `internal/combat` reads the order of a hand.
    **`sortHand` returns the permutation it applied** — it sorts a slice of indices and rebuilds
    rather than sorting in place — because a card sliding to its new seat has to know where it
    set off from and two identical cards cannot be told apart after the fact.
    `elementRank` and `categoryRank` are written out rather than read off the enums, for the
    reason `familyRank` is — `combat.Basic` leads its enum as the zero value and trails on
    screen, where the colours are what a mix is counted on.
  - `combat_flight.go` — **every card that moves**. Four things, all
    presentation-only, all on their own clock, and none of which may change an outcome:
    - The **deck stack**, and its yellow modal ring.
    - **`cardFlight`** — the discard flying off left and the deal flying back in, turning
      face up on the way. **A flight is raised only after `spendSelected` has already moved
      the card**, so it is a ghost of something that has happened rather than a thing in
      progress; that is what keeps `planning()`, the budget and the row's layout ignorant
      of it.
    - **`handSlide`** — a card moving from one slot in the row to another: a sort, or the
      row closing up after cards were spent. **The only mover whose journey begins and ends
      in the hand**, which is why it carries a row size at *each* end — a survivor leaves one
      row and lands in a differently sized one. Flat and full size, because it crosses inches
      rather than the screen.
    - **`resolvedCard`** — one of the player's cards flying out of the hand to its seat on
      the table. **The whole queue is dealt there when the round starts**, not
      a card at a time as each fires; playback drives which card is *lit*, not which cards
      exist. Because `ResolutionOrder` decides the row, it is laid out in phase order
      **without this file knowing what a phase is**. A combo brackets its own cards in
      `attentionYellow`, from the span the engine puts on the event — never worked out here.
  - `combat_table.go` — **the two hands facing each other**: the
    player's played cards left-aligned, the opponent's queued cards right-aligned, both full
    size in the band between the ring row and the Resolution feed. It is what shows a round as
    a confrontation rather than as a list. **Each row breaks between its attacks and its plans**
    (`tableGroupGap`), and the split is read off `combat.ResolutionOrder` rather than counted
    here — the gap is spent out of the same half-width, so it costs overlap rather than width
    and the two hands still cannot meet. Both rows come
    from `combat.ResolutionOrder`, so both say what *will* happen rather than what was planned.
    **The opponent's cards are drawn face up and that is temporary** — `concealEnemy` is
    the screen's concealment predicate and this row deliberately ignores it, on the owner's
    call, with `cards.Spec.FaceDown` already built as the lever for putting it back.
  - `seeds.go` — the named opening-hand catalogue.

  **A file boundary is not a reason to change what a function does.** Moving something between these files is a move, not a rewrite.
- `internal/actions/` — callbacks that act on the game as a whole: change screen, quit. They take `gs` and mutate it; they never draw. **Callbacks touching only one screen's state do not go here** — those are methods on the scene that owns the state.

Dependency direction: `main` → `game` → `screens` → `systems`/`entities`/`actions`/`decks` → `models`/`state`/`combat`/`assets` → `data`. Nothing lower reaches back up. `decks` sits above `combat` and `data` and below `screens`, which is the whole reason it is a package: it is the one place allowed to turn a JSON card list into rules types, and `tools/balance` imports it without importing a screen. `state` sits near the bottom and must stay there — if it starts importing `entities` or `models` again, screen state has leaked back into it.

**`data` is the bottom of the graph and imports nothing but the standard library**, which is what lets any layer read it. `internal/combat` does, for `data/combos.json` alone — the one list the *rules* consume rather than a layer above them. **Whether a new file in `data/` may be read by `combat` is decided by who consumes it, not by whether it is data**: enemy rosters, deck lists and rings are read by `screens`, `decks` and `entities`, and a rule reaching for one of those would mean the rules had grown an opinion about portraits or art keys. `data` must never import upward; `CheckCostTiers` takes its cost and category lookups as parameters for that reason and should keep doing so even though the edge now exists, because the check belongs to whoever is loading a deck.

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
