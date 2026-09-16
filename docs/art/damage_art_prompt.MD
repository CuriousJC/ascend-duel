# Damage badge art prompt

Sixty-six images: a small **diamond badge with a number drawn on it**, in eleven values and six
colors.

**The number is part of the drawing.** It is not printed on afterwards — you draw it, sized and
placed and colored to sit on the badge, and it reduces with the rest of the picture. That is the
whole reason this set exists rather than a blank badge with type over it: the badge is used at 32
pixels and at 16, and a numeral drawn into the art averages down along with the contour around it,
where type laid on top at final size arrives at 16 as a few unsupported strokes.

**The badge is used at 32 pixels and at 16 pixels.** There is no zoom and no larger view anywhere,
so detail that does not survive to 32 pixels is detail that does not exist — and the numeral has to
survive to 16.

**Deliver at 256x256.** Not 64, and not 32 either. The two sizes it is used at are reached by
reducing from 256, which is a clean factor of eight and sixteen. A file handed over at 32 is one of
those sizes already and a bad source for the other; one handed over at 64 is an image reduced once
being reduced again, with its outline averaged out of existence on the way. One reduction, from as
much detail as possible.

**Draw eleven badges and recolor each one five times.** This is the quality bar for the whole set,
not a convenience. The six versions of one value are meant to be **one drawing in six inks** —
identical numeral, identical position, identical weight, identical silhouette — so a hand holding
two of them in different colors shows the same badge twice. Redrawing per color is how one color's
`3` comes back fatter than the others, and that is the defect this set fails on.

**Individual files, not a sheet.** One image per badge, named as below. Do not assemble them into a
grid or a sprite sheet: a single badge has to be replaceable on its own.

---

## The prompt

High-resolution digital illustration of a single flat **badge**, rendered as a **game interface
element**, not as a picture of an object in a scene.

Canvas: **exactly 256x256**, transparent background, PNG with an alpha channel. Work as large as
you like natively and reduce once to 256 on the way out; do not hand back 64 or 32. It is drawn at
32 and 16 pixels, so **anything finer than about an eighth of the canvas width is lost.** Draw for
that: one bold shape, few parts, no ornament.

### The shape

**A diamond**: a square standing on one corner, with its four corners reaching the edge of the
canvas. Flat to the viewer, seen straight on, with no perspective, no tilt and no thickness drawn
as a visible side or edge-on rim. Straight edges and slightly softened corners; no facets and no
gemstone cut — it is a flat diamond shape, not a drawn jewel.

**It fills about 95% of the canvas, corner to corner**, with only a hairline of empty space outside
it. Treat 95% as the target rather than a ceiling: the image is placed in a fixed square and is
**never scaled up to fill it**, so a badge drawn small in the canvas is drawn small wherever it is
used, and nothing downstream can rescue it.

**Draw it fat.** A square on its corner narrows fast away from its centre line, and this one has to
carry a number across that centre — so err toward a cushioned, full-bodied diamond with generous
girth at the waist rather than a slim rhombus. See the next section, which measures what has to fit.

### The number

**The number is drawn onto the badge, as part of the picture.** Not printed on afterwards, not a
font laid over the art — you draw it, and it is the thing the badge exists to carry.

- **It sits dead centre**, optically centred in the diamond both ways, and occupies about **55% of
  the canvas across and 40% down**. That is big. A numeral sized for comfort at 256 is a numeral
  that has gone to mush at 16, so err large.
- **Heavy, blocky and simple.** Thick strokes, generous counters, no serifs, no thin joins, no
  flourishes, no outline font, no engraved or embossed treatment. The stroke must be **no thinner
  than a fifteenth of the canvas width** anywhere — the same rule the badge's own contour is under,
  and for the same reason.
- **One flat color, and it is the color that can be read against the badge**: near-white on a
  mid-to-dark badge, near-black on a light one. Pick per color so every one of the six reads, and
  keep it the same across the eleven values of one color so a hand does not show a white 3 beside a
  black 4.
- **Give it a thin dark edge if it needs one to hold**, but nothing decorative. No glow, no shadow
  under it, no bevel on it.

**The badge behind the number stays flat and even.** Shade the diamond like a solid object, but put
the modelling at the **outer** part — light gathering toward the upper-left edge, shadow toward the
lower-right — and leave the middle a broad, quiet, unbroken field of the base color for the numeral
to sit on. **No hot spot, specular glint, rim light, groove, rivet, boss or texture anywhere behind
the number.**

The badge is a surface with a number struck into it: a coin or an enamel token, not a shield boss
and not a decorated medallion.

#### The eleven values

| File token | What to draw |
|---|---|
| `quarter` | **1/4**, as a proper stacked fraction — a 1 above a bar above a 4, not "1/4" on one line. It carries the most ink of any numeral here into the same space, so it is the one most at risk at 16 pixels: draw the bar thick and the two digits heavy, and keep them tight |
| `half` | **1/2**, the same way — stacked, thick bar, heavy digits |
| `1` `2` `3` `4` `5` `6` `7` `8` `9` | the single digit, as large as the flat centre allows |

**The two fractions and the nine digits are the same badge with different numerals.** Do not shrink
the diamond to make room for a fraction and do not grow it for a `1`.

### The background, which is the instruction most often ignored

**Fully transparent everywhere the badge is not.** The alpha channel is empty right up to the
badge's own edge, and there is nothing else in the image at all.

In particular there is **no border and no frame**. Do not draw a rectangle around the picture, do
not outline the edges of the canvas, do not set the badge in a box, a panel, a plaque, a card, a
tile, a roundel, a badge frame or a crest, and do not leave a dark margin around the outside.
**The only dark line anywhere in the image is the contour that hugs the badge's own edge**, plus
whatever thin edge the numeral itself carries. If a second ring is going in further out, that is
the mistake.

Also none of: a ground, a ground plane, a cast shadow, a drop shadow, a glow behind the badge, a
vignette, a backing plate, a colored field. The badge is stamped directly onto a pale off-white
surface, so anything at all behind it becomes a visible rectangle sitting on that surface.

### Drawing

Form: one flat shape carrying one number. Spend the detail budget on two things only — the contour
at the edge, and the numeral in the middle — and none at all on the face between them.

Outline: a **heavy near-black contour** all the way around the badge — genuinely near-black, at the
bottom of the brightness range, and not merely a darker shade of the badge's own color.

**Make it thick: at least 8 pixels of the 256, and 10 is better.** The badge is reduced to 32 and
then to 16, and each reduction averages the outline into whatever is beside it — so a contour drawn
one or two pixels thin, or drawn as a dark version of the hue rather than as near-black, simply
disappears, and what lands is a soft blob with nothing holding it against the pale surface.

The test to apply: reduce your own image to 32x32 and look at it. **A dark edge should still be
visible all the way round, and the number must still be readable.** If the edge has gone soft the
contour was too thin or too light; if the number has closed up, its strokes were too fine or it was
drawn too small.

Shading: smooth and continuous, with a clear material response — a broad gentle ramp from a
lighter value along the upper-left of the edge to a darker one along the lower-right, with the
middle held flat as described above. Two or three broad tonal steps, not a gradient with many
stops. **It should read as a solid struck object with real weight, not as a flat vector shape** —
but the weight is carried at the edge. Anti-aliasing, soft gradients and blended edges are all
expected and wanted. Do not posterize, do not dither and do not quantize to a block grid. **Do not
draw pixel art** — everything around it is smooth, and a pixelated badge would be the one jagged
thing on the screen.

Light: from the **upper left** — the lit ramp on the upper-left of the edge, shadow falling away to
the lower right. Everything in this set is lit from that corner and anything lit from elsewhere
reads as crooked.

Color: **one dominant hue, in its own darker and lighter values.** Shadow is that hue darkened,
light is that hue lightened, the contour is near-black. Shade it as richly as the material wants
inside that — this is a limit on the *hue*, not on the number of tones.

**Every lit part is the badge's own hue made lighter — never a warm color, and never gold.** A blue
badge is lit in a paler blue; a violet badge is lit in a paler violet. No second hue anywhere: no
metal rim, no silver edge, no gold trim, no complementary rim light, no colored bounce. **And no
near-white hot spot**, which is the one thing that would break the number sitting on it. The
numeral is the one exception to the single-hue rule: it is near-white or near-black, whichever can
be read, and never a lighter shade of the badge's own color.

Avoid: scenery, environments, hands, characters, props, particles, sparks, smoke, motion lines,
runes, engraved text, symbols, gems, rivets, filigree, drop shadows, outer glows, and clutter of
any kind. **No text anywhere except the one numeral** — no label, no caption, no unit, no times
sign, no signature, no watermark. One number on the badge and nothing else written anywhere.

Priority order: **the number readable at 16 pixels -> the six colors of one value being one
drawing -> a dark edge still visible at 32 pixels -> filling the frame -> unmistakable hue ->
everything else.**

### The six colors

Five hues, and the hue is the only thing that changes. **Match these exactly.**

| Name | Hex |
|---|---|
| `fire` | `#EB782D` orange |
| `ice` | `#509BE6` blue |
| `lightning` | `#D6980C` amber gold |
| `earth` | `#4C8C34` green |
| `arcane` | `#8A54C8` violet |

**And a sixth in neutral gray**: mid-gray body, lighter gray toward the upper-left edge, darker
gray toward the lower-right, the same near-black contour. **No hue at all** — not a warm gray, not
a cool gray, not a tint of anything.

**The six versions of one value must be the same drawing in six colors.** Not six materials, not
six interpretations. Identical silhouette, identical numeral, identical numeral position, identical
light ramp, identical proportions — the only difference between `damage-diamond-3-fire.png` and
`damage-diamond-3-ice.png` is the hue. Laid side by side they should look like one picture printed
six times in six inks.

**And across values, everything but the numeral is the same too** — same diamond, same size, same
contour weight, same light ramp. The eleven are one badge struck with eleven numbers, not eleven
badges.

The practical way to get there: **draw the eleven badges once, then recolor each one six times.**
Do not redraw per color.

### Naming

`damage-diamond-<value>-<color>.png`, where `<value>` is the file token from the table above and
`<color>` is one of `fire`, `ice`, `lightning`, `earth`, `arcane`, `neutral`:

`damage-diamond-quarter-fire.png` · `damage-diamond-half-arcane.png` ·
`damage-diamond-3-ice.png` · `damage-diamond-9-neutral.png`

Sixty-six files. Give me a link I can click to download each one.

---

## Before you hand anything back

Check every image against these seven. A failure on any one of them means redraw rather than
explain.

0. **Reduce it to 32x32 and then to 16x16, and look at both.** Those are the sizes it is used at.
   **Can you still read the number at 16?** If it has closed up or gone to a smudge, the strokes
   were too fine or the numeral was drawn too small — go back and make it heavier and bigger. This
   check catches more than the others put together.
1. **Lay the six colors of one value side by side.** The numeral must be in the same place, at the
   same size, in the same weight, on the same silhouette. If one color's number sits lower or reads
   fatter, that one is a redraw — and it is one file, not the batch.
2. **Lay the eleven values of one color side by side.** Same diamond, same size, same contour. Only
   the numeral changes. A `1` on a bigger badge than the `9` beside it is wrong.
3. **Are the two fractions stacked and still legible at 16?** `quarter` is the hardest picture in
   this set. If it cannot be read, say so rather than handing it over — that rung needs a different
   answer, not a smaller font.
4. **Is the background actually transparent?** Not black, not white, not a dark square. Empty alpha,
   right up to the diamond.
5. **Is there a frame?** There must be no rectangle, box, border, panel, second ring or dark margin
   anywhere. The only dark lines are the contour on the diamond's edge and whatever thin edge the
   numeral carries.
6. **Is there a second hue, or any other text?** Every colored part should read as one color, the
   numeral should be near-white or near-black, and there should be nothing written on the badge
   except the one number.
