package decks

// The cheapest real hand a deck can form: the example a reference screen or a sheet shows
// beside a rung of the ladder.
//
// **It lives here rather than in a tool because two callers want it and one of them is the
// game.** The combos panel draws every rung with a worked example from the run's own deck, and
// tools/handsheet draws the same thing as pictures — so a second copy would be two answers to
// "what does an Elemental Full House look like in this deck", drifting the first time the deck
// changed. `decks` is where it belongs: it sits above `combat` and `data` and below any screen,
// which is the whole reason this package exists.
//
// It is **not** the matcher. `combat.BlowFor` decides what a played turn formed; this asks the
// opposite question — given a rung, which cards would build it most cheaply — and nothing in the
// rules reads it.

import (
	"sort"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// MatchValue is what one card counts as on an axis, and whether it counts at all. It mirrors the
// matcher's own rule: a card with no form or no colour carries no value on that axis, so it can
// never be counted on it.
//
// **A mirror rather than a call**, because the matcher's version is unexported and internal to how
// a turn is tallied. Three lines of its vocabulary is a smaller risk than exporting the rule; what
// would be a genuine fork is a second `matchCountOf`, and that is not here.
func MatchValue(c combat.Card, a combat.Axis) (int, bool) {
	switch a {
	case combat.AxisForm:
		f := c.Form()
		return int(f), f != combat.FormNone
	case combat.AxisElement:
		return int(c.Element), c.Element != combat.Basic
	default:
		return int(c.Concept), true
	}
}

// CheapestExample is the cheapest set of real cards in this deck that forms the rung, the action
// points it costs at the card's own printed cost, and whether the deck can form it at all.
//
// **Cheapest, because the budget is what actually binds.** A deck holds the cards for a form pair
// many times over; what decides whether one can be *played* is whether the two cheapest of them
// fit the round's action points. Showing the dearest copies would illustrate a hand nobody would
// queue.
//
// **It searches the whole deck rather than a dealt hand**, which is the difference from
// tools/handodds: that tool asks how often eight cards contain a rung, and this answers what the
// rung looks like when they do.
//
// The walk is over a sorted value list and never a map, per the determinism rules — the order
// decides which of two equally cheap sets is shown, and a map walk would show a different one
// every time the panel was opened.
func CheapestExample(deck []combat.Card, h combat.Hand) ([]combat.Card, int, bool) {
	byValue := map[int][]combat.Card{}
	for _, c := range deck {
		if v, ok := MatchValue(c, h.Match); ok {
			byValue[v] = append(byValue[v], c)
		}
	}

	values := make([]int, 0, len(byValue))
	for v := range byValue {
		values = append(values, v)
	}
	sort.Ints(values)

	// Within a value, cheapest first and then by concept, so a prefix of the list is both the
	// cheapest N copies and the same N every time.
	for _, v := range values {
		cs := byValue[v]
		sort.SliceStable(cs, func(i, j int) bool {
			if cs[i].Cost() != cs[j].Cost() {
				return cs[i].Cost() < cs[j].Cost()
			}
			if cs[i].Concept != cs[j].Concept {
				return cs[i].Concept < cs[j].Concept
			}
			return cs[i].Element < cs[j].Element
		})
	}

	var best []combat.Card
	bestCost := -1

	var fill func(group int, used []bool, picked []combat.Card, spent int)
	fill = func(group int, used []bool, picked []combat.Card, spent int) {
		if group == len(h.Groups) {
			if bestCost < 0 || spent < bestCost {
				bestCost = spent
				best = append([]combat.Card(nil), picked...)
			}
			return
		}
		want := h.Groups[group]
		for i, v := range values {
			cs := byValue[v]
			if used[i] || len(cs) < want {
				continue
			}
			cost := 0
			for _, c := range cs[:want] {
				cost += c.Cost()
			}
			used[i] = true
			fill(group+1, used, append(picked, cs[:want]...), spent+cost)
			used[i] = false
		}
	}
	fill(0, make([]bool, len(values)), nil, 0)

	if bestCost < 0 {
		return nil, 0, false
	}
	return best, bestCost, true
}
