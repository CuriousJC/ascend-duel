//go:build !debugtrace

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
