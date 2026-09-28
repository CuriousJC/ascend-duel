package session

// A consumable is anything a run carries into a fight and spends there: a rune, a stone, an
// essence and a cantrip.
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

	// ConsumableEssence is an essence out of the satchel: spent against one selected card, between
	// turns. **The same record the reward screen offers**, carried into a duel rather than aimed at
	// the deck the moment it is won.
	ConsumableEssence

	// ConsumableCantrip is a cantrip out of the scroll case: cast on nothing selected, between
	// turns, onto the fighter for the rest of the fight.
	ConsumableCantrip
)

// MaxConsumables is how many things the run can carry at once, **of every kind together**
// *(owner's call, 2026-09-28)*.
//
// **The cap is the pane's, not a kind's.** A run wears at most so many relics and carries at most
// so many consumables, and which kinds fill those seats is the player's business: two runes, a rune
// and a cantrip, two cantrips. It is a rule rather than a number the layout chose — the pane draws
// `held/2` on its corner, and a row saying two while the run carried a third is exactly the drift a
// displayed cap invites.
//
// **It is a rule about acquiring.** A purchase is refused against it — see Hold, HoldCantrip and the
// shop's seats, which go dim rather than taking vitae for something with no room — and nothing that
// hands the run a consumable without a purchase checks it: a fixture planting a pane, a save being
// resumed, a rock shower dropping stones into the pouch. The pane draws an over-full row as the
// `3/2` it is.
const MaxConsumables = 2

// ConsumableCount is how many things the run is carrying, every kind counted.
func (s *Session) ConsumableCount() int {
	return len(s.held) + len(s.satchel) + len(s.pouch) + len(s.scrolls)
}

// ConsumablesFull is whether the pane has no room. **Asked before anything carried is paid for**,
// which is the shop's business: see the pack seats, which go unavailable rather than taking vitae
// for a consumable that would be refused.
func (s *Session) ConsumablesFull() bool { return s.ConsumableCount() >= MaxConsumables }

// Consumable is one carried thing, ready to be drawn in a seat and spent out of it.
type Consumable struct {
	Kind ConsumableKind

	// At is the entry's index within its own kind's list — the sack seat for a rune, the pouch
	// position for a stone. **Not the seat in the pane**, which is this entry's place in the
	// merged row and is the caller's own loop variable.
	At int

	Rune    Rune
	Stone   Stone
	Essence Essence
	Cantrip Cantrip
}

// Name is what the card says it is, whichever kind it is.
func (c Consumable) Name() string {
	switch c.Kind {
	case ConsumableStone:
		return c.Stone.Name
	case ConsumableEssence:
		return c.Essence.Name
	case ConsumableCantrip:
		return c.Cantrip.Name
	default:
		return c.Rune.Name
	}
}

// Consumables is everything the run is carrying, in one row: the sack, then the satchel, then the
// scroll case, then the pouch.
//
// **What is aimed leads.** Spending a rune or an essence means selecting cards first, so both sit
// where the hand's own selection is being read toward; a stone needs nothing selected and reads the
// same wherever it stands, so it goes last. A cantrip needs nothing selected either, and sits
// between the two.
//
// A record key naming nothing in its catalog is dropped rather than drawn as a blank card, which is
// the same silence `heldRunes` has always kept — the loaders refuse an unknown key at the door, so
// one here means a save from a build that had a record this one does not.
func (s *Session) Consumables() []Consumable {
	out := make([]Consumable, 0, s.ConsumableCount())
	for i, key := range s.Held() {
		if p, ok := RuneByKey(key); ok {
			out = append(out, Consumable{Kind: ConsumableRune, At: i, Rune: p})
		}
	}
	for i, key := range s.Stowed() {
		if w, ok := essences[key]; ok {
			out = append(out, Consumable{Kind: ConsumableEssence, At: i, Essence: w})
		}
	}
	for i, key := range s.Scrolls() {
		if c, ok := cantrips[key]; ok {
			out = append(out, Consumable{Kind: ConsumableCantrip, At: i, Cantrip: c})
		}
	}
	for i, key := range s.Carried() {
		if st, ok := stones[key]; ok {
			out = append(out, Consumable{Kind: ConsumableStone, At: i, Stone: st})
		}
	}
	return out
}
