# Relic art prompts

The ring counterpart of this file is `ring_art_prompt.MD`, and the two are the same prompt
with one word changed. Two halves: the **general block** is the same for every relic and
is pasted verbatim, and the **specific block** is the one object being drawn, and is the
only part that is written per relic.

## General — identical for every relic

Pixel art illustration of a single object. No text anywhere in the image — no title, no
label, no caption, no signature. The object and its background, nothing else.

Canvas: exactly 200 x 280 pixels, portrait.

Pixel grid: every visible detail snaps to 5 x 5 pixel blocks, so the image reads as
40 x 56 chunky pixels. Curves, circles, flames, outlines, highlights and texture must all
be constructed from these discrete blocks. Hard edges only — no anti-aliasing, no
gradients, no soft shadows, no blur, no dithering across block edges.

Palette: at most 16 flat colours. Dark near-black outlines wherever shapes need
separation. Light source from the top-left.

The object is overwhelmingly the focal point. It must be immediately recognizable at a
glance and occupy most of the central composition, with a strong, readable silhouette and
a substantial dark outline. Use pixel-art highlights and shadows to give it enough
dimensionality to read as a solid object while keeping the geometry simple. Whatever
effect the object carries, the object itself must remain clearly visible through it — do
not bury it beneath flames, sparks, smoke, frost, energy or decorative effects.

Composition: subject centred, filling roughly the middle 60% of the canvas.

Keep these areas simple and low-contrast, and draw nothing in them — they are covered
after the fact by elements that are not part of this image, so nothing may be added to
represent them:

- the outer 15 px on all sides
- the bottom-right 45 x 45 px

Background: opaque, one dark tone with subtle blocky texture. Never transparent, never
white. Background texture must remain substantially lower contrast than the object.

Avoid environmental elements, scenery, props, characters, text of any kind, excessive
sparks, smoke or visual clutter.

Reference: 16-bit console RPG item art. Legible at 200 x 280 with no zoom.

Priority order: recognizable object -> convincing effect -> strong silhouette ->
dimensional pixel-art shading -> subtle background texture.

## Specific — written per relic

Say what the object *is* in the first sentence — the general block never names it — then
what it is doing. One paragraph.

### <name>

<what the object is, and the effect it carries>
