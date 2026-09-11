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

// Parasites is what the run should open holding in its bucket. Nil here.
func Parasites() []string { return nil }

// Stones is what the run should open carrying in its pouch. Nil here.
func Stones() []string { return nil }

// Enemy is the record key to fight instead of the climb's own. Empty here.
func Enemy() string { return "" }

// Teach reports whether this scenario starts the tutorial. Always false here.
func Teach() bool { return false }

// Seed is the run code to pin. Empty here, so the clock decides.
func Seed() string { return "" }

// Deck is the replacement deck. Nil here, so the authored one is used.
func Deck() []combat.Card { return nil }

// Screen is which scene to open on. Always the duel here.
func Screen() string { return "combat" }

// Fight, Vitae and Life are the run state a jumped-in screen would need. All zero here.
func Fight() int { return 0 }
func Vitae() int { return 0 }
func Life() int  { return 0 }

// DummyLife and DummyRounds are what a dummy fight would be given. Stated here as well as in
// scenario_on.go so a call site can name them in either build — they are two constants, not the
// fixture, and nothing reads them unless Dummy() is true.
const (
	DummyLife   = 999999
	DummyRounds = 999
)

// Dummy reports whether the fight is unkillable in both directions. Always false here.
func Dummy() bool { return false }

// RoundLimit is the clock the scenario wants. Zero here, so the run keeps its own.
func RoundLimit() int { return 0 }

// Actions is the action-point budget the scenario wants. Zero here, so the record's own stands.
func Actions() int { return 0 }
