# Shield art prompt

Six images: **one shield, drawn once and recolored six times** — five element colors and a neutral
gray.

**The shield is a game piece that moves.** It appears over a playing card, sits there, flies across
the table and lands on top of another card, covering most of it. It is drawn at about **140 pixels
wide and 175 tall** — two thirds of a playing card — and never larger.

**Several of them pile on one card.** Two, three, sometimes five shields are drawn on top of each
other, overlapping, each a little offset or turned from the one under it, and a player has to be
able to **count them at a glance**. That is the hardest thing this drawing has to do, and most of
the rules below exist for it: a shield whose edge is lost against an identical shield lying under it
turns a pile of three into one blob.

**Deliver at 256x256.** The drawn size is reached by reducing once from 256, from as much detail as
possible.

**Individual files, not a sheet.** One image per color, named as below. Do not assemble them into a
grid or a sprite sheet.

---

## The prompt

High-resolution digital illustration of a single **shield**, rendered as a **game piece**, not as a
picture of an object in a scene.

Canvas: **exactly 256x256**, transparent background, PNG with an alpha channel. Work as large as
you like natively and reduce once to 256 on the way out. It is drawn at about 140x175 pixels, so
**anything finer than about a twentieth of the canvas width is lost.** Draw for that: one bold
shape, a few large parts, no fine ornament.

### The object

**A heater shield, upright and straight on**: a flat top edge with softly rounded shoulders,
straight sides that taper in the lower half to a point at the bottom. Wide-shouldered — it should
read as broad and protective, not as a narrow kite.

- **It fills about 92% of the canvas height and about 75% of its width**, centred, with the point
  near the bottom edge and the top edge near the top. That is the target, not a ceiling: the image
  is placed in a fixed box and is **never scaled up to fill it**, so a shield drawn small in the
  canvas is drawn small everywhere.
- **Seen straight on.** No tilt, no perspective, no three-quarter view, no visible side thickness
  drawn as an edge-on rim. It will be rotated and offset when it is used; the drawing itself is
  square to the viewer.
- **A plain face**: a **broad raised rim** running all the way round the edge, and inside it one
  smooth, gently convex field, lit from the upper left and falling into shade toward the lower
  right, with nothing on it at all. No heraldry, no device, no emblem, no quarters,
  no charge, no rivets, no studs, no engraving, no filigree.
- **Nothing round on the face. No boss, no dome, no knob, no gem, no disc, no circle and no
  ring anywhere on the shield**, in the centre or anywhere else. This is the instruction most
  likely to be ignored, because a boss is what a shield usually has; this one has none.

**The rim is the most important part of the drawing.** It is what separates one shield from the
identical shield lying under it, so it has to be:

- **wide** — about a fourteenth of the canvas width all the way round, the same width on every side,
- **lighter than the field** — the rim is the shield's color at its lightest, the field a step
  deeper, so the outer band of each shield reads as a bright frame around a darker middle,
- **bordered on its outside by the heavy near-black contour** (see Drawing), so where one shield
  overlaps another there is always dark edge, light rim, darker field — three steps that cannot
  merge into the shield underneath.

**Fully opaque.** Not glass, not energy, not a translucent barrier, not glowing. A shield that lets
the one under it show through makes a pile uncountable. It is a solid object.

### The background, which is the instruction most often ignored

**Fully transparent everywhere the shield is not.** The alpha channel is empty right up to the
shield's own edge, and there is nothing else in the image at all.

In particular there is **no border and no frame**. Do not draw a rectangle around the picture, do
not outline the edges of the canvas, do not set the shield in a box, a panel, a plaque, a card, a
tile, a roundel or a crest, and do not leave a dark margin around the outside. **The only dark line
anywhere in the image is the contour that hugs the shield's own edge.**

Also none of: a ground, a ground plane, a cast shadow, a drop shadow, a glow behind the shield, a
halo, sparks, a vignette, a backing plate, a colored field. The shield is laid over playing cards —
an off-white card face, a dark painted card, a pale blue table — so anything at all behind it
becomes a visible rectangle sitting on top of that card.

### Drawing

Outline: a **heavy near-black contour** all the way around the shield — genuinely near-black, at
the bottom of the brightness range, and not merely a darker shade of the shield's own color. **At
least 8 pixels of the 256, and 10 is better.** It is what holds the shield against a pale card and
what keeps two overlapping shields apart. A contour drawn thin, or drawn as a dark version of the
hue, disappears on reduction.

Shading: smooth and continuous, with a clear material response — the shield should read as a
**solid, heavy, slightly convex object**, like enamelled metal. A broad gentle ramp from lighter at
the upper left to darker at the lower right across the field, the rim lit along its upper-left
stretch and shaded along its lower-right. Two or three broad tonal steps per part, not many stops. Anti-aliasing,
soft gradients and blended edges are all expected and wanted. **Do not draw pixel art**, do not
posterize, do not dither and do not quantize to a block grid.

Light: from the **upper left**. Everything in this set is lit from that corner and anything lit
from elsewhere reads as crooked.

Color: **one dominant hue, in its own darker and lighter values.** Shadow is that hue darkened,
light is that hue lightened, the contour is near-black. **Every lit part is the shield's own hue
made lighter — never a warm color, and never gold or silver.** No second hue anywhere: no metal
rim in a different color, no gold trim, no complementary rim light, no colored
bounce. The rim and the field are the same hue at different values.

Avoid: scenery, hands, characters, weapons, particles, sparks, smoke, flames, frost, lightning
bolts, leaves, runes, motion lines, bosses, domes, circles, gems, rivets, filigree, drop shadows,
outer glows, and clutter of any kind. **The element is said by the color alone** — a fire shield is an orange shield, not a
shield on fire. **No text anywhere** — no label, no letter, no signature, no watermark.

Priority order: **nothing on the face -> a pile of three overlapping copies still countable at 140 pixels wide -> the six
colors being one drawing -> a dark edge visible all the way round -> filling the frame ->
unmistakable hue -> everything else.**

### The six colors

Five hues, and the hue is the only thing that changes. **Match these exactly.**

| Name | Word | dark | core | light |
|---|---|---|---|---|
| `fire` | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice` | blue | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth` | green | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane` | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

**`core` is the hue to match**, and the element name and the color word are the same instruction —
`arcane` and "purple" both mean that row. Put the **field at about `core`**, the **rim between
`core` and `light`**, and the shaded faces toward `dark`,
above the near-black contour. **Stay inside the ramp**: anything between `dark` and `light` is in
range, anything outside it is a different color.

**And a sixth in neutral gray**: a mid-gray field, a lighter gray rim, a darker gray toward the
lower right, the same near-black contour. **No hue at all** — not a warm gray, not a cool gray, not
a tint of anything.

**The six must be the same drawing in six colors.** Identical silhouette, identical rim width,
identical face, identical light ramp — the only difference between `shield-fire.png` and
`shield-ice.png` is the hue. Laid side by side they should look like one picture printed six times
in six inks. **Draw the shield once, then recolor it six times.** Do not redraw per color.

### Naming

`shield-<color>.png`, where `<color>` is one of `fire`, `ice`, `lightning`, `earth`, `arcane`,
`neutral`.

Six files. Give me a single zip file of everything so I can download it all at once — never a link
per file.

---

## Before you hand anything back

Check every image against these. A failure on any one of them means redraw rather than explain.

0. **Make a pile and look at it.** Reduce the shield to 140 pixels wide, then lay three copies of
   it on top of each other, each shifted about a sixth of its width down and to the right of the
   last and turned a few degrees. **Can you count three without thinking?** If two of them merge
   into one shape, the rim is too narrow, too dark or too close to the field's color, or the
   contour is too thin — go back and widen and lighten the rim. Do the same with three copies of
   *different* colors. This check catches more than the others put together.
1. **Is there anything on the face?** A boss, a dome, a knob, a gem, a disc, a ring or a ridge
   anywhere on the shield is a redraw. The field inside the rim is plain.
2. **Lay the six colors side by side.** Same silhouette, same rim, same face, same size. If one
   reads fatter, that one is a redraw.
3. **Is the background actually transparent?** Not black, not white, not a dark square. Empty
   alpha, right up to the shield.
4. **Is there a frame?** There must be no rectangle, box, border, panel or dark margin anywhere.
   The only dark line is the contour on the shield's edge.
5. **Is it opaque?** Nothing behind the shield should show through any part of it.
6. **Is there a second hue, an emblem, or any text?** Every colored part should read as one color,
   the face should carry nothing but the rim and its own field, and nothing should be written anywhere.
