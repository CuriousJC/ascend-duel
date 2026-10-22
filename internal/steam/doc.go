// Package steam is the game's one door to the Steam client: starting the API, pumping its
// callbacks, and mirroring an earned achievement onto the player's Steam profile.
//
// **It is behind a build tag, and the untagged build carries none of it.**
//
//	go run -tags steam .     # go-steamworks linked; Steam used when it is there
//	go run .                 # nothing: every function is empty and Running is false
//
// The untagged build is the one CI checks, the one `go run .` gives, and the one the GitHub
// release ships, so none of them links Valve's library or needs `steam_api64.dll` beside the exe.
// Only a build bound for Steam takes the tag.
//
// **A tagged build needs `steam_api64.dll` beside it on Windows, and does not start without it.**
// go-steamworks loads the library from a package init, before main runs, so nothing here can
// catch that — which is why the tag exists rather than a runtime check. A Steam depot always
// carries the DLL. **With the DLL there, a missing client or `steam_appid.txt` is not fatal**:
// it is logged once and the game plays on with Steam off, the same way a machine with no audio
// device plays in silence.
//
// # The library and the DLL
//
// The binding is `github.com/hajimehoshi/go-steamworks` (Apache-2.0, purego — no cgo). On Linux
// it embeds Valve's `libsteam_api.so` itself. **On Windows it loads `steam_api64.dll` from the
// working directory**, and that file is Valve's, redistributable with the game under the Steamworks
// SDK agreement but not committable to this public repository — `*.dll` is gitignored. A local run
// copies it from the SDK's `redistributable_bin/win64/` into the repository root.
//
// # The App ID
//
// AppID is zero until the game has one, and zero skips RestartAppIfNecessary — which would
// otherwise relaunch through Steam for an app nobody owns. Launched outside Steam, the client reads
// the ID from `steam_appid.txt` in the working directory (gitignored); `480`, Valve's Spacewar, is
// enough to prove the API starts before the game has an ID of its own.
//
// # What may call it
//
// **An achievement reaches Steam only from `internal/screens/achieve.go`**, after the profile has
// taken it, so a chosen-seed run — which progresses nothing — unlocks nothing on Steam either. The
// profile is the record; Steam is a mirror of it. **`internal/combat` may never import this
// package**, and nothing here may change an outcome.
package steam
