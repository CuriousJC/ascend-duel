package data

import "testing"

// TestEveryMotifCanFieldEveryFight is the promise the realm generator is built on: pick any
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

// TestTheJourneyCanBeFilled fails when the realm bands stop being able to give every realm a motif
// of its own — which is a matching problem, not a per-realm one, and is why a roster can pass
// "every realm has a candidate" and still be unbuildable.
func TestTheJourneyCanBeFilled(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the shipped roster cannot fill the shipped journey: %v", r)
		}
	}()
	MustFillJourney(LoadMotifs(), LoadJourney().Realms)
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

// TestEveryCreatureCardWearsAFlavorOfItsOwnForm. checkRecord refuses a creature card with no
// form, with a flavor enemy_card_art.json does not have, or with a flavor of another form — the
// corner mark and the picture must never disagree about whether a bite is a stab. This states the
// rule and checks each refusal fires.
func TestEveryCreatureCardWearsAFlavorOfItsOwnForm(t *testing.T) {
	m := LoadMotifs()["goblins"]
	r := m.Records[0]
	good := r.Cards[0]

	refuses := func(name string, c MotifCard) {
		t.Helper()
		rec := r
		rec.Cards = []MotifCard{c}
		defer func() {
			if recover() == nil {
				t.Errorf("a card with %s loaded", name)
			}
		}()
		checkRecord(m, rec, "test")
	}

	noForm := good
	noForm.Form = ""
	refuses("no form", noForm)

	defend := good
	defend.Form = "defend"
	refuses("the defend form", defend)

	unknown := good
	unknown.Flavor = "stab-spoon"
	refuses("a flavor the catalog does not have", unknown)

	crossed := good
	crossed.Form = "crush"
	crossed.Flavor = "stab-bite"
	refuses("a flavor of another form", crossed)

	art := LoadEnemyCardArt()
	used := map[string]bool{}
	motifs := LoadMotifs()
	for _, key := range MotifOrder(motifs) {
		for _, rec := range motifs[key].Records {
			for _, c := range rec.Cards {
				used[c.Flavor] = true
			}
		}
	}
	for _, flavor := range EnemyCardArtOrder(art) {
		if !used[flavor] {
			t.Errorf("no creature card wears %s, so its pictures would never be drawn", flavor)
		}
	}
}
