# Enemy card art prompt

One picture per **attack flavor** per element: **the attacking part alone, made of swirling colored
vapor, on a transparent background.** A creature's cards wear these. A wolf's Bite and a lion's
Bite are one picture, so a picture is never about a creature. It is about a kind of blow: jaws
closing, claws raking, a club coming down.

The game lays it over the card's own face, which is sometimes plain off-white (#F0EFEA) and
sometimes a rich material. **The subject has to read clearly on every one of those**, which is
the whole brief.

---

## The canvas

**Portrait, aspect ratio exactly 5:7**, five wide to seven tall. Render as large as you work
natively; around 1060 x 1484 is right. It is reduced to **200 x 280** afterwards, so anything
finer than about a two-hundredth of the width is lost.

**The background is fully transparent**: a PNG with an alpha channel, where every pixel that is
not the subject has an alpha of zero. **Not a checkerboard pattern drawn to look transparent**,
not white, not a pale wash. If you cannot output real transparency, fill the background instead
with **one flat, uniform pure magenta, #FF00FF**, a color that must appear nowhere in the subject,
with no gradient, no texture and no shadow, so it can be removed exactly.

The picture is laid over the card and clipped to the card's rounded corners, so a little is lost
at the edges. Nothing important should sit in the outermost twentieth.

## What the card draws on top

The card prints four small things over the picture, at these positions on the finished 200 x 280
card:

| What | x | y |
|---|---|---|
| a small mark, top-left corner | 13–44 | 10–44 |
| a short stack of small bars down the left edge | 11–28 | 60–100 |
| a round badge, bottom-left corner | 13–44 | 238–268 |
| the card's name, centred along the bottom | 60–140 | 240–268 |

**These are not reserved areas.** Do not draw a band, a panel or a blank patch at any of them.
They are why the subject belongs in the open middle of the card, roughly **x 46 to 196, y 46 to
236**: a tall field right of a narrow left-hand column and above a strip along the bottom.

## What every one of these pictures is

**The part of a creature that does the hitting, and nothing else, made of vapor.** A sting is
only the stinger. A bite is only the jaws and their teeth. A club is only the club. There is no
creature, no body, no arm and no face behind it. Where the attacking part would join a body, it
frays into trailing smoke and stops.

- **The subject says what kind of blow it is**, and the record's own brief names it. Every
  picture belongs to one of three forms, which the card's corner mark also says: a **stab** is a
  point going in, a **slash** is an edge or a sweep going across, and a **crush** is weight
  arriving. The picture should read as its form at a glance, before it reads as anything else.
- **The color of the vapor says the element.** All of the subject is vapor, so all of it is
  colored.
- **One picture serves every cost.** The same flavor is worn by a creature's cheapest card and
  its dearest, so the subject is drawn at one committed, mid-sized scale. Do not draw it small and
  tentative or vast and overwhelming.

**The vapor holds a hard shape.** It is smoke or mist, denser at the core and fraying at the
trailing edges, but the attacking part itself keeps a crisp, readable silhouette: a fang is a
sharp point, a claw is a hooked edge, a boulder is a heavy angular mass. **Vapor that drifts into
a vague cloud has lost the subject**, and that is the failure to avoid.

### Where the subject goes

**The subject fills the open middle of the card**, using the full height of the field, about y 46
to y 236, and the full width right of the left-hand column. Diagonals are good: a blow crossing the
card from one corner toward the other reads as motion without any motion lines. **The attacking
part may run out toward an edge**, but its working end, the point, the edge or the striking face,
stays well inside the field.

## Drawing

**The subject carries a dark outline, and that is what lets it read on any ground.** Round the
whole silhouette runs a **near-black contour**, heavy enough to hold at card size, about a
hundredth of the canvas width at its thickest. It is an ink line with some life in it, thickening
on the shadow side and thinning where the vapor frays, and it follows the vapor's own drifting
edge, wisps and curls included. **Inside the line the subject is still vapor**: soft gradients,
drifting density, no hard edges within it.

**The vapor is saturated and opaque enough to carry its own color** on gold, on silver, on dark
steel and on off-white alike. Densest and richest along the working end, thinning toward the
trailing smoke, but even the thin parts stay clearly colored, and **the vapor never becomes so
transparent that the ground behind it shows through more than faintly.**

Anti-aliasing, soft gradients and blended edges are expected and wanted, including the
anti-aliased edge of the subject against the transparency. Do not posterize, do not dither, do not
quantize to a block grid, and do not restrict the palette to a fixed number of colors.

Light: from the **upper left**, with highlights on the upper-left faces and shadow falling away to
the lower right.

Avoid: any background at all, a ground plane, a cast shadow, a glow or halo spreading off the
subject into the background, a creature or any part of one beyond the attacking part, faces, eyes
(except where the brief makes an eye the source of a ray), scenery, props, borders, frames,
panels, particles floating free of the subject, sparks, motion lines, and clutter of any kind.
**No text anywhere**: no title, no label, no caption, no numeral, no signature, no watermark.

Priority order: **transparent background -> subject outlined and saturated enough to read on any
ground -> the blow readable as its form -> coherent color -> material shading.**

## The subject

The block pasted below this prompt is one record, in JSON, with an `Elements` list beside it:
**draw one picture per element in the list**, each the same subject in that element's color, and
name each file as `Files` gives it.

**Read `Draw` and nothing else as the subject.** `Flavor`, `Form` and `Name` are context, and none
of them is to be rendered as a word. If `Draw` is empty, stop and say so rather than inventing a
subject.

**The element is the color the vapor is built in**, and the five are the game's own:

| Element | Word | dark | core | light |
|---|---|---|---|---|
| `fire` | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice` | blue | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth` | green | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane` | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

The element name and the color word are the same instruction: `arcane` and "purple" both mean that
row. **`core` is the body of the vapor**, `light` its brightest wisps and highlights, and `dark`
its shadowed side, just inside the contour. Stay inside the row. Drifting to a neighbouring hue,
fire into red, lightning into orange, ice into teal, earth into olive or arcane into pink, is the
failure.

**The same flavor in two colors is the same picture in two hues, not two pictures.** A fire bite
and an ice bite are one set of jaws, one angle, one composition, one framing, and two colors of
vapor.

**What the element changes besides the hue is what the vapor is *like***: flame curls upward,
frost mist settles and goes brittle at the edges, a charged haze crackles, loam hangs heavy, and
witch-light glows from inside. It never changes the subject, its angle or its placement.

Name each image `<Flavor>-<element>`, like `stab-bite-fire`. Give me the file itself if there is
only one, and a single zip file of everything if there is more than one, so I can download it all
at once. Never a link per file.

---

## Before you hand anything back

1. **Is the background really transparent?** Open the PNG over a dark color and over a bright
   one. The ground should change with it; a checkerboard or a white box that stays put is a
   failure.
2. **Lay it over flat gold (#C9A227), flat dark slate (#1E2A38) and off-white (#F0EFEA).** Is the
   subject clearly visible on all three, its outline holding it on each, and its color still its
   element?
3. **Reduce it to 200 x 280 and look at it.** A subject that has gone mushy was too finely
   detailed; an outline that has vanished was too thin.
4. **Is there any text, numeral, frame or panel in the image?** There must be none.
5. **Is it only the attacking part?** A head behind the jaws or an arm behind the club is a
   creature, and the picture has to serve every creature that strikes this way.
6. **Does it read as its form?** Cover the corner of the card and ask whether this is a point
   going in, an edge going across, or weight arriving. It should be obvious.
