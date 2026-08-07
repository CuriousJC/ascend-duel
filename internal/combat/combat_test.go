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

// damageCount is how many damage events the log holds.
func damageCount(events []Event) int {
	n := 0
	for _, e := range events {
		if e.Kind == KindDamage {
			n++
		}
	}
	return n
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

// playedActions returns the actions in the order they resolved.
func playedActions(events []Event) []ActionKind {
	var played []ActionKind
	for _, e := range events {
		if e.Kind == KindAction {
			played = append(played, e.Action)
		}
	}
	return played
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

// actionsEqual compares two played sequences.
func actionsEqual(got, want []ActionKind) bool {
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

func TestSideATakesItsWholeTurnFirst(t *testing.T) {
	// A duelist's Spd buys action points, never priority: B is 500 times faster here and
	// still does not act until A has finished. Speed is how *much* you do, and nothing
	// else — initiative was the lever that said otherwise and it no longer exists.
	a := duelist(10, 1, 500)
	b := duelist(10, 500, 500)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Strike, Strike}, []ActionKind{Strike, Strike}, 1)

	want := []Side{SideA, SideA, SideB, SideB}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v even with B far faster", order, want)
	}
}

func TestATurnResolvesInCategoryOrder(t *testing.T) {
	// Setup, then attacks, then defenses, whatever order the cards were queued in. The
	// defenses go last within a turn because the *opponent* moves next, so a defense
	// raised at the end of a turn is up when the blow arrives.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	queued := []ActionKind{Dodge, Strike, Prepare, Guard, Heavy}
	events, _, _ := ResolveRound(a, b, queued, nil, 1)

	want := []ActionKind{Prepare, Guard, Strike, Heavy, Dodge}
	if got := playedActions(events); !actionsEqual(got, want) {
		t.Errorf("played %v, want %v", got, want)
	}
}

func TestQueuedOrderSurvivesInsideACategory(t *testing.T) {
	// Reordering across categories does nothing, but reordering *within* one is the whole
	// remaining point of dragging a card along the row — sequence combos will match on it.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	first, _, _ := ResolveRound(a, b, []ActionKind{Heavy, Quick}, nil, 1)
	second, _, _ := ResolveRound(a, b, []ActionKind{Quick, Heavy}, nil, 1)

	if got := playedActions(first); !actionsEqual(got, []ActionKind{Heavy, Quick}) {
		t.Errorf("played %v, want [Heavy Quick]", got)
	}
	if got := playedActions(second); !actionsEqual(got, []ActionKind{Quick, Heavy}) {
		t.Errorf("played %v, want [Quick Heavy]", got)
	}
}

func TestResolutionOrderIsWhatResolveRoundPlays(t *testing.T) {
	// The Resolution pane draws ResolutionOrder and the engine plays it. If these two
	// ever disagree the pane is lying to the player about their own round, so pin that
	// they are the same sequence rather than trusting the shared call.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)
	aPlan := []ActionKind{Heavy, Dodge, Prepare}
	bPlan := []ActionKind{Quick, Guard}

	events, _, _ := ResolveRound(a, b, aPlan, bPlan, 1)

	want := make([]ActionKind, 0, len(aPlan)+len(bPlan))
	for _, slot := range ResolutionOrder(aPlan, bPlan) {
		want = append(want, slot.Action)
	}

	if got := playedActions(events); !actionsEqual(got, want) {
		t.Errorf("played order = %v, ResolutionOrder = %v", got, want)
	}
}

func TestResolutionOrderNeverPutsAAfterB(t *testing.T) {
	// ResolveRound expires side B's defenses at the first B slot, which is only the start
	// of B's turn if A's slots never follow it. Pin the block structure that relies on.
	order := ResolutionOrder(
		[]ActionKind{Dodge, Strike, Prepare},
		[]ActionKind{Guard, Heavy, Riposte})

	seenB := false
	for _, slot := range order {
		if slot.Side == SideB {
			seenB = true
			continue
		}
		if seenB {
			t.Fatalf("side A acts after side B has started: %v", order)
		}
	}
}

func TestSlotIndexIsThePositionInItsOwnQueue(t *testing.T) {
	// Index is where the card sits in the player's queue, not where it lands in the round.
	// Anything wanting "how far through the round are we" has to count slots instead.
	order := ResolutionOrder([]ActionKind{Dodge, Prepare}, nil)

	if len(order) != 2 {
		t.Fatalf("got %d slots, want 2", len(order))
	}
	if order[0].Action != Prepare || order[0].Index != 1 {
		t.Errorf("first slot = %v index %d, want Prepare index 1", order[0].Action, order[0].Index)
	}
	if order[1].Action != Dodge || order[1].Index != 0 {
		t.Errorf("second slot = %v index %d, want Dodge index 0", order[1].Action, order[1].Index)
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

func TestOnlyAttacksDealDamageOnTheirOwn(t *testing.T) {
	// Riposte reports a damage figure so its card can draw one, but it only ever hits back
	// at something. Played into an opponent who does nothing, it deals nothing.
	for _, a := range []ActionKind{Prepare, Guard, Dodge, Riposte} {
		events, _, bAfter := ResolveRound(duelist(10, 10, 100), duelist(10, 10, 100),
			[]ActionKind{a}, nil, 1)

		if n := damageCount(events); n != 0 {
			t.Errorf("%v dealt damage %d times unprompted", a, n)
		}
		if bAfter.CurrentLife != 100 {
			t.Errorf("%v took the target to %d, want 100", a, bAfter.CurrentLife)
		}
	}
}

func TestGuardHalvesEveryAttackInTheOpponentsTurn(t *testing.T) {
	// Guard is a setup now, and it is broad: it covers the whole of the opposing turn
	// rather than one blow. That breadth is what it costs 3 for.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	open, _, _ := ResolveRound(a, b, nil, []ActionKind{Strike, Strike, Strike}, 1)
	guarded, _, _ := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Strike, Strike, Strike}, 1)

	if damageCount(open) != 3 || damageCount(guarded) != 3 {
		t.Fatalf("expected three hits either way, got %d open and %d guarded",
			damageCount(open), damageCount(guarded))
	}

	for _, e := range guarded {
		if e.Kind == KindDamage && e.Amount != 5 {
			t.Errorf("hit through a guard = %d, want 5 (half of Str 10)", e.Amount)
		}
	}
}

func TestAGuardCoversExactlyOneOpposingTurn(t *testing.T) {
	// Raised in round 1, still up for B's round-1 turn, gone by the time A's round-2 turn
	// begins. A defense expires at the start of its owner's next turn — not at the round
	// boundary, which would leave side B's defenses protecting it from nothing.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	round1, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Strike}, 1)
	if hit := firstDamage(t, round1, SideB); hit.Amount != 5 {
		t.Errorf("round 1 hit into a fresh guard = %d, want 5", hit.Amount)
	}
	if !a1.Guarded {
		t.Fatal("A's guard should still be standing at the round boundary")
	}

	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Quick}, []ActionKind{Strike}, 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("round 2 hit after the guard expired = %d, want full 10", hit.Amount)
	}
}

func TestSideBsGuardProtectsItInTheFollowingRound(t *testing.T) {
	// The asymmetry the expiry rule exists for. B acts last, so its guard cannot cover
	// anything in the round it was raised — it has to survive the boundary and cover A's
	// next turn, or the card would be worthless in B's hands.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, nil, []ActionKind{Guard}, 1)
	if !b1.Guarded {
		t.Fatal("side B's guard did not survive the round it was raised in")
	}

	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Strike}, nil, 2)
	if hit := firstDamage(t, round2, SideA); hit.Amount != 5 {
		t.Errorf("A's hit into B's carried guard = %d, want 5", hit.Amount)
	}
}

func TestAnIdleDuelistLosesItsGuard(t *testing.T) {
	// A turn happens whether or not anything is queued into it, and a defense expires at
	// the start of its owner's turn. Standing still therefore does not bank a guard —
	// which the old until-you-act rule allowed, deliberately, and phases dissolve.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, nil, 1)

	round2, _, _ := ResolveRound(a1, b1, nil, []ActionKind{Strike}, 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("hit in round 2 = %d, want full 10 — an idle turn still expires a guard", hit.Amount)
	}
}

func TestDodgeNegatesOneAttackAndIsSpentDoingIt(t *testing.T) {
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, aAfter, _ := ResolveRound(a, b,
		[]ActionKind{Dodge}, []ActionKind{Heavy, Strike}, 1)

	if n := damageCount(events); n != 1 {
		t.Fatalf("damage events = %d, want 1 — the first attack is negated, the second lands", n)
	}
	// The Heavy is negated, so the Strike is what gets through.
	if hit := firstDamage(t, events, SideB); hit.Amount != 10 {
		t.Errorf("the attack that got past the dodge dealt %d, want 10 (Strike)", hit.Amount)
	}
	if aAfter.CurrentLife != 490 {
		t.Errorf("A ended on %d, want 490 — one Strike through one Dodge", aAfter.CurrentLife)
	}
}

func TestTwoDodgesNegateTwoAttacks(t *testing.T) {
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, aAfter, _ := ResolveRound(a, b,
		[]ActionKind{Dodge, Dodge}, []ActionKind{Heavy, Strike}, 1)

	if n := damageCount(events); n != 0 {
		t.Errorf("damage events = %d, want 0 — both attacks are dodged", n)
	}
	if aAfter.CurrentLife != 500 {
		t.Errorf("A ended on %d, want 500", aAfter.CurrentLife)
	}
}

func TestRiposteNegatesAnAttackAndHitsBack(t *testing.T) {
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, aAfter, bAfter := ResolveRound(a, b,
		[]ActionKind{Riposte}, []ActionKind{Heavy}, 1)

	if aAfter.CurrentLife != 500 {
		t.Errorf("A took %d damage through a riposte, want none", 500-aAfter.CurrentLife)
	}
	if bAfter.CurrentLife != 495 {
		t.Errorf("B ended on %d, want 495 — a riposte hits back for half a Strike", bAfter.CurrentLife)
	}
	// The counter is dealt by the defender, so its event belongs to side A.
	if hit := firstDamage(t, events, SideA); hit.Amount != 5 {
		t.Errorf("counter damage = %d, want 5", hit.Amount)
	}
}

func TestRiposteIsSpentBeforeDodge(t *testing.T) {
	// Both stop a blow completely, so spending the one that hits back first is free, and
	// it puts the counter-damage as early in the opponent's turn as it can go.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Dodge, Riposte}, []ActionKind{Strike, Strike}, 1)

	var negations []ActionKind
	for _, e := range events {
		if e.Kind == KindNegated {
			negations = append(negations, e.Action)
		}
	}
	if !actionsEqual(negations, []ActionKind{Riposte, Dodge}) {
		t.Errorf("negations spent in order %v, want [Riposte Dodge]", negations)
	}
}

func TestARiposteCanKillTheAttacker(t *testing.T) {
	// Defenses converting into damage is the theory the category split rests on, so the
	// counter has to be able to finish a duel — and to end the attacker's turn when it does.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 3)

	events, _, bAfter := ResolveRound(a, b,
		[]ActionKind{Riposte}, []ActionKind{Strike, Strike}, 1)

	if bAfter.Alive() {
		t.Fatalf("B survived its own attack into a riposte with %d life", bAfter.CurrentLife)
	}
	if order := actionOrder(events); !sidesEqual(order, []Side{SideA, SideB}) {
		t.Errorf("actions = %v, want A's riposte and B's first strike only", order)
	}
}

func TestDefensesExpireWithTheTurnTheyCovered(t *testing.T) {
	// An unspent Dodge does not bank. It covers the opponent's next turn and then goes,
	// the same rule Guard follows.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Dodge}, nil, 1)
	if a1.Dodges != 1 {
		t.Fatalf("A ended round 1 with %d dodges, want 1 unspent", a1.Dodges)
	}

	round2, _, _ := ResolveRound(a1, b1, []ActionKind{Quick}, []ActionKind{Strike}, 2)
	if n := damageCount(round2); n != 2 {
		t.Errorf("damage events in round 2 = %d, want 2 — the dodge expired at A's turn", n)
	}
}

func TestPrepareFundsTheFollowingRound(t *testing.T) {
	d := duelist(10, 11, 100) // 5 AP before any bonus

	_, aAfter, _ := ResolveRound(d, duelist(10, 10, 100), []ActionKind{Prepare}, nil, 1)

	if aAfter.ActionPoints() != 7 {
		t.Errorf("budget after one Prepare = %d, want 7 (5 + 2)", aAfter.ActionPoints())
	}
}

func TestPrepareDoesNotFundTheRoundItIsPlayedIn(t *testing.T) {
	// The whole shape of the card: you pay now and you are paid later, which is what makes
	// it an investment rather than a discount.
	d := duelist(10, 11, 100)

	events, _, _ := ResolveRound(d, duelist(10, 10, 100), []ActionKind{Prepare}, nil, 1)

	for _, e := range events {
		if e.Kind == KindPrepared && e.Amount != prepareBonusAP {
			t.Errorf("prepared %d AP, want %d", e.Amount, prepareBonusAP)
		}
	}
	if d.ActionPoints() != 5 {
		t.Errorf("the duelist's own budget moved to %d mid-round, want 5", d.ActionPoints())
	}
}

func TestPreparesStackWithinARound(t *testing.T) {
	// Two in one round are worth +4. Deliberate: it is what puts a five-attack round in
	// reach without a ring discount, and that trade — a whole round spent setting up — is
	// the price of getting there.
	d := duelist(10, 11, 100) // 5 AP

	_, aAfter, _ := ResolveRound(d, duelist(10, 10, 100),
		[]ActionKind{Prepare, Prepare}, nil, 1)

	if aAfter.ActionPoints() != 9 {
		t.Errorf("budget after two Prepares = %d, want 9 (5 + 4)", aAfter.ActionPoints())
	}
}

func TestPrepareDoesNotCompoundAcrossRounds(t *testing.T) {
	// Preparing every round is worth a flat +2, not +2 then +4 then +6. The bonus is
	// replaced at the boundary rather than added to, which is what bounds the ramp.
	a := duelist(10, 11, 500) // 5 AP
	b := duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Prepare}, nil, 1)
	_, a2, _ := ResolveRound(a1, b1, []ActionKind{Prepare}, nil, 2)

	if a2.ActionPoints() != 7 {
		t.Errorf("budget after preparing twice in a row = %d, want a flat 7", a2.ActionPoints())
	}
}

func TestABonusLapsesIfItIsNotRenewed(t *testing.T) {
	a := duelist(10, 11, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Prepare}, nil, 1)
	_, a2, _ := ResolveRound(a1, b1, []ActionKind{Strike}, nil, 2)

	if a2.ActionPoints() != 5 {
		t.Errorf("budget after spending the bonus round = %d, want 5", a2.ActionPoints())
	}
}

func TestActionPointsFromSpeed(t *testing.T) {
	cases := []struct {
		spd, want int
	}{
		{0, 4},
		{15, 5}, // the shipped Monster1
		{20, 6}, // the shipped Fighter1
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

	if !d.CanAfford([]ActionKind{Heavy, Prepare}) { // 4 + 1
		t.Error("Heavy + Prepare costs 5 and should fit a 5 AP budget")
	}
	if d.CanAfford([]ActionKind{Guard, Riposte}) { // 3 + 3
		t.Error("Guard + Riposte costs 6 and should not fit a 5 AP budget")
	}
}

func TestCategoriesCoverEveryAction(t *testing.T) {
	// A new action with no category would silently fall into attack and resolve in the
	// wrong phase. Pin the whole table instead.
	want := map[ActionKind]Category{
		Prepare: CategorySetup,
		Guard:   CategorySetup,
		Quick:   CategoryAttack,
		Strike:  CategoryAttack,
		Heavy:   CategoryAttack,
		Dodge:   CategoryDefend,
		Riposte: CategoryDefend,
	}

	if len(AllActions) != len(want) {
		t.Fatalf("AllActions holds %d actions, the category table %d", len(AllActions), len(want))
	}
	for _, a := range AllActions {
		if got := a.Category(); got != want[a] {
			t.Errorf("%v is %v, want %v", a, got, want[a])
		}
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
	if n := damageCount(events); n != 1 {
		t.Errorf("damage events = %d, want 1 — the round should stop at the kill", n)
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

func TestBothSidesDefendingDoesNotAlias(t *testing.T) {
	// resolveAction passes duelists by value; if that ever became pointer-based, a mutual
	// defense is where an aliasing bug would surface — one side's raised guard leaking
	// onto the other.
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	_, a1, b1 := ResolveRound(a, b, []ActionKind{Guard}, []ActionKind{Dodge}, 1)
	if !a1.Guarded {
		t.Error("A raised a guard and did not end the round with one")
	}
	if b1.Guarded {
		t.Error("B's dodge left it guarded; A's flag has leaked across")
	}
	if a1.Dodges != 0 || b1.Dodges != 1 {
		t.Errorf("dodges: a=%d b=%d, want 0 and 1", a1.Dodges, b1.Dodges)
	}
}

func TestRoundIsDeterministic(t *testing.T) {
	a := duelist(7, 13, 300)
	b := duelist(9, 17, 300)

	aPlan := []ActionKind{Strike, Guard, Prepare}
	bPlan := []ActionKind{Quick, Riposte}

	first, a1, b1 := ResolveRound(a, b, aPlan, bPlan, 1)
	second, a2, b2 := ResolveRound(a, b, aPlan, bPlan, 1)

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

func TestEveryStyleObeysBothBounds(t *testing.T) {
	// The two bounds on a round apply to the opponent exactly as they apply to the player.
	// Swept across a wide speed range and across banked points, because both bounds move:
	// AP grows with Spd and with BonusAP, while the action cap does not, which is the
	// corner where an unbounded planner would overrun.
	for _, style := range PlanStyles() {
		for spd := 0; spd <= 120; spd += 3 {
			for _, bonus := range []int{0, 2, 4, 10} {
				d := duelist(10, spd, 100)
				d.BonusAP = bonus
				plan := PlanFor(style, d)

				if !d.CanAfford(plan) {
					t.Errorf("%v at Spd %d bonus %d: plan costs %d, budget is %d",
						style, spd, bonus, CostOf(plan), d.ActionPoints())
				}
				if len(plan) > d.MaxActions() {
					t.Errorf("%v at Spd %d bonus %d: %d actions, cap is %d",
						style, spd, bonus, len(plan), d.MaxActions())
				}
				if len(plan) == 0 {
					t.Errorf("%v at Spd %d bonus %d: plan is empty", style, spd, bonus)
				}
			}
		}
	}
}

func TestEveryStyleFightsInADifferentShape(t *testing.T) {
	// The whole reason styles exist. If two of them produce the same round against the same
	// duelist then one of them is not earning its place, and the player has nothing new to
	// find an answer to.
	d := duelist(10, 15, 100) // the shipped Monster1: 5 AP

	seen := map[string]PlanStyle{}
	for _, style := range PlanStyles() {
		key := planKey(PlanFor(style, d))
		if prev, clash := seen[key]; clash {
			t.Errorf("%v and %v both plan %s", prev, style, key)
		}
		seen[key] = style
	}
}

// planKey renders a plan so two of them can be compared as strings.
func planKey(plan []ActionKind) string {
	out := ""
	for i, a := range plan {
		if i > 0 {
			out += "+"
		}
		out += a.String()
	}
	return out
}

func TestSwarmAttacksMoreOftenThanBrute(t *testing.T) {
	// The specific thing styles were added to fix: a negation stops one blow, so an opponent
	// that attacks five times walks through the pair of them that shut a brute out
	// completely. Width is the mechanic, not damage.
	d := duelist(10, 15, 100)

	brute := PlanFor(StyleBrute, d)
	swarm := PlanFor(StyleSwarm, d)

	if len(swarm) <= len(brute) {
		t.Errorf("swarm plans %d actions, brute %d; swarm must be the wider round",
			len(swarm), len(brute))
	}
}

func TestSwarmSpendsSpareBudgetOnBiggerAttacksNotMoreOfThem(t *testing.T) {
	// A fast swarm has more points than it has slots. It must not waste them, and it must
	// not stop being a swarm to use them.
	d := duelist(10, 100, 100) // 14 AP against a cap of 5 actions
	plan := PlanFor(StyleSwarm, d)

	if len(plan) != d.MaxActions() {
		t.Errorf("plan is %d actions, want the full %d", len(plan), d.MaxActions())
	}
	if left := d.ActionPoints() - CostOf(plan); left > 0 {
		t.Errorf("left %d points unspent with upgrades still available", left)
	}
	for _, a := range plan {
		if a.Category() != CategoryAttack {
			t.Errorf("swarm queued %v, which is %v — every slot should be an attack", a, a.Category())
		}
	}
}

func TestWardenGuardsAndStillAttacks(t *testing.T) {
	d := duelist(10, 15, 100) // 5 AP: Guard is 3, leaving a Strike
	plan := PlanFor(StyleWarden, d)

	guards, attacks := 0, 0
	for _, a := range plan {
		switch a.Category() {
		case CategorySetup:
			guards++
		case CategoryAttack:
			attacks++
		}
	}
	if guards != 1 {
		t.Errorf("warden queued %d setups, want exactly 1 Guard", guards)
	}
	if attacks == 0 {
		t.Error("warden queued no attacks; a pure turtle cannot win and is not a fight")
	}
}

func TestWardenFallsBackToAttackingWhenItCannotAffordAGuard(t *testing.T) {
	// A duelist too slow to pay 3 for a Guard must still take a turn rather than stand there.
	d := duelist(10, 0, 100)
	d.BonusAP = -2 // floors ActionPoints at 1

	if plan := PlanFor(StyleWarden, d); len(plan) == 0 {
		t.Error("warden with a 1 AP budget planned nothing")
	}
}

func TestTacticianBanksThenUnloads(t *testing.T) {
	// The rhythm is the point: a light round the player can punish, then an oversized one
	// they have to answer. It reads which round it is in off its own BonusAP, so it needs no
	// memory beyond what Prepare already leaves behind.
	d := duelist(10, 15, 100) // 5 AP

	setup := PlanFor(StyleTactician, d)
	prepares := 0
	for _, a := range setup {
		if a == Prepare {
			prepares++
		}
	}
	if prepares == 0 {
		t.Fatalf("setup round planned no Prepare: %v", setup)
	}

	// Play the setup round through so the bank is real rather than assumed.
	_, after, _ := ResolveRound(d, duelist(10, 10, 500), setup, nil, 1)
	if after.BonusAP == 0 {
		t.Fatal("the setup round banked nothing")
	}

	payoff := PlanFor(StyleTactician, after)
	for _, a := range payoff {
		if a == Prepare {
			t.Errorf("payoff round is still preparing: %v", payoff)
		}
	}
	if CostOf(payoff) <= CostOf(setup) {
		t.Errorf("payoff round costs %d, setup round %d; the spike must be the bigger one",
			CostOf(payoff), CostOf(setup))
	}
}

func TestParsePlanStyleRoundTripsAndFallsBack(t *testing.T) {
	// Styles arrive as strings out of combatants.json, so a typo must produce a fightable
	// enemy rather than one that stands still.
	for _, style := range PlanStyles() {
		got, ok := ParsePlanStyle(style.String())
		if !ok || got != style {
			t.Errorf("%q parsed to %v (ok=%v), want %v", style.String(), got, ok, style)
		}
	}

	got, ok := ParsePlanStyle("wardne")
	if ok {
		t.Error("a typo reported itself as a known style")
	}
	if got != StyleBrute {
		t.Errorf("unknown style fell back to %v, want brute", got)
	}
}
