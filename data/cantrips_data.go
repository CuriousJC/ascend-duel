package data

// The cantrips: **what the player may do to the duelist for one fight.**
//
// A rune alters a card; a potion alters the duelist for the rest of the run. A cantrip alters the
// duelist for the fight it is cast in and no longer — it is carried into a duel in the consumables
// pane, cast between turns, and gone when the duel ends. That is the whole of what separates it
// from a potion, and it is why the two share the shape of their records: an `Effect` naming what
// moves and an `Amount` read against it.
//
// **The rules do not read this file.** A cantrip is carried by a run and cast onto the fighter
// standing in the room, so the parsing and the closed effect vocabulary live in `internal/session`,
// exactly as a potion's do. See internal/session/cantrip.go.

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

	// Effect is what the cantrip does to the fighter, from a closed vocabulary resolved by
	// `session.ParseCantripEffect`: `add-dmg` or `scale-life`.
	Effect string `json:"Effect"`

	// Amount is read against the effect: a flat figure for `add-dmg`, a percentage for
	// `scale-life` — so 200 is double.
	Amount int `json:"Amount"`

	// Text is what the cantrip says it does, in the tooltip. **Authored rather than computed**, the
	// rune's posture: the line is the record's claim and the effect is what the rules fire.
	Text string `json:"Text"`
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
