# Room glyph art prompt

Three small symbols that stand side by side in one row and say **how far through a realm the
player has got**: **a single door**, then **a double door**, then **a portal swirl**. A room the
player has reached is drawn in its realm's element color; a room still ahead is drawn in a plain
neutral gray. So each symbol is drawn **six times** — once per element and once neutral — eighteen
files in all.

---

## The prompt

High-resolution digital illustration of **three bold game icons**, one per file, each a single
chunky symbol on a **fully transparent** background, for a dark-fantasy card game. Think of the map
icons in a polished mobile game: simple, confident, instantly recognizable shapes — **icons, not
illustrations**.

### Where they are seen

The three sit **in a row across a small off-white card**, in the order door, double door, swirl,
each drawn **about 40 pixels tall**. The card's surface is a warm off-white, `#F0EFEA`. **Every
symbol has to read clearly against that off-white at that size**, in color and in gray alike.

### Every icon

- **Big.** The symbol fills **about 80% of the canvas height**, centered left to right, with a
  little clear space all round. It is the only thing in the picture.
- **The three share one floor line.** Both doors stand on the same baseline, about a tenth of the
  canvas up from the bottom edge, and their tops reach the same height. The swirl is centered at the
  same height as the middle of the doors and is about as tall as they are. Laid side by side the
  three must line up as a row, not bob up and down.
- **Solid, in its color's ramp**: smooth shading from the ramp's `light` toward the upper left to
  its `dark` away from it, the `core` between, with a soft polished sheen. One or two simple inner
  shapes at most, in the same ramp, to give it form.
- **A thin near-black contour** around the whole silhouette, about **a fiftieth of the canvas**,
  so the shape holds its edge on off-white. Thin and even — not a heavy comic-book outline.
- **Chunky shapes only.** Nothing in the symbol may be thinner than about **a tenth of the
  canvas** — no thin lines, no hinges, no handles smaller than that, no filigree. **Reduced to 40
  pixels tall, each must still be its shape.**
- No glow, no halo, no cast shadow, no sparkle, no background, no frame, no wall around the door,
  no floor drawn under it.

### The three icons

| File stem | What it is |
| --------- | ---------- |
| `room-door` | **A single arched door**, seen straight on: a tall rectangle with a **rounded arch top**, a little taller than it is wide — about **0.6 of the canvas wide** at its widest. Two or three **thick vertical planks** shown by shading, and one **chunky round ring handle** on the right, at about the middle of its height. Closed. Not a doorway with a hole through it, not a house, not a gate. |
| `room-double-door` | **A pair of arched doors side by side, meeting in the middle**, seen straight on: one wide arched opening split down the center into two leaves, **clearly wider than the single door** — about **0.85 of the canvas wide** — and the **same height** as it. Each leaf has thick vertical planks and **one chunky round ring handle beside the center seam**, the two rings facing each other. Closed. It must read as *grander* than the single door — bigger, double — never as two separate doors with a gap between them. |
| `room-portal` | **A round portal swirl**, seen straight on: **three or four thick spiral arms** winding in toward a **bright, nearly white round eye** at the center, the whole thing a rough disc about **0.8 of the canvas across**. Each arm is a thick band that tapers as it winds in. Solid and chunky, like a whirlpool drawn as an icon. Not concentric rings, not a target, not a galaxy with stars, not a spiral drawn as a thin line. |

### The six colorings

**The six files for one symbol are the same drawing**: same shape, same size, same place on the
canvas, same planks, same handles, same spiral — only the ramp changes. Draw each symbol once and
recolor it, so swapping one file for another never makes the symbol move.

The bright eye at the center of the portal is the color's own `light` pushed nearly to white, in
every coloring, so the swirl has a center whatever its ramp.

| Coloring | Hue | dark | core | light |
| -------- | --- | ---- | ---- | ----- |
| `fire` | orange | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `ice` | blue | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `lightning` | yellow | `#A98200` | `#F2D13D` | `#FFF38A` |
| `earth` | green | `#285E35` | `#4E9B57` | `#8BCB78` |
| `arcane` | purple | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| `neutral` | dark gray | `#3C3E41` | `#6B6D70` | `#A1A3A6` |

**The neutral coloring is a room the player has not reached yet.** It is the same symbol in plain
gray stone-like shading — quiet beside the colored ones, but still a clear, solid shape on the
off-white, not a faint ghost.

**The lightning yellow is the hardest on off-white.** Its contour is what carries it; check it
there before anything else.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as
you like and reduce once to 256 on the way out.

Avoid: text, letters, numbers, labels, keyholes, windows in the doors, walls, frames, floors,
backgrounds, scenery, glow, sparkle, rainbow coloring, thin lines, small details, and any heavy black
outline.

---

## Before you hand anything back

1. **Lay the three neutral files side by side on off-white `#F0EFEA`** and reduce the row to 40
   pixels tall. Is it a door, a bigger double door and a swirl, standing on one line?
2. **Do the same with the door and the double door in color and the swirl in neutral**, in each of
   the five colors. Is the colored pair obviously "reached" and the gray swirl obviously "not yet"?
3. **Is the yellow lightning set readable on the off-white**, contour and all?
4. **Flip through the six files of one symbol.** Nothing moves; only the color changes.
5. **Is everything outside the symbol fully transparent**, with no halo?

Name the eighteen files `<symbol>-<coloring>.png` — for example `room-door-fire.png`,
`room-double-door-neutral.png`, `room-portal-arcane.png` — and give me a single zip file of
everything so I can download it all at once — never a link per file.
