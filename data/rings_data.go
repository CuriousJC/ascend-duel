package data

// The rings: what the player can equip between fights.
//
// **A ring is written in a grammar as of 2026-08-17.** It was a record naming one element, which
// `internal/screens` turned into a flag — that held the four elemental rings and nothing else, because
// a form multiplier and a vitae ring have no element to be a bit under. A ring is now a list of
// `When` / `If` / `Then` rules: the moment that wakes it, what has to be true, and what happens. The
// full vocabulary, the code seat each moment lands on, and the questions to put to a new ring idea
// are in `.claude/skills/rings/SKILL.md`; MECHANICS.md holds the argument for the shape.
//
// **The strings here are resolved by `internal/session`**, which parses them into `combat` rules
// types and registers each ring. That is the same division the deck lists draw: this package holds a
// vocabulary of words and knows nothing about what any of them do, so the rules never read a file
// carrying an art key and this file never grows an opinion about damage.
//
// **A vocabulary word this file invents does not exist.** An unknown moment, a verb used at the wrong
// moment, a status key in no file — every one of them fails at load rather than producing a ring that
// wears cleanly and does nothing. See `combat.RegisterRing`, which is where the grammar is enforced.
//
// **Art is an assets key, not a path**, like every other named asset — see assets/embed.go. It has to
// be a key `LoadImageData` hands back rather than one `LoadAssets` does, because a ring's artwork is
// drawn *into* a card by internal/cards, which has no graphics context.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed rings.json
var ringsJSON []byte

// RingData is one ring.
type RingData struct {
	// RingRecord is the key, and what anything holding a ring stores. Kebab-case, matching
	// the art filenames rather than the display name — a record key that is a sentence is a
	// key nobody can type twice the same way.
	//
	// **It is the identity a growing ring's accumulator is filed under** on `Session`, which is why
	// it is the key and not a position in this file.
	RingRecord string `json:"RingRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Art is the assets.LoadImageData key for the picture on the face. An empty or unknown
	// name draws a ring with no artwork and logs once — the same choice the enemy portraits
	// make, and for the same reason: a card with a hole in it gets reported, a game that
	// refuses to start over a missing picture is worse.
	Art string `json:"Art"`

	// Text is one line saying what the ring does, for the long press that does not exist yet.
	// Written now because it is the thing whoever adds a ring will want to write down, and a
	// field added later is a field every existing entry is missing.
	Text string `json:"Text"`

	// Rules is what wearing this ring actually does. **A list, forced by the growing stat rings**,
	// which accumulate at one moment and apply at another; it generalises to any ring wanting two.
	Rules []RingRuleData `json:"Rules"`
}

// RingRuleData is one `When` / `If` / `Then`.
type RingRuleData struct {
	// When is the moment that wakes this rule: one of `card-cost`, `card-damage`, `attack-lands`,
	// `deck-built`, `fight-start`, `fight-won` or `prizes-dealt`. Closed, and each has one Go seat.
	When string `json:"When"`

	// If is what has to be true. **Absent means the rule always fires**, which is what the stat
	// rings and the two vitae rings want — hence a pointer rather than a struct, so "no predicate"
	// and "a predicate that constrains nothing" are not the same value.
	//
	// The three moments outside combat have no card to match one against, and a predicate on one of
	// them is refused at load rather than quietly matching everything.
	If *RingIfData `json:"If,omitempty"`

	// Then is what happens, as a list — which is what buys a ring that shocks *and* chills with no
	// new vocabulary at all.
	Then []RingEffectData `json:"Then"`
}

// RingIfData is a rule's predicate. **Every field that is set has to match**, so two of them narrow a
// rule rather than widening it.
type RingIfData struct {
	// Element is the card's colour: `fire`, `ice`, `lightning`, `earth` or `basic`.
	Element string `json:"Element,omitempty"`

	// Form is `stab`, `slash`, `crush` or `plan`.
	Form string `json:"Form,omitempty"`

	// Concept names one card by its label — `Strike`. Resolved at load the way a deck list is,
	// because a concept's ID is registration-ordered and must never be written in a file.
	//
	// **A concept ring is a much narrower object than a form ring** and pricing them alike is a
	// mistake waiting to happen: Striker covers 4 cards where Keen covers 12.
	Concept string `json:"Concept,omitempty"`
}

// RingEffectData is one entry in a rule's `Then`. Which fields mean anything depends on `Do`, the same
// way a card's Amount is read against its verb.
type RingEffectData struct {
	// Do is the effect verb: `adjust-cost`, `scale-damage`, `apply-status`, `set-element`, `add-dmg`,
	// `add-hp`, `grow`, `scale-propagation`, `adjust-picks` or `adjust-prize-vitae`.
	//
	// **One word carrying both the operation and its subject** *(owner's call, 2026-08-17)*, rather
	// than an operation crossed with a subject: two lists would buy a grid that is mostly
	// meaningless cells, and `apply-status` sits on neither axis.
	Do string `json:"Do"`

	// Amount is the figure, read against the verb: a signed cost delta, a percentage where 200 is
	// double, flat DMG or HP, or how much an accumulator grows.
	Amount int `json:"Amount,omitempty"`

	// Status is the record key `apply-status` applies — see statuses.json.
	Status string `json:"Status,omitempty"`

	// Element is what `set-element` recolours a matching card to.
	Element string `json:"Element,omitempty"`
}

// LoadRings parses the embedded ring list into a map keyed by RingRecord.
func LoadRings() map[string]RingData {
	var list []RingData
	if err := json.Unmarshal(ringsJSON, &list); err != nil {
		panic("Failed to unmarshal our RingData: " + err.Error())
	}

	out := make(map[string]RingData, len(list))
	for _, r := range list {
		out[r.RingRecord] = r
	}
	return out
}

// RingOrder is every record, sorted by key.
//
// **Sorted because LoadRings returns a map and Go randomises that order**, exactly like
// EnemyOrder. The ring pane draws whatever this walks, so map order would deal a different
// row of rings every launch — a determinism breach that would look like a bug in the layout.
//
// By key rather than by name or element: it is the one field guaranteed unique, and a sort
// on something that can tie is a sort that can still shuffle.
func RingOrder(rings map[string]RingData) []string {
	keys := make([]string, 0, len(rings))
	for k := range rings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
