# Figure symbols art prompt — the punctuation and the math signs

A set of **display symbols** to go with a sheet of bold inked capitals and digits that already
exists: the punctuation and math signs a card game writes beside its words and numbers. Delivered
as one sprite sheet per color plus a JSON file saying where each glyph is. The letters themselves
are drawn from `figure_letters_art_prompt.MD`, and the currency mark from
`figure_vitae_art_prompt.MD`.

**Attach two images when using this prompt**: the finished preview of the letters sheet, which
every symbol matches, and the previous symbols sheet, whose `!`, `?`, `/`, `%`, `'`, `,` and `:` are
to be kept. These
symbols are set in the same line as those letters and digits — `DUEL!`, `+1 AP`, `-54`,
`(12 + 3) = 15`, `HUNTER'S MARK` — so a symbol that looks like it came from a different hand, has
a thinner outline, or is shorter than the letters beside it, is the whole failure.

**The game assembles every expression itself, glyph by glyph.** Draw each glyph once, on its own.

---

## The prompt

High-resolution digital illustration of a **set of individual glyphs**: punctuation and math signs
for a game interface, drawn as bold, inked, graphic lettering matching the attached letters. They
are **not** a typeface rendered in a program, and they are **not** pixel art.

### The contour is 12 pixels thick — on every symbol

**This is the rule the set is judged on first.** On the attached letters, the black outline is
**12 pixels thick**: measure it across the side of the `I`, the `H` or the `E` at mid-height, and
it is 12 pixels of solid black between the transparent background and the fill. **Every symbol on
this sheet has that same 12-pixel outline.**

- **A small symbol does not get a thinner outline.** A dot, a comma, a hyphen or an apostrophe is
  outlined with the same 12 pixels of black as a capital `M`. On a small symbol that means the black
  is a large share of what is drawn, and that is correct: it is what makes the symbols sit in a line
  of heavy lettering instead of looking like a lighter font dropped into it.
- **A thin outline — 3 or 4 pixels — is the failure this prompt exists to prevent.** An earlier
  attempt drew every symbol with a 3-pixel line beside letters with a 12-pixel one, and the symbols
  read as a different typeface.
- **The contour is true black**, or within a few points of it. Not a dark shade of the fill's
  color, not dark gray, not dark brown, and not dark red on the crimson sheet.
- **It has an inked, graphic-novel quality**: confident, slightly tapered brush strokes with crisp
  edges, like bold comic-book lettering for an action title.
- **The black also sits a little lower than the fill, as a hard drop**, exactly as on the letters:
  a solid black shadow offset down and slightly right by about 6 pixels, merged with the contour so
  the two read as one black shape. No blur and no soft shadow.

**The fill is flat and bright, with two hard-edged tone bands and nothing more.**

- The body of the glyph is the color's `core` value (see the tables below).
- A **lighter band across the upper part** of each stroke, in the color's `light` value, with a
  crisp edge rather than a gradient.
- A **darker band along the bottom** of each stroke, in the color's `dark` value, also hard-edged.
- **No gradients, no texture, no noise, no streaks, no sparkle, no glow, no gloss, no white
  highlight line, no bevel, no metal sheen and no glitter.**

Light comes from **above and slightly to the left**: the light band sits on top, the dark band and
the drop underneath.

**The strokes are as heavy as the letters' strokes, and no heavier.** The fill of each stroke is
given in pixels in the table below, taken off the letters' own uprights and crossbars.

### Where a symbol sits, measured off the letters

Every cell is 160 pixels square. **These are the attached letters' own measurements**, taken off
the `I`, the `H` and the `E`, and every symbol is placed and sized against them:

| On the letters                                  | Pixels                         |
| ----------------------------------------------- | ------------------------------ |
| the outline's thickness                         | **12**                         |
| top of a capital's black outline                | **23** down from the cell top  |
| top of a capital's fill                         | 35                             |
| bottom of a capital's fill — the baseline       | **135**                        |
| bottom of a capital's black outline             | **147**                        |
| bottom of the drop shadow                       | 152                            |
| middle of a capital's fill                      | **85**                         |
| fill of an upright stroke (the `I`)             | **26** wide                    |
| fill of a crossbar (the bars of the `E`)        | **18** tall                    |
| black between two crossbars (inside the `E`)    | 21                             |

**Nothing a glyph draws comes within 4 pixels of any edge of its cell** — not the contour, not the
drop shadow, not a tail and not the end of a bracket. The game cuts each cell out on its own, so
anything reaching the edge arrives cut flat.

### The glyphs, box by box

**Every glyph fits the box given for it.** A box is the glyph's size **with its black outline
included and its drop shadow left out**, and it is centered left-to-right in the cell. "Top 23,
bottom 147" means the glyph's black runs exactly as high and as low as a capital's. **A symbol
bigger than its box is the failure to look for**: an earlier attempt drew `+` and `-` 130 pixels
wide, wider than any letter, and a `.` that was a ball the size of a digit.

| Glyph | Box, outline included     | Where it sits                                   | Strokes                                        |
| ----- | ------------------------- | ----------------------------------------------- | ---------------------------------------------- |
| `+`   | 74 wide x 74 tall         | centered on 85                                  | each arm's fill 18, like the `E`'s bars        |
| `-`   | 74 wide x 42 tall         | centered on 85                                  | one bar, fill 18                               |
| `=`   | 74 wide x 81 tall         | centered on 85                                  | two bars, fill 18 each, 21 of black between    |
| `(`   | 48 wide x 124 tall        | top 23, bottom 147                              | fill 18 at the middle, thinning to 12 at ends  |
| `)`   | 48 wide x 124 tall        | top 23, bottom 147                              | the mirror of `(`                              |
| `.`   | 50 wide x 50 tall         | bottom 147                                      | a round dot, fill 26 across                    |
| `,`   | 50 wide x 64 tall         | the dot's bottom on 147, the tail below it      | the `.` with a short tail                      |
| `:`   | 50 wide x 116 tall        | bottom 147                                      | two `.` dots, 16 of ground between them        |
| `!`   | 50 wide x 124 tall        | top 23, bottom 147                              | a bar, fill 26 at the top, over a `.`          |
| `?`   | 80 wide x 124 tall        | top 23, bottom 147                              | a hook, fill 26, over a `.`                    |
| `'`   | 34 wide x 44 tall         | top 23                                          | a wedge, fill 18 at the top                    |
| `/`   | 74 wide x 124 tall        | top 23, bottom 147                              | fill 26                                        |
| `%`   | 96 wide x 124 tall        | top 23, bottom 147                              | slash fill 18, two small rings                 |
| `&`   | 100 wide x 124 tall       | top 23, **bottom 147, on the baseline**         | fill 18 to 26                                  |
| `x`   | 70 wide x 72 tall         | centered on 85                                  | fill 18                                        |
| `×`   | 62 wide x 62 tall         | centered on 85                                  | fill 18                                        |

What each one looks like:

- **`+`, `-` and `=` are plain and flat-ended**, the same weight as each other. The `-` is a minus
  sign, as in `-54`, and also a hyphen between words.
- **`(` and `)` are thin arcs, like a slice cut from the edge of a large circle** — shallowly
  curved, tall and narrow, tapering toward both ends. **They are not a `C`**: a bracket that curls
  round into a thick hook reads as a letter, and an earlier attempt made `(12 + 3)` read as
  `C12 + 3Ↄ`.
- **`.` is a round dot**; **`,` is that dot with a short tail** curling down and left; **`:`** is two
  dots stacked.
- **`!` is a bar tapering toward the bottom, over the dot**, and **`?` is a hook over the dot**, with
  a counter open enough to survive shrinking.
- **`'` is a short, slightly tapered wedge** hanging down from the top line, used inside words such
  as `HUNTER'S`.
- **`/` leans right**; **`%` is a slash with a small ring at the upper left and another at the lower
  right**, each ring's opening kept open; **`&` is a heavy ampersand** sitting on the baseline like
  a capital.
- **`x` and `×` do the same job and must be two different ideas**, so they can be compared:
  - **`x` is brushy and letter-like**: two strokes crossing, each tapering at its ends with cut,
    angled terminals, as if written with the same brush as the capitals. Slightly curved rather than
    ruler-straight.
  - **`×` is geometric**: two perfectly straight bars of even thickness, crossing at right angles,
    with **squared-off ends**, like a `+` turned 45 degrees. No taper and no curve.
  - An earlier attempt drew the same rounded X at two sizes. Two sizes of one drawing is the
    failure.

**Seven of these were right in the attempt before this one** — `!`, `?`, `/`, `%`, `'`, `,` and `:` —
and that sheet is attached as well. Keep those seven close to how they were, fitted to the boxes
above; the work is `+`, `-`, `=`, `(`, `)`, `.`, `&`, `x` and `×`.

Sixteen glyphs, in this order:

| Row | Glyphs                                |
| --- | ------------------------------------- |
| 1   | `+` `-` `=` `(` `)` `.` `,` `:` `!`   |
| 2   | `?` `'` `/` `%` `&` `x` `×`           |

Nothing else. No letters, no digits, no currency sign, no plate or badge behind any of them.

### The sheet

**One PNG per color, all identical in layout.** Transparent background, PNG with an alpha channel.

- **A grid of 9 columns by 2 rows of equal cells, each cell 160 wide by 160 tall**, so every sheet
  is **exactly 1440 x 320**. The glyphs go in the order in the table above, left to right and top to
  bottom; the last two cells of row 2 stay empty.
- **The glyph is centered left-to-right in its cell**, and placed up and down against the lines
  above.
- **Everything outside the glyphs is fully transparent.** No ground, no grid lines, no frame round
  a cell, no frame round the sheet, no labels, no guide marks and no dark margin.

### The colors

Every color is the **same sixteen drawings, recolored**. The contour, the drop shadow, the shape of
every glyph, where each light band and dark band falls: all identical. Only the three fill values
change. **Draw the set once, then recolor it for every row below.**

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

#### Four more

| Sheet name | Color                 | dark      | core      | light     |
| ---------- | --------------------- | --------- | --------- | --------- |
| `attack`   | red                   | `#7A1E1E` | `#BA3434` | `#EE7A6A` |
| `relic`    | pink                  | `#9A2F68` | `#EB69AA` | `#FFB8DA` |
| `vitae`    | crimson               | `#5E0C16` | `#A81A2A` | `#E0525E` |
| `neutral`  | white to gray, no hue | `#A6A6A6` | `#E8E8E8` | `#FFFFFF` |

**`neutral` is used both as it is and recolored by the game**, which multiplies its fill by another
color at run time. That is why it has to be **pure gray, with no hue at all**. The black contour and
the drop shadow come through the multiply as black.

### The JSON

One file, `figure-symbols.json`, in the same shape as the letters sheet's:

```json
{
  "cellWidth": 160,
  "cellHeight": 160,
  "columns": 9,
  "rows": 2,
  "baseline": 136,
  "digitHeight": 112,
  "glyphs": {
    "+": { "x": 0, "y": 0, "w": 160, "h": 160, "left": 44, "right": 116, "advance": 66 },
    "...": {}
  },
  "sheets": {
    "fire": {
      "image": "figure-symbols-fire.png",
      "ramp": { "dark": "#9E2F13", "core": "#F06A24", "light": "#FFB24A" }
    },
    "...": {}
  }
}
```

The figures in that example are placeholders; measure every one off the drawn pixels.

- **Each glyph's key is the character itself**, exactly as in the table above. Two to get right:
  `"x"` is the lowercase x (U+0078) and `"×"` is the multiplication sign (U+00D7). The file is
  UTF-8.
- `x`, `y`, `w`, `h` are the glyph's cell in the sheet.
- **`left` and `right` are the first and last columns of the cell that the glyph's contour
  reaches**, not counting the drop shadow.
- **`advance` is how far to move right after drawing the glyph**, measured so that the glyph and a
  neighbouring capital or digit have their contours **just touching or overlapping by a few
  pixels**. It is never much more or much less than the distance from `left` to `right`.
- **Measure all three glyph by glyph.** A `.` and a `%` are nowhere near the same width, so the gap
  between `advance` and `right - left` differs from glyph to glyph. One figure copied onto every
  glyph is a sign the file was not measured.

### Naming

`figure-symbols-<name>.png` for each of the nine sheets (`figure-symbols-fire.png`,
`figure-symbols-neutral.png`, and so on), plus `figure-symbols.json`, plus
**`figure-symbols-preview.png`**: every sheet stacked on one image over a **light slate-blue
ground, `#7EACE4`**. Under the stacked sheets, the preview also sets these lines in the `neutral`
sheet, **using the attached letters for the letters and digits, each glyph placed at the height it
has in its own cell** so that the baselines line up:

- `DUEL!  +1 AP  -54`
- `(12 + 3) = 15`
- `12 x 3 = 36  12 × 3 = 36  X x ×`
- `HUNTER'S MARK  EBB & FLOW`
- `ABANDON RUN, SEED: 0019QS`
- `MUSIC 75%  3/5  SELL FOR 3?`
- `H!H  H?H  H/H  H%H  H&H  H(H)H`
- `H+H  H-H  H=H  H.H  HxH  H×H`

The preview is for looking at, not for use, so it is the one file with a background.

Eleven files. Give me a zipped file of everything so I can download

---

## Before you hand anything back

Check every sheet against these. A failure on any one means redraw rather than explain.

0. **Measure every glyph against its box.** The black of each, outline included and drop shadow
   left out, fits the box in the table, and sits where the table says. **A `+` or a `-` wider than
   74 pixels, or a `.` wider than 50, means redraw.**
1. **Measure the outline.** Take the `!`, the `/` and the `(` and count the black pixels across
   their side at mid-height. It must be about 12, the same as across the side of the `I` in the
   attached letters. **3 or 4 means redraw the whole set.**
2. **Read the last preview line.** Each symbol between two `H`s starts and ends at the same heights
   as the `H`s, and its black line is as heavy as theirs. A symbol visibly shorter or lighter than
   the `H` beside it is wrong.
3. **Read `(12 + 3) = 15`.** The brackets are thin arcs that read as brackets, not as a `C` and a
   backwards `C`, and the `+` and `=` are no bigger than the digits beside them.
4. **Read `DUEL!` and `HUNTER'S MARK`.** The `!` and the apostrophe look like part of the same
   lettering as the word, not like a lighter font.
5. **Check the edge of every cell, on every sheet.** Nothing within 4 pixels of any edge.
6. **Is the fill two crisp bands on a flat core?** No streaks, no sparkle, no white highlight line,
   no gradient and no gloss.
7. **Put `x` and `×` side by side.** One brushy and tapered, one geometric with squared ends — two
   ideas, not one drawing at two sizes. Both clearly smaller than the capital `X`.
8. **Read `-54` and `75%`.** No gap after the minus, and the percent sign does not run into the
   digits beside it.
9. **Lay the nine sheets side by side.** Every glyph in the same place, the same shape and the same
   size; only the fill color changes. Put `neutral` beside a pure gray swatch: any warmth or
   coolness means redo.
