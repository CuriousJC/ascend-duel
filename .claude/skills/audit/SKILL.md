---
name: audit
description: The whole-project audit - the layout, the patterns the code is written in, and the eight instruments that find rot the compiler cannot see: dead symbols, prose that names code which no longer exists, catalog boilerplate, package boundaries that have quietly stopped holding, and unbounded caches. Load it when the owner asks for an audit, a refactor pass, a health check or a "we are at a good point, what should we tidy", and before any refactor that crosses more than one package.
---

# The audit

**This is the milestone pass.** It is not a code review of a change — that is `/code-review` — and
it is not a bug hunt. It is the thing to run when the game is at a resting point and the question is
*what has gone quietly wrong while we were building*.

**The failure it exists to catch is rot that nothing fails on.** Every check in here found something
the first time it was run, and not one of those things broke a test, failed a vet, or showed up in a
diff. That is the whole character of the problem: the compiler holds the code together and nothing
at all holds the *prose* together, so the design record drifts away from the game one true-at-the-time
sentence at a time.

## Before anything: the rules this pass is under

- **Findings are actioned, not filed** *(owner's call, 2026-09-17)*. There is no audit report in
  `docs/`, no seventh stream, and no `TODO.md` entry — the owner's standing rule that `TODO.md` only
  grows when he asks for something tracked applies here exactly as it does everywhere else. What an
  audit produces is a working tree and a summary in the reply.
- **The map below is the deliverable that persists.** It is in this file rather than in the reply
  because it is the orientation the *next* audit needs, and re-deriving it costs a day.
- **Leave the work unstaged.** The owner reviews diffs in VS Code. Do not stage, commit or push.
- **A refactor that changes behavior is not a refactor.** Every move in here is meant to be
  outcome-neutral. Where a change does alter what the game does, it stops being part of the audit and
  becomes a thing to raise in the reply and let the owner decide.

## The map

Read this before reaching for an instrument; the instruments assume it.

### Shape, by weight

Roughly two thirds of the Go in this repo is one package. That is the single fact that governs every
structural question here.

| Layer | Packages | What it is |
|---|---|---|
| **Bottom** | `data` `profile` `seeds` `models` `assets` `idle` `trace` `music` | import nothing of ours |
| **Rules** | `combat` `pyramid` `tutorial` `achieve` `decks` `entities` `carddesc` | no Ebitengine, testable without a window |
| **Run** | `session` `state` | what outlives a fight |
| **Drawing** | `systems` `cards` `actions` | no `*ebiten.Image` created in `cards`, which is what lets the sheets render |
| **Screens** | `screens` | every scene, and the shared layer they all draw through |
| **Frame** | `game` `main` | the loop and the two controls belonging to no scene |

Regenerate the arrows rather than trusting a drawn picture:

```powershell
go list -f '{{.Name}}: {{join .Imports " "}}' ./... | grep curiousjc
```

### The patterns the code is written in

These are the grammars a change is expected to be written *in*. An audit asks whether each one still
holds everywhere, and whether a new thing has quietly invented a second way to say something.

- **A catalog is a JSON file plus a `_data.go` loader.** `data/<name>.json` and
  `data/<name>_data.go`, one `XData` struct, one `LoadX()`, and — where order matters — an `XOrder`
  over the sorted keys and an `XFileOrder` over the file's own order. Nothing in `data/` imports
  upward, so a catalog that needs rules types is handed to them by `internal/decks` or validated by
  the package that owns the vocabulary.
- **A closed vocabulary is refused at load, never defaulted.** Every word a file may write — a verb,
  a moment, an axis, an anchor, a condition, a trigger — is a constant in Go, and a record inventing
  one fails the launch. The failure this prevents is a record that sits in the catalog doing nothing
  and looks exactly like a record nobody has earned yet.
- **A widget is a struct in `models` and `Update*`/`Draw*` free functions in `systems`.** No method
  in `models` draws. `Button`, `Slider`, `Scrollbar`, `Tooltip`.
- **Name one color and scale it.** `ColorAtStrength` toward black on a dark ground,
  `ColorToward` toward the surface on a light one, and the game's ground is light — so
  `ColorAtStrength` is the exception, not the default.
- **Presentation may never change an outcome.** `ResolveRound` decides a whole round before playback
  starts. Playback speed, the speed setting, every flight, every debug flag, `trace`, `idle` and the
  demo may alter pacing and may not alter results.
- **Never serialize an ordinal.** `ConceptID`, `Element`, `StatusID`, `Phase` are append-only
  indices into arrays. What gets written to disk is a name.
- **Randomness is an injected `*rand.Rand` off a named, salted stream.** No package-level
  `math/rand`, and a stream is only ever advanced by its own concern.
- **A picture is generated art keyed by filename stem, or it is nothing.** No fallback that hides a
  missing file; a key naming nothing draws nothing, which is the honest failure.
- **Cards fly and dissolve; they never appear or swap.** Anything that changes position travels;
  anything that becomes a different card morphs into it.

### Where the prose lives, and why it rots

Six streams, and the audit's central problem is that two of them are loaded into context every
session and nothing checks them.

| Stream | Rots how |
|---|---|
| `CLAUDE.md` | names symbols that have been deleted; carries counts that went stale in a week |
| `MECHANICS.md` | states a rule the code stopped implementing |
| `.claude/skills/*` | describes a procedure against files that moved |
| package `doc.go` | **the worst of the four** — see below |
| file header comments | rarely wrong; they are edited in the commit that changes the file |
| `data/*.json` | cannot rot; it is the thing itself |

**A package's `doc.go` drifts further than any file comment, and the reason is structural.** The
repo's own rule puts a file's header comment *below* its `package` clause and keeps `doc.go` as the
only file whose comment sits above one. So an edit to `chrome.go` naturally updates the comment at
the top of `chrome.go` and touches nothing in `game/doc.go` — which is still describing a mute
button that became a settings cog, next to a ledger that arrived afterwards. **Every `doc.go` that
enumerates something is a list that will go out of date**: the files in a package, the moments in a
grammar, the flags in a struct, the scenes in a registry.

## The instruments

Eight, in the order to run them. The first four are cheap and find most of it.

### 0. Build the analyser

Two of the checks need a type-aware view of a package. The tool is in this skill's `tools/`
directory and **is deliberately not in the module** — it needs `golang.org/x/tools/go/packages`, and
a product that will be sold does not take a dependency for the benefit of an audit. It builds into
`.scratch/`, which is gitignored.

```powershell
mkdir -Force .scratch\pkgsplit
Copy-Item .claude\skills\audit\tools\pkgsplit.go .scratch\pkgsplit\main.go
Copy-Item .claude\skills\audit\tools\pkgsplit.go.mod .scratch\pkgsplit\go.mod
cd .scratch\pkgsplit; go mod tidy; go build -o pkgsplit.exe .; cd ..\..
```

`.claude` starts with a dot, so the go tool ignores it and `go vet ./...` never sees the source.

### 1. The baseline

Nothing else is trustworthy until these are clean, and all three were clean the first time.

```powershell
gofmt -l .
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go test ./...
```

### 2. Dead code — `staticcheck`

The single highest-yield instrument, and the one that needs saying out loud because it is not
installed by default and the stale copy on this machine crashed on Go 1.26.

```powershell
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
foreach ($t in "debugtrace","idleexit","demoplay","scenario") { staticcheck -tags $t ./... }
```

**Run it under every build tag and delete only the intersection.** Each tag selects a different
file, so a symbol can be live in one configuration and dead in another — and reading the five lists
one after another is not the same as intersecting them. Doing this by eye cost two wrong deletions
the first time, both live only in `combat_demo_on.go`:

```bash
staticcheck ./... 2>&1 | grep U1000 | sed 's/ is unused.*//' | sort > /tmp/dead.txt
for t in debugtrace idleexit demoplay scenario; do
  staticcheck -tags $t ./... 2>&1 | grep U1000 | sed 's/ is unused.*//' | sort > /tmp/u.txt
  comm -12 /tmp/dead.txt /tmp/u.txt > /tmp/keep && mv /tmp/keep /tmp/dead.txt
done
cat /tmp/dead.txt   # dead under all five; only these may be deleted
```

A symbol that survives this and is deleted anyway shows up as a *vet failure under one tag*, so run
the whole tag matrix after deleting — that is the backstop, and it is the reason the matrix is in
the baseline above rather than only at the end.

What it finds here, in order of how much it matters:

- **`U1000` — unused.** Constants left behind when a layout moved, helpers whose last caller went.
  Delete them. **The exception to check before deleting is a `_test.go` helper**, which may be
  waiting for a test somebody meant to write; ask rather than assume.
- **`SA4000` — identical expressions either side of an operator.** Both instances in this repo were
  in tests, and both were deliberate — a comparability assertion written as `x != x`, and a
  deliberate double call through `||`. Deliberate or not, a test that reads as a tautology is a test
  the next reader has to re-derive. Rewrite it so it states what it means; do not delete it.
- **`SA1019` — deprecated.** Ebitengine renames things across minors. A block of these is a
  mechanical migration and should be done in one pass, not one call site at a time.

### 3. Prose that names code which does not exist

**The instrument this skill exists for.** Nothing else in the toolchain looks at a comment.

The repo's convention is that a symbol named in prose is written in backticks, which is what makes
this checkable. Take every backticked identifier out of the Markdown and the Go comments, and look
each one up in the tree.

```bash
# every identifier the prose names, in backticks
grep -rhoE '`[A-Za-z_][A-Za-z0-9_.]*`' CLAUDE.md MECHANICS.md .claude/skills/*/SKILL.md \
  $(git ls-files '*.go') | tr -d '`' \
  | grep -vE '\.(go|json|md|MD|png|mid|exe|html|txt)$' | sort -u > /tmp/cited.txt

# strip the package qualifier and look each one up
sed -E 's/^[a-z][a-z0-9]*\.//; s/\(\)$//; s/\..*$//' /tmp/cited.txt | sort -u \
  | while read s; do
      [ -z "$s" ] && continue
      grep -rqE "\b$s\b" --include=*.go internal data tools main.go assets || echo "$s"
    done
```

**It is noisy and the noise is the point** — it will hand back English words, JSON field names and
stdlib symbols alongside the real hits. Read the list; the real ones are obvious. Then classify each:

- **A tombstone** — prose whose subject is a thing that was deleted, written to record the deletion.
  `CLAUDE.md`'s own rule is that cut means deleted rather than tombstoned, because these files are
  loaded into context every session and a record of things that do not exist is a running cost that
  grows without bound. **Delete the sentence** — unless the code still has a shape only the dead
  mechanic explains, or the idea keeps being re-proposed, in which case **ask before keeping it**.
- **A wrong rule** — prose stating something the code no longer does. This is the dangerous kind and
  it is never a judgment call: fix it.
- **A rename** — the thing exists under another name. Fix the reference.

### 4. `doc.go` against its own package

Every `doc.go` that lists something is a list to re-derive. There is no tool for this; it is a read.
Take each enumeration in the doc and check it against the code:

```bash
for f in $(find internal -name doc.go); do echo "### $f"; cat "$f"; done
```

The questions that caught every drift found so far:

- Does the **file list** name files that exist, and does it miss files that arrived since?
- Does a **count** in the prose — seven moments, four scenes, three flags, two debug views — still
  match what the code declares?
- Does the doc describe a **mechanism that was replaced**, in the present tense?
- Is the doc **contradicting itself**, a paragraph written before a change sitting above one written
  after it?

### 5. Catalog boilerplate

The `data` package grows one near-identical loader per catalog, so the census is worth running
whenever a catalog has been added.

```bash
grep -hnE '^func Load' data/*.go | wc -l
grep -hnE '^func [A-Za-z]+(File)?Order\(' data/*.go
grep -hoE 'panic\("[^"]*"' data/*.go | sort | uniq -c
```

Three things to look for: how many `Load*` bodies are the same unmarshal-and-panic; how many
`*Order` functions are `sort.Strings` over a map's keys; and whether the panic messages agree with
each other — the grammar drifts (`"our RelicData"` against `"runes.json"`) because each was written
next to its own catalog.

### 6. Package boundaries — `pkgsplit`

**The instrument that says whether a package boundary still means anything.** It type-checks a
package, takes a proposed partition of its files, and reports every unexported identifier declared
on one side and used on the other.

```powershell
.scratch\pkgsplit\pkgsplit.exe -pkg ./internal/screens -move "ground.go,travel.go,clock.go,modal.go"
```

Two numbers come back and they answer different questions:

- **Forward edges** — declared in the moved set, used by the rest. This is the *cost* of a split:
  every one has to be exported.
- **Back edges** — declared in the rest, used by the moved set. **This is the finding.** A file that
  is supposed to be shared, reaching into a particular screen, is a boundary that has already been
  broken; the split merely makes the compiler say so. Run it on a partition you have no intention of
  executing, purely to read the back edges.

**Zero back edges does not mean zero work, and here is what it misses.** All three of these were
found by the compiler after the tool reported a clean cut, and all three cost an hour:

- **It counts only unexported names.** A symbol already exported and used across the proposed line
  is invisible to it, because nothing would have to be renamed — but it is still a dependency, and
  if it points the wrong way it is still a cycle. `go build ./<moved package>` is the check.
- **A method on a type that stays behind cannot move with its file.** Go requires a method to be
  declared beside its receiver, and the tool reports *uses*, not receivers. Grep the moved files for
  methods on scene types before moving anything.
- **A generic constraint interface does not propagate a rename.** `gopls rename` will not follow
  `mover[T any]` to its implementations, so renaming one implementation's method silently breaks
  every other. Rename that whole family by hand, together.

**Use `gopls rename`, never a textual one.** A third of the names in this package's export list also
occur inside string literals, so `sed` corrupts them. Capitalisation is length-preserving, so a list
of declaration positions stays valid across the whole batch — but a method must be identified by
*receiver and name*, because several types share names like `tick`, `done` and `draw`, and
deduplicating by bare name picks an arbitrary one. `pkgsplit -positions` emits receiver-qualified
identities for exactly this reason.

**Drive the rest off the compiler's own output.** After the move it will name every unresolved
symbol with a file, a line and a column, and for a renamed field it names the replacement too
("but does have X"). Rewriting at that position cannot touch a comment or a string. Two cautions:
`go build` does not compile test files, so finish with a `go vet ./...` loop; and a loop that
re-qualifies whatever is still undefined will happily produce `ui.ui.ui.` if it cannot resolve
something, so collapse repeats each pass and stop when the error stops changing.

### 7. Caches and per-frame allocation

The game is a 60 TPS loop, so the two failure modes are work repeated every frame and caches that
only grow.

```bash
grep -rn "ebiten.NewImage" --include=*.go internal | grep -v _test
grep -rnE "^var \w*[Cc]ache|map\[.*\]\*ebiten.Image" --include=*.go internal | grep -v _test
grep -rn "Dispose" --include=*.go internal
```

For every cache, three questions: **what is the key**, **what bounds it**, and **is the stated bound
still true**. The last one is where the rot is — a cache keyed on a struct is bounded by that
struct's field space, and a comment claiming a bound was written when the struct was smaller. An
`*ebiten.Image` is GPU memory and nothing in this tree calls `Dispose`, so an unbounded image cache
is an unbounded VRAM leak.

### 8. Tool duplication

`CLAUDE.md` argues against a shared library under `tools/`, with `roster` and `hands` as the two
reasoned exceptions. That argument was written when the sheets genuinely shared nothing. Re-check the
premise rather than the conclusion:

```bash
grep -rhoE '^func [a-zA-Z][A-Za-z0-9]*' tools/*/*.go | sed 's/^func //' | sort | uniq -c | sort -rn | head -25
```

A helper name appearing in five or eight tools is five or eight copies of one function, and the
"they share nothing" premise has expired.

## What an audit does not do

- **It does not add entries to `TODO.md`.** Noticing something is not a reason to file it. Say it in
  the reply.
- **It does not touch `README.md`.** Owner territory; noticing it has gone stale is something to say,
  not something to fix.
- **It does not retune balance.** A cost, a multiplier or a stat line changed during a tidy-up is a
  balance change nothing simulates and no test catches.
- **It does not delete a `[?]` or a `TODO.md` entry** because the work looks done. Those are the
  owner's.
- **It does not rewrite a design decision it disagrees with.** Raise it once in the reply and move
  on.

## Finishing

```powershell
gofmt -l .
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go test ./...
staticcheck ./...
go run .            # it has to still play
```

Then summarise in the reply: what was deleted, what was moved, what was corrected, and the list of
things noticed and deliberately left alone. **The last list is the one the owner most needs**, since
it is the only record that a thing was considered.
