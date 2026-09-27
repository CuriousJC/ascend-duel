package data

import "testing"

func TestEveryBackdropLoads(t *testing.T) {
	// LoadMotifs panics on a bad backdrops.json, so reaching the loop is most of the test; what is
	// checked here is that the backdrops actually arrived on their motif rather than being dropped
	// by a loader that never looked for the file.
	var n int
	for _, m := range LoadMotifs() {
		for _, b := range m.Backdrops {
			n++
			if b.Backdrop == "" || b.Art == "" {
				t.Errorf("%s: a backdrop arrived half-read: %+v", m.Motif, b)
			}
		}
	}
	if n == 0 {
		t.Error("no motif carries a backdrop, so backdrops.json is not being read")
	}
}

func TestABackdropIsTheSameForARunRealmAndRoom(t *testing.T) {
	m := MotifData{Motif: "m", Backdrops: []Backdrop{
		{Backdrop: "m-outer-a", Tier: TierOuter, Art: "a", Affinities: []string{"fire"}},
		{Backdrop: "m-outer-b", Tier: TierOuter, Art: "b", Affinities: []string{"fire"}},
		{Backdrop: "m-outer-c", Tier: TierOuter, Art: "c", Affinities: []string{"fire"}},
	}}
	for seed := int64(0); seed < 50; seed++ {
		first := m.BackdropFor(TierOuter, "fire", seed, 3)
		for range 5 {
			if got := m.BackdropFor(TierOuter, "fire", seed, 3); got != first {
				t.Fatalf("seed %d realm 3 drew %q then %q", seed, first, got)
			}
		}
	}
}

func TestABackdropIsTheTiersAndTheElements(t *testing.T) {
	m := MotifData{Motif: "m", Backdrops: []Backdrop{
		{Backdrop: "m-outer-hall", Tier: TierOuter, Art: "hall", Affinities: []string{"fire", "ice"}},
		{Backdrop: "m-inner-vault", Tier: TierInner, Art: "vault", Affinities: []string{"fire"}},
	}}
	for seed := int64(0); seed < 20; seed++ {
		if got := m.BackdropFor(TierOuter, "ice", seed, 1); got != "hall-ice" {
			t.Fatalf("the outer ice room drew %q", got)
		}
		if got := m.BackdropFor(TierInner, "fire", seed, 1); got != "vault-fire" {
			t.Fatalf("the inner fire room drew %q", got)
		}
	}
	// No inner room is drawn in ice and no portal room at all: both are the default, which is the
	// visible signal that a fight has no backdrop of its own.
	if got := m.BackdropFor(TierInner, "ice", 7, 1); got != DefaultBackgroundArt {
		t.Errorf("an inner ice fight with no room drew %q", got)
	}
	if got := m.BackdropFor(TierBoss, "fire", 7, 1); got != DefaultBackgroundArt {
		t.Errorf("a portal room with no backdrop drew %q", got)
	}
	if got := (MotifData{}).BackdropFor(TierOuter, "fire", 7, 1); got != DefaultBackgroundArt {
		t.Errorf("a motif with no backdrops drew %q", got)
	}
}

func TestSeveralBackdropsAreAllReached(t *testing.T) {
	// A hash that always landed on the first candidate would pass the determinism test and make
	// every backdrop after the first unreachable.
	m := MotifData{Motif: "m", Backdrops: []Backdrop{
		{Backdrop: "m-boss-a", Tier: TierBoss, Art: "a", Affinities: []string{"earth"}},
		{Backdrop: "m-boss-b", Tier: TierBoss, Art: "b", Affinities: []string{"earth"}},
	}}
	seen := map[string]bool{}
	for seed := int64(0); seed < 100; seed++ {
		seen[m.BackdropFor(TierBoss, "earth", seed, 1)] = true
	}
	if !seen["a-earth"] || !seen["b-earth"] {
		t.Errorf("a hundred runs drew only %v", seen)
	}
}

func TestARecordsOwnElementDirectionReplacesTheMotifs(t *testing.T) {
	m := MotifData{
		Draw:        "motif",
		ElementDraw: map[string]string{"fire": "generic fire", "ice": "generic ice"},
	}
	r := MotifRecord{
		Draw:        "record",
		Affinities:  []string{"fire", "ice"},
		ElementDraw: map[string]string{"fire": "specific fire", "ice": DrawUnwritten},
	}
	if got := m.Brief(r, "fire"); len(got) != 3 || got[1] != "specific fire" {
		t.Errorf("fire brief is %q, want the record's own fire in place of the motif's", got)
	}
	// An unwritten override is no override: the motif's line stands.
	if got := m.Brief(r, "ice"); len(got) != 3 || got[1] != "generic ice" {
		t.Errorf("ice brief is %q, want the motif's ice", got)
	}
}
