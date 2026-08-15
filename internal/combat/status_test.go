package combat

import (
	"math/rand"
	"testing"
)

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
		events, _, bAfter := resolve(a, b, []Card{Of(Strike, e)}, nil, 1)

		if got := statusEvents(events, e); len(got) != 1 {
			t.Errorf("a %v Strike raised %d status events, want 1", e, len(got))
		}
		if !bAfter.Statuses[e].Active() {
			t.Errorf("a %v Strike left no %v status on the target", e, e)
		}
	}
}

func TestOnlyAttacksApplyAStatus(t *testing.T) {
	// **Decided 2026-08-12**: a plan card carries its element for combos and for the ring
	// discount and applies nothing. Otherwise a 1-AP Prepare would be as good a status delivery
	// as a 1-AP Jab, and the plan phase would quietly become the status engine.
	for _, a := range []ActionKind{Prepare, Plan, Defend} {
		attacker, target := duelist(10, 40, 500), duelist(10, 10, 500)
		events, _, bAfter := resolve(attacker, target, []Card{Of(a, Fire)}, nil, 1)

		if n := len(statusEvents(events, Fire)); n != 0 {
			t.Errorf("a fire %v applied a status %d times", a, n)
		}
		if bAfter.Statuses[Fire].Active() {
			t.Errorf("a fire %v left a burn on the opponent", a)
		}
	}
}

func TestABlockedBlowStillAppliesItsStatus(t *testing.T) {
	// **The status lands because the hand formed, not because the blow hurt** *(2026-08-14)*.
	// This reverses the rule that stood while defends *negated*: back then a stopped attack
	// carried nothing in, because nothing arrived. A Defend takes 50% off — so making the status
	// conditional on the final figure would let a defensive card silently un-apply an element the
	// attacker had already paid for, and under one blow per turn that would be every defensive
	// card in the game.
	for _, defence := range []ActionKind{Defend} {
		a, b := duelist(10, 10, 500), duelist(10, 40, 500)

		// B raises the defence in round one, A swings into it in round two.
		_, a1, b1 := resolve(a, b, nil, []Card{Plain(defence)}, 1)
		events, _, bAfter := resolve(a1, b1, []Card{Of(Strike, Fire)}, nil, 2)

		if n := len(statusEvents(events, Fire)); n != 1 {
			t.Errorf("a Strike met by a %v applied its burn %d times, want 1", defence, n)
		}
		if !bAfter.Statuses[Fire].Active() {
			t.Errorf("a Strike met by a %v left no burn", defence)
		}
	}
}

func TestOneColourInAHandIsOneStatusHoweverManyCardsCarryIt(t *testing.T) {
	// The mix counts **distinct** colours, not coloured cards, so this is the rule that decides
	// status volume now. Two fire Jabs are a mono fire Pair and land one burn — where under the
	// per-card model they landed two.
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)

	events, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire), Of(Jab, Fire)}, nil, 1)

	if n := len(statusEvents(events, Fire)); n != 1 {
		t.Errorf("two fire cards in one hand applied %d burns, want 1", n)
	}
	if got, want := bAfter.Statuses[Fire].Amount, burnPerHit; got != want {
		t.Errorf("a mono fire hand stacked to %d, want %d", got, want)
	}
}

func TestEachColourInTheHandLandsItsOwnStatus(t *testing.T) {
	// The other end of the same rule: a duo hand lands both, which is what the mix multiplier is
	// paying for besides damage.
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)

	_, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire), Of(Jab, Ice)}, nil, 1)

	if !bAfter.Statuses[Fire].Active() {
		t.Error("a duo fire/ice hand left no burn")
	}
	if !bAfter.Statuses[Ice].Active() {
		t.Error("a duo fire/ice hand left no chill")
	}
}

func TestACardOutsideTheHandCarriesNoColour(t *testing.T) {
	// Attack cards that build no hand are announced and contribute nothing — not damage and not
	// an element. `Strike, Jab, Strike` is a Strike Pair and the Jab is not in it, so a fire Jab
	// alongside two plain Strikes burns nobody.
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)

	events, _, bAfter := resolve(a, b,
		[]Card{Plain(Strike), Of(Jab, Fire), Plain(Strike)}, nil, 1)

	if n := len(statusEvents(events, Fire)); n != 0 {
		t.Errorf("a fire Jab outside the hand applied %d burns, want 0", n)
	}
	if bAfter.Statuses[Fire].Active() {
		t.Error("a card that earned nothing still left its element behind")
	}
}

func TestAHalvedAttackStillAppliesItsStatus(t *testing.T) {
	// **The status lands because the blow did, not because it hurt.** A Defend halves the hit
	// and the hit still connected, so making the status conditional on the final figure would
	// let a defensive card silently un-apply an element the attacker had already paid for.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := resolve(a, b, nil, []Card{Plain(Defend)}, 1)
	events, _, bAfter := resolve(a1, b1, []Card{Of(Strike, Ice)}, nil, 2)

	if n := len(statusEvents(events, Ice)); n != 1 {
		t.Errorf("a halved Strike applied its chill %d times, want 1", n)
	}
	if !bAfter.Statuses[Ice].Active() {
		t.Error("a halved ice Strike left no chill")
	}
}

func TestAStatusStacksInAmountAndRefreshesInDuration(t *testing.T) {
	// **Two rounds, not two cards** *(2026-08-14)*. A turn lands one blow and its mix applies one
	// status per distinct colour, so stacking is now something that happens *across* turns —
	// see TestOneColourInAHandIsOneStatusHoweverManyCardsCarryIt for the other half.
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)
	_, _, b2 := resolve(a1, b1, []Card{Of(Jab, Fire)}, nil, 2)

	if got, want := b2.Statuses[Fire].Amount, burnPerHit*2; got != want {
		t.Errorf("two fire hits stacked to %d, want %d", got, want)
	}
	// Refreshed rather than added: statusDuration, less the one round-end that has passed.
	if got, want := b2.Statuses[Fire].Rounds, statusDuration-1; got != want {
		t.Errorf("two fire hits left %d rounds, want %d — duration refreshes, it does not add",
			got, want)
	}
}

func TestAStatusIsGoneByTheEndOfTheRoundAfterItLanded(t *testing.T) {
	// The lifecycle, pinned as a relationship rather than as a number. A status has to survive
	// the round-end of the round that applied it — otherwise one applied by side B, who acts
	// second, would never bite anything at all — and it must not survive the next one.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Ice)}, nil, 1)
	if !b1.Statuses[Ice].Active() {
		t.Fatal("the chill did not survive the round it was applied in")
	}

	_, _, b2 := resolve(a1, b1, nil, nil, 2)
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

	_, _, bAfter := resolve(a, b, []Card{Of(Strike, Ice)}, nil, 1)

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

func TestAShockIsARollAndTheSourceDecidesIt(t *testing.T) {
	// **A roll again as of 2026-08-14**, reversing the deterministic version taken two days
	// earlier. One blow per turn is what forced it: a certain miss used to delete one attack out
	// of several and now deletes the whole turn, so a 1 AP lightning Jab could erase an 8 AP
	// Barrage outright.
	//
	// The same shocked duelist and the same turn, twice, with the two rolls decided rather than
	// seeded — see fixedSource.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Lightning)}, nil, 1)
	if !b1.Statuses[Lightning].Active() {
		t.Fatal("the lightning Strike left no shock")
	}

	missed, aMissed, _ := resolveWith(alwaysMisses(), a1, b1, nil, []Card{Plain(Strike)}, 2)
	landed, aLanded, _ := resolveWith(neverMisses(), a1, b1, nil, []Card{Plain(Strike)}, 2)

	if n := countKind(missed, KindMissed); n != 1 {
		t.Errorf("a losing roll missed %d times, want 1", n)
	}
	if aMissed.CurrentLife != a1.CurrentLife {
		t.Errorf("a missed attack still dealt %d damage", a1.CurrentLife-aMissed.CurrentLife)
	}

	if n := countKind(landed, KindMissed); n != 0 {
		t.Errorf("a winning roll missed %d times, want 0", n)
	}
	if aLanded.CurrentLife >= a1.CurrentLife {
		t.Error("a shocked attack that passed its roll dealt no damage")
	}
}

func TestMoreShockIsMoreLikelyButNeverCertain(t *testing.T) {
	// **The cap is what stops the roll becoming the old rule by another route.** Without it,
	// enough lightning is a certain miss again — and a defence that always works is exactly what
	// one blow per turn makes intolerable.
	d := duelist(10, 10, 500)
	if got := d.shockChancePct(); got != 0 {
		t.Errorf("an unshocked duelist has a %d%% chance to miss, want 0", got)
	}

	d.Statuses[Lightning] = Status{Amount: 1, Rounds: 1}
	one := d.shockChancePct()
	d.Statuses[Lightning] = Status{Amount: 2, Rounds: 1}
	two := d.shockChancePct()
	if two <= one {
		t.Errorf("two stacks gave %d%% against one stack's %d%% — stacks have to add", two, one)
	}

	d.Statuses[Lightning] = Status{Amount: 99, Rounds: 1}
	if got := d.shockChancePct(); got != shockMissCapPct {
		t.Errorf("99 stacks gave %d%%, want the cap of %d%%", got, shockMissCapPct)
	}
	if shockMissCapPct >= 100 {
		t.Errorf("the cap is %d%%, which allows a certain miss — the rule this replaced",
			shockMissCapPct)
	}
}

func TestAShockStackIsSpentWhetherOrNotTheRollLands(t *testing.T) {
	// **One stack per attack phase, win or lose.** A shock that only burned itself on a success
	// would last until it worked, which is a guarantee wearing a probability's clothes.
	for _, roll := range []struct {
		name string
		rng  *rand.Rand
		miss bool
	}{
		{"a losing roll", alwaysMisses(), true},
		{"a winning roll", neverMisses(), false},
	} {
		d := duelist(10, 10, 500)
		d.Statuses[Lightning] = Status{Amount: 2, Rounds: statusDuration}

		after, missed := spendShock(d, roll.rng)

		if missed != roll.miss {
			t.Errorf("%s reported missed=%v, want %v", roll.name, missed, roll.miss)
		}
		if got := after.Statuses[Lightning].Amount; got != 1 {
			t.Errorf("%s left %d stacks, want 1 — one is spent whatever happens", roll.name, got)
		}
	}
}

func TestAShockDeletesTheWholeTurnBecauseATurnIsOneBlow(t *testing.T) {
	// Under the multi-blow model a shock cancelled one attack out of several. A turn now resolves
	// a single blow, so a landed roll deletes all of it — which is the whole reason the certain
	// miss had to become a roll. See MECHANICS.md.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Lightning)}, nil, 1)
	events, aAfter, _ := resolveWith(alwaysMisses(), a1, b1, nil, []Card{Plain(Jab), Plain(Jab)}, 2)

	if n := countKind(events, KindMissed); n != 1 {
		t.Errorf("%d misses, want exactly 1 — a turn has one attack to miss", n)
	}
	if n := countKind(events, KindDamage); n != 0 {
		t.Errorf("%d damage events, want 0 — the blow that missed was the whole turn", n)
	}
	if aAfter.CurrentLife != a1.CurrentLife {
		t.Errorf("a missed turn still dealt %d damage", a1.CurrentLife-aAfter.CurrentLife)
	}
}

func TestAMissedAttackDoesNothingElseEither(t *testing.T) {
	// The miss happens before any defence is spent and before any status is applied. The attack
	// did not occur.
	//
	// **What it does not undo is the hand's own reward** — a stagger is paid on forming the hand,
	// not on connecting. That is deliberate and is pinned in combo_test.go.
	a, b := duelist(10, 10, 500), duelist(10, 40, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Lightning)}, nil, 1)

	// B is shocked and swings a fire Strike; A is holding a Defend for it.
	events, _, bAfter := resolveWith(alwaysMisses(), a1, b1,
		[]Card{Plain(Defend)}, []Card{Of(Strike, Fire)}, 2)

	if n := countKind(events, KindNegated); n != 0 {
		t.Error("a missed attack still spent the defence that was waiting for it")
	}
	if n := len(statusEvents(events, Fire)); n != 0 {
		t.Error("a missed attack still applied its burn")
	}
	if bAfter.Statuses[Fire].Active() {
		t.Error("a missed attack left its element on somebody")
	}
}

func TestFireTicksAtTheEndOfEveryRoundItSurvives(t *testing.T) {
	// The DoT: it lands at end of round, including the end of the round it was applied in, and
	// it persists across the boundary. Two ticks from one hit at the current duration.
	a, b := duelist(10, 10, 500), duelist(10, 10, 500)

	r1, a1, b1 := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)
	if n := countKind(r1, KindBurned); n != 1 {
		t.Errorf("round 1 burned %d times, want 1 — a DoT ticks at the end of the round it lands in", n)
	}

	r2, _, b2 := resolve(a1, b1, nil, nil, 2)
	if n := countKind(r2, KindBurned); n != 1 {
		t.Errorf("round 2 burned %d times, want 1 — the burn persists across the boundary", n)
	}

	r3, _, _ := resolve(a1, b2, nil, nil, 3)
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

	events, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)

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

	plain, _, _ := resolve(a, b, nil, []Card{Plain(Strike)}, 1)
	base := firstDamage(t, plain, SideB).Amount

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Earth)}, nil, 1)
	weighted, _, _ := resolve(a1, b1, nil, []Card{Plain(Strike)}, 2)

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

	first, a1, b1 := resolve(a, b, aPlan, bPlan, 1)
	for i := 0; i < 20; i++ {
		got, a2, b2 := resolve(a, b, aPlan, bPlan, 1)
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

func TestADeadDuelistDoesNotBurn(t *testing.T) {
	// **A corpse does not tick, and the reason is the log rather than the arithmetic.** The
	// first version burned regardless: a duelist killed on the opposing turn took a fire tick
	// afterwards and the Resolution feed read "falls / burns for 2 / falls". Whether a duelist
	// is dead is settled before either side's round-end runs, so skipping the tick introduces
	// no order dependence.
	a, b := duelist(10, 40, 500), duelist(10, 10, 500)

	// A fire Pair rather than a fire Jab beside a plain Strike: the pair is a *hand*, so both
	// cards count and the mix is fire. A mixed pile would resolve as its single biggest attack —
	// the plain Strike — and light nothing at all.
	turn := []Card{Of(Jab, Fire), Of(Jab, Fire)}

	// Learn what the hand deals rather than writing the arithmetic down a second time; the
	// multipliers are expected to be retuned and this test is not about them.
	probe, _, _ := resolve(a, b, turn, nil, 1)
	blow := firstDamage(t, probe, SideA).Amount

	// Light B and kill B with the same turn.
	b.CurrentLife = blow
	events, _, bAfter := resolve(a, b, turn, nil, 1)

	if bAfter.Alive() {
		t.Fatalf("the target survived on %d life; this test needs it dead", bAfter.CurrentLife)
	}
	if !bAfter.Statuses[Fire].Active() {
		t.Fatal("the fire hand left no burn, so this test proves nothing")
	}
	if n := countKind(events, KindBurned); n != 0 {
		t.Errorf("a dead duelist burned %d times", n)
	}
	if n := countKind(events, KindDefeated); n != 1 {
		t.Errorf("%d KindDefeated events, want exactly 1 — a duelist falls once", n)
	}
}
