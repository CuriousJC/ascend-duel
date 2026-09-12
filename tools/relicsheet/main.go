// Command relicsheet renders every relic in data/relics.json to a PNG and writes an HTML
// page that shows each one beside what it costs, what it says, and what it does.
//
//	go run ./tools/relicsheet
//
// It exists because a relic is the hardest collected thing to look at in a launched game:
// there are seventeen of them, a run wears at most five, and the shelf offers three the
// run is not already wearing, so seeing the whole catalogue means playing to a shop
// repeatedly and hoping. This draws all of them at once.
//
// # It is a report, and that is the difference from tools/cardsheet
//
// The card sheet deliberately writes its own contents out rather than reading the data
// files, because it is a drawing-board: it exists to show cards the rules cannot deal yet.
// This tool is the opposite and reads the real file, because the question it answers is
// "what does the catalogue actually hold" — a sheet of relics someone typed into a tool
// would answer nothing at all.
//
// It imports internal/session as well as data, which means the catalogue is *registered*
// before anything is drawn: an unknown moment, a verb at the wrong moment or a price of
// zero panics at init exactly as it would in the game. So the sheet cannot show a relic the
// game would refuse to start with, and the prices and sell-backs on it come from the shop's
// own rules rather than from arithmetic copied into a template.
//
// # What to look at
//
// **The art, and how much of it there is not.** Four relics have faces; the rest fall back
// to `default-relic.png`, and the page says which so a row of identical faces reads as a
// backlog rather than as a bug.
//
// **The authored line against the rules beside it.** `relics.json` carries a `Text` field
// that the hover tooltip prints verbatim, and nothing checks it against the rules — so a
// rule edited without its sentence is a relic that lies to the player. This page is the only
// place the two are visible together.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/relicsheet/` and **committed**
// *(owner's call, 2026-08-23)*: the sheets are how the catalogues get reviewed, and requiring a
// Go toolchain to see one meant only whoever just changed something ever looked. A clone opens
// `docs/sheets/index.html`.
//
// The price is a directory of near-identical binaries rewritten on every run, so **regenerate
// deliberately** — `go run ./tools/sheets` does all of them and rewrites the index.
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
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is the one the relics are actually drawn on — `screens.screenGround`, which is
// the fill behind both rows that hold a relic: the combat screen's worn row and the shop's
// shelf. Judging a card against a white browser page or a dark one would be the same
// failure as previewing art at a scale the game does not use.
const ground = "#a8bcd4"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "relicsheet"),
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

	records := data.LoadRelics()
	var plates []plate
	page := page{
		Ground: ground,
		Style:  styleFacts(cards.RelicStyle),
		Count:  len(records),
	}

	// **In the file's own order, not RelicOrder's sorted keys** *(2026-09-12)*. The catalogue is
	// authored in motif order — the flips together, the three ring families walking their ladders,
	// the weapons along the concept ladder — and sorting by key threw exactly that away, which is
	// what made an alphabetical page the wrong picture of a catalogue with families in it. File
	// order is as deterministic as sorted order and carries more.
	for _, key := range data.RelicFileOrder() {
		record := records[key]

		art, err := artwork(record.ArtKey())
		if err != nil {
			return err
		}
		spec := cards.Spec{
			Name:    record.Name,
			Element: cards.Relic,
			Art:     art,
			Counter: counterFor(key),
			Enabled: true,
		}
		cell, err := write(dir, faces, spec, cards.RelicStyle, "relic-"+key+".png", record.Name)
		if err != nil {
			return err
		}

		price, _ := session.RelicPrice(key)
		plates = append(plates, plate{
			Cell:   cell,
			Record: key,
			// **The page prints the full name and the card face does not** — the heading beside
			// a card is a place a relic is named in a sentence, which is exactly where the noun
			// still earns its place.
			Name:   record.Name,
			Text:   record.Text,
			Price:  price,
			Rarity: string(record.Rarity),
			Family: record.Family,
			Sell:   session.SellValue(key),
			Art:    record.Art,
			// **The subject paragraph, beside the picture it produced.** It lived in a worklist
			// under docs/art/ until 2026-09-12, struck the moment the art was filed — so the
			// words that made a picture went at the moment they became the record of how it was
			// made. It is a field on the record now, and this is where the two are read together.
			Draw:    record.Draw,
			Counter: spec.Counter,
			Default: record.Art == "",
			Rules:   ruleLines(record),
		})
		if record.Art == "" {
			page.Undrawn++
		}
		if record.Draw == "" {
			page.Unwritten++
		}
	}

	page.Tiers = groupByRarity(plates)
	page.Families = groupByFamily(plates)

	// The three states a relic card is drawn in, on one relic so the card underneath is
	// provably the same one. **Not "not owned"** — a relic the run has neither bought nor been
	// offered is not on screen at all, so that state does not exist to draw. The three that do
	// are the shelf's two, affordable and not, and the one the cursor is carrying.
	states, err := stateSpecs(records)
	if err != nil {
		return err
	}
	for _, st := range states {
		cell, err := write(dir, faces, st.spec, cards.RelicStyle,
			"state-"+st.name+".png", st.label)
		if err != nil {
			return err
		}
		page.States = append(page.States, cell)
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

	fmt.Printf("wrote %s and %d PNGs — %d of %d relics have art of their own and %d a subject\n",
		out, len(plates)+len(page.States), page.Count-page.Undrawn, page.Count,
		page.Count-page.Unwritten)
	for _, t := range page.Tiers {
		fmt.Printf("  %-9s %2d relics at %d vitae, sells for %d — %s%% of a shelf draw\n",
			t.Rarity, t.Count, t.Price, t.Sell, t.Share)
	}
	return nil
}

// stateSpecs is one relic in each of the three states a relic card is drawn in.
//
// **The relic it uses is whichever one has art**, picked off the catalogue rather than named
// here: a state row drawn on the default face would be showing what dimming does to a blank
// card, which is the one card where it is hardest to see.
func stateSpecs(records map[string]data.RelicData) ([]struct {
	name  string
	label string
	spec  cards.Spec
}, error) {
	var record data.RelicData
	for _, key := range data.RelicOrder(records) {
		if records[key].Art != "" {
			record = records[key]
			break
		}
	}
	if record.RelicRecord == "" {
		return nil, nil
	}

	art, err := artwork(record.ArtKey())
	if err != nil {
		return nil, err
	}
	base := cards.Spec{Name: record.Name, Element: cards.Relic, Art: art, Enabled: true}

	dragged := base
	dragged.Dragging = true

	dimmed := base
	dimmed.Enabled = false

	return []struct {
		name  string
		label string
		spec  cards.Spec
	}{
		{"rest", record.Name + " — worn, or on the shelf and affordable", base},
		{"disabled", record.Name + " — on the shelf, more vitae than the run holds", dimmed},
		{"dragging", record.Name + " — carried by the cursor", dragged},
	}, nil
}

// ruleLines turns a record's rules into one line each, in the file's own vocabulary.
//
// **Deliberately not prose.** The sentence a player reads is `Text`, which is printed beside
// these; generating a second English sentence from the rules would give the page two
// descriptions and no way to tell which one the game agrees with. These are the words in the
// file, laid out so the two can be compared.
func ruleLines(r data.RelicData) []string {
	out := make([]string, 0, len(r.Rules))
	for _, rule := range r.Rules {
		line := "when " + rule.When
		if cond := condition(rule.If); cond != "" {
			line += ", if " + cond
		}

		effects := make([]string, 0, len(rule.Then))
		for _, e := range rule.Then {
			effects = append(effects, effect(e))
		}
		out = append(out, line+" → "+strings.Join(effects, " and "))
	}
	return out
}

func condition(in *data.RelicIfData) string {
	if in == nil {
		return ""
	}
	var parts []string
	if in.Element != "" {
		parts = append(parts, in.Element)
	}
	if in.Form != "" {
		parts = append(parts, in.Form)
	}
	if in.Concept != "" {
		parts = append(parts, in.Concept)
	}
	return strings.Join(parts, " and ")
}

func effect(e data.RelicEffectData) string {
	switch {
	case e.Status != "":
		return e.Do + " " + e.Status
	case e.Element != "":
		return e.Do + " " + e.Element
	case e.Amount != 0:
		return fmt.Sprintf("%s %+d", e.Do, e.Amount)
	default:
		return e.Do
	}
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, st cards.Style, name, label string) (cell, error) {
	img, err := cards.Render(s, st, f)
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
	// **The image’s own size, not the style’s** *(2026-09-09)*. A relic card renders larger than the
	// card by Style.Bleed, so that the accumulator badge can hang off the bottom-right corner — and
	// a page quoting the style’s size would scale the picture down to fit it and show every relic
	// very slightly squashed.
	b := img.Bounds()
	return cell{File: name, Label: label, Width: b.Dx(), Height: b.Dy()}, nil
}

// artwork decodes one embedded picture. **A key that is in no embed is an error rather than a
// blank face**, unlike the game, which logs and draws the hole: the game must not refuse to
// start over a picture, and a review tool that quietly drew nothing would be hiding exactly
// what it is for.
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

// groupByRarity splits the catalogue into its three tiers, cheapest first.
//
// **In data.Rarities order rather than in whatever the file happens to hold**, so the page reads
// common → uncommon → rare every time and an empty tier still gets a heading — a tier nobody has
// authored into is a fact worth seeing, not a section to omit.
//
// **Share is the chance a single shelf draw lands in this tier**, as a whole percent: the tier's
// tickets over every relic's tickets. It is what turns "weight 10" into something reviewable — a
// tier holding half the catalogue at ten tickets each is a shelf that shows little else.
// groupByFamily splits the catalogue into the motifs its records are authored in.
//
// **In first-appearance order, which is the file's order**, so the page reads as data/relics.json
// does and a family lands where its siblings were written rather than where the alphabet puts it.
//
// **The mix is the rarity spread within the family**, printed on the heading. That is what keeps
// the pricing review the rarity grouping existed for: nearly every family is single-rarity, so a
// heading reading "15 relics, all common" answers "does one of these belong a tier up" for the
// whole block at once, and a family with a mix says so rather than hiding it. The three tier
// shares stay on the page header, where they are about the shelf rather than about a motif.
//
// A record with no Family lands under "unfamilied" rather than being dropped — an ungrouped relic
// is a thing to see, not a thing to omit.
func groupByFamily(plates []plate) []family {
	order := make([]string, 0, 8)
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
		f := family{Name: name, Relics: byName[name], Count: len(byName[name])}
		f.Noun = "relics"
		if f.Count == 1 {
			f.Noun = "relic"
		}
		counts := map[string]int{}
		for _, p := range f.Relics {
			counts[p.Rarity]++
		}
		parts := make([]string, 0, len(data.Rarities()))
		for _, r := range data.Rarities() {
			if n := counts[string(r)]; n > 0 {
				if n == f.Count {
					// **"all common" reads as a claim about a block, so a family of one just
					// names its tier** — "all uncommon" over a single relic is a sentence with
					// nothing to quantify.
					if f.Count == 1 {
						parts = append(parts, string(r))
					} else {
						parts = append(parts, "all "+string(r))
					}
				} else {
					parts = append(parts, fmt.Sprintf("%d %s", n, r))
				}
			}
		}
		f.Mix = strings.Join(parts, ", ")
		out = append(out, f)
	}
	return out
}

func groupByRarity(plates []plate) []tier {
	total := 0
	for _, p := range plates {
		total += data.Rarity(p.Rarity).Weight()
	}

	out := make([]tier, 0, len(data.Rarities()))
	for _, r := range data.Rarities() {
		t := tier{Rarity: string(r), Price: r.Price(), Sell: r.Sell(), Weight: r.Weight()}
		for _, p := range plates {
			if p.Rarity == string(r) {
				t.Relics = append(t.Relics, p)
			}
		}
		t.Count = len(t.Relics)
		if total > 0 {
			// **To a tenth of a percent, because the rare tier rounds to nothing otherwise.** Two
			// rare relics in a catalogue of forty-six is half a percent of a shelf seat, and a page
			// printing "0%" would say the tier is unreachable when what it is is scarce.
			t.Share = fmt.Sprintf("%.1f", float64(t.Count*r.Weight())*100/float64(total))
		}
		out = append(out, t)
	}
	return out
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":           st.Width,
		"height":          st.Height,
		"cornerRadius":    st.CornerRadius,
		"borderWidth":     st.BorderWidth,
		"artTop":          st.ArtTop,
		"artInset":        st.ArtInset,
		"artMaxH":         st.ArtMaxH,
		"counterHeight":   st.CounterHeight,
		"counterDiameter": 2 * st.CounterRadius,
		"counterRight":    st.CounterRight,
		"counterBottom":   st.CounterBottom,
		"counterSize":     int(st.CounterSize),
	}
}

// counterFor is the accumulator badge a fresh copy of this relic would wear — `1.0` for a relic
// that grows a multiplier, `+5` for one that grows a flat figure, and nothing at all for the rest
// of the catalogue, which is most of it.
//
// **It is `combat.CounterLabel` and not a second formatter**, which is the whole reason that
// function is exported: a sheet the sizing is judged on has to draw the figure the game draws,
// down to the decimal point. **Grown is zero on purpose** — this is the shelf's card, the one a
// player reads before buying, and it is also the widest the figure is *not*: see the page note.
func counterFor(key string) string {
	id, ok := combat.RelicByKey(key)
	if !ok {
		return ""
	}
	return combat.CounterLabel(combat.WornRelic{Relic: id})
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// plate is one relic: the card, and everything the file says about it.
type plate struct {
	Cell    cell
	Record  string
	Name    string
	Text    string
	Price   int
	Rarity  string
	Sell    int
	Art     string
	Draw    string
	Family  string
	Counter string
	Default bool
	Rules   []string
}

// tier is one rarity's worth of the catalogue: every relic at that price, with the tier's own
// numbers beside them.
//
// **The page is grouped by rarity as of 2026-08-22**, because that is the axis a review is actually
// conducted along: three tiers is the whole pricing decision, so what a reviewer needs to see is
// every common together and ask whether any of them belongs a tier up. An alphabetical list of
// forty-six relics answers a different question.
type tier struct {
	Rarity string
	Price  int
	Sell   int
	Weight int
	Count  int
	Share  string
	Relics []plate
}

// family is one motif's worth of the catalogue: every relic authored in that block, with the
// rarity spread across them.
type family struct {
	Name   string
	Count  int
	Noun   string
	Mix    string
	Relics []plate
}

type page struct {
	Ground    string
	Style     map[string]int
	Count     int
	Undrawn   int
	Unwritten int
	Tiers     []tier
	Families  []family
	States    []cell
}
