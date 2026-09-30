package data

// The tonics: **what the player may do to the run's own rules.**
//
// A potion changes the duelist's body; a relic changes the cards; a tonic changes the container the
// run is played in — what the shop charges, what a card costs, how many rounds a fight lasts, how
// many cards a hand holds, how many relics may be worn. It is drunk on the spot, it lasts for the
// rest of the run, and it can only ever be drunk once.
//
// **The shop offers one per realm, not the catalog.** Which one is a walk over a seeded order of
// this file — see internal/session/tonic.go — so the file is a pool rather than a shelf.
//
// **The rules do not read this file.** A tonic is bought by a run, so the parsing and the closed
// effect vocabulary live in `internal/session`, exactly as a potion's do.

import (
	_ "embed"
)

//go:embed tonics.json
var tonicsJSON []byte

// TonicData is one tonic as written in the file.
type TonicData struct {
	// TonicRecord is the key, and what anything naming a tonic stores. Kebab-case, like a potion's.
	TonicRecord string `json:"TonicRecord"`

	// Name is the tooltip's title and the card's name.
	Name string `json:"Name"`

	// Family, Art and Draw are the three fields no rule reads, on the terms every other catalog
	// carries them. **Empty Art means DefaultTonicArt**; the generic prompt for Draw is
	// docs/art/other_card_art_prompt.MD.
	Family string `json:"Family"`
	Art    string `json:"Art"`
	Draw   string `json:"Draw"`

	// Effect is which rule this moves, by name. **A closed vocabulary, checked in
	// `internal/session`** — a word this build has not got fails the launch.
	Effect string `json:"Effect"`

	// Amount is read against the Effect: a percentage off every price, AP off every card, the
	// multiple on vitae earned, one more card in hand, one more discard, one more relic.
	Amount int `json:"Amount"`

	// Requires is the record key of a tonic the run must already have drunk before this one can be
	// offered. Empty means none.
	Requires string `json:"Requires"`

	// Price is what the shop charges, before any discount.
	Price int `json:"Price"`
}

// LoadTonics parses the catalog as a slice, in file order. **Nothing that decides an outcome may
// walk it in file order** — the offer shuffles sorted keys, so reordering the file rerolls nothing.
func LoadTonics() []TonicData {
	return parse[TonicData](tonicsJSON, "tonics.json")
}

// DefaultTonicArt is the face a record with no Art of its own draws: the potions' default, until a
// tonic is painted.
const DefaultTonicArt = DefaultPotionArt

// ArtKey is the picture this tonic actually draws: its own if it has one, the default otherwise.
func (t TonicData) ArtKey() string {
	if t.Art == "" {
		return DefaultTonicArt
	}
	return t.Art
}
