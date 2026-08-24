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

// cardOfForm is a registered player card of one form, so a form-matching test does not have to
// write down which concept happens to be a crush this week.
func cardOfForm(t *testing.T, f Form) Card {
	t.Helper()

	for id := ConceptID(0); int(id) < ConceptCount(); id++ {
		c := Of(id, Basic)
		if c.Spec().Verb == VerbAttack && c.Form() == f {
			return c
		}
	}
	t.Fatalf("no attack card has form %v", f)
	return Card{}
}

// cardOfTier is the registered player attack of one form on one rung of its ladder.
func cardOfTier(t *testing.T, f Form, tier int) Card {
	t.Helper()

	for id := ConceptID(0); int(id) < ConceptCount(); id++ {
		c := ConceptOf(id)
		if c.Verb == VerbAttack && c.Form == f && c.Tier() == tier {
			return Of(id, Basic)
		}
	}
	t.Fatalf("no %v attack sits on rung %d", f, tier)
	return Card{}
}

func crushCard(t *testing.T) Card { t.Helper(); return cardOfForm(t, FormCrush) }
func slashCard(t *testing.T) Card { t.Helper(); return cardOfForm(t, FormSlash) }

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

func TestAFormRingDoublesEveryMatchingCardInTheTurn(t *testing.T) {
	// **Per card is the point** — three slash cards in a turn are three doublings inside the same
	// blow, not one. This checks the per-card figure, which is what the blow is summed from.
	keen := ring(t, "keen", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Form: FormSlash, HasForm: true},
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
		If:   RingCondition{Form: FormSlash, HasForm: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	b := ring(t, "keen b", RingRule{
		When: MomentCardDamage,
		If:   RingCondition{Form: FormSlash, HasForm: true},
		Then: []RingEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	both := bare.Wearing(WornRing{Ring: a}).Wearing(WornRing{Ring: b})

	if got, want := both.CardDamage(Plain(Slash)), bare.CardDamage(Plain(Slash))*4; got != want {
		t.Errorf("two slash rings deal %d, want %d", got, want)
	}
}

func TestAConceptRingReachesOneCardOnly(t *testing.T) {
	// A concept ring is a much narrower object than a form ring — 4 cards against 12 — which is
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
			Form: FormSlash, HasForm: true,
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
	// match on a form or a concept and apply a status with no colour involved at all - which is
	// the case the second half of this test pins.
	burning := MustStatus("burning")
	chilled := MustStatus("chilled")

	fire := ring(t, "names-fire", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Element: Fire, HasElement: true},
		Then: []RingEffect{{Do: DoApplyStatus, Status: burning}},
	})
	// A ring that reads the form rather than the colour, which is what makes deriving the ring
	// from the element impossible rather than merely fragile.
	slash := ring(t, "names-slash", RingRule{
		When: MomentAttackLands,
		If:   RingCondition{Form: FormSlash, HasForm: true},
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
		When: MomentCardDrawn,
		If:   RingCondition{Element: Lightning, HasElement: true},
		Then: []RingEffect{{Do: DoSetElement, Element: Ice}},
	})
	toEarth := ring(t, "ice to earth", RingRule{
		When: MomentCardDrawn,
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
		RingRule{When: MomentFightWon, Then: []RingEffect{{Do: DoGrowOnWin, Amount: 5}}})

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

func TestHPScalingCompoundsAndDefaultsToWhole(t *testing.T) {
	// Onslaught's half of the grammar: a fight-start scaling that goes *below* 100, which no other
	// scaling verb does. A bare duelist has to come out untouched, or every ring in the file would
	// be quietly resizing a life bar.
	if got := HPScale(nil); got != 100 {
		t.Errorf("nothing worn scales life to %d%%, want 100%%", got)
	}

	quarterOff := ring(t, "onslaught", RingRule{
		When: MomentFightStart,
		Then: []RingEffect{{Do: DoScaleHP, Amount: 75}},
	})

	if got := HPScale([]WornRing{{Ring: quarterOff}}); got != 75 {
		t.Errorf("one drawback ring scales life to %d%%, want 75%%", got)
	}

	// Compounding rather than adding, like every other multiplicative effect: two quarters off
	// leave 56%, not half.
	two := []WornRing{{Ring: quarterOff}, {Ring: quarterOff}}
	if got := HPScale(two); got != 56 {
		t.Errorf("two drawback rings scale life to %d%%, want 56%% — they are adding, not "+
			"compounding", got)
	}
}

func TestAnEchoSeatsTheLeadCardAgainAtDecreasingAmounts(t *testing.T) {
	// Echo's whole shape in one place: the lead card of the blow pays three terms rather than one,
	// the sum grows by exactly those terms, and the hand the cards formed is untouched.
	echo := ring(t, "echo", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Lead: true},
		Then: []RingEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	card := Of(Strike, Fire)

	if got := LandingAmounts(nil, card, true, 30); len(got) != 1 || got[0] != 30 {
		t.Errorf("a bare duelist pays %v for a 30 card, want [30]", got)
	}

	worn := []WornRing{{Ring: echo}}
	if got := LandingAmounts(worn, card, true, 30); len(got) != 3 ||
		got[0] != 30 || got[1] != 20 || got[2] != 10 {
		t.Errorf("Echo pays %v for a 30 lead card, want [30 20 10]", got)
	}

	// **Only the lead card**, which is what the Lead predicate is for.
	if got := LandingAmounts(worn, card, false, 30); len(got) != 1 {
		t.Errorf("Echo pays %v for a card that does not lead the blow, want one term", got)
	}

	// Two of them add a landing each rather than multiplying: five landings, not nine.
	if got := LandingAmounts([]WornRing{{Ring: echo}, {Ring: echo}}, card, true, 30); len(got) != 5 {
		t.Errorf("two echo rings pay %v, want five terms", got)
	}

	// The ladder itself: full, two thirds, one third, and nothing outside the range.
	for _, tc := range []struct{ k, want int }{{1, 0}, {2, 20}, {3, 10}, {4, 0}} {
		if got := EchoBonus(30, tc.k, 3); got != tc.want {
			t.Errorf("landing %d of 3 on a 30 card is %d, want %d", tc.k, got, tc.want)
		}
	}

	// Never nothing: a card small enough to round to zero still lands for 1.
	if got := EchoBonus(1, 3, 3); got != 1 {
		t.Errorf("the third landing of a 1-damage card is %d, want 1", got)
	}
}

func TestARepeatLandsEveryMatchingCardAtFullDamage(t *testing.T) {
	// The form repeat rings: every card the rule matches lands twice, both at full strength — where
	// an echo diminishes and only takes the lead card.
	repeat := ring(t, "aftershock", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Form: FormCrush, HasForm: true},
		Then: []RingEffect{{Do: DoRepeatCard, Amount: 2}},
	})
	worn := []WornRing{{Ring: repeat}}

	crush, slash := crushCard(t), slashCard(t)

	// **Not only the lead card**, which is the whole difference from Echo: a matching card in the
	// third seat repeats too.
	for _, lead := range []bool{true, false} {
		got := LandingAmounts(worn, crush, lead, 40)
		if len(got) != 2 || got[0] != 40 || got[1] != 40 {
			t.Errorf("a crush card (lead %v) pays %v, want [40 40]", lead, got)
		}
	}

	if got := LandingAmounts(worn, slash, true, 40); len(got) != 1 {
		t.Errorf("a slash card pays %v under a crush repeat, want one term", got)
	}
}

func TestRepeatsComeBeforeEchoesAndBothAreCapped(t *testing.T) {
	repeat := ring(t, "aftershock-2", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Form: FormCrush, HasForm: true},
		Then: []RingEffect{{Do: DoRepeatCard, Amount: 2}},
	})
	echo := ring(t, "echo-2", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Lead: true},
		Then: []RingEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	// Repeat first, then the echo ladder over the echo's own count: 30, 30 (the copy), then
	// two thirds and one third.
	got := LandingAmounts([]WornRing{{Ring: repeat}, {Ring: echo}}, crushCard(t), true, 30)
	want := []int{30, 30, 20, 10}
	if len(got) != len(want) {
		t.Fatalf("a repeated and echoed crush card pays %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("a repeated and echoed crush card pays %v, want %v", got, want)
		}
	}

	// Nothing may seat more landings than the event's arrays are wide.
	many := []WornRing{{Ring: repeat}, {Ring: repeat}, {Ring: repeat}, {Ring: echo}, {Ring: echo}}
	if got := LandingAmounts(many, crushCard(t), true, 30); len(got) > MaxEchoLandings {
		t.Errorf("five stacked rings pay %d terms, want at most %d", len(got), MaxEchoLandings)
	}
}

func TestAtrophyStepsThreeAPAttacksDownOneRung(t *testing.T) {
	// Atrophy's whole shape: the top rung of each form becomes the middle rung, nothing else moves,
	// and the ladder is read off the declared cost rather than off what the wearer pays.
	atrophy := ring(t, "atrophy", RingRule{
		When: MomentDeckBuilt,
		If:   RingCondition{Tier: 3, HasTier: true},
		Then: []RingEffect{{Do: DoDemoteCard, Amount: 1}},
	})
	worn := []WornRing{{Ring: atrophy}}

	for _, f := range []Form{FormStab, FormSlash, FormCrush} {
		top, mid := cardOfTier(t, f, 3), cardOfTier(t, f, 2)

		got, demoted := DemoteConcept(worn, top)
		if !demoted {
			t.Errorf("%s was not stepped down", top.Label())
			continue
		}
		if got != mid.Concept {
			t.Errorf("%s became %s, want %s", top.Label(), Of(got, Basic).Label(), mid.Label())
		}

		// The rungs below the top are left where they are.
		if _, moved := DemoteConcept(worn, mid); moved {
			t.Errorf("%s was stepped down, and only the 3 AP rung should move", mid.Label())
		}
		if _, moved := DemoteConcept(worn, cardOfTier(t, f, 1)); moved {
			t.Errorf("the bottom rung of %v was stepped down", f)
		}
	}
}

func TestAGrowOnHitRingGetsStrongerInsideOneFight(t *testing.T) {
	// The Enflamed family: the accumulator moves as a blow lands, so the second fire attack of a
	// fight is already worth more than the first — where every other growing ring waits for the
	// win. What is checked here is the arithmetic; the seat in resolveAttackPhase is what makes a
	// real blow reach it.
	enflamed := ring(t, "enflamed",
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

	d := duelist(100, 5, 100).Wearing(WornRing{Ring: enflamed})

	fire, ice := Of(Strike, Fire), Of(Strike, Ice)

	// Fresh, the ring is worth nothing: 100% of the card is the card.
	if got, want := d.CardDamage(fire), d.CardDamage(ice); got != want {
		t.Errorf("an ungrown Enflamed deals %d where a plain card deals %d", got, want)
	}

	// A blow with a fire card in it steps the accumulator; an ice-only blow does not.
	d = d.GrowOnHit([]Card{ice})
	if got := d.WornRings()[0].Grown; got != 0 {
		t.Errorf("an ice blow grew a fire ring to %d, want 0", got)
	}

	d = d.GrowOnHit([]Card{fire, ice})
	if got := d.WornRings()[0].Grown; got != 10 {
		t.Errorf("one fire blow grew the ring to %d, want 10", got)
	}

	// **Once per hit, not once per blow** *(owner's call, 2026-08-22)*: two fire cards in one hand
	// are two hits, where a status would land once.
	d = d.GrowOnHit([]Card{fire, fire})
	if got := d.WornRings()[0].Grown; got != 30 {
		t.Errorf("a two-fire-card blow grew the ring to %d, want 30 — it paid per blow", got)
	}

	// Three hits in, fire cards are worth 1.3x and nothing else has moved.
	if got, want := d.CardDamage(fire), d.CardDamage(ice)*130/100; got != want {
		t.Errorf("after three fire hits a fire card deals %d, want %d", got, want)
	}

	// **The growth does not itself grow.** Reading the step as Amount+Grown would compound it.
	before := d.WornRings()[0].Grown
	d = d.GrowOnHit([]Card{fire})
	if got := d.WornRings()[0].Grown - before; got != 10 {
		t.Errorf("the third fire blow stepped the ring by %d, want 10 — the step is compounding", got)
	}
}

func TestEchoAndAGrowOnHitRingCompound(t *testing.T) {
	// **The combination is the point.** A card an echo ring seats three times hit three times, so an
	// Enflamed Ring worn beside Echo grows three steps off one card rather than one. If this ever
	// starts counting cards again, the two rings quietly stop being a build.
	enflamed := ring(t, "enflamed-echo",
		RingRule{
			When: MomentAttackLands,
			If:   RingCondition{Element: Fire, HasElement: true},
			Then: []RingEffect{{Do: DoGrowOnHit, Amount: 10}},
		})
	echo := ring(t, "echo-enflamed", RingRule{
		When: MomentBlowFormed,
		If:   RingCondition{Lead: true},
		Then: []RingEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	fire := Of(Strike, Fire)

	alone := duelist(100, 5, 100).Wearing(WornRing{Ring: enflamed})
	if got := alone.GrowOnHit([]Card{fire}).WornRings()[0].Grown; got != 10 {
		t.Errorf("one fire card without Echo grew the ring by %d, want 10", got)
	}

	both := duelist(100, 5, 100).
		Wearing(WornRing{Ring: enflamed}).
		Wearing(WornRing{Ring: echo})
	if got := both.GrowOnHit([]Card{fire}).WornRings()[0].Grown; got != 30 {
		t.Errorf("one echoed fire card grew the ring by %d, want 30 — the echo's landings are "+
			"not being counted", got)
	}
}

func TestAFourOfAKindGrowsOnceForEachCard(t *testing.T) {
	// **Through the real round, not through the applier**, because the bug this guards against is a
	// seat rather than a formula: a four of a kind is one *blow*, and an accumulator that took the
	// blow as its unit would step once where the player threw four attacks.
	enflamed := ring(t, "enflamed-round",
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

	attacker := duelist(10, 8, 100).Wearing(WornRing{Ring: enflamed})
	fire := Of(Strike, Fire)

	_, after, _ := resolve(attacker, duelist(10, 5, 1000),
		[]Card{fire, fire, fire, fire}, nil, 1)

	if got := after.WornRings()[0].Grown; got != 40 {
		t.Errorf("a fire Four of a Kind grew the ring by %d, want 40 — one step per card that "+
			"landed, not one per blow", got)
	}

	// A hand of one colour among others still only pays for its own colour.
	_, mixed, _ := resolve(attacker, duelist(10, 5, 1000),
		[]Card{fire, Of(Strike, Ice), fire}, nil, 1)
	if got := mixed.WornRings()[0].Grown; got != 20 {
		t.Errorf("two fire cards beside an ice one grew the ring by %d, want 20", got)
	}
}

func TestMomentumBuildsAcrossTurnsAndAPlanCardWipesIt(t *testing.T) {
	// Momentum through the real round, because what it measures is a *turn* — the one unit no
	// applier-level test can see. Written as two rules with no negation anywhere: one grows on every
	// turn, one resets on a turn holding a plan card, and the reset is applied second.
	momentum := ring(t, "momentum",
		RingRule{
			When: MomentCardDamage,
			Then: []RingEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RingRule{
			When: MomentTurnTaken,
			Then: []RingEffect{{Do: DoGrowOnTurn, Amount: 20}},
		},
		RingRule{
			When: MomentTurnTaken,
			If:   RingCondition{Form: FormPlan, HasForm: true},
			Then: []RingEffect{{Do: DoResetGrowth}},
		})

	d := duelist(100, 8, 100).Wearing(WornRing{Ring: momentum})
	target := duelist(10, 5, 100000)
	strike := Of(Strike, Basic)

	_, d, target = resolve(d, target, []Card{strike}, nil, 1)
	if got := d.WornRings()[0].Grown; got != 20 {
		t.Errorf("one attacking turn left Momentum at %d, want 20", got)
	}

	_, d, target = resolve(d, target, []Card{strike}, nil, 2)
	if got := d.WornRings()[0].Grown; got != 40 {
		t.Errorf("two attacking turns left Momentum at %d, want 40", got)
	}

	// A turn with any plan card in it nets zero, however much else it held.
	_, d, target = resolve(d, target, []Card{strike, Plain(MustConcept("Prepare"))}, nil, 3)
	if got := d.WornRings()[0].Grown; got != 0 {
		t.Errorf("a turn holding a plan card left Momentum at %d, want 0", got)
	}

	// And it starts again from nothing.
	_, d, _ = resolve(d, target, []Card{strike}, nil, 4)
	if got := d.WornRings()[0].Grown; got != 20 {
		t.Errorf("the turn after a reset left Momentum at %d, want 20", got)
	}

	// **An empty turn is still a turn taken**, which is the reading that makes a streak about
	// planning rather than about swinging.
	_, d, _ = resolve(d, target, nil, nil, 5)
	if got := d.WornRings()[0].Grown; got != 40 {
		t.Errorf("an empty turn left Momentum at %d, want 40", got)
	}
}

func TestARingThatResetsItselfDoesNotBankItsGrowth(t *testing.T) {
	// The other half of Momentum: a streak belongs to the duel it was built in. KeepsGrowth is what
	// the run reads, and getting it wrong would turn one good fight into a permanent bonus.
	momentum := ring(t, "momentum-keeps",
		RingRule{When: MomentTurnTaken, Then: []RingEffect{{Do: DoGrowOnTurn, Amount: 20}}},
		RingRule{
			When: MomentTurnTaken,
			If:   RingCondition{Form: FormPlan, HasForm: true},
			Then: []RingEffect{{Do: DoResetGrowth}},
		})
	heart := ring(t, "heart-keeps",
		RingRule{When: MomentFightWon, Then: []RingEffect{{Do: DoGrowOnWin, Amount: 5}}})

	if KeepsGrowth(momentum) {
		t.Error("a ring holding a reset is banked between fights")
	}
	if !KeepsGrowth(heart) {
		t.Error("a ring with no reset is not banked between fights")
	}
}
