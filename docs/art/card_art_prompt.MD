# Playing card art prompt

One picture per playing card per element — a full-bleed illustration that fills the whole card and
is drawn **underneath** everything the card already says.

**This is the hard one, and the reason is in the next paragraph.** Every other full-bleed picture
in this game sits under a dark band with light type on it. These do not. The card's type stays
near-black and sits directly on the picture, so **the picture has to stay pale where the type
lands** or the card becomes unreadable. That is the whole brief.

---

## The canvas

**Portrait, aspect ratio exactly 5:7** — five wide to seven tall. Render as large as you work
natively; around 1060 x 1484 is right. It is reduced to **200 x 280** afterwards, so anything
finer than about a two-hundredth of the width is lost.

The picture **covers the card**: it is scaled to fill and clipped to the card's rounded corners,
so a little is lost at the edges. Nothing important should sit in the outermost twentieth.

## Where the type lands

The card prints four things over this picture, all in near-black, and they are drawn at these
positions on the finished 200 x 280 card:

| What | x | y |
|---|---|---|
| a small mark, top-left corner | 13–44 | 10–44 |
| a short stack of small bars down the left edge | 11–28 | 60–100 |
| a round badge, bottom-left corner | 13–44 | 238–268 |
| the card's name, centred along the bottom | 60–140 | 240–268 |

**These are not reserved areas and you must not draw a band, a panel or a blank patch at any of
them.** The picture carries on underneath all four. What they are is a map of **where the picture
has to stay quiet**: pale, low-contrast, uncluttered. Dark, busy or high-contrast passages in those
regions will have black type sitting on them and neither will be readable.

**Read as a shape, the type occupies a left-hand column and a strip along the bottom.** The mark,
the bars and the badge are all in the leftmost quarter, running top to bottom; the name lies across
the bottom of the card beside the badge.

**Everything else is free, and that is most of the card.** The whole middle of the picture — from
under the mark at the top to just above the name at the bottom, across the full width right of the
left column — carries nothing at all. That is roughly **x 46 to 196, y 46 to 236**: a tall open
field, and by some margin the largest clear area on the card.

**Put the subject in it, and let it be the whole subject.** There is no band to work around and
nothing to duck under. Compose the picture to fill that field the way you would fill any tall
frame.

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
- **The color of the vapor says the element.** It is the figure that is colored, not the weapon and
  not the ground.

**The figure is vapor and the weapon is not.** The body has no skin, no face, no clothing and no
hard edge — it is smoke or mist in a human shape, denser at the core, fraying at the hands, the
head and the trailing edges, with the background faintly visible through its thinner parts. The
weapon is the opposite: a solid, ordinary, sharply drawn physical object, held in a hand of vapor
that closes around the grip. **That contrast is the picture.** A figure drawn as a person, or a
weapon drawn as vapor, is the failure.

**No face.** The head is a suggestion in the vapor — a mass, a turn, a tilt — and never features.
A face makes it a character, and these are 95 cards, not 95 characters.

### Where the figure goes

**The figure stands in the open middle of the card and fills it.** Head and weapon high, feet low,
using the full height of the clear field — about y 46 to y 236 — and the full width right of the
left-hand column.

- **The weapon and the upper body go in the upper half**, which is where the eye lands and the half
  that has to be recognizable at a glance.
- **The legs and the base of the figure go in the lower half.**
- **The body is continuous from top to bottom.** Nothing crosses it and nothing has to be read
  through it, so draw it at whatever density the figure wants throughout — dense at the core,
  fraying at the edges — rather than pinching it anywhere.

**A weapon may run out toward an edge.** A lance, a spear or a greatsword should be drawn as long
as it wants to be and allowed to leave the frame, which is how the biggest rungs earn their scale.

## Value — the clause that matters most

**The picture is light.** Not white, not washed out, but firmly in the upper half of the value
range, the way a watercolour or a pale fresco is. The card it sits on is off-white (#F0EFEA) and
the type on it is near-black, so a dark illustration turns a readable card into a black-on-black
one — and unlike every other card in this game, there is **no dark band under the type to rescue
it**.

- **Nothing in the four type regions above may be darker than about 60% brightness.** Keep the
  left-hand column and the bottom strip pale and low-contrast.
- **Everywhere else the picture may go as rich as it likes**, and should: a picture that is pale
  everywhere is a picture of nothing. The whole middle of the card is free, so that is where the
  contrast belongs.
- **No dark vignette, no dark corners, and no dark ground plane across the bottom** — the bottom
  strip carries the badge and the card's name.

## Drawing

**The weapon and the figure are drawn in two different languages, and that is deliberate.**

**The weapon** is built out of clean, simplified, faceted geometry — flat planes meeting at crisp
angular breaks, the way a cut gem or a low-polygon model reads. It takes a **dark contour** around
its silhouette, heavy enough to hold it against the picture at card size, and a clear material
response: a long bright specular ramp along a metal edge, matte where it is wood or leather. Spend
the detail here, on the silhouette and the large planes, never on fine ornament — a feature smaller
than about a fortieth of the canvas width will not survive the reduction. **It is the only thing in
the picture with a hard edge.**

**The figure** has no contour at all and no hard edge anywhere. It is volume without outline: soft
gradients, drifting density, an edge that dissolves into the ground rather than meeting it. Denser
and more saturated at the core, thinning at the head, the hands and the trailing edges until the
background shows through. **Do not draw the vapor as a shape with a line round it**, and do not
facet it — the contrast between a faceted solid weapon and a bodiless figure is the whole picture,
and outlining the figure collapses it.

Anti-aliasing, soft gradients and blended edges are all expected and wanted throughout. Do not
posterize, do not dither, do not quantize to a block grid, and do not restrict the palette to a
fixed number of colors.

**No contour anywhere in the type regions** — see the map above.

Light: from the **upper left** — highlights on the upper-left faces, shadow falling away to the
lower right.

Background: **pale and simple.** A flat wash, a soft gradient, or the barest suggestion of a
setting. It carries a very faint texture, barely perceptible, that never resolves into a pattern.
It is emphatically **not** the near-black ground the relic and essence cards use — those are
different cards with different rules, and a near-black ground here would fail the value clause on
its own.

Avoid: recognizable characters, faces, scenery with a horizon, ground planes, props, borders, frames,
panels, particles, sparks, smoke, motion lines, and clutter of any kind. **No text anywhere** — no
title, no label, no caption, no numeral, no signature, no watermark. The card prints its own name,
its own cost and its own figures on top, and a drawn one would sit underneath the real one.

**Nothing is reserved and nothing is kept clear.** There is no title band, no name strip, no
caption box, no badge, no disc, no plaque and no slot of any kind. Do not leave a corner or an
edge empty for one, and do not draw one. Everything the card puts on top of this picture is placed
there afterwards, and the picture is meant to carry on underneath.

Priority order: **pale where the type lands -> readable subject in the clear regions -> material
shading -> coherent color -> subtle background texture.**

## The subject

The block pasted below this prompt is one record, in JSON.

**Read `Draw` and nothing else as the subject.** `Card` is which playing card it is and `Element`
is its color; both are context, and neither is to be rendered as a word. If `Draw` is empty, stop
and say so rather than inventing a subject.

**The element is the color the picture is built in**, and the five are the game's own:

| Element | Hex |
|---|---|
| `fire` | `#F06A24` orange |
| `ice` | `#4E9ED6` blue |
| `lightning` | `#F2D13D` yellow |
| `earth` | `#4E9B57` green |
| `arcane` | `#8A5BC7` violet |

**Lightened, because of the value clause.** These are the hues the rest of the card is drawn in, so
the picture has to agree with them — but a picture at full saturation in one of them would be too
dark under type. Treat each as the hue the picture is *tinted toward*, at the pale end of its own
range, with the saturated version reserved for the subject in the clear regions.

**The same card in two colors is the same picture in two hues, not two pictures.** A fire Jab and
an ice Jab are one weapon, one pose, one composition, one framing — and two colors of vapor. Draw
each card's figure once and recolor it five times.

**What the element changes besides the hue is what the vapor is *like*** — flame curls upward, frost
mist settles, a charged haze crackles, loam hangs heavy, witch-light glows from inside. The record's
own brief says which. It never changes the weapon, the pose or the placement.

Name the image after the record's own key — the `CardArtRecord` value, like `jab-fire` — and give
me a link I can click to download it.

---

## Before you hand anything back

1. **Overlay the type map.** Put a black rectangle at each of the four regions in the table above.
   Can you still see the picture, and could black text be read at every one of them? If any region
   has something dark or busy under it, move the subject rather than darkening the type.
2. **Reduce it to 200 x 280 and look at it.** That is the size it is used at. A subject that has
   gone mushy was too finely detailed.
3. **Is it light enough?** Squint at it. The overall impression should be pale. If it reads as a
   dark picture, it is wrong for this card whatever else is right about it.
4. **Is there a frame, a band or a panel?** There must be none — no border, no title strip, no
   reserved corner, no drawn badge.
5. **Is there any text or numeral in the image?** There must be none at all.
6. **Does the subject fill the open middle?** It should use the full height of the clear field and
   the full width right of the left-hand column. A figure that is small, or pinched anywhere across
   its middle, is leaving the best part of the card empty.
7. **Is the figure still vapor, and the weapon still solid?** If the figure has an outline, a face,
   skin or clothing it has become a person. If the weapon is translucent or wispy it has become
   vapor. Either one is a redraw.
8. **Put this card beside the cheapest and dearest of its own kind.** The weapon should be
   obviously smaller or larger and the pose obviously less or more committed. If two rungs read as
   the same picture, the scale ladder has not been drawn hard enough.
