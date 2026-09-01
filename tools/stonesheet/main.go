// Command stonesheet renders every stone in data/stones.json to a PNG and writes an HTML page
// that shows each one beside the rung it raises and what raising it is worth.
//
//	go run ./tools/stonesheet
//
// It exists for the reason tools/wormsheet does. A stone arrives four at a time inside a sealed
// bag, one bag per shop visit and only if five vitae can be spared — so seeing the whole
// catalogue in a launched game means buying a lot of rocks and being lucky about the draw. This
// draws all of them at once.
//
// # It is a report, not a drawing-board
//
// Same split as ringsheet against cardsheet: this reads the real file, through internal/session,
// which means the catalogue is *validated* before anything is drawn. A stone naming a rung the
// rules have not got panics at init exactly as it would in the game, so a stone this page refuses
// to draw is a stone the game refuses to start with.
//
// # What to look at
//
// **The +N against the rung beside it.** A stone's whole content is one number, and that number
// is computed rather than authored — `combat.StoneValue` is a tenth of the rung's catalogue
// multiplier — so this is the only place the ladder and what a rock does to it are visible
// together. A rung retuned in `hands.json` moves its stone's face here without anything being
// edited in `stones.json`, which is the point of the split and also the thing to sanity-check.
//
// **The gap between the cheap rungs and the dear ones.** Every stone costs the same five vitae
// inside the same bag, and a High Card stone is worth a tenth of 100 where a Five of a Kind
// stone is worth a tenth of a far larger number. Whether that spread is the intended bargain is
// a design question this page is for asking.
//
// **A rung with no stone.** Grouped by axis and walked in ladder order, so a hand the catalogue
// has not authored a stone for shows up as a gap rather than as an absence nobody notices.
// `loadStones` allows it; the game just never offers one.
//
// **The authored line against the rung it names.** `stones.json` carries a Text field the card
// prints verbatim, and nothing checks it against the `Hand` key beside it — a stone reading
// "raise FORM PAIR" while keyed to `form-two-pair` would be invisible everywhere but here.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/stonesheet/` and **committed**
// *(owner's call, 2026-08-23)*, on the same terms as every other sheet. A clone opens
// `docs/sheets/index.html`.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// ground is screens.screenGround, the cream a stone is actually offered on. Judging a card
// against a white browser page would be the same failure as previewing art at a scale the game
// does not use.
const ground = "#e2d0b0"

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

	// The boulder every stone card draws, rendered once. It is a generated glyph rather than a
	// file, which is the pattern this game reaches for first — see screens.stoneArt, which caches
	// it for the same arithmetic reason and is the code this mirrors.
	art := systems.RenderGlyph(systems.GlyphStone, systems.PaletteWhite)

	page := page{
		Ground:   ground,
		Style:    styleFacts(cards.WormStyle),
		Count:    len(session.Stones()),
		Rungs:    len(combat.Hands()),
		BagSize:  session.BagSize(),
		BagPrice: session.BagPrice(),
	}
	if page.Count > 0 {
		page.Share = fmt.Sprintf("%.1f", float64(page.BagSize)*100/float64(page.Count))
	}

	// **Walked by rung rather than by stone**, which is the one decision in this file. The
	// catalogue is one stone per rung and `StoneForHand` is a lookup rather than a choice, so
	// walking the ladder puts every stone in the table's own order for free *and* makes a rung
	// nobody authored a stone for visible as a gap. Walking `session.Stones()` would sort by
	// record key and hide exactly that.
	var plates []plate
	for _, h := range combat.Hands() {
		p := plate{
			Hand:        h.Name,
			HandKey:     h.Key,
			Axis:        h.Match.String(),
			Multiplier:  h.Multiplier,
			CardsWanted: h.Cards(),
		}

		st, ok := session.StoneForHand(h.Key)
		if !ok {
			plates = append(plates, p)
			page.Unstoned++
			continue
		}

		p.Has = true
		p.Record = st.Record
		p.Name = st.Name
		p.Text = st.Text
		p.Worth = session.StoneWorth(h.Key)
		p.Raised = h.Multiplier + p.Worth

		cell, err := write(dir, faces, specFor(st, art, true), "stone-"+st.Record+".png", st.Name)
		if err != nil {
			return err
		}
		p.Cell = cell
		plates = append(plates, p)
	}

	page.Groups = groupByAxis(plates)

	// The two states a stone card is drawn in. **Not "chosen"** — the bag's dialog dims the three
	// that were not kept rather than lighting the one that was, exactly as the worm offer does.
	if first, ok := firstStone(plates); ok {
		st, _ := session.StoneByKey(first.Record)
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

	fmt.Printf("wrote %s and %d PNGs — %d stones over %d rungs, %d drawn from a %d-vitae bag, %s%% of the catalogue a seat\n",
		out, page.Count+len(page.States), page.Count, page.Rungs,
		page.BagSize, page.BagPrice, page.Share)
	for _, g := range page.Groups {
		fmt.Printf("  %-8s %2d rungs, %2d stoned\n", g.Axis, len(g.Rungs), g.Stoned)
	}
	if page.Unstoned > 0 {
		fmt.Printf("  %d rungs have no stone\n", page.Unstoned)
	}
	return nil
}

// specFor is a stone as the card the bag's dialog draws, and it fills the same fields
// screens.stoneSpec does: a name, the authored line with the computed figure under it, and no
// element. **Basic, not a colour** — a stone raises a rung of the ladder and a rung is not one of
// the five, so its border is the mid grey `cards.BorderOf` gives `basic`.
func specFor(st session.Stone, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    st.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: cards.Basic,
		Art:     art,
		Text:    fmt.Sprintf("%s\n+%d", st.Text, session.StoneWorth(st.Hand)),
		Enabled: enabled,
	}
}

// firstStone is the first rung on the ladder that actually has a stone, for the states row. It is
// a search rather than `plates[0]` because a High Card nobody authored a stone for would otherwise
// draw the states row blank.
func firstStone(plates []plate) (plate, bool) {
	for _, p := range plates {
		if p.Has {
			return p, true
		}
	}
	return plate{}, false
}

// groupByAxis splits the ladder by what a rung counts on, in combat.AllAxes' order.
//
// **By axis rather than by multiplier across the whole catalogue**, which is the difference from
// tools/handsheet. That sheet interleaves all three axes deliberately, because a player choosing a
// hand is choosing among all of them at once. A stone is bought against one rung, so the question
// here is "is this axis' ladder priced sensibly against itself", and the rows have to be
// comparable for that.
func groupByAxis(plates []plate) []group {
	out := make([]group, 0, len(combat.AllAxes))
	for _, a := range combat.AllAxes {
		g := group{Axis: a.String()}
		for _, p := range plates {
			if p.Axis == a.String() {
				g.Rungs = append(g.Rungs, p)
				if p.Has {
					g.Stoned++
				}
			}
		}
		out = append(out, g)
	}
	return out
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, name, label string) (cell, error) {
	img, err := cards.Render(s, cards.WormStyle, f)
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
		Width: cards.WormStyle.Width, Height: cards.WormStyle.Height,
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

// plate is one rung of the ladder and the stone that raises it — or the absence of one, which is
// why every field about the stone is behind Has.
type plate struct {
	Cell cell

	Hand        string
	HandKey     string
	Axis        string
	Multiplier  int
	CardsWanted int

	Has    bool
	Record string
	Name   string
	Text   string
	Worth  int
	Raised int
}

// group is one axis' worth of the ladder.
type group struct {
	Axis   string
	Stoned int
	Rungs  []plate
}

type page struct {
	Ground   string
	Style    map[string]int
	Count    int
	Rungs    int
	Unstoned int
	BagSize  int
	BagPrice int
	Share    string
	Groups   []group
	States   []cell
}
