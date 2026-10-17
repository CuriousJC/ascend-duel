package data

// The relic traits: **every property a wearing can carry, in a grammar, and how often the shelf sells
// a relic carrying it.**
//
// A trait is a property of the wearing rather than of the relic — see MECHANICS.md §Traits. A record
// says what it does in three ways, any of which may be absent:
//
//   - `Slot` — false and the wearing takes no relic slot (weightless). Absent is true.
//   - `Lasts` — "fight" and the wearing lasts the fight it was put on in (ephemeral). Absent is the run.
//   - `Rules` — rules in the relic grammar, worn after the relic's own and firing at the same moments
//     (armored's turn-start shield). They never grow with the relic's accumulator.
//
// `Percent` is the chance a shelf seat's relic is sold carrying it; every seat rolls every trait.
//
// **Parsed in `internal/session`**, as `relics.json` is: the rules are the relic grammar and the
// names are the rules' vocabulary, and this package may not import `internal/combat`. A trait that
// lasts a fight may not be sold — a bought relic gone after one fight is vitae spent on nothing.

import (
	_ "embed"
	"fmt"
)

//go:embed relic_traits.json
var relicTraitsJSON []byte

// RelicTraitData is one relic trait as written in the file.
type RelicTraitData struct {
	// RelicTrait is the key — what a save and a scenario write. Kebab-case.
	RelicTrait string `json:"RelicTrait"`

	// Name is what the trait is called, and the line a worn relic's tooltip ends on. Text is what
	// it does, in a line, for the Compendium and the review sheet.
	Name string `json:"Name"`
	Text string `json:"Text"`

	// Family, Art and Draw are the three fields no rule reads — see the data skill. Art is an
	// assets.LoadImageData key, empty for the placeholder (ArtKey); Draw is the subject paragraph an
	// art generator is given.
	Family string `json:"Family"`
	Art    string `json:"Art"`
	Draw   string `json:"Draw"`

	// Percent is the chance, out of 100, that a shelf seat's relic is sold carrying the trait.
	// Zero is never.
	Percent int `json:"Percent"`

	// Slot is whether the wearing takes a relic slot. Nil is true.
	Slot *bool `json:"Slot"`

	// Lasts is how long the wearing lasts: "fight", or empty for as long as the run wears it.
	Lasts string `json:"Lasts"`

	// Rules are worn with the relic, in the relic grammar. See RelicRuleData.
	Rules []RelicRuleData `json:"Rules"`
}

// LastsFight reports whether the record says the wearing lasts one fight.
func (r RelicTraitData) LastsFight() bool { return r.Lasts == "fight" }

// LoadRelicTraits parses the catalog **in file order**, which is the order a tooltip lists them and
// each shelf seat rolls them. A percent outside 0..100, a `Lasts` that is not "fight" or empty, and
// a trait named twice panic.
func LoadRelicTraits() []RelicTraitData {
	list := parse[RelicTraitData](relicTraitsJSON, "relic_traits.json")
	mustBeUniquelyKeyed(list, "relic_traits.json", func(r RelicTraitData) string { return r.RelicTrait })
	for _, r := range list {
		mustBeAChance("relic_traits.json", r.RelicTrait, r.Percent)
		if r.Lasts != "" && r.Lasts != "fight" {
			panic(fmt.Sprintf("relic_traits.json: %q lasts %q, and a wearing lasts a \"fight\" or the run", r.RelicTrait, r.Lasts))
		}
	}
	return list
}

// ArtKey is the picture this trait draws: its own if it has one, the placeholder otherwise.
func (r RelicTraitData) ArtKey() string {
	if r.Art == "" {
		return PlaceholderArt
	}
	return r.Art
}
