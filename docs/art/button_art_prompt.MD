# Button face art prompt

A set of **button faces**: the blank bodies of a card game's interface buttons. The game prints each
button's label on top, and draws hover and press by brightening or darkening the face, so each file
is **one color at rest and nothing else**. The game scales every face to the button's height and
stretches it to the button's width, so a face is drawn as **two end caps and a middle that can be
stretched sideways without changing**.

**Draw each color once, on its own.** The game sizes and places the buttons; never draw a row of
buttons or a button with anything on it.

---

## The prompt

High-resolution digital illustration of **seven game-interface button faces**, one per file, in a bold
inked graphic style. They are not rendered in a vector program and they are not pixel art.

### The look

**A heavy, hand-inked black contour around every button, and a flat, saturated fill inside it.**
This is the whole of the style, and the contour is the part that matters most.

- **The contour is thick: about an eighth of the button's height**, all the way round, and it keeps
  that thickness through the curve of every corner. At a button 128 pixels tall that is a 16-pixel
  line. It is much heavier than looks comfortable at full size, and it is meant to be — the buttons
  are used at between a quarter and two-thirds of the size they are delivered at, over painted
  scenes of every color and every brightness, and **the black line is what carries them on all of
  those.**
- **The contour is true black**, or within a few points of it. Not a dark shade of the fill, not
  dark gray.
- **It has an inked, graphic-novel quality**: confident, crisp edges, like bold comic-book panel
  work. Not a uniform vector stroke, not sketchy or wobbly.
- **The black also sits a little lower than the button, as a hard drop**: a solid black shadow
  offset straight down by about a sixteenth of the button's height, merged with the contour so the
  two read as one black shape. No blur and no soft shadow.

**The fill is flat, with two hard-edged tone bands and nothing more.**

- The body of the button is the color's `core` value (see the table below).
- A **lighter band along the top** in the color's `light` value, with a crisp edge rather than a
  gradient, like one brush stroke of highlight. It follows the curve of the two top corners.
- A **darker band along the bottom** in the color's `dark` value, also hard-edged, following the two
  bottom corners.
- **No gradients, no texture, no noise, no glow, no gloss, no bevel, no metal sheen.**

Light comes from **above**: the light band on top, the dark band and the drop underneath.

### The shape

A **rounded rectangle whose corner radius is one quarter of its height** — at 128 pixels tall, a
32-pixel radius. Clearly rounded, but not a pill: there is a straight run down each end between the
two corners. Straight along the top and bottom. The two ends are mirror images of each other.

The button is **blank**: no text, no icon, no symbol, no inset panel.

### The seven colors

| File                 | dark      | core      | light     |
| -------------------- | --------- | --------- | --------- |
| `button-red`         | `#721C1F` | `#D0343A` | `#E28588` |
| `button-gray`        | `#32343B` | `#5C606C` | `#9D9FA6` |
| `button-yellow`      | `#7B6E21` | `#E1C83C` | `#EDDE8A` |
| `button-blue`        | `#26477E` | `#4682E6` | `#90B4F0` |
| `button-pink`        | `#81395D` | `#EB69AA` | `#F3A5CC` |
| `button-olive`       | `#343416` | `#5F5F28` | `#9F9F7E` |
| `button-disabled`    | —         | `#232323` | —         |

**The first six are the same drawing, recolored.** Same shape, same contour, same bands, same drop;
only the three values change. Draw one, then recolor it.

**The disabled face is the one exception**: the same shape and the same contour, but a **flat fill
with no bands and no drop shadow**. It is a control that is switched off, so it has no light on it
and does not stand up from the surface.

### The canvas, and the stretch

Each file is **exactly 512 x 128**, a **transparent** PNG: nothing but the button, its contour and
its drop on a fully transparent ground.

- **The button fills the canvas**: its contour touches the left and right edges and the top, and the
  drop reaches the bottom edge. No padding.
- **The left end cap lives in x 0–47, the right end cap in x 464–511.** All the rounding happens
  inside those 48 pixels at each side.
- **Every column from x 48 to x 463 is identical.** The middle is the part the game stretches to
  any width, so it must be the same from top to bottom at every column across it: straight contour
  on top, straight bands, straight contour and drop at the bottom, and nothing that varies along its
  length. No texture, no highlight streak that thins out, no taper, no pattern.

Avoid: text, letters, numbers, icons, symbols, gems, rivets, sparkles, glow, gradients, reflections,
and any decoration in the middle stretch.

---

## Before you hand anything back

1. **Stretch each face to 1600 x 128 and squeeze it to 110 x 128**, stretching only the middle, and
   look at both. Does anything in the middle smear or show a seam where it meets a cap?
2. **Reduce each to 128 x 32.** That is the smallest size it is used at. Is the black contour still
   there all the way round, and are the corners still visibly rounded?
3. **Lay a red, a gray and a disabled face over a dark picture and over a light one.** Do all
   three read as buttons on both, and does the disabled one read as switched off?
4. **Are the six colored faces the same drawing?** Toggle between any two; only the color should
   change.
5. **Is the corner radius a quarter of the height**, with a straight run left on each end?
6. **Is the ground fully transparent**, with no halo of near-white or near-black pixels round the
   contour?

Name the seven files exactly as in the table, each ending `.png`, and
give me a single zip file of everything so I can download it all at once — never a link per file.
