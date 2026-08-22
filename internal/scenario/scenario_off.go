//go:build !scenario

package scenario

import "github.com/curiousjc/ascend-duel/internal/combat"

// The whole feature compiled out. Every call site is one `if scenario.Active()` away from doing
// nothing, and `scenarios.json` is not embedded in this build at all — the `//go:embed` lives in
// scenario_on.go, so a release binary carries neither the fixture nor the code that reads it.

// Active reports whether a scenario is plugged in. Always false here.
func Active() bool { return false }

// Name is which scenario is running, for a log line. Empty here.
func Name() string { return "" }

// Note is the authored sentence saying what the scenario is for. Empty here.
func Note() string { return "" }

// Rings is what the run should open wearing. Nil here.
func Rings() []string { return nil }

// Hand is the opening hand to deal. Nil here.
func Hand() []combat.Card { return nil }

// Enemy is the record key to fight instead of the climb's own. Empty here.
func Enemy() string { return "" }
