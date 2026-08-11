// Description: state is the shared state passed to all the other components of the game.
package state

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GlobalState is what is genuinely shared: input, timing, layout, loaded resources,
// and which screen is active. It is threaded by pointer into every scene.
//
// A screen's own working state does NOT belong here — that lives on the scene that
// owns it (see screens.Scene). This struct previously carried the combat screen's
// duel log, playback cursor and combatants, which meant every screen could see them
// and none of them were anyone else's business.
type GlobalState struct {
	//Global Game Stuff
	// Two independent debug flags, because they answer different questions and are wanted
	// at different times. DebugPlacement is about where things are drawn — the grid, the
	// rulers, the scratch strings — and is safe to leave on while playing. DebugGameplay
	// changes what the player is allowed to know, so leaving it on means not playing the
	// real game. Neither may ever change an outcome; both are views.
	DebugPlacement bool
	DebugGameplay  bool

	// Version is what this build calls itself, set once by main from a linker-injected
	// string. It is genuinely global — the window title and the title screen both want it
	// and neither owns it — and it is a plain string, so state stays free of imports.
	//
	// **It exists so a bug report can name a build.** A screenshot or a "it crashed" from
	// someone who downloaded an exe is unusable if every build looks alike.
	Version string

	Debug1, Debug2 string
	ActiveScreen   ActiveScreen
	NewScreen      bool
	Count          int
	CountSecond    int
	MouseX         int
	MouseY         int
	ShouldClose    bool

	//Layout
	ScreenWidth  int
	ScreenHeight int

	//Data
	// **Two rosters since 2026-08-11**, where there was one map of combatants. The player and
	// the opponents stopped sharing a struct — an enemy has a plan style, a portrait and an
	// affix pool, a duelist has a card back — so one map would have been a map of records
	// where most fields were empty. See data/duelists_data.go.
	Enemies  map[string]data.EnemyData
	Duelists map[string]data.DuelistData

	//Assets
	Assets map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts  map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct

	// FontData is the same fonts as raw file bytes.
	//
	// Fonts above is Ebitengine's type, which can only draw into an *ebiten.Image.
	// internal/cards renders into a plain Go image so a command-line tool can call it
	// with no window, so it sets text through golang.org/x/image and needs the file
	// itself. Both come from the same embedded bytes, so the game and the contact sheet
	// cannot end up in different fonts.
	FontData map[string][]byte

	// ImageData is the same idea for pictures: raw file bytes for the images something has
	// to decode itself rather than take as an *ebiten.Image.
	//
	// **Only the images that need it are in here**, not everything in Assets. What needs it
	// is anything drawn *into* a card — the ring art, the enemy portraits — because
	// internal/cards renders to a plain Go image with no graphics context. Both maps read
	// the same embedded bytes, so the game and the contact sheet cannot end up showing
	// different pictures.
	ImageData map[string][]byte
}

// NewGlobalState used at the start of the game to start us off
func NewGlobalState() *GlobalState {
	return &GlobalState{
		// Boots straight into Combat rather than Title: that screen is where the work
		// is, and clicking through the title every run gets old. Put this back to Title
		// once the combat screen stops being the thing under construction.
		ActiveScreen: Combat,
		NewScreen:    true,
		Assets:       make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:        make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
	}
}

// PctX and PctY convert a percentage of the screen into a pixel coordinate. They are
// the intended way to place things: "40% across" reads better than 512 and matches the
// percentage ruler the debug overlay draws.
//
// These replaced a dozen cached fields for halves, thirds and quarters. Named fractions
// do not compose — there was no field for 40%, and the fix for that is not a field
// called TwoFifthsX.
//
// Percentages anchor a group; offsets *within* a group stay in pixels. Three buttons
// spaced 150px apart below a 33% anchor must keep that spacing when the anchor moves,
// which independent percentages would not do. Sizes are never percentages either.
func (gs *GlobalState) PctX(pct int) int { return gs.ScreenWidth * pct / 100 }
func (gs *GlobalState) PctY(pct int) int { return gs.ScreenHeight * pct / 100 }

type ActiveScreen int

const (
	Title ActiveScreen = iota
	Ascend
	Combat
	Credits
)

func (active ActiveScreen) String() string {
	switch active {
	case Title:
		return "Title"
	case Ascend:
		return "Ascend"
	case Combat:
		return "Combat"
	case Credits:
		return "Credits"
	default:
		return "Unknown"
	}
}
