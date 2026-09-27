---
name: achievements
description: How achievements, lifetime tallies and unlocks are earned, and the one rule every piece of player progress obeys - a run started on a chosen seed progresses nothing. Load before adding or changing an achievement in data/achievements.json, adding a trigger kind, a moment or a counter, adding any new kind of progress the profile keeps, or touching internal/screens/achieve.go. Also the home of the questions to ask of a proposed achievement.
---

# Achievements

An achievement is a record the player keeps across runs. It changes nothing in the rules — an
**unlock** is the thing that does, and an achievement may carry one. Both, and the lifetime tallies
that count achievements are measured against, are **player progress**: facts the profile keeps
after the run that earned them is gone.

## Where everything is

| Thing | File |
|---|---|
| The catalog | `data/achievements.json` |
| The vocabulary and the validator — trigger kinds, clause modes, axes, moment and counter names | `internal/achieve` (`go doc ./internal/achieve` is the story) |
| What a player has earned, on disk | `internal/profile` — `Achievements`, `Unlocks`, `Counters` |
| **The one place progress is written** | `internal/screens/achieve.go` |
| The page that shows them | `internal/screens/achievements.go` |
| The toast | `internal/ui/toast.go`, raised by `internal/game` |
| The design record | MECHANICS.md §Achievements, and the chosen-seed paragraph in §The profile |

## The rule: a chosen seed progresses nothing

**A run started on a code the player chose on the new-run dialog moves no progress at all** — no
tally, no award, no unlock, no toast. A chosen code is a journey that could have been looked up in
advance, so what it achieves is not the player's to keep. The dialog's **Achievements enabled** box
says which kind of run START will begin, and the run carries the answer as
`session.Session.SeedChosen`, saved with it.

How it is held, and what it means for anything new:

- **`progresses(gs)` in `internal/screens/achieve.go` is the one question.** It is true when there is
  a profile to move and the run under way was not started on a chosen seed. `earn`, `bumpCounters`
  and `settleCounters` all ask it before writing anything.
- **`achieve.go` is the only file in the program that writes progress to the profile.**
  `TestOnlyAchieveGoWritesProgress` walks every non-test Go file and fails on a call to `Bump`,
  `Award`, `Unlock` or `Discover`, or an assignment to `Counters`, `Achievements`, `Unlocks` or
  `HandsDiscovered`, anywhere else.
- **A new achievement written in the existing trigger kinds needs nothing extra.** A turn, a count
  or a moment all reach the profile through `earn`, which is already gated.
- **A new moment** is a constant in `internal/achieve/catalog.go` plus the one call site that raises
  it, and that call site calls `earnMoment` — never `gs.Profile.Award` directly.
- **A new kind of progress** — hands discovered, a new tally, a new list on the profile — is written
  by a function in `achieve.go` that asks `progresses` first. Add its method or field name to
  `profileWrites` in `achieve_test.go` in the same change, so the tripwire covers it too.
- **The fix for a red `TestOnlyAchieveGoWritesProgress` is to route the write through `achieve.go`**,
  never to exempt the file the write is in.
- `TestAChosenSeedProgressesNothing` is the behavioral half: a chosen-seed run's turn, win and
  settle leave the profile as it was, and the same code rolled does not.

**What is not progress, and is not gated:** the run's own state (it is thrown away with the run),
settings, and `TutorialSeen`. The journal and the ledger still record a chosen-seed run in full —
they are records of what happened, not rewards for it.

## Adding an achievement

1. Write the record in `data/achievements.json` against the vocabulary `internal/achieve` validates;
   a misspelled trigger, moment or counter fails the launch, which is deliberate.
2. If it needs a moment the code does not raise yet, add the constant and the `earnMoment` call —
   see above.
3. `go test ./internal/achieve ./internal/screens` — the catalog walk and the progress gate.
4. Read it on the Achievements page in a launched game.

## Questions to ask of a proposed achievement

*A stub, to be filled in by the owner.* The first one is settled:

- **Can it be earned on a chosen-seed run?** No, and nothing about the record can change that. If a
  proposal only makes sense as a reward for a particular seed, it is not an achievement.
