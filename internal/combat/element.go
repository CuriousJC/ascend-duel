package combat

// Elements are what a card is made of, and as of 2026-08-12 they are a rule rather than a
// colour. They lived on the combat screen as an unexported `element` until then, painting a
// border and meaning nothing — which is why three separate mechanics were all blocked on the
// same sentence in MECHANICS.md: *element must cross into `internal/combat`*. This file is that
// crossing, and it unblocks the element combos, the ring discount and the flip ring together.
//
// **An element does two things.** It is matchable by a combo `Step`, and a landed attack applies
// its element's status to whoever took the blow. Everything else about an element — its colour,
// its name on a card — stays presentation and stays in `internal/screens`.
//
// **Only attacks apply statuses** *(decided 2026-08-12)*. An ice Guard is an ice card for combo
// and discount purposes and applies nothing. The alternative was every card applying its status,
// which would make the 1-AP Jab and the 1-AP Prepare equally good status delivery and turn the
// prepare phase into the status engine. The cost of the rule chosen: element is mechanically
// inert on the eight concepts that are not attacks, and it buys them nothing until rings land.

// Element is what a card is made of. `Basic` is the absence of an element rather than a fifth
// colour, which is why it is the zero value: a card that names no element is a plain card, and
// so is a zero `Card`.
//
// **Append-only, like ActionKind and GlyphKind.** `Duelist.Statuses` is an array indexed by this
// value, so inserting an element mid-enum silently re-points every status a duelist is carrying.
// Add at the end.
type Element int

const (
	Basic Element = iota
	Fire
	Ice
	Lightning
	Earth
)

// ElementCount is how many elements exist, and the width of the status array. Deriving it from
// the last constant is what stops the two drifting when an element is appended.
const ElementCount = int(Earth) + 1

// AllElements is every element in declaration order. A slice rather than a range over the
// constants so callers walking it get a fixed order — the determinism rules apply here exactly
// as they do to AllActions.
var AllElements = []Element{Basic, Fire, Ice, Lightning, Earth}

var elementNames = [...]string{
	Basic:     "basic",
	Fire:      "fire",
	Ice:       "ice",
	Lightning: "lightning",
	Earth:     "earth",
}

func (e Element) String() string {
	if e < 0 || int(e) >= len(elementNames) {
		return "?"
	}
	return elementNames[e]
}

// ParseElement resolves the element names written in the card JSON. It reports failure rather
// than falling back to Basic, for the same reason ParseAction does: a deck quietly built out of
// the wrong element is a balance change nobody made.
func ParseElement(name string) (Element, bool) {
	for i, n := range elementNames {
		if n == name {
			return Element(i), true
		}
	}
	return Basic, false
}

// Card is one playable card: a concept and the element it is made of. It is the unit that
// crosses into this package now, replacing the bare ActionKind that used to.
//
// **Comparable, and it must stay that way.** The screen caches rendered card faces on a struct
// holding one of these, and `Slot` is compared in tests.
type Card struct {
	Action  ActionKind
	Element Element
}

// Plain is a card with no element, which is what an enemy draws and what most tests want.
func Plain(a ActionKind) Card { return Card{Action: a} }

// Of is a card of a named element.
func Of(a ActionKind, e Element) Card { return Card{Action: a, Element: e} }

// PlainCards lifts a list of concepts into elementless cards. It exists for tools/balance and
// for tests, which reason about concepts and have nothing to say about colour.
func PlainCards(actions ...ActionKind) []Card {
	out := make([]Card, len(actions))
	for i, a := range actions {
		out[i] = Plain(a)
	}
	return out
}

// Cost is what this card takes out of the round's budget.
//
// **It is a method on the card rather than on the action, and that is the seat the ring discount
// sits in.** MECHANICS.md records that a matching ring makes cost a property of the *pairing*
// rather than of the concept; nothing discounts anything yet, so this delegates. Cutting the
// seat now costs nothing and saves rewriting every call site a second time.
func (c Card) Cost() int { return c.Action.Cost() }

// Category is which phase this card resolves in. A fire Guard and a plain Guard are both
// prepares — the element never moves a card between phases.
func (c Card) Category() Category { return c.Action.Category() }

// Damage is what this card deals given the attacker's Strength, before any multiplier, blunting
// or defence.
func (c Card) Damage(str int) int { return c.Action.Damage(str) }

func (c Card) String() string {
	if c.Element == Basic {
		return c.Action.String()
	}
	return c.Element.String() + " " + c.Action.String()
}

// Actions strips the elements off a list of cards. It is for callers that genuinely only care
// about concepts — a hand tally, a label — and never for anything that resolves a round.
func Actions(cards []Card) []ActionKind {
	out := make([]ActionKind, len(cards))
	for i, c := range cards {
		out[i] = c.Action
	}
	return out
}
