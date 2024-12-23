package game

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
	frameOX      = 510
	frameOY      = 640
	frameWidth   = 65
	frameHeight  = 65
	frameCount   = 8
)

var (
	elementImage *ebiten.Image
)

type Game struct {
	buttons []Button
	count   int
}

func NewGame() *Game {
	return &Game{
		buttons: []Button{
			NewButton(100, 50, 120, 40, "Play", PlayAction),
			NewButton(100, 100, 120, 40, "Settings", SettingsAction),
			NewButton(100, 150, 120, 40, "Exit", ExitAction),
		},
	}
}

func (g *Game) Update() error {
	for i := range g.buttons {
		g.buttons[i].Update()
	}
	g.count++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Step 1: Compute the frame
	i := (g.count / 50) % frameCount
	sx, sy := frameOX+i*frameWidth, frameOY

	// Step 2: Set up transformations
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(frameWidth)/2, -float64(frameHeight)/2) // Center the origin
	op.GeoM.Translate(screenWidth/2, screenHeight/2)                   // Move to screen center
	op.GeoM.Scale(0.5, 0.5)                                            // Scale down

	// Step 3: Render the image
	screen.DrawImage(elementImage.SubImage(image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)).(*ebiten.Image), op)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("frame sx: %d, sy: %d", sx, sy))

	// Step 4: Draw the buttons
	for i := range g.buttons {
		g.buttons[i].Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {

	// Decode an image from the image file's byte slice.
	//TODO: switch this to something embedded in our own source code
	img, _, err := image.Decode(bytes.NewReader(images.Spritesheet_png))
	if err != nil {
		log.Fatal(err)
	}
	elementImage = ebiten.NewImageFromImage(img)

	return 320, 240
}
