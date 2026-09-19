# Rune art prompt

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
one. Anything placed on top of this picture is placed there afterwards, and the picture is meant
to carry on underneath.

## The body plan

A shape describes a body and never a style: materials, shading, outline, lighting, palette,
background and composition are settled above and are not to be restated here.

The body of the rune shoudl be a stylized drawing of what is being requested with thick pencil lines and simple approach to the subject matter with few details

## The subject

The block pasted below this prompt is one record, in JSON.

**Read `Draw` and nothing else as the subject.** It says what makes this rune its shape and not
its sibling — color, ornament, what the effect is doing to it — because the body plan above never
distinguishes two of a kind. Every other field — `Name`, `Text`, `Change`, `Target`, `Count`,
`Family` and the rest — is context for what the rune does and is never to be rendered: no title,
no caption, no rules text, no numbers anywhere in the image. If `Draw` is empty, stop and say so
rather than inventing a subject.

Name the image after the record's own key — the `RuneRecord` value — and give me a link I can
click to download it.
