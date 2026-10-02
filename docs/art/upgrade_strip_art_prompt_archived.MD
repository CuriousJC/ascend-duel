# Upgrade strip art prompt — archived, not live

> **Not the live prompt.** This is the strip version of the upgrade art: a band down the left side
> of the face, faded out across its width. The live prompt is `upgrade_art_prompt.MD`, which paints the
> whole face. The strips it produced are in `assets/archive/upgrade-strip/`, and the records they
> were drawn from — with the geometry they were laid out at — are at the foot of this file, in the
> shape the catalog had then (`data/edges.json`, filed under `assets/edge/` at 248 x 1088, drawn by
> a renderer that read the six draw properties). Restoring it is bringing that drawing back as well
> as the pictures and the records.

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

Name each image after its record's own key, ending `.png`, and give me the file itself if there is
only one, and a single zip file of everything if there is more than one, so I can download it all
at once — never a link per file.

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

## The records this prompt drew

```json
[
  {
    "EdgeRecord": "default",
    "Art": "edge-default",
    "Draw": "A plain band of brushed pewter: a cool mid-gray metal with a fine vertical grain, a slightly darker rolled lip along its left side and a thin bright highlight just inside it, otherwise quiet and even from top to bottom. Understated — it marks the card as altered without saying how.",
    "Width": 62,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 29,
    "FadeTo": 62,
    "Sides": "left"
  },
  {
    "EdgeRecord": "golden",
    "Art": "edge-golden",
    "Draw": "A band of burnished gold leaf: warm deep yellow metal with a raised, slightly rounded profile, a dark amber rolled lip along its left side, a bright near-white highlight running down the crest, and small irregular flecks and cracks where the leaf has been laid by hand. It should read unmistakably as real gold even when only a sliver of the left side shows.",
    "Width": 62,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 29,
    "FadeTo": 62,
    "Sides": "left"
  },
  {
    "EdgeRecord": "silver",
    "Art": "edge-silver",
    "Draw": "A band of polished sterling silver: cool, bright, white metal with a near-mirror finish, a dark blue-gray tarnished lip along its left side, a crisp near-white highlight running down the crest, and faint fine scratches and small hammered dimples along its length. It should read unmistakably as silver rather than steel or pewter — brighter, whiter and more reflective — even when only a sliver of the left side shows.",
    "Width": 62,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 29,
    "FadeTo": 62,
    "Sides": "left"
  },
  {
    "EdgeRecord": "wild",
    "Art": "edge-wild",
    "Draw": "A band of wet watercolor running through five colors from top to bottom — orange, yellow, green, blue, purple, in that order, each roughly a fifth of the length — where each color bleeds into the next in soft, wavy, uneven tide-lines rather than straight joins, with pigment pooling darker at the wavy edges and granulating slightly. Luminous and saturated, as though painted over a thin gilded base that glints through the paler patches.",
    "Width": 62,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 29,
    "FadeTo": 62,
    "Sides": "left"
  },
  {
    "EdgeRecord": "versatile",
    "Art": "edge-versatile",
    "Draw": "A band of three materials running from top to bottom — ivory, then steel, then granite, each roughly a third of the length — where each gives way to the next in soft, wavy, uneven seams rather than straight joins, as though the band were cast from all three at once. The ivory is smooth warm cream bone, the steel is brushed cool gray metal, the granite is speckled dark gray stone. A darker rolled lip runs down its left side through all three.",
    "Width": 62,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 29,
    "FadeTo": 62,
    "Sides": "left"
  },
  {
    "EdgeRecord": "heal",
    "Art": "edge-heal",
    "Draw": "A band of rose quartz: translucent soft rose stone, polished smooth and faintly lit from within, with milky cloudy veins drifting slowly down its length and a darker deep-rose lip along its left side. Gentle and warm; it should read as a living, mending stone rather than as metal.",
    "Width": 52,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 22,
    "FadeTo": 52,
    "Sides": "left"
  },
  {
    "EdgeRecord": "shield",
    "Art": "edge-shield",
    "Draw": "A band of hammered bronze: warm dark brown-gold metal, thick and heavy, its surface covered in overlapping round hammer marks like the face of a shield, with a raised rim along its left side studded with a row of small domed rivets running top to bottom, and a dull, battle-worn sheen.",
    "Width": 52,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 22,
    "FadeTo": 52,
    "Sides": "left"
  },
  {
    "EdgeRecord": "damage",
    "Art": "edge-damage",
    "Draw": "A band of raw red copper: hot red-gold metal with a rough, freshly struck surface, a scorched near-black lip along its left side, and glowing ember-red cracks and nicks running down its length, as if it has just been pulled from the forge.",
    "Width": 52,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 22,
    "FadeTo": 52,
    "Sides": "left"
  },
  {
    "EdgeRecord": "combo",
    "Art": "edge-combo",
    "Draw": "A band of deep garnet enamel set in dark gold: glossy wine-red vitreous enamel filling a channel, divided at irregular intervals down its length by thin gold cloisonné wires that branch and link into one another like the links of a chain, with a dark gold lip along its left side.",
    "Width": 52,
    "Inset": 4,
    "Opacity": 100,
    "FadeFrom": 22,
    "FadeTo": 52,
    "Sides": "left"
  },
  {
    "EdgeRecord": "held-damage",
    "Art": "edge-held-damage",
    "Draw": "A band of verdigris-crusted bronze: a dull blue-green patina over dark metal, matte and still, with the bare dark bronze showing through only along a worn lip on its left side. It should feel dormant — strength being held back rather than spent.",
    "Width": 44,
    "Inset": 4,
    "Opacity": 90,
    "FadeFrom": 14,
    "FadeTo": 44,
    "Sides": "left"
  },
  {
    "EdgeRecord": "held-scale",
    "Art": "edge-held-scale",
    "Draw": "A band of carved jade: dense, matte, pale sea-green stone with darker mottled clouds, carved with a shallow repeating ridge pattern that runs down its length like overlapping scales, and a darker lip along its left side. Quiet and heavy; it should feel like something kept rather than used.",
    "Width": 44,
    "Inset": 4,
    "Opacity": 90,
    "FadeFrom": 14,
    "FadeTo": 44,
    "Sides": "left"
  },
  {
    "EdgeRecord": "held-vitae",
    "Art": "edge-held-vitae",
    "Draw": "A band of dark blued steel: deep slate-blue heat-tempered metal with a faint oily iridescence, quiet and cold, set with one line of tiny crimson garnet beads running down its length just inside a darker lip along its left side, like drops of blood held in reserve.",
    "Width": 44,
    "Inset": 4,
    "Opacity": 90,
    "FadeFrom": 14,
    "FadeTo": 44,
    "Sides": "left"
  }
]
```
