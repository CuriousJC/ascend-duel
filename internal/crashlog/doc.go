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
package crashlog
