package decks

// The cheapest real hand a deck can form: the example a reference screen or a sheet shows
// beside a rung of the ladder.
//
// **It lives here rather than in a tool because two callers want it and one of them is the
// game.** The hands panel draws every rung with a worked example from the run's own deck, and
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

// Example is the set of cards that best *illustrates* a rung: cards that share what the rung
// counts on and differ in everything else, as cheaply as that can be done. It comes back with the
// action points it costs at the cards' own printed costs.
//
// **It was `CheapestExample` until 2026-08-24, and cheapest was the wrong question**
// *(owner's call)*. A deck holds four identical fire Stabs, so the cheapest Form Pair is two of
// them — which is also the cheapest *Card* Pair, and the cheapest *Elemental* Pair. Drawn as
// cards, all three rungs were the same picture, and the panel was silently claiming that a Form
// Pair wants two identical cards. **What a rung means is what it lets you get away with**: a stab
// for 1 AP beside a stab for 3 says a form pair does not care what you paid, and a fire stab
// beside a fire slash says an elemental pair does not care what shape it is.
//
// **So variety is the first key and cost is the tie-break.** Among the sets that demonstrate the
// rule equally well the cheapest is still chosen, because the budget is what decides whether a
// hand can actually be played — that half of the old argument survives intact.
//
// **The variety is only ever in what the rung does not count.** Every card in a group still
// carries the same value on the hand's own axis, so the set this returns is a set the matcher
// would score as that rung; see pickIllustrative.
//
// **A rung wanting more copies than the deck holds repeats one** *(owner's call, 2026-08-24)*.
// No attack concept ships more than its four elemental copies, so a Card Five of a Kind cannot be
// dealt — and this used to report that by coming back empty, which cost every caller a branch and
// put an apology on the panel where a picture belongs. **Whether a rung is reachable today is a
// fact about the deck, not about the ladder**: the rung exists, it pays what it pays, and a worm
// or a shop card could make it reachable tomorrow. So the illustration is always drawn, and a
// repeated card is the honest way to draw a hand of five from four.
//
// **It searches the whole deck rather than a dealt hand**, which is the difference from
// tools/handodds: that tool asks how often eight cards contain a rung, and this answers what the
// rung looks like when they do.
//
// The walk is over a sorted value list and never a map, per the determinism rules — the order
// decides which of two equally good sets is shown, and a map walk would show a different one
// every time the panel was opened.
func Example(deck []combat.Card, h combat.Hand) ([]combat.Card, int) {
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

	// The illustrative pick for each value, at each group size the rung asks for. Computed once
	// per value rather than inside the walk below, which would redo it for every arrangement of
	// the groups.
	type choice struct {
		cards []combat.Card
		cost  int
		shown int
	}
	picks := make([]map[int]choice, len(values))
	for i, v := range values {
		picks[i] = map[int]choice{}
		for _, want := range h.Groups {
			if _, done := picks[i][want]; done {
				continue
			}
			cs, cost := pickIllustrative(byValue[v], h.Match, want)
			picks[i][want] = choice{cards: cs, cost: cost, shown: variety(cs, h.Match)}
		}
	}

	var best []combat.Card
	bestCost, bestShown := -1, -1

	var fill func(group int, used []bool, picked []combat.Card, spent, shown int)
	fill = func(group int, used []bool, picked []combat.Card, spent, shown int) {
		if group == len(h.Groups) {
			// **Variety first, cost second.** Two sets that demonstrate the rule equally well are
			// separated by what they cost, which is the old rule surviving as the tie-break.
			if shown > bestShown || (shown == bestShown && spent < bestCost) {
				bestCost, bestShown = spent, shown
				best = append([]combat.Card(nil), picked...)
			}
			return
		}
		want := h.Groups[group]
		for i := range values {
			p, ok := picks[i][want]
			if used[i] || !ok {
				continue
			}
			used[i] = true
			fill(group+1, used, append(picked, p.cards...), spent+p.cost, shown+p.shown)
			used[i] = false
		}
	}
	fill(0, make([]bool, len(values)), nil, 0, 0)

	if bestCost < 0 {
		return nil, 0
	}
	return best, bestCost
}

// pickIllustrative chooses `want` cards out of a set that all carry the same value on the hand's
// axis, preferring the set that differs most in everything the hand does *not* count, and the
// cheapest of those.
//
// **Greedy rather than exhaustive, and that is a deliberate ceiling.** Twelve copies choose three
// 220 ways, times a value list, times the groups — and the answer a full search would find that
// this one misses is a set one attribute more varied than the one shown. Adding the card that
// contributes most that is not already on the row, cheapest first when nothing new is on offer,
// gets an illustration that makes the point.
//
// **A rung wanting more cards than the value has copies of takes one twice.** Every distinct card
// is spent first, so a repeat only ever appears once the row has said everything it can.
//
// The candidates arrive cheapest-first and ties are broken by the order they sit in, so two decks
// holding the same cards illustrate a rung the same way.
func pickIllustrative(cs []combat.Card, a combat.Axis, want int) ([]combat.Card, int) {
	var picked []combat.Card
	taken := make([]bool, len(cs))

	for len(picked) < want {
		bestAt, bestGain := -1, -1
		for i, c := range cs {
			if taken[i] {
				continue
			}
			// The gain is how much more of the rung's freedom the row would show with this card
			// on it. A first card gains nothing by definition, so the cheapest is taken.
			gain := variety(append(picked, c), a) - variety(picked, a)
			if gain > bestGain {
				bestAt, bestGain = i, gain
			}
		}
		if bestAt < 0 {
			// Every distinct card is on the row already and the rung wants more, so the pass
			// starts over and cards are repeated. See the note above.
			for i := range taken {
				taken[i] = false
			}
			continue
		}
		taken[bestAt] = true
		picked = append(picked, cs[bestAt])
	}

	cost := 0
	for _, c := range picked {
		cost += c.Cost()
	}
	return picked, cost
}

// variety is how much a set of cards says about what the rung leaves free: the distinct values it
// shows on each attribute the hand does not count, added up.
//
// **The hand's own axis is excluded**, because every card in a group carries the same value on it
// — counting it would add a constant to every candidate and say nothing. Cost is always counted:
// a stab for 1 AP beside a stab for 3 is the clearest statement a form pair can make about what
// it does not care about.
func variety(cs []combat.Card, a combat.Axis) int {
	if len(cs) == 0 {
		return 0
	}

	concepts, forms, elements, costs := map[int]bool{}, map[int]bool{}, map[int]bool{}, map[int]bool{}
	for _, c := range cs {
		concepts[int(c.Concept)] = true
		forms[int(c.Form())] = true
		elements[int(c.Element)] = true
		costs[c.Cost()] = true
	}

	n := len(costs) - 1
	if a != combat.AxisConcept {
		n += len(concepts) - 1
	}
	if a != combat.AxisForm {
		n += len(forms) - 1
	}
	if a != combat.AxisElement {
		n += len(elements) - 1
	}
	return n
}
