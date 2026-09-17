---
name: bug-hunter
description: The procedure for taking a bug the owner found while playing and turning it into a fix he can see. Build a scenario that isolates the bug and launch it so he confirms the repro before anything is fixed, then fix, then relaunch the same fixture, then delete it. Load before diagnosing or fixing any reported bug, and before authoring a scenario record for one.
---

# Hunting a bug the owner found while playing

A bug in this game is reported the way a player reports one — "the copy rune duplicated something
twice" — and the gap between that sentence and a line of code is where the time goes. This skill
closes it with `internal/scenario`, which is the one thing in the repo that can put an arbitrary
state in front of the owner in about ten seconds.

**The deliverable is not the fix. It is the owner watching the thing not happen any more.** Nothing
in this game simulates a duel and `internal/screens` cannot be unit-tested, so for most of what gets
reported the only proof that exists is a launched window. Build for that.

## The loop

1. **Read the report against the code** and say, in a sentence, which mechanism you think is
   implicated and where it lives. Do not fix anything yet.
2. **Author a scenario record that isolates it** and put it first in
   `internal/scenario/scenarios.json` — an unset `ASCEND_DUEL_SCENARIO` takes the first entry, so
   the launch command stays bare and the record is deleted at the end anyway.
3. **Launch it** and tell the owner, in one sentence, what to do and what he should see go wrong.
   **Stop there and wait.** This is not a formality and it is not skippable: a confirmed repro is
   what makes the fix a fix rather than a guess that happened to compile. Until he has seen it, the
   thing being fixed is your reading of a sentence.
4. **Fix it**, in the package that owns the decision — see "Where a bug belongs" below.
5. **Write the regression test** if the fix landed in a package that has no window. Rules bugs get
   one; screen bugs get the scenario alone.
6. **Relaunch the same fixture** so he sees the same situation behave. Then **delete the record**,
   `gofmt`, and vet every build tag.

**Steps 3 and 6 are the same fixture launched twice.** Changing it between them means the second
launch is not evidence about the first — so if the repro turns out to need a different fixture, that
is a new step 2 and a new confirmation, not an edit between the two launches.

**If the fixture does not reproduce it, say so and stop.** A fixture that fails to show the bug is
information: either the diagnosis is wrong or the bug needs something the fixture stripped. Both
are answered by asking rather than by fixing the code the report pointed at anyway.

## Building the repro record

The record struct and every field's reasoning are in `internal/scenario/scenario_on.go`, which is
the authority. What matters here is which dials a *bug* fixture reaches for:

- **`Dummy: true` first, almost always.** A bug is something to watch happen repeatedly, and a
  five-round clock over a creature that dies means producing the situation once and then replaying a
  duel to get back to it. A dummy makes the fight a bench.
- **`Actions` widens the budget** so the hand is the cards the bug needs rather than the cards six
  points can pay for. It does not lift `combat.MaxActions` — still five cards a turn.
- **`Deck`, not `Hand`, when the bug survives a refill.** `Hand` deals over a normal shuffle, so
  the second hand of the fight is the game's own; `Deck` replaces the pile. A bug about a card that
  comes back around wants `Deck`. `tools/scenariodeck` writes the block.
- **`Riders` on a deck line or a hand card** opens on cards a rune has *already been spent on* —
  the right fixture for "what the altered card does", the wrong one for "what spending the rune
  does". A bug in the spending wants `Runes` and the sack.
- **`Seed`** pins the deal, so the fixture is the same launch every time. Six Crockford base32
  characters, and the owner's own bug report may already carry one — the run code is on the run-over
  splash and in the settings screen's bottom-right corner. **A run code from a report beats anything
  you would compose**, because it is the run the bug actually happened in.
- **`Screen`, `Fight`, `Vitae`, `Life`** for a bug that is not in a duel. A shop or reward bug is
  otherwise a duel away.
- **`RelicSlots` and `RoundLimit`** when the bug is about wearing a lot, or about the clock.

**Strip everything the bug does not need.** A fixture carrying five relics and three runes when the
bug is one rune is a fixture that cannot distinguish "fixed" from "no longer reachable". Name only
what is implicated, and if you are not sure whether a relic is implicated, that is a second fixture
rather than a longer one.

**The `Note` says what is wrong, not what to look at.** Every other record in the file answers a
question about a mechanic; this one is temporary and its Note should read as a bug report — what was
seen, what should have been seen. It is printed at startup, so it is the first line in the log when
the owner relaunches an hour later.

## Launching

```powershell
$env:ASCEND_DUEL_PROFILE = "$env:TEMP\ascend-duel-bughunt"
go run -tags scenario .
```

**Set `ASCEND_DUEL_PROFILE` every time.** `saveRun` is not gated on the scenario and fires at every
phase transition, so a fixture that advances a station writes over whatever run the owner has in
progress. Pointing the directory somewhere throwaway is the whole guard, and it costs nothing —
`BootRun` never resumes under a scenario and `tutorialForThisRun` answers no to a scenario, so an
empty profile directory changes nothing else about the launch.

Run it **in the background** so the owner can talk while it is up. Compose tags where it helps:
`-tags "scenario debugtrace"` gives an event log and `trace/frame.png`; `-tags "scenario idleexit"`
closes the window by itself if he walks away.

**A misspelled key fails the launch at package init**, before a window opens, which is deliberate —
so `go vet -tags scenario ./...` before launching catches the Go half and the launch itself catches
the JSON half. A launch that dies in the log is not a repro.

## Where a bug belongs

The commonest wrong fix in this repo is fixing the screen for a rules bug, because the screen is
where it was seen.

- **`internal/combat` decides a round; the screen replays it.** If the *number* is wrong, the bug is
  in the rules. If the number is right and the picture disagrees with it, the bug is in the screen.
  Say which before touching either.
- **Presentation may never change an outcome.** A fix that makes a screen right by feeding something
  back into `ResolveRound` is not a fix. If the only way to make the picture right is to change a
  rule, that is a design question — state it and let the owner decide which of the two is wrong.
- **A duplication or a count bug is usually an identity bug.** `combat.Card.ID` is what lets a rune
  name a card while three piles hold copies of it, so something happening twice is often a card
  matched by value where it should have been matched by ID.
- **The run's deck and the fight's piles are different objects.** `internal/session` owns the run's
  deck and `internal/combat` owns the round; a consumable spent mid-fight touches both, and "twice"
  is what it looks like when one edit reaches both.
- **A picture drawn twice is not a thing happening twice.** `combat_handmorph.go` reads what changed
  off the faces rather than off the rune, and a suppression that was missed draws a second copy of a
  card that only ever existed once. Check the event log or the engine's own count before believing
  the screen.

## The regression test

**Write one when the fix lands in a package that does not link Ebitengine** — `combat`, `session`,
`decks`, `pyramid`, `tutorial`, `achieve`, `data`, `profile`, `seeds`. Write it **failing first**,
so the fix is what turns it green rather than something asserted afterwards about code already
changed.

**Do not write one for `internal/screens`** unless the thing being pinned is arithmetic rather than
drawing; that package needs a window, and the scenario is its test. If a screen bug turns out to be
pinnable by a windowless helper, splitting that helper out is the fix worth making.

Name it for the promise rather than for the bug — `TestACopiedCardIsOneCard`, not
`TestCopyRuneBugFix`. A test named after a bug stops meaning anything once nobody remembers the bug.

## Cleaning up

**Delete the scenario record with the fix**, in the same change. `scenarios.json` is a catalog of
live questions and a bug that is fixed is not one; the repo's cut-means-deleted rule applies, and
the regression test is what stands guard afterwards. If the fixture genuinely answers a standing
question about the mechanic rather than about the bug, say so and ask — do not decide alone that it
earns a place.

Then:

```powershell
gofmt -l .
go vet ./...; go vet -tags debugtrace ./...; go vet -tags idleexit ./...
go vet -tags demoplay ./...; go vet -tags scenario ./...
go test ./...
```

**Leave it all unstaged** and summarize what changed. No `git add`, no commit, no PR unless asked
for that specific step.

## What this skill does not do

- **It does not file anything.** A bug noticed on the way past is a sentence in the reply, never an
  entry in `TODO.md` and never a `[?]` in `MECHANICS.md`. The owner grows those lists.
- **It does not add to `data/`.** A fixture belongs in `internal/scenario/scenarios.json`, which is
  compiled out; `data/` is the game's own catalog and is loaded by every build.
- **It does not widen the fix.** Fix the reported bug. Anything else found along the way is reported
  in words and left alone.
