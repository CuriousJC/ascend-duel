package data

// The stones: **what the player may do to the hand ladder.**
//
// A worm record says what a stone's sibling changes about a *card*; a stone record says what it
// changes about a *rung*. One stone raises one hand's multiplier by a tenth of the figure
// `hands.json` writes down, for the rest of the run.
//
// **A stone names a hand by key, and by nothing else.** The keys are `hands.json`'s own, and the
// arithmetic — how much a tenth is, and how two stones on one rung stack — is
// `internal/combat`'s. This file holds the shape; the rules hold the truth, which is the same
// division every list here is under.
//
// **What is deliberately not here: the bump.** A record could carry `"Percent": 10` and it would
// be the `CostTier` mistake again — a rules vocabulary declared in JSON ahead of the rules. The
// tenth is one decision about the whole mechanic and it lives in `internal/combat/stone.go`. It
// becomes a field here the day two stones want to be worth different amounts, and not before.
//
// **The rules do not read this file.** A stone is bought by the run and applied to the run's
// fighter, so the parsing and the validation live in `internal/session`, exactly as a worm's do.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed stones.json
var stonesJSON []byte

// StoneData is one rung-raiser as written in the file.
type StoneData struct {
	// StoneRecord is the key, and what anything holding a stone stores. Kebab-case, like a ring's
	// and a worm's.
	StoneRecord string `json:"StoneRecord"`

	// Name is what is written across the top of the card. **A mineral rather than the rung**, so
	// the name and the text are not the same sentence twice: the text says which hand it raises.
	Name string `json:"Name"`

	// Hand is the rung this stone raises, by `hands.json` key. Resolved against the catalogue the
	// rules loaded — a stone naming a hand this build has not got fails the launch rather than
	// landing on whichever rung happens to sit first.
	Hand string `json:"Hand"`

	// Text is what the card says it does, in the same clipped register the worms use. A `\n` is an
	// authored line break, honoured by `cards.WrapText` — every stone carries one, because the
	// nineteen differ only in the rung they name and left to the measurer they would read as
	// nineteen layouts of one card.
	//
	// **The figure is not in it.** What a stone is worth depends on the rung's own multiplier, so
	// writing `+11` here would be a number that goes stale the moment `hands.json` is tuned. The
	// card face computes it; see `internal/screens/card_art.go`.
	Text string `json:"Text"`
}

// LoadStones parses the catalogue into a map keyed by StoneRecord.
func LoadStones() map[string]StoneData {
	var list []StoneData
	if err := json.Unmarshal(stonesJSON, &list); err != nil {
		panic("Failed to unmarshal stones.json: " + err.Error())
	}

	out := make(map[string]StoneData, len(list))
	for _, s := range list {
		out[s.StoneRecord] = s
	}
	return out
}

// StoneOrder is every record, sorted by key.
//
// **Sorted for the reason WormOrder is**: LoadStones returns a map, Go randomises map order, and
// the bag of rocks is a shuffle of this list — so an unsorted walk would make which stones a run
// is offered depend on map iteration and take the run's reproducibility with it. See the
// `randomness` skill.
func StoneOrder(stones map[string]StoneData) []string {
	names := make([]string, 0, len(stones))
	for n := range stones {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
