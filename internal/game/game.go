package game

//TODO:  Play with fonts that I've embedded (https://github.com/hajimehoshi/ebiten/blob/main/examples/fontfeature/main.go#L107)
//TODO: Maybe use the image.fill function instead of a gradient? (https://ebitengine.org/en/tour/fill.html)
//TODO:  Use a one byte vertical gradient for the buttons
//TODO: Make the buttons do stuff on the screen instead of changing hues

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 1280
	screenHeight = 960
)

type Game struct {
	Buttons             []Button
	count               int
	countSecond         int
	mouseX              int
	mouseY              int
	buttonHue128        int
	buttonSaturation128 int
	buttonValue128      int
	isDebug             bool
	Assets              map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts               map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct
}

func NewGame() *Game {
	return &Game{
		buttonHue128:        0,
		buttonSaturation128: 0,
		buttonValue128:      0,
		isDebug:             true,
		Assets:              make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:               make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
		Buttons:             []Button{},                              // Initialize the buttons slice
	}
}

func (g *Game) Update() error {
	for i := range g.Buttons {
		g.Buttons[i].Update()
	}
	g.count++
	g.mouseX, g.mouseY = ebiten.CursorPosition()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	// Drawing our buttons
	for i := range g.Buttons {
		g.Buttons[i].Draw(g, screen)
	}

	// Drawing our text with a particular font
	if g.count%60 == 0 {
		g.countSecond++
	}

	if g.isDebug {
		g.DrawDebugInfo(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	return screenWidth, screenHeight
}

func (g *Game) DrawDebugInfo(screen *ebiten.Image) {
	SecondTextOp := &text.DrawOptions{}
	SecondTextOp.GeoM.Translate(0, 900)
	SecondTextOp.LineSpacing = 30
	if g.countSecond%2 == 0 {
		text.Draw(screen, "EVEN", &text.GoTextFace{Source: g.Fonts["firaSansRegular"], Size: 20}, SecondTextOp)
	} else {
		text.Draw(screen, "ODD", &text.GoTextFace{Source: g.Fonts["robotoFlexRegular"], Size: 20}, SecondTextOp)
	}

	UpdateCountOp := &text.DrawOptions{}
	UpdateCountOp.GeoM.Translate(150, 900)
	text.Draw(screen, strconv.Itoa(g.count), &text.GoTextFace{Source: g.Fonts["firaSansRegular"], Size: 20}, UpdateCountOp)

	UpdateSecondOp := &text.DrawOptions{}
	UpdateSecondOp.GeoM.Translate(300, 900)
	text.Draw(screen, strconv.Itoa(g.countSecond), &text.GoTextFace{Source: g.Fonts["robotoFlexRegular"], Size: 20}, UpdateSecondOp)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Mouse: %d, %d", g.mouseX, g.mouseY), 450, 900)
}
