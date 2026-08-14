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

	_, aAfter, _ := resolve(a, b, PlainCards(Ritual), nil, 1)

	if aAfter.BonusAP != ritualBonusAP {
		t.Errorf("BonusAP after a Ritual = %d, want %d", aAfter.BonusAP, ritualBonusAP)
	}
	if aAfter.ActionPoints() != baseActionPoints+ritualBonusAP {
		t.Errorf("next round's budget = %d, want %d", aAfter.ActionPoints(), baseActionPoints+ritualBonusAP)
	}
}

// Brace is partial where Dodge is binary, and cheap where Retreat is total. Under one blow per
// turn it takes half of what arrives and is then spent.
func TestBraceHalvesOneAttackAndIsSpent(t *testing.T) {
	a := duelist(10, 0, 500)
	b := duelist(10, 0, 500)

	// B braces first. B acts second, so its brace is standing when A's next turn arrives.
	// A Heavy and a Strike form no hand, so the blow is the Heavy alone and the arithmetic is
	// about the brace rather than about a multiplier.
	_, a, b = resolve(a, b, nil, PlainCards(Brace), 1)
	events, _, bAfter := resolve(a, b, PlainCards(Heavy, Strike), nil, 2)

	if !hasKind(events, KindBraced) {
		t.Fatal("no KindBraced event - the brace did not apply")
	}
	if want := 500 - Heavy.Damage(10)*(100-braceReductionPct)/100; bAfter.CurrentLife != want {
		t.Errorf("life after a braced Heavy = %d, want %d", bAfter.CurrentLife, want)
	}
	if bAfter.DefendCount != 0 {
		t.Errorf("defends left = %d, want 0 - every defence is spent on the blow it answered", bAfter.DefendCount)
	}
}

// Two cards bought separately, so both bite. A rule that ignored one would make the cheaper card
// worthless exactly when the player had committed to both.
func TestBraceAndGuardBothApply(t *testing.T) {
	a := duelist(20, 0, 500)
	b := duelist(10, 0, 500)

	_, a, b = resolve(a, b, nil, PlainCards(Brace, Guard), 1)
	_, _, bAfter := resolve(a, b, PlainCards(Strike), nil, 2)

	braced := Strike.Damage(20) * (100 - braceReductionPct) / 100
	if want := 500 - braced/guardDivisor; bAfter.CurrentLife != want {
		t.Errorf("life after a Strike into brace+guard = %d, want %d", bAfter.CurrentLife, want)
	}
}

func TestFeintStripsARiposteWithoutTakingTheCounter(t *testing.T) {
	// The whole point of the card. Attacking into a Riposte normally costs the attacker str/2;
	// a Feint clears it and pays nothing for the privilege.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = resolve(a, b, nil, PlainCards(Riposte), 1)
	events, aAfter, bAfter := resolve(a, b, PlainCards(Feint), nil, 2)

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

// **The strip takes the biggest reduction, not the first card raised.** Raise order was retired
// when a turn stopped having more than one blow for the order to matter to; taking the strongest
// is what keeps Feint worth its 3 AP instead of spending it on whichever 1-point Brace happened
// to be queued first. Both orderings are checked, because a test that only queued one of them
// would pass against a hard-coded answer.
func TestFeintStripsTheStrongestDefend(t *testing.T) {
	for _, tc := range []struct {
		raised []Card
		want   ActionKind
	}{
		{PlainCards(Dodge, Retreat), Retreat},
		{PlainCards(Retreat, Dodge), Retreat},
		{PlainCards(Brace, Dodge), Dodge},
		{PlainCards(Dodge, Brace), Dodge},
	} {
		a := duelist(10, 0, 500)
		b := duelist(10, 0, 500)

		_, a, b = resolve(a, b, nil, tc.raised, 1)
		events, _, _ := resolve(a, b, PlainCards(Feint), nil, 2)

		stripped, ok := firstStripped(events)
		if !ok {
			t.Fatalf("%v: no KindStripped event", tc.raised)
		}
		if stripped != tc.want {
			t.Errorf("%v: feint stripped a %v, want the %v", tc.raised, stripped, tc.want)
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

	events, _, bAfter := resolve(a, b, PlainCards(Feint), nil, 1)

	if hasKind(events, KindStripped) {
		t.Error("stripped something that was not there")
	}
	if want := 100 - Feint.Damage(10); bAfter.CurrentLife != want {
		t.Errorf("defender life = %d, want %d", bAfter.CurrentLife, want)
	}
}

// The strip fires whatever is about to happen to the blow. Making it depend on a card the player
// cannot see would put a hidden interaction into a game whose whole point is reading the
// opponent, so a Retreat that would have stopped the blow outright still loses to it.
func TestFeintStripIsUnconditional(t *testing.T) {
	a := duelist(10, 0, 500)
	b := duelist(10, 0, 500)

	_, a, b = resolve(a, b, nil, PlainCards(Retreat), 1)
	events, _, bAfter := resolve(a, b, PlainCards(Feint), nil, 2)

	if !hasKind(events, KindStripped) {
		t.Error("the feint's strip did not fire against a Retreat - it is meant to be unconditional")
	}
	if bAfter.CurrentLife == 500 {
		t.Error("the Retreat was stripped, so the feint should have landed")
	}
}

// **Retreat is the one defence that still reaches zero.** Dodge takes three quarters for two
// points; Retreat takes all of it for four, which is most of a round spent doing nothing else.
// That is the tier, and it replaced the three-charge version when a turn stopped having three
// attacks to spend charges on.
func TestRetreatStopsTheBlowOutright(t *testing.T) {
	a := duelist(10, 0, 500)
	b := duelist(10, 0, 500)

	_, a, b = resolve(a, b, nil, PlainCards(Retreat), 1)
	_, _, bAfter := resolve(a, b, PlainCards(Jab, Jab, Jab, Jab), nil, 2)

	if bAfter.CurrentLife != 500 {
		t.Errorf("life after a Jab Barrage into a Retreat = %d, want 500 - it stops the blow outright",
			bAfter.CurrentLife)
	}
	if bAfter.DefendCount != 0 {
		t.Errorf("%d defends left, want 0 - the retreat is spent on the blow it answered", bAfter.DefendCount)
	}
}

func TestRetreatCostsTheAttackerNothing(t *testing.T) {
	// **A Retreat is a wall, not a counter.** Riposte is the defend card that hits back and it
	// pays a point less for the privilege of stopping only one blow; a Retreat that also
	// punished the attacker would leave Riposte with nothing to sell.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	_, a, b = resolve(a, b, nil, PlainCards(Retreat), 1)
	_, aAfter, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 2)

	if aAfter.CurrentLife != 100 {
		t.Errorf("attacker took %d damage into a Retreat, want 0", 100-aAfter.CurrentLife)
	}
}

func TestEveryRaisedDefenceMeetsTheBlow(t *testing.T) {
	// **Every card fires, not just the front one** *(2026-08-14)*. The defends were an ordered
	// queue for a day, on the theory that a player would choose which card met which blow; a turn
	// now lands one blow, so the choice had nothing to choose between and the whole set answers
	// it. Each card also emits its own event, which is what the Resolution feed narrates.
	str := 10
	a := duelist(str, 0, 100)
	b := duelist(str, 0, 100)

	_, a, b = resolve(a, b, nil, PlainCards(Brace, Dodge), 1)
	events, _, _ := resolve(a, b, PlainCards(Heavy, Strike), nil, 2)

	var braced, negated int
	for _, e := range events {
		switch e.Kind {
		case KindBraced:
			braced++
		case KindNegated:
			negated++
		}
	}
	if braced != 1 {
		t.Errorf("the Brace fired %d times, want 1", braced)
	}
	if negated != 1 {
		t.Errorf("the Dodge fired %d times, want 1", negated)
	}
}

func TestDefencesAreSpentWhetherOrNotTheyWereNeeded(t *testing.T) {
	// A defend answers the opponent's turn and then goes, spent or not. It cannot be banked
	// against a turn that queued no attack at all — the same rule Guard follows.
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	// A raises two defends into a turn with nothing to answer.
	_, a1, b1 := resolve(a, b, PlainCards(Dodge, Brace), nil, 1)
	if a1.DefendCount != 2 {
		t.Fatalf("A ended round 1 holding %d defends, want 2 unspent", a1.DefendCount)
	}

	// Round two: A queues nothing, so its own turn expires them before B swings.
	events, _, _ := resolve(a1, b1, nil, PlainCards(Strike, Strike), 2)

	open, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike), 1)
	if got, want := firstDamage(t, events, SideB).Amount, firstDamage(t, open, SideB).Amount; got != want {
		t.Errorf("a blow into expired defends dealt %d, want the full %d", got, want)
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
