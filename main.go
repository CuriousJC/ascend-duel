package main

import (
	"errors"
	"log"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// version is what this build calls itself, injected at link time by the release workflow:
//
//	go build -ldflags "-X main.version=v0.1.0" .
//
// It defaults to "dev" because an ordinary `go run .` injects nothing, and a build that
// guessed at a version number would be worse than one that admits it does not have one.
//
// **The point is that a bug report can name a build.** Someone who downloaded an exe and
// says "it crashed on the third fight" is only useful if the build they ran is identifiable,
// and the filename stops travelling with the binary the moment it is renamed or a screenshot
// is all you have. It is shown in the window title, which any screenshot of the window
// carries, and on the title screen.
var version = "dev"

func main() {
	// The window opens at the internal resolution; Layout keeps that resolution fixed
	// whatever the window is resized to afterwards.
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Ascending Duel " + version)
	ebiten.SetWindowClosingHandled(true)

	//Create our Game instance
	g := game.NewGame()

	// Both off: the screen is far enough along that the grid is now in the way of judging
	// how it actually looks. Turn placement back on when moving things, and gameplay on to
	// watch the opponent plan — remembering that what you see with it on is not what a
	// player sees.
	g.GlobalState.DebugPlacement = false
	g.GlobalState.DebugGameplay = false

	// Handed to the state rather than read from this package by the screens, so nothing
	// below main has to import it — the dependency direction only ever points down.
	g.GlobalState.Version = version

	//Load assets into memory one time at startup
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()
	g.GlobalState.Combatants = data.LoadCombatants()

	// Widgets are no longer wired up here. Each scene builds its own in Init, so main
	// does not need to know which screens have buttons or what pressing them does.

	//Run game is the infinite loop. A deliberate quit comes back as game.ErrClosing,
	//which is a normal exit rather than a failure — anything else is a real error.
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, game.ErrClosing) {
		log.Fatal(err)
	}
}
