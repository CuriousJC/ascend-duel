# Art prompts

**The words that produce a picture, kept beside each other rather than beside the picture.**
Everything here is an *input*: a prompt is pasted into an image generator and the PNG it
returns is filed under `assets/`. That makes this directory the opposite of its two
neighbors — `docs/sheets/` is generated output and `docs/analysis/` is a dated snapshot,
and both are things the repo produced. These are things the repo consumes.

## The rules

- **One prompt per catalog** *(owner's call, 2026-09-13)*. A prompt is the style block plus a
  composition plus a subject paragraph, and the style block is the whole reason a file exists: it
  is what makes forty relics look like forty relics from one game rather than forty separate
  commissions. It is pasted verbatim into every prompt, so the catalogs stay one game while each
  gets to say the things that are only true of it. A single file with a "pick one" section is a
  section somebody picks wrong.
- **A record never gets its own prompt.** The style and composition blocks are pasted verbatim; a
  record that needs one changed is a design decision that should move it for every card in that
  style.
- **The style block is measured off the shipped art, not written from taste** *(owner's call,
  2026-09-13)*. It says what the 138 pictures in `assets/relic/` actually are, so that a generator
  handed this prompt and a record lands somewhere the existing shelf would accept — which is the
  only test a prompt can be held to. **A clause nothing in `assets/` satisfies is a clause to
  delete, not a standard to keep.** The measurements are below.
- **The canvas is a fact about a card style**, taken from `internal/cards/style.go`: 200x280 at
  5:7, and the art *covers* a bleeding card rather than being fitted into a box. A prompt whose
  aspect ratio does not match gets cropped on the way in. Read the style before changing a number
  here — and read the art before changing a composition.
- **Commit the picture at the card's own 200x280, and keep the generator's output in
  `.scratch`.** The generator hands back a multiple — 1060x1484 for the relic batch — and that is
  the right thing to ask it for, because more pixels is a better source. It is the wrong thing to
  commit: 1.1 MB a card is about 155 MB across the catalog, and a reduction happening at draw
  time every frame is work done over and over for a picture that never changes. At the card's own
  size it is ~57 KB and nothing resamples at all.
  `TestEveryBleedingCardArtIsTheCardsOwnSize` is the tripwire.
- **`go run ./tools/relicart` is what does the filing**, so the reduction and the `"Art"` field are
  one command rather than three steps remembered in order. Drop the generator's PNG into
  `.scratch/to-process-relic-art/` named after the record and run it. Only relics have one
  today; a second catalog reaching this volume should get the same treatment rather than a
  second set of manual steps.
- **The generic prompt is what lives here; each relic's own description lives on its record**
  *(owner's call, 2026-09-12)*. A prompt is about no record at all. The subject paragraph is about exactly one, so it is `Draw` in
  `data/relics.json` — ignored by the engine, exactly as a status's `Badge` is, and pasted into
  the generator as the record's own JSON. A brief kept apart from the record is a brief that
  gets deleted when the picture it produced is filed.
- **The worklist is a query now.** A relic with an empty `Art` has no picture, a relic with an
  empty `Draw` has no brief, and `go run ./tools/relicsheet` prints both counts and marks both in
  pink. A file listing the same thing is a second copy to keep in step.
- **Nothing here is loaded by the game**, which is why it is not in `data/`. Everything in
  `data/` is the game's own catalog, `//go:embed`ed and read at launch.

## What the art actually is

**Measured across all 138 files in `assets/relic/`**, and the reason the style block reads the way
it does. The prompt that produced them asked for none of this — it demanded at most 16 flat
colors, chunky blocks on a 40x56 grid and hard edges with no anti-aliasing, and the generator
ignored every word of it. What came back is what the catalog now is, so the prompt was rewritten
to describe it.

| | |
|---|---|
| Source size | 1060 x 1484, exactly 5:7 |
| Committed size | 200 x 280 |
| Colors, source | 27,000 – 92,000 |
| Colors, committed | 4,700 – 12,900 |
| Block grid | **none** — median run of identical adjacent pixels is 1 pixel at both sizes |
| Edges | anti-aliased; smooth material gradients throughout |
| Subject width | ~80% of the canvas — median 20px clear left, 18px right |
| Subject height | ~70% — median 34px clear above, 54px below, so it sits a little high |
| Background | near-black, median luminance 28/255, low saturation, hue-tinted per card |
| Background flatness | no vignette: outer ring 28.9 against inner band 30.7 |
| Background texture | noise at ~1.6 luminance levels standard deviation — barely perceptible |

**Redo this before changing the style block again.** Color counts are `PIL.Image.getcolors`; the
grid is run lengths of identical adjacent pixels along sampled rows; the margins are the bounding
box of pixels more than 90 (summed channel distance) from the modal background color.

**`relicart -blocky` would have enforced the old spec** — it quantizes to the 40x56 grid on the
way down — and was never used. The committed catalog went through plain CatmullRom, which is
`reduce`'s default. That flag is now the odd one out rather than the road not taken.

## What is here

| File | Holds |
|---|---|
| `relic_art_prompt.MD` | the prompt for a relic — `cards.RelicStyle` |
| `relic_art_prompt_pixel_archived.MD` | **not live.** The same relic prompt with the render language pixel rather than smooth, kept so switching direction is a regeneration rather than an archaeology dig. The 137 pictures in `assets/relic/` were drawn this way and the other three catalogs were not |
| `essence_art_prompt.MD` | the prompt for an essence or a rune — `cards.EssenceStyle` goods that print their sentence on a scrim across the lower half of the picture. Same style block, and the ephemeral fading an essence has and a relic does not |
| `stone_art_prompt.MD` | the prompt for a stone — same style block and the same whole-object instruction, plus the three material ladders: the concept axis is silica, the form axis is plain rock, the element axis is gem |
| `other_card_art_prompt.MD` | the prompt for every *other* `cards.EssenceStyle` good — potions, the placeholder brand, the two sealed goods, and whatever is added next. Same style block and the same composition; the object is whole rather than coming apart |
| `rune_art_prompt.MD` | the closed list of rune body plans, pasted into the essence prompt. A creature needs a shape where an object does not |

**Neither bleeding card names itself** *(owner's call, 2026-09-11)*, so no prompt has to keep a
title band clear: the picture is the card. What a relic card draws over its art is one counter
disc in the bottom-right; what an `EssenceStyle` card draws over its art is the sentence saying
what it does, on a dark scrim from y 140 to y 265 of 280.

**Neither is a reservation** *(owner's call, 2026-09-13)*. Both go on top of a picture that
carries on underneath, so a prompt that described either would produce art with a hole in it —
and a generator told that a corner is special decorates it, which is how you get a drawn disc
sitting under the real one. **No prompt here asks for a title band, a disc, a badge or an empty
corner, and no prompt reserves anything**; both say so in as many words, at length, because it is
the instruction a generator is most inclined to helpfully ignore.

**Go and look at the art before changing a composition block.** The shipped relic pictures are a
centred subject at about 60% of the canvas on a dark textured ground that shows on all four
sides, with a chain or a haft free to run out to an edge — not full-bleed and not panelled, and
carrying no title and no counter anywhere. That is a fact about the pictures, and `style.go` does
not contain it: the style says how the card is *drawn*, not how the picture inside it is
*composed*. A dozen files in `assets/relic/` settle it in a minute.

The most an essence prompt adds is that the half carrying the recognition should be the upper
one, since the sentence lands on the lower. That is a placement hint rather than a band to leave
empty — a prompt that told the generator to draw nothing below the halfway line would produce a
card that is half picture and half wallpaper.
