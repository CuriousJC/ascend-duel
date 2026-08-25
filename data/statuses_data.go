package data

// The statuses: what a landed attack can leave standing on a duelist.
//
// **A status is its own thing as of 2026-08-17, and no longer the same object as an element.**
// Fire did not burn because it was fire — it burned because the rules held four constants indexed
// by colour, which made "a second fire status" inexpressible and left every ring with the same
// one thing to sell. The colour is now only a *predicate* a ring matches on; what happens is a
// record in this file, named by a ring's `apply-status` effect. See the `rings` skill.
//
// **`internal/combat` reads this file directly**, which is the third such file after
// `hands.json` and `duelist_cards.json` and passes the same who-consumes-it test: how much a
// status is worth, how long it lasts and which of four things it does are rules by definition. The
// engine cannot resolve a round without them, and its own tests could not run if a screen had to
// hand them over.
//
// **`Badge` is the exception and the engine ignores it**, exactly as it ignores a ring's `Art`. A
// badge belongs to the status rather than to whatever applied it — a status arriving by an affix
// or a boss rule has to draw the same picture — so the key lives beside the rest of the record and
// `internal/screens` is the layer that resolves it.

import (
	_ "embed"
	"encoding/json"
)

//go:embed statuses.json
var statusesJSON []byte

// StatusData is one status, whole.
type StatusData struct {
	// StatusRecord is the rules identity, and what a ring's `apply-status` effect names.
	// Kebab-case, like every other record key in `data/`.
	//
	// **It is what a save file would write**, because a registered status is named by an
	// append-only ID and an ordinal is exactly what CLAUDE.md forbids serializing.
	StatusRecord string `json:"StatusRecord"`

	// Name is the word the Resolution feed and a long press use — upper case, because it is a
	// state being shouted rather than a sentence.
	Name string `json:"Name"`

	// Badge is the assets.LoadImageData key for the picture drawn on the carrier's card. An
	// unknown key draws `defaulteffect_png`, so a status nobody has made art for shows a shape
	// nobody has learned rather than nothing at all.
	Badge string `json:"Badge"`

	// Effect is which of five closed kinds this status is: `damage-over-time`, `lose-actions`,
	// `miss-chance`, `damage-reduction` or `damage-amplification`.
	//
	// **A status is a file entry; a *kind* of status is a Go change.** The same posture the card
	// verbs and the ring effects take, and for the same reason: a vocabulary that could express
	// anything would be a scripting language, and the rules would stop being readable in one file.
	Effect string `json:"Effect"`

	// Amount is read against the effect kind, the way a card's is read against its verb:
	//
	//   - damage-over-time: percent of the *attacker's* DMG, ticked at the end of each round.
	//   - lose-actions:     cards off the front of every turn it lasts.
	//   - miss-chance:      percentage points of chance that an attack does not land.
	//   - damage-reduction: percentage points off the damage the carrier deals.
	//   - damage-amplification: percentage points *added* to the damage the carrier takes, blows and
	//     burn ticks alike. 100 is double. It is the only kind read off the duelist being hit.
	Amount int `json:"Amount"`

	// Rounds is how many round-ends a freshly applied status survives.
	//
	// **Two is the floor for a status that does anything**, and the reason is turn order: a status
	// applied during round N has to survive round N's ending to be felt in round N+1, so a
	// duration of 1 applied by whoever acts second would never bite anything at all.
	Rounds int `json:"Rounds"`

	// Text is one line saying what carrying this does, for the long press that does not exist yet.
	Text string `json:"Text"`
}

// LoadStatuses parses the embedded status list, in file order.
//
// **File order is registration order, and therefore ID order** — the same contract
// `LoadDuelistCards` has. A map would hand the registry a different set of IDs every launch, which
// is the determinism breach `RingOrder` and `EnemyOrder` exist to prevent from the other side.
func LoadStatuses() []StatusData {
	var list []StatusData
	if err := json.Unmarshal(statusesJSON, &list); err != nil {
		panic("Failed to unmarshal our StatusData: " + err.Error())
	}
	return list
}
