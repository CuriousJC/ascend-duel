package data

// The worms: **what the player may do to their deck between fights.**
//
// It is the card language's shape applied to a different subject. A card record says what a card
// *does* when it is played; a worm record says what it *changes* about a card that already
// exists — a target and, where the target needs one, a new value.
//
// **What is deliberately not here.** Which worms are offered, how many, and what they cost are
// not fields on a record: the offer is two drawn at random and nothing is bought yet. Whichever
// of those becomes real is a field beside Target when it does, and not before — the mistake
// `CostTier` was, which declared a rules vocabulary in JSON ahead of the rules, is the one to
// keep not making.
//
// **The rules do not read this file**, and cannot: a worm is applied to the *run's* deck, which
// is `internal/session`. That is the same test every file here answers — who consumes it — and
// it is why the parsing and the validation live over there rather than in `internal/combat`.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed worms.json
var wormsJSON []byte

// WormData is one alteration the player can be offered.
type WormData struct {
	// WormRecord is the key, and what anything holding a worm stores. Kebab-case, like a ring's.
	WormRecord string `json:"WormRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Target is which aspect of a card this worm changes. A closed vocabulary, resolved by
	// `session.ParseWormTarget`: `element`, `remove`, `duplicate`.
	//
	// **Closing it is the point**, exactly as with a card's verb. The set is short because
	// `combat.Card` is a concept plus an element and the element is the only per-instance field —
	// a worm that changed a card's *cost* would be changing the concept, and so every copy of
	// that card in the deck. Making cost per-card is a field on `combat.Card` and a change at
	// every `Cost()` call site, which is a price worth charging deliberately rather than
	// discovering because a JSON file asked for it.
	Target string `json:"Target"`

	// Value is the new value, read against the target. An element name for `element`; empty for
	// the targets that need none, and refused if one is supplied anyway.
	Value string `json:"Value,omitempty"`

	// Text is what the card says it does, in the same clipped register the action cards use —
	// the column is about a dozen characters wide.
	Text string `json:"Text"`
}

// LoadWorms parses the catalogue into a map keyed by WormRecord.
func LoadWorms() map[string]WormData {
	var list []WormData
	if err := json.Unmarshal(wormsJSON, &list); err != nil {
		panic("Failed to unmarshal worms.json: " + err.Error())
	}

	out := make(map[string]WormData, len(list))
	for _, w := range list {
		out[w.WormRecord] = w
	}
	return out
}

// WormOrder is every record, sorted by key.
//
// **Sorted because LoadWorms returns a map and Go randomises that order**, and this one decides
// an outcome rather than a layout: the offer is a shuffle of this list, so an unsorted walk would
// make which worms you are offered depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func WormOrder(worms map[string]WormData) []string {
	names := make([]string, 0, len(worms))
	for n := range worms {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
