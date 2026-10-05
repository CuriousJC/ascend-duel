# Icon button art prompt

A set of **square icon buttons**: each one a whole button — the rounded face and the picture on it —
drawn together as one image. They are not a blank face with a label printed over it, which is what
`button_art_prompt.MD` makes; these carry no text, and the picture is the button's whole meaning.

The game draws hover and press over the file — brightening it, sinking it down a few pixels — so
each file is **the button at rest and nothing else**.

---

## The prompt

High-resolution digital illustration of **four square game-interface buttons**, one per file, for a
dark-fantasy card game. Each is a rounded square with one icon on it.

### The button

- **A square with rounded corners**, the corner radius about **a fifth of the side**. Clearly
  rounded, not a circle and not a pill.
- **No heavy outline.** The edge of the button is a **thin, soft dark line — about a fortieth of the
  side**, a darker shade of the button's own color rather than black. No black drop shadow, no
  black ring, no inked comic-book contour.
- **The face is a gently lit surface**: the button's `core` color, lifting toward its `light` value
  at the top and settling toward its `dark` value at the bottom in a **smooth gradient**, with a
  faint lighter rim along the top edge where the light catches it. It reads as a slightly raised,
  softly rounded tile. No gloss streak, no glass reflection, no metal sheen, no texture, no noise.
- **The button fills the canvas**: its edge touches all four sides, with no padding and no shadow
  falling outside it. Everything outside the rounded corners is **fully transparent**.

Light comes from the **upper left** on the button and on its icon alike.

### The icon

- **Centered, filling about 70% of the button's width**, so there is a clear margin of face all
  round it.
- **Pale, against the face**: the icon is light and the button is mid-dark, so it reads by contrast
  at a glance. Smooth shading within the icon — lighter on the faces toward the upper left, a little
  darker away from it — and a **thin darker contour** of its own, about a fiftieth of the side:
  enough to hold its edge, nowhere near heavy.
- **It sits on the face, not above it**: no cast shadow onto the button, no glow, no halo.
- **Chunky shapes only.** Nothing may be thinner than about **a twelfth of the side**: the buttons
  are used at **32 to 68 pixels** across, so a thin line becomes a gray scratch and a small detail
  disappears. Each icon is recognized by its silhouette first.

### The four buttons

| File                 | Button color | Icon |
| -------------------- | ------------ | ---- |
| `icon-close`         | red          | An **X**: two thick bars crossed at right angles, rotated 45 degrees, with squared-off ends, in near-white. A symbol rather than a letter — no serifs, no brush strokes. |
| `icon-settings`      | gray         | A **cog** in light gray to near-white: a round body with **eight square teeth around the rim, four on the axes and four on the diagonals**, and a round hole through the middle about a third of the body's width. Teeth are blocky rectangles, not tapered wedges. Hueless — no warm or cool tint, no metal color. |
| `icon-hands`         | gray         | **Five playing cards held up in a fan, the way a poker player holds a hand, seen from the player's side.** Each card is upright and **portrait — taller than it is wide**. All five **bottom ends gather at one pivot point low in the center of the button**, and the **top ends spread apart in an arc**, so together the five make a **half-circle fan opening upward**, like a peacock's tail or a folding hand fan. The middle card stands straight up; the two either side lean out about 20 degrees, the outer two about 40 degrees. The cards point **up and out, never down** — they are held, not dropped. Each card overlaps the one to its left, so all five tops show. The cards are **ivory**, blank faced — no pips, no numbers, no suits, no letters — each with its own thin dark edge so the five can be counted. |
| `icon-ledger`        | gray         | **An old parchment scroll, open, like a royal proclamation or a treasure map** — the classic storybook scroll. A **flat sheet of ivory parchment**, taller than it is wide, laid open so its whole face shows, with **each end curled into a tight roll**: the top edge rolls backward and the bottom edge rolls forward, and the **spiral of the rolled paper shows at the ends of each curl**. The sheet is the biggest part of the picture and the two curls are small, about a sixth of its height each. Set it at a slight tilt. Across the open face, **three or four bold horizontal strokes** stand for lines of writing — strokes, not letters. **Not** a closed tube, a rolled-up log, a cylinder, a box, or a banner. |

### The button colors

| Word  | dark      | core      | light     |
| ----- | --------- | --------- | --------- |
| red   | `#721C1F` | `#D0343A` | `#E28588` |
| gray  | `#32343B` | `#5C606C` | `#9D9FA6` |
| ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |

**The three gray buttons are the same button**, with a different icon. Draw the button once and
put each icon on a copy of it; the red one is that same button recolored.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as you
like natively and reduce once to 256 on the way out.

Avoid: text, letters, numbers, card suits, labels, a frame or border around the button, a second
button, a background, scenery, sparkle, glow, gloss, bevel edges, and any heavy black outline.

---

## Before you hand anything back

1. **Reduce each to 32 x 32 and to 44 x 44, and look at both.** Is the icon still the thing it is —
   an X, a cog with eight teeth, five cards in a fan, an open scroll? Can the five cards still be
   counted at 44?
2. **Lay all four over a dark picture and a light one.** Do they read as buttons on both, with no
   black ring around them?
3. **Are the three gray buttons the same button?** Toggle between them; only the icon should change.
4. **Is everything outside the rounded corners fully transparent**, with no halo?

Name the four files exactly as in the table, each ending `.png`, and give me a single zip file of
everything so I can download it all at once — never a link per file.

## What happens to it afterwards

Not for the generator. The files go under `assets/button/` at 256 x 256, keyed by stem in
`assets/embed.go`, and a button wears one by setting `models.Button.Icon` — which replaces both the
colored face and the label. The X is drawn at 32, the cog at 44, and HANDS and LEDGER at
`ui.PanelButtonSize`, 68.
