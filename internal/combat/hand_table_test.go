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
// cards in this turn agree — and a rung named for anything else would be a second vocabulary for
// something the player already knows. **What they agree *on* is the axis** *(2026-08-19)*, and it
// is said in the name rather than left to be inferred from the cards that lit up.
func TestTheLadderIsThePokerHandsOnEveryAxis(t *testing.T) {
	want := map[string]string{
		"high-card": "High Card",

		"concept-pair":            "Card Pair",
		"concept-two-pair":        "Card Two Pair",
		"concept-three-of-a-kind": "Card Three of a Kind",
		"concept-full-house":      "Card Full House",
		"concept-four-of-a-kind":  "Card Four of a Kind",
		"concept-five-of-a-kind":  "Card Five of a Kind",

		"form-pair":            "Form Pair",
		"form-two-pair":        "Form Two Pair",
		"form-three-of-a-kind": "Form Three of a Kind",
		"form-full-house":      "Form Full House",
		"form-four-of-a-kind":  "Form Four of a Kind",
		"form-five-of-a-kind":  "Form Five of a Kind",

		"element-pair":            "Elemental Pair",
		"element-two-pair":        "Elemental Two Pair",
		"element-three-of-a-kind": "Elemental Three of a Kind",
		"element-full-house":      "Elemental Full House",
		"element-four-of-a-kind":  "Elemental Four of a Kind",
		"element-five-of-a-kind":  "Elemental Five of a Kind",
	}

	if got := len(Hands()); got != len(want) {
		t.Errorf("the catalogue holds %d hands, want %d - six rungs on each of three axes, plus the High Card", got, len(want))
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
	for _, key := range []string{"concept-pair", "form-three-of-a-kind", "element-four-of-a-kind"} {
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
// best-hand rule would pick the wrong one. It climbs once per axis *(2026-08-19)*, since every
// rung exists three times over. Across axes
// the numbers deliberately do not line up — a form pair is far commoner than a card pair and pays
// less — so this walks each ladder on its own.
func TestTheLadderClimbs(t *testing.T) {
	for _, axis := range []string{"concept", "form", "element"} {
		last := 0
		for _, rung := range []string{"pair", "two-pair", "three-of-a-kind", "full-house", "four-of-a-kind",
			"five-of-a-kind"} {
			key := axis + "-" + rung
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
}

// Every rung exists on every axis, and no axis is missing one. A rung built on two axes and not
// the third would be a hole a player could fall into without ever being told it was there.
func TestEveryRungExistsOnEveryAxis(t *testing.T) {
	for _, axis := range AllAxes {
		for _, rung := range []string{"pair", "two-pair", "three-of-a-kind", "full-house", "four-of-a-kind",
			"five-of-a-kind"} {
			key := axis.String() + "-" + rung
			h, ok := handByKey(key)
			if !ok {
				t.Errorf("the catalogue has no %q", key)
				continue
			}
			if h.Match != axis {
				t.Errorf("%s matches on %s", key, h.Match)
			}
		}
	}
}

// The narrower axis pays more at every rung, which is what makes aiming at a card hand worth the
// extra difficulty. A concept fixes a form, so a card hand is always also a form hand — if the
// form rung paid the same or better, nobody would ever have reason to build the narrower one.
func TestANarrowerAxisPaysMore(t *testing.T) {
	for _, rung := range []string{"pair", "two-pair", "three-of-a-kind", "full-house", "four-of-a-kind",
		"five-of-a-kind"} {
		card, ok := handByKey("concept-" + rung)
		if !ok {
			t.Fatalf("the catalogue has no concept-%s", rung)
		}
		for _, wider := range []string{"form", "element"} {
			h, ok := handByKey(wider + "-" + rung)
			if !ok {
				t.Fatalf("the catalogue has no %s-%s", wider, rung)
			}
			if h.Multiplier > card.Multiplier {
				t.Errorf("%s-%s pays x%d, beating the narrower concept-%s at x%d",
					wider, rung, h.Multiplier, rung, card.Multiplier)
			}
		}
	}
}

// A file describing a shape the rules cannot match is refused rather than half-loaded.
func TestAMalformedHandIsRefused(t *testing.T) {
	for _, tc := range []struct {
		what string
		rec  data.HandData
	}{
		{"no key", data.HandData{ID: 1, Name: "x", Match: "concept", Groups: []int{2}, Multiplier: 100}},
		{"no name", data.HandData{Key: "k", ID: 1, Match: "concept", Groups: []int{2}, Multiplier: 100}},
		{"no ID", data.HandData{Key: "k", Name: "x", Match: "concept", Groups: []int{2}, Multiplier: 100}},
		{"no groups", data.HandData{Key: "k", ID: 1, Name: "x", Match: "concept", Multiplier: 100}},
		{"a group of none", data.HandData{Key: "k", ID: 1, Name: "x", Match: "concept", Groups: []int{0}, Multiplier: 100}},
		{"no multiplier", data.HandData{Key: "k", ID: 1, Name: "x", Match: "concept", Groups: []int{2}}},
		{"more cards than a turn holds", data.HandData{
			Key: "k", ID: 1, Name: "x", Match: "concept", Groups: []int{4, 4}, Multiplier: 100}},
		// **An axis is required rather than defaulted**: an entry landing on the wrong one by
		// omission would be a balance change nobody made.
		{"no axis", data.HandData{Key: "k", ID: 1, Name: "x", Groups: []int{2}, Multiplier: 150}},
		{"an axis the rules do not have", data.HandData{
			Key: "k", ID: 1, Name: "x", Match: "suit", Groups: []int{2}, Multiplier: 150}},
		// **Four forms reach a blow as of 2026-08-23**, since plans join hands, so a five-group form
		// hand is the rung nobody could climb — and it would fail silently rather than loudly
		// without the check. It was four groups while `FormDefend` was filtered out of the matcher.
		{"more groups than the axis has values", data.HandData{
			Key: "k", ID: 1, Name: "x", Match: "form", Groups: []int{1, 1, 1, 1, 1}, Multiplier: 150}},
	} {
		if _, err := validateHand(tc.rec); err == nil {
			t.Errorf("%s: should have been refused", tc.what)
		}
	}
}

// The element axis holds four values and the form axis three, so a two-group hand is legal on both
// and this is the case the spread check must not refuse.
func TestAWellFormedHandOnEveryAxisIsAccepted(t *testing.T) {
	for _, axis := range AllAxes {
		rec := data.HandData{Key: "k", ID: 1, Name: "x", Match: axis.String(), Groups: []int{3, 2}, Multiplier: 300}
		got, err := validateHand(rec)
		if err != nil {
			t.Errorf("a full house on the %s axis was refused: %v", axis, err)
		}
		if got != axis {
			t.Errorf("a %s hand validated as %s", axis, got)
		}
	}
}
