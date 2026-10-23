# Checkbox art prompt

A **checkbox** for a game's settings screen, drawn as **separate layers that stack**: an empty box,
and the mark that sits on it. Every file is the same size, with the box in exactly the same place,
so the mark laid over either box lines up without being moved.

Two boxes and one mark, so three files in all.

---

## The prompt

High-resolution digital illustration of a **game-interface checkbox**, one layer per file, for a
dark-fantasy card game. The box is a rounded square of **polished stone**; the mark is a bold pale
check laid over it.

### The canvas, and where the box sits in it

- Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as
  you like natively and reduce once to 256 on the way out.
- **The box does not fill the canvas.** It is a square **164 pixels across, centered** — so there is
  a 46-pixel transparent margin on all four sides, which the box files leave empty.
- **The box is in exactly the same place in both box files**, and the mark is drawn against that
  same position.

### The box

- **A square with rounded corners**, the corner radius about **a fifth of the box's side**. Clearly
  rounded, not a circle and not a pill.
- **Polished stone**: smooth and dense, with **faint soft clouding** inside it — paler and darker
  drifts of the stone's own color, like real carved jade. Gentle and low-contrast. Not glass, not
  plastic, not metal.
- **It reads as a well to put something in**: the face is **slightly recessed** — a little darker
  toward the top edge and lighter toward the bottom, as if lit from the upper left and seen as a
  shallow hollow rather than a raised tile. Keep it subtle: a soft satin dish, not a deep hole.
- **The face is otherwise completely plain.** **No line, arc, band, inner border, groove, highlight
  stripe or second outline anywhere on it.** No hard gloss, no reflection, no sparkle.
- **The edge is a thin, soft dark line — about a fortieth of the box's side**, in the stone's own
  `dark`, not black. No drop shadow, no black ring, no inked comic-book contour.
- Everything outside the box's rounded corners is **fully transparent**.

### The two stones

The two box files are **the same box in every way but the stone**: same size, same place, same
corner radius, same recess, same clouding pattern. Only the color changes.

- **Jade** — a deep, slightly blue-leaning green. Not emerald, not grass green, not lime.
- **Slate** — the jade drained of its color: a dull, cool, mid-dark gray stone with the same soft
  clouding. This is the box when it cannot be used, so it should read as quiet and unavailable.

### The mark

- **A check mark that stays inside the box.** A short stroke down to the lower left, then a long
  stroke up to the upper right. The whole mark fits within the box's face with a clear margin of
  stone all round — about **110 pixels** across at most, centered on the box.
- **Pale ivory.** Smooth shading within the stroke, lighter toward the upper left, and a **thin
  darker contour** of its own, about a fiftieth of the canvas: enough to hold its edge on jade,
  nowhere near heavy.
- **A bold, confident stroke, like a brush mark made in one motion** — thick, with a slight taper at
  the ends where the brush lifted. Not a thin pen line, not a font glyph, not a geometric icon built
  from rectangles. The stroke is at least **a tenth of the canvas** thick at its widest: the checkbox
  is used at **32 to 68 pixels** across, and a thinner stroke becomes a scratch.
- **It sits on the box, not above it**: no cast shadow, no glow, no halo.
- **The mark file holds the mark alone**, on a fully transparent canvas — no box drawn under it.

### The colors

| Word  | dark      | core      | light     |
| ----- | --------- | --------- | --------- |
| jade  | `#1E4636` | `#3D7A62` | `#86BFA4` |
| slate | `#33383A` | `#5C6366` | `#8F9699` |
| ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |

Avoid: text, letters, numbers, labels, a frame or inner border, a rim highlight, a second box, a
background, scenery, sparkle, glow, hard gloss, bevel edges, a colored mark, bright or saturated
green, and any heavy black outline.

---

## Before you hand anything back

1. **Lay the mark over each box.** It is where it should be against the box — nothing has to be
   nudged.
2. **Flip between the two boxes.** Nothing moves; only the stone's color changes.
3. **Reduce a box with the mark on it to 32 x 32 and look at it.** Is the check still a check, and
   can you tell at a glance whether the box is marked?
4. **Is everything outside the box and the mark fully transparent**, with no halo?

Name the three files exactly as below, each ending `.png`, and give me a single zip file of
everything so I can download it all at once — never a link per file.

| File                     | Layer       |
| ------------------------ | ----------- |
| `checkbox-jade`          | box, jade   |
| `checkbox-disabled`      | box, slate  |
| `checkmark-check-inside` | the check   |
