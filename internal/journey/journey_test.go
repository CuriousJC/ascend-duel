package journey

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The journey's offers: one theme on realm one, a portal's worth above it, and never the same motif
// twice in a run — whichever portal the player walks through.

func TestNoMotifIsOfferedTwiceInARun(t *testing.T) {
	// Every offer is spent, so a realm passed over on realm two can never come back on realm five.
	// Walked over many seeds because the draw that could break it is the rare one: a band that only
	// just fits.
	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	for seed := int64(0); seed < 500; seed++ {
		p := New(motifs, shape, rand.New(rand.NewSource(seed)))
		seen := map[string]int{}
		for realm := 1; realm <= shape.Realms; realm++ {
			offers := p.ChoicesAt(realm)
			if len(offers) != data.OffersOn(realm) {
				t.Fatalf("seed %d: realm %d offers %d realms, want %d", seed, realm, len(offers), data.OffersOn(realm))
			}
			for _, f := range offers {
				if was, ok := seen[f.Motif]; ok {
					t.Fatalf("seed %d: %s is offered on realm %d and again on realm %d", seed, f.Motif, was, realm)
				}
				seen[f.Motif] = realm
				if !motifs[f.Motif].AllowsRealm(realm) {
					t.Fatalf("seed %d: %s is offered on realm %d, outside its band", seed, f.Motif, realm)
				}
			}
		}
	}
}

func TestRealmOneOffersOneMotif(t *testing.T) {
	// A run starts on realm one rather than walking into it, so there is nothing to choose there.
	p := New(data.LoadMotifs(), data.LoadJourney(), rand.New(rand.NewSource(7)))
	if n := len(p.ChoicesAt(1)); n != 1 {
		t.Fatalf("realm one offers %d realms", n)
	}
}

func TestTheSameSeedOffersTheSameJourney(t *testing.T) {
	// The offers are the seed's alone, which is what makes a run code plus its picks the whole path.
	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	a := New(motifs, shape, rand.New(rand.NewSource(42)))
	b := New(motifs, shape, rand.New(rand.NewSource(42)))
	for realm := 1; realm <= shape.Realms; realm++ {
		x, y := a.ChoicesAt(realm), b.ChoicesAt(realm)
		for i := range x {
			if x[i] != y[i] {
				t.Fatalf("realm %d offer %d differs between two builds of one seed: %+v and %+v", realm, i, x[i], y[i])
			}
		}
	}
}
