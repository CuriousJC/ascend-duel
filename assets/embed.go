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
// `upgrade/`, `sounds/` — and the //go:embed paths below are relative to this file, so a
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

// BUTTONS
//
// Every file under `button/`, as one family keyed by filename stem, so a new face or icon is a file
// dropped in and nothing edited here.
//
//   - **`button-<color>`** is the blank body of a word button, drawn at 512x128 and scaled by the
//     caller to the button's height tier: two end caps and a middle. See systems.DrawButton.
//   - **`icon-<name>`** is a square button drawn whole, face and picture together, at 256x256. See
//     models.Button.Icon. A stem ending `-pressed` is the same button drawn latched — see
//     systems.drawButtonIcon.
//   - **`icon-blank*` and `glyph-<name>`** are the parts of a square button assembled in code: a
//     blank tile and a picture laid over it.
//
// The prompts are under docs/art/; docs/art/README.md says which file each came from.
//
//go:embed button/*.png
var buttonArt embed.FS

// THE GUIDE
//
// The guide: the face on the tutorial's speech bubble. **A named one-off rather than a member of
// the creature family**, because it is not an opponent — nothing places it on a realm and nothing
// fights it, and filing it under `motifs/` would put it in the tree the roster is read against,
// where a picture with no record behind it is a record somebody has lost.
//
// A faceless human shape made of rainbow vapour, at 256x256 on a transparent ground for the bubble.
// See docs/art/guide_art_prompt.MD.
//
//go:embed game/guide.png
var guide_png []byte

// The same figure rendered for a card — 200x280 and opaque, because a bleeding card has nothing
// behind it — keyed `guide-card`. It is The Prismatic in the opponent's corner on every screen
// that is not a fight, and it is not the placeholder: the guide is somebody, where the question
// mark is a picture nobody has made.
//
//go:embed game/guide-card.png
var guide_card_png []byte

// THE PLACEHOLDER
//
// The one face every record with no picture of its own draws, in every catalog and on the roster —
// a rainbow vapor question mark at the card's own 200x280, keyed `placeholder` (data.PlaceholderArt).
// The Compendium draws it for a record the player has not found yet. **A named one-off rather than a
// member of any family**, because it belongs to none of them. See
// docs/art/placeholder_art_prompt.MD. The achievement icons carry their own square cut of it as
// `achievement/default.png`, at Steam's 256.
//
//go:embed game/placeholder.png
var placeholder_png []byte

// THE GEAR

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

// ACHIEVEMENT ICONS
//
// One square icon per achievement — `data/achievements.json` names it in `AchievedIcon` — committed
// at **256x256, Steam's own upload size**, which the achievements page draws at Steam's 64. **Keyed
// `achievement-<stem>`** for the upgrade art's reason: the map is flat, and `prism` or `kitchen-sink` is
// a name a relic could take. `default.png` is what a record with no icon draws. There is no
// unachieved file anywhere: that icon is this one in grayscale, see systems.ArtMarkGray.
//
//go:embed achievement/*.png
var achievementArt embed.FS

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
// **Not embedded by this package.** They are most of the embedded weight, so they are their own
// package, `assets/motifs`, and only what draws a creature or a room imports it — the game and the
// roster tools. Everything else that links this package, every test binary and review sheet
// included, carries none of them. The package registers itself through RegisterImages, so
// LoadImageData is still the one door; see `go doc ./assets/motifs`.

// registered is every family added from outside this package, in the order it was registered.
var registered []func(images map[string][]byte)

// RegisterImages adds a family to what LoadImageData hands back. **Called from a package's init**,
// so a blank import is the whole of opting in, and the family is filed before anything can read
// the map. A program that never imports the package draws a placeholder wherever one of its keys
// is named, which is what a missing picture does everywhere.
func RegisterImages(fill func(images map[string][]byte)) {
	registered = append(registered, fill)
}

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

// THE PARTICLES: small elemental sprites thrown around a hit's total and trailed behind its figure,
// keyed by filename stem — `particle-<element>-<n>` (a piece of the element) and `spark-<element>-<n>`
// (a plain fleck in its color). Still pictures; every movement is code. See
// docs/art/particle_art_prompt.MD, docs/art/spark_art_prompt.MD and ui.DrawParticles.
//
//go:embed particle/*.png
var particleArt embed.FS

// THE SHIELDS: a shield standing over the card that raised it, keyed by filename stem —
// `shield-<element>`, plus `shield-neutral`. See docs/art/shield_art_prompt.MD and ui.ShieldKey.
//
//go:embed shield/*.png
var shieldArt embed.FS

// THE SCROLLS: the papyrus a card's working is written on, keyed by filename stem —
// `scroll-light`, `scroll-mid`, `scroll-deep`. A top roll, a middle that stretches, a bottom roll.
// See docs/art/scroll_art_prompt.MD and ui.DrawScroll.
//
//go:embed scroll/*.png
var scrollArt embed.FS

// The relic faces, globbed as a family and keyed by filename stem — `relic/fire.png` is
// `fire`, which is what `data/relics.json` writes in its Art field.
//
// **A family rather than a var per file**, because a catalog this size as three edits a picture is
// hundreds of lines no reviewer could check. The cost is the documented one — a relic's key is tied
// to its filename, so renaming a file means editing `relics.json`. A relic with no Art draws the
// placeholder.
//
//go:embed relic/*.png
var relicArt embed.FS

// The essence faces, the same way and for the same reason.
//
//go:embed essence/*.png
var essenceArt embed.FS

// The rune faces, the same way.
//
//go:embed rune/*.png
var runeArt embed.FS

// The cantrip faces, the same way.
//
//go:embed cantrip/*.png
var cantripArt embed.FS

// The playing-card faces, one per card per element, globbed the same way — `card/jab-fire.png` is
// the key `jab-fire`, which is what `data/card_art.json` writes in its Art field.
//
// **There is no fallback face and there must not be.** Every other family falls back to the
// placeholder because a card with no picture would be a blank rectangle; a playing card already says what it
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

// The creature cards' faces, one per flavor per element — `enemy-card/stab-bite-fire.png` is the
// key `stab-bite-fire`, which is what `data.EnemyCardArtData.ArtKey` builds from the flavor's Art
// stem and the element the realm dealt the card as. No default face, for the playing cards' reason,
// and the directory is embedded whole so its README keeps the glob compiling before any art lands.
//
//go:embed enemy-card
var enemyCardArt embed.FS

// The damage badges, globbed the same way — `damage/damage-diamond-fire.png` is the key
// `damage-diamond-fire`, which is what cards.BadgeFallbackArtKey builds, and every numbered one is
// what cards.BadgeArtKey builds.
//
//go:embed damage/*.png
var damageArt embed.FS

// The stone faces, the same way. A stone with no Art draws the placeholder — see
// data.DefaultStoneArt.
//
//go:embed stone/*.png
var stoneArtFS embed.FS

// The shop's other cards: the potions and tonics in data/potions.json and data/tonics.json, and
// the sealed goods in data/goods.json. **One directory for several catalogs**, which is the one
// place a family is not one catalog — they share
// docs/art/other_card_art_prompt.MD, they are generated in one batch and they are filed by one
// command, so splitting them into assets/potion and assets/good would be two directories of three
// files that nothing ever tells apart. The keys are record ids either way and the map is flat.
//
// A potion or tonic with no Art draws the placeholder, and a sealed good borrows the picture of
// whatever is inside it — see data.PotionData.ArtKey and screens.goodArt.
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
	buttons := map[string][]byte{}
	embedFamily(buttons, buttonArt, "button")
	for key, raw := range buttons {
		assets[key] = loadImage(raw)
	}
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
	embedFamily(images, enemyCardArt, "enemy-card")
	embedFamily(images, damageArt, "damage")
	embedFamily(images, stoneArtFS, "stone")
	embedFamily(images, otherArt, "other")
	embedFamily(images, formArt, "form")
	embedFamily(images, buttonArt, "button")
	embedFamily(images, textureArt, "texture")
	embedPrefixed(images, upgradeArt, "upgrade-art", "upgrade-")
	embedPrefixed(images, deckArt, "deck", "deck-")
	embedPrefixed(images, achievementArt, "achievement", "achievement-")
	embedFamily(images, barArt, "bar")
	for _, fill := range registered {
		fill(images)
	}
	embedFamily(images, screenArt, "screen")
	embedFamily(images, portalArt, "portal")
	embedFamily(images, particleArt, "particle")
	embedFamily(images, shieldArt, "shield")
	embedFamily(images, scrollArt, "scroll")

	// Bob's face, for the reason the relic art is here: the tutorial draws him into a card
	// through internal/cards, which has no graphics context.
	images["guide_png"] = guide_png
	images["guide-card"] = guide_card_png

	// The placeholder, for the same reason: every catalog's card draws it through internal/cards.
	images["placeholder"] = placeholder_png

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

// EmbedTree is embedFamily for a directory of directories: every picture anywhere under root,
// keyed by filename stem, with the directories it sits in left out of the key. It is what lets
// `assets/motifs/` mirror `data/motifs/` without a record having to know which folder its picture
// was filed in. **Only pictures are filed**, so a Go file sitting in the tree is not a key.
//
// **Two files sharing a stem is fatal**, where embedFamily never has to ask: one flat directory
// cannot hold two files of one name, and a tree can.
func EmbedTree(images map[string][]byte, fsys fs.FS, root string) {
	where := map[string]string{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || imageStem(path.Base(p)) == path.Base(p) {
			return err
		}
		raw, err := fs.ReadFile(fsys, p)
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
