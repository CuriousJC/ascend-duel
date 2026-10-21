package data

// The card traits: **what a card can be besides its form, its element, its action and its upgrade.**
//
// A trait composes with everything else a card is and takes no upgrade's seat — see MECHANICS.md
// §Traits. A record names the trait and says what it does in two closed properties the rules read:
//
//   - `Counted` — false and the card is not one of the turn's five. Absent is true.
//   - `Landings` — how many times the card lands before any relic is read. Absent is once.
//
// **Read by `internal/combat` itself**, as `duelist_cards.json` is: both properties are rules. A
// record that changes nothing is refused at init, and so is a property this file does not have —
// a new one is a Go change plus the one place reading it.

import (
	_ "embed"
	"fmt"
)

//go:embed card_traits.json
var cardTraitsJSON []byte

// CardTraitData is one card trait as written in the file.
type CardTraitData struct {
	// CardTrait is the key — what a save and a scenario write. Kebab-case.
	CardTrait string `json:"CardTrait"`

	// Name is what the trait is called, and Text what it does, for the card's tooltip.
	Name string `json:"Name"`
	Text string `json:"Text"`

	// Family is the one field no rule reads — see the data skill. **There is no Art and no Draw**: a
	// trait is a property of a wearing, drawn on the relic or card that carries it, never a picture of
	// its own.
	Family string `json:"Family"`

	// Percent is the chance, out of 100, that a card dealt into an action bundle carries the trait.
	// Every card rolls every trait. Zero is never.
	Percent int `json:"Percent"`

	// Counted is whether the card counts against the turn's five. Nil is true.
	Counted *bool `json:"Counted"`

	// Landings is how many times the card lands before any relic is read. Zero is once.
	Landings int `json:"Landings"`
}

// LoadCardTraits parses the catalog **in file order**, which is the order a tooltip lists them and
// each bundled card rolls them. A percent outside 0..100 or a trait named twice panics.
func LoadCardTraits() []CardTraitData {
	list := parse[CardTraitData](cardTraitsJSON, "card_traits.json")
	mustBeUniquelyKeyed(list, "card_traits.json", func(r CardTraitData) string { return r.CardTrait })
	for _, r := range list {
		mustBeAChance("card_traits.json", r.CardTrait, r.Percent)
	}
	return list
}

// mustBeAChance panics on a percent that is not one.
func mustBeAChance(file, key string, pct int) {
	if pct < 0 || pct > 100 {
		panic(fmt.Sprintf("%s: %q at %d%%, which is not a chance", file, key, pct))
	}
}
