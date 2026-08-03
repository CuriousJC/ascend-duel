// Command glyphsheet renders every generated glyph to a PNG contact sheet, so the art can
// be reviewed by opening a file rather than by launching the game and finding a card that
// happens to use it.
//
//	go run ./tools/glyphsheet
//
// The sheet is written next to this tool and committed, which is the point of it: GitHub
// renders image diffs side by side, so a change to a silhouette or a palette shows up in
// review as a picture. That only works if it is regenerated whenever the glyph code
// changes — a stale sheet is worse than none, because it is a picture that lies.
//
// **Two rows per palette, and the upper one is the honest one.** It draws each glyph at
// systems.CardGlyphScale, exactly what the game puts on a card, which is what answers "can
// I actually read this". The lower row blows it up for judging the pixel work. An earlier
// version of this tool had only the large row, and the glyphs duly looked acceptable in
// review and clunky in play.
//
// It needs no graphics context, because systems.RenderGlyph draws into a plain Go image.
// Only the ebiten wrapper needs a window, and this never asks for one.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

func main() {
	out := flag.String("out", filepath.Join("tools", "glyphsheet", "glyphs.png"),
		"path to write the sheet to")
	zoom := flag.Int("zoom", 4, "pixels per glyph pixel on the inspection row")
	flag.Parse()

	if *zoom < systems.CardGlyphScale {
		log.Fatalf("inspection zoom must be at least the card scale (%d), got %d",
			systems.CardGlyphScale, *zoom)
	}

	sheet := buildSheet(*zoom)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatalf("creating output directory: %v", err)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("creating %s: %v", *out, err)
	}
	defer f.Close()

	if err := png.Encode(f, sheet); err != nil {
		log.Fatalf("encoding %s: %v", *out, err)
	}

	fmt.Printf("wrote %s — %d glyphs x %d palettes, at %dx (as drawn in game) and %dx\n",
		*out, len(systems.GlyphKinds()), len(systems.PaletteNames()),
		systems.CardGlyphScale, *zoom)
}

// buildSheet lays the glyphs out in a grid: one column per glyph, and for each palette a
// pair of rows — actual size above, inspection size below.
//
// The background is dark on purpose. These are drawn to sit on the combat screen's
// near-black, and judging pixel art against white flatters an outline that will not
// actually be there.
func buildSheet(inspect int) image.Image {
	const pad = 24

	kinds := systems.GlyphKinds()
	names := systems.PaletteNames()

	// The inspection row is the wider of the two, so it sets the column.
	colWidth := systems.GlyphSize * inspect
	actualSize := systems.GlyphSize * systems.CardGlyphScale
	bandHeight := actualSize + pad + colWidth

	sheet := image.NewRGBA(image.Rect(0, 0,
		len(kinds)*colWidth+(len(kinds)+1)*pad,
		len(names)*bandHeight+(len(names)+1)*pad,
	))

	draw.Draw(sheet, sheet.Bounds(),
		&image.Uniform{color.RGBA{R: 24, G: 24, B: 30, A: 255}},
		image.Point{}, draw.Src)

	for row, name := range names {
		bandTop := pad + row*(bandHeight+pad)

		for col, kind := range kinds {
			glyph := systems.RenderGlyph(kind, name)
			colLeft := pad + col*(colWidth+pad)

			// Actual size, centred over the column it shares with the big one below.
			blit(sheet, glyph, colLeft+(colWidth-actualSize)/2, bandTop, systems.CardGlyphScale)
			blit(sheet, glyph, colLeft, bandTop+actualSize+pad, inspect)
		}
	}

	return sheet
}

// blit copies a glyph onto the sheet, magnified by a whole number.
//
// Nearest-neighbour by hand: any smoothing would defeat the purpose, since the point is to
// see the pixels exactly as the game draws them.
func blit(dst *image.RGBA, src *image.RGBA, originX, originY, scale int) {
	for y := 0; y < systems.GlyphSize; y++ {
		for x := 0; x < systems.GlyphSize; x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			draw.Draw(dst, image.Rect(
				originX+x*scale, originY+y*scale,
				originX+(x+1)*scale, originY+(y+1)*scale,
			), &image.Uniform{c}, image.Point{}, draw.Src)
		}
	}
}
