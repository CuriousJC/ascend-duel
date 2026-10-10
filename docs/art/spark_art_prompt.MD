# Spark art prompt

Fifteen small spark sprites for a card game: **three spark shapes, each drawn in the colors of five
elements — fire, ice, lightning, earth, arcane**. A spark here is a plain bright fleck of light, not
a piece of any material. The shapes are the same in every element, so **the only difference between
a fire spark and an ice spark is its color**. The game handles all the motion: it spawns dozens of
these, scales them, spins some of them, fades them and moves them around a damage number and behind
it as the number flies across the screen. **Each file is a single still spark. It is not a burst, a
shower, a trail or an animation frame.**

---

## The prompt

High-resolution digital illustration of **fifteen small game spark sprites**, one per file, each a
single spark on a **fully transparent** background. Think of the flecks of colored light a polished
mobile card game throws off a hit: simple, bright, clean shapes.

### Every spark

- **One spark, centered, filling about 75% of the canvas**, with clear space all round. Its center
  sits on the center of the canvas, because the game spins and scales it about that point. Nothing
  else is in the picture.
- **Seen on screen between 8 and 32 pixels across.** Bold, simple shapes only: nothing in the spark
  is thinner than about **an eighth of the canvas**. No hairlines, no fine rays, no detail. **Shrunk
  to 12 pixels, each must still be its shape and its color.**
- **Colored in its element's ramp** (table below): a **hot center in the `light`**, the `core`
  across the body, and the `dark` only at the very rim, just inside the contour. Smooth radial
  shading from the bright middle outward, anti-aliased edges. **Saturated and bright**: these are
  the most vivid things on the screen for the moment they are up.
- **A thin near-black contour all the way round**, about **a fortieth of the canvas** thick,
  unbroken. The sparks fly over a **pale blue-gray table** as well as over darker painted scenery,
  and on the pale ground a spark with no outline disappears. The contour is what makes a yellow or a
  pale blue spark read there.
- **The light comes from the spark itself**, so the brightest point is its center and the shading
  is the same on every side. There is no light direction and no highlight in a corner.
- **No glow or halo outside the contour.** The game adds any glow it wants. A soft glow painted into
  the file turns into a gray smudge on the pale table and a hard-edged square when sparks overlap.
  Everything outside the contour is fully transparent.
- No background, no shadow, no motion blur, no speed lines, no trail, no smaller sparks around it,
  no frame, no text, no letters, no numbers.

### The five ramps

| Element   | dark      | core      | light     |
| --------- | --------- | --------- | --------- |
| fire      | `#9E2F13` | `#F06A24` | `#FFB24A` |
| ice       | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| earth     | `#285E35` | `#4E9B57` | `#8BCB78` |
| arcane    | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

Fire is **orange**, ice is **blue**, lightning is **yellow**, earth is **green** and arcane is
**purple**. Stay inside each element's ramp: no pure white center, no other hue creeping in.

### The three shapes

Each shape is drawn **identically in all five elements**: same outline, same size, same proportions,
only the colors change.

| Shape | What it is |
| ----- | ---------- |
| `1`   | **A dot**: a perfect round spark, a bright disc with its hottest point at the center fading to the core and a thin dark rim. |
| `2`   | **A star**: a fat four-pointed star, the four points equal, short and blunt, with a round hot center. Upright, its points straight up, down, left and right. |
| `3`   | **A streak**: an elongated spark like a stretched teardrop, rounded and brightest at the top, tapering to a blunt point at the bottom. **The round head is at the top.** About two and a half times as tall as it is wide. |

### The fifteen files

| File                 | Shape | Element   |
| -------------------- | ----- | --------- |
| `spark-fire-1`       | dot    | fire      |
| `spark-fire-2`       | star   | fire      |
| `spark-fire-3`       | streak | fire      |
| `spark-ice-1`        | dot    | ice       |
| `spark-ice-2`        | star   | ice       |
| `spark-ice-3`        | streak | ice       |
| `spark-lightning-1`  | dot    | lightning |
| `spark-lightning-2`  | star   | lightning |
| `spark-lightning-3`  | streak | lightning |
| `spark-earth-1`      | dot    | earth     |
| `spark-earth-2`      | star   | earth     |
| `spark-earth-3`      | streak | earth     |
| `spark-arcane-1`     | dot    | arcane    |
| `spark-arcane-2`     | star   | arcane    |
| `spark-arcane-3`     | streak | arcane    |

**The fifteen are one set.** Same contour thickness, same finish, same brightness. Laid out as a
grid, shapes across and elements down, every column must be one shape and every row one color.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as
you like and reduce once to 256 on the way out. If you cannot produce real transparency, use a flat
pure magenta `#FF00FF` background with nothing else on it, and say so.

Avoid: glow, halo, bloom, lens flare, motion blur, trails, speed lines, fine rays, smaller sparks,
backgrounds, shadows, text, letters, numbers, and any off-ramp color.

---

## Before you hand anything back

1. **Shrink every spark to 12 x 12.** Is it still a dot, a star or a streak? Is its color still
   obviously its element?
2. **Lay each one on a pale blue-gray ground (`#B8C4D6`) and on a near-black one.** Does it read on
   both? Is the contour unbroken all the way round?
3. **Lay the fifteen out as a grid**, shapes across and elements down. Is every column exactly one
   shape, differing only in color?
4. **Does every streak have its round head at the top?**
5. **Is everything outside each contour fully transparent**, with no glow, no halo and no fringe?

Name the fifteen files exactly as in the table, each ending `.png`, and give me a single zip file of
everything so I can download it all at once, never a link per file.
