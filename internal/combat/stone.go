package combat

import (
	"strconv"
	"strings"
)

// Stones: **the run's own opinion about what a hand is worth.**
//
// A hand's multiplier is written in `data/hands.json` and is a fact about the game. A *stone* is a
// fact about one run: it raises every rung of one *shape* — every Three of a Kind, say, whatever
// axis it counts on — each by a share of that rung's own catalog value, and it does it for the
// duelist holding it and nobody else. See Hand.Shape.
//
// **The count is still kept per rung.** A stone lands on each rung of its shape, so the rungs of
// one shape always carry the same count; keeping them apart costs nothing and leaves the ladder,
// the hands panel and a save file all reading one number per rung, as they always have.
//
// **The bump lives on the duelist rather than on the catalog** *(owner's call, 2026-08-27)*.
// `handTable` is package state built at init, shared by every fight, every tool and every test —
// a run reaching in to raise the Pair by 10 would raise it for the enemy planner, for
// `tools/handsheet` and for the next run in the same process. So a duelist carries a count per
// rung and the catalog is read *through* it; a duelist with no stones reads the table itself,
// unchanged and unallocated.
//
// **Ten percent of the base, per stone, with the total floored to the tenth** *(owner's call)*.
// Card Two Pair is 180, so its stones come to 18, 36, 54 and pay 1.9x, 2.1x, 2.3x: the fraction a
// single stone cannot show is carried into the next rather than thrown away, so a harder rung
// pulls ahead of an easier one however close their bases. Never 10% of the value the stone before
// it produced. A raised rung stays on the same tenths grid the catalog is written on.
//
// **The No Hand grows at thirty percent** *(owner's call)*, written as `stonePercent` on its
// record in `hands.json`: playing many cards that agree on nothing is the hard thing it rewards.
// Every other rung writes nothing and grows at `defaultStonePercent`.
//
// The arithmetic is integer for the reason everything in this package is: a hand that rounded
// differently from the rest of the damage path would be the one number in the game whose sum
// could not be checked by hand.
//
// **A hand at a multiplier below its own rung is still legal**, exactly as `hands.json` allows,
// so nothing here clamps. What it will not do is grow without a stone: `HandStones` is the whole
// input.
//
// **A stone is the run's *level* on a rung** *(owner's call, 2026-09-05)*, and it is one of two
// counters a run keeps against one hand. The other is how often the rung has been *played*, which
// lives on the run rather than here - see `session/play.go`. They are deliberately not derived
// from one another: this one is bought and moves the multiplier, that one is earned and moves
// nothing, and a single figure could not say which had happened.

// MaxHandSlots is the ceiling on how many rungs a duelist can carry a stone count for.
//
// **It exists because `Duelist` has to stay comparable** — `TestRoundIsDeterministic` compares two resolved duelists structurally, and a
// map on the struct would end that. Thirty-two is well clear of the eighteen rungs the catalog
// holds; a catalog that outgrew it panics at init rather than silently dropping the rungs past
// the end.
const MaxHandSlots = 32

// handSlots is each hand's seat in the boost array, by key, fixed at init from the catalog's own
// order.
//
// **A seat is a position in `handTable`, never a `HandID`.** IDs are sparse - 1, then 10, then
// 11..15, 21..25, 31..35 and 38 - so indexing by one would want an array twice the size, and it is
// the file's numbering rather than the rules', which is the sort of thing that moves.
//
// **It is never written down.** A seat is derived from the catalog this build loaded, so a save
// file records the hand's *key* and resolves it back through here, on exactly the terms
// `ConceptID` is under.
var handSlots = buildHandSlots()

func buildHandSlots() map[string]int {
	if len(handTable) > MaxHandSlots {
		panic("combat: hands.json holds more hands than a duelist can carry stones for")
	}
	out := make(map[string]int, len(handTable))
	for i, h := range handTable {
		out[h.Key] = i
	}
	return out
}

// HandSlot is the seat a hand's stone count sits in, and whether the catalog holds that hand at
// all. **The bool is the validation**: a stone naming a rung this build has not got is refused by
// its caller rather than landing on seat zero, which is the No Hand.
func HandSlot(key string) (int, bool) {
	i, ok := handSlots[key]
	return i, ok
}

// Shape is what a rung asks its cards to agree on, with the axis left out: `"3"` for every Three
// of a Kind, `"3+2"` for every Full House, `"1"` for the No Hand. It is the rung's `Groups`
// written as one string, so two rungs share a shape exactly when they want the same group sizes.
//
// **A shape is what a stone raises.** A Three of a Kind counted on the card, on the form and on
// the element is one idea read three ways, and a stone buys that idea rather than one reading of
// it — so it names a shape and every rung carrying it moves together, each by a share of its own
// multiplier. **Derived from `Groups` rather than written beside them**, so a rung cannot claim a
// shape its groups disagree with.
func (h Hand) Shape() string { return ShapeOf(h.Groups) }

// ShapeOf is the shape a list of group sizes spells. See Hand.Shape.
func ShapeOf(groups []int) string {
	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = strconv.Itoa(g)
	}
	return strings.Join(parts, "+")
}

// HandsShaped is every rung carrying one shape, by key, in catalog order. Empty for a shape the
// catalog does not hold, which is how a stone naming one is refused.
func HandsShaped(shape string) []string {
	var out []string
	for _, h := range handTable {
		if h.Shape() == shape {
			out = append(out, h.Key)
		}
	}
	return out
}

// HandShapes is every shape the ladder holds, once each, in the order its first rung appears.
// It is what a catalog of stones is checked against: a shape with no stone is a set of rungs that
// can never be raised.
func HandShapes() []string {
	var out []string
	seen := map[string]bool{}
	for _, h := range handTable {
		if sh := h.Shape(); !seen[sh] {
			seen[sh] = true
			out = append(out, sh)
		}
	}
	return out
}

// HandKeys is every hand's key, in catalog order. It is what a catalog of stones is checked
// against, and what a screen listing them walks.
func HandKeys() []string {
	out := make([]string, 0, len(handTable))
	for _, h := range handTable {
		out = append(out, h.Key)
	}
	return out
}

// defaultStonePercent is what one stone adds to a rung whose record writes no `stonePercent`: a
// tenth of its catalog multiplier.
const defaultStonePercent = 10

// stonePercent is what one stone adds to this rung, as a percentage of its catalog multiplier.
func (h Hand) stonePercent() int {
	if h.StonePercent == 0 {
		return defaultStonePercent
	}
	return h.StonePercent
}

// StoneValue is what `n` stones are worth on a rung, read off its catalog multiplier.
//
// **The total is floored to `handStep`, never each stone.** `n` stones are worth `n` times the
// rung's percentage of its base, and only the sum is put back on the tenths grid — so a 1.4x rung
// at a tenth per stone goes 1.5, 1.6, 1.8 rather than standing still beside the Pair. A stone may
// therefore be worth a step more than the one before it; over any run of stones the sum is the
// percentage exactly, to within one step.
func StoneValue(h Hand, n int) int {
	if n <= 0 {
		return 0
	}
	return h.Multiplier * h.stonePercent() * n / multiplierScale / handStep * handStep
}

// HandStoneCount is how many stones this duelist holds for one rung, by key. Zero for a rung the
// catalog does not hold, which is a rung nothing could have put a stone on.
func (d Duelist) HandStoneCount(key string) int {
	i, ok := handSlots[key]
	if !ok {
		return 0
	}
	return d.HandStones[i]
}

// WithHandStone returns this duelist holding one more stone for a rung, and reports whether the
// rung exists. **A copy rather than a mutation**, like `Wearing`: a `Duelist` is a value here and
// every rule in this package hands one back rather than changing one in place.
func (d Duelist) WithHandStone(key string) (Duelist, bool) {
	i, ok := handSlots[key]
	if !ok {
		return d, false
	}
	d.HandStones[i]++
	return d, true
}

// anyHandStones reports whether this duelist has a stone at all. It is what lets the common case —
// every enemy, every test, a run that has bought nothing — read the catalog itself rather than a
// copy of it.
func (d Duelist) anyHandStones() bool {
	for _, n := range d.HandStones {
		if n != 0 {
			return true
		}
	}
	return false
}

// HandTable is the ladder as this duelist plays it: the catalog, with each rung raised by
// whatever stones are on it.
//
// **It is the one place a stone becomes a number**, so the resolver, the preview and the hands
// panel cannot come to three different answers about what a Pair pays.
func (d Duelist) HandTable() []Hand { return d.handsFrom(handTable) }

// handsFrom applies this duelist's stones to a given ladder.
//
// **It takes the ladder rather than reading the global one** so the resolver's own `hands`
// parameter — which the tests replace — keeps meaning what it says.
func (d Duelist) handsFrom(hands []Hand) []Hand {
	if !d.anyHandStones() {
		return hands
	}
	out := make([]Hand, len(hands))
	copy(out, hands)
	for i := range out {
		seat, ok := handSlots[out[i].Key]
		if !ok {
			continue
		}
		out[i].Multiplier += StoneValue(out[i], d.HandStones[seat])
	}
	return out
}

// BlowFor is what this duelist's turn amounts to, **read through their own stones**.
//
// It is what a screen previewing an attack calls. The bare `BlowFor` still exists and still reads
// the catalog unaltered; it is the right answer only for a duelist holding no stones, which is
// why the preview on the combat screen goes through this one.
func (d Duelist) BlowFor(turn []Slot) Blow { return blowFor(turn, d.handsFrom(handTable)) }
