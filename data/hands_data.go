package data

// The hand catalogue as data: which hands exist, and what each is worth. `internal/combat` reads
// this and turns it into rules.
//
// **This file is the one place `data` is read by the rules rather than by a layer above them**,
// and it is why `internal/combat` imports this package at all. Every other list here is consumed
// by `screens`, `decks` or `entities`; a hand is consumed by the resolver itself. That is the
// line worth holding if a seventh list is proposed — ask who reads it, not whether it is data.
//
// **One list, and a hand is only ever a damage multiplier** *(2026-08-17)*. A hand counts cards that
// agree on one axis — the same concept, the same form or the same element — and pays a multiplier;
// that is the whole vocabulary. Statuses come from elements and rings, and nothing a hand forms
// buys anything besides damage.
//
// **What is deliberately not here: any meaning.** The strings below are names, and joining them to
// the rules is `internal/combat`'s job, exactly as it is for `CardData.Concept`. A malformed
// catalogue is refused *there*, on the same principle CheckCostTiers already works on — data holds
// the shape, the rules hold the truth.

import (
	_ "embed"
	"encoding/json"
)

//go:embed hands.json
var handsJSON []byte

// handFile is the shape of hands.json.
type handFile struct {
	Hands []HandData `json:"hands"`
}

// HandData is one rung of the of-a-kind ladder as written in the file.
//
// **A hand names what it counts copies *of*** *(2026-08-19)*. `match` is one of three axes —
// `concept`, `form`, `element` — so the same rung exists three times over and each can be priced
// on its own rarity. There is still no category filter: the matcher only ever sees attack cards
// aimed at the opponent, so a hand saying which categories it counts would be repeating a rule it
// cannot change.
type HandData struct {
	// Key is a stable slug — `pair`, `four-of-a-kind`. It is how a caller asks for a rung without
	// knowing its number, and it is deliberately independent of Name so a hand can be renamed
	// without breaking lookups.
	Key string `json:"key"`

	// ID is the hand's number. Discovery will persist on the profile, so this must not move under
	// a player who has already found a hand.
	ID int `json:"id"`

	// Name is what the hand is called on screen.
	Name string `json:"name"`

	// Match is the axis the hand counts on: `concept`, `form` or `element`. It is required — a
	// missing one is refused rather than defaulted, because an entry that landed on the wrong axis
	// by omission would be a balance change nobody made.
	Match string `json:"match"`

	// Groups is how many cards of each distinct value **on the hand's own axis** the hand wants.
	// `[3,2]` is a full house and can never be satisfied by five cards sharing one value.
	Groups []int `json:"groups"`

	// Multiplier is the hand's damage multiplier, in **percent** — 150 is 1.5x. An integer because
	// `internal/combat` is integer arithmetic throughout, and a float here would be the one number
	// in the game that rounds differently from the rest.
	Multiplier int `json:"multiplier"`
}

// LoadHands parses the catalogue, in file order.
//
// **File order is load-bearing rather than cosmetic** — a slice, never a map, per the
// determinism rules in CLAUDE.md.
//
// It panics on a file it cannot read, like the other loaders here. The rules do the rest of the
// checking: `internal/combat` refuses a catalogue describing a shape it cannot match.
func LoadHands() []HandData {
	var file handFile
	if err := json.Unmarshal(handsJSON, &file); err != nil {
		panic("Failed to unmarshal hands.json: " + err.Error())
	}
	if len(file.Hands) == 0 {
		panic("hands.json declares no hands")
	}
	return file.Hands
}
