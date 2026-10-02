# Playing card art prompt

One picture per playing card per element: **a figure on a transparent background**. The game lays
it over the card's own face, which is sometimes plain off-white (#F0EFEA) and sometimes a rich
material — burnished gold, polished silver, glowing red copper, garnet enamel set in gold, dark blued
steel, a five-color watercolor wash. **The figure has to read clearly on every one of those**, which
is the whole brief.

---

## The canvas

**Portrait, aspect ratio exactly 5:7** — five wide to seven tall. Render as large as you work
natively; around 1060 x 1484 is right. It is reduced to **200 x 280** afterwards, so anything
finer than about a two-hundredth of the width is lost.

**The background is fully transparent** — a PNG with an alpha channel, where every pixel that is not
the figure or its weapon has an alpha of zero. **Not a checkerboard pattern drawn to look
transparent**, not white, not a pale wash. If you cannot output real transparency, fill the
background instead with **one flat, uniform pure magenta, #FF00FF**, a color that must appear nowhere
in the figure or the weapon, with no gradient, no texture and no shadow, so it can be removed
exactly.

The picture is laid over the card and clipped to the card's rounded corners, so a little is lost at
the edges. Nothing important should sit in the outermost twentieth.

## What the card draws on top

The card prints four small things over the picture, at these positions on the finished 200 x 280
card:

| What | x | y |
|---|---|---|
| a small mark, top-left corner | 13–44 | 10–44 |
| a short stack of small bars down the left edge | 11–28 | 60–100 |
| a round badge, bottom-left corner | 13–44 | 238–268 |
| the card's name, centred along the bottom | 60–140 | 240–268 |

**These are not reserved areas** — do not draw a band, a panel or a blank patch at any of them. They
are why the figure belongs in the open middle of the card, roughly **x 46 to 196, y 46 to 236**: a
tall field right of a narrow left-hand column and above a strip along the bottom.

## What every one of these pictures is

**A humanoid figure made of swirling colored vapor, holding one real solid weapon, caught in a
pose.** That is the whole catalog: 95 pictures of the same idea, and the three things that vary are
the three things a player has to read off the card at a glance.

- **The weapon says what kind of attack it is.** A thrusting weapon, an edged weapon or a blunt
  weapon — spears and blades and clubs — and the record's own brief names which.
- **The size of the weapon, and how committed the pose is, says how big the attack is.** The
  cheapest cards carry something barely longer than a hand and move almost not at all; the dearest
  carry something enormous and have thrown their whole body into it. That ladder is the one thing
  that must survive being seen small, so **draw the difference in scale hard** — a splinter against
  a lance, a wrist-flick against a full charge. Two cards a rung apart should not be mistakable.
- **The color of the vapor says the element.** It is the figure that is colored, not the weapon.

**The figure is vapor and the weapon is not.** The body has no skin, no face, no clothing — it is
smoke or mist in a human shape, denser at the core and fraying at the hands, the head and the
trailing edges. The weapon is the opposite: a solid, ordinary, sharply drawn physical object, held in
a hand of vapor that closes around the grip. **That contrast is the picture.** A figure drawn as a
person, or a weapon drawn as vapor, is the failure.

**No face.** The head is a suggestion in the vapor — a mass, a turn, a tilt — and never features.

### Where the figure goes

**The figure stands in the open middle of the card and fills it.** Head and weapon high, feet low,
using the full height of the field — about y 46 to y 236 — and the full width right of the left-hand
column. The weapon and the upper body go in the upper half; the legs and the base in the lower half.
**A weapon may run out toward an edge** — a lance, a spear or a greatsword should be as long as it
wants to be and may leave the frame, which is how the biggest rungs earn their scale.

## Drawing

**The figure carries a dark outline, and that is what lets it read on any ground.** Round the whole
silhouette of the vapor runs a **near-black contour**, heavy enough to hold at card size — about a
hundredth of the canvas width at its thickest. It is an ink line with some life in it, thickening on
the shadow side and thinning where the vapor frays, and it follows the vapor's own drifting edge —
wisps and curls included — rather than tracing a simplified body shape. **Inside the line the figure
is still vapor**: soft gradients, drifting density, no facets and no hard edges within it.

**The vapor is saturated and opaque enough to carry its own color** on gold, on silver, on dark
steel and on off-white alike. Denser and richest at the core, thinning toward the edges — but even
the thin parts stay clearly colored rather than fading to nothing, and **the vapor never becomes so
transparent that the ground behind it shows through more than faintly.**

**The weapon** is built out of clean, simplified, faceted geometry — flat planes meeting at crisp
angular breaks, the way a cut gem or a low-polygon model reads. It takes a **dark contour** round its
silhouette, the same weight as the figure's, and a clear material response: a long bright specular
ramp along a metal edge, matte where it is wood or leather. Spend the detail here, on the silhouette
and the large planes, never on fine ornament — a feature smaller than about a fortieth of the canvas
width will not survive the reduction.

Anti-aliasing, soft gradients and blended edges are expected and wanted, including the anti-aliased
edge of the figure against the transparency. Do not posterize, do not dither, do not quantize to a
block grid, and do not restrict the palette to a fixed number of colors.

Light: from the **upper left** — highlights on the upper-left faces, shadow falling away to the
lower right.

Avoid: any background at all, a ground plane, a cast shadow on the ground, a glow or halo spreading
off the figure into the background, recognizable characters, faces, scenery, props, borders, frames,
panels, particles floating free of the figure, sparks, motion lines, and clutter of any kind. **No
text anywhere** — no title, no label, no caption, no numeral, no signature, no watermark.

Priority order: **transparent background -> figure outlined and saturated enough to read on any
ground -> readable weapon and scale -> material shading -> coherent color.**

## The subject

The block pasted below this prompt is one record, in JSON.

**Read `Draw` and nothing else as the subject.** `Card` is which playing card it is and `Element`
is its color; both are context, and neither is to be rendered as a word. If `Draw` is empty, stop
and say so rather than inventing a subject.

**The element is the color the vapor is built in**, and the five are the game's own:

| Element | Word | dark | core | light |
|---|---|---|---|---|
| `fire` | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice` | blue | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth` | green | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane` | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

The element name and the color word are the same instruction — `arcane` and "purple" both mean that
row. **`core` is the body of the vapor**, `light` its brightest wisps and highlights, and `dark` its
shadowed side, just inside the contour. Stay inside the row: drifting to a neighbouring hue — fire
into red, lightning into orange, ice into teal, earth into olive, arcane into pink — is the failure.

**The same card in two colors is the same picture in two hues, not two pictures.** A fire Jab and
an ice Jab are one weapon, one pose, one composition, one framing — and two colors of vapor.

**What the element changes besides the hue is what the vapor is *like*** — flame curls upward, frost
mist settles, a charged haze crackles, loam hangs heavy, witch-light glows from inside. The record's
own brief says which. It never changes the weapon, the pose or the placement.

Name each image after its record's own key — the `CardArtRecord` value, like `jab-fire` — and
give me the file itself if there is only one, and a single zip file of everything if there is more
than one, so I can download it all at once — never a link per file.

---

## Before you hand anything back

1. **Is the background really transparent?** Open the PNG over a dark color and over a bright one.
   The ground should change with it; a checkerboard or a white box that stays put is a failure.
2. **Lay it over flat gold (#C9A227), flat dark slate (#1E2A38) and off-white (#F0EFEA).** Is the
   figure clearly visible on all three, its outline holding it on each, and its color still its
   element?
3. **Reduce it to 200 x 280 and look at it.** A subject that has gone mushy was too finely detailed;
   an outline that has vanished was too thin.
4. **Is there any text, numeral, frame or panel in the image?** There must be none.
5. **Does the figure fill the open middle?** It should use the full height of the field and the full
   width right of the left-hand column.
6. **Is the figure still vapor, and the weapon still solid?** An outline round the vapor is wanted;
   skin, a face or clothing is not. A translucent or wispy weapon has become vapor.
7. **Put this card beside the cheapest and dearest of its own kind.** The weapon should be obviously
   smaller or larger and the pose obviously less or more committed.
