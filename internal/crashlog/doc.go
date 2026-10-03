// Package crashlog is what a shipped build owes a bug report: the file written when the game
// panics, and the running note of the small failures that did not.
//
// **A crash a player cannot describe is a crash nobody can fix.** Everything else in this repo
// that watches the game — internal/trace, the scripted demo, the debug flags — is for whoever is
// building it, on a machine they own, with a toolchain to hand. This is for the other case: an exe
// on somebody else's computer, going wrong once, with nobody watching but the person it happened
// to.
//
// # What it writes
//
// One file per panic, `crash-<utc>-<code>.json`, in the profile's own directory. **The time leads
// so the directory sorts by when**, and the run code follows so a report names the run a player can
// read off the screen — a code is not unique, since a pinned seed is the same code every launch, so
// it cannot lead. Old reports are pruned on the way in: a config directory that grows without
// bound is a bug that only shows up on the machine of the player who plays most.
//
// **It goes through profile.Store rather than through `os`.** Nothing above the storage boundary
// may know a save is a file — see the platform-readiness ticket in TODO.md — and a second thing
// reaching for `filepath` is a second thing to port.
//
// # What is in it, and in what order
//
// The tiers are written outermost first, so a report truncated by anything still opens with the
// thing a bug report most needs:
//
//  1. **Identity** — the schema, the build, the run code, the install id, the UTC time, the
//     platform, the screen, the run's phase, the tick, the panic and the goroutine stack.
//  2. **The run** — profile.RunSnapshot verbatim, which is already a save-safe schema.
//  3. **The ledger's records so far.** **The snapshot on disk is written only at phase boundaries**,
//     so a crash mid-duel has the room's start state and nothing since; the records held in memory
//     are the only thing that knows what happened in the fight being played.
//  4. **The recent problems** — the last several non-fatal failures, which were a log.Printf each
//     and evaporated with the process.
//
// # What it may never do
//
// **Nothing here may ever be fatal.** A report that cannot be written, a read-only directory, an
// entropy source that will not answer: all of them are "this crash is not recorded", which is the
// rule internal/profile and the audio device are already under. A crash handler that panicked
// would turn one crash into two.
//
// **Capture may never change an outcome.** It joins playback speed, the debug flags,
// internal/trace, internal/idle and the scripted demo: combat.ResolveRound never sees any of it.
//
// **The report carries no path, no machine name and no user name.** The platform and the build
// version are the whole of the environment, and the install id is sixteen random characters that
// identify nobody. That rule is here rather than in the sending code deliberately: a report is
// written long before there is anywhere to send it, and a field added now on the assumption that
// it stays local is a field that leaves the machine the day the send button lands.
//
// **internal/combat may never import it**, for internal/trace's reason: the rules package stays
// free of everything, which is what makes it testable without a window.
//
// # How it is wired
//
//   - **A `recover()` at the top of `Game.Update` and `Game.Draw`, and one in `main`** for the panics
//     raised while the catalogs load. `internal/game/crash.go` is the handler.
//   - **A crash is a whole screen, not a dialog** — `state.Crashed` and `screens.CrashScene`. A dialog
//     draws the scene underneath it, and the scene underneath is the one that has just panicked, which is
//     how one crash becomes two. The chrome stands down, the ledger is closed, the toast queue is
//     dropped, and the page draws with the ground, the fonts and two buttons. **There is no way back**: a
//     crashed process is one whose state is not trustworthy, so the way out is quitting.
//   - **A second panic quits rather than writing a second report.** The only thing still running after
//     the first is the screen written to report it.
//   - **The non-fatal notice is the other half**: the confirm box's shape with one answer,
//     `ui.ProblemNotice`, chrome on the toast's terms. **It never raises during a duel** — it waits for a
//     phase boundary, because a box in front of a round in playback stops a fight to talk about a file.
//     It takes the modal red and says PROBLEM where the toast says ACHIEVEMENT.
//   - **Two verbs, and which one a call site wants is a judgment about the player.** `Note` records and
//     logs; `Tell` does that and also queues a notice. "The score has no device" is a Note; "this run is
//     not being saved" is a Tell.
//   - **A report takes a copy of every file named beside it**, which is the journal. `Write` takes the
//     names from its caller, so nothing here learns what a journal is, and a copy lands under the
//     report's own base name with the companion's own extension. **A report is counted by its own
//     `.json` when pruning**, or the allowance would shrink the day a second companion joined the first.
//   - **The screenshot is not a companion**, because it is not a file the game was already writing: it
//     is bytes that exist only because a panic happened, so it is a parameter rather than a name in the
//     list and it goes through `profile.Store.WriteBytes`. **Only a panic inside `Draw` has a picture**,
//     since the image being drawn into holds as much of the frame as got drawn before the fault; a panic
//     in `Update` happens between two frames and the report has no `screenshot` field. `game.shotOf` is
//     the readback, on the game goroutine because it is a GPU operation, and `EncodeShot` is the encode
//     and the ceiling — the split `internal/trace` makes.
//   - **A report has a ceiling and sheds from the bottom.** `Encode` marshals, and if the document is
//     over it drops the scene tier, then the problem ring, then the ledger a fight at a time, oldest
//     first, and names what went in `shed`. **The identity and the run snapshot are never shed.** **It
//     decides in memory and writes the file once**: a crash handler gets one chance at the disk, and a
//     file deleted and rewritten is a second chance for the rewrite to be the one that fails.
//   - **A screen describes itself through `ui.Reporter`, and it is optional** so a new screen is not
//     broken by not having one. **The method may read plain fields and nothing else** — the scene being
//     asked has just panicked, so a lookup or a layout or a rule read back off the run is a second crash
//     inside the first. `CombatScene.Report` carries the shape of the screen's state machines rather
//     than their contents: no cards, because the journal holds every selection and the ledger holds what
//     the engine made of them.
package crashlog
