# Card art prompt

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

## The subject — pasted as the record

The block below is one record from the game's own catalogue, in JSON.

**Read `Draw` and nothing else as the subject.** `Name`, `Text` and `Rules` are context for
what the object does and are never to be rendered — no title, no caption, no rules text, no
numbers anywhere in the image. If `Draw` is empty, stop and say so rather than inventing a
subject.

All images should be named based on the provided RelicRecord and a link provided so I can easily click to download

```json
{
  "RelicRecord": "coiled-ring",
  "Name": "Coiled Ring",
  "Art": "",
  "Draw": "tightly braided jade metal relic",
  "Text": "Deals 5 more DMG for each crush card\nkept in hand.",
  "Rarity": "common"
}
```
