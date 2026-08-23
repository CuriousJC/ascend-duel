package data

// The card language: the one schema every card in the game is written in, the player's twelve
// and every enemy's own.
//
// **A card carries its own rules as of 2026-08-16.** Cost, damage, category and form used to be
// switch statements over a closed `combat.ActionKind` enum, and this file carried a `CostTier`
// column that was checked against them. That worked for twelve concepts. It cannot hold three or
// four bespoke cards for each of ninety-six enemies — roughly four hundred concepts, each wanting
// its own multiplier and its own name — so the card stopped being an enum value and became a
// record. This is the record.
//
// **The check that used to live here is gone with the duplication it guarded.** `CheckCostTiers`
// asserted a declared tier against the rules; there is nothing to assert now, because the file is
// the rules. What replaces it is validation at registration — see `combat.RegisterConcept`, which
// knows the verb vocabulary and rejects a record naming one it does not have.
//
// **A card never names a status, and that is load-bearing.** What an element *does* is decided by
// the source of that element on the card's owner — a ring for the player, an elemental affix for
// an enemy — and a ring may later decide which of several fire statuses a fire card applies. A
// card that named its own status would be deciding something that is not its to decide. So the
// schema carries a colour and a target and stops there.

import (
	_ "embed"
	"encoding/json"
)

//go:embed duelist_cards.json
var duelistCardsJSON []byte

// CardData is one concept, whole.
//
// **Two fields are absent on purpose.** There is no `Category` — attack-or-plan falls out of the
// verb, and carrying both would let a file say a card is an attack that banks points. And there is
// no flat damage figure, only a multiplier: what a card deals is its wielder's DMG scaled, which
// is what makes a card's face a function of the card alone.
type CardData struct {
	// Label is what the card face says, and — scoped by whoever registers it — the concept's
	// rules identity. An enemy's labels are scoped to that enemy, so forty creatures may all
	// have a `Bite` at forty different multipliers without colliding. See combat.ScopedKey.
	Label string `json:"Label"`

	// Verb is what the card does: attack, defend, bank or draw. A closed vocabulary, exactly like
	// the reward kinds in hands.json — adding a fifth is a Go change, and is meant to be.
	Verb string `json:"Verb"`

	// Amount is read against the verb, which is what lets one field carry four meanings without a
	// generic one nobody could state:
	//
	//   - attack: percent of the wielder's DMG, so 100 is 1x and 50 is the cheap rung.
	//   - defend: percent taken off the one blow it answers.
	//   - bank:   action points banked for the following round.
	//   - draw:   cards added to the following round's hand.
	Amount int `json:"Amount"`

	// Cost is action points out of the round's budget.
	Cost int `json:"Cost"`

	// Form is which group of cards this concept belongs to — stab, slash, crush or plan. Empty
	// means none, which is what every enemy card is: a form is the player's deck axis, the thing
	// a pair is counted on and the mark in the card's corner, and an enemy card claiming to be a
	// crush would be saying something untrue about a deck the player cannot build hands against.
	Form string `json:"Form"`

	// Elements is which colours this concept ships in. Empty means `basic` alone.
	//
	// **Every enemy card says basic today**, and the field exists anyway. An enemy's colour does
	// nothing until an elemental affix attunes it, and affixes are not built — colouring an enemy
	// card now would hand it a free status, which is exactly what the source rule forbids.
	Elements []string `json:"Elements"`

	// Copies is how many of each element's card the deck holds.
	//
	// **It is the only axis an enemy deck has.** Enemy cards are all one colour, so `Elements`
	// cannot produce a count for them and this carries the whole deck size: a four-concept slime
	// is a fourteen-card deck because its copies say 6, 4, 2, 2. Drop it and an enemy would draw
	// its entire deck every round and never have a decision.
	//
	// For the player it is the other way round — the nine attacks ship one per colour and the
	// three plans ship four of one colour, which is the same four cards reached along different
	// axes. Both are needed and neither substitutes for the other.
	Copies int `json:"Copies"`
}

// LoadDuelistCards parses the player's deck list, in file order. A slice rather than a map keyed
// by label: the deck is built by walking this in order, and Go randomises map iteration — see the
// determinism rules in CLAUDE.md. File order is also grid order, so the JSON reads as the table in
// MECHANICS.md.
func LoadDuelistCards() []CardData {
	var cards []CardData
	if err := json.Unmarshal(duelistCardsJSON, &cards); err != nil {
		panic("Failed to unmarshal duelist_cards.json: " + err.Error())
	}
	return cards
}
