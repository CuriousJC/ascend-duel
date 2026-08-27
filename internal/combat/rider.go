package combat

// Riders: **a rule carried by one card rather than by the duelist.**
//
// A ring waits on a finger and fires for every card that matches it. A rider is the same idea
// aimed the other way: it belongs to one card of the run, travels with it through the shuffle,
// the hand and the discard, and fires only when that card is played. It is what a parasite
// leaves behind — see `internal/session/parasite.go`, which is the only thing that attaches one.
//
// **Why the kind is a Go enum and not a data record.** Everything else a parasite does happens to
// the *run*: a card is removed, a concept is swapped, a purse is filled. A rider is the one thing
// that has to be read while a round is resolving, and `internal/combat` is at the bottom of the
// graph and reads no JSON. So the vocabulary is closed here, exactly as `Verb` and `Element` are,
// and a parasite record naming a rider this build has not got is refused at init rather than
// attaching something that does nothing.
//
// **The amount rides on the card, not in a registry.** A rider is a kind plus a figure, and both
// sit in the card's own array — so the rules need no lookup table, no init ordering, and nothing
// to keep in step with a catalogue. It is the same trade `Card.CostDelta` already made.

// RiderKind is what a rider does when its card is played.
//
// **Append-only, and it may never be serialized as a number.** The run snapshot writes the
// *name* — see `session.Session.Snapshot` — for the reason every other ordinal in this package
// does: a file outlives the build that wrote it, and an ordinal in an old one will eventually
// mean something else.
type RiderKind int

const (
	// RiderNone is an empty seat. It is the zero value, which is what lets a plain `Card{}`
	// literal keep working and what makes an unridden card the common case it should be.
	RiderNone RiderKind = iota

	// RiderHealOnPlay restores life to the card's owner as the card is played. Amount is the
	// figure, in life.
	//
	// **It fires per card played, after a chill has taken what it takes.** A card a chill ate was
	// never played, so it heals nothing — which is what makes a rider on a cheap card that always
	// gets through worth something over one on the card at the front of a fat turn.
	RiderHealOnPlay
)

// RiderKinds is every kind in a fixed order, for anything that walks them.
func RiderKinds() []RiderKind { return []RiderKind{RiderHealOnPlay} }

func (k RiderKind) String() string {
	switch k {
	case RiderHealOnPlay:
		return "heal-on-play"
	default:
		return "none"
	}
}

// ParseRiderKind resolves a kind from its name, and reports failure rather than falling back to
// one. A parasite quietly attaching the wrong rider because its name was misspelled is a mechanic
// nobody designed — the same posture ParseVerb takes.
func ParseRiderKind(name string) (RiderKind, bool) {
	for _, k := range RiderKinds() {
		if k.String() == name {
			return k, true
		}
	}
	return RiderNone, false
}

// Rider is one rule attached to one card: what it does and how much of it.
//
// Comparable by construction, which is the whole reason it is a pair of ints — see Card.Riders.
type Rider struct {
	Kind   RiderKind
	Amount int
}

// MaxCardRiders is how many riders one card may carry.
//
// **A fixed array rather than a slice, and that is not an optimisation.** `combat.Card` must stay
// comparable: the combat screen caches a rendered face on a struct holding one, and
// TestRoundIsDeterministic compares rounds by value. A slice field would end both. The same
// constraint made `Duelist.Rings` a fixed array of WornRing, and this follows it.
//
// **Three, because the face has room for three badges** and a card whose face cannot say what it
// carries is the failure the whole alteration mechanic is written to avoid. It is a layout number
// as much as a rules one; raising it means finding the room first.
const MaxCardRiders = 3

// RiderList is the riders a card actually carries, in the order they were attached.
//
// **Attachment order is the order they fire**, on the same terms worn order is a rule for rings:
// it is the only order the player can see. Nothing today is order-sensitive — a heal is a heal
// whichever ran first — but the moment one is, the answer has to already be the visible one.
func (c Card) RiderList() []Rider {
	out := make([]Rider, 0, MaxCardRiders)
	for _, r := range c.Riders {
		if r.Kind == RiderNone {
			continue
		}
		out = append(out, r)
	}
	return out
}

// RiderCount is how many riders this card carries.
func (c Card) RiderCount() int {
	n := 0
	for _, r := range c.Riders {
		if r.Kind != RiderNone {
			n++
		}
	}
	return n
}

// AddRider attaches one, and reports whether there was room.
//
// **It stacks rather than merging.** Two heal riders on one card are two riders of ten, not one
// of twenty — which is what keeps the badge row honest about how many parasites have been spent
// on a card, and matches the way `amount` worms compound rather than replace.
func (c Card) AddRider(r Rider) (Card, bool) {
	if r.Kind == RiderNone {
		return c, false
	}
	for i, seat := range c.Riders {
		if seat.Kind == RiderNone {
			c.Riders[i] = r
			return c, true
		}
	}
	return c, false
}

// HealOnPlay is the life this card restores as it is played, summed over its riders.
//
// **Zero for almost every card in the game**, which is the shape every rider reader should take:
// ask the card, get nothing, carry on. Nothing branches on whether a card is ridden.
func (c Card) HealOnPlay() int {
	total := 0
	for _, r := range c.Riders {
		if r.Kind == RiderHealOnPlay {
			total += r.Amount
		}
	}
	return total
}
