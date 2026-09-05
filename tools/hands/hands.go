// Package hands is the arithmetic the hand ladder is tuned against: the starting deck, the round's
// bounds, and how often each rung of the ladder can actually be built.
//
// **It is shared by tools/handodds and tools/handsheet, and that is the point** *(owner's call,
// 2026-09-05)*. The two tools were separate on purpose — the sheet deliberately did not sample,
// because "two tools reporting the same probability by different methods would be two numbers that
// can disagree". The odds now belong on the sheet, so the disagreement had to be made impossible
// rather than avoided: there is one method, in one place, and the two commands print the same run
// of it. Same argument as tools/roster, which is the other place a shared library beat two copies.
//
// **The sample is pinned**, so the two agree exactly rather than approximately: same seed, same
// trial count, same deck, therefore the same table to the last decimal. A tool that printed a
// slightly different number every run could not be used to see whether a change moved anything —
// and two tools that disagreed in the third decimal would be a bug report nobody could close.
//
// # What "reachable" means, and what it does not
//
// **Can this hand afford some set of cards forming this rung.** That is the player's question —
// they choose what to queue — and it is deliberately a different question from `combat.BlowFor`,
// which reads a turn somebody has already committed to. So this is a small, honest implementation
// of *choosing*, not a second copy of the matcher.
//
// The model and its limits, stated so a number is not read for more than it is worth:
//
//   - One hand dealt uniformly from the whole deck — round one. Later rounds draw from a depleted
//     pile and keep what was not spent, which this does not model.
//   - The real budget and the real bound: `Actions` from data/duelists.json, and five cards.
//   - **Every card counts, defences included** *(2026-08-23)*. They carry an element and a form and
//     join hands like anything else, bringing no damage — which is not measured here anyway. It is
//     reachability, and a hand of defences is as reachable as any other.
//
// Nothing here asserts anything. The output is a table to tune against, and the tuning is a
// judgement call about how much a rarer hand should pay.
package hands

import (
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
)

const (
	// Seed pins the sample. Per the determinism rules it is an explicit source rather than the
	// global one — see the randomness skill.
	Seed = 20260819

	// Trials is how many hands both commands deal. **Two million because of the rarest rung**: a
	// Card Five of a Kind turns up in about six hands in a hundred thousand, so a smaller sample
	// quotes its third decimal off a few dozen hits.
	Trials = 2000000

	// HandSize is what a fight opens with. It lives on the combat screen, which links Ebitengine
	// and cannot be imported by a tool with no window, so this is the one figure written down
	// twice — and both commands print it, so a stale copy is visible rather than silent.
	HandSize = 8
)

// MaxCards is the count bound on a turn, read off the rules rather than repeated. It asks a bare
// duelist because a ring or a brand is meant to be able to raise it.
var MaxCards = combat.Duelist{}.MaxActions()

// Budget is the fighter's action points. **Named, never the first entry of the map** — the roster
// is keyed and Go randomises map order, so taking whichever came out first would make every figure
// below depend on nothing.
func Budget() int { return data.LoadDuelists()["Fighter1"].Actions }

// StartingDeck is what a run opens with, built the way the combat screen builds it. It is not
// imported from there: internal/screens links Ebitengine and neither command has a window.
func StartingDeck() []combat.Card {
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

// Built is every rung of two cards or more, in catalogue order. The High Card is left out: it is
// the fallback rather than something a player reaches for, and it is reachable from any turn with
// an attack in it.
func Built() []combat.Hand {
	var out []combat.Hand
	for _, h := range combat.Hands() {
		if h.Cards() >= 2 {
			out = append(out, h)
		}
	}
	return out
}

// Attacks is how many of the deck can hit.
func Attacks(deck []combat.Card) int {
	n := 0
	for _, c := range deck {
		if c.Category() == combat.CategoryAttack {
			n++
		}
	}
	return n
}

// PerValue is how many cards share the commonest value on an axis, which is the whole reason the
// three ladders are priced apart.
//
// **It counts defences** *(2026-08-23)*, which is most of what it is: the element axis went from
// nine cards a colour to twelve when they were coloured, and the form axis gained a fourth value.
// It asks `decks.MatchValue` — the matcher's own rule — rather than reading the enums, so a card
// that carries no value on an axis is left out here for the same reason it is left out of a hand.
func PerValue(deck []combat.Card, a combat.Axis) int {
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

// Odds is one rung's share of the sample.
type Odds struct {
	Hand combat.Hand

	// Reachable is the percentage of hands that could build this rung inside the budget.
	Reachable float64

	// Best is the percentage of hands where this rung is the dearest-paying one reachable —
	// what a player who took the most they could would actually have landed.
	Best float64
}

// OneIn is the reachability as odds: 1 in N hands. Zero when nothing reached it, which the caller
// has to render as "never" rather than as one in nothing.
func (o Odds) OneIn() float64 {
	if o.Reachable <= 0 {
		return 0
	}
	return 100 / o.Reachable
}

// Table is a whole sample: what was dealt, and what came of it.
type Table struct {
	Trials   int
	Budget   int
	HandSize int
	DeckSize int

	// Nothing is the percentage of hands that built no rung at all — the turns the High Card names.
	Nothing float64

	Rungs []Odds
}

// Find is one rung's odds by key, and whether the sample holds it. The High Card is not in a
// sample, so a caller walking the whole catalogue has to be able to ask and be told no.
func (t Table) Find(key string) (Odds, bool) {
	for _, o := range t.Rungs {
		if o.Hand.Key == key {
			return o, true
		}
	}
	return Odds{}, false
}

// Measure deals the sample and counts what each hand could have built.
func Measure(deck []combat.Card, budget, handSize, trials int) Table {
	rungs := Built()
	reach := make([]int, len(rungs))
	top := make([]int, len(rungs))
	nothing := 0

	rng := rand.New(rand.NewSource(Seed))
	shoe := append([]combat.Card(nil), deck...)
	hand := make([]combat.Card, handSize)

	for t := 0; t < trials; t++ {
		rng.Shuffle(len(shoe), func(i, j int) { shoe[i], shoe[j] = shoe[j], shoe[i] })
		copy(hand, shoe[:handSize])

		best, any := -1, false
		for i, h := range rungs {
			cost := MinCost(hand, h)
			if cost < 0 || cost > budget {
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

	pct := func(n int) float64 { return 100 * float64(n) / float64(trials) }

	out := Table{
		Trials:   trials,
		Budget:   budget,
		HandSize: handSize,
		DeckSize: len(deck),
		Nothing:  pct(nothing),
	}
	for i, h := range rungs {
		out.Rungs = append(out.Rungs, Odds{Hand: h, Reachable: pct(reach[i]), Best: pct(top[i])})
	}
	return out
}

// MinCost is the cheapest a hand can form this rung, or -1 if it cannot form it at all.
//
// **Cheapest, because the budget is the binding constraint.** A hand almost always holds the cards
// for a form pair; what decides whether one can be *played* is whether the two cheapest of them fit
// in the round's action points alongside nothing else. Spending the dearest copies would answer a
// question nobody asks.
func MinCost(hand []combat.Card, h combat.Hand) int {
	if h.Cards() > MaxCards {
		return -1
	}

	// costs by value on this hand's axis, each list sorted so a prefix sum is the cheapest N.
	byValue := map[int][]int{}
	for _, c := range hand {
		v, ok := decks.MatchValue(c, h.Match)
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
