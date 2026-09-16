// Command marksheet renders every form mark and every cost tick, at the sizes the game draws them
// and enlarged, and writes an HTML page.
//
//	go run ./tools/marksheet
//
// # Why it exists
//
// **The form marks stopped being glyphs on 2026-09-16.** They were four drawings tinted to the
// element at draw time, which is why the old glyph contact sheet could show them: one picture per
// form, in a neutral palette, beside the generated silhouettes. They are now *authored per
// element* — four forms times five elements, plus a neutral set, plus six ticks — and a contact
// sheet cannot say anything useful about a set that size, because the question changed with the
// set. **That sheet was deleted on 2026-09-16 and this page took its job**, generated glyphs and
// all.
//
// The question now is whether the marks work **as a matrix**. A player counts a hand by comparing
// four silhouettes along a row and five hues down a column, so the two failures are a form that
// cannot be told from its neighbour and an element that cannot be told from its neighbour — and
// neither is visible one picture at a time.
//
// # What to look at
//
// **The actual-size rows first.** A mark is drawn at 32 pixels on a hand card and 16 in the deck
// overlay, and there is no zoom anywhere in the game. Reviewing only the enlarged row is how a mark
// comes to look acceptable in review and clunky in play — the rule the old glyph sheet
// established and the one thing this page inherits from it wholesale.
//
// **Down a column: are these five the same drawing?** They are meant to be one picture printed in
// five inks. A form whose fire version is a different shape from its ice version has the element
// doing the form's job, and the hand stops being countable.
//
// **Across a row: are these four the same weight?** The marks sit in one square box and are
// centered on their ink, never scaled up to fill it — so a narrow drawing is drawn narrow. The club
// landed ten pixels wide against the shield's twenty-two on 2026-09-16 and read as the quietest
// card in the hand; the ink figures under each cell are what that was caught with.
//
// **The neutral column against the five.** The gray marks are what the deck panel's filter column
// and the wildcard upgrade draw, so they have to read as the same objects with the color taken out
// rather than as a sixth element.
//
// **The ticks stacked, not just one.** A card draws one, two or three of the same picture down its
// left column, and what a stack reads as is not a property of one bar — three of them a `DashGap`
// apart is either a quantity or a hatch pattern.
//
// **The contour at the small size.** A near-black outline that averages away leaves a soft shape
// with nothing holding it against a pale card. The page prints, per cell, what share of the drawn
// pixels are dark — a batch arrived on 2026-09-16 with that share at zero and it was invisible by
// eye at review size.
//
// # It is a report
//
// Every picture comes from `systems.ArtMark` at the size `internal/cards` asks for, through
// `cards.MarkArtKey` and `cards.TickArtKey` — the game's own key builders — so a mark this page
// cannot draw is one the game cannot draw either, and a key naming no file shows as a gap rather
// than as a silent fallback. There is no fallback left to hide one, and no generator either: the
// silhouette generator in `internal/systems` was deleted on 2026-09-16 once the last kind it drew
// became an asset.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/marksheet/` and **committed**, on the
// same terms as every other sheet. A clone opens `docs/sheets/index.html`.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// ground is screens.screenGround, the light slate blue a card is held on, and the cells are drawn
// on cards.Surface, the face a mark is actually stamped on. **Both, because a mark is drawn on the
// card and the card is held on the table** — reviewing either against a white browser page is the
// same failure as previewing art at a scale the game does not use.
const (
	ground  = "#a8bcd4"
	surface = "#f0efea"
)

// zoom is what the enlarged rows are blown up by, in the page's CSS. Whole-number and
// nearest-neighbor, so the row is the real pixels bigger rather than a second, kinder rendering.
const zoom = 6

// darkFloor is what counts as the contour when the page measures one: a channel figure, not a
// judgment. It is the same threshold the batch check used on 2026-09-16.
const darkFloor = 70

// elementRow is the five a card is counted on plus the neutral drawing, in the order the page walks
// them. **Basic is the neutral one** — see cards.MarkArtKey, which answers the gray drawing for
// everything that is not one of the five.
var elementRow = []cards.Element{
	cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane, cards.Basic,
}

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "marksheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

// cell is one drawing on the page: where its PNG landed, and the figures worth reading beside it.
type cell struct {
	File string
	W, H int
	Key  string

	// Ink is the drawn size of the picture inside its box, which is what the card actually shows —
	// the mark is centered on its ink and never scaled to fill. Pixels is how much of it is opaque,
	// so two cells can be compared for weight rather than for bounding box.
	Ink    string
	Pixels int

	// Dark is the share of opaque pixels that are near-black: the contour, measured rather than
	// trusted. Zero means it averaged away on the reduction.
	Dark int

	// Missing says the key named no file, which is the one failure that must not look like a design
	// decision.
	Missing bool
}

type formRow struct {
	Form  string
	Cells []cell
}

type sizeBlock struct {
	Size  int
	Where string
	Forms []formRow
}

type tickBlock struct {
	Size   string
	Where  string
	Ones   []cell
	Threes []cell
}

type page struct {
	Ground   string
	Surface  string
	Zoom     int
	Elements []string
	Marks    []sizeBlock
	Ticks    []tickBlock
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	p := page{Ground: ground, Surface: surface, Zoom: zoom}
	for _, e := range elementRow {
		p.Elements = append(p.Elements, labelOf(e))
	}

	// The two sizes anything is drawn at are Hand's and Mini's, read off the styles rather than
	// written down here, so a style that moves moves this page with it.
	for _, st := range []struct {
		style cards.Style
		where string
	}{
		{cards.Hand, "a hand card"},
		{cards.Mini, "the deck overlay"},
	} {
		block := sizeBlock{Size: st.style.FormSize, Where: st.where}
		for _, f := range cards.Forms() {
			row := formRow{Form: f.String()}
			for _, e := range elementRow {
				c, err := writeMark(dir, f, e, st.style.FormSize)
				if err != nil {
					return err
				}
				row.Cells = append(row.Cells, c)
			}
			block.Forms = append(block.Forms, row)
		}
		p.Marks = append(p.Marks, block)

		tb := tickBlock{
			Size:  fmt.Sprintf("%dx%d", st.style.DashWidth, st.style.DashHeight),
			Where: st.where,
		}
		for _, e := range elementRow {
			one, err := writeTick(dir, e, st.style, 1)
			if err != nil {
				return err
			}
			three, err := writeTick(dir, e, st.style, 3)
			if err != nil {
				return err
			}
			tb.Ones = append(tb.Ones, one)
			tb.Threes = append(tb.Threes, three)
		}
		p.Ticks = append(p.Ticks, tb)
	}

	return writeIndex(dir, p)
}

// writeMark draws one form in one element at one size, exactly as a card asks for it.
func writeMark(dir string, f cards.Form, e cards.Element, size int) (cell, error) {
	key := cards.MarkArtKey(f, e)
	img := systems.ArtMark(key, size, size)
	if img == nil {
		return cell{Missing: true, Key: key}, nil
	}
	name := fmt.Sprintf("mark-%s-%s-%d.png", f, labelOf(e), size)
	if err := writePNG(filepath.Join(dir, name), img); err != nil {
		return cell{}, err
	}
	ink, opaque, dark := measure(img)
	return cell{
		File: name, W: size, H: size, Key: key,
		Ink:    fmt.Sprintf("%dx%d", ink.Dx(), ink.Dy()),
		Pixels: opaque, Dark: dark,
	}, nil
}

// writeTick draws n ticks stacked the way a card of that cost stacks them, at one style's size.
func writeTick(dir string, e cards.Element, st cards.Style, n int) (cell, error) {
	key := cards.TickArtKey(e)
	bar := systems.ArtMark(key, st.DashWidth, st.DashHeight)
	if bar == nil {
		return cell{Missing: true, Key: key}, nil
	}

	h := n*st.DashHeight + (n-1)*st.DashGap
	out := image.NewRGBA(image.Rect(0, 0, st.DashWidth, h))
	for i := 0; i < n; i++ {
		y := i * (st.DashHeight + st.DashGap)
		draw.Draw(out, image.Rect(0, y, st.DashWidth, y+st.DashHeight), bar, bar.Bounds().Min, draw.Over)
	}

	name := fmt.Sprintf("tick-%s-%d-%d.png", labelOf(e), st.DashWidth, n)
	if err := writePNG(filepath.Join(dir, name), out); err != nil {
		return cell{}, err
	}
	ink, opaque, dark := measure(bar)
	return cell{
		File: name, W: st.DashWidth, H: h, Key: key,
		Ink:    fmt.Sprintf("%dx%d", ink.Dx(), ink.Dy()),
		Pixels: opaque, Dark: dark,
	}, nil
}

// measure is the three figures under a cell: the ink's bounds, how many pixels are opaque, and what
// share of those are near-black. See the file comment for why each is worth printing.
func measure(img *image.RGBA) (image.Rectangle, int, int) {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	opaque, dark := 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.A < 8 {
				continue
			}
			opaque++
			if int(c.R) < darkFloor && int(c.G) < darkFloor && int(c.B) < darkFloor {
				dark++
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x >= maxX {
				maxX = x + 1
			}
			if y >= maxY {
				maxY = y + 1
			}
		}
	}
	if opaque == 0 {
		return image.Rectangle{}, 0, 0
	}
	return image.Rect(minX, minY, maxX, maxY), opaque, 100 * dark / opaque
}

// labelOf names an element for a filename and a heading. **Basic is written "neutral"**, which is
// what the asset is called and what the drawing is for — "basic" is the rules' word for a card with
// no element and would read here as a sixth color.
func labelOf(e cards.Element) string {
	if e == cards.Basic {
		return "neutral"
	}
	return e.String()
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeIndex(dir string, p page) error {
	// mul is the only function the template needs: the enlarged rows are the real pixel sizes
	// multiplied by the zoom, so the browser scales rather than the tool writing a second PNG.
	t, err := template.New("page").
		Funcs(template.FuncMap{"mul": func(a, b int) int { return a * b }}).
		Parse(pageHTML)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := t.Execute(f, p); err != nil {
		return err
	}
	fmt.Printf("wrote %s and the mark PNGs beside it\n", filepath.Join(dir, "index.html"))
	return nil
}
