package pyramid

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The roster shuffle. It builds its own records rather than loading `data/enemies.json`, so it
// is a test of the shuffle and not of whatever the roster happens to hold today.
//
// What it is defending is the reason the shuffle is banded at all: a run has to open on
// something a starting fighter can beat. A flat shuffle of all 96 would eventually deal a
// floor-eight opponent as fight one, and losing to one of those looks exactly like losing to
// bad draws, which is what makes it invisible in play.

func testRoster() map[string]data.EnemyData {
	recs := map[string]data.EnemyData{}
	for _, e := range []struct {
		name  string
		floor [2]int
	}{
		{"aardvark", [2]int{1, 2}}, {"badger", [2]int{1, 2}}, {"cobra", [2]int{1, 2}},
		{"dingo", [2]int{1, 2}}, {"emu", [2]int{1, 2}},
		{"ferret", [2]int{3, 5}}, {"gopher", [2]int{3, 5}}, {"heron", [2]int{3, 5}},
		{"ibis", [2]int{8, 8}}, {"jackal", [2]int{8, 8}},
	} {
		recs[e.name] = data.EnemyData{EnemyRecord: e.name, ValidFloors: e.floor}
	}
	return recs
}

func TestTheRosterShuffleKeepsTheFloorClimb(t *testing.T) {
	recs := testRoster()
	sorted := data.EnemyOrder(recs)

	// Every seed has to hold this, not the one that happened to be tried first.
	for seed := int64(0); seed < 50; seed++ {
		got := shuffleWithinFloors(sorted, recs, rand.New(rand.NewSource(seed)))

		if len(got) != len(sorted) {
			t.Fatalf("seed %d: shuffle returned %d names against %d in", seed, len(got), len(sorted))
		}

		// The bands must come out in the order they went in. Comparing the floor of each
		// name against the floor of the name the sort put in that position says it without
		// re-implementing the sort's tie-breaks.
		for i := range got {
			if recs[got[i]].ValidFloors != recs[sorted[i]].ValidFloors {
				t.Fatalf("seed %d: position %d holds %s from floors %v, where the sorted order has floors %v",
					seed, i, got[i], recs[got[i]].ValidFloors, recs[sorted[i]].ValidFloors)
			}
		}

		// And nothing may be dropped or duplicated: every enemy stays reachable by playing,
		// which is the whole point of walking the list rather than sampling it.
		a := append([]string(nil), got...)
		b := append([]string(nil), sorted...)
		sort.Strings(a)
		sort.Strings(b)
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("seed %d: the shuffled roster is not the same set of names", seed)
			}
		}
	}
}

func TestTheRosterShuffleActuallyVariesTheOpener(t *testing.T) {
	recs := testRoster()
	sorted := data.EnemyOrder(recs)

	// The point of the change: a different first fight from run to run. Five floor-one
	// enemies over fifty seeds should turn up more than one opener, and a shuffle that
	// returns its input unchanged is the failure worth catching.
	seen := map[string]bool{}
	for seed := int64(0); seed < 50; seed++ {
		seen[shuffleWithinFloors(sorted, recs, rand.New(rand.NewSource(seed)))[0]] = true
	}
	if len(seen) < 2 {
		t.Errorf("fifty seeds opened on %d different enemies", len(seen))
	}
}

func TestTheRosterShuffleIsDeterministic(t *testing.T) {
	recs := testRoster()
	sorted := data.EnemyOrder(recs)

	// Same seed, same run. This is what makes the planned replay-from-a-seed possible, and
	// what a stray `rand.Shuffle` on the global source would silently break.
	first := shuffleWithinFloors(sorted, recs, rand.New(rand.NewSource(99)))
	second := shuffleWithinFloors(sorted, recs, rand.New(rand.NewSource(99)))

	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("the same seed dealt %s then %s at position %d", first[i], second[i], i)
		}
	}
}
