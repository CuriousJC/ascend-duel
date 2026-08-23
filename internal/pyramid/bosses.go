package pyramid

// The stairway protectors: which boss stands on which floor.
//
// **One boss per floor, chosen once per run, and never met in an ordinary room.** The roster is
// shuffled inside floor bands and walked room by room (see roster.go); the bosses are a separate
// pool and a separate draw, because a floor has one stairway and several bosses are authored for
// it — so a run picks which of them it climbs past rather than meeting all of them.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
)

// pickBosses chooses one boss for each floor the pool covers, lowest floor first.
//
// It takes the source rather than reaching for one, for the reason shuffleWithinFloors does: the
// caller owns which stream is advanced, and a fixed seed plus a fake pool is what makes this
// testable.
//
// **A floor with no boss authored for it takes the next floor's**, rather than the run reaching a
// stairway with nobody on it. That is a fallback for a half-filled pool during authoring, not a
// design: the shipped file covers floors 1 to 8.
func pickBosses(recs map[string]data.BossData, rng *rand.Rand) []string {
	names := data.BossOrder(recs)
	if len(names) == 0 {
		return nil
	}

	highest := 0
	byFloor := map[int][]string{}
	for _, n := range names {
		f := recs[n].Floor
		byFloor[f] = append(byFloor[f], n)
		if f > highest {
			highest = f
		}
	}

	out := make([]string, 0, highest)
	var last string
	for floor := 1; floor <= highest; floor++ {
		band := byFloor[floor]
		if len(band) == 0 {
			out = append(out, last)
			continue
		}
		last = band[rng.Intn(len(band))]
		out = append(out, last)
	}

	// A pool whose lowest floor is not 1 would leave holes at the front, which the loop above
	// fills with an empty string. Backfill them from the first floor that has somebody.
	for i := range out {
		if out[i] != "" {
			break
		}
		out[i] = names[0]
	}
	return out
}

// BossAt is the record key of the boss guarding a given floor's stairway, wrapping past the
// highest authored floor for the same reason EnemyAt wraps past the end of the roster: nothing
// caps the fight index, so a run played past the eighth floor keeps finding an opponent.
//
// It returns the empty string only for a pyramid built from an empty boss pool, which a loaded
// game cannot produce.
func (p *Pyramid) BossAt(floor int) string {
	if len(p.bosses) == 0 {
		return ""
	}
	if floor < 1 {
		floor = 1
	}
	return p.bosses[(floor-1)%len(p.bosses)]
}
