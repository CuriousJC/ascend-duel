package main

import (
	"errors"
	"log"
	"time"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/session"
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

// fixedRunSeed pins the run seed for a debugging session, where the same enemies in the same
// order is the point. **Zero means roll a new one from the clock every launch**, which is the
// shipping behaviour and the default.
const fixedRunSeed int64 = 0

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

	// **The run's seed, chosen once, here, and printed.** Everything random in a run derives
	// from it, each consumer seeding its own source — see GlobalState.RunSeed and the
	// five-streams rule in CLAUDE.md.
	//
	// **`fixedRunSeed` is the toggle: zero rolls a new one, anything else pins it.** Pinning
	// is for a debugging session, where the same enemies in the same order is the whole point
	// — the counterpart of `deckSeed` pinning the shuffle, and the same trade. It is one
	// number in one place rather than a flag, because there is no keyboard to toggle it with
	// and a runtime switch would be a control the player can reach.
	//
	// The clock is read exactly once, at the one moment a run is allowed to be unpredictable.
	// The seed is logged either way, because there will eventually be a field to type one
	// back into and a seed nobody can see is a run nobody can ask about.
	g.GlobalState.RunSeed = fixedRunSeed
	if g.GlobalState.RunSeed == 0 {
		g.GlobalState.RunSeed = time.Now().UnixNano()
	}
	log.Printf("run seed %d", g.GlobalState.RunSeed)

	//Load assets into memory one time at startup
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()
	g.GlobalState.FontData = assets.LoadFontData()
	g.GlobalState.ImageData = assets.LoadImageData()
	g.GlobalState.Enemies = data.LoadEnemies()
	g.GlobalState.Duelists = data.LoadDuelists()
	g.GlobalState.Rings = data.LoadRings()

	// **The run starts here** *(2026-08-17)*, because a run outlives every screen and no scene
	// may build one — two would be two runs. It carries the deck, which the combat screen deals
	// from and the post-battle screen alters, and it is where the worn rings and the purse go
	// when buying exists.
	//
	// Built from the authored starting list. When a title-screen "New Run" arrives this moves
	// there and becomes one line in that action instead.
	// **A scenario dresses the run before it starts.** Compiled out unless `-tags scenario`, in
	// which case this is the seat that puts a chosen set of rings on — `StartingRings` is the same
	// debug seat a hand-edited list would use, so nothing new has to be able to force a worn row.
	// See internal/scenario.
	if scenario.Active() {
		session.StartingRings = scenario.Rings()
	}

	g.GlobalState.Run = session.Start(g.GlobalState.Enemies, g.GlobalState.RunSeed)

	// The score is a MIDI file synthesised to PCM here at startup rather than a
	// recorded track — see internal/music for why. It loops for the whole session
	// across every screen, and there is no way to mute it yet, which wants an
	// on-screen button rather than a hotkey: the input vocabulary has no keyboard.
	//
	// Not having a sound device is not a reason to refuse to run, so a failure here is
	// reported and stepped over. Nothing below this line depends on it.
	if err := music.Start(assets.LoadMusic()["ascending_mid"]); err != nil {
		log.Println(err)
	}

	// Widgets are no longer wired up here. Each scene builds its own in Init, so main
	// does not need to know which screens have buttons or what pressing them does.

	//Run game is the infinite loop. A deliberate quit comes back as game.ErrClosing,
	//which is a normal exit rather than a failure — anything else is a real error.
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, game.ErrClosing) {
		log.Fatal(err)
	}
}
