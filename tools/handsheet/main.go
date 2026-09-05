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
// # It prints two different numbers, and the page says which is which
//
// **What a rung costs** is what the example above it costs *once you hold the cards* — the cheapest
// action points it can be played for, and whether that fits a round. **How often you hold them** is
// the reachability beside it: two million hands dealt off this deck, counting how many could have
// afforded some set forming the rung. The multipliers are priced against the second one.
//
// **Both come from tools/hands, and so does tools/handodds's table** *(owner's call, 2026-09-05)*.
// This tool used to refuse to sample at all, on the argument that two tools reporting the same
// probability by different methods would be two numbers that can disagree. That argument was right
// and the odds belong on the sheet anyway — so the disagreement is made impossible instead of
// avoided: one method, one pinned sample, printed in two places. `go run ./tools/handodds` is still
// the tuning view, with the axes kept apart and the `-ap` flag.
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
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/tools/hands"
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

	deck := hands.StartingDeck()
	budget := hands.Budget()

	// **The sample is taken before a card is drawn**, and it is the slow part of this command —
	// two million hands, a few seconds. It is not optional and not cached: a cached table is a
	// table that can go stale against the deck beside it, which is the same failure a stale sheet
	// is. See tools/hands for why both commands deal the identical sample.
	odds := hands.Measure(deck, budget, hands.HandSize, hands.Trials)

	page := page{
		Ground:    ground,
		Style:     styleFacts(cards.Mini),
		Budget:    budget,
		MaxCards:  hands.MaxCards,
		DeckSize:  len(deck),
		Attacks:   hands.Attacks(deck),
		PerValue:  map[string]int{},
		HandSize:  odds.HandSize,
		Trials:    odds.Trials,
		Nothing:   pct(odds.Nothing),
		TooManyAP: 0,
	}
	for _, a := range []combat.Axis{combat.AxisConcept, combat.AxisForm, combat.AxisElement} {
		page.PerValue[a.String()] = hands.PerValue(deck, a)
	}

	// **Sorted by multiplier, then by cards, then by key.** The last two are only there so two
	// rungs paying the same amount land in the same order every run — the sort has to be total or
	// the page reshuffles itself between runs of an unchanged file.
	ladder := combat.Hands()
	sort.SliceStable(ladder, func(i, j int) bool {
		a, b := ladder[i], ladder[j]
		if a.Multiplier != b.Multiplier {
			return a.Multiplier < b.Multiplier
		}
		if a.Cards() != b.Cards() {
			return a.Cards() < b.Cards()
		}
		return a.Key < b.Key
	})

	drawn := map[string]cell{}
	for _, h := range ladder {
		r := row{
			Key:        h.Key,
			Name:       h.Name,
			Match:      h.Match.String(),
			Groups:     groupText(h.Groups),
			Cards:      h.Cards(),
			Multiplier: h.Multiplier,
			Pays:       fmt.Sprintf("%d.%02dx", h.Multiplier/100, h.Multiplier%100),
		}

		if o, ok := odds.Find(h.Key); ok {
			r.Sampled = true
			r.Dealt = pct(o.Dealt)
			r.Reachable = pct(o.Reachable)
			r.Score = pct(o.Score())
			r.OneIn = oneIn(o)
			r.Best = pct(o.Best)
			// **The meter is the score, not either column alone.** It is the figure the ladder is
			// priced on, so a rung whose bar is short should be a rung that pays a lot.
			r.Bar = bar(o.Score())
		}

		example, cost := decks.Example(deck, h)
		r.Cost = cost
		r.Affordable = cost <= budget && h.Cards() <= hands.MaxCards
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

	fmt.Printf("wrote %s and %d PNGs — %d hands, %d a round cannot play, %d hands sampled\n",
		out, len(drawn), len(page.Rows), page.TooManyAP, odds.Trials)
	for _, r := range page.Rows {
		fmt.Printf("  %6s  %-28s %-8s %-10s %-30s %8s dealt %9s playable %8s score\n",
			r.Pays, r.Name, r.Match, r.Groups, verdict(r, budget), r.Dealt, r.Reachable, r.Score)
	}
	return nil
}

// pct writes a percentage the way both the page and the terminal want it: three decimals, because
// the rarest rung on the ladder is six hands in a hundred thousand and two decimals round it to
// nothing.
func pct(v float64) string { return fmt.Sprintf("%.3f%%", v) }

// oneIn is the deal as odds — "1 in 260" — which is the form a rare rung is actually read
// in. A rung nothing reached is said in words rather than as one in nothing.
func oneIn(o hands.Odds) string {
	if o.OneIn() <= 0 {
		return "never"
	}
	return fmt.Sprintf("1 in %.0f", o.OneIn())
}

// bar is the reachability as a width, for the page's meter.
//
// **Square-rooted, not linear** *(2026-09-05)*. The ladder runs from 100% to 0.006%, so a linear
// bar draws every rung above a Full House as full and every rung below it as an invisible sliver —
// which is the whole interesting half of the ladder rendered as nothing. The root keeps the rare
// end visible while leaving the order intact; it is a picture of the ranking, and the figure beside
// it is the fact.
func bar(reachable float64) int {
	if reachable <= 0 {
		return 0
	}
	w := int(100 * math.Sqrt(reachable/100))
	if w < 1 {
		w = 1
	}
	return w
}

// verdict is the one-line answer for the terminal: what this rung's example costs, and whether a
// round can pay it.
func verdict(r row, budget int) string {
	switch {
	case !r.Affordable && r.Cards > hands.MaxCards:
		return fmt.Sprintf("%d cards — over the %d-card cap", r.Cards, hands.MaxCards)
	case !r.Affordable:
		return fmt.Sprintf("%d AP — over the %d a round has", r.Cost, budget)
	default:
		return fmt.Sprintf("%d AP", r.Cost)
	}
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
	case combat.Arcane:
		return cards.Arcane
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
	case combat.FormDefend:
		return cards.FormDefend
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

	// Affordable is whether a round can play the example: inside the action points and inside
	// the card cap.
	Affordable bool

	Cost   int
	Shares []string
	Cells  []cell

	// What the sample said about this rung. **Sampled is false for the High Card**, which is not
	// in the sample at all: it is the fallback every turn with an attack in it lands on, so a
	// reachability for it would be a number about nothing.
	Sampled   bool
	Dealt     string
	Score     string
	Reachable string
	OneIn     string
	Best      string

	// Bar is the meter's width in percent, square-rooted — see bar().
	Bar int
}

type page struct {
	Ground   string
	Style    map[string]int
	Budget   int
	MaxCards int
	DeckSize int
	Attacks  int
	PerValue map[string]int

	// The sample behind every reachability on the page: how big a hand was dealt, how many were
	// dealt, and how many of them built nothing at all.
	HandSize int
	Trials   int
	Nothing  string

	TooManyAP int

	Rows []row
}
