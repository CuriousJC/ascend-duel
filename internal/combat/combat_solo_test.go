package combat

import "testing"

// A solo attacker's turn: **every attack resolves on its own, in order**, with no hand read off
// the set. It is what an enemy is as of 2026-08-17 — see Duelist.SoloAttacks.
//
// These are the properties the hand-forming phase does *not* have, so nothing above covers them: one
// damage event per card rather than one per turn, no hand event at all, and a defence that has to
// survive long enough to answer every blow of the turn instead of just the first.

// soloist is a duelist whose attack cards form no hands.
func soloist(dmg, actions, life int) Duelist {
	d := duelist(dmg, actions, life)
	d.SoloAttacks = true
	return d
}

func handCount(events []Event) int {
	n := 0
	for _, e := range events {
		if e.Kind == KindHand {
			n++
		}
	}
	return n
}

func TestASoloAttackerSwingsOncePerCard(t *testing.T) {
	a := soloist(10, 9, 500)
	b := duelist(10, 5, 500)

	events, _, after := resolve(a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	if n := damageCount(events); n != 3 {
		t.Errorf("three Strikes produced %d damage events, want 3 — one per card", n)
	}
	if n := handCount(events); n != 0 {
		t.Errorf("a solo attacker formed %d hands, want none", n)
	}

	// **The face damage and nothing else.** The hand-forming version of this turn is a Strike Flurry:
	// the same three cards plus DMG times the multiplier, which is the whole of what this removes.
	if want := 500 - 3*ConceptOf(Strike).Amount*a.DMG/100; after.CurrentLife != want {
		t.Errorf("three Strikes left %d life, want %d — the sum of the cards' own damage",
			after.CurrentLife, want)
	}
}

func TestASoloAttackerLandsItsCardsInQueueOrder(t *testing.T) {
	// A ladder of three different cards, so the order is readable off the figures. Jab is the
	// 0.5x rung and Smash the 2x one.
	a := soloist(10, 9, 500)
	b := duelist(10, 5, 500)

	events, _, _ := resolve(a, b, PlainCards(Smash, Jab, Strike), nil, 1)

	var got []int
	for _, e := range events {
		if e.Kind == KindDamage {
			got = append(got, e.Amount)
		}
	}

	want := []int{
		Plain(Smash).Damage(a.DMG),
		Plain(Jab).Damage(a.DMG),
		Plain(Strike).Damage(a.DMG),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("blow %d dealt %d, want %d — the queue's order is what lands", i, got[i], want[i])
		}
	}
}

func TestADefendAnswersEverySwingOfASoloTurn(t *testing.T) {
	// **The rule this exists to hold.** A defence covers one opposing *turn*; spending it on the
	// first blow would make it nearly worthless against the only opponents that swing more than
	// once, which is every enemy in the game.
	b := soloist(10, 9, 500)
	a := duelist(10, 5, 500)

	open, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike), 1)
	shielded, after, _ := resolve(a, b, PlainCards(testGuard), PlainCards(Strike, Strike), 1)

	cut := 100 - ConceptOf(testGuard).Amount
	for i, e := range damages(shielded) {
		want := damages(open)[i].Amount * cut / 100
		if e.Amount != want {
			t.Errorf("blow %d through a testGuard dealt %d, want %d", i, e.Amount, want)
		}
	}

	// And it is gone once the turn it covered is over, exactly as one blow would have spent it.
	if after.DefendCount != 0 {
		t.Errorf("the testGuard survived the turn it answered, holding %d", after.DefendCount)
	}
}

func TestASoloAttackerStopsAtTheKill(t *testing.T) {
	// Three swings into a duelist that cannot take two: the third must not be swung at a corpse,
	// which is the same rule the round-level defeat check follows.
	a := soloist(10, 9, 500)
	b := duelist(10, 5, 15)

	events, _, after := resolve(a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	if after.CurrentLife != 0 {
		t.Fatalf("the target ended on %d life, want 0", after.CurrentLife)
	}
	if n := damageCount(events); n != 2 {
		t.Errorf("%d blows landed on a target that died to the second, want 2", n)
	}
}

func TestAShockedSoloAttackerMissesWithEverything(t *testing.T) {
	// **One roll for the turn, not one per card.** A shock is "the turn's attack misses", and a
	// roll per card would change what the status means as well as how far the package's one random
	// stream advances in a round.
	a := soloist(10, 9, 500)
	a.Statuses[statusOf(Lightning)] = Status{Amount: 50, Rounds: 2}
	b := duelist(10, 5, 500)

	events, _, after := resolveWith(alwaysMisses(), a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	misses := 0
	for _, e := range events {
		if e.Kind == KindMissed {
			misses++
		}
	}
	if misses != 3 {
		t.Errorf("%d cards reported a miss, want 3 — each says what became of it", misses)
	}
	if n := damageCount(events); n != 0 {
		t.Errorf("%d blows landed through a shock that missed, want 0", n)
	}
	if after.CurrentLife != 500 {
		t.Errorf("the target lost %d life to a missed turn", 500-after.CurrentLife)
	}
}

func TestEverySoloAttackAnnouncesItself(t *testing.T) {
	// One beat per slot is what playback counts to know which card is lit — see
	// TestEverySlotIsEitherTakenOrChilled, which the hand-forming phase is held to for the same
	// reason. It has to hold when the blows are separate too.
	a := soloist(10, 9, 500)
	b := duelist(10, 5, 500)

	cards := PlainCards(Strike, Jab, Brace)
	events, _, _ := resolve(a, b, cards, nil, 1)

	actions := 0
	for _, e := range events {
		if e.Kind == KindAction && e.Side == SideA {
			actions++
		}
	}
	if actions != len(cards) {
		t.Errorf("%d cards announced themselves out of %d queued", actions, len(cards))
	}
}

func TestASoloPlannerTakesTheMostDamageItCanAfford(t *testing.T) {
	// The hand-forming planner would rather have three cheap copies of one card, because three of a
	// kind is a Flurry. A solo planner has no multiplier to chase, so it wants the biggest total —
	// and with the budget for it, that is the expensive card plus whatever still fits.
	d := soloist(10, 4, 100)
	hand := PlainCards(Smash, Jab, Jab, Jab)

	plan := PlanFor(d, hand)

	got := 0
	for _, c := range plan {
		got += c.Damage(d.DMG)
	}

	// Smash is 3 AP and Jab 1, so a 4-point budget buys Smash and one Jab: 20 + 5. Three Jabs
	// would be 15, which is the plan the hand table used to prefer.
	if want := Plain(Smash).Damage(d.DMG) + Plain(Jab).Damage(d.DMG); got != want {
		t.Errorf("the plan lands %d damage, want %d — %v", got, want, planKey(plan))
	}
}

// damages is every damage event in a log, in order.
func damages(events []Event) []Event {
	var out []Event
	for _, e := range events {
		if e.Kind == KindDamage {
			out = append(out, e)
		}
	}
	return out
}
