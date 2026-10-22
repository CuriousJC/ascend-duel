//go:build !steam

package steam

// Start does nothing in this build and never asks for a relaunch. See the package doc.
func Start() (relaunch bool) { return false }

// Running is always false in this build.
func Running() bool { return false }

// Tick does nothing in this build.
func Tick() {}

// SetAchievements does nothing in this build.
func SetAchievements(apiNames ...string) {}
