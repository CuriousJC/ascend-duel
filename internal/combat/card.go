package combat

// **A card is a concept plus an element, and it is the unit this whole package deals in.**
// []Card goes through ResolveRound, ResolutionOrder, Slot, PlanFor and CostOf — which is what
// let the combat screen delete its own card struct rather than map one across the boundary.
//
// It is a small comparable value on purpose: two ints. Nothing here holds a name, a cost or a
// picture — those are the concept's, looked up through Spec(), so a card renamed in
// data/duelist_cards.json is renamed once and a saved deck of cards is a list of pairs.
//
// **What a card is told two ways** also lives here: a Category, which says when it resolves, and
// a Form, which says what kind of thing it is. They answer different questions and are read by
// different code — the resolver reads Category and never Form; the hand matcher reads Form and
// never Category — which is exactly why they are two fields rather than one enum.
//
// **FormNone is a real answer**, and it belongs to every enemy card. A form is the player's deck
// axis, so an enemy card claiming to be a crush would be saying something untrue about a deck
// the player cannot build hands against.
//
// Split out of element.go and combat.go on 2026-08-21, which held the element enum, the card and
// the card's taxonomy in two files that were mostly about something else.

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

	// FormOverride is what this one card counts as on the form axis, when a parasite has told it
	// to be something else. **FormNone means unmodified**, so a plain card is still the zero value
	// and every `Card{Concept: x}` literal keeps working — the same trade AmountPct made.
	//
	// **It is a per-card modifier and not a concept swap**, because that is the difference the
	// owner asked for: a Ward turned into a crush is still a Ward, still defends, and now counts
	// as a crush when the hand is matched. That produces a card that shields and matches on an
	// attack axis, which is a legal weird thing rather than a bug *(owner's call, 2026-09-02)*.
	//
	// **Only `Card.Form` reads it**, which is the one chokepoint the whole game already asks —
	// the matcher, the form rings, the sort and the card face all go through it.
	FormOverride Form

	// ID is which card in the run this is, and it is the card's identity rather than its
	// description.
	//
	// **Two cards that look identical are still two cards** *(owner's call, 2026-08-24)*. Every
	// other field says what a card *is*; this says *which one*. A run assigns one to every card it
	// owns and the number survives the shuffle, the hand, the discard and the reshuffle — so
	// something holding a card in a pile can always ask the run what that same card looked like
	// before a ring got to it.
	//
	// **That is what the element flip needs.** A flip fires as a card is drawn and the drawn card
	// carries only the colour it became — it does not remember what it was, because a rule reading
	// what a card *used* to be is exactly the ordering the owner ruled out. The original is not
	// gone, it simply lives where it always did: on the card the run owns, reachable by this
	// number. The deck panel draws either face from it; see screens/deckpanel.go.
	//
	// **Zero means no identity, and that is the common case in this package.** Every enemy card,
	// every card a test writes as a literal and every `Plain`/`Of` has ID 0 — an enemy wears no
	// rings and a test deck has no run behind it. Nothing here may *require* an ID; it is a handle
	// the layers above use, and the rules go on resolving a card that has none.
	//
	// **It does not make a card less comparable.** The screen's face cache keys on `cards.Spec`,
	// which is built from what a card looks like and never from this, so two cards that differ
	// only by ID still share one rendered face.
	ID int

	// Riders are the rules this one card carries, and they are what a parasite leaves behind.
	//
	// **A fixed array, because a card must stay comparable** — the screen's face cache and
	// TestRoundIsDeterministic both depend on it, and a slice here would end both. Empty seats are
	// RiderNone, so an unridden card is still the zero value and every `Card{Concept: x}` literal
	// in this package keeps working. See rider.go for what a rider is and why its vocabulary is a
	// Go enum rather than a data record.
	Riders [MaxCardRiders]Rider
}

// Plain is a card with no element, which is what an enemy draws and what most tests want.
func Plain(id ConceptID) Card { return Card{Concept: id} }

// Of is a card of a named element.
func Of(id ConceptID, e Element) Card { return Card{Concept: id, Element: e} }

// PlainCards lifts a list of concepts into elementless cards. It exists for tests, which reason
// about concepts and have nothing to say about colour.
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

// Amount is the card's figure, read against its verb: a defence percentage, shields raised, or the
// damage multiplier.
//
// **It is the seat a worm's scaling sits in**, the same shape Cost is, and it is why the three
// places that used to read `Spec().Amount` directly now go through the card. A modified card that
// still reported its concept's figure would behave differently from what its own face says.
//
// **A defence is clamped below 100, a shield count at maxShields, and everything is floored at 1.**
// `RegisterConcept` refuses a concept declaring either out of range; a worm stacking onto one has to
// obey the same rule, and it clamps rather than being refused — a reward that silently did nothing
// would be worse than one that hits its ceiling.
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
	if s.Verb == VerbShield && amount > maxShields {
		amount = maxShields
	}
	return amount
}

// maxDefendPct is the most a single card may take off a blow. **Nothing stops a blow outright** —
// see reductionFor, which holds the same rule from the other side.
const maxDefendPct = 99

// maxShields is the most one card may raise, and it is baseMaxActions because that is the most
// attacks an opposing turn can contain. A sixth shield could never be spent — it is not a bigger
// reward, it is a number the readout has to draw and nothing can ever take away.
//
// **It bounds one card, not a duelist.** Three Guards in a turn is nine shields and is meant to be:
// what stops that is the action budget, exactly as it stops nine attacks.
const maxShields = baseMaxActions

// MaxShields is the same number for a caller outside this package.
//
// **The screen needs it because it draws a shield before the engine raises one** *(2026-09-02)*: a
// defend card's pips fly to the duelist card on the beat that card is scored into the hand, which
// is the attack phase — several beats before the defend phase raises them. The flight predicts,
// and a prediction has to know the ceiling the raise will be held to or it can draw a sixth pip on
// a row that holds five. See screens.shieldFlight, and Duelist.raiseShields, which is the cap
// itself.
const MaxShields = maxShields

// Category is which phase this card resolves in, and it falls out of the verb: an attack resolves
// in the attack phase and everything else in the defend phase. A fire Guard and a plain Guard are
// both defences — the element never moves a card between phases.
func (c Card) Category() Category {
	if c.Spec().Verb == VerbAttack {
		return CategoryAttack
	}
	return CategoryDefend
}

// Form is which group of cards this one belongs to. Enemy cards belong to none.
func (c Card) Form() Form {
	if c.FormOverride != FormNone {
		return c.FormOverride
	}
	return c.Spec().Form
}

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

// Category is which phase of a turn an action resolves in, and the axis the whole round
// is built on. It is a property of the action, not an independent choice: a fire Lunge
// and a plain Lunge are both attacks.
//
// **There are two of them as of 2026-08-15**, down from prepare/attack/defend. The deck is
// three attack forms and one defend form, and the three-way split was describing a deck that
// no longer exists: everything that is not an attack sits in the same phase, and what separates
// those cards is what they do in it rather than when they happen.
//
// **The second phase is named for everything in it** *(2026-08-31)*. The player's three cards raise
// shields and ninety creature cards raise a percentage guard, and there is no longer a third thing
// in there for the name to be a stretch over.
type Category int

const (
	CategoryAttack Category = iota
	CategoryDefend
)

// Categories is every phase in resolution order, and the order a turn is played in.
//
// **Attacks first, defences second.** A defence has to go up at the *end* of your turn, because
// the opponent acts afterwards and that is the blow it answers. Resolving them first would mean
// every guard and every shield expired before anything could be aimed at it.
//
// It is also the order the combat screen lays a turn out in, which is not a coincidence: the row
// on the table reads left to right in exactly this sequence.
func Categories() []Category {
	return []Category{CategoryAttack, CategoryDefend}
}

func (c Category) String() string {
	switch c {
	case CategoryAttack:
		return "attack"
	case CategoryDefend:
		return "defend"
	default:
		return "?"
	}
}

// ParseCategory resolves a category from its name, which is how hands.json writes a hand's
// scope. It reports failure rather than falling back, for the same reason ParseAction does: a
// hand quietly counting the wrong phase is a balance change nobody made.
func ParseCategory(name string) (Category, bool) {
	for _, c := range Categories() {
		if c.String() == name {
			return c, true
		}
	}
	return CategoryAttack, false
}

// Form is which group of cards an action belongs to: three ways of hitting, plus the defences.
//
// **It is what the card's corner says, and it is not the same axis as Category** *(2026-08-15)*.
// Category is when a card resolves and there are two of those; a form is what kind of card it
// is and there are four. Every form but Defend resolves in the attack phase, so the form is the
// finer distinction and the one worth putting on a card face.
//
// **FormNone is the zero value and is a real answer**, not a failure: the opponent's cards
// belong to no form. Forms are the player's deck axis — three ways of building a pair — and
// an enemy Attack claiming to be a crush would be saying something untrue about a deck the player
// cannot build hands against.
type Form int

const (
	FormNone Form = iota
	FormStab
	FormSlash
	FormCrush
	FormDefend
)

// Forms is every real form, in a fixed order, for anything that walks them.
func Forms() []Form {
	return []Form{FormStab, FormSlash, FormCrush, FormDefend}
}

func (f Form) String() string {
	switch f {
	case FormStab:
		return "stab"
	case FormSlash:
		return "slash"
	case FormCrush:
		return "crush"
	case FormDefend:
		return "defend"
	default:
		return "none"
	}
}

// ParseForm resolves a form from its name, which is how a deck list writes one. It reports
// failure rather than falling back, for the same reason ParseAction does.
func ParseForm(name string) (Form, bool) {
	for _, f := range Forms() {
		if f.String() == name {
			return f, true
		}
	}
	return FormNone, false
}
