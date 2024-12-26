package game

import (
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
	frameOX      = 0
	frameOY      = 0
	frameWidth   = 300
	frameHeight  = 300
	frameCount   = 1
)

type Game struct {
	Buttons []Button
	count   int
	mouseX  int
	mouseY  int
	Assets  map[string]*ebiten.Image          // Store images as a map in the Game struct
	Fonts   map[string]*text.GoTextFaceSource //Store fonts as a map in the Game struct
}

func NewGame() *Game {
	return &Game{
		Assets:  make(map[string]*ebiten.Image),          // Initialize the assets map
		Fonts:   make(map[string]*text.GoTextFaceSource), // Initialize the fonts map
		Buttons: []Button{},                              // Initialize the buttons slice
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

	// Drawing our image that follows the mouse
	sx, sy := frameOX, frameOY
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2) // Center the origin
	op.GeoM.Translate(float64(g.mouseX), float64(g.mouseY))            // Follow the mouse
	//op.GeoM.Scale(0.5, 0.5)                                 // Scale down                                      // Scale down
	screen.DrawImage(g.Assets["firering_png"].SubImage(image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)).(*ebiten.Image), op)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Mouse: %d, %d", g.mouseX, g.mouseY), 25, 25)

	// Drawing our buttons
	for i := range g.Buttons {
		g.Buttons[i].Draw(screen)
	}

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	return screenWidth, screenHeight
}
