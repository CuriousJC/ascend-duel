package systems

// prose_line.go — a line of reading text, set in the prose glyphs wherever the set covers it.
//
// **DrawText takes exactly what text.Draw takes**, so moving a line onto the glyphs is renaming one
// call: the same face says how big it is, the same options say where it goes, how it is aligned and
// what ink it is in. A line with a character the set does not carry is drawn in the font instead,
// whole, so no line is half one typeface and half the other.
//
// **The face's size is read as a tier and turned into a capital height** by ProseCap. A line keeps
// its tier; only how it is drawn changes. A heading set larger than the tiers takes the same sheet
// scaled with them.
//
// **A dark gray ink draws the sheet as it is** — white under its outline — the rule the figure
// sheet follows too. Multiplied into the white, a dark gray merges letter and outline into one heavy
// shape, and a line on a painted backdrop would vanish into it. See drawsSheetAsIs.

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The capital heights the three text tiers are set at in the prose glyphs, in screen pixels. The set
// was drawn for a capital as small as proseCapFloor.
const (
	proseCapSmall  = 10
	proseCapMedium = 12
	proseCapLarge  = 14
	proseCapFloor  = 10
)

// ProseCap is the capital height a line set at font `size` takes in the prose glyphs. The three
// tiers answer their own figure; a size off the tiers scales with the Large one and never drops
// under the floor the set was drawn for.
func ProseCap(size float64) float64 {
	switch size {
	case TextSmall:
		return proseCapSmall
	case TextMedium:
		return proseCapMedium
	case TextLarge:
		return proseCapLarge
	}
	c := size * proseCapLarge / TextLarge
	if c < proseCapFloor {
		c = proseCapFloor
	}
	return c
}

// ProseCapOf is the capital height a line at font `size` takes in proportion, with no tier and no
// floor — for a card face, where a small copy of a card carries small type on purpose.
func ProseCapOf(size float64) float64 { return size * proseCapLarge / TextLarge }

// MeasureText is how far the pen travels over s as DrawText would set it: in the prose glyphs when
// the set covers it, in the face otherwise. It is the measure to lay out beside a DrawText call.
func MeasureText(s string, face *text.GoTextFace) float64 {
	if setsInProse(s, face) {
		return proseAdvance(s, ProseCap(face.Size))
	}
	w, _ := text.Measure(s, face, 0)
	return w
}

// DrawText draws s as text.Draw would — the options' translation is the anchor, its alignments place
// the line against it, its color scale is the ink — in the prose glyphs when the set covers s.
func DrawText(dst *ebiten.Image, s string, face *text.GoTextFace, op *text.DrawOptions) {
	if !setsInProse(s, face) {
		text.Draw(dst, s, face, op)
		return
	}
	capHeight := ProseCap(face.Size)
	ax, ay := op.GeoM.Apply(0, 0)

	x := ax
	switch w := MeasureProse(s, capHeight); op.PrimaryAlign {
	case text.AlignCenter:
		x -= w / 2
	case text.AlignEnd:
		x -= w
	}

	// **The baseline lands where the font's would have**, so a line moved onto the glyphs stays on
	// the line it was laid out on.
	m := face.Metrics()
	y := ay + m.HAscent
	switch op.SecondaryAlign {
	case text.AlignCenter:
		y = ay - (m.HAscent+m.HDescent)/2 + m.HAscent
	case text.AlignEnd:
		y = ay - m.HDescent
	}

	cs := op.ColorScale
	ink := color.RGBA{R: unit(cs.R()), G: unit(cs.G()), B: unit(cs.B()), A: unit(cs.A())}
	DrawProseInk(dst, s, ink, x, y, capHeight)
}

// setsInProse says whether DrawText sets s in the glyphs: every character covered.
func setsInProse(s string, _ *text.GoTextFace) bool {
	return ProseCovers(s)
}

// DrawProseInk is DrawProse choosing the drawing by the ink: a light ink multiplies the sheet, a
// dark one draws it as it is.
func DrawProseInk(dst *ebiten.Image, s string, ink color.RGBA, x, y, capHeight float64) {
	if inkIsDark(ink) {
		ink = color.RGBA{}
	}
	DrawProse(dst, s, ink, x, y, capHeight)
}

// inkIsDark says whether an ink is too dark to multiply into the prose sheet.
func inkIsDark(c color.RGBA) bool { return drawsSheetAsIs(c, 128) }

// drawsSheetAsIs says whether an ink is a dark gray — under `below` luminance, out of 255, and
// carrying almost no hue — which draws a glyph sheet as it is rather than multiplying into it.
//
// **Only a gray.** A dark gray multiplied in merges letter and outline into one heavy shape and
// says nothing the white would not; a dark *color* — arcane's purple — is the word's meaning, and
// drawn white it would be a colored word that lost its color.
func drawsSheetAsIs(c color.RGBA, below float64) bool {
	if c.A == 0 {
		return false
	}
	hi := max(c.R, c.G, c.B)
	lo := min(c.R, c.G, c.B)
	return 0.299*float64(c.R)+0.587*float64(c.G)+0.114*float64(c.B) < below && hi-lo < inkGrayChroma
}

// inkGrayChroma is the most an ink's channels may spread and still be a gray.
const inkGrayChroma = 48

// unit turns a color-scale channel back into a byte, clamped.
func unit(v float32) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

// WrapLine is WrapRuns for a line DrawLine will set: wrapped against the prose glyphs when they
// cover it, against the face otherwise.
func WrapLine(line models.TipLine, face *text.GoTextFace, max float64) []models.TipLine {
	if lineInProse(line, face) {
		return WrapProse(line, ProseCap(face.Size), max)
	}
	return WrapRuns(line, face, max)
}

// DrawLine is DrawRuns in the prose glyphs when they cover the line: the runs keep their inks and
// materials, exactly as a tooltip's do, and the baseline lands where the face's would have. y is the
// top of the line, as DrawRuns takes it.
func DrawLine(screen *ebiten.Image, line models.TipLine, face *text.GoTextFace, x, y int, plain color.RGBA) {
	if !lineInProse(line, face) {
		DrawRuns(screen, line, face, x, y, plain)
		return
	}
	drawProseRuns(screen, line, ProseCap(face.Size), float64(x), float64(y)+face.Metrics().HAscent, plain)
}

// lineInProse says whether a line of runs is set in the glyphs: every run covered.
func lineInProse(line models.TipLine, _ *text.GoTextFace) bool {
	for _, run := range line {
		if run.Text != "" && !ProseCovers(run.Text) {
			return false
		}
	}
	return true
}
