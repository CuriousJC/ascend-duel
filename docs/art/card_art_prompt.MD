# Card art prompt

**One prompt for every card that carries a picture** — rings, relics, parasites, worms and
stones. They are all the same card: a full-bleed 5:7 portrait behind a face that writes no
title, so a separate prompt per catalogue was three copies of one paragraph drifting apart.

## How to use it

1. Paste everything under **The prompt** below, verbatim, into the image generator.
2. Pick **one** of the two composition blocks and paste that with it. Which one is a fact
   about the card style, not a preference — see the table.
3. Replace the last line with the subject: what the object or creature *is*, and what the
   effect is doing to it, in one paragraph. The prompt never names the subject, so this is the
   only place the generator learns there is a ring on fire rather than a ring made of teeth.
   The per-record paragraphs live in the worklists — `rings_to_draw.md` for rings.
4. Save the result to `.scratch/<kind>-art/<key>.png`, reduce it to 200x280, and commit it as
   `assets/<kind>/<key>.png`. The reduction is not optional — see [README.md](README.md).
5. Set `"Art": "<key>"` on the record and strike it from the worklist.

| Card style | Draws | Composition | What covers the art |
|---|---|---|---|
| `cards.RingStyle` | rings, relics | **Whole card** | a counter disc in the bottom-right corner |
| `cards.WormStyle` | parasites, worms, stones | **Upper band** | the text band, from y 140 to y 265 |

**A parasite needs one thing more**: a body plan, from the closed list in
[parasite_art_prompt.MD](parasite_art_prompt.MD). A ring is an object and can be described from
nothing; a creature handed no shape is a different animal every time.

---

## The prompt

Pixel art illustration of a single subject. No text anywhere in the image — no title, no
label, no caption, no signature. The subject and its background, nothing else.

Canvas: portrait, aspect ratio exactly 5:7 (five wide to seven tall). Render at whatever
resolution you work at natively; the image is reduced to 200 x 280 afterwards, so no detail
smaller than about a two-hundredth of the width will survive. Draw for that.

Chunky pixel blocks: build every visible detail out of square blocks roughly one fortieth of
the canvas width, so the whole image reads as about 40 blocks across by 56 down. Curves,
outlines, highlights and texture must all be constructed from these discrete blocks, aligned
to one consistent grid. Hard edges only — no anti-aliasing, no gradients, no soft shadows, no
blur, no dithering across block edges.

Palette: at most 16 flat colours. Dark near-black outlines wherever shapes need separation.
Light source from the top-left.

The subject is overwhelmingly the focal point. It must be immediately recognizable at a glance
and occupy most of the central composition, with a strong, readable silhouette and a
substantial dark outline. Use pixel-art highlights and shadows to give it enough
dimensionality to read as a solid object while keeping the geometry simple. Whatever effect
the subject carries, the subject itself must remain clearly visible through it — do not bury
it beneath flames, sparks, smoke, frost, energy or decorative effects.

Background: opaque, one dark tone with subtle blocky texture. Never transparent, never white.
Background texture must remain substantially lower contrast than the subject.

Avoid environmental elements, scenery, props, characters, text of any kind, excessive sparks,
smoke or visual clutter.

Reference: 16-bit console RPG item art. It must stay legible after being reduced to 200 x 280
and viewed with no zoom, which is the size it is actually seen at.

Priority order: recognizable subject -> convincing effect -> strong silhouette -> dimensional
pixel-art shading -> subtle background texture.

## Composition — paste one

### Whole card

Composition: subject centred, filling roughly the middle 60% of the canvas.

Keep these areas simple and low-contrast, and draw nothing in them — they are covered after
the fact by elements that are not part of this image, so nothing may be added to represent
them:

- a margin around all four sides: 7.5% of the width on the left and right edges, and 5.5% of
  the height on the top and bottom edges
- the bottom-right corner: a box 22.5% of the width across and 16% of the height down

### Upper band

Composition: subject centred horizontally and sitting in the upper two thirds of the canvas,
filling roughly the middle 60% of the width and the band from 7% to 50% of the height down.

Keep these areas simple and low-contrast, and draw nothing in them — they are covered after
the fact by elements that are not part of this image, so nothing may be added to represent
them:

- a margin around all four sides: 7.5% of the width on the left and right edges, and 5.5% of
  the height on the top and bottom edges
- everything below 50% of the height

## The subject — written per record

The subject to draw: <what it is, and the effect it carries>
