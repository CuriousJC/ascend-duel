package data

// The playable duelists: who the player can be, and which deck they play from.
//
// **A duelist and a deck go together, and the `Deck` field is that link.** The plan is to offer
// different duelists as different decks, so the deck's back is how you tell at a glance whose
// cards are on the table. The deck itself — its back and its discards — is `data/decks.json`; a
// duelist naming one the catalog does not hold is refused at load.

import (
	_ "embed"
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

	// Deck names the record in `data/decks.json` this duelist plays from: its back, and how many
	// discards a round allows.
	Deck string `json:"Deck"`
	// **Three stats, and every one of them is the number it sounds like** *(2026-08-16)*. Speed
	// and Constitution were conversions into the action-point budget and into life; they went the
	// day after Strength went, and for the same reason. See combat.Duelist.
	//
	// DMG is what one Bash deals in this duelist's hands. It was `Strength` until
	// 2026-08-16, when the stat and the figure it converted into turned out to be one number
	// wearing two names — see combat.Duelist.DMG.
	DMG     int `json:"DMG"`
	Actions int `json:"Actions"`
	HP      int `json:"HP"`
}

// LoadDuelists parses the embedded duelist list into a map keyed by DuelistRecord, refusing a
// duelist whose deck the catalog does not hold.
//
// **Refused rather than defaulted**, because the deck carries a rule — the discards — and a duelist
// quietly playing from some other deck would be a balance change nobody made.
func LoadDuelists() map[string]DuelistData {
	duelists := keyed(duelistsJSON, "duelists.json", func(d DuelistData) string { return d.DuelistRecord })
	decks := LoadDecks()
	for _, k := range sortedKeys(duelists) {
		if _, ok := decks[duelists[k].Deck]; !ok {
			panic("duelists.json: " + k + " plays from deck " + `"` + duelists[k].Deck + `"` + ", which decks.json does not hold")
		}
	}
	return duelists
}
