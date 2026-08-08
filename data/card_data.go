package data

// The deck list as data. Which concepts exist, which elements each ships in, and how many
// copies of each card the starting deck holds.
//
// **What is deliberately not here: cost, category and damage.** Those are rules, they live in
// `internal/combat`, and the dependency direction forbids that package importing this one — so
// a cost written here could only ever be a second copy of a number defined elsewhere. What this
// file carries instead is `CostTier`, checked against the rules rather than trusted: see
// CheckCostTiers. That gives a readable grid in a file the designer can open without giving the
// grid the authority to disagree with the engine.
//
// Cost is also about to stop being a property of the card at all. Ring discounts make it a
// property of the card/element *pairing* — see MECHANICS.md — at which point a flat cost column
// here would have been a number that was wrong rather than merely duplicated.

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed cards.json
var cardsJSON []byte

// CardData is one concept: its name, which phase it resolves in, its cost tier, the elements it
// ships in, and how many copies of each of those the starting deck holds.
//
// **A concept ships as one card per element**, which is the rule for adding concepts rather than
// a description of this particular deck. Twelve concepts x five elements is the 60-card deck.
type CardData struct {
	// Concept is the card's name, and must match a combat.ActionKind's String(). The loader in
	// screens resolves it with combat.ParseAction; an unknown name is a load failure rather
	// than a silently missing card, because a deck quietly short a concept is a balance change
	// nobody made on purpose.
	Concept string `json:"Concept"`

	// Category is which phase this concept resolves in — prepare, attack or defend. Recorded
	// for readability and checked against the rules, never read as authority. See CheckCostTiers.
	Category string `json:"Category"`

	// CostTier is the concept's action-point cost, 1 to 4. It is documentation with a check:
	// the grid in MECHANICS.md is three categories by these four tiers, and being able to read
	// the grid off this file is the reason the field exists at all.
	CostTier int `json:"CostTier"`

	// Elements is which elements this concept ships in. Element names match the screen's
	// `element` type; `basic` is the absence of an element rather than a fifth colour.
	//
	// Every concept currently lists all five. It is a per-concept list rather than a global
	// constant so that a concept which is deliberately elementless — or one that only exists in
	// two colours — can be expressed without a special case in the loader.
	Elements []string `json:"Elements"`

	// Copies is how many of each element's card the starting deck holds. One today for every
	// concept, so the deck is exactly concepts x elements. It exists because duplicates are the
	// only way to tune deck composition without adding or removing a whole concept, and finding
	// that out later would mean changing this struct rather than a number.
	Copies int `json:"Copies"`
}

// LoadCards parses the embedded deck list, in file order. A slice rather than a map keyed by
// concept: the deck is built by walking this in order, and Go randomises map iteration — see
// the determinism rules in CLAUDE.md. File order is also grid order, so the JSON reads as the
// table in MECHANICS.md.
func LoadCards() []CardData {
	var cards []CardData
	if err := json.Unmarshal(cardsJSON, &cards); err != nil {
		panic("Failed to unmarshal our CardData: " + err.Error())
	}
	return cards
}

// CheckCostTiers verifies every declared tier and category against the rules, and is what keeps
// this file documentation rather than a second source of truth. The caller supplies the lookups
// because `data` cannot import `internal/combat` — the dependency direction runs the other way,
// and that is exactly what makes the check necessary in the first place.
//
// It returns every disagreement rather than the first, so a designer who has retuned several
// costs sees the whole list in one run instead of finding them one relaunch at a time.
func CheckCostTiers(cards []CardData, costOf func(concept string) (int, bool), categoryOf func(concept string) (string, bool)) []error {
	var problems []error

	for _, c := range cards {
		cost, ok := costOf(c.Concept)
		if !ok {
			problems = append(problems, fmt.Errorf("cards.json: concept %q is not an action the rules define", c.Concept))
			continue
		}
		if cost != c.CostTier {
			problems = append(problems, fmt.Errorf(
				"cards.json: %s declares CostTier %d but the rules cost it %d", c.Concept, c.CostTier, cost))
		}

		if cat, ok := categoryOf(c.Concept); ok && cat != c.Category {
			problems = append(problems, fmt.Errorf(
				"cards.json: %s declares category %q but the rules resolve it in %q", c.Concept, c.Category, cat))
		}

		if c.Copies < 1 {
			problems = append(problems, fmt.Errorf("cards.json: %s has Copies %d, which puts no card in the deck", c.Concept, c.Copies))
		}
		if len(c.Elements) == 0 {
			problems = append(problems, fmt.Errorf("cards.json: %s lists no elements, so it ships as no cards", c.Concept))
		}
	}

	return problems
}
