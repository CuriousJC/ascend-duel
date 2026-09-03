package actions

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// OpenSettings goes to the settings screen, remembering where the player was so Back can put
// them there. The chrome's cog does the same thing from every other screen — see game/chrome.go,
// which cannot call this because actions sits below it and the frame is not a scene.
func OpenSettings(gs *state.GlobalState) {
	gs.ReturnScreen = gs.ActiveScreen
	gs.ActiveScreen = state.Settings
	gs.NewScreen = true
}

// OpenAchievements and OpenCredits go to the two menu screens, remembering where the player was so
// each screen's Back can put them there.
//
// **Three functions rather than one taking a destination**, matching OpenSettings. A shared
// `openScreen(gs, dest)` would be shorter and would also be the seam through which a *run* screen
// gets opened this way — and a run screen entered without its phase being set is a screen showing a
// station the run is not standing on. These three are the screens that are allowed to work like
// this, and the list being explicit is what says so.
func OpenAchievements(gs *state.GlobalState) {
	gs.ReturnScreen = gs.ActiveScreen
	gs.ActiveScreen = state.Achievements
	gs.NewScreen = true
}

func OpenCredits(gs *state.GlobalState) {
	gs.ReturnScreen = gs.ActiveScreen
	gs.ActiveScreen = state.Credits
	gs.NewScreen = true
}

// QuitGame asks the loop to stop. game.Update turns this into game.ErrClosing.
func QuitGame(gs *state.GlobalState) {
	fmt.Println("Exit Button Clicked")
	gs.ShouldClose = true
}
