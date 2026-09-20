package state

import (
	"image"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ScreenWidth and ScreenHeight are the fixed internal resolution. Layout always reports
// these regardless of window size, so Ebitengine scales and letterboxes to fit and every
// absolute coordinate in the game is safe against resizing.
//
// **1920x1080 since 2026-09-04** *(owner's call)*, from the 1280x960 the game had been since
// it existed. Two things drove it. The shape was 4:3, which is not a PC game shape any more and
// left every screen narrow — the shop could not fit three panes of full-size cards side by side,
// and `handPitch` compressed a hand of eight until the cards overlapped by 34 pixels. And the
// *scale* was wrong: Layout's buffer is stretched to the window, so on a 1080p monitor 960 tall
// meant a factor of 1.125 — a non-integer resample of art the glyph rules go to some length to
// keep at 1:1 — plus pillarboxing to 1440 of 1920. At 1080p the factor is exactly 1, and exactly
// 2 on a 4K panel.
//
// **They live here rather than in internal/game, which is where they were until the change**
// *(2026-09-04)*. That package sits above internal/screens in the import graph, so no screen and
// no screen's test could name them — which is why fourteen test files had 1280 and 960 typed into
// them and would have gone on passing while testing a screen size the game no longer runs at.
// internal/state is below everything that measures a screen, so this is now one edit.
const (
	ScreenWidth  = 1920
	ScreenHeight = 1080
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

	// DebugAnimations opens the animation gallery's door: a square in the frame's bottom strip that
	// is drawn only while this is on. **A third flag rather than a lodger on DebugPlacement**,
	// which is the rule those two are already under — they answer different questions and are
	// wanted at different times, and "where is this drawn" is not "what movements does the game
	// have". Like both of them it is a *view*, set once in main.go, off by default, and it may
	// never change an outcome. See screens.AnimationsScene.
	DebugAnimations bool

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

	// InputFocus is the rectangles still accepting clicks, and InputGated is whether the
	// restriction is on at all. Together they are the tutorial's shield: while a step says "press
	// this", the cursor anywhere else clicks nothing.
	//
	// **A list rather than one rectangle** *(2026-09-08)*. It was one, and one is a bounding box:
	// an anchor naming a *set* of cards could only hand over the box around them, so anything
	// sitting between two of them was lit and clickable while not being part of the set. That is
	// exactly what happened to the tutorial's matching cards — the taught four sit at seats 0, 1, 2
	// and 4 under the default cost sort, and the arcane card at seat 3 was inside the square. The
	// player could queue it, which spends the budget the fourth taught card needed, and the lesson
	// then committed a hand it had just promised would be something else.
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
	InputFocus []image.Rectangle
	InputGated bool

	// LedgerOpens is how many times the run's account has been opened this session.
	//
	// **A counter and not a bool, so a step can watch for an opening of its own.** The tutorial
	// asks the player to open the ledger, and a flag set the first time it was ever opened would
	// be a step already satisfied before it was drawn — the same reason `tutorial.Run` measures
	// rounds against a baseline rather than against zero.
	//
	// **It lives here for the reason ModalOpen and InputGated do**: the button belongs to the
	// frame and the lesson belongs to a scene, so it is a thing the two have to agree on. It is
	// bumped by `internal/game` when the panel opens and read by whichever scene is publishing
	// tutorial facts — never cleared each tick, because unlike the two above it is a tally of
	// something that happened rather than an assertion about this frame.
	LedgerOpens int

	// HandSort is how a dealt hand is arranged, and which of the three sort tabs is latched.
	// See screens.handSort, whose ordinals this holds — cost is the zero value, so a state
	// nobody has touched is already sorted the way a fresh screen is.
	//
	// **It lives here because it is one preference over every screen that deals a hand**
	// *(owner's call, 2026-09-05)*, rather than one per screen. It was a field on CombatScene,
	// which was right while the combat screen was the only place a hand was laid out; the essence
	// screen deals one too, and a player who arranged their hand by element and then met a row
	// sorted by cost would be setting the same preference twice.
	//
	// **It is a reading preference, not a fact about a run**, so it is not saved to the profile
	// and no scene's Init resets it. It is a plain int for the reason Version and RunSeed are:
	// state stays free of imports, and screens holds the enum.
	//
	// Sorting a *queued* hand re-prices it — see combat_sort.go — which is a rule about the
	// combat screen rather than about this field.
	HandSort int

	//Layout
	//
	// **These are the live figures Layout published**, which is always the two constants
	// above. They are fields as well as constants because a test builds a GlobalState
	// without ever calling Layout, and because everything that measures the screen reads
	// them off the state it was handed rather than off a package it may not import.
	ScreenWidth  int
	ScreenHeight int

	//Data
	// Motifs is the whole roster, one entry per themed floor, each holding the creatures that
	// can stand in its three rooms. **A floor takes one motif and one element**; see
	// internal/pyramid.
	Motifs map[string]data.MotifData

	// Records is every motif's creatures flattened by key, because a screen hydrating an opponent
	// holds a record key and not the motif it came from. Built once beside Motifs rather than
	// searched, since it is read on every entry to the combat screen.
	Records map[string]data.MotifRecord

	// Tower is how tall the climb is and how fast it steepens. It is read wherever a stat is
	// grown to the fight it is met at, which is one place — entities.NewEnemyFrom.
	Tower data.TowerData

	// **The player and the opponents do not share a struct.** A creature has affinities, a
	// picture family and a tier; a duelist has a card back. See data/duelists_data.go.
	Duelists map[string]data.DuelistData

	// Relics is what the player can equip. **Genuinely global for the same reason the rosters
	// are** — it is loaded once from data/relics.json and no screen owns it. What is *equipped*
	// is not here: that is run state and belongs on Run, below — bought and sold in the shop.
	Relics map[string]data.RelicData

	// Run is what the player is carrying up the tower — the deck today, the worn relics and the
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

	// Summary is what the last run came to, kept alive across the moment the run itself is thrown
	// away. **The one piece of state that deliberately outlives the thing it describes**: the
	// run-over screen is drawn after Run has gone to nil, because a summary that held the Session
	// open would be a finished run that is still resumable.
	//
	// Nil except between a run ending and the player leaving that screen. See screens.endRun.
	Summary *session.RunSummary

	// Store is the directory the player's two files live in, and Profile is what has been read
	// out of the first of them. **Genuinely global for the reason Run is**: the profile outlives
	// every screen, is written from three different scenes, and no scene owns it.
	//
	// **Profile is never nil once main has run**, even on a machine that cannot save — an inert
	// store hands back a fresh profile rather than nothing, so a caller awards an achievement
	// without first asking whether the filesystem cooperated. See internal/profile.
	Store   profile.Store
	Profile *profile.Profile

	// EarnedThisSession is the queue of achievements landed but not yet shown, by key, oldest
	// first. **A queue rather than a flag**, because a single turn can earn three at once — a
	// five-element Prism is also an Elementalist and a Spectrum — and a toast that showed one and
	// dropped the rest would be the game quietly forgetting what the player just did.
	//
	// **It is here for the reason ModalOpen is**: it is written by `internal/screens`, wherever an
	// award happens, and drained by the frame in `internal/game`, which is what draws the toast.
	// Neither can reach the other, and the toast belongs to no scene — an achievement can land on
	// the combat screen, on the post-battle screen or on the way between them.
	//
	// **It holds keys, not records.** The catalog is a package away from either reader and a key
	// is what the profile stores, so nothing here has to learn what an achievement is.
	EarnedThisSession []string

	// Resumed is whether the run currently on Run came off the disk rather than being started
	// fresh. **Two readers**: the tutorial's trigger, because a lesson that opens by describing the
	// hand the player is holding cannot begin halfway up a tower; and the title screen's Continue,
	// which is exactly the question "is there a climb to go back to".
	Resumed bool

	// SeedPinned is whether RunSeed was chosen deliberately rather than rolled off the clock —
	// `main.fixedRunSeed`, or a scenario's own code.
	//
	// **It exists so that New Run does not silently break a pin.** A pinned seed is a debugging
	// session where the same tower in the same order is the whole point, and a menu button that
	// rerolled it would undo from the title screen what was set in the source. See screens.NewRun.
	//
	// **The tutorial outranks it**, as it always has: a taught run is dealt the script's own code,
	// because the lesson promises the player the hand they are holding and that is a fact about one
	// deal. So this says "do not reroll off the clock", not "this seed is final".
	SeedPinned bool

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

	// PendingGood is the sealed good the shop has just paid for, by record key, waiting for the
	// screen that opens it to pick it up.
	//
	// **A screen cannot be handed an argument**, so a scene that needs to know what it is showing
	// reads it off the state — the shape state.Summary already has for the run-over splash. It is
	// cleared by the screen that consumes it, so a second visit cannot open a good nobody bought.
	PendingGood string

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
	// is anything drawn *into* a card — the relic art, the enemy portraits — because
	// internal/cards renders to a plain Go image with no graphics context. Both maps read
	// the same embedded bytes, so the game and the contact sheet cannot end up showing
	// different pictures.
	ImageData map[string][]byte
}

// NewGlobalState used at the start of the game to start us off
func NewGlobalState() *GlobalState {
	return &GlobalState{
		// **Boots to the title screen again as of 2026-09-03** *(owner's call)*. It booted
		// straight into Combat for as long as that screen was the thing under construction and
		// clicking through a menu every launch was pure cost. What changed is that the menu now
		// *decides something*: whether this is a new climb or the one on disk, which is a question
		// nothing else in the game asks. A run that started before the player was asked is a run
		// they cannot decline. See screens/title.go and screens/run.go.
		ActiveScreen: Title,
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

	// Shop is the second between-fight scene: relics on a shelf, bought and sold with vitae.
	Shop

	Credits

	// Settings is the program's own screen: how loud the score is and how fast the game moves.
	//
	// **It is reachable from everywhere, so it is the one screen that has to remember where it
	// came from** — see ReturnScreen. Appended rather than filed next to Title because
	// ActiveScreen is append-only like every other ordinal in the game.
	Settings

	// RunOver is the splash a finished run ends on: what it came to, and the code that would deal
	// it again. **It is the one screen that draws something no longer in the game** — the run is
	// already gone by the time it is up, which is why what it draws is a session.RunSummary held on
	// the state rather than the run itself. Appended, because ActiveScreen is append-only.
	RunOver

	// Achievements is what the player has earned across every run they have ever played.
	//
	// **It is the program's screen rather than a run's**, like Settings and Credits: it never
	// touches session.Phase, and it reads the profile rather than the run — so it says the same
	// thing whether it is opened between climbs or halfway up one. Appended, because ActiveScreen
	// is append-only.
	Achievements

	// Animations is the gallery: every gesture the game can make, by name, on a loop. **A debug
	// page behind DebugAnimations**, reached from a square in the frame that is not drawn with the
	// flag off — the placement grid's arrangement. Not a station of a run, like the four above it,
	// and appended because ActiveScreen is append-only.
	Animations

	// Goods is a sealed good, opened: the three or four things that were inside it, and the one of
	// them the player takes.
	//
	// **It is a screen rather than a dialog** *(owner's call, 2026-09-19)*. What is on it is a
	// decision about the deck and the build — which stone, which rune, which essence and which card
	// it lands on — and a panel covering the screen to ask that hides the relics and the deck the
	// answer depends on. So the build band is up, the draw pile is up, and the deck panel opens over
	// it like anywhere else.
	//
	// **Not a station of a run**, like Settings and the two menu screens: it never touches
	// session.Phase, it is reached only from the shop and it goes back there. Appended, because
	// ActiveScreen is append-only.
	Goods
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
	case Achievements:
		return "Achievements"
	case RunOver:
		return "RunOver"
	case Animations:
		return "Animations"
	case Goods:
		return "Goods"
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
	for _, r := range gs.InputFocus {
		if at.In(r) {
			return true
		}
	}
	return false
}

// CursorAllowed is InputAllowed asked about wherever the cursor is right now, which is what
// almost every caller means.
func (gs *GlobalState) CursorAllowed() bool {
	return gs.InputAllowed(image.Pt(gs.MouseX, gs.MouseY))
}
