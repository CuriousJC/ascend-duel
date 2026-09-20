# Creature art prompt

**One image per record per element.** A creature's card is its picture, edge to edge, so what is
being made is a portrait rather than an icon.

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

High-resolution digital illustration of a **single creature**, seen from the front, from roughly
the chest up, filling a portrait card.

Canvas: **exactly 200x280**, portrait orientation, **fully opaque** PNG. Work as large as you like
natively and reduce once to 200x280 on the way out.

Framing: the creature is centred left to right and sits **slightly high** in the frame, filling
about **80% of the width** and about **70% of the height**. It faces the viewer, square on or very
nearly, and looks out of the picture. Crop it at roughly the chest — this is a portrait, not a
full body and not a scene. The bottom third of the canvas is comparatively quiet.

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

**Reserve nothing.** Do not leave a band, a bar, a corner or a disc clear or empty for anything to
be placed on later, and do not draw a plaque, a title bar, a frame, a border or a panel anywhere.
Numbers and a health bar are drawn on top of this picture by the game, and the picture carries on
underneath them — so a generator told a corner is special will decorate it, and a drawn bar sitting
under the real one is worse than none.

**One creature, alone.** No second figure, no companion, no swarm behind it, no rider, no mount.

Avoid: text of any kind, letters, numbers, symbols, runes, glyphs, sigils, magic circles, health
bars, icons, watermarks, signatures, frames, vignettes, lens flare, motion lines, and speech
bubbles.

---

## Before you hand it back

1. **Reduce it to 100x140 and look at it.** That is close to the size it is used at. Is the
   creature still legible, or has the detail averaged into mush?
2. **Is the silhouette the motif's?** Held next to its siblings, does it read as the same kind of
   thing — same body plan, same materials — or as a different creature that happens to share a
   colour?
3. **Is the element unmistakable** without reading the name, and is it the element doing what
   layer 3 said it does to *this* motif?
4. **Is the ground opaque to all four edges**, tinted rather than pure black, and empty of scenery?
5. **Is there anything in the corners** — a mark, a device, a darkening, a reserved patch? There
   should not be.
6. **Is there a contour** all the way round, thick enough to survive the reduction?

Name it `<Art>-<element>.png`, exactly as the brief says, and give me a link I can click to
download it.
