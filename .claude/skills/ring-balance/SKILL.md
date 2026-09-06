---
name: ring-balance
description: The ring taxonomy and the aggregate view of the catalogue - what a ring *is* (offense, defense, tempo, economy, enabler, growth, drawback) and how flat, multiplicative, repeat, status and stateful payloads are spread across rarity, element, form and hand. Load before answering any question about the catalogue as a whole - is offense over-represented at common, does every element have a cost ring, what would this proposal do to the shape of the shelf - or before adding a category, an axis, or a new verb that has to be classified. The `rings` skill is one ring at a time; this one is all of them at once.
---

# Ring balance — the catalogue as a shape

**The `rings` skill answers "is this ring good".** This one answers "is the *catalogue* good" —
what the 137 records add up to, where the weight sits, and which of the empty cells are questions.

**Nothing here is authored.** Every classification is derived from a ring's own rules by
`classify.py`, and that is the decision the whole skill rests on *(owner's call, 2026-09-06)*: a
ring with `scale-damage` **is** offense, so writing `"Tags": ["offense"]` into `rings.json` would
state the same fact twice and let the second copy go stale silently. A ring retuned from
`add-hand-damage` to `adjust-cost` would keep the tag it was born with and nothing would fail.
This is `CostTier` again — see the `data` skill.

**The corollary is that new categories are cheap.** The taxonomy is three Python dicts at the top
of one file. Adding an axis, splitting a category, or reclassifying a verb is an edit there and a
re-run — no data migration, no Go change, no record touched. That is what makes it safe to keep
this derived while the categories are still being argued about.

## Running it

```powershell
python .claude/skills/ring-balance/classify.py                    # every axis by rarity
python .claude/skills/ring-balance/classify.py --by category payload
python .claude/skills/ring-balance/classify.py --by breadth scope
python .claude/skills/ring-balance/classify.py --list defense     # the rings in a bucket
python .claude/skills/ring-balance/classify.py --ring "Enflamed"  # one ring on every axis
python .claude/skills/ring-balance/classify.py --shelf            # counts against what a player sees
python .claude/skills/ring-balance/classify.py --holes            # a filled family with an empty sibling
```

`--by` takes any two of the axes below, so the crosstab is not a fixed set of reports.
`--list` searches every axis at once, so `--list rare`, `--list stateful` and `--list Element=fire`
all work.

**A ring is counted in every cell it occupies**, so a column sums to more than 137. That is the
many-to-many being visible rather than hidden, and the report says so on every table.

## The axes

**One is many-to-many; the rest are one value per rule and therefore several per ring.** A ring
with three rules can be per-card and per-blow at once, and that is information rather than noise.

### `category` — what the ring is *for* (many-to-many)

| Category | Means | Derived from |
|---|---|---|
| **offense** | it makes a blow bigger | every damage verb, plus `apply-status` where the status is `damage-over-time` or `damage-amplification` |
| **defense** | damage you do not take | `add-hp`, `scale-hp`, and `apply-status` where the status is `lose-actions`, `miss-chance` or `damage-reduction` |
| **tempo** | it buys action points, never damage | `adjust-cost`, `demote-card` |
| **economy** | the run's purse and its picks | `scale-propagation`, `adjust-prize-vitae`, `adjust-picks` |
| **enabler** | it changes what you are holding so something else can fire | `set-element` |
| **growth** | its value is a function of time, and it holds state | every `grow-*` verb and `reset-growth` |
| **drawback** | it takes something away | any multiplicative verb with `Amount < 100` |

**Denial is defense** *(owner's call, 2026-09-06)*. CHILLED steals a card off the front of their
turn, which is damage that never gets thrown — so it sits with `add-hp` rather than in a category
of its own. The alternative was a **control** bucket, and it was declined because the split it
draws is between two ways of achieving the same thing rather than between two things.

**`apply-status` is the one verb that cannot be classified by its own name**, and it is why the
status catalogue is read too: BURNING is offense and CHILLED is defense through the identical
verb. `STATUS_EFFECTS` maps the five effect *kinds*, not the five records, so a sixth status
classifies itself.

**`grow-on-hit` is offense *and* growth**, which is the case that made many-to-many necessary. The
Enflamed family both grows and swings; a taxonomy forcing a choice would have to lie about one.

### `payload` — flat or scaling, and what kind of scaling

`flat` / `multiplicative` / `repeat` / `status` / `stateful` / `enabler`.

**This is the +DMG versus xDMG axis and it is deliberately not folded into `category`.** They cut
across each other: offense comes in all five payloads, and a flat economy ring and a flat offense
ring have more in common with each other, price-wise, than either has with its multiplicative
sibling. It is the axis that answers **how much of the catalogue compounds** — rings fire left to
right and multiply, so two multiplicative rings are a build and two flat ones are an addition.

**`repeat` is kept apart from `multiplicative` on purpose.** `repeat-card` at 2 landings and
`scale-damage` at 200 are the same number and not the same effect: a repeat re-runs the whole
per-landing pipeline, so it compounds with `grow-on-hit` and with anything reading a landing, where
a scale does not.

**`stateful` is where the growing rings land**, and it is the payload that cannot be priced on
fight one — an accumulator is uncapped by decision, so a stateful ring is priced on where it ends
up at the top of the tower.

### `scope` — how often the rule gets to matter

`per-card` / `per-blow` / `per-turn` / `per-fight` / `per-run`, read straight off `When`.

The question it answers is **how long a fight has to run before a ring has paid for itself**. A
`per-run` ring pays once whatever happens; a `per-card` ring pays on every matching card of every
turn. Two rings with the same number are not the same ring if their scopes differ, and this is the
axis most likely to be missed when pricing.

### `breadth` — how much of the deck the rule can see

`unconditional` / `element` / `form` / `concept` / `tier` / `hand` / `positional`.

Already the pricing lever the `rings` skill names: **a form ring covers twelve cards, an element
ring five, a concept ring four**. Same verb, three prices. Crossed with `rarity` this is the
sharpest balance question the tool answers — a rare keyed on one concept is a rare that does
nothing in most runs.

### `build` — *which* element, form, concept, tier or rung

The predicate's **value**, as `Element=fire` or `Hand=pair`. This is what `--holes` walks, and it
is the axis that finds a colour with no cost ring.

### `rarity` — the record's own tier

The only authored axis, and the only pricing dial there is. Tickets are 10 / 4 / 1, so **anything
added to common devalues every rare in the game** — `--shelf` computes the share off those weights,
and `go run ./tools/ringsheet` prints the same three figures on the catalogue page.

## Reading the report

**Counts are not shares, and `--shelf` prints both.** 91 of 137 rings are offense — 66% of the
catalogue — but the shelf draws on rarity tickets, so offense is 79% of a *seat*. Tempo is the
opposite: 9.5% of the records and 2.0% of a seat, because twelve of its thirteen rings are rare.
**Always say which of the two you are quoting**, and quote the share when the question is about
what a run actually meets.

**A hole is a question, never a finding.** `--holes` reports a filled family with an empty
sibling, and roughly half of them are the grammar being right: defend takes no `scale-damage`
because a defend card deals nothing. A category keyed on no card at all — economy reads the purse
— is skipped rather than reported against every element, which was the first version and was pure
noise.

**Nothing in the repo measures what a ring does to a duel.** This counts records; it does not
simulate. Every balance conclusion drawn from it is judgement standing on a count, and should be
said that way — the same caveat the `rings` skill puts on a suggested rarity.

## Writing a report down

**A run worth reviewing goes in `docs/analysis/` as a dated snapshot** — `ring-balance-YYYY-MM-DD.md`,
with the date in the name and the commands that produced it in the file. Read
[docs/analysis/README.md](../../../docs/analysis/README.md) for the convention: a snapshot is
*output*, never a source of truth, and it is superseded by a new dated file rather than edited in
place. The first one is `docs/analysis/ring-balance-2026-09-06.md`.

**Do not write one on every run.** The tool is cheap and the file is a standing cost; a snapshot
earns its place when it is going to be read and argued with, not as a log.

## Extending it

**A new verb in `internal/combat/ring.go` must be added to `VERBS` or the tool exits.** Same for a
new moment, a new predicate and a new status effect kind. That is deliberate and matches the
game's own loaders: a classification that silently defaulted an unknown verb to nothing would
under-report the exact thing that just changed.

- **A new category** is a value in the `VERBS` table plus a row in the table above and an entry in
  `ORDER`. Categories are a tuple per verb, so a verb joining two is one edit.
- **A new axis** is a key in `AXES`, filled in `classify()`. It gets `--by` and `--list` for free.
- **A category the rules cannot state** is the one case that would need an authored field. It has
  not come up. If it does, the field goes on `rings.json` as `Tags` and is **additive only** —
  derived categories are never overridden by it, or the file could contradict the rules again.

## When to reach for this over the `rings` skill

| Question | Skill |
|---|---|
| does this proposed ring already exist, what does it cost to build, what tier | `rings` |
| what is the catalogue short of, is offense over-weighted, what does adding N commons do | this one |
| which cell of the grammar is empty | `rings` — `coverage.py` walks `(When, Do, If)` |
| which *kind of thing* is empty | this one — `--holes` walks category against build |

The two are complementary and the overlap is deliberate: `coverage.py` is the grammar's shape and
`classify.py` is the game's shape. A cell can be full in one and empty in the other.
