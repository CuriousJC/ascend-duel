# Particle art prompt

Fifteen small particle sprites for a card game — **three for each of five elements: fire, ice,
lightning, earth, arcane**. A particle is one tiny piece of an element: an ember, an ice shard, a
spark, a pebble, a glowing mote. The game handles all the motion: it spawns dozens of these, scales
them, spins some of them, fades them and moves them around a damage number and behind it as the
number flies across the screen. **Each file is a single still object. It is not an effect, a burst,
a trail or an animation frame.**

---

## The prompt

High-resolution digital illustration of **fifteen small game particle sprites**, one per file, each
a single object on a **fully transparent** background. Think of the sparks, embers and shards a
polished mobile card game throws off a hit: simple, bright, instantly recognizable shapes. They
should look like **little objects, not paintings**.

### Every particle

- **One object, centered, filling about 75% of the canvas**, with clear space all round. Its center
  of mass sits on the center of the canvas, because the game spins and scales it about that point.
  Nothing else is in the picture.
- **Seen on screen between 12 and 40 pixels across.** Chunky shapes only: nothing in the object is
  thinner than about **an eighth of the canvas**. No hairlines, no fine detail, no filigree. **Shrunk
  to 16 pixels, each must still be its shape and its color.**
- **Colored in its element's ramp** (table below): the `core` across most of the object, the `light`
  as one bright highlight or a hot inner core, the `dark` in the shadowed side and the deepest
  creases. Smooth shading and a soft polished sheen, anti-aliased edges. **Saturated and bright** —
  these are the most vivid things on the screen for the moment they are up.
- **A thin near-black contour all the way round**, about **a thirtieth of the canvas** thick,
  unbroken. The particles fly over a **pale blue-gray table** as well as over darker painted
  scenery, and on the pale ground a particle with no outline disappears. The contour is what makes
  a yellow spark or a pale ice flake read there.
- **The light is even and centered**, never from one side. Some particles are spun as they fly, so a
  highlight painted in the upper-left corner would swing round to the bottom. Put the brightest part
  in the middle of the object, or along its spine.
- **Directional particles point straight up.** A flame lick, an ice shard and a lightning arc each
  have a front, and that front is the top of the canvas. The game turns them to face the way they
  are moving.
- **No glow or halo outside the contour.** The game adds any glow it wants. A soft glow painted into
  the file turns into a gray smudge on the pale table and a hard-edged square when particles
  overlap. Everything outside the contour is fully transparent.
- No background, no ground, no shadow, no motion blur, no speed lines, no streaks, no trail, no
  sparkle stars around the object, no frame, no text, no letters, no numbers.

### The five ramps

| Element   | dark      | core      | light     |
| --------- | --------- | --------- | --------- |
| fire      | `#9E2F13` | `#F06A24` | `#FFB24A` |
| ice       | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| earth     | `#285E35` | `#4E9B57` | `#8BCB78` |
| arcane    | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

Fire is **orange**, ice is **blue**, lightning is **yellow**, earth is **green** and arcane is
**purple**. Stay inside each element's ramp: no white-hot fire, no gray rock, no pink arcane.

### The fifteen particles

The three in each element are **different shapes**, so a cloud of them reads as varied rather than
as one stamp repeated. The first is always the round one, the second the pointed one and the third
the odd one.

| File                   | What it is |
| ---------------------- | ---------- |
| `particle-fire-1`      | **An ember**: a round glowing coal, slightly lumpy, with a hot `light` center and a darker crust at the rim. |
| `particle-fire-2`      | **A flame lick**: one fat teardrop flame, round at the bottom and rising to a single curving point at the top. A smaller lighter teardrop inside its base. **Points up.** |
| `particle-fire-3`      | **A twin flame**: two small flame tongues rising from one shared rounded base, the left one taller. |
| `particle-ice-1`       | **A frost chip**: a small chunky hexagonal crystal, flat-faced, each facet a slightly different blue so it reads as cut ice. |
| `particle-ice-2`       | **An ice shard**: one long crystal splinter, a narrow diamond, sharp at the top and blunter at the bottom, with a bright edge down its spine. **Points up.** About three times as tall as it is wide, but its narrowest part still no thinner than an eighth of the canvas. |
| `particle-ice-3`       | **A snowflake**: six thick, equal, evenly spaced arms around a solid hexagonal center, each arm ending in a short thick fork. Perfectly six-fold. Not an asterisk of thin lines. |
| `particle-lightning-1` | **A spark**: a fat four-pointed star with a round hot center, the points short and blunt. |
| `particle-lightning-2` | **An arc**: a short thick zigzag bolt fragment with two sharp bends, tapering to a point at the top. **Points up.** |
| `particle-lightning-3` | **A crackle**: a small chunky Y-shaped fork of lightning, a thick stem branching into two short arms. |
| `particle-earth-1`     | **A pebble**: a smooth rounded stone, slightly flattened, with one lighter worn face. |
| `particle-earth-2`     | **A rock chip**: an angular, sharp-cornered splinter of stone with flat-faced planes, each plane a slightly different green so the facets read. **Points up.** |
| `particle-earth-3`     | **A dust puff**: a small rounded clump of three or four overlapping soft balls of green dust, solid rather than wispy, with a clear contour like the rest. |
| `particle-arcane-1`    | **A mote**: a glowing round orb with a hot `light` center and a chunky ring tilted around it like a small planet. |
| `particle-arcane-2`    | **A crescent sliver**: a fat crescent moon with its horns pointing up. **Points up.** |
| `particle-arcane-3`    | **A sigil chip**: a small solid triangle with a round hole in its middle, like a fragment of a magic circle. Simple geometry. No letters, no runes, no writing. |

**The fifteen are one set.** Same contour thickness, same finish, same brightness and the same
overall size. Laid side by side they must read as one family, differing only in shape and color.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as
you like and reduce once to 256 on the way out. If you cannot produce real transparency, use a flat
pure magenta `#FF00FF` background with nothing else on it, and say so.

Avoid: glow, halo, bloom, motion blur, trails, streaks, speed lines, smoke wisps, thin lines, small
details, sparkle stars, backgrounds, shadows, text, letters, numbers, and any off-ramp color.

---

## Before you hand anything back

1. **Shrink every particle to 16 x 16.** Is it still an ember, a flame, a chip, a shard, a
   snowflake, a spark, an arc, a fork, a pebble, a rock chip, a puff, a mote, a crescent, a sigil?
   Is its color still obviously its element?
2. **Lay each one on a pale blue-gray ground (`#B8C4D6`) and on a near-black one.** Does it read on
   both? Is the contour unbroken all the way round?
3. **Spin each round particle a quarter turn.** Does the light still look right, with no highlight
   stuck in one corner?
4. **Do the flame lick, the ice shard, the arc, the rock chip and the crescent all point straight
   up?**
5. **Is everything outside each contour fully transparent**, with no glow, no halo and no fringe?
6. **Line all fifteen up side by side.** Same size, same contour, same finish: one family?

Name the fifteen files exactly as in the table, each ending `.png`, and give me a single zip file of
everything so I can download it all at once, never a link per file.
