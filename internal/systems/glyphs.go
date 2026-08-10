package systems

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/assets"
)

// Procedurally generated pixel-art glyphs for the numbers on an action card: what it hits
// for, and what it costs.
//
// Drawn in code rather than loaded from a file because generated art has no provenance
// question at all, which is exactly the problem that makes the Tyrian set a release
// blocker. That is the reason to keep doing interface art this way.
//
// **Two of them are files now, and the reason the rule survives is who drew them.** The
// attack sword and the defend shield are hand-drawn pixel art by KingSherman1820, one of
// the two copyright holders — so the provenance question the generator exists to dodge does
// not arise. See glyphArt below. Art from anywhere else still needs a licence before it can
// be in a game that will be sold, and generating it remains the cheaper answer.
//
// The important part is that this is a *generator*, not a bitmap. A glyph is a filled
// silhouette described by horizontal spans; the outline is derived from the silhouette by
// asking which pixels have a neighbour outside it, and the interior shading is computed
// from where a pixel sits in its row and down the sprite. Nothing is hand-placed, so a
// shape can be nudged without repainting it.
//
// Two earlier versions are worth not repeating. The first was a hand-typed character map
// with one colour and one alpha shade: unresizable, and two values cannot make a bevel. The
// second was this generator at 32x32 drawn on the card at 2x, which gave the game chunky
// two-pixel blocks and left the shading invisible at the only size that matters.

// GlyphSize is both the authored and the displayed size. Art is drawn at 1:1 — magnifying
// it is how the bevel became decoration that only showed up in previews.
const GlyphSize = 64

// CardGlyphScale is how many screen pixels one glyph pixel occupies on an action card.
//
// It lives here rather than with the card so the contact-sheet tool can read it. The
// sheet's actual-size row is drawn at exactly this scale, which means a preview can never
// quietly disagree with the game — reviewing the art at eight times its real size is how
// it came to look acceptable in a picture and clunky in play.
//
// Integer only, and now 1: the art is authored at the size it is shown.
const CardGlyphScale = 1

// GlyphKind selects one of the generated glyphs.
type GlyphKind int

// The clock that used to sit between these two went with initiative on 2026-08-06 — it was
// the card's glyph for a number that no longer exists. It was built parametrically, out of a
// disc() helper that went with it; both are one `git show` away if a round shape is wanted
// again. See TODO.md.
// **Append, never insert.** These are ordinals and the glyph cache keys on them, so
// adding a kind in the middle silently re-points every existing entry — the same hazard
// MECHANICS.md records for the concept enum and its combo IDs.
const (
	// GlyphDamage is a sword: what the action hits for.
	GlyphDamage GlyphKind = iota
	// GlyphActionPoints is a runner: what it costs to play.
	//
	// **No longer drawn on a card.** Cost became a stack of dash marks in the corner
	// when the card was redesigned, which is what freed the height the category glyph
	// now uses. The runner is kept because it is still on the contact sheet and is the
	// obvious art for an action-point figure elsewhere — a budget readout, a ring.
	GlyphActionPoints

	// The three category glyphs: what phase a card resolves in. These replaced the
	// category *word* under the card's name — a picture in the corner where a line of
	// text used to be, which is the same trade the damage and cost badges made when they
	// replaced "init 3".
	//
	// GlyphAttack deliberately shares the sword with GlyphDamage rather than getting a
	// second weapon. They say related things and a card only ever shows one of them in
	// the category slot; two different swords on one face would imply a distinction that
	// is not there.
	GlyphAttack
	// GlyphDefend is a kite shield.
	GlyphDefend
	// GlyphPrepare is an open book.
	GlyphPrepare
)

// GlyphKinds is every glyph, in a fixed order. The contact sheet walks this rather than
// ranging a map, which Go deliberately randomises.
func GlyphKinds() []GlyphKind {
	return []GlyphKind{GlyphDamage, GlyphActionPoints, GlyphAttack, GlyphDefend, GlyphPrepare}
}

// Palette is the set of roles a glyph is painted with. Five values make the bevel — one
// colour scaled down cannot, which is why glyphs are the deliberate exception to the
// name-one-colour rule that governs widgets.
//
// Accent is for detail drawn over the fill, like the clock's hands.
type Palette struct {
	Outline   color.RGBA
	Specular  color.RGBA
	Highlight color.RGBA
	Mid       color.RGBA
	Shade     color.RGBA
	Accent    color.RGBA
}

// PaletteName keys a palette. Today there is one, on purpose.
//
// **Colour is being kept unspent.** Every glyph is drawn in one hueless palette so that
// when an element or a block type arrives it can land on colour and mean something on
// arrival. Painting the glyphs different colours now would look better today and would
// spend the only channel left for saying "this Strike is fire" — the reader would already
// have learned that the sword is grey, and the element would read as an inconsistency.
type PaletteName string

const PaletteWhite PaletteName = "white"

var palettes = map[PaletteName]Palette{
	// Near-white, shaded to grey. Neutral in the strongest sense: it has no hue at all, so
	// the first coloured palette to arrive will read as meaning something rather than as
	// one more decorative choice among several.
	//
	// Accent is near-black rather than a colour, so detail painted over the fill — the
	// clock's hands — reads the way hands on a white dial actually do.
	PaletteWhite: {
		Outline:   color.RGBA{R: 22, G: 24, B: 30, A: 255},
		Specular:  color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Highlight: color.RGBA{R: 232, G: 236, B: 242, A: 255},
		Mid:       color.RGBA{R: 190, G: 196, B: 206, A: 255},
		Shade:     color.RGBA{R: 132, G: 139, B: 152, A: 255},
		Accent:    color.RGBA{R: 46, G: 50, B: 60, A: 255},
	},
}

// PaletteNames is every palette, in a fixed order, for the contact sheet.
func PaletteNames() []PaletteName { return []PaletteName{PaletteWhite} }

// PaletteOf returns a palette by name, falling back to white so a bad key draws something
// rather than nothing.
func PaletteOf(name PaletteName) Palette {
	if p, ok := palettes[name]; ok {
		return p
	}
	return palettes[PaletteWhite]
}

// span is an inclusive run of pixels on one row. A row may hold several — a runner's
// trailing arm, torso and leading arm are three separate runs on the same line.
type span struct{ x0, x1 int }

// shape is a glyph's silhouette plus any detail painted over it.
//
// size is the canvas it was authored on, defaulting to GlyphSize. **A shape may not be
// drawn at any size but its own** — the rim is derived one pixel thick and the shading
// is computed per row, so resampling either destroys the rim or smears the bevel. A
// glyph wanted smaller is therefore a *different shape authored smaller*, not the same
// spans scaled, which is why the category glyphs below are their own drawings rather
// than the 64px ones divided by three.
type shape struct {
	size   int
	fill   map[int][]span
	accent map[int][]span
}

func (s shape) canvas() int {
	if s.size > 0 {
		return s.size
	}
	return GlyphSize
}

// Nothing in a silhouette may be thinner than about five pixels. The rim is derived, so it
// takes one pixel off each side of every feature — a three-pixel crossguard renders as two
// rows of outline around a single row of metal, and reads as a scratch. Every span below is
// sized against that floor, which is the main constraint this technique imposes.

// A sword pointing up: tapered blade, deep crossguard, slim grip, modest pommel. The grip
// and pommel are narrower than the blade on purpose — a hilt as thick as the blade reads as
// a cross rather than as a weapon.
var swordShape = shape{
	fill: map[int][]span{
		4:  {{31, 32}},
		5:  {{30, 33}},
		6:  {{29, 34}},
		7:  {{28, 35}},
		8:  {{27, 36}},
		9:  {{26, 37}},
		10: {{25, 38}}, 11: {{25, 38}}, 12: {{25, 38}}, 13: {{25, 38}},
		14: {{25, 38}}, 15: {{25, 38}}, 16: {{25, 38}}, 17: {{25, 38}},
		18: {{25, 38}}, 19: {{25, 38}}, 20: {{25, 38}}, 21: {{25, 38}},
		22: {{25, 38}}, 23: {{25, 38}}, 24: {{25, 38}}, 25: {{25, 38}},
		26: {{25, 38}}, 27: {{25, 38}}, 28: {{25, 38}}, 29: {{25, 38}},
		30: {{25, 38}}, 31: {{25, 38}}, 32: {{25, 38}}, 33: {{25, 38}},
		34: {{25, 38}},
		35: {{8, 55}}, 36: {{8, 55}}, 37: {{8, 55}}, 38: {{8, 55}},
		39: {{9, 54}}, 40: {{11, 52}}, 41: {{13, 50}},
		42: {{28, 35}}, 43: {{28, 35}}, 44: {{28, 35}}, 45: {{28, 35}},
		46: {{28, 35}}, 47: {{28, 35}}, 48: {{28, 35}}, 49: {{28, 35}},
		50: {{28, 35}}, 51: {{28, 35}},
		52: {{25, 38}}, 53: {{24, 39}}, 54: {{24, 39}}, 55: {{24, 39}},
		56: {{25, 38}}, 57: {{27, 36}},
	},
}

// A runner in stride, facing right: head forward of the hips, torso leaning into it, one
// arm driving forward and one trailing, one knee up and the other leg extended behind.
//
// A running figure says "how fast" in a way a jumping jack does not — a jack is symmetrical
// and symmetry reads as standing still, which is the wrong idea for action points.
//
// The stride is the whole silhouette. Limbs stay eight or more pixels thick so the derived
// rim leaves something inside them.
var runShape = shape{
	fill: map[int][]span{
		// Head, forward of centre, with a neck pinch so it reads as a head rather than as
		// the top of the torso.
		8: {{38, 45}}, 9: {{36, 47}}, 10: {{35, 48}},
		11: {{35, 48}}, 12: {{35, 48}}, 13: {{35, 48}},
		14: {{35, 48}}, 15: {{36, 47}}, 16: {{38, 45}},
		17: {{37, 44}}, 18: {{36, 44}},

		// Shoulders and the two arms: one driving forward and up, one trailing back. Every
		// limb span has to *overlap* the torso it hangs off — leave a one-pixel gap and the
		// rim closes around both pieces and the arm renders as a bar floating in space.
		19: {{31, 45}, {44, 54}},
		20: {{30, 45}, {45, 56}},
		21: {{29, 44}, {44, 57}, {22, 30}},
		22: {{29, 44}, {44, 57}, {20, 29}},
		23: {{28, 44}, {44, 56}, {18, 28}},
		24: {{28, 43}, {43, 54}, {16, 27}},
		25: {{27, 43}, {15, 26}},
		26: {{27, 42}, {14, 25}},
		27: {{26, 42}, {13, 24}},

		// Torso, leaning forward.
		28: {{26, 41}}, 29: {{25, 41}}, 30: {{25, 40}}, 31: {{24, 40}},
		32: {{24, 39}}, 33: {{24, 39}}, 34: {{23, 38}}, 35: {{23, 38}},
		36: {{23, 37}}, 37: {{24, 37}},

		// Hips, then the stride: leading thigh up and forward, trailing leg extended back.
		38: {{24, 36}, {36, 44}},
		39: {{23, 35}, {38, 46}},
		40: {{22, 34}, {40, 48}},
		41: {{20, 32}, {41, 49}},
		42: {{18, 30}, {42, 50}},
		43: {{17, 28}, {42, 50}},
		44: {{15, 26}, {41, 49}},
		45: {{14, 25}, {40, 48}},

		// Lower legs: the leading shin drops, the trailing foot kicks up behind.
		46: {{13, 24}, {39, 47}},
		47: {{12, 23}, {39, 47}},
		48: {{11, 22}, {38, 46}},
		49: {{11, 21}, {38, 46}},
		50: {{10, 20}, {38, 46}},
		51: {{10, 20}, {38, 46}},
		52: {{9, 20}, {38, 47}},
		53: {{9, 21}, {38, 49}},
		54: {{10, 22}, {39, 50}},
	},
}

// The category glyphs are small — a third of a damage badge — because they sit above the
// cost dashes in a column a second full badge would fill.
//
// **A generated one is a separate drawing, not a big one shrunk.** Nothing generated here
// can be scaled: the rim is derived one pixel thick, so a third-size copy of a 64px shape
// is a third-size copy of its outline and the interior disappears. At 22 pixels the detail
// budget is almost nothing — a 5px feature is two pixels of rim around three of fill — so
// the shape has to work as pure outline, with at most one accent stroke, and is told apart
// by *proportion* before any detail is legible.
//
// Only the book is still generated. The sword and the shield are drawn art now and are
// downsampled rather than authored small, which is a thing a *painting* survives and a
// derived-rim silhouette does not — see glyphArt.
const categoryGlyphSize = 22

// The generated 22px sword and shield that used to sit here went when Sherman's art
// arrived — see glyphArt below. Both are one `git show` away if the drawn versions are
// ever wanted back.

// An open book, seen end-on: **the two covers form a V**, splayed wide at the top and
// meeting at the spine, with the page block filling the body below.
//
// The first attempt drew it face-on — a wide slab with a gutter line down it — and it read
// as a envelope or a folded card, because face-on the only thing saying "book" is a detail
// too fine to survive 22 pixels. End-on, the V *is* the silhouette, and a silhouette is
// what survives.
//
// The notch between the covers runs ten pixels wide at the top down to two at the spine,
// which is as deep a V as fits without the two halves becoming separate objects.
var smallBookShape = shape{
	size: categoryGlyphSize,
	fill: map[int][]span{
		// The covers, splaying apart upward. Each thickens toward the spine, the way a
		// page block does.
		4:  {{2, 5}, {16, 19}},
		5:  {{2, 6}, {15, 19}},
		6:  {{2, 7}, {14, 19}},
		7:  {{2, 8}, {13, 19}},
		8:  {{2, 8}, {13, 19}},
		9:  {{2, 9}, {12, 19}},
		10: {{2, 9}, {12, 19}},

		// Below the spine the two halves are one mass: the closed part of the book.
		11: {{2, 19}}, 12: {{2, 19}}, 13: {{2, 19}},
		14: {{3, 18}},
		15: {{5, 16}},
	},
	accent: map[int][]span{
		// A page inside each cover, drawn parallel to the splay. Two strokes is all there
		// is room for, and it is what turns a V into a V *with something between it*.
		6:  {{4, 5}, {16, 17}},
		7:  {{4, 6}, {15, 17}},
		8:  {{4, 6}, {15, 17}},
		9:  {{5, 7}, {14, 16}},
		10: {{5, 7}, {14, 16}},
	},
}

var glyphShapes = map[GlyphKind]shape{
	GlyphDamage:       swordShape,
	GlyphActionPoints: runShape,
	GlyphPrepare:      smallBookShape,
}

// glyphArt is the hand-drawn half of the set: a kind whose picture is a PNG rather than a
// silhouette in code. A kind listed here has no entry in glyphShapes and never reaches the
// generator.
//
// **The palette is ignored for these.** A generated glyph is painted from a five-value
// Palette at draw time; a drawing already carries its own colours and is blitted as
// authored. That is a deliberate spend of the colour channel the hueless palette was
// holding — the card's border also carries the element, so an attack card now says
// something in colour twice. It was the owner's call on 2026-08-10.
//
// **Authored at 64 and drawn at 32.** The art is a 64x64 canvas and the category slot is
// nothing like that big, so it is downsampled by a whole factor of two. Halving is the one
// resize hand-drawn art tolerates: a 2x2 block averages to one pixel and a black outline
// survives as a darker pixel rather than vanishing. The generated glyphs cannot do this —
// their rim is derived one pixel thick and averaging it away is exactly the failure the
// "author it small instead" rule exists for.
type glyphArtwork struct {
	// key is the assets.LoadImageData key, not a path. Same indirection as everywhere
	// else: a file can be refiled without touching this.
	key string
	// size is the glyph's size on screen. It must divide artCanvas exactly.
	size int
}

// artCanvas is the size the drawn glyphs are authored at.
const artCanvas = 64

// categoryArtSize is the drawn category glyphs' size on a card: half of what they were
// authored at, which is what the owner asked to look at first. Change this and the card's
// left column moves — internal/cards/style.go measures with SizeOf and
// TestLeftColumnDoesNotCollide fails rather than letting the stack overlap.
const categoryArtSize = artCanvas / 2

var glyphArt = map[GlyphKind]glyphArtwork{
	GlyphAttack: {key: "shermansword_png", size: categoryArtSize},
	GlyphDefend: {key: "shermanshield_png", size: categoryArtSize},
}

// glyphCache holds rendered images, keyed by what was asked for. Building one walks a
// thousand pixels, which is nothing once and wasteful sixty times a second per card.
type glyphKey struct {
	kind    GlyphKind
	palette PaletteName
}

var glyphCache = map[glyphKey]*ebiten.Image{}

// Glyph returns the ebiten image for a glyph, building it on first use.
func Glyph(kind GlyphKind, name PaletteName) *ebiten.Image {
	key := glyphKey{kind, name}
	if img, ok := glyphCache[key]; ok {
		return img
	}

	img := ebiten.NewImageFromImage(RenderGlyph(kind, name))
	glyphCache[key] = img
	return img
}

// RenderGlyph draws a glyph into a plain Go image.
//
// Deliberately free of ebiten: creating an *ebiten.Image needs a graphics context, and the
// contact-sheet tool in cmd/glyphsheet has no window. Keeping the renderer pure is what
// lets the art be inspected without launching the game.
func RenderGlyph(kind GlyphKind, name PaletteName) *image.RGBA {
	if art, ok := glyphArt[kind]; ok {
		return renderArt(kind, art)
	}

	pal := PaletteOf(name)
	shp := glyphShapes[kind]
	size := shp.canvas()
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	mask := map[image.Point]bool{}
	for y, spans := range shp.fill {
		for _, s := range spans {
			for x := s.x0; x <= s.x1; x++ {
				mask[image.Pt(x, y)] = true
			}
		}
	}
	if len(mask) == 0 {
		return img
	}

	top, bottom := verticalExtent(mask)

	for p := range mask {
		// The rim is derived, never drawn, so it is exactly one pixel everywhere and
		// stays correct when the silhouette changes.
		if isEdge(p, mask) {
			img.Set(p.X, p.Y, pal.Outline)
			continue
		}

		t := rowPosition(p, mask)
		v := 0.0
		if bottom > top {
			v = float64(p.Y-top) / float64(bottom-top)
		}

		// Light from the upper left. Both thresholds drift left as the shape descends,
		// which is what rounds it rather than making it look like a flat gradient.
		c := pal.Shade
		switch {
		case t < 0.24 && v < 0.26:
			c = pal.Specular
		case t < 0.26-0.12*v:
			c = pal.Highlight
		case t < 0.66-0.12*v:
			c = pal.Mid
		}
		img.Set(p.X, p.Y, c)
	}

	// Accent last, and only inside the fill, so detail never punches through the rim.
	for y, spans := range shp.accent {
		for _, s := range spans {
			for x := s.x0; x <= s.x1; x++ {
				p := image.Pt(x, y)
				if mask[p] && !isEdge(p, mask) {
					img.Set(x, y, pal.Accent)
				}
			}
		}
	}

	return img
}

// artCache holds decoded and downsampled artwork. Decoding a PNG and averaging it down is
// far more work than walking a silhouette, and neither depends on anything that changes.
var artCache = map[GlyphKind]*image.RGBA{}

// renderArt decodes a drawn glyph and halves it to the size the card wants.
//
// A decode failure is fatal rather than silent. The bytes are embedded in the binary, so
// there is no runtime condition under which this can fail on one machine and not another —
// it means the file is not a PNG, which is a build problem to fix and not a case to fall
// back from with a blank corner on every attack card.
func renderArt(kind GlyphKind, art glyphArtwork) *image.RGBA {
	if img, ok := artCache[kind]; ok {
		return img
	}

	data := assets.LoadImageData()[art.key]
	if len(data) == 0 {
		log.Fatalf("glyph art %q is not in assets.LoadImageData", art.key)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("failed to decode glyph art %q: %v", art.key, err)
	}

	// Into an RGBA of its own first: PNGs with alpha decode to NRGBA, and the averaging
	// below is only correct on premultiplied values, which image.RGBA is by definition.
	b := src.Bounds()
	full := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(full, full.Bounds(), src, b.Min, draw.Src)

	if b.Dx() != artCanvas || b.Dy() != artCanvas {
		log.Fatalf("glyph art %q is %dx%d, want %dx%d", art.key, b.Dx(), b.Dy(), artCanvas, artCanvas)
	}
	if art.size <= 0 || artCanvas%art.size != 0 {
		log.Fatalf("glyph art %q asks for %dpx, which does not divide the %dpx canvas",
			art.key, art.size, artCanvas)
	}

	img := downsample(full, artCanvas/art.size)
	artCache[kind] = img
	return img
}

// downsample averages each factor x factor block down to one pixel.
//
// **Averaged, not sampled.** Nearest-neighbour at half size keeps every other pixel and
// throws the rest away, which deletes a one-pixel outline wherever it lands on an odd
// column — a drawn shield loses half its rim and reads as torn. Averaging keeps it as a
// darker pixel. This is the opposite of the rule for generated glyphs, and the difference
// is that a painting has interior detail to average whereas a derived rim has nothing
// behind it.
//
// image.RGBA is alpha-premultiplied, so the four channels average independently and a
// transparent neighbour correctly darkens nothing.
func downsample(src *image.RGBA, factor int) *image.RGBA {
	if factor <= 1 {
		return src
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()/factor, b.Dy()/factor))
	n := factor * factor

	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			var r, g, bl, a int
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					c := src.RGBAAt(b.Min.X+x*factor+dx, b.Min.Y+y*factor+dy)
					r += int(c.R)
					g += int(c.G)
					bl += int(c.B)
					a += int(c.A)
				}
			}
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(bl / n), A: uint8(a / n),
			})
		}
	}
	return out
}

// SizeOf is the size a glyph is drawn at, which is not the same for every glyph.
//
// The damage sword and the runner are 64. The category glyphs are much smaller, because
// they sit above the cost dashes in a column that a second 64-pixel badge would fill — 22
// for the generated book, 32 for the drawn sword and shield. Callers must measure with this
// rather than assuming GlyphSize, or a small glyph will be given a large hole to sit in.
func SizeOf(kind GlyphKind) int {
	if art, ok := glyphArt[kind]; ok {
		return art.size
	}
	return glyphShapes[kind].canvas()
}

// isEdge reports whether a filled pixel touches empty space on any side.
func isEdge(p image.Point, mask map[image.Point]bool) bool {
	return !mask[image.Pt(p.X-1, p.Y)] ||
		!mask[image.Pt(p.X+1, p.Y)] ||
		!mask[image.Pt(p.X, p.Y-1)] ||
		!mask[image.Pt(p.X, p.Y+1)]
}

// rowPosition is how far across its own row a pixel sits, 0 at the left edge and 1 at the
// right. Measured per row rather than across the sprite so a narrow blade gets the same
// rounding as a wide crossguard instead of being uniformly lit.
func rowPosition(p image.Point, mask map[image.Point]bool) float64 {
	left, right := p.X, p.X
	for mask[image.Pt(left-1, p.Y)] {
		left--
	}
	for mask[image.Pt(right+1, p.Y)] {
		right++
	}
	if right == left {
		return 0
	}
	return float64(p.X-left) / float64(right-left)
}

// verticalExtent is the first and last occupied row.
func verticalExtent(mask map[image.Point]bool) (top, bottom int) {
	first := true
	for p := range mask {
		if first || p.Y < top {
			top = p.Y
		}
		if first || p.Y > bottom {
			bottom = p.Y
		}
		first = false
	}
	return top, bottom
}
