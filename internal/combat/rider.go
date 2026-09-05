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

	// RiderShieldOnPlay raises shields as the card is played. Amount is how many.
	//
	// **It is the one rider that gives an attack card a defence's job**, which is the point: a
	// Jab that also puts a shield up is a card doing two things, and the whole reason the owner
	// wanted it is that the game otherwise makes the player choose. It goes through
	// `Duelist.raiseShields`, so the five-shield cap holds exactly as it does for a Guard.
	RiderShieldOnPlay

	// RiderDamageOnPlay adds to the duelist's own DMG for the blow this card is played into.
	// Amount is the figure, in DMG.
	//
	// **It is a bonus to the duelist, not to the card.** The owner's words: "add 10 to the
	// duelist's base damage for that calculation and then use it for all of the calcs". So a +10
	// on one card of a five-card hand makes every term of that hand bigger, which is what makes
	// it worth more than a card that simply hits harder.
	RiderDamageOnPlay

	// RiderDamageInHand is RiderDamageOnPlay for a card that stayed *in the hand*. Amount is the
	// figure, in DMG.
	//
	// **The card is never played and never spent.** It pays into every turn it is still holding —
	// dealt on the first turn and kept for three, it paid three times. That is the whole shape of
	// the four in-hand riders: they reward not playing a card, which is the one axis the game had
	// nothing on.
	RiderDamageInHand

	// RiderScaleInHand scales the duelist's DMG for the blow while this card sits unplayed in the
	// hand. Amount is a percentage, so 200 is twice.
	RiderScaleInHand

	// RiderVitaeInHand pays vitae while this card sits unplayed in the hand. Amount is the figure.
	//
	// **The rules hold a copy of the purse, not the purse** *(owner's call, 2026-09-05)*. This
	// steps `Duelist.Vitae` so a ring that reads vitae sees it — Rampant — and announces a
	// KindVitae for the feed. The *run* is paid the difference between the purse the duel opened
	// with and the one it closes with, which is the same direction of travel every other run-level
	// consequence takes: `internal/combat` decides, and the layer that owns the thing being
	// changed does the changing.
	RiderVitaeInHand

	// RiderScaleInCombo scales the duelist's DMG for the blow when this card is one of the cards
	// the hand was formed from. Amount is a percentage, so 200 is twice.
	//
	// **Formed from, not merely played.** `Blow.Cards` is the scoring set, and a turn can play a
	// card that pays nothing into it — a lone Ward beside a pair, a third element in a two-card
	// hand. So this asks a question the player can lose: the card has to make the hand.
	RiderScaleInCombo
)

// RiderKinds is every kind in a fixed order, for anything that walks them.
func RiderKinds() []RiderKind {
	return []RiderKind{
		RiderHealOnPlay,
		RiderShieldOnPlay,
		RiderDamageOnPlay,
		RiderDamageInHand,
		RiderScaleInHand,
		RiderVitaeInHand,
		RiderScaleInCombo,
	}
}

func (k RiderKind) String() string {
	switch k {
	case RiderHealOnPlay:
		return "heal-on-play"
	case RiderShieldOnPlay:
		return "shield-on-play"
	case RiderDamageOnPlay:
		return "damage-on-play"
	case RiderDamageInHand:
		return "damage-in-hand"
	case RiderScaleInHand:
		return "scale-in-hand"
	case RiderVitaeInHand:
		return "vitae-in-hand"
	case RiderScaleInCombo:
		return "scale-in-combo"
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

// riderTotal sums one kind's amounts over a card's riders.
//
// **Every reader below is one line of this**, which is what keeps "ask the card, get nothing,
// carry on" the shape of every rider question: nothing branches on whether a card is ridden.
func (c Card) riderTotal(kind RiderKind) int {
	total := 0
	for _, r := range c.Riders {
		if r.Kind == kind {
			total += r.Amount
		}
	}
	return total
}

// riderScale is one kind's percentages over a card's riders, compounded.
//
// **Compounded rather than added**, which is the posture `TargetAmount` already takes for worms:
// two doublings on one card are four times, not three. 100 is the identity, and a card carrying
// none of this kind returns it.
func (c Card) riderScale(kind RiderKind) int {
	pct := 100
	for _, r := range c.Riders {
		if r.Kind == kind {
			pct = pct * r.Amount / 100
		}
	}
	return pct
}

// ShieldOnPlay is the shields this card raises as it is played, summed over its riders.
func (c Card) ShieldOnPlay() int { return c.riderTotal(RiderShieldOnPlay) }

// DamageOnPlay is the DMG this card adds to its duelist for the blow it is played into.
func (c Card) DamageOnPlay() int { return c.riderTotal(RiderDamageOnPlay) }

// DamageInHand is the DMG this card adds to its duelist while it sits unplayed in the hand.
func (c Card) DamageInHand() int { return c.riderTotal(RiderDamageInHand) }

// VitaeInHand is the vitae this card pays while it sits unplayed in the hand.
func (c Card) VitaeInHand() int { return c.riderTotal(RiderVitaeInHand) }

// ScaleInHand is the percentage this card scales its duelist's DMG by while it sits unplayed.
func (c Card) ScaleInHand() int { return c.riderScale(RiderScaleInHand) }

// ScaleInCombo is the percentage this card scales its duelist's DMG by when it is one of the
// cards the hand was formed from.
func (c Card) ScaleInCombo() int { return c.riderScale(RiderScaleInCombo) }

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

// blowDMG is the duelist's DMG for one blow, after every rider with something to say about it.
//
// **One figure, applied to the duelist rather than to any card** *(owner's call, 2026-09-02)*.
// `Card.Damage` is linear in DMG, so raising the duelist's figure for the length of one
// calculation raises every term of the hand by the same proportion — which is what makes the
// printed bracket go on summing to the printed total. Adding to a single card's amount would have
// made the bonus a fact about that card, and the owner's version is a fact about the turn.
//
// **Flat first, then the percentages.** A +10 and a doubling on the same turn is (DMG+10)x2, so
// the two kinds of rider compose rather than race; percentages compound with each other for the
// reason `Card.riderScale` gives.
//
// **Three populations, and they are asked different questions.** The turn's cards answer "were you
// played"; the held cards answer "did you stay in hand"; and the blow's own cards answer "did you
// make the hand", which is the narrowest of the three and the only one the player can miss.
func blowDMG(base int, turn []Slot, held []Card, blow Blow) int {
	dmg := base
	for _, slot := range turn {
		dmg += slot.Card.DamageOnPlay()
	}
	for _, c := range held {
		dmg += c.DamageInHand()
	}

	pct := 100
	for _, c := range held {
		pct = pct * c.ScaleInHand() / 100
	}
	for _, i := range blow.Cards {
		pct = pct * turn[i].Card.ScaleInCombo() / 100
	}
	dmg = dmg * pct / 100

	// A duelist cannot be argued below the floor `Card.Damage` already keeps. Nothing today
	// reduces DMG, but the clamp is what makes that still true the first time something does.
	if dmg < 0 {
		dmg = 0
	}
	return dmg
}

// vitaeHeld is the vitae the unplayed hand pays this turn, summed over every card still in it.
func vitaeHeld(held []Card) int {
	total := 0
	for _, c := range held {
		total += c.VitaeInHand()
	}
	return total
}
