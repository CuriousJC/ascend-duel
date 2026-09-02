package game

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/idle"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ErrClosing is returned from Update to stop the game loop on a deliberate quit.
// Ebitengine treats any non-nil error from Update as "stop", so a clean exit has to
// travel as an error; main checks for this one specifically so quitting does not
// look like a crash.
var ErrClosing = errors.New("closing")

// ScreenWidth and ScreenHeight are the fixed internal resolution. Layout always
// reports these regardless of window size, so Ebiten scales and letterboxes to fit
// and every absolute coordinate in the game is safe against resizing.
const (
	ScreenWidth  = 1280
	ScreenHeight = 960
)

type Game struct {
	GlobalState *state.GlobalState

	// One registry, replacing the two parallel switches Update and Draw used to
	// carry. Those could drift out of sync — a screen added to one and forgotten in
	// the other silently did nothing — and now cannot.
	scenes map[state.ActiveScreen]screens.Scene

	// settingsButton is the game's chrome, and the only widget not owned by a scene.
	//
	// **CLAUDE.md's rule is that scenes build their own widgets, and this is deliberately
	// outside it rather than an exception to it.** The score is started once in main and
	// loops across every screen for the whole session, and the game's one clock is the same
	// number on every screen, so the control that opens both belongs at the same level. The alternative was the same button on four scenes, four
	// placements to keep in step and four callbacks into one package — which is a worse
	// answer to "who owns this" than admitting the game has a frame.
	//
	// Built lazily in Update, because it needs nothing from a scene and nothing from a
	// window, and NewGame runs before assets and fonts are loaded.
	settingsButton *models.Button

	// ledgerButton opens the run's account, and ledger is the panel it opens. **Both are chrome
	// for the same reason the cog is** — true for the whole run, wanted on every screen, owned by
	// no scene — and the panel could not have been a screen at all: navigating away from the
	// combat screen and back re-runs its Init, which deals a fresh duel. See chrome.go.
	ledgerButton *models.Button
	ledger       screens.LedgerPanel
}

func NewGame() *Game {
	return &Game{
		GlobalState: state.NewGlobalState(),
		scenes: map[state.ActiveScreen]screens.Scene{
			state.Title:      &screens.TitleScene{},
			state.Ascend:     &screens.AscendScene{},
			state.Combat:     &screens.CombatScene{},
			state.PostBattle: &screens.PostBattleScene{},
			state.Shop:       &screens.ShopScene{},
			state.Credits:    &screens.CreditsScene{},
			state.Settings:   &screens.SettingsScene{},
		},
	}
}

// scene returns the active scene, falling back to the title screen if ActiveScreen
// somehow names one that was never registered.
func (g *Game) scene() screens.Scene {
	if s, ok := g.scenes[g.GlobalState.ActiveScreen]; ok {
		return s
	}
	return g.scenes[state.Title]
}

func (g *Game) Update() error {

	if g.GlobalState.ShouldClose || ebiten.IsWindowBeingClosed() {
		//Would handle any saving of state or confirmation here
		return ErrClosing
	}
	// Handling Mouse Position
	g.GlobalState.MouseX, g.GlobalState.MouseY = ebiten.CursorPosition()

	// Counters
	g.GlobalState.Count++
	if g.GlobalState.Count%60 == 0 {
		g.GlobalState.CountSecond++
	}

	// The tick every trace line is stamped with. Set once here rather than passed to each
	// call, and it is the simulation counter rather than a clock, so a trace lines up with
	// a replay of the same seed.
	trace.Tick(g.GlobalState.Count)

	// Close an unattended window that nobody is using. Compiled out entirely unless the
	// idleexit tag is set, so this is a no-op returning false in any build that ships.
	//
	// Focus is passed in rather than assumed, because the case this exists for is a window
	// sitting in the background — see internal/idle. It sets ShouldClose rather than
	// returning ErrClosing directly, so the exit runs through exactly the same path as the
	// window's close button and there is only one way the game ends.
	if idle.Tick(
		g.GlobalState.MouseX, g.GlobalState.MouseY,
		ebiten.IsFocused(),
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
	) {
		g.GlobalState.ShouldClose = true
	}

	scene := g.scene()

	// **The ledger owns the frame while it is up, and the scene is not updated at all.** It is a
	// panel over every screen rather than one screen's dialog, so there is no scene to set
	// state.ModalOpen and no scene that could be trusted to go inert for it. What that costs is
	// pacing — a duel's playback stops while the account is open — which is the same thing every
	// dialog on the combat screen already does and, like all of them, cannot change an outcome:
	// ResolveRound decided the round before any of it was drawn.
	if g.ledger.IsOpen() {
		g.GlobalState.ModalOpen = false
		g.GlobalState.InputGated = false
		g.ledger.Update(g.GlobalState)
		return nil
	}

	// One-shot init on entering a screen. Doing it here rather than inside each
	// scene's Update means no scene has to remember the NewScreen dance.
	if g.GlobalState.NewScreen {
		scene.Init(g.GlobalState)
		g.GlobalState.NewScreen = false
	}

	// Cleared here and re-asserted by whichever scene actually has a dialog up, so a screen
	// left with its overlay open cannot leave the chrome hidden for the rest of the session.
	// See state.ModalOpen.
	g.GlobalState.ModalOpen = false

	// **The tutorial's input shield, cleared on the same terms and for the same reason.** A
	// screen left mid-step — a scene change, a skipped tutorial — must not leave the rest of the
	// session shielded around a rectangle nothing is drawing. Whoever still wants it re-asserts
	// it below. See state.InputGated.
	g.GlobalState.InputGated = false

	// Scene errors propagate rather than being discarded. Ebitengine stops the loop
	// on any non-nil error from Update, so a scene returning one is fatal by design —
	// which is the only sensible reading of an error a scene cannot handle itself.
	if err := scene.Update(g.GlobalState); err != nil {
		return err
	}

	// The frame's own controls, after the scene, so they read the modal flag the scene has
	// just written. See chrome.go.
	g.updateChrome(g.GlobalState)
	return nil
}

// Draw runs as needed to update the screen at each frame
func (g *Game) Draw(screen *ebiten.Image) {

	// An action that switches screens sets ActiveScreen and NewScreen together, but the
	// incoming scene's Init does not run until the next Update. Draw would otherwise
	// run first and touch state the Init has not built yet. Skipping that single frame
	// is cheaper than nil-guarding every scene's Draw.
	if g.GlobalState.NewScreen {
		return
	}

	g.scene().Draw(g.GlobalState, screen)

	// The frame, over the screen it frames. See chrome.go for why the mute button lives here
	// rather than on four scenes, and why it stands down over a dialog.
	g.drawChrome(g.GlobalState, screen)

	// **Over the chrome as well as the screen**, because it covers both: the cog stands down under
	// it the way it does under any dialog, and the panel's own X is the way out.
	g.ledger.Draw(g.GlobalState, screen)

	// Debug Info will front-run everything and is drawn last on the screen
	if g.GlobalState.DebugPlacement {
		g.DrawDebugInfo(screen)
	}

	// Last of all, so a capture holds exactly what was on screen, debug overlay included.
	trace.Frame(screen)
}

// Layout reports the fixed internal resolution. The window size is ignored — Ebiten
// scales the result to the window and letterboxes the remainder.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	g.GlobalState.ScreenWidth = ScreenWidth
	g.GlobalState.ScreenHeight = ScreenHeight

	return ScreenWidth, ScreenHeight
}

// DrawDebugInfo is the final drawing and will place information on the screen at the specified row if requested
func (g *Game) DrawDebugInfo(screen *ebiten.Image) {
	debugYRow := 900
	SecondTextOp := &text.DrawOptions{}
	SecondTextOp.GeoM.Translate(0, float64(debugYRow))
	SecondTextOp.LineSpacing = 30
	if g.GlobalState.CountSecond%2 == 0 {
		text.Draw(screen, "EVEN", &text.GoTextFace{Source: g.GlobalState.Fonts["firaSansRegular"], Size: 20}, SecondTextOp)
	} else {
		text.Draw(screen, "ODD", &text.GoTextFace{Source: g.GlobalState.Fonts["robotoFlexRegular"], Size: 20}, SecondTextOp)
	}

	UpdateCountOp := &text.DrawOptions{}
	UpdateCountOp.GeoM.Translate(150, float64(debugYRow))
	text.Draw(screen, strconv.Itoa(g.GlobalState.Count), &text.GoTextFace{Source: g.GlobalState.Fonts["firaSansRegular"], Size: 20}, UpdateCountOp)

	UpdateSecondOp := &text.DrawOptions{}
	UpdateSecondOp.GeoM.Translate(300, float64(debugYRow))
	text.Draw(screen, strconv.Itoa(g.GlobalState.CountSecond), &text.GoTextFace{Source: g.GlobalState.Fonts["robotoFlexRegular"], Size: 20}, UpdateSecondOp)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Mouse: %d, %d", g.GlobalState.MouseX, g.GlobalState.MouseY), 450, debugYRow)

	//Global State Debug Messages
	debugState := fmt.Sprintf("Debug1: %s \nDebug2: %s", g.GlobalState.Debug1, g.GlobalState.Debug2)
	ebitenutil.DebugPrintAt(screen, debugState, 450, debugYRow+30)

	g.drawLayoutGuides(screen)
	g.drawPercentTicks(screen)
}

// drawLayoutGuides rules the screen at the halves, thirds and quarters. These were the
// only positions the old cached layout fields could express; they survive as guides
// because they are still the positions things get placed at most often.
//
// Thirds are exact division rather than PctY(33) — 33% of 960 is four pixels off the
// real third, which is visible when eyeballing a sprite against the line.
func (g *Game) drawLayoutGuides(screen *ebiten.Image) {
	w := float32(g.GlobalState.ScreenWidth)
	h := float32(g.GlobalState.ScreenHeight)

	guideX := func(x float32, c color.Color) { vector.StrokeLine(screen, x, 0, x, h, 3, c, false) }
	guideY := func(y float32, c color.Color) { vector.StrokeLine(screen, 0, y, w, y, 1, c, false) }

	halves := color.RGBA{R: 50, G: 205, B: 50, A: 255}
	guideY(h/2, halves)
	guideX(w/2, halves)

	thirds := color.RGBA{R: 255, G: 105, B: 180, A: 75}
	guideY(h/3, thirds)
	guideY(h/3*2, thirds)
	guideX(w/3, thirds)
	guideX(w/3*2, thirds)

	quarters := color.RGBA{R: 50, G: 105, B: 180, A: 75}
	guideY(h/4, quarters)
	guideY(h/4*3, quarters)
	guideX(w/4, quarters)
	guideX(w/4*3, quarters)
}

// drawPercentTicks marks every 10% along the top and left edges, so a position can be
// read off the screen while laying things out. Short ticks rather than full-width
// rules, so they locate without obscuring what is being positioned.
func (g *Game) drawPercentTicks(screen *ebiten.Image) {
	const (
		step      = 10 // percent between ticks
		tickScale = 10 // tick length as a percent of the perpendicular dimension
	)

	tickColor := color.RGBA{R: 220, G: 220, B: 220, A: 140}
	w := float32(g.GlobalState.ScreenWidth)
	h := float32(g.GlobalState.ScreenHeight)
	tickDown := h * tickScale / 100   // length of the ticks hanging off the top edge
	tickAcross := w * tickScale / 100 // length of the ticks running off the left edge

	for pct := step; pct < 100; pct += step {
		// Along the top edge: vertical ticks marking X positions.
		x := w * float32(pct) / 100
		vector.StrokeLine(screen, x, 0, x, tickDown, 1, tickColor, false)
		ebitenutil.DebugPrintAt(screen, strconv.Itoa(pct), int(x)+2, 2)

		// Down the left edge: horizontal ticks marking Y positions.
		y := h * float32(pct) / 100
		vector.StrokeLine(screen, 0, y, tickAcross, y, 1, tickColor, false)
		ebitenutil.DebugPrintAt(screen, strconv.Itoa(pct), 2, int(y)+2)
	}
}
