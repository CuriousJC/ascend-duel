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
//
// # Working on it
//
// The idle window is two minutes, and DUELLO_IDLE_SECONDS shortens it for a quick check.
//
//   - **Everything is gated on window focus, cursor movement included.** That is the whole trick, not a
//     nicety: an unattended run sits in the background while whoever launched it does something else,
//     and a cursor crossing the desktop over an unfocused window would otherwise read as someone
//     playing. The one case it exists for would be the one case it never fired in.
//   - **It sets `ShouldClose` rather than returning `ErrClosing`**, so the exit runs through the same
//     path as the window's close button and there is only one way the game ends.
package idle
