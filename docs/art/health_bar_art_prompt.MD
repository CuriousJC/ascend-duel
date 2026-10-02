# Health bar art prompt

The pieces a card game builds its **health bars** from. A health bar is a long rounded trough with a
red fill lying in it, and the fill shrinks from the right as the fighter loses life. The game
stretches every piece to whatever width it needs, so the trough and the fill are each drawn as
**two end caps and a middle that can be stretched sideways without changing**. A third piece, a
short dashed marker, is laid across the bar at fixed points along it.

**Draw each piece once, on its own.** The game lays them out, stretches them and stacks them;
never draw a whole bar with its fill and markers as one picture.

---

## The prompt

High-resolution digital illustration of **three game-interface pieces**, one per file, in a bold
inked graphic style. They are not rendered in a vector program and they are not pixel art.

### The look

**A heavy, hand-inked black contour around every piece, and a flat, saturated fill inside it.**
This is the whole of the style, and the contour is the part that matters most.

- **The contour on the trough and the fill is thick: about an eighth of the piece's height**, all
  the way round. At a piece 64 pixels tall that is an 8-pixel line. It is much heavier than looks
  comfortable at full size, and it is meant to be — the pieces are used at about a third of the
  size they are delivered at, over painted portraits of every color and brightness, and **the
  black line is what carries them on all of those.**
- **The contour is true black**, or within a few points of it. Not a dark shade of the fill, not
  dark gray.
- **It has an inked, graphic-novel quality**: confident, crisp edges, like bold comic-book panel
  work. Not a uniform vector stroke, not sketchy or wobbly.

**The color inside is flat, with two hard-edged tone bands and nothing more.**

- The body is the piece's `core` value (see the table below).
- A **lighter band along the top** in the piece's `light` value, with a crisp edge rather than a
  gradient, like one brush stroke of highlight.
- A **darker band along the bottom** in the piece's `dark` value, also hard-edged.
- **No gradients, no texture, no noise, no glow, no gloss, no bevel, no metal sheen.**

Light comes from **above**.

### The shape

The trough and the fill are both a **long, low capsule with fully rounded ends** — a pill shape.
Straight along the top and bottom, a half-circle at each end. The two ends are mirror images of
each other.

### The three pieces

| File                 | Piece                          | Reads as                                                     | dark      | core      | light     |
| -------------------- | ------------------------------ | ------------------------------------------------------------ | --------- | --------- | --------- |
| `health-bar-trough`  | the whole bar, empty           | **a long empty channel pressed into the surface** — see below | `#2A1C1C` | `#5C4242` | `#7E6262` |
| `health-bar-fill`    | the life that is left          | a solid red bar lying in the channel, standing proud of it   | `#7A1E1E` | `#C62E2E` | `#EE7A6A` |
| `health-bar-third`   | a marker across the bar        | a short dashed amber line, standing upright — see below      | `#7A4A06` | `#C47C0C` | `#F2B84A` |

**The trough is sunken, so its bands are reversed**: the **dark** band runs along the **top**
inside the contour, as the shadow thrown by the channel's upper lip, and the **light** band along
the bottom. It has **no drop shadow** — it is a hole, not a piece.

**The fill sits proud.** Bands the normal way round — light on top, dark underneath — and **a
solid black drop**: a hard shadow offset straight down by about a sixteenth of the piece's height,
merged with the contour so the two read as one black shape. No blur and no soft shadow. **The fill
is drawn at the same size as the trough**, and its outline is the same shape and weight, so laid
exactly on top of the trough it covers it completely.

**The marker is a vertical broken line**: **three short upright dashes** stacked one above the
other with two small even gaps between them, like a dashed line drawn top to bottom. Each dash is a
small rounded rectangle, taller than it is wide, in the amber with its own **black contour about
3 pixels thick**, and the light band on its upper end and the dark on its lower. **No drop
shadow.** The marker is laid over the bar at a third and at two thirds of its length, so it has to
read clearly on top of both the red fill and the dark empty channel — the black contour is what
does that.

### The canvas, and the stretch

**The trough and the fill are each exactly 256 x 64**, a **transparent** PNG: nothing but the
piece, its contour and (on the fill) its drop, on a fully transparent ground.

- **The piece fills the canvas**: its contour touches the left and right edges and the top, and the
  bottom contour (or the fill's drop) reaches the bottom edge. No padding.
- **The left end cap lives in x 0–31, the right end cap in x 224–255.** All the rounding happens
  inside those 32 pixels at each side.
- **Every column from x 32 to x 223 is identical.** The middle is the part the game stretches to
  any width, so it must be the same at every column across it: straight contour on top, straight
  bands, straight contour at the bottom, and nothing that varies along its length.

**The marker is exactly 32 x 64**, a transparent PNG, centered left to right.

- **The three dashes together run from y 8 to y 55** — the inside of the bar's contour — so the
  marker sits within the bar rather than poking out of it. Leave y 0–7 and y 56–63 transparent.
- **The dashes are about 12 pixels wide** including their contour, centered on x 16.
- Three dashes of equal height, two gaps of equal height, each gap about half a dash.

Avoid: text, letters, numbers, icons, symbols, sparkles, glow, gradients, reflections, hearts,
and any decoration in the middle stretch.

---

## Before you hand anything back

1. **Stretch the trough and the fill to 1024 x 64 and squeeze them to 96 x 64**, stretching only
   the middle. Does anything in the middle smear or show a seam where it meets a cap?
2. **Lay the fill on the trough at full width.** Does it cover the trough exactly, with no trough
   showing round the edge?
3. **Lay the fill on the trough at about half width**, both ends rounded, and put a marker at a
   third and at two thirds. Reduce that to 170 x 24 — close to the size it is used at. Is the bar
   still clearly one bar, partly full, and are both markers visible — the one on the red and the
   one on the empty channel?
4. **Look at the same reduction over a dark portrait and over a light one.** Does it hold on both?
5. **Is the trough clearly a channel** and the fill clearly a piece lying in it?
6. **Is the ground fully transparent**, with no halo of near-white or near-black pixels round any
   contour?

Name the three files exactly `health-bar-trough.png`, `health-bar-fill.png` and
`health-bar-third.png`, and give me a single zip file of everything so I can download it all at
once — never a link per file.
