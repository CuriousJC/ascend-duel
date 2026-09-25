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
// The guide: the face on the tutorial's speech bubble. **A named one-off rather than a member of
// the creature family**, because it is not an opponent — nothing places it on a floor and nothing
// fights it, and filing it under `enemy/` would put it in the glob the roster is read against,
// where a picture with no record behind it is a record somebody has lost.
//
// **It is the same figure as `enemy/default-enemy.png`**, rendered for a different canvas: a
// faceless human shape made of rainbow vapour, which is what both the guide and the not-yet-drawn
// creature have in common — neither of them is anybody in particular. See
// docs/art/guide_art_prompt.MD, which asks for the two renders together and says why one file
// cannot serve both: this one is 256x256 with a transparent ground for the bubble, and the card's
// is 200x280 and opaque because a bleeding card has nothing behind it.
//
//go:embed game/guide.png
var guide_png []byte

// THE GEAR
//
// The settings control in the game's chrome corner. **A named one-off rather than a member of the
// form family**, because it is not a card mark: it says something about the program where every
// mark in `form/` says something about a card.
//
// **Baked from the silhouette generator on 2026-09-16 and then the generator was deleted**
// *(owner's call)*. It was the last kind `internal/systems` drew, so the picture was rendered once
// at the size the chrome blits it and committed as an ordinary asset — no change to what is on
// screen, and replacing it with drawn art is now a file swap. `docs/art/gear_art_prompt.MD` is the
// brief for that replacement.
//
//go:embed game/gear.png
var gear_png []byte

// FORM MARKS AND COST TICKS
//
// A *family* rather than four more named vars, on the terms `relic/` and `enemy/` are already
// globbed: there is one mark per form per element and one tick per element, which is twenty-five
// files today and a multiplication rather than a list. Keyed by filename stem, so
// `form/formslash-fire.png` is `formslash-fire` and `form/tick-earth.png` is `tick-earth` —
// which is what `internal/cards` builds from a card's own form and element.
//
// **There is no fallback.** A card with no element — a relic, a fighter, anything Basic — draws
// the *neutral* mark and the neutral tick, which are files in this same family; see
// cards.MarkArtKey and cards.TickArtKey, both of which are total. The four hueless PNGs that used
// to serve that case were deleted on 2026-09-16 along with the whole drawn-glyph path.
//
//go:embed form/*.png
var formArt embed.FS

// UPGRADE ART
//
// The color a *visible upgrade* paints a card's left column from. A card the run has altered
// used to look exactly like one it had not — sixteen runes attach seven kinds of rider and
// the only place any of them was visible was the tooltip prose — and this is the first of them
// that says so on the face.
//
// **It is a color source, not a picture.** Nothing blits this file: `internal/systems` decodes
// it and `internal/cards` samples it to paint the form mark and the cost ticks, exactly where the
// element's own color would otherwise go. So it is authored at the form marks' 32x32, the box it
// has to cover, rather than at the 64 the drawn glyphs use.
//
// **Its own group rather than `form/`**, because more visible upgrades are coming and the next
// one will have nothing to do with the form mark.

// MATERIAL TEXTURES
//
// The stone and metal a *word* is set in on a dark panel: steel for SLASH, ivory for STAB, granite
// for CRUSH. **Not a picture anything blits whole** — `internal/systems` tiles one behind the glyph
// shapes of a form word and keeps what lands inside them, so what matters is the grain rather than
// the composition.
//
// **Seamless tiles at 114x114, box-reduced from a 1254 source by exactly eleven.** An exact integer
// factor is what keeps a seamless tile seamless; a fractional one bleeds the far edge into the near
// one and the join shows as a line down the middle of a letter. The reduction is what puts the
// grain at word scale: at native resolution one crystal is most of a capital and the word reads as
// a blotch.
//
// **Its own group rather than `form/`**, which is marks drawn *onto* a card — these are drawn
// *into* type, and a file in that glob would be a mark `cards.MarkArtKey` never names.
//
//go:embed texture/*.png
var textureArt embed.FS

//go:embed upgrade/wildcard.png
var wildcardupgrade_png []byte

// CREATURES
//
// **One picture per record per element**, keyed by filename stem: `enemy/goblins-serf-fire.png`
// is `goblins-serf-fire`, which is what `data.MotifRecord.ArtKey` builds out of the record's `Art`
// field and the element the floor dealt it as. A fire goblin serf and an ice goblin serf are two
// drawings of one creature.
//
// **Embedded as a directory rather than one var each, which is a deliberate exception to the
// three-edit rule** at the top of this file. That rule — the file, an //go:embed var, a map
// entry — is right for a handful of named assets and absurd for a roster of this size: it would
// be hundreds of lines no reviewer could check, drifting the first time a creature was renamed.
// So the pictures are a *family*, globbed in and keyed by stem.
//
// The consequence, stated because it is the thing the rule was protecting: **a picture's key is
// tied to its filename**, so renaming one means editing the `Art` field of the record that names
// it.
//
// **`default-enemy.png` is the whole of the fallback**, and it is what nearly every record draws
// today. A record whose own picture has not been generated yet falls back to it rather than
// drawing a hole, so a blank face means art nobody has made rather than a name nobody spelled
// right. One placeholder for every motif and both kinds of record: a per-motif placeholder is a
// picture somebody has to draw before the motif can be looked at.
//
// Provenance: generated from the prompts under `docs/art/`, like everything else in `assets/`.
//
//go:embed enemy/*.png
var portraits embed.FS

// The relic faces, globbed as a family and keyed by filename stem — `relic/fire.png` is
// `fire`, which is what `data/relics.json` writes in its Art field.
//
// **It became a family on 2026-09-11**, having been one `//go:embed` var per file. Five pictures
// is three edits each and readable; a catalog of a hundred and thirty-seven relics being drawn
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

// The essence faces, the same way and for the same reason. `essence/default-essence.png` is what an essence
// with no Art of its own draws — **a copy of the relic's default rather than a share of it**
// *(owner's call, 2026-08-22)*: two files that happen to look alike today are two files that can
// be replaced one at a time.
//
//go:embed essence/*.png
var essenceArt embed.FS

// The rune faces, a family of their own as of 2026-09-12. **They wore the essence's placeholder
// until then**, and the same argument that split the essence's off the relic's splits this off the
// essence's: the day either catalog gets art, one shared picture is a page where a drawn essence and
// an undrawn rune look identical, and a backlog you cannot see is a backlog nobody clears.
//
//go:embed rune/*.png
var runeArt embed.FS

// The playing-card faces, one per card per element, globbed the same way — `card/jab-fire.png` is
// the key `jab-fire`, which is what `data/card_art.json` writes in its Art field.
//
// **There is no default-card.png and there must not be.** Every other family has a fallback face
// because a card with no picture would be a blank rectangle; a playing card already says what it
// is through its mark, its ticks, its name and its text, so a record with no Art draws the card
// exactly as it looked before this family existed. One shared placeholder across 95 records would
// put 95 copies of one picture on the table at once. See data.DefaultCardArt.
//
// **The directory may legitimately hold nothing but a README**, which is what it holds today —
// `embed.FS` over a pattern matching no file is empty rather than an error, so the glob is safe
// to land before the art does.
//
//go:embed card
var cardArt embed.FS

// The damage badges, globbed the same way — `damage/damage-diamond-fire.png` is the key
// `damage-diamond-fire`, which is what cards.BadgeArtKey builds.
//
// **Three shapes in six colors, and only one shape is drawn.** Which one is
// `cards.DefaultBadgeShape`; the other twelve files are kept because the choice is still open and
// regenerating a batch to change one's mind is the expensive way to look at an alternative.
//
//go:embed damage/*.png
var damageArt embed.FS

// The stone faces, a family of their own as of 2026-09-14. **There is no default-stone.png**, which
// is the one place this family departs from the three above it: a stone with no Art draws the
// relics' default face, which is what an unpainted stone draws — see data.DefaultStoneArt.
// A generated fallback says "nobody has drawn this yet" better than a painted one can, and it
// leaves nothing to license.
//
//go:embed stone/*.png
var stoneArtFS embed.FS

// The shop's other cards, a family of their own as of 2026-09-15: the potions in
// data/potions.json and the sealed goods in data/goods.json. **One directory for two
// catalogs**, which is the one place a family is not one catalog — they share
// docs/art/other_card_art_prompt.MD, they are generated in one batch and they are filed by one
// command, so splitting them into assets/potion and assets/good would be two directories of three
// files that nothing ever tells apart. The keys are record ids either way and the map is flat.
//
// **There is no default-other.png.** A potion with no Art draws the relic catalog's default face
// and a sealed good borrows the picture of whatever is inside it — see data.PotionData.ArtKey and
// screens.goodArt, both of which predate this family and both of which still decide the empty case.
//
//go:embed other/*.png
var otherArt embed.FS

// The figure glyphs: the numerals and math symbols the combat screen sets into every figure that
// flies over the table. **A directory per set, and one set is drawn** — `systems.DefaultFigureSet`
// names it — so an alternative look is a second directory beside the first rather than a batch
// written over it. Each set is a sprite sheet per color plus one `figure-glyphs.json` saying where
// every glyph sits in a sheet and how far it advances.
//
//go:embed figure
var figureArt embed.FS

// FigureFile returns one file out of one figure set: `FigureFile("v2", "figure-glyphs.json")`.
//
// **Bytes, and an error rather than a panic**, because the set is named by a constant a reviewer
// flips and a set that is not there is a typo to report, not a corrupt binary.
func FigureFile(set, name string) ([]byte, error) {
	return figureArt.ReadFile("figure/" + set + "/" + name)
}

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
// Scores are Standard MIDI Files, not recorded audio: internal/music synthesizes them
// at startup. That is why a whole track is a kilobyte and why there is no soundfont
// here whose license would have to be cleared before the game could be sold.
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
	// **The relic and essence art joined them on 2026-09-11**, having been decoded here as well as
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
	embedFamily(images, essenceArt, "essence")
	embedFamily(images, runeArt, "rune")
	embedFamily(images, cardArt, "card")
	embedFamily(images, damageArt, "damage")
	embedFamily(images, stoneArtFS, "stone")
	embedFamily(images, otherArt, "other")
	embedFamily(images, formArt, "form")
	embedFamily(images, textureArt, "texture")
	embedFamily(images, portraits, "enemy")

	// Bob's face, for the reason the relic art is here: the tutorial draws him into a card
	// through internal/cards, which has no graphics context.
	images["guide_png"] = guide_png
	images["gear"] = gear_png

	// The status badges, for the same reason as the relic art: they are drawn *into* the enemy
	// card by internal/cards, which has no graphics context.
	images["fireeffect_png"] = fireeffect_png
	images["frozeneffect_png"] = frozeneffect_png
	images["thundereffect_png"] = thundereffect_png
	images["eartheffect_png"] = eartheffect_png
	images["defaulteffect_png"] = defaulteffect_png

	// The glyph art. internal/systems takes the bytes rather than an *ebiten.Image for the
	// same reason the relic art does: internal/cards draws into a plain Go image so the contact
	// sheets can be built with no window.

	// The four form marks, for the same reason again: internal/cards draws them into a card
	// and has no graphics context.

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
// hand-written walks for the two portrait families; the relic and essence art joined them and a
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
