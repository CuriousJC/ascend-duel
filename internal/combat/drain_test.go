package combat

import "testing"

// The drain verb: a relic that turns part of a landed blow back into its wearer's life.
//
// **The two halves that can break independently** are the arithmetic — a share of the figure that
// actually landed, not of the figure before the target's body shaped it — and the gating, which is
// the whole of what keeps a drain from paying for a blow the player never saw.

// drainRelic is a relic taking pct of every blow, with an optional predicate.
func drainRelic(t *testing.T, key string, pct int, cond RelicCondition) RelicID {
	t.Helper()

	return relic(t, key, RelicRule{
		When: MomentAttackLands,
		If:   cond,
		Then: []RelicEffect{{Do: DoDrainDamage, Amount: pct}},
	})
}

// drained is the life one side took back over a whole round.
func drained(events []Event, side Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindDrained && e.Side == side {
			total += e.Amount
		}
	}
	return total
}

// damageTo is what a side's blow actually landed, which is the figure a drain is a share of.
func damageTo(events []Event, target Side) int {
	for _, e := range events {
		if e.Kind == KindDamage && e.Target == target {
			return e.Amount
		}
	}
	return 0
}

func TestADrainRelicGivesBackAShareOfTheBlowThatLanded(t *testing.T) {
	id := drainRelic(t, "drain.share", 30, RelicCondition{})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 50
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Ice)}, nil, 1)

	// **A share of what the target took**, read off the damage event rather than recomputed here:
	// a test that did its own arithmetic would pass while the blow and the drain drifted apart.
	want := damageTo(events, SideB) * 30 / 100
	if want <= 0 {
		t.Fatalf("the blow landed %d, so this test is about nothing", damageTo(events, SideB))
	}
	if got := drained(events, SideA); got != want {
		t.Errorf("a 30%% drain on a blow of %d gave back %d, wanted %d",
			damageTo(events, SideB), got, want)
	}
	if after.CurrentLife != 50+want {
		t.Errorf("life went 50 to %d, wanted %d", after.CurrentLife, 50+want)
	}
}

func TestADrainFiresOncePerBlowHoweverManyCardsMatched(t *testing.T) {
	// **The status rule, not the growth rule.** The share comes out of one figure, so a hand of
	// four fire cards is one drain — a drain per card would pay four shares of the whole blow.
	id := drainRelic(t, "drain.once", 50, RelicCondition{Element: Fire, HasElement: true})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	beats := 0
	for _, e := range events {
		if e.Kind == KindDrained {
			beats++
		}
	}
	if beats != 1 {
		t.Errorf("two matching cards produced %d drains, wanted 1", beats)
	}
}

func TestADrainWhosePredicateMatchesNothingPaysNothing(t *testing.T) {
	id := drainRelic(t, "drain.miss", 50, RelicCondition{Element: Fire, HasElement: true})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, []Card{Of(Bash, Ice), Of(Bash, Ice)}, nil, 1)

	if got := drained(events, SideA); got != 0 {
		t.Errorf("an ice hand paid a fire drain %d", got)
	}
	if after.CurrentLife != 10 {
		t.Errorf("life moved to %d on a relic that should not have fired", after.CurrentLife)
	}
}

func TestADrainOnFullLifeWritesNoBeat(t *testing.T) {
	// **Nothing in the game heals above full**, so the relic did not fire — and a beat carrying a
	// zero out of the ring would say it did. Same rule the heal rider is under.
	id := drainRelic(t, "drain.full", 50, RelicCondition{})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	for _, e := range events {
		if e.Kind == KindDrained {
			t.Fatalf("a duelist at full life drained %d", e.Amount)
		}
	}
	if after.CurrentLife != after.MaxLife {
		t.Errorf("a drain at full life left %d of %d", after.CurrentLife, after.MaxLife)
	}
}

func TestABlockedBlowDrainsNothing(t *testing.T) {
	// **A shield eats the blow whole**, so there is no figure to take a share of. This is the gate
	// that stops a drain paying for an attack the player watched come to nothing.
	id := drainRelic(t, "drain.blocked", 50, RelicCondition{})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)
	b.Shields = 1

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	if got := drained(events, SideA); got != 0 {
		t.Errorf("a blocked blow drained %d", got)
	}
	if after.CurrentLife != 10 {
		t.Errorf("life moved to %d on a blow that was eaten", after.CurrentLife)
	}
}

func TestTwoDrainRelicsAddRatherThanCompound(t *testing.T) {
	first := drainRelic(t, "drain.add1", 20, RelicCondition{})
	second := drainRelic(t, "drain.add2", 30, RelicCondition{})

	a := duelist(10, 3, 200).Wearing(WornRelic{Relic: first}).Wearing(WornRelic{Relic: second})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	dmg := damageTo(events, SideB)
	want := dmg*20/100 + dmg*30/100
	if got := drained(events, SideA); got != want {
		t.Errorf("20%% and 30%% on a blow of %d gave back %d, wanted %d", dmg, got, want)
	}
}

func TestADrainIsRefusedAtAnyOtherMoment(t *testing.T) {
	_, err := RegisterRelic("relictest.drain.wrongmoment", "wrong moment", []RelicRule{{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoDrainDamage, Amount: 30}},
	}})
	if err == nil {
		t.Fatal("a drain at fight-start registered, and it would never have fired")
	}
}
