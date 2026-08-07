//go:build !idleexit

// Package idle closes the game after a stretch with nobody at the controls.
//
// **It exists for unattended runs, not for players.** Launching the game to look at a
// change means a window that then sits open forever waiting to be closed by hand; with this
// on it shuts itself and the run ends cleanly on its own. That is a development
// convenience, and a game that quits on a player who steps away to make tea is a bug, so it
// is behind a build tag exactly like internal/trace:
//
//	go run -tags idleexit .                  # closes itself after two minutes idle
//	go run -tags "debugtrace idleexit" .     # traced and self-closing
//	go run .                                 # nothing: Tick is empty and always false
//
// An ordinary `go build .` carries none of it — no timer, no environment lookup, no exit
// path a player could reach. Like trace, it must stay deletable in one commit, which is
// what keeps it acceptable in a product that will be sold.
//
// It may never change an outcome. It closes the window; it does not touch a duel.
package idle

// Tick reports whether the game has been idle long enough to close itself. Always false in
// this build — the whole feature is compiled out.
//
// Arguments are all cheap to produce at the call site, so unlike trace's log lines there is
// nothing here worth guarding behind an Enabled() check.
func Tick(mouseX, mouseY int, focused, clicking bool) bool { return false }
