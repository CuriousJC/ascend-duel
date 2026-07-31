package combat

import "testing"

// duelist builds a full-health duelist for tests.
func duelist(str, spd, life int) Duelist {
	return Duelist{Str: str, Spd: spd, MaxLife: life, CurrentLife: life}
}

// firstDamage returns the first damage event dealt by the given side.
func firstDamage(t *testing.T, events []Event, by Side) Event {
	t.Helper()
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == by {
			return e
		}
	}
	t.Fatalf("no damage event from side %s", by)
	return Event{}
}

// actionOrder returns the sides in the order they acted.
func actionOrder(events []Event) []Side {
	var order []Side
	for _, e := range events {
		if e.Kind == KindAction {
			order = append(order, e.Side)
		}
	}
	return order
}

// sidesEqual compares two action orders.
func sidesEqual(got, want []Side) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSideAActsBeforeSideB(t *testing.T) {
	// The player maps to side A and always swings first, however fast the enemy is.
	a := duelist(10, 1, 200)
	b := duelist(10, 500, 200)

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike}, []ActionKind{Strike}, 1)

	if order := actionOrder(events); !sidesEqual(order, []Side{SideA, SideB}) {
		t.Errorf("action order = %v, want [A B] even with B far faster", order)
	}
}

func TestActionsAlternateBetweenSides(t *testing.T) {
	// The queues interleave one action each rather than resolving as two volleys. This
	// is the ordering the resolution pane shows the player, and the thing later effects
	// are meant to be able to rearrange.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Strike, Strike}, []ActionKind{Quick, Quick}, 1)

	want := []Side{SideA, SideB, SideA, SideB}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v", order, want)
	}
}

func TestTheLongerQueueActsAloneOnceTheOtherRunsOut(t *testing.T) {
	// A faster duelist buys more actions, and the tail of the round is where that
	// advantage is actually spent.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Quick, Quick, Quick}, []ActionKind{Quick}, 1)

	want := []Side{SideA, SideB, SideA, SideA}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v", order, want)
	}
}

func TestStrikeDealsStrengthAsDamage(t *testing.T) {
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	events, _, bAfter := ResolveRound(a, b, []ActionKind{Strike}, nil, 1)

	got := firstDamage(t, events, SideA)
	if got.Amount != 10 {
		t.Errorf("Strike damage = %d, want 10 (attacker Str)", got.Amount)
	}
	if bAfter.CurrentLife != 90 {
		t.Errorf("target life after Strike = %d, want 90", bAfter.CurrentLife)
	}
}

func TestHeavyHitsTwiceAsHardAsStrike(t *testing.T) {
	a := duelist(10, 10, 500)
	b := duelist(1, 10, 500)

	strikeLog, _, _ := ResolveRound(a, b, []ActionKind{Strike}, nil, 1)
	heavyLog, _, _ := ResolveRound(a, b, []ActionKind{Heavy}, nil, 1)

	strike := firstDamage(t, strikeLog, SideA)
	heavy := firstDamage(t, heavyLog, SideA)

	if heavy.Amount != strike.Amount*2 {
		t.Errorf("Heavy = %d, Strike = %d; want Heavy to be double", heavy.Amount, strike.Amount)
	}
}

func TestQuickHitsForHalfButNeverZero(t *testing.T) {
	// Str 1 halves to 0 under integer division; a hit that does nothing would make
	// low-Strength duels unresolvable, so Quick floors at 1.
	events, _, _ := ResolveRound(duelist(1, 10, 100), duelist(1, 10, 100), []ActionKind{Quick}, nil, 1)

	if got := firstDamage(t, events, SideA); got.Amount != 1 {
		t.Errorf("Quick damage at Str 1 = %d, want 1", got.Amount)
	}
}

func TestPlayerGuardCoversTheEnemyReplyInTheSameRound(t *testing.T) {
	// A acts first in the alternation, so a Guard placed now is up for the reply that
	// immediately follows it.
	a := duelist(10, 10, 200)
	b := duelist(10, 10, 200)

	open, _, _ := ResolveRound(a, b, []ActionKind{Strike}, []ActionKind{Strike}, 1)
	guarded, _, _ := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Strike}, 1)

	openHit := firstDamage(t, open, SideB)
	guardedHit := firstDamage(t, guarded, SideB)

	if guardedHit.Amount != openHit.Amount/2 {
		t.Errorf("damage through guard = %d, unguarded = %d; want half",
			guardedHit.Amount, openHit.Amount)
	}
}

func TestEnemyGuardCarriesToTheNextRound(t *testing.T) {
	// A guard raised on the last action of a round is still up when the next round
	// starts — it lasts until its owner acts again, and round boundaries are not
	// actions. Without that, a Guard queued last would be dead weight.
	a := duelist(10, 10, 200)
	b := duelist(10, 10, 200)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Strike}, []ActionKind{Guard}, 1)
	if !b1.Guarded {
		t.Fatal("side B guard did not survive the round it was raised in")
	}

	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Strike}, nil, 2)
	hit := firstDamage(t, round2, SideA)

	if hit.Amount != 5 {
		t.Errorf("damage into a carried guard = %d, want 5 (half of Str 10)", hit.Amount)
	}
}

func TestGuardDropsWhenItsOwnerActsAgain(t *testing.T) {
	// A's guard covers B's reply in the same round, then drops the moment A next acts.
	// The flag is still set in the returned state — it clears at the start of A's next
	// action, not at the end of the round — so the behaviour is what this asserts, not
	// the bookkeeping.
	a := duelist(10, 10, 200)
	b := duelist(10, 10, 200)

	round1, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Strike}, 1)
	if hit := firstDamage(t, round1, SideB); hit.Amount != 5 {
		t.Errorf("damage into a fresh guard = %d, want 5", hit.Amount)
	}

	// A acts in round 2, which drops the guard before B's reply lands.
	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Quick}, []ActionKind{Strike}, 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("damage after guard expired = %d, want full 10", hit.Amount)
	}
}

func TestGuardHoldsWhileItsOwnerDoesNothing(t *testing.T) {
	// The flip side of "a guard lasts until you act again": queue nothing and it stays
	// up. Deliberate rather than accidental — standing still behind a guard deals no
	// damage, so it buys time and nothing else. If that ever proves too strong, the fix
	// is an expiry rule here, not a special case at the call site.
	a := duelist(10, 10, 200)
	b := duelist(10, 10, 200)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, nil, 1)

	round2, _, _ := ResolveRound(a1, b1, nil, []ActionKind{Strike}, 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 5 {
		t.Errorf("damage into a held guard = %d, want 5 — an idle duelist keeps its guard", hit.Amount)
	}
}

func TestActionPointsFromSpeed(t *testing.T) {
	cases := []struct {
		spd, want int
	}{
		{0, 4},
		{11, 5}, // the shipped Fighter1
		{31, 7}, // the shipped Monster1
		{100, 14},
	}

	for _, c := range cases {
		if got := (Duelist{Spd: c.spd}).ActionPoints(); got != c.want {
			t.Errorf("ActionPoints at Spd %d = %d, want %d", c.spd, got, c.want)
		}
	}
}

func TestCanAffordEnforcesTheBudget(t *testing.T) {
	d := duelist(10, 11, 100) // 5 AP

	if !d.CanAfford([]ActionKind{Heavy, Quick}) { // 4 + 1
		t.Error("Heavy + Quick costs 5 and should fit a 5 AP budget")
	}
	if d.CanAfford([]ActionKind{Heavy, Strike}) { // 4 + 2
		t.Error("Heavy + Strike costs 6 and should not fit a 5 AP budget")
	}
}

func TestDefeatStopsTheRoundEarly(t *testing.T) {
	// A queues three strikes into an enemy that dies on the first. The rest must not
	// resolve, and B must never get its reply.
	a := duelist(100, 10, 100)
	b := duelist(10, 10, 10)

	events, _, bAfter := ResolveRound(a, b,
		[]ActionKind{Strike, Strike, Strike}, []ActionKind{Strike}, 1)

	if bAfter.Alive() {
		t.Fatalf("side B survived with %d life, want defeated", bAfter.CurrentLife)
	}

	damageEvents := 0
	for _, e := range events {
		if e.Kind == KindDamage {
			damageEvents++
		}
	}
	if damageEvents != 1 {
		t.Errorf("damage events = %d, want 1 — the round should stop at the kill", damageEvents)
	}

	if order := actionOrder(events); !sidesEqual(order, []Side{SideA}) {
		t.Errorf("actions = %v, want only A's — B is dead and cannot reply", order)
	}
}

func TestLifeNeverGoesNegative(t *testing.T) {
	events, _, bAfter := ResolveRound(duelist(1000, 10, 100), duelist(1, 10, 5),
		[]ActionKind{Strike}, nil, 1)

	for _, e := range events {
		if e.Life < 0 {
			t.Fatalf("event %v reported negative life %d", e.Kind, e.Life)
		}
	}
	if bAfter.CurrentLife != 0 {
		t.Errorf("life after overkill = %d, want 0", bAfter.CurrentLife)
	}
}

func TestResolveRoundDoesNotMutateItsInputs(t *testing.T) {
	// The screen holds the live combatants; ResolveRound must not drain them as a
	// side effect of building the log.
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	ResolveRound(a, b, []ActionKind{Strike}, []ActionKind{Strike}, 1)

	if a.CurrentLife != 100 || b.CurrentLife != 100 {
		t.Errorf("inputs mutated: a=%d b=%d, want both 100", a.CurrentLife, b.CurrentLife)
	}
}

func TestBothSidesGuardingDoesNotAlias(t *testing.T) {
	// resolveAction passes duelists by value; if that ever became pointer-based, a
	// mutual Guard is where an aliasing bug would surface — one side's raised guard
	// leaking onto the other.
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Guard}, 1)
	if !a1.Guarded || !b1.Guarded {
		t.Fatalf("both raised a guard; got a=%v b=%v", a1.Guarded, b1.Guarded)
	}

	// Round 2, both strike. A acts first: its own guard clears as it does, so B's later
	// hit lands in full — but B's guard is still up when A's strike arrives, so that
	// one is halved.
	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Strike}, []ActionKind{Strike}, 2)

	if hit := firstDamage(t, round2, SideA); hit.Amount != 5 {
		t.Errorf("A's hit into B's carried guard = %d, want 5", hit.Amount)
	}
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("B's hit into A's expired guard = %d, want full 10", hit.Amount)
	}
}

func TestRoundIsDeterministic(t *testing.T) {
	a := duelist(7, 13, 300)
	b := duelist(9, 17, 300)

	first, a1, b1 := ResolveRound(a, b, []ActionKind{Strike, Guard}, []ActionKind{Quick, Quick}, 1)
	second, a2, b2 := ResolveRound(a, b, []ActionKind{Strike, Guard}, []ActionKind{Quick, Quick}, 1)

	if len(first) != len(second) {
		t.Fatalf("log lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, first[i], second[i])
		}
	}
	if a1 != a2 || b1 != b2 {
		t.Error("resulting duelists differ between identical runs")
	}
}

func TestEmptyQueueIsAHarmlessRound(t *testing.T) {
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	_, aAfter, bAfter := ResolveRound(a, b, nil, nil, 1)

	if aAfter.CurrentLife != 100 || bAfter.CurrentLife != 100 {
		t.Errorf("nobody acted but life changed: a=%d b=%d", aAfter.CurrentLife, bAfter.CurrentLife)
	}
}

func TestPlanGreedyFitsTheBudget(t *testing.T) {
	for spd := 0; spd <= 100; spd += 7 {
		d := duelist(10, spd, 100)
		plan := PlanGreedy(d)

		if !d.CanAfford(plan) {
			t.Errorf("Spd %d: greedy plan costs %d, budget is %d",
				spd, CostOf(plan), d.ActionPoints())
		}
		if len(plan) == 0 {
			t.Errorf("Spd %d: greedy plan is empty", spd)
		}
	}
}

func TestPlanGreedySpendsNearlyEverything(t *testing.T) {
	// Costs are 1/2/2/4, so a greedy fill should never leave anything behind.
	for spd := 0; spd <= 100; spd += 3 {
		d := duelist(10, spd, 100)
		if left := d.ActionPoints() - CostOf(PlanGreedy(d)); left != 0 {
			t.Errorf("Spd %d: greedy plan left %d points unspent", spd, left)
		}
	}
}
