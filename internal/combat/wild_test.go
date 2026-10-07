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
		t.Skip("the catalog has no element four of a kind")
	}

	turn := []Card{Of(Jab, Fire), Of(Cut, Fire), Of(Thump, Fire), wild(Slice, Ice)}
	cards, got, _, _, formed := matchHand(slots(turn), handTable)
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
// matcher — which is what a relic's element predicate reads.
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
		t.Skip("the catalog has no element three of a kind")
	}

	turn := []Card{Of(Jab, Fire), Of(Cut, Fire), Of(Thump, Ice), wild(Slice, Ice)}
	_, got, _, _, formed := matchHand(slots(turn), handTable)
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
		t.Fatal("the catalog has no pair")
	}

	turn := []Card{wild(Jab, Fire), wild(Cut, Ice)}
	cards, got, _, _, formed := matchHand(slots(turn), handTable)
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
		t.Skip("the catalog has no element full house")
	}

	// Two fire, one ice, one earth and one wildcard — five cards, so the count is not what stops
	// it. A [3,2] can reach the three only by spending the wildcard on fire, which leaves the
	// lone ice and the lone earth with nothing to top either of them up to two.
	turn := []Card{
		Of(Jab, Fire), Of(Cut, Fire),
		Of(Thump, Ice), Of(Smash, Earth),
		wild(Slice, Lightning),
	}
	if _, _, ok := matchCountOf(slots(turn), full.On(AxisElement)); !ok {
		return // correct: the wildcard cannot fill both groups
	}
	t.Fatal("one wildcard filled two groups of a full house")
}

// **The wildcard riders carry no amount, and the vocabulary says so.** They are the kinds whose
// Amount is meaningless, and internal/session refuses a value-less rider rune for every other kind.
func TestTheWildcardRidersCarryNoAmount(t *testing.T) {
	for _, k := range RiderKinds() {
		wildcard := WildAxisOf(k) >= 0
		if k.CarriesAmount() == wildcard {
			t.Errorf("%v: CarriesAmount is %v, and every rider but the wildcards is a kind plus a "+
				"figure", k, k.CarriesAmount())
		}
	}
	for _, k := range []RiderKind{RiderWildElement, RiderWildForm} {
		if WildAxisOf(k) < 0 {
			t.Errorf("%v is not in the wildcard table", k)
		}
	}
}

// **It is in the vocabulary both ways round**, so a rune record can name it and a run
// snapshot can write it down.
func TestTheWildcardRiderRoundTripsItsName(t *testing.T) {
	for _, k := range []RiderKind{RiderWildElement, RiderWildForm} {
		got, ok := ParseRiderKind(k.String())
		if !ok || got != k {
			t.Fatalf("ParseRiderKind(%q) is (%v, %v), want %v", k.String(), got, ok, k)
		}
	}
}

// versatile is a card carrying the form wildcard.
func versatile(id ConceptID, e Element) Card {
	return Of(id, e).SetRider(Rider{Kind: RiderWildForm})
}

// **A form wildcard tops a form group up.** Two slashes and a versatile stab are a form Three of a
// Kind — the element wildcard's mechanic on the other axis.
func TestAVersatileCardCompletesAFormGroup(t *testing.T) {
	want, ok := handByKey("form-three-of-a-kind")
	if !ok {
		t.Skip("the catalog has no form three of a kind")
	}
	turn := []Card{Of(Cut, Fire), Of(Slice, Ice), versatile(Jab, Earth)}
	cards, _, ok2 := matchCountOf(slots(turn), want.On(AxisForm))
	if !ok2 || len(cards) != 3 {
		t.Fatalf("two slashes and a versatile stab made (%v, %v), want all three", cards, ok2)
	}
}

// **Attack forms only.** A versatile attack card cannot make up a defend group — defend is not an
// attack form — and a versatile Brace still counts as the defend it is.
func TestAVersatileCardJoinsOnlyAttackFormsAndItsOwn(t *testing.T) {
	pair, ok := handByKey("pair")
	if !ok {
		t.Fatal("the catalog has no pair")
	}
	on := pair.On(AxisForm)

	if _, _, ok := matchCountOf(slots([]Card{Of(Brace, Fire), versatile(Jab, Ice)}), on); ok {
		t.Error("a versatile Jab made up a defend pair")
	}
	if _, _, ok := matchCountOf(slots([]Card{Of(Brace, Fire), versatile(Block, Ice)}), on); !ok {
		t.Error("a versatile Block no longer counts as defend")
	}
	if _, _, ok := matchCountOf(slots([]Card{Of(Thump, Fire), versatile(Block, Ice)}), on); !ok {
		t.Error("a versatile Block does not count as crush")
	}
}

// **Form only.** The versatile card widens the form axis and no other, and it is not an element
// wildcard.
func TestAVersatileCardIsFormOnly(t *testing.T) {
	c := versatile(Slice, Ice)
	for _, a := range AllAxes {
		if got, want := c.Wild(a), a == AxisForm; got != want {
			t.Errorf("Wild(%v) is %v, want %v", a, got, want)
		}
	}
}
