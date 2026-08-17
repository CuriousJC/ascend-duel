package combat

import (
	"strings"
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
}

// **The ladder wears poker's names** *(2026-08-17)*, because it asks poker's question — how many
// copies of a card are in this turn — and a rung named for anything else would be a second
// vocabulary for something the player already knows.
func TestTheLadderIsTheSixPokerHands(t *testing.T) {
	want := map[string]string{
		"high-card":       "High Card",
		"pair":            "Pair",
		"two-pair":        "Two Pair",
		"three-of-a-kind": "Three of a Kind",
		"full-house":      "Full House",
		"four-of-a-kind":  "Four of a Kind",
	}

	if got := len(Hands()); got != len(want) {
		t.Errorf("the catalogue holds %d hands, want the %d poker ones", got, len(want))
	}
	for key, name := range want {
		h, ok := handByKey(key)
		if !ok {
			t.Errorf("the catalogue has no %q", key)
			continue
		}
		if h.Name != name {
			t.Errorf("%s is called %q, want %q", key, h.Name, name)
		}
	}
}

// **One hand per catalogue key** *(2026-08-16)*. An entry used to expand into one hand per attack
// concept, numbered `base + int(concept)`, with bands a hundred apart — which held twelve concepts
// and could not hold the four hundred a per-enemy deck list produces.
func TestEachLadderRungIsOneHand(t *testing.T) {
	for _, key := range []string{"pair", "three-of-a-kind", "four-of-a-kind"} {
		id, ok := HandIDForKey(key)
		if !ok {
			t.Errorf("no hand called %s", key)
			continue
		}

		found := 0
		for _, h := range Hands() {
			if h.Key == key {
				found++
			}
		}
		if found != 1 {
			t.Errorf("%s is %d hands, want exactly 1", key, found)
		}
		if _, in := HandByID(id); !in {
			t.Errorf("%s claims ID %d, which is in no table", key, id)
		}
	}
}

// **A hand's name is the whole of its name** *(2026-08-17)*. It used to hold a `{card}` template
// filled from whichever concept formed it, and to be prefixed on screen by the element makeup, so
// one hand could print as "Duo Strike Flurry". Both axes are gone; what the feed prints is what the
// file says.
func TestAHandCarriesItsWholeName(t *testing.T) {
	for _, h := range Hands() {
		if h.Name == "" {
			t.Errorf("hand %q has no name", h.Key)
		}
		if strings.Contains(h.Name, "{card}") {
			t.Errorf("hand %q still holds a name template: %q", h.Key, h.Name)
		}
	}
}

// The rungs have to climb, or a bigger hand would pay less than the one inside it and the
// best-hand rule would pick the wrong one.
func TestTheLadderClimbs(t *testing.T) {
	last := 0
	for _, key := range []string{"pair", "two-pair", "three-of-a-kind", "full-house", "four-of-a-kind"} {
		h, ok := handByKey(key)
		if !ok {
			t.Fatalf("the catalogue has no %q", key)
		}
		if h.Multiplier <= last {
			t.Errorf("%s pays x%d, which is not more than the rung below it (x%d)", key, h.Multiplier, last)
		}
		last = h.Multiplier
	}
}

// A file describing a shape the rules cannot match is refused rather than half-loaded.
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
	} {
		if err := validateHand(tc.rec); err == nil {
			t.Errorf("%s: should have been refused", tc.what)
		}
	}
}
