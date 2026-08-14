package cards

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Text, drawn with golang.org/x/image rather than with Ebitengine's text/v2.
//
// text/v2 draws into an *ebiten.Image and this package must not make one, so the
// typesetting has to come from somewhere else. That is not a compromise for the tool's
// benefit only: because the *game* also blits what this package renders, both get
// letter-for-letter the same output, and the contact sheet stays trustworthy. A card
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

// drawText draws a string with its top-left corner at (x, y).
//
// font.Drawer positions by *baseline*, not by the top of the line, so the ascent has to
// be added. Ebitengine's text/v2 anchors at the top-left by default, and every layout
// constant on a Style was written against that behaviour, so matching it here is what
// keeps the numbers meaning what they say.
func drawText(dst *image.RGBA, f *Faces, size float64, s string, x, y int, c color.RGBA) error {
	face, err := f.at(size)
	if err != nil {
		return err
	}
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y+face.Metrics().Ascent.Ceil()),
	}
	d.DrawString(s)
	return nil
}

// drawTextHCentered draws a string centred horizontally on the card, top-aligned at y.
//
// Rings use it. An action card's name lines up with the glyph column beneath it, so
// left-aligned is right there; a ring has no column, and the same name then reads as
// having slipped off centre rather than as being aligned to anything.
func drawTextHCentered(dst *image.RGBA, f *Faces, size float64, s string, width, y int, c color.RGBA) error {
	face, err := f.at(size)
	if err != nil {
		return err
	}
	w := font.MeasureString(face, s).Ceil()
	return drawText(dst, f, size, s, (width-w)/2, y, c)
}

// drawTextCenteredIn draws a string centred inside a box that starts at left and is width
// wide, top-aligned at y.
//
// The effect text uses it. drawTextHCentered centres on the whole card, which is right for a
// name spanning the card and wrong for a block that only owns the space beside the cost
// column — every line would sit a little left of where it belongs.
func drawTextCenteredIn(dst *image.RGBA, f *Faces, size float64, s string, left, width, y int, c color.RGBA) error {
	face, err := f.at(size)
	if err != nil {
		return err
	}
	w := font.MeasureString(face, s).Ceil()
	return drawText(dst, f, size, s, left+(width-w)/2, y, c)
}

// drawTextRightAligned draws a string ending at x, top-aligned at y.
//
// The stat rows use it: a label against the left margin and a figure against the right, so
// the figures line up down the card as a column whatever their width. Left-aligning them
// under the labels would make "6" and "12 / 40" start together and end anywhere.
func drawTextRightAligned(dst *image.RGBA, f *Faces, size float64, s string, right, y int, c color.RGBA) error {
	face, err := f.at(size)
	if err != nil {
		return err
	}
	w := font.MeasureString(face, s).Ceil()
	return drawText(dst, f, size, s, right-w, y, c)
}

// TextWidth is how wide a string is at this size, in pixels.
//
// **Exported for the same reason WrapText is**: the wording lives in internal/screens and the
// column it has to fit lives here, so a test that holds one against the other needs both. It
// is what catches the case WrapText cannot report — a single word wider than the line, which
// wraps to one line and then overruns it.
func TextWidth(f *Faces, size float64, s string) (int, error) {
	face, err := f.at(size)
	if err != nil {
		return 0, err
	}
	return font.MeasureString(face, s).Ceil(), nil
}

// WrapText breaks a string into lines that each fit inside width pixels at this size.
//
// **Exported so a test can ask the question the card asks**: whether a piece of effect text
// fits the band it is drawn in. The wording lives in internal/screens and the geometry lives
// here, so neither package can answer that alone — see TestEveryCardTextFitsItsBand.
//
// It breaks on spaces only. A single word longer than the line is left whole and overruns,
// rather than being hyphenated at an arbitrary point: the strings on the cards are written
// here in the repo, so an overrun is an authoring mistake to fix and not a case to handle.
func WrapText(f *Faces, size float64, s string, width int) ([]string, error) {
	face, err := f.at(size)
	if err != nil {
		return nil, err
	}

	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if line != "" && font.MeasureString(face, try).Ceil() > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line = try
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines, nil
}
