# Scroll art prompt

A **papyrus scroll**: the strip of paper a card game writes its damage arithmetic on, one strip hanging
under each playing card. The game prints large numbers and symbols on top of it, and the strip has to
be **as long as the arithmetic** — sometimes a few lines, sometimes the whole height of the screen — so
it is drawn as **a top roll, a bottom roll, and a middle that repeats vertically**. The game draws the
top roll, then the middle repeated as many times as it needs, then the bottom roll; when the scroll
**unrolls**, the bottom roll travels down and the paper appears behind it.

**Draw one scroll, on its own.** The game sizes and places it; never draw writing on the paper, a hand
holding it, or more than one scroll in the file.

---

## The prompt

High-resolution digital illustration of **one game-interface scroll of papyrus**, hanging vertically,
in a bold inked graphic style. It is not rendered in a vector program and it is not pixel art.

### What it is for

**Big colored numbers are written over the paper**, in white-outlined figures in red, blue, yellow,
green, purple and pink. The paper is a ground for them: it has to look like old papyrus at a glance and
then **get out of the way**. Anything busy on the paper — dark marks, strong fibers, stains, writing —
fights the numbers and is wrong.

### The look

- **A hand-inked near-black contour round the whole silhouette**, about a fortieth of the canvas
  width thick: 12 pixels at 512 wide. Near-black is `#15171C`. Confident, crisp, graphic-novel edges;
  not a uniform vector stroke, not sketchy.
- **The paper is papyrus**: a sheet woven from flat reed strips. Show it with **faint, low-contrast
  fibers** — a few long thin streaks running down the sheet and fewer across it, only slightly darker
  or lighter than the paper. Visible on a close look, gone at a glance.
- **The color is flat with hard-edged tone bands**, the way a comic colorist would shade it: the
  paper in its `core` value, a slightly lighter `light` band down the left edge of the sheet, a
  slightly darker `dark` band down the right edge. No gradients, no glow, no gloss.
- **The rolls are the only place with real modelling**: each is a cylinder of the same papyrus wound
  on itself, its round end showing a tight spiral of the paper's edge on both sides. Light from the
  **upper left**: the `light` value along the top of each cylinder, the `dark` value along its
  underside and in the spiral.
- **The rolls are slightly wider than the sheet**, so each one overhangs the paper by a little on both
  sides, the way a real scroll's rod does.
- **No rods, knobs, ribbons, seals, tassels or gold.** Just paper rolled on itself.

### The color

| File            | dark      | core      | light     |
| --------------- | --------- | --------- | --------- |
| `scroll-mid`    | `#8A775E` | `#C5AA86` | `#D9C8B0` |

### The canvas, and the repeat

The file is **exactly 512 x 1024**, a **transparent** PNG: nothing but the scroll on a fully
transparent ground.

- **The rolls span the full width**, x 0–511. **The sheet sits inside them**, x 32–479, its contour
  running straight down both sides.
- **The top roll lives in y 0–159.** The cylinder sits at the top of the canvas and the paper comes
  out from under it, so the bottom of this band is already plain sheet.
- **The bottom roll lives in y 864–1023**, the same cylinder mirrored: plain sheet at the top of the
  band, running into the roll underneath it.
- **y 160–863 is the repeating middle**: plain sheet, side contours and side tone bands only. It is
  drawn many times stacked on top of itself, so **its top row must continue seamlessly from its bottom
  row** — every fiber that leaves the bottom edge comes back in at the top in the same place. Nothing in
  it may be a one-off mark that would be seen repeating: no stains, tears, holes, knots or blotches.
- **The edges where the rolls meet the middle must match** the middle's top and bottom rows exactly,
  so no seam shows where the game joins them.

Avoid: writing, letters, numbers, symbols, seals, wax, ribbons, rods, knobs, gold, stains, tears,
burn marks, glow, gradients, drop shadows, and anything dark or busy on the paper itself.

---

## Before you hand anything back

1. **Stack the middle band three times between the two rolls** and look along the joins. Is there a
   seam, a jump in a fiber, or a band that repeats visibly?
2. **Cut it short**: the top roll, one copy of the middle cropped to 100 pixels, the bottom roll. Does
   it still read as a scroll?
3. **Reduce the whole thing to 200 pixels wide.** That is the size it is used at. Is the contour still
   there all the way round, and do the rolls still read as rolled paper?
4. **Write a big red `40` and a blue `x 4` across the paper** in thick white-outlined figures. Can both
   be read instantly, or do the fibers fight them?
5. **Is the ground fully transparent**, with no halo of near-white or near-black pixels round the
   contour?

Name the file exactly as in the table, ending `.png`.
