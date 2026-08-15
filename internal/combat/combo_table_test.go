package combat

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The catalogue is a file, so these are the tests that used to be unnecessary: a Go table could
// not be malformed and JSON can. **A hand silently dropped is a balance change nobody made**,
// which is why loadCatalogue panics rather than skipping an entry.

func TestTheShippingCatalogueLoads(t *testing.T) {
	for _, h := range Hands() {
		if h.Key == "" {
			t.Errorf("hand %q has no catalogue key", h.Name)
		}
		if h.Cards() == 0 {
			t.Errorf("hand %q counts nothing", h.Name)
		}
		// **A one-card hand may be worth nothing and a built one may not.** The High Card pays
		// no multiplier on purpose — what lands is the card's own face damage — and it is the only
		// hand nobody chooses to form. Anything asking for two cards and paying nothing is a typo.
		if h.Cards() > 1 && h.Multiplier <= 0 {
			t.Errorf("hand %q asks for %d cards and is worth nothing", h.Name, h.Cards())
		}
		if h.Multiplier < 0 {
			t.Errorf("hand %q has a negative multiplier", h.Name)
		}
	}
	for _, m := range Mixes() {
		if m.Key == "" {
			t.Errorf("mix %q has no catalogue key", m.Name)
		}
	}
}

// **The mixes have to partition every hand.** A turn showing a colour count no mix claims would
// form a hand and then have no makeup, which is the one way this model can produce a combo it
// cannot name. The loader panics on a gap; this is the positive side of that.
func TestEveryColourCountHasExactlyOneMix(t *testing.T) {
	for n := 0; n <= ElementCount-1; n++ {
		found := 0
		for _, m := range Mixes() {
			if m.Colours == n {
				found++
			}
		}
		if found != 1 {
			t.Errorf("%d colours is claimed by %d mixes, want exactly 1", n, found)
		}
	}
}

// Every attack card carries the whole ladder, because the file expands one entry per card. A new
// attack card gets pair, flurry and barrage by existing.
func TestEveryAttackCardCarriesTheWholeLadder(t *testing.T) {
	for _, key := range []string{"pair", "flurry", "barrage"} {
		for _, a := range AllActions {
			id, ok := HandIDFor(key, a)
			if a.Category() != CategoryAttack {
				if ok {
					t.Errorf("%v is not an attack but carries a %s (%d)", a, key, id)
				}
				continue
			}
			if !ok {
				t.Errorf("%v has no %s", a, key)
				continue
			}
			if _, found := HandByID(id); !found {
				t.Errorf("%v's %s claims ID %d, which is in no table", a, key, id)
			}
		}
	}
}

// The rungs have to climb, or a bigger hand would pay less than the one inside it and the
// best-hand rule would pick the wrong one.
func TestTheLadderClimbs(t *testing.T) {
	last := 0
	for _, key := range []string{"pair", "two-pair", "flurry", "full-house", "barrage"} {
		var h Hand
		for _, c := range Hands() {
			if c.Key == key {
				h = c
				break
			}
		}
		if h.Key == "" {
			t.Fatalf("the catalogue has no %q", key)
		}
		if h.Multiplier <= last {
			t.Errorf("%s pays x%d, which is not more than the rung below it (x%d)", key, h.Multiplier, last)
		}
		last = h.Multiplier
	}
}

// A file that names something the rules do not have, or describes a shape they cannot match, is
// refused rather than half-loaded.
func TestAMalformedHandIsRefused(t *testing.T) {
	for _, tc := range []struct {
		what string
		rec  data.HandData
	}{
		{"no key", data.HandData{ID: 1, Name: "x", Groups: []int{2}, Multiplier: 100}},
		{"no name", data.HandData{Key: "k", ID: 1, Groups: []int{2}, Multiplier: 100}},
		{"no ID", data.HandData{Key: "k", Name: "x", Groups: []int{2}, Multiplier: 100}},
		{"no groups", data.HandData{Key: "k", ID: 1, Name: "x", Multiplier: 100}},
		{"a group of none", data.HandData{Key: "k", ID: 1, Name: "x", Groups: []int{0}, Multiplier: 100}},
		{"no multiplier", data.HandData{Key: "k", ID: 1, Name: "x", Groups: []int{2}}},
		{"more cards than a turn holds", data.HandData{
			Key: "k", ID: 1, Name: "x", Groups: []int{4, 4}, Multiplier: 100}},
		{"expanding two groups", data.HandData{
			Key: "k", ID: 1, Name: "x", Groups: []int{2, 2}, Multiplier: 100, Expand: expandAttacks}},
		{"unknown expansion", data.HandData{
			Key: "k", ID: 1, Name: "x", Groups: []int{2}, Multiplier: 100, Expand: "per-duelist"}},
	} {
		if err := validateHand(tc.rec); err == nil {
			t.Errorf("%s: should have been refused", tc.what)
		}
	}
}

func TestAMalformedMixIsRefused(t *testing.T) {
	for _, tc := range []struct {
		what string
		rec  data.MixData
	}{
		{"no key", data.MixData{ID: 1, Name: "x"}},
		{"no name", data.MixData{Key: "k", ID: 1}},
		{"no ID", data.MixData{Key: "k", Name: "x"}},
		{"negative colours", data.MixData{Key: "k", ID: 1, Name: "x", Colours: -1}},
		{"more colours than exist", data.MixData{Key: "k", ID: 1, Name: "x", Colours: ElementCount}},
	} {
		if err := validateMix(tc.rec); err != nil {
			continue
		}
		t.Errorf("%s: should have been refused", tc.what)
	}
}

// Scope is joined to the rules by the same parser the deck lists use, and a typo is reported
// rather than falling back to something playable.
func TestAHandNamingAnUnknownScopeIsRefused(t *testing.T) {
	rec := data.HandData{
		Key: "k", ID: 1, Name: "x", Groups: []int{2}, Multiplier: 100,
		Scope: []string{"riposting"},
	}
	if err := validateHand(rec); err != nil {
		t.Fatalf("the fixture should pass shape validation, got %v", err)
	}
	if _, err := expandHand(rec); err == nil {
		t.Error("a scope the rules do not have should be refused")
	}
}
