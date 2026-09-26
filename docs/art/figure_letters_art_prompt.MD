# Figure letters art prompt — the capitals and the digits

A set of **display glyphs**: the capital alphabet and the ten digits, in a bold inked style,
delivered as one sprite sheet per color plus a JSON file saying where each glyph is. The
punctuation and math symbols that go with them are a separate sheet with its own prompt,
`figure_symbols_art_prompt.MD`, drawn to match this one.

**These are not text in a paragraph.** They are the short, loud words and figures a card game
writes over its table and on its controls: a `-54` flying into a health bar, a price like
`6 VITAE`, a button reading `DUEL!`, a heading reading `THE LEDGER`, a callout reading `MISS`. They
are one to about twenty characters long, often moving, and they have to be **read at a glance**
against a busy table. Nothing drawn with this set is ever a sentence.

**The game assembles every word and every number itself, glyph by glyph.** Draw each glyph once,
on its own; never draw a whole word or number as one picture.

---

## The prompt

High-resolution digital illustration of a **set of individual glyphs**: display capitals and
numerals for a game interface, drawn as bold, inked, graphic lettering. They are **not** a
typeface rendered in a program, and they are **not** pixel art.

### The look

**A heavy, hand-inked black contour around every glyph, and a vivid, saturated fill inside it.**
This is the whole of the style, and the contour is the part that matters most.

- **The contour is thick.** It is **about an eighth of the glyph's height** all the way round: at
  a glyph 100 pixels tall that is a 12-pixel line. That is much heavier than looks comfortable at
  full size, and it is meant to be. The glyphs are used much smaller than they are delivered, and
  sit on light and dark grounds, on tables of every color, on buttons and over pictures. **The
  black line is what carries them on all of those**, so a thin or dark-colored outline defeats the
  point of the set.
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

**Letters and digits are one family.** A word next to a number, such as `6 VITAE` or `+1 AP`, must
read as one piece of lettering, so the capitals and the digits share one height, one stroke
weight, one contour and one hand.

- **Heavy, chunky display capitals and numerals.** Bold and fairly wide, with **generous
  counters**, the enclosed or nearly enclosed spaces in `A` `B` `D` `O` `P` `Q` `R` `0` `4` `6` `8`
  `9` and the openings of `C` `G` `S`, so the black contour does not close them up when the glyph
  is small. The letters have more counters than the digits and are where this fails first.
- **Capitals only.** There are no lowercase letters in the set.
- **Upright**, not italic. Glyphs are laid next to each other by the game and centered on points,
  and a slant makes a word look like it is sliding.
- **Slightly irregular, in a lively rather than a sloppy way.** A tiny amount of hand-drawn
  character is good, such as a stroke that swells a little or a corner that is a touch sharper
  than its twin. But every glyph must be instantly readable, and **all the capitals and digits must
  share one height and one weight**. An `I` or a `1` that looks lighter than an `M` or an `8` makes
  a word look misspelled.
- **No serifs, no swashes, no decorative terminals, no 3D extrusion drawn as a visible side.** The
  drop shadow is the only depth.
- **Width follows the letter.** `M` and `W` are wide, `I` is narrow, and most sit in between.
  Do not stretch narrow letters to a common width.
- **`I` is a plain upright bar with no crossbars**, heavy enough to hold its own beside an `M`.
  **`1` has a flag and a base**, so that the two cannot be confused: `I` has no flag, `1` always
  does.
- **`O` and `0` must be told apart**: `0` is noticeably narrower and more oval than `O`, which is
  close to round. **`S` and `5`, `B` and `8`, `Z` and `2`, `G` and `6`** are the other pairs to keep
  distinct at small size.
- **`X` is a full-height capital letter** sitting on the baseline like every other letter. A small
  multiplication x is drawn on the symbols sheet, not here.
- **`Q` has a clear tail** that crosses the bottom right of the bowl. It stays inside the cell,
  with the same margin as everything else.

### The glyphs

Thirty-six glyphs, in this order:

| Row | Glyphs                              |
| --- | ----------------------------------- |
| 1   | `A` `B` `C` `D` `E` `F` `G` `H` `I` |
| 2   | `J` `K` `L` `M` `N` `O` `P` `Q` `R` |
| 3   | `S` `T` `U` `V` `W` `X` `Y` `Z` `0` |
| 4   | `1` `2` `3` `4` `5` `6` `7` `8` `9` |

Nothing else. No lowercase, no accented letters, no punctuation, no symbols, no plate or badge
behind any of them.

### The sheet

**One PNG per color, all identical in layout.** Transparent background, PNG with an alpha channel.

- **A grid of 9 columns by 4 rows of equal cells, each cell 160 wide by 160 tall**, so every sheet
  is **exactly 1440 x 640**, with every cell filled. The glyphs go in the order in the table above,
  left to right and top to bottom.
- **Every capital and every digit is the same height, about 112 pixels from its top to its
  baseline**, and every glyph in the sheet sits on **one baseline, 136 pixels down from the top of
  its cell**.
- **Nothing a glyph draws comes within 4 pixels of any edge of its cell** — not the contour, not
  the drop shadow, not the tail of `Q`, and not the bottom of a round letter such as `O`, `S`, `U`
  or `0`, which dips slightly below the baseline. The game cuts each cell out on its own, so a
  shadow that reaches the edge arrives cut flat, and anything that crosses into the next cell
  arrives as a stray mark on another glyph. If a glyph does not fit, the glyph is too big, not the
  cell too small.
- **The glyph is centered left-to-right in its cell.**
- **Everything outside the glyphs is fully transparent.** No ground, no grid lines, no frame round
  a cell, no frame round the sheet, no labels, no guide marks and no dark margin. The cells are an
  arrangement, not a drawing.

### The colors

Every color is the **same thirty-six drawings, recolored**. The contour, the drop shadow, the
shape of every glyph, where each light band and dark band falls: all identical. Only the three
fill values change. **Draw the set once, then recolor it for every row below.** Do not redraw per
color; two colors of one `R` that differ in shape is the defect this set fails on.

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

**`neutral` is used both as it is and recolored by the game**, which multiplies its fill by
another color at run time. That is why it has to be **pure gray, with no hue at all**: not warm,
not cool, not cream, not blue-white. Any tint in it shows up in every color made from it. The black
contour and the drop shadow come through the multiply as black, which is what keeps a recolored
glyph looking like the others.

### The JSON

One file, `figure-letters.json`, describing all the sheets:

```json
{
  "cellWidth": 160,
  "cellHeight": 160,
  "columns": 9,
  "rows": 4,
  "baseline": 136,
  "digitHeight": 112,
  "spaceAdvance": 48,
  "glyphs": {
    "A": { "x": 0, "y": 0, "w": 160, "h": 160, "left": 20, "right": 140, "advance": 112 },
    "...": {}
  },
  "sheets": {
    "fire": {
      "image": "figure-letters-fire.png",
      "ramp": { "dark": "#9E2F13", "core": "#F06A24", "light": "#FFB24A" }
    },
    "...": {}
  }
}
```

The figures in that example are placeholders; measure every one off the drawn pixels.

- **Each glyph's key is the character itself**: `"A"`, `"0"`, and so on. The file is UTF-8.
- `x`, `y`, `w`, `h` are the glyph's cell in the sheet.
- **`left` and `right` are the first and last columns of the cell that the glyph's contour
  reaches**, not counting the drop shadow.
- **`advance` is how far to move right after drawing the glyph** before starting the next one,
  measured so that two glyphs side by side have their contours **just touching or overlapping by a
  few pixels**. Heavy inked lettering reads best set tight, with the black lines merging into one
  outline round the whole word.
- **`spaceAdvance` is the width of a space between words**, in sheet pixels: about 40% of a
  capital's height, wide enough that `6 VITAE` reads as two words and no wider.

### Naming

`figure-letters-<name>.png` for each of the nine sheets (`figure-letters-fire.png`,
`figure-letters-neutral.png`, and so on), plus `figure-letters.json`, plus
**`figure-letters-preview.png`**: every sheet stacked on one image over a **light slate-blue
ground, `#7EACE4`**, which is the table the glyphs are mostly drawn on. Under the stacked sheets,
the preview also sets these lines in the `neutral` sheet, spaced by the JSON, so the spacing can be
judged as words rather than as cells:

- `6 VITAE`
- `DUEL  1 AP  54`
- `THE LEDGER  ABANDON RUN`
- `SEED 0019QS`
- `OIL 010  SB85  ZG26  VXV`

The preview is for looking at, not for use, so it is the one file with a background.

Eleven files. Give me a zipped file of everything so I can download

---

## Before you hand anything back

Check every sheet against these. A failure on any one means redraw rather than explain.

0. **Shrink the preview to a third of its size and look at it.** That is about the smallest the
   glyphs are ever drawn. **Every letter and digit must still read, and every counter must still be
   open.** If an `A`, `B`, `R` or `8` has become a black blob, the contour is eating the counter;
   widen the glyph or open the counter, and do not thin the contour to fix it.
1. **Read the last preview line at a third of its size.** `O` against `0`, `I` against `1`, `S`
   against `5`, `B` against `8`, `Z` against `2` and `G` against `6` must each be told apart
   without squinting.
2. **Is the contour black and heavy on every glyph?** Roughly an eighth of the glyph's height, true
   black, with the hard drop shadow merged into it. If any color's glyphs look outlined in a dark
   version of their own color, that sheet is wrong.
3. **Is the fill two crisp bands on a flat core?** No gradient, no gloss, no texture and no glow. If
   it looks airbrushed, it is wrong.
4. **Lay the nine sheets side by side.** Every glyph in the same place, the same shape and the same
   size, with the light and dark bands in the same places. Only the fill color changes.
5. **Do all the capitals and digits share one height, one weight and one baseline?** Set `I`, `M`,
   `1` and `8` next to each other and look for the odd one out. Then read `6 VITAE`: the digit and
   the word must look like one piece of lettering.
6. **Is `neutral` truly gray?** Put it beside a pure gray swatch. Any warmth or coolness means redo.
7. **Is the background transparent** on the nine sheets, with no grid, no frame and nothing
   outside the glyphs? Only the preview has a ground.
8. **Check the edge of every cell, on every sheet.** Nothing within 4 pixels of any edge: look
   hardest at the bottoms of `O`, `Q`, `S`, `U` and `0`. A shadow cut flat along a cell's bottom
   edge, or a stray mark at the top of a cell that belongs to the glyph above it, means redraw.
