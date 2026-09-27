package pyramid

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The climb's offers: one theme on floor one, a portal's worth above it, and never the same motif
// twice in a run — whichever portal the player walks through.

func TestNoMotifIsOfferedTwiceInARun(t *testing.T) {
	// Every offer is spent, so a realm passed over on floor two can never come back on floor five.
	// Walked over many seeds because the draw that could break it is the rare one: a band that only
	// just fits.
	motifs, tower := data.LoadMotifs(), data.LoadTower()
	for seed := int64(0); seed < 500; seed++ {
		p := New(motifs, tower, rand.New(rand.NewSource(seed)))
		seen := map[string]int{}
		for floor := 1; floor <= tower.Floors; floor++ {
			offers := p.ChoicesAt(floor)
			if len(offers) != data.OffersOn(floor) {
				t.Fatalf("seed %d: floor %d offers %d realms, want %d", seed, floor, len(offers), data.OffersOn(floor))
			}
			for _, f := range offers {
				if was, ok := seen[f.Motif]; ok {
					t.Fatalf("seed %d: %s is offered on floor %d and again on floor %d", seed, f.Motif, was, floor)
				}
				seen[f.Motif] = floor
				if !motifs[f.Motif].AllowsFloor(floor) {
					t.Fatalf("seed %d: %s is offered on floor %d, outside its band", seed, f.Motif, floor)
				}
			}
		}
	}
}

func TestFloorOneIsOfferedOneRealm(t *testing.T) {
	// A run starts on floor one rather than walking into it, so there is nothing to choose there.
	p := New(data.LoadMotifs(), data.LoadTower(), rand.New(rand.NewSource(7)))
	if n := len(p.ChoicesAt(1)); n != 1 {
		t.Fatalf("floor one offers %d realms", n)
	}
}

func TestTheSameSeedOffersTheSameClimb(t *testing.T) {
	// The offers are the seed's alone, which is what makes a run code plus its picks the whole path.
	motifs, tower := data.LoadMotifs(), data.LoadTower()
	a := New(motifs, tower, rand.New(rand.NewSource(42)))
	b := New(motifs, tower, rand.New(rand.NewSource(42)))
	for floor := 1; floor <= tower.Floors; floor++ {
		x, y := a.ChoicesAt(floor), b.ChoicesAt(floor)
		for i := range x {
			if x[i] != y[i] {
				t.Fatalf("floor %d offer %d differs between two builds of one seed: %+v and %+v", floor, i, x[i], y[i])
			}
		}
	}
}
