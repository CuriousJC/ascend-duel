package systems

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Procedurally generated pixel-art glyphs for the numbers on an action card: what it hits
// for, and what it costs.
//
// Drawn in code rather than loaded from a file because generated art has no provenance
// question at all, which is exactly the problem that makes the Tyrian set a release
// blocker. That is the reason to keep doing interface art this way.
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
const (
	// GlyphDamage is a sword: what the action hits for.
	GlyphDamage GlyphKind = iota
	// GlyphActionPoints is a runner: what it costs to play.
	GlyphActionPoints
)

// GlyphKinds is every glyph, in a fixed order. The contact sheet walks this rather than
// ranging a map, which Go deliberately randomises.
func GlyphKinds() []GlyphKind {
	return []GlyphKind{GlyphDamage, GlyphActionPoints}
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
type shape struct {
	fill   map[int][]span
	accent map[int][]span
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

var glyphShapes = map[GlyphKind]shape{
	GlyphDamage:       swordShape,
	GlyphActionPoints: runShape,
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
	pal := PaletteOf(name)
	shp := glyphShapes[kind]
	img := image.NewRGBA(image.Rect(0, 0, GlyphSize, GlyphSize))

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
