# Ring art prompts

Two halves. The **general block** is the same for every ring and is pasted verbatim.
The **specific block** is the one ring being drawn, and is the only part that is written
per ring.

## The generator decides the resolution, so nothing here may be written in pixels

*(measured 2026-09-11, across all seventeen rings generated so far)*

This block asked for a canvas of exactly 200 x 280 and for every detail to snap to a 5 x 5
pixel grid. **Both instructions were ignored, every time.** The generator returns
1060 x 1484 — its own fixed size, honouring the 5:7 portrait ratio and nothing else about
the pixel spec — and the returned images carry no block quantization at any period: edge
density measured by column position is flat, in the 1060 px original and in the 200 x 280
reduction alike. What comes back is pixel-art-*styled* illustration: chunky shapes, a dark
outline, flat-ish shading, and no grid.

**The art is good and the instruction was doing no work**, which is the finding. So every
figure in this block is now a **fraction of the canvas** rather than a count of pixels, and
the delivery size is stated as a separate fact about what happens to the image afterwards.
A reserved area written as `15 px` is a different fraction of the width than of the height
on a 5:7 canvas, and was being read against an image 5.3x larger than the one it described.

**If a genuinely blocky look is wanted, quantize rather than ask.** Downsample to 40 x 56
with nearest-neighbour and upscale back is ten lines and enforces what this block has only
ever requested. Nothing does that today, on purpose — the smooth version is what has been
committed and approved.

## General — identical for every ring

Pixel art illustration of a single ring. No text anywhere in the image — no title, no
label, no caption, no signature. The object and its background, nothing else.

Canvas: portrait, aspect ratio exactly 5:7 (five wide to seven tall). Render at whatever
resolution you work at natively; the image is reduced to 200 x 280 afterwards, so no detail
smaller than about a two-hundredth of the width will survive. Draw for that.

Chunky pixel blocks: build every visible detail out of square blocks roughly one fortieth
of the canvas width, so the whole image reads as about 40 blocks across by 56 down. Curves,
circles, flames, outlines, highlights and texture must all be constructed from these
discrete blocks, aligned to one consistent grid. Hard edges only — no anti-aliasing, no
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

- a margin around all four sides: 7.5% of the width on the left and right edges, and 5.5%
  of the height on the top and bottom edges
- the bottom-right corner: a box 22.5% of the width across and 16% of the height down

Background: opaque, one dark tone with subtle blocky texture. Never transparent, never
white. Background texture must remain substantially lower contrast than the ring.

Avoid environmental elements, scenery, props, characters, text of any kind, excessive
sparks, smoke or visual clutter.

Reference: 16-bit console RPG item art. It must stay legible after being reduced to
200 x 280 and viewed with no zoom, which is the size it is actually seen at.

Priority order: recognizable ring -> convincing effect -> strong silhouette -> dimensional
pixel-art shading -> subtle background texture.

## Specific — written per ring

### Burning Ring

The ring is burning: flames visibly originate from and cling to the ring's metal surface,
following portions of its circumference. The fire should feel integrated with the object
rather than appearing as a separate flame behind or around it. Use a small number of
chunky flame shapes with clear tongues and silhouettes. Include a few bright hottest
areas, but keep the fire readable at this tiny resolution.
