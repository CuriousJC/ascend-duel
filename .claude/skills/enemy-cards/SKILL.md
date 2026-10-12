---
name: enemy-cards
description: How a creature's attack cards are classified and drawn - the form every creature card carries (stab, slash, crush), the flavor that picks its picture out of data/enemy_card_art.json, which attack goes in which flavor, how each motif leans across the forms, the naming rules, and the art pipeline for the flavor pictures. Load before authoring or renaming a creature card, adding or changing a flavor, deciding a motif's form lean, writing a flavor brief, or generating and installing enemy card art.
---

# Enemy cards

**A creature card is written in the player's language.** It carries a `Form` — stab, slash or
crush — exactly as a player's attack does, so the corner mark means the same thing on both sides
of the table and a relic that one day answers incoming slashes reads the same word the player's
slash relics do. It also carries a `Flavor`, which is its picture: a short list of shared pictures
in `data/enemy_card_art.json`, each drawn once per element. A wolf's Bite and a lion's Bite are
two cards over one picture.

## Where everything is

| Thing | Where |
|---|---|
| A creature's cards, each with `Form` and `Flavor` | `data/motifs/<motif>/motif.json`, as `data.MotifCard` |
| The flavor catalog: `Flavor`, `Form`, `Name`, `Art`, `Draw` | `data/enemy_card_art.json`, `data/enemy_card_art_data.go` |
| The refusals | `checkRecord` in `data/motifs_data.go`, `LoadEnemyCardArt` |
| The generator prompt every flavor brief is pasted after | `docs/art/enemy_card_art_prompt.MD` |
| A creature concept's flavor, for whatever draws it | `decks.EnemyFlavor` |
| Where a creature card's picture is looked up | `ui.cardArtwork` |
| The review page: every flavor in five elements, every card wearing it, each motif's lean | `go run ./tools/enemycardsheet` → `docs/sheets/enemycardsheet/` |

## The rules the loader holds

- **Every creature card has a form, and it is stab, slash or crush.** Never defend — a creature
  raises no shields — and never empty. `data.EnemyForms` is the list.
- **Every creature card names a flavor the catalog has, of its own form.** A flavor belongs to
  exactly one form; `stab-bite` exists and `crush-bite` does not. A card naming a flavor of another
  form fails the launch, so the corner mark and the picture cannot disagree.
- **A flavor key reads `<form>-<flavor>`**, and `LoadEnemyCardArt` refuses one that does not begin
  with its own form.
- **Every flavor in the catalog is worn by at least one card** —
  `TestEveryCreatureCardWearsAFlavorOfItsOwnForm`. A flavor nobody wears is five pictures that are
  never drawn.

## What the rules do with a creature's form

- **A creature still forms no hands.** It is a `SoloAttacks` duelist: every card lands its own hit.
  Its form is a fact about the blow, not a hand axis.
- **A creature's card stands on no ladder.** `Counterpart`, `Neighbor` and `Ladder` in
  `internal/combat/concept.go` walk only the player's concepts (`ladderLen`, `onLadder`), because
  the two sides share one registry. **Do not widen that scan**: a ladder read over every concept
  turns a Slice told to be crush into an ogre's club. `TestACreatureCardStandsOnNoLadder` holds it.
- **No relic reads an incoming card's form today.** Every form predicate in `relics.json` fires on
  the wearer's own cards. A relic that answers a creature's form — "a shield that eats a slash
  heals you" — is now possible and is a design decision; the `relics` skill is where it starts.
- **The achievement counters count the player's own turn only**, so creature forms add no tallies.

## Which attack goes in which flavor

`data/enemy_card_art.json` is the list of flavors and the place a new one is added. These are the
decisions the assignment follows, and they hold across the whole roster:

| Form | Flavor | What goes in it |
|---|---|---|
| stab | `stab-blade` | a held point: dagger, spear, rapier, pike, lance |
| stab | `stab-shot` | something flying: arrows, bolts, bullets, thrown shards, spit |
| stab | `stab-sting` | a small natural point: stinger, barb, thorn, needle, venom, a probe |
| stab | `stab-horn` | a big natural point: beak, horn, tusk, spike, a peck or a gore |
| stab | `stab-ray` | a piercing line: beam, gaze, bolt, spark, sigil, any spell that strikes as a line |
| stab | `stab-bite` | teeth going in: **every bite**, fang, mandible, nip, gnaw |
| slash | `slash-blade` | an edge in a hand: sword, axe, scythe, cleaver, a knife's cut |
| slash | `slash-claw` | claws or talons raking, a cat's swipe, a paw strike |
| slash | `slash-lash` | something long and flexible: whip, vine, tendril, tail |
| slash | `slash-blast` | a sweep of force: **every breath**, explosion, shockwave, roar, gust, wing-beat, storm |
| crush | `crush-club` | a blunt thing in a hand: club, hammer, mace, sceptre, mattock |
| crush | `crush-body` | the body as the weapon: fist, kick, stomp, headbutt, charge, slam |
| crush | `crush-grip` | holding and squeezing: grab, coil, constrict, engulf, smother, **every swallow** |
| crush | `crush-mass` | weight arriving: boulder, avalanche, cave-in, a falling gate, a lobbed or thrown rock |

**The fuzzy ones are decided, so do not re-argue them per card**: a bite is a stab, a breath is a
slash, a gaze is a stab, a swallow is a grip, a thrown rock is a mass. **A card whose name is
abstract** — a curse, a song, a boss's finale — takes the flavor of what the creature physically
does, and its name is changed to say so (below).

## A motif leans on a form

**A realm is chosen by motif**, so a player building against one form needs realms that suit them
and realms that punish them. Each motif leads with one form — the form with the most copies dealt
across its records — and the roster splits the leads evenly across the three:

| Lead | Motifs | Second |
|---|---|---|
| stab | Aliens, Animals, Draconics, Elves, Insects, Magic-users | mostly slash; Animals crush |
| slash | Avians, Barbarians, Big Cats, Cat-people, Plants, Vampires | mostly stab; Barbarians crush |
| crush | Dwarves, Goblins, Golems, Mutants, Ogres, Slimes | slash; Ogres and Slimes almost nothing |

- **Some motifs are near-pure, and that is a design rather than an accident.** Slimes are all
  crush. Ogres are crush with one bite. Dwarves never stab. Goblins are the deliberate generalist,
  spread almost evenly.
- **A new card should move its motif toward its lead, not away.** Read the motif's row on the
  enemy card sheet before authoring one, and after.
- **A new motif takes a lead.** Pick the form the motif has fewest leads in unless the creatures
  plainly argue otherwise, and say which when proposing it.

## Naming a creature card

- **The name says what the attack is.** "Debt Called" says nothing a player can act on; "Debt Ray"
  says a stab is coming. A name that would leave a player guessing the form is a name to change.
- **No creature card shares a name with a player card.** Check against `data/duelist_cards.json`:
  a creature's thrust is a Knife Jab, not a Jab.
- **No name means two forms anywhere on the roster.** A Maul that is a bite on one creature and a
  hammer on another is one word with two meanings; rename one. A name may span two flavors of one
  form — a Rake that is a claw and a bramble lash — because the form still agrees.
- **Short.** The name is drawn along the bottom of the card at about 80 pixels wide.

## Authoring a creature card

1. Load the `motifs` skill as well — it holds the record grammar, costs and coverage.
2. Pick the flavor from the table above, and take its form from the flavor.
3. Write `"Form"` and `"Flavor"` beside `"Cost"`, before `"Copies"`, as every card does.
4. Name it by the rules above.
5. `go test ./data/...`, then `go run ./tools/enemycardsheet` and read the motif's row and the
   flavor's list. Regenerate `go run ./tools/motifsheet` too: its deck strips show names and marks.

## Adding a flavor

**A flavor is five pictures**, so add one only when a set of cards genuinely draws as none of the
existing ones — never for one creature's signature attack. Split a flavor rather than stretch it
when its picture would have to show two different things.

1. A record in `data/enemy_card_art.json`: `Flavor` as `<form>-<name>`, `Form`, a short `Name`, an
   empty `Art`, and a `Draw` brief.
2. Move the cards that belong to it, in the motif files.
3. The test above fails until some card wears it.

## The pictures

**One picture per flavor per weight per element**, the attacking part alone, made of the same
colored vapor as the player's card art, with no creature behind it: a sting is only the stinger, a
bite only the jaws. The place to iterate is `docs/art/enemy_card_art_prompt.MD` for what is true of
every flavor, each record's `Draw` for the blow at medium weight, and its `Ladder` for the other
three.

- **The weight is how hard the hit lands, and it is derived, never authored.**
  `data.EnemyCardWeight` reads it off the card's multiplier: light under 0.75x, medium under 1.5x,
  heavy under 2.75x, crushing above. So a creature's 0.5x Probe and its 3x Spine Burst wear two
  pictures of one sting, and a card retuned across a line changes picture with nothing edited.
  **Do not move a line without re-reading the roster**: the lines sit in the gaps between the
  amounts the cost ladder produces, and a line through a cluster splits cards that should match.
- **A flavor's `Weights` lists the weights drawn in all five elements**, written by the install
  step; an unlisted weight draws its cards with no picture.

- **The five elements are five drawings, not one recolored.** Same blow, size, place and
  direction; fresh silhouette and edges, and the vapor's character changes with the element. The
  prompt's table says how.
- **To generate**: paste the prompt, then a batch JSON — `{"Batch": "<name>", "Records": [...]}`,
  each record the catalog's `Flavor`, `Form`, `Name` and `Draw` plus a `Weight`, its
  `WeightBrief` (the `Ladder` line, or `Draw` for medium), and `Elements` and `Files` lists. Four
  records of five elements is one agent's 20 pictures; keep a flavor's weights in as few batches as
  the packing allows, and the flavors most cards wear first. It comes back as `<Batch>.zip`
  holding a `<Batch>/` folder of `<Flavor>-<weight>-<element>.png`, so two agents drawing one
  flavor never collide.
- **To review**: each batch folder goes in `artreview/enemy-card-<label>/`, then
  `go run ./tools/artcompare -catalog enemy-card`, which draws every flavor in five elements as a
  hand card. A raw take is the wrong size for its weight until it is installed; fit a copy with the
  same rule as `tools/relicart` before judging size.
- **To install**: put the picks in `.scratch/to-process-enemy-card-art/` and run
  `go run ./tools/relicart -kind enemy-card`. It reduces each to 200x280 and **scales the figure to
  fill its weight's box** — `data.EnemyCardFieldFor`, a share of `data.EnemyCardArtField` set by
  `data.EnemyCardWeightShare`, cropped to what it draws, up or down, centered — so the ladder steps
  evenly whatever size a batch was drawn at. Then it adds a
  weight to the flavor's `Weights` (and its stem to `Art`) only once all five elements of that
  weight are in `assets/enemy-card/` — a weight still short says which element it is waiting for
  and keeps drawing no picture. It round-trips `data/enemy_card_art.json` through encoding/json,
  so the file stays machine-formatted: two-space indent, keys in struct order.
- **`Art` is the stem, and the weight and element are appended**: `ArtKey(weight, element)` is
  `<Art>-<weight>-<element>`. Empty `Art` or an unlisted weight draws the card as it always
  looked, with no picture — never a placeholder, for `data.DefaultCardArt`'s reason.
- **`TestEveryEnemyCardArtIsAFigureInTheField`** fails on a listed weight missing an element, a
  picture not at 200x280, or a pixel outside the field — the last is what catches a picture copied
  in by hand.
- **Then** `go run ./tools/enemycardsheet` — it errors on an `Art` stem with a missing element
  file, where the game silently draws no picture.

## Still open

- **Whether `stab-horn` and `stab-bite` stay two flavors.** They are close cousins, and stab has
  the most flavors of the three forms.
