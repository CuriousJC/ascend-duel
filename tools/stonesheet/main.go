// Command stonesheet renders every stone in data/stones.json to a PNG and writes an HTML page
// that shows each one beside the rungs it raises and what raising them is worth.
//
//	go run ./tools/stonesheet
//
// It exists for the reason tools/essencesheet does. A stone arrives four at a time inside a sealed
// bag, one bag per shop visit and only if five vitae can be spared — so seeing the whole
// catalog in a launched game means buying a lot of rocks and being lucky about the draw. This
// draws all of them at once.
//
// # It is a report, not a drawing-board
//
// Same split as relicsheet against cardsheet: this reads the real file, through internal/session,
// which means the catalog is *validated* before anything is drawn. A stone naming a shape the
// rules have not got panics at init exactly as it would in the game, so a stone this page refuses
// to draw is a stone the game refuses to start with.
//
// # What to look at
//
// **One stone, several rungs, several figures.** A stone raises a *shape* — every Three of a
// Kind, on the card, the form and the element alike — and each rung moves by a tenth of its own
// catalog multiplier, so one rock is worth a different +N on every rung it touches. The figures are
// computed rather than authored, so a rung retuned in `hands.json` moves its row here without
// anything being edited in `stones.json`.
//
// **The gap between the cheap shapes and the dear ones.** Every stone costs the same inside the
// same bag, and a No Hand stone is worth a tenth of 100 where a Five of a Kind stone moves three
// rungs each by a tenth of a far larger number. Whether that spread is the intended bargain is
// a design question this page is for asking.
//
// **A shape with no stone.** Walked in ladder order, so a shape the catalog has not authored a
// stone for shows up as a gap rather than as an absence nobody notices.
//
// **The authored line against the shape it names.** `stones.json` carries a Text field and a
// Groups field side by side and nothing checks one against the other — a stone reading "raise
// FULL HOUSE" while keyed to `[2, 2]` would be invisible everywhere but here.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/stonesheet/` and **committed**, on the
// same terms as every other sheet. A clone opens `docs/sheets/index.html`.
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
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a stone is actually offered on. Judging a card
// against a white browser page would be the same failure as previewing art at a scale the game
// does not use.
const ground = "#a8bcd4"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "stonesheet"),
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

	page := page{
		Ground: ground,
		Style:  styleFacts(cards.EssenceStyle),
		Count:  len(session.Stones()),
		Bags:   goodsPhrase(session.ContentsStones),
	}
	page.Share = goodsShare(session.ContentsStones, page.Count)

	// **Walked by shape in ladder order rather than by stone**, so the page reads bottom rung to
	// top *and* a shape nobody authored a stone for is visible as a gap. Walking `session.Stones()`
	// would sort by record key and hide exactly that.
	var plates []plate
	for _, shape := range combat.HandShapes() {
		p := plate{Shape: shape}
		for _, h := range combat.Hands() {
			if h.Shape() != shape {
				continue
			}
			worth := session.StoneWorth(h.Key)
			p.Rungs = append(p.Rungs, rung{
				Hand: h.Name, HandKey: h.Key, Axis: axisLabel(h), CardsWanted: h.Cards(),
				Multiplier: h.Multiplier, Worth: worth, Raised: h.Multiplier + worth,
			})
		}
		page.Rungs += len(p.Rungs)

		st, ok := session.StoneForShape(shape)
		if !ok {
			plates = append(plates, p)
			page.Unstoned++
			continue
		}

		p.Has = true
		p.Record = st.Record
		p.Name = st.Name
		p.Text = st.Text
		p.DefaultArt = st.Art == data.DefaultStoneArt

		art, err := stoneFace(st)
		if err != nil {
			return err
		}
		cell, err := write(dir, faces, specFor(st, art, true), "stone-"+st.Record+".png", st.Name)
		if err != nil {
			return err
		}
		p.Cell = cell
		plates = append(plates, p)
	}
	page.Plates = plates

	// The two states a stone card is drawn in. **Not "chosen"** — the bag's dialog dims the three
	// that were not kept rather than lighting the one that was, exactly as the essence offer does.
	if first, ok := firstStone(plates); ok {
		st, _ := session.StoneByKey(first.Record)
		art, err := stoneFace(st)
		if err != nil {
			return err
		}
		for _, s := range []struct {
			name    string
			label   string
			enabled bool
		}{
			{"rest", st.Name + " — in the bag", true},
			{"disabled", st.Name + " — the rock not kept", false},
		} {
			cell, err := write(dir, faces, specFor(st, art, s.enabled), "state-"+s.name+".png", s.label)
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

	fmt.Printf("wrote %s and %d PNGs — %d stones over %d rungs, bags at %s, %s%% of the catalog a seat\n",
		out, page.Count+len(page.States), page.Count, page.Rungs, page.Bags, page.Share)
	if page.Unstoned > 0 {
		fmt.Printf("  %d shapes have no stone\n", page.Unstoned)
	}
	return nil
}

// stoneFace is the picture one stone draws.
//
// **The fallback is resolved before this is called** *(2026-09-16)*: `session.Stone.Art` comes
// through `data.StoneData.ArtKey`, which answers the catalog's default face for an unpainted
// record — so this page and the game cannot disagree about what an undrawn stone looks like, which
// is the whole reason that decision is in `data/`.
//
// **A key naming no embedded file is an error rather than a blank face**, exactly as the essence
// sheet's artwork is — a review tool that quietly drew nothing would hide what it is for.
func stoneFace(st session.Stone) (image.Image, error) {
	raw := assets.LoadImageData()[st.Art]
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s draws %q, which is in no embed", st.Record, st.Art)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", st.Art, err)
	}
	return img, nil
}

// specFor is a stone as the card the bag's dialog draws, and it fills the same fields
// ui.StoneSpec does: a name, the picture and no element. **Basic, not a color** — a stone raises a
// shape of the ladder and a shape is not one of the five, so its border is the mid gray
// `cards.BorderOf` gives `basic`. **No figure on the face**, because one stone moves several rungs
// by different amounts; the rows beside the card carry them.
func specFor(st session.Stone, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    st.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: cards.Basic,
		Art:     art,
		Enabled: enabled,
	}
}

// firstStone is the first shape on the ladder that actually has a stone, for the states row. It
// is a search rather than `plates[0]` because a No Hand nobody authored a stone for would otherwise
// draw the states row blank.
func firstStone(plates []plate) (plate, bool) {
	for _, p := range plates {
		if p.Has {
			return p, true
		}
	}
	return plate{}, false
}

// mergedLabel is what a rung read on more than one axis is filed under.
const mergedLabel = "any axis"

// axisLabel is the axis a rung counts on, as its row says it.
func axisLabel(h combat.Hand) string {
	if len(h.Axes) > 1 {
		return mergedLabel
	}
	return h.Match.String()
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

// plate is one shape of the ladder and the stone that raises it — or the absence of one, which is
// why every field about the stone is behind Has.
type plate struct {
	Cell  cell
	Shape string
	Rungs []rung

	Has        bool
	Record     string
	Name       string
	Text       string
	DefaultArt bool
}

// rung is one hand the plate's stone raises, and what one stone does to it.
type rung struct {
	Hand        string
	HandKey     string
	Axis        string
	CardsWanted int
	Multiplier  int
	Worth       int
	Raised      int
}

type page struct {
	Ground   string
	Style    map[string]int
	Count    int
	Rungs    int
	Unstoned int
	Bags     string
	Share    string
	Plates   []plate
	States   []cell
}

// goodsPhrase is every sealed good holding this catalog, as one sentence: "3 for 3, 4 for 5 or 5
// for 6 vitae". goodsShare is how much of the catalog gets a seat, as a figure or a range.
//
// **Asked of the catalog rather than written down**, so a page quoting what a sack costs cannot
// disagree with what the shop charges. **A phrase rather than a size and a price** *(2026-09-15)*:
// a catalog is held at three sizes now, and a page naming one of them would be quoting the shop
// accurately about a third of what it sells.
func goodsPhrase(c session.GoodContents) string {
	held := goodsHolding(c)
	parts := make([]string, 0, len(held))
	for _, g := range held {
		parts = append(parts, fmt.Sprintf("%d for %d", g.Size, g.Price))
	}
	switch len(parts) {
	case 0:
		return "nothing on the shelf holds them"
	case 1:
		return parts[0] + " vitae"
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " or " + parts[len(parts)-1] + " vitae"
	}
}

func goodsShare(c session.GoodContents, count int) string {
	held := goodsHolding(c)
	if count == 0 || len(held) == 0 {
		return ""
	}
	lo := float64(held[0].Size) * 100 / float64(count)
	hi := float64(held[len(held)-1].Size) * 100 / float64(count)
	if lo == hi {
		return fmt.Sprintf("%.1f", lo)
	}
	return fmt.Sprintf("%.1f-%.1f", lo, hi)
}

// goodsHolding is every good holding one catalog, smallest first.
func goodsHolding(c session.GoodContents) []session.Good {
	var out []session.Good
	for _, key := range session.GoodKeys() {
		if g, ok := session.GoodByKey(key); ok && g.Contains == c {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size < out[j].Size })
	return out
}
