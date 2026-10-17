package combat

// Traits: **a property a thing carries beside what it is, rather than instead of any of it.**
//
// A card is a form, an element and an action, and then one upgrade — and a second upgrade replaces
// the first. A trait is not an upgrade: it composes with all four, so a golden fire Bash can also be
// mirrored, and a rune writing a new upgrade leaves the trait alone. A worn relic is the same shape
// one axis over: its rules are the relic, and whether this wearing takes a slot, lasts past the
// fight or raises a shield every turn is a fact about the wearing.
//
// **The traits are data, written in a grammar**, as everything else that grows by authoring is.
// `data/card_traits.json` is read here, as `duelist_cards.json` is: a card trait is closed properties
// the rules read — `Counted`, `Landings`. `data/relic_traits.json` is parsed in `internal/session`,
// as `relics.json` is, because a relic trait carries rules in the relic grammar and that parsing
// lives there; it arrives through RegisterRelicTrait.
//
// **Both sets are bitmasks**, because a card and a worn relic must stay comparable — the screen's
// face cache, `TestRoundIsDeterministic` and `Duelist` being a value all depend on it. A bit is a
// registration position, so **never serialized as a number**: a snapshot writes the names.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// maxTraits is the width of a trait set, and so how many traits of one kind may be registered.
const maxTraits = 16

// CardTrait is a set of card traits. One trait is a set with one bit.
type CardTrait uint16

// cardTraitSpec is one card trait's record, resolved.
type cardTraitSpec struct {
	Key, Name, Text string

	// Counted is whether the card counts against MaxActions. False is the effortless card: not one
	// of the five, still paying its AP and still joining the hand. See Duelist.RoomFor.
	Counted bool

	// Landings is how many times the card lands before any relic is read: an attack throws that
	// many full-strength hits, ahead of a relic's repeats, which add to it; a defense raises its
	// shields that many times. One is a plain card.
	Landings int
}

var cardTraitRegistry []cardTraitSpec

// The card traits are read at package init, like the deck: a bad record fails the launch.
var _ = registerCardTraits()

func registerCardTraits() bool {
	for _, r := range data.LoadCardTraits() {
		counted := r.Counted == nil || *r.Counted
		if _, err := RegisterCardTrait(r.CardTrait, r.Name, r.Text, counted, r.Landings); err != nil {
			panic("card_traits.json: " + err.Error())
		}
	}
	return true
}

// RegisterCardTrait adds one card trait. **A trait that changes nothing is refused** — counted and
// landing once is a plain card wearing a word. Landings of zero is read as one.
func RegisterCardTrait(key, name, text string, counted bool, landings int) (CardTrait, error) {
	if key == "" || name == "" || text == "" {
		return 0, fmt.Errorf("a card trait wants a key, a name and a text (%q, %q, %q)", key, name, text)
	}
	if _, taken := ParseCardTrait(key); taken {
		return 0, fmt.Errorf("card trait %q is registered twice", key)
	}
	if len(cardTraitRegistry) == maxTraits {
		return 0, fmt.Errorf("card trait %q is past the %d a card can carry", key, maxTraits)
	}
	if landings == 0 {
		landings = 1
	}
	if landings < 1 || landings > MaxEchoLandings {
		return 0, fmt.Errorf("card trait %q lands %d times, and a card lands 1 to %d", key, landings, MaxEchoLandings)
	}
	if counted && landings == 1 {
		return 0, fmt.Errorf("card trait %q changes nothing about a card", key)
	}
	cardTraitRegistry = append(cardTraitRegistry, cardTraitSpec{
		Key: key, Name: name, Text: text, Counted: counted, Landings: landings,
	})
	return CardTrait(1) << (len(cardTraitRegistry) - 1), nil
}

// CardTraits is every card trait, in registration order — the file's.
func CardTraits() []CardTrait {
	out := make([]CardTrait, len(cardTraitRegistry))
	for i := range cardTraitRegistry {
		out[i] = CardTrait(1) << i
	}
	return out
}

// ParseCardTrait is the trait a key names.
func ParseCardTrait(key string) (CardTrait, bool) {
	for i, s := range cardTraitRegistry {
		if s.Key == key {
			return CardTrait(1) << i, true
		}
	}
	return 0, false
}

func (t CardTrait) spec() (cardTraitSpec, bool) {
	for i, s := range cardTraitRegistry {
		if t == CardTrait(1)<<i {
			return s, true
		}
	}
	return cardTraitSpec{}, false
}

// String is the trait's key, which is what a snapshot and a scenario write.
func (t CardTrait) String() string { s, _ := t.spec(); return s.Key }

// Name is what the trait is called, and Text what it does, for a tooltip.
func (t CardTrait) Name() string { s, _ := t.spec(); return s.Name }
func (t CardTrait) Text() string { s, _ := t.spec(); return s.Text }

// Has reports whether every trait in `t` is in the set.
func (s CardTrait) Has(t CardTrait) bool { return t != 0 && s&t == t }

// Names is the set as its traits' keys, in registration order. Nil for the empty set.
func (s CardTrait) Names() []string {
	var out []string
	for _, t := range CardTraits() {
		if s.Has(t) {
			out = append(out, t.String())
		}
	}
	return out
}

// HasTrait reports whether the card carries a trait.
func (c Card) HasTrait(t CardTrait) bool { return c.Traits.Has(t) }

// WithTrait is this card carrying `t` as well as whatever it already carries — or without it, when
// `on` is false.
func (c Card) WithTrait(t CardTrait, on bool) Card {
	if on {
		c.Traits |= t
	} else {
		c.Traits &^= t
	}
	return c
}

// Counted reports whether the card counts against MaxActions: every card but one carrying a trait
// that says it does not.
func (c Card) Counted() bool {
	for _, t := range CardTraits() {
		if s, _ := t.spec(); c.HasTrait(t) && !s.Counted {
			return false
		}
	}
	return true
}

// Landings is how many times the card lands before any relic is read: the most any of its traits
// says, and once with none. Two traits saying twice are twice, not four times.
func (c Card) Landings() int {
	n := 1
	for _, t := range CardTraits() {
		if s, _ := t.spec(); c.HasTrait(t) && s.Landings > n {
			n = s.Landings
		}
	}
	return n
}

// RelicTrait is a set of relic traits. One trait is a set with one bit.
type RelicTrait uint16

// relicTraitSpec is one relic trait's record, resolved.
type relicTraitSpec struct {
	Key, Name string

	// Slot is whether the wearing takes a relic slot. False is weightless: worn on top of a full
	// row. See relicSlots.
	Slot bool

	// LastsFight is whether the wearing lasts only the fight it was put on in — ephemeral. The
	// rules hold such a wearing only on the fight's own duelist, so this is what the drawing and
	// the shop read rather than what ends it.
	LastsFight bool

	// Rules are worn with the relic, after its own, in the relic grammar — armored's turn-start
	// shield. **They never grow**: a growing relic's accumulator feeds its own effects, never a
	// trait's, and a trait carrying a growth verb is refused.
	Rules []RelicRule
}

var relicTraitRegistry []relicTraitSpec

// RegisterRelicTrait adds one relic trait and returns its bit, or reports why it could not. The
// rules are held to the relic grammar — CheckRelic, under the trait's key — and may be empty, since
// weightless and ephemeral do nothing a rule says.
func RegisterRelicTrait(key, name string, slot, lastsFight bool, rules []RelicRule) (RelicTrait, error) {
	if key == "" || name == "" {
		return 0, fmt.Errorf("a relic trait wants a key and a name (%q, %q)", key, name)
	}
	if _, taken := ParseRelicTrait(key); taken {
		return 0, fmt.Errorf("relic trait %q is registered twice", key)
	}
	if len(relicTraitRegistry) == maxTraits {
		return 0, fmt.Errorf("relic trait %q is past the %d a wearing can carry", key, maxTraits)
	}
	if len(rules) > 0 {
		if err := CheckRelic(key, rules); err != nil {
			return 0, err
		}
	}
	if slot && !lastsFight && len(rules) == 0 {
		return 0, fmt.Errorf("relic trait %q changes nothing about a wearing", key)
	}
	own := make([]RelicRule, len(rules))
	for i, r := range rules {
		for _, e := range r.Then {
			if e.Do.grows() {
				return 0, fmt.Errorf("relic trait %q does %s, and a trait's rules never grow", key, e.Do)
			}
		}
		r.ungrown = true
		own[i] = r
	}
	relicTraitRegistry = append(relicTraitRegistry, relicTraitSpec{
		Key: key, Name: name, Slot: slot, LastsFight: lastsFight, Rules: own,
	})
	return RelicTrait(1) << (len(relicTraitRegistry) - 1), nil
}

// RelicTraits is every relic trait, in registration order — the file's.
func RelicTraits() []RelicTrait {
	out := make([]RelicTrait, len(relicTraitRegistry))
	for i := range relicTraitRegistry {
		out[i] = RelicTrait(1) << i
	}
	return out
}

// ParseRelicTrait is the trait a key names.
func ParseRelicTrait(key string) (RelicTrait, bool) {
	for i, s := range relicTraitRegistry {
		if s.Key == key {
			return RelicTrait(1) << i, true
		}
	}
	return 0, false
}

func (t RelicTrait) spec() (relicTraitSpec, bool) {
	for i, s := range relicTraitRegistry {
		if t == RelicTrait(1)<<i {
			return s, true
		}
	}
	return relicTraitSpec{}, false
}

// String is the trait's key, which is what a snapshot and a scenario write.
func (t RelicTrait) String() string { s, _ := t.spec(); return s.Key }

// Name is what the trait is called — the line a worn relic's tooltip ends on.
func (t RelicTrait) Name() string { s, _ := t.spec(); return s.Name }

// Has reports whether every trait in `t` is in the set.
func (s RelicTrait) Has(t RelicTrait) bool { return t != 0 && s&t == t }

// Names is the set as its traits' keys, in registration order. Nil for the empty set.
func (s RelicTrait) Names() []string {
	var out []string
	for _, t := range RelicTraits() {
		if s.Has(t) {
			out = append(out, t.String())
		}
	}
	return out
}

// any reports whether some trait in the set answers `p`.
func (s RelicTrait) any(p func(relicTraitSpec) bool) bool {
	for _, t := range RelicTraits() {
		if sp, _ := t.spec(); s.Has(t) && p(sp) {
			return true
		}
	}
	return false
}

// TakesNoSlot reports whether a wearing with these traits is weightless.
func (s RelicTrait) TakesNoSlot() bool {
	return s.any(func(sp relicTraitSpec) bool { return !sp.Slot })
}

// LastsOneFight reports whether a wearing with these traits is ephemeral.
func (s RelicTrait) LastsOneFight() bool {
	return s.any(func(sp relicTraitSpec) bool { return sp.LastsFight })
}

// Weightless reports whether the wearing takes no slot, and Ephemeral whether it lasts the fight it
// was put on in — each true when any of its traits says so.
func (w WornRelic) Weightless() bool { return w.Traits.TakesNoSlot() }
func (w WornRelic) Ephemeral() bool  { return w.Traits.LastsOneFight() }

// Rules is everything this wearing fires: the relic's own rules, then each trait's in registration
// order. **Every walk over a worn relic's rules goes through here**, so a trait's rules wake at the
// same moments, in the same worn order, as the relic it is on.
func (w WornRelic) Rules() []RelicRule {
	own := RelicOf(w.Relic).Rules
	if w.Traits == 0 {
		return own
	}
	out := append([]RelicRule(nil), own...)
	for _, t := range RelicTraits() {
		if sp, _ := t.spec(); w.Traits.Has(t) {
			out = append(out, sp.Rules...)
		}
	}
	return out
}

// grown is what the wearing's accumulator adds to one of its rules' figures: the accumulator for the
// relic's own rules, nothing for a trait's.
func (w WornRelic) grown(r RelicRule) int {
	if r.ungrown {
		return 0
	}
	return w.Grown
}
