# Parasite art prompts

**Three blocks, where a ring has two.** The **general block** is the same for every parasite —
and is the ring prompt's general block with one word changed, deliberately, because every card in
the game that carries a picture is the same card. The **shape block** is one entry chosen from
the closed list below and is pasted verbatim too; *a prompt without one is incomplete and must
not be run*, because the general block never says what kind of creature this is. The **specific
block** is the one parasite being drawn, and is the only part written per record.

**Why a parasite needs a block a ring does not.** A ring is an object and can be described from
nothing. A parasite is a *creature*, and a generator handed "a parasite" with no body plan
invents a different animal every time — which is how a catalogue of thirty-seven ends up looking
like thirty-seven separate commissions. The shape block is what makes the set read as one
bestiary, and it is the only thing about this file that is not the ring's.

## What the card does with it

A parasite draws through `cards.WormStyle`, which is full-bleed like `cards.RingStyle`: the
picture is the whole face, and **the card does not write its name anywhere**. One thing sits on
top of the art — the **text band**, from 140 down to 265, carrying the sentence that says what
the parasite does, on a dark scrim.

So the picture is clear from the top border down to about **y 140**, and the subject belongs in
that upper two thirds. That is the one composition note that differs from a ring, whose only
overlay is the counter disc in its bottom-right corner. Everything else below is shared.

## General — identical for every parasite

**Every figure in this block is a fraction of the canvas, never a count of pixels** — see
`ring_art_prompt.MD`, which carries the measurement and the reasoning. The generator ignores
a stated canvas size and renders at its own, so a `15 px` margin was being read against an
image 5.3x larger than the one it described.

Pixel art illustration of a single creature. No text anywhere in the image — no title, no
label, no caption, no signature. The creature and its background, nothing else.

Canvas: portrait, aspect ratio exactly 5:7 (five wide to seven tall). Render at whatever
resolution you work at natively; the image is reduced to 200 x 280 afterwards, so no detail
smaller than about a two-hundredth of the width will survive. Draw for that.

Chunky pixel blocks: build every visible detail out of square blocks roughly one fortieth
of the canvas width, so the whole image reads as about 40 blocks across by 56 down. Curves,
legs, mandibles, carapace segments, outlines, highlights and texture must all be constructed
from these discrete blocks, aligned to one consistent grid. Hard edges only — no anti-aliasing,
no gradients, no soft shadows, no blur, no dithering across block edges.

Palette: at most 16 flat colours. Dark near-black outlines wherever shapes need
separation. Light source from the top-left.

The creature is overwhelmingly the focal point. It must be immediately recognizable as the body
plan named in the shape block at a glance, presented broadside — head to one side, tail to the
other — and occupy most of the central composition, with a strong, readable silhouette and a
substantial dark outline. Use pixel-art highlights and shadows to give it enough dimensionality
to read as a solid creature while keeping the geometry simple. Whatever effect the parasite
carries, the creature itself must remain clearly visible through it — do not bury it beneath
flames, sparks, smoke, frost, energy or decorative effects.

Composition: subject centred horizontally and sitting in the upper two thirds of the canvas,
filling roughly the middle 60% of the width and the band from 7% to 50% of the height down.

Keep these areas simple and low-contrast, and draw nothing in them — they are covered
after the fact by elements that are not part of this image, so nothing may be added to
represent them:

- a margin around all four sides: 7.5% of the width on the left and right edges, and 5.5%
  of the height on the top and bottom edges
- everything below 50% of the height

Background: opaque, one dark tone with subtle blocky texture. Never transparent, never
white. Background texture must remain substantially lower contrast than the creature.

Avoid environmental elements, scenery, props, characters, text of any kind, excessive
sparks, smoke or visual clutter. No host — the parasite is drawn alone, never attached to a card,
a limb or a body.

Reference: 16-bit console RPG bestiary art. It must stay legible after being reduced to
200 x 280 and viewed with no zoom, which is the size it is actually seen at.

Priority order: recognizable body plan -> convincing effect -> strong silhouette -> dimensional
pixel-art shading -> subtle background texture.

## Shape — one entry, pasted verbatim

**Closed list. Pick exactly one.** A parasite that does not fit one of these is a signal to argue
for an eleventh entry, not a licence to write a new body plan inline — an inline shape is the
whole failure this block exists to prevent.

- **Boring beetle** — a hard-shelled beetle with a long drilling snout and ridged wing cases,
  legs tucked close to a compact body.
- **Grub** — a soft, legless, deeply segmented larva with a blunt darker head capsule, curled
  into a shallow C.
- **Scarab** — a domed, armoured beetle with heavy ridged elytra and short splayed digging legs.
- **Tick** — a squat creature that is mostly swollen abdomen, with a tiny head and eight short
  legs clustered at the front.
- **Centipede** — a long, flat, many-segmented creature with a leg to each segment and a pair of
  fangs at the head.
- **Spider** — a compact cephalothorax and bulbous abdomen carried on eight angular legs, low to
  the ground.
- **Mantis** — an upright narrow thorax with folded raptorial forelimbs and a triangular head.
- **Wasp** — a sharply waisted body with folded wings held along the back and a visible sting.
- **Moth** — a furred body under broad scaled wings held flat and open to either side.
- **Casing** — not a living creature at all: a spun or chitinous cocoon, seam-ridged, sealed,
  with no visible limbs or head.

**Which family takes which is a design decision, not a guess.** The catalogue already names most
of them: the five `-bore` parasites are boring beetles, the four `-grub` are grubs, `Rock Beetle`
/ `Carapace` / `Shell` are scarabs, `Leech` / `Hunger` are ticks, and `Cocoon` / `Husk` / `Brood`
are casings. Everything else is open and is settled in the specific block.

## Specific — written per parasite

Name the shape block being used on the first line, then say what makes this one that creature and
not its sibling — colour, ornament, what the effect is doing to it. One paragraph. The general
block never names the parasite and the shape block never distinguishes two of a kind, so
everything that tells `Emberbore` from `Rimebore` lives here.

### <name>

Shape: <one entry from the list above>

<what distinguishes this one, and the effect it carries>
