package systems

// figures.go — the figure glyphs: authored numerals and math symbols, set side by side into a
// figure and drawn at any height.
//
// **A figure is assembled here, glyph by glyph, off one sprite sheet per color.** The art is a
// set under `assets/figure/<set>/`: a sheet per color on a fixed grid, and a JSON file giving each
// glyph's cell, the columns its contour reaches and how far the pen moves after it. Nothing here
// knows what a color *means* — the caller names a sheet, and may multiply it by an ink.
//
// **The glyph list is closed and a string outside it is not drawn here.** `FigureCovers` is the
// question a caller asks first; a word such as MISS or a hand's name stays in the font.
//
// **Smooth art, linear filtering.** The sheets are drawn large and reduced to the height asked
// for, and Ebitengine takes a mipmap on the way down, so a figure keeps its contour at a third of
// the sheet's size.

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/assets"
)

// DefaultFigureSet is which set of figure glyphs the game draws — a directory under
// `assets/figure/` — unless FigureSetEnv names another.
const DefaultFigureSet = "v2"

// FigureSetEnv names a figure set for one launch, so two sets are compared by relaunching rather
// than by editing: `ASCEND_DUEL_FIGURES=v1 go run .`. **A set that does not exist falls back to
// DefaultFigureSet** with a line in the log, because a misspelled review knob must not cost the
// game its numbers.
const FigureSetEnv = "ASCEND_DUEL_FIGURES"

// FigureNeutral is the sheet a caller multiplies by an ink of its own. It is pure gray under a
// black contour, so any ink comes through as itself and the outline stays black.
const FigureNeutral = "neutral"

// figureGlyph is one glyph's record in a set's JSON: its cell, the first and last columns its
// contour reaches, and how far the pen moves after it — all in sheet pixels.
type figureGlyph struct {
	X, Y, W, H  int
	Left, Right int
	Advance     int
}

type figureSheet struct {
	Image string
}

type figureSetFile struct {
	CellWidth   int
	CellHeight  int
	Baseline    int
	DigitHeight int
	Glyphs      map[string]figureGlyph
	Sheets      map[string]figureSheet
}

// figureSet is a loaded set: the metrics, and each sheet cut into one sub-image per glyph.
type figureSet struct {
	meta   figureSetFile
	glyphs map[rune]figureGlyph
	sheets map[string]map[rune]*ebiten.Image
}

var (
	drawnFigures *figureSet
	figuresTried bool
)

// figures returns the drawn set, loading it on first use. **A set that will not load is nil**, and
// every caller then answers as though no glyph were covered, so the game falls back to the font
// rather than drawing nothing.
func figures() *figureSet {
	if figuresTried {
		return drawnFigures
	}
	figuresTried = true
	if name := os.Getenv(FigureSetEnv); name != "" && name != DefaultFigureSet {
		fs, err := loadFigureSet(name)
		if err == nil {
			log.Printf("figure set %q, from %s", name, FigureSetEnv)
			drawnFigures = fs
			return fs
		}
		log.Printf("figure set %q from %s: %v; drawing %q", name, FigureSetEnv, err, DefaultFigureSet)
	}
	fs, err := loadFigureSet(DefaultFigureSet)
	if err != nil {
		log.Printf("figure set %q: %v", DefaultFigureSet, err)
	}
	drawnFigures = fs
	return fs
}

func loadFigureSet(name string) (*figureSet, error) {
	raw, err := assets.FigureFile(name, "figure-glyphs.json")
	if err != nil {
		return nil, err
	}
	var meta figureSetFile
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	fs := &figureSet{meta: meta, glyphs: map[rune]figureGlyph{}, sheets: map[string]map[rune]*ebiten.Image{}}
	for k, g := range meta.Glyphs {
		r := []rune(k)
		if len(r) == 1 {
			fs.glyphs[r[0]] = g
		}
	}
	for sheetName, sh := range meta.Sheets {
		png, err := assets.FigureFile(name, sh.Image)
		if err != nil {
			return nil, err
		}
		src, _, err := image.Decode(bytes.NewReader(png))
		if err != nil {
			return nil, err
		}
		whole := ebiten.NewImageFromImage(src)
		cut := map[rune]*ebiten.Image{}
		for r, g := range fs.glyphs {
			cut[r] = whole.SubImage(image.Rect(g.X, g.Y, g.X+g.W, g.Y+g.H)).(*ebiten.Image)
		}
		fs.sheets[sheetName] = cut
	}
	return fs, nil
}

// FigureCovers says whether every character of str has a glyph, so the whole string can be drawn
// as a figure. An empty string is not covered.
func FigureCovers(str string) bool {
	fs := figures()
	if fs == nil || str == "" {
		return false
	}
	for _, r := range str {
		if _, ok := fs.glyphs[r]; !ok {
			return false
		}
	}
	return true
}

// FigureHasSheet says whether the drawn set carries a sheet by this name.
func FigureHasSheet(sheet string) bool {
	fs := figures()
	if fs == nil {
		return false
	}
	_, ok := fs.sheets[sheet]
	return ok
}

// MeasureFigure is the width of str set as a figure whose digits stand `height` pixels tall: the
// contour of the first glyph to the contour of the last, the way the pen actually lays them.
func MeasureFigure(str string, height float64) float64 {
	fs := figures()
	if fs == nil {
		return 0
	}
	s := height / float64(fs.meta.DigitHeight)
	w, _ := fs.span(str)
	return float64(w) * s
}

// span is str's width in sheet pixels, and each glyph's pen position from the first contour.
func (fs *figureSet) span(str string) (int, []int) {
	var pens []int
	pen, last := 0, figureGlyph{}
	for _, r := range str {
		g, ok := fs.glyphs[r]
		if !ok {
			continue
		}
		pens = append(pens, pen)
		pen += g.Advance
		last = g
	}
	if len(pens) == 0 {
		return 0, nil
	}
	return pens[len(pens)-1] + (last.Right - last.Left), pens
}

// DrawFigure sets str from one sheet, centered on (cx, cy) with its digits `height` pixels tall,
// then scaled about that center by `scale`.
//
// **Centered on the digits' body, not on the cell**: the cell has room under the baseline for the
// drop shadow, and centering on the cell would put every figure a few pixels high of the point it
// was laid out on. **A zero-alpha ink draws the sheet as it is**; any other multiplies it.
func DrawFigure(dst *ebiten.Image, str, sheet string, ink color.RGBA, cx, cy, height, scale float64, alpha float32) {
	fs := figures()
	if fs == nil || alpha <= 0 {
		return
	}
	cut, ok := fs.sheets[sheet]
	if !ok {
		cut, ok = fs.sheets[FigureNeutral]
		if !ok {
			return
		}
	}
	width, pens := fs.span(str)
	s := height / float64(fs.meta.DigitHeight) * scale
	body := float64(fs.meta.Baseline) - float64(fs.meta.DigitHeight)/2

	i := 0
	for _, r := range str {
		g, ok := fs.glyphs[r]
		if !ok {
			continue
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(float64(pens[i]-g.Left)-float64(width)/2, -body)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(cx, cy)
		if ink.A > 0 {
			op.ColorScale.ScaleWithColor(ink)
		}
		op.ColorScale.ScaleAlpha(alpha)
		dst.DrawImage(cut[r], op)
		i++
	}
}
