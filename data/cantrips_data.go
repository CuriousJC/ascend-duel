package data

// The cantrips: **what the player may do to the duelist for one fight.**
//
// A rune alters a card; a potion alters the duelist for the rest of the run. A cantrip alters the
// duelist for the fight it is cast in and no longer — it is carried into a duel in the consumables
// pane, cast between turns, and gone when the duel ends.
//
// **Every cantrip is a scroll and a relic.** Casting the scroll puts its cantrip-relic on the
// duelist for the rest of the fight, weightless and ephemeral, so whatever a cantrip does is said in
// the relic grammar and answered by the relic machinery — there is no second vocabulary for altering
// a fight. The relic is written inline on the cantrip and never enters `relics.json`: a
// cantrip-relic is not a relic the shop can sell, and the two catalogs are kept apart.
//
// **The rules do not read this file.** A cantrip is carried by a run, so the parsing lives in
// `internal/session`, which registers each cantrip-relic's rules with `combat.RegisterRelic` exactly
// as it registers the catalog's. See internal/session/cantrip.go.

import (
	_ "embed"
)

//go:embed cantrips.json
var cantripsJSON []byte

// CantripData is one record of data/cantrips.json.
type CantripData struct {
	// CantripRecord is the key, and what anything holding a cantrip stores. Kebab-case, like a
	// rune's and a relic's.
	CantripRecord string `json:"CantripRecord"`

	// Name is what the cantrip is called — the tooltip's title and the ledger's noun.
	Name string `json:"Name"`

	// Family, Art and Draw are the three fields no rule reads, on the terms every other catalog
	// carries them: the block a record was authored beside, the assets key of its face (empty means
	// DefaultCantripArt), and the subject paragraph an art generator is given.
	Family string `json:"Family"`
	Art    string `json:"Art"`
	Draw   string `json:"Draw"`

	// Text is what the scroll says it does, in the tooltip. **Authored rather than computed**, the
	// rune's posture: the line is the record's claim and the relic's rules are what fire.
	Text string `json:"Text"`

	// Relic is the cantrip-relic casting this scroll puts on the duelist for the fight.
	Relic CantripRelicData `json:"Relic"`
}

// CantripRelicData is the relic a cantrip casts: a relic record's face and rules, without the
// fields that only mean something on a shelf. **No key of its own** — it is registered under the
// cantrip's record key, since one cantrip makes exactly one relic. **No Rarity and no Unlock**,
// because it is never sold and never offered.
type CantripRelicData struct {
	// Name is what the relic is called in the row's tooltip and in the fight's account.
	Name string `json:"Name"`

	// Art and Draw are the relic's own face and brief, beside the scroll's: a cantrip has two
	// pictures. Empty Art draws DefaultCantripRelicArt.
	Art  string `json:"Art"`
	Draw string `json:"Draw"`

	// Text is the relic's line, as a worn relic's tooltip says it.
	Text string `json:"Text"`

	// Rules are written exactly as a relic's are — see RelicRuleData.
	Rules []RelicRuleData `json:"Rules"`
}

// DefaultCantripRelicArt is the face a cantrip-relic with no Art of its own draws: the relic
// catalog's placeholder, until the cantrip-relics have a default of their own.
const DefaultCantripRelicArt = DefaultRelicArt

// ArtKey is the picture this cantrip-relic actually draws.
func (r CantripRelicData) ArtKey() string {
	if r.Art == "" {
		return DefaultCantripRelicArt
	}
	return r.Art
}

// DefaultCantripArt is the face a record with no Art of its own draws —
// `assets/cantrip/default-cantrip.png`, a seat of the catalog's own for the reason the rune's
// default is one: two catalogs wearing one placeholder are two backlogs nobody can tell apart.
const DefaultCantripArt = "default-cantrip"

// ArtKey is the picture this cantrip actually draws: its own if it has one, the default otherwise.
func (c CantripData) ArtKey() string {
	if c.Art == "" {
		return DefaultCantripArt
	}
	return c.Art
}

// LoadCantrips parses the catalog into a map keyed by CantripRecord.
func LoadCantrips() map[string]CantripData {
	return keyed(cantripsJSON, "cantrips.json", func(c CantripData) string { return c.CantripRecord })
}

// CantripOrder is every record, sorted by key.
//
// **Sorted because LoadCantrips returns a map and Go randomizes that order**, and this one decides
// an outcome: what a bundle of scrolls holds is drawn off this list. See the `randomness` skill.
func CantripOrder(cantrips map[string]CantripData) []string {
	return sortedKeys(cantrips)
}

// CantripFileOrder is every record id in the order data/cantrips.json writes them, for a review
// page. **Nothing that decides an outcome may walk this.**
func CantripFileOrder() []string {
	return fileOrder(cantripsJSON, "cantrips.json", func(c CantripData) string { return c.CantripRecord })
}
