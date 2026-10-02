package data

// The upgrade art: **how an upgraded card says so, as a picture covering its face.**
//
// A card carries at most one upgrade, and the upgrade is drawn as authored art over the whole face
// inside the border ring, under the form mark, the cost ticks, the badge and the name. It is the
// playing cards' own pattern one layer down: a record per thing, an `Art` key, a `Draw` brief, and a
// picture committed at the card's own 200x280.
//
// **One record per upgrade, plus `default`**, which an upgrade without a record of its own draws
// and whose picture a record with no `Art` borrows. The key is the upgrade's name exactly as
// `systems.Upgrade.String` writes it; `internal/cards` refuses a key naming no upgrade at
// package init, because `data` may not import the vocabulary to check it here.
//
// **Nothing in the rules reads any of it**, the who-consumes-it test every file here answers: this
// is consulted by the card renderer and by `tools/upgradesheet`, and by nothing that resolves a
// round.

import (
	_ "embed"
)

//go:embed upgrade_art.json
var upgradeArtJSON []byte

// UpgradeArtData is the picture one upgrade lays over a card.
type UpgradeArtData struct {
	// UpgradeArtRecord is the key: an upgrade's name — `golden`, `wild` — or `default`.
	UpgradeArtRecord string `json:"UpgradeArtRecord"`

	// Art is the picture's filename stem under `assets/upgrade-art/`, which is the record's own key
	// once it is drawn. **Empty means undrawn**, and the record draws the default's picture — see
	// ArtKey.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given, beside the generic prompt in
	// `docs/art/upgrade_art_prompt.MD`. **Nothing in the game reads it.**
	Draw string `json:"Draw"`
}

// DefaultUpgradeArt is the record every upgrade without one of its own draws.
const DefaultUpgradeArt = "default"

// UpgradeArtPrefix is what the image map files this family under: `assets/upgrade-art/golden.png`
// is the key `upgrade-golden`. **The prefix is the asset map's, not the record's**, because the map
// is flat and an upgrade's name — `versatile`, `held-vitae` — is also an essence's or a rune's.
const UpgradeArtPrefix = "upgrade-"

// ArtKey is the image map key this record draws. **The fallback is here rather than in the
// renderer** for RelicData.ArtKey's reason: the review sheet draws upgrades too, and a fallback
// living in one drawing is a fallback the other does not have.
func (u UpgradeArtData) ArtKey() string {
	if u.Art == "" {
		return UpgradeArtPrefix + DefaultUpgradeArt
	}
	return UpgradeArtPrefix + u.Art
}

// LoadUpgradeArt parses the embedded catalog into a map keyed by UpgradeArtRecord, refusing a
// catalog with no default.
func LoadUpgradeArt() map[string]UpgradeArtData {
	art := keyed(upgradeArtJSON, "upgrade_art.json", func(u UpgradeArtData) string { return u.UpgradeArtRecord })
	if _, ok := art[DefaultUpgradeArt]; !ok {
		panic("upgrade_art.json: no " + DefaultUpgradeArt + " record")
	}
	return art
}

// UpgradeArtOrder is every record, sorted by key.
func UpgradeArtOrder(art map[string]UpgradeArtData) []string {
	return sortedKeys(art)
}
