package data

// The deck list as data. Which concepts exist, which elements each ships in, and how many
// copies of each card the starting deck holds.
//
// **What is deliberately not here: cost, category and damage.** Those are rules, they live in
// `internal/combat`, and a cost written here could only ever be a second copy of a number
// defined elsewhere. What this file carries instead is `CostTier`, checked against the rules
// rather than trusted: see CheckCostTiers. That gives a readable grid in a file the designer can
// open without giving the grid the authority to disagree with the engine.
//
// **CheckCostTiers takes its lookups as parameters even though `internal/combat` does import
// this package** — see `combos_data.go`. It could call `ActionKind.Cost()` directly now. It does
// not, because the check belongs to whoever is *loading a deck*, and the two deck loaders live in
// `screens` and `decks`; handing them the lookup keeps this file free of any opinion about what a
// concept means, which is the property that makes it documentation rather than a second engine.
//
// Cost is also about to stop being a property of the card at all. Ring discounts make it a
// property of the card/element *pairing* — see MECHANICS.md — at which point a flat cost column
// here would have been a number that was wrong rather than merely duplicated.

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// **Two lists, one shape** *(2026-08-11)*. `duelist_cards.json` is the player's starting
// deck and `enemy_cards.json` is what an opponent draws from. They are separate files
// because they want tuning against each other rather than together — the player's deck is
// twelve concepts in five colours and an enemy's is a handful of basics — and one file with
// a "side" column would have made every future edit a filter.
//
// **What a separate file cannot currently change is a number.** Cost, category and damage
// come from `internal/combat` for both sides, so an enemy Strike costs and hits exactly what
// yours does. Making those side-aware is a rules change rather than a data one; the shape
// below is ready for it, in that a per-side cost would be a field here checked the same way
// CostTier already is.
//
//go:embed duelist_cards.json
var duelistCardsJSON []byte

//go:embed enemy_cards.json
var enemyCardsJSON []byte

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

// LoadDuelistCards parses the player's deck list, in file order. A slice rather than a map
// keyed by concept: the deck is built by walking this in order, and Go randomises map
// iteration — see the determinism rules in CLAUDE.md. File order is also grid order, so the
// JSON reads as the table in MECHANICS.md.
func LoadDuelistCards() []CardData {
	return loadCards(duelistCardsJSON, "duelist_cards.json")
}

// LoadEnemyCards parses what an opponent draws from.
//
// **It is one list for every enemy**, not one per record. Which of them a Warden actually
// plays is decided by its style rather than by its deck, which is the smallest thing that
// makes an enemy deck real; per-enemy lists are the obvious next step and want a field on
// EnemyData rather than a change here.
//
// Every card is `basic`. Enemy cards are never drawn on screen, so an element would be a
// colour nobody sees — and MECHANICS.md has affixes *transforming* an enemy deck into an
// element, which is a thing to do to a basic deck rather than a property to author into one.
func LoadEnemyCards() []CardData {
	return loadCards(enemyCardsJSON, "enemy_cards.json")
}

func loadCards(raw []byte, name string) []CardData {
	var cards []CardData
	if err := json.Unmarshal(raw, &cards); err != nil {
		panic("Failed to unmarshal " + name + ": " + err.Error())
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
func CheckCostTiers(name string, cards []CardData, costOf func(concept string) (int, bool), categoryOf func(concept string) (string, bool)) []error {
	var problems []error

	for _, c := range cards {
		cost, ok := costOf(c.Concept)
		if !ok {
			problems = append(problems, fmt.Errorf("%s: concept %q is not an action the rules define", name, c.Concept))
			continue
		}
		if cost != c.CostTier {
			problems = append(problems, fmt.Errorf(
				"%s: %s declares CostTier %d but the rules cost it %d", name, c.Concept, c.CostTier, cost))
		}

		if cat, ok := categoryOf(c.Concept); ok && cat != c.Category {
			problems = append(problems, fmt.Errorf(
				"%s: %s declares category %q but the rules resolve it in %q", name, c.Concept, c.Category, cat))
		}

		if c.Copies < 1 {
			problems = append(problems, fmt.Errorf("%s: %s has Copies %d, which puts no card in the deck", name, c.Concept, c.Copies))
		}
		if len(c.Elements) == 0 {
			problems = append(problems, fmt.Errorf("%s: %s lists no elements, so it ships as no cards", name, c.Concept))
		}
	}

	return problems
}
