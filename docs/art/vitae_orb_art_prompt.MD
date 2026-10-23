# Vitae orb art prompt

**An orb of blood**: the currency of a dark-fantasy card game, drawn as a single object rather than
as a sign. It is seen **on its own**, large on a reward screen and small beside a number on a card,
and **inline**, written after a number in a line of bold lettering the way `$` or `€` marks money
(`12 ●`, `+4 ●`, `SELL FOR 3 ●`) — so it is drawn **twice, as the same orb**: once with full detail,
which the game shrinks for the card, and once in the lettering's own inked style.

Two files. **For the inline one, attach the finished preview of the letters sheet and match it.**

---

## The prompt

High-resolution digital illustration of **one round orb of blood**, on a **fully transparent**
background, for a dark-fantasy card game. A game icon with real material to it — not a flat
symbol, not a photograph.

### What it is

- **A perfect sphere of liquid blood held in a thin, clear glass skin**, like a bauble or a
  potion bottle with no neck. The blood fills it completely. It is dark, rich and alive: deep
  crimson in the body, blackish red in the shadowed lower right, a brighter red where the light
  passes through it near the lower left edge.
- **Slow movement inside**: two or three soft, darker swirls drifting in the blood, as if it has
  just been shaken. Gentle and low-contrast — the orb reads as one round red thing first.
- **One crisp highlight**, a soft curved window of reflected light on the glass at the **upper
  left**, and a much smaller second glint just below it. Light comes from the upper left.
- **A thin near-black contour** around the whole sphere, about **a fiftieth of the canvas**, so it
  holds its edge on a pale background.
- **Nothing else**: no stand, no cap, no cork, no chain, no frame, no drips running off it, no
  splashes, no droplets around it, no gore, no veins, no skull, no eye, no heart shape, no rune or
  letter on it, no cast shadow, no glow, no halo, no sparkle around it.

### The two files

| File | What it is |
| ---- | ---------- |
| `vitae-orb` | **The orb**, shown at **up to about 160 pixels across** and also shrunk to **about 24 pixels across**. Full detail: the swirls, the glass skin, the light passing through the lower edge, the two highlights. **It has to survive the shrink**: the main highlight bold enough to stay visible at 24 pixels, the swirls soft enough that they do not average the body into a dark dot. At 24 pixels it must still read as a shiny red ball. |
| `vitae-orb-inline` | **The same orb as a glyph in the attached lettering**, drawn in a **160 x 160 cell** — see *The inline orb* below. |

**Put the two side by side and they are obviously the same orb**, one dressed in the lettering's
ink.

### Where it sits

- **Centered, filling about 90% of the canvas**, with a little clear space all round.
- It is seen against two pale grounds: a **warm off-white `#F0EFEA`** card, and a **light slate
  blue `#7EACE4`** screen. It must read clearly on both — a deep red orb with a dark contour does.
- In the game it **swells briefly when more currency arrives** and shrinks back. That is done by
  scaling the picture, so draw nothing that suggests motion and nothing outside the sphere.

### The inline orb

It sits in a line of bold capitals and digits that already exists, so it takes **their** style
rather than the large orb's, exactly:

- **A true black contour 12 pixels thick**, the same as the letters' — measure it across the side of
  their `I` at mid-height. No thicker.
- **A hard black drop shadow**, about 5 pixels down and slightly right, merged with the contour.
  No blur.
- **A flat fill in the blood `core`, with a hard-edged `light` crescent along the upper left and a
  hard-edged `dark` crescent along the lower right**, and **one** hard-edged soft-white highlight at
  the upper left. No gradients, no swirls, no glass, no texture, no glow. It is the orb reduced to
  the letters' flat, inked shading.
- **Where it sits in the cell**, against the letters:

  | On the letters                        | Pixels                        |
  | ------------------------------------- | ----------------------------- |
  | outline thickness                     | **12**                        |
  | top of a capital's black outline      | **23** down from the cell top |
  | bottom of a capital's black outline   | **147**                       |
  | bottom of the drop shadow             | 152                           |

  **The orb's black, outline included and drop shadow left out, is a circle about 112 across**,
  centered left to right, with its bottom on the capitals' bottom at 147 — so it is a little shorter
  than a capital and stands on the same line, the way a round letter does. Nothing it draws comes
  within 4 pixels of any edge of the cell.
- **It must not be mistaken for the letter `O` or the digit `0`** in the attached sheet: it is a
  solid filled disc with a highlight, and they are open rings.

### The colors

| Word  | dark      | core      | light     |
| ----- | --------- | --------- | --------- |
| blood | `#4A0710` | `#A81A2A` | `#E0485A` |

The orb's body is `core`, shading to `dark` away from the light; `light` is the red where light
passes through near the lower left edge. The highlights are a soft warm white, not pure white.

### The canvas

`vitae-orb` is **exactly 256 x 256** and `vitae-orb-inline` is **exactly 160 x 160**, each a
**transparent** PNG with an alpha channel. Work as large as you like and reduce once to the final
size on the way out.

Avoid: text, letters, numbers, symbols on the orb, coins, a stand or cap, liquid outside the
sphere, drips, splashes, gore, a heart shape, backgrounds, scenery, glow, halos, sparkle, a cast
shadow, orange or pink tints, and — on `vitae-orb` — any heavy black outline.

---

## Before you hand anything back

1. **Reduce `vitae-orb` to 24 x 24 on off-white `#F0EFEA` and on slate blue `#7EACE4`.** Is it a
   shiny red ball on both, with its highlight and its edge still visible?
2. **Reduce `vitae-orb` to 160 x 160 on both grounds.** Is it rich and alive, blood rather than red
   plastic or a cherry?
3. **Set `vitae-orb-inline` after `12` and after `SELL FOR 3` in the attached lettering.** Does it
   stand on the line, at the letters' weight, and read as a currency mark rather than as an `O`?
4. **Side by side, are the two obviously the same orb?**
5. **Is everything outside the sphere fully transparent**, with no halo?

Name the two files exactly `vitae-orb.png` and `vitae-orb-inline.png`, and give me a single zip
file of both so I can download them at once — never a link per file.
