// Command seeds finds deck seeds that deal a particular opening hand, and re-checks the ones
// already catalogued in internal/screens.
//
// **A seed is an opening hand.** The shuffle is deterministic, so asking "give me a hand I can
// click a Strike Flurry out of" is answered by a number rather than by relaunching the game
// until one turns up. The catalogue that number lands in is `seedCatalog`, and this is the
// tool that fills it.
//
//	go run ./tools/seeds          # re-check the catalogue, then search for each shape
//	go run ./tools/seeds -n 50000 # search harder
//
// **Re-run it whenever `startingDeck` or `handSize` changes.** A seed is a fact about one
// particular deck; change the deck and every catalogued number silently becomes a hand nobody
// asked for. The re-check at the top is there to make that loud — a demo testing a Flurry
// against a hand that no longer holds three Strikes is worse than no demo, because it passes.
//
// It is a tool rather than a test for the same reason `tools/balance` is: the answer is a
// table to read and act on, not a pass or a fail. What *would* make a reasonable test one day
// is the re-check alone.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/screens"
)

// A want is a named shape of hand and the predicate that recognises it. The name matches the
// catalogue entry it fills, so the re-check and the search speak the same language.
type want struct {
	name  string
	desc  string
	match func(counts map[combat.ConceptID]int) bool
}

var wants = []want{
	{
		"three-strikes",
		"three or more Strikes: a Three of a Kind that can be clicked",
		func(c map[combat.ConceptID]int) bool { return c[combat.Strike] >= 3 },
	},
	{
		"four-strikes",
		"four Strikes: a Four of a Kind, the top of the ladder",
		func(c map[combat.ConceptID]int) bool { return c[combat.Strike] >= 4 },
	},
	{
		"three-smashes",
		"three or more Smashes: 9 AP, unaffordable, but the cards are there",
		func(c map[combat.ConceptID]int) bool { return c[combat.Smash] >= 3 },
	},
	{
		"both-verbs",
		"a plan card and an attack: both verbs in one round",
		func(c map[combat.ConceptID]int) bool {
			plans, attacks := 0, 0
			for a, n := range c {
				if combat.Plain(a).Category() == combat.CategoryPlan {
					plans += n
				} else {
					attacks += n
				}
			}
			return plans >= 1 && attacks >= 1
		},
	},
	{
		"all-plans",
		"a Prepare, a Plan and a Defend: the whole plan vocabulary in hand",
		func(c map[combat.ConceptID]int) bool {
			return c[combat.Prepare] >= 1 && c[combat.Plan] >= 1 && c[combat.Defend] >= 1
		},
	},
}

func main() {
	limit := flag.Int64("n", 20000, "how many seeds to search")
	flag.Parse()

	recheck()
	fmt.Println()
	search(*limit)
}

// recheck re-deals every catalogued seed and reports whether it still matches the shape it was
// catalogued for. **This is the half of the tool that matters most** — searching finds new
// numbers, but a catalogued number quietly going wrong is the failure that hides.
func recheck() {
	fmt.Println("catalogue")
	fmt.Println(strings.Repeat("-", 78))

	names, values, descs := screens.Seeds()
	bad := 0

	for i, name := range names {
		hand := screens.OpeningHand(values[i])
		counts := tally(hand)

		status := "  ??"
		if w, ok := wantByName(name); ok {
			if w.match(counts) {
				status = "  ok"
			} else {
				status = "FAIL"
				bad++
			}
		}

		fmt.Printf("%s  %-18s seed %-6d %s\n", status, name, values[i], handLabel(hand))
		fmt.Printf("      %s\n", descs[i])
	}

	if bad > 0 {
		fmt.Printf("\n%d catalogued seed(s) no longer deal what they claim.\n", bad)
		fmt.Println("The deck or the hand size changed. Take replacements from the search below")
		fmt.Println("and update seedCatalog in internal/screens/seeds.go.")
	}
}

// search walks seeds from 1 upward and reports the first few that match each shape. First
// rather than best: any matching hand does the job, and a low number is easier to read in a
// diff and to type than a large one.
func search(limit int64) {
	fmt.Printf("search (seeds 1..%d)\n", limit)
	fmt.Println(strings.Repeat("-", 78))

	const keep = 3
	found := make(map[string][]int64, len(wants))

	for seed := int64(1); seed <= limit; seed++ {
		counts := tally(screens.OpeningHand(seed))
		for _, w := range wants {
			if len(found[w.name]) >= keep {
				continue
			}
			if w.match(counts) {
				found[w.name] = append(found[w.name], seed)
			}
		}
	}

	for _, w := range wants {
		hits := found[w.name]
		if len(hits) == 0 {
			fmt.Printf("  %-18s none in %d seeds - try -n larger, or the deck cannot make it\n",
				w.name, limit)
			continue
		}
		fmt.Printf("  %-18s %v\n", w.name, hits)
		fmt.Printf("      %s\n", w.desc)
		fmt.Printf("      seed %d deals %s\n", hits[0], handLabel(screens.OpeningHand(hits[0])))
	}
}

func wantByName(name string) (want, bool) {
	for _, w := range wants {
		if w.name == name {
			return w, true
		}
	}
	return want{}, false
}

func tally(hand []combat.ConceptID) map[combat.ConceptID]int {
	counts := make(map[combat.ConceptID]int, len(hand))
	for _, a := range hand {
		counts[a]++
	}
	return counts
}

// handLabel renders a hand as counts in a fixed order. **Sorted by the player's concept list
// rather than by ranging the tally**, because Go randomises map iteration and a tool whose output
// reshuffled between runs could not be diffed. See CLAUDE.md.
func handLabel(hand []combat.ConceptID) string {
	counts := tally(hand)

	var parts []string
	for _, a := range combat.PlayerConcepts() {
		if n := counts[a]; n > 0 {
			parts = append(parts, fmt.Sprintf("%dx%v", n, combat.ConceptOf(a).Label))
		}
	}
	return strings.Join(parts, " ")
}
