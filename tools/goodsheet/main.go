// Command goodsheet renders every sealed good in data/goods.json to a PNG and writes an HTML page
// that shows each one beside the offer it actually makes.
//
//	go run ./tools/goodsheet
//
// It exists for the reason tools/stonesheet does, one shelf further out. Two goods stand on a
// shop's shelf and they are one catalog each, so meeting the nine in a launched game means a lot
// of visits and a lot of rerolling. This draws all of them at once.
//
// # It is a report, not a drawing-board
//
// Same split as relicsheet against cardsheet: this reads the real file through internal/session,
// which means the catalog is *validated* before anything is drawn. A good naming a catalog this
// build has not got panics at init exactly as it would in the game, so a good this page refuses to
// draw is a good the game refuses to start with.
//
// # What to look at
//
// **The tooltip, because the face is a picture and a name.** A sealed good stopped printing its
// offer across the lower half of its art on 2026-09-15, so what a player can know before paying is
// said entirely by resting on the card. The tip is built here the way screens.goodTip builds it —
// the count and the noun computed, the prose read off the record — and this is the only place the
// authored lines and the computed ones can be read as the one paragraph a player sees.
//
// **The marginal price of an option.** Every good gives the player exactly one of what it holds,
// whatever its size, so what a larger vessel sells is not more cards — it is a wider choice. The
// page prices that margin rather than the contents: what the extra options cost over the size
// below. A ladder charging more for the fourth option than for the fifth is a ladder whose middle
// rung nobody has a reason to buy, and it is invisible reading three prices down a column.
//
// **The size against the catalog it draws from.** A sack showing five of a catalog that holds six
// is a sealed good that is barely sealed — the choice it sells is nearly the whole shelf. The
// share is computed from the live catalogs rather than authored, so authoring a sixth rune moves
// it here.
//
// **A contents nobody has authored a vessel for.** session.GoodContents is a closed vocabulary and
// every value gets a heading whether or not goods.json uses it, so a catalog with nothing holding
// it shows as a gap rather than as an absence nobody notices — the stone sheet's rule, applied to
// the other axis.
//
// **The dialog's heading and hint.** Both are resolved at load — an empty Title becomes the Name
// and an empty Hint becomes the computed "take one of the N, the rest are gone" — so what is
// printed here is what the dialog says rather than what the file wrote.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/goodsheet/` and **committed**
// *(owner's call, 2026-08-23)*, on the same terms as every other sheet. A clone opens
// `docs/sheets/index.html`.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a good is actually offered on. Judging a
// card against a white browser page would be the same failure as previewing art at a scale the
// game does not use.
const ground = "#a8bcd4"

// packsOffered is screens.packsOffered, how many goods stand on one shelf. Said here because the
// page's whole framing is how little of the catalog one visit shows; it is not a rule this tool
// owns, and a shelf that grew would want editing in both places.
const packsOffered = 2

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "goodsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	goods := session.Goods()
	page := page{
		Ground:  ground,
		Style:   styleFacts(cards.EssenceStyle),
		Count:   len(goods),
		Seats:   packsOffered,
		Catalog: len(allContents),
	}

	// **Walked by contents rather than by record**, which is the one decision in this file. The
	// shelf picks a contents first and takes whichever size the shuffle landed on, so grouping the
	// page the way the roll groups the catalog puts each ladder in one block and makes a contents
	// nobody has authored a vessel for visible as a gap. Walking session.Goods() would put the
	// sizes in file order and hide it.
	for _, c := range allContents {
		g := group{Contents: c.Noun(), Held: catalogSize(c)}
		for _, good := range sizeOrder(goods, c) {
			p, err := build(dir, faces, good, g.Held, briefs[good.Record])
			if err != nil {
				return err
			}
			// **The margin is over the size below, not over the smallest.** What a player weighs
			// is this vessel against the one they could have had instead, and the ladder is walked
			// smallest first so the previous plate is that one.
			if n := len(g.Goods); n > 0 {
				prev := g.Goods[n-1]
				p.StepSize = p.Size - prev.Size
				p.StepPrice = p.Price - prev.Price
				p.HasStep = p.StepSize > 0
			}
			g.Goods = append(g.Goods, p)
			if p.Borrowed {
				page.Borrowed++
			}
			if p.Draw == "" {
				page.Unwritten++
			}
		}
		if len(g.Goods) == 0 {
			page.Unheld++
		}
		page.Groups = append(page.Groups, g)
	}

	// The two states a sealed good is drawn in. **Not "selected"** — a good is bought rather than
	// chosen out of a set, so the shelf dims what the purse cannot reach and lights nothing.
	if first, ok := firstGood(page.Groups); ok {
		good, _ := session.GoodByKey(first.Record)
		art, err := goodArt(good)
		if err != nil {
			return err
		}
		for _, s := range []struct {
			name    string
			label   string
			enabled bool
		}{
			{"rest", good.Name + " — on the shelf", true},
			{"disabled", good.Name + " — more than the purse holds", false},
		} {
			cell, err := write(dir, faces, specFor(good, art, s.enabled), "state-"+s.name+".png", s.label)
			if err != nil {
				return err
			}
			page.States = append(page.States, cell)
		}
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d PNGs — %d goods over %d contents, %d standing on a shelf at a time; "+
		"%d drawing a borrowed face and %d with no subject\n",
		out, page.Count+len(page.States), page.Count, page.Catalog, page.Seats,
		page.Borrowed, page.Unwritten)
	for _, g := range page.Groups {
		if len(g.Goods) == 0 {
			fmt.Printf("  %-10s no vessel holds them\n", g.Contents)
			continue
		}
		fmt.Printf("  %-10s %d sizes — %s vitae — out of %d in the catalog\n",
			g.Contents, len(g.Goods), g.Ladder(), g.Held)
	}
	return nil
}

// allContents is the closed vocabulary session.GoodContents is, spelled out so the page can give
// every value a heading whether or not goods.json uses it. A fourth contents is a stream to seed
// and a dialog to draw, so it arrives here in the same commit it arrives in internal/session.
var allContents = []session.GoodContents{
	session.ContentsStones,
	session.ContentsEssences,
	session.ContentsRunes,
}

// briefs is each good's Draw, keyed by record.
//
// **Read off data.LoadGoods rather than session.Good**, which does not carry it: Draw is the
// subject paragraph an art generator is given and the engine ignores it, exactly as a status's
// Badge is ignored — so it never had a reason to be resolved onto the run's side of the catalog.
// A review sheet is the one reader it has.
var briefs = func() map[string]string {
	out := map[string]string{}
	for _, rec := range data.LoadGoods() {
		out[rec.GoodRecord] = rec.Draw
	}
	return out
}()

// build renders one good and gathers everything the page says about it.
func build(dir string, faces *cards.Faces, good session.Good, held int, draw string) (plate, error) {
	art, err := goodArt(good)
	if err != nil {
		return plate{}, err
	}
	cell, err := write(dir, faces, specFor(good, art, true), "good-"+good.Record+".png", good.Name)
	if err != nil {
		return plate{}, err
	}

	title, lines := goodTip(good)
	return plate{
		Cell:     cell,
		Record:   good.Record,
		Name:     good.Name,
		Family:   good.Family,
		Contents: good.Contains.Noun(),
		Size:     good.Size,
		Price:    good.Price,
		Per:      fmt.Sprintf("%.2f", float64(good.Price)/float64(good.Size)),
		Share:    share(good.Size, held),
		TipTitle: title,
		Tip:      lines,
		Title:    good.Title,
		Hint:     good.Hint,
		Art:      good.Art,
		Draw:     draw,
		Borrowed: good.Art == "",
	}, nil
}

// specFor is a good as the card the shelf draws, and it fills the same fields ui.goodSpec does: a
// name, a picture, and nothing else.
//
// **No Text, deliberately.** The face carried "4 stones / keep 1" across the lower half of its art
// until 2026-09-15; what it says now is said by the tooltip, which this page prints beside the card
// rather than on it. A sheet drawing a band the game does not draw would be a picture of a card
// that does not exist.
//
// **Basic, not a color** — a good holds a catalog rather than an element, so its border is the mid
// gray cards.BorderOf gives `basic`, exactly as a stone's and a potion's are.
func specFor(good session.Good, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    good.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: cards.Basic,
		Art:     art,
		Enabled: enabled,
	}
}

// goodTip is what resting on one says, built the way screens.goodTip builds it: the count and the
// noun computed from the record, the prose read off it, the price last.
//
// **Spelled here rather than called.** internal/screens links Ebitengine, so no review tool can
// import it — the same wall internal/carddesc exists to get round. The cost is a second copy of
// four lines, and the tripwire is this page: a tooltip that stopped matching would be visibly
// wrong beside the card it describes.
func goodTip(good session.Good) (string, []string) {
	lines := make([]string, 0, len(good.Tip)+2)
	lines = append(lines, fmt.Sprintf("%d %s, and you keep one", good.Size, good.Contains.Noun()))
	lines = append(lines, good.Tip...)
	return good.Name, append(lines, fmt.Sprintf("%d vitae", good.Price))
}

// goodArt is the picture one good draws, resolved exactly as screens.goodArt resolves it: its own
// if it has one, and otherwise the default face of whatever is inside it.
//
// **A key naming no embedded file is an error rather than a blank face**, as every other sheet's
// artwork is — a review tool that quietly drew nothing would hide what it is for.
func goodArt(good session.Good) (image.Image, error) {
	key := good.Art
	if key == "" {
		switch good.Contains {
		case session.ContentsStones:
			key = data.DefaultStoneArt
		case session.ContentsRunes:
			key = data.DefaultRuneArt
		default:
			key = data.DefaultEssenceArt
		}
	}
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s draws %q, which is in no embed", good.Record, key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

// sizeOrder is every good holding one catalog, smallest first — the order a ladder is read in.
func sizeOrder(goods []session.Good, c session.GoodContents) []session.Good {
	var out []session.Good
	for _, g := range goods {
		if g.Contains == c {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size < out[j].Size })
	return out
}

// catalogSize is how many records the contents are drawn from, asked of the live catalog rather
// than written down — so authoring a sixth rune moves every share on this page.
func catalogSize(c session.GoodContents) int {
	switch c {
	case session.ContentsStones:
		return len(session.Stones())
	case session.ContentsRunes:
		return len(session.Runes())
	default:
		return len(session.Essences())
	}
}

// share is how much of a catalog one good puts in front of the player, to a tenth of a percent.
func share(size, held int) string {
	if held == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f", float64(size)*100/float64(held))
}

// firstGood is the first good on the page, for the states row. A search rather than the first
// group's first plate, because a contents nobody has authored a vessel for would otherwise draw
// the states row blank.
func firstGood(groups []group) (plate, bool) {
	for _, g := range groups {
		if len(g.Goods) > 0 {
			return g.Goods[0], true
		}
	}
	return plate{}, false
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, name, label string) (cell, error) {
	img, err := cards.Render(s, cards.EssenceStyle, f)
	if err != nil {
		return cell{}, fmt.Errorf("rendering %s: %w", name, err)
	}

	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return cell{}, fmt.Errorf("creating %s: %w", name, err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return cell{}, fmt.Errorf("encoding %s: %w", name, err)
	}
	return cell{
		File: name, Label: label,
		Width: cards.EssenceStyle.Width, Height: cards.EssenceStyle.Height,
	}, nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":        st.Width,
		"height":       st.Height,
		"cornerRadius": st.CornerRadius,
		"borderWidth":  st.BorderWidth,
		"artTop":       st.ArtTop,
		"artInset":     st.ArtInset,
		"artMaxH":      st.ArtMaxH,
		"textBandTop":  st.TextBandTop,
	}
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// plate is one sealed good: the card, the offer it makes, and what it costs over the size below it.
type plate struct {
	Cell cell

	Record   string
	Name     string
	Family   string
	Contents string
	Size     int
	Price    int
	Per      string
	Share    string

	TipTitle string
	Tip      []string
	Title    string
	Hint     string

	// HasStep is false for the smallest vessel of a catalog, which has nothing to be compared
	// against. StepPrice is what the extra options cost, and it is the figure the ladder is
	// reviewed on.
	HasStep   bool
	StepSize  int
	StepPrice int

	Art      string
	Draw     string
	Borrowed bool
}

// group is one contents' ladder of vessels. Held is how many records the contents are drawn from.
type group struct {
	Contents string
	Held     int
	Goods    []plate
}

// Ladder is the sizes and prices as one phrase, for the tool's own report line.
func (g group) Ladder() string {
	out := ""
	for i, p := range g.Goods {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%d for %d", p.Size, p.Price)
	}
	return out
}

type page struct {
	Ground    string
	Style     map[string]int
	Count     int
	Seats     int
	Catalog   int
	Unheld    int
	Borrowed  int
	Unwritten int
	Groups    []group
	States    []cell
}
