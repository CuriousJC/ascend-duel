package data

// The essences: **what the player may do to their deck between fights.**
//
// It is the card language's shape applied to a different subject. A card record says what a card
// *does* when it is played; an essence record says what it *changes* about a card that already
// exists — a target and, where the target needs one, a new value.
//
// **What is deliberately not here.** Which essences are offered, how many, and what they cost are
// not fields on a record: the offer is two drawn at random and nothing is bought yet. Whichever
// of those becomes real is a field beside Target when it does, and not before — the mistake
// `CostTier` was, which declared a rules vocabulary in JSON ahead of the rules, is the one to
// keep not making.
//
// **The rules do not read this file**, and cannot: an essence is applied to the *run's* deck, which
// is `internal/session`. That is the same test every file here answers — who consumes it — and
// it is why the parsing and the validation live over there rather than in `internal/combat`.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed essences.json
var essencesJSON []byte

// EssenceData is one alteration the player can be offered.
type EssenceData struct {
	// EssenceRecord is the key, and what anything holding an essence stores. Kebab-case, like a relic's.
	EssenceRecord string `json:"EssenceRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Family is the motif this essence belongs to — the block of siblings it was authored beside,
	// and the heading it is reviewed under on the essence sheet.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw.** It groups the review page and
	// nothing else reads it; an essence with no Family still loads and is still offered.
	//
	// **It is authored rather than derived**, the relic catalog's field brought over. Nearly every
	// value here is implied by the record's own Target and Value, so it is legibility for whoever is
	// authoring rather than a fact the file knows and the rules do not — and it can go quietly out of
	// date when an essence is retargeted, with no test failing. Re-read the block when you change
	// what an essence does.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the default
	// essence face** — see ArtKey.
	//
	// **The fallback is `ArtKey` rather than a constant in a screen**, which is the shape
	// `RelicData.ArtKey` has, and for its reason: a fallback living in `internal/screens` is one the
	// review tool does not have, which is how a sheet comes to disagree with the game.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this essence — what the thing
	// *is* and what it is doing, in one sentence. **Nothing in the game reads it**, exactly like
	// Art's own key and a relic's Draw.
	//
	// **Empty means nobody has written one yet**, which — read against an empty Art — is what the
	// essence sheet reports as the backlog.
	Draw string `json:"Draw"`

	// Target is which aspect of a card this essence changes. **A closed vocabulary, and
	// `session.EssenceTarget` is the list** — `element`, `remove`, `duplicate`, `cost`, `amount`,
	// `promote`, `demote`, `form` — resolved by `session.ParseEssenceTarget`, which reports failure
	// rather than falling back.
	//
	// **Closing it is the point**, exactly as with a card's verb: an essence quietly registered as a
	// recolor because its target was misspelled is a mechanic nobody designed. **What a new target
	// costs is a per-card field on `combat.Card`**, since everything else about a card lives on the
	// shared concept and altering one copy would otherwise alter every copy in the deck. That price
	// is worth charging deliberately rather than discovering because a JSON file asked for it — see
	// `session.EssenceTarget`, which holds the argument for each target it has been paid for.
	Target string `json:"Target"`

	// Value is the new value, read against the target: an element name for `element`, a form name
	// for `form`, a signed delta for `cost`, a percentage for `amount`. **Empty for the targets that
	// need none, and refused if one is supplied anyway** — a value nothing reads is somebody
	// expecting something the mechanic does not do.
	Value string `json:"Value,omitempty"`

	// Text is what the card says it does, in the same clipped register the action cards use — the
	// column is about a dozen characters wide, and `TestEveryEssenceTextFitsItsCard` fails on a
	// string that wraps past the band rather than letting it run off the bottom of the card.
	//
	// **A `\n` is an authored line break** *(2026-08-23)*, honored by `cards.WrapText` before the
	// width is measured and split back into lines by the tooltip. It can only ever add a line, since
	// a too-wide authored line still wraps, so it is not a way past the column.
	//
	// **The elemental essences have to break in the same place**, because they differ only in the
	// element they name and left to the measurer they would read as five layouts of one card. They
	// do it by wrapping rather than by an authored break today, and
	// `TestTheElementalEssencesAllBreakInTheSamePlace` is what holds the set together either way.
	Text string `json:"Text"`
}

// DefaultEssenceArt is the face a record with no Art of its own draws: assets/essence/default-essence.png.
//
// **Keys are not file paths** — LoadImageData files that picture under this, which is what a
// lookup has to spell. Writing the filename instead is why the first version of the essence card drew
// nothing and logged `no artwork named "default-relic"`.
const DefaultEssenceArt = "default-essence"

// ArtKey is the picture this essence actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: an essence is drawn in
// the reward screen and in tools/essencesheet, and a fallback living in a screen is a fallback the
// review tool does not have — which is exactly how a sheet comes to disagree with the game.
func (w EssenceData) ArtKey() string {
	if w.Art == "" {
		return DefaultEssenceArt
	}
	return w.Art
}

// EssenceFileOrder is every record id in the order data/essences.json writes them.
//
// **File order rather than EssenceOrder's sorted keys**, and it is for the review page alone: the
// catalog is authored in motif order — the five recolors together, the two that resize a card
// beside each other — and sorting by key throws exactly that away. It is as deterministic as
// sorted order and carries more.
//
// **Nothing that decides an outcome may walk this.** The offer is a shuffle of EssenceOrder, which is
// sorted for the reason the randomness skill gives; this is a layout.
func EssenceFileOrder() []string {
	var list []EssenceData
	if err := json.Unmarshal(essencesJSON, &list); err != nil {
		panic("Failed to unmarshal essences.json: " + err.Error())
	}
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.EssenceRecord)
	}
	return out
}

// LoadEssences parses the catalog into a map keyed by EssenceRecord.
func LoadEssences() map[string]EssenceData {
	var list []EssenceData
	if err := json.Unmarshal(essencesJSON, &list); err != nil {
		panic("Failed to unmarshal essences.json: " + err.Error())
	}

	out := make(map[string]EssenceData, len(list))
	for _, w := range list {
		out[w.EssenceRecord] = w
	}
	return out
}

// EssenceOrder is every record, sorted by key.
//
// **Sorted because LoadEssences returns a map and Go randomizes that order**, and this one decides
// an outcome rather than a layout: the offer is a shuffle of this list, so an unsorted walk would
// make which essences you are offered depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func EssenceOrder(essences map[string]EssenceData) []string {
	names := make([]string, 0, len(essences))
	for n := range essences {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
