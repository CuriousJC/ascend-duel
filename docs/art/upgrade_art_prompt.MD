# Upgrade art prompt

High-resolution digital illustration of a single decorative surface — the gilded material that
covers the whole face of a playing card. No text anywhere in the image — no title, no label, no
caption, no signature, no watermark. The surface, nothing else.

Canvas: **portrait, aspect ratio exactly 5:7** — five wide to seven tall, around 1060 x 1484.

The surface: one material **filling the whole canvas, edge to edge on all four sides**. It runs off
the canvas at the top, the bottom and both sides; it has no border, no frame, no corners, no end
caps and no finials. The game cuts the corners and draws its own border round it.

The surface is **fully opaque and at full strength everywhere**, right up to every edge. Do not
fade it, feather it, vignette it or soften any edge — the game draws the card over it.

**Every part of the face carries the material**, with the bold marks spread across it rather than
gathered in one place, and nothing in it that draws the eye to a single spot. The left edge matters
most: cards are often seen stacked with only a narrow sliver of their left side showing, so that
sliver alone has to read as the material. Light comes from the upper left.

Scale: the face is reduced to about 192 pixels wide by 272 tall on the card, and to half that on a
small card. Anything smaller than about a fiftieth of the width is lost. Variation from top to
bottom matters as much as across.

Rendering: smooth, continuous shading with a clear material response — metal takes a long bright
specular ramp and small near-white glints, paint stays soft and wet. Anti-aliasing and soft
gradients are expected. Do not posterize, do not dither, do not quantize to a block grid.

Avoid: a card outline, a frame, a border running round four sides, corners, scenery, objects,
characters, a central emblem, gems set into the surface, text or runes, and anything that is not
the material.

Priority order: full-face surface -> recognizable material -> a strong left edge -> top-to-bottom
variation.

The subject — what the surface is made of — is the record's own `Draw` in `data/upgrade_art.json`,
pasted after this block.

Name each image after its record's own key — the `UpgradeArtRecord` value, like `golden` — and give me the file itself if there is only one, and a single zip file of
everything if there is more than one, so I can download it all at once — never a link per file.

## The color words

**A color word in the brief names one of these ramps**, the game's own colors, which the face has to
sit beside.

| Word    | Ramp      | dark      | core      | light     |
| ------- | --------- | --------- | --------- | --------- |
| orange  | fire      | `#9E2F13` | `#F06A24` | `#FFB24A` |
| yellow  | lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| green   | earth     | `#285E35` | `#4E9B57` | `#8BCB78` |
| blue    | ice       | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| purple  | arcane    | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| ivory   | ivory     | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |
| steel   | steel     | `#59636B` | `#9DA8B0` | `#D8DEE2` |
| granite | granite   | `#3C3E41` | `#6B6D70` | `#A1A3A6` |

## What happens to it afterwards

Not for the generator. The batch goes in `.scratch/to-process-upgrade-art/` and
`go run ./tools/relicart -kind upgrade` files it: reduced to the card's own **200 x 280**, the size
every other card picture is committed at, and written to `assets/upgrade-art/<record>.png`. The game
lays it over the face inside the border ring, clipped to the card's rounded corners, and draws the
figure, the form mark, the cost ticks, the badge and the name on top. `go run ./tools/upgradesheet`
draws every upgrade on its own and stacked as the deck panel stacks them, which is where the art is
judged — **with the card drawn over it, never bare**.

A strip version — a band down the left side, faded out — is kept, not live, in
`upgrade_strip_art_prompt_archived.MD`.
