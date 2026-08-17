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

// **One hand per catalogue key, not one per card** *(2026-08-16)*. An entry used to expand into
// one hand per attack concept, numbered `base + int(concept)`, with bands a hundred apart — which
// held twelve concepts and could not hold the four hundred a per-enemy deck list produces. The
// name is filled in at match time instead.
func TestEachLadderRungIsOneHand(t *testing.T) {
	for _, key := range []string{"pair", "flurry", "barrage"} {
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

// A templated name is filled from whatever concept formed the hand, which is the whole reason one
// entry can cover every card in the game.
func TestATemplatedHandIsNamedAfterTheCardThatFormedIt(t *testing.T) {
	pair, ok := HandByName("{card} Pair")
	if !ok {
		t.Fatal("the catalogue has no templated pair entry")
	}

	if got := HandName(pair, Strike); got != "Strike Pair" {
		t.Errorf("a pair of Strikes is called %q, want \"Strike Pair\"", got)
	}

	// A hand with no template is left alone rather than mangled.
	twoPair, ok := HandByName("Two Pair")
	if !ok {
		t.Fatal("the catalogue has no Two Pair")
	}
	if got := HandName(twoPair, Strike); got != "Two Pair" {
		t.Errorf("Two Pair became %q", got)
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
