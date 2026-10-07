# Button and glyph art prompt

Parts for the buttons of a card game's interface, all cut from the same polished stone. A set of
square buttons already exists — rounded tiles of polished **jade** with a pale ivory icon on each,
and the same tile in **amber** for the one that is switched on. **Attach two of them, one jade and
one amber, and match them.**

1. **Word-button faces** — wide blank buttons a label is printed on afterwards, in jade, amber,
   carnelian and a switched-off stone.
2. **One more square tile** — the switched-off stone, blank.
3. **One dice glyph**, laid over the square tiles.

Six files in all.

---

## The stones

Every button is the same material: **polished stone**, smooth and dense, with **faint soft
clouding** inside it — paler and darker drifts of the stone's own color, gentle and low-contrast.
Lit softly from the upper left: the stone's `light` at the top, its `core` across the middle, its
`dark` at the bottom, in a smooth gradient. A soft satin polish. **No line, arc, band, rim, groove,
highlight stripe, inner border or second outline anywhere on a face**, no gloss streak, no sparkle.
The edge is a **thin, soft dark line** in the stone's own `dark` — never black, never heavy.

| Word      | What it is | dark      | core      | light     |
| --------- | ---------- | --------- | --------- | --------- |
| jade      | a deep, slightly blue-leaning green; not emerald, not grass, not lime | `#1E4636` | `#3D7A62` | `#86BFA4` |
| amber     | warm honey-gold polished amber, like fossil resin; not yellow, not orange | `#6B3F0E` | `#B5731F` | `#E3A84E` |
| carnelian | a deep polished red stone; not pink, not orange | `#5E1A16` | `#A8352B` | `#D9786A` |
| slate     | **switched off**: a dull, dark gray-green stone, lit **flat** — no gradient, no polish, the clouding barely there. It must read at a glance as a button that cannot be pressed. | `#1F2422` | `#2E3532` | `#3E4643` |

## 1. The word-button faces

Wide blank buttons. A word is printed across the middle afterwards, so the face carries nothing.

- **A rounded rectangle whose corner radius is a quarter of its height.** Clearly rounded at both
  ends with a straight run between, straight along the top and bottom. The two ends are mirror
  images.
- **Each file is exactly 512 x 128**, a transparent PNG. The button fills the canvas, its edge
  touching all four sides, everything outside the rounded corners fully transparent.
- **The left end lives in x 0–47 and the right end in x 464–511.** All the rounding happens inside
  those 48 pixels at each side.
- **The middle, x 48–463, repeats side by side to make a button of any width**, so it must **tile
  seamlessly left to right**: what runs off its right edge at x 463 continues exactly from its left
  edge at x 48 — the clouding, the gradient and the edge lines all meet with no seam. Its top and
  bottom edges are straight. Keep the clouding soft and even across it, with no feature that would
  read as the same blotch repeating.
- The same drawing in every stone; only the colors change.

| File               | Stone     |
| ------------------ | --------- |
| `button-jade`      | jade      |
| `button-amber`     | amber     |
| `button-carnelian` | carnelian |
| `button-disabled`  | slate     |

## 2. The switched-off square tile

**The attached square jade tile, in slate, with no icon** — the same rounded square (corner radius
about a fifth of the side), the same thin soft edge, filling the 256 x 256 canvas, everything
outside the corners fully transparent, the face completely plain.

| File                  | Stone |
| --------------------- | ----- |
| `icon-blank-disabled` | slate |

## 3. The dice glyph

**One die, seen straight on**: a single face of a die, a square with softly rounded corners, about
**70% of the canvas across**, in pale ivory with smooth shading lighter toward the upper left and a
thin darker contour. On it, **five big round pips** in the pattern of a five — one in each corner and
one in the center — drawn darker, cut into the surface. **Each pip about a fifth of the face's
width**, so they can be counted at 44 pixels. Not a cube, not at an angle, no other faces showing.
Everything outside the die fully transparent.

The ivory is `#DCCFAE`, shading to `#FFF3D2` and `#9B8B68`.

| File         | What it is |
| ------------ | ---------- |
| `glyph-dice` | one die face showing five |

---

## The canvas

The word faces are **512 x 128**; everything else is **256 x 256**. All are **transparent** PNGs
with an alpha channel. Work as large as you like and reduce once on the way out.

Avoid: text, letters, numbers, labels, a frame, a background, scenery, glow, sparkle, gloss, bevel
edges, any line or arc on a face, heavy black outlines, and any decoration in a word face's middle.

## Before you hand anything back

1. **Repeat each word face's middle three times side by side**, with its two ends on, and look at the
   joins. Any seam, or a blotch visibly repeating?
2. **Hold each stone beside the attached buttons.** Is the jade the same jade, the amber the same
   amber? Does slate read as switched off next to them?
3. **Reduce the die to 44 x 44.** Can the five pips be counted?
4. **Is everything outside each button and the die fully transparent**, with no halo?

Name the files exactly as in the tables, each ending `.png`, and give me a single zip file of
everything so I can download it all at once — never a link per file.
