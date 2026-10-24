---
name: relic-balance
description: The aggregate view of the relic catalog, on the nine build aspects of docs/design/build-aspects.md (economy, composition, hands, action capacity, tempo, consistency, offense, defense, engine capacity) - how the catalog spreads across them, and how flat, multiplicative, repeat and stateful payloads sit across rarity, element, form and hand. Load before answering any question about the catalog as a whole - is offense over-represented at common, which aspects are thin, does every element have a cost relic, what would this proposal do to the shape of the shelf - or before adding an axis, or a new verb, moment or predicate that has to be classified. The `relics` skill is one relic at a time; this one is all of them at once.
---

# Relic balance — the catalog as a shape

**The `relics` skill answers "is this relic good".** This one answers "is the *catalog* good" —
what the records add up to, where the weight sits, and which of the empty cells are questions.

**The aspects are the design document's, and only the design document defines them.**
[docs/design/build-aspects.md](../../../docs/design/build-aspects.md) holds the nine build aspects —
the priority a player builds toward, the question each one asks, and how they overlap — and the
questions to put to coverage. **Read it before drawing a conclusion from a count.** This skill and
its tool are the mechanical half: the sorting of the relic grammar into those nine, and the counts
that come out of it. A disagreement between the two is a row of `classify.py` to fix, never a
second definition.

**Nothing here is authored.** Every classification is derived from a relic's own rules by
`classify.py` *(owner's call)*: a relic with `scale-damage` **is** offense, so writing
`"Tags": ["offense"]` into `relics.json` would state the same fact twice and let the second copy go
stale silently. A relic retuned from `add-hand-dmg` to `adjust-cost` would keep the tag it was born
with and nothing would fail. This is `CostTier` again — see the `data` skill.

**`Family` in `relics.json` is not a counter-example to this** *(owner's call)*. It is a motif name —
"Jade rings", "Weapons" — authored so the relic sheet has headings a person can read, and it groups
that page and nothing else. Nothing here reads it: if a Family and the rules ever disagree, the
rules win and the label is the thing to fix.

**The Compendium's AFFINITY chips are not the aspects either.** They are a filter vocabulary —
elements, forms, ATTACK, DEFEND, AP, HAND, SUPPORT — derived in
`internal/screens/compendium_affinity.go` for finding a relic on a page, not for judging the
catalog. A new verb owes a row to both; see the `relics` skill.

## Running it

```powershell
python .claude/skills/relic-balance/classify.py                    # every axis by rarity
python .claude/skills/relic-balance/classify.py --by aspect payload
python .claude/skills/relic-balance/classify.py --by primary breadth
python .claude/skills/relic-balance/classify.py --list defense     # the relics in a bucket
python .claude/skills/relic-balance/classify.py --relic "Weight of the Flame"  # one relic on every axis
python .claude/skills/relic-balance/classify.py --shelf            # counts against what a player sees
python .claude/skills/relic-balance/classify.py --holes            # a filled family with an empty sibling
```

`--by` takes any two of the axes below, so the crosstab is not a fixed set of reports.
`--list` searches every axis at once, so `--list rare`, `--list stateful` and `--list Element=fire`
all work.

**A relic is counted in every cell it occupies**, so a column sums to more than the catalog's size.
That is the overlap the design document calls part of the design, made visible rather than hidden,
and the report says so on every table.

## The axes

### `primary` and `aspect` — which build priorities a relic serves

**`primary` is one value per relic and `aspect` is every value it reaches.** The design document
asks for exactly this — one primary aspect and any meaningful secondary ones — and the tool derives
both:

- **A verb carries its aspects, primary first.** `surge-on-block` is action capacity and defense;
  `add-hand-dmg` is hands and offense. The `VERBS` table is the whole sorting.
- **The relic's primary is the first aspect of its first effect.** A relic's first rule is the one
  its sentence leads with, so that is the one it is for.
- **A predicate adds an aspect.** A rule keyed on an element, a form, a concept, a tier or a favored
  axis rewards a deck built toward it, which is **composition**; a rule keyed on a rung rewards
  assembling it, which is **hands**. This is why composition reaches most of the catalog: nearly
  every relic is narrowed by what the deck holds.

Two sortings worth knowing before changing one:

- **Growth is tempo.** A `grow-*` verb is value as a function of time — the design document's
  "immediate power versus buildup" — and the accumulator is read by the relic's other effects, which
  carry their own aspects. `grow-on-hit` and `grow-on-streak` are offense and tempo, since the
  growing figure is damage.
- **A setter is composition.** `set-element` and `set-form` change what is held so something else
  can fire.

### `drawback` — it takes something away

`drawback` when a multiplicative verb scales *down* (`Amount < 100`) or the clock is shortened. It is
not an aspect — a drawback is the price of a relic, not a priority a player builds toward — so it
is an axis of its own, crossed with the aspects like any other.

### `payload` — flat or scaling, and what kind of scaling

`flat` / `multiplicative` / `repeat` / `stateful` / `enabler`.

**This is the +DMG versus xDMG axis and it is deliberately not folded into the aspects.** They cut
across each other: offense comes in several payloads, and a flat economy relic and a flat offense
relic have more in common with each other, price-wise, than either has with its multiplicative
sibling. It is the axis that answers **how much of the catalog compounds** — relics fire left to
right and multiply, so two multiplicative relics are a build and two flat ones are an addition.

**`repeat` is kept apart from `multiplicative` on purpose.** `repeat-card` at 2 landings and
`scale-damage` at 200 are the same number and not the same effect: a repeat re-runs the whole
per-landing pipeline, so it compounds with `grow-on-hit` and with anything reading a landing, where
a scale does not.

**`stateful` is where the growing relics land**, and it is the payload that cannot be priced on
fight one — an accumulator is uncapped by decision, so a stateful relic is priced on where it ends
up at the top of the journey.

### `scope` — how often the rule gets to matter

`per-card` / `per-blow` / `per-turn` / `per-fight` / `per-run`, read straight off `When`.

The question it answers is **how long a fight has to run before a relic has paid for itself** —
which is the design document's tempo question asked of one relic. A `per-run` relic pays once
whatever happens; a `per-card` relic pays on every matching card of every turn. Two relics with the
same number are not the same relic if their scopes differ, and this is the axis most likely to be
missed when pricing.

### `breadth` — how much of the deck the rule can see

`unconditional` / `element` / `form` / `concept` / `tier` / `favored` / `hand` / `positional`.

Already the pricing lever the `relics` skill names: **a form relic covers every card of a form,
an element relic every card of a color, a concept relic one card in its colors**. Same verb, three
prices. Crossed with `rarity` this is the sharpest balance question the tool answers — a rare keyed
on one concept is a rare that does nothing in most runs.

### `build` — *which* element, form, concept, tier or rung

The predicate's **value**, as `Element=fire` or `Hand=pair`. This is what `--holes` walks, and it
is the axis that finds a color with no cost relic.

### `rarity` — the record's own tier

The only authored axis, and the only pricing dial there is. Tickets are 10 / 4 / 1, so **anything
added to common devalues every rare in the game** — `--shelf` computes the share off those weights,
and `go run ./tools/relicsheet` prints the same figures on the catalog page. **A mythic holds no
tickets**: each seat rolls for one apart, at `data.MythicOdds`, so its shelf share is printed as
none.

## Reading the report

**Counts are not shares, and `--shelf` prints both.** The shelf draws on rarity tickets, so an
aspect made mostly of commons is a bigger share of a *seat* than of the records, and one made mostly
of rares is a much smaller one. **Always say which of the two you are quoting**, and quote the share
when the question is about what a run actually meets.

**Equal coverage is not the goal**, as the design document says, and engine capacity in particular
is meant to be thin. A small aspect is a question to put to the owner with the document's coverage
questions beside it — is it thin by intent, is it reached only indirectly, what does a player give
up to pursue it — never a gap to fill on sight.

**A hole is a question, never a finding.** `--holes` reports a filled family with an empty
sibling, and roughly half of them are the grammar being right: defend takes no `scale-damage`
because a defend card deals nothing. An aspect keyed on no card at all — most of economy reads the
purse — is skipped rather than reported against every element, which would be pure noise.

**Nothing in the repo measures what a relic does to a duel.** This counts records; it does not
simulate. Every balance conclusion drawn from it is judgment standing on a count, and should be
said that way — the same caveat the `relics` skill puts on a suggested rarity.

## Writing a report down

**A run worth reviewing goes in `docs/analysis/` as a dated snapshot** — `relic-balance-YYYY-MM-DD.md`,
with the date in the name and the commands that produced it in the file. Read
[docs/analysis/README.md](../../../docs/analysis/README.md) for the convention: a snapshot is
*output*, never a source of truth, and it is superseded by a new dated file rather than edited in
place.

**Do not write one on every run.** The tool is cheap and the file is a standing cost; a snapshot
earns its place when it is going to be read and argued with, not as a log.

## Extending it

**A new verb in `internal/combat/relic.go` must be added to `VERBS` or the tool exits.** Same for a
new moment in `SCOPES` and a new predicate in `BREADTHS`. That is deliberate and matches the game's
own loaders: a classification that silently defaulted an unknown verb to nothing would under-report
the exact thing that just changed. **Which aspects a new verb carries is the owner's call** — say
what you would sort it into and why, against the design document, and let him decide.

- **A new aspect is a design-document change first.** The nine live in
  [build-aspects.md](../../../docs/design/build-aspects.md); a tenth is written there, and then it is
  a value in `ASPECTS` and the verbs that reach it. Never the other way round.
- **A new axis** is a key in `AXES`, filled in `classify()`. It gets `--by` and `--list` for free.
- **An aspect the rules cannot state** is the one case that would need an authored field. It has
  not come up. If it does, the field goes on `relics.json` as `Tags` and is **additive only** —
  derived aspects are never overridden by it, or the file could contradict the rules again.

## When to reach for this over the `relics` skill

| Question | Skill |
|---|---|
| does this proposed relic already exist, what does it cost to build, what tier | `relics` |
| what is the catalog short of, is offense over-weighted, what does adding N commons do | this one |
| which build priorities does the game support, and how do they interact | the design document, then this one for the counts |
| which cell of the grammar is empty | `relics` — `coverage.py` walks `(When, Do, If)` |
| which *kind of thing* is empty | this one — `--holes` walks aspect against build |

The two are complementary and the overlap is deliberate: `coverage.py` is the grammar's shape and
`classify.py` is the game's shape. A cell can be full in one and empty in the other.
