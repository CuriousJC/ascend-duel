# The material language of relic art

**Status: a proposal for review. Nothing here is decided.** This reads the `Draw` field of
every record in `data/relics.json` back to itself and asks what the metals, gems and
materials have come to mean. Most of the language is already there and was arrived at one
family at a time; what follows separates the part that is consistent from the part that
only looks consistent, and says what the empty `Draw` fields would inherit if the language
were written down.

Nothing here is read by the engine. `Draw` is the brief handed to the art generator and
`Art` is the filename it produces; both are ignored at runtime, exactly as a status's
`Badge` is. So this is a document about authoring, and the only cost of a rule being wrong
is a picture that says the wrong thing.

---

## 1. What is already consistent

### Colour is the element, everywhere, without exception

| Element | Gem | Raw matter |
|---|---|---|
| fire | red | red/orange flame, burning coals, molten rock |
| ice | blue | pale blue ice, frost, icicles, shards |
| lightning | yellow | yellow electricity, ball lightning, sparks |
| earth | green | brown stone, rock, pebbles, mud |
| arcane | purple | purple flame, purple energy, glowing veins |

This is the strongest rule in the file and it holds across seven families — the five
elemental status rings, the ten paired status rings, the five growth amulets, the five
robes, the twenty elemental flips, the five elemental damage rings and the five jars.

**The one wrinkle is earth, and it is probably correct rather than a bug.** Earth's *gem*
is green (`weighted`, `growth-earth`, `hoarfrost`'s pair) and earth's *matter* is brown
(`earth`, `discount-earth`, `jar-earth`). Green reads as a gemstone and brown does not, so
the split is defensible — but it should be stated rather than left to be rediscovered:
**cut earth is green, raw earth is brown.**

### Shape is the form

| Form | Weapon shape | Gemstone |
|---|---|---|
| stab | spear, needle, foil, point | marble |
| slash | scimitar, sickle, blade, edge | malachite |
| crush | mace, club, hammer, weight | moonstone |

`repeat-stab` / `repeat-slash` / `repeat-crush` and `growth-stab` / `growth-slash` /
`growth-crush` use the three stones and agree with each other. The form statuses and the
form weapons use the three shapes.

### The form statuses are the cleanest thing in the catalogue

`sundering`, `bruise` and `strike` say **shape for the form the relic triggers on, colour
for the element of the status it applies** — an obsidian scimitar with a purple gem is
"slash applies WEAKENED". Two facts, two channels, no collision. It is worth naming as the
pattern every two-axis relic should follow.

### The motif says the verb

| Motif | What it means | Families |
|---|---|---|
| a ring wrapped into a circle twice | the card lands twice | Elemental repeats, Form repeats |
| a scale amulet, one side weighed down | grows while worn | Elemental growth, Form growth |
| a clear glass jar with a lid | the relic holds a kind of card | Elements held, Forms held |
| a ring of one substance, no band, no gem | the element itself, undiluted | Elemental damage |
| N ouroboros rings intertwined | a hand of N | Hand multipliers |
| a robe | a discount | Elemental discounts |
| a ring with a line along one edge | erosion, a rung lost | Tier demotions |
| a gem with N smaller gems beside it | a primary set and a kicker pair | the three ring families, Paired statuses |

None of these collide. The ring families and the paired statuses both already use "one big
gem, two small ones" for the same idea — a counted set plus a pair — which is the sort of
coincidence worth making deliberate.

---

## 2. What only looks consistent

### The axis is said three different ways

The three hand axes each have a band material, and then two other families name the same
axes with entirely different materials:

| Axis | Ring band | Jar lid | Growth amulet |
|---|---|---|---|
| concept | jade | — | — |
| form | indigo | indigo | gold |
| element | sandstone | copper | silver |

Indigo agrees with itself across the ring and the jar, which is the only half of this that
works. Element is sandstone as a band, copper as a lid and silver as an amulet chain; form
is indigo twice and then gold. **A player cannot learn from this**, and the fix is cheap
because it is a one-line edit per record: pick one material per axis and use it for the
band, the lid, the chain and anything else that has to name an axis.

The obvious candidates are the ring bands, since they are the largest family and already
established: **jade = concept, indigo = form, sandstone = element**. That makes the five
elemental jar lids sandstone rather than copper, and the five `growth-*` amulets sandstone
rather than silver — which also frees silver up, see below.

### The counted gem is a different stone in each family

| Family | Counted set | Kicker pair |
|---|---|---|
| Jade rings (concept) | diamond | black gem |
| Indigo rings (form) | golden gem | silver gem |
| Sandstone rings (element) | turquoise | ruby |
| Hand multipliers | bronze ring | silver ring |

Four families doing one job with four vocabularies. Since the band already names the axis,
**the gems do not have to** — they could be one stone throughout (diamond for the counted
set, black for the kicker), and then a jade ring with three diamonds and two black gems
reads as "concept full house" with nothing memorised beyond the band.

The counter-argument is that a ring is small on a 200x280 card and three diamonds on jade
may be harder to tell from three diamonds on sandstone than three turquoise are. That is
an art question rather than a design one, and it is the reason to leave this one open
rather than settle it here.

### The weapon ladder does not climb

The fifteen weapons are the three concept ladders, 0 AP to 4 AP, and the materials are
currently unordered:

| Rung | stab | slash | crush |
|---|---|---|---|
| 0 AP | sharpened stick — wood | bread knife — steel | wooden spoon — wood |
| 1 AP | rusty dagger — rusted iron | hunting knife — steel | brass knuckles — brass |
| 2 AP | epee — steel | katana — silver, damascus | hammer — copper and silver |
| 3 AP | wood spear — copper tip | cleaver — silver and copper | warhammer — silver, black steel |
| 4 AP | lance — obsidian, silver | axe — black steel, silver | maul — stone and wood |

Read down a column and it mostly climbs; read across a row and it does not. A bread knife
at rung 0 is steel while a rung-3 spear is wood, and the rung-4 maul is stone and wood
while the rung-4 axe is black steel. **A materials ladder would make the AP cost readable
off the picture** — the player sees what a relic is worth before reading its line:

```
0 AP   wood
1 AP   rusted iron, brass
2 AP   steel
3 AP   copper and silver
4 AP   black steel and obsidian
```

Every weapon already carries one of these; what changes is which. **The argument against is
that the art is already drawn** — this is the biggest family in the catalogue and six of
them would need redrawing, so it is the most expensive item here as well as the one with
the most to gain.

### Gold and copper each say two things

- **gold** is the band of the five elemental status rings, the chain of the three form
  growth amulets, and the metal of `bulwark` and `banker`.
- **copper** is the band of the ten paired status rings, the lid of the five elemental
  jars, the body of the five twisted elemental rings, and a handle on three weapons.

Copper's reading in the twisted rings and the paired statuses is the same idea — **an
alloy, two things mixed** — which is a good meaning for a relic that does two things at
once. If that is the rule, copper should leave the jar lids and the weapon handles. Gold
currently means nothing at all and could take the one group that has no material: the
standing stats and the spoils, the relics that are about the player rather than about a
card.

### `scald` names the wrong element, and `storm` the wrong metal

The paired status rings are otherwise exact — the big gem is the attacking element, the two
small gems are the second status's element, bleeding into the band:

- `scald` is "Fire attacks BURN and CHILL", so its two small gems should be **blue**. They
  are authored yellow. The steam in the same sentence is right; the gems are not.
- `storm` is the only one of the ten on a **gold** band; the other nine are copper.

Both are one-word fixes and neither record has art committed yet.

---

## 3. What the empty briefs would inherit

Twenty-one records have no `Draw`. Under the language above, most of them write themselves:

| Record | What the language implies |
|---|---|
| `whetted` / `hefted` / `tapered` | robes, as the elemental discounts are — but in the form's stone rather than an element's colour: malachite green, moonstone grey, marble white |
| `weight-of-the-earth` / `struck-by-lightning` / `sharp-as-ice` | the form-status pattern exactly: the form's weapon shape in the element's colour |
| `swarm` / `crown` | a rung is a material, so a swarm of the 1 AP material and a crown of the 3 AP one |
| `demotion-one` / `-two` / `-three` | already rings with an eroding line; the line's colour should be the rung it erodes **to** |
| `momentum` / `ebb-and-flow` | shields, and the game has no shield motif yet — the one genuinely new picture in the list |
| `echo` | the twice-wrapped ring is taken; three of something, each fainter than the last |
| `onslaught` / `braced` | a discount is a robe, and these are discounts on everything |
| `might` / `bulwark` / `heart` | `bulwark` is already a gold armband; the other two should match it as gold worn on the body rather than on a hand |
| `rampant` / `fire-of-life` | vitae, which has no material yet |
| `dual-wield` | two of the form stones, one of each |
| `banker` / `soul-taker` / `hungry` | `banker` is already a gold band with red gems; spoils are about the purse, and gold is the obvious metal if it stops being the status band |

**Two things the language does not cover at all: shields, and vitae.** Both need a material
chosen rather than derived, and both come up more than once.

---

## 4. The shortest version

If only one thing is taken from this:

- **Colour is the element. Shape is the form.** Both already hold — write them down.
- **One material per axis**, used by the ring band, the jar lid and the amulet chain alike.
  Jade, indigo and sandstone are the candidates.
- **Materials are a ladder, and the ladder is the AP cost.** Wood, rust, steel, silver,
  black steel — the most valuable change and the most expensive, since the weapons are
  already drawn.
- **Fix `scald`'s gems and `storm`'s band.** Neither has art yet.
- **Pick a material for shields and one for vitae** before the next batch of briefs.
