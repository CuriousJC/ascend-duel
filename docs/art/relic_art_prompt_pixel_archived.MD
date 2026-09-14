# Relic art prompt — archived pixel version

**Not the live prompt. Do not paste this one unless the owner has said to.**
`relic_art_prompt.MD` beside it is what a new relic is drawn from, and it asks for the same
smooth render every other catalog now uses.

**This is here so the choice stays open.** The 137 pictures in `assets/relic/` were generated
from a prompt that asked for chunky blocks and sixteen flat colors, and they still read that way
against the essences, runes and stones, which were drawn smooth. Keeping a working pixel prompt
means switching direction is a regeneration rather than an archaeology dig.

**It is today's relic prompt with the render language swapped, not the file that drew the 137.**
That file was `card_art_prompt.MD` — generic across every card, and it reserved a margin on all
four sides plus a box in the bottom-right corner. Reserving was reversed on 2026-09-13 because
the card draws its counter on top of the picture and a prompt that describes the corner produces
either a hole or a second drawn disc under the real one. So the composition and subject blocks
below are the live ones, and this file could be pasted as it stands. The original is
`git show c480557:docs/art/card_art_prompt.MD` if the reservations are ever wanted back.

---

Pixel art illustration of a single object. No text anywhere in the image — no title, no label, no
caption, no signature, no watermark. The object and its background, nothing else.

Canvas: portrait, aspect ratio exactly 5:7 (five wide to seven tall). Render as large as you work
natively; around 1060 x 1484 is what this catalog was drawn at. It is reduced to 200 x 280
afterwards, so anything finer than about a two-hundredth of the width is lost. Draw for that:
bold, simple forms and few of them.

Pixel blocks: build every visible detail out of square blocks roughly one fortieth of the canvas
width, so the whole image reads as about 40 blocks across by 56 down. Curves, outlines,
highlights and texture must all be constructed from these discrete blocks, aligned to one
consistent grid. A feature smaller than one block cannot be drawn.

Edges: hard only. No anti-aliasing, no gradients, no soft shadows, no blur, and no dithering
across block edges. Every boundary is a step between two blocks.

Palette: at most 16 flat colors across the whole image. Shading is done by stepping between flat
tones, never by blending — a lit face is a lighter block, not a ramp. Dark near-black outlines
wherever shapes need separation.

Light: from the upper left — lighter blocks on the upper-left faces, darker blocks falling away
to the lower right.

Color: one dominant material hue plus one or two accents at most, so the image reads as a small
coherent scheme rather than a spread of colors.

Background: opaque, one dark tone with subtle blocky texture, tinted toward a hue that sets the
object off instead of competing with it. No vignette, no second light source, no scenery, no
horizon, no ground plane. The background texture stays substantially lower contrast than the
object. Never transparent, never white, never bright.

Avoid: scenery, environments, ground planes, hands, characters, props, borders, frames, panels,
particles, sparks, smoke, motion lines, and clutter of any kind.

Priority order: readable silhouette -> recognizable object -> dimensional pixel shading ->
coherent color scheme -> subtle background texture.

## Composition

Composition: one object, centered and large. It spans roughly 80% of the width and roughly 70% of
the height, sitting a little above center — about a tenth of the width left clear on each side,
and rather more clear below the object than above it.

A long object — a spear, a cleaver, a staff, a hammer — runs corner to corner on the diagonal so
that it can be drawn as large as possible. A small symmetrical one — a ring, an amulet, an orb, a
pendant — faces the viewer square on. Part of the object may run out toward an edge, a chain or a
haft or a trailing end, but it sits inside the canvas rather than bleeding off it.

**Nothing is reserved and nothing is kept clear.** There is no title, no name band, no caption
strip, no counter, no disc, no badge, no coin, no gem socket, no plaque, no banner and no slot of
any kind for text or a number. Do not leave a corner or an edge empty for one, and do not draw
one. Whatever the card puts on top of this picture, it puts there itself at draw time, and the
picture is meant to carry on underneath.

## The subject

The block pasted below this prompt is one record from `data/relics.json`, in JSON.

**Read `Draw` and nothing else as the subject.** A relic is an object and can be described from
nothing, so `Draw` is the whole brief. Every other field — `Name`, `Text`, `Rules`, `Rarity`,
`Family` and the rest — is context for what the relic does and is never to be rendered: no title,
no caption, no rules text, no numbers anywhere in the image. If `Draw` is empty, stop and say so
rather than inventing a subject.

Name the image after the record's own key — the `RelicRecord` value — and give me a link I can
click to download it.

## What this actually produced

Measured across the 137 pictures in `assets/relic/`, against the three catalogs drawn from the
smooth prompt. **The generator did not obey the sixteen-color clause** — a committed relic holds
thousands of colors, and the 1060x1484 source tens of thousands — but it obeyed the *direction*,
which is what separates the catalogs by eye.

| | committed colors, median | flat-pixel share | block-approximability at 5px |
|---|---|---|---|
| relic — this prompt | 9,225 | 45.3% | 5.75, the blockiest |
| rune — smooth prompt | 8,672 | 43.4% | 6.50 |
| essence — smooth prompt | 13,114 | 49.3% | 7.32 |
| stone — smooth prompt | 17,940 | 38.8% | 6.75 |

So a pixel prompt here buys a picture that reads blocky rather than one that is literally
quantized. **`go run ./tools/relicart -blocky` is the other half** if a true grid is wanted: it
quantizes to 40 blocks across on the way down to 200x280 and has never been run on a committed
picture.
