package combat

import "testing"

// handsFormed returns the hands one side formed. A turn forms at most one, so this is a list only
// so that "none" and "one" are the same shape.
func handsFormed(events []Event, by Side) []HandID {
	var out []HandID
	for _, e := range events {
		if e.Kind == KindCombo && e.Side == by && e.Hand != HandNone {
			out = append(out, e.Hand)
		}
	}
	return out
}

// comboEventFor is the KindCombo event one side's attack phase raised.
func comboEventFor(events []Event, by Side) (Event, bool) {
	for _, e := range events {
		if e.Kind == KindCombo && e.Side == by {
			return e, true
		}
	}
	return Event{}, false
}

// damageDealtBy totals the damage one side landed on the other.
func damageDealtBy(events []Event, by Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == by {
			total += e.Amount
		}
	}
	return total
}

// staggeredActions returns the actions one side lost to a stagger.
func staggeredActions(events []Event, by Side) []ActionKind {
	var out []ActionKind
	for _, e := range events {
		if e.Kind == KindStaggered && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// sideActions returns the actions one side actually took.
func sideActions(events []Event, by Side) []ActionKind {
	var out []ActionKind
	for _, e := range events {
		if e.Kind == KindAction && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// comboCards is the cards one event says formed its hand.
func comboCards(e Event) []int { return e.ComboCards[:e.ComboCardCount] }

// --- one blow per turn --------------------------------------------------------------------

// **The rule the whole model rests on.** However many attack cards a turn queues, exactly one
// figure of damage lands.
func TestATurnDealsDamageExactlyOnce(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	for _, turn := range [][]Card{
		PlainCards(Strike),
		PlainCards(Strike, Jab),
		PlainCards(Strike, Strike),
		PlainCards(Strike, Strike, Strike),
		PlainCards(Jab, Jab, Jab, Jab, Jab),
	} {
		events, _, _ := resolve(a, b, turn, nil, 1)
		if n := kindCount(events, KindDamage); n != 1 {
			t.Errorf("%v dealt damage %d times, want exactly 1", Actions(turn), n)
		}
	}
}

// Every attack card is still announced, even the ones that contribute nothing — the screen counts
// one beat per slot to know how far through the round playback is.
func TestEveryAttackCardIsAnnouncedEvenOutsideTheHand(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike), nil, 1)

	if took := sideActions(events, SideA); len(took) != 3 {
		t.Fatalf("three cards were played and %d were announced: %v", len(took), took)
	}
}

// **A card that builds to no hand contributes nothing to the blow.** `Strike, Jab, Strike` is a
// Strike Pair and the Jab is not in it.
func TestACardOutsideTheHandAddsNothing(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	withJab, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike), nil, 1)
	without, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	if got, want := damageDealtBy(withJab, SideA), damageDealtBy(without, SideA); got != want {
		t.Errorf("the Jab added %d damage; it is in no hand and should add nothing", got-want)
	}

	e, ok := comboEventFor(withJab, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if got := comboCards(e); len(got) != 2 {
		t.Errorf("the pair says it was formed from %v, want the two Strikes only", got)
	}
}

// --- the damage formula -------------------------------------------------------------------

// **damage = the hand's own cards, plus DMG times the multiplier.** Two Strikes at Str 10 are 20
// of cards and a 1.5x pair on a DMG of 10, so 35.
func TestDamageIsTheHandsCardsPlusDMGTimesTheMultiplier(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	pair, ok := HandByName("Strike Pair")
	if !ok {
		t.Fatal("the catalogue has no Strike Pair")
	}

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	want := Strike.Damage(10)*2 + Strike.Damage(10)*pair.Multiplier/multiplierScale
	if got := damageDealtBy(events, SideA); got != want {
		t.Errorf("a plain Strike Pair dealt %d, want %d", got, want)
	}
}

// **The multiplier is additive across the two axes**, not multiplicative — a 1.5x pair that is
// also a 2x duo is 3.5x, not 3x. That is what keeps the top of the ladder at a few hundred damage
// rather than several thousand.
func TestTheHandAndMixMultipliersAdd(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	pair, _ := HandByName("Strike Pair")
	duo, _ := MixByName("Duo")

	events, _, _ := resolve(a, b, []Card{Of(Strike, Fire), Of(Strike, Ice)}, nil, 1)

	e, ok := comboEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if want := pair.Multiplier + duo.Multiplier; e.Multiplier != want {
		t.Errorf("a duo pair reports x%d, want x%d (%d + %d)",
			e.Multiplier, want, pair.Multiplier, duo.Multiplier)
	}
}

// --- which hand forms ---------------------------------------------------------------------

// **The best-paying hand wins.** Five Strikes hold a pair, a flurry and a barrage as well as an
// onslaught; only the onslaught pays.
func TestTheBestPayingHandIsTheOneThatForms(t *testing.T) {
	a, b := duelist(10, 0, 10000), duelist(10, 0, 10000)

	for _, tc := range []struct {
		n    int
		want string
	}{
		{2, "Strike Pair"},
		{3, "Strike Flurry"},
		{4, "Strike Barrage"},
		{5, "Strike Onslaught"},
	} {
		turn := make([]Card, tc.n)
		for i := range turn {
			turn[i] = Plain(Strike)
		}

		want, ok := HandByName(tc.want)
		if !ok {
			t.Fatalf("the catalogue has no %q", tc.want)
		}
		events, _, _ := resolve(a, b, turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != want.ID {
			t.Errorf("%d Strikes formed %v, want %s alone", tc.n, got, tc.want)
		}
	}
}

// Two Pair and Full House need two different concepts, so five of one card can be neither.
func TestTheTwoConceptHands(t *testing.T) {
	a, b := duelist(10, 0, 10000), duelist(10, 0, 10000)

	twoPair, _ := HandByName("Two Pair")
	fullHouse, _ := HandByName("Full House")
	onslaught, _ := HandByName("Jab Onslaught")

	for _, tc := range []struct {
		turn []Card
		want HandID
		what string
	}{
		{PlainCards(Jab, Jab, Strike, Strike), twoPair.ID, "two pairs"},
		{PlainCards(Jab, Jab, Jab, Strike, Strike), fullHouse.ID, "three and two"},
		{PlainCards(Jab, Jab, Jab, Jab, Jab), onslaught.ID, "five of one card"},
	} {
		events, _, _ := resolve(a, b, tc.turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s formed %v, want one hand", tc.what, got)
		}
	}
}

// **A counted hand does not care what sits between its cards.** Three Strikes with a Jab among
// them is a Flurry; the run-matcher this replaced needed them adjacent and formed nothing.
func TestAHandIgnoresWhatSitsBetweenItsCards(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	flurry, _ := HandByName("Strike Flurry")
	events, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike, Strike), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 1 || got[0] != flurry.ID {
		t.Fatalf("three Strikes around a Jab formed %v, want a Strike Flurry", got)
	}
}

// The ladder is scoped to attacks, so three Guards are three of a card and form nothing.
func TestTheLadderCountsAttacksAlone(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, PlainCards(Guard, Guard, Guard), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("three prepares formed %v, want nothing", got)
	}
	if n := kindCount(events, KindDamage); n != 0 {
		t.Fatalf("a turn of prepares dealt damage %d times, want 0", n)
	}
}

// **With no hand, the biggest single attack is the blow** — and it earns no multiplier, so it
// deals exactly what the card deals.
func TestWithNoHandTheBiggestAttackIsTheBlow(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Heavy, Strike), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("three different attacks formed %v, want no hand", got)
	}
	if got, want := damageDealtBy(events, SideA), Heavy.Damage(10); got != want {
		t.Errorf("dealt %d, want the Heavy's %d and nothing else", got, want)
	}
}

// --- element mixes ------------------------------------------------------------------------

// The mix is the count of *distinct non-basic* colours in the hand that formed, and basic never
// counts in either direction.
func TestTheMixCountsDistinctColoursAndIgnoresBasic(t *testing.T) {
	a, b := duelist(10, 0, 10000), duelist(10, 0, 10000)

	for _, tc := range []struct {
		what string
		turn []Card
		want string
	}{
		{"two basics", PlainCards(Strike, Strike), "Drab"},
		{"a basic and an ice", []Card{Plain(Strike), Of(Strike, Ice)}, "Mono"},
		{"two ice", []Card{Of(Strike, Ice), Of(Strike, Ice)}, "Mono"},
		{"ice and fire", []Card{Of(Strike, Ice), Of(Strike, Fire)}, "Duo"},
		{"ice, fire and a basic", []Card{Of(Strike, Ice), Of(Strike, Fire), Plain(Strike)}, "Duo"},
		{"three colours", []Card{Of(Strike, Ice), Of(Strike, Fire), Of(Strike, Earth)}, "Trio"},
		{"four colours", []Card{
			Of(Strike, Ice), Of(Strike, Fire), Of(Strike, Earth), Of(Strike, Lightning),
		}, "Rainbow"},
	} {
		want, ok := MixByName(tc.want)
		if !ok {
			t.Fatalf("the catalogue has no %q", tc.want)
		}
		events, _, _ := resolve(a, b, tc.turn, nil, 1)
		e, found := comboEventFor(events, SideA)
		if !found {
			t.Fatalf("%s: no attack phase event", tc.what)
		}
		if e.Mix != want.ID {
			got, _ := MixByID(e.Mix)
			t.Errorf("%s read as %s, want %s", tc.what, got.Name, tc.want)
		}
	}
}

// **Only the cards in the hand decide the mix.** An off-colour card that contributed to no hand
// cannot recolour the one that formed.
func TestACardOutsideTheHandDoesNotColourIt(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, []Card{Of(Strike, Ice), Of(Jab, Fire), Of(Strike, Ice)}, nil, 1)

	mono, _ := MixByName("Mono")
	e, ok := comboEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if e.Mix != mono.ID {
		got, _ := MixByID(e.Mix)
		t.Errorf("the pair read as %s, want Mono — the fire Jab is in no hand", got.Name)
	}
}

// **One status per colour in the hand**, so mono lands one and a rainbow lands four.
func TestTheMixLandsOneStatusPerColour(t *testing.T) {
	a, b := duelist(10, 0, 10000), duelist(10, 0, 10000)

	events, _, bAfter := resolve(a, b, []Card{
		Of(Strike, Fire), Of(Strike, Ice), Of(Strike, Earth), Of(Strike, Lightning),
	}, nil, 1)

	if n := kindCount(events, KindStatus); n != 4 {
		t.Errorf("a rainbow landed %d statuses, want 4", n)
	}
	for _, e := range []Element{Fire, Ice, Earth, Lightning} {
		if !bAfter.Statuses[e].Active() {
			t.Errorf("%v did not land", e)
		}
	}
}

func TestADrabHandLandsNoStatus(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	if n := kindCount(events, KindStatus); n != 0 {
		t.Errorf("a drab pair landed %d statuses, want 0 — basic is not a colour", n)
	}
}

// A lone attack that formed no hand still applies its own element, which is the rule that
// predates hands and was deliberately kept.
func TestALoneAttackStillAppliesItsElement(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, bAfter := resolve(a, b, []Card{Of(Strike, Ice), Plain(Jab)}, nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("a Strike and a Jab formed %v, want no hand", got)
	}
	if !bAfter.Statuses[Ice].Active() {
		t.Error("the ice Strike was the blow and should still have chilled")
	}
}

// --- stagger ------------------------------------------------------------------------------

func TestAFlurryCostsTheOpponentTheirNextAction(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)

	events, _, _ := resolve(a, b,
		PlainCards(Strike, Strike, Strike),
		PlainCards(Jab, Strike), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 || lost[0] != Jab {
		t.Fatalf("B should lose exactly its first action, got %v", lost)
	}
}

func TestOnslaughtTakesTheOpponentsWholeTurn(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)

	events, _, _ := resolve(a, b,
		PlainCards(Jab, Jab, Jab, Jab, Jab),
		PlainCards(Strike, Strike, Guard), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 3 {
		t.Fatalf("Onslaught should take all three of B's actions, got %v", lost)
	}
	if took := sideActions(events, SideB); len(took) != 0 {
		t.Fatalf("B should take no action at all, got %v", took)
	}
}

// A hand scored off cards a stagger deleted would let a staggered duelist stagger back with a
// turn it never took. Two Strikes survive, so a pair forms and emphatically not an onslaught.
func TestStaggeredCardsCannotFormAHand(t *testing.T) {
	a := duelist(10, 0, 20000)
	b := duelist(10, 0, 20000)
	b.Staggered = 3

	events, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike, Strike, Strike, Strike), 1)

	pair, _ := HandByName("Strike Pair")
	if got := handsFormed(events, SideB); len(got) != 1 || got[0] != pair.ID {
		t.Fatalf("two surviving Strikes should form a Strike Pair, got %v", got)
	}
}

// Side B acts last, so a stagger B earns has no turn left to bite in and carries to the round
// after. That is the one asymmetry phases impose.
func TestSideBsStaggerLandsInTheFollowingRound(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)

	events, aAfter, _ := resolve(a, b, PlainCards(Gather), PlainCards(Strike, Strike, Strike), 1)

	if lost := staggeredActions(events, SideA); len(lost) != 0 {
		t.Fatalf("A already acted, so nothing can be taken from it this round, got %v", lost)
	}
	if aAfter.Staggered != 1 {
		t.Fatalf("the stagger should be held on A for next round, got %d", aAfter.Staggered)
	}
}

// --- the event ----------------------------------------------------------------------------

// **A counted hand is not contiguous**, which is the case a start-and-length bracket could not
// describe: Two Pair is two cards, a card that earned nothing, and two more.
func TestTheEventNamesScatteredCards(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Jab, Heavy, Strike, Strike), nil, 1)

	e, ok := comboEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	got := comboCards(e)
	if len(got) != 4 || got[0] != 0 || got[1] != 1 || got[2] != 3 || got[3] != 4 {
		t.Fatalf("two pair says it was formed from %v, want [0 1 3 4] around the Heavy", got)
	}
}

// The attack phase is announced before the blow lands, so a boosted figure never arrives before
// the reason for it.
func TestTheHandIsAnnouncedBeforeTheDamage(t *testing.T) {
	a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	comboAt, damageAt := -1, -1
	for i, e := range events {
		if e.Kind == KindCombo && comboAt < 0 {
			comboAt = i
		}
		if e.Kind == KindDamage && damageAt < 0 {
			damageAt = i
		}
	}
	if comboAt < 0 || damageAt < 0 {
		t.Fatal("expected both a combo and a damage event")
	}
	if comboAt > damageAt {
		t.Error("the damage landed before the hand that explains it was announced")
	}
}

// --- housekeeping -------------------------------------------------------------------------

func TestARoundWithNoRandomnessIsDeterministic(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)
	turn := PlainCards(Strike, Strike, Strike)

	e1, a1, b1 := resolve(a, b, turn, PlainCards(Jab, Strike), 1)
	e2, a2, b2 := resolve(a, b, turn, PlainCards(Jab, Strike), 1)

	if a1 != a2 || b1 != b2 {
		t.Fatal("the same round resolved twice must end in the same state")
	}
	if len(e1) != len(e2) {
		t.Fatalf("event logs differ in length: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if e1[i] != e2[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, e1[i], e2[i])
		}
	}
}

// TestEverySlotIsEitherTakenOrStaggered is the invariant the screen's highlight rests on:
// CombatScene.currentSlot counts one beat per slot, taken or lost, and would light the wrong card
// for the rest of the round if a slot went unaccounted for.
func TestEverySlotIsEitherTakenOrStaggered(t *testing.T) {
	a, b := duelist(10, 0, 20000), duelist(10, 0, 20000)
	aPlan := PlainCards(Strike, Strike, Strike)
	bPlan := PlainCards(Gather, Jab, Strike, Dodge)

	events, _, _ := resolve(a, b, aPlan, bPlan, 1)
	order := ResolutionOrder(aPlan, bPlan)

	var beats []ActionKind
	for _, e := range events {
		if e.Kind == KindAction || e.Kind == KindStaggered {
			beats = append(beats, e.Action)
		}
	}

	if len(beats) != len(order) {
		t.Fatalf("every slot needs exactly one beat: %d slots, %d beats", len(order), len(beats))
	}
	for i, slot := range order {
		if beats[i] != slot.Card.Action {
			t.Fatalf("beat %d is %v, but slot %d is %v", i, beats[i], i, slot.Card.Action)
		}
	}

	if lost := staggeredActions(events, SideB); len(lost) == 0 {
		t.Fatal("this fixture is meant to stagger side B")
	}
}
