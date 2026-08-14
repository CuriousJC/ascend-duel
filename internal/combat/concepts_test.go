package combat

import "testing"

// The 2026-08-08 concepts: Ritual, Brace, Feint and Retreat. Each gets the case that says what
// makes it differ in *kind* from the tier below it rather than merely in size — that being the
// rule the concept grid in MECHANICS.md is held to, and the rule a cost ladder quietly breaks.
//
// **Sift is deliberately untested here.** Its effect is on the hand, the hand lives on the
// scene, and this package cannot see a deck by design. It is the one concept `tools/balance`
// cannot exercise either.

// hasKind reports whether the log holds an event of the given kind.
func hasKind(events []Event, kind EventKind) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// kindCount is how many events of a kind the log holds.
func kindCount(events []Event, kind EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestRitualBanksMoreThanGatherAndAtABetterRate(t *testing.T) {
	// Ritual sells two things: **slots** — four Gathers bank +8 but spend four of five action
	// slots, where one Ritual banks +6 and leaves four slots to fight with — and, since
	// 2026-08-14, **rate**. It nets +2 AP per point spent against Gather's +1. If the rate ever
	// falls back to Gather's the card is only selling slots again, which is a different card and
	// a decision somebody should make deliberately rather than by editing a constant.
	gatherNet, ritualNet := gatherBonusAP-costGather, ritualBonusAP-costRitual
	if ritualNet <= gatherNet {
		t.Errorf("Gather nets %+d AP and Ritual nets %+d — Ritual is meant to be the better bank",
			gatherNet, ritualNet)
	}
	if ritualBonusAP <= gatherBonusAP {
		t.Errorf("Ritual banks %d against Gather's %d", ritualBonusAP, gatherBonusAP)
	}

	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, aAfter, _ := ResolveRound(a, b, PlainCards(Ritual), nil, 1)

	if aAfter.BonusAP != ritualBonusAP {
		t.Errorf("BonusAP after a Ritual = %d, want %d", aAfter.BonusAP, ritualBonusAP)
	}
	if aAfter.ActionPoints() != baseActionPoints+ritualBonusAP {
		t.Errorf("next round's budget = %d, want %d", aAfter.ActionPoints(), baseActionPoints+ritualBonusAP)
	}
}

func TestBraceHalvesOneAttackAndIsSpent(t *testing.T) {
	// Brace is partial where Dodge is binary, and single where Guard is turn-wide. Two attacks
	// into one brace land as half, then full.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	// B braces first. B acts second, so its brace is standing when A's next turn arrives.
	_, a, b = ResolveRound(a, b, nil, PlainCards(Brace), 1)
	events, _, bAfter := ResolveRound(a, b, PlainCards(Strike, Strike), nil, 2)

	if !hasKind(events, KindBraced) {
		t.Fatal("no KindBraced event — the brace did not apply")
	}
	if n := kindCount(events, KindBraced); n != 1 {
		t.Errorf("KindBraced fired %d times, want 1 — a brace is spent on one blow", n)
	}

	// Str 10: half a Strike is 5, then a full one is 10.
	if want := 100 - 5 - 10; bAfter.CurrentLife != want {
		t.Errorf("life after a braced Strike then a clean one = %d, want %d", bAfter.CurrentLife, want)
	}
	if bAfter.DefendCount != 0 {
		t.Errorf("defends left = %d, want 0 — the brace was spent on the first blow", bAfter.DefendCount)
	}
}

func TestBraceAndGuardBothApply(t *testing.T) {
	// Two cards bought separately, so both bite: a quarter, not a half. A rule that ignored one
	// would make the cheaper card worthless exactly when the player had committed to both.
	a := duelist(20, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, PlainCards(Brace, Guard), 1)
	_, _, bAfter := ResolveRound(a, b, PlainCards(Strike), nil, 2)

	if want := 100 - 20/braceDivisor/guardDivisor; bAfter.CurrentLife != want {
		t.Errorf("life after a Strike into brace+guard = %d, want %d (quartered)", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsARiposteWithoutTakingTheCounter(t *testing.T) {
	// The whole point of the card. Attacking into a Riposte normally costs the attacker str/2;
	// a Feint clears it and pays nothing for the privilege.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, PlainCards(Riposte), 1)
	events, aAfter, bAfter := ResolveRound(a, b, PlainCards(Feint), nil, 2)

	if !hasKind(events, KindStripped) {
		t.Fatal("no KindStripped event — the feint did not remove the riposte")
	}
	if hasKind(events, KindNegated) {
		t.Error("a stripped riposte must not also negate: the feint should land")
	}
	if aAfter.CurrentLife != 100 {
		t.Errorf("attacker took %d counter-damage, want 0 — a feint takes no riposte", 100-aAfter.CurrentLife)
	}
	if want := 100 - Feint.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d — the feint should land after stripping", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsWhicheverDefendWasRaisedFirst(t *testing.T) {
	// **The strip follows raise order, not a precedence table.** It picked Ripostes over Dodges
	// while the defences were four independent counters with nothing to say which came first;
	// now that they queue, the card the defender put in front is the card that pays. Both
	// orderings are checked, because a test that only queued one of them would pass against a
	// hard-coded answer.
	for _, tc := range []struct {
		raised []Card
		want   ActionKind
	}{
		{PlainCards(Dodge, Riposte), Dodge},
		{PlainCards(Riposte, Dodge), Riposte},
		{PlainCards(Brace, Riposte), Brace},
	} {
		a := duelist(10, 0, 100)
		b := duelist(10, 0, 100)

		_, a, b = ResolveRound(a, b, nil, tc.raised, 1)
		events, _, _ := ResolveRound(a, b, PlainCards(Feint), nil, 2)

		stripped, ok := firstStripped(events)
		if !ok {
			t.Fatalf("%v: no KindStripped event", tc.raised)
		}
		if stripped != tc.want {
			t.Errorf("%v: feint stripped a %v, want the %v raised first", tc.raised, stripped, tc.want)
		}
	}
}

// firstStripped is the defend card the first KindStripped event names.
func firstStripped(events []Event) (ActionKind, bool) {
	for _, e := range events {
		if e.Kind == KindStripped {
			return e.Action, true
		}
	}
	return 0, false
}

func TestFeintWithNothingToStripIsStillAnAttack(t *testing.T) {
	// It must never be dead. With no negation up it is an overpriced Strike, not a no-op.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	events, _, bAfter := ResolveRound(a, b, PlainCards(Feint), nil, 1)

	if hasKind(events, KindStripped) {
		t.Error("stripped something that was not there")
	}
	if want := 100 - Feint.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d", bAfter.CurrentLife, want)
	}
}

func TestFeintStripIsUnconditional(t *testing.T) {
	// The strip fires whatever is about to happen to the blow. Making it depend on a card the
	// player cannot see would put a hidden interaction into a game whose whole point is reading
	// the opponent — so a Retreat with charges to spare still loses one to a Feint that it then
	// goes on to negate.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, PlainCards(Retreat), 1)
	events, _, bAfter := ResolveRound(a, b, PlainCards(Feint, Jab, Jab, Jab), nil, 2)

	if !hasKind(events, KindStripped) {
		t.Error("the feint's strip did not fire against a Retreat — it is meant to be unconditional")
	}

	// **The count is the assertion, because the charges cannot be read afterwards**: B's own turn
	// comes next and expires whatever is left. Three charges, one spent by the strip, so two
	// attacks are stopped and two land — where an unstripped Retreat would have stopped three.
	if n := kindCount(events, KindNegated); n != retreatCharges-1 {
		t.Errorf("negated %d attacks, want %d — the strip should have cost a charge",
			n, retreatCharges-1)
	}
	if want := 100 - 2*Jab.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d", bAfter.CurrentLife, want)
	}
}

func TestRetreatStopsThreeAttacksAndThenIsSpent(t *testing.T) {
	// Dodge stops one blow for two points; Retreat stops three for four. That is the tier: it
	// buys volume of negation, which is what makes it the answer to a swarm where a Dodge is the
	// answer to one big swing.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, PlainCards(Retreat), 1)
	events, _, bAfter := ResolveRound(a, b, PlainCards(Jab, Jab, Jab, Jab), nil, 2)

	if n := kindCount(events, KindNegated); n != retreatCharges {
		t.Errorf("negated %d attacks, want %d", n, retreatCharges)
	}
	if want := 100 - Jab.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("life after four Jabs into a Retreat = %d, want %d — the fourth lands",
			bAfter.CurrentLife, want)
	}
	if bAfter.DefendCount != 0 {
		t.Errorf("%d defends left, want 0 — the retreat is spent after three", bAfter.DefendCount)
	}
}

func TestRetreatCostsTheAttackerNothing(t *testing.T) {
	// **A Retreat is a wall, not a counter.** Riposte is the defend card that hits back and it
	// pays a point less for the privilege of stopping only one blow; a Retreat that also
	// punished the attacker would leave Riposte with nothing to sell.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = ResolveRound(a, b, nil, PlainCards(Retreat), 1)
	_, aAfter, _ := ResolveRound(a, b, PlainCards(Strike, Strike), nil, 2)

	if aAfter.CurrentLife != 100 {
		t.Errorf("attacker took %d damage into a Retreat, want 0", 100-aAfter.CurrentLife)
	}
}

func TestDefensesAnswerInTheOrderTheyWereRaised(t *testing.T) {
	// The rule the whole ordered list exists for. A Brace queued in front of a Dodge answers the
	// first attack — halving it — and the Dodge stops the second; reversing the queue reverses
	// which blow is stopped dead. Under the old fixed precedence the Dodge always went first and
	// the order the player chose meant nothing.
	str := 10
	for _, tc := range []struct {
		raised []Card
		want   int // damage taken across two Strikes
	}{
		{PlainCards(Brace, Dodge), Strike.Damage(str) / braceDivisor},
		{PlainCards(Dodge, Brace), Strike.Damage(str) / braceDivisor},
	} {
		a := duelist(str, 0, 100)
		b := duelist(str, 0, 100)

		_, a, b = ResolveRound(a, b, nil, tc.raised, 1)
		events, _, bAfter := ResolveRound(a, b, PlainCards(Strike, Strike), nil, 2)

		if got := 100 - bAfter.CurrentLife; got != tc.want {
			t.Errorf("%v: took %d damage, want %d", tc.raised, got, tc.want)
		}

		// Which blow each card answered is the part that actually differs, and it is what a
		// player reading the Resolution feed sees.
		braced, negated := -1, -1
		for i, e := range events {
			if e.Kind == KindBraced && braced < 0 {
				braced = i
			}
			if e.Kind == KindNegated && negated < 0 {
				negated = i
			}
		}
		if braced < 0 || negated < 0 {
			t.Fatalf("%v: both defences should have fired, got braced=%d negated=%d",
				tc.raised, braced, negated)
		}
		if first := tc.raised[0].Action; (first == Brace) != (braced < negated) {
			t.Errorf("%v: %v was raised first but did not answer the first attack", tc.raised, first)
		}
	}
}

func TestClearDefensesClearsEveryDefensiveField(t *testing.T) {
	// ClearDefenses is the one place that has to know about every defensive field, and a new one
	// left out of it would stand forever — the worst failure mode available here, and a silent
	// one. Pin the whole set rather than only the fields this session added.
	d := Duelist{Guarded: true}
	for _, k := range []ActionKind{Brace, Dodge, Riposte, Retreat} {
		d = d.raiseDefend(k)
	}

	got := ClearDefenses(d)

	if got.Guarded || got.DefendCount != 0 || got.Defends != ([maxPendingDefends]PendingDefend{}) {
		t.Errorf("ClearDefenses left something standing: %+v", got)
	}
}

func TestTheConceptGridIsThreeByFourAndFilled(t *testing.T) {
	// Three categories by four cost tiers, one concept per cell. This is the structural claim
	// MECHANICS.md makes about the card set, and it is the thing that quietly breaks the first
	// time somebody adds a fifth attack or a second 2-AP defence.
	//
	// It also catches a concept falling through Cost()'s default arm, which returns a Strike's
	// price — a mistake that would otherwise hide behind an entirely plausible number.
	tiers := map[Category]map[int]ActionKind{}

	for _, a := range AllActions {
		if a.Cost() < 1 || a.Cost() > 4 {
			t.Errorf("%v costs %d, outside the 1-4 tiers the grid is built on", a, a.Cost())
		}

		cat := a.Category()
		if tiers[cat] == nil {
			tiers[cat] = map[int]ActionKind{}
		}
		if other, taken := tiers[cat][a.Cost()]; taken {
			t.Errorf("%v and %v are both %v at %d AP — the grid holds one concept per cell", other, a, cat, a.Cost())
		}
		tiers[cat][a.Cost()] = a
	}

	for _, cat := range Categories() {
		if len(tiers[cat]) != 4 {
			t.Errorf("%v has %d of 4 cost tiers filled", cat, len(tiers[cat]))
		}
	}
}

func TestParseActionRoundTripsEveryConcept(t *testing.T) {
	// cards.json names concepts by string, so this is the join between the deck data and the
	// rules. A concept whose name does not round-trip is five cards missing from the deck.
	for _, a := range AllActions {
		got, ok := ParseAction(a.String())
		if !ok {
			t.Errorf("ParseAction(%q) failed for a concept the rules define", a.String())
			continue
		}
		if got != a {
			t.Errorf("ParseAction(%q) = %v, want %v", a.String(), got, a)
		}
	}

	if _, ok := ParseAction("Fient"); ok {
		t.Error("ParseAction accepted a typo — it must report failure so a bad deck fails loudly")
	}
}
