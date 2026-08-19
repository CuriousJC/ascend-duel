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

// Card is one playable card: a registered concept and the element it is made of.
//
// **Comparable, and it must stay that way.** The screen caches rendered card faces on a struct
// holding one of these, and `Slot` is compared in tests. Holding a `ConceptID` rather than the
// concept itself is what keeps that true now that a concept is a record with a string in it.
type Card struct {
	Concept ConceptID
	Element Element

	// CostDelta and AmountPct are **per-card modifiers, and they are what a worm writes**
	// *(2026-08-17)*. Everything else about a card lives on the shared concept, so altering one
	// copy of a Strike would otherwise alter every Strike in the deck.
	//
	// CostDelta is added to the concept's cost; AmountPct scales its amount, as a percentage, with
	// **zero meaning unmodified** so a plain card is still the zero value and every existing
	// `Card{Concept: x}` literal keeps working.
	//
	// **The bounds live in the methods, not here**, because a stack of worms has to clamp rather
	// than be refused — see Cost and Amount. A card is still comparable, which is what
	// TestRoundIsDeterministic and the screen's render cache both need.
	CostDelta int
	AmountPct int
}

// Plain is a card with no element, which is what an enemy draws and what most tests want.
func Plain(id ConceptID) Card { return Card{Concept: id} }

// Of is a card of a named element.
func Of(id ConceptID, e Element) Card { return Card{Concept: id, Element: e} }

// PlainCards lifts a list of concepts into elementless cards. It exists for tools/balance and
// for tests, which reason about concepts and have nothing to say about colour.
func PlainCards(ids ...ConceptID) []Card {
	out := make([]Card, len(ids))
	for i, id := range ids {
		out[i] = Plain(id)
	}
	return out
}

// Spec is this card's rules, looked up once.
func (c Card) Spec() Concept { return ConceptOf(c.Concept) }

// Cost is what this card takes out of the round's budget.
//
// **It is a method on the card rather than on the concept, and that is the seat the ring discount
// sits in.** MECHANICS.md records that a matching ring makes cost a property of the *pairing*
// rather than of the concept; nothing discounts anything yet, so this delegates. Cutting the
// seat now costs nothing and saves rewriting every call site a second time.
func (c Card) Cost() int {
	cost := c.Spec().Cost + c.CostDelta
	if cost < minCardCost {
		cost = minCardCost
	}
	return cost
}

// minCardCost is the floor a cheapening worm can drive a card to.
//
// **Zero, deliberately, and it moves the game onto its other bound.** A round is capped by cost
// *and* by count independently — `MaxActions` cards however cheap they are — so a free card is not
// unbounded, it is bounded by the count instead. That is a real shift in what limits a turn and it
// was taken with it in view (owner's call, 2026-08-17), not as an oversight about a number that
// happened not to have a floor.
const minCardCost = 0

// Amount is the card's figure, read against its verb: a defence percentage, action points banked,
// cards drawn, or the damage multiplier.
//
// **It is the seat a worm's scaling sits in**, the same shape Cost is, and it is why the three
// places that used to read `Spec().Amount` directly now go through the card. A modified card that
// still reported its concept's figure would behave differently from what its own face says.
//
// **A defence is clamped below 100 and everything is floored at 1.** `RegisterConcept` refuses a
// concept declaring a defence of 100 or more, because nothing may reduce a blow to zero; a worm
// stacking onto a Defend has to obey the same rule, and it clamps rather than being refused —
// a reward that silently did nothing would be worse than one that hits its ceiling.
func (c Card) Amount() int {
	s := c.Spec()

	amount := s.Amount
	if c.AmountPct != 0 {
		amount = amount * c.AmountPct / 100
	}
	if amount < 1 {
		amount = 1
	}
	if s.Verb == VerbDefend && amount > maxDefendPct {
		amount = maxDefendPct
	}
	return amount
}

// maxDefendPct is the most a single card may take off a blow. **Nothing stops a blow outright** —
// see reductionFor, which holds the same rule from the other side.
const maxDefendPct = 99

// Category is which phase this card resolves in, and it falls out of the verb: an attack resolves
// in the attack phase and everything else in the plan phase. A fire Defend and a plain Defend are
// both plans — the element never moves a card between phases.
func (c Card) Category() Category {
	if c.Spec().Verb == VerbAttack {
		return CategoryAttack
	}
	return CategoryPlan
}

// Form is which group of cards this one belongs to. Enemy cards belong to none.
func (c Card) Form() Form { return c.Spec().Form }

// Label is what this card is called on screen and in the Resolution feed.
func (c Card) Label() string { return c.Spec().Label }

// Damage is what this card deals in the hands of a duelist with this DMG, before any multiplier,
// blunting or defence.
//
// **The ladder is a number on the card now** *(2026-08-16)*. It was three switch cases — half DMG,
// DMG, double — which is exactly the 50/100/200 the player's nine attacks still declare. What
// changed is that an enemy may sit anywhere on it, and that a card at 150 needs no new rung.
//
// The floor keeps the cheapest cards from rounding away to nothing at a low DMG, which is where a
// duel starts. A card that is meant to deal nothing is not an attack.
func (c Card) Damage(dmg int) int {
	s := c.Spec()
	if s.Verb != VerbAttack {
		return 0
	}
	d := dmg * c.Amount() / 100
	if d < 1 {
		d = 1
	}
	return d
}

func (c Card) String() string {
	if c.Element == Basic {
		return c.Label()
	}
	return c.Element.String() + " " + c.Label()
}

// Concepts strips the elements off a list of cards. It is for callers that genuinely only care
// about concepts — a hand tally, a label — and never for anything that resolves a round.
func Concepts(cards []Card) []ConceptID {
	out := make([]ConceptID, len(cards))
	for i, c := range cards {
		out[i] = c.Concept
	}
	return out
}
