package combat

import "testing"

// slots is a turn as the matcher reads one: one side's cards, in the order they were queued.
func slots(cards []Card) []Slot { return ResolutionOrder(cards, nil) }

// wild is a card carrying the wildcard rider, which is the whole of what these tests set up.
func wild(id ConceptID, e Element) Card {
	return Of(id, e).SetRider(Rider{Kind: RiderWildElement})
}

// **A wildcard tops an elemental group up.** Three fire attacks and a wild ice one are a Four of a
// Kind on the element axis, which is the entire mechanic and the entire balance objection to it.
func TestAWildcardCompletesAnElementalGroup(t *testing.T) {
	want, ok := handByKey("element-four-of-a-kind")
	if !ok {
		t.Skip("the catalogue has no element four of a kind")
	}

	turn := []Card{Of(Jab, Fire), Of(Cut, Fire), Of(Bash, Fire), wild(Slice, Ice)}
	cards, got, _, formed := matchHand(slots(turn), handTable)
	if !formed {
		t.Fatal("three fire cards and a wildcard formed no hand at all")
	}
	if got.ID != want.ID {
		t.Fatalf("formed %q, want %q", got.Name, want.Name)
	}
	if len(cards) != 4 {
		t.Fatalf("the hand is %d cards, want all four", len(cards))
	}
}

// **The card keeps its own element.** A wild ice card is still ice for everything that is not the
// matcher — which is what decides whether it lands a chill.
func TestAWildcardStillCarriesItsOwnElement(t *testing.T) {
	c := wild(Slice, Ice)
	if c.Element != Ice {
		t.Fatalf("a wild ice card reports element %v, want Ice", c.Element)
	}
	if v, counts := MatchValue(c, AxisElement); !counts || v != int(Ice) {
		t.Fatalf("MatchValue reports (%d, %v), want ice and counting — a wildcard is read by the "+
			"matcher, not by the matcher's reading of one card", v, counts)
	}
}

// **Element only.** A wildcard that widened the concept or form axes would collapse the ladder,
// so the axis is asked for rather than assumed.
func TestAWildcardIsElementOnly(t *testing.T) {
	c := wild(Slice, Ice)
	for _, a := range AllAxes {
		got := c.Wild(a)
		want := a == AxisElement
		if got != want {
			t.Errorf("Wild(%v) is %v, want %v", a, got, want)
		}
	}
}

// **It never seeds a group it could have joined.** Two fire cards, one ice card and one wildcard
// make a fire three, not a fire pair beside an ice pair — the wildcard goes where the most of one
// element already is.
func TestAWildcardJoinsTheBiggestGroup(t *testing.T) {
	trips, ok := handByKey("element-three-of-a-kind")
	if !ok {
		t.Skip("the catalogue has no element three of a kind")
	}

	turn := []Card{Of(Jab, Fire), Of(Cut, Fire), Of(Bash, Ice), wild(Slice, Ice)}
	_, got, _, formed := matchHand(slots(turn), handTable)
	if !formed {
		t.Fatal("no hand formed")
	}
	if got.Multiplier < trips.Multiplier {
		t.Fatalf("formed %q at %d%%, want at least the three of a kind's %d%% — the wildcard "+
			"joined the smaller group", got.Name, got.Multiplier, trips.Multiplier)
	}
}

// **A turn of nothing but wildcards is a group.** They agree with each other, so the matcher
// refusing them would be saying that cards that match everything match nothing.
func TestATurnOfOnlyWildcardsStillForms(t *testing.T) {
	pair, ok := handByKey("pair")
	if !ok {
		t.Fatal("the catalogue has no pair")
	}

	turn := []Card{wild(Jab, Fire), wild(Cut, Ice)}
	cards, got, _, formed := matchHand(slots(turn), handTable)
	if !formed {
		t.Fatal("two wildcards formed no hand")
	}
	if got.Multiplier < pair.Multiplier {
		t.Fatalf("two wildcards formed %q at %d%%, want at least a pair", got.Name, got.Multiplier)
	}
	if len(cards) < 2 {
		t.Fatalf("the hand is %d cards, want both", len(cards))
	}
}

// **One wildcard is spent once.** Two groups both wanting it cannot both have it, which is what
// keeps a Full House out of reach of a turn that has not got the cards for one.
func TestAWildcardIsSpentOnce(t *testing.T) {
	full, ok := handByKey("element-full-house")
	if !ok {
		t.Skip("the catalogue has no element full house")
	}

	// Two fire, one ice, one earth and one wildcard — five cards, so the count is not what stops
	// it. A [3,2] can reach the three only by spending the wildcard on fire, which leaves the
	// lone ice and the lone earth with nothing to top either of them up to two.
	turn := []Card{
		Of(Jab, Fire), Of(Cut, Fire),
		Of(Bash, Ice), Of(Smash, Earth),
		wild(Slice, Lightning),
	}
	if _, _, ok := matchCountOf(slots(turn), full.On(AxisElement)); !ok {
		return // correct: the wildcard cannot fill both groups
	}
	t.Fatal("one wildcard filled two groups of a full house")
}

// **The rider carries no amount, and the vocabulary says so.** It is the one kind whose Amount is
// meaningless, and internal/session refuses a value-less rider parasite for every other kind.
func TestTheWildcardRiderCarriesNoAmount(t *testing.T) {
	if RiderWildElement.CarriesAmount() {
		t.Error("the wildcard rider claims to carry an amount")
	}
	for _, k := range RiderKinds() {
		if k == RiderWildElement {
			continue
		}
		if !k.CarriesAmount() {
			t.Errorf("%v claims to carry no amount, and every rider but the wildcard is a "+
				"kind plus a figure", k)
		}
	}
}

// **It is in the vocabulary both ways round**, so a parasite record can name it and a run
// snapshot can write it down.
func TestTheWildcardRiderRoundTripsItsName(t *testing.T) {
	got, ok := ParseRiderKind(RiderWildElement.String())
	if !ok || got != RiderWildElement {
		t.Fatalf("ParseRiderKind(%q) is (%v, %v), want the wildcard",
			RiderWildElement.String(), got, ok)
	}
}
