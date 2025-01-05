// embed.go
package assets

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// IMAGES
//
//go:embed title.png
var title_png []byte

//go:embed title-easter-egg.png
var titleEaster_png []byte

//go:embed spritesheet.png
var spritesheet_png []byte

//go:embed fire-ring.png
var firering_png []byte

//go:embed frozen-ring.png
var frozenring_png []byte

// FONTS
//
//go:embed FiraSans-Regular.ttf
var firaSansRegular []byte

//go:embed RobotoFlex.ttf
var robotoFlexRegular []byte

//go:embed Kubasta.ttf
var kubasta []byte

// LoadAssets returns a mapped set of images for the game
func LoadAssets() map[string]*ebiten.Image {
	assets := make(map[string]*ebiten.Image)

	assets["title_png"] = loadImage(title_png)
	assets["titleEaster_png"] = loadImage(titleEaster_png)
	assets["spritesheet_png"] = loadImage(spritesheet_png)
	assets["firering_png"] = loadImage(firering_png)
	assets["frozenring_png"] = loadImage(frozenring_png)

	return assets
}

// LoadFonts returns a mapped set of fonts for the game
func LoadFonts() map[string]*text.GoTextFaceSource {
	fonts := make(map[string]*text.GoTextFaceSource)

	fonts["firaSansRegular"] = loadFont(firaSansRegular)
	fonts["robotoFlexRegular"] = loadFont(robotoFlexRegular)
	fonts["kubasta"] = loadFont(kubasta)

	return fonts

}

// loadFont Function flip embedded font into GoTextFaceSource
func loadFont(data []byte) *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	return s
}

// loadImage Function flip embedded image into ebiten Image
func loadImage(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal("failed to load image:", err)
	}
	return ebiten.NewImageFromImage(img)
}
