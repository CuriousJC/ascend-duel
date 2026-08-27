package combat

// Stones: **the run's own opinion about what a hand is worth.**
//
// A hand's multiplier is written in `data/hands.json` and is a fact about the game. A *stone* is a
// fact about one run: it raises one rung of the ladder by a tenth of that rung's catalogue value,
// and it does it for the duelist holding it and nobody else.
//
// **The bump lives on the duelist rather than on the catalogue** *(owner's call, 2026-08-27)*.
// `handTable` is package state built at init, shared by every fight, every tool and every test —
// a run reaching in to raise Card Pair by 11 would raise it for the enemy planner, for
// `tools/handsheet` and for the next run in the same process. So a duelist carries a count per
// rung and the catalogue is read *through* it; a duelist with no stones reads the table itself,
// unchanged and unallocated.
//
// **Ten percent of the base, per stone, floored** *(owner's call, 2026-08-27)*. Card Pair is 115,
// so a stone is worth 11 and two stones are worth 22 — never 11.5 rounded up, and never 10% of
// the value the stone before it produced. The arithmetic is integer for the reason everything in
// this package is: a hand that rounded differently from the rest of the damage path would be the
// one number in the game whose sum could not be checked by hand.
//
// **A hand at a multiplier below its own rung is still legal**, exactly as `hands.json` allows,
// so nothing here clamps. What it will not do is grow without a stone: `HandStones` is the whole
// input.

// MaxHandSlots is the ceiling on how many rungs a duelist can carry a stone count for.
//
// **It exists because `Duelist` has to stay comparable**, exactly as `MaxStatuses` and
// `MaxWornRings` do — `TestRoundIsDeterministic` compares two resolved duelists with `==`, and a
// map on the struct would end that. Thirty-two is well clear of the nineteen rungs the catalogue
// holds; a catalogue that outgrew it panics at init rather than silently dropping the rungs past
// the end.
const MaxHandSlots = 32

// handSlots is each hand's seat in the boost array, by key, fixed at init from the catalogue's own
// order.
//
// **A seat is a position in `handTable`, never a `HandID`.** IDs are sparse — 1, then 10..15, then
// 20..25 — so indexing by one would want an array eight times the size, and it is the file's
// numbering rather than the rules', which is the sort of thing that moves.
//
// **It is never written down.** A seat is derived from the catalogue this build loaded, so a save
// file records the hand's *key* and resolves it back through here, on exactly the terms
// `ConceptID` and `StatusID` are under.
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

// HandSlot is the seat a hand's stone count sits in, and whether the catalogue holds that hand at
// all. **The bool is the validation**: a stone naming a rung this build has not got is refused by
// its caller rather than landing on seat zero, which is the High Card.
func HandSlot(key string) (int, bool) {
	i, ok := handSlots[key]
	return i, ok
}

// HandKeys is every hand's key, in catalogue order. It is what a catalogue of stones is checked
// against, and what a screen listing them walks.
func HandKeys() []string {
	out := make([]string, 0, len(handTable))
	for _, h := range handTable {
		out = append(out, h.Key)
	}
	return out
}

// stoneStep is what one stone adds to one rung: a tenth of the rung's catalogue multiplier,
// floored. Zero for a rung so cheap that a tenth of it rounds away — which the catalogue has none
// of, since the lowest multiplier in the game is the High Card's 100.
func stoneStep(base int) int { return base / 10 }

// StoneValue is what `n` stones are worth on a rung whose catalogue multiplier is `base`.
//
// **`n` steps, not one step compounded.** Each stone is worth a tenth of the number the file
// writes down, so the tenth stone is worth exactly what the first was.
func StoneValue(base, n int) int {
	if n <= 0 {
		return 0
	}
	return n * stoneStep(base)
}

// HandStoneCount is how many stones this duelist holds for one rung, by key. Zero for a rung the
// catalogue does not hold, which is a rung nothing could have put a stone on.
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
// every enemy, every test, a run that has bought nothing — read the catalogue itself rather than a
// copy of it.
func (d Duelist) anyHandStones() bool {
	for _, n := range d.HandStones {
		if n != 0 {
			return true
		}
	}
	return false
}

// HandTable is the ladder as this duelist plays it: the catalogue, with each rung raised by
// whatever stones are on it.
//
// **It is the one place a stone becomes a number**, so the resolver, the preview and the hands
// panel cannot come to three different answers about what a Card Pair pays.
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
		out[i].Multiplier += StoneValue(out[i].Multiplier, d.HandStones[seat])
	}
	return out
}

// BlowFor is what this duelist's turn amounts to, **read through their own stones**.
//
// It is what a screen previewing an attack calls. The bare `BlowFor` still exists and still reads
// the catalogue unaltered; it is the right answer only for a duelist holding no stones, which is
// why the preview on the combat screen goes through this one.
func (d Duelist) BlowFor(turn []Slot) Blow { return blowFor(turn, d.handsFrom(handTable)) }
