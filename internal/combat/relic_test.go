package combat

import (
	"strings"
	"testing"
)

// The relic grammar: what registration refuses, and what wearing one actually does.
//
// **The relics here are built rather than loaded**, because this package cannot read `relics.json` —
// see the file comment in relic.go. That is the point of the split and it is what these tests
// exercise: a rules-level relic is a key, a name and a list of rules, and everything about how it was
// spelled in a file is somebody else's problem.

// relic registers one relic for a test and fails the test rather than the process if it will not take.
// Keys are prefixed so nothing here can collide with the four `internal/session` registers.
func relic(t *testing.T, key string, rules ...RelicRule) RelicID {
	t.Helper()

	id, err := RegisterRelic("relictest."+key, key, rules)
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

func refused(t *testing.T, key string, rules ...RelicRule) {
	t.Helper()

	if _, err := RegisterRelic("relictest.refused."+key, key, rules); err == nil {
		t.Errorf("%s registered, and it should not have", key)
	}
}

// --- what the grammar refuses ------------------------------------------------------------------

func TestAVerbAtTheWrongMomentIsRefused(t *testing.T) {
	// **The failure this prevents is the quiet one**: a rule that loads, never fires, and looks
	// exactly like a relic that does nothing. Every verb belongs to one moment and the table in
	// relic.go is the authority.
	refused(t, "cost at fight-start", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAdjustCost, Amount: -1}},
	})
	refused(t, "status at card-cost", RelicRule{
		When: MomentCardCost,
		Then: []RelicEffect{{Do: DoApplyStatus, Status: 0}},
	})
}

func TestAPredicateOnACardlessMomentIsRefused(t *testing.T) {
	// `fight-start`, `fight-won` and `prizes-dealt` have no card to match an If against, so a rule
	// carrying one is either a misunderstanding or a rule that would silently match everything.
	refused(t, "fight-start with an If", RelicRule{
		When: MomentFightStart,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDMG, Amount: 10}},
	})
}

func TestAnEffectWithNothingToDoIsRefused(t *testing.T) {
	// A zero is a typo in a file authored once, not a reward to be clamped — see checkEffect for
	// why this is the opposite call from the one a worm's amount takes.
	refused(t, "zero damage scale", RelicRule{
		When: MomentCardDamage,
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 0}},
	})
	refused(t, "zero cost delta", RelicRule{
		When: MomentCardCost,
		Then: []RelicEffect{{Do: DoAdjustCost, Amount: 0}},
	})
	refused(t, "a flip to basic", RelicRule{
		When: MomentDeckBuilt,
		Then: []RelicEffect{{Do: DoSetElement, Element: Basic}},
	})
	refused(t, "a relic with no rules at all")
}

func TestAStatusTheFilesDoNotHoldIsRefused(t *testing.T) {
	refused(t, "unknown status", RelicRule{
		When: MomentAttackLands,
		Then: []RelicEffect{{Do: DoApplyStatus, Status: StatusID(StatusCount() + 1)}},
	})
}

// --- what wearing one does --------------------------------------------------------------------

func TestADiscountIsAPropertyOfThePairing(t *testing.T) {
	// The whole reason cost moved off the card: the same Strike costs one duelist less than
	// another, so nothing may ask a card what it costs without saying who is holding it.
	thrifty := relic(t, "thrifty", RelicRule{
		When: MomentCardCost,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAdjustCost, Amount: -1}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRelic{Relic: thrifty})

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
	free := relic(t, "free", RelicRule{
		When: MomentCardCost,
		Then: []RelicEffect{{Do: DoAdjustCost, Amount: -9}},
	})

	d := duelist(10, 5, 100).Wearing(WornRelic{Relic: free})
	if got := d.CardCost(Plain(Smash)); got != 0 {
		t.Errorf("a deeply discounted Smash costs %d, want 0", got)
	}
}

func TestAFormRelicDoublesEveryMatchingCardInTheTurn(t *testing.T) {
	// **Per card is the point** — three slash cards in a turn are three doublings inside the same
	// blow, not one. This checks the per-card figure, which is what the blow is summed from.
	keen := relic(t, "keen", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Form: FormSlash, HasForm: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRelic{Relic: keen})

	if got, want := worn.CardDamage(Plain(Slice)), bare.CardDamage(Plain(Slice))*2; got != want {
		t.Errorf("a Slice under the keen relic deals %d, want %d", got, want)
	}
	if got, want := worn.CardDamage(Plain(Strike)), bare.CardDamage(Plain(Strike)); got != want {
		t.Errorf("the slash relic reached a crush card: %d, want %d", got, want)
	}
}

func TestTwoMatchingRelicsCompound(t *testing.T) {
	// **Compounding is intended**: two slash relics are x4 and that is a build. It is also why worn
	// order is a rule — multiplicative effects are order-sensitive.
	a := relic(t, "keen a", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Form: FormSlash, HasForm: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	b := relic(t, "keen b", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Form: FormSlash, HasForm: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	both := bare.Wearing(WornRelic{Relic: a}).Wearing(WornRelic{Relic: b})

	if got, want := both.CardDamage(Plain(Slice)), bare.CardDamage(Plain(Slice))*4; got != want {
		t.Errorf("two slash relics deal %d, want %d", got, want)
	}
}

func TestAConceptRelicReachesOneCardOnly(t *testing.T) {
	// A concept relic is a much narrower object than a form relic — 4 cards against 12 — which is
	// the distinction this holds and the reason the two must not be priced alike.
	striker := relic(t, "striker", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Concept: Strike, HasConcept: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRelic{Relic: striker})

	if got, want := worn.CardDamage(Plain(Strike)), bare.CardDamage(Plain(Strike))*2; got != want {
		t.Errorf("a Strike under the striker relic deals %d, want %d", got, want)
	}
	for _, id := range []ConceptID{Bash, Smash, Slice} {
		if got, want := worn.CardDamage(Plain(id)), bare.CardDamage(Plain(id)); got != want {
			t.Errorf("%v under a Strike relic deals %d, want %d", ConceptOf(id).Label, got, want)
		}
	}
}

func TestTwoPredicatesNarrowARuleRatherThanWidenIt(t *testing.T) {
	both := relic(t, "fire slash", RelicRule{
		When: MomentCardDamage,
		If: RelicCondition{
			Element: Fire, HasElement: true,
			Form: FormSlash, HasForm: true,
		},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})

	bare := duelist(10, 5, 100)
	worn := bare.Wearing(WornRelic{Relic: both})

	if got, want := worn.CardDamage(Of(Slice, Fire)), bare.CardDamage(Of(Slice, Fire))*2; got != want {
		t.Errorf("a fire slash deals %d, want %d", got, want)
	}
	for _, c := range []Card{Of(Slice, Ice), Of(Strike, Fire)} {
		if got, want := worn.CardDamage(c), bare.CardDamage(c); got != want {
			t.Errorf("%v matched a rule wanting both predicates: %d, want %d", c, got, want)
		}
	}
}

func TestAStatusNamesTheRelicThatAppliedIt(t *testing.T) {
	// **The screen flies the word out of the relic that caused it**, so the event has to say which
	// relic that was. Nothing else can: the card's colour is not the answer, because a relic may
	// match on a form or a concept and apply a status with no colour involved at all - which is
	// the case the second half of this test pins.
	burning := MustStatus("burning")
	chilled := MustStatus("chilled")

	fire := relic(t, "names-fire", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: burning}},
	})
	// A relic that reads the form rather than the colour, which is what makes deriving the relic
	// from the element impossible rather than merely fragile.
	slash := relic(t, "names-slash", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Form: FormSlash, HasForm: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: chilled}},
	})

	a := duelist(10, 8, 500).Wearing(WornRelic{Relic: fire}).Wearing(WornRelic{Relic: slash})
	b := duelist(10, 5, 500)

	// One fire slash matches both relics at once, so both statuses land off one card.
	events, _, _ := resolve(a, b, []Card{Of(Slice, Fire)}, nil, 1)

	got := map[StatusID]RelicID{}
	for _, e := range events {
		if e.Kind == KindStatus {
			got[e.Status] = e.Relic
		}
	}
	if len(got) != 2 {
		t.Fatalf("a fire slash under two relics announced %d statuses, want 2", len(got))
	}
	if got[burning] != fire {
		t.Errorf("the burn is credited to relic %d, want the fire relic %d", got[burning], fire)
	}
	if got[chilled] != slash {
		t.Errorf("the chill is credited to relic %d, want the slash relic %d", got[chilled], slash)
	}
}

func TestTheFirstRelicToApplyAStatusIsTheOneCredited(t *testing.T) {
	// Two relics, one status, one blow. The dedup keeps it to a single event; worn order decides
	// whose it is, which is the tie-break every other compounding effect already takes.
	burning := MustStatus("burning")
	rule := RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: burning}},
	}
	first := relic(t, "credit-first", rule)
	second := relic(t, "credit-second", rule)

	a := duelist(10, 8, 500).Wearing(WornRelic{Relic: first}).Wearing(WornRelic{Relic: second})
	events, _, _ := resolve(a, duelist(10, 5, 500), []Card{Of(Jab, Fire)}, nil, 1)

	n := 0
	for _, e := range events {
		if e.Kind != KindStatus {
			continue
		}
		n++
		if e.Relic != first {
			t.Errorf("the burn is credited to relic %d, want the one worn first, %d", e.Relic, first)
		}
	}
	if n != 1 {
		t.Errorf("two relics applying one status announced it %d times, want 1", n)
	}
}

func TestOneBlowLandsOneOfEachStatus(t *testing.T) {
	// Two fire cards match a fire relic twice. The status does not stack, so applying it twice is
	// the same as applying it once — but announcing it twice would describe two things that did
	// not happen. See statusesFrom.
	burning := MustStatus("burning")
	fire := relic(t, "fire", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: burning}},
	})

	a := duelist(10, 8, 500).Wearing(WornRelic{Relic: fire})
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
	// **`Then` is a list**, which is what buys a relic that shocks *and* chills with no new
	// vocabulary at all — the Storm relic, whole, in one entry.
	shocked, chilled := MustStatus("shocked"), MustStatus("chilled")
	storm := relic(t, "storm", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Lightning, HasElement: true},
		Then: []RelicEffect{
			{Do: DoApplyStatus, Status: shocked},
			{Do: DoApplyStatus, Status: chilled},
		},
	})

	a := duelist(10, 5, 500).Wearing(WornRelic{Relic: storm})
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
	toIce := relic(t, "lightning to ice", RelicRule{
		When: MomentCardDrawn,
		If:   RelicCondition{Element: Lightning, HasElement: true},
		Then: []RelicEffect{{Do: DoSetElement, Element: Ice}},
	})
	toEarth := relic(t, "ice to earth", RelicRule{
		When: MomentCardDrawn,
		If:   RelicCondition{Element: Ice, HasElement: true},
		Then: []RelicEffect{{Do: DoSetElement, Element: Earth}},
	})

	worn := []WornRelic{{Relic: toIce}, {Relic: toEarth}}

	if e, ok := FlipElement(worn, Of(Strike, Lightning)); !ok || e != Ice {
		t.Errorf("a lightning card became %v (flipped %v), want ice — the second flip chained", e, ok)
	}
	if e, ok := FlipElement(worn, Of(Strike, Ice)); !ok || e != Earth {
		t.Errorf("an ice card became %v (flipped %v), want earth", e, ok)
	}
	if _, ok := FlipElement(worn, Of(Strike, Fire)); ok {
		t.Error("a fire card was flipped by relics that do not name it")
	}
}

func TestTheAccumulatorRidesOnTheWornRelic(t *testing.T) {
	// A growing relic's own amounts are read as `Amount + accumulator`, and the accumulator travels
	// with the worn relic because it belongs to a run rather than to the registry.
	heart := relic(t, "heart",
		RelicRule{When: MomentFightStart, Then: []RelicEffect{{Do: DoAddHP, Amount: 5}}},
		RelicRule{When: MomentFightWon, Then: []RelicEffect{{Do: DoGrowOnWin, Amount: 5}}})

	fresh := []WornRelic{{Relic: heart}}
	grown := []WornRelic{{Relic: heart, Grown: 20}}

	if got := AddedHP(fresh); got != 5 {
		t.Errorf("a fresh heart relic adds %d HP, want 5", got)
	}
	if got := AddedHP(grown); got != 25 {
		t.Errorf("a heart relic at +20 adds %d HP, want 25", got)
	}
	if got := Growth(fresh[0]); got != 5 {
		t.Errorf("a heart relic grows by %d, want 5", got)
	}
	if got := Growth(grown[0]); got != 5 {
		t.Errorf("a grown heart relic grows by %d, want 5 — growth must not compound on itself", got)
	}
}

func TestPropagationScalesAndCompounds(t *testing.T) {
	// The relic scales what the run's own cap produced, and two of them compound like every other
	// relic effect. The cap itself is the run's business — see session.propagate.
	banker := relic(t, "banker", RelicRule{
		When: MomentFightWon,
		Then: []RelicEffect{{Do: DoScalePropagation, Amount: 200}},
	})

	if got := ScalePropagation(nil, 5); got != 5 {
		t.Errorf("bare propagation of 5 came out as %d", got)
	}
	if got := ScalePropagation([]WornRelic{{Relic: banker}}, 5); got != 10 {
		t.Errorf("one banker turned 5 into %d, want 10", got)
	}
	if got := ScalePropagation([]WornRelic{{Relic: banker}, {Relic: banker}}, 5); got != 20 {
		t.Errorf("two bankers turned 5 into %d, want 20", got)
	}
}

func TestARelicIsOnlyWornOnceTheHandIsNotFull(t *testing.T) {
	// Five worn at once, until brands expand it. RelicSlots is the cap and Wearing is where it
	// bites; a sixth is dropped rather than overwriting the fifth.
	//
	// **The array is no longer the cap**, which is the thing this test now has to say twice: a
	// duelist carrying no number of its own wears DefaultRelicSlots, and one carrying a number wears
	// that — up to MaxWornRelics, which is only how wide the array is.
	worn := relic(t, "filler", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAddDMG, Amount: 1}},
	})

	fill := func(d Duelist) Duelist {
		for i := 0; i < MaxWornRelics+3; i++ {
			d = d.Wearing(WornRelic{Relic: worn})
		}
		return d
	}

	d := fill(duelist(10, 5, 100))
	if d.RelicCount != DefaultRelicSlots {
		t.Errorf("a duelist ended up wearing %d relics, cap is %d", d.RelicCount, DefaultRelicSlots)
	}
	if got := AddedDMG(d.WornRelics()); got != DefaultRelicSlots {
		t.Errorf("%d relics added %d DMG, want %d", d.RelicCount, got, DefaultRelicSlots)
	}

	raised := duelist(10, 5, 100)
	raised.RelicSlots = DefaultRelicSlots + 1
	if raised = fill(raised); raised.RelicCount != DefaultRelicSlots+1 {
		t.Errorf("a duelist with %d slots wore %d relics", DefaultRelicSlots+1, raised.RelicCount)
	}

	// A cap past the array's width is clamped to it rather than writing off the end. The run is
	// free to be wrong about this; the struct is not.
	past := duelist(10, 5, 100)
	past.RelicSlots = MaxWornRelics + 5
	if past = fill(past); past.RelicCount != MaxWornRelics {
		t.Errorf("a duelist with %d slots wore %d relics, and the array holds %d",
			MaxWornRelics+5, past.RelicCount, MaxWornRelics)
	}
}

func TestAnEnemyWearsNothing(t *testing.T) {
	// **Relics are the duelist's only.** The zero value is an empty hand, which is what an enemy is
	// hydrated with — so an enemy's colours are inert by construction rather than by a rule written
	// down somewhere else.
	var enemy Duelist
	if n := len(enemy.WornRelics()); n != 0 {
		t.Errorf("a zero duelist wears %d relics", n)
	}
	if got := enemy.statusesFrom([]Card{Of(Strike, Fire)}); len(got) != 0 {
		t.Errorf("a relicless duelist's fire Strike applied %d statuses", len(got))
	}
}

func TestHPScalingCompoundsAndDefaultsToWhole(t *testing.T) {
	// Onslaught's half of the grammar: a fight-start scaling that goes *below* 100, which no other
	// scaling verb does. A bare duelist has to come out untouched, or every relic in the file would
	// be quietly resizing a life bar.
	if got := HPScale(nil); got != 100 {
		t.Errorf("nothing worn scales life to %d%%, want 100%%", got)
	}

	quarterOff := relic(t, "onslaught", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoScaleHP, Amount: 75}},
	})

	if got := HPScale([]WornRelic{{Relic: quarterOff}}); got != 75 {
		t.Errorf("one drawback relic scales life to %d%%, want 75%%", got)
	}

	// Compounding rather than adding, like every other multiplicative effect: two quarters off
	// leave 56%, not half.
	two := []WornRelic{{Relic: quarterOff}, {Relic: quarterOff}}
	if got := HPScale(two); got != 56 {
		t.Errorf("two drawback relics scale life to %d%%, want 56%% — they are adding, not "+
			"compounding", got)
	}
}

func TestAnEchoSeatsTheLeadCardAgainAtDecreasingAmounts(t *testing.T) {
	// Echo's whole shape in one place: the lead card of the blow pays three terms rather than one,
	// the sum grows by exactly those terms, and the hand the cards formed is untouched.
	echo := relic(t, "echo", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Lead: true},
		Then: []RelicEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	card := Of(Strike, Fire)

	if got := LandingAmounts(nil, card, true, 30); len(got) != 1 || got[0] != 30 {
		t.Errorf("a bare duelist pays %v for a 30 card, want [30]", got)
	}

	worn := []WornRelic{{Relic: echo}}
	if got := LandingAmounts(worn, card, true, 30); len(got) != 3 ||
		got[0] != 30 || got[1] != 20 || got[2] != 10 {
		t.Errorf("Echo pays %v for a 30 lead card, want [30 20 10]", got)
	}

	// **Only the lead card**, which is what the Lead predicate is for.
	if got := LandingAmounts(worn, card, false, 30); len(got) != 1 {
		t.Errorf("Echo pays %v for a card that does not lead the blow, want one term", got)
	}

	// Two of them add a landing each rather than multiplying: five landings, not nine.
	if got := LandingAmounts([]WornRelic{{Relic: echo}, {Relic: echo}}, card, true, 30); len(got) != 5 {
		t.Errorf("two echo relics pay %v, want five terms", got)
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
	// The form repeat relics: every card the rule matches lands twice, both at full strength — where
	// an echo diminishes and only takes the lead card.
	repeat := relic(t, "aftershock", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Form: FormCrush, HasForm: true},
		Then: []RelicEffect{{Do: DoRepeatCard, Amount: 2}},
	})
	worn := []WornRelic{{Relic: repeat}}

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
	repeat := relic(t, "aftershock-2", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Form: FormCrush, HasForm: true},
		Then: []RelicEffect{{Do: DoRepeatCard, Amount: 2}},
	})
	echo := relic(t, "echo-2", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Lead: true},
		Then: []RelicEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	// Repeat first, then the echo ladder over the echo's own count: 30, 30 (the copy), then
	// two thirds and one third.
	got := LandingAmounts([]WornRelic{{Relic: repeat}, {Relic: echo}}, crushCard(t), true, 30)
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
	many := []WornRelic{{Relic: repeat}, {Relic: repeat}, {Relic: repeat}, {Relic: echo}, {Relic: echo}}
	if got := LandingAmounts(many, crushCard(t), true, 30); len(got) > MaxEchoLandings {
		t.Errorf("five stacked relics pay %d terms, want at most %d", len(got), MaxEchoLandings)
	}
}

func TestAtrophyStepsThreeAPAttacksDownOneRung(t *testing.T) {
	// Atrophy's whole shape: the top rung of each form becomes the middle rung, nothing else moves,
	// and the ladder is read off the declared cost rather than off what the wearer pays.
	atrophy := relic(t, "atrophy", RelicRule{
		When: MomentDeckBuilt,
		If:   RelicCondition{Tier: 3, HasTier: true},
		Then: []RelicEffect{{Do: DoDemoteCard, Amount: 1}},
	})
	worn := []WornRelic{{Relic: atrophy}}

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

func TestAGrowOnHitRelicGetsStrongerInsideOneFight(t *testing.T) {
	// The Enflamed family: the accumulator moves as a blow lands, so the second fire attack of a
	// fight is already worth more than the first — where every other growing relic waits for the
	// win. What is checked here is the arithmetic; the seat in resolveAttackPhase is what makes a
	// real blow reach it.
	enflamed := relic(t, "enflamed",
		RelicRule{
			When: MomentCardDamage,
			If:   RelicCondition{Element: Fire, HasElement: true},
			Then: []RelicEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RelicRule{
			When: MomentAttackLands,
			If:   RelicCondition{Element: Fire, HasElement: true},
			Then: []RelicEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	d := duelist(100, 5, 100).Wearing(WornRelic{Relic: enflamed})

	fire, ice := Of(Strike, Fire), Of(Strike, Ice)

	// Fresh, the relic is worth nothing: 100% of the card is the card.
	if got, want := d.CardDamage(fire), d.CardDamage(ice); got != want {
		t.Errorf("an ungrown Enflamed deals %d where a plain card deals %d", got, want)
	}

	// A landing of a fire card steps the accumulator; an ice one does not.
	d = d.GrowOnLanding(ice)
	if got := d.WornRelics()[0].Grown; got != 0 {
		t.Errorf("an ice landing grew a fire relic to %d, want 0", got)
	}

	d = d.GrowOnLanding(fire)
	if got := d.WornRelics()[0].Grown; got != 10 {
		t.Errorf("one fire landing grew the relic to %d, want 10", got)
	}

	// **Once per landing** *(owner's call, 2026-08-22, and per card inside the blow since
	// 2026-08-26)*: two fire cards in one hand are two steps, where a status would land once.
	d = d.GrowOnLanding(fire).GrowOnLanding(fire)
	if got := d.WornRelics()[0].Grown; got != 30 {
		t.Errorf("three fire landings left the relic at %d, want 30", got)
	}

	// Three landings in, fire cards are worth 1.3x and nothing else has moved.
	if got, want := d.CardDamage(fire), d.CardDamage(ice)*130/100; got != want {
		t.Errorf("after three fire landings a fire card deals %d, want %d", got, want)
	}

	// **The growth does not itself grow.** Reading the step as Amount+Grown would compound it.
	before := d.WornRelics()[0].Grown
	d = d.GrowOnLanding(fire)
	if got := d.WornRelics()[0].Grown - before; got != 10 {
		t.Errorf("the fourth fire landing stepped the relic by %d, want 10 — the step is compounding", got)
	}
}

func TestEchoAndAGrowOnHitRelicCompound(t *testing.T) {
	// **The combination is the point.** A card an echo relic seats three times hit three times, so an
	// Enflamed Ring worn beside Echo grows three steps off one card rather than one. If this ever
	// starts counting cards again, the two relics quietly stop being a build.
	enflamed := relic(t, "enflamed-echo",
		RelicRule{
			When: MomentAttackLands,
			If:   RelicCondition{Element: Fire, HasElement: true},
			Then: []RelicEffect{{Do: DoGrowOnHit, Amount: 10}},
		})
	echo := relic(t, "echo-enflamed", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Lead: true},
		Then: []RelicEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	fire := Of(Strike, Fire)

	// **Through the real round**, because the echo's extra landings are seated by the sum: they are
	// terms of one blow rather than cards of a turn, so nothing below the round can see them.
	alone := duelist(100, 5, 100).Wearing(WornRelic{Relic: enflamed})
	_, grown, _ := resolve(alone, duelist(10, 5, 100000), []Card{fire}, nil, 1)
	if got := grown.WornRelics()[0].Grown; got != 10 {
		t.Errorf("one fire card without Echo grew the relic by %d, want 10", got)
	}

	both := duelist(100, 5, 100).
		Wearing(WornRelic{Relic: enflamed}).
		Wearing(WornRelic{Relic: echo})
	_, grown, _ = resolve(both, duelist(10, 5, 100000), []Card{fire}, nil, 1)
	if got := grown.WornRelics()[0].Grown; got != 30 {
		t.Errorf("one echoed fire card grew the relic by %d, want 30 — the echo's landings are "+
			"not being counted", got)
	}
}

func TestAFourOfAKindGrowsOnceForEachCard(t *testing.T) {
	// **Through the real round, not through the applier**, because the bug this guards against is a
	// seat rather than a formula: a four of a kind is one *blow*, and an accumulator that took the
	// blow as its unit would step once where the player threw four attacks.
	enflamed := relic(t, "enflamed-round",
		RelicRule{
			When: MomentCardDamage,
			If:   RelicCondition{Element: Fire, HasElement: true},
			Then: []RelicEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RelicRule{
			When: MomentAttackLands,
			If:   RelicCondition{Element: Fire, HasElement: true},
			Then: []RelicEffect{{Do: DoGrowOnHit, Amount: 10}},
		})

	attacker := duelist(10, 8, 100).Wearing(WornRelic{Relic: enflamed})
	fire := Of(Strike, Fire)

	_, after, _ := resolve(attacker, duelist(10, 5, 1000),
		[]Card{fire, fire, fire, fire}, nil, 1)

	if got := after.WornRelics()[0].Grown; got != 40 {
		t.Errorf("a fire Four of a Kind grew the relic by %d, want 40 — one step per card that "+
			"landed, not one per blow", got)
	}

	// A hand of one colour among others still only pays for its own colour.
	_, mixed, _ := resolve(attacker, duelist(10, 5, 1000),
		[]Card{fire, Of(Strike, Ice), fire}, nil, 1)
	if got := mixed.WornRelics()[0].Grown; got != 20 {
		t.Errorf("two fire cards beside an ice one grew the relic by %d, want 20", got)
	}
}

func TestMomentumBuildsAcrossTurnsAndADefenceWipesIt(t *testing.T) {
	// Momentum through the real round, because what it measures is a *turn* — the one unit no
	// applier-level test can see. Written as two rules with no negation anywhere: one grows on every
	// turn, one resets on a turn holding a defence, and the reset is applied second.
	momentum := relic(t, "momentum",
		RelicRule{
			When: MomentCardDamage,
			Then: []RelicEffect{{Do: DoScaleDamage, Amount: 100}},
		},
		RelicRule{
			When: MomentTurnTaken,
			Then: []RelicEffect{{Do: DoGrowOnTurn, Amount: 20}},
		},
		RelicRule{
			When: MomentTurnTaken,
			If:   RelicCondition{Form: FormDefend, HasForm: true},
			Then: []RelicEffect{{Do: DoResetGrowth}},
		})

	d := duelist(100, 8, 100).Wearing(WornRelic{Relic: momentum})
	target := duelist(10, 5, 100000)
	strike := Of(Strike, Basic)

	_, d, target = resolve(d, target, []Card{strike}, nil, 1)
	if got := d.WornRelics()[0].Grown; got != 20 {
		t.Errorf("one attacking turn left Momentum at %d, want 20", got)
	}

	_, d, target = resolve(d, target, []Card{strike}, nil, 2)
	if got := d.WornRelics()[0].Grown; got != 40 {
		t.Errorf("two attacking turns left Momentum at %d, want 40", got)
	}

	// A turn with any plan card in it nets zero, however much else it held.
	_, d, target = resolve(d, target, []Card{strike, Plain(Ward)}, nil, 3)
	if got := d.WornRelics()[0].Grown; got != 0 {
		t.Errorf("a turn holding a plan card left Momentum at %d, want 0", got)
	}

	// And it starts again from nothing.
	_, d, _ = resolve(d, target, []Card{strike}, nil, 4)
	if got := d.WornRelics()[0].Grown; got != 20 {
		t.Errorf("the turn after a reset left Momentum at %d, want 20", got)
	}

	// **An empty turn is still a turn taken**, which is the reading that makes a streak about
	// planning rather than about swinging.
	_, d, _ = resolve(d, target, nil, nil, 5)
	if got := d.WornRelics()[0].Grown; got != 40 {
		t.Errorf("an empty turn left Momentum at %d, want 40", got)
	}
}

func TestARelicThatResetsItselfDoesNotBankItsGrowth(t *testing.T) {
	// The other half of Momentum: a streak belongs to the duel it was built in. KeepsGrowth is what
	// the run reads, and getting it wrong would turn one good fight into a permanent bonus.
	momentum := relic(t, "momentum-keeps",
		RelicRule{When: MomentTurnTaken, Then: []RelicEffect{{Do: DoGrowOnTurn, Amount: 20}}},
		RelicRule{
			When: MomentTurnTaken,
			If:   RelicCondition{Form: FormDefend, HasForm: true},
			Then: []RelicEffect{{Do: DoResetGrowth}},
		})
	heart := relic(t, "heart-keeps",
		RelicRule{When: MomentFightWon, Then: []RelicEffect{{Do: DoGrowOnWin, Amount: 5}}})

	if KeepsGrowth(momentum) {
		t.Error("a relic holding a reset is banked between fights")
	}
	if !KeepsGrowth(heart) {
		t.Error("a relic with no reset is not banked between fights")
	}
}

// --- the hand bonus ----------------------------------------------------------------------------

// handEventOf is the KindHand event one side produced, for the tests that read the blow's own sum.
func handEventOf(t *testing.T, events []Event, side Side) Event {
	t.Helper()

	for _, e := range events {
		if e.Kind == KindHand && e.Side == side {
			return e
		}
	}
	t.Fatalf("no hand event for %v", side)
	return Event{}
}

func TestAHandRuleIsRefusedAnywhereButBlowFormed(t *testing.T) {
	// **Only blow-formed knows what formed**, exactly as only blow-formed knows which card leads.
	// A `Hand` predicate anywhere else would match nothing and read as a relic that does nothing.
	pair, ok := HandIDForKey("pair")
	if !ok {
		t.Fatal("the ladder has no concept-pair, so this test cannot say what it means")
	}

	refused(t, "hand at card-damage", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Hand: pair, HasHand: true},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	refused(t, "add-hand-damage at card-damage", RelicRule{
		When: MomentCardDamage,
		Then: []RelicEffect{{Do: DoAddHandDamage, Amount: 5}},
	})
}

func TestAHandRuleMayNotAlsoNameACard(t *testing.T) {
	// **A hand is a fact about the whole blow and an element is a fact about one card**, so a rule
	// carrying both is asking a question with no answer — which of the hand's cards would have to
	// be fire? It is refused rather than resolved to one reading nobody wrote down.
	pair, _ := HandIDForKey("pair")

	refused(t, "hand and element", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: pair, HasHand: true, Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddHandDamage, Amount: 5}},
	})
}

func TestAHandRelicPaysOnlyItsOwnRung(t *testing.T) {
	// **The predicate is the whole relic**, so the test that matters is the negative one: a relic
	// naming Form Three of a Kind must be worth nothing on the turns that build something else.
	// **Three of one concept, which is the rung three identical cards actually build.** Card
	// Three of a Kind pays 250 where the form rung pays 150, and the matcher takes the best — so a
	// test naming the form rung here would be describing a hand the game never forms.
	trips, ok := HandIDForKey("concept-three-of-a-kind")
	if !ok {
		t.Fatal("the ladder has no concept-three-of-a-kind")
	}
	forged := relic(t, "forged", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: trips, HasHand: true},
		Then: []RelicEffect{{Do: DoAddHandDamage, Amount: 3}},
	})

	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: forged})
	slash := slashCard(t)
	crush := crushCard(t)

	// Three of one form is the rung the relic names.
	events, _, _ := resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash, slash}, nil, 1)
	trip := handEventOf(t, events, SideA)
	if trip.HandBonus != 3 {
		t.Errorf("a form three of a kind paid %d, want 3", trip.HandBonus)
	}

	// Two of one form and one of another cannot be that rung.
	events, _, _ = resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash, crush}, nil, 1)
	other := handEventOf(t, events, SideA)
	if other.Hand == trips {
		t.Fatal("two slashes and a crush formed a form three of a kind, so this test proves nothing")
	}
	if other.HandBonus != 0 {
		t.Errorf("a hand that is not the named rung paid %d, want 0", other.HandBonus)
	}
}

func TestTheHandBonusIsAddedBeforeTheMultiplier(t *testing.T) {
	// **This is the whole design decision** *(owner's call, 2026-09-05)*: the bonus is a term of
	// the base sum rather than something added to the answer, so a rung's bonus is worth more on
	// the rung that pays more. A bonus applied after the multiplier would be the same relic at
	// every rung, and nothing on screen would say which it was.
	trips, _ := HandIDForKey("concept-three-of-a-kind")
	forged := relic(t, "forged-order", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: trips, HasHand: true},
		Then: []RelicEffect{{Do: DoAddHandDamage, Amount: 3}},
	})

	slash := slashCard(t)
	cards := []Card{slash, slash, slash}

	bare, _, _ := resolve(duelist(10, 5, 100), duelist(10, 5, 100000), cards, nil, 1)
	worn, _, _ := resolve(duelist(10, 5, 100).Wearing(WornRelic{Relic: forged}),
		duelist(10, 5, 100000), cards, nil, 1)

	before := handEventOf(t, bare, SideA)
	after := handEventOf(t, worn, SideA)

	if after.Base != before.Base+3 {
		t.Errorf("the bonus put %d into the base sum, want 3", after.Base-before.Base)
	}
	if want := scaleDamage(before.Base+3, after.Multiplier); after.Amount != want {
		t.Errorf("the blow came to %d, want %d — the bonus is not being multiplied with the cards",
			after.Amount, want)
	}
	if after.Amount == before.Amount+3 {
		t.Error("the bonus landed after the multiplier, and it is meant to land before it")
	}
}

func TestTwoHandRelicsOnOneRungAdd(t *testing.T) {
	// **Flat terms in a sum, so there is nothing to compound** — unlike the multipliers, where worn
	// order decides the result. This is what makes the rung relics the one family whose order on the
	// hand does not matter.
	pair, _ := HandIDForKey("pair")
	rule := RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: pair, HasHand: true},
		Then: []RelicEffect{{Do: DoAddHandDamage, Amount: 4}},
	}
	one := relic(t, "pairbonus-one", rule)
	two := relic(t, "pairbonus-two", rule)

	slash := slashCard(t)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: one}).Wearing(WornRelic{Relic: two})

	events, _, _ := resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash}, nil, 1)
	e := handEventOf(t, events, SideA)
	if e.HandBonus != 8 {
		t.Errorf("two relics on one rung paid %d, want 8", e.HandBonus)
	}
	if !e.HandBonusSeats[0] || !e.HandBonusSeats[1] {
		t.Error("both relics paid, so both seats have to be attributable")
	}
}

// --- the held bonus ----------------------------------------------------------------------------

func TestAHeldRuleIsRefusedAlongsideABlowPredicate(t *testing.T) {
	// **A held card is in neither pile the blow predicates name.** `Lead` is the first card played
	// and `Hand` is the rung the played cards formed, so either beside this verb is a rule whose
	// two halves are about different things.
	pair, _ := HandIDForKey("pair")

	refused(t, "held and lead", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Lead: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})
	refused(t, "held and hand", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: pair, HasHand: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})
	refused(t, "held at card-damage", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})
}

func TestTheHeldBonusPaysPerMatchingCardKeptBack(t *testing.T) {
	// **Once per match, not once per turn.** The whole point of the relic is that a second held
	// fire card is worth as much as the first — a flat per-card term, which is what makes holding
	// a colour a decision rather than a threshold.
	smoulder := relic(t, "smoulder", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})

	fire := Of(Strike, Fire)
	ice := Of(Strike, Ice)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: smoulder})

	for _, tc := range []struct {
		name string
		held []Card
		want int
	}{
		{"nothing held", nil, 0},
		{"one fire held", []Card{fire}, 5},
		{"two fire held", []Card{fire, fire}, 10},
		{"the wrong colour held", []Card{ice, ice}, 0},
		{"one of each", []Card{fire, ice}, 5},
	} {
		got, seats := HeldBonus(wearer.WornRelics(), tc.held)
		if got != tc.want {
			t.Errorf("%s paid %d, want %d", tc.name, got, tc.want)
		}
		if paid := seats[0]; paid != (tc.want > 0) {
			t.Errorf("%s attributed the bonus to seat 0 = %v, want %v", tc.name, paid, tc.want > 0)
		}
	}
}

func TestTheHeldBonusReachesTheBlowAndIsMultiplied(t *testing.T) {
	// **Through the real round**, because the seat is the thing being tested: the held hand is a
	// parameter of ResolveRound that only the blow's own sum ever reads, and a verb wired to the
	// wrong pile would still pass every unit test of HeldBonus.
	smoulder := relic(t, "smoulder-round", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})

	played := []Card{Of(Slice, Earth), Of(Slice, Earth)}
	held := []Card{Of(Strike, Fire), Of(Strike, Fire)}

	bare, _, _ := ResolveRoundHolding(duelist(10, 5, 100), duelist(10, 5, 100000),
		played, nil, held, nil, 1, Sources{})
	worn, _, _ := ResolveRoundHolding(duelist(10, 5, 100).Wearing(WornRelic{Relic: smoulder}),
		duelist(10, 5, 100000), played, nil, held, nil, 1, Sources{})

	before := handEventOf(t, bare, SideA)
	after := handEventOf(t, worn, SideA)

	if after.HeldBonus != 10 {
		t.Errorf("two held fire cards paid %d into the blow, want 10", after.HeldBonus)
	}
	if after.Base != before.Base+10 {
		t.Errorf("the held bonus put %d into the base sum, want 10", after.Base-before.Base)
	}
	if want := scaleDamage(before.Base+10, after.Multiplier); after.Amount != want {
		t.Errorf("the blow came to %d, want %d — the held bonus is not being multiplied with the cards",
			after.Amount, want)
	}
}

func TestAHeldCardPaysAgainEveryTurnItIsStillHeld(t *testing.T) {
	// **It is a fact about the hand, not an event.** A card kept back is not spent, so the relic
	// pays for it again next turn — which is what separates this from anything that fires once.
	bedrock := relic(t, "bedrock", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Element: Earth, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})

	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: bedrock})
	played := []Card{Of(Slice, Fire), Of(Slice, Fire)}
	held := []Card{Of(Strike, Earth)}

	for round := 1; round <= 3; round++ {
		events, _, _ := ResolveRoundHolding(wearer, duelist(10, 5, 100000),
			played, nil, held, nil, round, Sources{})
		e := handEventOf(t, events, SideA)
		if e.HeldBonus != 5 {
			t.Errorf("round %d paid %d for the same held card, want 5", round, e.HeldBonus)
		}
	}
}

// defendCardForTest is any shield card the registry holds, for the turn-taken rules that count them.
func defendCardForTest(t *testing.T) Card {
	t.Helper()

	for id := ConceptID(0); int(id) < ConceptCount(); id++ {
		if ConceptOf(id).Form == FormDefend {
			return Of(id, Basic)
		}
	}
	t.Fatal("no defend concept is registered")
	return Card{}
}

// TestGrowPerCardCountsRatherThanFires. The whole difference between the two turn-taken growth
// verbs: grow-on-turn takes one step for a turn holding any match, this one takes a step per match.
// A relic worth the same for one shield as for three would be grow-on-turn under a longer name.
func TestGrowPerCardCountsRatherThanFires(t *testing.T) {
	id := relic(t, "ebbtest.perCard", RelicRule{
		When: MomentTurnTaken,
		If:   RelicCondition{Form: FormDefend, HasForm: true},
		Then: []RelicEffect{{Do: DoGrowPerCard, Amount: 20}},
	})

	shield := defendCardForTest(t)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})

	after := wearer.TurnTaken([]Card{shield, shield, shield, slashCard(t)})
	if after.Relics[0].Grown != 60 {
		t.Errorf("three shields grew the relic by %d, want 60", after.Relics[0].Grown)
	}
}

// TestGrowPerCardIgnoresATurnWithNoMatch. A turn of pure attacks is not a step of zero, it is no
// step at all — the same reading anyMatches gives grow-on-turn.
func TestGrowPerCardIgnoresATurnWithNoMatch(t *testing.T) {
	id := relic(t, "ebbtest.noMatch", RelicRule{
		When: MomentTurnTaken,
		If:   RelicCondition{Form: FormDefend, HasForm: true},
		Then: []RelicEffect{{Do: DoGrowPerCard, Amount: 20}},
	})

	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})
	if after := wearer.TurnTaken([]Card{slashCard(t)}); after.Relics[0].Grown != 0 {
		t.Errorf("a turn with no shield in it grew the relic by %d, want 0", after.Relics[0].Grown)
	}
}

// TestGrowPerCardIsRefusedWithNothingToCount. A per-card step with no predicate would count every
// card of the turn, which is a thing the file cannot say it meant — and is grow-on-turn's job.
func TestGrowPerCardIsRefusedWithNothingToCount(t *testing.T) {
	refused(t, "perCardBare", RelicRule{
		When: MomentTurnTaken,
		Then: []RelicEffect{{Do: DoGrowPerCard, Amount: 20}},
	})
}

// TestTheVitaeBonusReachesTheBlowAndIsMultiplied. Rampant's figure is a fact about the run, not
// about a card, so it joins Base after every card term and is scaled with the rest of them.
func TestTheVitaeBonusReachesTheBlowAndIsMultiplied(t *testing.T) {
	id := relic(t, "rampanttest.pays", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAddDamagePerVitae, Amount: 1}},
	})

	slash := slashCard(t)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})
	wearer.Vitae = 30 // the purse session.Equip seeded the duel with

	events, _, _ := resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash}, nil, 1)
	e := handEventOf(t, events, SideA)

	if e.VitaeBonus != 30 {
		t.Errorf("the purse paid %d, want 30", e.VitaeBonus)
	}
	if !e.VitaeBonusSeats[0] {
		t.Error("the relic that pays has to be attributable, or the figure cannot fly from it")
	}
	if want := scaleDamage(e.Base, e.Multiplier); e.Amount != want {
		t.Errorf("the blow landed %d where its own base and multiplier say %d", e.Amount, want)
	}
}

// TestDamagePerVitaeIsAskedOfTheRelicsAndNotTheDuelist. The seam that keeps the purse out of the
// rules: combat reports the *rate*, and whoever knows what the run is carrying does the sum.
func TestDamagePerVitaeIsAskedOfTheRelicsAndNotTheDuelist(t *testing.T) {
	one := relic(t, "rampanttest.rateOne", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAddDamagePerVitae, Amount: 1}},
	})
	two := relic(t, "rampanttest.rateTwo", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAddDamagePerVitae, Amount: 2}},
	})

	worn := []WornRelic{{Relic: one}, {Relic: two}}
	if got := DamagePerVitae(worn); got != 3 {
		t.Errorf("two relics rated %d a vitae between them, want 3", got)
	}
	if got := DamagePerVitae(nil); got != 0 {
		t.Errorf("a bare duelist is rated %d a vitae, want 0", got)
	}
}

// TestThePurseIsReReadEveryBlow. The correction that made Rampant right: vitae moves *during* a
// fight — a card kept in hand pays one every turn it is held — so a relic reading the purse has to
// be re-asked at each blow. A figure resolved once at fight-start pays a late turn at opening
// prices, which is the bug this holds against.
func TestThePurseIsReReadEveryBlow(t *testing.T) {
	id := relic(t, "rampanttest.reread", RelicRule{
		When: MomentFightStart,
		Then: []RelicEffect{{Do: DoAddDamagePerVitae, Amount: 1}},
	})

	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})
	wearer.Vitae = 10
	played := []Card{slashCard(t), slashCard(t)}
	held := []Card{carrying(Jab, RiderVitaeInHand, 3)}

	// **Riders fire before the blow**, by the design note in resolveTurn — so the 3 this turn's
	// held card pays is already in the purse the blow reads. The opening balance of 10 therefore
	// buys 13 on round one, and the bonus climbs by 3 a round after that.
	for round, want := 1, 13; round <= 3; round, want = round+1, want+3 {
		events, after, _ := ResolveRoundHolding(wearer, duelist(10, 5, 100000),
			played, nil, held, nil, round, Sources{})

		e := handEventOf(t, events, SideA)
		if e.VitaeBonus != want {
			t.Errorf("round %d paid %d on a purse of %d, want %d", round, e.VitaeBonus, wearer.Vitae, want)
		}
		wearer = after
	}
}

// TestAHandScalerIsASecondMultiplierAndNotABiggerHand. The rule the owner set: `Multiplier` is the
// ladder's own figure and a relic may not move it — the banner, the hand row and the sum all show
// the rung the player actually built. What the relic does is scale the result afterwards.
func TestAHandScalerIsASecondMultiplierAndNotABiggerHand(t *testing.T) {
	pair, _ := HandIDForKey("pair")
	id := relic(t, "pairing", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: pair, HasHand: true},
		Then: []RelicEffect{{Do: DoScaleHandDamage, Amount: 200}},
	})

	slash := slashCard(t)
	bare := duelist(10, 5, 100)
	wearer := bare.Wearing(WornRelic{Relic: id})

	plain, _, _ := resolve(bare, duelist(10, 5, 100000), []Card{slash, slash}, nil, 1)
	scaled, _, _ := resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash}, nil, 1)

	was, now := handEventOf(t, plain, SideA), handEventOf(t, scaled, SideA)
	if now.HandScale != 200 {
		t.Errorf("the relic scaled the hand by %d, want 200", now.HandScale)
	}
	if now.Multiplier != was.Multiplier {
		t.Errorf("the relic moved the hand's own multiplier from %d to %d, and it may not",
			was.Multiplier, now.Multiplier)
	}
	if now.Base != was.Base {
		t.Errorf("the relic moved Base from %d to %d, and it may only scale the result", was.Base, now.Base)
	}
	if now.Amount != was.Amount*2 {
		t.Errorf("the blow landed %d against %d unworn, want double", now.Amount, was.Amount)
	}
	if !now.HandScaleSeats[0] {
		t.Error("the relic that scaled has to be attributable, or the figure cannot fly from it")
	}
}

// TestAHandScalerPaysOnlyItsOwnRung. Same guard the flat rung relics carry: the matcher reports one
// rung, so a relic naming a different one is silent.
func TestAHandScalerPaysOnlyItsOwnRung(t *testing.T) {
	trips, _ := HandIDForKey("concept-three-of-a-kind")
	id := relic(t, "tripsonly", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: trips, HasHand: true},
		Then: []RelicEffect{{Do: DoScaleHandDamage, Amount: 300}},
	})

	slash := slashCard(t)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})

	events, _, _ := resolve(wearer, duelist(10, 5, 100000), []Card{slash, slash}, nil, 1)
	if e := handEventOf(t, events, SideA); e.HandScale != 100 {
		t.Errorf("a pair paid a Three of a Kind relic %d, want the identity", e.HandScale)
	}
}

// TestMinFormsCountsTheScoringSet. Dual Wield's predicate: a pair built from two different weapons
// pays, and a pair of the same weapon does not.
func TestMinFormsCountsTheScoringSet(t *testing.T) {
	pair, _ := HandIDForKey("pair")
	id := relic(t, "dualwield", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Hand: pair, HasHand: true, MinForms: 2},
		Then: []RelicEffect{{Do: DoScaleHandDamage, Amount: 300}},
	})

	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})

	// Two fire cards of different forms: an elemental pair covering two weapons.
	mixed, _, _ := resolve(wearer, duelist(10, 5, 100000),
		[]Card{Of(Slice, Fire), Of(Strike, Fire)}, nil, 1)
	if e := handEventOf(t, mixed, SideA); e.HandScale != 300 {
		t.Errorf("a pair of two different forms paid %d, want 300", e.HandScale)
	}

	// Two fire slashes are one form, whatever else they are.
	same, _, _ := resolve(wearer, duelist(10, 5, 100000),
		[]Card{Of(Slice, Fire), Of(Slice, Fire)}, nil, 1)
	if e := handEventOf(t, same, SideA); e.HandScale != 100 {
		t.Errorf("a pair of one form paid %d, want the identity", e.HandScale)
	}
}

// TestMinFormsIsRefusedAnywhereButBlowFormed. Same seam Hand and Lead sit on: only one moment knows
// what formed.
func TestMinFormsIsRefusedAnywhereButBlowFormed(t *testing.T) {
	refused(t, "minFormsAtCardDamage", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{MinForms: 2},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})
}

// TestTheVitaeScalerGrowsWithThePurse. Fire of Life: a percentage point a vitae, read against the
// live purse rather than a figure fixed at the door.
func TestTheVitaeScalerGrowsWithThePurse(t *testing.T) {
	id := relic(t, "fireoflife", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoScaleDamagePerVitae, Amount: 1}},
	})

	card := Of(Slice, Fire)
	wearer := duelist(10, 5, 100).Wearing(WornRelic{Relic: id})

	bare := wearer.CardDamage(card)
	wearer.Vitae = 50
	if got, want := wearer.CardDamage(card), bare*150/100; got != want {
		t.Errorf("a fire card on a purse of 50 dealt %d, want %d (%d at nothing held)", got, want, bare)
	}

	// And it says nothing about a card it does not name.
	ice := Of(Slice, Ice)
	if got := wearer.CardDamage(ice); got != duelist(10, 5, 100).CardDamage(ice) {
		t.Errorf("an ice card was moved to %d by a fire relic", got)
	}
}

// **The badge is always one decimal place, and that is a promise the card's layout is built on**
// *(owner's call, 2026-09-09)*. The disc behind the figure is sized for two or three characters and
// a fourth is allowed to spill past its curve; a second decimal would make every multiplier past
// 10x a five-character figure and the spill would stop being a spill. `%.1f` is what holds it, so
// this is the test that fails if anyone reaches for `%v` or a variable precision.
//
// **The flat branch carries no point at all**, which is the other half of the reading: a decimal
// point means a multiplier and a `+` means a flat figure, which is what let the `x` go.
func TestTheCounterLabelIsAlwaysOneDecimalPlace(t *testing.T) {
	scaling := relic(t, "counter-scaling",
		RelicRule{When: MomentCardDamage, Then: []RelicEffect{{Do: DoScaleDamage, Amount: 100}}},
		RelicRule{When: MomentTurnTaken, Then: []RelicEffect{{Do: DoGrowOnTurn, Amount: 20}}},
	)
	flat := relic(t, "counter-flat",
		RelicRule{When: MomentFightStart, Then: []RelicEffect{{Do: DoAddHP, Amount: 5}}},
		RelicRule{When: MomentTurnTaken, Then: []RelicEffect{{Do: DoGrowOnTurn, Amount: 5}}},
	)

	for _, grown := range []int{0, 5, 20, 60, 400, 950, 4900} {
		got := CounterLabel(WornRelic{Relic: scaling, Grown: grown})
		point := strings.IndexByte(got, '.')
		if point < 0 {
			t.Errorf("a multiplier grown %d reads %q, which has no decimal point", grown, got)
			continue
		}
		if rest := got[point+1:]; len(rest) != 1 {
			t.Errorf("a multiplier grown %d reads %q, which has %d digits after the point, want 1",
				grown, got, len(rest))
		}

		if got := CounterLabel(WornRelic{Relic: flat, Grown: grown}); strings.ContainsRune(got, '.') {
			t.Errorf("a flat figure grown %d reads %q, which carries a decimal point", grown, got)
		}
	}
}
