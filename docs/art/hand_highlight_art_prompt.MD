# Hand highlight art prompt

A **highlight plate**: the panel a row of playing cards sits on to show they belong together. The
game lays one plate behind every run of neighbouring cards it wants to mark — one card, two side by
side, five in a row — so the plate is drawn as **two end caps and a middle that can be stretched
sideways without changing**.

**Draw the plate once, on its own.** Never draw cards on it, and never draw two plates.

---

## The prompt

High-resolution digital illustration of **one game-interface highlight plate**, in a bold inked
graphic style. It is not rendered in a vector program and it is not pixel art.

### What it is for

Playing cards stand on top of the plate and cover almost all of it. **Only a band about one
eighteenth of the plate's height shows round the edge**, on all four sides. So the plate is read by
its **rim**: that band is the whole of what the player sees, and it is what has to say "these cards
are together" against a pale slate-blue tabletop.

### The look

- **A heavy, hand-inked near-black contour all the way round**, about a twentieth of the plate's
  height thick. Near-black is `#15171C`. Confident, crisp, graphic-novel edges; not a uniform
  vector stroke, not sketchy.
- **Inside the contour, a bright luminous rim** about a twelfth of the height wide, in clean
  neutral white with no hue: `light #FFFFFF`, `core #F2F3F5`, `dark #C9CCD3`. Two hard-edged tone
  bands: the `light` value along the top inner edge, the `dark` value along the bottom inner edge,
  `core` between. No gradients, no texture, no noise, no glow, no gloss, no metal sheen.
- **Inside the rim, the plate's body**, flat `core`. It is covered by cards in play, so it carries
  nothing at all.
- Light comes from **above**.
- **No color.** The plate is white light and black ink only; every hue in the game already means
  something.

### The shape

A **wide rectangle with softly rounded corners** — corner radius about a tenth of the height. The
two ends are mirror images of each other.

### The canvas, and the stretch

The file is **exactly 512 x 256**, a **transparent** PNG: nothing but the plate on a fully
transparent ground.

- **The plate fills the canvas**: its contour touches all four edges. No padding, no drop shadow.
- **The left end cap lives in x 0–95, the right end cap in x 416–511.** All the corner rounding
  happens inside those 96 pixels at each side.
- **Every column from x 96 to x 415 is identical.** The middle is the part the game stretches to
  any width, so it must be the same at every column across it: straight contour on top and bottom,
  straight rim bands, flat body, and nothing that varies along the length.

### File

`hand-highlight.png`.
