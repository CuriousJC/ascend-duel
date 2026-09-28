package systems

// prose.go — the prose glyphs: the reading set, set into a line glyph by glyph and drawn at any
// capital height.
//
// **One sheet, white under a thin black outline, and every color is a multiply.** An ink scales the
// white to itself and leaves the outline black, which is what lets one drawing serve a dark panel, a
// light card and every element's word. A material is the same multiply with a tile instead of a
// color. Nothing here knows what an ink means.
//
// **Sized by capital height in screen pixels**, not by a point size: the sheet is drawn at one size
// and reduced, so the height of an `H` is the one number that says how big a line is. The set was
// drawn for a capital as small as ten pixels.
//
// **It shares the figure set's grammar** — cells, contour columns, advances, a space that is pen
// travel — and its span arithmetic, so a line is laid out exactly as a figure is.

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/assets"
)

// proseFile is the set's JSON. The cells and the glyph records are the figure set's; the heights are
// the prose set's own.
type proseFile struct {
	figureSetFile
	CapHeight  int
	XHeight    int
	LineHeight int
}

// proseSet is the loaded set: the figure machinery over one sheet, and the prose metrics.
type proseSet struct {
	figureSet
	capHeight  int
	lineHeight int
	cut        map[rune]*ebiten.Image
}

var (
	drawnProse *proseSet
	proseTried bool
)

// prose returns the set, loading it on first use. **A set that will not load is nil**, and every
// caller then answers as though nothing were covered, so a line falls back to the font rather than
// drawing nothing.
func prose() *proseSet {
	if proseTried {
		return drawnProse
	}
	proseTried = true
	ps, err := loadProse()
	if err != nil {
		log.Printf("prose glyphs: %v", err)
	}
	drawnProse = ps
	return ps
}

func loadProse() (*proseSet, error) {
	raw, err := assets.ProseFile("prose-glyphs.json")
	if err != nil {
		return nil, err
	}
	var meta proseFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	ps := &proseSet{
		figureSet:  figureSet{meta: meta.figureSetFile, glyphs: map[rune]figureGlyph{}},
		capHeight:  meta.CapHeight,
		lineHeight: meta.LineHeight,
		cut:        map[rune]*ebiten.Image{},
	}
	for k, g := range meta.Glyphs {
		if r := []rune(k); len(r) == 1 {
			ps.glyphs[r[0]] = g
		}
	}
	png, err := assets.ProseFile("prose-glyphs.png")
	if err != nil {
		return nil, err
	}
	src, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, err
	}
	whole := ebiten.NewImageFromImage(src)
	for r, g := range ps.glyphs {
		ps.cut[r] = whole.SubImage(image.Rect(g.X, g.Y, g.X+g.W, g.Y+g.H)).(*ebiten.Image)
	}
	return ps, nil
}

// ProseCovers says whether every character of str has a prose glyph. A space is covered; an empty
// string is not.
func ProseCovers(str string) bool {
	ps := prose()
	if ps == nil || str == "" {
		return false
	}
	for _, r := range str {
		if r == ' ' {
			continue
		}
		if _, ok := ps.glyphs[r]; !ok {
			return false
		}
	}
	return true
}

// MeasureProse is the width of str set with capitals `capHeight` pixels tall, from the first glyph's
// outline to the last one's.
func MeasureProse(str string, capHeight float64) float64 {
	ps := prose()
	if ps == nil || ps.capHeight <= 0 {
		return 0
	}
	w, _ := ps.span(str)
	return float64(w) * capHeight / float64(ps.capHeight)
}

// ProseLineHeight is the distance from one baseline to the next for capitals `capHeight` tall.
func ProseLineHeight(capHeight float64) float64 {
	ps := prose()
	if ps == nil || ps.capHeight <= 0 {
		return capHeight * 1.6
	}
	return float64(ps.lineHeight) * capHeight / float64(ps.capHeight)
}

// DrawProse sets str with its first outline at x and its baseline at y, capitals `capHeight` pixels
// tall. **A zero-alpha ink draws the sheet as it is** — white under black; any other multiplies the
// white by it and leaves the outline black.
func DrawProse(dst *ebiten.Image, str string, ink color.RGBA, x, y, capHeight float64) {
	ps := prose()
	if ps == nil || ps.capHeight <= 0 {
		return
	}
	_, pens := ps.span(str)
	s := capHeight / float64(ps.capHeight)
	i := 0
	for _, r := range str {
		g, ok := ps.glyphs[r]
		if !ok {
			continue
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(float64(pens[i]-g.Left), -float64(ps.meta.Baseline))
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(x, y)
		if ink.A > 0 {
			op.ColorScale.ScaleWithColor(ink)
		}
		dst.DrawImage(ps.cut[r], op)
		i++
	}
}

// proseAdvance is how far the pen travels over str, spaces included — what the next run on the line
// starts after. It differs from MeasureProse, which stops at the last outline.
func proseAdvance(str string, capHeight float64) float64 {
	ps := prose()
	if ps == nil || ps.capHeight <= 0 {
		return 0
	}
	pen := 0
	for _, r := range str {
		if r == ' ' {
			pen += ps.meta.SpaceAdvance
			continue
		}
		if g, ok := ps.glyphs[r]; ok {
			pen += g.Advance
		}
	}
	return float64(pen) * capHeight / float64(ps.capHeight)
}

// proseAscent is how far above the baseline the tallest glyph reaches, for capitals `capHeight`
// tall — where a line's top edge is, when a caller lays lines out by their tops.
func proseAscent(capHeight float64) float64 {
	ps := prose()
	if ps == nil || ps.capHeight <= 0 {
		return capHeight
	}
	// The ascenders reach a little above the capitals; the sheet's own figure for it is the space
	// between the cell's top and the baseline, less the margin every glyph keeps.
	return float64(ps.meta.Baseline-proseCellMargin) * capHeight / float64(ps.capHeight)
}

// proseCellMargin is the air every glyph keeps from its cell's edges, in sheet pixels.
const proseCellMargin = 4
