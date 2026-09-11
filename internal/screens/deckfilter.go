package screens

// **The deck panel's filter: which cards the panel is being asked about.**
//
// The panel used to say three things in numbers under the grid — how much of each form, how much
// at each price, how much of each colour — and a player who read one of them had no way to ask the
// follow-up. "Ten crush" does not say how much of it is cheap, and "seven fire" does not say how
// much of that is crush. Every one of those counts is now a button, and pressing it marks the
// cards it counted.
//
// **Three axes, ANDed; the values inside one axis are ORed** *(owner's call, 2026-09-11)*. Crush
// with 2 AP with fire is the cards that are all three; crush with slash is either. That is the
// shape a deck question actually has — the axes narrow and the values inside one widen — and it is
// the only combining rule under which every button both adds and removes something.
//
// **Nothing here can change anything.** It is a reading preference over a picture of a deck, the
// same standing as deckView's two toggles and the hand's sort column: the piles are never touched,
// `ResolveRound` never sees it, and closing the panel leaves the fight where it was.

import (
	"sort"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// deckFilter is what the column's buttons have picked, one set per axis.
//
// **Maps rather than slices, and nil is the whole deck.** An axis nobody has touched holds nothing
// and matches everything, which is what makes `empty` a question about all three at once and what
// lets the zero value be the panel as it has always looked.
type deckFilter struct {
	forms    map[combat.Form]bool
	costs    map[int]bool
	elements map[cards.Element]bool
}

// deckAxis names one of the three, so a count can be taken with that axis set aside — see
// countingAxis. **A closed set of three**, not an open vocabulary: it is the axes a card *has*,
// and a fourth would be a fourth block of buttons rather than a value in this type.
type deckAxis int

const (
	axisNone deckAxis = iota
	axisForm
	axisCost
	axisElement
)

// on reports whether one value of one axis is picked.
func (f deckFilter) onForm(v combat.Form) bool      { return f.forms[v] }
func (f deckFilter) onCost(v int) bool              { return f.costs[v] }
func (f deckFilter) onElement(v cards.Element) bool { return f.elements[v] }

// empty is "the panel is not being asked anything", which is what stops every card in the grid
// being marked at once. **All three axes**, because a filter with one axis set is still a filter.
func (f deckFilter) empty() bool {
	return len(f.forms) == 0 && len(f.costs) == 0 && len(f.elements) == 0
}

// toggleForm, toggleCost and toggleElement are the three button presses. **A press that empties an
// axis leaves the map behind rather than nilling it**, which is deliberate: `empty` counts entries,
// not maps, so there is one answer to "is anything picked" and no second way to spell "no".
func (f *deckFilter) toggleForm(v combat.Form) {
	if f.forms == nil {
		f.forms = map[combat.Form]bool{}
	}
	toggleIn(f.forms, v)
}

func (f *deckFilter) toggleCost(v int) {
	if f.costs == nil {
		f.costs = map[int]bool{}
	}
	toggleIn(f.costs, v)
}

func (f *deckFilter) toggleElement(v cards.Element) {
	if f.elements == nil {
		f.elements = map[cards.Element]bool{}
	}
	toggleIn(f.elements, v)
}

// clear drops every axis. **The one control that is not a toggle**, and it exists because three
// axes of multi-select is easy to get into a state nobody meant and tedious to click back out of.
func (f *deckFilter) clear() {
	f.forms, f.costs, f.elements = nil, nil, nil
}

func toggleIn[K comparable](m map[K]bool, k K) {
	if m[k] {
		delete(m, k)
		return
	}
	m[k] = true
}

// matches is whether one card answers the filter, with one axis optionally set aside.
//
// **`except` is what makes a button's own count honest.** A count taken under the whole filter
// would read zero on every value of an axis that already has a selection — pick crush and slash
// reads 0, although pressing it would add fourteen cards. Counting an axis with its own selection
// ignored answers the question the number is actually being asked: *what would this button be
// worth*. Passing axisNone is the plain reading, which is what the grid's marking uses.
func (f deckFilter) matches(form combat.Form, cost int, element cards.Element, except deckAxis) bool {
	if except != axisForm && len(f.forms) > 0 && !f.forms[form] {
		return false
	}
	if except != axisCost && len(f.costs) > 0 && !f.costs[cost] {
		return false
	}
	if except != axisElement && len(f.elements) > 0 && !f.elements[element] {
		return false
	}
	return true
}

// deckCounts is the three blocks of figures beside the three blocks of buttons, each taken with
// its own axis set aside.
//
// **It counts the laid-out grid, exactly as the tallies it replaced did.** The grid is where the
// alterations toggle has already chosen a face and FULL/PLAYED has already chosen which cards are
// lit, so a count taken anywhere else would be a second answer to both questions.
type deckCounts struct {
	byForm    map[combat.Form]int
	byCost    map[int]int
	byElement map[cards.Element]int

	// costs is every cost the lit cards actually carry, ascending, so the AP block draws a button
	// per price the deck holds rather than a fixed 0..4 that goes stale the day a worm invents a
	// rung. **Taken with no axis set aside**, because a button for a price nothing in the picked
	// set has is a button that would be pressed and mark nothing.
	costs []int

	// total is how many cards answer the whole filter — the figure the form block's heading
	// carries, and the one that says whether the panel is showing you anything at all.
	total int
}

// countsOf walks a laid-out grid once per axis.
//
// **Only lit cards count**, which is the FULL/PLAYED toggle reaching the figures for free.
func countsOf(slots []pileSlot, holder combat.Duelist, f deckFilter) deckCounts {
	c := deckCounts{
		byForm:    map[combat.Form]int{},
		byCost:    map[int]int{},
		byElement: map[cards.Element]int{},
	}

	seen := map[int]bool{}
	for _, s := range slots {
		if !s.lit {
			continue
		}
		form := s.card.Form()
		cost := holder.CardCost(s.card)
		element := artFor(s.card.Element)

		if f.matches(form, cost, element, axisForm) {
			c.byForm[form]++
		}
		if f.matches(form, cost, element, axisCost) {
			c.byCost[cost]++
		}
		if f.matches(form, cost, element, axisElement) {
			c.byElement[element]++
		}
		if f.matches(form, cost, element, axisNone) {
			c.total++
		}
		seen[cost] = true
	}

	for cost := range seen {
		c.costs = append(c.costs, cost)
	}
	// Sorted, because Go randomises map order and a row of buttons that swapped places between
	// frames would be unpressable as well as unreadable.
	sort.Ints(c.costs)
	return c
}
