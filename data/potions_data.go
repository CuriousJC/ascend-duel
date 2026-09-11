package data

// The potions: **what the player may do to the duelist themself.**
//
// A relic is worn and comes off; a stone raises a rung; a worm eats a card. A potion is the one
// thing in the shop that changes the *body* — the life it is carrying, the ceiling that life sits
// under, or the damage it hits for — and it is drunk on the spot, so nothing is carried and nothing
// is held. That is what keeps it out of the consumables pane the parasites live in.
//
// **Three fields carry the whole mechanic**: an `Effect` naming which of the three it moves, an
// `Amount` read against it, and a `Price`. There is no `Target`, because a potion has only ever one
// — the duelist drinking it.
//
// **The rules do not read this file.** A potion is bought by a run and applied to the run's
// fighter, so the parsing and the closed effect vocabulary live in `internal/session`, exactly as a
// worm's target does. See internal/session/potion.go.

import (
	_ "embed"
	"encoding/json"
)

//go:embed potions.json
var potionsJSON []byte

// PotionData is one potion as written in the file.
type PotionData struct {
	// PotionRecord is the key, and what anything naming a potion stores. Kebab-case, like a relic's,
	// a worm's and a stone's.
	PotionRecord string `json:"PotionRecord"`

	// Name is what is written across the top of the card — a vessel rather than the effect, so the
	// name and the text are not the same sentence twice.
	Name string `json:"Name"`

	// Effect is which of the duelist's three figures this moves, by name. **A closed vocabulary,
	// checked in `internal/session`** — `heal`, `dmg`, `life` — and a word this build has not got
	// fails the launch rather than producing a card that takes vitae and does nothing.
	Effect string `json:"Effect"`

	// Amount is read against the Effect: life restored, DMG added, ceiling raised. It is a field
	// here and not a constant in Go for the reason a stone's tenth is the other way round — three
	// potions moving three different figures have nothing one number could mean for all of them.
	Amount int `json:"Amount"`

	// Price is what the shop charges. **Written per record rather than derived from a tier**,
	// unlike a relic: there is no rarity here and no shelf draw to weight, so a tier would be a
	// pricing mechanism with one member per band.
	Price int `json:"Price"`

	// Text is what the card says it does. A `\n` is an authored line break, honoured by
	// `cards.WrapText`, for the reason every stone carries one: three cards differing only in a
	// figure would otherwise read as three layouts of one card.
	Text string `json:"Text"`
}

// LoadPotions parses the catalogue **as a slice, in file order**.
//
// It is the one shape `LoadDuelistCards` uses and for the same reason: file order is shelf order.
// The whole catalogue stands on the shelf every visit — nothing is drawn, weighted or shuffled —
// so there is no map to iterate and no sorted walk to protect, and the order the three vessels
// stand in is a decision made by editing the file.
func LoadPotions() []PotionData {
	var list []PotionData
	if err := json.Unmarshal(potionsJSON, &list); err != nil {
		panic("Failed to unmarshal potions.json: " + err.Error())
	}
	return list
}
