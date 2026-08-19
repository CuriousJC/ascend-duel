// handodds says how often each rung of the hand ladder can actually be built, which is what the
// multipliers in data/hands.json are priced against.
//
//	go run ./tools/handodds
//	go run ./tools/handodds -n 200000        # a quicker, noisier sample
//	go run ./tools/handodds -ap 8            # what a turn with a banked Prepare can afford
//
// **It exists for the reason tools/seeds does: the numbers are facts about one particular deck.**
// A hand's rarity is a function of how many cards share a concept, a form and an element, so
// changing `data/duelist_cards.json`, the hand size or the action budget silently invalidates every
// figure the ladder was tuned against. Re-run it after touching any of those, exactly as the seed
// catalogue is re-checked.
//
// **What it measures is reachability, not what the matcher would pick**: can this hand afford some
// set of cards forming this rung. That is the player's question — they choose what to queue — and
// it is deliberately a different question from `combat.BlowFor`, which reads a turn somebody has
// already committed to. It is therefore a small, honest reimplementation of *choosing*, and not a
// second copy of the matcher.
//
// The model and its limits, stated so a number is not read for more than it is worth:
//
//   - One hand dealt uniformly from the whole deck — round one. Later rounds draw from a depleted
//     pile and keep what was not spent, which this does not model.
//   - The real budget and the real bound: `Actions` from data/duelists.json, and five cards. `-ap`
//     overrides the budget, for the one rung the plain one cannot reach at all — see the flag.
//   - Attacks only, since a plan can never join a hand.
//
// It is not a test. There is nothing here to assert — the output is a table to tune against, and
// the tuning is a judgement call about how much a rarer hand should pay.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// oddsSeed pins the sample so two runs of an unchanged deck print the same table. A tool that
// printed a slightly different number every time could not be used to see whether a change moved
// anything. Per the determinism rules, it is an explicit source rather than the global one.
const oddsSeed = 20260819

func main() {
	trials := flag.Int("n", 2000000, "how many hands to deal")
	// **-ap is for the rung the fighter's own budget cannot reach at all.** Five cards of one
	// colour cost 7 AP at the cheapest — the colour has one card per form per tier — so an
	// elemental five of a kind is a 0% row against the 6 AP a round-one turn has, and a rung
	// measured at zero cannot be priced off `ln(1/P)`. Banking makes it real: a Prepare carries 2
	// AP into the next round, so `-ap 8` is what that turn can afford. **A number taken from a
	// run of this is a number about a banked turn and has to be labelled as one** — the ladder
	// itself is priced against the plain budget.
	budget := flag.Int("ap", budgetOf(), "action points a turn may spend")
	flag.Parse()

	deck := startingDeck()
	handSize := handSizeOf()

	fmt.Printf("%d attack cards of %d, hand of %d, %d AP, %d cards to a turn, %d hands dealt\n\n",
		countAttacks(deck), len(deck), handSize, *budget, maxCards, *trials)
	fmt.Printf("cards sharing a value: %d per concept, %d per form, %d per element\n\n",
		perValue(deck, combat.AxisConcept), perValue(deck, combat.AxisForm), perValue(deck, combat.AxisElement))

	rungs := builtHands()
	reach := make([]int, len(rungs))
	top := make([]int, len(rungs))
	nothing := 0

	rng := rand.New(rand.NewSource(oddsSeed))
	shoe := append([]combat.Card(nil), deck...)
	hand := make([]combat.Card, handSize)

	for t := 0; t < *trials; t++ {
		rng.Shuffle(len(shoe), func(i, j int) { shoe[i], shoe[j] = shoe[j], shoe[i] })
		copy(hand, shoe[:handSize])

		best, any := -1, false
		for i, h := range rungs {
			cost := minCost(hand, h)
			if cost < 0 || cost > *budget {
				continue
			}
			reach[i]++
			any = true
			// The rungs are walked in catalogue order, which is axis by axis and cheapest rung
			// first, so "the last one that fit" is the top of that axis's ladder only if the whole
			// axis is contiguous. Compare on the multiplier instead, which needs no such promise.
			if best < 0 || h.Multiplier > rungs[best].Multiplier {
				best = i
			}
		}
		if best >= 0 {
			top[best]++
		}
		if !any {
			nothing++
		}
	}

	pct := func(n int) float64 { return 100 * float64(n) / float64(*trials) }

	fmt.Printf("%-28s %6s %10s %10s %12s\n", "hand", "pays", "reachable", "1 in", "best in hand")
	lastAxis := combat.Axis(-1)
	for i, h := range rungs {
		if h.Match != lastAxis {
			fmt.Println()
			lastAxis = h.Match
		}
		p := pct(reach[i])
		one := 0.0
		if p > 0 {
			one = 100 / p
		}
		fmt.Printf("%-28s %6d %8.3f%% %10.0f %11.2f%%\n", h.Name, h.Multiplier, p, one, pct(top[i]))
	}
	fmt.Printf("\nno built hand of any kind: %.3f%% - the turns the High Card names\n", pct(nothing))
}

// maxCards is the count bound on a turn, read off the rules rather than repeated. It is a method on
// a duelist because a ring or a brand is meant to be able to raise it, so this asks a bare one.
var maxCards = combat.Duelist{}.MaxActions()

// builtHands is every rung of two cards or more, in catalogue order. The High Card is left out: it
// is the fallback rather than something a player reaches for, and it is reachable from any turn
// with an attack in it.
func builtHands() []combat.Hand {
	var out []combat.Hand
	for _, h := range combat.Hands() {
		if h.Cards() >= 2 {
			out = append(out, h)
		}
	}
	return out
}

// startingDeck is what a run opens with, built the way the combat screen builds it. It is not
// imported from there: `internal/screens` links Ebitengine and this tool has no window, which is
// the same wall `tools/balance` runs into.
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

// handSizeOf and budgetOf read the real numbers rather than repeating them. The hand size lives on
// the combat screen, which cannot be imported here, so it is the one figure written down twice — and
// it is printed in the header so a stale copy is visible rather than silent.
func handSizeOf() int { return 8 }

// budgetOf is the fighter's action points. **Named, never the first entry of the map** — the roster
// is keyed and Go randomises map order, so taking whichever came out first would make the table
// depend on nothing. tools/balance names the same record.
func budgetOf() int { return data.LoadDuelists()["Fighter1"].Actions }

func countAttacks(deck []combat.Card) int {
	n := 0
	for _, c := range deck {
		if c.Category() == combat.CategoryAttack {
			n++
		}
	}
	return n
}

// perValue is how many attack cards share the commonest value on an axis — 4 per concept, 12 per
// form, 9 per element in the shipping deck, which is the whole reason the three ladders are priced
// apart.
func perValue(deck []combat.Card, a combat.Axis) int {
	counts := map[int]int{}
	for _, c := range deck {
		if c.Category() != combat.CategoryAttack {
			continue
		}
		v, ok := valueOf(c, a)
		if !ok {
			continue
		}
		counts[v]++
	}
	most := 0
	for _, n := range counts {
		if n > most {
			most = n
		}
	}
	return most
}

// valueOf mirrors the matcher's own rule: a card with `FormNone` or `Basic` carries no value on
// that axis and cannot be counted on it.
func valueOf(c combat.Card, a combat.Axis) (int, bool) {
	switch a {
	case combat.AxisForm:
		f := c.Form()
		return int(f), f != combat.FormNone && f != combat.FormPlan
	case combat.AxisElement:
		return int(c.Element), c.Element != combat.Basic
	default:
		return int(c.Concept), true
	}
}

// minCost is the cheapest a hand can form this rung, or -1 if it cannot form it at all.
//
// **Cheapest, because the budget is the binding constraint.** A hand almost always holds the cards
// for a form pair; what decides whether one can be *played* is whether the two cheapest of them fit
// in the round's action points alongside nothing else. Spending the dearest copies would answer a
// question nobody asks.
func minCost(hand []combat.Card, h combat.Hand) int {
	if h.Cards() > maxCards {
		return -1
	}

	// costs by value on this hand's axis, each list sorted so a prefix sum is the cheapest N.
	byValue := map[int][]int{}
	for _, c := range hand {
		if c.Category() != combat.CategoryAttack {
			continue
		}
		v, ok := valueOf(c, h.Match)
		if !ok {
			continue
		}
		byValue[v] = append(byValue[v], c.Cost())
	}

	// A sorted walk, never a map walk — the values decide an output figure. See the randomness
	// skill; this is the same rule that governs EnemyOrder.
	values := make([]int, 0, len(byValue))
	for v := range byValue {
		values = append(values, v)
	}
	sort.Ints(values)

	prefix := make([][]int, len(values))
	for i, v := range values {
		costs := byValue[v]
		sort.Ints(costs)
		p := make([]int, len(costs)+1)
		for j, c := range costs {
			p[j+1] = p[j] + c
		}
		prefix[i] = p
	}

	best := -1
	var fill func(group int, used []bool, spent int)
	fill = func(group int, used []bool, spent int) {
		if group == len(h.Groups) {
			if best < 0 || spent < best {
				best = spent
			}
			return
		}
		want := h.Groups[group]
		for i := range values {
			if used[i] || len(prefix[i]) <= want {
				continue
			}
			used[i] = true
			fill(group+1, used, spent+prefix[i][want])
			used[i] = false
		}
	}
	fill(0, make([]bool, len(values)), 0)

	return best
}
