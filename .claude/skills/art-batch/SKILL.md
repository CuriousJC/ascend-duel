---
name: art-batch
description: How art gets reviewed and installed - generating a couple of options for one record and picking between them, or taking a whole replacement batch: which prompt and which Draw field the generator is given, backing up what is there before anything is overwritten, building the side-by-side page with tools/artcompare, filing the winner with tools/relicart, and regenerating what changed. Load before generating, comparing, installing or replacing any art in assets/ - relic, card, essence, rune, stone, other.
---

# Installing a batch of art

Art arrives as a directory of PNGs from a generator, named for the stems the catalog already
writes in its `Art` fields. **The job is never "copy them in".** A batch is dozens of independent
decisions, some of the new pictures are worse than what they replace, and the replaced picture is
gone the moment it is overwritten — so the order below exists to keep the old one reachable while
the choice is being made.

`tools/artcompare` is the tool. Its own doc comment holds the argument; this is the procedure.

## Two entrances, and they are not the same job

**A replacement batch** — new pictures for records that already have one. That is the numbered
procedure below: back up, install, compare, apply picks.

**Options for a record that has no picture yet** — "generate and compare a couple of options for
this relic". Nothing is being replaced, so there is no backup and no `current` set worth showing;
what the page compares is candidate against candidate. Read
[the options path](#options-for-a-record-that-has-no-picture-yet) at the foot of this file, then
come back to step 4.

## The order, and why it is this order

### 1. Look before touching anything

```powershell
Get-ChildItem <batch dir> | Measure-Object            # how many
Get-ChildItem <batch dir> | Where-Object { $_.Length -eq 0 }   # the empty ones
```

**A batch out of a generator can hold zero-byte and truncated files**, and it has. They are not a
reason to stop; they are a reason not to copy them over good pictures. Name them in the reply.

Check the names against the catalog and the sizes against what the family commits at — a relic,
essence, rune, stone or other card is authored at **200x280**, the card faces at 200x280, and a
form mark at 256x256. A batch delivered at some other size is a thing to raise before installing,
not after.

### 2. Back up what is installed, before overwriting anything

```powershell
New-Item -ItemType Directory -Force artreview\<catalog>-original | Out-Null
Copy-Item assets\<catalog>\*.png artreview\<catalog>-original\
```

Git has the old pictures either way, but a review needs them **side by side as files**, and
`artreview/<catalog>-original/` is found by the tool as the set `original` with no flags. Do this
even when the batch looks like a clear improvement: the comparison is what establishes that.

### 3. Install the batch, skipping the files that are not pictures

```powershell
Get-ChildItem <batch dir>\*.png | Where-Object { $_.Length -gt 0 } |
  ForEach-Object { Copy-Item $_.FullName (Join-Path "assets\<catalog>" $_.Name) -Force }
```

**The length filter is the whole point.** A zero-byte file copied over a good picture is the one
mistake in this procedure that loses something, and the empty ones then show as gaps on the page,
which is how they get noticed at all.

**A straight copy is only right for pictures already at the card's own size whose records already
name them.** A batch straight out of the generator is 1060x1484 and its records may have an empty
`Art`, and **that is `tools/relicart`'s job, not a `Copy-Item`** — it reduces to the card's size,
commits under the right asset directory and writes the `Art` field, which is the step that fails
silently when it is done by hand:

```powershell
go run ./tools/relicart -kind relic -n      # say what would happen, touch nothing
go run ./tools/relicart -kind relic
```

Its inbox is `.scratch/to-process-<kind>-art/` and its kinds are `relic`, `essence`, `rune`,
`stone`, `card` and `other`. A stem naming no record is refused rather than filed.

The installed art is always the set `current`, so installing first is what makes the page a
comparison of *the game as it now stands* against what it was.

### 4. Build the page

```powershell
go run ./tools/artcompare                    # card, every set it finds
go run ./tools/artcompare -catalog relic
go run ./tools/artcompare -catalog relic -set warm=artreview\relic-warm -set cool=artreview\relic-cool
```

- Catalogs: `card`, `relic`, `essence`, `rune`, `stone`, `other`. Enemy and boss portraits are
  **not** in the table — licensed creature art arrives once and nobody generates three of it.
- Any number of sets. `artreview/<catalog>-<label>/` is discovered as `<label>`; `-set label=dir`
  replaces discovery outright.
- Output is `artreview/out-<catalog>/index.html`. **Open it for the owner** rather than describing
  it — this is a page whose whole purpose is to be looked at.

Every cell is `cards.Render` at that catalog's own style, so the comparison is of the card as it
will be dealt, type over picture. **Never preview the raw art instead**: a playing card's picture
is drawn under five pieces of near-black type and an essence's under the sentence it prints, and
art that reads well bare can lose all of it there. That is the same failure the old glyph sheet
had when it previewed at a scale the game did not use.

### 5. Apply the picks

The page writes a copy list — one `Copy-Item` per record, grouped by the set it came from. The
owner pastes it back. **Do not run it blind:**

```powershell
foreach ($n in @("<stem>","<stem>")) {
  $a=(Get-FileHash "assets\<catalog>\$n.png").Hash
  $b=(Get-FileHash "<source dir>\$n.png").Hash
  "{0,-18} same={1}" -f $n, ($a -eq $b)
}
```

**Lines that are already no-ops are normal and worth saying so.** A stem whose new file was
zero-byte was never installed, so a pick of the old picture for it is confirmation rather than a
change. Report how many of the lines actually moved a file.

### 6. Regenerate what the pictures changed, and nothing else

A catalog's own sheet draws its art, so it is stale the moment a batch lands:

```powershell
go run ./tools/cardsheet      # after assets/card
go run ./tools/relicsheet     # after assets/relic
go run ./tools/essencesheet   # after assets/essence
go run ./tools/runesheet      # after assets/rune
go run ./tools/stonesheet     # after assets/stone
```

**Regenerate the one sheet, not `tools/sheets`.** A full run rewrites every binary under
`docs/sheets/`, most of that weight being the two roster sheets and their photographic portraits,
and a sheet rebuilt in a commit that changed nothing about it is pure history weight.

Then rebuild the compare page, so `current` shows what is actually installed.

### 7. Report, and stop

Say what changed, which picks moved a file, which lines were no-ops, and what is still wrong with
the batch — the empty files, a missing stem, a record still drawing its default. **Leave everything
unstaged**; the owner reviews diffs in VS Code.

## What `artreview/` is

Gitignored, anchored `/artreview/`, and the whole working directory of this procedure:

```
artreview/
  <catalog>-original/     the pictures that were installed before the batch
  <catalog>-<label>/      a candidate batch, found as the set <label>
  out-<catalog>/          the built page — rewritten every run, delete whenever
```

**Nothing under it is ever committed.** The winners are installed into `assets/` and committed
there; the losers were never the game. That is what lets `tools/artcompare` be a committed tool
with no committed output, and it is why this page does not live in `docs/sheets/` — that directory
is a report on the catalog that shipped, and this is a picture of a decision being made.

## The things that go wrong

- **Overwriting before backing up.** The one irreversible step in the procedure. Step 2 before
  step 3, always, even for a batch that is obviously better.
- **Copying zero-byte files over good ones.** Filter on length; do not trust a directory listing.
- **Judging the raw art.** See step 4.
- **Renaming to fit.** A batch whose stems do not match the catalog is a batch to ask about — the
  `Art` field on the record is the name, and quietly renaming files makes `data/` and `assets/`
  disagree in a way only a launch catches. `TestEveryRelicDrawsSomething` and its siblings fail on
  a key naming no file, which is the tripwire, not a substitute for asking.
- **Adding a record to `data/` because a picture turned up for it.** A picture with no record is a
  file nobody draws; say so. Authoring the record is a catalog decision and belongs to the owner.
- **Running `tools/sheets` out of habit.** See step 6.
- **Leaving an orphan PNG behind.** `tools/relicsheet` writes a file per relic and never cleans
  up, so a removed record leaves a picture in `docs/sheets/relicsheet/` that no page links.

## Options for a record that has no picture yet

"Generate and compare a couple of options for this relic." Nothing is being replaced, so steps 2
and 3 do not apply and the page is candidate against candidate.

**What the generator is given is two things, and they live apart.** The generic prompt is
`docs/art/<kind>_art_prompt.MD` — the style, the composition, the ground, the lighting, and the
clause saying the card reserves nothing, so a prompt must not describe a title band or a corner
kept clear. What is about *this* record is the `Draw` field on the record itself, pasted in as the
record's own JSON. There is one prompt per catalog: `relic_art_prompt.MD`,
`essence_art_prompt.MD`, `stone_art_prompt.MD`, `other_card_art_prompt.MD`, and
`rune_art_prompt.MD`, which is not a prompt but the closed list of rune body plans pasted into the
essence one.

1. **Check the record exists and carries a `Draw`.** An empty `Draw` means nobody has written the
   brief; writing one is authoring and is the owner's call — offer a draft, do not file it as
   settled. A record that does not exist at all is a catalog decision, not an art one. The
   [`relics`](../relics/SKILL.md) and [`data`](../data/SKILL.md) skills own that half.
2. **Hand over the prompt plus the record's JSON**, verbatim, once per option. **Nothing here
   generates images** — the pictures come back from an image model the owner runs.
3. **Put each option in its own set**: `artreview/relic-a/`, `artreview/relic-b/`, each holding
   the file named for the record's stem. Two directories of one file is normal and the tool does
   not mind.
4. **Compare at whatever size they came back.** `cards.Render` scales a bleeding picture to cover
   the card with CatmullRom, which is the same filter and the same single step `tools/relicart`
   reduces with — so the preview is faithful to what would be committed. Reduce first only if the
   generator returned something not at the aspect the card wants.
   ```powershell
   go run ./tools/artcompare -catalog relic -set a=artreview\relic-a -set b=artreview\relic-b
   ```
   **`current` is worth including whenever the record already draws a picture**, even a default
   one, because "is this better than the fallback" is a real question. With `-set` given, discovery
   is replaced outright, so name it explicitly: `-set current=assets\relic`.
5. **File the winner with `tools/relicart`**, not by hand — copy it into
   `.scratch/to-process-relic-art/` under the record's stem and run the tool, so the reduction
   happens and `Art` is written. Then regenerate that catalog's sheet.

**Two options is not the only shape.** The tool takes any number of sets, and one prompt run three
ways is three directories; the page's per-set "only" button is how each one is read as a whole
before they are read against each other.

## Adding a catalog to the tool

One entry in `catalogs` in `tools/artcompare/catalogs.go`: the flag word, the assets directory,
the JSON it reads, and a `Subjects` func building one `subject` per record — the key, the picture's
stem from the record's own `Art` field, a group heading, a caption, and the `cards.Spec` **without
its Art**. The caller varies only `Spec.Art`, which is what makes a row a fair comparison.

**Take the stem from `Art` and not from `ArtKey()`.** The key methods fall back to a default face,
so an undrawn record would compare `default-relic` against itself; an empty `Art` means undrawn and
the tool lists it at the foot of the page, which is the catalog's own backlog and worth seeing.
