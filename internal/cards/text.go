package cards

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/systems"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Text, set from the two glyph sheets as plain images, rather than with Ebitengine's text/v2.
//
// text/v2 draws into an *ebiten.Image and this package must not make one, so the
// typesetting has to come from somewhere else: `systems.GlyphSheet` composites the same sheets the
// screen draws, and golang.org/x/image's font drawing is the fallback for a character they lack.
// That is not a compromise for the tool's benefit only: because the *game* also blits what this
// package renders, both get letter-for-letter the same output, and the contact sheet stays
// trustworthy. A card
// whose text was set by one library in the sheet and another in the game would look
// right in review and wrong in play.

// Faces holds a parsed font and the sizes it has been asked for. Building a face is
// expensive enough to be worth keeping, and Render asks for the same three or four sizes
// for every card it draws.
//
// **Not safe for concurrent use.** A font.Face carries a mutable glyph cache, and
// nothing here renders in parallel: the game builds cards on a cache miss during Update,
// and the tool runs a single loop.
type Faces struct {
	font  *opentype.Font
	sizes map[float64]font.Face
}

// NewFaces parses a TrueType or OpenType font for use by Render. The bytes come from the
// assets package, which is the only place a font file is embedded — this package does not
// embed its own copy, or the game and the sheet could be set in different fonts.
func NewFaces(ttf []byte) (*Faces, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("cards: parsing the font: %w", err)
	}
	return &Faces{font: f, sizes: map[float64]font.Face{}}, nil
}

// at returns a face at the given pixel size.
//
// DPI is 72 so that one point is one pixel, which makes the sizes here mean the same
// thing as the sizes in the screen's existing card code. Anything else would silently
// rescale every label on the card.
func (f *Faces) at(size float64) (font.Face, error) {
	if face, ok := f.sizes[size]; ok {
		return face, nil
	}
	face, err := opentype.NewFace(f.font, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("cards: making a %gpx face: %w", size, err)
	}
	f.sizes[size] = face
	return face, nil
}

// lettering is which glyph sheet a line on a card is set from. **Every line is one of two**: prose
// for what is read — a name, a stat's label and figure, the effect text — and the interface's own
// lettering for a readout — the life fraction, a badge's numeral, a relic's counter. The font is only
// what a line falls back to when its sheet lacks a character, whole, so no line is half one typeface
// and half the other.
type lettering int

const (
	prose lettering = iota
	uiLetters
)

// sheet is the plain-Go glyph sheet a lettering is drawn from, and the height of its unit — the
// capital, or the digit — for a line at font size `size`.
func (k lettering) sheet(size float64) (*systems.GlyphSheet, float64) {
	if k == uiLetters {
		return systems.FigureGlyphs(systems.FigureNeutral), systems.UIHeightOf(size)
	}
	return systems.ProseGlyphs(), systems.ProseCapOf(size)
}

// span is how wide s is set as lettering k — from the first outline to the last — and how far the
// pen travels over it, which is where a following run starts. In the font the two are the same.
func span(k lettering, f *Faces, size float64, s string) (width, advance int, err error) {
	if g, h := k.sheet(size); g.Covers(s) {
		return int(math.Ceil(g.Width(s, h))), int(math.Ceil(g.Advance(s, h))), nil
	}
	face, err := f.at(size)
	if err != nil {
		return 0, 0, err
	}
	w := font.MeasureString(face, s).Ceil()
	return w, w, nil
}

// Measure is how wide a string is at a point size, and how tall one line of it is.
//
// **Exported so a caller outside this package can ask whether something fits** without being
// handed a font.Face and left to get the DPI right. `internal/screens` uses it to check the relic
// names in `data/relics.json` against the room a relic card leaves above its artwork — the file is
// read there and the geometry lives here, so the join needs one of the two to be askable.
//
// **The width is the prose glyphs' when they cover the string**, as the card draws it. The line
// height stays the font's: every layout constant on a Style was written against it.
func (f *Faces) Measure(size float64, s string) (width, lineHeight int, err error) {
	face, err := f.at(size)
	if err != nil {
		return 0, 0, err
	}
	m := face.Metrics()
	width, _, err = span(prose, f, size, s)
	return width, m.Ascent.Ceil() + m.Descent.Ceil(), err
}

// drawText draws a string in the prose glyphs with its top-left corner at (x, y).
//
// **The baseline lands where the font's would have** — y plus the face's ascent — because every
// layout constant on a Style was written against a top-left anchor, which is what Ebitengine's
// text/v2 used and what the font drawing here matched.
func drawText(dst *image.RGBA, f *Faces, size float64, s string, x, y int, c color.RGBA) error {
	return drawTextAs(prose, dst, f, size, s, x, y, c)
}

// drawTextAs is drawText in either lettering.
func drawTextAs(k lettering, dst *image.RGBA, f *Faces, size float64, s string, x, y int, c color.RGBA) error {
	face, err := f.at(size)
	if err != nil {
		return err
	}
	baseline := y + face.Metrics().Ascent.Ceil()
	if g, h := k.sheet(size); g.Covers(s) {
		g.Draw(dst, s, c, float64(x), float64(baseline), h)
		return nil
	}
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(s)
	return nil
}

// drawTextHCentered draws a string centered horizontally on the card, top-aligned at y.
//
// Relics use it. An action card's name lines up with the glyph column beneath it, so
// left-aligned is right there; a relic has no column, and the same name then reads as
// having slipped off center rather than as being aligned to anything.
func drawTextHCentered(dst *image.RGBA, f *Faces, size float64, s string, width, y int, c color.RGBA) error {
	return drawTextHCenteredAs(prose, dst, f, size, s, width, y, c)
}

// drawTextHCenteredAs is drawTextHCentered in either lettering.
func drawTextHCenteredAs(k lettering, dst *image.RGBA, f *Faces, size float64, s string, width, y int, c color.RGBA) error {
	w, _, err := span(k, f, size, s)
	if err != nil {
		return err
	}
	return drawTextAs(k, dst, f, size, s, (width-w)/2, y, c)
}

// drawTextRightAligned draws a string ending at x, top-aligned at y.
//
// The stat rows use it: a label against the left margin and a figure against the right, so
// the figures line up down the card as a column whatever their width. Left-aligning them
// under the labels would make "6" and "12 / 40" start together and end anywhere.
func drawTextRightAligned(dst *image.RGBA, f *Faces, size float64, s string, right, y int, c color.RGBA) error {
	w, _, err := span(prose, f, size, s)
	if err != nil {
		return err
	}
	return drawText(dst, f, size, s, right-w, y, c)
}

// TextWidth is how far a string carries the pen at this size, in pixels, as the card sets it.
//
// **Exported for the same reason WrapText is**: the wording lives in internal/screens and the
// column it has to fit lives here, so a test that holds one against the other needs both. It
// is what catches the case WrapText cannot report — a single word wider than the line, which
// wraps to one line and then overruns it.
func TextWidth(f *Faces, size float64, s string) (int, error) {
	_, advance, err := span(prose, f, size, s)
	return advance, err
}

// WrapText breaks a string into lines that each fit inside width pixels at this size.
//
// **Exported so a test can ask the question the card asks**: whether a piece of effect text
// fits the band it is drawn in. The wording lives in internal/screens and the geometry lives
// here, so neither package can answer that alone — see TestEveryCardTextFitsItsBand.
//
// **It breaks at every space** *(owner's call, 2026-09-05)*, one word to a line — except that a
// figure stays on its unit's line, `-1 AP` drawn as `AP -1`; see unitLines. Which is what a
// relic's name already does. Width no longer decides where a line ends, so a *set* of cards breaks
// in the same place for free: the elemental essences differ only in the color they name, and FIRE
// used to fit the line where LIGHTNING all but filled it, which made four layouts of one card.
// That is what the authored newline was for, and it is why there is no longer one — a newline in
// the source now reads as an ordinary space.
//
// A single word longer than the line is still left whole and overruns, rather than being
// hyphenated at an arbitrary point: the strings on the cards are written here in the repo, so an
// overrun is an authoring mistake to fix and not a case to handle.
func WrapText(f *Faces, size float64, s string, width int) ([]string, error) {
	face, err := f.at(size)
	if err != nil {
		return nil, err
	}

	// **One word to a line.** The face is still resolved above, because a word wider than the
	// column is the one thing this cannot fix and TestNoEffectTextWordIsWiderThanItsColumn is
	// what reports it.
	_, _ = face, width

	return unitLines(strings.Fields(strings.ReplaceAll(s, "\n", " "))), nil
}

// units are the words a figure belongs to. **A closed list, and closing it is the point**: the
// alternative is guessing from shape, and "CARD" beside "+1" would read as a unit as easily as
// "AP" does.
var units = map[string]bool{"AP": true, "DMG": true, "HP": true, "VITAE": true}

// unitLines is the word-per-line split with one exception: **a figure stays on its unit's line,
// and the unit leads** *(owner's call, 2026-09-05)*. "-1 AP" is drawn "AP -1" on one line, and
// "DMG +50%" stays as it is — a number alone on a line says nothing until the eye reaches the line
// under it, and the pair is one fact rather than two words.
//
// **The figure may sit on either side of its unit in the authored text**, because both read
// naturally in a sentence — "EASIER -1 AP" and "GAINS DMG +50%" — and neither ordering is the one
// drawn.
func unitLines(words []string) []string {
	var out []string
	for i := 0; i < len(words); i++ {
		word := words[i]

		if i+1 < len(words) {
			if units[word] && isFigure(words[i+1]) {
				out = append(out, word+" "+words[i+1])
				i++
				continue
			}
			if isFigure(word) && units[words[i+1]] {
				out = append(out, words[i+1]+" "+word)
				i++
				continue
			}
		}
		out = append(out, word)
	}
	return out
}

// isFigure reports whether a word is a number a unit could belong to: a digit somewhere, and
// nothing but a sign, digits, a point and a trailing % or x around it. **"+50%" and "-1" are
// figures and "EASIER" is not**, which is the whole question this answers.
func isFigure(word string) bool {
	digits := false
	for _, r := range word {
		switch {
		case r >= '0' && r <= '9':
			digits = true
		case r == '+' || r == '-' || r == '.' || r == '%' || r == 'x':
			// part of a figure
		default:
			return false
		}
	}
	return digits
}
