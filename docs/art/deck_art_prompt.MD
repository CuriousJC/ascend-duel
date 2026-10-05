# Deck art prompt

High-resolution digital illustration of the **back of a playing card** — one repeating pattern
covering the whole face. No text anywhere in the image — no title, no label, no number, no
caption, no signature, no watermark. The pattern, nothing else.

Canvas: **portrait, aspect ratio exactly 5:7** — five wide to seven tall, around 1060 x 1484.

The pattern: **one motif repeated on a regular grid, filling the whole canvas edge to edge on all
four sides**. Every repeat is the same object at the same size, lit the same way, on the same
grid. The grid is centered on the canvas, so whatever the edges cut off is cut off equally left
and right and equally top and bottom. The record says how many repeats run across; the rows
follow from that and continue to the top and bottom edges.

**It reads the same turned upside down.** A card back is seen from both ends of the table, so the
pattern has half-turn symmetry: rotated 180 degrees about the canvas center, it is the same
picture. Light still comes from the upper left on every repeat.

**No focal point.** Nothing in the middle is bigger, brighter or different from its neighbors —
no central emblem, no medallion, no one stone set apart. The back says whose deck this is by its
pattern as a whole, and every card in the deck wears it, so a spot that draws the eye is a spot
the eye finds on every card in the pile.

The pattern is **fully opaque and at full strength everywhere**, right up to every edge. Do not
fade it, feather it, vignette it or soften any edge — the game cuts the rounded corners and draws
its own thin rim round it.

Scale: the back is reduced to **200 x 280 pixels** on the table, to about **150 x 210** on the
smaller pile between fights, and it is seen squeezed edge-on as a card flips. One repeat at four
across is about 50 pixels wide on the table and 37 on the small pile, so each repeat has to read
as its object from its silhouette and its two or three biggest tones; fine facet lines and small
glints are a bonus, not the thing it is recognized by. Anything smaller than about a fiftieth of
the width is lost.

The ground between the repeats is **near-black**, hue-tinted toward the motif, with barely
perceptible noise. The edge of the face matters: cards stand in a pile three deep, offset a few
pixels, so the outermost rows and columns must be as rich as the middle.

Rendering: smooth, continuous shading with a clear material response — a gem takes crisp bright
facets, near-white glints on the upper-left faces and deep color in the lower ones; metal takes a
long bright specular ramp. Each repeat has a near-black contour so it holds against the ground at
small sizes. Anti-aliasing and soft gradients are expected. Do not posterize, do not dither, do
not quantize to a block grid.

Avoid: a card outline, a frame, a border running round four sides, rounded corners, a central
emblem, a single larger motif, scenery, characters, card suits, letters, numerals, runes, and
anything that breaks the grid.

Priority order: an even repeating pattern edge to edge -> each repeat recognizable at 50 pixels ->
half-turn symmetry -> rich material rendering.

The subject — what repeats, how many across, and what sets it — is the record's own `Draw` in
`data/decks.json`, pasted after this block.

**Make three options for every record**, so one can be chosen. Each option follows the whole of
this prompt and the whole of the record's `Draw`; they differ only in what the brief leaves open —
the proportions of the cut, how the facets catch the light, how wide the setting lines run, and
where within the brief's color range the stones sit. Three near-copies of one picture is one
option, not three.

Name each image after its record's own key — the `DeckRecord` value, like `rainbow-gem` — with
`-a`, `-b` and `-c` for the three options: `rainbow-gem-a`, `rainbow-gem-b`, `rainbow-gem-c`. Give
me a single zip file of everything, so I can download it all at once — never a link per file.

## The color words

**A color word in the brief names one of these ramps**, the game's own colors, which the back has
to sit beside.

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

Not for the generator. The chosen option drops its `-a`/`-b`/`-c` suffix, and the batch goes in `.scratch/to-process-deck-art/` and
`go run ./tools/relicart -kind deck` files it: reduced to the card's own **200 x 280** and written
to `assets/deck/<record>.png`, with `Art` set on the record. The game lays it inside a one-pixel
neutral rim, clipped to the card's rounded corners, and that one picture is every back the game
draws: the draw pile in the hand row at full size, the smaller pile on the shop, reward and sealed
good screens, and every card mid-flip as it is dealt. `go run ./tools/cardsheet` draws every deck's
back at both sizes beside a face, which is where the art is judged; a deck with no picture yet
draws the plain dark back.
