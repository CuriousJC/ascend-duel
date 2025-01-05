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
		return errors.New("Closing")
	}
	// Handling Mouse Position
	g.GlobalState.MouseX, g.GlobalState.MouseY = ebiten.CursorPosition()

	// Counters
	g.GlobalState.Count++
	if g.GlobalState.Count%60 == 0 {
		g.GlobalState.CountSecond++
	}

	switch g.GlobalState.ActiveScreen {
	case state.Title:
		screens.UpdateTitleScreen(g.GlobalState)
	case state.Ascend:
		//Ascend Screen
	case state.Combat:
		screens.UpdateCombatScreen(g.GlobalState)
	case state.Credits:
		//Credits
	default:
		screens.UpdateTitleScreen(g.GlobalState)
	}

	return nil

}

// Draw runs as needed to update the screen at each frame
func (g *Game) Draw(screen *ebiten.Image) {

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

// Layout is called when the window is resized and will resize the screen to the new size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	//Not playing any games with altering the size right now
	g.GlobalState.ScreenWidth = outsideWidth
	g.GlobalState.ScreenHeight = outsideHeight

	g.GlobalState.FirstThirdX = g.GlobalState.ScreenWidth / 3
	g.GlobalState.SecondThirdX = g.GlobalState.ScreenWidth / 3 * 2
	g.GlobalState.FirstThirdY = g.GlobalState.ScreenHeight / 3
	g.GlobalState.SecondThirdY = g.GlobalState.ScreenHeight / 3 * 2

	g.GlobalState.HalfwayX = g.GlobalState.ScreenWidth / 2
	g.GlobalState.HalfwayY = g.GlobalState.ScreenHeight / 2

	return g.GlobalState.ScreenWidth, g.GlobalState.ScreenHeight
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

}
