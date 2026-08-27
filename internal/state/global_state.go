package state

import (
	"image"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
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

	// RunSeed is the number every random choice in a run derives from, set once by main and
	// never re-rolled while the process lives.
	//
	// **It is the per-run seed the determinism rules have been writing against**, arriving
	// early because the enemy order wanted it (2026-08-11). The rules say a run will one day
	// be replayable from a seed typed into a field; this is that number, generated from the
	// clock for now because there is no field to type it into and no Session to hold it.
	//
	// **It is always in `seeds.Space` — a six-character Crockford base32 run code** *(2026-08-25)*, so every
	// run can be written down and typed back. `main` folds the clock into that range with
	// `seeds.Normalize` and prints `seeds.Code`; nothing downstream widens it again. Zero is
	// a legitimate run (`000000`) rather than "unset", so anything meaning "no seed chosen"
	// has to say so with its own flag.
	//
	// **Reading the clock here does not break "no time.Now() in game rules".** That rule is
	// about decisions taken *during* a run — a rule that consults the wall clock cannot be
	// replayed. Choosing the seed is the one place a run is allowed to be unpredictable,
	// which is exactly why it is done once, in main, and written to the log.
	//
	// It is a plain int64, so state stays free of imports. Every consumer seeds its **own**
	// source from it — never a shared one — per the five-streams rule in CLAUDE.md.
	RunSeed int64

	Debug1, Debug2 string
	ActiveScreen   ActiveScreen
	NewScreen      bool
	Count          int
	CountSecond    int
	MouseX         int
	MouseY         int
	ShouldClose    bool

	// ModalOpen is a scene declaring that it has a dialog up, so the game's own chrome —
	// today just the mute button — stands down rather than sitting live on top of it.
	//
	// **It is genuinely shared and that is why it is here**: it is the one thing a scene and
	// the frame around it have to agree on. The deck overlay's rule is that everything behind
	// it goes dead and the single control that closes it is the only one that still looks
	// live, because there is no Escape key and no right click to fall back on — a modal has to
	// make its exit the brightest thing on screen or it is a trap. Chrome drawn after the
	// scene would break that by construction.
	//
	// **The frame clears it every tick and a scene that has a dialog re-asserts it**, rather
	// than each scene being trusted to turn it off. A screen left while its overlay was open
	// would otherwise leave this stuck on and the chrome invisible for the rest of the
	// session, on screens that have never heard of a modal.
	ModalOpen bool

	// InputFocus is the one rectangle still accepting clicks, and InputGated is whether the
	// restriction is on at all. Together they are the tutorial's shield: while a step says "press
	// this", the cursor anywhere else clicks nothing.
	//
	// **It lives here for the reason ModalOpen does** — it is a thing a scene and the frame around
	// it have to agree on, and the mute button in the corner is exactly the control that would
	// otherwise stay live under a lesson telling the player there is only one thing to press.
	//
	// **It gates on the cursor, not on the widget.** Every click in this game starts as a point:
	// `systems.UpdateButton` hit-tests one, and the handful of places in `internal/screens` that
	// read the mouse directly all build one first. So one predicate over the cursor position
	// covers every control there is, where a per-widget rule would be a list that a newly added
	// widget is missing from — and a control that is silently *not* covered by the shield is the
	// failure this is built to prevent.
	//
	// **The frame clears it every tick and the tutorial re-asserts it**, the same discipline
	// ModalOpen keeps: a screen left mid-step must not leave the rest of the session unclickable.
	InputFocus image.Rectangle
	InputGated bool

	//Layout
	ScreenWidth  int
	ScreenHeight int

	//Data
	// **Two rosters since 2026-08-11**, where there was one map of combatants. The player and
	// the opponents stopped sharing a struct — an enemy has a plan style, a portrait and an
	// affix pool, a duelist has a card back — so one map would have been a map of records
	// where most fields were empty. See data/duelists_data.go.
	Enemies map[string]data.EnemyData

	// Bosses is the stairway protectors, in their own map for the reason data/bosses_data.go
	// gives: they are placed by a different rule and must not be shuffled into an ordinary
	// room. A screen hydrating an opponent looks here when the roster does not know the record.
	Bosses   map[string]data.BossData
	Duelists map[string]data.DuelistData

	// Rings is what the player can equip. **Genuinely global for the same reason the rosters
	// are** — it is loaded once from data/rings.json and no screen owns it. What is *equipped*
	// is not here: that is run state and belongs on Run, below — bought and sold in the shop.
	Rings map[string]data.RingData

	// Run is what the player is carrying up the tower — the deck today, the worn rings and the
	// purse next. **Genuinely global**: the combat screen deals from it and the post-battle
	// screen alters it, and it has to outlive a fight, which no scene does.
	//
	// **This is the one field that makes `state` import `internal/combat`, transitively**
	// *(2026-08-17)*. The rule it bends says global state must not import `combat`, `entities`
	// or `models` — written to stop *screen* state leaking back in here, which is a different
	// thing from a run. A run belongs beside ActiveScreen, not on whichever screen happened to
	// need it first.
	//
	// Nil until main builds it. Scenes must not create one: two would be two runs.
	Run *session.Session

	// Store is the directory the player's two files live in, and Profile is what has been read
	// out of the first of them. **Genuinely global for the reason Run is**: the profile outlives
	// every screen, is written from three different scenes, and no scene owns it.
	//
	// **Profile is never nil once main has run**, even on a machine that cannot save — an inert
	// store hands back a fresh profile rather than nothing, so a caller awards an achievement
	// without first asking whether the filesystem cooperated. See internal/profile.
	Store   profile.Store
	Profile *profile.Profile

	// Resumed is whether this session picked a run up off the disk rather than starting one.
	// **Read once, by the tutorial's trigger**: a lesson that opens by describing the hand the
	// player is holding cannot begin halfway up a tower.
	Resumed bool

	// ProfileWritable is whether what was read may be written back. False for a corrupt file and
	// for one written by a newer build — see profile.LoadProfile, which holds the whole migration
	// policy. Awards still land in memory for the session; nothing reaches the disk.
	ProfileWritable bool

	// ReturnScreen is where the settings screen goes back to.
	//
	// **The settings screen is the only one that can be entered from anywhere**, so it is the only
	// one that cannot name its successor the way every other screen does — `advance` walks the run
	// forward, and settings is not a station of a run. Whoever opens it records where the player
	// was; Back puts them there.
	//
	// It deliberately does not touch `session.Phase`. The run stays exactly where it was standing,
	// which is what makes opening settings mid-duel a look at a dialog rather than a decision.
	ReturnScreen ActiveScreen

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

	// PostBattle is the first of the between-fight scenes: one alteration to the deck, offered
	// from a hand dealt off it. The shop follows it and a room choice is to come, and each is an
	// ordinary scene here rather than a mode of the combat screen.
	PostBattle

	// Shop is the second between-fight scene: rings on a shelf, bought and sold with vitae.
	Shop

	Credits

	// Settings is the program's own screen: how loud the score is and how fast the game moves.
	//
	// **It is reachable from everywhere, so it is the one screen that has to remember where it
	// came from** — see ReturnScreen. Appended rather than filed next to Title because
	// ActiveScreen is append-only like every other ordinal in the game.
	Settings
)

func (active ActiveScreen) String() string {
	switch active {
	case Title:
		return "Title"
	case Ascend:
		return "Ascend"
	case Combat:
		return "Combat"
	case PostBattle:
		return "PostBattle"
	case Shop:
		return "Shop"
	case Settings:
		return "Settings"
	case Credits:
		return "Credits"
	default:
		return "Unknown"
	}
}

// InputAllowed reports whether a click at this point may do anything.
//
// **The one question every input site asks**, and the reason the tutorial's gating did not have
// to be written into each of them separately. With no gate up it is always true, so a caller that
// adds the check costs nothing in the ordinary case.
//
// A caller that has a rectangle rather than a point — a hover band, a drag target — asks about the
// cursor, because that is what the player is actually pointing with.
func (gs *GlobalState) InputAllowed(at image.Point) bool {
	if !gs.InputGated {
		return true
	}
	return at.In(gs.InputFocus)
}

// CursorAllowed is InputAllowed asked about wherever the cursor is right now, which is what
// almost every caller means.
func (gs *GlobalState) CursorAllowed() bool {
	return gs.InputAllowed(image.Pt(gs.MouseX, gs.MouseY))
}
