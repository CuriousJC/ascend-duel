# Error capture — the plan

**What a shipped build owes a bug report.** A crash a player cannot describe is a crash nobody can
fix, and a demo is the first time anyone but the owners runs this game on a machine the owners have
never seen.

This is a plan rather than a record: it says what to build and in what order. **It retires when the
last PR lands**, and each entry is struck as it merges. `MECHANICS.md` is still what the game *is*
and `TODO.md` is still the open work list; nothing here is a source of truth about either.

---

## The three layers, and why they are three

| Layer | Holds | Answers |
|---|---|---|
| **Journal** | the player's choices, in order | *how do I get back to this?* |
| **Records** | what the engine decided, as flat data | *what actually happened?* |
| **Prose** | the words a record is read as | *what does the panel say?* |

**The journal is inputs and the records are outputs, and they are not interchangeable.** A record is
what `ResolveRound` produced; it cannot re-drive the resolver, and a replay fed from one would be
comparing the engine to a recording of itself. A journal is what the player chose, which is the only
thing that can put a run back where it was.

**Prose is derived, never stored.** The panel, the export and a crash report all read the same
records through one translator, so a wording change reaches every ledger already on disk and a
machine reading a crash file gets data rather than English.

**The seed is not enough on its own.** It rebuilds the tower, the motifs, the elements and the
shuffles, but the deck changes with what the player takes and cuts and the hands follow from the
deck. Same seed and different choices is a different fight two. The journal is what closes that gap.

---

## The envelope every record is written in

**One line of JSON per record, append-only.** A file appended a line at a time has the last thing
before a panic already on disk; a single document written at a phase boundary does not.

```json
{"t": 12345, "kind": "select", "card": "a3f1", "label": "Jab", "seat": 3}
```

- **`t` is the simulation tick**, `GlobalState.Count`, never a wall clock. The rule `internal/trace`
  is under: a tick lines up with a replay of the same seed and a clock does not.
- **`kind` is one word from a closed vocabulary**, append-only, and **a name rather than an
  ordinal** — the rule every saved file is under, and this one outlives its build harder than a save
  does.
- **Fields are flat, named and few.** No nested objects; no arrays but a target list.
- **The first line of a file is its header**: schema number, build version, run code, install id,
  platform, and when it was opened.

**Never store `combat.Event`.** `HandGrown`, `HandRelicScale` and `HandLanding` are 25x5 arrays
each, so an event is about 2.5 KB and nearly all of it zero. A record is a kind and a handful of
named fields, which is what makes a run's worth of them affordable.

**JSON rather than CSV.** An act line, a hand line and a term line carry different fields, and a
table holding all three is either wide and sparse or has a stringly-typed blob column — the
sparse-array problem in a different costume.

---

## The kinds

**The list is derived from what the game already words**, not invented: every line
`internal/screens/combat_prose.go` and `internal/screens/ledger_after.go` produce maps to exactly
one record kind. That is what makes the translator total by construction rather than by agreement.

**Choice kinds** — the player acted. These are the journal.

`select` · `unselect` · `duel` · `rune` · `essence` · `potion` · `stone` · `good` · `buy` · `take` ·
`cut` · `phase` · `screen` · `run`

**Outcome kinds** — the engine decided. These are the ledger.

`fight-begin` · `act` · `hand` · `term` · `shield` · `status` · `life` · `vitae` · `fight-end`

**A kind with no translation fails a test.** The treatment `EventKind` gets from its choreography
entry: a record with no words is a blank line, and a blank line and a line nobody wrote look
identical.

---

## The files, and where they go

All of them sit in the profile directory — `%APPDATA%\ascend-duel` or `~/.config/ascend-duel`, moved
together by `ASCEND_DUEL_PROFILE` — never beside the executable, on the rule `internal/profile` is
already under.

| File | Written | Kept |
|---|---|---|
| `journal.jsonl` | as the run is played | the run in progress, and no longer |
| `crash-<utc>-<code>.json` | on a panic | the last few crashes |
| `crash-<utc>-<code>.jsonl` | on a panic | the journal as it stood, beside its report |
| `crash-<utc>-<code>.png` | on a panic, when a frame is available | beside its report |

**There is one journal and it belongs to the run being played.** Starting a run truncates it. A run
that ended without going wrong is a run nobody is going to ask about, so keeping it is a directory
that grows for no reader — and a single fixed name means nothing has to sweep up after it.

**A crash takes a copy of the journal under its own name**, which is what makes the one-file rule
safe: the run that blew up is exactly the run worth retracing, and it would otherwise be truncated
by the next launch.

**A crash report's name leads with the time so the directory sorts by when**, and the run code
follows so a report names the run a player can read off the screen. A run code is not unique — a
pinned seed is the same code every launch — so it cannot lead.

**Pruning the crash files is not optional.** A config directory that grows without bound is a bug
that shows up only on the machine of the player who plays most.

---

## The rules that hold across all of it

- **Nothing here may ever be fatal.** A journal that cannot be written, a crash file that cannot be
  saved, a read-only directory: all of them are "this session is not recorded". The rule
  `internal/profile` and the audio device are already under.
- **Capture may never change an outcome.** It joins playback speed, the debug flags,
  `internal/trace`, `internal/idle` and the scripted demo. `ResolveRound` never sees any of it.
- **It writes through the storage boundary, not through `os`.** See the platform-readiness ticket in
  `TODO.md`: nothing above the backend may know a save is a file, and a second thing reaching for
  `filepath` is a second thing to port.
- **`internal/combat` may never import any of it**, for `internal/trace`'s reason: the rules package
  stays free of everything, which is what makes it testable without a window.
- **No file name is taken from further up without being checked.** `Store.WriteExport` already
  refuses a name carrying a separator or a `..`, and every file here goes through it.

---

## The order

### 1. Records and the translator

Turn the ledger's storage from worded spans into flat records. `combat_prose.go` and
`ledger_after.go` become the translator and run at read time instead of write time; the panel and
the export draw exactly what they draw today.

- **The translator is one place.** Two files write ledger lines today and both move; a third would
  be the seam through which the panel and the export come to disagree.
- **Rounds are immutable once recorded**, so the panel caches one translation per round rather than
  re-wording a run's worth of lines every frame.
- **A run in progress from an older build** either keeps its stored prose or loses its ledger. One
  decision, taken deliberately, and the profile rule still holds: a file from a newer build is read
  and never written over.
- **Ink names stay out of the record.** A record says an element, a relic key or a verb; which
  swatch that is drawn in is the panel's answer — one step further along the argument that already
  keeps colors out of the save.

### 2. Crash capture, and the two dialogs

`internal/crashlog`, a `recover()` at the top of `Game.Update` and `Game.Draw`, and one in `main`
for the panics raised while the catalogs load.

Tiers, in the order they go in the file:

1. **Identity** — schema number, build version, run code, install id, UTC, platform, the active
   screen, the run's phase, the tick, the panic value and the goroutine stack.
2. **The run** — `profile.RunSnapshot` verbatim; it is already a save-safe schema.
3. **The ledger's records so far** — **the snapshot is written only at phase boundaries, so a crash
   mid-duel has the room's start state and nothing since.** The in-memory records are the only thing
   that knows what happened in the fight being played.
4. **A ring buffer of recent problems** — the last several non-fatal failures, which are
   `log.Printf` today and evaporate.

**Two dialog shapes, deliberately.**

- **A non-fatal notice** is the confirm box's shape with one answer. It queues like the achievement
  toast and **never raises during a duel** — it waits for a phase boundary. It is not a toast: a
  toast means the player did something good.
- **A fatal crash is a whole screen**, not a dialog over the scene that just panicked — drawing that
  scene again is how one crash becomes two. It draws with fonts and the ground and nothing else, and
  says what happened, the run code, where the report went, and Quit. Two buttons, focus-walkable, so
  the controller refactor inherits it.

**An install id is generated into `profile.json` here**, random and identifying nobody, so several
reports from one player can be grouped. A run code is not an id.

### 3. The journal

The append-only choice file: the header record, the choice kinds, written as the player plays.

- **It survives the run**, unlike the ledger, which dies with `run.json`. That is the point: the run
  worth getting back is usually the one that just ended badly.
- **It records choices, never outcomes.** A journal holding what the engine produced stops being a
  way to retrace a run and becomes a second, worse ledger.
- **Legible rather than exhaustive.** Replay is a person at the keyboard, so a record names a relic
  key, a card label and a seat — not a drag path and not a pointer position.

### 4. The richer crash payload

- **The screen**, as a PNG. Captured live on a panic inside `Draw`; for a panic inside `Update`,
  from a rolling last-good frame. `ReadPixels` is a GPU-to-CPU readback that stalls the frame it
  happens on, which is why `internal/trace` throttles to one every two seconds — the same throttle
  applies.
- **Scene state**, through an **optional** interface a scene opts into, so a new screen is not
  broken by not having one. The combat screen is the one worth writing: the round, the playback
  cursor, the selected seats, the shields, the opponent.
- **Payload bounds.** A PNG is hundreds of kilobytes and a journal grows with the run, so a report
  has a ceiling and sheds tiers from the bottom — the screen first — rather than writing a file too
  big to send.

### 5. Sending a report

**Its own PR, and the thing the rest of the design is shaped around.**

**Two separate questions, and conflating them is the mistake this section exists to prevent.**
*May the game offer to send a report* is a setting, and it is **opt-out** — the offer is there
unless the player turns it off. *Does this particular report go* is **always an explicit answer to
an explicit question**, every time, however the setting stands. **Nothing is ever transmitted by a
setting alone.**

- **The setting governs the button, not the sending.** A player who leaves it alone gets a crash
  screen with a send button on it; a player who turns it off gets a crash screen without one. Either
  way the report is written to disk and either way nothing leaves until somebody presses something.
- **There is no remember-my-answer.** A confirmation that can be switched off is an opt-in setting
  wearing a confirmation's clothes, and the whole point is that the last thing before a report leaves
  the machine is a person deciding.
- **The crash screen carries the opt-out itself**, beside the send button. Someone who did not
  realize the offer was on gets to turn it off at the moment they find out, rather than being sent
  to a settings screen to hunt for it — and having turned it off there, they never see the offer
  again.
- **The confirmation says what is in the report** and offers to open it, because "send a bug report"
  is not consent to something unseen.
- **The client holds no credentials.** A mail secret inside a shipped executable is extracted the day
  someone cares and cannot be taken back once the binary is out. The shape is an anonymous-write
  endpoint with the rate limiting on the server — or, enough for a demo, a button that opens the
  folder and one that copies the report.
- **The report carries no path, no machine name and no user name.** Platform and build version are
  the whole of the environment.
- **A failed send is not a failed crash report.** The file is on disk either way.

### 6. The replay harness

**Undecided, and deliberately last.** Feeding a journal back through the game automatically is worth
building only if playing one back by hand turns out to be too slow. What it would buy beyond that is
a regression test: replay the journal, produce records, compare to the records stored — a divergence
names the round it happened in.
