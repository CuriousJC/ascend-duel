# Card edge art prompt

High-resolution digital illustration of a single tall decorative band — the gilded edging that
runs down one side of a playing card. No text anywhere in the image — no title, no label, no
caption, no signature, no watermark. The band and its background, nothing else.

Canvas: portrait, aspect ratio 2:3 (two wide to three tall), around 1024 x 1536.

The band: one straight vertical band, running the **full height of the canvas, top edge to bottom
edge**, centered left to right, and **about one-third of the canvas width** — roughly 350 pixels
wide on a 1024-wide canvas, so the band is a little over four times as tall as it is wide. Its two
long sides are perfectly straight, perfectly vertical and parallel. It does not taper, curve, bend,
end, or turn a corner; it has no end caps, no finials and no corner pieces. It is cut off cleanly
by the top and bottom of the canvas.

The band is **fully opaque and at full strength across its whole width**, right up to its right
side. Do not fade it, feather it, vignette it or soften its right side into the background — the
game fades it out itself.

Its profile across its width is not symmetrical. **The left side of the band is the outside edge
of the card**: a darker rolled lip, as if the band turns over the card's edge. **Put the strongest
detail in the left half** — the lip, a bright crest just inside it, the boldest marks — because
the right half is faded away to nothing on the card and lies under the card's text. The right half
is the band's quieter continuation: the same material and color, calm and even, with nothing in it
that stops abruptly or draws the eye. Light comes from the upper left.

Scale: the band is reduced to about 62 pixels wide by 270 tall on the card, and to half that on a
small card. Anything smaller than about a fiftieth of the band's width is lost. Variation from top
to bottom matters as much as across: the band is often seen as a narrow sliver of its left side
when cards are stacked, so the left side alone has to read as the material.

Rendering: smooth, continuous shading with a clear material response — metal takes a long bright
specular ramp and small near-white glints, paint stays soft and wet. Anti-aliasing and soft
gradients are expected. Do not posterize, do not dither, do not quantize to a block grid.

Background: flat pure black, #000000, everywhere that is not the band, with no texture, no
vignette, no glow and no shadow — the band is cut out of it by its edges.

Avoid: a whole card, a frame, a border running round four sides, corners, scenery, objects,
characters, gems set into the band, text or runes, and anything outside the band.

Priority order: straight full-height band -> recognizable material -> strong left side with the
dark lip -> calm, even right half -> top-to-bottom variation.

The subject — what the band is made of — is the record's own `Draw` in `data/edges.json`, pasted
after this block.

## The color words

**A color word in the brief names one of these ramps**, the game's own colors, which the band has
to sit beside.

| Word   | Ramp      | dark      | core      | light     |
| ------ | --------- | --------- | --------- | --------- |
| orange | fire      | `#9E2F13` | `#F06A24` | `#FFB24A` |
| yellow | lightning | `#A98200` | `#F2D13D` | `#FFF38A` |
| green  | earth     | `#285E35` | `#4E9B57` | `#8BCB78` |
| blue   | ice       | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| purple | arcane    | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |
| ivory  | ivory     | `#9B8B68` | `#DCCFAE` | `#FFF3D2` |
| steel  | steel     | `#59636B` | `#9DA8B0` | `#D8DEE2` |
| granite | granite  | `#3C3E41` | `#6B6D70` | `#A1A3A6` |

## What happens to it afterwards

Not for the generator. The band is cut out of the black by its columns, resized to **248 x 1088** —
four times the 62 x 272 the hand card draws, so the game's reduction is one averaging step — and
filed as `assets/edge/edge-<record>.png`, the stem `data/edges.json` names in `Art`. The game
stretches it to the record's `Width` and the face's height, fades it out from `FadeFrom` to
`FadeTo`, and mirrors it onto the right edge where the record's `Sides` is `both`, so only the
left band is ever drawn. A record whose
`Width` moves far from 62 wants art at its own proportions, or the picture is stretched sideways.

`Width`, `Inset`, `Opacity`, `FadeFrom`, `FadeTo` and `Sides` in `data/edges.json` are how it sits
on the card; `go run ./tools/upgradesheet` draws every upgrade on its own and stacked as the deck panel
stacks them, which is where both the art and those figures are judged.
