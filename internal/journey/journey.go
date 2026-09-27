package journey

import (
	"math/rand"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
)

// Journey is one run's journey: every realm's theme and the creature standing in each of its rooms.
//
// **A run holds one and it outlives every fight.** It is built once, from the loaded motifs and
// the run's own seed, and walked by fight index — so a defeat and a retry meet the same opponent
// again rather than a different one.
//
// **A realm is a motif and an element.** The three rooms are three records of that motif, each
// dealt as that element, so a fire goblin realm is three goblins in fire and the player can plan
// against what they walked into.
type Journey struct {
	// choices is what each realm is offered as: one theme on realm one, where a run starts, and a
	// portal's worth on every realm above it.
	//
	// **Every one is rolled up front and every one is spent.** A motif offered on a realm is struck
	// from every realm above it whether or not the player walks through its portal, so what a realm
	// offers is a function of the run code alone and never of an earlier pick. Which offer was taken
	// is the run's — see internal/session — and this stays a pure function of the seed.
	choices [][]Realm
}

// Realm is one themed realm: which motif it is, which element its creatures are dealt as, and
// the record standing in each of its rooms.
type Realm struct {
	Motif   string
	Element string

	// Rooms is one record key per room, indexed by Room — outer, inner, portal.
	Rooms [FightsPerRealm]string
}

// New builds a run's journey from the loaded motifs, the journey's own shape, and a seeded source.
//
// It takes the source rather than reaching for one, so the caller owns which stream is being
// advanced, and it is a plain function of its arguments so a test can hand it a fake roster and a
// fixed seed. Never `rand.Shuffle` — the package-level one draws from a global shared with every
// other caller and would make a run unreproducible.
//
// **A motif is never offered twice in one run.** Each realm's offers are struck off before the next
// realm is rolled, and **a draw is only kept if the realms above can still be filled without it** —
// data.FillsSlots is asked after every pick, so a motif a later realm depends on is never spent on
// an earlier one that had alternatives. data.MustFillJourney refuses, at load, a roster where no
// order of draws could work, so the panic below is unreachable on a loaded catalog.
func New(motifs map[string]data.MotifData, shape data.JourneyData, rng *rand.Rand) *Journey {
	order := data.MotifOrder(motifs)
	used := map[string]bool{}

	p := &Journey{}
	for realm := 1; realm <= shape.Realms; realm++ {
		var free []string
		for _, key := range order {
			if !used[key] && motifs[key].AllowsRealm(realm) {
				free = append(free, key)
			}
		}

		// Shuffled rather than picked by index, so the offers are a sample of what is left rather
		// than the front of a sorted list.
		rng.Shuffle(len(free), func(i, j int) { free[i], free[j] = free[j], free[i] })

		want := data.OffersOn(realm)
		var taken []string
		for _, key := range free {
			if len(taken) == want {
				break
			}
			used[key] = true
			// What is still owed: the rest of this realm's offers, then every realm above.
			owed := data.JourneySlots(realm+1, shape.Realms)
			for range want - len(taken) - 1 {
				owed = append(owed, realm)
			}
			if !data.FillsSlots(motifs, owed, used) {
				delete(used, key)
				continue
			}
			taken = append(taken, key)
		}
		if len(taken) < want {
			panic("journey: realm " + strconv.Itoa(realm) + " cannot be offered " + strconv.Itoa(want) + " motifs of its own")
		}

		set := make([]Realm, 0, want)
		for _, key := range taken {
			set = append(set, rollRealm(motifs[key], rng))
		}
		p.choices = append(p.choices, set)
	}
	return p
}

// rollRealm gives a motif an element and fills its three rooms.
//
// **Every element is reachable**, because data.LoadMotifs refuses a motif that cannot field all
// three rooms at all five — so this never has to ask whether the theme it rolled is buildable.
func rollRealm(m data.MotifData, rng *rand.Rand) Realm {
	f := Realm{
		Motif:   m.Motif,
		Element: data.AffinityElements[rng.Intn(len(data.AffinityElements))],
	}
	for i, tier := range data.TierOrder {
		can := m.Candidates(tier, f.Element)
		if len(can) == 0 {
			// Unreachable on a loaded catalog, and a name is better than a zero value if the
			// loader is ever loosened.
			panic("journey: " + m.Motif + " cannot field a " + f.Element + " " + tier)
		}
		f.Rooms[i] = can[rng.Intn(len(can))].Record
	}
	return f
}

// RealmAt is a realm's first offer, counting realms from one: the theme realm one is fought at, and
// what any realm is fought at until a pick says otherwise. The run's own answer, pick included, is
// session.Session's — see internal/session/journey.go.
//
// **Past the top of the journey it wraps**, rather than the run stopping: the growth curve has no
// ceiling and neither does the journey. What wraps is which realms are met again, not how hard they
// are — see ScaleToFight, which keeps counting.
func (p *Journey) RealmAt(realm int) Realm {
	if len(p.choices) == 0 {
		return Realm{}
	}
	if realm < 1 {
		realm = 1
	}
	set := p.choices[(realm-1)%len(p.choices)]
	return set[0]
}

// ChoicesAt is every theme a realm is offered as, in the order its portals stand. One on realm one,
// data.PortalOffers above it.
func (p *Journey) ChoicesAt(realm int) []Realm {
	if len(p.choices) == 0 {
		return nil
	}
	if realm < 1 {
		realm = 1
	}
	set := p.choices[(realm-1)%len(p.choices)]
	return append([]Realm(nil), set...)
}

// Realms is how many realms the journey was rolled for.
func (p *Journey) Realms() int { return len(p.choices) }

// RealmOf is which realm a fight index falls on, counting realms from one. The inverse of
// FirstFightInRealm.
func RealmOf(fight int) int {
	if fight < 0 {
		return 1
	}
	return fight/FightsPerRealm + 1
}

// FightsPerRealm is how many fights a realm holds — outer room, inner room, portal room — and the
// third of them is its boss. See the journey section of MECHANICS.md.
//
// **It lives here rather than in `internal/screens`** because the combat screen names the room
// under the duelist card and the growth curve counts rooms off the same number, and anything
// headless that maps a realm onto a fight index has to agree with the screen about how deep a
// realm is without being able to import it.
const FightsPerRealm = 3

// FirstFightInRealm is the fight index of a realm's outer room, counting realms from one.
func FirstFightInRealm(realm int) int {
	if realm < 1 {
		return 0
	}
	return (realm - 1) * FightsPerRealm
}

// Room is which of a realm's rooms a fight index is: outer, inner, or the portal room that is the
// realm's boss.
//
// **It is derived, never stored**, exactly as RealmOf is: the fight counter already says where the
// run is, and a second field saying the same thing is a second field to keep in step.
type Room int

const (
	// RoomOuter is the first room of a realm, RoomInner the second, RoomPortal the third — and
	// the third is the boss. Ordinals are positions in a realm, so this is not append-only in the
	// way an ID enum is; it is arithmetic on FightsPerRealm.
	RoomOuter Room = iota
	RoomInner
	RoomPortal
)

// RoomOf is which room of its realm a fight index falls in.
func RoomOf(fight int) Room {
	if fight < 0 {
		return RoomOuter
	}
	return Room(fight % FightsPerRealm)
}
