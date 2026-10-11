# Effect art prompt

Ten small pictures for a card game's effects: **nine particle sprites in three neutral sets — ash,
rubble and rune motes — and one rune sigil ring**. A particle is one tiny still object. The game
handles all the motion: it spawns dozens of them, scales them, spins them, fades them and moves them
off a card that is burning, crumbling or being used up. The sigil is turned and grown by the game
under a card as it fires. **Each file is a single still picture. It is not a burst, a cloud, a trail
or an animation frame.**

---

## The prompt

High-resolution digital illustration of **ten game effect sprites**, one per file, each on a **fully
transparent** background. Simple, clean, instantly recognizable shapes, like the effects of a
polished mobile card game.

### Every particle (the first nine files)

- **One object, centered, filling about 75% of the canvas**, with clear space all round. Its center
  of mass sits on the center of the canvas, because the game spins and scales it about that point.
  Nothing else is in the picture.
- **Seen on screen between 10 and 32 pixels across.** Chunky shapes only: nothing in the object is
  thinner than about **an eighth of the canvas**. No hairlines, no fine detail. **Shrunk to 16
  pixels, each must still be its shape.**
- **Colored in its set's ramp** (table below): the `core` across most of the object, the `light` as
  one highlight, the `dark` in the shadowed side and the creases. Smooth shading, anti-aliased edges.
- **A thin near-black contour all the way round**, about **a thirtieth of the canvas** thick,
  unbroken. These fly over a **pale blue-gray table** as well as over darker scenery, and on the pale
  ground an object with no outline disappears.
- **The light is even and centered**, never from one side, because the game spins them. Put the
  brightest part in the middle of the object.
- **No glow or halo outside the contour.** The game adds any glow it wants. Everything outside the
  contour is fully transparent.
- No background, no shadow, no motion blur, no trail, no smaller bits around it, no frame, no text,
  no letters, no numbers.

### The three ramps

| Set        | What it is                    | dark      | core      | light     |
| ---------- | ----------------------------- | --------- | --------- | --------- |
| ash        | **charcoal gray**             | `#2A2826` | `#55504B` | `#8A837B` |
| rubble     | **dark gray stone**           | `#3C3E41` | `#6B6D70` | `#A1A3A6` |
| rune motes | **cream ivory light**         | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |

**No other hue.** Ash may carry the faintest warm tint at one edge and nothing more; the rubble is
plain gray rock, never green or brown; the motes are pale cream, never yellow, gold, pink or purple.

### The nine particles

| File          | What it is |
| ------------- | ---------- |
| `ash-1`       | **An ash flake**: a thin-looking but chunky curled flake of burnt paper, an irregular rounded shape with one edge curling over. |
| `ash-2`       | **A cinder**: a small round lump of charcoal, matte, slightly cracked across its face. |
| `ash-3`       | **A torn scrap**: a small ragged triangle of burnt card, its edges uneven, flat to the viewer. |
| `rubble-1`    | **A pebble**: a smooth rounded stone, slightly flattened, one lighter worn face. |
| `rubble-2`    | **A chip**: an angular sharp-cornered splinter of stone with flat faces, each face a slightly different gray so the facets read. |
| `rubble-3`    | **A grit clump**: three or four small stones fused into one rough lump, solid, with a clear outer contour. |
| `rune-mote-1` | **A mote**: a glowing round orb, brightest at its center, with a soft lighter inner ring. |
| `rune-mote-2` | **A diamond**: a fat four-sided gem shape, a rhombus standing on its point, with a bright facet down its middle. |
| `rune-mote-3` | **A ring mote**: a chunky round ring, a fat torus seen face on, with a round hole through its middle. |

**The nine are one set.** Same contour thickness, same finish, same overall size. Laid side by side
they must read as one family, differing only in shape and color.

### The sigil (the tenth file)

`rune-sigil` is a **circular magic sigil**, seen flat and face on: an outer ring, an inner ring, and
between them a band of simple **geometric** marks — short bars, dots, small triangles and diamonds —
spaced evenly all the way round, with a six-pointed star of straight lines inside the inner ring.

- **Drawn as light**: pale cream lines (`#FFF3D2`, shading to `#DCCFAE`) on a **fully transparent**
  background. **No contour, no dark line anywhere** — the game draws it additively, so it brightens
  whatever is under it, and any dark pixel simply disappears.
- **Rotationally symmetric, with no up.** The game turns it continuously, so it must look equally
  right at every angle: no text, no single feature at the top, no light direction, no shadow.
- **Bold lines**, each at least **a fortieth of the canvas** thick, because it is seen from about 150
  to 400 pixels across.
- **The ring fills the canvas** edge to edge, its outer ring touching a circle about 95% of the
  canvas wide, perfectly centered.
- **No letters, no runes from any real alphabet, no writing of any kind**: shapes only.

### The canvas

The nine particles are each **exactly 256 x 256**, the sigil **exactly 512 x 512**, all **transparent**
PNGs with an alpha channel. Work as large as you like and reduce once on the way out. If you cannot
produce real transparency, use a flat pure magenta `#FF00FF` background with nothing else on it, and
say so.

Avoid: glow, halo, bloom, motion blur, trails, smoke, sparkle stars, backgrounds, shadows, text,
letters, numbers, real-world alphabets, and any hue outside the ramps above.

---

## Before you hand anything back

1. **Shrink every particle to 16 x 16.** Is it still a flake, a cinder, a scrap, a pebble, a chip, a
   clump, a mote, a diamond, a ring?
2. **Lay each particle on a pale blue-gray ground (`#B8C4D6`) and on a near-black one.** Does it read
   on both? Is the contour unbroken?
3. **Are the rubble plain gray and the motes pale cream**, with no other hue creeping in?
4. **Turn the sigil a few degrees at a time.** Does it look equally right at every angle? Is there no
   dark pixel anywhere in it, and no letter?
5. **Is everything outside each object fully transparent**, with no glow and no fringe?

Name the ten files exactly as above, each ending `.png`, and give me a single zip file of everything
so I can download it all at once, never a link per file.
