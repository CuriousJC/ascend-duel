package combat

import (
	"sort"
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

		// **The spread rungs, and the cost axis** *(owner's call, 2026-09-05)*. Everything above
		// counts copies; these count *difference*, which is the other thing a set of cards can
		// have in common. Prism, Spectrum and Elementalist are the elemental ladder of it, Arsenal
		// is the form one, and the two cost rungs are the axis that arrived with them.
		"element-prism":        "Prism",
		"element-spectrum":     "Spectrum",
		"element-elementalist": "Elementalist",
		"form-arsenal":         "Arsenal",

		"cost-rising-attack": "Rising Attack",
		"cost-weaponmaster":  "Weaponmaster",
	}

	if got := len(Hands()); got != len(want) {
		t.Errorf("the catalogue holds %d hands, want %d - six rungs on each of three of-a-kind axes, four spread rungs, two cost rungs, plus the High Card", got, len(want))
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

// The rungs have to climb, or a hand would pay less than one it *contains* and the best-hand rule
// would pick the wrong one.
//
// **It climbs by containment, not by poker's order** *(owner's call, 2026-09-05)*. This used to
// walk pair, two pair, three of a kind, full house, four of a kind, five of a kind and require each
// to beat the last — which is poker's ladder, and poker has thirteen ranks. This deck has four
// forms and five elements, so on those axes **spreading is harder than stacking**: a Form Two Pair
// wants two distinct forms and four cards and is measurably rarer than a Form Three of a Kind. The
// old order forced the ladder to pay the commoner hand more, which is the inversion tools/handodds
// was showing and this test was enforcing.
//
// What survives is the part that was always sound: a rung whose groups **dominate** another's on
// the same axis contains it, so it must pay more. `[3]` contains `[2]`; `[3,2]` contains both `[3]`
// and `[2,2]`; `[5]` contains `[4]`. `[2,2]` and `[3]` contain neither, so the file is free to
// price them in whatever order the measurements say.
func TestTheLadderClimbs(t *testing.T) {
	hands := Hands()
	for _, big := range hands {
		for _, small := range hands {
			if big.Key == small.Key || big.Match != small.Match {
				continue
			}
			if !dominates(big.Groups, small.Groups) {
				continue
			}
			if big.Multiplier <= small.Multiplier {
				t.Errorf("%s pays x%d and contains %s at x%d, so nobody would ever build it",
					big.Key, big.Multiplier, small.Key, small.Multiplier)
			}
		}
	}
}

// dominates reports whether every group of `small` can be matched to a distinct group of `big` that
// is at least as large — which is what it means for a hand to contain another on the same axis.
//
// Greedy on both sorted descending is exact: the largest demand is best served by the largest
// supply, so if that pairing fails no other succeeds.
func dominates(big, small []int) bool {
	if len(small) > len(big) {
		return false
	}
	b := append([]int(nil), big...)
	sm := append([]int(nil), small...)
	sort.Sort(sort.Reverse(sort.IntSlice(b)))
	sort.Sort(sort.Reverse(sort.IntSlice(sm)))
	for i, want := range sm {
		if b[i] < want {
			return false
		}
	}
	return true
}

// Every of-a-kind rung exists on every of-a-kind axis, and no axis is missing one. A rung built on
// two axes and not the third would be a hole a player could fall into without ever being told it
// was there.
//
// **The cost axis is deliberately outside this** *(owner's call, 2026-09-05)*. Concept, form and
// element carry the whole six-rung ladder because the same shape means something on each of them.
// Cost carries two bespoke rungs instead — Rising Attack, three cards of three different costs, and
// Weaponmaster, three cards of one cost in three different forms — because the of-a-kind rungs on
// that axis would be redundant: a cost Three of a Kind is a rung with no idea in it, sitting under
// Weaponmaster and above nothing. So this walks the three axes that carry the ladder, and the check
// underneath it is what holds the cost axis to the catalogue.
func TestEveryRungExistsOnEveryOfAKindAxis(t *testing.T) {
	for _, axis := range []Axis{AxisConcept, AxisForm, AxisElement} {
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

// **A hand's key names the axis it counts on**, which is what makes the of-a-kind walk above a
// check rather than a coincidence, and what keeps a hand added to the cost axis from being filed
// under a name that says element.
func TestEveryHandsKeyNamesItsAxis(t *testing.T) {
	for _, h := range Hands() {
		if h.Key == highCardKey {
			// The fallback belongs to no axis: it is what a turn forms when it built nothing.
			continue
		}
		if !strings.HasPrefix(h.Key, h.Match.String()+"-") {
			t.Errorf("hand %q counts on %s, so its key should begin %q", h.Key, h.Match, h.Match.String()+"-")
		}
	}
}

// A card hand pays more than the form hand inside it, which is what makes aiming at the narrower
// one worth the extra difficulty. **A concept fixes a form**, so any set of cards sharing a concept
// also shares a form — if the form rung paid the same or better, nobody would ever have reason to
// build the card one.
//
// **Element is not in this and never was entitled to be** *(owner's call, 2026-09-05)*. A concept
// does *not* fix an element — a fire Jab and an ice Jab are one concept and two colours — so there
// is no containment between the two axes and no reason the card rung must outpay the elemental one.
// The measurements say it often should not: an Elemental Two Pair is rarer than a Form Two Pair and
// is priced above it. This used to require concept > element at every rung, which was an assumption
// wearing a proof's clothes.
func TestACardHandPaysMoreThanTheFormHandInsideIt(t *testing.T) {
	for _, rung := range []string{"pair", "two-pair", "three-of-a-kind", "full-house", "four-of-a-kind",
		"five-of-a-kind"} {
		card, ok := handByKey("concept-" + rung)
		if !ok {
			t.Fatalf("the catalogue has no concept-%s", rung)
		}
		form, ok := handByKey("form-" + rung)
		if !ok {
			t.Fatalf("the catalogue has no form-%s", rung)
		}
		if form.Multiplier >= card.Multiplier {
			t.Errorf("form-%s pays x%d, matching or beating the concept-%s it is contained by at x%d",
				rung, form.Multiplier, rung, card.Multiplier)
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
