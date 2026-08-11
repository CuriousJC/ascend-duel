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
- **`TODO.md` is the work list**, and still holds the reasoning behind decisions already
  implemented. It is long; prefer `MECHANICS.md` for "what should this do".
- When the two disagree, `MECHANICS.md` is newer and wins — say so rather than guessing.

## Caveman mode is on in this repo — a trial, and it expires

**Load the `caveman` skill at the start of every session and stay in it.** Started
2026-08-10, to be judged **on or after 2026-08-17**. It compresses replies in the terminal and
nothing else.

- **The trial has an end date on purpose.** On 2026-08-17, ask whether it stays. If the
  answer is no, delete this section and `.claude/skills/caveman/` and everything is back to
  normal. If it stays, delete these two sentences and leave the rest.
- **It never touches anything persisted** — code, comments, commit messages, PR bodies,
  `MECHANICS.md`, `TODO.md`, this file. The skill's own Boundaries section says so and it is
  the reason the trial is safe: the design record keeps its longhand reasoning. Compression
  applies to what is typed back into the terminal.
- **It does not override the instructions that say to argue.** Raising a structural
  objection, saying where a claim came from, and saying plainly when something is unverified
  all still happen — in fewer words, not fewer times. **If terseness is eating the
  reasoning rather than the padding, that is the trial failing**, and it is worth saying so
  rather than quietly writing shorter.
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
- **Assets need provable licenses.** Do not add assets with unclear provenance; "found
  it online" is not sufficient for a paid release.
  **The Tyrian art was the known blocker and it is gone as of 2026-08-09** — every file
  is deleted and nothing references one. What replaced the enemies is PVGames creature
  art from the Humble *Isometric Assets Galore* bundle, whose licence permits shipping
  inside a game; the pack and its terms are in `.scratch/flat-creatures` (gitignored).
  Everything else in `assets/` is either first-party or generated at runtime.
- The pre-relicense Apache 2.0 grant on already-published commits is irrevocable and
  the owners have accepted that. Do not propose rewriting history over it.

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
holds a catalogue of named seeds — `strike-flurry`, `strike-onslaught`, `all-categories` — so a
hand that demonstrates something can be asked for by name instead of found by relaunching.
`deckSeedName` picks which one a launch deals.

**Re-run `tools/seeds` after touching `data/duelist_cards.json`, `startingDeck` or `handSize`.** A seed is
a fact about one particular deck; change the deck and every catalogued number silently deals
something else. The tool re-checks the catalogue before it searches and says which entries no
longer match — it rejected two guessed numbers the first time it ran, and invalidated **all five
entries at once** when the deck went 30 → 60 on 2026-08-08. A demo testing a Flurry against a hand
with two Strikes in it is worse than no demo, because it passes.

**A bigger deck needs a bigger search.** Five Strikes in a hand of eight is about 1 hand in 98,000
from 60 cards, against 1 in 3,000 from 30, so `strike-onslaught` took `-n 600000` to find where
the default 20,000 used to be plenty. A seed the tool reports as unfindable may only mean the
search was too short — check the arithmetic before concluding the deck cannot deal it.

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

## Releasing — `.github/workflows`

**CI** runs on every PR, on **Windows and Linux**, under all three build-tag configurations.
**Release** has two entrances and both produce the same release. Pushing a `v*` tag still
fires it, so *tagging is releasing*:

```powershell
git tag -a v0.1.0 -m "..."; git push origin v0.1.0
```

- **Or run it by hand from the Actions tab** — *Release* → *Run workflow*, on `main`, with
  the version typed in. **The workflow creates the tag itself**, at the commit it built, via
  `gh release create --target`. That is the normal path now; a local tag-and-push is no
  longer needed to cut a release.
- **The manual path is guarded, because a typed version has no `v*` filter in front of it.**
  A `version` job runs first and fails the whole run if the branch is not `main`, if the
  string is not `vMAJOR.MINOR.PATCH`, or if that tag already exists — the last one because
  `gh` would otherwise attach binaries to a tag naming a different commit. The input is read
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
so the player can retry and make different choices. Nothing is stochastic yet, which is
exactly why these rules are cheap to follow — retrofitting determinism is expensive, so
do not write code that forecloses it.

- **Never call the `math/rand` package-level functions** (`rand.Intn`, `rand.Float64`,
  `rand.Shuffle`, …). They draw from a global source shared with every other caller,
  which makes a run unreproducible. Randomness comes from an explicit `*rand.Rand`
  carried on state and seeded once per run.
- **Five separate streams: enemy selection, loot offers, floor offers, and *two* card
  shuffles — the player's and the opponent's.** Never share one source between them, or a
  change to loot generation silently rerolls every enemy in the tower. A stream is only
  ever advanced by its own concern. Tower layout is fixed (8 floors × 3 fights, endless
  later) and draws no randomness.
- **The two shuffles were one stream in the plan and had to become two** *(2026-08-11)*,
  when enemies got a deck. Sharing would make the player's opening hand a function of how
  many cards the opponent happened to draw, so **every entry in `seeds.go` would break the
  first time an enemy deck was retuned** — and a named hand has to stay a fact about the
  player's deck alone. That is the shape of argument to apply if a sixth stream is
  proposed: ask what it would silently reroll.
- **Both shuffles exist now.** The player's lives on `CombatScene` as `rng`, seeded in
  `Init` from `deckSeed`; the opponent's lives on `decks.EnemyPile`, seeded from
  `decks.EnemySeed`. Both constants are placeholders for the per-run seed — every launch
  deals the same hands, which is what makes a problem reproducible while the screen is
  being built. When `Session` state lands, both read from there and nothing else about the
  deck code changes.
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

## The combat screen — see the `combat-screen` skill

Its layout, its card and action-box widget, its hidden information, and the resolution-order
rule the screen has to obey all live in
[.claude/skills/combat-screen/SKILL.md](.claude/skills/combat-screen/SKILL.md). **Load it
before touching any of the combat screen's files — `internal/screens/combat.go`,
`combat_deck.go`, `combat_panes.go`, `combat_hud.go`, `combat_actionbox.go` — or
`internal/combat`, or anything about how a round is drawn or played back.**

It is a skill because it is the screen under active construction — it was over half this
file and it grows every session, while mattering only when that screen is the work. The
general UI conventions below still apply to it and stay here.

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
- ebitenui was evaluated and declined — see the entry in `TODO.md` for the reasoning and
  the repo data behind it. Do not reach for it without revisiting that.

The action box is a *game* widget, not a UI widget: draggable action cards with live
action-point validation. General-purpose toolkits are weakest at exactly that, so
hand-rolling costs little and buys full control.

### Glyphs are generated, not drawn — and not scaled from one colour

[internal/systems/glyphs.go](internal/systems/glyphs.go) generates the pixel-art
glyphs on the action cards, drawn at 1:1 — 64x64 for the damage sword and the runner, 22x22
for the three category glyphs. **Generated art has no provenance question**, which is
exactly the problem the Tyrian set used to be, so this is the pattern to prefer for
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
  still one palette, `white`, with no hue at all, and every glyph uses it. Holding them
  hueless from 2026-08-03 is what left colour free to mean "element", and they should stay
  that way — a coloured glyph on a coloured card says it twice and leaves nothing for the
  next distinction. **What carries the element is now the card's *border*, not its surface**
  (see the colour section below).
- **A hueless glyph on an off-white card loses most of its bevel, and that is accepted.**
  `Specular` is pure white and `Highlight` is `{232,236,242}`, so against the off-white
  surface the lit side of a bevel largely disappears and a glyph reads as outline plus
  shading. The near-black `Outline` carries legibility, so nothing is broken — but the
  five-value palette is mostly spent, and `cards.Surface` is one constant if that ever
  needs re-testing.
- **Glyphs are not all one size, and `systems.SizeOf(kind)` is the authority.** The damage
  sword and the runner are 64; the three category glyphs are **22**. Never assume
  `GlyphSize` at a call site — a small glyph centred in a 64-pixel hole is the failure.
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
- **The card's height budget is spent differently now, not enlarged.** It was two 64-pixel
  badges (damage and cost). Cost became a stack of dash marks and the category *word*
  became a 22-pixel glyph, so the column is one badge, one small glyph and some bars, and
  the card is the same 180x264 with ~94px of deliberately empty surface at the bottom for a
  long-press description. Adding a badge back spends that.
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

**The sheet measures each glyph rather than assuming 64**, since they are no longer all
one size, and centres a small one on its own width so a row reads as different sizes rather
than as misalignment.

**The sheet draws each glyph twice: at `systems.CardGlyphScale` and enlarged.** The
actual-size row is the one that answers "can I read this". The scale constant lives in
`systems` precisely so the sheet reads the same number the card does and the preview cannot
drift from the game — an earlier version showed only the enlarged row, and the glyphs duly
looked acceptable in review and clunky in play.

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
  card still plays the game.
- **There is no way to mute it yet.** That wants an on-screen button, not a hotkey — the
  input vocabulary has no keyboard. See `TODO.md`.

### Cards: the border carries the element, not the surface

**Reversed on 2026-08-09.** A card used to be an element-coloured surface with an
incidental border. It is now a **constant off-white surface** (`cards.Surface`) with a
**thick coloured border** carrying the element, and the whole card is drawn by
`internal/cards`. Three things fell out of that reversal and are easy to re-break:

- **A near-white border on an off-white card is invisible.** The elementless card was
  `{235,235,235}` as a *surface* and read fine; as a border it vanishes. `basic` is
  therefore a mid grey in `cards.BorderOf`, and a test fails if it is "restored".
- **`ColorAtStrength` is the wrong tool on a light card, and this has already caused one
  bug.** It scales toward *black*, which reads as quieter only against a dark ground. On
  an off-white card a border scaled down comes out darker than the surface and therefore
  *louder* than the live card beside it — exactly what put the Resolution pane's idle rows
  in front of its lit one. Use `systems.ColorToward(c, ground, pct)`, which moves a colour
  toward whatever it actually sits on. Card state is expressed as distance to the surface.
- **Cost is dash marks and the category is a glyph**, not text and not a numeral. Costs run
  1..4; a fifth tier grows the dash stack into the damage badge and is a layout change, not
  just a bigger number. `TestLeftColumnDoesNotCollide` fails rather than rendering it.

Rings reuse the whole format with a pink border and artwork instead of glyphs, and no cost
or category because a ring is neither played from a hand nor resolved in a round. Nothing
in the game builds one yet — `tools/cardsheet` is the only place they exist.

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



**And it applies against a dark ground.** `ColorAtStrength` scales toward black, so on a
light surface it makes things louder rather than quieter — see the card section above.
`systems.ColorToward` is the light-ground counterpart and the two are not interchangeable.

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

`ActiveDebug` split into `DebugPlacement` and `DebugGameplay` on 2026-08-02, because they
answer different questions and are wanted at different times.

- **`DebugPlacement`** — the grid, the rulers, the `Debug1`/`Debug2` scratch strings. About
  *where things are drawn*. Safe to leave on while playing. It defaulted on while the combat
  screen was being laid out; it no longer does, so a change that needs the guides has to turn
  it on deliberately.
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

- `assets/` — `//go:embed`s every image and font into the binary and exposes `LoadAssets()` / `LoadFonts()`, returning `map[string]*ebiten.Image` and `map[string]*text.GoTextFaceSource`. **A new asset needs three edits: the file, an `//go:embed` var, and a map entry in the loader.** The map key is the lookup name used everywhere else (e.g. `gs.Assets["giantrat_png"]`, `gs.Fonts["kubasta"]`), and it is **independent of where the file sits** — see the Art section. There are two extra loaders for callers that cannot take an Ebitengine type: `LoadFontData()` and `LoadImageData()` hand back raw bytes, because `internal/cards` and `tools/cardsheet` render without a graphics context. `gs.FontData` carries the fonts for exactly that reason.
- `data/` — JSON next to a small Go loader, which is the pattern for all static game data. **Four files since 2026-08-11**, where there were two:

  | File | Loader | Holds |
  |---|---|---|
  | `duelists.json` | `LoadDuelists` | who the player can be, and their card back |
  | `enemies.json` | `LoadEnemies` | 96 opponents: stats, style, portrait, valid floors |
  | `duelist_cards.json` | `LoadDuelistCards` | the player's deck list |
  | `enemy_cards.json` | `LoadEnemyCards` | what an opponent draws from |

  **The player and the enemies split because their fields stopped overlapping** — an enemy has a plan style, a portrait and an affix pool; a duelist has a card back and, eventually, a deck. One struct meant every field was optional and none of them meant anything.

  `PlanStyle` names how an enemy fights and is parsed by `combat.ParsePlanStyle`, falling back to brute — **enemy behaviour is data, so the roster is tunable without touching Go.** `ValidFloors` is `[lowest, highest]` against the planned 8-floor tower, so a Dragon is not on floor one; nothing generates floors yet, so today it only sorts the fight order. `data.EnemyOrder` is the sorted walk — never range the map, per the determinism rules.

  - **The two card lists are one shape, tuned separately.** Concepts, the elements each ships in, and how many copies. `startingDeck` is built from the duelist list, so **deck size is a consequence of a file you can read** — 12 concepts × 5 elements = 60. Enemy cards are all `basic`, because an enemy card is never drawn on screen and MECHANICS.md has affixes *transforming* a basic deck into an element.
  - **Cost, category and damage are deliberately *not* in either**; they are rules and live in `internal/combat`, which cannot import `data`. **So a separate enemy file cannot yet give an enemy Strike a different cost** — that is a rules change, not a data one. The JSON's `CostTier` is documentation *with a check*: `data.CheckCostTiers` asserts every declared tier and category against `ActionKind.Cost()`/`.Category()` and both deck builders **panic at package init** on any disagreement. A deck quietly five cards short is a balance change nobody made, so it fails on launch instead. Concept names are joined to the rules by `combat.ParseAction`, which reports failure rather than falling back.
- `internal/decks/` — **the opponent's deck, and the only package between `data` and `internal/combat`** (added 2026-08-11). It exists so the combat screen and `tools/balance` share one enemy deck: the balance tool plays whole duels headlessly and cannot import `internal/screens`, which links Ebitengine. **No Ebitengine here, ever**, for that reason. `EnemyPile` is the three piles plus a shuffle; **the enemy's hand does not persist between rounds**, unlike the player's, because a style only takes attacks plus a Guard or Gather and everything else would accumulate until the hand locked up — which it did. The player's hand may persist because Discard exists, and that is the lever an enemy has not got.
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
  the deck overlay) and `RingStyle`. Text is set with `golang.org/x/image` because
  Ebitengine's `text/v2` needs an `*ebiten.Image`; both the game and the tool go through
  this, so the sheet cannot drift from what is drawn. `internal/screens/card_art.go` is the
  bridge and holds the cache — rendering writes every pixel in Go and is far too slow to do
  per frame.
- `internal/music/` — the score, **synthesised at startup from a MIDI file** (added 2026-08-09). See the section below; the short version is that `smf.go` and `synth.go` are pure arithmetic and tested, and only `music.go` touches Ebitengine's audio.
- `internal/screens/combat_demo_{on,off}.go` — the scripted-demo driver, behind `demoplay`. Same two-file shape, and it lives beside the screen it drives rather than in a package of its own because it reaches into that screen's own methods (`toggle`, `startRound`). It holds its script in package state so `combat.go` gains only two call sites. **It may never change an outcome**, the same constraint as trace and idle.
- `internal/combat/` — the duel rules, **the opponent's planners, and the combo table**. **No Ebitengine import, ever.** **A planner takes the hand it was dealt** since 2026-08-11 — `PlanFor(style, duelist, hand)` — so a style is how a hand is *played*, not what is played, and a brute that draws no Heavy does not swing one. The shuffle that produced the hand stays outside this package, in `internal/decks`, which is what keeps the rules free of randomness and of a clock. `ResolveRound` returns an event log plus the end state; the screen replays it and never computes an outcome. It is tested because it needs no window — and that property, not the package name, is the rule. `internal/music` and `internal/cards` are tested for the same reason. `internal/screens` now has three small tests too, which is a **deliberate narrow exception**: they compare constants and walk switch statements, create no `ebiten.Image`, and run headless. They exist because they guard cross-package invariants a compiler cannot see — the card footprint against the renderer, the element and category mappings, the deck row's sort and geometry. Do not read them as licence to test the rest of the screen, and do not reach for a window to keep one alive. **Combos are a framework, not a pile of cases** — `combo.go` is one pattern (a run of cards) and one closed reward vocabulary (damage multiplier, banked AP, opponent alteration). Adding a combo is one table entry; adding a *reward kind* is a field on `Effect` plus one place applying it, and that cost is charged on purpose. See `MECHANICS.md`. **Never change these rules to make a screen look right** — if a screen contradicts the engine, say so and let the owner decide which one is wrong. That is a game-design call, and it ripples into the tests and the balance.
- `internal/screens/` — one `Scene` implementation per screen, owning its own state and widgets, calling into `systems` to draw them.
- **The combat screen is six files, split 2026-08-07 when `combat.go` reached 87 KB.** They are one package and Go does not care where a declaration sits, so these are *reading* boundaries — the point is that an edit no longer starts by finding your place in 2,000 lines. Grouped by what a change is usually about:
  - `combat.go` — the scene: `CombatScene`, `Init`, `Update`, `Draw`, `startRound`, playback (`advancePlayback`, `applyEvent`, `currentSlot`), the caption text, `nextFight`, and the trace layout dump.
  - `combat_deck.go` — the cards and the piles: `element`, `actionCard`, `buildStartingDeck` (which reads `data/duelist_cards.json`), the deck seed, the shuffle and draw, `spendSelected`, Sift's random discard, and the deck overlay.
    **The overlay shows every card you own**, in five rows of one element each, at
    `cards.Mini` overlapped so all but six pixels of each shows. Two rules govern it and
    both have been broken once: *a card does not move when it is played, it only dims* — so
    the hand is included rather than excluded, and availability is the **last** sort key,
    never the first — and the pitch is a constant sized for a full row, never derived from
    how many cards are currently in one. Rows sort attack, defend, prepare, cheapest first;
    `categoryRank` is written out rather than read off the enum, because the enum's order is
    *resolution* order and that is a rule.
  - `combat_panes.go` — Action Flow and Resolution: the placements and colours, `paneRow`, `drawPane`, `resolutionLines`, and the prose that turns an event into a sentence.
  - `combat_hud.go` — everything around the round: the character strip, the caption box, `drawBox`, the enemy sprite, and both health bars.
  - `combat_actionbox.go` — the hand and its drag-to-reorder, unchanged by the split.
  - `combat_flight.go` — **every card that moves** *(added 2026-08-10)*. Three things, all
    presentation-only, all on their own clock, and none of which may change an outcome:
    - The **deck stack** that replaced the Deck button, and its yellow modal ring.
    - **`cardFlight`** — the discard flying off left and the deal flying back in, turning
      face up on the way. **A flight is raised only after `spendSelected` has already moved
      the card**, so it is a ghost of something that has happened rather than a thing in
      progress; that is what keeps `planning()`, the budget and the row's layout ignorant
      of it.
    - **`resolvedCard`** — a card firing during playback: up out of the hand, held below the
      Resolution pane, then stacked in the bottom-left corner, at full size throughout. The
      pile is the round's history, and because `ResolutionOrder` regroups into prepare →
      attacks → defenses it grows in phase order **without this file knowing what a phase
      is**. A combo brackets its own cards in `attentionYellow`, from the span the engine
      puts on the event — never worked out here.
  - `seeds.go` — the named opening-hand catalogue.

  **The split was a pure move**: every line went across unaltered and the `demoplay` text report was byte-identical before and after. Keep it that way — a file boundary is not a reason to change what a function does.
- `internal/actions/` — callbacks that act on the game as a whole: change screen, quit. They take `gs` and mutate it; they never draw. **Callbacks touching only one screen's state do not go here** — those are methods on the scene that owns the state.

Dependency direction: `main` → `game` → `screens` → `systems`/`entities`/`actions`/`decks` → `models`/`state`/`combat`/`data`/`assets`. Nothing lower reaches back up. `decks` sits above `combat` and `data` and below `screens`, which is the whole reason it is a package: it is the one place allowed to turn a JSON card list into rules types, and `tools/balance` imports it without importing a screen. `state` sits near the bottom and must stay there — if it starts importing `entities` or `models` again, screen state has leaked back into it.

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

**The map keys did not move with the files.** They are the lookup names used across the
game, and `data/*.json` writes them down — tying a key to a path would mean a data
migration every time a file was refiled. A named asset is still three edits: the file, an
`//go:embed` var, and a map entry.

**The enemy portraits are the exception, and they are a *family* rather than named
assets** *(2026-08-11)*. There are 96 of them, so `//go:embed enemy/*-portrait.png` pulls
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

**Enemy sprites are gone** *(2026-08-11)*. `assets/enemy/*.png` was one west-facing idle
frame per creature; the enemy is drawn as a card now, so nothing used them, and cutting 96
more frames to keep the pattern would have been the expensive half of that change. Git has
the four that existed. The full animation sheets stay in `.scratch/flat-creatures`
(gitignored) — that folder's README documents the grid, the frame table and the facing
order — so animating enemies later means going back for them rather than starting over.

**Nothing in the game draws a loose sprite any more.** `Combatant` has no `Sprite` field,
`entities` imports no Ebitengine at all, and `drawCombatant`/`DrawHealthBar` went with the
sprites. Both duelists state their life as a number: the player in the character block, the
enemy on its card.
