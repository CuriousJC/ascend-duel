package combat

import "testing"

// The ring grammar: what registration refuses, and what wearing one actually does.
//
// **The rings here are built rather than loaded**, because this package cannot read `rings.json` —
// see the file comment in ring.go. That is the point of the split and it is what these tests
// exercise: a rules-level ring is a key, a name and a list of rules, and everything about how it was
// spelled in a file is somebody else's problem.

// ring registers one ring for a test and fails the test rather than the process if it will not take.
// Keys are prefixed so nothing here can collide with the four `internal/session` registers.
func ring(t *testing.T, key string, rules ...RingRule) RingID {
	t.Helper()

	id, err := RegisterRing("ringtest."+key, key, rules)
	if err != nil {
		t.Fatalf("%s did not register: %v", key, err)
	}
	return id
}

func refused(t *testing.T, key string, rules ...RingRule) {
	t.Helper()

	if _, err := RegisterRing("ringtest.refused."+key, key, rules); err == nil {
		t.Errorf("%s registered, and it should not have", key)
	}
}

// --- what the grammar refuses ------------------------------------------------------------------

func TestAVerbAtTheWrongMomentIsRefused(t *testing.T) {
	// **The failure this prevents is the quiet one**: a rule that loads, never fires, and looks
	// exactly like a ring that does nothing. Every verb belongs to one moment and the table in
	// ring.go is the authority.
	refused(t, "cost at fight-start", RingRule{
		When: MomentFightStart,
		Then: []RingEffect{{Do: DoAdjustCost, Amount: -1}},
	})
	refused(t, "status at card-cost", RingRule{
		When: MomentCardCost,
		Then: []RingEffect{{Do: DoApplyStatus, Status: 0}},
	})
}

func TestAPredicateOnACardlessMomentIsRefused(t *testing.T) {
	// `fight-start`, `fight-won` and `prizes-dealt` have no card to match an If against, so a rule
	// carrying one is either a misunderstanding or a rule that would silently match everything.
	refused(t, "fight-start with an If", RingRule{
		When: MomentFightStart,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoAddDMG, Amount: 10}},
	})
}

func TestAnEffectWithNothingToDoIsRefused(t *testing.T) {
	// A zero is a typo in a file authored once, not a reward to be clamped — see checkEffect for
	// why this is the opposite call from the one a worm's amount takes.
	refused(t, "zero damage scale", RingRule{
		When: MomentCardDamage,
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 0}},
	})
	refused(t, "zero cost delta", RingRule{
		When: MomentCardCost,
		Then: []RingEffect{{Do: DoAdjustCost, Amount: 0}},
	})
	refused(t, "a flip to basic", RingRule{
		When: MomentDeckBuilt,
		Then: []RingEffect{{Do: DoSetElement, Element: Basic}},
	})
	refused(t, "a ring with no rules at all")
}

func TestAStatusTheFilesDoNotHoldIsRefused(t *testing.T) {
	refused(t, "unknown status", RingRule{
		When: MomentAttackLands,
		Then: []RingEffect{{Do: DoApplyStatus, Status: StatusID(StatusCount() + 1)}},
	})
}

// --- what wearing one does --------------------------------------------------------------------

func TestADiscountIsAPropertyOfThePairing(t *testing.T) {
	// The whole reason cost moved off the card: the same Strike costs one duelist less than
	// another, so nothing may ask a card what it costs without saying who is holding it.
	thrifty := ring(t, "thrifty", RingRule{
		When: MomentCardCost,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoAdjustCost, Amount: -1}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRing{Ring: thrifty})

	hot, cold := Of(Strike, Fire), Of(Strike, Ice)

	if got, want := worn.CardCost(hot), bare.CardCost(hot)-1; got != want {
		t.Errorf("a discounted fire Strike costs %d, want %d", got, want)
	}
	if got, want := worn.CardCost(cold), bare.CardCost(cold); got != want {
		t.Errorf("the discount reached an ice Strike: %d, want %d", got, want)
	}
}

func TestNoDiscountTakesACardBelowFree(t *testing.T) {
	// Free is the floor and the count cap is what bounds a turn of free cards — see minCardCost.
	// What must not happen is a negative cost paying for the card beside it.
	free := ring(t, "free", RingRule{
		When: MomentCardCost,
		Then: []RingEffect{{Do: DoAdjustCost, Amount: -9}},
	})

	d := duelist(10, 5, 100).Wearing(WornRing{Ring: free})
	if got := d.CardCost(Plain(Smash)); got != 0 {
		t.Errorf("a deeply discounted Smash costs %d, want 0", got)
	}
}

func TestAFamilyRingDoublesEveryMatchingCardInTheTurn(t *testing.T) {
	// **Per card is the point** — three slash cards in a turn are three doublings inside the same
	// blow, not one. This checks the per-card figure, which is what the blow is summed from.
	keen := ring(t, "keen", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Family: FamilySlash, HasFamily: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRing{Ring: keen})

	if got, want := worn.CardDamage(Plain(Slash)), bare.CardDamage(Plain(Slash))*2; got != want {
		t.Errorf("a Slash under the keen ring deals %d, want %d", got, want)
	}
	if got, want := worn.CardDamage(Plain(Strike)), bare.CardDamage(Plain(Strike)); got != want {
		t.Errorf("the slash ring reached a crush card: %d, want %d", got, want)
	}
}

func TestTwoMatchingRingsCompound(t *testing.T) {
	// **Compounding is intended**: two slash rings are x4 and that is a build. It is also why worn
	// order is a rule — multiplicative effects are order-sensitive.
	a := ring(t, "keen a", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Family: FamilySlash, HasFamily: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	b := ring(t, "keen b", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Family: FamilySlash, HasFamily: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	both := bare.Wearing(WornRing{Ring: a}).Wearing(WornRing{Ring: b})

	if got, want := both.CardDamage(Plain(Slash)), bare.CardDamage(Plain(Slash))*4; got != want {
		t.Errorf("two slash rings deal %d, want %d", got, want)
	}
}

func TestAConceptRingReachesOneCardOnly(t *testing.T) {
	// A concept ring is a much narrower object than a family ring — 4 cards against 12 — which is
	// the distinction this holds and the reason the two must not be priced alike.
	striker := ring(t, "striker", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Concept: Strike, HasConcept: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRing{Ring: striker})

	if got, want := worn.CardDamage(Plain(Strike)), bare.CardDamage(Plain(Strike))*2; got != want {
		t.Errorf("a Strike under the striker ring deals %d, want %d", got, want)
	}
	for _, id := range []ConceptID{Bash, Smash, Slash} {
		if got, want := worn.CardDamage(Plain(id)), bare.CardDamage(Plain(id)); got != want {
			t.Errorf("%v under a Strike ring deals %d, want %d", ConceptOf(id).Label, got, want)
		}
	}
}

func TestTwoPredicatesNarrowARuleRatherThanWidenIt(t *testing.T) {
	both := ring(t, "fire slash", RingRule{
		When: MomentCardDamage,
		If: RingCondition{
			Element: Fire, HasElement: true,
			Family: FamilySlash, HasFamily: true,
		},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRing{Ring: both})

	if got, want := worn.CardDamage(Of(Slash, Fire)), bare.CardDamage(Of(Slash, Fire))*2; got != want {
		t.Errorf("a fire slash deals %d, want %d", got, want)
	}
	for _, c := range []Card{Of(Slash, Ice), Of(Strike, Fire)} {
		if got, want := worn.CardDamage(c), bare.CardDamage(c); got != want {
			t.Errorf("%v matched a rule wanting both predicates: %d, want %d", c, got, want)
		}
	}
}

func TestAStatusNamesTheRingThatAppliedIt(t *testing.T) {
	// **The screen flies the word out of the ring that caused it**, so the event has to say which
	// ring that was. Nothing else can: the card's colour is not the answer, because a ring may
	// match on a family or a concept and apply a status with no colour involved at all - which is
	// the case the second half of this test pins.
	burning := MustStatus("burning")
	chilled := MustStatus("chilled")

	fire := ring(t, "names-fire", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoApplyStatus, Status: burning}},
	})
	// A ring that reads the family rather than the colour, which is what makes deriving the ring
	// from the element impossible rather than merely fragile.
	slash := ring(t, "names-slash", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Family: FamilySlash, HasFamily: true},
		Then: []RingEffect{{Do: DoApplyStatus, Status: chilled}},
	})

	a := duelist(10, 8, 500).Wearing(WornRing{Ring: fire}).Wearing(WornRing{Ring: slash})
	b := duelist(10, 5, 500)

	// One fire slash matches both rings at once, so both statuses land off one card.
	events, _, _ := resolve(a, b, []Card{Of(Slash, Fire)}, nil, 1)

	got := map[StatusID]RingID{}
	for _, e := range events {
		if e.Kind == KindStatus {
			got[e.Status] = e.Ring
		}
	}
	if len(got) != 2 {
		t.Fatalf("a fire slash under two rings announced %d statuses, want 2", len(got))
	}
	if got[burning] != fire {
		t.Errorf("the burn is credited to ring %d, want the fire ring %d", got[burning], fire)
	}
	if got[chilled] != slash {
		t.Errorf("the chill is credited to ring %d, want the slash ring %d", got[chilled], slash)
	}
}

func TestTheFirstRingToApplyAStatusIsTheOneCredited(t *testing.T) {
	// Two rings, one status, one blow. The dedup keeps it to a single event; worn order decides
	// whose it is, which is the tie-break every other compounding effect already takes.
	burning := MustStatus("burning")
	rule := RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoApplyStatus, Status: burning}},
	}
	first := ring(t, "credit-first", rule)
	second := ring(t, "credit-second", rule)

	a := duelist(10, 8, 500).Wearing(WornRing{Ring: first}).Wearing(WornRing{Ring: second})
	events, _, _ := resolve(a, duelist(10, 5, 500), []Card{Of(Jab, Fire)}, nil, 1)

	n := 0
	for _, e := range events {
		if e.Kind != KindStatus {
			continue
		}
		n++
		if e.Ring != first {
			t.Errorf("the burn is credited to ring %d, want the one worn first, %d", e.Ring, first)
		}
	}
	if n != 1 {
		t.Errorf("two rings applying one status announced it %d times, want 1", n)
	}
}

func TestOneBlowLandsOneOfEachStatus(t *testing.T) {
	// Two fire cards match a fire ring twice. The status does not stack, so applying it twice is
	// the same as applying it once — but announcing it twice would describe two things that did
	// not happen. See statusesFrom.
	burning := MustStatus("burning")
	fire := ring(t, "fire", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoApplyStatus, Status: burning}},
	})

	a := duelist(10, 8, 500).Wearing(WornRing{Ring: fire})
	b := duelist(10, 5, 500)

	events, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire), Of(Jab, Fire)}, nil, 1)

	if n := countKind(events, KindStatus); n != 1 {
		t.Errorf("two fire cards announced %d statuses, want 1", n)
	}
	if !bAfter.Statuses[burning].Active() {
		t.Error("two fire cards left no burn at all")
	}
}

func TestOneRuleCanApplyTwoStatuses(t *testing.T) {
	// **`Then` is a list**, which is what buys a ring that shocks *and* chills with no new
	// vocabulary at all — the Storm ring, whole, in one entry.
	shocked, chilled := MustStatus("shocked"), MustStatus("chilled")
	storm := ring(t, "storm", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Element: Lightning, HasElement: true},
		Then: []RingEffect{
			{Do: DoApplyStatus, Status: shocked},
			{Do: DoApplyStatus, Status: chilled},
		},
	})

	a := duelist(10, 5, 500).Wearing(WornRing{Ring: storm})
	b := duelist(10, 5, 500)

	_, _, bAfter := resolve(a, b, []Card{Of(Strike, Lightning)}, nil, 1)

	if !bAfter.Statuses[shocked].Active() || !bAfter.Statuses[chilled].Active() {
		t.Errorf("one storm hit left shocked=%v chilled=%v, want both",
			bAfter.Statuses[shocked].Active(), bAfter.Statuses[chilled].Active())
	}
}

func TestFlipsDoNotCompose(t *testing.T) {
	// Every flip reads the card's *original* element, so lightning->ice and fire->ice both land on
	// their own sources and cannot chain. Without it, two flips could cascade a deck to one colour
	// and the order they were bought in would change the result.
	toIce := ring(t, "lightning to ice", RingRule{
		When: MomentDeckBuilt,
		If:   RingCondition{Element: Lightning, HasElement: true},
		Then: []RingEffect{{Do: DoSetElement, Element: Ice}},
	})
	toEarth := ring(t, "ice to earth", RingRule{
		When: MomentDeckBuilt,
		If:   RingCondition{Element: Ice, HasElement: true},
		Then: []RingEffect{{Do: DoSetElement, Element: Earth}},
	})

	worn := []WornRing{{Ring: toIce}, {Ring: toEarth}}

	if e, ok := FlipElement(worn, Of(Strike, Lightning)); !ok || e != Ice {
		t.Errorf("a lightning card became %v (flipped %v), want ice — the second flip chained", e, ok)
	}
	if e, ok := FlipElement(worn, Of(Strike, Ice)); !ok || e != Earth {
		t.Errorf("an ice card became %v (flipped %v), want earth", e, ok)
	}
	if _, ok := FlipElement(worn, Of(Strike, Fire)); ok {
		t.Error("a fire card was flipped by rings that do not name it")
	}
}

func TestTheAccumulatorRidesOnTheWornRing(t *testing.T) {
	// A growing ring's own amounts are read as `Amount + accumulator`, and the accumulator travels
	// with the worn ring because it belongs to a run rather than to the registry.
	heart := ring(t, "heart",
		RingRule{When: MomentFightStart, Then: []RingEffect{{Do: DoAddHP, Amount: 5}}},
		RingRule{When: MomentFightWon, Then: []RingEffect{{Do: DoGrow, Amount: 5}}})

	fresh := []WornRing{{Ring: heart}}
	grown := []WornRing{{Ring: heart, Grown: 20}}

	if got := AddedHP(fresh); got != 5 {
		t.Errorf("a fresh heart ring adds %d HP, want 5", got)
	}
	if got := AddedHP(grown); got != 25 {
		t.Errorf("a heart ring at +20 adds %d HP, want 25", got)
	}
	if got := Growth(fresh[0]); got != 5 {
		t.Errorf("a heart ring grows by %d, want 5", got)
	}
	if got := Growth(grown[0]); got != 5 {
		t.Errorf("a grown heart ring grows by %d, want 5 — growth must not compound on itself", got)
	}
}

func TestPropagationScalesAndCompounds(t *testing.T) {
	// The ring scales what the run's own cap produced, and two of them compound like every other
	// ring effect. The cap itself is the run's business — see session.propagate.
	banker := ring(t, "banker", RingRule{
		When: MomentFightWon,
		Then: []RingEffect{{Do: DoScalePropagation, Amount: 200}},
	})

	if got := ScalePropagation(nil, 5); got != 5 {
		t.Errorf("bare propagation of 5 came out as %d", got)
	}
	if got := ScalePropagation([]WornRing{{Ring: banker}}, 5); got != 10 {
		t.Errorf("one banker turned 5 into %d, want 10", got)
	}
	if got := ScalePropagation([]WornRing{{Ring: banker}, {Ring: banker}}, 5); got != 20 {
		t.Errorf("two bankers turned 5 into %d, want 20", got)
	}
}

func TestARingIsOnlyWornOnceTheHandIsNotFull(t *testing.T) {
	// Five worn at once, until brands expand it. The array is the cap and Wearing is where it
	// bites; a sixth is dropped rather than overwriting the fifth.
	worn := ring(t, "filler", RingRule{
		When: MomentFightStart,
		Then: []RingEffect{{Do: DoAddDMG, Amount: 1}},
	})

	d := duelist(10, 5, 100)
	for i := 0; i < MaxWornRings+3; i++ {
		d = d.Wearing(WornRing{Ring: worn})
	}

	if d.RingCount != MaxWornRings {
		t.Errorf("a duelist ended up wearing %d rings, cap is %d", d.RingCount, MaxWornRings)
	}
	if got := AddedDMG(d.WornRings()); got != MaxWornRings {
		t.Errorf("%d rings added %d DMG, want %d", d.RingCount, got, MaxWornRings)
	}
}

func TestAnEnemyWearsNothing(t *testing.T) {
	// **Rings are the duelist's only.** The zero value is an empty hand, which is what an enemy is
	// hydrated with — so an enemy's colours are inert by construction rather than by a rule written
	// down somewhere else.
	var enemy Duelist
	if n := len(enemy.WornRings()); n != 0 {
		t.Errorf("a zero duelist wears %d rings", n)
	}
	if got := enemy.statusesFrom([]Card{Of(Strike, Fire)}); len(got) != 0 {
		t.Errorf("a ringless duelist's fire Strike applied %d statuses", len(got))
	}
}
