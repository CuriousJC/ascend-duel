package data

// The playable duelists: who the player can be, and what their cards look like from behind.
//
// **Split out of combatants.json on 2026-08-11.** One file held the player and the four
// enemies because they were the same struct, and they were the same struct because both are
// three stats and a sprite. They have stopped being: an enemy has a plan style, a portrait
// and an affix pool, and a duelist has a card back and — eventually — a deck. Keeping them
// in one record meant every field was optional and no field meant anything.
//
// **A duelist and a card back go together, and that is the point of the field.** The plan is
// to offer different duelists as different *decks*, so the back is how you tell at a glance
// whose deck is on the table. It is a name rather than a picture: the mark is drawn in code
// by internal/cards, which is what keeps interface art free of a provenance question.

import (
	_ "embed"
	"encoding/json"
)

//go:embed duelists.json
var duelistsJSON []byte

// DuelistData is one playable duelist.
//
// **It carries no sprite and no plan style.** The character block replaced the fighter's
// sprite on the combat screen, and a duelist is planned by the person holding the mouse —
// so both fields were permanently empty on the one record that used them.
type DuelistData struct {
	// DuelistRecord is the key, matching what a screen asks for. `Fighter1` is the one that
	// exists; it stays that name because the balance tool and the tests use it.
	DuelistRecord string `json:"DuelistRecord"`

	// Name is what the duelist is called on screen, which is not the record key — the same
	// separation the enemies have between a record and a roster name.
	Name string `json:"Name"`

	// CardBack names the mark drawn on the back of this duelist's cards: triangle, diamond
	// or chevron. Parsed by cards.ParseBackMark, which falls back rather than failing — a
	// deck whose backs are the wrong shape is a cosmetic bug, and refusing to start the game
	// over one would be worse.
	CardBack string `json:"CardBack"`

	Strength     int `json:"Strength"`
	Speed        int `json:"Speed"`
	Constitution int `json:"Constitution"`
}

// LoadDuelists parses the embedded duelist list into a map keyed by DuelistRecord.
func LoadDuelists() map[string]DuelistData {
	var list []DuelistData
	if err := json.Unmarshal(duelistsJSON, &list); err != nil {
		panic("Failed to unmarshal our DuelistData: " + err.Error())
	}

	out := make(map[string]DuelistData, len(list))
	for _, d := range list {
		out[d.DuelistRecord] = d
	}
	return out
}
