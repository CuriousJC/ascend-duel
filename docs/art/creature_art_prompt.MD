# Creature art prompt

**One image per record per element.** A creature's card is its picture, edge to edge, so what is
being made is a portrait rather than an icon. The bottom 80 pixels carry the creature's status
badges, its damage figure and its health bar, and most of that strip is covered by an opaque dark
band — so the portrait is composed against the 200 x 200 above it.

## A brief is four layers

Nothing here is the whole prompt. A finished brief is these four blocks in this order, and the
first is the only one in this file:

| layer | where it lives | what it says |
|---|---|---|
| 1. style and composition | **this file** | canvas, framing, ground, light, rendering — true of every creature in the game |
| 2. the motif | `Draw` on the motif's file header | the body plan: what makes a goblin a goblin |
| 3. the element, for that motif | `ElementDraw` on the same header | what fire does *to a goblin*, which is not what it does to a slime |
| 4. the record | `Draw` on the record | this particular creature: what it is doing, what it carries |

**Layer 3 is per motif on purpose.** An ice slime *is* ice, all the way through; an ice goblin is a
goblin wearing frost. One elemental block written for the whole roster would be right for one of
them and wrong for the other.

`go run ./tools/creatureprompt -record <key> -element <element>` assembles all four and prints the
prompt to paste.

---

## The prompt — layer 1

High-resolution digital illustration of a **single creature**, seen from the front, filling a
portrait card.

Canvas: **exactly 200x280**, portrait orientation, **fully opaque** PNG. Work as large as you like
natively and reduce once to 200x280 on the way out.

**Nothing the picture needs may live below y 200.** No part of the face, no hands, no weapon
head, no detail that makes the creature recognizable. From **y 216 to y 280** an opaque dark band
is laid over the picture and nothing painted there is seen at all; the sixteen pixels above that
band are bare picture with small badges over them, so keep them quiet and dark too.

**Paint the whole strip anyway.** The picture is opaque to all four edges and the ground carries on
across those lines without changing what it is doing. Do not leave the strip blank, do not fade it
out, and do not draw a panel, a plaque, a ledge, a shelf, a bar or a band of any kind there — the
real band goes on top, and a drawn one under it is worse than none.

Framing: work inside the **200 x 200** above the strip. The creature is centred left to right,
fills about **80% of the width** and about **85% of that height**, and sits slightly high in it. It
faces the viewer, square on or very nearly. **Land the bottom of the body around y 200** so it runs
down into the strip rather than stopping short of it — this is a portrait, not a full body and not
a scene.

**Where it is cropped depends on what the creature is built like.** A creature with shoulders and a
head — anything humanoid, anything four-limbed, anything with a face — is cropped at roughly the
chest and looks out of the picture, and the head is what the framing is built around. A creature
with no shoulders and no face — a slime, a swarm, a plant, a mass, a machine — is not cropped at a
body part at all: it fills the same box with its whole self, and what the framing is built around
is its **silhouette**, since that is the only thing telling it from its siblings. Do not invent a
head, a face or a pair of eyes on something that has none to make it easier to compose.

Ground: a **near-black background, tinted toward the creature's element** — deep and dark enough
that the creature reads against it, never a flat pure black. Carry **barely perceptible noise** so
it is not a dead digital field. The whole canvas is covered: no transparency anywhere, and the
ground runs right to all four edges. No scenery, no horizon, no floor, no room, no props — the
ground is a colour, not a place.

Light: from the **upper left**. Highlights on the upper-left surfaces, shadow falling away to the
lower right. Everything in this set is lit from that corner.

Rendering: **smooth material shading with real weight** — anti-aliased edges, soft gradients,
blended transitions, and a **near-black contour** around the creature's outside so it holds against
the ground. Do not posterize, do not dither, do not quantize to a block grid, and **do not draw
pixel art**.

**Reserve nothing.** The strip at the bottom is an instruction about what may live there, not an
empty space: keep the paint going through it. Do not leave any corner, strip or disc clear or empty
for something to be placed on later, and do not draw a plaque, a title bar, a frame, a border, a
panel or a bar anywhere at all.

**One creature, alone.** No second figure, no companion, no swarm behind it, no rider, no mount.

Avoid: text of any kind, letters, numbers, symbols, runes, glyphs, sigils, magic circles, health
bars, icons, watermarks, signatures, frames, vignettes, lens flare, motion lines, and speech
bubbles.

## The color words

**A color word in a brief names one of these ramps.** "A purple orb" is an orb in arcane; "an ivory
haft" is a haft in the stab material. These are the game's own colors and the picture has to sit
beside cards drawn in them, so the word means this range here rather than whatever you would
otherwise reach for.

| Word | Ramp | dark | core | light |
|---|---|---|---|---|
| orange | fire | `#9E2F13` | `#F06A24` | `#FFB24A` |
| yellow | lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| green | earth | `#285E35` | `#4E9B57` | `#8BCB78` |
| blue | ice | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| purple | arcane | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| cream, ivory | ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |
| cool gray, steel | steel | `#59636B` | `#9DA8B0` | `#D8DEE2` |
| dark gray, granite | granite | `#3C3E41` | `#6B6D70` | `#A1A3A6` |

The element name and the color word are the same instruction — `arcane` and "purple" both mean that
row, and so do `ivory` and "cream". The last three rows are the game's three form materials:
**ivory is stab, steel is slash, granite is crush.**

**How the three columns are used.** `core` is the creature's own color, carrying the large planes
and whatever the eye reads the hue as at a glance. `light` is the surfaces turned toward the light
and the top of the specular ramp — **the lit part is always the creature's own hue made lighter,
never a warm color and never gold.** `dark` is the surfaces turned away and the color the shadow
settles into, above the near-black contour, which stays near-black.

**The element this picture is being drawn as chooses the ramp**, and it governs two things: the
ground is a near-black tinted toward it, and whatever layer 3 says the element does to this motif is
drawn in it.

**The ramp directs the picture; it does not fence it.** The word's colors should be the **dominant**
ones, so the creature is unmistakably that element at card size. Other colors are welcome wherever
the subject wants them: skin, fur, scale, leather, a metal fitting, a warm glint in an eye, a
contour. What is wrong is **drift** — a creature whose dominant hue has slid to a neighbouring one,
or washed out far enough that the element stops being readable. A color word not in this table
(brown, black, white, gold, copper, bone) is an ordinary word and carries no ramp.

---

## Before you hand it back

1. **Cover the bottom 80 pixels and look at what is left.** That is nearly all anyone sees. Is
   the creature complete above the line, or has something that mattered gone under it?
2. **Reduce it to 100x140 and look at it.** That is close to the size it is used at. Is the
   creature still legible, or has the detail averaged into mush?
3. **Is the silhouette the motif's?** Held next to its siblings, does it read as the same kind of
   thing — same body plan, same materials — or as a different creature that happens to share a
   colour?
4. **And is it its own?** Held next to the other creatures of its own motif, is its outline
   different enough to tell them apart in the dark? For a creature with no face this is the whole
   of what distinguishes it.
5. **Is the element unmistakable** without reading the name, and is it the element doing what
   layer 3 said it does to *this* motif?
6. **Is the ground opaque to all four edges**, tinted toward the element's ramp rather than pure
   black, and empty of scenery?
7. **Is there anything in the corners** — a mark, a device, a darkening, a reserved patch? There
   should not be.
8. **Is there a contour** all the way round, thick enough to survive the reduction?

Name it `<Art>-<element>.png`, exactly as the brief says, and give me a link I can click to
download it.
