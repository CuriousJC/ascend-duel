// Command badgesheet renders every damage badge — each drawn multiplier in each color, at the
// sizes the game draws them and enlarged — and writes an HTML page.
//
//	go run ./tools/badgesheet
//
// # Why it exists
//
// **The badge carries its own numeral as of 2026-09-16**, which turned a set of six pictures into
// a set of sixty-six: eleven values on the ladder times five elements plus a neutral. Six can be
// checked by eye one at a time; sixty-six cannot, and the failure mode is specific and invisible
// that way — **the numeral drifting in weight, size or position between colors**, so a hand shows a
// fat 3 beside a thin one. Every picture on this page is generated, and the five versions of one
// value are supposed to be one drawing in five inks.
//
// It is the mark sheet's argument applied to a second matrix, and the mark sheet earned it twice:
// a contour that averaged to nothing and a club drawn half the ink of the shield beside it were
// both invisible one file at a time and obvious on a page.
//
// # What to look at
//
// **The actual-size table first.** A badge is drawn at 32 pixels on a hand card and 16 in the deck
// overlay, and there is no zoom anywhere in the game. Reviewing only the enlarged table is how a
// numeral comes to look acceptable in review and illegible in play.
//
// **Across a row: are these six the same drawing?** One value in five colors plus the neutral, and
// they are meant to be one picture printed in six inks. A row where one badge's numeral sits lower,
// or is a different weight, is the defect this page exists for — and it is a redraw of one file
// rather than of the batch, which is why these are individual images and not a sprite sheet.
//
// **Down a column: can you read every value at 16 pixels?** The two fractions are the hard ones,
// and `quarter` is the hardest: it carries the most ink of any numeral in the set into the
// smallest space. If it mushes, the rung needs a different mark rather than a smaller font.
//
// **The contour at the small size.** A near-black outline that averages away leaves a soft shape
// with nothing holding it against a pale card. The page prints, per cell, what share of the drawn
// pixels are dark — the same measurement the mark sheet makes, and for the same reason.
//
// **The gaps.** A value with no file is drawn as a gap rather than as a fallback, so a batch that
// came back eleven-of-twelve says so here instead of in a duel.
//
// # It is a report
//
// Every picture comes from `systems.ArtMark` at the size `internal/cards` asks for, through
// `cards.BadgeArtKey` — the game's own key builder — so a badge this page cannot draw is one the
// game cannot draw either. It walks `cards.BadgeValues`, so a value added to the ladder appears
// here with nothing edited.
//
// **It draws the shipped shape and the two that are not.** `cards.DefaultBadgeShape` is what the
// game uses; the other outlines are on the page underneath, unnumbered, because the choice of
// shape is still open and a page that showed only the chosen one could not be used to change it.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/badgesheet/` and **committed**, on the
// same terms as every other sheet. A clone opens `docs/sheets/index.html`.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// ground is screens.screenGround, the light slate blue a card is held on, and surface is
// cards.Surface, the face a badge is actually stamped on. **Both, because a badge is drawn on the
// card and the card is held on the table** — reviewing either against a white browser page is the
// same failure as previewing art at a scale the game does not use.
const (
	ground  = "#a8bcd4"
	surface = "#f0efea"
)

// zoom is what the enlarged table is blown up by, in the page's CSS. Whole-number and
// nearest-neighbor, so the row is the real pixels bigger rather than a second, kinder rendering.
const zoom = 6

// darkFloor is what counts as the contour when the page measures one: a channel figure, not a
// judgment. The mark sheet's threshold, so the two pages report the same number the same way.
const darkFloor = 70

// drawnSizes are the two the game asks for: a hand card's badge, and the deck overlay's.
var drawnSizes = []struct {
	Size  int
	Where string
}{
	{32, "hand card"},
	{16, "deck overlay"},
}

// elementRow is the five a card is counted on plus the neutral drawing, in the order the page walks
// them. **Basic is the neutral one** — see cards.BadgeArtKey, which answers the gray drawing for
// everything that is not one of the five.
var elementRow = []cards.Element{
	cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane, cards.Basic,
}

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "badgesheet"),
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

	// Ink is the drawn size of the picture inside its box, which is what the card actually shows.
	// Pixels is how much of it is opaque, so two cells can be compared for weight rather than for
	// bounding box — which is how a numeral drawn heavier in one color is caught.
	Ink    string
	Pixels int

	// Dark is the share of opaque pixels that are near-black: the contour and the numeral together,
	// measured rather than trusted. Zero means it averaged away on the reduction.
	Dark int

	// Missing says the key named no file, which is the one failure that must not look like a
	// design decision.
	Missing bool
}

type valueRow struct {
	Label string
	Token string
	Cells []cell
}

type sizeBlock struct {
	Size   int
	Where  string
	Values []valueRow
}

type shapeRow struct {
	Shape string
	Cells []cell
}

type pageData struct {
	Ground, Surface string
	Zoom            int
	Shape           string
	Sizes           []sizeBlock
	Shapes          []shapeRow
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data := pageData{
		Ground: ground, Surface: surface, Zoom: zoom,
		Shape: string(cards.DefaultBadgeShape),
	}

	for _, sz := range drawnSizes {
		block := sizeBlock{Size: sz.Size, Where: sz.Where}
		for _, pct := range cards.BadgeValues {
			row := valueRow{Label: label(pct), Token: cards.BadgeValueToken(pct)}
			for _, e := range elementRow {
				c, err := write(dir, cards.BadgeArtKey(cards.DefaultBadgeShape, pct, e), sz.Size)
				if err != nil {
					return err
				}
				row.Cells = append(row.Cells, c)
			}
			block.Values = append(block.Values, row)
		}
		data.Sizes = append(data.Sizes, block)
	}

	// The three outlines, unnumbered, at hand size. **The shape is still an open choice**, and a
	// page drawing only the shipped one could not be used to change it.
	for _, shape := range []cards.BadgeShape{cards.BadgeDiamond, cards.BadgeCircle, cards.BadgeStarburst} {
		row := shapeRow{Shape: string(shape)}
		for _, e := range elementRow {
			key := "damage-" + string(shape) + "-neutral"
			if e != cards.Basic {
				key = "damage-" + string(shape) + "-" + e.String()
			}
			c, err := write(dir, key, 32)
			if err != nil {
				return err
			}
			row.Cells = append(row.Cells, c)
		}
		data.Shapes = append(data.Shapes, row)
	}

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("page").Parse(pageHTML)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("badgesheet: wrote %s\n", filepath.Join(dir, "index.html"))
	return nil
}

// label is how a multiplier is written for a human reading the page — the same wording the card
// prints, so a row heading and a badge cannot disagree.
func label(pct int) string {
	switch pct {
	case 25:
		return "1/4"
	case 50:
		return "1/2"
	}
	return fmt.Sprintf("%d", pct/100)
}

// write renders one badge at one size, files it, and measures it.
//
// **An empty key or a missing file is a gap rather than a fallback.** The game has none here
// either — see cards.BadgeArtKey, which answers "" for a value nobody drew — and a page that
// quietly substituted a picture would be a page that cannot report a batch coming back short.
func write(dir, key string, size int) (cell, error) {
	c := cell{Key: key, W: size, H: size}
	if key == "" {
		c.Missing = true
		return c, nil
	}

	img := systems.ArtMark(key, size, size)
	if img == nil {
		c.Missing = true
		return c, nil
	}

	name := fmt.Sprintf("%s-%d.png", key, size)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return c, err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return c, err
	}

	c.File = name
	ink, opaque, dark := measure(img)
	c.Ink = fmt.Sprintf("%dx%d", ink.Dx(), ink.Dy())
	c.Pixels = opaque
	if opaque > 0 {
		c.Dark = dark * 100 / opaque
	}
	return c, nil
}

// measure is the ink's bounding box, how many pixels of it are opaque, and how many of those are
// near-black. The third figure is the contour and the numeral together: what has to survive the
// reduction for the badge to read at all.
func measure(g *image.RGBA) (image.Rectangle, int, int) {
	b := g.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	opaque, dark := 0, 0

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := g.RGBAAt(x, y)
			if px.A == 0 {
				continue
			}
			opaque++
			if int(px.R) < darkFloor && int(px.G) < darkFloor && int(px.B) < darkFloor {
				dark++
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX {
		return image.Rectangle{}, 0, 0
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), opaque, dark
}
