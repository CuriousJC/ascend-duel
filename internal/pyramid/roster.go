package pyramid

// The fight order: which opponent stands in which room.
//
// **The shuffle is within floor bands, never across them.** The roster is sorted by the floor an
// enemy is valid on, and reordering inside each run of equal ValidFloors is what varies a run
// without letting a floor-6 boss turn up in the first room.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
)

// shuffleWithinFloors reorders a floor-sorted roster inside each run of equal ValidFloors,
// leaving the bands themselves where they are.
//
// It takes the source rather than reaching for one, so the caller owns which stream is being
// advanced, and it is a plain function of its arguments so a test can hand it a fake roster
// and a fixed seed. Never `rand.Shuffle` — the package-level one draws from a global source
// shared with every other caller and would make a run unreproducible.
func shuffleWithinFloors(names []string, recs map[string]data.EnemyData, rng *rand.Rand) []string {
	out := append([]string(nil), names...)

	for start := 0; start < len(out); {
		end := start + 1
		for end < len(out) && recs[out[end]].ValidFloors == recs[out[start]].ValidFloors {
			end++
		}

		band := out[start:end]
		rng.Shuffle(len(band), func(i, j int) { band[i], band[j] = band[j], band[i] })

		start = end
	}
	return out
}
