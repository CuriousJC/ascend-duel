# Background art prompt

**One picture per record, and a record belongs to one element.** A duel is fought in front of a
backdrop, and the floor's element decides which set of backdrops the fight draws from. So what is
made here is the table the whole combat screen is laid on — a landscape scene at the screen's own
size, with the cards, the panels and the type drawn over most of it.

## A brief is two layers

| layer                    | where it lives        | what it says                                                                 |
| ------------------------ | --------------------- | ---------------------------------------------------------------------------- |
| 1. style and composition | **this file**         | canvas, the map of what covers what, key, focus, rendering — true of every backdrop |
| 2. the record            | `Draw` on the record  | this particular place: what it is, what is lying about                       |

The record names its element, and the element picks the color word the picture is keyed in.

---

## The prompt — layer 1

A painted **backdrop for a card game**: a single wide view of one place, seen at a distance and
slightly out of focus, over which a game is laid. It is scenery rather than a subject — nothing in
it is the point of the picture, and nobody is in it.

Canvas: **exactly 1920x1080**, landscape 16:9, fully opaque, **delivered as a JPEG at quality 90** — not a PNG. Work at 3840x2160 if you can
and reduce once to 1920x1080 on the way out.

### The size of the place

**This is where a duel is fought, so the place is big enough to hold one.** Two fighters are going
to face each other here, standing well apart, with room to step, lunge and circle — a duel, not a
brawl in a corner. Draw the room for that, and leave the fighters out.

- **A wide, clear floor is the heart of the picture.** From the foreground to the far side there is
  an open expanse many paces deep and wider still: a hall, a great chamber, a courtyard, a
  clearing, a plateau. Nothing stands in the middle of it — no table, no desk, no chest, no
  furniture of any kind in the fighting space.
- **The furnishings live at the edges.** Shelves, benches, cauldrons, braziers, crates, trees and
  rocks line the walls or the boundary and stand well back from the centre, so they frame the
  floor rather than crowd it.
- **Scale it by the architecture.** Ceilings, where there are any, are high — two or three storeys,
  vaulted, or open to the sky. Doorways and windows are tall and read as distant. An ordinary adult
  standing at the far side of the floor would be no more than about a sixth of the picture's
  height.
- **The viewpoint is well back and a little above standing height**, looking across the floor
  toward the far side, so the depth of the space reads at a glance.
- **A brief naming a small place means its grand version.** A "workshop" is a workshop hall, a
  "study" a great library, a "forge" a foundry floor, a "cave" a cavern. Keep the character the
  brief asks for and give it the room a duel needs.

### What covers what

Most of the canvas ends up under something. Paint all of it — the picture is opaque to all four
edges and carries on under everything laid over it — but put what matters where it will be seen.
Coordinates are pixels from the top-left of the 1920x1080 canvas.

| Region                                  | x           | y          | What lies over it                                    |
| --------------------------------------- | ----------- | ---------- | ---------------------------------------------------- |
| two portrait cards, top corners         | 20–220 and 1700–1900 | 20–300 | opaque cards: completely hidden                 |
| the top band between them               | 225–1695    | 20–320     | two large **translucent** panels: seen, but dimmed and softened |
| the open middle, centre-left            | 250–1260    | 380–650    | large lettering, shown now and then over the picture |
| the right-hand middle                   | 1265–1900   | 375–660    | a row of opaque cards: mostly hidden                 |
| the lower band                          | 230–1690    | 650–960    | a row of opaque cards: completely hidden             |
| the bottom strip                        | 0–1920      | 960–1080   | buttons and small numerals over the picture          |
| the left edge                           | 0–230       | 330–1080   | a small card deck at the very bottom; otherwise open |

**The open middle is the most-seen stretch of the picture**, and the natural home for the far
distance of the scene — a far wall, windows, a sky, a vista. Let it be part of the place rather
than a gap in it; it does not need to be empty or plain.

**Put the place's character at the edges, along the top and in the open middle.** The lower half
is mostly hidden; let it be floor, ground, a table top or the foot of a wall, receding.

### Key, focus and depth

**Key and mood are the subject's to choose.** A candlelit study may be dim and warm, a snowfield
bright, a forge dark and glowing. What is wrong at any key is **extremes over large areas**: no
large patch of pure black, no large blown-out white, no single blinding light source in view.

**Soft focus, moderate contrast.** Render it as if seen through a slight haze or a shallow depth of
field: forms are recognisable, edges are soft, fine detail is suppressed, and the contrast is gentle
next to the cards'. The cards laid over it are crisp, detailed illustrations with hard near-black
contours, and the backdrop has to sit *behind* them — a sharp, detailed backdrop fights every card
on the screen.

**Atmospheric depth.** Things further away get softer and drift toward the element's color, as in
air.

Light: falling mainly from the **upper left**, soft rather than harsh. Light sources inside the
scene — a lamp, a fire, a glowing bottle — are welcome as long as none is a blinding hot spot.

Rendering: **painterly digital illustration** — smooth blended gradients, soft brushwork,
anti-aliased throughout. Not photographic, not pixel art, not flat vector, not posterized. No
dithering and no visible block grid.

### Nobody is here

**The place is empty of people and creatures.** No person, no figure, no silhouette, no
shadow of one, no animal, no familiar, no monster, no statue or portrait of a face. The game's
opponent is drawn on a card in the top-right corner and is the only living thing on the screen;
anything alive in the backdrop competes with it. Say who lives here with what they left behind —
a book left open, a cauldron still steaming, a cloak over a chair — never with the people
themselves. **If a brief names a person, draw their belongings and leave them out.**

### Reserve nothing

Do not draw a frame, a border, a panel, a plaque, a table-edge band, a title bar, a vignette or
an empty box anywhere. The regions above are a map of what is laid on top, not places to leave
blank or to decorate as though something belongs there. Paint the scene straight through them.

Avoid: legible text of any kind, letters, numbers, signatures, watermarks, logos, user-interface
elements, cards, playing cards, health bars, icons, lens flare, heavy vignetting, motion lines, and
speech bubbles. Ornament that reads as pattern — a carved border on a shelf, a decorative circle on
a floor — is fine; writing is not.

## The color words

**The record's element picks one of these ramps, and a color word in a brief names one.** "Purple
glassware" is glassware in arcane; "steel fittings" are fittings in the slash material. These are
the game's own colors and the backdrop has to sit under cards drawn in them.

| Word               | Ramp      | dark      | core      | light     |
| ------------------ | --------- | --------- | --------- | --------- |
| orange             | fire      | `#9E2F13` | `#F06A24` | `#FFB24A` |
| yellow             | lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| green              | earth     | `#285E35` | `#4E9B57` | `#8BCB78` |
| blue               | ice       | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| purple             | arcane    | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| cream, ivory       | ivory     | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |
| cool gray, steel   | steel     | `#59636B` | `#9DA8B0` | `#D8DEE2` |
| dark gray, granite | granite   | `#3C3E41` | `#6B6D70` | `#A1A3A6` |

The element name and the color word are the same instruction — `arcane` and "purple" both mean
that row. The last three rows are the game's three form materials: **ivory is stab, steel is
slash, granite is crush.**

**How a backdrop uses its ramp is not how a card does.** The ramp is the **cast of the whole
scene** — the color of its air, its light and its shadows — rather than the color of one object.
`dark` carries the shadows and the dim corners, `core` the lit surfaces and the accents — a lamp's
glow, a bottle, a banner — and `light` the brightest light in the place. The scene should be
unmistakably that element's color at a glance from across the room, the way a room lit through
tinted glass is.

**The ramp directs the picture; it does not fence it.** Wood, stone, cloth, skin and brass are
their own colors, softened into the key. What is wrong is **drift** — a backdrop whose overall
cast has slid to a neighbouring element's hue, or one so saturated with its own that the cards'
element colors on top of it stop reading.

---

## Before you hand it back

1. **Cover every opaque region in the table with a grey box** and look at what is left. Is the
   place still recognisable from the edges, the top and the open middle, and is anything
   important under a box?
2. **Squint at it.** Is there a large patch of pure black, a blown-out white, or a blinding light?
   There should not be.
3. **Is it soft?** Held beside a crisp illustrated card, does the backdrop sit behind it, or do
   the two argue about which is in focus?
4. **Could two people fight a duel on this floor?** Is there a wide, clear, open space in the
   middle, deep and wide, with every piece of furniture pushed back to the edges?
5. **Is the element unmistakable** from the overall cast, without a prop having to say it?
6. **Is the place empty?** No person, creature, silhouette, statue or face anywhere in it.
7. **Is it a JPEG, opaque to all four edges, exactly 1920x1080**, with no frame, vignette or reserved
   patch anywhere?

Name it `<Art>.jpg`, exactly as the brief says, and give me a link I can click to download it.
