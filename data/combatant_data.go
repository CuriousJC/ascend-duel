package data

import (
	_ "embed"
	"encoding/json"
)

//go:embed combatants.json
var combatantsJSON []byte

type CombatantData struct {
	CombatantRecord  string   `json:"CombatantRecord"`
	SpriteSheet      string   `json:"SpriteSheet"`
	SpriteRect       [4]int   `json:"SpriteRect"`
	Strength         int      `json:"Strength"`
	Speed            int      `json:"Speed"`
	Constitution     int      `json:"Constitution"`
	AvailableAffixes []string `json:"AvailableAffixes"`
}

var Combatants map[string]CombatantData

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
