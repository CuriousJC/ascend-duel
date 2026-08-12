package combat

import "testing"

// The four element statuses, and the lifecycle all of them share.
//
// **Every test here is written against the rule rather than the constant** where it can be —
// a chill costs `chillPerHit` action points, not "one" — so tuning a number does not fail a
// test that was checking the mechanic. The exceptions are the ones pinning a *relationship*
// (a burn ticks twice, a status is gone by the round after), which is the thing that must not
// move without somebody deciding it should.

// statusEvents returns the KindStatus events for one element.
func statusEvents(events []Event, e Element) []Event {
	var out []Event
	for _, ev := range events {
		if ev.Kind == KindStatus && ev.Element == e {
			out = append(out, ev)
		}
	}
	return out
}

func countKind(events []Event, k EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == k {
			n++
		}
	}
	return n
}

func TestALandedElementalAttackAppliesItsStatus(t *testing.T) {
	// The trigger rule: an attack that connects applies its element, and nothing else does.
	for _, e := range []Element{Fire, Ice, Lightning, Earth} {
		a, b := duelist(10, 10, 500), duelist(10, 10, 500)
		events, _, bAfter := ResolveRound(a, b, []Card{Of(Strike, e)}, nil, 1)

		if got := statusEvents(events, e); len(got) != 1 {
			t.Errorf("a %v Strike raised %d status events, want 1", e, len(got))
		}
		if !bAfter.Statuses[e].Active() {
			t.Errorf("a %v Strike left no %v status on the target", e, e)
		}
	}
}

func TestOnlyAttacksApplyAStatus(t *testing.T) {
	// **Decided 2026-08-12**: a prepare or a defend carries its element for combos and for the
	// ring discount and applies nothing. Otherwise a 1-AP Gather would be as good a status
	// delivery as a 1-AP Jab, and the prepare phase would quietly become the status engine.
	for _, a := range []ActionKind{Gather, Sift, Guard, Ritual, Brace, Dodge, Riposte, Mirror} {
		attacker, target := duelist(10, 40, 500), duelist(10, 10, 500)
		events, _, bAfter := ResolveRound(attacker, target, []Card{Of(a, Fire)}, nil, 1)

		if n := len(statusEvents(events, Fire)); n != 0 {
			t.Errorf("a fire %v applied a status %d times", a, n)
		}
		if bAfter.Statuses[Fire].Active() {
			t.Errorf("a fire %v left a burn on the opponent", a)
		}
	}
}

func TestAStoppedAttackAppliesNoStatus(t *testing.T) {
	// A Dodge, a Riposte and a Mirror all stop the blow dead, so there is nothing to carry the
	// element in on. The Feint strip is deliberately unconditional and this is deliberately
	// not: a status is a consequence of connecting.
	for _, defence := range []ActionKind{Dodge, Riposte, Mirror} {
		a, b := duelist(10, 10, 500), duelist(10, 40, 500)

		// B raises the defence in round one, A swings into it in round two.
		_, a1, b1 := ResolveRound(a, b, nil, []Card{Plain(defence)}, 1)
		events, _, bAfter := ResolveRound(a1, b1, []Card{Of(Strike, Fire)}, nil, 2)

		if n := len(statusEvents(events, Fire)); n != 0 {
			t.Errorf("a Strike stopped by a %v still applied a burn (%d events)", defence, n)
		}
		if bAfter.Statuses[Fire].Active() {
			t.Errorf("a Strike stopped by a %v left a burn behind", defence)
		}
	}
}

func TestAGuardedAttackStillAppliesItsStatus(t *testing.T) {
	// **The status lands because the blow did, not because it hurt.** A Guard halves the hit
	// and the hit still connected, so making the status conditional on the final figure would
	// let a defensive card silently un-apply an element the attacker had already paid for.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := ResolveRound(a, b, nil, []Card{Plain(Guard)}, 1)
	events, _, bAfter := ResolveRound(a1, b1, []Card{Of(Strike, Ice)}, nil, 2)

	if n := len(statusEvents(events, Ice)); n != 1 {
		t.Errorf("a guarded Strike applied its chill %d times, want 1", n)
	}
	if !bAfter.Statuses[Ice].Active() {
		t.Error("a guarded ice Strike left no chill")
	}
}

func TestAStatusStacksInAmountAndRefreshesInDuration(t *testing.T) {
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)
	_, _, bAfter := ResolveRound(a, b, []Card{Of(Jab, Fire), Of(Jab, Fire)}, nil, 1)

	if got, want := bAfter.Statuses[Fire].Amount, burnPerHit*2; got != want {
		t.Errorf("two fire hits stacked to %d, want %d", got, want)
	}
	// Refreshed rather than added: statusDuration, less the one round-end that has passed.
	if got, want := bAfter.Statuses[Fire].Rounds, statusDuration-1; got != want {
		t.Errorf("two fire hits left %d rounds, want %d — duration refreshes, it does not add",
			got, want)
	}
}

func TestAStatusIsGoneByTheEndOfTheRoundAfterItLanded(t *testing.T) {
	// The lifecycle, pinned as a relationship rather than as a number. A status has to survive
	// the round-end of the round that applied it — otherwise one applied by side B, who acts
	// second, would never bite anything at all — and it must not survive the next one.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []Card{Of(Strike, Ice)}, nil, 1)
	if !b1.Statuses[Ice].Active() {
		t.Fatal("the chill did not survive the round it was applied in")
	}

	_, _, b2 := ResolveRound(a1, b1, nil, nil, 2)
	if b2.Statuses[Ice].Active() {
		t.Error("the chill outlived the round after the one that applied it")
	}
}

func TestIceCutsTheTargetsBudget(t *testing.T) {
	// Ice is read at budget time rather than subtracted when it lands, which is what makes it
	// bite the round *after* the blow — the budget for the round in progress was committed
	// before the attack resolved.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)
	before := b.ActionPoints()

	_, _, bAfter := ResolveRound(a, b, []Card{Of(Strike, Ice)}, nil, 1)

	if got, want := bAfter.ActionPoints(), before-chillPerHit; got != want {
		t.Errorf("a chilled duelist has %d AP, want %d (was %d)", got, want, before)
	}
}

func TestABudgetNeverFallsBelowOneHoweverColdItGets(t *testing.T) {
	// The existing floor in ActionPoints has to hold against the new subtraction, or a duelist
	// hit by enough ice would have a negative budget and could not take a turn at all.
	d := duelist(10, 0, 500)
	d.Statuses[Ice] = Status{Amount: 99, Rounds: 1}

	if got := d.ActionPoints(); got != 1 {
		t.Errorf("a deeply chilled duelist has %d AP, want the floor of 1", got)
	}
}

func TestLightningMakesTheNextAttackMissOutright(t *testing.T) {
	// **Deterministic, decided 2026-08-12.** A roll would need an injected source and a sixth
	// determinism stream; a certain miss keeps the package pure and matches the rule combos
	// already follow — what you committed to cannot be silently undone.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	_, a1, b1 := ResolveRound(a, b, []Card{Of(Strike, Lightning)}, nil, 1)
	if !b1.Statuses[Lightning].Active() {
		t.Fatal("the lightning Strike left no shock")
	}

	before := a1.CurrentLife
	events, a2, _ := ResolveRound(a1, b1, nil, []Card{Plain(Strike)}, 2)

	if n := countKind(events, KindMissed); n != 1 {
		t.Errorf("a shocked duelist's attack missed %d times, want 1", n)
	}
	if a2.CurrentLife != before {
		t.Errorf("a missed attack still dealt %d damage", before-a2.CurrentLife)
	}
}

func TestAShockIsSpentByTheAttackItStops(t *testing.T) {
	// One stack, one miss. A shock that stopped every attack in a turn would be a whole-turn
	// negation for the price of a Jab, which is what Mirror costs 4 points to do.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := ResolveRound(a, b, []Card{Of(Jab, Lightning)}, nil, 1)
	events, _, _ := ResolveRound(a1, b1, nil, []Card{Plain(Jab), Plain(Jab)}, 2)

	if n := countKind(events, KindMissed); n != 1 {
		t.Errorf("%d attacks missed, want exactly 1 — a shock stack stops one blow", n)
	}
	if n := countKind(events, KindDamage); n != 1 {
		t.Errorf("%d attacks landed, want 1 — the second Jab is not shocked", n)
	}
}

func TestAMissedAttackDoesNothingElseEither(t *testing.T) {
	// The miss happens before the Feint strip, before any negation is spent and before a
	// status is applied. The attack did not occur.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := ResolveRound(a, b, []Card{Of(Jab, Lightning)}, nil, 1)

	// B is shocked, holds a Feint, and A is holding a Riposte for it.
	events, _, bAfter := ResolveRound(a1, b1, []Card{Plain(Riposte)}, []Card{Of(Feint, Fire)}, 2)

	if n := countKind(events, KindStripped); n != 0 {
		t.Error("a missed Feint still stripped a negation")
	}
	if n := len(statusEvents(events, Fire)); n != 0 {
		t.Error("a missed Feint still applied its burn")
	}
	if bAfter.Statuses[Fire].Active() {
		t.Error("a missed attack left its element on somebody")
	}
}

func TestFireTicksAtTheEndOfEveryRoundItSurvives(t *testing.T) {
	// The DoT: it lands at end of round, including the end of the round it was applied in, and
	// it persists across the boundary. Two ticks from one hit at the current duration.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	r1, a1, b1 := ResolveRound(a, b, []Card{Of(Jab, Fire)}, nil, 1)
	if n := countKind(r1, KindBurned); n != 1 {
		t.Errorf("round 1 burned %d times, want 1 — a DoT ticks at the end of the round it lands in", n)
	}

	r2, _, b2 := ResolveRound(a1, b1, nil, nil, 2)
	if n := countKind(r2, KindBurned); n != 1 {
		t.Errorf("round 2 burned %d times, want 1 — the burn persists across the boundary", n)
	}

	r3, _, _ := ResolveRound(a1, b2, nil, nil, 3)
	if n := countKind(r3, KindBurned); n != 0 {
		t.Errorf("round 3 burned %d times, want 0 — the burn has expired", n)
	}
}

func TestABurnCanKill(t *testing.T) {
	// Fire is the one thing in the game that ends a duel without an action, so the log has to
	// say so — the screen reads KindDefeated to end the fight and would otherwise leave a dead
	// duelist standing.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	// Enough life to survive the Jab and not the tick, so it is unambiguously the fire that
	// finished it rather than the blow that lit it.
	b.CurrentLife = Jab.Damage(a.Str) + 1

	events, _, bAfter := ResolveRound(a, b, []Card{Of(Jab, Fire)}, nil, 1)

	if bAfter.Alive() {
		t.Fatalf("the burn left the target on %d life", bAfter.CurrentLife)
	}
	if countKind(events, KindDefeated) == 0 {
		t.Error("a duelist killed by a burn produced no KindDefeated event")
	}
}

func TestEarthBluntsWhatItsVictimDeals(t *testing.T) {
	// Earth is the only status that reaches forward into what its victim *does*. It applies
	// attacker-side, before any of the defender's cards touch the blow.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	plain, _, _ := ResolveRound(a, b, nil, []Card{Plain(Strike)}, 1)
	base := firstDamage(t, plain, SideB).Amount

	_, a1, b1 := ResolveRound(a, b, []Card{Of(Strike, Earth)}, nil, 1)
	weighted, _, _ := ResolveRound(a1, b1, nil, []Card{Plain(Strike)}, 2)

	got := firstDamage(t, weighted, SideB).Amount
	want := blunt(base, weightPerHit)
	if got != want {
		t.Errorf("a weighted Strike dealt %d, want %d (%d blunted by %d%%)",
			got, want, base, weightPerHit)
	}
	if got >= base {
		t.Errorf("a weighted Strike dealt %d against an unweighted %d — earth did nothing", got, base)
	}
}

func TestAWeightCannotBluntEverything(t *testing.T) {
	// Without the cap, four earth Jabs make an opponent harmless for 4 AP — a cheaper answer to
	// a duel than winning it.
	d := duelist(10, 10, 500)
	d.Statuses[Earth] = Status{Amount: 1000, Rounds: 1}

	if got := d.weightPct(); got != weightCapPct {
		t.Errorf("a hugely weighted duelist is blunted %d%%, want the cap of %d%%", got, weightCapPct)
	}
	if blunt(100, d.weightPct()) <= 0 {
		t.Error("a capped weight still reduced a blow to nothing")
	}
}

func TestBluntingRoundsTowardZeroLikeEveryOtherReduction(t *testing.T) {
	// Earth is the first percentage in a package documented as pure integer arithmetic. The
	// rounding rule matters more than the direction: it has to match guardDivisor and
	// scaleDamage so a player can predict it from the reductions they already know.
	if got, want := blunt(15, 10), 13; got != want { // 13.5
		t.Errorf("15 blunted by 10%% = %d, want %d", got, want)
	}
	if got, want := blunt(1, 50), 0; got != want {
		t.Errorf("1 blunted by 50%% = %d, want %d", got, want)
	}
	if got := blunt(20, 0); got != 20 {
		t.Errorf("an unweighted blow was changed to %d", got)
	}
}

func TestStatusesLeaveARoundStillDeterministic(t *testing.T) {
	// The rule the whole package is built on, re-checked against the one feature added since
	// that could plausibly have broken it. Nothing in a status consults a clock or a map.
	a, b := duelist(10, 20, 500), duelist(10, 20, 500)
	aPlan := []Card{Of(Strike, Fire), Of(Jab, Ice)}
	bPlan := []Card{Of(Jab, Lightning), Of(Jab, Earth)}

	first, a1, b1 := ResolveRound(a, b, aPlan, bPlan, 1)
	for i := 0; i < 20; i++ {
		got, a2, b2 := ResolveRound(a, b, aPlan, bPlan, 1)
		if len(got) != len(first) {
			t.Fatalf("run %d produced %d events, first run produced %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("run %d event %d = %+v, first run = %+v", i, j, got[j], first[j])
			}
		}
		if a2 != a1 || b2 != b1 {
			t.Fatalf("run %d ended in a different state", i)
		}
	}
}
