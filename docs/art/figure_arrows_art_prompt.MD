# Figure arrows art prompt — up and down

Two glyphs: **an up arrow and a down arrow**, drawn as solid wedges, to go with a sheet of bold
inked capitals and digits that already exists (`figure_letters_art_prompt.MD`). They are the labels
of small buttons — a row of six wheels, each turned by a button above it and a button below it —
and are set on those buttons exactly as a word like `CANCEL` is set on a bigger one.

**Attach the finished preview of the letters sheet, and match it.** An arrow that looks like it
came from a different hand, has a thinner outline, or is lighter than the letters is the whole
failure.

---

## The glyphs

- **`▲` is a solid triangle pointing straight up**; **`▼` is the same triangle pointing straight
  down** — the one drawing turned over, not a second drawing.
- **A broad, stable wedge, wider than it is tall**: the base about one and a quarter times the
  height. Not a tall spike and not a flat sliver.
- **The corners are slightly softened**, as a heavy brush stroke rounds them, but the point is
  still a point: it reads as an arrow at a glance, never as a rounded mound or a dome.
- **Solid all the way through.** No hole, no inner outline, no chevron (`^` shape) with an open
  middle, no shaft or stem below the head, no second smaller triangle.
- **Nothing else**: no circle, no badge, no plate behind it, no motion lines.

## The look

The same style as the attached letters, exactly.

- **A black contour 12 pixels thick — the same as the letters', and no thicker or thinner.**
  Measure it across the side of their `I` at mid-height: 12 pixels. It wraps all three sides and all
  three corners at that thickness.
- **True black**, or within a few points of it — not a dark shade of the fill's color.
- **A hard black drop shadow**, about 6 pixels down and slightly right, merged with the contour.
  No blur.
- **A flat fill in the color's `core`, with a hard-edged `light` band along the top of the shape
  and a hard-edged `dark` band along the bottom.** On `▼` the bands stay where the light puts them —
  light on top, dark underneath — so it is **not** simply `▲` rotated with its shading upside down.
- **No gradients, no texture, no streaks, no sparkle, no gloss, no white highlight line, no glow
  and no bevel.**
- **Light from above and slightly to the left.**

## Where it sits

Each glyph is drawn in a **160 x 160 cell**, against these measurements of the attached letters:

| On the letters                              | Pixels                        |
| ------------------------------------------- | ----------------------------- |
| outline thickness                           | **12**                        |
| top of a capital's black outline            | **23** down from the cell top |
| bottom of a capital's black outline         | **147**                       |
| middle of a capital's fill                  | **85**                        |
| bottom of the drop shadow                   | 152                           |

**Each arrow's black, outline included and drop shadow left out, fits a box 100 wide by 80 tall,
centered on 85 up and down and centered left to right** — so an arrow is as heavy as a capital but
sits in the middle of the line, the way `+` and `=` do. Nothing it draws comes within 4 pixels of
any edge of its cell.

## The sheet

**One PNG per color, each exactly 320 x 160** — two 160 x 160 cells side by side, `▲` on the left
and `▼` on the right — transparent background, PNG with an alpha channel. Everything outside the
glyphs is fully transparent: no ground, no grid lines, no frame, no labels.

Every color is **the same two drawings, recolored**: only the three fill values change.

**Match these values exactly.** `core` is the body of the glyph, `light` is the band along the top,
and `dark` is the band along the bottom.

| Sheet name  | Color                 | dark      | core      | light     |
| ----------- | --------------------- | --------- | --------- | --------- |
| `fire`      | orange                | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice`       | blue                  | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow                | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth`     | green                 | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane`    | purple                | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| `attack`    | red                   | `#7A1E1E` | `#BA3434` | `#EE7A6A` |
| `relic`     | pink                  | `#9A2F68` | `#EB69AA` | `#FFB8DA` |
| `vitae`     | crimson               | `#5E0C16` | `#A81A2A` | `#E0525E` |
| `neutral`   | white to gray, no hue | `#A6A6A6` | `#E8E8E8` | `#FFFFFF` |

**`neutral` is the one the buttons use**, and the game multiplies its fill by other colors at run
time, so it has to be **pure gray, with no hue at all**. The black contour and drop shadow come
through the multiply as black.

## The JSON

**One file, `figure-arrows.json`:**

```json
{
  "cellWidth": 160,
  "cellHeight": 160,
  "columns": 2,
  "rows": 1,
  "baseline": 136,
  "digitHeight": 112,
  "glyphs": {
    "▲": { "x": 0,   "y": 0, "w": 160, "h": 160, "left": 30, "right": 130, "advance": 104 },
    "▼": { "x": 160, "y": 0, "w": 160, "h": 160, "left": 30, "right": 130, "advance": 104 }
  },
  "sheets": {
    "neutral": {
      "image": "figure-arrows-neutral.png",
      "ramp": { "dark": "#A6A6A6", "core": "#E8E8E8", "light": "#FFFFFF" }
    },
    "...": {}
  }
}
```

- **The keys are the characters themselves**: `"▲"` is U+25B2 (black up-pointing triangle) and
  `"▼"` is U+25BC (black down-pointing triangle). The file is UTF-8.
- `x`, `y`, `w`, `h` are the glyph's cell in the sheet.
- **`left` and `right` are the first and last columns the contour reaches**, not counting the drop
  shadow, and **`advance` is how far to move right after it**, measured so its contour just touches
  a neighbouring capital's. Measure all three off the drawn pixels; the figures above are
  placeholders.

## Naming

`figure-arrows-<name>.png` for each of the nine colors, plus `figure-arrows.json`, plus
**`figure-arrows-preview.png`**: all nine sheets stacked over a **light slate-blue ground,
`#7EACE4`**, and under them, in `neutral` and using the attached letters, the lines `▲ A ▼` and
`H▲H  H▼H`, each glyph at the height it has in its own cell so the lines read as one row. Then the
preview's `neutral` sheet again at a fifth of its size, which is about the size the game draws it.

Eleven files. Give me a single zip file of everything so I can download it all at once — never a
link per file.

---

## Before you hand anything back

Check every sheet against these. A failure on any one means redraw rather than explain.

0. **Measure each arrow against its box**: 100 wide by 80 tall, outline included and drop shadow
   left out, centered on 85. An arrow taller than the capitals beside it in `H▲H` means redraw.
1. **Measure the outline.** Count the black pixels across the slanted side at its middle: about 12,
   the same as across the side of the `I` in the attached letters, and true black.
2. **Is `▼` the same shape as `▲`, turned over, with its light band still on top?**
3. **At a fifth of the size, are both still unmistakably arrows** — a point, not a mound — and
   solid, with no hole?
4. **Are the nine sheets the same drawings**, only the fill changing, and is `neutral` pure gray?
5. **Is nothing within 4 pixels of a cell's edge?**
