// Command handsheet draws every rung of the hand ladder as an actual hand of cards, cheapest
// first, and writes an HTML page showing them together.
//
//	go run ./tools/handsheet
//
// It exists because the ladder is currently only readable as JSON. `data/hands.json` says a Card
// Full House is `[3,2]` on the concept axis for 425%, and what that *looks like* — which five
// cards, and whether a round can afford them — is arithmetic somebody has to do in their head
// against a deck they also have to remember. This does it and shows the cards.
//
// # It is a report, and it draws from the real deck
//
// Same posture as tools/ringsheet: every hand comes from `internal/combat`'s live catalogue and
// every example is built out of cards that are actually in `data/duelist_cards.json`. A rung the
// shipping deck cannot form at all is therefore visible as a rung with no example, which is a
// fact about the deck worth seeing rather than a gap to paper over with an invented card.
//
// # Ordered by multiplier, which is the ladder's own claim about difficulty
//
// The rows run cheapest-paying to dearest, across all three axes at once rather than axis by
// axis. That is the comparison the multipliers are making — an Elemental Three of a Kind at 195
// says it is worth about what a Form Full House is not — and it is exactly the comparison the
// file's axis-by-axis layout hides.
//
// **The multiplier is a claim, not a measurement.** What a rung actually costs to reach is
// `go run ./tools/handodds`, which deals two million hands and prints how often each one can be
// built. This tool deliberately does not sample: two tools reporting the same probability by
// different methods is two numbers that can disagree. What it prints instead is what a rung costs
// *once you hold the cards* — the cheapest action points it can be played for, and whether that
// fits a round — and it says on the page which question is which.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/handsheet/` and **committed**
// *(owner's call, 2026-08-23)*: the sheets are how the catalogues get reviewed, and requiring a
// Go toolchain to see one meant only whoever just changed something ever looked. A clone opens
// `docs/sheets/index.html`.
//
// The price is a directory of near-identical binaries rewritten on every run, so **regenerate
// deliberately** — `go run ./tools/sheets` does all of them and rewrites the index.
package main

import (
	"flag"
	"fmt"
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
	"github.com/curiousjc/ascend-duel/internal/decks"
)

// ground is the combat screen's table — screens.screenGround — because a hand is laid out on it.
const ground = "#e2d0b0"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "handsheet"),
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

	deck := startingDeck()
	budget := budgetOf()

	page := page{
		Ground:    ground,
		Style:     styleFacts(cards.Mini),
		Budget:    budget,
		MaxCards:  maxCards,
		DeckSize:  len(deck),
		Attacks:   countAttacks(deck),
		PerValue:  map[string]int{},
		Unbuilt:   0,
		TooManyAP: 0,
	}
	for _, a := range []combat.Axis{combat.AxisConcept, combat.AxisForm, combat.AxisElement} {
		page.PerValue[a.String()] = perValue(deck, a)
	}

	// **Sorted by multiplier, then by cards, then by key.** The last two are only there so two
	// rungs paying the same amount land in the same order every run — the sort has to be total or
	// the page reshuffles itself between runs of an unchanged file.
	hands := combat.Hands()
	sort.SliceStable(hands, func(i, j int) bool {
		a, b := hands[i], hands[j]
		if a.Multiplier != b.Multiplier {
			return a.Multiplier < b.Multiplier
		}
		if a.Cards() != b.Cards() {
			return a.Cards() < b.Cards()
		}
		return a.Key < b.Key
	})

	drawn := map[string]cell{}
	for _, h := range hands {
		r := row{
			Key:        h.Key,
			Name:       h.Name,
			Match:      h.Match.String(),
			Groups:     groupText(h.Groups),
			Cards:      h.Cards(),
			Multiplier: h.Multiplier,
			Pays:       fmt.Sprintf("%d.%02dx", h.Multiplier/100, h.Multiplier%100),
		}

		example, cost, ok := decks.CheapestExample(deck, h)
		r.Buildable = ok
		if ok {
			r.Cost = cost
			r.Affordable = cost <= budget && h.Cards() <= maxCards
			r.Shares = sharedValues(example, h.Match)
			for _, c := range example {
				cell, err := cardOnce(dir, faces, drawn, c)
				if err != nil {
					return err
				}
				r.Cells = append(r.Cells, cell)
			}
			if !r.Affordable {
				page.TooManyAP++
			}
		} else {
			page.Unbuilt++
		}

		page.Rows = append(page.Rows, r)
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

	fmt.Printf("wrote %s and %d PNGs — %d hands, %d the deck cannot form, %d a round cannot play\n",
		out, len(drawn), len(page.Rows), page.Unbuilt, page.TooManyAP)
	for _, r := range page.Rows {
		fmt.Printf("  %6s  %-28s %-8s %-6s %s\n", r.Pays, r.Name, r.Match, r.Groups, verdict(r, budget))
	}
	return nil
}

// verdict is the one-line answer for the terminal: what the cheapest copy of this rung costs, and
// whether a round can pay it.
func verdict(r row, budget int) string {
	switch {
	case !r.Buildable:
		return "the deck cannot form it at all"
	case !r.Affordable && r.Cards > maxCards:
		return fmt.Sprintf("%d cards — over the %d-card cap", r.Cards, maxCards)
	case !r.Affordable:
		return fmt.Sprintf("%d AP — over the %d a round has", r.Cost, budget)
	default:
		return fmt.Sprintf("%d AP", r.Cost)
	}
}

// maxCards is the count bound on a turn, read off the rules rather than repeated. It asks a bare
// duelist because a ring or a brand is meant to be able to raise it.
var maxCards = combat.Duelist{}.MaxActions()

// budgetOf is the fighter's action points. **Named, never the first entry of the map** — the
// roster is keyed and Go randomises map order, so taking whichever came out first would make the
// page depend on nothing. tools/handodds names the same record.
func budgetOf() int { return data.LoadDuelists()["Fighter1"].Actions }

// startingDeck is what a run opens with, built the way the combat screen builds it. It is not
// imported from there: internal/screens links Ebitengine and this tool has no window, which is
// the same wall tools/handodds runs into.
func startingDeck() []combat.Card {
	var out []combat.Card
	for _, rec := range data.LoadDuelistCards() {
		id, ok := combat.ConceptByKey(rec.Label)
		if !ok {
			panic("duelist_cards.json: the rules did not register a card called " + rec.Label)
		}
		for _, name := range rec.Elements {
			e, ok := combat.ParseElement(name)
			if !ok {
				panic("duelist_cards.json: " + rec.Label + " names unknown element " + name)
			}
			for i := 0; i < rec.Copies; i++ {
				out = append(out, combat.Of(id, e))
			}
		}
	}
	return out
}

func countAttacks(deck []combat.Card) int {
	n := 0
	for _, c := range deck {
		if c.Category() == combat.CategoryAttack {
			n++
		}
	}
	return n
}

// perValue is how many attack cards share the commonest value on an axis, which is the whole
// reason the three ladders are priced apart. Same figure tools/handodds prints in its header.
func perValue(deck []combat.Card, a combat.Axis) int {
	counts := map[int]int{}
	for _, c := range deck {
		if v, ok := decks.MatchValue(c, a); ok {
			counts[v]++
		}
	}
	most := 0
	for _, n := range counts {
		if n > most {
			most = n
		}
	}
	return most
}

// sharedValues names what the example's groups actually agree on, in the axis's own words —
// "three Strikes and two Cuts", "four fire". It is the caption that makes a row of five cards
// legible as a *hand* rather than as five cards.
func sharedValues(example []combat.Card, a combat.Axis) []string {
	type run struct {
		name string
		n    int
	}
	var runs []run
	for _, c := range example {
		name := axisName(c, a)
		if len(runs) > 0 && runs[len(runs)-1].name == name {
			runs[len(runs)-1].n++
			continue
		}
		runs = append(runs, run{name: name, n: 1})
	}

	out := make([]string, 0, len(runs))
	for _, r := range runs {
		out = append(out, fmt.Sprintf("%d× %s", r.n, r.name))
	}
	return out
}

func axisName(c combat.Card, a combat.Axis) string {
	switch a {
	case combat.AxisForm:
		return strings.ToLower(c.Form().String())
	case combat.AxisElement:
		return c.Element.String()
	default:
		return c.Label()
	}
}

func groupText(groups []int) string {
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		parts = append(parts, fmt.Sprint(g))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// cardOnce renders a card the first time it is wanted and reuses the file afterwards. The ladder
// asks for the same Strike in a dozen rows; eighteen rows of five cards is ninety renders of about
// twenty distinct pictures.
func cardOnce(dir string, f *cards.Faces, drawn map[string]cell, c combat.Card) (cell, error) {
	name := strings.ToLower(c.Label() + "-" + c.Element.String())
	if got, ok := drawn[name]; ok {
		return got, nil
	}

	spec := cards.Spec{
		Name:    c.Label(),
		Form:    form(c.Form()),
		Cost:    c.Cost(),
		Element: artFor(c.Element),
		Enabled: true,
	}
	got, err := write(dir, f, spec, cards.Mini, "card-"+name+".png",
		fmt.Sprintf("%s — %s, %d AP", c.Label(), c.Element.String(), c.Cost()))
	if err != nil {
		return cell{}, err
	}
	drawn[name] = got
	return got, nil
}

// artFor and form map the rules' enums onto the drawing package's, exactly as internal/screens
// does. Two enums, because internal/cards knows how to draw a card and nothing about how a round
// resolves.
func artFor(e combat.Element) cards.Element {
	switch e {
	case combat.Fire:
		return cards.Fire
	case combat.Ice:
		return cards.Ice
	case combat.Lightning:
		return cards.Lightning
	case combat.Earth:
		return cards.Earth
	default:
		return cards.Basic
	}
}

func form(f combat.Form) cards.Form {
	switch f {
	case combat.FormStab:
		return cards.FormStab
	case combat.FormSlash:
		return cards.FormSlash
	case combat.FormCrush:
		return cards.FormCrush
	case combat.FormPlan:
		return cards.FormPlan
	default:
		return cards.FormNone
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
	return cell{File: name, Label: label, Width: st.Width, Height: st.Height}, nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":        st.Width,
		"height":       st.Height,
		"cornerRadius": st.CornerRadius,
		"borderWidth":  st.BorderWidth,
	}
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// row is one rung of the ladder: what it pays, and the cheapest real hand that forms it.
type row struct {
	Key        string
	Name       string
	Match      string
	Groups     string
	Cards      int
	Multiplier int
	Pays       string

	// Buildable is whether the shipping deck holds the cards at all. **False is a finding, not an
	// error** — a rung wanting five copies of a concept cannot be built from a deck with four.
	Buildable bool

	// Affordable is whether a round can play the cheapest copy: inside the action points and
	// inside the card cap.
	Affordable bool

	Cost   int
	Shares []string
	Cells  []cell
}

type page struct {
	Ground   string
	Style    map[string]int
	Budget   int
	MaxCards int
	DeckSize int
	Attacks  int
	PerValue map[string]int

	Unbuilt   int
	TooManyAP int

	Rows []row
}
