package data

// The parasites: **what the player may do to their deck during a fight.**
//
// A worm is won and spent between rooms; a parasite is bought, carried, and spent between the
// turns of a duel. They overlap on purpose — both alter the run's deck and some of them do the
// same thing — and they are separate things because *when* you spend one is most of what it is.
// See MECHANICS.md.
//
// **A record is a target, a count and a value**, which is the worm record's shape plus the one
// field a worm never needed: how many cards this one eats. A worm was always aimed at exactly
// one card; a parasite may name two.
//
// **The rules read one field of this and only one.** A rider has to be consulted while a round is
// resolving, so its vocabulary is a closed Go enum in `internal/combat` — everything else here is
// applied to the *run's* deck, which is `internal/session`, and that is where a record is parsed
// and refused. Same who-consumes-it test every file in this package answers.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed parasites.json
var parasitesJSON []byte

// ParasiteData is one consumable the player can be sold.
type ParasiteData struct {
	// ParasiteRecord is the key, and what anything holding a parasite stores. Kebab-case, like a
	// worm's and a ring's.
	ParasiteRecord string `json:"ParasiteRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Target is what this parasite does, from a closed vocabulary resolved by
	// `session.ParseParasiteTarget`: `rider`, `remove`, `swap`, `vitae`.
	//
	// **Closing it is the point**, exactly as with a card's verb and a worm's target. A new target
	// is a Go change plus one place applying it, never something a JSON file can assert into
	// existence.
	Target string `json:"Target"`

	// Rider names which rule a `rider` parasite attaches — `combat.ParseRiderKind` resolves it.
	// Empty for every other target, and refused if one is supplied anyway.
	Rider string `json:"Rider,omitempty"`

	// Value is read against the target: a figure for `rider` and `vitae`, a concept key for
	// `swap`, and nothing at all for `remove`.
	Value string `json:"Value,omitempty"`

	// Count is how many cards of the run this parasite takes. **Zero is a real answer** and means
	// it takes none — a parasite that fills the purse is aimed at the run rather than at a card,
	// and the board piece asks for no target at all.
	Count int `json:"Count"`

	// Text is what the card says it does, in the clipped register the action cards use — the
	// column is about a dozen characters wide, and a `\n` is an authored line break honoured by
	// `cards.WrapText`.
	Text string `json:"Text"`
}

// LoadParasites parses the catalogue into a map keyed by ParasiteRecord.
func LoadParasites() map[string]ParasiteData {
	var list []ParasiteData
	if err := json.Unmarshal(parasitesJSON, &list); err != nil {
		panic("Failed to unmarshal parasites.json: " + err.Error())
	}

	out := make(map[string]ParasiteData, len(list))
	for _, p := range list {
		out[p.ParasiteRecord] = p
	}
	return out
}

// ParasiteOrder is every record, sorted by key.
//
// **Sorted because LoadParasites returns a map and Go randomises that order**, and this one
// decides an outcome rather than a layout: what a bucket holds is a shuffle of this list, so an
// unsorted walk would make a purchase depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func ParasiteOrder(parasites map[string]ParasiteData) []string {
	names := make([]string, 0, len(parasites))
	for n := range parasites {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
