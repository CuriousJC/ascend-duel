# Guide art prompt

**One subject, two renders.** A faceless human-shaped figure made of luminous rainbow vapour.

It is used in two places that want two different canvases, so the artist is asked for two files of
the same figure rather than one file cropped twice:

- `guide.png` — **256x256, transparent**. It sits inside a speech bubble as a small round-ish
  portrait, drawn into a box with other things around it, so it needs empty space behind it.
- `default-enemy.png` — **200x280, with a ground**. It fills a whole card, edge to edge inside the
  border, with nothing else on the card behind it. A transparent one would leave a hole.

**Why vapour, and why every colour at once.** This figure is the stand-in — the guide who is not a
creature, and the face a creature wears before its own picture has been made. Everything else in
this set is one material in one hue; this is the one that is none of them, so it takes all of them
and settles on nothing. It has no face for the same reason: a face is a character, and this is a
placeholder that must not start reading as a specific someone.

---

## The colour words

The vapour runs through these five and only these five. Use them as the hues it shifts between; do
not introduce a sixth, and do not let any one of them dominate the figure.

| name | hex | reads as |
|---|---|---|
| orange | `#F06A24` | warm, the hottest note |
| yellow | `#F2D13D` | the brightest note |
| green | `#4E9B57` | cool-warm middle |
| blue | `#4E9ED6` | cool |
| purple | `#8A5BC7` | deepest, coolest |

Lighter and darker versions of each are fine and wanted — `#FFB24A`, `#FFF38A`, `#8BCB78`,
`#A8DDF4`, `#C8A4F2` on the light side; `#9E2F13`, `#A98200`, `#285E35`, `#245A86`, `#4C2A78` on
the dark side. What matters is that all five hues are present and that the transitions between
them are smooth, so it reads as one body of shifting colour rather than as five coloured patches.

---

## The prompt

High-resolution digital illustration of a **single human-shaped figure made entirely of glowing,
drifting vapour**, seen from the front, from roughly the chest up.

**The figure has no face and no features of any kind.** No eyes, no mouth, no nose, no hair, no
ears, no mask, no helmet, no expression. The head is a smooth featureless ovoid of vapour. Do not
suggest features through shading, through darker patches, or through the way the vapour parts — a
viewer must not be able to find a face in it however long they look.

**It is unmistakably a person in silhouette.** Head, neck, shoulders, the upper chest and the tops
of the arms are all clearly there and clearly human, in proportion, standing square to the viewer.
The shape is calm and upright, not crouched, not lunging, not dramatic. Read at a glance it should
say "a person", and only on a second look say "there is nothing solid there".

**It is made of vapour, not cloth and not fire.** Think coloured smoke or steam held in the shape
of a body: soft-edged, translucent, with visible internal drift and slow curling movement,
thickening toward the core of the body and thinning out at the edges. The shoulders and the crown
of the head hold their shape; the outermost edges break up into wisps and tendrils that fade to
nothing. There is no skin, no surface, no outline of a body underneath — the vapour *is* the body.

**Colour: all five hues at once, shifting smoothly through the figure.** The colours move through
the body in soft bands and currents rather than in blocks or stripes — one hue flowing into the
next the way oil moves on water, with no hard boundary anywhere between them. No single hue owns
the figure and no single hue owns the head. Keep the transitions gradual and the whole thing
luminous, as though the colour is lit from inside rather than lit from outside.

**Light: it glows from within**, brightest at the core of the chest and at the centre of the head,
falling off toward the edges as the vapour thins. On top of that inner glow there is a gentle
external light from the **upper left**, catching the upper-left shoulder and the upper-left of the
head a little brighter. Everything in this set is lit from that corner.

**Shading:** smooth and continuous, with real depth and volume. Anti-aliasing, soft gradients and
blended edges are all wanted. Do not posterize, do not dither, do not quantize to a block grid,
and **do not draw pixel art**.

**Avoid:** text of any kind, letters, numbers, symbols, runes, glyphs, sigils, magic circles.
No weapon, no staff, no orb, no jewellery, no clothing, no armour, no wings, no horns, no halo, no
crown. No lightning bolts, no sparks, no stars, no glitter, no lens flare. No second figure, no
hands raised, no gesture. No frame, no border, no vignette, no watermark, no signature. One figure
and nothing else.

---

### `guide.png` — 256x256, transparent

Canvas: **exactly 256x256**, transparent background, PNG with an alpha channel. Work as large as
you like natively and reduce once to 256 on the way out.

Framing: the figure is centred, head and shoulders, filling about **80% of the canvas width** and
about **85% of its height**, with a clear margin of empty space at the top and both sides. Crop it
at roughly the chest — this is a portrait, not a full body.

Background: **fully transparent everywhere the figure is not.** No ground, no panel, no circle
behind it, no cast shadow, no rectangle, no box. The outermost wisps of vapour should fade to full
transparency rather than stopping at a hard edge.

It is used at **about 90 pixels**, reduced from 256. Anything finer than a twentieth of the canvas
width is lost, so keep the wisps generous rather than hair-thin.

---

### `default-enemy.png` — 200x280, with a ground

Canvas: **exactly 200x280**, portrait orientation, fully opaque, PNG. Work as large as you like
natively and reduce once to 200x280 on the way out.

Framing: the figure is centred left to right and sits **slightly high** in the frame, filling about
**80% of the width** and about **70% of the height**. Crop it at roughly the chest, as above. Leave
the bottom third of the canvas comparatively empty — quiet ground with a little vapour drifting
down into it.

Ground: a **near-black background**, very slightly tinted so it is not a flat pure black — a deep
neutral charcoal with the faintest cool cast, carrying barely perceptible noise so it is not a dead
digital field. The whole canvas is covered: no transparency anywhere, and the ground runs right to
all four edges. The figure's outer wisps fade into that ground rather than stopping at an edge.

**Reserve nothing.** Do not leave a band, a bar, a corner or a disc clear or empty for anything to
be placed on later, and do not draw a plaque, a title bar, a frame or a panel anywhere. The picture
carries on underneath whatever is drawn on top of it.

---

## Before you hand them back

1. **Look for a face.** Turn each image upside down, squint at it, look at it small. If you can
   find eyes or a mouth anywhere in the vapour, the shading has invented one — flatten it out.
2. **Reduce `guide.png` to 90x90 and look at it.** Is it still obviously a person? Have the wisps
   averaged into mush?
3. **Reduce `default-enemy.png` to 100x140 and look at it.** Same question.
4. **Count the hues.** All five should be findable in each image, and none of them should be the
   one you notice first.
5. **Is `guide.png` genuinely transparent** behind the figure — not black, not white, not a dark
   square? **Is `default-enemy.png` genuinely opaque** to all four edges?
6. **Is there a hard line anywhere between two colours?** There should not be. If there is, the
   bands are too distinct.

Name them `guide.png` and `default-enemy.png` and give me links I can click to download them.
