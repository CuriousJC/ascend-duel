package game

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/state"
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
}

func NewGame() *Game {
	return &Game{
		GlobalState: state.NewGlobalState(),
	}
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

	// Screen errors propagate rather than being discarded. Ebitengine stops the loop
	// on any non-nil error from Update, so a screen returning one is fatal by design —
	// which is the only sensible reading of an error a screen cannot handle itself.
	switch g.GlobalState.ActiveScreen {
	case state.Title:
		return screens.UpdateTitleScreen(g.GlobalState)
	case state.Ascend:
		return nil //Ascend Screen
	case state.Combat:
		return screens.UpdateCombatScreen(g.GlobalState)
	case state.Credits:
		return nil //Credits
	default:
		return screens.UpdateTitleScreen(g.GlobalState)
	}
}

// Draw runs as needed to update the screen at each frame
func (g *Game) Draw(screen *ebiten.Image) {

	// An action that switches screens sets ActiveScreen and NewScreen together, but the
	// new screen's Init does not run until the next Update. Draw would otherwise run
	// first and touch entities the Init has not built yet. Skipping that single frame
	// is cheaper than nil-guarding every Draw*Screen.
	if g.GlobalState.NewScreen {
		return
	}

	switch g.GlobalState.ActiveScreen {
	case state.Title:
		screens.DrawTitleScreen(g.GlobalState, screen)
	case state.Ascend:
		//Ascend Screen
	case state.Combat:
		screens.DrawCombatScreen(g.GlobalState, screen)
	case state.Credits:
		//Credits
	default:
		screens.DrawTitleScreen(g.GlobalState, screen)
	}

	// Debug Info will front-run everything and is drawn last on the screen
	if g.GlobalState.ActiveDebug {
		g.DrawDebugInfo(screen)
	}
}

// Layout reports the fixed internal resolution. The window size is ignored — Ebiten
// scales the result to the window and letterboxes the remainder.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	g.GlobalState.ScreenWidth = ScreenWidth
	g.GlobalState.ScreenHeight = ScreenHeight

	g.GlobalState.FirstThirdX = g.GlobalState.ScreenWidth / 3
	g.GlobalState.SecondThirdX = g.GlobalState.ScreenWidth / 3 * 2
	g.GlobalState.FirstThirdY = g.GlobalState.ScreenHeight / 3
	g.GlobalState.SecondThirdY = g.GlobalState.ScreenHeight / 3 * 2

	g.GlobalState.HalfwayX = g.GlobalState.ScreenWidth / 2
	g.GlobalState.HalfwayY = g.GlobalState.ScreenHeight / 2

	g.GlobalState.FirstQuarterX = g.GlobalState.ScreenWidth / 4
	g.GlobalState.ThirdQuarterX = g.GlobalState.ScreenWidth / 4 * 3
	g.GlobalState.FirstQuarterY = g.GlobalState.ScreenHeight / 4
	g.GlobalState.ThirdQuarterY = g.GlobalState.ScreenHeight / 4 * 3

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

	// Layout Lines
	vector.StrokeLine(screen, 0, float32(g.GlobalState.HalfwayY), 5000, float32(g.GlobalState.HalfwayY), 1, color.RGBA{R: 50, G: 205, B: 50, A: 255}, false)
	vector.StrokeLine(screen, float32(g.GlobalState.HalfwayX), 0, float32(g.GlobalState.HalfwayX), 5000, 3, color.RGBA{R: 50, G: 205, B: 50, A: 255}, false)

	vector.StrokeLine(screen, 0, float32(g.GlobalState.FirstThirdY), 5000, float32(g.GlobalState.FirstThirdY), 1, color.RGBA{R: 255, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, 0, float32(g.GlobalState.SecondThirdY), 5000, float32(g.GlobalState.SecondThirdY), 1, color.RGBA{R: 255, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, float32(g.GlobalState.FirstThirdX), 0, float32(g.GlobalState.FirstThirdX), 5000, 3, color.RGBA{R: 255, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, float32(g.GlobalState.SecondThirdX), 0, float32(g.GlobalState.SecondThirdX), 5000, 3, color.RGBA{R: 255, G: 105, B: 180, A: 75}, false)

	vector.StrokeLine(screen, 0, float32(g.GlobalState.FirstQuarterY), 5000, float32(g.GlobalState.FirstQuarterY), 1, color.RGBA{R: 50, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, 0, float32(g.GlobalState.ThirdQuarterY), 5000, float32(g.GlobalState.ThirdQuarterY), 1, color.RGBA{R: 50, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, float32(g.GlobalState.FirstQuarterX), 0, float32(g.GlobalState.FirstQuarterX), 5000, 3, color.RGBA{R: 50, G: 105, B: 180, A: 75}, false)
	vector.StrokeLine(screen, float32(g.GlobalState.ThirdQuarterX), 0, float32(g.GlobalState.ThirdQuarterX), 5000, 3, color.RGBA{R: 50, G: 105, B: 180, A: 75}, false)

}
