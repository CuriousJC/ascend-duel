//go:build !debugtrace

// Package trace is developer instrumentation: a running account of what the game did, and
// periodic screen captures, so a problem can be diagnosed from the output rather than from
// someone describing what they saw.
//
// **It is compiled out by default.** Every function here is empty, so an ordinary
// `go build .` carries none of it — no strings, no file writes, no PNG encoder. Turn it on
// with a build tag:
//
//	go run -tags debugtrace .
//
// A build tag rather than the `DebugPlacement` / `DebugGameplay` runtime flags, because
// this is a different kind of thing. Those two are *views* a player could conceivably be
// given; this exists only for whoever is building the game, and it should not be in the
// binary that ships. Keeping it deletable in one commit is the point — if trace calls start
// spreading through the screens, that property is gone.
//
// Two rules it shares with the debug flags:
//
//   - **It may never change an outcome.** `ResolveRound` neither sees it nor calls it.
//   - **`internal/combat` must never import it.** trace imports Ebitengine, and the whole
//     value of the rules package is that it does not — that is what keeps it testable
//     without a window. The *screen* traces the event log combat hands back.
//
// Lines carry the simulation tick rather than a wall clock, so they line up with a replay.
package trace

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Enabled reports whether tracing is compiled in. Guard any call site that has to build
// its arguments — `Logf` is a no-op here, but Go still evaluates what is passed to it.
func Enabled() bool { return false }

// Tick records the simulation tick that later lines are stamped with.
func Tick(int) {}

// Logf writes one line under a short category.
func Logf(string, string, ...any) {}

// Rect writes one named rectangle, for dumping computed layout.
func Rect(string, image.Rectangle) {}

// Section writes a heading, so a dump of many lines reads as a group.
func Section(string) {}

// Frame captures the screen to a PNG, throttled to one every few seconds.
func Frame(*ebiten.Image) {}
