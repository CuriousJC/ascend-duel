---
name: motifs
description: The roster grammar - how a creature is written as data under data/motifs, the closed vocabularies a record draws on, what a realm is, how the growth curve reads a base stat line, and what the loader refuses. Load before adding a motif file, adding or changing a record, authoring creatures or bosses, touching data/journey.json, or wiring anything that picks an opponent. Also the motif analyzer: given a proposed record, which tier/element fights it fills, which are still short, whether the motif would still pass, and whether its bases sit where its neighbours do. And the completeness worklist: what each motif is still missing - briefs, creature art, rooms, room art - and what to author next.
---

# Motifs

**A realm is a motif and an element.** The journey picks one whole motif and one of the five
elements, and the realm's three rooms — outer chamber, inner chamber, portal room — are three records
of that motif dealt as that element. So a fire goblin realm is three goblins in fire, and what the
player walked into is something they can plan against.

That is the whole reason the roster is a directory rather than a file. The question asked of the
catalog is never "is this creature good"; it is "can this motif fill a realm at every element",
and a creature pool split across two files is a question neither half can answer — so a motif's
creatures stay in one file, and its rooms sit beside them in a second.

## Where everything is

| Thing | File |
|---|---|
| The roster | `data/motifs/<motif>/motif.json`, one directory per motif |
| The rooms a motif's fights are drawn in | `data/motifs/<motif>/backdrops.json`, optional |
| The structs, the loader, every refusal | `data/motifs_data.go`, and `data/backdrops.go` for the rooms |
| The pictures | `assets/motifs/<motif>/creature/*.png` and `.../backdrop/*.jpg`, keyed by filename stem |
| How full each motif is, and what is still TBD | `go run ./tools/motifreport` → `docs/sheets/motifreport/` |
| Every room's pictures, beside the brief each was painted from | `go run ./tools/backdropsheet` → `docs/sheets/backdropsheet/` |
| The journey's height and its two growth rates | `data/journey.json`, `data/journey_data.go` |
| The journey: which motif and element each realm takes | `internal/journey` |
| A record's cards becoming a deck | `internal/decks/enemy.go` |
| A record becoming a fighter | `internal/entities/combatant.go` |
| The review page | `go run ./tools/motifsheet` |
| The art brief for one picture | `go run ./tools/creatureprompt` (`-backdrop <key>` for a room) |
| The style block every creature shares | `docs/art/creature_art_prompt.MD` |
| The style and tier blocks every room shares | `docs/art/background_art_prompt.MD` |

## The file

```json
{
  "Motif": "goblins",
  "Name": "Goblins",
  "Draw": "Goblins are small, wiry humanoids with too much face ...",
  "ElementDraw": {
    "fire": "Fire arrives as burn scarring and soot ...",
    "ice":  "Frost rides on a goblin rather than filling it ..."
  },
  "ValidRealms": [1, 6],
  "Records": [ ... ]
}
```

- **`Motif` is the key, and the directory name has to match it.** A motif directory holds
  `motif.json` and optionally `backdrops.json`, and any other file is refused. It is also the art family and the name
  of the generator prompt this motif's pictures come from.
- **`Draw` and `ElementDraw` are the shared half of the art brief** — see *A picture is four
  layers* below. Both are authored and ignored by everything that plays the game. An
  `ElementDraw` key that is not an element is refused; a missing one is not.
- **`Text` and `ElementText` are what the player reads** — on the portal that offers the motif, the
  realm's name and then these two: what the creatures are, and what the element does to them. Same
  shape as `Draw` and `ElementDraw` and the same rules — an `ElementText` key that is not an element
  is refused, and an unwritten line shows on the portal as TBD and is counted by the motif report.
  **Keep them to a sentence or two**: a portal panel is 620 pixels of prose wide.
- **`ValidRealms` is motif-level**, inclusive, `[0, 0]` for any realm. It is not per-record,
  because a realm takes a whole motif: a motif whose outer creatures were valid on realms 1 to 3
  and whose boss was valid on 4 to 6 could never theme a realm at all.

## The record

```json
{
  "Record": "goblins-outer-bomber",
  "Name": "Goblin Bomber",
  "Tier": "outer",
  "Art": "goblins-bomber",
  "Draw": "Hunched over a satchel of lit fuses ...",
  "Affinities": ["fire", "ice", "lightning", "earth"],
  "HP": 100, "DMG": 5, "Actions": 5,
  "Cards": [
    { "Label": "Fuse", "Verb": "attack", "Amount": 50,  "Cost": 1, "Copies": 4 },
    { "Label": "Lob",  "Verb": "attack", "Amount": 100, "Cost": 2, "Copies": 3 },
    { "Label": "Big Bang", "Verb": "attack", "Amount": 200, "Cost": 3, "Copies": 1 }
  ]
}
```

- **`Record` must read `<motif>-<tier>-<slug>`**, and the prefix is checked rather than trusted. A
  key that disagrees with its own tier is a record the coverage report counts in the wrong column,
  and the report is what the realm generator believes.
- **`Tier` is closed**: `outer`, `inner`, `boss`. Its index is the last term of the growth step.
- **`Title` is boss-only** and is refused on a creature.
- **`Art` is the picture family stem.** The face drawn is `<Art>-<element>.png`, so one record
  carries one picture per element it can be dealt as. A missing file falls back to
  `default-enemy`, so a blank face means art nobody has made rather than a name nobody spelled
  right.
- **`Affinities` is a non-empty subset of the five elements**, no repeats. `basic` is not one of
  them: a creature takes its realm's colour, and a realm has one.
- **`ElementDraw` on a record is optional and replaces the motif's line for that element** where
  it is written. It is for the one creature the motif's generic element direction does not fit.
  `data.MotifData.ElementDrawFor` is the rule.
- **`Cards` is authored per record.** Two creatures of one motif are two different fights, so they
  hold different cards rather than the same cards at different weights. **A card may not name its
  own elements** — the colour is the realm's — and costs run 1 to 3, because the cost column is
  tick marks stacked down a fixed band.

## The bases are step-zero quantities

`HP` and `DMG` say what a creature is worth **in the very first room of the journey**, whatever realm
it is actually met on. `journey.ScaleToFight` puts it where it stands:

```
step = (realm - 1) * FightsPerRealm + tierIndex      // outer 0, inner 1, boss 2
HP   = base.HP  grown at journey.HPGrowth,  once per step
DMG  = base.DMG grown at journey.DMGGrowth, once per step
```

**Stepping per fight rather than per realm** is what makes a realm's boss harder than its own
inner chamber and the next realm's outer chamber harder than that boss, with no constraint between
two separate numbers to get wrong.

So the two mistakes to watch for when authoring:

- **Do not write a late-band creature as a high stat line.** A dragon banded to realms 5–8 is not
  "250 HP". It is "about 2.5x a goblin", and the curve does the rest. Author it against its debut
  realm and the step multiplier lands on top of a number that already had the realm in it, so you
  get roughly double what you pictured.
- **Do not write the tier into the base twice.** A boss record legitimately has a bigger base than
  an outer one, because it is a bigger creature — but it is also two steps further along, so the
  gap you author is on top of the gap the curve already gives.

**The ratio between two motifs is invariant under the curve**, so varying bases across motifs is
safe and stays tunable: retuning `HPGrowth` moves everything by the same factor and never distorts
the gaps you authored.

`Actions` is authored and **never scaled**. Growing it would hand a high-realm creature more cards
rather than a harder version of its own.

## A picture is four layers

A record carries one picture **per element it can be dealt as**, so the roster is records times
affinities, and a brief written per picture would be the same paragraph typed hundreds of times.
It is assembled instead, in this order:

| layer | where | what it says |
|---|---|---|
| 1. style and composition | `docs/art/creature_art_prompt.MD` | canvas, framing, ground, light, rendering — true of every creature |
| 2. the motif | `Draw` on the file header | the body plan: what makes a goblin a goblin |
| 3. the element, for that motif | `ElementDraw` on the same header — or on the record, which replaces it | what fire does *to a goblin* |
| 4. the record | `Draw` on the record | this creature: what it is doing, what it carries |

**Layer 3 is per motif rather than global, and that is the decision worth knowing.** An ice slime
*is* ice, all the way through; an ice goblin is a goblin wearing frost. One elemental block written
for the whole roster would be right for one of them and wrong for the other.

`data.MotifData.Brief(record, element)` joins layers 2 to 4 and is the one place they are joined,
so a review sheet and a generated prompt cannot assemble them differently.
`go run ./tools/creatureprompt` is the command; `-gaps` says what is still unwritten.

**`goblins/motif.json` is the worked example.** Copy its shape rather than inventing one.

## The rooms

A motif's `backdrops.json` is a list of rooms, and a room is **one place drawn once per element**:

```json
{
  "Backdrop": "goblins-outer-tinker-studio",
  "Name": "Tinker Studio",
  "Tier": "outer",
  "Art": "goblins-outer-tinker-studio",
  "Affinities": ["fire", "ice", "lightning", "earth", "arcane"],
  "Draw": "A goblin tinker's studio hall, with nobody in it ...",
  "ElementDraw": { "fire": "The workshop runs hot ...", "ice": "Everything is frozen over ..." }
}
```

- **The key reads `<motif>-<tier>-<slug>`** and `Tier` is the creature's vocabulary — `boss` is the
  portal room. The prefix is checked.
- **The picture is `<Art>-<element>.jpg`**, one per affinity, under `assets/motifs/<motif>/backdrop/`,
  and **`Art` must read `<motif>-<tier>-<slug>` like the key** — the door is painted in, so the
  picture belongs to one tier; moving a room to another tier fails the launch until it is renamed.
  An art family may not be shared with a creature or another room: the map is flat.
- **`Draw` is the room and `ElementDraw` is what each element does to it**, and a room's brief is
  four layers the way a creature's is: the style and the tier's door live in
  `docs/art/background_art_prompt.MD`, these two on the record. **The door's size is never on the
  record** — small for outer, large for inner, two rainbow portals for the portal room, all three in
  the prompt. A record may say what its door *is* in this place (an arch of burning trees, a carved
  opening in a cave wall) and where the portals stand, as long as it keeps the tier's scale.
  An `ElementDraw` key the room does not take as an affinity is refused.
- **A portal room has two rainbow portals, always.** It is a requirement, not a style: a `boss`
  room's `Draw` that names a single door, one portal or none is wrong, and nothing in the loader
  can catch it. Give the room a centrepiece and say the two portals flank it — the goblins'
  throne, the plants' world tree.
- **A room drawn in one element needs no `ElementDraw`.** When the five elements are five different
  places rather than one place in five casts — `plants/backdrops.json` is the example — author five
  rooms of one affinity each and put everything in `Draw`. The motif report owes an `ElementDraw`
  only to a room with more than one affinity.
- **`MotifData.BackdropFor(tier, element, seed, realm)`** is the pick, a hash over the candidates in
  file order — derived, never rolled. **Nothing refuses a motif with no rooms**: a fight with none
  draws `default-background`, which has no door, so a gap is visible in play and counted on the
  report.

**The caveat every authored-and-ignored field carries applies, more gently than usual.** A brief
can go out of date and no test fails — but it describes a *picture*, so it can only ever disagree
with the art, never with the game. It is not the `CostTier` mistake.

## The coverage rule

> For every motif, for every one of the five elements, **at least two records** can field the
> outer chamber and **at least two** can field the inner. The portal room needs **one**.

`data.MinCoverageFor(tier)` is the figure, and `LoadMotifs` **panics** on a hole — the realm
generator will eventually present a choice and it must not be able to offer an impossible one.

**The boss is one because a boss is a name.** A chamber is a room the journey fills and wants a pool
to fill it from; a portal room is the creature a realm is remembered by, so it is authored for its
element. That is what lets a motif field five bosses of one element each — and three bosses at
four affinities is equally fine, and is what the rest of the roster does.

**Nothing is prescribed about how you satisfy the chambers.** Three records at four affinities
works; two at five works; four at two each works if they are chosen well. One clean pattern,
offered as a pattern rather than a rule: three records per tier, each taking four of the five
elements and each omitting a *different* one. That gives two elements three candidates and three
elements two, and there is nothing to check by hand.

`data.CoverageOf(m).Holes()` is the report. The sheet and this skill read the same function, so
there is one answer to "does this cover".

## The other refusals

Everything below is a panic at package init, in `data/motifs_data.go`:

- a `Motif` key that repeats, or that disagrees with its filename
- a `Record` key that repeats anywhere, or whose prefix disagrees with its motif and tier
- a `Tier` outside the three, an affinity that is not an element, an empty or repeating affinity list
- a title on a creature, a record with no art family, a non-positive stat
- a record with no cards, a card with no copies, a cost outside 1..3, a card naming its own elements
- an inverted realm band
- **the journey cannot be offered without a repeat** — `data.MustFillJourney`. Realm one is offered
  one motif and every realm above it `data.PortalOffers`, one behind each portal, and **every offer is
  spent** whichever the player takes, so the journey needs `data.JourneySlots(1, Realms)` distinct motifs
  each inside its own band. `data.FillsSlots` is the matching, and `internal/journey` asks it again
  after every draw so a seeded roll never spends a motif a later realm needed.

## What a record may never do

- **Carry an element on a card.** The colour is the realm's and the whole deck takes it.
- **Raise a shield.** Only the player does; every creature deck is pure attack. See `CLAUDE.md`.
- **Say what it is worth on the realm it debuts.** See above.

## The analyzer

Given a proposed record or a proposed motif, answer these in order.

1. **Which fights does it fill?** Its tier crossed with each of its affinities — that is
   `len(Affinities)` cells of the grid.
2. **What is still short?** Run the coverage grid for the motif with the record added and name
   every cell under two. Say which elements the remaining records have to cover, not just that
   there is a hole.
3. **Would the motif still load?** Walk the refusals above. The key format and the card costs are
   the two that get written wrong most often.
4. **Where do its bases sit?** Compare against the other records of its own tier in the same
   motif, and against the same tier in a motif of a similar band. A base is a step-zero quantity —
   if the proposal was reasoned from the realm it debuts on, say so and restate it as a multiple.
5. **Is its deck a shape the motif does not already have?** Three creatures in a tier that all
   deal the same copies at the same costs are one creature with three pictures. The three shapes
   in use are roughly swarm (many cheap), balanced, and heavy (few big); a fourth is welcome, a
   fourth copy of one of them is not.
6. **Does the realm band still fill the journey?** If the proposal is a whole motif, check that adding it
   does not narrow another realm's options, and that `MustFillJourney` still passes at
   `journey.json`'s `Realms` — which, with two offers a realm above the first, is a much tighter
   demand than one motif per realm. Retiring or narrowing a motif is where it bites.

Report holes and duplicates as a list, not prose, and say plainly which of them block a launch and
which are only worth knowing.

## What each motif is missing, and what to do next

**`go run ./tools/motifreport` is the worklist, and it is the answer to "what needs doing".** It
prints one line per motif to stdout and writes `docs/sheets/motifreport/index.html`, with every
gap listed per motif. Run it rather than reading the files, and quote what it prints — a count
written down anywhere else is stale by the next authoring session.

It scores four things, each off the loaded data and the files actually under `assets/motifs/`:

| Score | What counts |
|---|---|
| briefs | the motif's `Draw` and five `ElementDraw`, its portal `Text` and five `ElementText`, every record's `Draw`, every room's `Draw`, and a room's `ElementDraw` per affinity when it has more than one |
| creature art | one picture per record per affinity |
| rooms | one per fight — a tier in an element, fifteen to a motif — that some backdrop covers |
| room art | one per fight whose covering backdrop is painted |

**Coverage is not on it because it cannot be short**: the loader refuses a motif with a hole, so a
motif that launches has its creature grid filled. What can be short is everything a picture or a
screen is made from.

**Turning it into direction:** finish one motif before starting the next, so a realm is either
whole or visibly not. Within a motif, briefs come before art, because the art is generated from
them; rooms are three tiers, and the goblins are the worked example of all three. Name the motif
and the specific gaps, and let the owner choose which to take.

## Adding a motif

1. Write `data/motifs/<motif>/motif.json`. The directory name is the key.
2. `go test ./data/...` — the loader's refusals are the first thing to satisfy.
3. `go run ./tools/motifsheet` and look at the page, including the coverage grid on the heading.
4. Write the art direction: the header's `Draw`, the five `ElementDraw` blocks, and a `Draw` on
   each record. `go run ./tools/creatureprompt -gaps` says what is still missing.
5. Check the art keys: each record needs one picture per affinity, and until they exist every one
   of them draws the placeholder. Nothing fails; the report is where you see it.
6. Write the rooms in `backdrops.json`, one per tier at least, and run `go run ./tools/motifreport`
   — it says which of the fifteen fights still fall back to the default backdrop.

**Do not regenerate every sheet out of habit.** `go run ./tools/motifsheet` rewrites a strip per
record on its own; a full `go run ./tools/sheets` rewrites every binary under `docs/sheets/` and most of that
weight is this page.
