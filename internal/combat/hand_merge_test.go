package combat

import "testing"

// **The Pair is one rung read three ways** *(owner's call, 2026-09-05)*. Card Pair, Form Pair and
// Elemental Pair were three catalogue entries, three stones and three relics describing the same two
// cards; a player forming a pair does not care which axis let them, so the ladder says so once.
//
// This is the whole of what the merge has to be true for: two cards agreeing on *any* of the three
// of-a-kind axes form it, and the hand that fires is the same hand each time.
func TestThePairFormsOnWhicheverAxisAgrees(t *testing.T) {
	pair, ok := handByKey("pair")
	if !ok {
		t.Fatal("the catalogue has no pair")
	}

	for _, tc := range []struct {
		what string
		turn []Card
	}{
		// Two Bashes agree on the concept, and so on everything narrower than it.
		{"two of one card", PlainCards(Bash, Bash)},
		// A fire Bash and an ice Bash are one concept and two colours.
		{"one card in two colours", []Card{Of(Bash, Fire), Of(Bash, Ice)}},
	} {
		a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)
		events, _, _ := resolve(a, b, tc.turn, nil, 1)

		e, ok := handEventFor(events, SideA)
		if !ok {
			t.Fatalf("%s: no attack phase event", tc.what)
		}
		if e.Hand != pair.ID {
			t.Errorf("%s formed hand %d, want the Pair (%d)", tc.what, e.Hand, pair.ID)
		}
	}
}

// **The merged rung reports which axis satisfied it.** Nothing in the rules branches on that today,
// but the blow is what the feed and the screen read, and a hand that could not say how it was
// formed would be a hand nothing could ever explain.
func TestTheMergedPairReportsTheAxisThatFormedIt(t *testing.T) {
	for _, tc := range []struct {
		what  string
		cards []Card
		want  Axis
	}{
		{"two of one card", PlainCards(Bash, Bash), AxisConcept},
		// A stab and a slash, one fire and one ice, agree on neither concept nor element.
		{"two forms that match", []Card{Of(Jab, Fire), Of(Thrust, Ice)}, AxisForm},
		// Two different concepts in two different forms, agreeing only on their colour.
		{"two colours that match", []Card{Of(Jab, Fire), Of(Slice, Fire)}, AxisElement},
	} {
		turn := make([]Slot, len(tc.cards))
		for i, c := range tc.cards {
			turn[i] = Slot{Card: c, Index: i}
		}
		blow := BlowFor(turn)
		if blow.Hand.Key != "pair" {
			t.Fatalf("%s formed %q, want the pair", tc.what, blow.Hand.Key)
		}
		if blow.Hand.Match != tc.want {
			t.Errorf("%s reports the %s axis, want %s", tc.what, blow.Hand.Match, tc.want)
		}
		if len(blow.Hand.Axes) != 1 {
			t.Errorf("%s reports %v; a formed hand is read on the one axis that satisfied it",
				tc.what, blow.Hand.Axes)
		}
	}
}

// **The Pair sits at the identity and is still worth building** *(owner's call, 2026-09-05)*. The
// multiplier scales the hand's *own cards*, so two cards summed at 1x beat the one card a High Card
// lands — which is why the loader allows a multi-card rung at 100 and refuses one below it.
func TestAPairAtTheIdentityStillBeatsAHighCard(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	one, _, _ := resolve(a, b, PlainCards(Bash), nil, 1)
	two, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)

	single, ok := handEventFor(one, SideA)
	if !ok {
		t.Fatal("no attack phase event for the lone Bash")
	}
	paired, ok := handEventFor(two, SideA)
	if !ok {
		t.Fatal("no attack phase event for the two Bashes")
	}
	if paired.Amount <= single.Amount {
		t.Errorf("a pair lands %d against a high card's %d, so nobody would build it",
			paired.Amount, single.Amount)
	}
}
