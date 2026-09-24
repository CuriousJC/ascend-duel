package data

import "testing"

// TestEveryMotifCanFieldEveryFight is the promise the floor generator is built on: pick any
// motif and any element and each of the three rooms has as many records as its tier requires.
//
// LoadMotifs already panics on a hole, so this failing means the check was loosened rather than
// that a file drifted. It is here because the panic is the tripwire and this is the statement of
// what the tripwire is for.
func TestEveryMotifCanFieldEveryFight(t *testing.T) {
	motifs := LoadMotifs()
	if len(motifs) == 0 {
		t.Fatal("no motifs loaded")
	}
	for _, key := range MotifOrder(motifs) {
		if holes := CoverageOf(motifs[key]).Holes(); len(holes) > 0 {
			t.Errorf("%s: %v", key, holes)
		}
	}
}

// TestTheTowerCanBeClimbed fails when the floor bands stop being able to give every floor a motif
// of its own — which is a matching problem, not a per-floor one, and is why a roster can pass
// "every floor has a candidate" and still be unbuildable.
func TestTheTowerCanBeClimbed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the shipped roster cannot climb the shipped tower: %v", r)
		}
	}()
	MustBeClimbable(LoadMotifs(), LoadTower().Floors)
}

// TestEveryRecordNamesItsOwnMotifAndTier holds the key format the coverage report trusts. A key
// that lies about its tier is counted in the wrong column, and the column is what the generator
// believes.
func TestEveryRecordNamesItsOwnMotifAndTier(t *testing.T) {
	motifs := LoadMotifs()
	for _, key := range MotifOrder(motifs) {
		m := motifs[key]
		for _, r := range m.Records {
			want := m.Motif + "-" + r.Tier + "-"
			if len(r.Record) <= len(want) || r.Record[:len(want)] != want {
				t.Errorf("%s should be keyed %s<slug>", r.Record, want)
			}
		}
	}
}

// TestArtIsPerElement holds the one thing a missing picture may do, which is fall back rather than
// draw nothing — and that a record dealt as two elements asks for two different files.
func TestArtIsPerElement(t *testing.T) {
	motifs := LoadMotifs()
	for _, key := range MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			seen := map[string]bool{}
			for _, e := range r.Affinities {
				k := r.ArtKey(e)
				if k == DefaultEnemyArt {
					t.Errorf("%s dealt as %s asks for the placeholder rather than its own picture", r.Record, e)
				}
				if seen[k] {
					t.Errorf("%s asks for %s twice", r.Record, k)
				}
				seen[k] = true
			}
			if got := r.ArtKey(""); got != DefaultEnemyArt {
				t.Errorf("%s with no element should fall back to %s, got %s", r.Record, DefaultEnemyArt, got)
			}
		}
	}
}
