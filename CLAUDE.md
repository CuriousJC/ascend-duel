# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## Project

Ascending Duel — a roguelike where you duel your way up a tower, collecting relics and brands of
power. Written in Go with [Ebitengine v2](https://ebitengine.org/)
(`github.com/hajimehoshi/ebiten/v2`). Module path: `github.com/curiousjc/ascend-duel`.

## Where things are written down

Six streams, each with one job. Reach for the right one rather than searching all of them.

| Stream | File | Read it when |
|---|---|---|
| **How to work** | `CLAUDE.md` — this file | always; it is loaded every session |
| **Procedure** | `.claude/skills/*/SKILL.md` | on trigger — see the index below |
| **What the game *is*** | [MECHANICS.md](MECHANICS.md) | designing or implementing any mechanic, before proposing a design change |
| **What the game *contains*** | `data/*.json` | any question about a catalog — what is in it, how many, what one record says |
| **What to build next** | [TODO.md](TODO.md) | picking up work |
| **Unfiltered** | [ideas.md](ideas.md) | the inbox; entries get promoted into MECHANICS or TODO and struck from here |

- **`MECHANICS.md` is the design record.** Decided unless marked `[?]`. It holds the element
  set and their statuses, cards and types, hands, relics, brands, vitae, the tower, enemies,
  and the phase-based resolution experiment.
- **`TODO.md` is open work only.** Completed entries are deleted rather than archived, so it
  says what is left, not what happened. Prefer `MECHANICS.md` for "what should this do".
- **Never add an entry to `TODO.md` unless the owner asks for that specific thing to be tracked**
. Noticing something during a change is not a reason to file it.
  Say it in the reply and let the owner decide — the list is theirs to grow, and a list that
  accumulates every observation anybody had is one nobody reads. The same goes for `[?]` entries
  in `MECHANICS.md`: an open question is filed because the owner wants it open, not because the
  work turned one up.
- When the two disagree, `MECHANICS.md` is newer and wins — say so rather than guessing.
- **`data/` is the catalog, and this file never says what is in it**. How many relics there are,
  which essences exist, what a rune's line reads — all of that is a `data/*.json` read or a
  `docs/sheets/` page away, and it is **pre-v1 and changing constantly**, so a count written
  down here is wrong within the week and wrong *silently*: nothing compiles it, nothing tests
  it, and it is loaded into context every session to mislead.

  So the rule is a capability rather than a fact: **say which file answers the question and
  which tool draws it, never the answer.** "The relic catalog is in `data/relics.json`, reviewed
  with `go run ./tools/relicsheet`" survives any amount of authoring; "forty-six relics" did not
  survive a fortnight. The same goes for anything else that grows by someone authoring a record
  — creatures, bosses, achievements, stones, hand rungs, portraits.

  **A closed vocabulary is the exception and is not a catalog.** The five elements, the four
  forms, the three verbs, the three axes are design invariants: a sixth element is a decision, not
  an edit, so naming them here is naming a rule. If a list can grow by authoring, it belongs in
  `data/`; if growing it is a design change, it belongs here.

- **`README.md` is not one of the streams, and it is only touched when the owner asks for that
  specific thing**. It is the front page a stranger reads, so it is
  the owner's voice rather than a working document, and a change to it is a change to how the
  project introduces itself — which is a decision, not a side effect of the work that happened to
  touch the same subject. **The same no-counts rule applies to it**, and for the harder version of
  the reason: nothing here is loaded into a session to mislead, but a stale figure on the front
  page is read by people who have no way to check it. It says which catalogs exist and points at
  `data/` and `docs/sheets/`; it does not say how many of anything there is. Noticing that it has
  gone out of date is something to say in the reply, not something to fix.

- **Write what is true now. Never write why it changed**. Not in
  `MECHANICS.md`, not in `TODO.md`, not in a skill, not in a doc comment, not in a test's comment.
  **The release notes are where change lives** — `.github/release-notes/<tag>.md` — and git history
  is underneath them. If the owner wants to know what moved, that is where he will look.

  The banned shapes, in rising order of how easy they are to write without noticing:

  - **A tombstone** — "the percentage guard was deleted", "there is no mute latch any more". A
    record of something that does not exist.
  - **A reversal** — "this reverses the old rule", "it has been both ways", "it was X until
    2026-09-15". A record of an argument that is over.
  - **A justification by history** — "the argument then was ... what changed is ...", "it used to
    be a fixed array because ...". The current reason stands on its own or it is not a reason.
  - **A dated attribution on a rule that is simply the rule.** `*(owner's call, DATE)*` earns its
    place on a decision someone might otherwise re-open; on an ordinary statement of fact it is
    another sentence of history.

  **What stays is the constraint, stated forward.** "A shield eats a whole attack; nothing reduces a
  blow by arithmetic" is a rule. "The percentage guard was tried and cut, so nothing reduces a blow
  by arithmetic" is the same rule carrying a corpse. **If a mistake is worth warning about, warn
  about the mistake** — "do not price a rung below 100: it pays a player less for building more" —
  rather than narrating the time somebody made it.

  The cost this is paying is real and is accepted: these files are loaded into context every
  session, they grow without bound because nothing ever retires a line of history, and a reader
  looking for the current rule has to work out which half of the paragraph is still true.

### The skill index — every skill and what trips it

**This table is the tripwire.** The individual sections further down explain *why* a skill
exists; this says only what makes you reach for one, so nothing is missed by not having read to
the bottom of a long file. **A new skill is a row here** — a skill nobody knows to load is a
skill that does not exist.

| Skill | Load it before |
|---|---|
| [`caveman`](.claude/skills/caveman/SKILL.md) | **every session, at the start** — see below; it is on by default in this repo |
| [`github-workflow`](.claude/skills/github-workflow/SKILL.md) | any `git` or `gh` command — branching, committing, pushing, opening or merging a PR, cleaning up after one, or when a merge is refused |
| [`data`](.claude/skills/data/SKILL.md) | adding a file to `data/`, adding or changing a field on one, authoring cards / enemies / relics / essences, or writing a loader |
| [`randomness`](.claude/skills/randomness/SKILL.md) | adding any roll, adding or seeding a stream, touching a salt or a seed, writing a shuffle, or deciding whether a mechanic should be random at all |
| [`combat-screen`](.claude/skills/combat-screen/SKILL.md) | touching any `internal/screens/combat*.go`, `internal/combat`, or anything about how a round is drawn or played back |
| [`motifs`](.claude/skills/motifs/SKILL.md) | adding a motif file, adding or changing a record under `data/motifs/`, authoring creatures or bosses, touching `data/tower.json`, or wiring anything that picks an opponent |
| [`relics`](.claude/skills/relics/SKILL.md) | designing, **discussing** or **analysing** a proposed relic, adding to `relics.json` or `statuses.json`, adding a moment or an effect verb, or wiring anything that reads a worn relic |
| [`art-batch`](.claude/skills/art-batch/SKILL.md) | generating art options for a record and choosing between them, a folder of generated pictures turning up to be looked at, or installing, replacing or comparing anything in `assets/` |
| [`relic-balance`](.claude/skills/relic-balance/SKILL.md) | any question about the relic catalog **as a whole** — is offense over-weighted at common, does every element have a cost relic, what a batch of new relics does to the shape of the shelf — or adding a category, an axis, or a verb that has to be classified |
| [`bug-hunter`](.claude/skills/bug-hunter/SKILL.md) | any bug the owner found while playing — before diagnosing it, before fixing it, and before authoring a scenario record to reproduce it |
| [`audit`](.claude/skills/audit/SKILL.md) | the milestone pass — the owner asking for an audit, a refactor sweep or a health check — and before any refactor that crosses more than one package |

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
  it online" is not sufficient for a paid release. **Everything in `assets/` is first-party**: art
  generated from the prompts in `docs/art/`, and a score synthesized at startup from a MIDI file
  this repo owns. Nothing in it came from anywhere else, and nothing in it needs a third party's
  permission to ship.
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
go run -tags scenario .     # a chosen set of relics, a chosen opening hand, a chosen enemy
ASCEND_DUEL_SCENARIO=seven-term-sum go run -tags scenario .   # a named one
go run ./tools/marksheet    # every form mark and cost tick as a matrix, at both drawn sizes
go run ./tools/sheets       # regenerate every review sheet and the index that links them
go run ./tools/cardsheet    # every card variation to PNGs + an HTML page, then refresh the tab
go run ./tools/relicsheet    # every relic to PNGs + a page grouped by rarity: art, price, text, rules
go run ./tools/essencesheet    # every essence to PNGs + a page grouped by what it changes about a card
go run ./tools/handsheet    # every rung of the hand ladder as a real hand, by multiplier, with its odds
go run ./tools/motifsheet   # the roster motif by motif: card, stat line, deck, coverage grid
go run ./tools/creatureprompt -record goblins-outer-bomber -element ice   # the four-layer brief
go run ./tools/creatureprompt -gaps      # which art briefs are still unwritten, roster-wide
go run ./tools/stonesheet   # every stone against the rung it raises, grouped by axis
go run ./tools/runesheet # every rune: the line it prints against the rule that fires
go run ./tools/upgradesheet  # every visible card upgrade, on every form mark, in every upgrade style
go run ./tools/goodsheet     # every sealed good beside the offer it actually makes
go run ./tools/badgesheet    # every damage badge: eleven values by six colors, in all three outlines
go run ./tools/scenariosheet # every debug fixture: what it plugs in and the command that launches it
go run ./tools/scenariodeck -form slash -size 40   # writes a scenario's Deck block to stdout
go run ./tools/relicart      # files generated relic art: reduce, commit, set "Art" on the record
go run ./tools/relicart -kind essence       # the same, for data/essences.json and assets/essence
go run ./tools/relicart -kind rune   # the same, for data/runes.json and assets/rune
go run ./tools/relicart -kind stone  # the same, for data/stones.json and assets/stone
go run ./tools/relicart -kind other  # the potions and the sealed goods together, into assets/other
go run ./tools/artcompare   # one catalog's art, every candidate batch side by side, as real cards
go run ./tools/artcompare -catalog relic    # the same for relics; card, essence, rune, stone, other
go run ./tools/seeds        # re-check the named deck seeds, and search for new ones
go run ./tools/handodds     # how often each rung of the hand ladder can actually be built
```

**A sheet can be narrowed with a chip bar, and `tools/sheetfilter` is the whole of it** — the
third shared library under `tools/`. A sheet declares its facets, tags each record with
`class="sheet-item"` and a `data-` attribute per facet, wraps each heading and its records in
`class="sheet-group"`, and drops the bar in above the contents; values within a facet are an OR
and facets are an AND. The motif sheet is cut on floor, room and element, and the relic sheet on
rarity and family. Two rules to keep:

- **A facet's values are counted off the records, never typed into the tool.** A motif authored
  onto a ninth floor puts a ninth chip up with nothing edited — the same reason nothing in this
  file writes down how many of anything there is.
- **The page is complete before the script runs.** Filtering hides what is already in the file,
  so a sheet stays one static file with no build step and reads whole with scripting off. That is
  the line a new facet may not cross: nothing is fetched and nothing is templated in the browser.

**The sheets are committed, under `docs/sheets/`**. They write there
rather than beside their own tools, and `docs/sheets/index.html` is the page a bare clone opens to
see every card, relic, essence, hand, stone, rune, upgrade, creature and boss in the game. A
regenerated artefact is worth committing here because of the audience: a sheet needing a Go
toolchain and a remembered command is a sheet only ever seen by whoever just changed the thing it
shows.

**The scenario sheet is the one page that is not a catalog**: it is a picture of
the debug fixtures in `internal/scenario/scenarios.json`, none of which is reachable from a shipped
binary. It is on the index anyway, because "what can I boot into" is the question asked most often
and answered worst.

**`tools/artcompare` is a sheet-shaped tool that is deliberately not a sheet**. It draws one
catalog's records once per **set** of pictures and puts them side by side, so a replacement
batch is judged record by record rather than all at once — and it takes any number of sets,
because a generator produces options rather than an answer. Every cell is `cards.Render` at the
catalog's own style, so what is compared is the card as it will be dealt, type over picture;
clicking the one to keep builds the copy list at the top of the page. Six catalogs: `card`,
`relic`, `essence`, `rune`, `stone`, `other`.

**`artreview/` is gitignored and is the whole of its working directory.** A batch dropped in
`artreview/<catalog>-<label>/` is found as the set `<label>`, the installed `assets/` directory is
always the set `current`, and the page is written to `artreview/out-<catalog>/`. That is what lets
the tool be committed with nothing committed under it: **the winners are installed into `assets/`
and committed there, and the losers were never the game.** A page about a decision does not belong
in `docs/sheets/`, which is a report on the catalog that shipped.

**The procedure is the [`art-batch`](.claude/skills/art-batch/SKILL.md) skill** — back up before
overwriting, skip the zero-byte files a generator ships, build the page, apply the picks it writes,
regenerate the one sheet that changed. Load it before installing or replacing any art in `assets/`.

**The cost is history weight, so regenerate deliberately.** A full run rewrites every binary under
`docs/sheets/`, and a sheet rebuilt in a commit that changed nothing about it is pure weight. **Most
of that weight is the two roster sheets**, which carry a photographic portrait per creature and per
boss — so a commit touching only `relics.json` should regenerate the relic sheet alone rather than
reaching for the one command out of habit. **`go run ./tools/sheets` is the one command** — it runs
them all and rewrites the index, because a handful of commands remembered in the right order is how
all but one end up current and one ends up lying. A stale sheet is worse than none: it is a picture
of a catalog that no longer exists.

**A seed is an opening hand**, because the shuffle is deterministic. `internal/screens/seeds.go`
holds a catalog of named seeds, so a hand that demonstrates something can be asked for by name
instead of found by relaunching.
`deckSeedName` picks which one a launch deals.

**Stones raise a rung for one run and never touch the catalog**. `data/stones.json`
holds one per hand, a run's counts ride on `combat.Duelist.HandStones`, and `handTable` is read
*through* them — so the ladder every tool and test sees is the shipped one. See MECHANICS.md
§Stones and `internal/combat/stone.go`, which owns the arithmetic. **The corollary for tuning:**
`tools/handodds` and `tools/handsheet` describe the game as shipped and say nothing about a run
that has been buying rocks.

**A run keeps two counters against every rung, and only one of them is a stone**. A stone is the
rung's **level**: bought, saved by hand key, and read through `Duelist.HandTable` into what the
hand pays. The **plays** are how often the run has actually formed the rung: earned, saved
beside the stones in `run.json`, and read by nothing in `internal/combat` at all.
`internal/session/play.go` owns the tally and `CombatScene.recordHandsPlayed` is the one place
it grows — **off the resolved event log, never off the playback**, exactly as `payHeldVitae`
takes the round's purse. The hands panel is where both are read. See MECHANICS.md §Play counts.

**There is one Pair rung, not one per axis.** `pair` is written `"match": "any"` in
`data/hands.json` and read on whichever of concept / form / element the turn satisfies —
`combat.Hand.Axes` is the list and `Hand.On(axis)` is one reading. **It pays 1x**, the identity,
so the loader allows a multi-card rung *at* 100 and refuses one below it: what a pair buys is
that two cards are summed where a No Hand lands one. **`combat.Axis` is three values** — concept,
form, element — and a hand can only say what its cards must *agree* on.

**Shields are the whole of the defensive half.** The player's defend cards — `Flinch`, `Brace`,
`Block` and `Guard` at 0/1/2/3 AP — raise that many shields, and **one shield eats one incoming
attack whole**. See MECHANICS.md §Shields. Five things to know before touching any of it:

- **A shield eats the creature's *heaviest* blow, not its first**. `combat.shieldedSlots` is the
  whole rule, and it decides the mask at the top of the creature's turn rather than as each card
  arrives — which is what lets the screen show the exchange before anything swings. Ranked on
  `CardDamage` alone, deliberately: weight, vulnerability and every guard are one multiplier
  over the whole turn and cannot reorder two cards, so projecting the pipeline per card would be
  a second resolver agreeing with the first. **The screen draws it as a broken window** —
  `cards.MarkShattered` for the settled mark, `internal/screens/combat_shatter.go` for the pip
  crossing the table and the crack opening. **A mark is not an upgrade, and the drawing does not
  tell them apart** — both cover the whole face, so what separates them is ownership: an upgrade
  is what a card permanently *is* and is painted into the face, a mark is the card's situation
  and is painted over the top of it. `Render` fixes that order. See MECHANICS.md §Shields and
  §An upgrade washes the whole card.
- **`cards.Mark` is a bitmask and marks compose**. A card can be broken *and* pointed
  at; `internal/cards/mark.go`'s `drawMark` owns the order they are painted in, so one pair of facts
  draws one way. **Append-only, and worse to insert into than an ordinal enum** — claiming a bit in
  the middle changes what every existing value means, not just the ones after it. Three marks today:
  `MarkShattered`, `MarkHighlit` and `MarkPicked` — the last being the deck panel's filter column
  pointing at the cards its figures counted, in the relic pink rather than a sixth hue.
- **The asymmetry is the mechanic.** Only the player raises shields and creatures raise nothing at
  all, because every creature is a solo attacker (`SoloAttacks`, one blow per card) while the
  player forms hands and lands one figure a turn. A count facing a hand would delete a whole turn.
  `combat.blockedByShield` carries the note; the rules do not enforce it.
- **The verb vocabulary is two words: attack and shield.** Nothing banks, nothing draws, and
  nothing shaves a fraction off a blow — **`Duelist.ActionPoints()` is the whole of a turn's
  budget** with nothing that adds to it mid-round, and **every creature deck is pure attack**,
  which is why a creature's whole
  personality is which blows come round how often. `go run ./tools/motifsheet` is where a deck's
  size is read; no figure for it is written down here, because it moves whenever a creature is
  retuned and nothing fails when it does.
- **One card raises at most five shields; a duelist holds as many as the turn paid for.** The
  five is `combat.MaxShields` = `MaxActions`, refused at `RegisterConcept` and clamped in
  `Card.Amount`. **`Duelist.raiseShields` does not clamp the total** — three Guards in a turn is
  nine shields and is meant to be, because what bounds a turn is the action budget. **The pip row
  is a separate number in a separate package**: `screens.maxShieldPips` is `cards.MaxEffects`,
  being what the bottom band fits — so a duelist behind ten draws a full row and the true count is
  on the engine. A row that can say a big number has not been designed.
- **The dealt defenses are Brace and Block; Flinch and Guard ship at zero copies**, so the deck
  holds 2 defend concepts × 5 elements and 55 cards in total. Moving that is a balance change:
  **re-run `tools/handodds` and `tools/seeds` after any edit here**, and read MECHANICS.md §The
  deck is a starting position before drawing a conclusion from either: they describe fight one
  of a relicless run and nothing else.

**Every fight is five rounds long, and the clock is a rule rather than a countdown**. A duelist
still standing at the end of the last round dies, through the same door a killing blow uses.
`combat.Duelist.RoundLimit` is what the resolver checks and `combat.DefaultRoundLimit` is the
five; **zero is no clock at all**, which is what every creature and every bare `Duelist{}` in a
test carries — a default of five in the rules would have put the whole existing suite on a
timer. The run owns the number (`session.Session.RoundLimit`, carried to the fighter by `Equip`,
saved with the run) so a relic or a brand that buys a sixth round has one field to write. See
MECHANICS.md §The round limit. Two things to know before touching it:

- **It is read last, after every other way a round can end.** A win on round five is a win and a
  death to the final blow is a death to the blow — `combat.FightOver` is the guard, and the first
  version of this killed the winner.
- **The bar on the combat screen decides nothing.** It is a picture of `CombatScene.round`; the
  clock is checked inside the resolved round, per presentation-may-never-change-an-outcome. **What
  nothing catches is the balance**: a hard cap makes every fight a damage check and nothing here
  simulates a duel, so a floor whose creatures have outrun what a run can build is unwinnable and
  no test goes red.

**Runes alter the deck *during* a fight, and they are the one mechanic allowed near a live
round**. `data/runes.json` is the catalog, `internal/session/rune.go` validates and applies,
`internal/combat/rider.go` holds the one vocabulary the rules have to read, and
`internal/screens/combat_rune.go` is the run's half — there is no board piece and no dialog. A
rune is a card in the **consumables pane** on the top row (`consumables.go`), and it is aimed by
**selecting the cards in the hand first and clicking the rune second** — the rule that joins the
two is `targeting.go`. See MECHANICS.md §Runes. A handful of things to know before touching any
of it:

- **Between turns, never inside one.** Spending is gated on `planning()`, because `ResolveRound`
  decides a whole round before playback starts and a card altered mid-playback would show a face
  disagreeing with a blow already computed. This is the presentation-may-never-change-an-outcome
  rule meeting the one mechanic that wanted to break it.
- **A card is a form, an element and an action — and then one upgrade.** The first three
  compose freely and a `normal` rune moves one of them; an `upgrade` rune writes the fourth, and
  **whatever was there is gone**. `combat.MaxCardRiders` is **1**, `Card.SetRider` replaces
  rather than stacks, and `data/runes.json` declares `"Change": "normal"|"upgrade"` on every
  record — authored, and refused at load if it disagrees with what its target actually does. Two
  Siphons on one card is one card forgetting the other, not twenty life. See MECHANICS.md
  §Normal and upgrade.
- **`combat.Card.Riders` is a fixed array** because a card must stay comparable — the screen's
  face cache and `TestRoundIsDeterministic` both depend on it, exactly as `Duelist.Relics` does. A
  seat is also what makes "no upgrade" the zero value rather than a case.
- **The card goes gold and the border does not.** `cards.UpgradeStyle` is `wash-face` /
  `border` / `wash` and `DefaultUpgradeStyle` is **`wash-face`** — everything inside the border
  ring washed, the ring left alone. **The border is already saying the card's state**, so an
  upgrade over it would be a second thing in the one place the card says the first; it also keeps
  the card's outline against the table. `tools/upgradesheet` draws all three, as a review knob,
  because how loud an upgrade should be is still open.
- **Every rider draws, and none of them touches the left column.** `systems.Upgrade` is the
  presentation vocabulary — one entry per rider kind — `internal/screens.upgradeForRider` is the
  total table where a rider becomes one, and neither `internal/cards` nor `internal/systems`
  learns what a rider is. The left column states the element and almost no upgrade is about the
  element, so a left column in gold would be the element slot saying something else. **Most of
  the colors are placeholders on a full wheel** — see
  `systems.upgradeTint`, and `go run ./tools/upgradesheet` to retune them. The wildcard is the one
  that keeps a picture for its ink and the one that leaves the form mark hueless.
- **The wildcard is read while the hand is *formed*, not while the turn resolves**, so it lives in
  `matchCountOf` rather than in `playRiders` — the one rider that does. `combat.RiderWildElement`.
- **Gold and silver gamble on every play, and the roll is in the resolver.**
  `combat.RiderGolden` and `RiderSilver`; `combat.Sources` is the struct carrying
  **two** streams into `ResolveRound` — `Roll` for the shock, `Luck` for the gamble — and they are
  never interchanged. The grant moves the fighting duelist *and* announces `KindGrantedDMG` /
  `KindGrantedLife` for `screens.settleGrants` to make permanent on the run; silver goes through the
  purse and needs no event. **A card that gambles on every play is worth however often it is
  played**, which on a cheap starting card is dozens of times a run — the dial is the denominator in
  `data/runes.json`. See MECHANICS.md §Gold and silver.
- **Targets are card identities, not deck positions.** A rune may name two cards and is spent
  while three piles hold copies of the same cards, so `combat.Card.ID` is what makes it
  possible.

**Re-run `tools/handodds` after touching the deck, and read the hand multipliers against what it
prints.** The ladder is priced off how hard each rung is to land — the Pair scores 100% and pays
100, a Card Four of a Kind scores 0.64% and pays 479 — and every one of those figures is a fact
about `data/duelist_cards.json`, the hand size and the action budget. Change any of them and the
ladder is tuned against a deck that no longer exists, silently, because nothing fails.

**It reports two columns and a score, and the difference matters.** **Dealt** is whether a hand
of eight holds the cards for the rung at all; **playable** is whether it
could also pay for them inside the round. They come apart hard on the five-card rungs — a Form Full
House is dealt in 93% of hands and payable in 8% — and the **score** is the geometric mean of the
two, which is what the multipliers are priced against. `go run ./tools/handodds -price` prints what
the curve would charge beside what the file charges and marks every row where they differ. **The
Pair is the one marked row and it is deliberate** — the curve would charge 110 for a certain hand
and the file charges the identity, because the Pair is the ladder's floor rather than a reward — so
a second mark is a real signal. See MECHANICS.md for why neither column
alone can price a ladder. **It counts every card,
defenses included** — they carry an element and a form and join hands like anything
else, bringing no damage with them. MECHANICS.md holds the
table and the rule that turned it into multipliers.

**Re-run `tools/seeds` after touching `data/duelist_cards.json`, `startingDeck` or `handSize`.**
A seed is a fact about one particular deck; change the deck and every cataloged number silently
deals something else. The tool re-checks the catalog before it searches and says which entries
no longer match — a change to the deck size has invalidated every entry at once before. A demo
testing a Three of a Kind against a hand with two Bashes in it is worse than no demo, because it
passes.

**A rarer hand needs a bigger search, and the impossible ones are worth re-checking.** Whether a
hand is dealable at all is arithmetic over the *current* deck — how many copies of a concept there
are, the hand size, the action budget — and all three have moved. **Do that arithmetic against
`data/duelist_cards.json` before concluding a hand cannot be dealt**: a hand wanting five copies of
a concept was impossible until an element grew one to five cards, and the note here saying so was
true for nine days. A hand the tool reports as unfindable usually means the search was too short,
but not always.

**Four build tags, and they compose.** Each selects a different file in its package, so one
configuration can compile while another does not. Vet and build every one you might have
broken:

```powershell
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go run -tags "debugtrace idleexit" .    # traced and self-closing: the unattended run
```

**`demoplay` is how the combat screen gets looked at without anybody sitting at it.** It plays a
scripted round or two — selection, DUEL!, playback — and writes the screen to `demo/*.png`, then
closes. It exists because the screen is the one thing `go test` cannot check: a hand line, a
marked verb, a highlight on the right row are all things you have to *see*. It is the
`tools/marksheet` idea applied to a live screen, and the same rule applies — a stale picture is
worse than none, so regenerate rather than trust an old capture. `demo/` is gitignored; fifty
near-identical PNGs are not a diff anyone wants.

```powershell
go test ./...                                   # all tests
go test ./internal/combat -run TestName         # a single test
git commit -s                                   # sign-off, per CONTRIBUTING.md
```

Tests live in `internal/combat` — the only package that can be tested without a
window, by design. Keep it that way: rules go in `combat`, not in screens.

**"Needs no window" is not the same as "needs no display server", and CI found the difference
the hard way.** On Linux, `ebiten/internal/ui` calls `glfw.Init()` from a package `init()`, so
*linking* Ebitengine into a test binary is enough to panic on a missing `DISPLAY` — before a
single test function runs. Four of the tested packages link it: `internal/screens` and
`internal/models` directly, and `internal/cards` and `internal/music` because their tests import
`assets`, which hands back `*ebiten.Image`. `internal/combat`, `internal/session` and the rest
are genuinely clean. Both workflows therefore run the Linux test step under `xvfb-run -a`, which
supplies a throwaway X server nothing ever draws to. Windows is unaffected — Ebitengine is pure
Go there. **If a package's tests start importing `assets`, they have joined that group**; the
package's own no-Ebitengine rule still holds and is still worth holding, but it no longer buys a
display-free test run.

**What cannot be unit-tested gets a tool instead.** `internal/screens` needs a window, so anything
it decides is checked by launching the game, and the sheets under `docs/sheets/` are how the
catalogs get reviewed.

**Nothing simulates a duel.** An unwinnable enemy is invisible while playing, because losing slowly
looks exactly like losing to bad draws — so a cost, a stat line or a planner can be changed today
without anything catching what it did. `internal/combat`, `internal/decks` and `internal/pyramid`
are all free of Ebitengine, which is what would let a headless simulation be written. Keep them
that way.

## Releasing — `.github/workflows`

**CI** runs on every PR, on **Windows and Linux**, and vets and builds under **every build tag** —
untagged, `debugtrace`, `idleexit`, `demoplay` and `scenario`. **Every tag has to be in that
list**: each selects a file the untagged build never compiles, so a tag left out is a break that
stays green in CI and is found by hand.
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
  traveling with the binary the moment it is renamed.
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
- **Work on `up-N`**, incrementing every PR. Branch off `main`, never commit to it.
- **Leave work unstaged.** The owner reviews diffs in VS Code. Do not commit, push or open a
  PR without being asked for that specific step.

## Determinism — see the `randomness` skill

Runs will eventually be **replayable from a seed**, and **combat is already stochastic** —
lightning rolls, and the gold and silver upgrades — so the rules that protect replayability are
live rather than theoretical, and they are easy to break without noticing.

The procedure, the stream table and the argument a new roll has to make live in
[.claude/skills/randomness/SKILL.md](.claude/skills/randomness/SKILL.md). **Load it before
adding any roll, adding or seeding a stream, touching a salt or a seed, writing a shuffle, or
deciding whether a mechanic should be random at all.**

Four things stay here, because they are the tripwire — the failure is not knowing the skill
exists:

- **A run seed is a six-character code** — `internal/seeds/code.go`,
  the only place the alphabet exists. `GlobalState.RunSeed` is still an `int64` because every
  stream derives from it by arithmetic, but it is always inside `seeds.Space` (32^6, about 1.07
  billion runs) so it can be written down. **The alphabet is Crockford base32** — no `I`, `L`,
  `O` or `U` — because a code is transcribed by eye and `0`/`O` and `1`/`I`/`L` fail *quietly*,
  landing on a different valid run rather than an error; `U` goes so a random code cannot spell
  something. **`Parse` still folds `O`→`0` and `I`/`L`→`1`**, since a person typing those meant
  the digit; `U` does not fold. Case is not information: `Parse` takes either, `Code` emits
  upper. **`fixedRunSeed` and a scenario's `Seed` are both written as codes**, and one that is
  not a code fails the launch rather than quietly rolling a fresh run. **Zero is the run
  `000000`, not "unset"** — nothing may test `RunSeed == 0` to mean "no seed".
- **Never call the `math/rand` package-level functions** (`rand.Intn`, `rand.Shuffle`, …).
  They draw from a global source shared with every other caller, which makes a run
  unreproducible. Randomness comes from an explicit `*rand.Rand` carried on state.
- **Every consumer gets its own salted stream off `GlobalState.RunSeed`**, and a stream is
  only ever advanced by its own concern. Sharing one means a change to either silently rerolls
  the other. The skill's table says which are live, and `seeds.All()` is the list the tests
  walk.
- **No `time.Now()` in game rules, and never let map iteration order decide anything.** Go
  randomizes map order deliberately; iterate a sorted key slice.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before
  playback begins, so animation speed, the player's game-speed setting and any skip button may alter
  pacing and must not alter results. Same constraint as the debug flags, `internal/trace`,
  `internal/idle` and the scripted demo.

**Rewrite a random-sounding rule rather than let it in.** Lightning is the deliberate
exception, not the precedent, and a second roll needs its own argument in `MECHANICS.md` — the
skill says what the first one cost.

## The combat screen — see the `combat-screen` skill

Its layout, its card and action-box widget, its hidden information, and the resolution-order
rule the screen has to obey all live in
[.claude/skills/combat-screen/SKILL.md](.claude/skills/combat-screen/SKILL.md). **Load it
before touching any of the combat screen's files — `internal/screens/combat*.go` — or
`internal/ui`, or `internal/combat`, or anything about how a round is drawn or played back.**
**A symbol the skill names may be in `internal/ui`** rather than `internal/screens` — the
drawing layer is its own package, so a grep over both is the way to find one.

It is a skill because it is the screen under active construction: it grows every session
while mattering only when that screen is the work. The general UI conventions below still
apply to it and stay here.

Two things worth knowing without opening it:

- **`internal/combat` decides rounds; the screen only replays them.** Never change the rules
  to make a screen look right — say so and let the owner decide which one is wrong.
- **`combat.ResolutionOrder` is the single authority on play order**, and both `ResolveRound`
  and the table's two rows read it rather than deriving their own.

## UI: clicks and drag-and-drop, and a gamepad has to be able to do all of it

**The game ships on Steam Deck**, so every control has to be reachable with a standard gamepad and
nothing may be designed that only a pointer can do. What is *built* today is the pointer
vocabulary below, and building against it is correct — the controller layer is a refactor that
happens before release, not a thing each new screen bolts on. What each new screen owes it is one
question answered while the screen is being designed: **could a focus ring walk this?** A control
that only answers to a cursor's position, an action with no discrete step, or an ordering that
exists only as a drag is a screen that will have to be redesigned rather than adapted. See the
platform-readiness section of [TODO.md](TODO.md) for the ticket and its constraints.

**Two rules the controller work does not get to bend.** There is no virtual cursor — a pad drives
focus, never a hidden mouse. And a semantic action reaches only controls that are visible on the
screen, exactly as a key does.

These apply everywhere, the combat screen included. The pointer vocabulary is:

- **Left click** — buttons and selection.
- **Drag and drop** — the action box, and anything else that needs ordering or moving.
- **Hover** — rest the cursor on something and a tooltip explains it. A card's
  damage arithmetic term by term, a relic's rule, a status badge's meaning. `models.Tooltip` and
  `systems.DrawTooltip` are the widget; the wording is `internal/ui/tips.go`.
- **Long press** — the same reveal, for a touchscreen or a controller, where there is no cursor to
  rest. **Not built**, and it is the only reason hover did not simply replace it: see MECHANICS.md
  §Hover and long press, where the record of hover being *rejected* was reversed.
- **A key may be a shortcut for a button that is on the screen, and nothing else**. Escape
  presses the settings cog — `internal/game/chrome.go` — and it is live exactly when that button
  is, gated on the same `chromeShowing` predicate and the same `gs.InputGated` shield. **A key
  that does something no visible control does is forbidden**, and so is anything that makes the
  keyboard *required*: a shortcut is a faster way to reach a control a player could always have
  clicked. A player who never touches the keyboard misses nothing.
- **One typed-text field in the whole game** — entering a seed to replay a run. Nothing
  else anywhere accepts typed input.

**No right click, ever.** There is no context menu and no secondary action. Anything
that feels like it wants one needs a different design. **A gamepad's spare face buttons are not a
way back in**: a pad gets the reveal a cursor gets by hovering and an explicit mode for the
ordering a cursor gets by dragging, and neither is a second thing a control does.

- **Wanting a text field is a design smell.** Find the click or drag version instead.
  A settings value is a row of buttons or a slider, never a number you type.

### The game boots to the title screen, and the title screen owns the run

The menu *decides something* — **New Run** or **Continue** — which is a question nothing else in
the game asks, and a run that started before the player was asked is a run they cannot decline.

- **`internal/screens/run.go` is the run's whole lifecycle**: `BootRun` (resume off disk or build
  fresh), `NewRun`, `ContinueRun`, `AbandonRun`. **They live in `screens` because a screen cannot
  import `main`** — once starting a run is a button, building one is something a screen does.
  `main` rolls a seed and calls `BootRun`.
- **`state.SeedPinned` says the seed was chosen rather than rolled**, so `NewRun` does not silently
  break `fixedRunSeed` or a scenario's `Seed` from a menu button. **The tutorial still outranks
  it** — a taught run is dealt the script's own code.
- **`Continue` is gated on `gs.Resumed`.**
- **`demoplay` needed a door.** The scripted demo drives the combat scene rather than navigating to
  it, so a menu in front of the duel is a menu it sits on forever. `screens.DemoPlaysItself` is the
  predicate, in the usual two-file `_on`/`_off` shape, read by `BootRun`.

**Abandon Run is on the settings screen** — the one screen reachable from everywhere. That does file
a run decision under the program's screen, which is the distinction `settings.go`'s own doc comment
draws; the objection was raised and overruled, and the cost is paid in the layout rather than
pretended away. It is the only thing on that screen that touches `gs.Run`, and it does so by calling
`AbandonRun`.

**A death is the same event, so it is the same function.** `EndRunInDefeat` is `AbandonRun` under a
second name, because the only difference between dying and giving up is which screen the player was
standing on — and two paths from "this climb is finished" to "the file is gone" is one path that can
be got wrong. **There is no retry**, because a roguelike where a death can be taken back is not
one. `CombatScene.died` is the path and the button in the DUEL! slot reads `defeatButtonLabel`.

**A run ends on a splash, not on the title screen**. `screens.endRun` is the one door: it takes
`gs.Run.Summarize(gs.RunSeed, ended)` **before** deleting anything, puts it on `gs.Summary`,
then nils the run and goes to `state.RunOver`. The ordering is the whole thing — the numbers
come off the ledger the run was carrying, and a clear-then-summarize leaves a blank page that
nothing fails on. `session.RunSummary` is plain ints and strings so the screen never learns what
a `Session` is; **it is the one piece of state that deliberately outlives what it describes**,
and `RunOverScene.leave` drops it.

**The run code is on that splash and in the settings screen's bottom-right corner**, because a
six-character code that exists to be transcribed has to be somewhere a player can read it.
`screens.abandonLabel` is what puts it on the settings screen, and it names no code with no run
standing.

### Five screens that are not stations of a run

**Settings, Achievements, Credits, RunOver and Goods.** None appears in `screens/flow.go` —
which is what "not a station" means mechanically — and the chrome stands down on all five. The
first three are reached by an `actions.Open*` call and record `gs.ReturnScreen` so Back works
from anywhere; **RunOver is the exception and goes to the title outright**, because the screen
it came from was drawing a run that has ended and there is nowhere to put the player back to.

**Goods is the fifth and it is the one with a door of its own**. It is a sealed good, opened:
the three to five things inside it, and the one of them the player takes. It is reached from
exactly one place and returns to exactly one place — the shop's shelf — so `screens.openGoods`
sits beside the screen rather than in `actions`, whose explicit list is about screens openable
from anywhere. **There is no way out but taking a card**, because the good is already paid for;
the good travels as `gs.PendingGood`, a record key, since a screen cannot be handed an argument.
See `internal/screens/goods.go`.

**What it cost the shop is a re-entry guard.** Leaving for a good and coming back runs
`ShopScene.Init` again, and a second deal would restock the shelf, forget which goods had been
opened, un-drink the potions and replay the shopkeeper. `ShopScene.visit` is the run and the fight
the scene's state belongs to, and a matching one is picked up rather than dealt again — read it
before adding anything to that Init.

- **Adding one is two edits, not three**: an ordinal in `state.ActiveScreen` (append-only, and
  its `String` case) and an entry in the registry in `internal/game`. There is no phase, because
  there is no station. A screen the chrome should stand down on is a third: `chromeShowing` in
  `internal/game/chrome.go`.
- **`actions` has one function per screen rather than one taking a destination.** A shared
  `openScreen(gs, dest)` would be shorter and would also be the seam through which a *run*
  screen gets opened without its phase being set. The explicit list is what says which screens
  may work this way.
- **The achievements catalog is `data/achievements.json`**, and it earns a loader because a
  record has to say *what earns it* — a name and a sentence would not have. `internal/achieve`
  is the loader and the validator; the screen draws what it hands over and decides nothing. See
  MECHANICS.md §Achievements.
- **`TitleScene.menu()` is the one list the title menu is built from.** Init, Update and Draw all
  read it, because three hand-written orders are three places a new entry gets forgotten — which is
  how a button ends up drawn and not clickable.

### The third dialog shape: a question, not a view

`internal/ui/confirm.go`. The first two shapes are the near-full-screen `modalToggle` panel and
the tutorial's bubble; this is a small centered box with two answers, and it is deliberate rather
than a drift.

- **A confirm is a question, not a page.** A modal takes the screen because what it holds *is* a
  page — a whole deck, a ladder, a run's account. A dialog that covers the screen to ask six
  words reads as something having gone wrong, and it hides the thing being asked about.
- **It stays in the family**: same scrim, same bevelled panel, same pink stroke, and the
  destructive answer takes `modalCloseColor` — the only red in the game. It does **not** borrow
  the X: an X means "put this away", and a question with one has three answers where it should
  have two.
- **The callbacks are rebound every frame**, because one dialog serves two callers and a callback
  wired once at build time answers the previous caller's question.
- **Cancel is the safe answer and it is on the left.**

### The settings screen, and the second widget in the game

[internal/screens/settings.go](internal/screens/settings.go) is the program's own screen: a music
volume bar, a game-speed bar, and Back. `models.Slider` + `systems.UpdateSlider`/`DrawSlider` is
the widget, built the same way `models.Button` is and following the same rules — one named color,
a bevelled face, a cached image repainted only when something visible changed.

- **It is reached from the cog in the game's chrome, from any screen**, so it is the only screen
  that cannot name its successor: `state.ReturnScreen` is where Back goes, recorded by whoever
  opened it. **It never touches `session.Phase`** — settings is not a station of a run, so opening
  it mid-climb is a look at a dialog rather than a decision.
- **Adding it was the usual three edits minus one.** No `session.Phase` and no entry in
  `screens/flow.go`, because it is not part of the run loop; just the `ActiveScreen` and the
  registry in `internal/game`.
- **A slider is 0..1 and knows nothing about what it is setting.** The scene maps that onto the
  range — `speedFor`/`speedValue` against `profile.SpeedMin`..`SpeedMax`. A widget carrying its
  own bounds would put a game decision inside something every screen shares.
- **`OnChange` fires while dragging and `OnCommit` once on release.** That split is what makes a
  volume bar audible under the cursor while costing one write to disk rather than a hundred.
- **The travel is inset by half a knob at each end.** Without it a value of 1 needs the cursor
  past the control's own right edge, so a full-width bar can never be turned all the way up —
  `TestBothEndsOfTheTravelAreReachable`.
- **`models.Slider.Ink` exists because the game has two grounds.** The near-white default is for a
  dark screen; this one is painted on `screenGround` and passes `groundInk`. Same reason
  `ColorToward` exists beside `ColorAtStrength`.
- **The sounds bar is deliberately absent.** There is no sound system yet, and a slider setting a
  number nothing reads is a control that lies about what it does.
### Cards fly; they never appear

**A card that changes where it is on screen travels there**. Drawn, discarded,
played to the table, re-sorted in the hand, won as a prize — every one of those is a journey with
a start, a duration and an eased arrival, never a card in one place on one frame and another place
on the next.

It is a rule rather than a flourish because of what a card *is* here: the same object moving
between piles that the player is tracking by eye. A card that appears in the middle of the screen
is a card that was never anywhere else, so the player has to re-read it to find out what happened
instead of having watched it happen.

- **`internal/screens/combat_flight.go` is the pattern**, and `travel` — a delay, an age, a
  duration, an eased progress — is the clock every mover shares. The post-battle screen uses the
  same one for the won card's flight to the center.
- **Ease out, so a card leaves quickly and lands gently.** `easeOut` is what makes an arrival read
  as landing rather than as stopping.
- **A flight is raised after the model has already moved**, so it is a ghost of something that has
  happened. That is what keeps the state machine ignorant of animation — see `spendSelected`.
- **It may never change an outcome**, the same constraint as playback speed and the debug flags. A
  flight is something to look at.
- **The exception is an absence**: a removed card has nothing to fly, so the seat it would have
  landed in is drawn empty.

**A hand arrives in three stages, and every hand in a fight arrives the same way.**
`internal/screens/combat_deal.go` is the sequence: cards fly out of the pile left to right **in
pile order**, then the flip cascade plays **one beat per worn ring** over them, then the row
**sorts itself**. The opening hand goes through it like every other, so no hand in the game
simply appears. Five things follow:

- **The sort is last.** Sorting before anything is animated would fly each dealt card straight
  to its final slot — one journey per card, and a hand that never shows the player what the
  shuffle gave them. The cost is a second movement per card; what it buys is the deal having
  something to say.
- **The hand holds the finished cards from the first frame.** What the
  deal owns is the **faces** — the pile's, then one per ring — so a card selected while it is still
  showing its lightning face is the earth card the engine will score. That is `shownLife`'s division
  applied to a card: the model moves first and the drawing catches up.
- **One beat for a whole ring, not one per card.** The shield break's rule and the rune's: eight
  cards changing one after another is eight pauses over a hand the player is waiting to play, and
  what the beat says is one thing about the relic rather than eight about cards.
- **The ring toasts while its cards change** — it rattles, rocks left then right, and its border
  lights, which is the same toast the sum already gives a relic that is firing. **Two clocks on
  purpose**, since one is playback and the other is a hand arriving and they cannot both be
  running. `screens.relicToast` is the gesture and it carries all three marks together, so a
  fourth cannot reach the sum's toast and miss the cascade's.
- **A ring that touches nothing in this hand is not a beat.** A rattle over a row where nothing
  changes is the screen saying a relic fired when it did not.

### Cards change in front of you, too

**A card that becomes a different card dissolves into it**. Same
argument as the flight one axis over: a card that changes between two frames has to be *re-read* to
find out what happened, instead of having been watched happening. `internal/ui/cardmorph.go` is
the machinery and `internal/cards/dissolve.go` is the pattern the face comes apart in.

- **A morph is two finished faces and a clock**, and it knows nothing about where it is on
  screen — the caller owns the rectangle, exactly as it does for a `travel`. That is what lets
  the post-battle screen run one in the middle of the table and the combat screen run several in
  the hand.
- **Which of two faces it holds is what it does.** `morphInto` replaces, `morphAway` eats,
  `morphIn` arrives out of nothing. **There is no style enum**, because an enum beside the faces
  is a second way of saying the same thing and a way for the two to disagree.
- **It is not a `cards.Mark`.** A mark is the card's situation and the card is still itself
  afterwards, so it has a settled picture `internal/cards` can bake. A morph ends as a *different
  card*, so there is nothing to bake and no second rasterizer — what that package owns is the
  geometry, on the same terms it owns `ShatterCracks`.
- **The pattern is derived from the card's name, never rolled**, exactly like the crack pattern and
  as the explicit exception the randomness skill records. So one card always comes apart the same
  way, through a resize, an interruption or a re-entry.
- **It reads as patches because the delays come off a smooth lattice.** Independent per-square
  delays pass every other test and look like television static;
  `TestNeighboringSquaresGoTogether` is the tripwire.
- **The flare on a turning square is light, not a hue.** The square is drawn a second time
  additively at its peak, so it brightens in its own colors — the wheel is full and a burning edge
  in a fifth hue would be claiming one.
- **The card lands first, is still for a beat, and then changes.** A dissolve running over a moving
  card puts the one thing worth watching on a target the eye is still chasing. `morphWaitTicks` is
  the pause and it is the travel's own delay, so one clock is the whole transition.
- **It may never change an outcome.** The post-battle screen still holds the real deck edit in
  `applyNow` until the stage is over; the morph is a picture of a decision already taken.

**Two callers, and the second is the hand.** `internal/screens/combat_handmorph.go` is a rune
changing cards where they stand, mid-fight.

- **What changed is read off the faces, not off the rune.** `handFaces` is taken before the
  apply and again after, and the three shapes fall out of the comparison: a face that differs is a
  replacement, a card that has appeared is a copy, a card that has gone was eaten. **No rune has
  a case anywhere in the drawing**, which is what stops a new one arriving with no picture.
- **One beat for all of them**. Every card a
  rune took changes at once; three dissolves in sequence would be three pauses over a hand the
  player is building.
- **A morph carries the card's identity, never a seat index**, so a sort or a drag under a running
  one moves the picture with the card. The captured rectangle is the fallback for a card that is no
  longer in the hand — the eaten case, which has no seat to look up.
- **The row draws its own**, as a fourth suppression beside `inboundTo`, `resolvedInHand` and
  `slidingTo`, and it is checked last: a morph still running when DUEL! is pressed gives way to the
  round rather than painting a second copy of the card.
- **Nothing waits for it.** A rune is spent while planning, so the card under a running morph is
  already the new card and is selectable while it changes — the same rule that makes a flying card
  clickable. It is deliberately not in `combatTheater.running()`, which is the playback cursor's
  question.

- **No UI toolkit dependency.** Widgets are hand-rolled following the
  `models.Button` + `systems.UpdateButton`/`DrawButton` split. Add new widgets the same
  way: a plain struct in `models`, behavior in `systems`, owned by the scene that uses
  it.
- **ebitenui was evaluated and declined.** Everything the game needs is a *game* widget,
  which is where general-purpose toolkits are weakest, and a toolkit is one more dependency
  to license-check against a product that will be sold. **The one trigger for revisiting is
  the seed text field** — a text input with a caret, selection and clipboard is the single
  widget genuinely cheaper to take than to build.

The action box is a *game* widget, not a UI widget: draggable action cards with live
action-point validation. General-purpose toolkits are weakest at exactly that, so
hand-rolling costs little and buys full control.

### Interface art is authored, and there is no glyph generator

**Every mark the interface draws is a file, not code.** There is **one authored drawing per form
per element, plus a neutral set, plus one cost tick per element** — a multiplication rather than
a list, which is exactly why it is not an enum: thirty-odd append-only enum values would be
thirty-odd cache slots naming pictures the rules know nothing about.

**The provenance argument is what a generator was for, and it is answered differently**: the art
is generated by an image model from a prompt this repo owns, `docs/art/`, rather than drawn by a
third party — so there is still nothing to clear, and the pictures are files rather than code.
**Do not reintroduce a silhouette generator.** A derived one-pixel rim means a smaller glyph is a
*different drawing*, where a painting has interior detail to average and survives both
reductions.

**`systems.ArtMark` is the whole of how art reaches the drawing** — `internal/systems/artmark.go`,
keyed by **asset name** rather than by an enum, cached, and free of Ebitengine so the review sheets
can call it. `ArtMarkImage` is the texture-shaped door for a screen. `go run ./tools/marksheet` is
the page it is all reviewed on.

- **There is no fallback anywhere.** `cards.MarkArtKey` and `cards.TickArtKey` are *total* — every
  form and every element answers a key, with anything that is not one of the five elements
  answering the neutral drawing — so a blank corner means a missing file rather than a card with no
  element. A key naming nothing comes back nil and draws nothing, which is the honest failure.
- **Nothing is tinted.** The hue is in the drawing. `cards.tintInk` is gone and so is the shield
  pip's multiply; `shieldFlight` and `shieldRow` carry a `cards.Element` where they carried a
  `color.RGBA`, which is what lets a pip draw the shield its card was showing.
- **A card's corner carries a drawn form mark** — a spear, a scimitar, a club and a shield for
  stab, slash, crush and defend, from `assets/form/`. `MarkArtKey` answers `""` for `FormNone`, so
  a relic and both fighter cards leave the slot empty.
- **Art is committed at 256x256** (640x256 for a tick) and the game reduces it to the 32 and the 16
  it draws at — one averaging step from a rich source rather than two from a thin one, which is
  what keeps the near-black contour alive at card size. A batch delivered at 64 arrived with no
  contour left at all and it was invisible by eye.

- **The card is 162x224, and it is a column and a paragraph.** The form mark sits in a 32px
  box at (10,8) — **inside the card, not hanging off the corner**, because a mark carrying
  detail loses it to the card's curve where a plain silhouette would survive the crop; under it
  the cost dashes
  make a **26px column**; and **the effect text takes everything right of that**, centered in it
  both ways, at 18pt. `blitGlyph` clips whatever it composites to the rounded shape, which is what
  keeps a mark placed hard into a corner from squaring the card off.
- **The damage badge carries the card's multiplier, and the numeral is drawn into the art.**
  `cards.BadgeArtKey` picks it by shape, value and element out of `assets/damage/`; a value with
  no drawn badge falls back to the blank badge with its figure printed on top, which is the one
  fallback in the interface art. **Which outline it takes is still open** —
  `cards.DefaultBadgeShape` is the one line that picks, and `go run ./tools/badgesheet` is the
  matrix it is chosen on. `cards.Spec` still has no `Damage` field: the badge says what the
  *card* multiplies by, never what the wielder would deal with it.
- **A card's picture is a function of the card *and who is holding it*.** A slash in the hands
  of someone wearing Keen must not read "2x DMG" and deal four times their DMG — the card's
  multiplier and the relic's scaling are applied in different places. `screens.held` is the
  pairing — cost, DMG and worn relics, traveling together — and **the figure a relic has moved
  is written in the relic pink**, via `Spec.TextInk` and `Spec.TextHighlight`, which colors that
  run of the line and not the sentence around it: a pink verb would say the relic changed the
  card rather than the number. The *damage* is still not printed: the face carries the
  multiplier and the tooltip carries the arithmetic.
- **The wording is the constraint now, not the space.** The text column is ~128px — a dozen or
  so characters a line — so effect text has to be short words, and `DMG` rather than `damage`.
  `TestNoEffectTextWordIsWiderThanItsColumn` fails on a word that will not fit and
  `TestEveryCardTextFitsItsBand` on a string that wraps past the band;
  `TestLeftColumnDoesNotCollide` and `TestTheCostColumnStaysOutOfTheTextColumn` hold the column
  against its neighbors.
- **A `\n` in effect text is an authored line break**, honored by
  `cards.WrapText` before the width is measured, and split back into lines by the tooltip.
  It exists because width-wrapping cannot make a *set* of cards break in the same place: the
  five elemental essences differ only in the element they name, and `FIRE` sits comfortably on
  the line where `LIGHTNING` all but fills it, so left to the measurer the four read as four
  layouts of one card. **A break can only ever add a line** — an authored line too wide for
  the band still wraps — so it is not a way past the column.
- **A glyph may be placed at a negative offset** to hang off an edge. `blitGlyph` clips it to
  the rounded silhouette via `insideRounded`, and `fadeRegion` skips transparent pixels — both
  so a corner glyph cannot fill in the transparent corner and square the card off.
- **`systems.ArtMark` returns a plain Go image and is free of Ebitengine on purpose** — creating
  an `*ebiten.Image` needs a graphics context and the review tools have no window. `ArtMarkImage`
  wraps and caches it for a screen.

**`go run ./tools/marksheet` is where every mark in the game is reviewed** — the form marks by
element, the neutral set and the cost ticks, all on one page at the sizes the game draws them
and enlarged. The damage badges have their own matrix, `go run ./tools/badgesheet`.

**The question the page exists for is the matrix.** A player counts a hand by comparing four
silhouettes along a row and five hues down a column, so the two failures are a form that cannot be
told from its neighbour and an element that cannot be told from its neighbour — and neither is
visible one drawing at a time. It prints, under every cell, the ink's drawn size, the opaque pixel
count and **what share of those pixels are near-black**: that last figure is the contour, and a
batch arrived with it at zero once and was invisible by eye at review size.

**Actual size before enlarged, always.** The rule the old sheet established and the one thing this
page inherits wholesale: reviewing only the blown-up row is how a mark comes to look acceptable in
review and clunky in play. There is no zoom anywhere in the game.

### Audio is generated too, and for the same reason

[internal/music](internal/music) plays the score. `assets/ascending.mid` is a **Standard
MIDI File — a kilobyte of notes** — and `internal/music` synthesizes it to PCM once at
startup. `main.go` starts it after assets load; it loops for the whole session across
every screen. Editing the tune means editing the MIDI file. **Nothing is baked and there
is no build step.**

- **Ebitengine cannot play MIDI.** Its audio package decodes MP3, Ogg Vorbis and WAV,
  and that is all. The three ways past that were converting to Ogg offline, embedding a
  SoundFont plus a synthesizer, or generating the audio in Go.
- **The third was chosen for the glyph argument**: generated output has no provenance
  question. A SoundFont is megabytes with a license to clear, and a rendered Ogg carries
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
  card still plays the game — and `music.Available()` reports it, so the volume bar on the
  settings screen disables itself rather than silently doing nothing.
- **There is no mute, only a level**. `SetLevel(0..1)` is the whole
  control and zero is the only silence there is; the mute latch went because a latch and a bar
  are two controls over one number that then have to be kept from disagreeing. The bar is on the
  settings screen — a control, never a hotkey, since the input vocabulary has no keyboard.
- **`fullVolume` is the ceiling the bar's 1 actually means**, and it is a third of the device's
  range because this is background music under combat sounds that do not exist yet. "How loud is
  the score allowed to get" stays one decision in `internal/music` rather than a figure typed
  into a scene.
- **Volume, not Pause.** Pausing would hold the score at the bar it was on, so coming back from
  silence mid-duel would drop the player into a phrase they had already heard. A track that keeps
  running underneath puts them wherever the music would have got to.
- **The game boots silent for a new player** — a fresh `profile.Settings` has `MusicVolume: 0`,
  and `main` applies the saved settings *before* `Start` opens the device, so a returning player
  gets back the level they chose rather than a moment of the wrong one. Music that begins on its
  own is the first thing a new player reaches for a control to stop.

### The frame: the two controls that belong to no screen

[internal/game/chrome.go](internal/game/chrome.go) draws the **settings button** — a 44px
square in the bottom-left corner of every screen, carrying a generated cog — and, beside it, the
**ledger button**: the run's account of itself, on every screen. See MECHANICS.md §The ledger,
`internal/screens/ledger.go` for the panel and `internal/session/ledger.go` for what it holds.

**The third thing in the frame is the achievement toast.** It is not a control — it is the game
telling the player they did something, and waiting to be clicked out of. It qualifies on the
same three tests and could not be a scene's for the ledger's reason plus one of its own: an
achievement can land during a duel, on the post-battle screen, or on the transition between
them. Like the ledger it takes the frame while it is up and the active scene is not updated at
all. **Its queue is `state.EarnedThisSession`**, written wherever an award happens in
`internal/screens` and drained one box at a time — a five-element turn earns three achievements
together.

**The ledger is chrome for the usual three reasons and one it does not share**: it is true for the
whole run, wanted on every screen and owned by no scene — and unlike the settings it *could not*
have been a screen, because leaving the combat screen and coming back re-runs `Init`, which deals a
fresh duel. A ledger that navigated would destroy the fight it was opened to read about. While its
panel is up, `internal/game` does not update the active scene at all; that freezes pacing and, like
every other dialog, cannot change an outcome.

**The cog opens the settings screen and nothing else.** The corner does not mute — there is no
mute anywhere, only a level — and what it buys instead is one place for the game speed and the
volume to live together.

- **The third widget is `models.Scrollbar`**, built the way
  `models.Button` and `models.Slider` are — a plain struct in `models`, behavior in `systems`. It
  **counts rows, not pixels**, so a panel cannot land half a line off, and it is a drag because the
  input vocabulary has no wheel and adding one would be a fourth verb rather than a widget.
- **It is deliberately outside "scenes own their own widgets" rather than an exception to
  it.** The score is started once in `main` and loops for the whole session across every
  screen, and the game's one clock is the same number on every screen, so the control that
  opens both belongs at the same level. The alternative was the same button on six scenes:
  six placements to keep in step and six callbacks into one package.
- **The bar for joining the frame is high, and the file says so.** Something true for the
  whole session, on every screen, owned by no scene. A frame is easy to grow by accident.
- **`state.ModalOpen` is what it cost.** A scene sets it while it has a dialog up and the
  chrome neither updates nor draws — otherwise the button would sit live on top of the deck
  overlay, whose whole design is that the control closing it is the only lit thing on
  screen. **The frame clears the flag each tick and the scene re-asserts it**, so leaving a
  screen with its overlay open cannot hide the chrome for the rest of the session.
- **It also stands down on the settings screen itself**, which is the one screen where the corner
  would be a door into the room the player is already standing in. `chromeShowing` is the one
  predicate both halves ask; that screen carries its own Back button.
- **Never disabled.** The mute button it replaced was, on a machine whose audio device would not
  open. This one opens a screen, which always works — it is the *volume bar* on that screen that
  goes dead, and it says why underneath itself rather than merely going gray.
- **Square and iconic because the corner is 52 pixels wide** on the combat screen — the hand
  band starts at x=52 and the action-point figure sits on its left edge, so a labeled
  button does not fit beside them.
- **The cog is `assets/game/gear.png`, and it has eight teeth: four on the axes, four on the
  diagonals**. Four was tried first and read as a compass rose — at
  32 pixels a gear is recognized by the *count* of its teeth before any one of them is legible,
  and the hole in the middle is what makes it a cog rather than a flower. That is a constraint on
  any replacement drawing, not a description of how this one was made.

### Cards: the left column carries the element, the border carries state

A card is a **constant off-white surface** (`cards.Surface`) with a **neutral gray border**, and
the element is said by **the left column — the form mark and the cost ticks under it**
. Both are authored per element and neither is tinted. The whole card is drawn by
`internal/cards`. Five things follow and are easy to re-break:

- **The border is not the element and must not drift back to being one.**
  `cards.borderBase` is what a border is actually drawn from and it returns the same gray for
  every element; `cards.BorderOf` still holds the element colors and is still what the mark, the
  deck panel's row labels and the arithmetic panel read. `TestTheBorderIsTheSameWhateverTheElement`
  fails if an element gets its border back. The argument: a border is the loudest thing on a
  card, and spending it on the one fact the player already knows from the row the card is in
  leaves the corner mark — the thing a hand is counted on — hueless.
- **Relic keeps its pink border.** Pink was never an element; it is the "you cannot play this"
  signal, and `TestARelicStillBordersPink` holds it against a change that neutralizes the four.
- **The ticks are the element too, and share the border's state.** A tick is a *drawing* and
  cannot be handed a color, so what the two share is the **distance** — `Spec.atState` moves the
  border toward the surface and `tickFade`/`tickFadeTarget` walk the drawn tick exactly as far,
  toward the same one. A second copy of that switch is how a selected card ends up with a lit
  border and resting ticks, and how a disabled card fades its ticks toward the wrong surface;
  `TestTheTicksAndTheBorderShareOneState` fails on both and caught the second one.
- **Nothing in the left column is tinted.** Every form mark and every cost tick is authored in
  its element, so the hue is in the art: `MarkArtKey` and `TickArtKey` pick a drawing rather
  than a color.
- **A near-white border on an off-white card is invisible.** `basic` is therefore a mid gray
  in `cards.BorderOf`, and a test fails if it is set to a near-white. It is also the color every
  card's border now draws in.
- **`ColorAtStrength` is the wrong tool on a light card.** It scales toward *black*, which
  reads as quieter only against a dark ground. On
  an off-white card a border scaled down comes out darker than the surface and therefore
  *louder* than the live card beside it, which is how a pane's idle rows end up in front of
  its lit one. Use `systems.ColorToward(c, ground, pct)`, which moves a color
  toward whatever it actually sits on. Card state is expressed as distance to the surface.
- **Cost is tick marks and the form is a corner mark**, not text and not a numeral. **A tick is
  16x4**, so four of them stack in 31 pixels and the cost column ends well up the face. Every
  card in the game runs 1..3, the player's and every enemy's; a fourth tick grows the stack
  further down the card and is a layout change, not just a bigger number.
  `TestLeftColumnDoesNotCollide` fails rather than rendering it. **A card declares its own
  cost now**, so nothing stops a data file writing 5 — which is a reason to
  read this line before authoring one, not a reason for the renderer to clamp.

**A card's picture is either a panel on it or the whole of it, and `Style.ArtBleed` is which**
. `internal/cards/bleed.go` owns the second path: the art is scaled to
*cover* the card, clipped to the border's inner curve, and drawn first with everything else on
top. `RelicStyle` and `EssenceStyle` bleed — so relics, runes, essences, stones and the two
sealed goods are all one format — and `EnemyStyle` and `DuelistStyle` still fit a picture into
`ArtTop`/`ArtInset`/`ArtMaxH`. The two do not compose, and the art is authored against the choice:
a fitted box wants a square and a bleeding card wants the card's own 200x280. Five things follow:

- **A bleeding card carries no title**. `ShowName` is false on both, because the picture is the
  card, and a title bar across a full-bleed illustration covers the one thing worth looking at
  in order to repeat it. The full name still titles every tooltip, which is where a player who
  does not recognize a picture yet goes. `TestTheEnemyNamesItselfAboveItsPortrait` holds both
  halves — a naming card centers its name across the top, a bleeding card has none.
- **What survives on top of the art is one scrim, and only a card with something to say gets one.**
  A relic draws its counter disc in the bottom-right. `EssenceStyle` declares a text band from 140
  to 265, derived from the offsets the type is drawn at rather than authored twice — and **no card
  in the game fills it**: the essences joined the runes, the stones, the potions and the sealed
  goods in saying their rule in a tooltip, so every `EssenceStyle` face
  is the whole picture. A scrim is a ground for type and a ground under nothing is a stain.
- **Type on a scrim needs the other ink set.** Every ink in `internal/cards` is near-black,
  because it is written against the off-white `Surface`; `cards.onScrim` swaps the three named
  inks for light ones and lifts an authored element color toward white. **The one
  place that table is not a straight translation is `LabelInk`**, which is a stat row's quiet word
  everywhere else and is an essence's whole sentence here.
- **Art is committed at 200x280 and the generator's output stays in `.scratch`.** The batch came
  back at 1060x1484 — 1.1 MB a relic, about 155 MB across the catalog — and a 5.3x reduction
  redone at draw time is work repeated every frame for a picture that never changes. Reduced
  once, it is ~57 KB each and nothing resamples.
  `TestEveryBleedingCardArtIsTheCardsOwnSize` is the tripwire.
- **The generator's generic prompt lives in `docs/art/`; each record's own description lives on
  the record**. A prompt is about no record at all, which is why it
  is not in `data/`. What *is* about one record is the subject paragraph, and that is `Draw` —
  **ignored by the engine**, exactly as a status's `Badge` is, and pasted into the generator as
  the record's own JSON. A brief kept apart from the record is a brief that gets deleted when
  the picture it produced is filed. **There is no worklist file** — a
  record with an empty `Art` is still to draw and one with an empty `Draw` has no brief, and the
  catalog's own sheet counts both and marks both in pink.
- **One prompt per catalog**. `docs/art/relic_art_prompt.MD` and
  `docs/art/essence_art_prompt.MD` share a style block word for word and differ in the
  composition, in the ephemeral fading an essence has and a relic does not, and in the sentence an
  essence card prints across the lower half of its picture. `docs/art/rune_art_prompt.MD` is not a
  prompt but the closed list of rune body plans, pasted into the essence one.
  **`docs/art/stone_art_prompt.MD` and `docs/art/other_card_art_prompt.MD` cover the rest of
  `EssenceStyle`** — the stones, and then the potions, the three sealed goods and the placeholder
  brand. Both share the style and composition blocks verbatim, and both say the object is
  **whole** where the essence prompt says it is coming apart. **A stone's material is a rule
  rather than a record's own idea**, which is what earns it a file: the concept axis is silica,
  the form axis is plain rock, the element axis is gem, and each ladder ascends in finish. A new
  `EssenceStyle` good starts in the catch-all and earns a file the same way.
- **The relic catalog is pixel art and the other three are not**. The 137
  pictures in `assets/relic/` were generated from a prompt asking for chunky blocks and sixteen
  flat colors; the essences, runes and stones came from the smooth block every prompt carries
  today, so a relic card and a stone card do not look like one game.
  **`docs/art/relic_art_prompt_pixel_archived.MD` is the pixel prompt, kept live-shaped and
  clearly marked not-live**, so the direction can still be reversed by regenerating rather than by
  reconstructing a prompt from git. Closing this means regenerating one side or the other — 137
  relics, or 58 of everything else — and it is the owner's call which.
- **A prompt reserves nothing.** No title band, no counter disc, no corner kept clear. The card
  draws its counter over the bottom-right and its sentence on a scrim across the lower half, and
  both go on top of a picture that carries on underneath — so a prompt describing either produces
  art with a hole in it, or art with a *drawn* disc sitting under the real one, because a
  generator told a corner is special will decorate it. Both prompts say so in as many words.
- **A prompt describes the shipped art rather than an ideal, and `docs/art/README.md` carries the
  measurements.** The picture is one object at about 80% of the width and 70% of the height,
  sitting slightly high on a near-black hue-tinted ground with barely perceptible noise; smooth
  material shading, anti-aliased edges, a near-black contour, light from the upper left. It is not
  quantized pixel art and never was — the first prompt demanded 16 flat colors on a 40x56 block
  grid, the generator ignored all of it, and the catalog is what came back. **A clause nothing in
  `assets/` satisfies is a clause to delete.** The one thing an essence prompt adds is the fading,
  plus a note that the half carrying the recognition should be the upper one since the sentence
  lands on the lower: a placement hint, not a band to leave empty.

- **Six catalogs carry `Family` and `Draw`, and nothing that plays the game reads either** .
 `relics.json`, `essences.json`, `runes.json`, `potions.json` and `goods.json` carry `Art`
 beside them; every record under `data/motifs/` carries `Draw` and `Art`, with every `Draw`
 reading `TBD` — the roster's pictures are all still to be generated, so the field is
 a seat for briefs to be written into a motif at a time. **`Family` is the motif a record was
 authored beside** and is what its
 review sheet groups by; it is authored rather than derived for the relic catalog's reason, and
 it carries the same caveat — **it can go quietly out of date when a record is retuned and no
 test fails**, so re-read the block when you change what something does. **A creature has no `Family` field**: the file it
 is in *is* its motif, so a field repeating the name at the top of the file would say nothing.

**Relic, essence and rune art is a globbed family, keyed by filename stem** — `relic/fire.png`
is `fire`, which is what `data/relics.json` writes in
its `Art` field. **Each has its own default face** — `default-relic`, `default-essence`,
`default-rune`, reached through the record's `ArtKey()` rather than through a constant in a
screen: a fallback living in `internal/screens` is a fallback the review tool does not have, which
is how a sheet comes to disagree with the game. The runes wore the essence's placeholder until
they split, and they split because one shared picture is a page where a drawn essence and an undrawn
rune are the same face.
Same exception to the three-edit rule the enemy portraits take, and the same cost: a key is
tied to its filename, so renaming a file means editing the JSON. `assets.embedFamily` is the one
walk all four families go through. **Most relics still have no artwork and draw
`default-relic.png`** — `data.RelicData.ArtKey` is the fallback and `TestEveryRelicDrawsSomething`
fails on a key naming no file, so a blank face means art nobody has painted rather than a name
nobody spelled right.

**`go run ./tools/relicsheet` is how the catalog gets looked at.** A run wears five and the
shelf offers three, so seeing the catalog in a launched game means playing to a shop over and
over. The sheet draws each with its price, its authored `Text` and its rules side by side —
which is also the only place the sentence a player reads can be checked against the rules that
actually fire.

**`tools/essencesheet` and `tools/handsheet` are the same idea on the other two catalogs**
. An essence is offered two at a time after a won fight, so the whole catalog is five
fights away; the sheet draws them all grouped by what each one changes about a card, with the
authored `Text` against the rule that fires, exactly as the relic sheet does. The hand sheet draws
every rung of the ladder as an *actual hand of real cards* — the set the shipping deck can form
that best *illustrates* the rung — ordered by ascending multiplier across every axis at once,
which is the comparison `hands.json`'s axis-by-axis layout hides. **The example varies everything
the rung does not count**: a Pair is drawn as a 1 AP stab beside a 3 AP
one, because cheapest-set picked two identical cards and made every reading of a pair the same
picture. `decks.Example` is the one answer, shared with the hands panel;
cost is the tie-break among equally illustrative sets.

**It carries the reachability, and `tools/hands` is what makes that safe.** Two tools reporting
the same probability by different methods are two numbers that can disagree, so the deck, the
round's bounds, `MinCost` and the sample all live in `tools/hands` with the seed and the trial
count pinned — `handsheet` and `handodds` print the identical table to the last decimal. It is a
shared library under `tools/` and it earns the exception for `roster`'s reason: these are the
same question read two ways. **The cost is about thirty seconds on every
`tools/handsheet` run**, which a full `tools/sheets` pays too.

**Two figures, and the page says which is which.** The AP beside a rung is what that example costs
*once you hold the cards*; **reachable** is how often you hold them, and it is what the multipliers
are priced against. `handodds` stays the tuning view — the axes kept apart, and the `-ap` flag for
a turn holding cost discounts.

**`tools/stonesheet` and `tools/runesheet` do it for the two consumable catalogs**
. Both arrive four at a time inside a sealed good, so the whole of either is several
shop visits and a lot of luck away in a launched game. The stone sheet is **walked by rung rather
than by stone** — the catalog is one stone per rung, so walking the ladder orders the page for
free *and* makes a rung nobody authored a stone for show as a gap rather than as an absence nobody
notices. It is grouped by axis, with a merged rung under `any axis` and an axis with no rungs left
dropped rather than drawn empty, which is deliberately not the hand sheet's layout: that one
interleaves all three by multiplier because a player forming a hand chooses among all of them at
once, where a stone is bought against one rung. **It is also the only place the ladder and the +N
are visible together**, and the +N is computed from `hands.json` rather than authored, so a retuned
rung moves the card's face with nothing edited in `stones.json`. The rune sheet is relic-sheet
shaped — the authored line against the resolved rule — and earned a page before it had many
records, because a rune is the least readable record in `data/`: which of `Rider`, `Value` and
`Count` the rules read depends entirely on the target.

**`tools/motifsheet` does it for the roster**. A creature is met one at a time, three rooms to a
floor, and its whole personality is a deck the player only ever sees the played half of — so "do
this motif's three rooms read as a climb" was a question answered by reading JSON. The page groups
by motif, prints the motif's HP, DMG and AP spread and its tier mix in the heading, and draws every
record as **one composite strip**: the opponent's own card as the combat screen draws it, then its
deck, one card per concept with the copy count in the table under it.

- **A strip rather than a file per card**, because a file per card is several hundred binaries
  rewritten on every full run for the same pixels. This sheet is most of the committed weight; see
  the note above about regenerating only what changed.
- **It carries the coverage grid**, which is the one thing about a motif that cannot be seen by
  looking at its records one at a time: a floor picks a motif and an element, so what has to hold
  is that every element can field all three rooms at least twice. It is drawn from
  `data.CoverageOf` — the same function the loader refuses a file with — so the page and the
  launch cannot disagree about it.
- **The deck size is read off `internal/decks`**, not added up from `Copies` in a template, and
  importing that package registers every concept at init — so a card naming a verb the rules do
  not have fails the sheet exactly as it fails a launch.

**It groups by family, in the file's own order.** `data/relics.json` is authored in motif
order — the flips together, the ring families walking their ladders, the weapons along the
concept ladder — and sorting the page by key is the one ordering that throws all of that away.
`Family` is the field and `data.RelicFileOrder` is the walk.

**`Family` restates the rules in words, and it is authored for the owner's own reading**
. Nearly every value is implied by the record's `(When, Do, predicate)`
— the flips are all `card-drawn`/`set-element`, the weapons all `card-damage`/`scale-damage` on a
Concept — so a derived grouping would reproduce it almost exactly. It is authored anyway because a
signature is something to decode and "Jade rings" is something to read, and the three ring families
differ by the gem in the picture as much as by the axis in the rule.

**This is not the `CostTier` mistake, and the difference is who consults it.** `CostTier` was a
figure the *rules* also knew, so a file could contradict the game; nothing resolves a round
differently because of this string, and the `relic-balance` skill's derived taxonomy is untouched
and still the authority on what a relic *is*. What the field can still do is go quietly out of
date — a relic retuned into a different family keeps its old label and no test fails — so
**re-read the block when you change a relic's rules**, and treat a disagreement as a label to fix
rather than as a second opinion.

**The pricing review survives the move because nearly every family is one tier throughout.** A
heading reading "15 relics, all common" asks "does one of these belong a tier up" of the whole
block at once, and a family with a mix prints the mix rather than hiding it — which is the question
the rarity grouping existed to answer. The three tier shares moved to the page header, where they
are about the shelf rather than about a motif: the share is the tier's tickets over the
catalog's, to a tenth of a percent, because a scarce tier rounds to `0%` and would read as
unreachable.

**Every word naming an element is written in that element's color**.
`cards.ElementSpans` is the one vocabulary — the five element names plus each status's `Name` and
`Verb`, read off `statuses.json`, longest first — and `cards.SplitSpans` is the one cut, matching
whole words only and ignoring case so a relic writing `Fire` and an essence writing `FIRE` share an
entry. Four things to know before touching it:

- **The vocabulary lives in `internal/cards` because that is the only windowless package all three
  readers share.** The card faces are set there, the screen text is set by Ebitengine's `text/v2` in
  `internal/screens`, and the nine review sheets build their own `cards.Spec` on purpose — a table
  anywhere higher would leave every sheet uncolored, since a sheet cannot import `screens` without
  linking a window. It costs a `cards` → `data` arrow, which is downward like every other; the
  precedent is `Badge`, which `statuses.json` already carried for a reader the engine ignores.
- **`Spec.Highlights` is a fixed array**, because `cards.Spec` is the render cache's key in
  `internal/screens` and must stay comparable — the same constraint `Stats` is under and
  `combat.Card.Riders` is under one package over. `TestEveryTextFitsItsHighlights` holds the whole
  authored catalog against `MaxTextHighlights`, so an author who runs out of room fails a test
  rather than shipping a half-lit sentence.
- **`models.Tooltip.Title` and `.Lines` are both runs rather than strings**, and
  `screens.tipLine`/`tipLines` are the one door every `Point` call
  goes through — which is what stops a new tooltip shipping as the only panel in
  the game whose relic text is gray. `internal/systems` draws the runs and never learns why one is
  colored, because it cannot see `internal/cards` at all.
- **The fight log colors through the ledger's *named* inks**, not through a stored color. A line is
  written once and read back three fights later, so a color baked into it would be the color the
  build that wrote it happened to use — see `session.LedgerSpan.Ink` and `screens.elementInkNames`.

**Hue belongs to the elements, and the wheel is full**. Fire, ice, lightning, earth and arcane
take five hues; pink is a relic and a pane's chrome; red and blue are the attack and defend
verbs; green and gray are the two duelists. **There is no unclaimed hue left**, so a new thing
wanting to stand out is marked by *weight, case, a swatch or an underline* rather than by a
color. **The ground itself takes blue**, which is a real collision with the defend verb and is
accepted rather than solved: the table is a surface and a verb is a mark on it, so the two are
never being compared, but it is why the AP bar's empty cells had to stop traveling 80% of the
way to the ground and settle at 50 — see `combat_actionbox.go`. A *new* thing wanting blue has
nowhere left to stand. **The hand's own name is the case that proves it**: the relic pink would
put the two things that multiply a blow in one color in the same sum, and deep purple collides
with arcane — so it takes the ground's own ink and is *marked* instead. See
`screens.handNameInk` and `session.InkHand`, and note the second argument: three of the four
axes a hand counts on are not elemental at all.

### The palette: eight ramps, and only the middle one is drawn

**`docs/art/palette.json` is where the colors are written down** — five elements and three form
materials, each as `dark`, `core` and `light`, with the plain-English `hue` beside them, the
`axis` they belong to, and the `form` a material stands for. It is reference rather than a
catalog: nothing loads it, and it is not in `data/` for that reason. What it is *for* is the
generator, which is handed one ramp when a picture has to sit in one element's or one form's
range.

**`core` is the only column the game draws in**, and it is a Go value in two places —
`cards.borderColors` for the five elements and `cards.InkIvory` / `InkSteel` / `InkGranite` for the
three materials. So `core` is written twice, here and in the file, and nothing checks that the two
agree: **re-read `palette.json` against `internal/cards` before trusting either**, and treat a
disagreement as a value to fix rather than as a second opinion.

**The three materials are the form axis** — **ivory is stab, steel is
slash, granite is crush** — and `cards.FormWords` is the one table that says so. **Defend has no
material and is not in it**: blue already belongs to the verb and there is no fourth ramp, so a word
there would be a color with nothing behind it.

**A form word on a dark panel is set in its material rather than in its color.** `assets/texture/`
holds one seamless tile per material and `internal/systems/tooltip_texture.go` keeps it inside the
glyph shapes, so SLASH is a piece of brushed steel in the shape of the word. **A card face cannot
do this** — `internal/cards` draws into a plain Go image with no graphics context and masking is a
GPU blend — so a span carries the *name* of a material and the flat `core` beside it, and a drawing
that cannot composite one falls back to the ink. Same split `systems.ArtMark` and `ArtMarkImage`
already make.

**The tiles are committed at 114x114, reduced from the source by exactly eleven.** An integer factor
is what keeps a seamless tile seamless, and the reduction is what puts the grain at word scale: at
native resolution one crystal is most of a capital and the word reads as a blotch.

**A change here moves the art out of step with the type, and nothing fails.** The form marks and
cost ticks in `assets/form/` are authored in their element, and **every prompt under `docs/art/`
writes the ramps out in full** — the closed-set prompts (`card_art_prompt.MD`,
`damage_art_prompt.MD`, `glyph_art_prompt.MD`) as their element table, and the subject prompts as a
shared **The color words** block, which is what lets a brief say "a purple orb" and land in arcane.
`gear_art_prompt.MD` is the one without a ramp, since the cog is neutral gray by rule. **Move a
`core` and all of them have to follow**, or a card's corner mark and the word naming its element
are two different colors.

### Color: name one color and scale it — and the light comes off that color too

**The rule governs widget *state*; the bevel is the surface's own light**. Those are
different questions and separating them is what let bevelling land without every widget in the game
being handed a palette: a button naming crimson and brightening toward it on press is state, and
the lit top edge it has whatever state it is in is surface.

**`systems.BevelEdges` derives both edges from the fill itself** — `ColorToward` toward white for
the light, because a saturated color has nowhere to climb by scaling, and `ColorAtStrength` for
the shade. So a widget still names one color, everywhere, with no palette anywhere: the one thing
that ever needed a six-value one was a generated silhouette, whose light had to be drawn because
it had no fill to compute light from, and there are no generated silhouettes left.

- **`BevelFace` for a control, `BevelRect` for anything else**, and the depth differs on purpose:
  `BevelWidth` is 3 for a button, `PaneBevelWidth` is 2 for a panel, which is the largest surface
  on screen and the one where a heavy bevel reads as chrome rather than as a surface.
- **Sunken is a meaning, not a variant.** A pressed or latched button swaps its two edges, which
  is how a face says "in" — brightness could not, since hover already owns the bright end of the
  ramp. The deck panel and the fight log are raised because they cover the game; **the relic
  pane is flat** , because it covers nothing and the bevelled cards standing on it are what
  should be read.
- **Disabled has no bevel at all.** Unavailable first, itself second — the same argument that makes
  it ignore `BaseColor`.
- **`internal/cards` bevels the outer 2px of a card's 3px border**, rasterized in plain Go since
  that package has no graphics context, and splits light from shade on the *anti-diagonal* rather
  than by edge — a per-edge rule has to answer for the corners and every answer leaves a seam on
  the curve. It shares `BevelEdges` with the widgets so both are lit from the same corner.

**And it assumes the thing being dimmed sits on a dark ground.** `ColorAtStrength` scales
toward black, so on a light surface it makes things louder rather than quieter — see the card
section above. `systems.ColorToward` is the light-ground counterpart and the two are not
interchangeable.

**Every screen's ground is a light slate blue** (`screens.screenGround`), so `ColorAtStrength`
is the exception rather than the default, and reaching for it to dim something drawn straight
onto the table is a bug waiting to be seen.

**Its lightness is what is load-bearing, not its hue**, and that is the sentence to read before
changing it. `groundInk` is near-black and every dim on the table is
`ColorToward(x, screenGround, pct)`; both are only correct on a light ground. **A darker ground
is not a color change, it is a re-tune of every figure on the table.**

**The screen is painted by `screens.fillGround`, not by `screen.Fill`** — a subtle vertical
gradient, lighter at the top, lit from the same corner `systems.BevelEdges` lights every card and
button from. **`screenGround` stays a single color anyway**: everything that dims toward the
ground needs one answer to "what color is the table", and a per-pixel one would make a figure's
dimming depend on where it happened to be drawn. The gradient's two ends are derived from it.

**A color that is "one step off the ground" must be derived, never written down.**
`relicPaneBackColor` was a hand-picked tan and would have silently stopped being one step off
anything the moment the ground moved; it is `ColorAtStrength(screenGround, 91)` now. It still
governs buttons, because a button paints its own dark face and its label is white — that face is
the ground its states are scaled against, not the screen. Text written directly on the table
takes `screens.groundInk`.

A widget names the color it wants at **full strength**, and its other states are
scaled down from that with `systems.ColorAtStrength`. `models.Button.BaseColor` is the
reference case: the button rests at 65%, hovers at 82% and reaches the named color at
100%, so pressing it lights it up to exactly the color in the source.

- **Scale a color, never add to it.** Adding a fixed step to every channel walks a
  saturated color toward white — crimson hovering to a washed-out pink — and a channel
  already near 255 has nowhere to go. Scaling holds the hue.
- A zero-alpha color means "use the default", so widgets that never pick one are
  unaffected.
- Disabled deliberately ignores the widget's color. A disabled control should read as
  unavailable first and as itself second.

### Three debug flags, and they are not interchangeable

`DebugPlacement`, `DebugGameplay` and `DebugAnimations` answer different questions and are wanted
at different times. Keep them separate.

- **`DebugPlacement`** — the grid, the rulers, the `Debug1`/`Debug2` scratch strings. About
  *where things are drawn*. Safe to leave on while playing, but off by default, so a change
  that needs the guides has to turn it on deliberately.
- **`DebugGameplay`** — perfect information, starting with the opponent's queued actions.
  About *what the player is allowed to know*. **Off by default**: with it on you are not
  playing the game, you are inspecting it, and it is easy to tune balance against a view no
  player will ever have. What it currently reveals is the combat screen's, and lives in the
  `combat-screen` skill.
- **`DebugAnimations`** — the door to the **animation gallery**, a square marked
  `A` off the end of the frame's bottom strip that opens `screens.AnimationsScene`. About *what
  movements the game has and what each one is called*. **A third flag rather than a lodger on
  `DebugPlacement`**, which is the rule those two are already under: "where is this drawn" is not
  "what gestures exist".

**The gallery exists to give the gestures names.** There are a dozen distinct movements on the
combat screen and each was reachable only by producing the situation it belongs to — a break needs a
shield eating an attack, a toast needs a relic firing into a sum, the cascade needs two flip rings
and a hand with the right colors in it. So "make the toast louder" was a sentence with no shared
referent. The page lists every gesture by name, plays it on a loop, and prints the **symbol that
implements it** beside it.

- **Every entry calls the game's own drawing.** An entry that reproduced a gesture is the
  stale-sheet failure — a picture of something the game does not do — so a shared one is split out
  of its caller instead. `outboundGeoM` and `drawDealtCard` were split for exactly this.
- **It is a screen rather than a tool**, unlike everything under `docs/sheets/`: those work because
  `internal/cards` renders without a graphics context, and **motion needs a window and a clock**. A
  still of a dissolve is a picture of a card with holes in it.
- **It is not a station of a run** and has no phase — the shape Settings, Achievements and Credits
  share. **Adding a gesture is one entry in `animGestures`.**
- **The button is the one thing in the frame that is not chrome by the frame's own test.** It is
  instrumentation rather than something true of the whole session, and it is there because the frame
  is the only place a debug page is reachable from every screen. With the flag off the strip is
  exactly the two controls it was.

None of the three may ever change an outcome. All are views, the same constraint that applies to
playback speed — `ResolveRound` never sees any of them.

All three are set once in `main.go`; there is no runtime toggle, because a hotkey would need the
keyboard and the input vocabulary does not have one. **`DebugPlacement` and `DebugGameplay` default
to off, and `DebugAnimations` is on while the gallery is being built** — it belongs off before this
ships, on the same argument the other two are under.

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

[internal/scenario](internal/scenario) plugs **a chosen set of relics, a chosen opening hand, a
chosen enemy — and a chosen screen** into a launched game.

```powershell
go run -tags scenario .                                        # the first entry in the file
ASCEND_DUEL_SCENARIO=seven-term-sum go run -tags scenario .    # a named one
go run .                                                       # nothing: every function is a zero value
```

It exists because an interaction between relics is currently a twenty-minute question. A relic is
bought from a shelf of three, a hand is dealt from a shuffled deck, and an enemy is whoever the
climb put in the room — so "does Echo actually multiply Enflamed's growth" cannot be *looked at*
without playing toward it. The rules are unit-tested; what no test can answer is what the
combination looks like on screen. It is the relic-and-hand counterpart of `deckSeedName` and
`session.StartingRelics`, which each do one axis of the same job.

- **`scenarios.json` lives beside the package, not in `data/`.** Everything in `data/` is the
  game's own catalog, loaded by every build. A scenario describes a thing being *tested*, and
  filing it with the cards would embed a debug fixture in a release binary.
- **A build tag for the reason trace and idle have one**, and the same two-file `_on`/`_off`
  shape. This hands the player a chosen hand and a chosen row of relics; it must not ship, and it
  has to stay deletable in one commit. The `//go:embed` is in the `_on` file, so an untagged build
  carries neither the fixture nor the reader.
- **It deliberately changes outcomes, unlike everything else that is compiled out.** `trace`,
  `idle`, the demo and both debug flags are views and may never alter a result. This is a
  *fixture* — which is exactly the argument for the build tag rather than a runtime flag.
- **Three call sites, each one guarded line**: `main` sets `session.StartingRelics`, `Init` picks
  the enemy, `resetDeck` plugs the hand. Nothing else in the game knows the package exists.
- **The hand is dealt over the shuffle rather than through it.** The draw pile is untouched, so
  the second hand of the fight is a normal one and the fixture is only the opening.
- **A misspelled relic, card or enemy fails the launch**, at package init, before a window opens.
  A fixture that quietly tests something else is worse than a game that will not start.
- **It also opens the game on a named screen**: `"Screen": "reward"`
  or `"shop"`, with `Fight`, `Vitae` and `Life` saying what state to arrive in. A between-fights
  screen was otherwise a twenty-minute question — the reward screen's narration and the shop's
  shelf both needed a duel played to reach them, every time. It sets the run's *phase* and lets
  `screens/flow.go` decide the scene, so the run never disagrees with what is on screen.
  `reward-payout` and `shop-shelf` are the two entries.
- **`"Essences"` plants the satchel**, beside `Runes` and `Stones`. An essence is
  normally spent the instant it is taken, so a run *carrying* one into a duel is the one state no
  amount of playing reaches — see MECHANICS.md §An essence can be carried into a fight, and the
  `ladder-wrap` fixture.
- **It can also pin the seed and replace the whole deck**. `"Seed"` is a six-character Crockford
  base32 run code and outranks `fixedRunSeed`, and `"Deck"` sets the run's deck outright rather
  than dealing over the shuffle the way `"Hand"` does — through `session.StartingDeckList`,
  which is the deck counterpart of `StartingRelics`. The tutorial is what wanted both: a first
  lesson has to be able to promise what the player is holding, and "these five all match, play
  them all" stops being true the moment a refill deals a sixth card nobody mentioned.
- **A deck line and a hand card may carry `"Riders"`**, by the names
  `combat.RiderKind` writes, with a figure after a colon where the kind takes one —
  `"damage-on-play:10"`, or the bare `"wild-element"` for the one that does not. It exists because
  `Runes` is the right fixture for looking at the *dialog* and the wrong one for looking at what
  an altered card does to a hand: getting there means playing a turn to spend the consumable and
  then reading a hand that is already half spent. The `wildcards` entry is what wanted it.
- **`"Teach": true` starts the tutorial on the run**, and is the only way to start it today — see
  the tutorial section below.
- **Every entry carries a `Note` saying what question it answers**, printed at startup. A fixture
  whose purpose nobody remembers is a fixture that gets deleted.
- **`"Dummy": true` is a fight that cannot end**. Both duelists get
  `scenario.DummyLife` and the clock goes to `scenario.DummyRounds`, so a scenario can be *played
  with* rather than survived — every blow, every shield break, every status and every signal, for
  as long as it is interesting. **It is not a record under `data/motifs/`, deliberately**: a
  training dummy is a fixture and `data/` is the game's own catalog, loaded by every build, drawn
  on the roster sheet and reachable by the climb's own roll. So it changes the *stats* of whichever
  opponent was already there, which means the fight keeps a real portrait, a real deck and a real
  set of blows. **The clock is 999 rather than off**, because `session.SetRoundLimit` refuses to
  stop the clock and the fixture goes the long way round rather than being given a back door into
  the rules. `"RoundLimit": N` is the same dial on its own, for looking at the clock itself, and
  `"Actions": N` widens the turn's budget — which is what makes a bench a place to pick the cards
  you want rather than the cards six points can pay for. **It does not lift `combat.MaxActions`**,
  the count bound: a turn is still five cards however cheap they are.
- **`tools/scenariodeck` writes the `Deck` block, and that is deliberately a generator rather than
  a filter vocabulary**. `-form slash -size 40`, `-elements fire,ice`,
  `-cost 1-2`, `-riders golden:5`; it prints JSON to stdout and **never touches a file**. The
  obvious alternative was `"DeckOf": {"Form": "slash", "Share": 50}` read at launch, and that is a
  *second card-selection language* living in a debug fixture, which has to stay in step with
  `data/duelist_cards.json` and with `internal/decks` — and being a debug fixture is exactly why
  nobody would notice when it drifted. What lands in the file is the literal list the fixture
  already supports, so `scenarios.json` stays a thing that can be read and checked. **The filters
  are meant to be extended**: the next axis is one `flag.String` and one clause in `pick`.
- **`tools/scenariosheet` is how the fixtures get found.** They are the fastest way to look at
  anything in this game and were the least discoverable thing in the repo — the only ways to find
  one were to read the JSON or to misspell a key. The page carries each fixture's Note against what
  it actually plugs in, with the launch command ready to copy. **It reads the JSON off disk rather
  than importing the package**, because importing it would mean building `tools/sheets` under
  `-tags scenario`; the cost is a second view of the record struct and the tripwire is
  `DisallowUnknownFields`, which fails the sheet loudly when a field is added to one and not the
  other.

### `internal/profile` is what survives a run, and it is the only thing that touches the disk

[internal/profile](internal/profile) owns the two files the game writes: `profile.json` (the
player — the tutorial watched, achievements, unlocks, and the settings) and `run.json` (the run
in progress). See MECHANICS.md §The profile for what they mean; what matters here is where they
go and what may never happen to them.

- **They live under `os.UserConfigDir()`, never beside the executable** — `%APPDATA%scend-duel` on
  Windows, `~/.config/ascend-duel` on Linux. Steam installs into a tree a normal process cannot
  write to, where a write either fails or is silently redirected into `%LOCALAPPDATA%\VirtualStore`,
  which is worse because it works in testing. A per-executable directory is also per-install rather
  than per-user. `ASCEND_DUEL_PROFILE` overrides the **directory**, moving both files together.
- **Nothing here may ever be fatal.** Missing, corrupt or unwritable are all "a new player, and this
  session is not recorded" — the same rule the audio device is under. A launch refused over a save
  file would be a worse bug than any it prevents.
- **A file from a newer build is read and never written over.** It is the one mistake that cannot
  be repaired afterwards, so `LoadProfile` reports writability separately and the game respects it.
  Unrecognized fields are carried through a save verbatim for the same reason.
- **What is written down is a name, never a number.** `ConceptID`, `Element`, `StatusID`,
  and `session.Phase` are all append-only ordinals indexing arrays and caches. A file
  outlives the build that wrote it, so an ordinal in one will eventually mean something else. This
  is where that rule stops being theoretical.
- **A setting's zero value is not its default, and that is the one trap in the file.** An older
  profile has no settings block at all, so both fields read as zero — and a speed of zero would
  stop every clock in the game. `LoadProfile` normalizes and clamps; nothing about a save file may
  ever fail a launch, so an out-of-range number is brought into range rather than rejected.
- **The call sites are `internal/screens/save.go` and nothing else.** A run is saved by
  `advanceRun` at each phase transition, the achievement is awarded where a fight is won, and the
  tutorial is marked seen where the overlay ends. Persistence is deliberately not something a scene
  does.
- **The climb is not saved — it is rebuilt from the run code.** True only while the fight order is a
  function of the seed; `TestTheClimbIsRebuiltFromTheSeed` fails the day the room choice makes it a
  decision, which is when it has to go into the snapshot.
- **A run is snapshotted between phases, never inside a duel.** `session.Session` is snapshotted and
  is still not *replayable* — the replay story is a seed plus a choice log, because a deck edit is a
  choice. Resume wants state, replay wants a path; do not let a snapshot be used as a replay.

### The tutorial is a seventh thing, and it is *not* compiled out

[internal/tutorial](internal/tutorial) is the teaching run: which step is up, what it points at,
and what has to happen before it moves on. `data/tutorial.json` is the script and
`internal/screens/tutorial.go` is Bob's bubble, the red square and the leader line to it.

**It ships.** Unlike trace, idle, the demo and the scenario fixture, a tutorial is a feature the
player is meant to meet — so there is no build tag and it is in every binary.

**It fires on its own**, off the profile: a player `profile.json` has not recorded as taught
gets taught, on the first fight of a fresh run. `main.teachThisRun` is the whole trigger, and it
declines for a resumed run and for a scenario — a lesson that opens by describing the hand you
are holding cannot begin halfway up a tower. **A launch on a clean machine therefore opens into
the tutorial**, which is a thing to know before wondering why Bob turned up. `"Teach": true` in
a scenario still forces it whatever the profile says, and is the only way to see it a second
time; the counterpart is `ASCEND_DUEL_PROFILE` pointed at an empty directory, which makes any
launch a new player's.

- **The state machine is free of Ebitengine**, like `internal/combat` and for the same payoff: the
  whole script is walked end to end in a test rather than by playing to the end of it.
- **A scene publishes `tutorial.Facts` once a frame; it does not fire events.** A condition is a
  predicate over what is true now. The alternative — a `Did("duel-pressed")` call at every site
  where something can happen — fails silently when one is forgotten, where a scene that forgets to
  publish reports the zero value and stalls immediately.
- **Three vocabularies, all closed and none defaulted**: anchors, conditions, and the lock derived
  from the condition. See the `data` skill.
- **A step waiting for NEXT holds the round where it is**.
  `tutorial.Run.HoldsRound` is the predicate and `advancePlayback` is the one reader. It exists for
  the shield step, which is the first in the lesson to land *inside* a playing round — a round has
  three acts (the duelist swings, the shields break what they can reach, the creature swings with
  what is left) and the middle one had no beat of its own, so the break appeared and the creature
  was already answering. **Only a NEXT step holds**, which is what stops a step waiting on an
  outcome from stopping the thing it is waiting for; `TestOnlyANextStepHoldsTheRound` is the
  tripwire. It changes pacing and cannot change an outcome.
- **A break lives inside its own round.** `seatEnemyCards` drops the marks, because that is the line
  where a seat number stops meaning what it meant — the opponent's row is re-planned the moment a
  round ends, so a mark left standing cracks whichever card the planner has just put in that seat.
  Anything wanting to point at a break has to do it during the round, which is why the step above
  holds one.
- **An anchor names what the step is *asking for*, not what it is about**. `matching-cards` and
  `matching-cards-left` are the same set minus what is already queued, and they exist as two
  because the two steps using them say different things: "take the other three" asks, and "one
  of those four is a Brace" describes. Sharing one anchor lit four cards under a sentence about
  three — and since the anchor is the click gate, the card already taken was the one thing the
  step invited you to click, which undoes the step before it.
  `TestTheStepAsksOnlyForTheCardsStillToTake` is the tripwire. **The red comes off each card as
  it is taken**, so the row says how much is left without a counter.
- **A card is tinted, a control is framed**. An anchor naming cards gets
  the scrim and no rectangle: the cards wear `cards.MarkHighlit`, which is the same red the frame
  was. A frame outside a card is a thing on the screen *near* the card where a tinted card is the
  card answering, and round a set of cards a frame is a lot of loose rectangles. `Anchor.NamesCards`
  is the closed table saying which; `screens.marksFor` reads the same `gs.InputFocus` list the
  spotlight is handed, so lit and clickable stay one set by construction rather than by agreement.
- **The lit square and the one legal click are the same rectangle**, computed once. A lit hole the
  player cannot click, or a clickable region that is not lit, would each be worse than no tutorial.
- **An anchor may name several rectangles, and for a *set* of cards it must**.
  `tutorialRects` and `state.InputFocus` are both lists because the two anchors naming a set —
  `matching-cards` and `shattered-cards` — point at cards that need not be adjacent, and the
  bounding box round them is the set *plus whatever is between two of them*. That was a live bug:
  the tutorial matches on **element**, so the taught four are four different concepts, and under the
  default cost-led sort they land at seats 0, 1, 2 and 4 with an arcane card at seat 3 — lit,
  clickable, and worth 2 AP out of a 6 AP budget the taught set needs all of. Queue it and the
  fourth taught card can never be paid for, so the lesson commits a Three of a Kind having just
  promised a Four. **The old note claimed they were contiguous and it reasoned about the wrong
  axis** — cards sharing a *concept* land together whichever key leads; cards sharing an *element*
  do not. `TestTheMatchingCardsGateLightsOnlyTheTaughtCards` walks every seat of the real dealt hand
  through `InputAllowed` and is the tripwire. The spotlight scrims the gaps between holes, so lit
  and clickable stay the same area.
- **The tutorial runs on the real deck, and `matching-cards` is what pays for that** . It was a
fixture deck of exactly five Jabs, so the lesson's "take them all" step could wait on
`hand-emptied` — a condition only a hand with nothing else in it can ever reach, since a real
hand of eight against a five-card cap and a six-point budget leaves cards behind by the rules of
the game. The anchor is the largest matching set in the hand and `matching-queued` is its
condition; because the lock leaves only those cards clickable, the hand the player builds is the
hand Bob just described. **It is the one anchor computed from the cards rather than from a
layout** — `CombatScene.matchingCards` is the single answer both the square and the condition
read.
- **Which axis a set is counted on is authored, not assumed**.
  `data/tutorial.json`'s `Match` is `concept`, `form` or `element`, and a script that points at a
  matching set without naming one is **refused at load** — an axis that defaulted would be a lesson
  pointing confidently at the wrong cards. The lesson matches on `element`.
- **The script carries the run it needs: `Seed`, `Enemy` and `Match`.** **A promise and the
  thing that makes it true belong in one file** — pinned from a scenario instead, the lesson
  runs on whatever the clock rolled and describes a hand it has not dealt, because the profile
  can start it with no fixture in sight. The scenario entry keeps only `"Teach": true`.
- **The taught fight is two rounds, and the shield is why.** Run code `0009D4` deals `Jab Brace
  Thrust Bash`, all arcane, for exactly 6 AP — an Elemental Four of a Kind dealing 69 into a
  GiantBat's 80. **One of the four is a Brace**, which teaches the thing a
  hand of pure attacks cannot: a defense carries an element and joins a hand like anything else,
  bringing no damage with it. Because it brings none, the creature lives on 11, takes its turn —
  Swoop, Drain, Nip — and **the Brace's one shield eats the Drain whole while the other two land**,
  60 life down to 53. A creature that dies in one blow never swings, so a lesson about shields
  cannot be taught in a round that kills. The player then reads the ledger and finishes it.
  **The Drain is the bat's one big card**, which is what the heaviest-blow rule makes visible:
  the shield saves ten rather than five, and the step that explains it has a broken card on the
  table to point at.
- **The other four cards are an arcane, an earth, a fire and an ice**, so there is no competing set,
  and the first card dealt is one of the four — which the opening step needs, since it queues
  `first-card` and a stray would break both the budget and the hand.
- **Finding a replacement seed is `TestFindATutorialSeed`**, skipped unless `SEEDSEARCH=1` is
set. Every test below ends "the fix is a new seed, not a weaker check", and the constraints live
in four files that a candidate has to satisfy all at once. It is a test rather than a tool
because the shop internals it deals from are unexported, and `tools/seeds` cannot answer this
one — that tallies concepts and the tutorial matches on element. **It proposes and asserts
nothing**: take a candidate, pin it, and let the four tests below confirm it. **Prefer a marked
candidate**, which keeps the cards the steps name — a seed dealing a different four means
re-authoring the lesson rather than changing one string. **Expect to re-run it whenever
`relics.json` gains, loses or renames a record.** The shelf is a weighted draw over the
catalog's sorted keys, so any of those three reshuffles what the taught seed lands on, and the
shop step is the only part of the lesson a catalog edit can break silently. **That is the cost
of drawing the taught shop rather than pinning it**, and pinning it is the fix to argue for if
it keeps happening. A replacement seed is chosen to deal the identical four cards against the
identical creature, so no step text has to change; the taught color does, and the lesson never
names it.

**Both halves of the promise are tested, and they check each other.**
  `TestTheTutorialsBlowWoundsTheTutorialsEnemyWithoutKillingIt` in `internal/combat` proves the
  rules resolve that turn to a wound — **it is two-sided**, failing if the blow starts killing, if
  it leaves more than half the creature standing, or if the taught set stops holding exactly one
  shield. `TestTheTutorialsSeedDealsTheHandTheLessonDescribes` in `internal/screens` proves the seed
  actually deals it — the set's size, that it is the only one that size, that the first card belongs
  to it, that it is affordable, that it does *not* kill, and that its four cards are the four the
  combat test writes out by hand. **If either goes red the answer is a new seed, not a weaker
  check**; `go run ./tools/seeds` is the search.
- **A third test holds the creature's half of it**.
  `TestTheTutorialsShieldEatsTheCreaturesHeaviestBlow` in `internal/screens` plans the bat's turn
  exactly as the screen does, resolves the whole round, and checks that one blow is blocked, that it
  is the heaviest, that the heaviest is the *only* card that size — a creature whose deck flattened
  out would make the lesson true and pointless — and that the step naming the card names the right
  one. The two above are about the player's blow; this is about what comes back at them, which is
  the half the shield steps describe.
- **The ledger step is the one anchor naming a control the frame owns**. `state.LedgerOpens`
  is a tally bumped by `internal/game` when the panel opens, published as a fact and read by
  `ledger-opened` against a baseline — the same trick `round-done` uses, because the account is
  reachable from every screen and an opening from three steps ago is not this step's. **It advances
  one frame late on purpose**: the panel takes the whole frame and the scene beneath it is not
  updated at all, so the step gives way when the player closes the account rather than while it is
  covering Bob.
- **`gs.InputGated` / `gs.InputFocus` is the shield**, and it gates on the *cursor* rather than per
  widget — one predicate in `systems.UpdateButton` plus the handful of places in `internal/screens`
  that read the mouse directly. A per-widget rule is a list a new widget is missing from.
- **`internal/game` clears the gate every tick and the tutorial re-asserts it**, exactly as
  `state.ModalOpen` works, so a screen left mid-step cannot leave the session unclickable.
- **It is deliberately not a `modalToggle`.** Every other dialog takes one footprint and scrims the
  whole screen; a thing whose job is to point at what is underneath cannot be the thing covering it.
  That is a second dialog shape, decided on purpose.
- **`TestTheTutorialsBlowWoundsTheTutorialsEnemyWithoutKillingIt` in `internal/combat` is the one
  to keep.** The lesson promises a blow that wounds and does *not* kill, and four files tuned for
  their own reasons can break that promise silently in either direction — the taught cards'
  `Amount`, the ladder's multiplier, the duelist's `DMG`, the bat's `HP`.

**The machinery refuses the mistakes it can detect** — an ungated action step, a click with nothing
named to click, a lock disagreeing with its condition. **What it cannot check is whether an anchor
shows the player how to satisfy the step's condition**, and that is where every bug in this feature
so far has been: a step pointing at the shop shelf while waiting for the player to press *Leave*
reads as a lock-up. Read each new step against its own condition.

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
| how do I do X safely (git, data, relics, randomness, the combat screen) | the skill — see the index above |

### The dependency graph

**Generated from the real imports, not drawn from memory**. The picture that used
to be here had two arrows the code contradicted. Regenerate it rather than patch it:

```powershell
go list -f '{{.Name}}: {{join .Imports " "}}' ./... | grep curiousjc
```

| Package | imports, of ours |
|---|---|
| `seeds` `models` `assets` `idle` `trace` `music` | *nothing* |
| `data` `profile` | *nothing* |
| `scenario` | data, combat *(compiled out unless `-tags scenario`)* |
| `pyramid` | data |
| `combat` | data |
| `tutorial` | data |
| `achieve` | data, combat |
| `carddesc` | combat |
| `decks` | data, combat |
| `entities` | data, combat, pyramid |
| `session` | data, combat, pyramid, profile, seeds, tutorial |
| `state` | data, session |
| `systems` | assets, models, state |
| `cards` | systems |
| `actions` | state |
| `ui` | data, achieve, carddesc, cards, combat, decks, entities, models, pyramid, session, state, systems |
| `screens` | all of the above, plus `ui` and `scenario` |
| `game` | screens, ui, state, systems, models, music, idle, trace |
| `main` | game, session, assets, data, music, scenario |

Six facts about it that are load-bearing:

- **`data` is the bottom and must never import upward.** That is why creature concepts are
  registered by `internal/decks` rather than handed over by `data`: a creature's cards live in its
  motif file beside art keys and floor bands, so the rules reading that file directly would cross
  the who-consumes-it line.
- **`profile` imports nothing of ours, like `seeds`, and that is what makes saving possible at
  all.** It owns the two files on disk and knows nothing about a run: `session` converts itself to
  and from a plain snapshot struct, so the persistence layer never learns what a card is and the
  arrow points down like every other. `state` carries the loaded profile beside the run.
- **`tutorial` sits beside `combat` at the bottom and imports only `data`.** It is a state machine
  over a script — which step is up, what it points at, what advances it — and it is free of
  Ebitengine for the reason `combat` is: the whole script can be walked in a test rather than by
  playing to the end of it. `session` holds the cursor, because a lesson outlives a fight; the
  rectangle behind an anchor is `screens`, because a rectangle is a fact about a layout.
- **`seeds` imports nothing and `combat` deliberately does not import it.** The rules take an
  injected `*rand.Rand` and stay ignorant of where it came from.
- **`carddesc` is the words a card says about itself**, and it is here rather than in
  `internal/screens` for `decks`' reason: the review sheets have to print the *same* strings the
  game shows, and a tool cannot import a package that links Ebitengine. It holds the tooltip's
  stat block — the title, the AP, the effect figure, the upgrade's lines — and no color, no
  widths and no arithmetic that needs a worn relic. `screens.cardTip` calls it and appends its
  own damage chain.
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
- **`internal/ui` is the drawing layer and it knows about no screen at all**. It came out of
  `internal/screens`, which was two thirds of the Go in the repo in one package: what moved is
  everything a scene draws *through* — the table, the clock, the movers, the card faces, the
  panels belonging to no screen, the prose — and what stayed is the scenes. **The arrow only
  points one way, and that is checkable rather than a habit**:
  `.claude/skills/audit/tools/pkgsplit.go` reports every unexported name that would have to
  cross a proposed line in either direction, and a *back edge* — a shared file reaching into one
  screen — is the finding. There are none today.
  - **Sizes are the frame's, placement is often the screen's.** A control's measurements live in
    `ui/frame.go`; `ControlColumnSlot`, which counts up from the action-point bar, stayed on the
    combat screen. That is the line to reason against when deciding where something new goes.
  - **It still links Ebitengine**, so the split buys readability and a boundary, **not** a
    display-free test run. The packages that can be tested without one are still `internal/combat`,
    `internal/session` and the rest below them.
- **Nothing above `screens` knows a scene exists except `game`**, which holds the registry. That
  is what makes adding a screen a local change.

### The game loop, in code

`main.go` builds the `game.Game`, loads assets/fonts/data once, builds the run, then hands control
to `ebiten.RunGame`. It does **not** wire up widgets — scenes build their own, and the one control
belonging to no scene is built by `game` itself.

`internal/game` then drives `Update` / `Draw` / `Layout` at a fixed 1920x1080 internal resolution,
picking the active scene out of one registry. `internal/ui/scene.go` is the `Scene` contract;
`Init` may run more than once, because a screen can be re-entered.

### The run loop, in play

**The run owns where it is, and one file moves it on.** A scene that has finished calls
`screens.advance`; nothing names its successor.

```
fight  →  reward  →  shop  →  choice  →  fight ...
```

- **`session.Phase` is the station** — see `internal/session/flow.go`, which holds the order.
- **`screens.phaseScreens` is which scene draws it** — see `internal/screens/flow.go`. A phase with
  no scene registered is walked past rather than drawn blank, which is what lets the loop name
  a station before it has a screen. The room choice is the one being walked past today.
- **Adding a screen is therefore three edits**: a phase in `session/flow.go`, an entry in
  `screens/flow.go`, and an entry in the registry in `internal/game`. No existing scene changes.
- **`screens.enterRun` is the same table read at the door**. Continue puts the player
  on whichever scene draws the station the run was saved at, falling back to the combat screen for a
  phase with no scene — the same courtesy `advance` extends during play.

### The three rules a change most often breaks

- **`internal/combat` decides rounds; a screen only replays them.** Never change a rule to make a
  screen look right — say which of the two is wrong and let the owner decide. That is a
  game-design call and it ripples into the tests and the balance.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before
  playback begins, so playback speed, **the player's game-speed setting** — `screens.SetSpeed`
  scaling `clock.go`'s one beat — a dialog that pauses the cursor,
  the debug flags, `internal/trace`, `internal/idle` and the scripted demo may all alter pacing and
  none of them may alter results.
- **Working state belongs to the narrowest thing that needs it.** One screen reads it → the scene.
  It has to outlive a fight → `internal/session`. Every screen genuinely needs it →
  `state.GlobalState`. Nothing else earns a place in global state.

### What is *not* in a package doc, because it is a tripwire

- **Never call the `math/rand` package-level functions.** See the `randomness` skill.
- **A new `EventKind` needs a choreography entry**, or `internal/screens` fails a test. An event
  with no picture and an event whose picture was forgotten otherwise look identical.
- **`ConceptID`, `StatusID` and `Element` are append-only**, and none may be
  serialized. Arrays and caches are indexed by the ordinal, so inserting one mid-enum silently
  re-points everything already stored.
- **Re-run `tools/handodds` and `tools/seeds` after touching the deck.** Both measure facts about
  one particular deck, and nothing fails when they go stale.
- **`internal/combat` holds the purse while a round resolves, and `KindVitae` is not the
  payment.** `Duelist.Vitae` is seeded from the run at the top of each round,
  stepped as the round pays, and the run is handed the **difference** — see `screens.payHeldVitae`.
  Summing `KindVitae` events to move a purse is the old way and now double-pays. The rules got a
  purse because a relic wanted to read one; the doc comments saying they have none are corrected.
- **An achievement nobody can earn is invisible**, and that is what `internal/achieve` exists to
  refuse. Every word `data/achievements.json` may write — a trigger kind, a clause mode, an
  axis, a moment name, a counter name — is a closed vocabulary checked at package init, so a
  misspelling fails the launch rather than producing a row that sits locked forever. **A new
  moment is a constant in `internal/achieve/catalog.go` plus the one call site that raises it**,
  never something a file can assert into existence. See MECHANICS.md §Achievements.
- **`profile.Counters` is bumped in memory and written when a duel ends**. A card played is not
  a disk write; `screens.settleCounters` is the one place the tallies land, on a win and on a
  defeat alike. A crash mid-duel loses that duel's counts, which was taken deliberately rather
  than discovered.
- **Re-run `tools/relicsheet` after touching `relics.json`, and delete the PNG of a relic you
  removed.** The sheet writes a file per relic and never cleans up, so a deleted record leaves
  an orphan picture in `docs/sheets/relicsheet/` that no page links and nothing fails on.

### Drawing idioms

- Sprites are drawn via `colorm.DrawImage` so a `colorm.ColorM` can tint/hue-shift them; buttons
  and shapes use `vector.DrawFilled*` into a scratch `ebiten.NewImage`.
- Positioning convention: translate by `-w/2, -h/2` first to center the origin, then translate
  to the target coordinate. Buttons store `ScreenX`/`ScreenY` as their *center*, and both
  `UpdateButton` (hit testing) and `DrawButton` re-derive the top-left from it.
- **Rounded rectangles are done one way: `internal/cards/shape.go`, in plain Go**
. There were two for a while — health bars drew an opaque mask and composited
  it with `ebiten.BlendSourceIn`, which cards could never use, since that path takes an
  `*ebiten.Image`, its body is `vector.DrawFilledCircle`, and `BlendSourceIn` is a GPU blend
  mode, none of which exist without a graphics context. `internal/cards` must render without
  one so the review tools can call it, so the window-free rasterizer is the one that survived:
  the mask lost its last caller when both fighters became cards and their bars moved into
  `internal/cards`. Corners are hard-edged there, because the glyphs on them are 1:1 pixel art.
  **A new rounded shape goes in `shape.go` whatever is drawing it** — a GPU-side rasterizer
  would put the second silhouette back, and it is the one a review tool cannot reach.

## Art

**`assets/` is grouped by what a file is for**: `game/` (fonts, title screens), `enemy/`,
`relic/`, `effect/`, `upgrade/`, `sounds/`. The `//go:embed` paths are relative to `embed.go`, so
refiling something is one line there and nothing anywhere else.

**A map key is not tied to a file path.** Keys are the lookup names used across the game and
`data/*.json` writes them down; tying one to a path would mean a data migration every time a
file was refiled. A named asset is three edits: the file, an `//go:embed` var, and a map entry.

**The creature pictures are the exception, and they are a *family* rather than named
assets.** There are far too many to name one at a time, so `//go:embed enemy/*.png` pulls the
directory in as an `embed.FS` and `LoadImageData` walks it, keying each by filename stem.
**There is one picture per record per element** — `enemy/goblins-serf-fire.png` is
`goblins-serf-fire`, which is what `data.MotifRecord.ArtKey` builds out of the record's `Art`
field and the element the floor dealt it as — so a fire goblin serf and an ice goblin serf are two
drawings of one creature. **The consequence is exactly what the three-edit rule protects against:
a picture's key is tied to its filename**, so renaming one means editing the `Art` field of the
record that names it. Reach for the glob only when a *set* of files is being added; a one-off
asset still gets its own var.

**`default-enemy.png` is the whole of the fallback**, and nearly every record draws it today. A
record whose own picture has not been generated yet falls back to it rather than drawing a hole,
so a blank face means art nobody has made rather than a name nobody spelled right.
`TestEveryOpponentHasSomethingToDraw` holds that the placeholder is embedded and
`TestNoTwoRecordsDrawTheSamePicture` holds that no two records claim one key — the map is flat,
and two records on one key is one lookup with two answers.

They are handed out as **bytes, not `*ebiten.Image`** — they are drawn into a card by
`internal/cards`, which has no graphics context, and decoding every one of them at startup would
cost tens of megabytes of resident memory for pictures most runs never show.

**`assets/effect/` is the status badges**, drawn as a centered row along the bottom of the enemy
card by `internal/cards` — so they go through `LoadImageData` as bytes, exactly like the relic art
and for the same reason. `screens.statusBadges` is the lookup and **it is read off each record's own
`Badge` in `statuses.json`** rather than keyed by element — because a badge belongs to the
*status*, so a status arriving by an affix or a boss rule draws the same picture whatever brought
it. `default-effect.png` is the fallback.

**Nothing in the game draws a loose sprite.** There are no creature sprites in `assets/`;
`Combatant` has no `Sprite` field and `entities` imports no Ebitengine at all. **Both duelists
are cards**, in opposite corners, and both state their life the same way — a bar over a
fraction, at identical offsets on the two styles so the pair can be compared across the
screen without measuring. **The enemy's carries a badge row under its fraction and the
player's does not**, which is not a break of that rule: an enemy wears no relics, so nothing can
put a status on the player to draw.

**A creature is one still picture and there is no animation anywhere.** Animating one means
generating frames from the same brief `tools/creatureprompt` assembles, not going back for a sprite
sheet — there is none to go back for.
