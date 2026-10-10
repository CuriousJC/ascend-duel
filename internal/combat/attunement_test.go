package combat

import "testing"

// The Attunements and the Awakenings: a pierce that stops a fizzle, a surge per card played, and a
// first card that raises every attack queued behind it.

// TestAPierceLandsAHitThatWouldHaveFizzled. An ice Bash at an ice goblin lands for an ice-attuned
// duelist, and a fire one is untouched by an ice attunement.
func TestAPierceLandsAHitThatWouldHaveFizzled(t *testing.T) {
	attune := relic(t, "attune-ice", RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoPierceElement, Element: Ice}},
	})
	a := duelist(10, 8, 5000).Wearing(WornRelic{Relic: attune})

	events, _, after := resolve(a, creature(Ice, 10, 6, 5000), []Card{Of(Bash, Ice)}, nil, 1)
	if n := kindCount(events, KindFizzled); n != 0 {
		t.Errorf("an attuned ice Bash fizzled on an ice creature")
	}
	if after.CurrentLife == 5000 {
		t.Errorf("an attuned ice Bash landed nothing on an ice creature")
	}

	events, _, _ = resolve(a, creature(Fire, 10, 6, 5000), []Card{Of(Bash, Fire)}, nil, 1)
	if n := kindCount(events, KindFizzled); n != 1 {
		t.Errorf("an ice attunement stopped a fire Bash fizzling on a fire creature")
	}
}

// TestAPierceOfBasicIsRefused. A basic card never fizzles, so the relic would do nothing.
func TestAPierceOfBasicIsRefused(t *testing.T) {
	if _, err := RegisterRelic("relictest.attune-basic", "x", []RelicRule{{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoPierceElement}},
	}}); err == nil {
		t.Error("a pierce of basic registered")
	}
}

// TestASurgePerCardBanksForTheNextTurnOnly. Two slashes and a Bash bank two points for the next
// turn, whatever the hits did, and the turn after that spends them.
func TestASurgePerCardBanksForTheNextTurnOnly(t *testing.T) {
	attune := relic(t, "attune-slash", RelicRule{
		When: MomentTurnTaken,
		If:   RelicCondition{Form: FormSlash, HasForm: true},
		Then: []RelicEffect{{Do: DoSurgePerCard, Amount: 1}},
	})
	a := duelist(10, 8, 5000).Wearing(WornRelic{Relic: attune})

	// The goblin is ice, so the ice Cut fizzles — and is still counted.
	_, a1, _ := resolve(a, creature(Ice, 10, 6, 5000),
		[]Card{Of(Cut, Ice), Of(Slice, Fire), Of(Bash, Fire)}, nil, 1)
	if a1.Surge != 2 || a1.ActionPoints() != a.Actions+2 {
		t.Errorf("two slashes banked %d AP, want 2", a1.Surge)
	}

	_, a2, _ := resolve(a1, creature(Ice, 10, 6, 5000), nil, nil, 2)
	if a2.Surge != 0 {
		t.Errorf("the slash surge outlived the turn it bought: %d", a2.Surge)
	}
}

// TestASurgePerCardNamesSomethingToCount. With no `If` there is nothing to count.
func TestASurgePerCardNamesSomethingToCount(t *testing.T) {
	if _, err := RegisterRelic("relictest.surge-bare", "x", []RelicRule{{
		When: MomentTurnTaken,
		Then: []RelicEffect{{Do: DoSurgePerCard, Amount: 1}},
	}}); err == nil {
		t.Error("a surge per card with no If registered")
	}
}

func awakening(t *testing.T, key string, cond RelicCondition) RelicID {
	cond.First = true
	return relic(t, key, RelicRule{
		When: MomentBlowFormed,
		If:   cond,
		Then: []RelicEffect{{Do: DoAwaken, Amount: 10}},
	})
}

// TestAnAwakeningRaisesEveryAttackAfterTheFirstCard. A fire opener: the cards behind it each gain
// 10 before the hand multiplies, the opener gains nothing, and a defense gains nothing.
func TestAnAwakeningRaisesEveryAttackAfterTheFirstCard(t *testing.T) {
	flame := awakening(t, "awaken-fire", RelicCondition{Element: Fire, HasElement: true})
	turn := []Card{Of(Bash, Fire), Of(Jab, Ice), Of(Brace, Ice), Of(Cut, Earth)}

	bare, _, _ := resolve(duelist(10, 9, 5000), duelist(10, 6, 5000), turn, nil, 1)
	woke, _, _ := resolve(duelist(10, 9, 5000).Wearing(WornRelic{Relic: flame}),
		duelist(10, 6, 5000), turn, nil, 1)
	b, w := handEventOf(t, bare, SideA), handEventOf(t, woke, SideA)

	slots := ResolutionOrder(turn, nil)
	if w.Awaken != 10 || !w.AwakenSeats[0] || slots[w.AwakenOpener].Index != 0 {
		t.Fatalf("the hand says it woke %d from seats %v opened by seat %d, want 10 from seat 0 opened by the first card",
			w.Awaken, w.AwakenSeats, w.AwakenOpener)
	}
	for n := 0; n < w.HandCardCount; n++ {
		queued := slots[w.HandCards[n]].Index
		want := 0
		if queued > 0 && turn[queued].Spec().Verb == VerbAttack {
			want = 10
		}
		if w.HandAwaken[n] != want {
			t.Errorf("term %d (queued %d) woke %d, want %d", n, queued, w.HandAwaken[n], want)
		}
		if w.HandAmounts[n] != b.HandAmounts[n]+want {
			t.Errorf("term %d came to %d, want the bare %d plus %d", n, w.HandAmounts[n], b.HandAmounts[n], want)
		}
		if got := scaleDamage(w.HandAmounts[n], w.Multiplier); w.HitAmounts[n] != got {
			t.Errorf("term %d's hit is %d, want the hand's multiplier over its term: %d", n, w.HitAmounts[n], got)
		}
	}
}

// TestAnAwakeningReadsOnlyTheFirstCard. A fire card second wakes nothing, and neither does a turn
// opening on a defense of another element — the first card is literally the first.
func TestAnAwakeningReadsOnlyTheFirstCard(t *testing.T) {
	flame := awakening(t, "awaken-fire-first", RelicCondition{Element: Fire, HasElement: true})
	a := duelist(10, 9, 5000).Wearing(WornRelic{Relic: flame})

	for _, turn := range [][]Card{
		{Of(Jab, Ice), Of(Bash, Fire)},
		{Of(Brace, Ice), Of(Bash, Fire), Of(Jab, Fire)},
	} {
		events, _, _ := resolve(a, duelist(10, 6, 5000), turn, nil, 1)
		if e := handEventOf(t, events, SideA); e.Awaken != 0 {
			t.Errorf("a turn opening on %v woke %d", turn[0], e.Awaken)
		}
	}

	// A fire Brace first does wake it: the opener need not be an attack.
	events, _, _ := resolve(a, duelist(10, 6, 5000), []Card{Of(Brace, Fire), Of(Jab, Ice)}, nil, 1)
	if e := handEventOf(t, events, SideA); e.Awaken != 10 {
		t.Errorf("a fire Brace opener woke %d, want 10", e.Awaken)
	}
}

// TestFirstIsOnlyAnAwakeningsPredicate. It is read at blow-formed alone, and only by awaken; an
// awaken without it would raise every card but one for no reason a player could see.
func TestFirstIsOnlyAnAwakeningsPredicate(t *testing.T) {
	for name, rule := range map[string]RelicRule{
		"first-elsewhere": {When: MomentCardDamage, If: RelicCondition{First: true},
			Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}}},
		"first-on-echo": {When: MomentBlowFormed, If: RelicCondition{First: true},
			Then: []RelicEffect{{Do: DoEchoAttack, Amount: 2}}},
		"awaken-bare": {When: MomentBlowFormed,
			Then: []RelicEffect{{Do: DoAwaken, Amount: 10}}},
	} {
		if _, err := RegisterRelic("relictest."+name, name, []RelicRule{rule}); err == nil {
			t.Errorf("%s registered", name)
		}
	}
}

// TestARelicPricingTheCardMultipliesTheAwakening. The 10 joins the card before its relics, so a
// 2x on the card doubles it too.
func TestARelicPricingTheCardMultipliesTheAwakening(t *testing.T) {
	flame := awakening(t, "awaken-fire-priced", RelicCondition{Element: Fire, HasElement: true})
	club := relic(t, "club-earth", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Element: Earth, HasElement: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	turn := []Card{Of(Bash, Fire), Of(Jab, Earth)}

	bare, _, _ := resolve(duelist(10, 9, 5000).Wearing(WornRelic{Relic: club}), duelist(10, 6, 5000), turn, nil, 1)
	woke, _, _ := resolve(duelist(10, 9, 5000).Wearing(WornRelic{Relic: flame}).Wearing(WornRelic{Relic: club}),
		duelist(10, 6, 5000), turn, nil, 1)
	b, w := handEventOf(t, bare, SideA), handEventOf(t, woke, SideA)
	if w.HandAmounts[1] != b.HandAmounts[1]+20 {
		t.Errorf("the earth Jab came to %d, want the bare %d plus the 10 doubled", w.HandAmounts[1], b.HandAmounts[1])
	}
}
