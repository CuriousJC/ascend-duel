package combat

import "testing"

// ridden is a card carrying one heal rider, which is the only rider that exists.
func ridden(id ConceptID, heal int) Card {
	c, ok := Plain(id).AddRider(Rider{Kind: RiderHealOnPlay, Amount: heal})
	if !ok {
		panic("a fresh card had no room for a rider")
	}
	return c
}

// healedBy is the total life a side restored across a whole round.
func healedBy(events []Event, side Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindHealed && e.Side == side {
			total += e.Amount
		}
	}
	return total
}

func TestARiddenCardHealsItsOwnerAsItIsPlayed(t *testing.T) {
	a := duelist(10, 3, 100)
	a.CurrentLife = 50
	b := duelist(0, 0, 100)

	events, after, _ := resolve(a, b, []Card{ridden(Strike, 10)}, nil, 1)

	if got := healedBy(events, SideA); got != 10 {
		t.Errorf("a rider worth 10 healed %d", got)
	}
	if after.CurrentLife != 60 {
		t.Errorf("life went %d to %d, wanted 60", 50, after.CurrentLife)
	}
}

func TestTwoRidersOnOneCardBothFire(t *testing.T) {
	// **Riders stack rather than merge** — see Card.AddRider. Two tens are twenty, and the reason
	// to hold it is that it is what keeps the card's face honest about how many parasites have
	// been spent on it.
	c := ridden(Strike, 10)
	c, ok := c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 10})
	if !ok {
		t.Fatal("a card with one rider had no room for a second")
	}

	a := duelist(10, 3, 100)
	a.CurrentLife = 50

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{c}, nil, 1)

	if got := healedBy(events, SideA); got != 20 {
		t.Errorf("two riders worth 10 healed %d, wanted 20", got)
	}
	if after.CurrentLife != 70 {
		t.Errorf("life ended at %d, wanted 70", after.CurrentLife)
	}
}

func TestAHealNeverGoesAboveFullLife(t *testing.T) {
	// **Nothing in the game heals above full**, and the event says what actually happened rather
	// than what the rider names — a figure in the log the bar cannot show would be the log lying.
	a := duelist(10, 3, 100)
	a.CurrentLife = 95

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if after.CurrentLife != 100 {
		t.Errorf("life ended at %d, wanted the cap at 100", after.CurrentLife)
	}
	if got := healedBy(events, SideA); got != 5 {
		t.Errorf("the event reported %d healed, wanted the 5 that actually landed", got)
	}
}

func TestAHealOnFullLifeIsSilent(t *testing.T) {
	// A rider that fired and changed nothing must not put a line in the feed saying life was
	// restored. The rider is still spent, because it is a property of the card rather than a charge.
	a := duelist(10, 3, 100)

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if after.CurrentLife != 100 {
		t.Errorf("life moved to %d on a full-life heal", after.CurrentLife)
	}
	for _, e := range events {
		if e.Kind == KindHealed {
			t.Fatalf("a heal that restored nothing wrote an event: %+v", e)
		}
	}
}

func TestAChilledCardHealsNothing(t *testing.T) {
	// **A card a chill ate was never played**, which is the whole reason riders fire after the
	// chill and before the blow. Getting this backwards would make a rider on the front card of a
	// turn immune to the one thing that can delete it.
	a := duelist(10, 3, 100)
	a.CurrentLife = 50
	a.Statuses[statusOf(Ice)] = Status{Amount: 1, Rounds: 2}

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if got := healedBy(events, SideA); got != 0 {
		t.Errorf("a chilled card healed %d", got)
	}
	if after.CurrentLife != 50 {
		t.Errorf("life moved to %d on a turn nothing was played in", after.CurrentLife)
	}
}

func TestAnUnriddenCardIsTheZeroValue(t *testing.T) {
	// Nothing may *require* a rider, and the common case has to stay the plain literal every test
	// in this package writes.
	c := Plain(Strike)
	if c.RiderCount() != 0 || c.HealOnPlay() != 0 || len(c.RiderList()) != 0 {
		t.Errorf("a plain card reported riders: %+v", c)
	}
}

func TestACardTakesNoMoreRidersThanTheFaceCanShow(t *testing.T) {
	// MaxCardRiders is a layout number as much as a rules one — a card whose face cannot say what
	// it carries is the failure the alteration mechanic exists to avoid — so the refusal is here
	// rather than a silent drop.
	c := Plain(Strike)
	for i := 0; i < MaxCardRiders; i++ {
		var ok bool
		if c, ok = c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 1}); !ok {
			t.Fatalf("rider %d of %d was refused", i+1, MaxCardRiders)
		}
	}
	if _, ok := c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 1}); ok {
		t.Errorf("a card took more than %d riders", MaxCardRiders)
	}
}

func TestEveryRiderKindHasANameThatParsesBack(t *testing.T) {
	// The snapshot writes a rider's *name*, never its ordinal, so a kind whose name does not round
	// trip is a save file that cannot be resumed. See profile.RiderSnapshot.
	for _, k := range RiderKinds() {
		got, ok := ParseRiderKind(k.String())
		if !ok || got != k {
			t.Errorf("rider %d spells itself %q, which parses back as %d/%v", k, k.String(), got, ok)
		}
	}
	if _, ok := ParseRiderKind("no-such-rider"); ok {
		t.Error("an unknown rider name resolved to something")
	}
}

func TestARiderDoesNotStopACardBeingComparable(t *testing.T) {
	// The screen's face cache and TestRoundIsDeterministic both compare cards by value, which is
	// why Riders is a fixed array. A slice here would not compile at all; this is what says so out
	// loud, so the field is not "tidied up" into one later.
	if ridden(Strike, 10) != ridden(Strike, 10) {
		t.Error("two identically ridden cards did not compare equal")
	}
	if ridden(Strike, 10) == ridden(Strike, 20) {
		t.Error("two differently ridden cards compared equal")
	}
}
