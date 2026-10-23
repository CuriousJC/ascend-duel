package cards

// orbs.go — a figure drawn as an orb: a glossy sphere with the figure written across its middle in
// white, in the interface's own lettering, and no label. The orb's picture is what says what the
// figure is — a fighter's stat, or a playing card's damage in the card's own element — so a card
// laid out in orbs carries no words for its figures at all.

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// StatOrb is one stat row drawn as an orb: the picture's asset key, its center on the card, and
// the side of the square it is drawn in. A zero Size draws the row as a row.
type StatOrb struct {
	Art  string
	X, Y int
	Size int
}

// The orbs the fighter cards draw, by asset key. **The pairing of figure to picture is the cards'**,
// and the two fighters share theirs: DMG is amber and AP lapis on both, so the same figure on the
// two sides of the table is the same object.
const (
	OrbDMG   = "orb-amber"
	OrbAP    = "orb-lapis"
	OrbVitae = "orb-vitae"
	OrbRealm = "orb-rainbow"
)

// DamageOrbArtKey is the orb a playing card's damage is written on: the card's own element's, and
// the DMG orb for a card with no element of the five.
func DamageOrbArtKey(e Element) string {
	if !hasArt(e) {
		return OrbDMG
	}
	return "orb-" + e.String()
}

// The figure inside an orb: its size against the orb's, and the widest share of the orb it may take.
const (
	orbFigureShare = 0.64
	orbFigureFit   = 0.80
)

// orbFigureInk is white, which draws the figure sheet as it was drawn: white under its outline.
var orbFigureInk = color.RGBA{R: 255, G: 255, B: 255, A: 255}

// StatOrbBox is the square stat row i is drawn in as an orb, and whether the style draws one there.
// **The screens aim at a stat through this**, so a figure flying to the purse lands on the orb the
// card actually drew.
func StatOrbBox(st Style, i int) (image.Rectangle, bool) {
	if i < 0 || i >= len(st.StatOrbs) || st.StatOrbs[i].Size <= 0 {
		return image.Rectangle{}, false
	}
	o := st.StatOrbs[i]
	return image.Rect(o.X-o.Size/2, o.Y-o.Size/2, o.X-o.Size/2+o.Size, o.Y-o.Size/2+o.Size), true
}

// hasStatOrbs reports whether any stat row on this style is an orb.
func (st Style) hasStatOrbs() bool {
	for _, o := range st.StatOrbs {
		if o.Size > 0 {
			return true
		}
	}
	return false
}

// drawStatOrb draws stat i as its orb, with its figure across the middle. **Every figure on an orb
// is white**, because a colored figure on a colored sphere is a figure the sphere eats.
func drawStatOrb(dst *image.RGBA, s Spec, st Style, f *Faces, i int, line StatLine) error {
	box, _ := StatOrbBox(st, i)
	o := st.StatOrbs[i]
	return drawOrb(dst, s, st, f, o.Art, line.Value, box)
}

// drawDamageOrb draws a playing card's damage as the orb of its element, centered on the badge's
// slot, with the card's multiplier across it as a decimal — see orbDecimal.
func drawDamageOrb(dst *image.RGBA, s Spec, st Style, f *Faces) error {
	cx, cy := st.GlyphInset+st.BadgeSize/2, st.BadgeTop+st.BadgeSize/2
	n := st.BadgeOrbSize
	box := image.Rect(cx-n/2, cy-n/2, cx-n/2+n, cy-n/2+n)
	return drawOrb(dst, s, st, f, DamageOrbArtKey(s.Element), orbDecimal(s.BadgePct, s.Badge), box)
}

// orbDecimal is a multiplier written for an orb: as a decimal, 0.5 and 4.5 and 2, on the player's
// cards and the enemy's alike. **Never a fraction** *(owner's call)*: a stacked fraction at orb size
// reads worse than the decimal it stands for. A card with no multiplier keeps its badge's words.
func orbDecimal(pct int, badge string) string {
	if pct <= 0 {
		return badge
	}
	return strconv.FormatFloat(float64(pct)/100, 'f', -1, 64)
}

// drawOrb draws the orb `art` in `box` with `figure` across its middle. **The figure fades with the
// card**: a disabled card's orb walks toward the surface, and a figure at full strength over it
// would be the one thing on the card still saying it can be played.
func drawOrb(dst *image.RGBA, s Spec, st Style, f *Faces, art, figure string, box image.Rectangle) error {
	size := box.Dx()
	fade := glyphFade(s)
	if glyph := systems.ArtMark(art, size, size); glyph != nil {
		blitGlyph(dst, box, glyph, 1, fade, st)
	}
	ink := orbFigureInk
	ink.A = uint8(255 * (100 - fade) / 100)
	cx, cy := float64(box.Min.X+box.Max.X)/2, float64(box.Min.Y+box.Max.Y)/2
	return drawOrbDigits(dst, f, figure, cx, cy, float64(size)*orbFigureShare, float64(size)*orbFigureFit, ink)
}

// drawOrbDigits writes one run of figure lettering centered on (cx, cy), at `size`, set smaller if
// it would be wider than `fit`.
func drawOrbDigits(dst *image.RGBA, f *Faces, s string, cx, cy, size, fit float64, ink color.RGBA) error {
	if s == "" {
		return nil
	}
	width, _, err := span(uiLetters, f, size, s)
	if err != nil {
		return err
	}
	if float64(width) > fit {
		size *= fit / float64(width)
		if width, _, err = span(uiLetters, f, size, s); err != nil {
			return err
		}
	}
	_, lineHeight, err := f.Measure(size, s)
	if err != nil {
		return err
	}
	return drawTextAs(uiLetters, dst, f, size, s, int(math.Round(cx))-width/2, int(math.Round(cy))-lineHeight/2, ink)
}
