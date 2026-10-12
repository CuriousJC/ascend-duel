# Enemy card art prompt

One picture per **attack flavor**, per **weight**, per element: **the attacking part alone, made of
swirling colored vapor, on a transparent background.** A creature's cards wear these. A wolf's Bite
and a lion's Bite are one picture, so a picture is never about a creature. It is about a kind of
blow, jaws closing, claws raking, a club coming down, and how hard it lands.

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
They are why the subject belongs in the open middle of the card, roughly **x 30 to 170, y 52 to
230**: a tall field centred left to right, below the corner mark and above a strip along the
bottom.

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
- **The weight says how hard the blow lands**, and a player must see it at a glance: a creature
  holding a light sting and a crushing sting is holding two different threats. See the weights
  below.

**The vapor holds a hard shape.** It is smoke or mist, denser at the core and fraying at the
trailing edges, but the attacking part itself keeps a crisp, readable silhouette: a fang is a
sharp point, a claw is a hooked edge, a boulder is a heavy angular mass. **Vapor that drifts into
a vague cloud has lost the subject**, and that is the failure to avoid.

### The four weights

Every flavor is drawn at four weights, and each record names one. **The weights of a flavor are one
blow at four strengths**: the same attacking part, the same direction and the same family of shape,
escalating. A light bite and a crushing bite are plainly the same kind of jaws.

| Weight | What it is |
|---|---|
| `light` | a quick, small, single strike: slim, short, sharp, little trail |
| `medium` | the blow as `Draw` describes it |
| `heavy` | the blow at full commitment: thicker, denser vapor, a longer and heavier trail |
| `crushing` | the most of it there can be: massive, multiplied or overwhelming, the vapor dense and the outline heavy |

**Draw every weight large, filling the open field.** The game scales each picture to its weight's
size when it installs it, so a light picture is shown smaller than a heavy one whatever size it was
drawn at. A light subject drawn small loses its detail when it is scaled; drawn large, it keeps it.

**So heavier has to read in the shape, not the size.** What separates the weights is mass, density
and number: a slim point against a broad one, two claws against five, one stone against an
avalanche, a thin outline against a heavy one, wispy vapor against packed vapor. Shrink every
picture of a flavor to the same size and the four weights must still be told apart at a glance.

### Where the subject goes

**The subject fills the open middle of the card and stays inside it, centred left to right**:
about x 30 to 170 and y 52 to 230 on the finished card, which on the canvas is roughly **the middle
70% of the width and the middle 64% of the height** (x 160 to 900, y 275 to 1220 at 1060 x 1484). Diagonals are good: a blow
crossing the field from one corner toward the other reads as motion without any motion lines.

**Nothing crosses into the left edge, the top-left corner or the bottom strip, trailing smoke
included.** The card's mark, its bars, its badge and its name are printed there, and vapor running under them
makes them unreadable. A heavy subject, a fist, a boulder, a coil, a wave, is the one most tempted
to fill the card; draw it smaller rather than let it spill. The same applies to the top edge: the
subject starts below the corner mark, not behind it.

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
panels, particles floating free of the subject, motion lines, and clutter of any kind.
**No text anywhere**: no title, no label, no caption, no numeral, no signature, no watermark.

Priority order: **transparent background -> subject outlined and saturated enough to read on any
ground -> the blow readable as its form -> coherent color -> material shading.**

## The subject

The block pasted below this prompt is one batch, in JSON: a `Batch` name and a list of `Records`,
each with an `Elements` list and a `Files` list beside it: **draw one picture per element in each
record's list**, each its own drawing of that record's subject in that element's vapor, and name
each file as `Files` gives it. Every record is a different subject; never let one record's picture
borrow another's.

**A record's subject is `Draw` at its `Weight`.** `Draw` describes the blow at medium weight: the
attacking part, its direction and its composition. `WeightBrief` says what the blow is at this
record's weight; keep `Draw`'s attacking part, direction and placement, and change what
`WeightBrief` changes. A medium record's `WeightBrief` is `Draw` itself. `Flavor`, `Form`, `Name`
and `Weight` are context, and none of them is to be rendered as a word. If `Draw` is empty, stop and
say so rather than inventing a subject.

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

**The five elements of one flavor are five drawings of one idea, never one drawing recolored.**
They are siblings: the same attacking part, at the same size, in the same place on the card,
driving the same way, so a fire bite and an ice bite are plainly the same blow. Everything else is
drawn fresh for each element: the exact silhouette, how the edge breaks up, the number and shape of
the curls, where the vapor trails off, and a few degrees of angle. **Laid side by side, no two may
trace over each other.** A set that is one outline flood-filled in five colors is the failure.

**The element is the vapor's character as much as its color**, and it should change the shape of
the edges, not only their hue:

| Element | What the vapor is like |
|---|---|
| `fire` | hot orange vapor: licking, upward-curling tongues that thin into smoke at the trailing edge, brightest at the working end, shedding a few embers that stay attached to the trail |
| `ice` | cold blue vapor: slow, heavy, settling mist shot through with tiny suspended frost crystals, its edges going **brittle and faceted** rather than curling, points and edges hardening into clear shards |
| `lightning` | crackling yellow vapor: a charged, unsettled haze streaked with **hairline arcs that fork across the subject**, its edges jagged and broken into angles rather than smooth curls |
| `earth` | green loam vapor: a slow, dense, **granular** cloud like turned soil hanging in the air, clotted and heavy, carrying flecks of grit, its trailing edge crumbling rather than curling |
| `arcane` | purple witch-light vapor: a smooth, **self-lit** shimmer that glows faintly from inside, its curls long and calligraphic, fraying into drifting motes rather than smoke |

Embers, arcs, crystals, grit and motes belong to the subject and stay attached to it or to its
trail. Nothing floats free across the card.

Name each image exactly as `Files` gives it, `<Flavor>-<weight>-<element>.png`, like
`stab-bite-heavy-fire.png`,
with nothing added to the name. **Another artist may be drawing the same flavor in parallel**, so
the batch, not the filename, is what keeps the two apart: give me **one zip named `<Batch>.zip`**,
holding **one folder named `<Batch>`** with every image inside it. Never a link per file and never
loose files.

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
7. **Lay a record's five pictures side by side.** Are they plainly the same blow, and plainly five
   drawings? An outline that traces exactly over its neighbour is a recolor; redraw it.
8. **Does it read as its weight, at the same size as the others?** A light picture is slim, single
   and quick; a crushing one is massive, dense and many. Shrink a light and a crushing record of
   one flavor to the same height and lay them side by side: the difference should still be obvious
   from across a room.
9. **Did every record in the list come back, in every element it lists?** Count the files against
   `Files`, and check they are all inside the `<Batch>` folder of `<Batch>.zip`.
10. **Count anything countable.** Fingers, claws, fangs, segments: if `Draw` gives a number, the
   picture has exactly that many in every element. A wrong count is the most common reason a
   picture is thrown away.
11. **Does anything cross into the left edge, the bottom strip or the top-left corner?**
   Reduce to 200 x 280 and check that x below 30, x above 170, y below 52 and y above 230 are
   empty.
