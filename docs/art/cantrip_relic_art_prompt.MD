# Cantrip-relic art prompt

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

Outline: **no drawn outline.** A spirit is light, and a line around it makes it a sticker. Its
edge is where its own glow meets the dark ground, soft and luminous; only the eyes, horns, claws
and other small solid details may carry a thin dark edge.

Light: from the upper left — highlights on the upper-left faces, shadow falling away to the lower
right.

Color: one dominant material hue plus one or two accents at most, so the image reads as a small
coherent scheme rather than a spread of colors.

Background: **fully transparent** — a PNG with an alpha channel, where everything that is not the
spirit or its own light has an alpha of zero. **Not a checkerboard drawn to look transparent**, not
a dark wash, not a black box. The glow and the haze are part of the spirit: draw them as soft,
partly see-through light in the spirit's own color fading to nothing, exactly as they would fade
into a dark ground. If you cannot output real transparency, use an opaque near-black ground instead,
around a tenth of full brightness — **perfectly flat and plain**, no texture, no noise, no vignette,
no second light. See "The two uses" below for why.

An emissive object — lava, a lit gem, a worked enchantment — may bloom softly onto the background
immediately around it. Nothing else casts anything, and there is no cast shadow.

Avoid: scenery, environments, ground planes, hands, characters, props, borders, frames, panels,
particles, sparks, smoke, motion lines, and clutter of any kind.

Priority order: readable silhouette -> recognizable object -> material shading -> coherent color
scheme -> subtle background texture.

## The color words

**A color word in the brief names one of these ramps.** "A purple orb" is an orb in arcane;
"an ivory haft" is a haft in the stab material. These are the game's own colors and the picture has
to sit beside cards drawn in them, so the word means this range here rather than whatever the
generator would otherwise reach for.

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

The element name and the color word are the same instruction — `arcane` and "purple" both mean
that row, and so do `ivory` and "cream". The last three rows are the game's three form materials:
**ivory is stab, steel is slash, granite is crush.**

**How the three columns are used.** `core` is the object's own color, carrying the large planes and
whatever the eye reads the hue as at a glance. `light` is the faces turned toward the light and the
top of the specular ramp — **the lit part is always the object's own hue made lighter, never a warm
color and never gold.** `dark` is the faces turned away and the color the shadow settles into,
above the near-black contour, which stays near-black.

**The ramp directs the picture; it does not fence it.** The word's colors should be the
**dominant** ones — the large planes and whatever the eye reads the object's color as — so that the
picture is unmistakably that word at card size. Other colors are welcome wherever the subject
wants them: an accent, a metal fitting, a warm glint inside a gem, a rope, a contour. What is
wrong is **drift** — an object whose dominant hue has slid to a neighbouring one, or washed out far
enough that the word stops being readable. A color word not in this table (brown, black, white,
gold, copper, bone) is an ordinary word and carries no ramp.

## The two uses

The game shows this one picture twice. **As the relic the cantrip puts on**, it is the whole card,
laid over the game's own near-black ground. **As the cantrip's scroll**, it is laid at about half
size on the open sheet of a cream parchment scroll. So the spirit is drawn once, for the dark, and
the game supplies the ground each time — which is why the canvas carries none of its own, and why
the near-black fallback has to be one flat, plain color the game can lift the spirit off cleanly.

Nothing about the spirit itself changes for this. Draw it as you would on a dark ground, glowing and
ethereal: no drawn outline, no hard edges added to make it stand out.

## The unmaking

The spirit is **ephemeral**: it is made of light and coming apart into mist, and a soft haze hangs
around it. It is a charm worn for one fight and gone at the end of it, so it must read as something
happening to this animal — edges going to wisps and motes, a tail or a wingtip thinning until the
background shows through — and not as a filter laid over the whole picture. **The silhouette
survives it.** The animal stays recognizable at a glance; the fading takes the edges and the
extremities, never the head or the body.

**Keep the unmaking light.** The game fades this card further on its own, toward its foot, and
breaks its border, so a picture already mostly gone ends up as nothing on screen. The middle of the
animal stays solid and fully colored.

**The glow and the haze stay close to the animal.** They bloom softly right around it and fade to
nothing well before the canvas edge: everything that is not the animal or its own light is empty.

## Composition

One animal, centered and large. It spans roughly 75% of the width and roughly 70% of the height,
sitting a little above center, with empty space all round it — **nothing touches or runs off the
canvas edge**, the haze included, because the scroll lays the whole spirit on its sheet and
anything at the edge would be cut square.

Facing the viewer or in a strong three-quarter view, in a pose that reads as one shape at half
size.

## The subject

The block pasted below this prompt is one cantrip record, in JSON, keyed by `CantripRecord`. Its one
picture is the cantrip-relic's spirit, briefed in its `Relic` block; the scroll is made from the same
picture afterwards.

**Read `Relic.Draw` and nothing else as the subject.** Every other field — `Name`, `Text`, `Family`, `Rules`
and the rest, at either level — is context for what the cantrip does and is never to be rendered:
no title, no caption, no rules text, no numbers anywhere in the image. If `Relic.Draw` is empty,
stop and say so rather than inventing a subject.

Name each image after its record's own key with `-relic` on the end — `cantrip-chill-relic` for the
record `cantrip-chill` — and give me the file itself if there is only one, and a single zip file of
everything if there is more than one, so I can download it all at once — never a link per file.
