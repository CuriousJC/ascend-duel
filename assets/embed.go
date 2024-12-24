// embed.go
package assets

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	_ "image/png"
	"log"
)

//go:embed spritesheet.png
var spritesheet_png []byte

//go:embed fire-ring.png
var firering_png []byte

//go:embed frozen-ring.png
var frozenring_png []byte

// Function to load embedded PNG images as ebiten.Images
func LoadImage(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal("failed to load image:", err)
	}
	return ebiten.NewImageFromImage(img)
}

func LoadAssets() map[string]*ebiten.Image {
	assets := make(map[string]*ebiten.Image)

	assets["spritesheet_png"] = LoadImage(spritesheet_png)
	assets["firering_png"] = LoadImage(firering_png)
	assets["frozenring_png"] = LoadImage(frozenring_png)

	return assets
}
