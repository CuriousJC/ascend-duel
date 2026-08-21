package combat

// Elements are what a card is made of, and as of 2026-08-12 they are a rule rather than a
// colour. They lived on the combat screen as an unexported `element` until then, painting a
// border and meaning nothing — which is why three separate mechanics were all blocked on the
// same sentence in MECHANICS.md: *element must cross into `internal/combat`*. This file is that
// crossing, and it unblocks the element hands, the ring discount and the flip ring together.
//
// **An element does two things.** It is matchable by a hand `Step`, and a landed attack applies
// its element's status to whoever took the blow. Everything else about an element — its colour,
// its name on a card — stays presentation and stays in `internal/screens`.
//
// **Only attacks apply statuses** *(decided 2026-08-12)*. An ice Guard is an ice card for hand
// and discount purposes and applies nothing. The alternative was every card applying its status,
// which would make the 1-AP Jab and the 1-AP Prepare equally good status delivery and turn the
// prepare phase into the status engine. The cost of the rule chosen: element is mechanically
// inert on the eight concepts that are not attacks, and it buys them nothing until rings land.

// Element is what a card is made of. `Basic` is the absence of an element rather than a fifth
// colour, which is why it is the zero value: a card that names no element is a plain card, and
// so is a zero `Card`.
//
// **Append-only, like GlyphKind.** `Duelist.Statuses` and `Duelist.Rings` are arrays indexed by
// this value, so inserting an element mid-enum silently re-points every status a duelist is
// carrying. Add at the end.
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
