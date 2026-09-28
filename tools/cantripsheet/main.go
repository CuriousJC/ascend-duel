// Command cantripsheet renders every cantrip in data/cantrips.json to a PNG and writes an HTML
// page that shows each one beside the rule it fires and what that rule does to a real duelist.
//
//	go run ./tools/cantripsheet
//
// It is the rune sheet's idea on the catalog beside it. A cantrip is bought sealed in a bundle of
// scrolls and cast between the turns of a fight, so the whole catalog is several shop visits and a
// lot of luck away in a launched game.
//
// # It is a report, not a drawing-board
//
// This reads the real file through internal/session, so the catalog is *validated* before anything
// is drawn: an unknown effect, a Might adding nothing, an Endurance scaling to 100% or less all panic
// at init exactly as they would in the game.
//
// # What to look at
//
// **The line against the rule.** `Text` is what a player reads on resting on a cantrip, and nothing
// checks it against the effect. "+10 DMG" over a record carrying `Amount: 5` is the failure this page
// exists to make visible.
//
// **The worked example.** Every plate casts the cantrip onto the shipped duelist — at full life and
// wounded to half — through `session.Cantrip.Cast`, the function the combat screen calls. So the
// figures are the game's, not a second arithmetic, and "doubles your life" can be read against what
// doubling actually does to 30 of 60.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/cantripsheet/` and committed on the
// terms every other sheet is. A clone opens `docs/sheets/index.html`.
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

// ground is screens.screenGround, the light slate blue a cantrip is offered on.
const ground = "#a8bcd4"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "cantripsheet"),
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

	body, err := duelist()
	if err != nil {
		return err
	}

	// **The page walks the file's own order and the bundle walks the sorted one**, the split every
	// catalog sheet makes: nothing on this page decides an outcome.
	order := data.CantripFileOrder()

	page := page{
		Ground:  ground,
		Count:   len(order),
		Bundles: goodsPhrase(session.ContentsCantrips),
		Cap:     session.MaxConsumables,
		Body:    fmt.Sprintf("%d DMG, %d life", body.DMG, body.MaxLife),
	}

	var plates []plate
	for _, key := range order {
		c, ok := session.CantripByKey(key)
		if !ok {
			return fmt.Errorf("cantrips.json writes %q and internal/session resolved no such cantrip", key)
		}

		art, err := artwork(c.Art)
		if err != nil {
			return err
		}
		cell, err := write(dir, faces, specFor(c, art, true), "cantrip-"+c.Record+".png", c.Name)
		if err != nil {
			return err
		}
		plates = append(plates, plate{
			Cell:     cell,
			Record:   c.Record,
			Name:     c.Name,
			Text:     c.Text,
			Rule:     ruleLine(c),
			Examples: examples(c, body),
			Family:   c.Family,
			Draw:     c.Draw,
			Art:      c.Art,
			Default:  c.Art == data.DefaultCantripArt,
		})
		if c.Art == data.DefaultCantripArt {
			page.Undrawn++
		}
		if c.Draw == "" {
			page.Unwritten++
		}
	}
	page.Families = groupByFamily(plates)
	page.Effects = effectCounts(plates, order)

	// The two states a carried cantrip is drawn in: lit while the player plans, dim while a round
	// plays or on a screen where nothing can be cast.
	if len(order) > 0 {
		first, _ := session.CantripByKey(order[0])
		art, err := artwork(first.Art)
		if err != nil {
			return err
		}
		for _, s := range []struct {
			name, label string
			enabled     bool
		}{
			{"rest", first.Name + " — castable, while planning", true},
			{"disabled", first.Name + " — dim, mid-round or out of a fight", false},
		} {
			cell, err := write(dir, faces, specFor(first, art, s.enabled), "state-"+s.name+".png", s.label)
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

	fmt.Printf("wrote %s and %d PNGs — %d cantrips, %d with art of their own and %d with a subject; "+
		"bundles at %s\n",
		out, len(plates)+len(page.States), page.Count,
		page.Count-page.Undrawn, page.Count-page.Unwritten, page.Bundles)
	return nil
}

// duelist is the shipped duelist's own figures, off data/duelists.json — the body every example on
// the page is cast onto. **The record rather than an equipped fighter**: this page is drawn against
// no run, so a relic or a potion would be a run's opinion the page does not have.
func duelist() (combat.Duelist, error) {
	recs := data.LoadDuelists()
	if len(recs) == 0 {
		return combat.Duelist{}, fmt.Errorf("duelists.json is empty")
	}
	keys := make([]string, 0, len(recs))
	for k := range recs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	r := recs[keys[0]]
	return combat.Duelist{DMG: r.DMG, MaxLife: r.HP, CurrentLife: r.HP}, nil
}

// examples is the cantrip cast onto the duelist at full life and wounded to half, **through Cast**,
// so the page prints the game's figures rather than working them out a second time. A figure the
// cast did not move is left out of the line, which is what makes a Might's life line say nothing.
func examples(c session.Cantrip, body combat.Duelist) []string {
	hurt := body
	hurt.CurrentLife = body.MaxLife / 2

	var out []string
	for _, d := range []combat.Duelist{body, hurt} {
		now := c.Cast(d)
		var parts []string
		if now.DMG != d.DMG {
			parts = append(parts, fmt.Sprintf("DMG %d → %d", d.DMG, now.DMG))
		}
		if now.MaxLife != d.MaxLife || now.CurrentLife != d.CurrentLife {
			parts = append(parts, fmt.Sprintf("life %d/%d → %d/%d",
				d.CurrentLife, d.MaxLife, now.CurrentLife, now.MaxLife))
		}
		if len(parts) == 0 {
			parts = append(parts, "moves nothing")
		}
		out = append(out, strings.Join(parts, ", "))
	}

	// **Twice, because stacking is a rule** — each cast reads the duelist the last one left.
	twice := c.Cast(c.Cast(body))
	out = append(out, fmt.Sprintf("cast twice at full: DMG %d, life %d/%d",
		twice.DMG, twice.CurrentLife, twice.MaxLife))
	return out
}

// specFor is a cantrip as the card the pane draws — the fields ui.DrawCantripCard fills: a name and
// a picture, no form, no cost, no sentence.
func specFor(c session.Cantrip, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    c.Name,
		Form:    cards.FormNone,
		Element: cards.Basic,
		Art:     art,
		Enabled: enabled,
	}
}

// ruleLine is what the cantrip does, in the file's own vocabulary. **Deliberately not prose**, for
// the rune sheet's reason: the sentence a player reads is Text, printed beside this.
func ruleLine(c session.Cantrip) string {
	switch c.Effect {
	case session.CantripScaleLife:
		return fmt.Sprintf("%s %d%%: life and ceiling x%d/100, for the fight", c.Effect, c.Amount, c.Amount)
	default:
		return fmt.Sprintf("%s %d: +%d DMG, for the fight", c.Effect, c.Amount, c.Amount)
	}
}

// effectCounts is how many cantrips sit at each effect, every effect listed — an effect nobody has
// authored into is a mechanic built and never reached for.
func effectCounts(plates []plate, order []string) []effect {
	all := []session.CantripEffect{session.CantripAddDMG, session.CantripScaleLife}
	out := make([]effect, 0, len(all))
	for _, e := range all {
		n := 0
		for _, key := range order {
			if c, ok := session.CantripByKey(key); ok && c.Effect == e {
				n++
			}
		}
		out = append(out, effect{Name: e.String(), Count: n})
	}
	return out
}

// groupByFamily splits the catalog into the blocks its records are authored in, in the file's own
// order. A record with no Family lands under "unfamilied" rather than being dropped.
func groupByFamily(plates []plate) []family {
	var order []string
	byName := map[string][]plate{}
	for _, p := range plates {
		name := p.Family
		if name == "" {
			name = "unfamilied"
		}
		if _, seen := byName[name]; !seen {
			order = append(order, name)
		}
		byName[name] = append(byName[name], p)
	}

	out := make([]family, 0, len(order))
	for _, name := range order {
		f := family{Name: name, Cantrips: byName[name], Count: len(byName[name]), Noun: "cantrips"}
		if f.Count == 1 {
			f.Noun = "cantrip"
		}
		out = append(out, f)
	}
	return out
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

// artwork decodes one embedded picture. **A key that is in no embed is an error rather than a
// blank face**: a review tool that quietly drew nothing would be hiding exactly what it is for.
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

// goodsPhrase is every sealed good holding this catalog, as one sentence: "3 for 3, 4 for 5 or 5
// for 6 vitae". **Asked of the catalog rather than written down**, so the page cannot quote a price
// the shop does not charge.
func goodsPhrase(c session.GoodContents) string {
	var held []session.Good
	for _, key := range session.GoodKeys() {
		if g, ok := session.GoodByKey(key); ok && g.Contains == c {
			held = append(held, g)
		}
	}
	sort.Slice(held, func(i, j int) bool { return held[i].Size < held[j].Size })

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

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// plate is one cantrip: the card, and everything the file says about it.
type plate struct {
	Cell     cell
	Record   string
	Name     string
	Text     string
	Rule     string
	Examples []string

	// Family, Draw and Art are the three fields the engine ignores. Default says the picture is the
	// placeholder rather than one of its own.
	Family  string
	Draw    string
	Art     string
	Default bool
}

type effect struct {
	Name  string
	Count int
}

type family struct {
	Name     string
	Count    int
	Noun     string
	Cantrips []plate
}

type page struct {
	Ground    string
	Count     int
	Bundles   string
	Cap       int
	Body      string
	Undrawn   int
	Unwritten int
	Effects   []effect
	Families  []family
	States    []cell
}
