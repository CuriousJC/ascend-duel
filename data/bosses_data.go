package data

// The bosses: the thirty stairway protectors, one of whom stands in the third room of every
// floor.
//
// **A separate file from `enemies.json`, not a flag on it** *(2026-08-23)*. The two are drawn
// from different pools and placed by different rules — an enemy is shuffled inside a floor
// *band* and may turn up in either of a floor's first two rooms, while a boss belongs to one
// floor and only ever stands on its stairway. A `IsBoss` column on the roster would have made
// every selection read the flag before it could trust `ValidFloors`, and would have let a
// record be both.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed bosses.json
var bossesJSON []byte

// BossData is one stairway protector. The fields are the enemy record's, with one difference
// and one convention:
//
//   - `Floor` is a single number rather than a band, because a boss guards the stairway of
//     exactly one floor. A band would be asking which floor's boss this is twice.
//   - Its stats are pitched **above the enemies of its own floor** — roughly 1.6x the HP and
//     1.3x the DMG of the band it sits in — and its deck is bigger and dearer than a roster
//     deck: 60/120/250/300 against the roster's 50/100/200, and a 60% guard against 50%. The
//     ascent curve then scales it like anything else, so these are floor-one-relative numbers.
type BossData struct {
	// BossRecord is the key, and it is the character's bare first name — `Bayaz`, `Maw`. The
	// portrait file is that name lowercased with `-boss` appended, which is what keeps the
	// thirty out of the enemy portraits' key space; see the //go:embed in assets/embed.go.
	BossRecord string `json:"BossRecord"`

	// Name is what the boss is called on screen, and it is the bare first name — `Jerry`, `Thera`.
	//
	// **The title moved into its own field on 2026-08-24** *(owner's call)*. It was one string,
	// `Jerry the Toll-Taker`, and the card could not hold it: `EnemyStyle` centres a name on one
	// line without wrapping, so half the thirty rendered with a letter clipped off each end. The
	// card takes the name; the title is what a hover will say.
	Name string `json:"Name"`

	// Title is the rest of what the boss is called — `the Toll-Taker`, `of the Low Steps` — and
	// it is written to follow the name with a space between, so the two concatenate into the one
	// string this used to be.
	//
	// **Nothing in the game reads it yet.** It is here for the hover the input vocabulary already
	// has a widget for, and it is a separate field rather than something split off the name at
	// render time because there is no rule that finds the seam: `Bayaz, First of the Magi` breaks
	// at a comma and `The Maw` has no title at all.
	//
	// An empty title is a boss called only by its name, and is drawn as one rather than as a name
	// with a trailing space.
	Title string `json:"Title"`

	Portrait string `json:"Portrait"`

	DMG     int `json:"DMG"`
	Actions int `json:"Actions"`
	HP      int `json:"HP"`

	// Floor is which floor's stairway this boss guards, counting from one against the
	// 8-floor tower.
	Floor int `json:"Floor"`

	AvailableAffixes []string `json:"AvailableAffixes"`

	Cards []CardData `json:"Cards"`
}

// FullName is the name and the title as one string — `Jerry the Toll-Taker`, and just `The Maw`
// for a boss with no title. One function, so the two halves cannot be joined differently by the
// hover and by a review sheet.
func (b BossData) FullName() string {
	if b.Title == "" {
		return b.Name
	}
	return b.Name + " " + b.Title
}

// Enemy is this boss as an enemy record, so everything downstream — the deck builder, the
// combatant, the card — reads one shape and does not have to know which pool an opponent came
// from. The floor becomes a band of one, which is exactly what a boss's placement means.
func (b BossData) Enemy() EnemyData {
	return EnemyData{
		EnemyRecord:      b.BossRecord,
		Name:             b.Name,
		Portrait:         b.Portrait,
		DMG:              b.DMG,
		Actions:          b.Actions,
		HP:               b.HP,
		ValidFloors:      [2]int{b.Floor, b.Floor},
		AvailableAffixes: b.AvailableAffixes,
		Cards:            b.Cards,
	}
}

// LoadBosses parses the embedded list into a map keyed by BossRecord.
func LoadBosses() map[string]BossData {
	var list []BossData
	if err := json.Unmarshal(bossesJSON, &list); err != nil {
		panic("Failed to unmarshal our BossData: " + err.Error())
	}

	out := make(map[string]BossData, len(list))
	for _, b := range list {
		out[b.BossRecord] = b
	}
	return out
}

// BossOrder is every record, sorted lowest floor first and by name inside a floor.
//
// Sorted for the reason EnemyOrder is: the map Go hands back iterates in a different order
// every run, and the climb picks a floor's boss out of this.
func BossOrder(recs map[string]BossData) []string {
	names := make([]string, 0, len(recs))
	for n := range recs {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := recs[names[i]], recs[names[j]]
		if a.Floor != b.Floor {
			return a.Floor < b.Floor
		}
		return names[i] < names[j]
	})
	return names
}
