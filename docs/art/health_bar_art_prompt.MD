# Health bar art prompt

The pieces a card game builds its **health bars** from. A health bar is a long rounded channel with
a red fill lying in it, and the fill shrinks from the right as the fighter loses life. The game
stretches the channel and the fill to whatever width it needs, so each is drawn as **two end caps
and a middle that can be stretched sideways without changing**. A third piece, a short marker, is
laid across the bar at fixed points along it.

**Draw each piece once, on its own.** The game lays them out, stretches them and stacks them; never
draw a whole bar with its fill and markers as one picture.

---

## The prompt

High-resolution digital illustration of **three game-interface pieces**, one per file, cut from
**polished stone** — the same material as a set of rounded jade and amber buttons that already
exist. **Attach one of those buttons and match its finish.**

### The look

- **Polished stone, smooth and dense**, lit softly from the upper left, with a soft satin polish.
- **The shading runs top to bottom only**: each piece is its `light` value along the top, its `core`
  through the middle and its `dark` along the bottom, in a **smooth vertical gradient**. Because the
  middle of a piece is stretched sideways, **nothing on it may vary from left to right** — no
  clouding, no veins, no speckle, no texture along its length.
- **A dark edge line all the way round each piece, about a sixteenth of its height**, in a very dark
  shade of the piece's own color — darker and firmer than the buttons' edge, because these are drawn
  small over painted portraits of every brightness and the edge is what holds them there. Not pure
  black, not a heavy cartoon contour.
- **No glow, no sparkle, no hard gloss streak, no bevel, no metal sheen.**

### The shape

The channel and the fill are both a **long, low capsule with fully rounded ends** — a pill shape.
Straight along the top and bottom, a half-circle at each end. The two ends are mirror images of each
other.

### The three pieces

| File                | Piece                   | Reads as | dark      | core      | light     |
| ------------------- | ----------------------- | -------- | --------- | --------- | --------- |
| `health-bar-trough` | the whole bar, empty    | **an empty channel carved into dark stone** — see below | `#1F1818` | `#3E3030` | `#5A4848` |
| `health-bar-fill`   | the life that is left   | a polished red stone bar lying in the channel | `#6E1616` | `#C02C2C` | `#F07A6A` |
| `health-bar-third`  | a marker across the bar | a short upright amber mark — see below | `#6B3F0E` | `#B5731F` | `#F2B84A` |

**The channel is sunken, so its shading is reversed**: dark along the **top** inside its edge, as the
shadow thrown by the channel's upper lip, lightening toward the bottom. No shadow falls outside it —
it is a hollow, not a piece.

**The fill sits in it, polished.** Shading the normal way round — light on top, dark underneath —
with a soft lighter sheen along the upper third, like light on a rounded red gem, running the whole
length evenly. **The fill is drawn at the same size as the channel**, with the same outline shape and
edge weight, so laid exactly on top of the channel it covers it completely. No drop shadow.

**The marker is a vertical broken line**: **three short upright dashes** stacked one above the other
with two small even gaps between them. Each dash is a small rounded rectangle, taller than it is
wide, in polished amber shaded light at its upper end and darker at its lower, with **its own dark
edge about 3 pixels thick**. No shadow. It is laid over the bar at a third and at two thirds of its
length, so it must read clearly on top of both the red fill and the dark empty channel — its dark
edge is what does that.

### The canvas, and the stretch

**The channel and the fill are each exactly 256 x 64**, a **transparent** PNG: nothing but the
piece and its edge, on a fully transparent ground.

- **The piece fills the canvas**: its edge touches all four sides. No padding.
- **The left end cap lives in x 0–31, the right end cap in x 224–255.** All the rounding happens
  inside those 32 pixels at each side.
- **Every column from x 32 to x 223 is identical.** The middle is the part the game stretches to
  any width, so it must be the same at every column across it: straight edge on top, the same
  vertical gradient, straight edge at the bottom, and nothing that varies along its length.

**The marker is exactly 32 x 64**, a transparent PNG, centered left to right.

- **The three dashes together run from y 8 to y 55** — the inside of the bar's edge — so the marker
  sits within the bar rather than poking out of it. Leave y 0–7 and y 56–63 transparent.
- **The dashes are about 12 pixels wide** including their edge, centered on x 16.
- Three dashes of equal height, two gaps of equal height, each gap about half a dash.

Avoid: text, letters, numbers, icons, symbols, sparkles, glow, texture, clouding, reflections,
hearts, and any decoration in the middle stretch.

---

## Before you hand anything back

1. **Stretch the channel and the fill to 1024 x 64 and squeeze them to 96 x 64**, stretching only
   the middle. Does anything in the middle smear or show a seam where it meets a cap?
2. **Lay the fill on the channel at full width.** Does it cover the channel exactly, with no channel
   showing round the edge?
3. **Lay the fill on the channel at about half width**, both ends rounded, and put a marker at a
   third and at two thirds. Reduce that to 170 x 24 — close to the size it is used at. Is the bar
   still clearly one bar, partly full, and are both markers visible — the one on the red and the one
   on the empty channel?
4. **Look at the same reduction over a dark portrait and over a light one.** Does the edge hold it on
   both?
5. **Is the channel clearly a hollow** and the fill clearly a polished piece lying in it?
6. **Is the ground fully transparent**, with no halo round any edge?

Name the three files exactly `health-bar-trough.png`, `health-bar-fill.png` and
`health-bar-third.png`, and give me a single zip file of everything so I can download it all at
once — never a link per file.
