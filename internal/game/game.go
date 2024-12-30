package game

import (
	"fmt"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

	// Handling Mouse Position
	g.GlobalState.MouseX, g.GlobalState.MouseY = ebiten.CursorPosition()

	// Counters
	g.GlobalState.Count++
	if g.GlobalState.Count%60 == 0 {
		g.GlobalState.CountSecond++
	}

	// Handle updates to state from the Title Screen
	screens.UpdateTitleScreen(g.GlobalState)

	return nil

}

func (g *Game) Draw(screen *ebiten.Image) {
	screens.DrawTitleScreen(screen, g.GlobalState)

	if g.GlobalState.ActiveDebug {
		g.DrawDebugInfo(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	return outsideWidth, outsideHeight
}

func (g *Game) DrawDebugInfo(screen *ebiten.Image) {
	SecondTextOp := &text.DrawOptions{}
	SecondTextOp.GeoM.Translate(0, 900)
	SecondTextOp.LineSpacing = 30
	if g.GlobalState.CountSecond%2 == 0 {
		text.Draw(screen, "EVEN", &text.GoTextFace{Source: g.GlobalState.Fonts["firaSansRegular"], Size: 20}, SecondTextOp)
	} else {
		text.Draw(screen, "ODD", &text.GoTextFace{Source: g.GlobalState.Fonts["robotoFlexRegular"], Size: 20}, SecondTextOp)
	}

	UpdateCountOp := &text.DrawOptions{}
	UpdateCountOp.GeoM.Translate(150, 900)
	text.Draw(screen, strconv.Itoa(g.GlobalState.Count), &text.GoTextFace{Source: g.GlobalState.Fonts["firaSansRegular"], Size: 20}, UpdateCountOp)

	UpdateSecondOp := &text.DrawOptions{}
	UpdateSecondOp.GeoM.Translate(300, 900)
	text.Draw(screen, strconv.Itoa(g.GlobalState.CountSecond), &text.GoTextFace{Source: g.GlobalState.Fonts["robotoFlexRegular"], Size: 20}, UpdateSecondOp)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Mouse: %d, %d", g.GlobalState.MouseX, g.GlobalState.MouseY), 450, 900)
}
