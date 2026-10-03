# Screen backdrop art

You are painting a backdrop for a screen of a card game: the picture that fills the whole screen
behind the game's cards. This file is how every such backdrop is painted. It comes with a JSON
**record** describing one picture, and you paint that record by the rules below.

**The record says what is in the picture. This file says how it is painted.** Where the two seem
to disagree, the record decides the subject and this file decides everything else: the canvas, the
rendering, the focus, the light and what may never appear.

---

## Reading the record

A record has these fields:

| Field     | What it tells you                                                                       |
| --------- | --------------------------------------------------------------------------------------- |
| `Art`     | the filename: `<Art>.jpg`                                                               |
| `Draw`    | the picture itself: what fills the canvas, its mood and its colors                      |
| `Anchors` | optional: rectangles the picture is placed against — each a `Name`, a `Kind`, and `X`, `Y`, `W`, `H` in pixels |

`Backdrop`, `Name` and `Screens` are labels; ignore them.

**`Draw` may place something against an anchor** — a light behind a card, a portal round a button,
a glow in a place. When it does, it names the anchor by its `Name`, and that rectangle is where the
thing goes. Those positions are the point of such a picture: paint them where the record says.

An anchor's `Kind` says what stands there:

| Kind     | What it is                                                                              |
| -------- | --------------------------------------------------------------------------------------- |
| `card`   | a playing card the game lays over the picture, opaque, covering its rectangle completely |
| `button` | a button the game lays over the picture, opaque, covering its rectangle completely      |
| `place`  | part of the picture itself: nothing covers it, and the shape the brief names fills it  |

A record with no `Anchors` places nothing; the picture is even across the canvas.

---

## Every screen backdrop

A painted **backdrop for a card game**: one full-screen picture that the game's cards are laid
over. It is a ground rather than a subject — nothing in it competes with the cards for attention.

### Canvas

**Exactly 1920x1080**, landscape 16:9, fully opaque, **delivered as a JPEG at quality 90** (or the
nearest your tools allow) — not a PNG. Work at 3840x2160 if you can. **Whatever the generator
returns, normalize every final file** to that: convert a PNG to JPEG, flatten any transparency, and
resize to exactly 1920x1080. If the aspect is not 16:9, crop evenly from the edges rather than
squashing — and **never crop or shift a picture whose record has anchors**:
generate it again, because the position is the point.

Coordinates anywhere in a record are pixels from the top-left of the 1920x1080 canvas.

### The cards and buttons

**Cards and buttons are opaque and cover what is under them completely.** Each card is a crisp,
detailed illustration with hard near-black contours on an off-white or dark face, bordered in one of
the game's colors; each button is a bevelled dark bar with a word on it. The backdrop has to sit
*behind* every one of them:

- **Paint straight through every `card` and `button` rectangle.** The picture is opaque to all four
  edges and carries on under them. Nothing is left blank where one will stand, and nothing is drawn
  as if one belongs there — no slot, no frame, no plinth, no outline.
- **What the player sees of anything placed against a card or a button is around it**, never behind
  it. A light behind a card shows above, below and beside the card, so it has to be larger than the
  card it stands behind, not the same size.
- **Two things close together stay two.** When a record places a thing against each of two
  neighboring anchors, the gap between them stays visibly between the two things.
- **A `place` is painted, not covered**: the shape the record names fills that rectangle, centered
  on it, and is seen whole except where a `card` or `button` anchor stands inside it.

### Color

**A screen backdrop belongs to no element unless its record says so.** These screens are reached
from every part of the game, and every card on them is marked in one of the five element colors.
A backdrop leaning toward one element makes that element's cards quieter and the others louder.

**Unless the record names a color, the picture is neutral gray**, drawn from the steel ramp below
with granite for its deepest shading — a cool gray, but plainly gray rather than pale blue.

**When a record names a color, it means one of these ramps.** These are the game's own colors, and
the backdrop sits under cards drawn in them.

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

**"All five" or "every element" in a record means the first five rows together, none leading** —
the screen is still owned by no element, it is being fair to all of them at once.

### Key, focus and light

**Key and mood are the record's to choose.** What is wrong at any key is **extremes over large
areas**: no large patch of pure black, no large blown-out white, no single blinding light.

**Soft focus, moderate contrast.** Render it as if seen through haze: forms are recognisable, edges
are soft, fine detail is suppressed, and the contrast is gentle next to the cards'. A sharp,
detailed backdrop fights every card on the screen.

Light: falling mainly from **above and slightly to the upper left**, soft rather than harsh — the
direction every card and button in the game is lit from.

Rendering: **painterly digital illustration** — smooth blended gradients, soft brushwork,
anti-aliased throughout. Not photographic, not pixel art, not flat vector, not posterized. No
dithering, no visible block grid, no banding in the gradients.

### Nobody is here

**The picture is empty of people and creatures.** No person, no figure, no silhouette, no animal,
no monster, no statue, no face — including a face or a figure that fog, cloud, smoke or light
happens to form. If a wisp starts to read as a face, a hand, an animal or a letter, smooth it out.
The cards are the only things on this screen that are about anyone.

### Reserve nothing

Do not draw a frame, a border, a panel, a plaque, a title bar, a vignette, a darkened edge or an
empty box anywhere. Paint the picture straight through to all four edges.

Avoid: legible text of any kind, letters, numbers, signatures, watermarks, logos, user-interface
elements, cards, playing cards, icons, lens flare, heavy vignetting, motion lines and speech
bubbles.

---

## Before you hand it back

1. **Cover every `card` and `button` anchor with a gray box** and look at what is left. Is the
   picture still whole, and is everything the record placed against one visible around its box?
2. **Is everything the record placed where `Anchors` says?** Measure it: within about 10 px.
3. **Squint at it.** Is there a large patch of pure black, a blown-out white, or a blinding light?
4. **Is it soft?** Held beside a crisp illustrated card, does the backdrop sit behind it?
5. **Is the color the record's?** Gray when the record names none, and leaning toward no single
   element unless the record asks for one.
6. **Is it empty?** No figure, creature, statue or face anywhere, formed or drawn.
7. **Is it a JPEG, opaque to all four edges, exactly 1920x1080**, with no frame, vignette or
   reserved patch?

**Hand back the bare `<Art>.jpg`.**
