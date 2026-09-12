# Art prompts

**The words that produce a picture, kept beside each other rather than beside the picture.**
Everything here is an *input*: a prompt is pasted into an image generator and the PNG it
returns is filed under `assets/`. That makes this directory the opposite of its two
neighbours — `docs/sheets/` is generated output and `docs/analysis/` is a dated snapshot,
and both are things the repo produced. These are things the repo consumes.

## The rules

- **One prompt for every card, not one per catalogue.** A prompt is the shared block plus a
  subject paragraph, and the shared block is the whole reason the file exists: it is what makes
  forty relics look like forty relics from one game rather than forty separate commissions. It is
  shared across the catalogues too — a relic, a relic and a parasite are the same card with a
  different picture in it, and three copies of one paragraph is three paragraphs drifting apart.
- **The shared block is pasted verbatim and is never edited for one record.** A record that
  needs it changed is a design decision that should move it for everything.
- **The canvas and the reserved areas are facts about a card style**, taken from
  `internal/cards/style.go`. They are the one part of a prompt that can go quietly wrong: the
  art is *fitted* into a box by `drawArt` — scaled to fit and centred, never cropped — so a
  prompt whose aspect ratio does not match its box letterboxes, and a prompt drawn at fewer
  pixels than the box loses its pixel grid on the way in. Read the style before changing a
  number here.
- **Commit the picture at the card's own 200x280, and keep the generator's output in
  `.scratch`.** The generator hands back a multiple — 1060x1484 for the relic batch — and that is
  the right thing to ask it for, because more pixels is a better source. It is the wrong thing to
  commit: a 5.3x reduction happening at draw time softens exactly the hard block edges the
  prompt spends most of its words demanding, and 1.1 MB a card is about 155 MB across a 137-ring
  catalogue. At the card's size it is ~57 KB and nothing resamples at all.
  `TestEveryBleedingCardArtIsTheCardsOwnSize` is the tripwire.
- **`go run ./tools/relicart` is what does the filing**, so the reduction and the `"Art"` field are
  one command rather than three steps remembered in order. Drop the generator's PNG into
  `.scratch/to-process-relic-art/` named after the record and run it. Only relics have one
  today; a second catalogue reaching this volume should get the same treatment rather than a
  second set of manual steps.
- **The generic prompt is what lives here; each relic's own description lives on its record**
  *(owner's call, 2026-09-12)*. This prompt is shared by every card that carries a picture and
  is about no record at all. The subject paragraph is about exactly one, so it is `Draw` in
  `data/relics.json` — ignored by the engine, exactly as a status's `Badge` is, and pasted into
  the generator as the record's own JSON. A brief kept apart from the record is a brief that
  gets deleted when the picture it produced is filed.
- **The worklist is a query now.** A relic with an empty `Art` has no picture, a relic with an
  empty `Draw` has no brief, and `go run ./tools/relicsheet` prints both counts and marks both in
  pink. A file listing the same thing is a second copy to keep in step.
- **Nothing here is loaded by the game**, which is why it is not in `data/`. Everything in
  `data/` is the game's own catalogue, `//go:embed`ed and read at launch.

## What is here

| File | Holds |
|---|---|
| `card_art_prompt.MD` | the prompt, for every card that carries a picture — relics on `cards.RelicStyle`, parasites and worms and stones on `cards.WormStyle`. Two composition blocks, one per style; pick one |
| `parasite_art_prompt.MD` | the closed list of parasite body plans, pasted into that prompt. A creature needs a shape where an object does not |

**Neither bleeding card names itself** *(owner's call, 2026-09-11)*, so no prompt has to keep a
title band clear: the picture is the card. What a relic card draws over its art is one counter
disc in the bottom-right; what a worm card draws over its art is the sentence saying what it
does. Those are the only two reservations any of these files carries.
