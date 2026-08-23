package pyramid

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
)

// Pyramid is one run's climb: the opponents, in the order they will be met.
//
// **A run holds one and it outlives every fight.** It is built once, from the loaded roster and
// the run's own seed, and walked by fight index — so a defeat and a retry meet the same
// opponent again rather than a different one.
//
// **The order is the whole 96-record roster standing in for a generator.** Nothing caps the
// fight index against the 8-floor tower, deliberately: playing past the end wraps rather than
// stopping, and the stats keep climbing. When a real generator arrives it replaces New and
// nothing else.
type Pyramid struct {
	order []string

	// bosses is one record per floor, the stairway protectors. Separate from `order` because
	// they are drawn from their own pool and stand only in the third room of a floor; see
	// bosses.go.
	bosses []string
}

// New builds a run's fight order from the loaded enemy records, shuffled within each floor band.
//
// It takes the source rather than reaching for one, so the caller owns which stream is being
// advanced, and it is a plain function of its arguments so a test can hand it a fake roster and
// a fixed seed.
func New(recs map[string]data.EnemyData, bosses map[string]data.BossData, rng *rand.Rand) *Pyramid {
	return &Pyramid{
		order:  shuffleWithinFloors(data.EnemyOrder(recs), recs, rng),
		bosses: pickBosses(bosses, rng),
	}
}

// EnemyAt is the record key of whoever stands in a given room, wrapping past the end of the
// roster. It returns the empty string only for an empty pyramid, which a loaded game cannot
// produce — data's loader panics on an empty roster long before this.
//
// **Every third room is the floor's stairway and answers from the boss pool instead**
// *(2026-08-23)*. The roster index skips those rooms rather than counting them, so a boss
// standing in one does not consume a creature: `fight - fight/FightsPerFloor` is how many
// ordinary rooms came before this one.
func (p *Pyramid) EnemyAt(fight int) string {
	if fight < 0 {
		fight = 0
	}
	if RoomOf(fight) == RoomStairway {
		if boss := p.BossAt(FloorOf(fight)); boss != "" {
			return boss
		}
	}
	if len(p.order) == 0 {
		return ""
	}
	return p.order[(fight-fight/FightsPerFloor)%len(p.order)]
}

// Rooms is how many opponents the order holds.
func (p *Pyramid) Rooms() int { return len(p.order) }

// FloorOf is which floor a fight index falls on, counting floors from one. The inverse of
// FirstFightOnFloor.
func FloorOf(fight int) int {
	if fight < 0 {
		return 1
	}
	return fight/FightsPerFloor + 1
}

// FightsPerFloor is how many fights a floor holds — outer room, inner room, stairway — and the
// third of them is its boss. See the tower section of MECHANICS.md.
//
// **It lives here rather than in `internal/screens`** because the combat screen names the room
// under the duelist card and the ascent curve counts rooms off the same number, and anything
// headless that maps a floor band onto a fight index has to agree with the screen about how deep
// a floor is without being able to import it.
const FightsPerFloor = 3

// FirstFightOnFloor is the fight index of a floor's outer room, counting floors from one.
func FirstFightOnFloor(floor int) int {
	if floor < 1 {
		return 0
	}
	return (floor - 1) * FightsPerFloor
}

// Room is which of a floor's rooms a fight index is: outer, inner, or the stairway that is the
// floor's boss.
//
// **It is derived, never stored**, exactly as FloorOf is: the fight counter already says where the
// run is, and a second field saying the same thing is a second field to keep in step.
type Room int

const (
	// RoomOuter is the first room of a floor, RoomInner the second, RoomStairway the third — and
	// the third is the boss. Ordinals are positions in a floor, so this is not append-only in the
	// way an ID enum is; it is arithmetic on FightsPerFloor.
	RoomOuter Room = iota
	RoomInner
	RoomStairway
)

// RoomOf is which room of its floor a fight index falls in.
func RoomOf(fight int) Room {
	if fight < 0 {
		return RoomOuter
	}
	return Room(fight % FightsPerFloor)
}
