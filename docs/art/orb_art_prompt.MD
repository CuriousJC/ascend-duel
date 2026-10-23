# Orb art prompt

**A family of orbs**: small glossy spheres for a dark-fantasy card game, each one a badge with a
number written in white across its middle. One orb already exists — **an orb of blood in a thin
glass skin**, described below — and every orb here is **its sibling**: the same size, the same
construction, the same light, in a different substance.

The work is split into **three batches**, drawn separately. **You will be told which batch to draw;
draw only that one.** Everything above the batches applies to all three.

---

## The prompt

High-resolution digital illustration of **one round orb**, on a **fully transparent** background,
for a dark-fantasy card game. A game icon with real material to it — not a flat symbol, not a
photograph.

### The orb that already exists

Every orb here has to sit beside this one as one family, so it is the reference for size, light and
finish:

- **A perfect sphere of dark liquid blood held in a thin, clear glass skin**, like a bauble with no
  neck. Deep crimson body (`#A81A2A`), blackish red (`#4A0710`) in the shadowed lower right, a
  brighter red (`#E0485A`) where light passes through near the lower left edge.
- **Two or three soft, darker swirls** drifting inside, gentle and low-contrast, so it reads as one
  round red thing first.
- **One crisp highlight** — a soft curved window of reflected light on the glass at the upper left —
  and **a much smaller second glint** just below it, both a soft warm white.
- **A thin near-black contour** around the sphere, and nothing else in the picture.

It is rich and glossy, halfway between a painted game icon and a real object: smooth shading, soft
inner depth, crisp anti-aliased edges. Every orb below has that same finish.

### Every orb

- **A perfect sphere, seen straight on**, centered, **filling about 90% of the canvas**, with a
  little clear space all round. Every orb in every batch is the same size and in the same place, so
  that swapping one file for another never makes the orb move or grow.
- **Lit from the upper left**, exactly like the blood orb: **one crisp highlight**, a soft curved
  window of reflected light at the upper left, and a much smaller second glint just below it. The
  body shades darker toward the lower right. The highlights are a soft warm white, not pure white.
- **A thin near-black contour** around the whole sphere, about **a fiftieth of the canvas**, so it
  holds its edge on a pale background. Thin and even — not a heavy comic-book outline.
- **The middle stays mid-to-dark.** A **bold white number with a black outline** will be laid
  across the center of every orb — `3`, `12`, `40` — so the central third of the sphere must not be
  pale or busy. Keep the brightest light to the upper-left highlight and the rim. **A shiny metal
  orb reflects a dark room**, not a bright sky, so its middle stays dark too.
- **It must survive being shrunk.** The orb is seen at about 160 pixels across, at about 64, and on
  the playing cards at **32**. At 32 pixels it must still read as a shiny ball of its substance: the
  main highlight bold enough to stay, the texture coarse enough that it does not average into mud.
  Anything finer than about an eighth of the canvas is lost at that size — let the texture be bold
  and few rather than fine and many.
- **Nothing else**: no stand, no cap, no setting, no frame, no ring around it, no chain, no rune,
  letter or number drawn on it, no cast shadow, no glow, no halo, no sparkle around it, no particles
  outside the sphere.

### Where it sits

Every orb is seen against two pale grounds: a **warm off-white `#F0EFEA`** card, and a **light
slate blue `#7EACE4`** screen. It must read clearly on both.

### The two options

**Every orb is drawn twice, as option `a` and option `b`**, so the substance can be chosen by
looking. The two options of one orb are **the same substance done two ways** — the batch tables say
what each way is. They are not two copies of one picture.

### The colors

Where a table gives colors, `core` is the orb's body, `dark` is the side away from the light, and
`light` is where light catches or passes through. They are a guide to the hue, not flat fills — the
material's own texture and shading go on top.

### The canvas

Each file is **exactly 256 x 256**, a **transparent** PNG with an alpha channel. Work as large as
you like and reduce once to 256 on the way out.

Avoid: text, letters, numbers, symbols on the orb, a stand or setting, anything outside the sphere,
backgrounds, scenery, glow, halos, sparkle, a cast shadow, a pale or busy center, and any heavy black
outline.

---

## Batch 1 — the elements and the marble

Sixteen files.

### The five element orbs

One orb per element, in the element's colors.

| Option | What it is |
| ------ | ---------- |
| `a` | **A glass orb filled with the element itself**, built exactly like the blood orb: a thin clear glass skin holding the element's living matter, moving slowly inside — see the table below. Low-contrast movement; it reads as one round colored thing first. |
| `b` | **A solid polished gemstone sphere in the element's color**, like a cut and polished marble of colored stone: smooth, glossy, faintly translucent at the edges, with a soft inner depth and no moving matter. |

| File stem | Element | What is inside option `a` | dark | core | light |
| --------- | ------- | ------------------------- | ---- | ---- | ----- |
| `orb-fire` | fire | slow rolling flame and embers | `#9E2F13` | `#F06A24` | `#FFB24A` |
| `orb-ice` | ice | drifting frost and pale ice crystals | `#245A86` | `#4E9ED6` | `#A8DDF4` |
| `orb-lightning` | lightning | a slow coil of crackling electricity | `#A98200` | `#F2D13D` | `#FFF38A` |
| `orb-earth` | earth | swirling green mineral dust, like powdered jade | `#285E35` | `#4E9B57` | `#8BCB78` |
| `orb-arcane` | arcane | drifting purple vapor with faint glowing veins | `#4C2A78` | `#8A5BC7` | `#C8A4F2` |

**The yellow lightning orb is the hardest.** Its middle must stay dark enough for a white number —
push its body toward `dark` in the center and keep the bright yellow to the rim and the coil.

### The rainbow marble

**A marble that carries all five element colors at once** — orange, yellow, green, blue and purple,
the five `core`s above — in ribbons that wind around the sphere. It is drawn at **three levels of
saturation**, each as option `a` and `b`:

| Option | What it is |
| ------ | ---------- |
| `a` | **A glass swirl marble**, the classic kind: clear glass with twisted ribbons of color running through it, curving around the sphere from pole to pole. |
| `b` | **A polished stone marble**, like veined marble rock: a soft stone body with the five colors running through it as broad, soft veins and clouds. |

| File stem | Saturation |
| --------- | ---------- |
| `orb-marble-muted` | **Muted**: the five colors heavily grayed, dusty and soft, as if faded by age — clearly still five hues, but quiet. |
| `orb-marble-soft` | **Soft**: halfway — gentle pastel-leaning colors, clearly a rainbow, not loud. |
| `orb-marble-vivid` | **Vivid**: the five colors at full strength. |

**The three levels are the same marble**: same ribbons, same veins, in the same places. Only the
saturation changes. The center stays mid-to-dark at every level — let the ribbons cross the middle
in their darker tones.

---

## Batch 2 — the metals and the organics

Twenty-four files. Every orb is a solid sphere of the material.

| File stem | Material | dark | core | light | Option `a` | Option `b` |
| --------- | -------- | ---- | ---- | ----- | ---------- | ---------- |
| `orb-gold` | Gold | `#6E4A0C` | `#C9A227` | `#F4DC7A` | mirror-polished yellow gold, reflecting a dark room | soft hammered gold, like crumpled gold leaf: a warm buttery surface broken into faint flat facets |
| `orb-silver` | Silver | `#4E5258` | `#A9AEB4` | `#E6E9EC` | mirror-polished silver, reflecting a dark room | antique silver: soft satin metal with darker gray tarnish settled in a faint cloudy pattern |
| `orb-platinum` | Platinum | `#5A5D60` | `#B4B6B8` | `#E2E3E4` | satin-brushed platinum: a soft, even, slightly grainy matte metal, warmer and grayer than steel | mirror-polished platinum, cooler and whiter than silver |
| `orb-copper` | Copper | `#5A2412` | `#B8653E` | `#E8A37C` | polished rosy-orange copper with a faint hammered texture | **glowing red copper**: copper heated from within, a deep ember-red glow under the metal |
| `orb-bronze` | Bronze | `#4A3016` | `#9C6F3E` | `#D6AE7A` | brushed bronze: a warm brown-gold metal with fine straight brushing lines | aged bronze, darker, with a few soft patches of pale green verdigris |
| `orb-iron` | Iron | `#2A2C2E` | `#55585B` | `#8A8D90` | dark cast iron: matte, grainy and slightly pitted, a dull charcoal gray | rusted iron: patches of orange-brown rust over dark charcoal metal |
| `orb-steel` | Steel | `#59636B` | `#9DA8B0` | `#D8DEE2` | brushed steel: a cool blue-gray metal with fine straight brushing lines and a streak of reflected light | **dark blued steel**: steel heat-blued to a deep blue-black with a faint purple sheen |
| `orb-hematite` | Hematite | `#0E0F10` | `#2C2E31` | `#7A7E83` | mirror-polished hematite: near-black metallic stone, like a dark mirror, with a cool gray sheen | satin hematite with a faint dark-red undertone |
| `orb-ivory` | Ivory | `#9B8B68` | `#DCCFAE` | `#FFF3D2` | smooth polished ivory: a warm creamy white, soft and waxy, faintly translucent at the rim | ivory showing its faint cross-hatched grain, a fine diamond lattice in slightly darker cream |
| `orb-bone` | Bone | `#8A7A58` | `#D4C4A0` | `#F2E8CE` | polished bone: a pale cream with a scatter of small darker pores and faint streaks | aged bone, yellowed toward tan, with a few hairline cracks |
| `orb-wood` | Wood | `#2E1A0E` | `#6B3F22` | `#A87148` | a turned sphere of dark walnut: rich red-brown wood with flowing grain lines curving around it, oiled to a soft gloss | burl wood: the same brown wood with its grain in swirls and small dark eyes |
| `orb-pearl` | Pearl | `#8E8A86` | `#E4E0DA` | `#FFFFFF` | a lustrous white pearl with a soft pink-and-green iridescent sheen moving across it | **a dark pearl**: charcoal black with a peacock-green and purple iridescent sheen |

**The pale materials — silver, platinum, ivory, bone and the white pearl — are the hard ones.**
Their middles must still carry a white number: shade them so the center sits in their mid-tone,
toward `dark`, and the palest light stays at the highlight and the upper-left rim.

---

## Batch 3 — the stones and gems

Twenty-six files. Every orb is a solid polished sphere of the stone.

| File stem | Material | dark | core | light | Option `a` | Option `b` |
| --------- | -------- | ---- | ---- | ----- | ---------- | ---------- |
| `orb-jade` | Jade | `#1E4A2A` | `#4F9A5E` | `#9CD3A2` | translucent mid-green jade with soft paler clouding and faint veins inside | dark imperial jade, deep green and glassy, light glowing through its edge |
| `orb-indigo` | Indigo | `#12124A` | `#2B2E9A` | `#5A60D0` | deep saturated indigo-blue, matte and velvety, like dyed felt | glossy deep indigo enamel, smooth as lacquer |
| `orb-sandstone` | Sandstone | `#8A6E4A` | `#D2B88E` | `#EEDDBE` | fine-grained warm sandstone: a pale tan stone covered in small sandy specks | sandstone in soft horizontal bands of tan, cream and rust |
| `orb-granite` | Granite | `#2E3032` | `#8A8C8E` | `#D0D2D4` | polished granite: bold black, white and gray speckles packed together, glossy | rough-honed granite: the same speckle, matte and slightly gritty |
| `orb-marble` | Marble | `#5A5E62` | `#D8DADC` | `#FAFAFA` | white marble with soft gray veins wandering across it | black marble with fine white veins |
| `orb-malachite` | Malachite | `#0A3A26` | `#1E8A56` | `#5CCB8E` | malachite: vivid green stone in bold wavy concentric bands of dark and light green | malachite cut so its bands circle an eye at the center |
| `orb-moonstone` | Moonstone | `#6E7884` | `#C8D0D8` | `#EAF2FF` | milky translucent moonstone with a soft blue glow floating inside it | rainbow moonstone: the same milky stone, its inner glow shifting blue to violet to gold |
| `orb-bloodstone` | Bloodstone | `#0E2A1E` | `#24503A` | `#4E7E62` | dark green stone scattered with small red flecks | dark green stone with larger, denser red spatters |
| `orb-obsidian` | Obsidian | `#050505` | `#16161A` | `#5A5A64` | glossy volcanic black glass, with a sharp white reflection and faint smoky streaks | obsidian with a faint gold-and-rainbow sheen under the surface |
| `orb-quartz` | Quartz | `#8A9098` | `#DDE2E8` | `#FFFFFF` | a clear crystal ball with soft white inner fractures and planes catching the light | **smoky quartz**: the same crystal ball, tinted a deep smoky brown |
| `orb-slate` | Slate | `#1E2226` | `#3E444A` | `#6E767E` | matte dark blue-gray slate with faint flaky layers | wet-polished slate, dark and glossy |
| `orb-amber` | Amber | `#6A2E04` | `#D8820E` | `#FFC24A` | clear golden-orange amber, glowing where light passes through, with soft inner fractures | dark cognac amber, deep reddish-brown, with a few tiny trapped bubbles |
| `orb-lapis` | Lapis | `#0E1E5A` | `#1E3FA0` | `#4E72D8` | deep royal-blue lapis lazuli with small flecks of gold scattered through it | deep blue lapis with gold flecks and soft white veins |

**The clear quartz is the one orb that is see-through.** Draw it as a crystal ball that shows a
little of whatever is behind it, with its fractures and the contour carrying its shape — it is in
the set to be looked at, even if a white number may not read on it.

---

## Before you hand anything back

1. **Reduce every orb to 32 x 32 on off-white `#F0EFEA` and on slate blue `#7EACE4`.** Is each still
   a shiny ball of its substance, with its highlight and its edge visible?
2. **Lay a bold white `12` with a black outline across the middle of each, at 64 pixels.** Can the
   number be read on every orb?
3. **Line the batch up in a row at the same size.** Same size, same place, same light from the upper
   left, same thin contour — one family, and a family the blood orb described above would fit into?
4. **Flip through the `a` and `b` of one orb.** Same size and place, the substance done two ways?
5. **Batch 1 only: flip through the three marbles of one option.** Nothing moves; only the
   saturation changes.
6. **Is everything outside the sphere fully transparent**, with no halo?

Name each file `<stem>-<option>.png` — for example `orb-fire-a.png`, `orb-marble-muted-b.png`,
`orb-copper-b.png` — and give me a single zip file of the whole batch so I can download it all at
once — never a link per file.
