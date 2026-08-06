//go:build idleexit

package idle

import (
	"fmt"
	"os"
	"strconv"
)

// ticksPerSecond matches the Ebitengine update rate the game runs at. Counting ticks rather
// than reading a clock keeps this consistent with the rest of the project, where wall-clock
// decisions are avoided so a run can be replayed.
const ticksPerSecond = 60

// defaultIdleSeconds is how long the game waits before closing itself. Two minutes is long
// enough to launch, look at something and click around, and short enough that an unattended
// run does not leave a window open for the rest of the session.
const defaultIdleSeconds = 120

// idleEnvVar overrides the timeout, in whole seconds. Zero or negative disables closing
// entirely, which is the escape hatch for a tagged build someone wants to sit and play.
//
//	ASCEND_DUEL_IDLE_SECONDS=30 go run -tags idleexit .
const idleEnvVar = "ASCEND_DUEL_IDLE_SECONDS"

var (
	limit  = resolveLimit()
	idle   int
	lastX  int
	lastY  int
	primed bool
	fired  bool
)

// resolveLimit reads the timeout once, at package init.
func resolveLimit() int {
	seconds := defaultIdleSeconds

	if raw := os.Getenv(idleEnvVar); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			fmt.Printf("idle: ignoring %s=%q (%v), using %ds\n", idleEnvVar, raw, err, seconds)
		} else {
			seconds = parsed
		}
	}

	if seconds <= 0 {
		fmt.Printf("idle: disabled (%s=%d)\n", idleEnvVar, seconds)
		return 0
	}

	fmt.Printf("idle: will close after %ds without input\n", seconds)
	return seconds * ticksPerSecond
}

// Tick advances the idle counter and reports whether the game should close.
//
// **Everything is gated on `focused`, including cursor movement.** That is the whole trick
// rather than a nicety: an unattended run sits in the background while whoever launched it
// gets on with something else, and their cursor crossing the desktop over an unfocused
// window would otherwise read as someone playing. The one case this exists for would be the
// one case it never fired in.
//
// Returns true once and only once. The game loop sets a closing flag on the strength of it
// and keeps running for a frame or two afterwards, and a function that went on saying
// "close" would stack a second exit on top of the first.
func Tick(mouseX, mouseY int, focused, clicking bool) bool {
	if limit == 0 || fired {
		return false
	}

	// The first tick records where the cursor started, rather than counting the jump from
	// the zero value as movement.
	if !primed {
		lastX, lastY, primed = mouseX, mouseY, true
		return false
	}

	moved := mouseX != lastX || mouseY != lastY
	lastX, lastY = mouseX, mouseY

	if focused && (moved || clicking) {
		idle = 0
		return false
	}

	idle++
	if idle < limit {
		return false
	}

	fired = true
	fmt.Printf("idle: no input for %ds, closing\n", limit/ticksPerSecond)
	return true
}
