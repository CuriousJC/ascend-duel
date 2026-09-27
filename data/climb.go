package data

// **How many motifs a climb spends, and whether a roster can pay for them.**
//
// A floor after the first is reached through a portal room, and its boss opens two portals: two
// motifs, each in an element, and the player walks through one. **Both are spent** — a motif offered
// and passed over is never offered again in the run, so what a run code offers never depends on
// what the player picked before. Floor one is where a run begins and is offered nothing.
//
// That makes the climb's demand a list of slots — one for floor one, two for every floor above it —
// and the question every caller asks is whether the slots can each be given a distinct motif whose
// band allows that floor. MustBeClimbable asks it of the whole tower at load; internal/pyramid asks
// it of what is left after every draw, which is what stops a seeded roll from spending a motif a
// later floor needed.

// PortalOffers is how many motifs a portal room opens onto.
const PortalOffers = 2

// OffersOn is how many motifs floor `floor` is offered through, counting floors from one. **Floor
// one is offered one**: a run starts there rather than walking into it through a portal.
func OffersOn(floor int) int {
	if floor <= 1 {
		return 1
	}
	return PortalOffers
}

// ClimbSlots is the demand floors `from` to `to` make, one entry per motif to be offered, each entry
// the floor it is offered on.
func ClimbSlots(from, to int) []int {
	var out []int
	for f := max(from, 1); f <= to; f++ {
		for range OffersOn(f) {
			out = append(out, f)
		}
	}
	return out
}

// FillsSlots reports whether every slot — a floor number each — can be given its own motif, none of
// them in `used`, each one allowed on its slot's floor.
//
// **Bipartite matching by augmenting paths**, rather than a greedy walk: bands overlap in shapes a
// fewest-choices-first rule gets wrong once a floor takes two. The roster is tens of motifs and the
// tower tens of slots, so the textbook algorithm is instant. Motifs are walked in MotifOrder so the
// answer never depends on map order.
func FillsSlots(recs map[string]MotifData, slots []int, used map[string]bool) bool {
	var keys []string
	for _, key := range MotifOrder(recs) {
		if !used[key] {
			keys = append(keys, key)
		}
	}
	if len(slots) > len(keys) {
		return false
	}

	owner := make([]int, len(keys)) // which slot holds each motif, or -1
	for i := range owner {
		owner[i] = -1
	}

	var place func(slot int, seen []bool) bool
	place = func(slot int, seen []bool) bool {
		for k, key := range keys {
			if seen[k] || !recs[key].AllowsFloor(slots[slot]) {
				continue
			}
			seen[k] = true
			if owner[k] < 0 || place(owner[k], seen) {
				owner[k] = slot
				return true
			}
		}
		return false
	}

	for slot := range slots {
		if !place(slot, make([]bool, len(keys))) {
			return false
		}
	}
	return true
}

// PortalLines is what a portal says about this motif in an element: the motif's Text, then its
// ElementText for that element. **An unwritten line comes back as DrawUnwritten** rather than
// empty, so a portal with a gap in it shows the gap instead of a panel that looks finished.
func (m MotifData) PortalLines(element string) (motif, inElement string) {
	motif, inElement = m.Text, m.ElementText[element]
	if !written(motif) {
		motif = DrawUnwritten
	}
	if !written(inElement) {
		inElement = DrawUnwritten
	}
	return motif, inElement
}
