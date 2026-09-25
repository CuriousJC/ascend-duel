package combat

import "testing"

// The drain verb: a relic that turns part of every landed hit back into its wearer's life.
//
// **The two halves that can break independently** are the arithmetic — a share of the figure that
// actually landed, hit by hit, not of the figure before the target's body shaped it — and the
// gating, which is the whole of what keeps a drain from paying for a hit the player never saw.

// drainRelic is a relic taking pct of every hit, with an optional predicate.
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

// hitsOn is every figure a side's hits actually landed, each of which a drain is a share of.
func hitsOn(events []Event, target Side) []int {
	var out []int
	for _, e := range events {
		if e.Kind == KindDamage && e.Target == target {
			out = append(out, e.Amount)
		}
	}
	return out
}

// shareOf is pct of every hit, rounded one hit at a time.
func shareOf(hits []int, pct int) int {
	total := 0
	for _, h := range hits {
		total += h * pct / 100
	}
	return total
}

func TestADrainRelicGivesBackAShareOfEveryHitThatLanded(t *testing.T) {
	id := drainRelic(t, "drain.share", 30, RelicCondition{})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 50
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Ice)}, nil, 1)

	// **A share of what the target took**, read off the damage events rather than recomputed here:
	// a test that did its own arithmetic would pass while the hits and the drain drifted apart.
	hits := hitsOn(events, SideB)
	want := shareOf(hits, 30)
	if want <= 0 {
		t.Fatalf("the hits landed %v, so this test is about nothing", hits)
	}
	if got := drained(events, SideA); got != want {
		t.Errorf("a 30%% drain on hits of %v gave back %d, wanted %d", hits, got, want)
	}
	if after.CurrentLife != 50+want {
		t.Errorf("life went 50 to %d, wanted %d", after.CurrentLife, 50+want)
	}
}

func TestADrainFiresOncePerMatchingHit(t *testing.T) {
	// **Each hit is its own figure**, so a drain takes its share of every hit whose card it matches
	// and of nothing else — two fire hits and an ice hit is two drains.
	id := drainRelic(t, "drain.perhit", 50, RelicCondition{Element: Fire, HasElement: true})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Jab, Ice), Of(Bash, Fire)}, nil, 1)

	var on []int
	for _, e := range events {
		if e.Kind == KindDrained {
			on = append(on, e.Hit)
		}
	}
	if len(on) != 2 || on[0] != 0 || on[1] != 2 {
		t.Errorf("drains came off hits %v, wanted [0 2] — the two fire hits", on)
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

func TestABlockedHitDrainsNothing(t *testing.T) {
	// **A shield eats a hit whole**, so there is no figure to take a share of. This is the gate
	// that stops a drain paying for an attack the player watched come to nothing.
	id := drainRelic(t, "drain.blocked", 50, RelicCondition{})

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)
	b.Shields = 2

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	if got := drained(events, SideA); got != 0 {
		t.Errorf("two blocked hits drained %d", got)
	}
	if after.CurrentLife != 10 {
		t.Errorf("life moved to %d on hits that were eaten", after.CurrentLife)
	}
}

func TestTwoDrainRelicsAddRatherThanCompound(t *testing.T) {
	first := drainRelic(t, "drain.add1", 20, RelicCondition{})
	second := drainRelic(t, "drain.add2", 30, RelicCondition{})

	a := duelist(10, 3, 200).Wearing(WornRelic{Relic: first}).Wearing(WornRelic{Relic: second})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire), Of(Bash, Fire)}, nil, 1)

	hits := hitsOn(events, SideB)
	want := shareOf(hits, 20) + shareOf(hits, 30)
	if got := drained(events, SideA); got != want {
		t.Errorf("20%% and 30%% on hits of %v gave back %d, wanted %d", hits, got, want)
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
