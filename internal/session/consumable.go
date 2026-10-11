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

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

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
// so many consumables, and which kinds fill those seats is the player's business: runes, cantrips,
// any mix. It is a rule rather than a number the layout chose — the pane draws `held/3` on its
// corner, and a row saying three while the run carried a fourth is exactly the drift a displayed cap
// invites.
//
// **It is a rule about acquiring.** A purchase is refused against it — see Hold, HoldCantrip and the
// shop's seats, which go dim rather than taking vitae for something with no room — and nothing that
// hands the run a consumable without a purchase checks it: a fixture planting a pane, a save being
// resumed, a rock shower dropping stones into the pouch. The pane draws an over-full row as the
// `4/3` it is.
const MaxConsumables = 3

// ConsumableCount is how many things the run is carrying, every kind counted, weightless included.
func (s *Session) ConsumableCount() int {
	return s.held.len() + s.satchel.len() + s.pouch.len() + s.scrolls.len()
}

// WeightedConsumables is how many carried things count against MaxConsumables: everything but the
// weightless ones. It is the figure the pane's corner reads.
func (s *Session) WeightedConsumables() int {
	return s.held.weighted() + s.satchel.weighted() + s.pouch.weighted() + s.scrolls.weighted()
}

// ConsumablesFull is whether the pane has no room. **Asked before anything carried is paid for**,
// which is the shop's business: see the pack seats, which go unavailable rather than taking vitae
// for a consumable that would be refused.
//
// **A weightless entry takes no seat**, so a pane carrying three plus any number of weightless
// copies is full and one carrying two plus ten is not.
func (s *Session) ConsumablesFull() bool { return s.WeightedConsumables() >= MaxConsumables }

// ConsumableSalePrice is what one carried thing fetches when the shop buys it back, whatever kind
// it is.
//
// **One figure for every kind.** A sealed good of four costs 5 and one of the four is kept, so a
// sale at 2 is a loss against what the thing cost to find — the relic rule, where selling never
// pays back the purchase. A record that should be worth more is a multiplier on that record in its
// own catalog, read over this base, rather than a second constant here.
const ConsumableSalePrice = 2

// SellConsumable takes one carried thing out of whichever list holds it and pays
// ConsumableSalePrice. It reports whether it was there.
//
// **By the entry's own position in its own list**, `At`, because every list may hold two of one
// record and a sale must not be ambiguous about which — the rule spending is under.
func (s *Session) SellConsumable(c Consumable) bool {
	var dropped bool
	switch c.Kind {
	case ConsumableRune:
		dropped = s.Drop(c.At)
	case ConsumableEssence:
		dropped = s.DropStowed(c.At)
	case ConsumableCantrip:
		dropped = s.DropScroll(c.At)
	case ConsumableStone:
		return s.SellCarried(c.At)
	}
	if dropped {
		s.AddVitae(ConsumableSalePrice)
	}
	return dropped
}

// Consumable is one carried thing, ready to be drawn in a seat and spent out of it.
type Consumable struct {
	Kind ConsumableKind

	// At is the entry's index within its own kind's list — the sack seat for a rune, the pouch
	// position for a stone. **Not the seat in the pane**, which is this entry's place in the
	// merged row and is the caller's own loop variable.
	At int

	// Weightless says this entry takes no seat in the pane — a copy the Eternity Pearl made. It is
	// spent, sold and drawn like any other, and floats where the others rest.
	Weightless bool

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
			out = append(out, Consumable{Kind: ConsumableRune, At: i, Rune: p, Weightless: s.held.weightless(i)})
		}
	}
	for i, key := range s.Stowed() {
		if w, ok := essences[key]; ok {
			out = append(out, Consumable{Kind: ConsumableEssence, At: i, Essence: w,
				Weightless: s.satchel.weightless(i)})
		}
	}
	for i, key := range s.Scrolls() {
		if c, ok := cantrips[key]; ok {
			out = append(out, Consumable{Kind: ConsumableCantrip, At: i, Cantrip: c,
				Weightless: s.scrolls.weightless(i)})
		}
	}
	for i, key := range s.Carried() {
		if st, ok := stones[key]; ok {
			out = append(out, Consumable{Kind: ConsumableStone, At: i, Stone: st,
				Weightless: s.pouch.weightless(i)})
		}
	}
	return out
}

// copyWeightless puts a weightless copy of one carried thing into its own kind's list, at the end.
// It reports whether the entry was there.
func (s *Session) copyWeightless(c Consumable) bool {
	var l *carriedList
	switch c.Kind {
	case ConsumableRune:
		l = &s.held
	case ConsumableEssence:
		l = &s.satchel
	case ConsumableCantrip:
		l = &s.scrolls
	case ConsumableStone:
		l = &s.pouch
	default:
		return false
	}
	if c.At < 0 || c.At >= l.len() {
		return false
	}
	l.add(l.keys[c.At], true)
	return true
}

// CopyAtFightStart is the `fight-begun` moment: for every copy the worn relics make, one carried
// entry is picked off `rng` — weightless copies included — and a weightless copy of it goes into
// its own kind's list. It hands back the copies, in the order they were made.
//
// **Once per fight.** The combat screen sets a fight up again when it is re-entered and when a run
// is resumed inside one, so the fight the pearl last fired in is kept on the run and a second call
// for the same fight makes nothing.
//
// **Nothing carried, nothing copied**, and the roll is not taken — an empty pane advances no stream.
func (s *Session) CopyAtFightStart(rng *rand.Rand) []Consumable {
	n := combat.CopiesAtFightStart(s.WornRelics())
	if n <= 0 || rng == nil || s.pearlFight == s.fight+1 {
		return nil
	}
	s.pearlFight = s.fight + 1

	var out []Consumable
	for k := 0; k < n; k++ {
		pool := s.Consumables()
		if len(pool) == 0 {
			break
		}
		pick := pool[rng.Intn(len(pool))]
		if s.copyWeightless(pick) {
			out = append(out, pick)
		}
	}
	return out
}
