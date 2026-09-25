# Art prompts

**The words that produce a picture, kept beside each other rather than beside the picture.**
Everything here is an *input*: a prompt is pasted into an image generator and the PNG it
returns is filed under `assets/`. That makes this directory the opposite of its two
neighbors — `docs/sheets/` is generated output and `docs/analysis/` is a dated snapshot,
and both are things the repo produced. These are things the repo consumes.

## The rules

- **One prompt per catalog.** A prompt is the style block plus a composition plus a subject
  paragraph, and the style block is the whole reason a file exists: it is what makes a catalog
  look like one game rather than a pile of separate commissions. It is pasted verbatim into every prompt, so the catalogs stay one game while each
  gets to say the things that are only true of it. A single file with a "pick one" section is a
  section somebody picks wrong.
- **A record never gets its own prompt.** The style and composition blocks are pasted verbatim; a
  record that needs one changed is a design decision that should move it for every card in that
  style.
- **The style block is measured off the shipped art, not written from taste.** It says what the
  pictures in `assets/relic/` actually are, so that a generator handed this prompt and a record
  lands somewhere the existing shelf would accept — which is the only test a prompt can be held
  to. **A clause nothing in `assets/` satisfies is a clause to
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
  one command rather than three steps remembered in order. Drop the generator's PNG into the
  catalog's inbox named after the record, and run the command with that catalog's `-kind`:
  `relic`, `essence`, `rune`, `stone` or `other`. **A stem naming no record is refused rather than
  filed**, because a misspelled key is invisible in play — the card simply draws the fallback.
- **`-kind other` is one inbox over two files.** The potions and the sealed goods
  share one prompt, so they are generated in one batch and there is nothing to be gained by making
  the person splitting a batch of six decide which three are which. `.scratch/to-process-other-art`
  is the inbox, `assets/other` the directory, and the tool writes each `"Art"` back into whichever
  of `data/potions.json` and `data/goods.json` holds the record. An id appearing in both is
  refused: one flat asset map means one id is one picture.
- **The generic prompt is what lives here; each record's own description lives on the record.**
  A prompt is about no record at all. The subject paragraph is about exactly one, so it is `Draw` in
  `data/relics.json` — ignored by the engine, exactly as a status's `Badge` is, and pasted into
  the generator as the record's own JSON. A brief kept apart from the record is a brief that
  gets deleted when the picture it produced is filed.
- **The worklist is a query.** A record with an empty `Art` has no picture, a record with an
  empty `Draw` has no brief, and each catalog's sheet prints both counts and marks both in pink.
  A file listing the same thing is a second copy to keep in step.
- **The glyph prompt is the exception to every rule above, and says so at the top.** It has no
  record behind it — there is no `Draw` field for a form mark, because a form is a closed
  vocabulary in Go rather than a catalog anybody authors into — so the subject lives in the prompt
  itself, as a table of four objects. It is also the only prompt whose output is **not** filed by
  `tools/relicart`: the marks go into `assets/form/` under their own names and `assets.embedFamily`
  globs them, keyed by filename stem. `go run ./tools/marksheet` is the review page.
- **Nothing here is loaded by the game**, which is why it is not in `data/`. Everything in
  `data/` is the game's own catalog, `//go:embed`ed and read at launch.

## What the art actually is

**Measured across `assets/relic/`**, and the reason the style block reads the way it does. It is
not quantized pixel art and never was: a prompt demanding at most 16 flat colors, chunky blocks
on a 40x56 grid and hard edges with no anti-aliasing produced none of those things, and what the
catalog *is* is what the style block has to describe.

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

**Re-measure before changing the style block again.** Color counts are `PIL.Image.getcolors`;
the grid is run lengths of identical adjacent pixels along sampled rows; the margins are the
bounding box of pixels more than 90 (summed channel distance) from the modal background color.

**`relicart -blocky` enforces the quantized spec** — it snaps to the 40x56 grid on the way down —
and nothing in `assets/` was filed with it. The committed catalog went through plain CatmullRom,
which is `reduce`'s default.

## What is here

| File | Holds |
|---|---|
| `relic_art_prompt.MD` | the prompt for a relic — `cards.RelicStyle` |
| `relic_art_prompt_pixel_archived.MD` | **not live.** The same relic prompt with the render language pixel rather than smooth, kept so switching direction is a regeneration rather than an archaeology dig. The relic pictures were drawn this way and the other catalogs were not |
| `essence_art_prompt.MD` | the prompt for an essence or a rune — `cards.EssenceStyle` goods that print their sentence on a scrim across the lower half of the picture. Same style block, and the ephemeral fading an essence has and a relic does not |
| `stone_art_prompt.MD` | the prompt for a stone — same style block and the same whole-object instruction, plus the three material ladders: the concept axis is silica, the form axis is plain rock, the element axis is gem |
| `other_card_art_prompt.MD` | the prompt for every *other* `cards.EssenceStyle` good — the potions in `data/potions.json`, the sealed goods in `data/goods.json`, the placeholder brand, and whatever is added next. Same style block and the same composition; the object is whole rather than coming apart |
| `rune_art_prompt.MD` | the closed list of rune body plans, pasted into the essence prompt. A creature needs a shape where an object does not |
| `card_art_prompt.MD` | **the third odd one out**: a full-bleed picture for a *playing* card, one per card per element out of `data/card_art.json`, which is the largest backlog in the repo. It is 5:7 and full-bleed like the relic prompt, and unlike every other bleeding card it has **no scrim**: `cards.Hand` sets `ArtUnder` rather than `ArtBleed`, so the card keeps its near-black ink set and the picture has to stay pale where the type lands. The prompt carries the measured map of where that is. Filed into `assets/card/`, globbed by `assets.embedFamily` and keyed by filename stem, not through `tools/relicart`. **There is no default face** — an unauthored record draws the card exactly as it looked before the catalog existed |
| `damage_art_prompt.MD` | **the second odd one out**: the damage badge in a card's bottom-left corner — a badge carrying the multiplier an attack deals. Square and transparent like the glyph prompt, and the one prompt whose output has *type printed on it by the game*, which is why it asks for a flat unbroken centre and no hot spot. **It asks for three shapes on one diameter** — circle, starburst, diamond — in six colors each, because which outline the badge takes is still open. Filed like the form marks: into `assets/damage/` under its own names, globbed by `assets.embedFamily` and keyed by filename stem, not through `tools/relicart` |
| `glyph_art_prompt.MD` | **the odd one out**: it produces the marks drawn *on top of* a card rather than a picture *on* one — the form mark in the corner and the cost ticks under it. Square and 5:2 rather than 5:7, transparent rather than near-black, one hue rather than a scene, and delivered at 256 because the game reduces to the two sizes it draws at |
| `figure_art_prompt.MD` | the numerals and math symbols the combat screen assembles into the figures that fly over the table — terms, sums, hits, drains. **The only prompt that asks for a sprite sheet plus a JSON of glyph metrics** rather than one picture per file, because the game sets the glyphs side by side itself. Heavy black inked contour and a two-band flat fill, in the five element colors, attack red, relic pink and a pure-gray `neutral` the game multiplies by any other ink |
| `palette.json` | **not a prompt**: the eight color ramps this game draws in — the five elements and the three form materials — each as `dark`, `core` and `light`, beside the plain-English `hue` the ramp is (orange, yellow, green, blue, purple; cream, cool gray, dark gray), the `axis` it belongs to, and, for a material, the `form` it stands for (ivory is stab, steel is slash, granite is crush). `core` is the value the game uses; the other two are here for a generator being asked for a picture in one element's or one form's range. `internal/cards` holds the live copy of `core`, in `borderColors` and in `FormWords`, because a color the rules-adjacent drawing reads has to be a Go value and not a file read at launch. **Every prompt carries these values written out**, so a brief can say "a purple orb" or "an ivory haft" and the generator knows which range that is — the three closed-set prompts as their element table, the subject prompts as a shared **The color words** block. Move a value here and every prompt has to follow |

**Neither bleeding card names itself**, so no prompt has to keep a title band clear: the picture
is the card. What a relic card draws over its art is one counter disc in the bottom-right.
`EssenceStyle` declares a text band from y 140 to y 265 of 280 and **no card in the catalog fills
it** — the essences, runes, stones, potions and goods all say their rule in a tooltip — so an
`EssenceStyle` face is the whole picture.

**Neither is a reservation.** Both go on top of a picture that
carries on underneath, so a prompt that described either would produce art with a hole in it —
and a generator told that a corner is special decorates it, which is how you get a drawn disc
sitting under the real one. **No prompt here asks for a title band, a disc, a badge or an empty
corner, and no prompt but the creature one reserves anything**; they say so in as many words, at
length, because it is the instruction a generator is most inclined to helpfully ignore.

**The creature prompt is the one that constrains a strip.** An opponent card writes its badges,
its DMG and its health bar across the bottom 80 pixels, and a scrim covers that strip from y 216
down — so nothing the creature is recognized by may live below y 200, and the sixteen pixels the
badges sit on have to stay quiet. The paint still runs through all of it and the ground still
crosses those lines unchanged: what is withheld is anything that has to be seen, never the paint.
That scrim is derived from the figure's own offset, which is what keeps it off the duelist card —
that one has no portrait, writes no figure there, and stays off-white.

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
