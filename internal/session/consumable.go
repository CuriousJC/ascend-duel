package session

// A consumable is anything a run carries into a fight and spends there: a rune today, a stone
// beside it, and whatever else earns a seat later.
//
// **The pane is a consumables pane rather than a sack of runes** *(owner's call, 2026-09-19)*, so
// the vocabulary it reads is one kind with a discriminant rather than one list per thing carried.
// A new consumable is a `ConsumableKind`, a field on the struct and a case in the two screen
// tables that draw and spend one — never a second row, a second count and a second pane.
//
// **Each entry remembers where it came from.** `At` is the index into whichever list the run keeps
// that kind in, because those lists are what spending walks: a rune is dropped out of the sack by
// seat and a stone is taken out of the pouch by position, and both may hold two of the same record.

// ConsumableKind is which kind of carried thing an entry is. **Append-only**, like every other
// ordinal in this game, though nothing serializes it today — the run's own files still write the
// sack and the pouch as their own lists of record keys.
type ConsumableKind int

const (
	// ConsumableRune is a rune out of the sack: spent against selected cards, between turns.
	ConsumableRune ConsumableKind = iota

	// ConsumableStone is a stone out of the pouch: spent on nothing at all, since the rung it
	// raises is written on the record.
	ConsumableStone
)

// Consumable is one carried thing, ready to be drawn in a seat and spent out of it.
type Consumable struct {
	Kind ConsumableKind

	// At is the entry's index within its own kind's list — the sack seat for a rune, the pouch
	// position for a stone. **Not the seat in the pane**, which is this entry's place in the
	// merged row and is the caller's own loop variable.
	At int

	Rune  Rune
	Stone Stone
}

// Name is what the card says it is, whichever kind it is.
func (c Consumable) Name() string {
	if c.Kind == ConsumableStone {
		return c.Stone.Name
	}
	return c.Rune.Name
}

// Consumables is everything the run is carrying, in one row: the sack first, then the pouch.
//
// **The sack leads because a rune is the one that has to be aimed.** Spending a rune means
// selecting cards first, so the runes sit where the hand's own selection is being read toward; a
// stone needs nothing selected and reads the same wherever it stands.
//
// A record key naming nothing in its catalog is dropped rather than drawn as a blank card, which is
// the same silence `heldRunes` has always kept — the loaders refuse an unknown key at the door, so
// one here means a save from a build that had a record this one does not.
func (s *Session) Consumables() []Consumable {
	out := make([]Consumable, 0, len(s.held)+len(s.pouch))
	for i, key := range s.Held() {
		if p, ok := RuneByKey(key); ok {
			out = append(out, Consumable{Kind: ConsumableRune, At: i, Rune: p})
		}
	}
	for i, key := range s.Carried() {
		if st, ok := stones[key]; ok {
			out = append(out, Consumable{Kind: ConsumableStone, At: i, Stone: st})
		}
	}
	return out
}
