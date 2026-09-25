# Figure glyph art prompt

A set of **number glyphs**: the digits and the handful of math symbols a game writes on screen
when it works out a hit, in a bold inked style, delivered as one sprite sheet per color plus a JSON
file saying where each glyph is.

**These are not text in a paragraph.** They are the figures that pop up over a card game's table
mid-fight: a sum like `(12 x 3) x 1.5 = 54` that assembles itself one term at a time, a `-54` flying
into a health bar, a `+3` flying back out of it. Each figure is on screen for a second or two,
often moving, sometimes growing or shrinking as it travels. They have to be **read at a glance
while they move**, against a busy table, and they are the loudest type on the screen.

**The game assembles every expression itself, glyph by glyph.** Draw each glyph once, on its own;
never draw a whole number or a whole expression as one picture.

---

## The prompt

High-resolution digital illustration of a **set of individual glyphs**: display numerals and math
symbols for a game interface, drawn as bold, inked, graphic lettering. They are **not** a typeface
rendered in a program, and they are **not** pixel art.

### The look

**A heavy, hand-inked black contour around every glyph, and a vivid, saturated fill inside it.**
This is the whole of the style, and the contour is the part that matters most.

- **The contour is thick.** It is **about an eighth of the glyph's height** all the way round: at
  a glyph 100 pixels tall that is a 12-pixel line. That is much heavier than looks comfortable at
  full size, and it is meant to be. The glyphs are used much smaller than they are delivered, and
  sit on light and dark grounds, on tables of every color, and over pictures. **The black line is
  what carries them on all of those**, so a thin or dark-colored outline defeats the point of the
  set.
- **The contour is true black**, or within a few points of it. Not a dark shade of the fill's
  color, not dark gray, not dark brown.
- **It has an inked, graphic-novel quality**: confident, slightly tapered brush strokes with crisp
  edges. It should not look like a uniform vector stroke from a drawing program, and it should not
  look sketchy or wobbly. Think bold comic-book lettering for an action title.
- **The black also sits a little lower than the fill, as a hard drop.** Put a solid black shadow
  offset down and slightly right of each glyph by about a twentieth of its height, merged with the
  contour so the two read as one black shape. No blur and no soft shadow; it is the same black as
  the contour. It gives the glyph weight, as if it were a thick cut-out lying on the table.

**The fill is flat and bright, with two hard-edged tone bands and nothing more.**

- The body of the glyph is the color's `core` value (see the tables below).
- A **lighter band across the upper part** of each stroke, in the color's `light` value, with a
  crisp edge rather than a gradient. It follows the top of the stroke like a highlight painted on
  with one brush stroke.
- A **darker band along the bottom** of each stroke, in the color's `dark` value, also hard-edged.
- **No gradients, no texture, no noise, no glow, no gloss, no bevel, no metal sheen and no
  glitter.** Two crisp bands on a flat core, inside a heavy black line. That is all.

Light comes from **above and slightly to the left**: the light band sits on top, the dark band and
the drop underneath.

### The letterforms

- **Heavy, chunky display numerals.** Bold and wide, with generous counters (the holes in `0`, `4`,
  `6`, `8`, `9`) so that the black contour does not close them up when the glyph is small.
- **Upright**, not italic. Glyphs are laid next to each other by the game and centered on points,
  and a slant makes a sum look like it is sliding.
- **Slightly irregular, in a lively rather than a sloppy way.** A tiny amount of hand-drawn
  character is good, such as a stroke that swells a little or a corner that is a touch sharper
  than its twin. But every digit must be instantly readable, and **all the digits must share one
  height and one weight**. A `1` that looks lighter than an `8` makes a number look misspelled.
- **No serifs, no swashes, no decorative terminals, no 3D extrusion drawn as a visible side.** The
  drop shadow is the only depth.
- **The `1` has a flag and a base**, so that it is as wide and as heavy as its neighbours and does
  not read as an `l` or a stick.
- **The `x` is a multiplication sign**, not the letter x. It is two short, thick crossing strokes,
  smaller than a digit and **centered on the digits' middle height**, as `+` and `=` are.
- **`+`, `-`, `x` and `=` all sit centered on the digits' middle height and are the same visual
  weight as each other.** They are smaller than a digit, about 60% of its height across, and their
  strokes are as thick as a digit's strokes.
- **The brackets `(` and `)` are the full height of a digit or a touch taller**, gently curved,
  and a little lighter in stroke than a digit, so they read as punctuation holding something
  rather than as a figure.
- **The `.` is a solid, heavy dot on the baseline**, big enough to be seen when the glyph is small,
  about the width of a digit's stroke.

### The glyphs

Seventeen glyphs, in this order:

| Row | Glyphs                              |
| --- | ----------------------------------- |
| 1   | `0` `1` `2` `3` `4` `5` `6` `7` `8` |
| 2   | `9` `+` `-` `x` `=` `(` `)` `.`     |

Nothing else. No letters, no `%`, no `/`, no `!`, no currency sign, no plate or badge behind any of
them.

### The sheet

**One PNG per color, all identical in layout.** Transparent background, PNG with an alpha channel.

- **A grid of 9 columns by 2 rows of equal cells, each cell 128 wide by 160 tall**, so every sheet
  is **exactly 1152 x 320**. The glyphs go in the order in the table above, left to right and top
  to bottom; the last cell of row 2 stays empty.
- **Every digit is the same height, about 112 pixels from its top to its baseline**, and every
  glyph in the sheet sits on **one baseline, 136 pixels down from the top of its cell**. The drop
  shadow hangs below the baseline, which is what the room under it is for.
- **The glyph is centered left-to-right in its cell**, and nothing it draws, contour and drop
  shadow included, may cross the edge of its cell. If a glyph does not fit, the glyph is too big,
  not the cell too small.
- **Everything outside the glyphs is fully transparent.** No ground, no grid lines, no frame round
  a cell, no frame round the sheet, no labels, no guide marks and no dark margin. The cells are an
  arrangement, not a drawing.

### The colors

Every color is the **same seventeen drawings, recolored**. The contour, the drop shadow, the shape
of every glyph, where each light band and dark band falls: all identical. Only the three fill
values change. **Draw the set once, then recolor it for every row below.** Do not redraw per color;
two colors of one `3` that differ in shape is the defect this set fails on.

**Match these values exactly.** `core` is the body of the glyph, `light` is the band along the top,
and `dark` is the band along the bottom.

#### The five element colors

| Sheet name  | Color  | dark      | core      | light     |
| ----------- | ------ | --------- | --------- | --------- |
| `fire`      | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice`       | blue   | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth`     | green  | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane`    | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

#### Three more

| Sheet name | Color                 | dark      | core      | light     |
| ---------- | --------------------- | --------- | --------- | --------- |
| `attack`   | red                   | `#7A1E1E` | `#BA3434` | `#EE7A6A` |
| `relic`    | pink                  | `#9A2F68` | `#EB69AA` | `#FFB8DA` |
| `neutral`  | white to gray, no hue | `#A6A6A6` | `#E8E8E8` | `#FFFFFF` |

**`neutral` is used both as it is and recolored by the game**, which multiplies its fill by
another color at run time. That is why it has to be **pure gray, with no hue at all**: not warm,
not cool, not cream, not blue-white. Any tint in it shows up in every color made from it. The black
contour and the drop shadow come through the multiply as black, which is what keeps a recolored
glyph looking like the others.

### The JSON

One file, `figure-glyphs.json`, describing all the sheets:

```json
{
  "cellWidth": 128,
  "cellHeight": 160,
  "columns": 9,
  "rows": 2,
  "baseline": 136,
  "digitHeight": 112,
  "glyphs": {
    "0": {
      "x": 0,
      "y": 0,
      "w": 128,
      "h": 160,
      "left": 18,
      "right": 110,
      "advance": 86
    },
    "...": {}
  },
  "sheets": {
    "fire": {
      "image": "figure-glyphs-fire.png",
      "ramp": { "dark": "#9E2F13", "core": "#F06A24", "light": "#FFB24A" }
    },
    "...": {}
  }
}
```

- `x`, `y`, `w`, `h` are the glyph's cell in the sheet.
- **`left` and `right` are the first and last columns of the cell that the glyph's contour
  reaches**, not counting the drop shadow. The game uses them to set glyphs close together without
  their black lines colliding.
- **`advance` is how far to move right after drawing the glyph** before starting the next one,
  measured so that two digits side by side have their contours **just touching or overlapping by a
  few pixels**. Heavy inked numerals read best set tight, with the black lines merging into one
  outline round the whole number.
- **Measure `left`, `right` and `advance` off the drawn pixels**, glyph by glyph. A `1` and a `0`
  do not take the same width, and one figure for all of them spaces the number badly.

### Naming

`figure-glyphs-<name>.png` for each of the eight sheets (`figure-glyphs-fire.png`,
`figure-glyphs-neutral.png`, and so on), plus `figure-glyphs.json`, plus
**`figure-glyphs-preview.png`**: every sheet stacked on one image over a **light slate-blue ground,
`#7EACE4`**, which is the table the glyphs are mostly drawn on. The preview is for looking at, not
for use, so it is the one file with a background.

Ten files. Give me a zipped file of everything so I can download

---

## Before you hand anything back

Check every sheet against these. A failure on any one means redraw rather than explain.

0. **Shrink the preview to a third of its size and look at it.** That is about the smallest the
   glyphs are ever drawn. **Every digit must still read, and every counter must still be open.** If
   an `8` has become a black blob, the contour is eating the counter; widen the glyph or open the
   counter, and do not thin the contour to fix it.
1. **Is the contour black and heavy on every glyph?** Roughly an eighth of the glyph's height, true
   black, with the hard drop shadow merged into it. If any color's glyphs look outlined in a dark
   version of their own color, that sheet is wrong.
2. **Is the fill two crisp bands on a flat core?** No gradient, no gloss, no texture and no glow. If
   it looks airbrushed, it is wrong.
3. **Lay the eight sheets side by side.** Every glyph in the same place, the same shape and the same
   size, with the light and dark bands in the same places. Only the fill color changes.
4. **Do all the digits share one height, one weight and one baseline?** Set `1`, `4` and `8` next to
   each other and look for the odd one out.
5. **Are `+`, `-`, `x` and `=` centered on the digits' middle height**, and is the `x` a
   multiplication sign rather than a letter?
6. **Is `neutral` truly gray?** Put it beside a pure gray swatch. Any warmth or coolness means redo.
7. **Is the background transparent** on the eight sheets, with no grid, no frame and nothing
   outside the glyphs? Only the preview has a ground.
8. **Does every glyph stay inside its own cell**, contour and shadow included?
