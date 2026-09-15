# Other-card art prompt

High-resolution digital illustration of a single object. No text anywhere in the image — no title, no label, no caption, no signature, no
watermark. The object and its background, nothing else.

Canvas: portrait, aspect ratio exactly 5:7 (five wide to seven tall). Render as large as you work
natively; around 1060 x 1484 is what this catalog was drawn at. It is reduced to 200 x 280
afterwards, so anything finer than about a two-hundredth of the width is lost. Draw for that:
bold, simple forms and few of them.

Form: build the object out of clean, simplified, faceted geometry — flat planes meeting at crisp
angular breaks, the way a cut gem or a low-polygon model reads — rather than soft organic curves.
Spend the detail on the silhouette and on the large planes, never on fine ornament. A feature
smaller than about a fortieth of the canvas width will not survive the reduction.

Shading: smooth and continuous within each plane, with a clear material response — metal takes a
long bright specular ramp and a small near-white hot spot, stone stays matte, gems and enchanted
things glow from inside. Anti-aliasing, soft gradients and blended edges are all expected and
wanted. Do not posterize, do not dither, do not quantize to a block grid, and do not restrict the
palette to a fixed number of colors.

Outline: a dark, near-black contour all the way around the object, heavy enough to hold it off
the background once the image is at card size.

Light: from the upper left — highlights on the upper-left faces, shadow falling away to the lower
right.

Color: one dominant material hue plus one or two accents at most, so the image reads as a small
coherent scheme rather than a spread of colors.

Background: opaque and near-black, around a tenth of full brightness, tinted toward a hue that
sets the object off instead of competing with it. Essentially flat — no vignette, no second light
source, no scenery, no horizon, no ground plane. Carry a very faint texture across it, a subtle
noise or weave that is barely perceptible and never resolves into a pattern. Never transparent,
never white, never bright.

An emissive object — lava, a lit gem, a worked enchantment — may bloom softly onto the background
immediately around it. Nothing else casts anything, and there is no cast shadow.

Avoid: scenery, environments, ground planes, hands, characters, props, borders, frames, panels,
particles, sparks, smoke, motion lines, and clutter of any kind.

Priority order: readable silhouette -> recognizable object -> material shading -> coherent color
scheme -> subtle background texture.

## The object is whole

The object is **solid, finished and intact**. Nothing here is dissolving, fading, coming apart or
being unmade. A potion is a stoppered vessel and a sealed good is sealed. Draw them as objects
that would still be there tomorrow.

Wear is allowed and unmaking is not: a chipped edge, a scuffed facet, a worn stopper or a
weathered surface all read as an object with a history, where a thinning edge reads as an object
on its way out.

## Composition

Composition: one object, centered and large. It spans roughly 80% of the width and roughly 70% of
the height, sitting a little above center — about a tenth of the width left clear on each side,
and rather more clear below the object than above it.

A long object — a spear, a cleaver, a staff, a hammer — runs corner to corner on the diagonal so
that it can be drawn as large as possible. A small symmetrical one — a ring, an amulet, an orb, a
pendant — faces the viewer square on. Part of the object may run out toward an edge, a chain or a
haft or a trailing end, but it sits inside the canvas rather than bleeding off it.

**Nothing is reserved and nothing is kept clear.** There is no title, no name band, no caption
strip, no counter, no disc, no badge, no coin, no gem socket, no plaque, no banner and no slot of
any kind for text or a number. Do not leave a corner or an edge empty for one, and do not draw
one. Whatever the card puts on top of this picture, it puts there itself at draw time, and the
picture is meant to carry on underneath.

The card prints its sentence across the lower half of the picture, on a dark scrim it draws
itself. That is not a band to leave empty — draw the object through it exactly as above — but it
does mean the half carrying the recognition should be the upper one.

## The subject

The block pasted below this prompt is one record, in JSON, from whichever of these catalogs it
came from:

| Catalog             | Key field      |
| ------------------- | -------------- |
| `data/potions.json` | `PotionRecord` |
| `data/goods.json`   | `GoodRecord`   |

**A sealed good is the vessel and never its contents.** The bag, the vial and the sack each hold
four of a catalog the player cannot read before paying, so the picture is a closed container — a
cinched sack, a stoppered vial, a tied pouch — with what is inside suggested by the shape pressing
against it and never drawn loose beside it. A picture that showed the four would be saying the one
thing the mechanic exists not to say.

The placeholder brand still has no record of its own — it is built in
`internal/screens/shop_potions.go` rather than authored — so its subject is written out by hand
instead of pasted, and the name is the one the shop draws on the card.

**Read `Draw` and nothing else as the subject.** Every other field — `Name`, `Text`, `Effect`,
`Amount`, `Price`, `Size`, `Contains`, `Family` and the rest — is context for what the card does and is
never to be rendered: no title, no caption, no rules text, no numbers anywhere in the image. If
`Draw` is empty, stop and say so rather than inventing a subject.

Name the image after the record's own key and give me a link I can click to download it. A batch
lands in `.scratch/to-process-other-art/` and is filed by `go run ./tools/relicart -kind other`,
which reduces it to the card's own size and writes `Art` back onto the record — potions and sealed
goods together, since the one thing they share is this prompt.
