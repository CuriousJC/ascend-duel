// Package profile is what survives a run: the two files the game writes to disk, and nothing else.
//
// **This is the first thing in the game that persists.** Everything else is either authored data
// read out of `//go:embed`, or state that dies when the process does. That makes this package the
// place where every "never serialize an ordinal" rule in the repo stops being theoretical, so it
// holds one rule above all others:
//
// **What gets written down is a name, never a number.** `combat.ConceptID`, `combat.Element`,
// `combat.StatusID`, `systems.GlyphKind` and `session.Phase` are all append-only enums whose
// ordinals are indices into arrays and caches — inserting one mid-enum silently re-points every
// value already stored. A saved file outlives the build that wrote it, so an ordinal in one is an
// ordinal that will eventually mean something else. Every field here is a string key or a plain
// count.
//
// **Two files, not one, and that is a decision rather than tidiness** *(owner's call, 2026-08-25)*:
//
//   - `profile.json` is the player. Achievements, unlocks, whether the tutorial has been watched.
//     It accumulates and is never thrown away.
//   - `run.json` is the run in progress. It is written at every phase transition and deleted when
//     the run ends.
//
// Keeping them apart is what makes a corrupt save cost a run rather than a career. They share a
// directory so that turning on Steam Cloud later is a configuration change rather than a refactor:
// its simplest mode syncs one named directory.
//
// **Where the directory is: `os.UserConfigDir()`, never beside the executable.** The game will be
// sold on Steam, which installs under `C:\Program Files (x86)\...` — a tree a normal user process
// cannot write to. A write there either fails, or is silently redirected by Windows' UAC
// virtualisation into `%LOCALAPPDATA%\VirtualStore`, which is worse: it works in testing and
// diverges invisibly afterwards. The same applies to a tarball unpacked into `/opt` or a read-only
// Flatpak mount. A per-executable directory is also per-install rather than per-user, so two
// accounts on one machine would share one set of achievements.
//
// **Nothing here is fatal.** A missing file is a new player, a corrupt file is a new player, and an
// unwritable directory is a session whose progress is not recorded. The same rule the audio device
// gets: a machine that cannot save still plays the game. Failing to launch over a save file would
// be a worse bug than any it could prevent.
//
// **It imports nothing of ours**, like `internal/seeds` and for the same reason: it sits at the
// bottom of the graph so that `session` can save itself without anything below `session` learning
// what a run is. The snapshot in run.go is plain data — `session` knows how to fill one in and how
// to read one back, and this package knows how to put one on disk.
package profile
