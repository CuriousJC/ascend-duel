package data

// The potions: **what the player may do to the duelist themself.**
//
// A relic is worn and comes off; a stone raises a rung; an essence eats a card. A potion is the one
// thing in the shop that changes the *body* — the life it is carrying, the ceiling that life sits
// under, or the damage it hits for — and it is drunk on the spot, so nothing is carried and nothing
// is held. That is what keeps it out of the consumables pane the runes live in.
//
// **Three fields carry the whole mechanic**: an `Effect` naming which of the three it moves, an
// `Amount` read against it, and a `Price`. There is no `Target`, because a potion has only ever one
// — the duelist drinking it.
//
// **The rules do not read this file.** A potion is bought by a run and applied to the run's
// fighter, so the parsing and the closed effect vocabulary live in `internal/session`, exactly as a
// essence's target does. See internal/session/potion.go.

import (
	_ "embed"
	"encoding/json"
)

//go:embed potions.json
var potionsJSON []byte

// PotionData is one potion as written in the file.
type PotionData struct {
	// PotionRecord is the key, and what anything naming a potion stores. Kebab-case, like a relic's,
	// an essence's and a stone's.
	PotionRecord string `json:"PotionRecord"`

	// Name is what is written across the top of the card — a vessel rather than the effect, so the
	// name and the text are not the same sentence twice.
	Name string `json:"Name"`

	// Family is the motif the record was authored beside, read by a review sheet and by nothing
	// else. See the data skill on the three fields no catalog's rules read.
	Family string `json:"Family"`

	// Art is an assets.LoadImageData key. **Empty means the catalog's default face** — see ArtKey.
	Art string `json:"Art"`

	// Draw is the subject paragraph an art generator is given: what the vessel is, never what it
	// does. The generic prompt is docs/art/other_card_art_prompt.MD. Ignored by the engine.
	Draw string `json:"Draw"`

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
}

// LoadPotions parses the catalog **as a slice, in file order**.
//
// It is the one shape `LoadDuelistCards` uses and for the same reason: file order is shelf order.
// The whole catalog stands on the shelf every visit — nothing is drawn, weighted or shuffled —
// so there is no map to iterate and no sorted walk to protect, and the order the three vessels
// stand in is a decision made by editing the file.
func LoadPotions() []PotionData {
	var list []PotionData
	if err := json.Unmarshal(potionsJSON, &list); err != nil {
		panic("Failed to unmarshal potions.json: " + err.Error())
	}
	return list
}

// DefaultPotionArt is the face a record with no Art of its own draws.
//
// **It is the relic catalog's default, borrowed rather than copied** *(2026-09-14)*, which is the
// one place the potions depart from the essences and the runes: those two each took a copy of it
// the day they earned a family of their own, and the potions have no family yet because nobody has
// drawn one. It becomes `potion/default-potion.png` the day the first bottle is painted, and the
// change is this constant and one //go:embed line.
const DefaultPotionArt = DefaultRelicArt

// ArtKey is the picture this potion actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a potion is drawn by
// the shop and by a review sheet, and a fallback living in internal/screens is one the review tool
// does not have.
func (p PotionData) ArtKey() string {
	if p.Art == "" {
		return DefaultPotionArt
	}
	return p.Art
}
