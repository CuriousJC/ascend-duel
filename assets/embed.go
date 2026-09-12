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

// Files are grouped into directories by what they are for — `game/`, `enemy/`, `relic/`,
// `effect/`, `upgrade/`, `sounds/` — and the //go:embed paths below are relative to this file, so a
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

// THE GUIDE
//
// Bob, the guide: the face on the tutorial's speech bubble. **A named one-off rather than a
// member of either portrait family**, because he is not an opponent — nothing places him on a
// floor, nothing fights him, and filing him under `boss/` would put him in the glob that
// `data/bosses.json` is read against, where a portrait with no record behind it is a record
// somebody has lost.
//
// 256x256, the same canvas the boss portraits are cut to, so he sits in a card's art box on the
// same terms as everything else drawn into one.
//
// Provenance: the same set the thirty boss portraits came from.
//
//go:embed game/guide.png
var guide_png []byte

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

// FORM MARKS
//
// The four form marks an action card carries in its corner — stab, slash, crush, defend.
// Drawn as pixel art rather than described in code, for the reason the two Sherman glyphs
// above are: a spear with a socket and a shoulder is a drawing, and the span language in
// internal/systems/glyphs.go is a poor way to write one.
//
// **Authored at 32 and drawn at 32.** Unlike the Sherman pair these are not halved, because
// their outline is one pixel: averaging a 2x2 block that is half rim and half surface turns a
// black edge into a grey one, and at this size that edge is the whole of what holds the shape
// against an off-white card. See glyphArtwork.canvas in internal/systems/glyphs.go.
//
// Provenance: cut down by the owner from a spritesheet authored for this game, so there is
// nothing to clear and nothing to attribute to a third party.
//
//go:embed form/stab.png
var formstab_png []byte

//go:embed form/slash.png
var formslash_png []byte

//go:embed form/crush.png
var formcrush_png []byte

//go:embed form/defend.png
var formdefend_png []byte

// UPGRADE ART
//
// The colour a *visible upgrade* paints a card's left column from. A card the run has altered
// used to look exactly like one it had not — sixteen parasites attach seven kinds of rider and
// the only place any of them was visible was the tooltip prose — and this is the first of them
// that says so on the face.
//
// **It is a colour source, not a picture.** Nothing blits this file: `internal/systems` decodes
// it and `internal/cards` samples it to paint the form mark and the cost ticks, exactly where the
// element's own colour would otherwise go. So it is authored at the form marks' 32x32, the box it
// has to cover, rather than at the 64 the drawn glyphs use.
//
// **Its own group rather than `form/`**, because more visible upgrades are coming and the next
// one will have nothing to do with the form mark.

//go:embed upgrade/wildcard.png
var wildcardupgrade_png []byte

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

// The thirty boss portraits, globbed for the same reason and keyed the same way — the stem,
// so `boss/bayaz-boss.png` is `bayaz-boss`, which is what `data/bosses.json` writes in its
// Portrait field. **The `-boss` suffix is what keeps the two families out of each other's key
// space**: the map is flat, and a boss called `Sentry` beside a creature portrait of the same
// name would otherwise be one lookup with two answers.
//
// Their provenance is the same PVGames bundle as the creature portraits.
//
//go:embed boss/*-boss.png
var bossPortraits embed.FS

// The relic faces, globbed as a family and keyed by filename stem — `relic/fire.png` is
// `fire`, which is what `data/relics.json` writes in its Art field.
//
// **It became a family on 2026-09-11**, having been one `//go:embed` var per file. Five pictures
// is three edits each and readable; a catalogue of a hundred and thirty-seven relics being drawn
// is not, and the var names were the key, so every one of them was also a line in two loaders.
// The cost is the documented one — a relic's key is now tied to its filename, so renaming a file
// means editing `relics.json`.
//
// `relic/default-relic.png` is in here like any other and is what a relic with no Art of its own
// falls back to: most of `data/relics.json` has no picture yet, and a pink border around an empty
// face reads as a card that failed to load rather than as one waiting for art.
//
//go:embed relic/*.png
var relicArt embed.FS

// The worm faces, the same way and for the same reason. `worm/default-worm.png` is what every
// worm and every parasite draws until they have art of their own — **a copy of the relic's
// default rather than a share of it** *(owner's call, 2026-08-22)*: two files that happen to
// look alike today are two files that can be replaced one at a time.
//
//go:embed worm/*.png
var wormArt embed.FS

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
	assets["fireeffect_png"] = loadImage(fireeffect_png)
	assets["frozeneffect_png"] = loadImage(frozeneffect_png)
	assets["thundereffect_png"] = loadImage(thundereffect_png)
	assets["eartheffect_png"] = loadImage(eartheffect_png)
	assets["defaulteffect_png"] = loadImage(defaulteffect_png)
	// The enemy portraits are deliberately absent. They are drawn *into* a card by
	// internal/cards, which has no graphics context, so they are handed out as bytes by
	// LoadImageData instead — and decoding 96 of them here at startup would cost about
	// 20 MB of resident memory for pictures most of which no run ever shows.
	//
	// **The relic and worm art joined them on 2026-09-11**, having been decoded here as well as
	// handed over as bytes. Nothing ever read the decoded copy — every caller goes through
	// `screens.artwork`, which decodes out of `ImageData` and caches — and full-bleed art is
	// authored at 1060x1484, which is 6 MB of RGBA each. Fifteen of those is ninety megabytes
	// nothing looks at.
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

	// The four families read out of an embedded directory rather than listed one by one. See
	// embedFamily, and the //go:embed lines above for what each key ends up being.
	embedFamily(images, relicArt, "relic")
	embedFamily(images, wormArt, "worm")
	embedFamily(images, portraits, "enemy")
	embedFamily(images, bossPortraits, "boss")

	// Bob's face, for the reason the relic art is here: the tutorial draws him into a card
	// through internal/cards, which has no graphics context.
	images["guide_png"] = guide_png

	// The status badges, for the same reason as the relic art: they are drawn *into* the enemy
	// card by internal/cards, which has no graphics context.
	images["fireeffect_png"] = fireeffect_png
	images["frozeneffect_png"] = frozeneffect_png
	images["thundereffect_png"] = thundereffect_png
	images["eartheffect_png"] = eartheffect_png
	images["defaulteffect_png"] = defaulteffect_png

	// The glyph art. internal/systems takes the bytes rather than an *ebiten.Image for the
	// same reason the relic art does: RenderGlyph draws into a plain Go image so the contact
	// sheets can be built with no window.
	images["shermansword_png"] = shermansword_png
	images["shermanshield_png"] = shermanshield_png

	// The four form marks, for the same reason again: internal/cards draws them into a card
	// and has no graphics context.
	images["formstab_png"] = formstab_png
	images["formslash_png"] = formslash_png
	images["formcrush_png"] = formcrush_png
	images["formdefend_png"] = formdefend_png

	// The upgrade inks, for the same reason: internal/cards samples them into a card face and
	// has no graphics context.
	images["wildcardupgrade_png"] = wildcardupgrade_png

	return images
}

// embedFamily files every PNG in one embedded directory into images, keyed by filename stem —
// `enemy/ogrewarlord-portrait.png` is `ogrewarlord-portrait`, which is what `data/enemies.json`
// writes in its Portrait field, and `relic/fire.png` is `fire`.
//
// **Four directories read the same way, so it is one function** *(2026-09-11)*. It was two
// hand-written walks for the two portrait families; the relic and worm art joined them and a
// third and fourth copy of the same eight lines is how one of them comes to skip a file or key
// it differently.
//
// **A read failure is impossible in a built binary** — the files are compiled in — so a panic is
// the honest response to one rather than a silently short roster.
func embedFamily(images map[string][]byte, fsys embed.FS, dir string) {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		log.Fatalf("failed to read the embedded %s directory: %v", dir, err)
	}
	for _, e := range entries {
		raw, err := fsys.ReadFile(dir + "/" + e.Name())
		if err != nil {
			log.Fatalf("failed to read embedded %s/%s: %v", dir, e.Name(), err)
		}
		images[strings.TrimSuffix(e.Name(), ".png")] = raw
	}
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
