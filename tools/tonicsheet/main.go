// Command tonicsheet renders every tonic in data/tonics.json to a PNG and writes an HTML page that
// shows each one beside the rule it resolves to, and walks a handful of run codes through the
// journey to show what each realm would offer.
//
//	go run ./tools/tonicsheet
//
// A tonic is offered once per realm, one at a time, in an order the run code shuffles — so the
// whole catalog is a full run or several away in a launched game, and "does Olympian's ever turn up
// before Athlete's" is a question about the offer walk rather than about any one record.
//
// # It is a report, not a drawing-board
//
// This reads the real file through internal/session, so the catalog is *validated* before anything
// is drawn: an unknown effect, a requirement naming no record, a discount of 100% all panic at init
// exactly as they would in the game.
//
// # What to look at
//
// **The rule against the tooltip.** The monospace line is what the record resolves to; the lines
// under it are the tooltip the shop shows, which is the whole of what a player is told.
//
// **The walks.** Each sample run code is played through every realm twice — once buying every tonic
// it is offered, once buying none — through the same `OfferTonic` and `DrinkTonic` the shop calls,
// so a requirement that holds a tonic back, and the realm it comes back in, is read off the page.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/tonicsheet/` and committed on the terms
// every other sheet is.
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

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/journey"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a tonic is offered on.
const ground = "#a8bcd4"

// sampleCodes are the run codes the walks are drawn for. **Fixed rather than rolled**, so the page
// only changes when the catalog or the walk does; any codes would do.
var sampleCodes = []string{"000000", "00H602", "7YVCSR", "K4R7N2"}

// samplePrices are the list prices the Jeweller's example is worked on: a relic at each rarity, a
// sealed pack, the first two rerolls, and a tonic.
var samplePrices = []struct {
	What  string
	Price int
}{
	{"common relic", data.Rarity("common").Price()},
	{"uncommon relic", data.Rarity("uncommon").Price()},
	{"rare relic", data.Rarity("rare").Price()},
	{"sealed pack", 5},
	{"first reroll", 2},
	{"second reroll", 4},
	{"tonic", 10},
}

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "tonicsheet"),
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

	all := session.Tonics()
	page := page{Ground: ground, Count: len(all)}

	for _, t := range all {
		rec := record(t.Record)
		art, err := artwork(t.Art)
		if err != nil {
			return err
		}
		cell, err := write(dir, faces, specFor(t, art), "tonic-"+t.Record+".png", t.Name)
		if err != nil {
			return err
		}
		p := plate{
			Cell:     cell,
			Record:   t.Record,
			Name:     t.Name,
			Rule:     ruleLine(t),
			Tip:      tipLines(t),
			Requires: t.Requires,
			Price:    t.Price,
			Draw:     rec.Draw,
			Art:      t.Art,
			Default:  t.Art == data.DefaultTonicArt,
		}
		if p.Default {
			page.Undrawn++
		}
		if p.Draw == "" {
			page.Unwritten++
		}
		page.Plates = append(page.Plates, p)
	}

	for _, e := range session.TonicEffects() {
		n := 0
		for _, t := range all {
			if t.Effect == e {
				n++
			}
		}
		page.Effects = append(page.Effects, effect{Name: e.String(), Count: n})
	}

	for _, t := range all {
		if t.Effect != session.TonicDiscount {
			continue
		}
		for _, sp := range samplePrices {
			page.Prices = append(page.Prices, price{
				Tonic: t.Name, What: sp.What, List: sp.Price,
				Paid: max(1, sp.Price-session.DiscountCut(sp.Price, t.Amount)),
			})
		}
	}

	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	for _, code := range sampleCodes {
		w, err := walkRun(motifs, shape, code)
		if err != nil {
			return err
		}
		page.Walks = append(page.Walks, w)
	}
	page.Realms = shape.Realms

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("wrote %s and %d PNGs — %d tonics, %d with art of their own and %d with a subject\n",
		out, len(page.Plates), page.Count, page.Count-page.Undrawn, page.Count-page.Unwritten)
	return nil
}

// walkRun plays one run code through every fight of the journey twice, buying every tonic offered
// and buying none, and reports what each realm offered. **Through the shop's own calls** —
// WonFight, OfferTonic, DrinkTonic — so the page shows the game's walk rather than a second one.
func walkRun(motifs map[string]data.MotifData, shape data.JourneyData, code string) (walk, error) {
	seed, err := seeds.Parse(code)
	if err != nil {
		return walk{}, fmt.Errorf("sample code %q: %w", code, err)
	}
	w := walk{Code: code}
	for _, buy := range []bool{true, false} {
		s := session.Start(motifs, shape, seed)
		s.AddVitae(1 << 20)
		offers := make([]string, shape.Realms)
		for fight := 0; fight < shape.Realms*journey.FightsPerRealm; fight++ {
			s.WonFight(1, 1)
			t, ok := s.OfferTonic()
			if !ok {
				continue
			}
			realm := journey.RealmOf(fight)
			offers[realm-1] = t.Name
			if buy {
				s.DrinkTonic(t.Record)
			}
		}
		for i := range offers {
			if offers[i] == "" {
				offers[i] = "—"
			}
		}
		if buy {
			w.BuyAll = offers
		} else {
			w.BuyNone = offers
		}
	}
	return w, nil
}

// record is a tonic's raw record, for the field internal/session does not carry: Draw.
func record(key string) data.TonicData {
	for _, r := range data.LoadTonics() {
		if r.TonicRecord == key {
			return r
		}
	}
	return data.TonicData{}
}

// specFor is a tonic as the shop's seat draws it: a name and a picture, no form, no cost.
func specFor(t session.Tonic, art image.Image) cards.Spec {
	return cards.Spec{Name: t.Name, Form: cards.FormNone, Element: cards.Basic, Art: art, Enabled: true}
}

// ruleLine is what the tonic resolves to, in the file's own vocabulary.
func ruleLine(t session.Tonic) string {
	line := fmt.Sprintf("%s %d", t.Effect, t.Amount)
	if t.Requires != "" {
		line += ", requires " + t.Requires
	}
	return line + fmt.Sprintf(", %d vitae", t.Price)
}

// tipLines is what resting on the tonic in the shop says, less its price. **A copy of
// screens.tonicTip**, since a tool cannot import a package that links a window; a disagreement
// between the two is the page being stale.
func tipLines(t session.Tonic) []string {
	switch t.Effect {
	case session.TonicDiscount:
		return []string{fmt.Sprintf("everything in the shop costs %d%% less", t.Amount), "and at least 1 vitae less"}
	case session.TonicCostCut:
		return []string{fmt.Sprintf("your cards cost %d AP less", t.Amount), "but never less than 1 AP"}
	case session.TonicPressure:
		return []string{"every fight is 1 round shorter", fmt.Sprintf("and every vitae you earn is x%d", t.Amount)}
	case session.TonicHandSize:
		return []string{fmt.Sprintf("%d more card in every hand", t.Amount)}
	case session.TonicDiscards:
		return []string{fmt.Sprintf("%d more discard every round", t.Amount)}
	case session.TonicRelicSlots:
		return []string{fmt.Sprintf("wear %d more relic", t.Amount)}
	}
	return nil
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
	return cell{File: name, Label: label, Width: cards.EssenceStyle.Width, Height: cards.EssenceStyle.Height}, nil
}

// artwork decodes one embedded picture. **A key in no embed is an error rather than a blank face**.
func artwork(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded image called %q", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

type plate struct {
	Cell     cell
	Record   string
	Name     string
	Rule     string
	Tip      []string
	Requires string
	Price    int
	Draw     string
	Art      string
	Default  bool
}

type effect struct {
	Name  string
	Count int
}

type price struct {
	Tonic string
	What  string
	List  int
	Paid  int
}

type walk struct {
	Code    string
	BuyAll  []string
	BuyNone []string
}

type page struct {
	Ground    string
	Count     int
	Undrawn   int
	Unwritten int
	Realms    int
	Plates    []plate
	Effects   []effect
	Prices    []price
	Walks     []walk
}
