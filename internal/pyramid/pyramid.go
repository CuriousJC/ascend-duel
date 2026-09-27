package pyramid

import (
	"math/rand"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
)

// Pyramid is one run's climb: every floor's theme and the creature standing in each of its rooms.
//
// **A run holds one and it outlives every fight.** It is built once, from the loaded motifs and
// the run's own seed, and walked by fight index — so a defeat and a retry meet the same opponent
// again rather than a different one.
//
// **A floor is a motif and an element.** The three rooms are three records of that motif, each
// dealt as that element, so a fire goblin floor is three goblins in fire and the player can plan
// against what they walked into.
type Pyramid struct {
	// choices is what each floor is offered as: one theme on floor one, where a run starts, and a
	// portal's worth on every floor above it.
	//
	// **Every one is rolled up front and every one is spent.** A motif offered on a floor is struck
	// from every floor above it whether or not the player walks through its portal, so what a floor
	// offers is a function of the run code alone and never of an earlier pick. Which offer was taken
	// is the run's — see internal/session — and this stays a pure function of the seed.
	choices [][]Floor
}

// Floor is one themed floor: which motif it is, which element its creatures are dealt as, and
// the record standing in each of its rooms.
type Floor struct {
	Motif   string
	Element string

	// Rooms is one record key per room, indexed by Room — outer, inner, portal.
	Rooms [FightsPerFloor]string
}

// New builds a run's climb from the loaded motifs, the tower's own shape, and a seeded source.
//
// It takes the source rather than reaching for one, so the caller owns which stream is being
// advanced, and it is a plain function of its arguments so a test can hand it a fake roster and a
// fixed seed. Never `rand.Shuffle` — the package-level one draws from a global shared with every
// other caller and would make a run unreproducible.
//
// **A motif is never offered twice in one run.** Each floor's offers are struck off before the next
// floor is rolled, and **a draw is only kept if the floors above can still be filled without it** —
// data.FillsSlots is asked after every pick, so a motif a later floor depends on is never spent on
// an earlier one that had alternatives. data.MustBeClimbable refuses, at load, a roster where no
// order of draws could work, so the panic below is unreachable on a loaded catalog.
func New(motifs map[string]data.MotifData, tower data.TowerData, rng *rand.Rand) *Pyramid {
	order := data.MotifOrder(motifs)
	used := map[string]bool{}

	p := &Pyramid{}
	for floor := 1; floor <= tower.Floors; floor++ {
		var free []string
		for _, key := range order {
			if !used[key] && motifs[key].AllowsFloor(floor) {
				free = append(free, key)
			}
		}

		// Shuffled rather than picked by index, so the offers are a sample of what is left rather
		// than the front of a sorted list.
		rng.Shuffle(len(free), func(i, j int) { free[i], free[j] = free[j], free[i] })

		want := data.OffersOn(floor)
		var taken []string
		for _, key := range free {
			if len(taken) == want {
				break
			}
			used[key] = true
			// What is still owed: the rest of this floor's offers, then every floor above.
			owed := data.ClimbSlots(floor+1, tower.Floors)
			for range want - len(taken) - 1 {
				owed = append(owed, floor)
			}
			if !data.FillsSlots(motifs, owed, used) {
				delete(used, key)
				continue
			}
			taken = append(taken, key)
		}
		if len(taken) < want {
			panic("pyramid: floor " + strconv.Itoa(floor) + " cannot be offered " + strconv.Itoa(want) + " motifs of its own")
		}

		set := make([]Floor, 0, want)
		for _, key := range taken {
			set = append(set, rollFloor(motifs[key], rng))
		}
		p.choices = append(p.choices, set)
	}
	return p
}

// rollFloor gives a motif an element and fills its three rooms.
//
// **Every element is reachable**, because data.LoadMotifs refuses a motif that cannot field all
// three rooms at all five — so this never has to ask whether the theme it rolled is buildable.
func rollFloor(m data.MotifData, rng *rand.Rand) Floor {
	f := Floor{
		Motif:   m.Motif,
		Element: data.AffinityElements[rng.Intn(len(data.AffinityElements))],
	}
	for i, tier := range data.TierOrder {
		can := m.Candidates(tier, f.Element)
		if len(can) == 0 {
			// Unreachable on a loaded catalog, and a name is better than a zero value if the
			// loader is ever loosened.
			panic("pyramid: " + m.Motif + " cannot field a " + f.Element + " " + tier)
		}
		f.Rooms[i] = can[rng.Intn(len(can))].Record
	}
	return f
}

// FloorAt is a floor's first offer, counting floors from one: the theme floor one is fought at, and
// what any floor is fought at until a pick says otherwise. The run's own answer, pick included, is
// session.Session's — see internal/session/climb.go.
//
// **Past the top of the tower it wraps**, rather than the run stopping: the ascent curve has no
// ceiling and neither does the climb. What wraps is which floors are met again, not how hard they
// are — see ScaleToFight, which keeps counting.
func (p *Pyramid) FloorAt(floor int) Floor {
	if len(p.choices) == 0 {
		return Floor{}
	}
	if floor < 1 {
		floor = 1
	}
	set := p.choices[(floor-1)%len(p.choices)]
	return set[0]
}

// ChoicesAt is every theme a floor is offered as, in the order its portals stand. One on floor one,
// data.PortalOffers above it.
func (p *Pyramid) ChoicesAt(floor int) []Floor {
	if len(p.choices) == 0 {
		return nil
	}
	if floor < 1 {
		floor = 1
	}
	set := p.choices[(floor-1)%len(p.choices)]
	return append([]Floor(nil), set...)
}

// Floors is how many floors the climb was rolled for.
func (p *Pyramid) Floors() int { return len(p.choices) }

// FloorOf is which floor a fight index falls on, counting floors from one. The inverse of
// FirstFightOnFloor.
func FloorOf(fight int) int {
	if fight < 0 {
		return 1
	}
	return fight/FightsPerFloor + 1
}

// FightsPerFloor is how many fights a floor holds — outer room, inner room, portal room — and the
// third of them is its boss. See the tower section of MECHANICS.md.
//
// **It lives here rather than in `internal/screens`** because the combat screen names the room
// under the duelist card and the ascent curve counts rooms off the same number, and anything
// headless that maps a floor onto a fight index has to agree with the screen about how deep a
// floor is without being able to import it.
const FightsPerFloor = 3

// FirstFightOnFloor is the fight index of a floor's outer room, counting floors from one.
func FirstFightOnFloor(floor int) int {
	if floor < 1 {
		return 0
	}
	return (floor - 1) * FightsPerFloor
}

// Room is which of a floor's rooms a fight index is: outer, inner, or the portal room that is the
// floor's boss.
//
// **It is derived, never stored**, exactly as FloorOf is: the fight counter already says where the
// run is, and a second field saying the same thing is a second field to keep in step.
type Room int

const (
	// RoomOuter is the first room of a floor, RoomInner the second, RoomPortal the third — and
	// the third is the boss. Ordinals are positions in a floor, so this is not append-only in the
	// way an ID enum is; it is arithmetic on FightsPerFloor.
	RoomOuter Room = iota
	RoomInner
	RoomPortal
)

// RoomOf is which room of its floor a fight index falls in.
func RoomOf(fight int) Room {
	if fight < 0 {
		return RoomOuter
	}
	return Room(fight % FightsPerFloor)
}
