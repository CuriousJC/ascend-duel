---
name: unlocks
description: Matching achievements to relic unlocks - the ledger of which achievement opens which relics, the achievements that open nothing yet, and the relics still in the default pool. Load before pairing an achievement with a relic, proposing a relic be locked or unlocked, asking which achievements or relics are still unpaired, or investigating whether the achievement count and the locked-relic count should line up.
---

# Unlocks: pairing achievements with relics

An achievement changes nothing; an **unlock key** it grants is what keeps a relic off the shelf until
the player has earned it. The mechanism — the grant, the gate, the run taking its set when it
starts, the launch-time backfill — is the [`achievements`](../achievements/SKILL.md) skill's
§Unlocks and MECHANICS.md §Unlocks. **This skill is the matching**: which deed opens which relic, and
what is still unpaired on either side.

## The two rules

- **Every relic unlock has an achievement behind it.** A relic writing `"Unlock": "<key>"` that no
  achievement grants could never be offered, and the loader refuses it at launch. This one is
  enforced.
- **Whether every achievement opens a relic is open.** It may, and whether the two counts should
  line up is the question this skill exists to investigate — not a rule to apply. Do not pair an
  achievement with a relic, or lock a relic, on your own reading: propose, and let the owner decide.

## The ledger

```powershell
python .claude/skills/unlocks/unlocks.py             # every pairing, then both unpaired lists
python .claude/skills/unlocks/unlocks.py --unpaired  # only what is still to match
```

It reads `data/achievements.json` and `data/relics.json` and prints three things:

- **Paired** — each unlock key, the achievements granting it and the relics behind it, with rarity
  and family.
- **Achievements that open nothing**, with their trigger, so a deed can be read against a relic.
- **Relics in the default pool**, by family in the file's order with each family's rarity mix,
  because a family is usually locked or left open as a whole.

The figures it prints are the catalog as it stands; nothing here writes them down.

## Proposing a pairing

The question to answer is the one the achievements skill already asks: **does this relic only pay
off once the player has done what this achievement asks?** A Five of a Kind relic on the shelf of a
player who has never formed a five is a relic they cannot yet judge.

- **A deed that teaches the relic is the natural key** — the hand-formed achievements open the
  relics built on that hand, the use counters open the rings that repeat that form or element.
- **A family that climbs a ladder unlocks up it**: the harder deed opens the rarer relic. MECHANICS.md
  §Unlocks has the Four and Five of a Kind table as the worked example.
- **A deed with nothing to teach may still open something** — a milestone opening a relic as a
  reward rather than a lesson. Say which kind each proposal is; the owner decides whether the second
  kind belongs in the game.
- **Name the key for what it opens, not for the deed** (`element-five-relics`, not `monochrome`), so
  a second achievement can grant the same key and the deed can be reworded without moving relics.
- **One key can open a family and be granted by several achievements**; the first one earned opens
  it.

## What a pairing costs

- **Locking a relic moves the shelf.** A new player draws from the default pool alone, so every tier's
  share changes. Run `go run ./tools/relicsheet` and report a new player's shelf shares beside the
  full catalog's — the [`relic-balance`](../relic-balance/SKILL.md) skill reads them.
- **A relic locked behind a hard deed is a late-profile relic.** Say how hard the deed is — a
  hand's odds from `tools/handodds`, a count's size, a collection's reach — so the owner is pricing
  the wait and not just the pairing.
- **Unlocks are re-derived from awards at every launch**, so pairing an achievement players already
  hold opens the relic for them on their next launch. That is usually wanted; say so when it is not.

## Making the change

Both halves in one edit, or the launch fails: `Unlocks` on the achievement record and `Unlock` on
each relic record. Then `go test ./internal/achieve ./internal/session`, regenerate the relic sheet
alone, and re-run the ledger to show the pairing landed.
