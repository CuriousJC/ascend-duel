# The ring catalogue, 2026-09-06

**Snapshot date: 2026-09-06.** 137 rings, 800 shelf tickets.
Produced by `python .claude/skills/ring-balance/classify.py` — every figure below is derived from
`data/rings.json` and `data/statuses.json` and can be re-run. See
[the skill](../../.claude/skills/ring-balance/SKILL.md) for the taxonomy.

**Nothing in the repo simulates a duel**, so this counts records. Every conclusion below is
judgement standing on a count, and the ones that are judgement say so.

---

## 1. Counts are not shares, and the gap is the story

The shelf draws on rarity tickets — 10 / 4 / 1 — so a category living at one tier has a share that
its record count does not predict. `--shelf`:

| Category | Rings | % of catalogue | **% of a shelf seat** |
|---|---:|---:|---:|
| offense | 91 | 66.4% | **78.9%** |
| enabler | 20 | 14.6% | 10.0% |
| defense | 18 | 13.1% | 6.8% |
| **tempo** | 13 | 9.5% | **2.0%** |
| growth | 11 | 8.0% | 6.6% |
| economy | 3 | 2.2% | 3.8% |
| drawback | 2 | 1.5% | 0.6% |

**Four in five shelf seats are an offense ring.** That is not automatically wrong — this is a game
about hitting things — but it means every other axis of a build is competing for one seat in five,
and the shelf is three seats.

**Tempo is the sharpest distortion in the file.** Thirteen rings, and twelve of them are rare:

```
Atrophy, Cold, Dirty, Eerie, Erode, Hefted, Static, Tapered, Warm, Whetted, Whittle  (rare)
Onslaught (rare, also drawback)          Braced (uncommon)
```

Nine and a half per cent of the catalogue is 2.0% of what a player sees. **Twelve of the game's
twenty-six rares are cost discounts**, so nearly half the rare tier is one idea. A player who
wants to make their deck cheaper has, in practice, no route to it: the whole rare tier is 3.2% of
a shelf seat.

That is the one finding here I would call a finding rather than an observation. The rest are
questions.

---

## 2. Flat and scaling are evenly split — until you look at where each one lives

| Payload | Rings | % shelf | What it means |
|---|---:|---:|---|
| multiplicative | 47 | 40.8% | xDMG — compounds with itself, left to right |
| flat | 44 | 39.6% | +DMG, ±cost, +HP — adds |
| enabler | 20 | 10.0% | the `set-element` flip grid |
| status | 18 | 5.2% | applies a status; classified by the status's own effect |
| stateful | 11 | 6.6% | an accumulator; priced on the top of the tower, not fight one |
| repeat | 9 | 4.5% | re-runs the landing pipeline, so it compounds with everything per-landing |

Even overall. Crossed with `breadth`, it stops being even:

| | uncond. | element | form | concept | tier | hand | positional |
|---|---:|---:|---:|---:|---:|---:|---:|
| flat | 8 | 10 | 7 | **0** | 3 | **17** | 0 |
| multiplicative | 4 | 14 | 8 | **15** | 2 | 6 | 0 |
| repeat | 0 | 5 | 3 | 0 | 0 | 0 | 1 |
| status | 0 | 15 | 3 | 0 | 0 | 0 | 0 |
| stateful | 3 | 5 | 5 | 0 | 0 | 0 | 0 |
| enabler | 0 | **20** | 0 | 0 | 0 | 0 | 0 |

Three things fall out:

- **Every concept ring is multiplicative, and all fifteen are common.** There is no flat concept
  ring in the game. Given a concept covers four cards where a form covers twelve, a *flat* concept
  ring is the natural cheap version — a big flat number on a narrow trigger — and nobody has
  written one. Open question, not a defect.
- **Every hand ring is per-blow, and flat outnumbers multiplicative 17 to 6.** That is the ladder
  ring family: a common flat `add-hand-damage` and a scarcer `scale-hand-damage` per rung. It is
  the most deliberately-shaped corner of the catalogue and it reads that way.
- **The enabler column is one cell.** All twenty flips are element-keyed. A flip keyed on *form*
  — "slash cards are drawn as fire" — is legal grammar with nothing in it. See §5.

---

## 3. Scope: almost everything happens inside a blow

| Scope | Rings | Reads |
|---|---:|---|
| per-card | 69 | `card-cost`, `card-damage`, `card-drawn` |
| per-blow | 66 | `attack-lands`, `blow-formed` |
| per-fight | 9 | `deck-built`, `fight-start` |
| per-run | 4 | `fight-won`, `prizes-dealt` |
| per-turn | 2 | `turn-taken` |

**Four rings in the whole game pay out at the run level** — Banker, Hungry, Soul Taker and Heart —
and **two fire on `turn-taken`**, Momentum and Ebb & Flow. Those two moments are
nearly unused vocabulary.

That is worth reading against §1: the reason a build has nothing but offense to buy is partly that
the moments where a non-offense ring would naturally live are the emptiest ones in the grammar.

---

## 4. Element parity is close to exact — with one asymmetry

Rings keyed on each element, by category:

| | offense | defense | tempo | enabler | growth | drawback | total |
|---|---:|---:|---:|---:|---:|---:|---:|
| fire | 8 | 2 | 1 | 4 | 1 | 1 | 13 |
| earth | 6 | 3 | 1 | 4 | 1 | . | 13 |
| ice | 6 | 3 | 1 | 4 | 1 | . | 13 |
| lightning | 6 | 3 | 1 | 4 | 1 | . | 13 |
| **arcane** | 7 | **1** | 1 | 4 | 1 | . | **12** |

**Arcane is a defense ring short.** Earth, ice and lightning each field three; arcane fields one
(Millstone). The reason is visible in the status catalogue — the three defensive
statuses are CHILLED, SHOCKED and WEIGHTED, seventeen rings apply one, and arcane appears in
exactly one of them (Millstone, WEIGHTED). Ice reaches three, lightning three, earth three, fire
two. So a player building arcane is building the one colour
with no defensive line.

**Whether that is a hole or a personality is a design call, not a count.** Arcane is the one colour
whose rings are almost entirely offensive — seven of twelve, against a defensive line of one — and
making every element identical would erase whatever distinguishes them. The finding is that the asymmetry exists
and appears to be emergent rather than authored.

Forms are flatter — crush, slash and stab are 7 rings each, defend is 3 — and **slash is the only
form with no defense ring at all**, which is the `--holes` output. Given slash cards deal damage
and defend cards raise shields, a defensive slash ring wants a reason to exist before it is
written.

---

## 5. The empty cells worth asking about

`--holes` reports two. The rest of this section is read off the build axis by hand.

1. **`element-elementalist` is the one rung of the eighteen with no ring.** Every other rung has
   at least one — most have two, a common flat and a scarcer multiplier — and the top elemental
   rung has neither. That looks like an omission rather than a decision, and it is the cheapest
   thing on this page to fix: a record in `rings.json`, no Go.
2. **`set-element` is element-keyed only.** Twenty flips, all `Element=X → Y`. A form-keyed flip is
   legal today and would read differently — it recolours by *what a card does* rather than by what
   it already is, which is a build enabler rather than a colour-fixer.
3. **Economy touches no card at all.** All three rings are unconditional, because the three economy
   verbs read the purse and the prize row. Nothing is wrong; it is worth knowing that economy
   cannot be given an element without a new verb.
4. **Drawback is two rings.** Onslaught (rare) and Fire of Life (uncommon). The skill names a
   drawback as what the rare tier is *for*, and 24 of 26 rares are pure upside. If rare is meant to
   read as a trade rather than as a bigger number, this is where it would show.
5. **Tier predicates are five rings** — Atrophy, Erode and Whittle (tempo, one per rung) and
   Crown and Swarm (offense, on tiers 3 and 1). `Tier` is the least-used card predicate in the
   grammar, and the offense pair has no tier-2 sibling.

---

## 6. What I would put in front of the owner, in order

Judgement, and labelled as such. Nothing here measures a duel.

1. **Decide whether cost discounts are supposed to be rare-only.** This is the one number that
   looks like an accident: twelve rares doing one job, 2.0% of a shelf seat between them. Either a
   discount is a rare thing (in which case twelve of them is many) or it is not (in which case some
   belong at uncommon). Moving three of the element cost rings down to uncommon would take tempo
   from 2.0% to roughly 5% of a shelf seat, without a single number being retuned.
2. **Author the `element-elementalist` rung ring.** One record, closes the last gap in the most
   deliberately-complete family in the file.
3. **Ask whether arcane's single defense ring is the colour's personality.** If yes, write it down
   in `MECHANICS.md` so it stops looking like a gap on every future run of this report. If no, it
   is one or two records.
4. **Consider whether the rare tier wants more drawbacks.** Two of twenty-six.
5. **Leave the offense share alone until something measures a duel.** 78.9% looks lopsided on
   paper and may be exactly right in play; nothing in the repo can currently tell the difference,
   and rebalancing on a record count would be tuning against a number that does not mean what it
   looks like.

---

## Reproducing this

```powershell
python .claude/skills/ring-balance/classify.py            # §1 tables, by rarity
python .claude/skills/ring-balance/classify.py --shelf    # §1 shelf shares
python .claude/skills/ring-balance/classify.py --by payload breadth   # §2
python .claude/skills/ring-balance/classify.py --by category scope    # §3
python .claude/skills/ring-balance/classify.py --by category build    # §4
python .claude/skills/ring-balance/classify.py --holes    # §5
```
