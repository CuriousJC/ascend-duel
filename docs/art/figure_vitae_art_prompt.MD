# Figure vitae art prompt — the currency mark

A single glyph: **the vitae mark**, the currency sign of a card game, written after a number the
way `$` or `€` marks money — `6 ¤`, `+4 ¤`, `SELL FOR 3 ¤`. It is drawn in the same bold inked
style as a sheet of capitals and digits that already exists (`figure_letters_art_prompt.MD`) and
sits in a line with them.

**It is made in two rounds.** Round 1 draws **six different candidates in one color** so they can
be compared and one chosen. Round 2 takes the chosen one and delivers it in every color, as a sheet
the game can use. Use the section for the round being run.

**Attach the finished preview of the letters sheet to either round, and match it.**

---

## The mark

**It is a stylized capital V**, and anyone must recognize it as a V at a glance, the way `€`, `£`
and `¥` are obviously letters and obviously not plain letters.

- **Both upper tips of the V end in an open curl**, calligraphic in feel: the end of each arm turns
  outward and down into a short hook or flick, as if the V were written with a broad pen that
  flicked at the end of each stroke.
- **The curls are open. There is no enclosed hole anywhere in the mark.** A curl turns and stops;
  it never closes back on itself into a ring, and no background shows through the inside of it.
  Every earlier attempt drew each tip as a closed loop with a hole in it, and at play size the two
  holes read as a pair of eyes and the whole mark as a face or a heart. **No holes, no rings, no
  loops, no eyes.**
- **It is not a heart.** The top of the V is two separate tips with clear background between them
  and down into the V's opening. The arms never swell and merge into a rounded top, and the
  silhouette has a deep notch at the top center, as a V does.
- **The curl is part of the stroke, not something attached to it.** The arm thins, turns and ends
  in one continuous line. A ball, a bolt or a ring sitting on the end of a straight arm is not a
  curl.
- **The strokes have calligraphic thick-and-thin contrast**: the arms are heaviest near the point
  and thin as they rise into the curls, so the mark reads as written rather than as a letter from
  the alphabet beside it.
- **The point at the bottom is sharp** and sits on the baseline.
- **It must not be mistaken for the plain letter `V`** in the attached letters. The word `VITAE`
  and the mark often appear on one screen, and the curls and the stroke contrast are what tell them
  apart.
- **Nothing else**: no coin, no circle round it, no badge, no crossbar, no sparkles or rays.

## The look

The same style as the attached letters, exactly.

- **A black contour 12 pixels thick — the same as the letters', and no thicker.** Measure it across
  the side of their `I` at mid-height: 12 pixels. An earlier attempt drew the mark with an outline
  close to twice that, which made it heavier than every letter beside it. **It wraps the curls at
  the same 12 pixels.** Because the curls are open, a 12-pixel outline has nothing to fill in.
- **True black**, or within a few points of it. Not dark red.
- **A hard black drop shadow**, about 5 pixels down and slightly right, merged with the contour.
  No blur.
- **A flat fill in the color's `core`, with a hard-edged `light` band along the top of each stroke
  and a hard-edged `dark` band along the bottom.** No gradients, no texture, no streaks, no
  sparkle, no gloss, no white highlight line, no glow and no bevel.
- **Light from above and slightly to the left.**

## Where it sits

The mark is drawn in a **160 x 160 cell**, against these measurements of the attached letters:

| On the letters                              | Pixels                        |
| ------------------------------------------- | ----------------------------- |
| outline thickness                           | **12**                        |
| top of a capital's black outline            | **23** down from the cell top |
| bottom of a capital's black outline         | **147**                       |
| bottom of the drop shadow                   | 152                           |
| fill of an upright stroke (the `I`)         | 26 wide                       |

**The mark's black, outline included and drop shadow left out, fits a box about 104 wide by 124
tall**: its highest point at 23 and its point at 147, exactly as high and as low as a capital, and
centered left-to-right. **The fill of each arm is about 26 wide near the point, thinning to about
14 at the curls.** Nothing it draws comes within 4 pixels of any edge of its cell.

---

## Round 1 — six candidates

**One PNG, `figure-vitae-candidates.png`, 480 x 320**: a grid of **3 columns by 2 rows of
160 x 160 cells**, transparent background, one candidate per cell, all six in the **crimson**
color below.

**Six different ideas, not six copies.** Every candidate follows the rules above — open curls, no
holes, not a heart — and they differ in how the tips end:

1. **an outward hook** — each tip turns outward and down in a short hook, less than half a turn;
2. **a brush flick** — each tip lifts off in a quick upward-and-outward taper, the stroke thinning to
   a point, as a brush does leaving the paper;
3. **a teardrop terminal** — each tip ends in a solid, filled teardrop that leans outward, the way a
   calligraphic `f` or `J` ends: a swelling, never a hole;
4. **one curl and one plain tip** — the left arm ends in an outward hook and the right arm ends
   straight and sharp, for a hand-written asymmetry;
5. **a lead-in stroke** — the left arm begins with a small curved entry stroke coming in from the
   left, as when a V is written in one motion, and the right arm ends in a short outward flick;
6. **the freest one** — whatever reads most clearly as a currency mark made from a V, within the
   rules.

Set side by side, **the six must be told apart by their silhouettes alone.** Six drawings of one
shape with small differences is the failure: an earlier round drew the same looped V six times.

**Also deliver `figure-vitae-candidates-preview.png`**: the six candidates over a **light
slate-blue ground, `#7EACE4`**, and under each one, using the attached letters, the line
`6 ¤  6 VITAE`, with the candidate in the place of the `¤`, so each is judged beside a number and
beside the word it must not be confused with. Then the whole preview again at a third of its size.

Two files. Give me a zipped file so I can download.

### Before you hand Round 1 back

0. **At a third of the size, does every candidate still read as a V** — and not as a heart, a face
   or a pair of eyes?
1. **Is every candidate clearly different from the plain `V` in `VITAE` beside it?**
2. **Is there no hole anywhere** — no closed loop, no ring, no background showing through a curl?
3. **Is the outline 12 pixels on every candidate** — measured, not guessed — true black and not
   dark red? Much more than 12 is as wrong as much less.
4. **Are the six genuinely different silhouettes?**

---

## Round 2 — the chosen mark, in every color

**Attach the chosen candidate as well as the letters preview.** Redraw that candidate cleanly, the
same shape, following every rule above, and deliver it in every color.

**One PNG per color, each exactly 160 x 160** — a single cell — transparent background. Every color
is **the same drawing, recolored**: only the three fill values change.

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

**`vitae` is the mark's own color** and the one it is most often drawn in; the round 1 candidates
are drawn in it. **`neutral` must be pure gray**, with no hue at all, because the game multiplies it
by other colors at run time.

**One JSON file, `figure-vitae.json`:**

```json
{
  "cellWidth": 160,
  "cellHeight": 160,
  "columns": 1,
  "rows": 1,
  "baseline": 136,
  "digitHeight": 112,
  "glyphs": {
    "¤": { "x": 0, "y": 0, "w": 160, "h": 160, "left": 20, "right": 140, "advance": 112 }
  },
  "sheets": {
    "vitae": {
      "image": "figure-vitae-vitae.png",
      "ramp": { "dark": "#5E0C16", "core": "#A81A2A", "light": "#E0525E" }
    },
    "...": {}
  }
}
```

- **The glyph's key is `"¤"`, the currency sign (U+00A4).** The file is UTF-8.
- **`left` and `right` are the first and last columns the contour reaches**, not counting the drop
  shadow, and **`advance` is how far to move right after it**, measured so its contour just touches
  a neighbouring capital's. Measure all three off the drawn pixels; the figures above are
  placeholders.

**Naming:** `figure-vitae-<name>.png` for each of the nine colors, plus `figure-vitae.json`, plus
`figure-vitae-preview.png`: all nine over `#7EACE4`, and under them, using the attached letters,
the lines `6 ¤`, `+4 ¤` and `SELL FOR 3 ¤` in `vitae`, and `6 ¤  6 VITAE` in `neutral`.

Eleven files. Give me a zipped file of everything so I can download.

### Before you hand Round 2 back

0. **Is it the chosen candidate**, the same shape, cleanly redrawn?
1. **Are the nine files the same drawing**, only the fill changing, and is `neutral` pure gray?
2. **Is the outline 12 pixels, true black, on every color**, and is there still no hole anywhere?
3. **Is nothing within 4 pixels of the cell's edge?**
