//go:build steam

package steam

import (
	"log"

	steamworks "github.com/hajimehoshi/go-steamworks"
)

// AppID is the game's Steam App ID. **Zero until it has one**, which skips the relaunch check; see
// the package doc.
const AppID uint32 = 0

// running is whether Start brought the API up. Every other call is a no-op without it.
var running bool

// Start brings the Steam API up, and reports whether the game should exit so Steam can relaunch
// it. **Called first thing in main**, before the window, because the overlay hooks the graphics
// device as it is made.
//
// Every failure is logged and leaves Steam off; none of them is a reason not to play.
func Start() (relaunch bool) {
	defer func() {
		if cause := recover(); cause != nil {
			log.Printf("steam: not available (%v); playing without it", cause)
			running, relaunch = false, false
		}
	}()

	if AppID != 0 && steamworks.RestartAppIfNecessary(AppID) {
		return true
	}
	if err := steamworks.Init(); err != nil {
		log.Printf("steam: not available (%v); playing without it", err)
		return false
	}
	running = true
	log.Print("steam: running")
	return false
}

// Running reports whether Start brought the API up.
func Running() bool { return running }

// Tick pumps Steam's callbacks. Called once a frame.
func Tick() {
	if running {
		steamworks.RunCallbacks()
	}
}

// SetAchievements marks achievements earned on Steam and stores them once. Idempotent on Steam's
// side, so the launch-time mirror can send every held achievement without asking which are new.
// **One store for the batch**, because Steam throttles StoreStats.
//
// An API name the Steamworks partner site does not define is refused by Steam and logged; the
// profile has already taken the award, so nothing is lost.
func SetAchievements(apiNames ...string) {
	if !running || len(apiNames) == 0 {
		return
	}
	stats := steamworks.SteamUserStats()
	set := false
	for _, name := range apiNames {
		if stats.SetAchievement(name) {
			set = true
			continue
		}
		log.Printf("steam: %s was refused — is it defined for this App ID?", name)
	}
	if set {
		stats.StoreStats()
	}
}
