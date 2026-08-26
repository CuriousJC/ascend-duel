package combat

import "testing"

// Reordering a worn row, and the badge figure a growing ring reports.
//
// **Worn order is the order rings fire in**, so a reorder is a rules change and not an arrangement.
// These hold the two halves the screens now depend on: that the move is a permutation which carries
// each accumulator with its own ring, and that GrowthEffect reports the effect the accumulator
// feeds rather than the step that feeds it.

// growing is a ring that scales damage and grows on every win — the Heart/Enflamed shape, with the
// numeric effect first and the growth verb second, which is how rings.json writes them.
func growing(t *testing.T, key string, amount, step int) RingID {
	t.Helper()

	return ring(t, key,
		RingRule{When: MomentCardDamage, Then: []RingEffect{{Do: DoScaleDamage, Amount: amount}}},
		RingRule{When: MomentFightWon, Then: []RingEffect{{Do: DoGrowOnWin, Amount: step}}})
}

func TestMovingARingReordersTheRow(t *testing.T) {
	a, b, c := growing(t, "move-a", 100, 5), growing(t, "move-b", 100, 5), growing(t, "move-c", 100, 5)

	d := duelist(10, 5, 100).
		Wearing(WornRing{Ring: a}).
		Wearing(WornRing{Ring: b}).
		Wearing(WornRing{Ring: c})

	// The last ring to the front, which is the drag a player makes to give a multiplier the last
	// word — everything after it now compounds on top.
	moved := d.MoveRing(2, 0)

	want := []RingID{c, a, b}
	got := moved.WornRings()
	if len(got) != len(want) {
		t.Fatalf("wearing %d rings after the move, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].Ring != id {
			t.Errorf("seat %d holds %v, want %v", i, RingOf(got[i].Ring).Key, RingOf(id).Key)
		}
	}
}

// **The number belongs to the ring and not to the finger**, which is the property that lets a row be
// dragged around at all. A reorder that left the accumulators where they were would quietly hand one
// ring's growth to another.
func TestAMovedRingKeepsItsGrowth(t *testing.T) {
	a, b := growing(t, "grown-a", 100, 5), growing(t, "grown-b", 100, 5)

	d := duelist(10, 5, 100).
		Wearing(WornRing{Ring: a, Grown: 40}).
		Wearing(WornRing{Ring: b, Grown: 10})

	moved := d.MoveRing(0, 1)

	for _, w := range moved.WornRings() {
		want := 10
		if w.Ring == a {
			want = 40
		}
		if w.Grown != want {
			t.Errorf("%s has grown %d after the move, want %d", RingOf(w.Ring).Key, w.Grown, want)
		}
	}
}

// A drop resolved against a row that has changed underneath it must not take the frame with it.
func TestMovingARingOutOfRangeIsANoOp(t *testing.T) {
	a, b := growing(t, "range-a", 100, 5), growing(t, "range-b", 100, 5)
	d := duelist(10, 5, 100).Wearing(WornRing{Ring: a}).Wearing(WornRing{Ring: b})

	for _, move := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {1, 1}} {
		got := d.MoveRing(move[0], move[1])
		if got.WornRings()[0].Ring != a || got.WornRings()[1].Ring != b {
			t.Errorf("MoveRing(%d, %d) moved something", move[0], move[1])
		}
	}
}

// **The badge says what the ring is doing, not how far it has counted.** A ring scaling damage by
// 100 with 50 banked is doing 1.5x, and the figure the screen prints has to be the 150.
func TestGrowthEffectReportsTheEffectTheAccumulatorFeeds(t *testing.T) {
	id := growing(t, "figure", 100, 5)

	e, ok := GrowthEffect(WornRing{Ring: id, Grown: 50})
	if !ok {
		t.Fatal("a growing ring reported no growth effect")
	}
	if e.Do != DoScaleDamage {
		t.Errorf("growth feeds %v, want scale-damage", e.Do)
	}
	if e.Amount != 150 {
		t.Errorf("growth figure is %d, want 150", e.Amount)
	}
	if !Scaling(e.Do) {
		t.Error("scale-damage is not reported as a percentage, so a badge would print it flat")
	}
}

// A ring with no accumulator has no badge, which is most of the catalogue.
func TestARingThatDoesNotGrowReportsNoGrowth(t *testing.T) {
	id := ring(t, "static",
		RingRule{When: MomentCardDamage, Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}}})

	if Grows(id) {
		t.Error("a ring with no growth verb reports that it grows")
	}
	if _, ok := GrowthEffect(WornRing{Ring: id}); ok {
		t.Error("a ring with no growth verb reported a growth figure")
	}
}

// **A flat verb must not be reported as a percentage**, or Heart's badge would read "0.5x" where the
// ring is adding fifty points of life.
func TestAFlatGrowingRingIsNotScaling(t *testing.T) {
	id := ring(t, "flat-growth",
		RingRule{When: MomentFightStart, Then: []RingEffect{{Do: DoAddHP, Amount: 5}}},
		RingRule{When: MomentFightWon, Then: []RingEffect{{Do: DoGrowOnWin, Amount: 5}}})

	e, ok := GrowthEffect(WornRing{Ring: id, Grown: 45})
	if !ok {
		t.Fatal("a growing ring reported no growth effect")
	}
	if Scaling(e.Do) {
		t.Errorf("%v is reported as a percentage", e.Do)
	}
	if e.Amount != 50 {
		t.Errorf("growth figure is %d, want 50", e.Amount)
	}
}

// **The order of the cards in a turn changes what they are worth** *(owner's call, 2026-08-26)*.
// This is the rule that replaced "the hand's order is not a rule": a growing ring steps between the
// landings of one blow, so the card that goes first pays for the card behind it.
func TestAGrowingRingStepsBetweenTheCardsOfOneBlow(t *testing.T) {
	enflamed := ring(t, "order-enflamed",
		RingRule{
			When: MomentCardDamage,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	attacker := duelist(100, 8, 100).Wearing(WornRing{Ring: enflamed})
	fire := Of(Strike, Fire)

	log, _, _ := resolve(attacker, duelist(10, 5, 100000), []Card{fire, fire}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	if hand.HandCardCount != 2 {
		t.Fatalf("the blow has %d terms, want 2", hand.HandCardCount)
	}

	first, second := hand.HandAmounts[0], hand.HandAmounts[1]
	if second <= first {
		t.Errorf("the second fire card paid %d against the first's %d — the ring did not step "+
			"between them", second, first)
	}
	if want := first * 110 / 100; second != want {
		t.Errorf("the second fire card paid %d, want %d — one step of the ring on top of the first",
			second, want)
	}
}

// **A card that does not match the ring does not step it**, so an ice card between two fire ones
// leaves the second fire card exactly one step up rather than two.
func TestOnlyAMatchingCardStepsTheRingMidBlow(t *testing.T) {
	enflamed := ring(t, "order-mixed",
		RingRule{
			When: MomentCardDamage,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	attacker := duelist(100, 8, 100).Wearing(WornRing{Ring: enflamed})
	fire, ice := Of(Strike, Fire), Of(Strike, Ice)

	log, _, _ := resolve(attacker, duelist(10, 5, 100000), []Card{fire, ice, fire}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	if want := hand.HandAmounts[0] * 110 / 100; hand.HandAmounts[2] != want {
		t.Errorf("the second fire card paid %d, want %d — the ice card stepped a fire ring",
			hand.HandAmounts[2], want)
	}
}

// **The multiplier travels with the term, not with the card.** It is what the sum draws beside each
// figure now that the card face carries no ring at all.
func TestTheHandEventCarriesEachTermsRingMultipliers(t *testing.T) {
	enflamed := ring(t, "growth-scale",
		RingRule{
			When: MomentCardDamage,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	attacker := duelist(100, 8, 100).Wearing(WornRing{Ring: enflamed})
	fire := Of(Strike, Fire)

	log, _, _ := resolve(attacker, duelist(10, 5, 100000), []Card{fire, fire, fire}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}

	for i, want := range []int{100, 110, 120} {
		if got := hand.HandRingScale[i][0]; got != want {
			t.Errorf("term %d was counted at %d%% by the ring on seat 0, want %d%%", i, got, want)
		}
		if got := hand.HandGrown[i][0]; got != (i+1)*10 {
			t.Errorf("after term %d the ring reads %d, want %d", i, got, (i+1)*10)
		}
	}
}

// A duelist wearing no growing ring says nothing about growth, which is every blow of a run that
// has not bought one.
func TestABlowWithNoRingsReportsNoneFiring(t *testing.T) {
	log, _, _ := resolve(duelist(100, 8, 100), duelist(10, 5, 100000),
		[]Card{Of(Strike, Fire), Of(Strike, Fire)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	for i := 0; i < hand.HandCardCount; i++ {
		for seat, pct := range hand.HandRingScale[i] {
			if pct != 0 {
				t.Errorf("term %d reports seat %d firing at %d%%, and no ring is worn", i, seat, pct)
			}
		}
	}
}

// firstOfKind is the first event of one kind in a log.
func firstOfKind(log []Event, k EventKind) (Event, bool) {
	for _, e := range log {
		if e.Kind == k {
			return e, true
		}
	}
	return Event{}, false
}

// **An echo compounds inside itself** *(owner's call, 2026-08-26)*. A lead card an echo ring seats
// three times steps a growing ring three times, and each of those landings is priced at the figure
// the one before it left — so the ladder is not three fractions of one number, it is three fractions
// of three growing numbers.
//
// It is the deliberate reading rather than the only one: an echo could have been a single step for
// the card that earned it. This test is here because nothing else would notice if the loop went back
// to pricing all three landings off the same damage figure — the step count would still be three and
// TestEchoAndAGrowOnHitRingCompound would still pass.
func TestAnEchoedCardCompoundsAgainstItsOwnGrowth(t *testing.T) {
	enflamed := ring(t, "echo-compound-enflamed",
		RingRule{
			When: MomentCardDamage,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})
	echo := ring(t, "echo-compound-echo", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Lead: true},
		Then: []RingEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	attacker := duelist(100, 8, 100).
		Wearing(WornRing{Ring: enflamed}).
		Wearing(WornRing{Ring: echo})

	log, _, _ := resolve(attacker, duelist(10, 5, 100000),
		[]Card{Of(Strike, Fire)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	if hand.HandCardCount != 3 {
		t.Fatalf("the echoed card paid %d terms, want 3", hand.HandCardCount)
	}

	// The first landing is the card bare; the two behind it are the echo's fractions of a card that
	// has grown a step each time.
	base := hand.HandAmounts[0]
	for i, want := range []int{
		EchoBonus(base*110/100, 2, 3),
		EchoBonus(base*120/100, 3, 3),
	} {
		term := i + 1
		if got := hand.HandAmounts[term]; got != want {
			flat := EchoBonus(base, term+1, 3)
			t.Errorf("echo landing %d paid %d, want %d — %d is the figure it would pay if every "+
				"landing were priced off the same damage", term, got, want, flat)
		}
	}

	// And the multiplier said beside each term moves with it, which is what the player is shown.
	for i, want := range []int{100, 110, 120} {
		if got := hand.HandRingScale[i][0]; got != want {
			t.Errorf("echo landing %d was counted at %d%%, want %d%%", i, got, want)
		}
	}
}

// **A ring that never grows is still accounted for in the sum** *(owner's call, 2026-08-26)*. The
// first pass reported growth alone, and a fire ring doubling every fire card stayed invisible: its
// work was inside the term's figure with nothing on screen saying so. With nothing on the card face
// any more, the sum is the only place it can be seen.
func TestEveryDamageRingIsAccountedForPerTerm(t *testing.T) {
	fire := ring(t, "seat-fire", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	enflamed := ring(t, "seat-enflamed",
		RingRule{
			When: MomentCardDamage,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	attacker := duelist(100, 8, 100).
		Wearing(WornRing{Ring: fire}).
		Wearing(WornRing{Ring: enflamed})

	log, _, _ := resolve(attacker, duelist(10, 5, 100000),
		[]Card{Of(Strike, Fire), Of(Strike, Fire)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}

	// Seat 0 is the flat doubler and says the same thing both terms; seat 1 grows between them.
	for i, want := range [][2]int{{200, 100}, {200, 110}} {
		if got := hand.HandRingScale[i][0]; got != want[0] {
			t.Errorf("term %d: the flat ring fired at %d%%, want %d%%", i, got, want[0])
		}
		if got := hand.HandRingScale[i][1]; got != want[1] {
			t.Errorf("term %d: the growing ring fired at %d%%, want %d%%", i, got, want[1])
		}
	}
}

// A ring that does not match the card has no beat at all, which is what the zero means.
func TestARingThatDoesNotMatchDoesNotFire(t *testing.T) {
	fire := ring(t, "seat-nomatch", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	attacker := duelist(100, 8, 100).Wearing(WornRing{Ring: fire})

	log, _, _ := resolve(attacker, duelist(10, 5, 100000), []Card{Of(Strike, Ice)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	if got := hand.HandRingScale[0][0]; got != 0 {
		t.Errorf("a fire ring fired at %d%% on an ice card, want not at all", got)
	}
}

// **An echo's extra terms are attributed to the ring that bought them** *(owner's call,
// 2026-08-26)*. Every figure in a blow's sum is accompanied by the card that produced it shaking,
// and an echo ring is the one contributor with no figure of its own: it buys a *term*, not a
// multiplier. Without this it would sit still through the three terms it alone is responsible for.
func TestAnEchosExtraTermsNameTheRingThatBoughtThem(t *testing.T) {
	echo := ring(t, "landing-echo", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Lead: true},
		Then: []RingEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	attacker := duelist(100, 8, 100).Wearing(WornRing{Ring: echo})

	log, _, _ := resolve(attacker, duelist(10, 5, 100000), []Card{Of(Strike, Fire)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	if hand.HandCardCount != 3 {
		t.Fatalf("the echoed card paid %d terms, want 3", hand.HandCardCount)
	}

	// The card's own landing is nobody's doing; the two behind it are the ring's.
	if hand.HandLanding[0][0] {
		t.Error("the card's own first landing is attributed to a ring")
	}
	for _, term := range []int{1, 2} {
		if !hand.HandLanding[term][0] {
			t.Errorf("echo term %d names no ring, so nothing would shake for it", term)
		}
	}
}

// A ring that seats no extra landing is not the reason for any term.
func TestACardThatLandsOnceNamesNoRing(t *testing.T) {
	log, _, _ := resolve(duelist(100, 8, 100), duelist(10, 5, 100000),
		[]Card{Of(Strike, Fire)}, nil, 1)

	hand, ok := firstOfKind(log, KindHand)
	if !ok {
		t.Fatal("the blow produced no hand event")
	}
	for seat, named := range hand.HandLanding[0] {
		if named {
			t.Errorf("a single landing is attributed to seat %d", seat)
		}
	}
}
