package combat

import "testing"

// The round limit. These are about the *clock* rather than about a duel: a fight that ends on its
// own is the ordinary case and the interesting one is the fight that does not.

// countKind, the helper these lean on, is in status_test.go.

// A duelist on a clock fights its rounds and then dies on the last one, whatever is left of them.
//
// **The four survivable rounds are checked as well as the fatal one**, because a clock that fired
// early and a clock that never fired look identical from a test that only ever resolves round five.
func TestTheClockKillsOnTheLastRoundAndNotBefore(t *testing.T) {
	a := duelist(1, 5, 300)
	a.RoundLimit = DefaultRoundLimit
	b := duelist(1, 5, 300)

	for round := 1; round < DefaultRoundLimit; round++ {
		events, after, _ := resolve(a, b, nil, nil, round)
		if n := countKind(events, KindTimeUp); n != 0 {
			t.Fatalf("round %d of %d raised %d time-up events, want none",
				round, DefaultRoundLimit, n)
		}
		if !after.Alive() {
			t.Fatalf("round %d of %d killed a duelist nothing attacked", round, DefaultRoundLimit)
		}
	}

	events, after, _ := resolve(a, b, nil, nil, DefaultRoundLimit)
	if n := countKind(events, KindTimeUp); n != 1 {
		t.Fatalf("the last round raised %d time-up events, want exactly one", n)
	}
	if after.Alive() {
		t.Errorf("the duelist survived the clock on %d life", after.CurrentLife)
	}
	if after.CurrentLife != 0 {
		t.Errorf("the clock left %d life, want none of it", after.CurrentLife)
	}
}

// The event says what it took and the fall comes after it. Both halves are what the feed reads:
// a figure of zero would narrate a duelist dying to nothing, and a fall announced first would read
// as the death having some other cause.
func TestTheClockAnnouncesWhatItTookAndThenTheFall(t *testing.T) {
	a := duelist(1, 5, 173)
	a.RoundLimit = 1
	b := duelist(1, 5, 300)

	events, _, _ := resolve(a, b, nil, nil, 1)

	timeUp := -1
	for i, e := range events {
		if e.Kind == KindTimeUp {
			timeUp = i
			break
		}
	}
	if timeUp < 0 {
		t.Fatal("a one-round clock raised no time-up event")
	}
	if got := events[timeUp].Amount; got != 173 {
		t.Errorf("the clock reports taking %d, want the whole 173 the duelist had", got)
	}
	if got := events[timeUp].Life; got != 0 {
		t.Errorf("the clock leaves %d life on the event, want 0", got)
	}
	if events[timeUp].Side != SideA || events[timeUp].Target != SideA {
		t.Errorf("the clock names side %v against target %v, want both to be the duelist it killed",
			events[timeUp].Side, events[timeUp].Target)
	}
	if timeUp+1 >= len(events) || events[timeUp+1].Kind != KindDefeated {
		t.Error("the fall does not follow the clock immediately")
	}
}

// **Zero is no clock**, which is what every enemy and every bare literal in this package carries.
// A default of five here would have put the whole existing suite on a timer.
func TestADuelistWithNoLimitIsNeverTimedOut(t *testing.T) {
	a := duelist(1, 5, 300)
	b := duelist(1, 5, 300)

	for round := 1; round <= 20; round++ {
		events, after, _ := resolve(a, b, nil, nil, round)
		if n := countKind(events, KindTimeUp); n != 0 {
			t.Fatalf("round %d raised a time-up event on a duelist with no limit", round)
		}
		a = after
	}
}

// A duelist who died on the last round died to whatever killed them, not to the clock. Otherwise a
// fight finishing on round five would announce a second death over a body — the same failure the
// burn tick documents.
func TestTheClockDoesNotFireOverABody(t *testing.T) {
	a := duelist(1, 5, 1)
	a.RoundLimit = 1
	b := duelist(500, 5, 300)

	events, after, _ := resolve(a, b, nil, PlainCards(Strike), 1)

	if after.Alive() {
		t.Fatal("the test's blow did not kill, so it proves nothing about the clock")
	}
	if n := countKind(events, KindTimeUp); n != 0 {
		t.Errorf("the clock fired %d times over a duelist already dead", n)
	}
	if n := countKind(events, KindDefeated); n != 1 {
		t.Errorf("%d falls announced over one death", n)
	}
}

// The clock is read after every other way a round can end, so an enemy killed on the final round
// is a won fight rather than a mutual one. This is the case a player actually plays for.
func TestKillingOnTheLastRoundIsAWin(t *testing.T) {
	a := duelist(500, 5, 300)
	a.RoundLimit = DefaultRoundLimit
	b := duelist(1, 5, 1)

	_, after, enemy := resolve(a, b, PlainCards(Strike), nil, DefaultRoundLimit)

	if enemy.Alive() {
		t.Fatal("the test's blow did not kill, so it proves nothing about the ordering")
	}
	if !after.Alive() {
		t.Error("the clock killed a duelist who had just won the fight")
	}
}

// A limit already passed still fires, which is what `>=` buys: something that lowers the limit
// mid-run cannot be stepped over by a fight that was already beyond it.
func TestAPassedLimitStillFires(t *testing.T) {
	a := duelist(1, 5, 300)
	a.RoundLimit = 2
	b := duelist(1, 5, 300)

	events, after, _ := resolve(a, b, nil, nil, 9)
	if n := countKind(events, KindTimeUp); n != 1 {
		t.Fatalf("round 9 against a limit of 2 raised %d time-up events, want one", n)
	}
	if after.Alive() {
		t.Error("a duelist past their limit survived the round")
	}
}
