package systems

// glyphs_plain.go — both glyph sheets, set into a plain Go image with no graphics context.
//
// **The card faces are drawn by `internal/cards`, which may never make an `*ebiten.Image`**, so
// the review sheets can render them with no window — the same reason `ArtMark` hands back a plain
// image. So the sheets are decoded here a second time as plain images, and a line is composited
// glyph by glyph with `x/image/draw`: the same cells, the same advances and the same span
// arithmetic the screen's drawing reads, so a word is laid out identically on a card and on the
// table.
//
// **The same two rules as the screen's drawing**: a line the sheet does not cover is not drawn here
// — the caller asks Covers first and keeps its font for the rest — and a dark gray ink draws the
// sheet as it is, white under its outline; see drawsSheetAsIs.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"

	"github.com/curiousjc/ascend-duel/assets"
)

// GlyphSheet is one sheet of either set, decoded for plain-Go drawing.
type GlyphSheet struct {
	set  figureSet // metrics and glyph records; no Ebitengine images
	img  image.Image
	unit int // the sheet pixels a height is measured against: the capital, or the digit

	// darkBelow is the luminance, out of 255, under which an ink draws the sheet as it is.
	darkBelow float64
}

var (
	plainProse      *GlyphSheet
	plainProseTried bool
	plainFigures    = map[string]*GlyphSheet{}
)

// ProseGlyphs is the prose sheet for plain-Go drawing, or nil when it will not load.
func ProseGlyphs() *GlyphSheet {
	if plainProseTried {
		return plainProse
	}
	plainProseTried = true
	gs, err := loadPlainProse()
	if err != nil {
		log.Printf("prose glyphs, plain: %v", err)
	}
	plainProse = gs
	return gs
}

func loadPlainProse() (*GlyphSheet, error) {
	raw, err := assets.ProseFile("prose-glyphs.json")
	if err != nil {
		return nil, err
	}
	var meta proseFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	png, err := assets.ProseFile("prose-glyphs.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, err
	}
	return &GlyphSheet{set: glyphRecords(meta.figureSetFile), img: img, unit: meta.CapHeight,
		darkBelow: 128}, nil
}

// FigureGlyphs is one sheet of the figure set for plain-Go drawing — FigureNeutral for a line
// multiplied by its own ink — or nil when it will not load.
func FigureGlyphs(sheet string) *GlyphSheet {
	if gs, ok := plainFigures[sheet]; ok {
		return gs
	}
	gs, err := loadPlainFigure(sheet)
	if err != nil {
		log.Printf("figure glyphs %q, plain: %v", sheet, err)
	}
	plainFigures[sheet] = gs
	return gs
}

func loadPlainFigure(sheet string) (*GlyphSheet, error) {
	name := DefaultFigureSet
	if env := os.Getenv(FigureSetEnv); env != "" {
		if _, err := assets.FigureFile(env, "figure-glyphs.json"); err == nil {
			name = env
		}
	}
	raw, err := assets.FigureFile(name, "figure-glyphs.json")
	if err != nil {
		return nil, err
	}
	var meta figureSetFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	sh, ok := meta.Sheets[sheet]
	if !ok {
		return nil, fmt.Errorf("set %q has no sheet %q", name, sheet)
	}
	png, err := assets.FigureFile(name, sh.Image)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, err
	}
	return &GlyphSheet{set: glyphRecords(meta), img: img, unit: meta.DigitHeight,
		darkBelow: figureUntinted * 255}, nil
}

// glyphRecords is a set's metrics and glyph table with no images attached.
func glyphRecords(meta figureSetFile) figureSet {
	fs := figureSet{meta: meta, glyphs: map[rune]figureGlyph{}}
	for k, g := range meta.Glyphs {
		if r := []rune(k); len(r) == 1 {
			fs.glyphs[r[0]] = g
		}
	}
	return fs
}

// Covers says whether every character of s has a glyph on this sheet. A space is covered when the
// set declares one; an empty string is not.
func (g *GlyphSheet) Covers(s string) bool {
	if g == nil || s == "" || g.unit <= 0 {
		return false
	}
	for _, r := range s {
		if r == ' ' {
			if g.set.meta.SpaceAdvance == 0 {
				return false
			}
			continue
		}
		if _, ok := g.set.glyphs[r]; !ok {
			return false
		}
	}
	return true
}

// Width is s from the first glyph's outline to the last one's, at a unit `height` pixels tall.
func (g *GlyphSheet) Width(s string, height float64) float64 {
	w, _ := g.set.span(s)
	return float64(w) * height / float64(g.unit)
}

// Advance is how far the pen travels over s, spaces included — where a following run starts.
func (g *GlyphSheet) Advance(s string, height float64) float64 {
	pen := 0
	for _, r := range s {
		if r == ' ' {
			pen += g.set.meta.SpaceAdvance
			continue
		}
		if gl, ok := g.set.glyphs[r]; ok {
			pen += gl.Advance
		}
	}
	return float64(pen) * height / float64(g.unit)
}

// Draw sets s into dst with its first outline at x and its baseline at y, the unit `height` pixels
// tall. A zero or dark ink draws the sheet as it is; any other multiplies it.
func (g *GlyphSheet) Draw(dst *image.RGBA, s string, ink color.RGBA, x, y, height float64) {
	if g == nil || g.unit <= 0 {
		return
	}
	if ink.A == 0 || drawsSheetAsIs(ink, g.darkBelow) {
		ink = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	scale := height / float64(g.unit)
	_, pens := g.set.span(s)
	i := 0
	for _, r := range s {
		gl, ok := g.set.glyphs[r]
		if !ok {
			continue
		}
		// The cell's top-left on the destination: the pen, less the contour's own left edge, and
		// the cell's baseline on the line's.
		ox := x + float64(pens[i]-gl.Left)*scale
		oy := y - float64(g.set.meta.Baseline)*scale
		g.drawGlyph(dst, gl, ink, ox, oy, scale)
		i++
	}
}

// drawGlyph resamples one cell into a scratch image the size of its footprint, then lays it over
// dst multiplied by the ink.
func (g *GlyphSheet) drawGlyph(dst *image.RGBA, gl figureGlyph, ink color.RGBA, ox, oy, scale float64) {
	x0, y0 := int(math.Floor(ox)), int(math.Floor(oy))
	x1 := int(math.Ceil(ox + float64(gl.W)*scale))
	y1 := int(math.Ceil(oy + float64(gl.H)*scale))
	foot := image.Rect(x0, y0, x1, y1).Intersect(dst.Bounds())
	if foot.Empty() {
		return
	}
	tmp := image.NewRGBA(foot)
	// Source to destination: scale the cell, then move it to its fractional place.
	aff := f64.Aff3{
		scale, 0, ox - float64(gl.X)*scale,
		0, scale, oy - float64(gl.Y)*scale,
	}
	src := image.Rect(gl.X, gl.Y, gl.X+gl.W, gl.Y+gl.H)
	draw.CatmullRom.Transform(tmp, aff, g.img, src, draw.Src, nil)

	for py := foot.Min.Y; py < foot.Max.Y; py++ {
		for px := foot.Min.X; px < foot.Max.X; px++ {
			si := tmp.PixOffset(px, py)
			sa := uint32(tmp.Pix[si+3]) * uint32(ink.A) / 255
			if sa == 0 {
				continue
			}
			sr := uint32(tmp.Pix[si+0]) * uint32(ink.R) / 255 * uint32(ink.A) / 255
			sg := uint32(tmp.Pix[si+1]) * uint32(ink.G) / 255 * uint32(ink.A) / 255
			sb := uint32(tmp.Pix[si+2]) * uint32(ink.B) / 255 * uint32(ink.A) / 255
			di := dst.PixOffset(px, py)
			inv := 255 - sa
			dst.Pix[di+0] = uint8(sr + uint32(dst.Pix[di+0])*inv/255)
			dst.Pix[di+1] = uint8(sg + uint32(dst.Pix[di+1])*inv/255)
			dst.Pix[di+2] = uint8(sb + uint32(dst.Pix[di+2])*inv/255)
			dst.Pix[di+3] = uint8(sa + uint32(dst.Pix[di+3])*inv/255)
		}
	}
}
