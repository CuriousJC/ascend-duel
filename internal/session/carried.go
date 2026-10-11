package session

// carried.go — one kind's carried consumables: record keys in acquisition order, each knowing
// whether it weighs anything.

// carriedList is the sack, the pouch, the satchel or the scroll case.
//
// **Weightless is a property of one entry, never of a record**, the rule a worn relic's wearing is
// under: two Agates in the pouch may be one that counts against the pane's cap and one that does
// not. So the flag travels with the entry through every drop and every reorder, and a position is
// still the only way to name one.
type carriedList struct {
	keys []string
	free []bool
}

// add appends one entry.
func (l *carriedList) add(key string, weightless bool) {
	l.keys = append(l.keys, key)
	l.free = append(l.free, weightless)
}

// drop takes the entry at i out, and reports whether it was there.
func (l *carriedList) drop(i int) bool {
	if i < 0 || i >= len(l.keys) {
		return false
	}
	l.keys = append(l.keys[:i], l.keys[i+1:]...)
	l.free = append(l.free[:i], l.free[i+1:]...)
	return true
}

// move slides the entry at `from` to `to`, its flag with it, and reports whether anything moved.
func (l *carriedList) move(from, to int) bool {
	n := len(l.keys)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return false
	}
	key, free := l.keys[from], l.free[from]
	if from < to {
		copy(l.keys[from:to], l.keys[from+1:to+1])
		copy(l.free[from:to], l.free[from+1:to+1])
	} else {
		copy(l.keys[to+1:from+1], l.keys[to:from])
		copy(l.free[to+1:from+1], l.free[to:from])
	}
	l.keys[to], l.free[to] = key, free
	return true
}

// list is a copy of the keys, in order.
func (l *carriedList) list() []string {
	out := make([]string, len(l.keys))
	copy(out, l.keys)
	return out
}

// len is how many entries there are, weightless included.
func (l *carriedList) len() int { return len(l.keys) }

// weighted is how many entries count against the pane's cap.
func (l *carriedList) weighted() int {
	n := 0
	for _, f := range l.free {
		if !f {
			n++
		}
	}
	return n
}

// weightless reports whether the entry at i takes no seat.
func (l *carriedList) weightless(i int) bool { return i >= 0 && i < len(l.free) && l.free[i] }

// lighten marks the entry at i weightless, for a snapshot being read back.
func (l *carriedList) lighten(i int) {
	if i >= 0 && i < len(l.free) {
		l.free[i] = true
	}
}

// freeAt is the positions of the weightless entries, for a snapshot.
func (l *carriedList) freeAt() []int {
	var out []int
	for i, f := range l.free {
		if f {
			out = append(out, i)
		}
	}
	return out
}
