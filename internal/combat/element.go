package combat

// Elements are what a card is made of, and they are a rule rather than a color.
//
// **An element is matchable by a hand `Step` and by a relic's predicate**, a creature carries one
// and fizzles a hit of its own, and a shield of a hit's element banks a surge. Everything else
// about an element — its color, its name on a card — is presentation and stays out of this
// package.

// Element is what a card is made of. `Basic` is the absence of an element rather than a fifth
// color, which is why it is the zero value: a card that names no element is a plain card, and
// so is a zero `Card`.
//
// **Append-only, like ConceptID.** Arrays and caches are indexed by this value, so inserting an
// element mid-enum silently re-points everything already stored. Add at the end.
type Element int

const (
	Basic Element = iota
	Fire
	Ice
	Lightning
	Earth

	// Arcane is the fifth color *(owner's call, 2026-08-25)*, appended rather than inserted for the
	// reason above.
	Arcane
)

// ElementCount is how many elements exist, and the width of any array indexed by one. Deriving it from
// the last constant is what stops the two drifting when an element is appended.
const ElementCount = int(Arcane) + 1

// AllElements is every element in declaration order. A slice rather than a range over the
// constants so callers walking it get a fixed order — the determinism rules apply here exactly
// as they do to AllActions.
var AllElements = []Element{Basic, Fire, Ice, Lightning, Earth, Arcane}

var elementNames = [...]string{
	Basic:     "basic",
	Fire:      "fire",
	Ice:       "ice",
	Lightning: "lightning",
	Earth:     "earth",
	Arcane:    "arcane",
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
