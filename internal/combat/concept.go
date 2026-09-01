package combat

// The concept registry: what a card *is*, as data rather than as a switch statement.
//
// **This replaced `ActionKind` on 2026-08-16.** A card's cost, damage, category and form were
// four switch statements over a closed enum of fourteen constants, and a fifth turned it into a
// name. That held twelve player cards. It could not hold three or four bespoke cards for each of
// ninety-six enemies, so the card stopped being an enum value and became a record — see
// `data.CardData`, which is the file format, and MECHANICS.md, which is the argument.
//
// **A Card holds a ConceptID and an Element, and stays comparable.** That is what keeps
// `TestRoundIsDeterministic` comparing resolved duelists with `==`, and what lets the screen cache
// a rendered card face on a struct holding one.
//
// **IDs are assigned in registration order and must never be serialized.** The player's twelve
// register first, at package init, so they hold the low IDs; enemy concepts register after, as
// their decks are built. That is deterministic — the walk is file order inside a sorted roster —
// but it is deterministic *for one build of one data set*, which is a different promise from
// stability. CLAUDE.md's save format is a seed plus a choice log and already says to serialize
// names rather than iota ordinals; this is the same hazard with a wider blast radius.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// Verb is what a card does. **A closed vocabulary, and closing it is the point** — the same
// posture `hands.json` takes with its reward kinds. Adding a fifth verb is a Go change plus one
// place applying it, and that cost is charged deliberately: a card language that could express
// anything would be a scripting language, and the rules would stop being readable in one file.
type Verb int

const (
	// VerbAttack deals damage, and it is every attack card in the game.
	VerbAttack Verb = iota

	// VerbDefend reduces the one blow it answers, by Amount percent.
	VerbDefend

	// VerbShield hands its duelist Amount shields, and one shield eats one incoming attack
	// outright — see Duelist.Shields and blockedByShield.
	//
	// **It is a count, not a percentage, and that is what separates it from VerbDefend.** The two
	// answer different offences: an enemy is a solo attacker, so its turn is several discrete
	// blows a player can decide how many of to take, while the player forms hands and lands one
	// figure, which a count could only ever delete whole. See MECHANICS.md §Shields.
	VerbShield
)

// Verbs is every verb in a fixed order, for anything that walks them.
func Verbs() []Verb { return []Verb{VerbAttack, VerbDefend, VerbShield} }

func (v Verb) String() string {
	switch v {
	case VerbDefend:
		return "defend"
	case VerbShield:
		return "shield"
	default:
		return "attack"
	}
}

// ParseVerb resolves a verb from its name. It reports failure rather than falling back: a card
// quietly registered as an attack because its verb was misspelled is a balance change nobody made.
func ParseVerb(name string) (Verb, bool) {
	for _, v := range Verbs() {
		if v.String() == name {
			return v, true
		}
	}
	return VerbAttack, false
}

// ConceptID identifies a registered concept. It is an index into the registry, so it is cheap to
// compare, cheap to count with, and meaningless outside the process that assigned it.
type ConceptID int

// NoConcept is the absence of one. A zero ConceptID is a real concept — the first registered — so
// this is deliberately negative rather than zero, and a zero `Card` therefore names the player's
// first card rather than nothing. Nothing relies on that; the engine never invents a card.
const NoConcept ConceptID = -1

// Concept is one card's rules, whole.
type Concept struct {
	// Key is the rules identity and is unique across the registry. The player's cards use their
	// bare label; an enemy's are scoped to that enemy — see ScopedKey.
	Key string

	// Label is what the card face says, and what the Resolution feed calls it. Two enemies may
	// share one.
	Label string

	Verb   Verb
	Amount int
	Cost   int
	Form   Form
}

// registry is every concept the process knows, in registration order, plus the key index.
//
// **Package state, and mutable, which is unusual for this package and worth justifying.** The
// alternative was threading a catalogue through `ResolveRound`, `ResolutionOrder`, `PlanFor`,
// every `Card` method and every test, to describe something that is loaded once from embedded
// data and never changes afterwards. What the rules must not carry is a *clock* or a *global
// random source*; a lookup table read from an embedded file is neither. Registration is
// deterministic and additive, and nothing removes or rewrites an entry.
var (
	registry   []Concept
	registryBy = map[string]ConceptID{}
)

// ScopedKey is how an enemy's card is named in the registry: the record it belongs to, then its
// label. Forty creatures may each have a `Bite` at a different multiplier, and none of them may
// collide with another's or with the player's.
func ScopedKey(scope, label string) string {
	if scope == "" {
		return label
	}
	return scope + "." + label
}

// RegisterConcept adds one concept and returns its ID, or reports why it could not.
//
// **It is where the file format is validated**, which is the job `data.CheckCostTiers` used to do
// from the other side of the dependency edge. That check existed because the JSON declared a cost
// the rules also knew; the JSON *is* the rules now, so what is left to check is that a record
// makes sense at all — a verb the vocabulary has, a cost that can be paid, an amount that does
// something.
func RegisterConcept(scope string, c data.CardData) (ConceptID, error) {
	key := ScopedKey(scope, c.Label)

	if c.Label == "" {
		return NoConcept, fmt.Errorf("a card in %q has no label", scope)
	}
	if id, taken := registryBy[key]; taken {
		return id, fmt.Errorf("%s is registered twice", key)
	}

	verb, ok := ParseVerb(c.Verb)
	if !ok {
		return NoConcept, fmt.Errorf("%s names verb %q, which is not one of %s", key, c.Verb, verbList())
	}

	form := FormNone
	if c.Form != "" {
		f, ok := ParseForm(c.Form)
		if !ok {
			return NoConcept, fmt.Errorf("%s names form %q, which the rules do not have", key, c.Form)
		}
		form = f
	}

	if c.Cost < 0 {
		return NoConcept, fmt.Errorf("%s costs %d", key, c.Cost)
	}
	if c.Amount <= 0 {
		return NoConcept, fmt.Errorf("%s has Amount %d, so it does nothing", key, c.Amount)
	}
	if verb == VerbShield && c.Amount > maxShields {
		// More shields than an opponent can throw attacks is a figure that can never be spent, and
		// a readout counting past the row it is drawn in. See maxShields.
		return NoConcept, fmt.Errorf("%s raises %d shields, and nothing may raise more than %d", key, c.Amount, maxShields)
	}
	if verb == VerbDefend && c.Amount >= 100 {
		// Nothing reduces a blow to zero — see defendReductionPct's successor in combat.go. A
		// card that did would delete a whole opposing turn, which is a dominant strategy rather
		// than a decision.
		return NoConcept, fmt.Errorf("%s defends for %d%%, and nothing may stop a blow outright", key, c.Amount)
	}

	id := ConceptID(len(registry))
	registry = append(registry, Concept{
		Key:    key,
		Label:  c.Label,
		Verb:   verb,
		Amount: c.Amount,
		Cost:   c.Cost,
		Form:   form,
	})
	registryBy[key] = id
	return id, nil
}

func verbList() string {
	names := make([]string, 0, len(Verbs()))
	for _, v := range Verbs() {
		names = append(names, v.String())
	}
	return strings.Join(names, ", ")
}

// ConceptOf is the concept behind an ID. An ID the registry does not hold returns a zero Concept,
// which costs nothing, does nothing and is named "?" — the engine never invents an ID, so this is
// a guard rather than a path.
func ConceptOf(id ConceptID) Concept {
	if id < 0 || int(id) >= len(registry) {
		return Concept{Label: "?"}
	}
	return registry[id]
}

// ConceptByKey finds a registered concept by its key.
func ConceptByKey(key string) (ConceptID, bool) {
	id, ok := registryBy[key]
	return id, ok
}

// MustConcept is ConceptByKey for callers that would rather fail at startup than carry a card that
// does nothing — the named player concepts below, and the tools.
func MustConcept(key string) ConceptID {
	id, ok := registryBy[key]
	if !ok {
		panic("combat: no concept named " + key)
	}
	return id
}

// ConceptCount is how many concepts are registered. It is the width anything indexing by ID has
// to allocate, and it grows as enemy decks are built.
func ConceptCount() int { return len(registry) }

// AllConcepts is every registered concept, in registration order — the player's deck first, then
// every enemy's, as their decks were built.
//
// **What it holds depends on what has been loaded**, which is unusual for this package and is the
// price of concepts being data. `internal/decks` registers the roster at its own package init, so
// anything importing that package sees all of them and anything importing only the rules sees the
// player's twelve. Callers that mean "the player's deck" should say PlayerConcepts.
func AllConcepts() []ConceptID {
	out := make([]ConceptID, len(registry))
	for i := range registry {
		out[i] = ConceptID(i)
	}
	return out
}

// RegisteredKeys is every key, sorted. For a tool or a test that wants to walk the registry
// without depending on registration order.
func RegisteredKeys() []string {
	out := make([]string, 0, len(registry))
	for _, c := range registry {
		out = append(out, c.Key)
	}
	sort.Strings(out)
	return out
}

// playerConcepts registers the player's deck list at package init and hands back the key index for
// the named constants below.
//
// **`internal/combat` reads `duelist_cards.json` directly, and that edge is deliberate.** CLAUDE.md
// decides whether a file in `data/` may be read by the rules on *who consumes it* rather than on
// whether it is data — enemy rosters and art keys are a screen's business, and a rules package
// reaching for one would mean the rules had grown an opinion about portraits. A card's cost and
// damage are rules by definition. This is the second such file after `hands.json`, and for the
// same reason.
var playerConcepts = registerPlayerConcepts()

func registerPlayerConcepts() map[string]ConceptID {
	out := map[string]ConceptID{}
	for _, c := range data.LoadDuelistCards() {
		id, err := RegisterConcept("", c)
		if err != nil {
			panic("duelist_cards.json: " + err.Error())
		}
		out[c.Label] = id
	}
	return out
}

// The player's eighteen, named so the rules' own tests, the balance tool and the screen can say
// `combat.Strike` rather than looking a string up.
//
// **Six of them ship at zero copies** *(2026-08-24)* — the 0 AP and 4 AP rung of each attack form.
// They are not in the starting deck and cannot be bought; the only way to hold one is a Shrink or a
// Grow worm walking a card off the end of the middle three. They are registered concepts all the
// same, because `Neighbour` derives the ladder from this registry and a rung that does not exist is
// a rung a worm cannot step onto.
//
// **They are resolved from the file rather than defining it.** Renaming a card in
// `duelist_cards.json` fails here, at startup, with the name that went missing — which is the loud
// failure a deck quietly one concept short never gave.
//
// Each depends on `playerConcepts`, which is what orders these after registration: Go initialises a
// package-level variable after everything its initialiser references.
var (
	// Stab.
	Poke   = mustPlayer("Poke")
	Jab    = mustPlayer("Jab")
	Thrust = mustPlayer("Thrust")
	Lunge  = mustPlayer("Lunge")
	Impale = mustPlayer("Impale")

	// Slash.
	Nick   = mustPlayer("Nick")
	Cut    = mustPlayer("Cut")
	Slash  = mustPlayer("Slash")
	Cleave = mustPlayer("Cleave")
	Sever  = mustPlayer("Sever")

	// Crush.
	Tap       = mustPlayer("Tap")
	Bash      = mustPlayer("Bash")
	Strike    = mustPlayer("Strike")
	Smash     = mustPlayer("Smash")
	Pulverize = mustPlayer("Pulverize")

	// Defend. One card per shield, priced at one AP each — the ladder the three attack forms
	// use, with the count in place of the damage multiplier.
	Ward  = mustPlayer("Ward")
	Brace = mustPlayer("Brace")
	Guard = mustPlayer("Guard")
)

func mustPlayer(label string) ConceptID {
	id, ok := playerConcepts[label]
	if !ok {
		panic("duelist_cards.json no longer has a card called " + label)
	}
	return id
}

// PlayerConcepts is every concept from the player's deck list, in file order. The deck overlay and
// the card sheet walk it.
func PlayerConcepts() []ConceptID {
	out := make([]ConceptID, 0, len(playerConcepts))
	for id := ConceptID(0); int(id) < len(registry) && int(id) < len(playerConcepts); id++ {
		out = append(out, id)
	}
	return out
}

// Tier is where a card sits on its form's ladder, and it is **the concept's own cost**.
//
// The player's attacks are a 3x3 grid — three forms by three tiers at 1/2/3 AP for 0.5x/1x/2x
// damage — so the price *is* the rung. Reading the declared cost rather than a per-card one is
// deliberate: a worm that cheapened a Strike must not thereby turn it into a Jab.
func (c Concept) Tier() int { return c.Cost }

// Neighbour is the concept one rung up or down the same form's ladder, or false if there is
// none — the top of a form cannot be promoted and the bottom cannot be demoted.
//
// **A form with no name has no ladder.** Every enemy card is `FormNone`, and they share this
// registry with the player's, so matching on the zero form would step a Goblin's Bite onto a
// Slime's. The player's nine attacks are the only cards with a form, which is exactly the set
// that has a ladder to walk.
//
// It scans the registry rather than reading a table, so the ladder is a *consequence* of what
// `data/duelist_cards.json` declares rather than a second list to keep in step with it.
func Neighbour(id ConceptID, step int) (ConceptID, bool) {
	from := ConceptOf(id)
	if from.Form == FormNone || from.Verb != VerbAttack {
		return NoConcept, false
	}

	want := from.Tier() + step
	for other := range registry {
		c := registry[other]
		if c.Form == from.Form && c.Verb == VerbAttack && c.Tier() == want {
			return ConceptID(other), true
		}
	}
	return NoConcept, false
}
