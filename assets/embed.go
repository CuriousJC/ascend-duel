package assets

// embed.go

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"log"
	"strings"

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
// One portrait per enemy: the vendor's 2048x2048 facing portrait, cropped to its subject and
// scaled to fit the enemy card's art box. 96 of them, 2.1 MB together.
//
// **The west-facing idle sprite frames went on 2026-08-11.** The enemy is drawn as a card
// now, so nothing used them — and cutting 96 more frames for a drawing that does not exist
// would have been the expensive half of this change. They are in git, and the full animation
// sheets are still in `.scratch/flat-creatures` if enemies are ever animated.
//
// **Embedded as a directory rather than one var each, which is a deliberate exception to the
// three-edit rule** at the top of this file. That rule — the file, an //go:embed var, a map
// entry — is right for a handful of named assets and absurd for ninety-six: it would be 192
// lines that no reviewer could check and that would drift the first time a creature was
// renamed. So the portraits are a *family*, globbed in and keyed by filename stem.
//
// The consequence, stated because it is the thing the rule was protecting: **a portrait's
// key is now tied to its filename**, so renaming `ogrewarlord-portrait.png` renames its key
// and `data/enemies.json` has to follow. That is the price of not hand-maintaining 96
// entries, and it is checked — an enemy whose Portrait names no file draws a card with a
// hole in it and logs once.
//
// Provenance: PVGames, bought in the Humble *Isometric Assets Galore* bundle. The licence
// permits shipping them inside a game; see the README in that folder.
//
//go:embed enemy/*-portrait.png
var portraits embed.FS

//go:embed ring/fire-ring.png
var firering_png []byte

//go:embed ring/ice-ring.png
var icering_png []byte

//go:embed ring/lightning-ring.png
var lightningring_png []byte

//go:embed ring/needle-ring.png
var needlering_png []byte

//go:embed ring/earth-ring.png
var earthring_png []byte

// The face a ring with no artwork of its own falls back to. Most of `data/rings.json` is
// rings written since the four elemental ones were drawn — a form multiplier, the two vitae
// rings, the growing stat rings — and none of them has a picture yet. Without this they draw
// as a pink border around an empty face, which reads as a card that failed to load rather
// than as one waiting for art. Same choice as `default-effect.png`, one layer up.
//
//go:embed ring/default-ring.png
var defaultring_png []byte

// The face every worm draws, until worms have art of their own. **A copy of the ring's default
// rather than a share of it** *(owner's call, 2026-08-22)*: two files that happen to look alike
// today are two files that can be replaced one at a time, where one file used by both would have
// to be forked the moment either gets a real picture.
//
//go:embed worm/default-worm.png
var defaultworm_png []byte

//go:embed effect/fire-effect.png
var fireeffect_png []byte

//go:embed effect/frozen-effect.png
var frozeneffect_png []byte

//go:embed effect/thunder-effect.png
var thundereffect_png []byte

//go:embed effect/earth-effect.png
var eartheffect_png []byte

// The badge an element with no artwork of its own falls back to, so a status always shows
// *something* rather than nothing — a status that is on and invisible is worse than one drawn
// as a shape you have not learned yet.
//
//go:embed effect/default-effect.png
var defaulteffect_png []byte

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
	assets["icering_png"] = loadImage(icering_png)
	assets["lightningring_png"] = loadImage(lightningring_png)
	assets["needlering_png"] = loadImage(needlering_png)
	assets["earthring_png"] = loadImage(earthring_png)
	assets["defaultring_png"] = loadImage(defaultring_png)
	assets["defaultworm_png"] = loadImage(defaultworm_png)
	assets["fireeffect_png"] = loadImage(fireeffect_png)
	assets["frozeneffect_png"] = loadImage(frozeneffect_png)
	assets["thundereffect_png"] = loadImage(thundereffect_png)
	assets["eartheffect_png"] = loadImage(eartheffect_png)
	assets["defaulteffect_png"] = loadImage(defaulteffect_png)
	// The enemy portraits are deliberately absent. They are drawn *into* a card by
	// internal/cards, which has no graphics context, so they are handed out as bytes by
	// LoadImageData instead — and decoding 96 of them here at startup would cost about
	// 20 MB of resident memory for pictures most of which no run ever shows.
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
	images["icering_png"] = icering_png
	images["lightningring_png"] = lightningring_png
	images["needlering_png"] = needlering_png
	images["earthring_png"] = earthring_png
	images["defaultring_png"] = defaultring_png
	images["defaultworm_png"] = defaultworm_png

	// The status badges, for the same reason as the ring art: they are drawn *into* the enemy
	// card by internal/cards, which has no graphics context.
	images["fireeffect_png"] = fireeffect_png
	images["frozeneffect_png"] = frozeneffect_png
	images["thundereffect_png"] = thundereffect_png
	images["eartheffect_png"] = eartheffect_png
	images["defaulteffect_png"] = defaulteffect_png

	// The glyph art. internal/systems takes the bytes rather than an *ebiten.Image for the
	// same reason the ring art does: RenderGlyph draws into a plain Go image so the contact
	// sheets can be built with no window.
	images["shermansword_png"] = shermansword_png
	images["shermanshield_png"] = shermanshield_png

	// The enemy portraits, keyed by filename stem: `enemy/ogrewarlord-portrait.png` is
	// `ogrewarlord-portrait`, which is what `data/enemies.json` writes in its Portrait
	// field. Read out of the embedded directory rather than listed, for the reason above the
	// //go:embed.
	//
	// A read failure here is impossible in a built binary — the files are compiled in — so a
	// panic is the honest response to one rather than a silent short roster.
	entries, err := portraits.ReadDir("enemy")
	if err != nil {
		log.Fatal("failed to read the embedded portraits: ", err)
	}
	for _, e := range entries {
		raw, err := portraits.ReadFile("enemy/" + e.Name())
		if err != nil {
			log.Fatal("failed to read embedded portrait: ", err)
		}
		images[strings.TrimSuffix(e.Name(), ".png")] = raw
	}

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
