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
	"strings"
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

	// Name is the ring's full name — what a tooltip says and what this file is read by. **The
	// card face carries FaceName instead**, which is this without the trailing "Ring".
	Name string `json:"Name"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the
	// default ring face** — see ArtKey; an unknown name draws a ring with no artwork and logs
	// once, the same choice the enemy portraits make, and for the same reason: a card with a
	// hole in it gets reported, a game that refuses to start over a missing picture is worse.
	Art string `json:"Art"`

	// Text is one line saying what the ring does, for the long press that does not exist yet.
	// Written now because it is the thing whoever adds a ring will want to write down, and a
	// field added later is a field every existing entry is missing.
	Text string `json:"Text"`

	// Rarity is how often the shop offers it, and — through that — what it costs. **One word
	// decides both**: `common`, `uncommon` or `rare`. A ring does not name a price, because a
	// catalogue where every ring priced itself drifted into seventeen numbers nobody could hold
	// against each other; three tiers can be read at a glance and a ring can only be moved between
	// them.
	//
	// **What it sells back for is not a field either.** That is the tier's own figure — 1, 2 or 3
	// — and it is one rule of the shop.
	Rarity Rarity `json:"Rarity"`

	// Rules is what wearing this ring actually does. **A list, forced by the growing stat rings**,
	// which accumulate at one moment and apply at another; it generalises to any ring wanting two.
	Rules []RingRuleData `json:"Rules"`
}

// RingRuleData is one `When` / `If` / `Then`.
type RingRuleData struct {
	// When is the moment that wakes this rule: one of `card-cost`, `card-damage`, `attack-lands`,
	// `deck-built`, `fight-start`, `fight-won`, `prizes-dealt`, `blow-formed` or `turn-taken`.
	// Closed, and each has one Go seat.
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

	// Tier is the rung of its form's ladder a card sits on, which is the cost printed on it: 1, 2
	// or 3. **The declared cost and never the wearer's**, so a discount ring cannot quietly move a
	// card out of a rule's reach.
	Tier int `json:"Tier,omitempty"`

	// Lead narrows the rule to the **first attack card of the blow** — the only predicate that is
	// not a fact about the card. Meaningful at `blow-formed` and refused anywhere else, since no
	// other moment knows which card leads.
	Lead bool `json:"Lead,omitempty"`

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
	// `add-hp`, `scale-hp`, `grow-on-win`, `grow-on-hit`, `scale-propagation`, `adjust-picks`,
	// `adjust-prize-vitae`,
	// `echo-attack`, `repeat-card`, `demote-card`, `grow-on-hit`, `grow-on-turn` or `reset-growth`.
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

// DefaultRingArt is the face a record with no Art of its own draws.
//
// **Most of the file has no art**, and it will stay that way: the four elemental rings were drawn
// before the grammar existed, and every ring written since — the form multipliers, the two vitae
// rings, the growing stat rings — is a rule with no picture. A pink border around an empty face
// reads as a card that failed to load; this reads as one waiting for art.
const DefaultRingArt = "defaultring_png"

// ArtKey is the picture this ring actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the one call site** because a ring is drawn in three places — the
// worn row, the shop shelf and tools/ringsheet — and a fallback living in a screen is a fallback
// the review tool does not have, which is exactly how a sheet comes to disagree with the game.
func (r RingData) ArtKey() string {
	if r.Art == "" {
		return DefaultRingArt
	}
	return r.Art
}

// FaceName is the name as it is written across the top of a ring card: the record's Name with a
// trailing "Ring" removed.
//
// **The card is already obviously a ring** *(owner's call, 2026-08-21)* — pink border, a picture
// of a ring, in a row of rings — so the word says nothing and costs the name its width. "Frozen
// Lightning Ring" is three lines on a 162-pixel card where "Frozen Lightning" is two, and the
// line that goes is the one carrying no information.
//
// **The full name survives everywhere it is read rather than looked at**: the tooltip titles a
// ring with `Name`, and so does the shop. This is a fact about the face, which is why it is a
// second method rather than an edit to `rings.json` — a file that spelled the name without its
// noun would leave nothing able to say "Frozen Lightning Ring" in a sentence.
//
// A record named only "Ring" keeps it, since a card with no name on it is worse than a
// redundant one.
func (r RingData) FaceName() string {
	short := strings.TrimSpace(strings.TrimSuffix(r.Name, "Ring"))
	if short == "" {
		return r.Name
	}
	return short
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
