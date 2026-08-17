package data

// The enemy roster: who the player fights, how each one behaves, and what it looks like.
//
// **This was combatants.json and held the player too, until 2026-08-11.** See
// duelists_data.go for why the two split — the short version is that the fields stopped
// overlapping, so one struct meant every field was optional.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed enemies.json
var enemiesJSON []byte

// EnemyData is one opponent: its stats, how it fights, where in the tower it belongs, and
// its picture.
//
// **96 records since 2026-08-11**, one per portrait in the vendor packs, where there were
// four. Stats and floors are hand-assigned per creature and are meant to be edited — a Goblin
// and a Bio-Titan should not be the same numbers with a different picture.
type EnemyData struct {
	EnemyRecord string `json:"EnemyRecord"`

	// Name is what the enemy is called on screen, which is not the record key. A record is a
	// slug (`OgreWarlord`); a name has spaces and reads as a creature (`Ogre Warlord`).
	// Before this the screen printed the record, which is why the roster was called
	// Monster1..Tactician1 — names that described the *style* because there was nowhere else
	// to say one.
	Name string `json:"Name"`

	// Portrait is the assets key for the facing portrait drawn on this enemy's card.
	//
	// **The sprite sheet and its rect are gone** *(2026-08-11)*. The enemy is a card now, so
	// the west-facing idle frames had no drawing left that used them; keeping a second
	// picture per enemy would have meant cutting 96 more frames for something nothing reads.
	// `git` has them if animation comes back.
	Portrait string `json:"Portrait"`

	// **Three stats, and every one of them is the number it sounds like** *(2026-08-16)*. Speed
	// and Constitution were conversions — `4 + Speed/10` was the action-point budget and
	// `Constitution * 5` was life — so the roster was tuned in units nobody could act on. Speed was
	// the worse of the two: twenty-four distinct values across these records produced three
	// distinct budgets, so most of the hand-tuning in that column was never felt.
	//
	// DMG is what a 1x attack deals in this enemy's hands. Actions is its action-point budget, and
	// cards cost 1 to 3 of it. HP is life, and every enemy's was doubled when the fields changed —
	// the roster was written against a game where a turn landed several small blows, and one blow
	// per turn plus combo multipliers made every fight far shorter than it reads.
	DMG     int `json:"DMG"`
	Actions int `json:"Actions"`
	HP      int `json:"HP"`

	// ValidFloors is the inclusive range of tower floors this enemy may appear on, as
	// [lowest, highest]. The tower is 8 floors (MECHANICS.md), so 1 is the entrance and 8 is
	// the top.
	//
	// **It exists so the roster can be randomised without a Dragon on floor one.** Nothing
	// generates a floor yet — the combat screen walks every record in order — so today this
	// only sorts the roster. When floor generation lands it is the filter it draws from.
	//
	// A range rather than a list of floors: it is what almost every entry wants to say, it
	// reads at a glance in the JSON, and it is one pair of numbers to hand-tune rather than
	// up to eight. A creature that genuinely belongs on floors 2 and 7 and nowhere between
	// needs this to become a list, and that is a change to this field and AllowsFloor.
	ValidFloors [2]int `json:"ValidFloors"`
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

	// Cards is this enemy's own deck, and it is what replaced `PlanStyle` on 2026-08-16.
	//
	// **An enemy's personality is what it holds, not which branch a switch takes.** There were four
	// styles — brute, swarm, warden, tactician — named by a string here and implemented as four
	// planners; every enemy in the game drew from one shared list of `Attack` and `Heavy`. So a
	// Dragon and a Slime differed by a label, three of the four styles were unreachable (the warden
	// asked for a Defend by name and the shared list held none), and MECHANICS.md's affixes — which
	// *transform* a deck — had almost nothing to transform.
	//
	// An enemy holding six cheap copies of one card is a swarm. One holding four expensive ones is
	// a brute. One holding shields is a warden. The player learns a deck rather than a label.
	//
	// **`Copies` is the difficulty dial, and it is sharper than it looks.** A turn resolves one
	// blow and counted hands multiply it, so four copies of a 1 AP card in one turn is a Barrage at
	// 5x — the shape the old roster treated as weakest is now the strongest. See TODO.md.
	Cards []CardData `json:"Cards"`
}

// AllowsFloor reports whether this enemy may appear on a given floor.
//
// **A zero range means every floor**, so a record written without the field is fightable
// rather than unreachable. A record nothing can ever select is worse than one that turns up
// in the wrong place: the wrong place is visible, and the never is not.
func (e EnemyData) AllowsFloor(floor int) bool {
	if e.ValidFloors == [2]int{} {
		return true
	}
	return floor >= e.ValidFloors[0] && floor <= e.ValidFloors[1]
}

// LoadEnemies parses the embedded roster into a map keyed by EnemyRecord.
func LoadEnemies() map[string]EnemyData {
	var list []EnemyData
	if err := json.Unmarshal(enemiesJSON, &list); err != nil {
		panic("Failed to unmarshal our EnemyData: " + err.Error())
	}

	out := make(map[string]EnemyData, len(list))
	for _, e := range list {
		out[e.EnemyRecord] = e
	}
	return out
}

// EnemyOrder is every record, sorted shallowest floor first and by name inside a floor.
//
// **Sorted because LoadEnemies returns a map and Go randomises that order.** Anything walking
// the roster — the combat screen's fight order, the balance table — has to agree on it, and
// the determinism rules forbid a choice that depends on map iteration. Floor first means
// walking the list is walking up the tower, which is what makes it a usable stand-in until
// floors generate themselves.
func EnemyOrder(recs map[string]EnemyData) []string {
	names := make([]string, 0, len(recs))
	for n := range recs {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := recs[names[i]], recs[names[j]]
		if a.ValidFloors != b.ValidFloors {
			if a.ValidFloors[0] != b.ValidFloors[0] {
				return a.ValidFloors[0] < b.ValidFloors[0]
			}
			return a.ValidFloors[1] < b.ValidFloors[1]
		}
		return names[i] < names[j]
	})
	return names
}
