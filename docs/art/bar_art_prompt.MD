# Bar cell art prompt

A set of **bar cells**: the pieces a card game builds its resource bars from. A bar is a row of
cells laid end to end, one cell per point of the resource, and each cell shows one of three states.
The game stretches every cell to whatever width the row needs, so a cell is drawn as **two end caps
and a middle that can be stretched sideways without changing**.

**Draw each state once, on its own.** The game lays the cells out, spaces them and stretches them;
never draw a whole bar or a row of cells as one picture.

---

## The prompt

High-resolution digital illustration of **three game-interface bar cells**, one per file, in a bold
inked graphic style. They are not rendered in a vector program and they are not pixel art.

### The look

**A heavy, hand-inked black contour around every cell, and a flat, saturated fill inside it.** This
is the whole of the style, and the contour is the part that matters most.

- **The contour is thick: about an eighth of the cell's height**, all the way round. At a cell
  64 pixels tall that is an 8-pixel line. It is much heavier than looks comfortable at full size,
  and it is meant to be — the cells are used at a quarter of the size they are delivered at, over
  painted scenes of every color and every brightness, and **the black line is what carries them on
  all of those.**
- **The contour is true black**, or within a few points of it. Not a dark shade of the fill, not
  dark gray.
- **It has an inked, graphic-novel quality**: confident, crisp edges, like bold comic-book panel
  work. Not a uniform vector stroke, not sketchy or wobbly.
- **The black also sits a little lower than the cell, as a hard drop**: a solid black shadow offset
  straight down by about a sixteenth of the cell's height, merged with the contour so the two read
  as one black shape. No blur and no soft shadow.

**The fill is flat, with two hard-edged tone bands and nothing more.**

- The body of the cell is the state's `core` value (see the table below).
- A **lighter band along the top** in the state's `light` value, with a crisp edge rather than a
  gradient, like one brush stroke of highlight.
- A **darker band along the bottom** in the state's `dark` value, also hard-edged.
- **No gradients, no texture, no noise, no glow, no gloss, no bevel, no metal sheen.**

Light comes from **above**: the light band on top, the dark band and the drop underneath.

### The shape

A **long, low lozenge with rounded ends** — a pill, a capsule, a gem-cut bar. Straight along the top
and bottom, rounded or slightly chamfered at the two ends. The two ends are mirror images of each
other.

### The three states

| File               | State                        | Reads as                                                           | dark      | core      | light     |
| ------------------ | ---------------------------- | ------------------------------------------------------------------ | --------- | --------- | --------- |
| `bar-cell-empty`   | a point still to be spent    | **an empty socket pressed into the surface** — see below           | `#15171C` | `#2B2F38` | `#454B57` |
| `bar-cell-spent`   | a point spent, within budget | a solid amber piece sitting in the socket, standing proud of it    | `#7A4A06` | `#C47C0C` | `#F2B84A` |
| `bar-cell-over`    | a point spent past the budget | the same piece in warning red                                     | `#7A1E1E` | `#BA3434` | `#EE7A6A` |

**The empty cell is sunken, so its bands are reversed**: the **dark** band runs along the **top**
inside the contour, as the shadow thrown by the socket's upper lip, and the **light** band along the
bottom. It has **no drop shadow** — it is a hole, not a piece. Its outline is the same shape and the
same weight as the other two, so an empty and a spent cell side by side are clearly the same slot,
empty and filled.

**The spent and over cells are the same drawing, recolored.** Same shape, same contour, same bands,
same drop; only the three values change. Draw one, then recolor it.

### The canvas, and the stretch

Each file is **exactly 256 x 64**, a **transparent** PNG: nothing but the cell, its contour and its
drop on a fully transparent ground.

- **The cell fills the canvas**: its contour touches the left and right edges and its top, and the
  drop reaches the bottom edge. No padding.
- **The left end cap lives in x 0–31, the right end cap in x 224–255.** All the rounding, the
  chamfer, any shaping of the ends happens inside those 32 pixels at each side.
- **Every column from x 32 to x 223 is identical.** The middle is the part the game stretches to
  any width, so it must be the same from top to bottom at every column across it: straight contour
  on top, straight bands, straight contour and drop at the bottom, and nothing that varies along
  its length. No texture, no highlight streak that thins out, no taper, no pattern.

Avoid: text, letters, numbers, icons, symbols, gems set into the cell, sparkles, glow, gradients,
reflections, and any decoration in the middle stretch.

---

## Before you hand anything back

1. **Stretch each cell to 1024 x 64 and squeeze it to 80 x 64**, stretching only the middle, and
   look at both. Does anything in the middle smear or show a seam where it meets a cap?
2. **Reduce each to 64 x 16.** That is close to the size it is used at. Is the black contour still
   there all the way round, and can you tell the three states apart at a glance?
3. **Lay a row of six — three spent, three empty — over a dark picture and over a light one.** Does
   the row read as six slots, three filled, on both?
4. **Is the empty cell clearly a hole** and the spent one clearly a piece sitting proud?
5. **Are the spent and over cells the same drawing?** Toggle between them; only the color should
   change.
6. **Is the ground fully transparent**, with no halo of near-white or near-black pixels round the
   contour?

Name the three files exactly `bar-cell-empty.png`, `bar-cell-spent.png` and `bar-cell-over.png`,
and give me a link I can click to download them.
