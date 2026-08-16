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

	// DMG is what one Attack deals in this enemy's hands — the same field the player's record
	// carries, renamed off `Strength` on 2026-08-16. See combat.Duelist.DMG.
	DMG          int `json:"DMG"`
	Speed        int `json:"Speed"`
	Constitution int `json:"Constitution"`

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

	// PlanStyle is how this combatant fights: brute, swarm, warden or tactician. It is a
	// string here and a combat.PlanStyle after hydration, so the roster is tunable without
	// touching Go — which is the whole reason enemy shape is data and not code.
	//
	// Empty or unrecognised falls back to brute, so a record predating this field, or one
	// with a typo, still produces a fightable enemy rather than one that stands still.
	//
	// **The deck arrived on 2026-08-11 and a style is now how a hand is *played*, not what
	// is played.** It used to synthesise actions from nothing — a brute produced Heavies
	// whether or not a Heavy existed anywhere. Every enemy now draws from
	// `enemy_cards.json` and the style chooses among what it was dealt, which is what makes
	// the affix plan below mean anything: transforming a deck does nothing to a planner that
	// never read one.
	//
	// The deck itself is still one shared list rather than a field here. Per-enemy decks are
	// the obvious next step and this is where that field would go.
	PlanStyle string `json:"PlanStyle"`
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
