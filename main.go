package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/crashlog"
	"github.com/curiousjc/ascend-duel/internal/game"
	"github.com/curiousjc/ascend-duel/internal/journal"
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
// and the filename stops traveling with the binary the moment it is renamed or a screenshot
// is all you have. It is shown in the window title, which any screenshot of the window
// carries, and on the title screen.
var version = "dev"

// fixedRunSeed pins the run seed for a debugging session, where the same enemies in the same
// order is the point. **Empty means roll a new one from the clock every launch**, which is the
// shipping behavior and the default.
//
// It is written as a **run code** — six Crockford base32 characters, the same spelling a player reads off
// the screen and will one day type back in. A code that is not one fails the launch rather than
// silently rolling a fresh run, because a pin nobody notices is off is worse than no pin.
const fixedRunSeed = ""

// The allowance left for the window's own furniture and the desktop's: a title bar and a border
// at the top, a taskbar at the bottom. **Deliberately generous** — a window a few pixels smaller
// than it could be is a window with a margin around it, and one a few pixels larger is one whose
// bottom row of controls is behind the taskbar, which is the bug this whole function exists for.
const (
	windowChromeX = 32
	windowChromeY = 96
)

// windowSize is how big the window opens: **the largest whole eighth of the internal resolution
// that fits on the monitor** *(2026-09-04)*.
//
// **Eighths rather than a best fit**, because the window and the internal resolution are the two
// ends of a scale factor and a ragged one resamples pixel art by a fraction that changes with the
// monitor. A whole eighth keeps the ratio simple and the same on every machine that lands on it,
// and 8/8 — the 1:1 case the art is authored for — is reached on any display with room for it
// rather than having to be asked for.
//
// **It is a launch size and not a constraint.** The player can resize, maximize, or take the
// fullscreen toggle on the settings screen; Layout does not care what any of them do.
//
// **A monitor that will not answer falls back to three quarters**, which is the figure this was
// before it was computed and is small enough to fit anything the game will run on.
func windowSize() (int, int) {
	w, h := 0, 0
	if m := ebiten.Monitor(); m != nil {
		w, h = m.Size()
	}
	if w <= 0 || h <= 0 {
		return state.ScreenWidth * 3 / 4, state.ScreenHeight * 3 / 4
	}

	for eighths := 8; eighths > 2; eighths-- {
		ww := state.ScreenWidth * eighths / 8
		wh := state.ScreenHeight * eighths / 8
		if ww <= w-windowChromeX && wh <= h-windowChromeY {
			return ww, wh
		}
	}
	// Smaller than a quarter is not a window anybody can play in; let it be too big and be
	// resized rather than open at something unreadable.
	return state.ScreenWidth * 2 / 8, state.ScreenHeight * 2 / 8
}

func main() {
	// **The window is not the internal resolution, and that is the point** *(2026-09-04)*. Layout
	// reports state.ScreenWidth x ScreenHeight whatever the window is, so the two are free to
	// differ and Ebitengine scales between them. They were the same number until the internal
	// resolution went to 1920x1080, at which point opening the window at it put a 1080-pixel-tall
	// window on a 1080-pixel-tall desktop: the title bar and the taskbar take their cut off the
	// top and the bottom, so the button strip and the deck pile were simply not on screen.
	ebiten.SetWindowSize(windowSize())
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Ascending Duel " + version)
	ebiten.SetWindowClosingHandled(true)

	//Create our Game instance
	g := game.NewGame()

	// **The store is opened before anything is loaded, so a panic during the load has somewhere to
	// write.** Open creates no directory and cannot fail — see internal/profile — so this costs
	// nothing on the launches where nothing goes wrong, and it is the difference between a crash
	// report and a console nobody is reading on the launches where something does.
	g.GlobalState.Store = profile.Open()
	g.GlobalState.Version = version
	defer reportLoadPanic(g)

	// Both off: the screen is far enough along that the grid is now in the way of judging
	// how it actually looks. Turn placement back on when moving things, and gameplay on to
	// watch the opponent plan — remembering that what you see with it on is not what a
	// player sees.
	g.GlobalState.DebugPlacement = false
	g.GlobalState.DebugGameplay = false

	// The animation gallery's door. **On while the gallery is being built**, which is the one of
	// the three that is not off today — it belongs off before this ships, on the same argument the
	// other two are under. See screens.AnimationsScene, which is the shared vocabulary for talking
	// about one of the game's gestures.
	g.GlobalState.DebugAnimations = true

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
	g.GlobalState.Motifs = data.LoadMotifs()
	g.GlobalState.Records = data.MotifRecords(g.GlobalState.Motifs)
	g.GlobalState.Tower = data.LoadTower()
	data.MustBeClimbable(g.GlobalState.Motifs, g.GlobalState.Tower.Floors)
	g.GlobalState.Duelists = data.LoadDuelists()
	g.GlobalState.Relics = data.LoadRelics()

	// **The run starts here** *(2026-08-17)*, because a run outlives every screen and no scene
	// may build one — two would be two runs. It carries the deck, which the combat screen deals
	// from and the post-battle screen alters, and it is where the worn relics and the purse go
	// when buying exists.
	//
	// Built from the authored starting list. When a title-screen "New Run" arrives this moves
	// there and becomes one line in that action instead.
	// **A scenario dresses the run before it starts.** Compiled out unless `-tags scenario`, in
	// which case this is the seat that puts a chosen set of relics on — `StartingRelics` is the same
	// debug seat a hand-edited list would use, so nothing new has to be able to force a worn row.
	// See internal/scenario.
	if scenario.Active() {
		// **The fingers are set before the relics**, and both before the run is built: `New` wears
		// the list as it goes, so a cap raised after the fact would arrive too late to let a sixth
		// relic on. See session.StartingRelicSlots.
		session.StartingRelicSlots = scenario.RelicSlots()
		session.StartingRelics = scenario.Relics()

		// **A chosen deck, where the relics are a chosen row.** Nil unless the fixture says
		// otherwise, so this is the authored deck for every scenario that does not care.
		session.StartingDeckList = scenario.Deck()

		// **A chosen sack, on the same terms.** The board piece is otherwise a shop and a fight
		// away from any launch; see internal/scenario.
		session.StartingRunes = scenario.Runes()
		session.StartingStones = scenario.Stones()
		session.StartingEssences = scenario.Essences()
	}

	// **The profile is opened before the run, because it can decide what the run is.** A run in
	// progress is resumed rather than started, and a player who has never been taught is taught.
	//
	// Nothing here is fatal: a missing, corrupt or unwritable profile is a new player, and a
	// machine that cannot write still plays the game. See internal/profile.
	prof, writable, err := profile.LoadProfile(g.GlobalState.Store)
	if err != nil {
		crashlog.Tell("Your profile could not be read, so this is a new player: %v", err)
	}
	g.GlobalState.Profile, g.GlobalState.ProfileWritable = prof, writable

	// **The install id is made here and saved in the same breath.** It groups several crash reports
	// from one player and identifies nobody — see profile.Profile.InstallID, and internal/crashlog
	// for what it is for. Loading deliberately does not mint one: a profile that may not be written
	// over would otherwise carry an id it could never record, and an id that changes every launch is
	// the one thing an install id may not be.
	if writable && prof.EnsureInstallID() {
		if err := profile.SaveProfile(g.GlobalState.Store, prof); err != nil {
			crashlog.Note("could not record the install id: %v", err)
		}
	}

	// **What the player chose about the program, put into force before anything reads it.** The
	// music level has to be in before Start opens the device, or a returning player gets a moment
	// of the wrong volume; the speed has to be in before the first scene's Init, since a screen
	// can compute a duration in it. See internal/screens/settings.go.
	screens.ApplySettings(prof.Settings)

	// **The journal is opened before the run, because BootRun is what writes its first line.** It
	// touches no file here — a journal that made its file at startup would leave one behind for
	// every player who launched the game and never played — so this is a store and a sentence.
	//
	// **The wording of the failure lives here rather than in internal/journal.** A crash report
	// takes a copy of the journal, so internal/crashlog has to be able to name that file; one of
	// the two has to point at the other, and the one that knows what a player should be told is
	// this one. A run that is not being written down cannot be got back, which is why it is a Tell.
	g.GlobalState.Journal = journal.New(g.GlobalState.Store, func(err error) {
		crashlog.Tell("This run's choices are not being recorded: %v", err)
	})

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

		// **And it may move the clock**, which is a run-level number rather than a screen's —
		// `session.SetRoundLimit` is the one door, and it clamps rather than obeying, so a fixture
		// cannot stop the clock through it. A dummy fight needs this: five rounds is five rounds
		// whoever is standing there, and an unkillable opponent otherwise kills the player on the
		// clock at the end of round five. See internal/scenario.
		if n := scenario.RoundLimit(); n > 0 && g.GlobalState.Run != nil {
			g.GlobalState.Run.SetRoundLimit(n)
			log.Printf("scenario %s: a %d-round clock", scenario.Name(), n)
		}

		// **And it may widen the hand.** Unlike the clock this had to be in place *before* the run
		// was built — see the StartingRelicSlots line above — so what happens here is a resumed run
		// being brought up to the fixture's number, and a log line saying what the hand is on.
		if n := scenario.RelicSlots(); n > 0 && g.GlobalState.Run != nil {
			g.GlobalState.Run.SetRelicSlots(n)
			log.Printf("scenario %s: %d relic slots", scenario.Name(), n)
		}
	}

	// The score is a MIDI file synthesized to PCM here at startup rather than a
	// recorded track — see internal/music for why. It loops for the whole session
	// across every screen. How loud it is comes off the profile above — see the settings screen,
	// which is reached from the cog in the game's chrome. That is a button rather than a hotkey:
	// the input vocabulary has no keyboard.
	//
	// Not having a sound device is not a reason to refuse to run, so a failure here is
	// reported and stepped over. Nothing below this line depends on it.
	if err := music.Start(assets.LoadMusic()["ascending_mid"]); err != nil {
		// **Noted rather than told.** A machine with no audio device is a machine that plays the
		// game in silence, which the volume bar on the settings screen already says out loud; a box
		// in the player's way about it would be the game complaining about their hardware.
		crashlog.Note("music: %v", err)
	}

	// Widgets are no longer wired up here. Each scene builds its own in Init, so main
	// does not need to know which screens have buttons or what pressing them does.

	//Run game is the infinite loop. A deliberate quit comes back as game.ErrClosing,
	//which is a normal exit rather than a failure — anything else is a real error.
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, game.ErrClosing) {
		log.Fatal(err)
	}
}

// reportLoadPanic catches a panic raised before the game loop is running.
//
// **The recovers in internal/game cover Update and Draw, and everything above is this one.** The
// catalogs are loaded, validated and cross-checked before a window opens — data.MustBeClimbable is
// a panic by design — so the one failure most likely to reach a player who has just downloaded an
// exe is a failure with no frame to have happened in.
//
// **It exits rather than drawing anything.** There is no window yet and no fonts to draw with, so
// the report and a line in the log are the whole of what can be said. A crash screen needs a game,
// and a game is exactly what failed to start.
func reportLoadPanic(g *game.Game) {
	cause := recover()
	if cause == nil {
		return
	}
	gs := g.GlobalState
	report := crashlog.Build(crashlog.State{
		Version:   gs.Version,
		Screen:    "loading",
		RunSeed:   gs.RunSeed,
		InstallID: installID(gs),
	}, cause, crashlog.Stack())

	path, err := crashlog.Write(gs.Store, report)
	if err != nil {
		log.Printf("crash while loading: %v (no report written: %v)", cause, err)
	} else {
		log.Printf("crash while loading: %v (report written to %s)", cause, path)
	}
	os.Exit(1)
}

// installID is the id off a profile that may not have been loaded yet.
func installID(gs *state.GlobalState) string {
	if gs.Profile == nil {
		return ""
	}
	return gs.Profile.InstallID
}

// startScenarioAt puts a scenario's run where the fixture says, and the game on the screen that
// draws it. **One function so the guarded call site above stays one line**, which is what keeps the
// whole feature deletable in a commit.
//
// **Life defaults to whatever the duelist has**, because a scenario that only wants to see the shop
// should not have to say how much life the last fight left. Zero would be a corpse on the card.
func startScenarioAt(g *game.Game) {
	gs := g.GlobalState

	// **The record's own ceiling is what the fixture's Life is measured against.** A run wearing a
	// relic that raises max life is carrying the same *wound* under a higher ceiling, which is how
	// the game treats every wound — see Session.LifeAtFightStart.
	max := 0
	if d, ok := gs.Duelists["Fighter1"]; ok {
		max = d.HP
	}

	life := scenario.Life()
	if life <= 0 {
		life = max
	}
	gs.Run.JumpTo(scenario.Fight(), scenario.Vitae(), life, max)

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
