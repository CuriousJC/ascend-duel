package combat

// The concept registry: what a card *is*, as data rather than as a switch statement.
//
// **This replaced `ActionKind` on 2026-08-16.** A card's cost, damage, category and family were
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
// posture `combos.json` takes with its reward kinds. Adding a fifth verb is a Go change plus one
// place applying it, and that cost is charged deliberately: a card language that could express
// anything would be a scripting language, and the rules would stop being readable in one file.
type Verb int

const (
	// VerbAttack deals damage. Aimed at the opponent it is every attack card in the game; aimed
	// at self it is recoil.
	VerbAttack Verb = iota

	// VerbDefend reduces the one blow it answers, by Amount percent.
	VerbDefend

	// VerbBank stores action points for the following round.
	VerbBank

	// VerbDraw widens the following round's hand.
	VerbDraw
)

// Verbs is every verb in a fixed order, for anything that walks them.
func Verbs() []Verb { return []Verb{VerbAttack, VerbDefend, VerbBank, VerbDraw} }

func (v Verb) String() string {
	switch v {
	case VerbDefend:
		return "defend"
	case VerbBank:
		return "bank"
	case VerbDraw:
		return "draw"
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

// Target is who a card lands on.
//
// **It is not merely derivable from the verb, and recoil is why.** Every other pairing in the grid
// is the obvious one — a defend shields its own duelist, a bank fills its own budget — so without
// an attack that can be aimed inward this field would carry no information and no rule would read
// it. Recoil is what makes it decide something.
type Target int

const (
	TargetOpponent Target = iota
	TargetSelf
)

func (t Target) String() string {
	if t == TargetSelf {
		return "self"
	}
	return "opponent"
}

// ParseTarget resolves a target from its name. An empty name is not an error — it means the
// obvious target for the verb, which `defaultTarget` decides.
func ParseTarget(name string) (Target, bool) {
	switch name {
	case "opponent":
		return TargetOpponent, true
	case "self":
		return TargetSelf, true
	default:
		return TargetOpponent, false
	}
}

// defaultTarget is where a card with no declared target lands: an attack on the opponent,
// everything else on its own duelist.
func defaultTarget(v Verb) Target {
	if v == VerbAttack {
		return TargetOpponent
	}
	return TargetSelf
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
	Target Target
	Family Family
}

// Recoils reports whether this concept turns its damage on its own owner.
func (c Concept) Recoils() bool { return c.Verb == VerbAttack && c.Target == TargetSelf }

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

	target := defaultTarget(verb)
	if c.Target != "" {
		t, ok := ParseTarget(c.Target)
		if !ok {
			return NoConcept, fmt.Errorf("%s names target %q, want \"opponent\" or \"self\"", key, c.Target)
		}
		target = t
	}

	// **The unbuilt half of the grid is refused rather than accepted and ignored.** Banking or
	// drawing against an opponent is designed — drain and mill — and nothing resolves either, so a
	// card asking for one would silently act on its own duelist instead. See MECHANICS.md.
	if verb != VerbAttack && target != TargetSelf {
		return NoConcept, fmt.Errorf("%s is a %s aimed at the opponent, which nothing resolves yet", key, verb)
	}

	family := FamilyNone
	if c.Family != "" {
		f, ok := ParseFamily(c.Family)
		if !ok {
			return NoConcept, fmt.Errorf("%s names family %q, which the rules do not have", key, c.Family)
		}
		family = f
	}

	if c.Cost < 0 {
		return NoConcept, fmt.Errorf("%s costs %d", key, c.Cost)
	}
	if c.Amount <= 0 {
		return NoConcept, fmt.Errorf("%s has Amount %d, so it does nothing", key, c.Amount)
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
		Target: target,
		Family: family,
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
// damage are rules by definition. This is the second such file after `combos.json`, and for the
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

// The player's twelve, named so the rules' own tests, the balance tool and the screen can say
// `combat.Strike` rather than looking a string up.
//
// **They are resolved from the file rather than defining it.** Renaming a card in
// `duelist_cards.json` fails here, at startup, with the name that went missing — which is the loud
// failure a deck quietly one concept short never gave.
//
// Each depends on `playerConcepts`, which is what orders these after registration: Go initialises a
// package-level variable after everything its initialiser references.
var (
	// Stab.
	Jab    = mustPlayer("Jab")
	Thrust = mustPlayer("Thrust")
	Lunge  = mustPlayer("Lunge")

	// Slash.
	Cut    = mustPlayer("Cut")
	Slash  = mustPlayer("Slash")
	Cleave = mustPlayer("Cleave")

	// Crush.
	Bash   = mustPlayer("Bash")
	Strike = mustPlayer("Strike")
	Smash  = mustPlayer("Smash")

	// Plan. The concept named Plan is one card of the family named plan, which is a collision of
	// words rather than of meanings: the family is what the card's corner says and this is the
	// card that draws.
	Prepare = mustPlayer("Prepare")
	Plan    = mustPlayer("Plan")
	Defend  = mustPlayer("Defend")
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
