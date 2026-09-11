// Command scenariodeck writes the `Deck` block of a scenario, so a fixture wanting "all slashes"
// or "half stab, half crush" is a command rather than a spreadsheet.
//
//	go run ./tools/scenariodeck -form slash -size 40
//	go run ./tools/scenariodeck -form stab,crush -size 40 -elements fire,ice
//	go run ./tools/scenariodeck -cost 1-2 -riders golden:5 -size 24
//
// It prints JSON to stdout, ready to paste into `internal/scenario/scenarios.json` as one entry's
// `"Deck"` value.
//
// # Why a generator and not a filter vocabulary in the fixture file
//
// The obvious alternative was `"DeckOf": {"Form": "slash", "Share": 50}` read by
// `internal/scenario` at launch. That is a *second card-selection language*, living in a debug
// fixture, which has to be kept in step with `data/duelist_cards.json` and with `internal/decks` —
// and being a debug fixture is exactly why nobody would notice when it drifted.
//
// A generator has the opposite shape. What lands in the file is the literal list the fixture
// already supports, so `scenarios.json` stays a thing you can read and check, `internal/scenario`
// grows no vocabulary, and the only thing that can go stale is a deck somebody generated once —
// which is visible in the diff, because it is written out.
//
// **So this tool is deliberately not wired into anything.** It writes to stdout and never touches
// a file: a tool that edited `scenarios.json` would be the thing that makes a generated deck feel
// like a live query again.
//
// # The filters
//
// One flag per axis, each one a comma-separated list, and an absent flag means "no opinion". They
// compose by intersection. **They are meant to be extended** *(owner's call, 2026-09-11)* — the
// point is not to guess every scenario anybody will want, it is that adding the next axis is one
// `flag.String` and one clause in `keep`.
//
// Everything a filter can name comes from `data/duelist_cards.json` through `internal/combat`'s
// registry, so a form or a card the game does not have fails here rather than at launch.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// defaultElements is the colour wheel a generated deck is dealt across when nobody named one.
// **All five, in the order `combat.Element` numbers them**, so a deck generated twice is the same
// deck — the determinism rule applies to a tool that writes a fixture exactly as it applies to the
// fixture itself.
var defaultElements = []string{"fire", "ice", "lightning", "earth", "arcane"}

func main() {
	forms := flag.String("form", "", "forms to include, comma separated: stab,slash,crush,defend")
	verbs := flag.String("verb", "", "verbs to include, comma separated: attack,defend,shield")
	names := flag.String("card", "", "card labels to include, comma separated: Jab,Cut")
	elements := flag.String("elements", "", "elements to deal across, comma separated (default: all five)")
	cost := flag.String("cost", "", "action-point band, as N or LOW-HIGH: 1, or 1-2")
	riders := flag.String("riders", "", "riders every generated card carries: golden:5, wild-element")
	size := flag.Int("size", 40, "how many cards the deck should hold")
	perLine := flag.Bool("split", false, "one line per card instead of one line per card-and-colour")
	flag.Parse()

	out, err := run(opts{
		forms:    list(*forms),
		verbs:    list(*verbs),
		names:    list(*names),
		elements: list(*elements),
		cost:     *cost,
		riders:   list(*riders),
		size:     *size,
		perLine:  *perLine,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(os.Stdout, out)
}

type opts struct {
	forms, verbs, names, elements, riders []string
	cost                                  string
	size                                  int
	perLine                               bool
}

func run(o opts) (string, error) {
	if o.size <= 0 {
		return "", fmt.Errorf("a deck of %d cards is not a deck", o.size)
	}

	lo, hi, err := band(o.cost)
	if err != nil {
		return "", err
	}

	if err := checkRiders(o.riders); err != nil {
		return "", err
	}

	elements := o.elements
	if len(elements) == 0 {
		elements = defaultElements
	}
	for _, e := range elements {
		if _, ok := combat.ParseElement(e); !ok {
			return "", fmt.Errorf("%q is not an element", e)
		}
	}

	picked, err := pick(o, lo, hi)
	if err != nil {
		return "", err
	}

	// **Round-robin across the picked cards and the chosen colours**, rather than an even split
	// with a remainder. A deck of 40 over 3 cards and 5 colours is fifteen combinations and forty
	// cards; walking them in order gives every combination either two copies or three, which is
	// the closest a deck of that size gets to "equal parts" without the tool having an opinion
	// about which combination deserves the spare card.
	type key struct {
		card    string
		element string
	}
	counts := map[key]int{}
	order := make([]key, 0, len(picked)*len(elements))
	for _, c := range picked {
		for _, e := range elements {
			k := key{c, e}
			order = append(order, k)
			counts[k] = 0
		}
	}
	for i := 0; i < o.size; i++ {
		counts[order[i%len(order)]]++
	}

	var b strings.Builder
	b.WriteString("    \"Deck\": [\n")
	first := true
	total := 0
	for _, k := range order {
		n := counts[k]
		if n == 0 {
			continue
		}
		total += n
		reps := 1
		if o.perLine {
			reps = n
			n = 1
		}
		for r := 0; r < reps; r++ {
			if !first {
				b.WriteString(",\n")
			}
			first = false
			b.WriteString(line(k.card, k.element, n, o.riders))
		}
	}
	b.WriteString("\n    ],\n")

	fmt.Fprintf(os.Stderr,
		"%d cards over %d card%s and %d colour%s, from %d matching the filters\n",
		total, len(picked), plural(len(picked)), len(elements), plural(len(elements)), len(picked))
	return b.String(), nil
}

// line is one `Deck` entry, written the way scenarios.json writes one: on a single line, because
// a deck list is read to be counted.
func line(card, element string, copies int, riders []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "      { \"Card\": %q, \"Element\": %q", card, element)
	if copies > 1 {
		fmt.Fprintf(&b, ", \"Copies\": %d", copies)
	}
	if len(riders) > 0 {
		b.WriteString(", \"Riders\": [")
		for i, r := range riders {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%q", r)
		}
		b.WriteString("]")
	}
	b.WriteString(" }")
	return b.String()
}

// pick is every card in `data/duelist_cards.json` that survives every filter, in the file's own
// order — which is ascending cost inside each form, so a generated deck reads as a ladder.
//
// **The file rather than `combat.AllConcepts()`**, because that registry also holds every enemy
// concept once `internal/decks` has been imported, and a duelist's deck may only hold the
// player's cards.
func pick(o opts, lo, hi int) ([]string, error) {
	forms, err := set(o.forms, "form", func(s string) bool {
		return s == "stab" || s == "slash" || s == "crush" || s == "defend"
	})
	if err != nil {
		return nil, err
	}
	verbs, err := set(o.verbs, "verb", func(s string) bool {
		return s == "attack" || s == "defend" || s == "shield"
	})
	if err != nil {
		return nil, err
	}
	named := map[string]bool{}
	for _, n := range o.names {
		if _, ok := combat.ConceptByKey(n); !ok {
			return nil, fmt.Errorf("%q is not a card in the player's deck", n)
		}
		named[n] = true
	}

	var out []string
	for _, c := range data.LoadDuelistCards() {
		if len(forms) > 0 && !forms[c.Form] {
			continue
		}
		if len(verbs) > 0 && !verbs[c.Verb] {
			continue
		}
		if len(named) > 0 && !named[c.Label] {
			continue
		}
		if c.Cost < lo || c.Cost > hi {
			continue
		}
		out = append(out, c.Label)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no card matches those filters — %s", available())
	}
	return out, nil
}

// set turns a comma list into a lookup, refusing a word the axis does not have. **An unknown word
// is an error rather than an empty result**, because a filter nobody spelled right silently
// generates the whole deck, which is the one failure a fixture generator must not have.
func set(in []string, axis string, ok func(string) bool) (map[string]bool, error) {
	out := map[string]bool{}
	for _, s := range in {
		if !ok(s) {
			return nil, fmt.Errorf("%q is not a %s", s, axis)
		}
		out[s] = true
	}
	return out, nil
}

// band parses the -cost flag: empty is every rung, `N` is exactly N, `LO-HI` is a range.
func band(s string) (int, int, error) {
	if s == "" {
		return 0, 1 << 30, nil
	}
	if lo, hi, found := strings.Cut(s, "-"); found {
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			return 0, 0, fmt.Errorf("%q is not a cost band", s)
		}
		b, err := strconv.Atoi(strings.TrimSpace(hi))
		if err != nil {
			return 0, 0, fmt.Errorf("%q is not a cost band", s)
		}
		if a > b {
			return 0, 0, fmt.Errorf("cost band %q runs backwards", s)
		}
		return a, b, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, 0, fmt.Errorf("%q is not a cost band", s)
	}
	return n, n, nil
}

// checkRiders resolves every rider name against the rules, so a misspelling fails here rather than
// producing a fixture that fails at launch. The spelling is `internal/scenario`'s — a kind, with a
// figure after a colon where the kind takes one.
func checkRiders(names []string) error {
	for _, name := range names {
		kind, amount, hasAmount := strings.Cut(name, ":")
		k, ok := combat.ParseRiderKind(kind)
		if !ok {
			return fmt.Errorf("%q is not a rider the rules have", kind)
		}
		if !k.CarriesAmount() {
			if hasAmount {
				return fmt.Errorf("the %s rider carries no figure, and %q gives it one", k, name)
			}
			continue
		}
		n, err := strconv.Atoi(amount)
		if err != nil || n <= 0 {
			return fmt.Errorf("the %s rider wants a figure and %q is not one", k, name)
		}
	}
	if len(names) > combat.MaxCardRiders {
		return fmt.Errorf("%d riders on one card, and a card holds %d", len(names), combat.MaxCardRiders)
	}
	return nil
}

// available is what the player's deck actually holds, for the message a failed filter prints. A
// tool that says "nothing matched" and stops is a tool that sends you to the JSON it exists to
// save you reading.
func available() string {
	byForm := map[string][]string{}
	for _, c := range data.LoadDuelistCards() {
		byForm[c.Form] = append(byForm[c.Form], fmt.Sprintf("%s(%d)", c.Label, c.Cost))
	}
	forms := make([]string, 0, len(byForm))
	for f := range byForm {
		forms = append(forms, f)
	}
	sort.Strings(forms)

	var parts []string
	for _, f := range forms {
		parts = append(parts, f+": "+strings.Join(byForm[f], " "))
	}
	return strings.Join(parts, "; ")
}

func list(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	out := strings.Split(s, ",")
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
