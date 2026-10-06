// Command interfacesheet renders the interface's own art — every lettering sheet, every button face,
// every square button and the parts square buttons are assembled from, the bar cells — and writes an
// HTML page.
//
//	go run ./tools/interfacesheet
//
// # Why it exists
//
// The catalogs each have a sheet; the interface had none, and its art is authored in batches the way
// a catalog's is: a lettering sheet per ink, a face per button color, a square button per control.
// What goes wrong is a matter of the set rather than of one file — a jade that drifts between two
// generations, one ink of the figure lettering missing a glyph the others have, a face that reads at
// 80 pixels and smears at 32 — and none of it is visible one file at a time.
//
// # What to look at
//
// **The lettering at the tier sizes.** Prose is drawn at the three reading tiers and the figures at
// two of their sizes, each on the table's own blue and on a dark panel, because both sets are white
// under an outline and a specimen on a white browser page says nothing about either.
//
// **The missing glyphs.** Every ink of the figure set is walked against the same alphabet, and a
// character one ink lacks is listed under it — a line falls back to the font whole when any glyph is
// missing, so one gap is a whole word changing typeface.
//
// **The buttons at their tier heights.** A face is drawn at each of the four button heights, and a
// square button at the sizes square buttons are used at. A button that reads at the largest and not
// the smallest is the defect.
//
// # It is a report
//
// Everything comes out of the embedded assets the game ships, through `assets.LoadImageData` and the
// same plain glyph sheets `internal/cards` sets type with, so a picture this page cannot find is one
// the game cannot either. Files are walked by name, so a new face, icon or glyph appears here with
// nothing edited.
//
// **The wide word faces are composed here**, ends and middle, at the screen's own rule — the ends
// scaled to the height, the middle filling the rest — because the game composes them on the GPU and
// this page has no graphics context. It is the one drawing on the page the game does not share.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/interfacesheet/` and committed, on the
// same terms as every other sheet.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// The two grounds every specimen is shown on: the table's own blue, and the dark of a panel.
var (
	tableGround = color.RGBA{R: 126, G: 172, B: 228, A: 255}
	panelGround = color.RGBA{R: 30, G: 32, B: 40, A: 255}
)

// The button heights, the square sizes, and the figure sizes the page draws at.
var (
	faceTiers   = []int{80, 68, 44, 32}
	squareSizes = []int{68, 44, 32}
	figureSizes = []float64{40, 20}
	proseTiers  = []float64{systems.TextSmall, systems.TextMedium, systems.TextLarge}
	figureInks  = []string{"neutral", "fire", "ice", "lightning", "earth", "arcane", "attack", "relic", "vitae"}
)

const (
	faceCap      = 48 // the end cap's width in a 512x128 face, per the button prompt
	figureLetter = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	figureDigits = "0123456789"
	figureMarks  = "+-x=()./%!?:,¤▲▼"
	proseSample  = "The quick brown fox jumps over the lazy dog."
)

func main() {
	out := flag.String("out", "docs/sheets/interfacesheet", "where to write the page")
	flag.Parse()
	if err := run(*out); err != nil {
		log.Fatal(err)
	}
}

// page is everything the template draws.
type page struct {
	Prose     []strip
	Figures   []figureRow
	Faces     []strip
	Squares   []squareRow
	Assembled []strip
	Bars      []strip
}

// strip is one composed picture with a caption.
type strip struct {
	File    string
	Caption string
	W, H    int
}

type figureRow struct {
	Ink     string
	File    string
	W, H    int
	Missing string
}

type squareRow struct {
	Name    string
	File    string
	W, H    int
	Pressed bool
}

func run(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	images := assets.LoadImageData()
	var p page
	var err error

	if p.Prose, err = proseStrips(out); err != nil {
		return err
	}
	if p.Figures, err = figureRows(out); err != nil {
		return err
	}
	if p.Faces, err = faceStrips(out, images); err != nil {
		return err
	}
	if p.Squares, err = squareRows(out, images); err != nil {
		return err
	}
	if p.Assembled, err = assembledStrips(out, images); err != nil {
		return err
	}
	if p.Bars, err = barStrips(out, images); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(out, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	return template.Must(template.New("page").Parse(pageHTML)).Execute(f, p)
}

// proseStrips sets the prose sheet's whole repertoire and a sentence at each reading tier, on both
// grounds.
func proseStrips(out string) ([]strip, error) {
	sheet := systems.ProseGlyphs()
	if sheet == nil {
		return nil, fmt.Errorf("the prose sheet will not load")
	}
	lines := []string{
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"abcdefghijklmnopqrstuvwxyz",
		"0123456789 !\"#%&'()*+,-./:;<=>?[]_|·×…–—",
		proseSample,
	}
	var strips []strip
	for _, tier := range proseTiers {
		cap := systems.ProseCap(tier)
		pitch := int(cap*2.4) + 4
		w := 0
		for _, l := range lines {
			if n := int(sheet.Width(l, cap)) + 24; n > w {
				w = n
			}
		}
		h := pitch*len(lines) + 12
		img := image.NewRGBA(image.Rect(0, 0, w*2+12, h))
		fill(img, image.Rect(0, 0, w, h), tableGround)
		fill(img, image.Rect(w+12, 0, 2*w+12, h), panelGround)
		for i, l := range lines {
			y := float64(pitch*(i+1)) - cap*0.6
			sheet.Draw(img, l, color.RGBA{}, 12, y, cap)
			sheet.Draw(img, l, color.RGBA{}, float64(w+24), y, cap)
		}
		name := fmt.Sprintf("prose-%g.png", tier)
		if err := write(filepath.Join(out, name), img); err != nil {
			return nil, err
		}
		strips = append(strips, strip{File: name, Caption: fmt.Sprintf("%gpt tier — capital %.0fpx", tier, cap),
			W: img.Bounds().Dx(), H: img.Bounds().Dy()})
	}
	return strips, nil
}

// figureRows sets the alphabet, the digits and the marks in every ink of the figure set, at both
// figure sizes, and lists what each ink is missing.
func figureRows(out string) ([]figureRow, error) {
	var rows []figureRow
	lines := []string{figureLetter, figureDigits + " " + figureMarks}
	for _, ink := range figureInks {
		sheet := systems.FigureGlyphs(ink)
		if sheet == nil {
			rows = append(rows, figureRow{Ink: ink, Missing: "the whole sheet — it will not load"})
			continue
		}
		var missing []string
		for _, r := range figureLetter + figureDigits + figureMarks {
			if !sheet.Covers(string(r)) {
				missing = append(missing, string(r))
			}
		}

		w, h := 0, 12
		for _, size := range figureSizes {
			for _, l := range lines {
				if n := int(sheet.Width(l, size)) + 24; n > w {
					w = n
				}
				h += int(size*1.6) + 6
			}
		}
		img := image.NewRGBA(image.Rect(0, 0, w*2+12, h))
		fill(img, image.Rect(0, 0, w, h), tableGround)
		fill(img, image.Rect(w+12, 0, 2*w+12, h), panelGround)
		y := 6.0
		for _, size := range figureSizes {
			for _, l := range lines {
				y += size*1.6 + 6
				sheet.Draw(img, l, color.RGBA{}, 12, y-size*0.4, size)
				sheet.Draw(img, l, color.RGBA{}, float64(w+24), y-size*0.4, size)
			}
		}
		name := "figure-" + ink + ".png"
		if err := write(filepath.Join(out, name), img); err != nil {
			return nil, err
		}
		rows = append(rows, figureRow{Ink: ink, File: name, W: img.Bounds().Dx(), H: img.Bounds().Dy(),
			Missing: strings.Join(missing, " ")})
	}
	return rows, nil
}

// faceStrips draws every word face at each button height, once at its own proportions and once
// stretched to a wide button.
func faceStrips(out string, images map[string][]byte) ([]strip, error) {
	var strips []strip
	for _, key := range keysWithPrefix(images, "button-") {
		face, err := decode(images[key])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		w, h := 0, 12
		for _, tier := range faceTiers {
			if n := tier*4 + tier*9 + 36; n > w {
				w = n
			}
			h += tier + 12
		}
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		fill(img, img.Bounds(), tableGround)
		y := 12
		for _, tier := range faceTiers {
			xdraw.CatmullRom.Scale(img, image.Rect(12, y, 12+tier*4, y+tier), face, face.Bounds(), xdraw.Over, nil)
			drawWide(img, face, image.Rect(24+tier*4, y, 24+tier*4+tier*9, y+tier))
			y += tier + 12
		}
		name := key + ".png"
		if err := write(filepath.Join(out, name), img); err != nil {
			return nil, err
		}
		strips = append(strips, strip{File: name, Caption: key, W: w, H: h})
	}
	return strips, nil
}

// drawWide composes a face into a rectangle wider than its proportions, by the screen's rule: each
// end cap scaled to the height, and the middle laid side by side a whole number of times, each copy
// stretched a little so they fill the span exactly.
func drawWide(dst *image.RGBA, face image.Image, r image.Rectangle) {
	b := face.Bounds()
	scale := float64(r.Dy()) / float64(b.Dy())
	capW := int(float64(faceCap)*scale + 0.5)
	put := func(src, at image.Rectangle) { xdraw.CatmullRom.Scale(dst, at, face, src, xdraw.Over, nil) }
	put(image.Rect(0, 0, faceCap, b.Dy()), image.Rect(r.Min.X, r.Min.Y, r.Min.X+capW, r.Max.Y))
	put(image.Rect(b.Dx()-faceCap, 0, b.Dx(), b.Dy()), image.Rect(r.Max.X-capW, r.Min.Y, r.Max.X, r.Max.Y))
	left, mid := r.Min.X+capW, r.Dx()-2*capW
	tile := float64(b.Dx()-2*faceCap) * scale
	copies := max(1, int(float64(mid)/tile+0.5))
	middle := image.Rect(faceCap, 0, b.Dx()-faceCap, b.Dy())
	for i := 0; i < copies; i++ {
		x0, x1 := left+mid*i/copies, left+mid*(i+1)/copies
		put(middle, image.Rect(x0, r.Min.Y, x1, r.Max.Y))
	}
}

// squareRows draws every whole square button at the sizes square buttons are used at.
func squareRows(out string, images map[string][]byte) ([]squareRow, error) {
	var rows []squareRow
	for _, key := range keysWithPrefix(images, "icon-") {
		if strings.HasPrefix(key, "icon-blank") {
			continue // a blank tile is a part, drawn under the assembled buttons
		}
		img, err := sizes(images[key], tableGround)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		name := key + ".png"
		if err := write(filepath.Join(out, name), img); err != nil {
			return nil, err
		}
		rows = append(rows, squareRow{Name: key, File: name, W: img.Bounds().Dx(), H: img.Bounds().Dy(),
			Pressed: strings.HasSuffix(key, "-pressed")})
	}
	return rows, nil
}

// assembledStrips lays every glyph over every blank tile, at the sizes square buttons are used at.
func assembledStrips(out string, images map[string][]byte) ([]strip, error) {
	tiles := keysWithPrefix(images, "icon-blank")
	var strips []strip
	for _, tileKey := range tiles {
		tile, err := decode(images[tileKey])
		if err != nil {
			return nil, err
		}
		glyphs := keysWithPrefix(images, "glyph-")
		pictures := []image.Image{tile}
		labels := []string{"(blank)"}
		for _, g := range glyphs {
			gi, err := decode(images[g])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", g, err)
			}
			pictures = append(pictures, gi)
			labels = append(labels, strings.TrimPrefix(g, "glyph-"))
		}
		total := 0
		for _, s := range squareSizes {
			total += s + 12
		}
		cell := squareSizes[0] + 16
		img := image.NewRGBA(image.Rect(0, 0, cell*len(pictures)+12, total+12))
		fill(img, img.Bounds(), tableGround)
		for i, pic := range pictures {
			y := 12
			for _, s := range squareSizes {
				r := image.Rect(12+i*cell, y, 12+i*cell+s, y+s)
				xdraw.CatmullRom.Scale(img, r, tile, tile.Bounds(), xdraw.Over, nil)
				if i > 0 {
					xdraw.CatmullRom.Scale(img, r, pic, pic.Bounds(), xdraw.Over, nil)
				}
				y += s + 12
			}
		}
		name := "assembled-" + tileKey + ".png"
		if err := write(filepath.Join(out, name), img); err != nil {
			return nil, err
		}
		strips = append(strips, strip{File: name, W: img.Bounds().Dx(), H: img.Bounds().Dy(),
			Caption: tileKey + ": " + strings.Join(labels, ", ")})
	}
	return strips, nil
}

// barStrips draws the bar cells and the health bar's pieces as they were delivered.
func barStrips(out string, images map[string][]byte) ([]strip, error) {
	var strips []strip
	for _, prefix := range []string{"bar-cell-", "health-bar-"} {
		for _, key := range keysWithPrefix(images, prefix) {
			pic, err := decode(images[key])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			b := pic.Bounds()
			img := image.NewRGBA(image.Rect(0, 0, b.Dx()+24, b.Dy()+24))
			fill(img, img.Bounds(), tableGround)
			draw.Draw(img, b.Add(image.Pt(12, 12)), pic, b.Min, draw.Over)
			name := key + ".png"
			if err := write(filepath.Join(out, name), img); err != nil {
				return nil, err
			}
			strips = append(strips, strip{File: name, Caption: key, W: img.Bounds().Dx(), H: img.Bounds().Dy()})
		}
	}
	return strips, nil
}

// sizes draws one square picture at every square size, side by side.
func sizes(raw []byte, ground color.RGBA) (*image.RGBA, error) {
	pic, err := decode(raw)
	if err != nil {
		return nil, err
	}
	w := 12
	for _, s := range squareSizes {
		w += s + 12
	}
	img := image.NewRGBA(image.Rect(0, 0, w, squareSizes[0]+24))
	fill(img, img.Bounds(), ground)
	x := 12
	for _, s := range squareSizes {
		r := image.Rect(x, 12+squareSizes[0]-s, x+s, 12+squareSizes[0])
		xdraw.CatmullRom.Scale(img, r, pic, pic.Bounds(), xdraw.Over, nil)
		x += s + 12
	}
	return img, nil
}

func keysWithPrefix(images map[string][]byte, prefix string) []string {
	var keys []string
	for k := range images {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func decode(raw []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func fill(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}

func write(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
