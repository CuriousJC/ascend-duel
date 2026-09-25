---
name: motifs
description: The roster grammar - how a creature is written as data under data/motifs, the closed vocabularies a record draws on, what a floor is, how the ascent curve reads a base stat line, and what the loader refuses. Load before adding a motif file, adding or changing a record, authoring creatures or bosses, touching data/tower.json, or wiring anything that picks an opponent. Also the motif analyzer: given a proposed record, which tier/element fights it fills, which are still short, whether the motif would still pass, and whether its bases sit where its neighbours do.
---

# Motifs

**A floor is a motif and an element.** The tower picks one whole motif and one of the five
elements, and the floor's three rooms — outer chamber, inner chamber, stairway — are three records
of that motif dealt as that element. So a fire goblin floor is three goblins in fire, and what the
player walked into is something they can plan against.

That is the whole reason the roster is a directory rather than a file. The question asked of the
catalog is never "is this creature good"; it is "can this motif fill a floor at every element",
and a motif split across two files is a question neither half can answer.

## Where everything is

| Thing | File |
|---|---|
| The roster | `data/motifs/<motif>.json`, one file per motif |
| The structs, the loader, every refusal | `data/motifs_data.go` |
| The tower's height and its two growth rates | `data/tower.json`, `data/tower_data.go` |
| The climb: which motif and element each floor takes | `internal/pyramid` |
| A record's cards becoming a deck | `internal/decks/enemy.go` |
| A record becoming a fighter | `internal/entities/combatant.go` |
| The review page | `go run ./tools/motifsheet` |
| The art brief for one picture | `go run ./tools/creatureprompt` |
| The style block every creature shares | `docs/art/creature_art_prompt.MD` |

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
  "ValidFloors": [1, 6],
  "Records": [ ... ]
}
```

- **`Motif` is the key, and the filename has to match it.** It is also the art family and the name
  of the generator prompt this motif's pictures come from.
- **`Draw` and `ElementDraw` are the shared half of the art brief** — see *A picture is four
  layers* below. Both are authored and ignored by everything that plays the game. An
  `ElementDraw` key that is not an element is refused; a missing one is not.
- **`ValidFloors` is motif-level**, inclusive, `[0, 0]` for any floor. It is not per-record,
  because a floor takes a whole motif: a motif whose outer creatures were valid on floors 1 to 3
  and whose boss was valid on 4 to 6 could never theme a floor at all.

## The record

```json
{
  "Record": "goblins-outer-bomber",
  "Name": "Goblin Bomber",
  "Tier": "outer",
  "Art": "goblins-bomber",
  "Draw": "TBD",
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
  and the report is what the floor generator believes.
- **`Tier` is closed**: `outer`, `inner`, `boss`. Its index is the last term of the ascent step.
- **`Title` is boss-only** and is refused on a creature.
- **`Art` is the picture family stem.** The face drawn is `<Art>-<element>.png`, so one record
  carries one picture per element it can be dealt as. A missing file falls back to
  `default-enemy`, so a blank face means art nobody has made rather than a name nobody spelled
  right.
- **`Affinities` is a non-empty subset of the five elements**, no repeats. `basic` is not one of
  them: a creature takes its floor's colour, and a floor has one.
- **`Cards` is authored per record.** Two creatures of one motif are two different fights, so they
  hold different cards rather than the same cards at different weights. **A card may not name its
  own elements** — the colour is the floor's — and costs run 1 to 3, because the cost column is
  tick marks stacked down a fixed band.

## The bases are step-zero quantities

`HP` and `DMG` say what a creature is worth **in the very first room of the tower**, whatever floor
it is actually met on. `pyramid.ScaleToFight` puts it where it stands:

```
step = (floor - 1) * FightsPerFloor + tierIndex      // outer 0, inner 1, boss 2
HP   = base.HP  grown at tower.HPGrowth,  once per step
DMG  = base.DMG grown at tower.DMGGrowth, once per step
```

**Stepping per fight rather than per floor** is what makes a floor's boss harder than its own
inner chamber and the next floor's outer chamber harder than that boss, with no constraint between
two separate numbers to get wrong.

So the two mistakes to watch for when authoring:

- **Do not write a late-band creature as a high stat line.** A dragon banded to floors 5–8 is not
  "250 HP". It is "about 2.5x a goblin", and the curve does the rest. Author it against its debut
  floor and the step multiplier lands on top of a number that already had the floor in it, so you
  get roughly double what you pictured.
- **Do not write the tier into the base twice.** A boss record legitimately has a bigger base than
  an outer one, because it is a bigger creature — but it is also two steps further along, so the
  gap you author is on top of the gap the curve already gives.

**The ratio between two motifs is invariant under the curve**, so varying bases across motifs is
safe and stays tunable: retuning `HPGrowth` moves everything by the same factor and never distorts
the gaps you authored.

`Actions` is authored and **never scaled**. Growing it would hand a high-floor creature more cards
rather than a harder version of its own.

## A picture is four layers

A record carries one picture **per element it can be dealt as**, so the roster is records times
affinities, and a brief written per picture would be the same paragraph typed hundreds of times.
It is assembled instead, in this order:

| layer | where | what it says |
|---|---|---|
| 1. style and composition | `docs/art/creature_art_prompt.MD` | canvas, framing, ground, light, rendering — true of every creature |
| 2. the motif | `Draw` on the file header | the body plan: what makes a goblin a goblin |
| 3. the element, for that motif | `ElementDraw` on the same header | what fire does *to a goblin* |
| 4. the record | `Draw` on the record | this creature: what it is doing, what it carries |

**Layer 3 is per motif rather than global, and that is the decision worth knowing.** An ice slime
*is* ice, all the way through; an ice goblin is a goblin wearing frost. One elemental block written
for the whole roster would be right for one of them and wrong for the other.

`data.MotifData.Brief(record, element)` joins layers 2 to 4 and is the one place they are joined,
so a review sheet and a generated prompt cannot assemble them differently.
`go run ./tools/creatureprompt` is the command; `-gaps` says what is still unwritten.

**`goblins.json` is the worked example.** Copy its shape rather than inventing one.

**The caveat every authored-and-ignored field carries applies, more gently than usual.** A brief
can go out of date and no test fails — but it describes a *picture*, so it can only ever disagree
with the art, never with the game. It is not the `CostTier` mistake.

## The coverage rule

> For every motif, for every one of the five elements, **at least two records** can field the
> outer chamber and **at least two** can field the inner. The stairway needs **one**.

`data.MinCoverageFor(tier)` is the figure, and `LoadMotifs` **panics** on a hole — the floor
generator will eventually present a choice and it must not be able to offer an impossible one.

**The boss is one because a boss is a name.** A chamber is a room the climb fills and wants a pool
to fill it from; a stairway is the creature a floor is remembered by, so it is authored for its
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
- an inverted floor band
- **floors 1..N cannot each be given a *distinct* motif** — `data.MustBeClimbable`, a matching
  check rather than a per-floor one. Three motifs that all say `[1, 2]` satisfy "floor 1 has a
  candidate" and "floor 2 has a candidate" while still leaving floor 3 empty, and a run never
  repeats a motif.

## What a record may never do

- **Carry an element on a card.** The colour is the floor's and the whole deck takes it.
- **Raise a shield.** Only the player does; every creature deck is pure attack. See `CLAUDE.md`.
- **Say what it is worth on the floor it debuts.** See above.
- **Name a picture that exists.** Most do not yet, and the fallback is deliberate.

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
   if the proposal was reasoned from the floor it debuts on, say so and restate it as a multiple.
5. **Is its deck a shape the motif does not already have?** Three creatures in a tier that all
   deal the same copies at the same costs are one creature with three pictures. The three shapes
   in use are roughly swarm (many cheap), balanced, and heavy (few big); a fourth is welcome, a
   fourth copy of one of them is not.
6. **Does the floor band still climb?** If the proposal is a whole motif, check that adding it
   does not narrow another floor's options, and that `MustBeClimbable` still passes at
   `tower.json`'s `Floors`.

Report holes and duplicates as a list, not prose, and say plainly which of them block a launch and
which are only worth knowing.

## Adding a motif

1. Write `data/motifs/<motif>.json`. The filename is the key.
2. `go test ./data/...` — the loader's refusals are the first thing to satisfy.
3. `go run ./tools/motifsheet` and look at the page, including the coverage grid on the heading.
4. Write the art direction: the header's `Draw`, the five `ElementDraw` blocks, and a `Draw` on
   each record. `go run ./tools/creatureprompt -gaps` says what is still missing.
5. Check the art keys: each record needs one picture per affinity, and until they exist every one
   of them draws the placeholder. Nothing fails; the sheet is where you see it.

**Do not regenerate every sheet out of habit.** `go run ./tools/motifsheet` rewrites 153 strips on
its own; a full `go run ./tools/sheets` rewrites every binary under `docs/sheets/` and most of that
weight is this page.
