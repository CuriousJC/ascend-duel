# Ring art prompts

Two halves. The **general block** is the same for every ring and is pasted verbatim.
The **specific block** is the one ring being drawn, and is the only part that is written
per ring.

## General — identical for every ring

Pixel art illustration of a single ring. No text anywhere in the image — no title, no
label, no caption, no signature. The object and its background, nothing else.

Canvas: exactly 200 x 280 pixels, portrait.

Pixel grid: every visible detail snaps to 5 x 5 pixel blocks, so the image reads as
40 x 56 chunky pixels. Curves, circles, flames, outlines, highlights and texture must all
be constructed from these discrete blocks. Hard edges only — no anti-aliasing, no
gradients, no soft shadows, no blur, no dithering across block edges.

Palette: at most 16 flat colours. Dark near-black outlines wherever shapes need
separation. Light source from the top-left.

The ring is overwhelmingly the focal point. It must be immediately recognizable as a ring
at a glance and occupy most of the central composition, with a strong, readable silhouette
and a substantial dark outline. Use pixel-art highlights and shadows to give it enough
dimensionality to read as a solid object while keeping the geometry simple. Whatever
effect the ring carries, the ring itself must remain clearly visible through it — do not
bury it beneath flames, sparks, smoke, frost, energy or decorative effects.

Composition: subject centred, filling roughly the middle 60% of the canvas.

Keep these areas simple and low-contrast, and draw nothing in them — they are covered
after the fact by elements that are not part of this image, so nothing may be added to
represent them:

- the outer 15 px on all sides
- the bottom-right 45 x 45 px

Background: opaque, one dark tone with subtle blocky texture. Never transparent, never
white. Background texture must remain substantially lower contrast than the ring.

Avoid environmental elements, scenery, props, characters, text of any kind, excessive
sparks, smoke or visual clutter.

Reference: 16-bit console RPG item art. Legible at 200 x 280 with no zoom.

Priority order: recognizable ring -> convincing effect -> strong silhouette -> dimensional
pixel-art shading -> subtle background texture.

## Specific — written per ring

### Burning Ring

The ring is burning: flames visibly originate from and cling to the ring's metal surface,
following portions of its circumference. The fire should feel integrated with the object
rather than appearing as a separate flame behind or around it. Use a small number of
chunky flame shapes with clear tongues and silhouettes. Include a few bright hottest
areas, but keep the fire readable at this tiny resolution.
