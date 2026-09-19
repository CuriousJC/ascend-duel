package combat

import "testing"

// The heal-share verb: a relic putting life back at the top of its wearer's own turn.
//
// **The two halves that can break independently** are the arithmetic — a share of the maximum
// rather than of what is left — and the ordering, which is what makes the life arrive in time for
// the turn it is meant to survive rather than after it.

// regenRelic is a relic restoring pct of maximum life at the top of a turn.
func regenRelic(t *testing.T, key string, pct int) RelicID {
	t.Helper()

	return relic(t, key, RelicRule{
		When: MomentTurnStart,
		Then: []RelicEffect{{Do: DoHealShare, Amount: pct}},
	})
}

// regained is the life one side was given back over a whole round.
func regained(events []Event, side Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindRegenerated && e.Side == side {
			total += e.Amount
		}
	}
	return total
}

func TestARegenRelicRestoresAShareOfMaximumLife(t *testing.T) {
	id := regenRelic(t, "regen.share", 25)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 40
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)

	// **Of the maximum, not of what is left.** A quarter of 100 is 25; a quarter of the 40 they
	// were standing on would be 10, which is the reading this verb exists not to take.
	if got := regained(events, SideA); got != 25 {
		t.Errorf("a 25%% regen on a 100-life duelist gave back %d, wanted 25", got)
	}
}

func TestARegenIsCappedAtFullLifeAndSaysSoBySayingNothing(t *testing.T) {
	id := regenRelic(t, "regen.cap", 25)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 90
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)

	if got := regained(events, SideA); got != 10 {
		t.Errorf("a 25%% regen on a duelist 10 short of full gave back %d, wanted 10", got)
	}
	if after.CurrentLife != 100 {
		t.Errorf("a capped regen left %d, wanted 100", after.CurrentLife)
	}
}

func TestARegenOnFullLifeWritesNoBeat(t *testing.T) {
	id := regenRelic(t, "regen.full", 25)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)

	for _, e := range events {
		if e.Kind == KindRegenerated {
			t.Fatalf("a duelist at full life regained %d", e.Amount)
		}
	}
}

func TestARegenArrivesBeforeTheTurnItHasToSurvive(t *testing.T) {
	// **The ordering is the whole point of a turn-start moment.** The life has to be on before
	// anything else in the turn happens, so the event is the first this side writes in the round.
	id := regenRelic(t, "regen.first", 25)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 40
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)

	for _, e := range events {
		if e.Side != SideA || e.Kind == KindRoundStart {
			continue
		}
		if e.Kind != KindRegenerated {
			t.Fatalf("the first thing side A did was %d, not the regen", e.Kind)
		}
		break
	}
}

func TestARegenFiresOnAnEmptyTurn(t *testing.T) {
	// **A turn taken with nothing in it is still a turn**, which is the rule turn-taken is already
	// under. A relic that only paid when you swung would be a different relic.
	id := regenRelic(t, "regen.empty", 25)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: id})
	a.CurrentLife = 40
	b := duelist(0, 0, 500)

	events, after, _ := resolve(a, b, nil, nil, 1)

	if got := regained(events, SideA); got != 25 {
		t.Errorf("an empty turn regained %d, wanted 25", got)
	}
	if after.CurrentLife != 65 {
		t.Errorf("life went 40 to %d, wanted 65", after.CurrentLife)
	}
}

func TestTwoRegenRelicsAddRatherThanCompound(t *testing.T) {
	first := regenRelic(t, "regen.add1", 10)
	second := regenRelic(t, "regen.add2", 20)

	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: first}).Wearing(WornRelic{Relic: second})
	a.CurrentLife = 10
	b := duelist(0, 0, 500)

	events, _, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)

	if got := regained(events, SideA); got != 30 {
		t.Errorf("10%% and 20%% of 100 gave back %d, wanted 30", got)
	}
}

func TestATurnStartRuleMayNotCarryAPredicate(t *testing.T) {
	// **This moment has no card and no turn to read.** A rule that quietly matched everything is
	// the silent-rule failure readsACard exists to refuse.
	_, err := RegisterRelic("relictest.regen.conditioned", "conditioned", []RelicRule{{
		When: MomentTurnStart,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoHealShare, Amount: 25}},
	}})
	if err == nil {
		t.Fatal("a conditioned turn-start rule registered, and its predicate would mean nothing")
	}
}

func TestHealShareIsRefusedAtAnyOtherMoment(t *testing.T) {
	_, err := RegisterRelic("relictest.regen.wrongmoment", "wrong moment", []RelicRule{{
		When: MomentAttackLands,
		Then: []RelicEffect{{Do: DoHealShare, Amount: 25}},
	}})
	if err == nil {
		t.Fatal("heal-share at attack-lands registered, and it would never have fired")
	}
}
