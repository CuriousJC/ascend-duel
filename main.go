package main

import (
	"errors"
	"log"
	"time"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
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
// order is the point. **Empty means roll a new one from the clock every launch**, which is the
// shipping behaviour and the default.
//
// It is written as a **run code** — six Crockford base32 characters, the same spelling a player reads off
// the screen and will one day type back in. A code that is not one fails the launch rather than
// silently rolling a fresh run, because a pin nobody notices is off is worse than no pin.
const fixedRunSeed = ""

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
	// `pinned` rather than a zero check on the seed itself: zero is the perfectly good run
	// `000000` now, so "unset" needs saying separately.
	pinned := false
	if fixedRunSeed != "" {
		seed, err := seeds.Parse(fixedRunSeed)
		if err != nil {
			log.Fatalf("fixedRunSeed %q: %v", fixedRunSeed, err)
		}
		g.GlobalState.RunSeed, pinned = seed, true
	}

	// **A scenario's own seed outranks the constant above.** A fixture that is about a particular
	// deal — the tutorial is, since it promises the player what they are holding — cannot be at
	// the mercy of whether `fixedRunSeed` was left at zero. Compiled out with the rest of the
	// package; see internal/scenario.
	if scenario.Active() && scenario.Seed() != "" {
		seed, err := seeds.Parse(scenario.Seed())
		if err != nil {
			log.Fatalf("scenario %s: seed %q: %v", scenario.Name(), scenario.Seed(), err)
		}
		g.GlobalState.RunSeed, pinned = seed, true
	}

	if !pinned {
		g.GlobalState.RunSeed = seeds.Normalize(time.Now().UnixNano())
	}

	// **Whether the seed was chosen or rolled, written down where a screen can read it.** The
	// title screen's New Run rerolls, and a pin set in this file must not be undone by a button.
	// See state.SeedPinned and screens.NewRun.
	g.GlobalState.SeedPinned = pinned

	//Load assets into memory one time at startup
	g.GlobalState.Assets = assets.LoadAssets()
	g.GlobalState.Fonts = assets.LoadFonts()
	g.GlobalState.FontData = assets.LoadFontData()
	g.GlobalState.ImageData = assets.LoadImageData()
	g.GlobalState.Enemies = data.LoadEnemies()
	g.GlobalState.Bosses = data.LoadBosses()
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

		// **A chosen deck, where the rings are a chosen row.** Nil unless the fixture says
		// otherwise, so this is the authored deck for every scenario that does not care.
		session.StartingDeckList = scenario.Deck()

		// **A chosen bucket, on the same terms.** The board piece is otherwise a shop and a fight
		// away from any launch; see internal/scenario.
		session.StartingParasites = scenario.Parasites()
		session.StartingStones = scenario.Stones()
	}

	// **The profile is opened before the run, because it can decide what the run is.** A run in
	// progress is resumed rather than started, and a player who has never been taught is taught.
	//
	// Nothing here is fatal: a missing, corrupt or unwritable profile is a new player, and a
	// machine that cannot write still plays the game. See internal/profile.
	g.GlobalState.Store = profile.Open()
	prof, writable, err := profile.LoadProfile(g.GlobalState.Store)
	if err != nil {
		log.Printf("profile: %v — carrying on as a new player", err)
	}
	g.GlobalState.Profile, g.GlobalState.ProfileWritable = prof, writable

	// **What the player chose about the program, put into force before anything reads it.** The
	// music level has to be in before Start opens the device, or a returning player gets a moment
	// of the wrong volume; the speed has to be in before the first scene's Init, since a screen
	// can compute a duration in it. See internal/screens/settings.go.
	screens.ApplySettings(prof.Settings)

	screens.BootRun(g.GlobalState)

	// **Logged after the run is built, not before it.** A resumed run brings its own seed and a
	// taught one brings the script's, so a code printed at the moment one was rolled would name a
	// tower nobody is playing.
	log.Printf("run code %s", seeds.Code(g.GlobalState.RunSeed))

	// **A scenario may also open the game somewhere other than the first duel** *(2026-08-22)*.
	// The run is put in the named room with the named purse, and the phase it lands on decides
	// which scene draws it — see internal/screens/flow.go, which is the one table mapping the two.
	// Compiled out with the rest of the package.
	if scenario.Active() {
		startScenarioAt(g)
	}

	// The score is a MIDI file synthesised to PCM here at startup rather than a
	// recorded track — see internal/music for why. It loops for the whole session
	// across every screen. How loud it is comes off the profile above — see the settings screen,
	// which is reached from the cog in the game's chrome. That is a button rather than a hotkey:
	// the input vocabulary has no keyboard.
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

// startScenarioAt puts a scenario's run where the fixture says, and the game on the screen that
// draws it. **One function so the guarded call site above stays one line**, which is what keeps the
// whole feature deletable in a commit.
//
// **Life defaults to whatever the duelist has**, because a scenario that only wants to see the shop
// should not have to say how much life the last fight left. Zero would be a corpse on the card.
func startScenarioAt(g *game.Game) {
	gs := g.GlobalState

	life := scenario.Life()
	if life <= 0 {
		if d, ok := gs.Duelists["Fighter1"]; ok {
			life = d.HP
		}
	}
	gs.Run.JumpTo(scenario.Fight(), scenario.Vitae(), life)

	switch scenario.Screen() {
	case "reward":
		gs.Run.SetPhase(session.PhaseReward)
		gs.ActiveScreen = state.PostBattle
	case "shop":
		gs.Run.SetPhase(session.PhaseShop)
		gs.ActiveScreen = state.Shop
	default:
		gs.Run.SetPhase(session.PhaseFight)
		gs.ActiveScreen = state.Combat
	}
	log.Printf("scenario %s: opening on the %s screen, room %d",
		scenario.Name(), scenario.Screen(), scenario.Fight())
}
