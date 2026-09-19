# Glyph art prompt

Thirty images: four small marks in five colors, the same four in neutral gray, five short bars,
and a sixth bar in neutral gray. They are stamped onto a pale off-white surface (#F0EFEA), they
are tiny, and they must be transparent outside their own silhouette.

**The thing that governs everything below is size.** A mark is used at **32 pixels** and at **16
pixels**. A bar is used at **20x8** and at **10x4**. There is no zoom and no larger view anywhere,
so detail that does not survive to 32 pixels is detail that does not exist.

**Deliver at 256x256 for the marks and 640x256 for the bars.** Not 64, and not 32 either. The
sizes they are used at are reached by reducing from those canvases — 256 to 32 is eight, 256 to 16
is sixteen, and both are clean. A file handed over at 32 is one of those sizes already and a bad
source for the other, and one handed over at 64 is an image reduced once being reduced again, with
its outline averaged out of existence on the way. One reduction, from as much detail as possible,
is the whole rule.

**Generate the whole set in one pass.** The marks are read as a set — four of them are compared
side by side at a glance — so drawing half now and half in a fortnight is how the two halves come
back subtly unlike each other. A partial run was tried once and the marks it produced had to be
redrawn anyway.

**A bar is one drawing, repeated.** One, two or three copies of the same picture are stacked in a
column, so there is one bar per color and no variants, no sequence and no sprite sheet — see the
second half of this file.

---

## Prompt one — the four marks

High-resolution digital illustration of a single weapon or shield, rendered as a **game interface
icon**, not as a picture of an object in a scene.

Canvas: **exactly 256x256**, transparent background, PNG with an alpha channel. Work as large as
you like natively and reduce once to 256 on the way out; do not hand back 64 or 32. It is used at
32 and 16 pixels, so **anything finer than about an eighth of the canvas width is lost.** Draw for
that: one bold shape, few parts, no ornament.

### The background, which is the instruction most often ignored

**Fully transparent everywhere the object is not.** The alpha channel is empty right up to the
object's own silhouette, and there is nothing else in the image at all.

In particular there is **no border and no frame**. Do not draw a rectangle around the picture, do
not outline the edges of the canvas, do not set the object in a box, a panel, a plaque, a card, a
tile, a roundel, a badge or a crest, and do not leave a dark margin around the outside. **The only
dark line anywhere in the image is the contour that hugs the object's own edge** — it follows the
silhouette and appears nowhere else. If a second outline is going in further out, that is the
mistake.

Also none of: a ground, a ground plane, a cast shadow, a drop shadow, a glow behind the object, a
vignette, a backing disc, a colored field. The mark is stamped directly onto a pale surface, so
anything at all behind it becomes a visible rectangle sitting on that surface — and a dark frame
around the edge is the most conspicuous version of that failure.

### Framing

**How much of the square an object takes is part of what tells it from the others**, so the two
groups are framed differently and the figures below are not suggestions.

**The two long weapons run corner to corner on the diagonal at about 95% of the canvas**, so that
each can be drawn as large as possible, and **they lean opposite ways**: the spear's point goes to
the **upper left**, the scimitar's to the **upper right**. That is the one thing telling those two
apart at a glance before any interior detail is legible.

**The club runs the full height of the canvas, tilted about 15 degrees** — near-vertical, leaning
its head to the upper right, top and bottom very nearly touching the edges of the square. It is
the heaviest object in the set and it has been coming back as the smallest, so treat 100% of the
height as the target and not as a ceiling.

**The shield stands upright and straight on, filling about 75% of the canvas** — centred, with
roughly an eighth of the canvas clear on each side and the same above and below. It is the one
squat object, so it reads as compact against the three weapons — but 75% means three quarters of
the frame, not a small object floating in a large one. Do not tilt it. **The space around it is
transparent, not a margin** — see above.

**The width matters as much as the height, and this is what has gone wrong.** The mark is placed
in a square box and is **never scaled up to fill it**, so a tall narrow object is drawn tall and
narrow — at size the club has been landing ten pixels wide in a thirty-two pixel box, with under
half the ink of the shield beside it, and reading as the quietest of the four. **Every object in
this set should put a comparable amount of ink down.** A long weapon earns it by running the
diagonal; the shield earns it by being broad; the club has to earn it by being *thick*.

### Drawing

Form: bold, simplified, faceted geometry — flat planes meeting at crisp angular breaks, the way a
cut gem or a low-polygon model reads. Spend every bit of the detail budget on the **silhouette**.
The four marks have to be told apart at 16 pixels by proportion alone, before a single interior
feature is legible, so exaggerate what makes each one itself: the spear narrow and needle-tipped
and leaning left, the scimitar a deep curve leaning right, the club a thick knotted bough barely
off the vertical, the shield wide-shouldered and tapering to a point.

**Nothing in the silhouette may be thinner than a twelfth of the canvas width.** A thin haft, a
narrow crossguard, a slender finial or a decorative spike disappears entirely at size — or worse,
survives as one gray pixel that reads as a scratch. Make hafts thick, guards chunky, and delete
anything the object does not need in order to be recognized.

Outline: a **heavy near-black contour** all the way around the object — genuinely near-black, at
the bottom of the brightness range, and not merely a darker shade of the object's own color.

**Make it thick: at least 8 pixels of the 256, and 10 is better.** This is the clause that has
failed silently every time. The mark is reduced to 32 and then to 16, and each reduction averages
the outline into whatever is beside it — so a contour drawn one or two pixels thin, or drawn as a
dark version of the hue rather than as near-black, simply disappears, and what lands is a soft
shape with nothing holding it in. **The last batch arrived with no contour left at all**: at 32
pixels not one pixel of it was dark.

The test to apply: reduce your own image to 32x32 and look at it. **A dark ring should still be
visible all the way round.** If the edge has gone soft, the contour was too thin or too light.

It follows the object's edge and appears nowhere else.

Shading: smooth and continuous within each plane, with a clear material response — a long bright
specular ramp up the lit faces and a small near-white hot spot where the light catches hardest.
Two or three broad tonal steps across the whole object, not a gradient with many stops. **It
should read as a solid object with real weight and material, not as a flat vector icon** — the
shading is what makes a spear look forged rather than printed. Anti-aliasing, soft gradients and
blended edges are all expected and wanted. Do not posterize, do not dither and do not quantize to
a block grid. **Do not draw pixel art** — everything around it is smooth, and a pixelated mark
would be the one jagged thing on the screen.

Light: from the **upper left** — highlights on the upper-left faces, shadow falling away to the
lower right. Everything in this set is lit from that corner and anything lit from elsewhere reads
as crooked.

Color: **one dominant hue, in its own darker and lighter values.** Shadow is that hue darkened,
light is that hue lightened, the contour is near-black, and the hot spot is near-white. Shade it
as richly as the material wants inside that — this is a limit on the *hue*, not on the number of
tones.

**Every lit surface is the object's own hue made lighter — never a warm color, and never gold.**
This is the one thing that has come back wrong every time: a "bright edge" or a "highlight" gets
drawn warm, and the mark ends up two-colored — a blue bar with an orange top, a purple bar with a
gold rim. A blue object is lit in a paler blue. A purple object is lit in a paler purple. The only
thing allowed to be near-colorless is the small hot spot at the very brightest point, and it is
white rather than yellow.

No second hue anywhere else either: no brown handle, no gray metal, no tan wood, no silver rim, no
leather binding, no gold trim, no complementary rim light, no rainbow highlight, no colored
bounce. A part of the object that would naturally be another material is drawn in the same hue as
the rest of it, lighter or darker.

The mark's hue is what it is told apart by at a glance. Two hues in one mark is two claims.

Avoid: scenery, environments, hands, characters, props, particles, sparks, smoke, motion lines,
runes, engraved text, gems set into hafts, wrapped cord detail, rivets, filigree, drop shadows,
outer glows, and clutter of any kind. **No text anywhere** — no label, no caption, no signature,
no watermark.

Priority order: **readable silhouette at 16 pixels -> unmistakable hue -> no second hue ->
material shading -> everything else.**

### The matrix

I need **twenty images**: four shapes, each in five colors.

| Shape | Object | The silhouette's job |
|---|---|---|
| `stab` | a **spear** | thin and straight on the diagonal, point to the **upper left** — a long thick shaft to a leaf point; the narrowest silhouette of the four |
| `slash` | a **scimitar** | a single deep curved blade on the diagonal, point to the **upper right** — broad-backed, widening toward the tip, a short plain guard; leaning against the spear so the two never read alike |
| `crush` | a **caveman club** | **full height of the square, tilted about 15 degrees with the head to the upper right.** A crude prehistoric bludgeon cut from a single tree branch, and a **fat** one: the grip at the bottom is already thick — about a tenth of the canvas wide — and it swells steadily to a heavy knotted head **at least a third of the canvas across**. Rough gnarled surface, a stump or two where limbs were hacked off. **One continuous piece** — no separate head fixed to a handle, no joint, no collar, no banding, no spikes. It is emphatically **not a hammer, not a mace and not a mallet**: the swelling is gradual along the whole length rather than a block sitting on a stick. **Err heavy.** The last two attempts were far too slender and read as a twig; this should look like something that would take two hands. Blunt, primitive, heavy |
| `defend` | a **shield** | **upright, 75% of the square** — a wide-shouldered heater shape tapering to a point at the bottom; the only non-weapon in the set. A plain face: a central boss or a single rim is enough, and **no heraldry, no device, no quarters, no charge** |

Five hues, and the hue is all that changes. **Match these exactly.**

| Name | Word | dark | core | light |
|---|---|---|---|---|
| `fire` | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice` | blue | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth` | green | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane` | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

**`core` is the hue to match**, and the element name and the color word are the same instruction —
`arcane` and "purple" both mean that row. `light` is the faces turned toward the light and the top
of the specular ramp; `dark` is the faces turned away and the color the shadow settles into, above
the near-black contour. **Stay inside the ramp**: anything between `dark` and `light` is in range,
anything outside it is a different color.

**There is deliberately no material column.** Every object in this set is the same substance —
one smooth faceted material, the same for all four shapes — and the color is **applied to it**,
never a different substance. The fire spear is not hot iron, the ice one is not ice, the earth one
is not stone and the lightning one is not brass. They are one spear, tinted five ways.

**The five versions of one shape must be the same drawing in five colors.** Not five materials,
not five objects, not five interpretations. Identical silhouette, identical facets, identical
highlights, identical proportions, identical position in the square — the only difference between
`formstab-fire.png` and `formstab-ice.png` is the hue. Laid side by side they should look like one
picture printed five times in five inks, and anything that does not survive that test has gone too
far.

The practical way to get there: **draw each shape once — one spear, one scimitar, one club, one
shield — then recolor each drawing five times.** Do not redraw it per color.

Name each file `form<shape>-<name>.png` — `formcrush-fire.png`, `formdefend-arcane.png` — and give
me a link I can click to download each one.

---

## Prompt one and a half — the four gray marks

**Four more images, the same four objects with no color at all.** They are used in a row that is
about the shape rather than the color, so a tinted mark there would be claiming something the row
is not about.

Draw the **same four drawings** — the spear, the scimitar, the club, the shield — under every rule
in prompt one, identical in silhouette, pose, lean, framing and size, and change only the color:

- a **neutral gray** in place of the hue, mid-gray across the body, lighter gray on the faces
  turned toward the upper left, darker gray on the faces turned away
- the same near-black contour, the same thickness
- the same near-white hot spot

**No hue at all** — not a warm gray, not a cool gray, not a tint of anything. These sit beside the
five colored versions and their whole job is to say "no color".

Name each file `form<shape>-neutral.png` — `formstab-neutral.png`, `formcrush-neutral.png`.

**These are recolors, not redraws** — the gray club has the same tilt, the same thickness and the
same framing as the five colored ones.

---

## Prompt two — the bar

**One drawing per color, repeated.** One, two or three copies of the same bar are stacked in a
column, so this is a single object and not a set, a sequence or a sheet.

The thing is **eight pixels tall**, so what it can carry is a **profile** — a lit top edge, a dark
under edge, ends that are shaped rather than square-cut — and nothing else. No engraving, no
texture, no ornament, nothing representational, no icon.

Canvas: **exactly 640x256** — 5:2, two and a half times as wide as it is tall — transparent
background, PNG with an alpha channel. Work larger if you like and reduce once to 640x256 on the
way out. It is used at 20x8 and at 10x4, so anything finer than about a thirtieth of the canvas
width is gone.

**The contour rule above applies here and matters more.** A bar eight pixels tall has room for a
lit edge, a body and a dark edge and nothing else, so if the near-black outline averages away the
bar is a smear of color. Draw it at least 8 pixels thick on the 640-wide canvas, and check it by
reducing to 20x8 yourself.

What to draw: **a single short horizontal bar** in its own hue, seen straight on — a small ingot,
or one rung of a ladder. It fills about **90% of the canvas width and the full height**, so it
reads as a bar rather than as a lozenge floating in space. Ends chamfered, tapered or slightly
rounded rather than cut square.

Lit from the upper left: a bright bevel along the top edge, a dark one along the bottom, a
near-black contour all the way round. Shade it with the same material response as the marks, so it
reads as a small forged object rather than as a colored rectangle.

**The bright top edge is the bar's own hue made lighter — not white, not yellow, not gold, not
orange, not warm.** A blue bar is lightened to a paler blue; a purple one to a paler purple. A warm
or golden top edge on a cool bar is the single most common thing that comes back wrong here, and it
is wrong every time.

It must read as an **object** rather than as a rectangle of color, while still reading
unmistakably as a horizontal bar at a glance. When three of these are stacked one above another
with a gap between them, they have to look like three of the same thing — so keep it symmetrical
left to right, and even enough top to bottom, that a stack reads as a stack and not as a broken
pattern.

Background: **fully transparent, and no border or frame of any kind.** Same instruction as the
marks and it matters more here, because a bar with any background at all is a visible box sitting
on a pale surface — and a bar is the shape most easily mistaken for something that wants a box
drawn round it. The only dark line in the image is the contour on the bar's own edge. No ground,
no shadow, no glow, no panel, no plate, no frame, no outer rectangle.

Same hues, same one-hue-only rule, same light, same smooth non-pixel rendering and the same
avoid-list as the marks above. The bar is the same material as the four marks.

### The six

One image per name, in the hues given in the matrix above: `fire`, `ice`, `lightning`, `earth`,
`arcane`. **The same bar in five colors** — same width, same weight, same ends, same light, same
everything but the hue.

**And a sixth in neutral gray**, on the terms the four gray marks are drawn on: mid-gray body,
lighter gray along the top edge, darker gray along the bottom, the same near-black contour, no hue
at all.

Name each file `tick-<name>.png` — `tick-fire.png`, `tick-neutral.png` — and give me a link I can
click to download each one.

---

## Before you hand anything back

Check every image against these six. A failure on any one of them means redraw rather than
explain.

0. **Reduce it to 32x32 and look at it.** That is the size it is used at. If the shape has gone
   soft, or the dark ring around it has vanished, the contour is too thin or too light — go back
   and thicken it. This check catches more than the other five put together.
1. **Is the background actually transparent?** Not black, not white, not a dark square. Empty
   alpha, right up to the object.
2. **Is there a frame?** There must be no rectangle, box, border, panel or dark margin anywhere.
   The only dark line is the contour on the object's own edge.
3. **Is there a second hue?** Hold the image at arm's length. Every colored part of it should read
   as one color — if an edge, a rim or a handle reads as a *different* color from the body, it is
   wrong. A near-white hot spot and a near-black contour are fine; a gold rim on a blue object is
   not.
4. **Is the subject big enough — in both directions?** The two long weapons run nearly corner to
   corner. The club runs the full height and is thick enough that its head is a third of the
   canvas across. The shield is about three quarters of the frame. **Then squint at the four
   together: they should look like roughly equal amounts of ink.** A club that is much lighter
   than the shield beside it is the failure this check exists for.
5. **Do the five versions of one shape match?** Same silhouette, same facets, same position, same
   size. If one of them is a different drawing rather than a different color, redo it.
