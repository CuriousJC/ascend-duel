package data

// The relics: what the player can equip between fights.
//
// **A relic is written in a grammar as of 2026-08-17.** It was a record naming one element, which
// `internal/screens` turned into a flag — that held the four elemental relics and nothing else, because
// a form multiplier and a vitae relic have no element to be a bit under. A relic is now a list of
// `When` / `If` / `Then` rules: the moment that wakes it, what has to be true, and what happens. The
// full vocabulary, the code seat each moment lands on, and the questions to put to a new relic idea
// are in `.claude/skills/relics/SKILL.md`; MECHANICS.md holds the argument for the shape.
//
// **The strings here are resolved by `internal/session`**, which parses them into `combat` rules
// types and registers each relic. That is the same division the deck lists draw: this package holds a
// vocabulary of words and knows nothing about what any of them do, so the rules never read a file
// carrying an art key and this file never grows an opinion about damage.
//
// **A vocabulary word this file invents does not exist.** An unknown moment, a verb used at the wrong
// moment, a status key in no file — every one of them fails at load rather than producing a relic that
// wears cleanly and does nothing. See `combat.RegisterRelic`, which is where the grammar is enforced.
//
// **Art is an assets key, not a path**, like every other named asset — see assets/embed.go. It has to
// be a key `LoadImageData` hands back rather than one `LoadAssets` does, because a relic's artwork is
// drawn *into* a card by internal/cards, which has no graphics context.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed relics.json
var relicsJSON []byte

// RelicData is one relic.
type RelicData struct {
	// RelicRecord is the key, and what anything holding a relic stores. Kebab-case, matching
	// the art filenames rather than the display name — a record key that is a sentence is a
	// key nobody can type twice the same way.
	//
	// **It is the identity a growing relic's accumulator is filed under** on `Session`, which is why
	// it is the key and not a position in this file.
	RelicRecord string `json:"RelicRecord"`

	// Name is the relic's full name — what a tooltip says and what this file is read by. **The
	// card face does not carry it**: a relic card is a full-bleed picture with no title, so the
	// name is read in the tooltip and in the shop rather than looked at on the face.
	Name string `json:"Name"`

	// Family is the motif this relic belongs to — the block of siblings it was authored beside,
	// and the heading it is reviewed under on the relic sheet.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw.** It groups the review page and
	// nothing else reads it; a relic with no Family still loads, still sells and still fires.
	//
	// **It is a restatement of the rules in words, and it is authored on purpose** *(owner's call,
	// 2026-09-12)*. Nearly every value here is already implied by the record's own
	// (When, Do, predicate) — the flips are all card-drawn/set-element, the weapons all
	// card-damage/scale-damage on a Concept — so this is not a fact the file knows and the rules do
	// not. **What it buys is legibility for whoever is authoring the catalogue**: a signature is
	// something to decode and "Jade rings" is something to read, and the three ring families differ
	// by the gem in the picture as much as by the axis in the rule.
	//
	// So the usual objection to an authored tag does not apply the way it does to CostTier, which
	// the rules also consulted: nothing resolves a round differently because of this string. What
	// it can still do is go quietly out of date — a relic retuned into a different family keeps the
	// label it was born with and no test fails. **Re-read the block when you change a relic's
	// rules**, and treat a Family that disagrees with the rules as a note to fix rather than as a
	// second opinion about what the relic is.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the
	// default relic face** — see ArtKey; an unknown name draws a relic with no artwork and logs
	// once, the same choice the enemy portraits make, and for the same reason: a card with a
	// hole in it gets reported, a game that refuses to start over a missing picture is worse.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this relic — what the object
	// *is* and what the effect is doing to it, in one sentence. **Nothing in the game reads it**,
	// exactly like Art's own key and a status's Badge; it is here because it is the one place a
	// relic's identity is written down beside the rules that made it.
	//
	// **It is on the record so that regenerating a picture does not mean writing its brief again**
	// *(owner's call, 2026-09-12)*. The *generic* prompt is still in docs/art/, because that one is
	// shared by every card and is about no record at all.
	//
	// **Empty means nobody has written one yet**, which — read against an empty Art — is what the
	// relic sheet reports as the backlog.
	Draw string `json:"Draw"`

	// Text is one line saying what the relic does, for the long press that does not exist yet.
	// Written now because it is the thing whoever adds a relic will want to write down, and a
	// field added later is a field every existing entry is missing.
	Text string `json:"Text"`

	// Rarity is how often the shop offers it, and — through that — what it costs. **One word
	// decides both**: `common`, `uncommon` or `rare`. A relic does not name a price, because a
	// catalogue where every relic priced itself drifted into seventeen numbers nobody could hold
	// against each other; three tiers can be read at a glance and a relic can only be moved between
	// them.
	//
	// **What it sells back for is not a field either.** That is the tier's own figure — 1, 2 or 3
	// — and it is one rule of the shop.
	Rarity Rarity `json:"Rarity"`

	// Rules is what wearing this relic actually does. **A list, forced by the growing stat relics**,
	// which accumulate at one moment and apply at another; it generalises to any relic wanting two.
	Rules []RelicRuleData `json:"Rules"`
}

// RelicRuleData is one `When` / `If` / `Then`.
type RelicRuleData struct {
	// When is the moment that wakes this rule: one of `card-cost`, `card-damage`, `attack-lands`,
	// `deck-built`, `fight-start`, `fight-won`, `prizes-dealt`, `blow-formed` or `turn-taken`.
	// Closed, and each has one Go seat.
	When string `json:"When"`

	// If is what has to be true. **Absent means the rule always fires**, which is what the stat
	// relics and the two vitae relics want — hence a pointer rather than a struct, so "no predicate"
	// and "a predicate that constrains nothing" are not the same value.
	//
	// The three moments outside combat have no card to match one against, and a predicate on one of
	// them is refused at load rather than quietly matching everything.
	If *RelicIfData `json:"If,omitempty"`

	// Then is what happens, as a list — which is what buys a relic that shocks *and* chills with no
	// new vocabulary at all.
	Then []RelicEffectData `json:"Then"`
}

// RelicIfData is a rule's predicate. **Every field that is set has to match**, so two of them narrow a
// rule rather than widening it.
type RelicIfData struct {
	// Element is the card's colour: `fire`, `ice`, `lightning`, `earth` or `basic`.
	Element string `json:"Element,omitempty"`

	// Form is `stab`, `slash`, `crush` or `defend`.
	Form string `json:"Form,omitempty"`

	// Tier is the rung of its form's ladder a card sits on, which is the cost printed on it: 1, 2
	// or 3. **The declared cost and never the wearer's**, so a discount relic cannot quietly move a
	// card out of a rule's reach.
	Tier int `json:"Tier,omitempty"`

	// Lead narrows the rule to the **first attack card of the blow** — the only predicate that is
	// not a fact about the card. Meaningful at `blow-formed` and refused anywhere else, since no
	// other moment knows which card leads.
	Lead bool `json:"Lead,omitempty"`

	// Concept names one card by its label — `Bash`. Resolved at load the way a deck list is,
	// because a concept's ID is registration-ordered and must never be written in a file.
	//
	// **A concept relic is a much narrower object than a form relic** and pricing them alike is a
	// mistake waiting to happen: Striker covers 4 cards where Keen covers 12.
	Concept string `json:"Concept,omitempty"`

	// Hand names one rung of the ladder by its `hands.json` key — `concept-full-house`. It narrows
	// the rule to blows that formed exactly that rung.
	//
	// **A key rather than the id, for the reason a concept is a label**: `HandID` is a number in a
	// file that outlives the build that wrote it. Meaningful at `blow-formed` and refused anywhere
	// else, and refused alongside any card predicate — a hand is a fact about the whole blow.
	Hand string `json:"Hand,omitempty"`

	// MinForms is how many distinct forms the blow's scoring cards must cover.
	MinForms int `json:"MinForms,omitempty"`
}

// RelicEffectData is one entry in a rule's `Then`. Which fields mean anything depends on `Do`, the same
// way a card's Amount is read against its verb.
type RelicEffectData struct {
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

// DefaultRelicArt is the face a record with no Art of its own draws.
//
// **Most of the file has no art**, and it will stay that way: the four elemental relics were drawn
// before the grammar existed, and every relic written since — the form multipliers, the two vitae
// relics, the growing stat relics — is a rule with no picture. A pink border around an empty face
// reads as a card that failed to load; this reads as one waiting for art.
const DefaultRelicArt = "default-relic"

// ArtKey is the picture this relic actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the one call site** because a relic is drawn in three places — the
// worn row, the shop shelf and tools/relicsheet — and a fallback living in a screen is a fallback
// the review tool does not have, which is exactly how a sheet comes to disagree with the game.
func (r RelicData) ArtKey() string {
	if r.Art == "" {
		return DefaultRelicArt
	}
	return r.Art
}

// LoadRelics parses the embedded relic list into a map keyed by RelicRecord.
func LoadRelics() map[string]RelicData {
	var list []RelicData
	if err := json.Unmarshal(relicsJSON, &list); err != nil {
		panic("Failed to unmarshal our RelicData: " + err.Error())
	}

	out := make(map[string]RelicData, len(list))
	for _, r := range list {
		out[r.RelicRecord] = r
	}
	return out
}

// RelicOrder is every record, sorted by key.
//
// **Sorted because LoadRelics returns a map and Go randomises that order**, exactly like
// EnemyOrder. The relic pane draws whatever this walks, so map order would deal a different
// row of relics every launch — a determinism breach that would look like a bug in the layout.
//
// By key rather than by name or element: it is the one field guaranteed unique, and a sort
// on something that can tie is a sort that can still shuffle.
// RelicFileOrder is every record id in the order data/relics.json writes them.
//
// **This is the motif order the file is authored in** — the flips together, the three ring
// families in ladder order, the weapons along the concept ladder — which is information
// RelicOrder's sorted keys throw away. The relic sheet groups by Family and walks this, so the
// page reads as the file does.
//
// It is a second walk of the JSON rather than an ordering stored on the map, because a map has no
// order to store one on and every other caller wants the sorted keys.
func RelicFileOrder() []string {
	var list []RelicData
	if err := json.Unmarshal(relicsJSON, &list); err != nil {
		panic("Failed to unmarshal our RelicData: " + err.Error())
	}
	out := make([]string, 0, len(list))
	for _, r := range list {
		out = append(out, r.RelicRecord)
	}
	return out
}

func RelicOrder(relics map[string]RelicData) []string {
	keys := make([]string, 0, len(relics))
	for k := range relics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
