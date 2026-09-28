# Prose glyphs art prompt — the reading set

A set of **text glyphs** for a card game's longer text: tooltips, the lines on a character card,
narration, dialog boxes, a tutorial's speech bubble, a log of what happened. Upper and lower case,
digits, and punctuation, delivered as **one sprite sheet** plus a JSON file saying where each glyph
is and how far apart to set it.

**This is the reading set, and it has a louder sibling.** The game already has a display set:
heavy, chunky capitals with a thick black outline and a drop shadow, used for short loud words like
`DUEL!` and `-54`. This set is its quieter relative, drawn in the same hand so the two sit on one
screen as one family — but made to be **read in sentences at small size**, so it is lighter, it has
lowercase, and it carries no shadow.

**The game sets every line itself, glyph by glyph**, and it colors the glyphs itself: it multiplies
the white fill by a color, or by a material texture, at the moment it draws. So the sheet is drawn
**once, in white**, and never in color. Draw each glyph once, on its own; never draw a word.

---

## The prompt

High-resolution digital illustration of a **set of individual glyphs**: a clean, readable text
alphabet for a game interface, in upper and lower case with digits and punctuation, drawn as
inked graphic lettering. It is **not** a typeface rendered in a program, and it is **not** pixel
art.

### The look

**A white letter inside a thin, crisp black outline.** Nothing else.

- **The fill is pure flat white, `#FFFFFF`, and nothing but white.** No tone bands, no highlight,
  no shading, no gradient, no texture, no gloss. This matters more than anything else in the
  prompt: the game multiplies the fill by other colors and by stone and metal textures, and **any
  shading in the fill shows up in every one of them.** A fill that is off-white, warm or cool, or
  darker toward the bottom is the one defect that spoils the whole set.
- **The outline is true black** (`#000000`, or within a few points of it) and **thin: about a
  twelfth of the capital height** all the way round. At a capital 72 pixels tall that is a
  6-pixel line. It keeps its width through every curve and into every counter.
- **It has an inked, confident quality**: crisp edges, like careful comic-book lettering in a
  caption box. Not a uniform vector stroke, not sketchy or wobbly.
- **No drop shadow, no 3D, no extrusion, no glow.** The outline is the only black.

### The letterforms

**A clear, sturdy, slightly condensed sans-serif hand**, the kind a comic letterer uses for
dialogue. Medium-bold: the white stroke inside the outline is **about a seventh of the capital
height** wide, so a letter reads as a solid shape and not as a line drawing.

- **Built for reading small.** The set is used with a capital as small as **10 pixels tall**, so
  every decision favors legibility at that size: **a tall x-height**, lowercase about **70% of the
  capital height**; **open counters and apertures** (`a` `e` `c` `s` `g`); and generous spacing
  inside letters so the outline does not close them up.
- **Upright**, not italic. **No serifs**, except where noted for telling letters apart.
- **One height, one weight, one hand.** All capitals share one height, all lowercase letters one
  x-height, all digits the capital height. Every glyph has the same stroke weight and the same
  outline.
- **Width follows the letter.** `m` and `M` and `W` are wide, `i` `l` `j` `t` and `I` are narrow.
  Do not force a common width.
- **Keep the confusable ones apart**, because a reader meets them in a sentence at 10 pixels:
  - `I` (capital i) has short **crossbars top and bottom**; `l` (lowercase L) is a **plain bar with
    a small curve to the right at its foot**; `1` has a **flag and a base**. The three must never
    look alike.
  - `O` is round; `0` is narrower and oval.
  - `rn` must not read as `m`; `cl` must not read as `d`; `vv` must not read as `w`.
  - `a` is **two-storey** (the `a` with a hook over the bowl) and `g` is **single-storey**.
- **Ascenders** (`b d f h k l`) rise a little above the capital height. **Descenders**
  (`g j p q y`) drop about a third of the capital height below the baseline.
- **Punctuation is sized to sit in text**: the period and comma are round dots about the width of a
  lowercase stroke plus its outline, sitting on the baseline; the comma and the semicolon's tail
  drop below it. **`-` is a short hyphen, `–` an en dash about as wide as an `n`, and `—` an em dash
  as wide as an `M`**, all at the height of the middle of the lowercase. `×` is a small
  multiplication sign at lowercase height, clearly not an `x`. `·` is a single raised dot at
  lowercase-middle height. `…` is three periods set close together as one glyph. `'` and `"` are
  straight upright ticks.

### The glyphs

Ninety-two glyphs, in this order, ten to a row:

| Row | Glyphs                                        |
| --- | --------------------------------------------- |
| 1   | `A` `B` `C` `D` `E` `F` `G` `H` `I` `J`       |
| 2   | `K` `L` `M` `N` `O` `P` `Q` `R` `S` `T`       |
| 3   | `U` `V` `W` `X` `Y` `Z` `0` `1` `2` `3`       |
| 4   | `4` `5` `6` `7` `8` `9` `a` `b` `c` `d`       |
| 5   | `e` `f` `g` `h` `i` `j` `k` `l` `m` `n`       |
| 6   | `o` `p` `q` `r` `s` `t` `u` `v` `w` `x`       |
| 7   | `y` `z` `.` `,` `;` `:` `!` `?` `'` `"`       |
| 8   | `(` `)` `[` `]` `<` `>` `/` `\` `-` `–`       |
| 9   | `—` `_` `+` `=` `×` `*` `&` `%` `#` `\|`      |
| 10  | `·` `…`                                       |

In row 8, `\` is the backslash; in row 9, the last glyph is the vertical bar `|` (written
`\|` above only so the table displays). The last row holds two glyphs; the rest of it is empty. Nothing else: no accented letters, no
curly quotes, no symbols beyond these.

### The sheet

**One PNG, `prose-glyphs.png`.** Transparent background, PNG with an alpha channel.

- **A grid of 10 columns by 10 rows of equal cells, each cell 96 wide by 128 tall**, so the sheet
  is **exactly 960 x 1280**. The glyphs go in the order in the table above, left to right and top to
  bottom; the eight cells after `…` are empty and fully transparent.
- **Every glyph sits on one baseline, 100 pixels down from the top of its cell.**
- **Capitals and digits are 72 pixels tall** from baseline to top, outline included. **Lowercase
  x-height is about 50 pixels.** Ascenders reach about 76 pixels above the baseline; descenders
  reach about 22 pixels below it.
- **Nothing a glyph draws comes within 4 pixels of any edge of its cell** — not the outline, not a
  descender, not the top of a bracket. The game cuts each cell out on its own, so anything that
  touches an edge arrives cut flat, and anything that crosses into the next cell arrives as a stray
  mark on another glyph. If a glyph does not fit, the glyph is too big, not the cell too small.
- **The glyph is centered left-to-right in its cell.**
- **Everything outside the glyphs is fully transparent.** No ground, no grid lines, no frame, no
  labels, no guide marks.

### The JSON

One file, `prose-glyphs.json`:

```json
{
  "cellWidth": 96,
  "cellHeight": 128,
  "columns": 10,
  "rows": 10,
  "baseline": 100,
  "capHeight": 72,
  "xHeight": 50,
  "lineHeight": 118,
  "spaceAdvance": 26,
  "glyphs": {
    "A": { "x": 0, "y": 0, "w": 96, "h": 128, "left": 14, "right": 82, "advance": 64 },
    "...": {}
  },
  "sheets": {
    "prose": { "image": "prose-glyphs.png" }
  }
}
```

The figures in that example are placeholders; **measure every one off the drawn pixels.**

- **Each glyph's key is the character itself**: `"A"`, `"a"`, `";"`, `"—"`, `"…"`. The file is
  UTF-8, and `"` and `\` are escaped as JSON requires.
- `x`, `y`, `w`, `h` are the glyph's cell in the sheet.
- **`left` and `right` are the first and last columns of the cell that the glyph's outline
  reaches.**
- **`advance` is how far to move right after drawing the glyph** before starting the next one,
  measured so that two glyphs side by side **leave a small, even gap between their outlines** —
  about the outline's own width. This is text, set for reading: the outlines do not touch.
- **`spaceAdvance` is the width of a space between words**: about a third of the capital height.
- **`lineHeight` is the distance from one baseline to the next** in a paragraph, set so that a
  descender on one line clears an ascender on the line below with room to spare.
- **`capHeight` and `xHeight`** are the measured heights of `H` and `x`.

### Naming

`prose-glyphs.png` and `prose-glyphs.json`, plus **`prose-glyphs-preview.png`**: the paragraph
below set with the JSON's spacing, **three times**, once on each of these grounds, at the sheet's
own size and again at a third of it:

1. **Dark panel, `#1E1E26`**: the glyphs as they are, white with black outline.
2. **Light card, `#F2EFE8`**: every glyph **multiplied by `#222026`**, so it reads as near-black text.
3. **Slate blue, `#7EACE4`**: the glyphs as they are.

The paragraph:

```
Every lightning card is dealt as an ice card; a shield eats one hit.
Round 3 of 5 — 12 vitae (+4). Deal 2× DMG to BURNING enemies…
Il1 O0 rn/m cl/d vv/w "Quoted," [bracketed] <tag> 50% & #7 · a_b \ c|d
```

The preview is for looking at, not for use, so it is the one file with a background.

Three files. Give me a single zip file of everything so I can download it all at once — never a link per file.

---

## Before you hand anything back

A failure on any one means redraw rather than explain.

1. **Is the fill pure white** in every glyph? Sample the middle of an `O`, an `m` and a `—`: each
   must read `#FFFFFF`. Any tint, band or shading means redo.
2. **Read the preview at a third of its size**, on all three grounds. Every word must read without
   squinting. If an `a`, `e` or `g` has closed up, open the counter; do not thin the outline to fix
   it.
3. **Read the fourth line at a third of its size.** `I`, `l` and `1` are three different shapes;
   `O` and `0` are told apart; `rn`, `cl` and `vv` do not become `m`, `d` and `w`.
4. **Is the outline black, thin and even** on every glyph, with no drop shadow anywhere?
5. **Do all capitals share one height, all lowercase one x-height, and every glyph one baseline and
   one weight?** Set `Hxlgy` side by side and look for the odd one out.
6. **Check the edge of every cell.** Nothing within 4 pixels of any edge: look hardest at the
   descenders, the brackets and the tops of `b d f h k l`.
7. **Is the background transparent**, with the eight empty cells fully empty?
