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

// Files are grouped into directories by what they are for — `game/`, `enemy/`, `ring/`,
// `effect/`, `sounds/` — and the //go:embed paths below are relative to this file, so a
// directory rename is a one-line edit per asset here and nothing anywhere else.
//
// **The map keys did not change with the move.** They are the lookup names used across the
// game and, for the enemies, written into `data/combatants.json` — so tying them to a
// filesystem path would mean a data migration every time a file was filed differently.
// The three edits a new asset needs are still: the file, an //go:embed var, and a map
// entry in the loader.

// IMAGES
//
//go:embed game/title.png
var title_png []byte

//go:embed game/title-easter-egg.png
var titleEaster_png []byte

// GLYPH ART
//
// The attack and defend category glyphs on an action card. These two are the exception to
// "interface art is generated": everything else in internal/systems/glyphs.go is a
// silhouette described in code, and these are hand-drawn pixel art.
//
// **The provenance question the generated glyphs exist to avoid does not apply here.**
// Drawn by KingSherman1820, one of the two copyright holders, for this game — so there is
// nothing to clear and nothing to attribute to a third party.
//
// Authored on a 64x64 canvas and drawn on the card at half that; see glyphArt in
// internal/systems/glyphs.go for why the halving happens at render time rather than here.
//
//go:embed game/sherman-sword.png
var shermansword_png []byte

//go:embed game/sherman-shield.png
var shermanshield_png []byte

// CREATURES
//
// One west-facing idle frame per enemy, cut out of the PVGames creature sheets in
// `.scratch/flat-creatures` and trimmed to its opaque bounds.
//
// **Frames, not sheets, and the difference is not small.** Those sheets are a 21x21 grid
// of every animation in every facing, 2.4 to 7.3 MB each — and LoadAssets decodes every
// image at startup, so embedding four of them would cost ~250 MB of resident memory to
// show four static sprites. The frames below are 41 KB together. The full sheets stay in
// `.scratch/`, so animating these later is a matter of going back for them.
//
// Provenance: PVGames, bought in the Humble *Isometric Assets Galore* bundle. The licence
// permits shipping them inside a game; see the README in that folder.
//
// The frame is 133 of 441 — idle, facing west — from `128 + 1*5`, the vendor's facing
// order being South, West, East, North, ...
//
//go:embed enemy/giantrat.png
var giantrat_png []byte

//go:embed enemy/dragonfly.png
var dragonfly_png []byte

//go:embed enemy/frogman.png
var frogman_png []byte

//go:embed enemy/direwolf.png
var direwolf_png []byte

//go:embed ring/fire-ring.png
var firering_png []byte

//go:embed ring/frozen-ring.png
var frozenring_png []byte

//go:embed ring/thunder-ring.png
var thunderring_png []byte

//go:embed effect/fire-effect.png
var fireeffect_png []byte

//go:embed effect/frozen-effect.png
var frozeneffect_png []byte

//go:embed effect/thunder-effect.png
var thundereffect_png []byte

// MUSIC
//
// Scores are Standard MIDI Files, not recorded audio: internal/music synthesises them
// at startup. That is why a whole track is a kilobyte and why there is no soundfont
// here whose licence would have to be cleared before the game could be sold.
//
//go:embed sounds/ascending.mid
var ascending_mid []byte

// FONTS
//
//go:embed game/FiraSans-Regular.ttf
var firaSansRegular []byte

//go:embed game/RobotoFlex.ttf
var robotoFlexRegular []byte

//go:embed game/Kubasta.ttf
var kubasta []byte

// LoadAssets returns a mapped set of images for the game
func LoadAssets() map[string]*ebiten.Image {
	assets := make(map[string]*ebiten.Image)

	assets["title_png"] = loadImage(title_png)
	assets["titleEaster_png"] = loadImage(titleEaster_png)
	assets["firering_png"] = loadImage(firering_png)
	assets["frozenring_png"] = loadImage(frozenring_png)
	assets["thunderring_png"] = loadImage(thunderring_png)
	assets["fireeffect_png"] = loadImage(fireeffect_png)
	assets["frozeneffect_png"] = loadImage(frozeneffect_png)
	assets["thundereffect_png"] = loadImage(thundereffect_png)
	assets["giantrat_png"] = loadImage(giantrat_png)
	assets["dragonfly_png"] = loadImage(dragonfly_png)
	assets["frogman_png"] = loadImage(frogman_png)
	assets["direwolf_png"] = loadImage(direwolf_png)

	return assets
}

// LoadMusic returns the raw bytes of each embedded score, keyed the same way as the
// image and font maps. They are handed back undecoded because the decoder lives in
// internal/music, and assets sits below it — nothing here may reach up.
func LoadMusic() map[string][]byte {
	music := make(map[string][]byte)

	music["ascending_mid"] = ascending_mid

	return music
}

// LoadFonts returns a mapped set of fonts for the game
func LoadFonts() map[string]*text.GoTextFaceSource {
	fonts := make(map[string]*text.GoTextFaceSource)

	fonts["firaSansRegular"] = loadFont(firaSansRegular)
	fonts["robotoFlexRegular"] = loadFont(robotoFlexRegular)
	fonts["kubasta"] = loadFont(kubasta)

	return fonts

}

// LoadImageData returns the raw bytes of embedded images, keyed like the other maps.
//
// LoadAssets above decodes into *ebiten.Image, which needs a graphics context and so
// cannot be called from a command-line tool. tools/cardsheet renders card artwork with
// no window, so it takes the file and decodes it with image/png itself.
//
// Only the images something actually needs this way are listed. Adding one here does not
// change what LoadAssets does — both read the same embedded bytes.
func LoadImageData() map[string][]byte {
	images := make(map[string][]byte)

	images["firering_png"] = firering_png
	images["frozenring_png"] = frozenring_png
	images["thunderring_png"] = thunderring_png

	// The glyph art. internal/systems takes the bytes rather than an *ebiten.Image for the
	// same reason the ring art does: RenderGlyph draws into a plain Go image so the contact
	// sheets can be built with no window.
	images["shermansword_png"] = shermansword_png
	images["shermanshield_png"] = shermanshield_png

	return images
}

// LoadFontData returns the raw bytes of each embedded font, keyed like the other maps.
//
// LoadFonts above hands back Ebitengine's GoTextFaceSource, which is what the screens
// draw with. internal/cards cannot use those: it renders to a plain Go image with no
// graphics context, so it sets text through golang.org/x/image and needs the file
// itself. Both read the same embedded bytes, so the game and the contact sheet cannot
// end up in different fonts.
func LoadFontData() map[string][]byte {
	fonts := make(map[string][]byte)

	fonts["firaSansRegular"] = firaSansRegular
	fonts["robotoFlexRegular"] = robotoFlexRegular
	fonts["kubasta"] = kubasta

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
