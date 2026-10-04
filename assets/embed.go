package assets

// embed.go

import (
	"bytes"
	"embed"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Files are grouped into directories by what they are for — `game/`, `motifs/`, `relic/`,
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

// BAR CELLS
//
// The three states a resource bar's cell can be in — a point still to spend, a point spent, a
// point spent past what there was — drawn once each and stretched by the caller to whatever width
// the row needs. **A cell is two end caps and a middle every column of which is identical**, so
// only the middle stretches; see ui.DrawBarCell and docs/art/bar_art_prompt.MD.
//
//go:embed bar/bar-cell-empty.png
var barCellEmpty_png []byte

//go:embed bar/bar-cell-spent.png
var barCellSpent_png []byte

//go:embed bar/bar-cell-over.png
var barCellOver_png []byte

// HEALTH BAR
//
// The whole bar directory again, as a family, for the health bar: its trough, its fill and the
// marker laid across it are drawn *into* a fighter card by internal/cards, which has no graphics
// context — so they go through LoadImageData as bytes, keyed by stem (`health-bar-trough`). See
// docs/art/health_bar_art_prompt.MD.
//
//go:embed bar/*.png
var barArt embed.FS

// BUTTON FACES
//
// The blank body of every button, one per color at rest plus the flat disabled face, drawn at
// 512x128 and scaled by the caller to the button's height tier. **Like a bar cell, a face is two
// end caps and a middle every column of which is identical**, so only the middle stretches; see
// systems.DrawButton and docs/art/button_art_prompt.MD. Keyed `button-<color>`, the file's stem.
//
//go:embed button/button-red.png
var buttonRed_png []byte

//go:embed button/button-gray.png
var buttonGray_png []byte

//go:embed button/button-yellow.png
var buttonYellow_png []byte

//go:embed button/button-blue.png
var buttonBlue_png []byte

//go:embed button/button-pink.png
var buttonPink_png []byte

//go:embed button/button-olive.png
var buttonOlive_png []byte

//go:embed button/button-disabled.png
var buttonDisabled_png []byte

// THE GUIDE
//
// The guide: the face on the tutorial's speech bubble. **A named one-off rather than a member of
// the creature family**, because it is not an opponent — nothing places it on a realm and nothing
// fights it, and filing it under `motifs/` would put it in the tree the roster is read against,
// where a picture with no record behind it is a record somebody has lost.
//
// **It is the same figure as `motifs/default-enemy.png`**, rendered for a different canvas: a
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
// A *family* rather than four more named vars, on the terms `relic/` and `motifs/` are already
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

// UPGRADE INKS
//
// The color a *visible upgrade* is named in where it is written rather than drawn: CHROMATIC in a
// tooltip title is set letter by letter across this file's five bands. **It is a color source, not
// a picture** — nothing blits it; `internal/systems` decodes it and `internal/cards` samples it.
// Authored at 32x32, `systems.UpgradeInkSize`.
//
//go:embed upgrade/wildcard.png
var wildcardupgrade_png []byte

// UPGRADE ART
//
// The picture an upgraded card wears over its face — `data/upgrade_art.json` names one per upgrade,
// and `internal/cards` lays it over the face inside the border ring. Committed at the card's own
// 200x280 like the playing cards' art. **Keyed `upgrade-<stem>`** rather than by the bare stem every
// other family uses, because an upgrade's name — `versatile`, `held-vitae` — is also an essence's or
// a rune's, and the map is flat. See embedPrefixed.
//
//go:embed upgrade-art/*.png
var upgradeArt embed.FS

// CARD BACKS
//
// The back of every card in one deck — `data/decks.json` names one per deck, and `internal/cards`
// lays it inside the back's one-pixel rim. Committed at the card's own 200x280. **Keyed `deck-<stem>`**
// for the upgrade art's reason: the map is flat, and a deck named after a gem is a name a relic could
// take. **The directory may hold nothing but its README**, as the playing cards' may, so the family
// lands before the art does. See embedPrefixed.
//
//go:embed deck
var deckArt embed.FS

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

// THE MOTIFS: every creature's picture and every duel's backdrop
//
// **One directory per motif, mirroring `data/motifs/<motif>/`**, so what a motif has been drawn
// is one folder to open: `motifs/goblins/creature/` and `motifs/goblins/backdrop/`. The two
// placeholders sit at the top of the tree, `motifs/default-enemy.png` and
// `motifs/default-background.jpg`, because they belong to no motif.
//
// **Keyed by filename stem, and the directory is not part of the key.**
// `motifs/goblins/creature/goblins-serf-fire.png` is `goblins-serf-fire`, which is what
// `data.MotifRecord.ArtKey` builds out of the record's `Art` field and the element the realm dealt
// it as; `motifs/goblins/backdrop/goblins-outer-tinker-studio-fire.jpg` is `goblins-outer-tinker-studio-fire`,
// which is what `data.Backdrop.ArtKey` builds the same way. So a file can be refiled without
// touching a record, and two files anywhere in the tree sharing a stem are one lookup with two
// answers — `embedTree` refuses that at load.
//
// **Embedded as a tree rather than one var each, which is a deliberate exception to the
// three-edit rule** at the top of this file. That rule — the file, an //go:embed var, a map
// entry — is right for a handful of named assets and absurd for a roster of this size: it would
// be hundreds of lines no reviewer could check, drifting the first time a creature was renamed.
// The consequence, stated because it is the thing the rule was protecting: **a picture's key is
// tied to its filename**, so renaming one means editing the `Art` field of the record that names
// it.
//
// **A missing picture falls back to its placeholder**, so a blank face means art nobody has made
// rather than a name nobody spelled right, and the plain backdrop in a duel means no backdrop
// fits that room. One placeholder of each kind for every motif: a per-motif placeholder is a
// picture somebody has to draw before the motif can be looked at.
//
// **A creature is a PNG and a backdrop is a JPEG.** A backdrop is opaque edge to edge at the
// screen's own 1920x1080, and a painted scene is about a sixth the size that way; the image/jpeg
// import above is what decodes one. Both are handed out as bytes and decoded by whoever draws
// them, because a decoded 1920x1080 picture is eight megabytes and a run only shows a handful.
//
// Provenance: generated from the prompts under `docs/art/`, like everything else in `assets/`.
//
//go:embed motifs
var motifArt embed.FS

// THE SCREEN BACKDROPS: the picture behind every screen that is not a duel
//
// A JPEG at 1920x1080 for the motif backdrops' reason, keyed by filename stem —
// `screen/screen-fog.jpg` is `screen-fog`, which is what `data/screen_art.json` writes in its Art
// field. See docs/art/screen_art_prompt.MD.
//
//go:embed screen/*.jpg
var screenArt embed.FS

// THE PORTAL SWIRL: the round, rim-faded swirl the portal screen turns in place over each painted
// portal, as the way through it. One transparent picture, rotated by the game rather than animated
// in frames. See docs/art/portal_swirl_art_prompt.MD and systems.DrawSwirl.
//
//go:embed portal/*.png
var portalArt embed.FS

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

// The cantrip faces, a family of their own for the rune's reason:
// `cantrip/default-cantrip.png` is a copy of the rune's placeholder rather than a share of it, so an
// undrawn cantrip and an undrawn rune are two backlogs that can be cleared one at a time.
//
//go:embed cantrip/*.png
var cantripArt embed.FS

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

// The prose glyphs: the reading set — upper and lower case, digits and punctuation — for every line
// of text longer than a label. One white sheet under a thin black outline, which the drawing
// multiplies by an ink or a material, plus `prose-glyphs.json` for the cells and the advances. See
// systems.DrawProse and docs/art/prose_art_prompt.MD.
//
//go:embed prose/prose-glyphs.png prose/prose-glyphs.json
var proseArt embed.FS

// ProseFile returns one file of the prose set.
func ProseFile(name string) ([]byte, error) {
	return proseArt.ReadFile("prose/" + name)
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
//go:embed sounds/duello.mid
var duello_mid []byte

// FONTS
//
//go:embed game/Kubasta.ttf
var kubasta []byte

// LoadAssets returns a mapped set of images for the game
func LoadAssets() map[string]*ebiten.Image {
	assets := make(map[string]*ebiten.Image)

	assets["title_png"] = loadImage(title_png)
	assets["barCellEmpty_png"] = loadImage(barCellEmpty_png)
	assets["barCellSpent_png"] = loadImage(barCellSpent_png)
	assets["barCellOver_png"] = loadImage(barCellOver_png)
	assets["button-red"] = loadImage(buttonRed_png)
	assets["button-gray"] = loadImage(buttonGray_png)
	assets["button-yellow"] = loadImage(buttonYellow_png)
	assets["button-blue"] = loadImage(buttonBlue_png)
	assets["button-pink"] = loadImage(buttonPink_png)
	assets["button-olive"] = loadImage(buttonOlive_png)
	assets["button-disabled"] = loadImage(buttonDisabled_png)
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

	music["duello_mid"] = duello_mid

	return music
}

// LoadFonts returns a mapped set of fonts for the game
func LoadFonts() map[string]*text.GoTextFaceSource {
	fonts := make(map[string]*text.GoTextFaceSource)

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

	// The families read out of an embedded directory rather than listed one by one. See
	// embedFamily, and the //go:embed lines above for what each key ends up being.
	embedFamily(images, relicArt, "relic")
	embedFamily(images, essenceArt, "essence")
	embedFamily(images, runeArt, "rune")
	embedFamily(images, cantripArt, "cantrip")
	embedFamily(images, cardArt, "card")
	embedFamily(images, damageArt, "damage")
	embedFamily(images, stoneArtFS, "stone")
	embedFamily(images, otherArt, "other")
	embedFamily(images, formArt, "form")
	embedFamily(images, textureArt, "texture")
	embedPrefixed(images, upgradeArt, "upgrade-art", "upgrade-")
	embedPrefixed(images, deckArt, "deck", "deck-")
	embedFamily(images, barArt, "bar")
	embedTree(images, motifArt, "motifs")
	embedFamily(images, screenArt, "screen")
	embedFamily(images, portalArt, "portal")

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

	// The upgrade ink, for the same reason: internal/cards samples it to color a word and has no
	// graphics context.
	images["wildcardupgrade_png"] = wildcardupgrade_png

	return images
}

// embedFamily files every PNG in one embedded directory into images, keyed by filename stem —
// `relic/fire.png` is `fire`, which is what `data/relics.json` writes in its Art field.
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
		images[imageStem(e.Name())] = raw
	}
}

// embedPrefixed is embedFamily with a prefix on every key: `upgrade-art/golden.png` under the
// prefix `upgrade-` is `upgrade-golden`. **The file is still named for its record**, so a picture
// is filed by the record's key exactly as every other family's is; the prefix only keeps the flat
// map from colliding with a family whose records share a name.
func embedPrefixed(images map[string][]byte, fsys embed.FS, dir, prefix string) {
	family := map[string][]byte{}
	embedFamily(family, fsys, dir)
	for k, raw := range family {
		images[prefix+k] = raw
	}
}

// embedTree is embedFamily for a directory of directories: every picture anywhere under root,
// keyed by filename stem, with the directories it sits in left out of the key. It is what lets
// `assets/motifs/` mirror `data/motifs/` without a record having to know which folder its picture
// was filed in.
//
// **Two files sharing a stem is fatal**, where embedFamily never has to ask: one flat directory
// cannot hold two files of one name, and a tree can.
func embedTree(images map[string][]byte, fsys embed.FS, root string) {
	where := map[string]string{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := fsys.ReadFile(p)
		if err != nil {
			return err
		}
		key := imageStem(path.Base(p))
		if other, clash := where[key]; clash {
			log.Fatalf("embedded %s and %s are both the picture %q", other, p, key)
		}
		where[key] = p
		images[key] = raw
		return nil
	})
	if err != nil {
		log.Fatalf("failed to read the embedded %s tree: %v", root, err)
	}
}

// imageStem is a family file's key: its name less the picture extension. Only a picture's extension
// comes off, so a stray file that is not a picture keeps its whole name rather than colliding with
// one that is.
func imageStem(name string) string {
	for _, ext := range []string{".png", ".jpg"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
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
