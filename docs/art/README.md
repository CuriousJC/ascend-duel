# Art prompts

**The words that produce a picture, kept beside each other rather than beside the picture.**
Everything here is an *input*: a prompt is pasted into an image generator and the PNG it
returns is filed under `assets/`. That makes this directory the opposite of its two
neighbours — `docs/sheets/` is generated output and `docs/analysis/` is a dated snapshot,
and both are things the repo produced. These are things the repo consumes.

## The rules

- **One file per card style, not one per record.** A prompt is a general block plus a
  specific block, and the general block is the whole reason the file exists: it is what makes
  forty rings look like forty rings from one game rather than forty separate commissions.
- **The general block is pasted verbatim and is never edited for one record.** A record that
  needs the general block changed is a record that needs a new file, or a design decision that
  should move the general block for everything.
- **The canvas and the reserved areas are facts about a card style**, taken from
  `internal/cards/style.go`. They are the one part of a prompt that can go quietly wrong: the
  art is *fitted* into a box by `drawArt` — scaled to fit and centred, never cropped — so a
  prompt whose aspect ratio does not match its box letterboxes, and a prompt drawn at fewer
  pixels than the box loses its pixel grid on the way in. Read the style before changing a
  number here.
- **Commit the picture at the card's own 200x280, and keep the generator's output in
  `.scratch`.** The generator hands back a multiple — 1060x1484 for the ring batch — and that is
  the right thing to ask it for, because more pixels is a better source. It is the wrong thing to
  commit: a 5.3x reduction happening at draw time softens exactly the hard block edges the
  prompt spends most of its words demanding, and 1.1 MB a card is about 155 MB across a 137-ring
  catalogue. At the card's size it is ~57 KB and nothing resamples at all.
  `TestEveryBleedingCardArtIsTheCardsOwnSize` is the tripwire.
- **Nothing here is loaded by the game**, which is why it is not in `data/`. Everything in
  `data/` is the game's own catalogue, `//go:embed`ed and read at launch.

## What is here

| File | Card style | Draws |
|---|---|---|
| `ring_art_prompt.MD` | `cards.RingStyle` | the rings in `data/rings.json` |
| `relic_art_prompt.MD` | `cards.RingStyle` | relics, which reuse the ring card |
| `parasite_art_prompt.MD` | `cards.WormStyle` | the parasites in `data/parasites.json`, and the worms and stones that share the style |

**Neither bleeding card names itself** *(owner's call, 2026-09-11)*, so no prompt has to keep a
title band clear: the picture is the card. What a ring card draws over its art is one counter
disc in the bottom-right; what a worm card draws over its art is the sentence saying what it
does. Those are the only two reservations any of these files carries.
