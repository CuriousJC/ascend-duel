package data

import (
	_ "embed"
	"encoding/json"
)

//go:embed combatants.json
var combatantsJSON []byte

type CombatantData struct {
	CombatantRecord string `json:"CombatantRecord"`
	SpriteSheet     string `json:"SpriteSheet"`
	SpriteRect      [4]int `json:"SpriteRect"`
	Strength        int    `json:"Strength"`
	Speed           int    `json:"Speed"`
	Constitution    int    `json:"Constitution"`
	// AvailableAffixes is the pool a floor can draw from to theme this enemy. The names are
	// element names on purpose — renamed from cold/hot/charged on 2026-08-05, because two
	// vocabularies for the same four things was a collision waiting to happen.
	//
	// An affix **transforms** the enemy's deck rather than adding to it: a brute has basic
	// attacks, a fire brute on a fire floor has all fire attacks. Nothing reads this yet;
	// enemies have no deck. See MECHANICS.md.
	//
	// `undying` was parked rather than deleted on 2026-08-05 — it is the one affix that was
	// never an element, and it wants revisiting once enemies have decks at all. `earth` is
	// absent because whether it can be a floor theme is still open.
	AvailableAffixes []string `json:"AvailableAffixes"`
}

// LoadCombatants parses the embedded JSON and returns a map keyed by CombatantRecord
func LoadCombatants() map[string]CombatantData {
	var tempCombatants []CombatantData

	// Unmarshal JSON into a temporary slice
	err := json.Unmarshal(combatantsJSON, &tempCombatants)
	if err != nil {
		panic("Failed to unmarshal our CombatantData")
	}

	// Convert slice to a map keyed by CombatantRecord
	combatantsMap := make(map[string]CombatantData)
	for _, c := range tempCombatants {
		combatantsMap[c.CombatantRecord] = c
	}

	return combatantsMap
}
