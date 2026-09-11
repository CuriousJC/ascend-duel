# Relics still to draw

**The subject paragraphs, one per relic, for the 114 relics with no artwork yet.** The prompt they
all share is in `card_art_prompt.MD` and is pasted verbatim — with its **whole card** composition
block — above whichever of these is being generated; this file is only ever the last line.

## How to use it

1. Fill in the `Draw:` line under a relic — what the object _is_ and what the effect is doing to
   it, in one paragraph. The prompt never names the object, so this is the only place the
   generator learns there is a relic on fire rather than a relic made of teeth.
2. Paste the prompt and its whole-card composition block, then the paragraph, into the generator.
3. Save the result as `.scratch/to-process-relic-art/<key>.png` — the **key is the filename**, and
   nothing else about the file matters.
4. Run `go run ./tools/relicart`. It reduces each picture to the card's own size, commits it to
   `assets/relic/`, sets `"Art"` on the record in `data/relics.json`, strikes the relic from this
   file, recomputes the counts in every heading above, and moves the original to
   `.scratch/processed-rings/`. `-n` says what it would do and writes nothing.

**The key is the filename and the filename is the key**, so `aftershock-ring` here is
`assets/relic/aftershock-ring.png` and `"Art": "aftershock-ring"`. It is already the record id, so
there is nothing to invent.

**A relic card carries no title** _(owner's call, 2026-09-11)_, so the picture is doing all of the
work of saying which relic this is. Two relics that scale damage on two different elements have to
be told apart by their art alone — which is the argument for making the _element_ the loudest
thing in a picture whenever a relic has one.

**Nothing here is struck by hand any more** _(2026-09-11)_. A worklist that keeps finished entries
is a worklist nobody trusts the length of, and the four steps that used to end an entry — reduce,
commit, record, strike — were done once per relic, a hundred and twenty times, with two of them
failing _silently_: an `"Art"` left empty just draws `default-relic.png`, and an entry left standing
is a relic that gets drawn twice. `tools/relicart` does all four off the filename, and refuses a file
whose stem names no record rather than filing it somewhere nothing will look.

## Common — 40 relics

### Cleaver

- **Key:** `cleaver-ring`
- **Full name:** Cleaver Ring
- **Rule:** Every Cleave deals double DMG.
- **Draw:**

### Coiled

- **Key:** `coiled-ring`
- **Full name:** Coiled Ring
- **Rule:** Deals 5 more DMG for each crush card
  kept in hand.
- **Draw:** tightly braided jade metal relic

### Confluent

- **Key:** `element-four-of-a-kind-hand-ring`
- **Full name:** Confluent Ring
- **Rule:** A Elemental Four of a Kind deals 7 more DMG before the multiplier.
- **Draw:**

### Convergent

- **Key:** `element-full-house-hand-ring`
- **Full name:** Convergent Ring
- **Rule:** A Elemental Full House deals 7 more DMG before the multiplier.
- **Draw:**

### Cutter

- **Key:** `cutter-ring`
- **Full name:** Cutter Ring
- **Rule:** Every Cut deals double DMG.
- **Draw:**

### Doubled

- **Key:** `concept-two-pair-hand-ring`
- **Full name:** Doubled Ring
- **Rule:** A Card Two Pair deals 4 more DMG before the multiplier.
- **Draw:**

### Elemental

- **Key:** `element-five-of-a-kind-hand-ring`
- **Full name:** Elemental Ring
- **Rule:** A Elemental Five of a Kind deals 14 more DMG before the multiplier.
- **Draw:**

### Fencer

- **Key:** `fencer-ring`
- **Full name:** Fencer Ring
- **Rule:** Every Lunge deals double DMG.
- **Draw:**

### Forged

- **Key:** `form-three-of-a-kind-hand-ring`
- **Full name:** Forged Ring
- **Rule:** A Form Three of a Kind deals 3 more DMG before the multiplier.
- **Draw:**

### Frostbank

- **Key:** `frostbank-ring`
- **Full name:** Frostbank Ring
- **Rule:** Deals 5 more DMG for each ice card
  kept in hand.
- **Draw:**

### Grinder

- **Key:** `grinder-ring`
- **Full name:** Grinder Ring
- **Rule:** Every Pulverize deals double DMG.
- **Draw:**

### Headsman

- **Key:** `headsman-ring`
- **Full name:** Headsman Ring
- **Rule:** Every Sever deals double DMG.
- **Draw:**

### Heart

- **Key:** `heart-ring`
- **Full name:** Heart Ring
- **Rule:** +5 HP, and 5 more for every fight won.
- **Draw:**

### Heavy

- **Key:** `heavy-ring`
- **Full name:** Heavy Ring
- **Rule:** Every crush card deals double DMG.
- **Draw:**

### Hungry

- **Key:** `hungry-ring`
- **Full name:** Hungry Ring
- **Rule:** Two prizes after a fight instead of one.
- **Draw:**

### Impaler

- **Key:** `impaler-ring`
- **Full name:** Impaler Ring
- **Rule:** Every Impale deals double DMG.
- **Draw:**

### Knocker

- **Key:** `knocker-ring`
- **Full name:** Knocker Ring
- **Rule:** Every Tap deals double DMG.
- **Draw:**

### Laden

- **Key:** `concept-full-house-hand-ring`
- **Full name:** Laden Ring
- **Rule:** A Card Full House deals 8 more DMG before the multiplier.
- **Draw:**

### Lancer

- **Key:** `lancer-ring`
- **Full name:** Lancer Ring
- **Rule:** Every Thrust deals double DMG.
- **Draw:**

### Lone

- **Key:** `high-card-hand-ring`
- **Full name:** Lone Ring
- **Rule:** A High Card deals 2 more DMG before the multiplier.
- **Draw:**

### Might

- **Key:** `might-ring`
- **Full name:** Might Ring
- **Rule:** +10 DMG for every fight.
- **Draw:**

### Momentum

- **Key:** `momentum-ring`
- **Full name:** Momentum Ring
- **Rule:** Every card gains 0.2x DMG each turn you raise no shields. A defend card resets it.
- **Draw:**

### Paired

- **Key:** `form-two-pair-hand-ring`
- **Full name:** Paired Ring
- **Rule:** A Form Two Pair deals 3 more DMG before the multiplier.
- **Draw:**

### Panoply

- **Key:** `form-five-of-a-kind-hand-ring`
- **Full name:** Panoply Ring
- **Rule:** A Form Five of a Kind deals 12 more DMG before the multiplier.
- **Draw:**

### Perfect

- **Key:** `concept-five-of-a-kind-hand-ring`
- **Full name:** Perfect Ring
- **Rule:** A Card Five of a Kind deals 16 more DMG before the multiplier.
- **Draw:**

### Poised

- **Key:** `poised-ring`
- **Full name:** Poised Ring
- **Rule:** Deals 5 more DMG for each stab card
  kept in hand.
- **Draw:**

### Potential

- **Key:** `potential-ring`
- **Full name:** Potential Ring
- **Rule:** Deals 5 more DMG for each lightning card
  kept in hand.
- **Draw:**

### Prodder

- **Key:** `prodder-ring`
- **Full name:** Prodder Ring
- **Rule:** Every Poke deals double DMG.
- **Draw:**

### Quartered

- **Key:** `concept-four-of-a-kind-hand-ring`
- **Full name:** Quartered Ring
- **Rule:** A Card Four of a Kind deals 10 more DMG before the multiplier.
- **Draw:**

### Sheathed

- **Key:** `sheathed-ring`
- **Full name:** Sheathed Ring
- **Rule:** Deals 5 more DMG for each slash card
  kept in hand.
- **Draw:**

### Skirmisher

- **Key:** `skirmisher-ring`
- **Full name:** Skirmisher Ring
- **Rule:** Every Nick deals double DMG.
- **Draw:**

### Slicer

- **Key:** `slicer-ring`
- **Full name:** Slicer Ring
- **Rule:** Every Slice card
  deals double DMG.
- **Draw:**

### Smasher

- **Key:** `smasher-ring`
- **Full name:** Smasher Ring
- **Rule:** Every Smash deals double DMG.
- **Draw:**

### Smoulder

- **Key:** `smoulder-ring`
- **Full name:** Smoulder Ring
- **Rule:** Deals 5 more DMG for each fire card
  kept in hand.
- **Draw:**

### Soul Taker

- **Key:** `soul-taker-ring`
- **Full name:** Soul Taker Ring
- **Rule:** A won room pays 5 more vitae.
- **Draw:**

### Striker

- **Key:** `striker-ring`
- **Full name:** Striker Ring
- **Rule:** Every Strike deals double DMG.
- **Draw:**

### Trined

- **Key:** `concept-three-of-a-kind-hand-ring`
- **Full name:** Trined Ring
- **Rule:** A Card Three of a Kind deals 5 more DMG before the multiplier.
- **Draw:**

### Twinflame

- **Key:** `element-two-pair-hand-ring`
- **Full name:** Twinflame Ring
- **Rule:** A Elemental Two Pair deals 3 more DMG before the multiplier.
- **Draw:**

### Twinned

- **Key:** `pair-hand-ring`
- **Full name:** Twinned Ring
- **Rule:** A Pair deals 2 more DMG before the multiplier.
- **Draw:**

### Wellspring

- **Key:** `wellspring-ring`
- **Full name:** Wellspring Ring
- **Rule:** Deals 5 more DMG for each arcane card
  kept in hand.
- **Draw:**

## Uncommon — 51 relics

### Aftershock

- **Key:** `aftershock-ring`
- **Full name:** Aftershock Ring
- **Rule:** Every crush card lands twice, both at full DMG.
- **Draw:**

### Backdraft

- **Key:** `backdraft-ring`
- **Full name:** Backdraft Ring
- **Rule:** Every fire card lands twice, both at full DMG.
- **Draw:**

### Braced

- **Key:** `braced-ring`
- **Full name:** Braced Ring
- **Rule:** Defend cards cost 1 AP less. +15 HP for every fight.
- **Draw:**

### Bruising

- **Key:** `bruising-ring`
- **Full name:** Bruising Ring
- **Rule:** Crush attacks apply WEIGHTED status to target. WEIGHTED enemies deal 25% less DMG.
- **Draw:**

### Crown

- **Key:** `crown-ring`
- **Full name:** Crown Ring
- **Rule:** Every 3 AP card deals 2.5x DMG.
- **Draw:**

### Dual Wield

- **Key:** `dual-wield-ring`
- **Full name:** Dual Wield Ring
- **Rule:** A pair built from two
  different forms deals 3x DMG.
- **Draw:**

### Dust Storm

- **Key:** `dust-storm-ring`
- **Full name:** Dust Storm Ring
- **Rule:** Every earth card is dealt as a lightning card.
- **Draw:**

### Echo

- **Key:** `echo-ring`
- **Full name:** Echo Ring
- **Rule:** Your first attack lands 3 times: full DMG, then 2/3, then 1/3.
- **Draw:**

### Enflamed

- **Key:** `enflamed-ring`
- **Full name:** Enflamed Ring
- **Rule:** Fire cards gain 0.1x DMG every time a fire attack lands. Grows while worn.
- **Draw:**

### Fire of Life

- **Key:** `fire-of-life-ring`
- **Full name:** Fire of Life Ring
- **Rule:** Fire cards gain 0.1x DMG
  for every 10 vitae you hold.
- **Draw:**

### Firestorm

- **Key:** `firestorm-ring`
- **Full name:** Firestorm Ring
- **Rule:** Every lightning card is dealt as a fire card.
- **Draw:**

### Flurry

- **Key:** `flurry-ring`
- **Full name:** Flurry Ring
- **Rule:** Every stab card lands twice, both at full DMG.
- **Draw:**

### Forked

- **Key:** `forked-ring`
- **Full name:** Forked Ring
- **Rule:** Every lightning card lands twice, both at full DMG.
- **Draw:**

### Frostbite

- **Key:** `frostbite-ring`
- **Full name:** Frostbite Ring
- **Rule:** Every fire card is dealt as an ice card.
- **Draw:**

### Frostbitten

- **Key:** `frostbitten-ring`
- **Full name:** Frostbitten Ring
- **Rule:** Ice cards gain 0.1x DMG every time a ice attack lands. Grows while worn.
- **Draw:**

### Frozen Lightning

- **Key:** `frozen-lightning-ring`
- **Full name:** Frozen Lightning Ring
- **Rule:** Every lightning card is dealt as an ice card.
- **Draw:**

### Fulgurite

- **Key:** `fulgurite-ring`
- **Full name:** Fulgurite Ring
- **Rule:** Every lightning card is dealt as an earth card.
- **Draw:**

### Glacier

- **Key:** `glacier-ring`
- **Full name:** Glacier Ring
- **Rule:** Every ice card is dealt as an earth card.
- **Draw:**

### Granite

- **Key:** `granite-ring`
- **Full name:** Granite Ring
- **Rule:** Earth cards gain 0.1x DMG every time a earth attack lands. Grows while worn.
- **Draw:**

### Heat Lightning

- **Key:** `heat-lightning-ring`
- **Full name:** Heat Lightning Ring
- **Rule:** Every fire card is dealt as a lightning card.
- **Draw:**

### Hexbolt

- **Key:** `hexbolt-ring`
- **Full name:** Hexbolt Ring
- **Rule:** Every arcane card is dealt as a lightning card.
- **Draw:**

### Hexfire

- **Key:** `hexfire-ring`
- **Full name:** Hexfire Ring
- **Rule:** Every arcane card is dealt as a fire card.
- **Draw:**

### Hexfrost

- **Key:** `hexfrost-ring`
- **Full name:** Hexfrost Ring
- **Rule:** Every arcane card is dealt as an ice card.
- **Draw:**

### Hexstone

- **Key:** `hexstone-ring`
- **Full name:** Hexstone Ring
- **Rule:** Every arcane card is dealt as a earth card.
- **Draw:**

### House of Pain

- **Key:** `house-of-pain-ring`
- **Full name:** House of Pain Ring
- **Rule:** Every Full House
  deals 3x DMG.
- **Draw:**

### Landslide

- **Key:** `landslide-ring`
- **Full name:** Landslide Ring
- **Rule:** Every earth card lands twice, both at full DMG.
- **Draw:**

### Leyline

- **Key:** `leyline-ring`
- **Full name:** Leyline Ring
- **Rule:** Every earth card is dealt as an arcane card.
- **Draw:**

### Lithium

- **Key:** `lithium-ring`
- **Full name:** Lithium Ring
- **Rule:** Lightning cards gain 0.1x DMG every time a lightning attack lands. Grows while worn.
- **Draw:**

### Magma

- **Key:** `magma-ring`
- **Full name:** Magma Ring
- **Rule:** Every earth card is dealt as a fire card.
- **Draw:**

### Meltdown

- **Key:** `meltdown-ring`
- **Full name:** Meltdown Ring
- **Rule:** Every ice card is dealt as a fire card.
- **Draw:**

### Obsidian

- **Key:** `obsidian-ring`
- **Full name:** Obsidian Ring
- **Rule:** Every fire card is dealt as an earth card.
- **Draw:**

### Pairing

- **Key:** `pairing-ring`
- **Full name:** Pairing Ring
- **Rule:** Every pair deals 2x DMG.
- **Draw:**

### Permafrost

- **Key:** `permafrost-ring`
- **Full name:** Permafrost Ring
- **Rule:** Every earth card is dealt as an ice card.
- **Draw:**

### Pinning

- **Key:** `pinning-ring`
- **Full name:** Pinning Ring
- **Rule:** Stab attacks apply SHOCKED status to target. SHOCKED enemies have 25% chance to miss.
- **Draw:**

### Pounding

- **Key:** `pounding-ring`
- **Full name:** Pounding Ring
- **Rule:** Crush cards gain 0.1x DMG every time a crush attack lands. Grows while worn.
- **Draw:**

### Quickening

- **Key:** `quickening-ring`
- **Full name:** Quickening Ring
- **Rule:** Stab cards gain 0.1x DMG every time a stab attack lands. Grows while worn.
- **Draw:**

### Recursion

- **Key:** `recursion-ring`
- **Full name:** Recursion Ring
- **Rule:** Every arcane card lands twice, both at full DMG.
- **Draw:**

### Rend

- **Key:** `rend-ring`
- **Full name:** Rend Ring
- **Rule:** Every slash card lands twice, both at full DMG.
- **Draw:**

### Runefrost

- **Key:** `runefrost-ring`
- **Full name:** Runefrost Ring
- **Rule:** Every ice card is dealt as an arcane card.
- **Draw:**

### Sharp as Ice

- **Key:** `sharp-as-ice-ring`
- **Full name:** Sharp as Ice Ring
- **Rule:** Every ice slash card
  deals 3x DMG.
- **Draw:**

### Sharpening

- **Key:** `sharpening-ring`
- **Full name:** Sharpening Ring
- **Rule:** Slash cards gain 0.1x DMG every time a slash attack lands. Grows while worn.
- **Draw:**

### Shatter

- **Key:** `shatter-ring`
- **Full name:** Shatter Ring
- **Rule:** Every ice card lands twice, both at full DMG.
- **Draw:**

### Spellbolt

- **Key:** `spellbolt-ring`
- **Full name:** Spellbolt Ring
- **Rule:** Every lightning card is dealt as an arcane card.
- **Draw:**

### Struck by Lightning

- **Key:** `struck-by-lightning-ring`
- **Full name:** Struck by Lightning Ring
- **Rule:** Every lightning stab card
  deals 3x DMG.
- **Draw:**

### Sundering

- **Key:** `sundering-ring`
- **Full name:** Sundering Ring
- **Rule:** Slash attacks apply WEAKENED status to target. WEAKENED enemies take 2x DMG from everything.
- **Draw:**

### Swarm

- **Key:** `swarm-ring`
- **Full name:** Swarm Ring
- **Rule:** Every 1 AP card deals 3x DMG.
- **Draw:**

### Thundersnow

- **Key:** `thundersnow-ring`
- **Full name:** Thundersnow Ring
- **Rule:** Every ice card is dealt as a lightning card.
- **Draw:**

### Triplicate Form

- **Key:** `triplicate-form-ring`
- **Full name:** Triplicate Form Ring
- **Rule:** Every Three of a Kind
  deals 3x DMG.
- **Draw:**

### Unravelled

- **Key:** `unravelled-ring`
- **Full name:** Unravelled Ring
- **Rule:** Arcane cards gain 0.1x DMG every time an arcane attack lands. Grows while worn.
- **Draw:**

### Weight of the Earth

- **Key:** `weight-of-the-earth-ring`
- **Full name:** Weight of the Earth Ring
- **Rule:** Every earth crush card
  deals 3x DMG.
- **Draw:**

### Witchfire

- **Key:** `witchfire-ring`
- **Full name:** Witchfire Ring
- **Rule:** Every fire card is dealt as an arcane card.
- **Draw:**

## Rare — 23 relics

### Atrophy

- **Key:** `atrophy-ring`
- **Full name:** Atrophy Ring
- **Rule:** Every 3 AP attack is dealt as its 2 AP version.
- **Draw:**

### Blightfire

- **Key:** `blightfire-ring`
- **Full name:** Blightfire Ring
- **Rule:** Arcane attacks WEAKEN and BURN the target.
- **Draw:**

### Brittle

- **Key:** `brittle-ring`
- **Full name:** Brittle Ring
- **Rule:** Ice attacks CHILL and WEAKEN the target.
- **Draw:**

### Caldera

- **Key:** `caldera-ring`
- **Full name:** Caldera Ring
- **Rule:** Earth attacks WEIGH and BURN the target.
- **Draw:**

### Cold

- **Key:** `cold-ring`
- **Full name:** Cold Ring
- **Rule:** Ice cards cost 1 AP less.
- **Draw:**

### Dirty

- **Key:** `dirty-ring`
- **Full name:** Dirty Ring
- **Rule:** Earth cards cost 1 AP less.
- **Draw:**

### Ebb & Flow

- **Key:** `ebb-and-flow-ring`
- **Full name:** Ebb & Flow Ring
- **Rule:** Every card gains 0.2x DMG
  for each shield card
  you play. Grows while worn.
- **Draw:**

### Eerie

- **Key:** `eerie-ring`
- **Full name:** Eerie Ring
- **Rule:** Arcane cards cost 1 AP less.
- **Draw:**

### Erode

- **Key:** `erode-ring`
- **Full name:** Erode Ring
- **Rule:** Every 2 AP card is dealt as its 1 AP version.
- **Draw:**

### Hefted

- **Key:** `hefted-ring`
- **Full name:** Hefted Ring
- **Rule:** Crush cards cost 1 AP less.
- **Draw:**

### Hoarfrost

- **Key:** `hoarfrost-ring`
- **Full name:** Hoarfrost Ring
- **Rule:** Ice attacks CHILL and WEIGH the target.
- **Draw:**

### Millstone

- **Key:** `millstone-ring`
- **Full name:** Millstone Ring
- **Rule:** Arcane attacks WEAKEN and WEIGH the target.
- **Draw:**

### Oak

- **Key:** `oak-ring`
- **Full name:** Oak Ring
- **Rule:** Every Four of a Kind
  deals 4x DMG.
- **Draw:**

### Onslaught

- **Key:** `onslaught-ring`
- **Full name:** Onslaught Ring
- **Rule:** Every card costs 1 AP less. Lose 25% of your HP.
- **Draw:**

### Palsy

- **Key:** `palsy-ring`
- **Full name:** Palsy Ring
- **Rule:** Lightning attacks SHOCK and WEAKEN the target.
- **Draw:**

### Pentacle

- **Key:** `pentacle-ring`
- **Full name:** Pentacle Ring
- **Rule:** Every Five of a Kind
  deals 5x DMG.
- **Draw:**

### Rampant

- **Key:** `rampant-ring`
- **Full name:** Rampant Ring
- **Rule:** Every blow deals 1 more DMG
  for each vitae you hold.
- **Draw:**

### Static

- **Key:** `static-ring`
- **Full name:** Static Ring
- **Rule:** Lightning cards cost 1 AP less.
- **Draw:**

### Tapered

- **Key:** `tapered-ring`
- **Full name:** Tapered Ring
- **Rule:** Stab cards cost 1 AP less.
- **Draw:**

### Tremor

- **Key:** `tremor-ring`
- **Full name:** Tremor Ring
- **Rule:** Earth attacks WEIGH and SHOCK the target.
- **Draw:**

### Warm

- **Key:** `warm-ring`
- **Full name:** Warm Ring
- **Rule:** Fire cards cost 1 AP less.
- **Draw:**

### Whetted

- **Key:** `whetted-ring`
- **Full name:** Whetted Ring
- **Rule:** Slash cards cost 1 AP less.
- **Draw:**

### Whittle

- **Key:** `whittle-ring`
- **Full name:** Whittle Ring
- **Rule:** Every 1 AP attack is dealt as its 0 AP version.
- **Draw:**
